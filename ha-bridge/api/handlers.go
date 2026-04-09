package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/moiseyalaev/ha-bridge/ha"
)

type Handler struct {
	cache  *ha.Cache
	client *ha.Client
}

func NewHandler(cache *ha.Cache, client *ha.Client) *Handler {
	return &Handler{cache: cache, client: client}
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	stale := h.cache.IsStale()
	status := "ok"
	if stale {
		status = "stale"
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": status})
}

func (h *Handler) Entities(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, h.cache.All())
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
	writeJSON(w, http.StatusOK, entity)
}

type writeAutomationRequest struct {
	ID   string `json:"id"`
	YAML string `json:"yaml"`
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
	if err := h.client.WriteAutomation(req.ID, req.YAML); err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "written"})
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}
