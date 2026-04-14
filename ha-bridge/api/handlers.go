package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/moiseyalaev/ha-bridge/ha"
)

type Handler struct {
	cache         *ha.Cache
	registryCache *ha.RegistryCache
	client        *ha.Client
}

func NewHandler(cache *ha.Cache, registryCache *ha.RegistryCache, client *ha.Client) *Handler {
	return &Handler{cache: cache, registryCache: registryCache, client: client}
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	stale := h.cache.IsStale()
	status := "ok"
	if stale {
		status = "stale"
	}
	registryStatus := "unavailable"
	if h.registryCache.IsReady() {
		registryStatus = "ok"
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"status":          status,
		"registry_status": registryStatus,
	})
}

func (h *Handler) Entities(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	entities := h.cache.All()

	// ?domains=light,switch,climate — filter to specific domains if provided
	if raw := r.URL.Query().Get("domains"); raw != "" {
		allowed := make(map[string]bool)
		for _, d := range strings.Split(raw, ",") {
			allowed[strings.TrimSpace(d)] = true
		}
		filtered := entities[:0]
		for _, e := range entities {
			domain := strings.SplitN(e.EntityID, ".", 2)[0]
			if allowed[domain] {
				filtered = append(filtered, e)
			}
		}
		entities = filtered
	}

	enriched := make([]ha.EnrichedEntity, 0, len(entities))
	for _, e := range entities {
		areaName, deviceName, manufacturer, model := h.registryCache.Enrich(e.EntityID)
		enriched = append(enriched, ha.EnrichedEntity{
			Entity:       e,
			AreaName:     areaName,
			DeviceName:   deviceName,
			Manufacturer: manufacturer,
			Model:        model,
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"registry_available": h.registryCache.IsReady(),
		"entities":           enriched,
	})
}

func (h *Handler) EntityByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/entities/")
	if id == "" {
		http.Error(w, "entity_id required", http.StatusBadRequest)
		return
	}
	entity, ok := h.cache.Get(id)
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	areaName, deviceName, manufacturer, model := h.registryCache.Enrich(entity.EntityID)
	enriched := ha.EnrichedEntity{
		Entity:       entity,
		AreaName:     areaName,
		DeviceName:   deviceName,
		Manufacturer: manufacturer,
		Model:        model,
	}
	writeJSON(w, http.StatusOK, enriched)
}

type writeAutomationRequest struct {
	ID     string                 `json:"id"`
	Config map[string]interface{} `json:"config"`
}

func (h *Handler) WriteAutomation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req writeAutomationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if req.ID == "" || len(req.Config) == 0 {
		http.Error(w, "id and config are required", http.StatusBadRequest)
		return
	}
	if err := h.client.WriteAutomation(req.ID, req.Config); err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "written"})
}

func (h *Handler) Version(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	version, err := h.client.FetchVersion()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"version": version})
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}
