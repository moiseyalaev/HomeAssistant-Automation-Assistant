package ha

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// RegistryClient connects to HA's WebSocket API to fetch area, device,
// and entity registry data. Registry data changes infrequently, so we
// fetch it once on connect and re-fetch after each reconnect.
//
// Does NOT subscribe to state_changed events — entity states continue
// to be polled via REST in client.go.
type RegistryClient struct {
	haURL string
	token string
	cache *RegistryCache
}

// NewRegistryClient creates a new RegistryClient.
func NewRegistryClient(haURL, token string, cache *RegistryCache) *RegistryClient {
	return &RegistryClient{
		haURL: strings.TrimRight(haURL, "/"),
		token: token,
		cache: cache,
	}
}

// Run is a blocking reconnect loop. It connects to HA WebSocket, authenticates,
// fetches all registries, then waits before reconnecting on failure.
func (rc *RegistryClient) Run() {
	backoff := time.Second
	const maxBackoff = 60 * time.Second

	for {
		err := rc.connect()
		if err != nil {
			log.Printf("registry client: error: %v — retrying in %s", err, backoff)
		}
		time.Sleep(backoff)
		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}

// wsMessage is a generic WebSocket message envelope.
type wsMessage struct {
	Type    string          `json:"type"`
	ID      int             `json:"id,omitempty"`
	Success bool            `json:"success,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
}

// connect performs one full connection cycle: authenticate + fetch registries.
func (rc *RegistryClient) connect() error {
	wsURL := toWSURL(rc.haURL) + "/api/websocket"
	log.Printf("registry client: connecting to %s", wsURL)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer conn.Close()

	// Step 1: receive auth_required
	var msg wsMessage
	if err := conn.ReadJSON(&msg); err != nil {
		return fmt.Errorf("read auth_required: %w", err)
	}
	if msg.Type != "auth_required" {
		return fmt.Errorf("expected auth_required, got %q", msg.Type)
	}

	// Step 2: send auth
	authMsg := map[string]string{
		"type":         "auth",
		"access_token": rc.token,
	}
	if err := conn.WriteJSON(authMsg); err != nil {
		return fmt.Errorf("write auth: %w", err)
	}

	// Step 3: receive auth_ok
	if err := conn.ReadJSON(&msg); err != nil {
		return fmt.Errorf("read auth_ok: %w", err)
	}
	if msg.Type == "auth_invalid" {
		return fmt.Errorf("authentication failed: auth_invalid")
	}
	if msg.Type != "auth_ok" {
		return fmt.Errorf("expected auth_ok, got %q", msg.Type)
	}
	log.Printf("registry client: authenticated")

	// Step 4: fetch all three registries sequentially.
	areas, err := rc.fetchAreas(conn)
	if err != nil {
		return fmt.Errorf("fetch areas: %w", err)
	}

	devices, err := rc.fetchDevices(conn)
	if err != nil {
		return fmt.Errorf("fetch devices: %w", err)
	}

	entities, err := rc.fetchEntities(conn)
	if err != nil {
		return fmt.Errorf("fetch entities: %w", err)
	}

	rc.cache.Set(areas, devices, entities)
	log.Printf("registry client: registry loaded — %d areas, %d devices, %d entities",
		len(areas), len(devices), len(entities))

	// Hold the connection open; reconnect if it drops.
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("connection lost: %w", err)
		}
	}
}

// fetchAreas sends the area registry list request and parses the response.
func (rc *RegistryClient) fetchAreas(conn *websocket.Conn) ([]Area, error) {
	req := map[string]interface{}{"id": 1, "type": "config/area_registry/list"}
	if err := conn.WriteJSON(req); err != nil {
		return nil, err
	}

	var msg wsMessage
	if err := conn.ReadJSON(&msg); err != nil {
		return nil, err
	}
	if !msg.Success {
		return nil, fmt.Errorf("area_registry/list returned success=false")
	}

	var raw []struct {
		AreaID string `json:"area_id"`
		Name   string `json:"name"`
	}
	if err := json.Unmarshal(msg.Result, &raw); err != nil {
		return nil, err
	}

	areas := make([]Area, 0, len(raw))
	for _, r := range raw {
		areas = append(areas, Area{ID: r.AreaID, Name: r.Name})
	}
	return areas, nil
}

// fetchDevices sends the device registry list request and parses the response.
func (rc *RegistryClient) fetchDevices(conn *websocket.Conn) ([]Device, error) {
	req := map[string]interface{}{"id": 2, "type": "config/device_registry/list"}
	if err := conn.WriteJSON(req); err != nil {
		return nil, err
	}

	var msg wsMessage
	if err := conn.ReadJSON(&msg); err != nil {
		return nil, err
	}
	if !msg.Success {
		return nil, fmt.Errorf("device_registry/list returned success=false")
	}

	var raw []struct {
		ID           string  `json:"id"`
		Name         *string `json:"name"`
		AreaID       *string `json:"area_id"`
		Manufacturer *string `json:"manufacturer"`
		Model        *string `json:"model"`
	}
	if err := json.Unmarshal(msg.Result, &raw); err != nil {
		return nil, err
	}

	devices := make([]Device, 0, len(raw))
	for _, r := range raw {
		d := Device{ID: r.ID}
		if r.Name != nil {
			d.Name = *r.Name
		}
		if r.AreaID != nil {
			d.AreaID = *r.AreaID
		}
		if r.Manufacturer != nil {
			d.Manufacturer = *r.Manufacturer
		}
		if r.Model != nil {
			d.Model = *r.Model
		}
		devices = append(devices, d)
	}
	return devices, nil
}

// fetchEntities sends the entity registry list request and parses the response.
func (rc *RegistryClient) fetchEntities(conn *websocket.Conn) ([]EntityMeta, error) {
	req := map[string]interface{}{"id": 3, "type": "config/entity_registry/list"}
	if err := conn.WriteJSON(req); err != nil {
		return nil, err
	}

	var msg wsMessage
	if err := conn.ReadJSON(&msg); err != nil {
		return nil, err
	}
	if !msg.Success {
		return nil, fmt.Errorf("entity_registry/list returned success=false")
	}

	var raw []struct {
		EntityID string  `json:"entity_id"`
		DeviceID *string `json:"device_id"`
		AreaID   *string `json:"area_id"`
	}
	if err := json.Unmarshal(msg.Result, &raw); err != nil {
		return nil, err
	}

	entities := make([]EntityMeta, 0, len(raw))
	for _, r := range raw {
		e := EntityMeta{EntityID: r.EntityID}
		if r.DeviceID != nil {
			e.DeviceID = *r.DeviceID
		}
		if r.AreaID != nil {
			e.AreaID = *r.AreaID
		}
		entities = append(entities, e)
	}
	return entities, nil
}

// toWSURL converts http:// to ws:// and https:// to wss://.
func toWSURL(u string) string {
	if strings.HasPrefix(u, "https://") {
		return "wss://" + strings.TrimPrefix(u, "https://")
	}
	return "ws://" + strings.TrimPrefix(u, "http://")
}
