package ha

import (
	"sync"
	"time"
)

// Entity mirrors the relevant fields from HA's /api/states response.
type Entity struct {
	EntityID    string                 `json:"entity_id"`
	State       string                 `json:"state"`
	Attributes  map[string]interface{} `json:"attributes"`
	LastChanged time.Time              `json:"last_changed"`
}

// Cache holds the entity list with a TTL and concurrent-read-safe access.
type Cache struct {
	mu       sync.RWMutex
	entities map[string]Entity
	cachedAt time.Time
	ttl      time.Duration
}

func NewCache() *Cache {
	return &Cache{
		entities: make(map[string]Entity),
		ttl:      10 * time.Second,
	}
}

func (c *Cache) Set(entities []Entity) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entities = make(map[string]Entity, len(entities))
	for _, e := range entities {
		c.entities[e.EntityID] = e
	}
	c.cachedAt = time.Now()
}

func (c *Cache) All() []Entity {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]Entity, 0, len(c.entities))
	for _, e := range c.entities {
		out = append(out, e)
	}
	return out
}

func (c *Cache) Get(entityID string) (Entity, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.entities[entityID]
	return e, ok
}

func (c *Cache) IsStale() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return time.Since(c.cachedAt) > c.ttl
}

// ---------------------------------------------------------------------------
// Registry types
// ---------------------------------------------------------------------------

// Area represents a Home Assistant area (room/zone).
type Area struct {
	ID   string
	Name string
}

// Device represents a Home Assistant device.
type Device struct {
	ID           string
	Name         string
	AreaID       string
	Manufacturer string
	Model        string
}

// EntityMeta holds registry metadata for an entity.
// EntityID is used internally by RegistryCache.Set to build the lookup map.
type EntityMeta struct {
	EntityID string // key — not exported in JSON
	DeviceID string
	AreaID   string // entity-level area; overrides Device.AreaID when set
}

// EnrichedEntity combines Entity state data with registry metadata.
type EnrichedEntity struct {
	Entity
	AreaName     string `json:"area_name,omitempty"`
	DeviceName   string `json:"device_name,omitempty"`
	Manufacturer string `json:"manufacturer,omitempty"`
	Model        string `json:"model,omitempty"`
}

// RegistryCache holds area, device, and entity registry data fetched via WebSocket.
type RegistryCache struct {
	mu       sync.RWMutex
	areas    map[string]Area
	devices  map[string]Device
	entities map[string]EntityMeta
	ready    bool
}

// NewRegistryCache creates an empty RegistryCache.
func NewRegistryCache() *RegistryCache {
	return &RegistryCache{
		areas:    make(map[string]Area),
		devices:  make(map[string]Device),
		entities: make(map[string]EntityMeta),
	}
}

// Set atomically replaces all registry data and marks the cache as ready.
func (rc *RegistryCache) Set(areas []Area, devices []Device, entities []EntityMeta) {
	areaMap := make(map[string]Area, len(areas))
	for _, a := range areas {
		areaMap[a.ID] = a
	}

	deviceMap := make(map[string]Device, len(devices))
	for _, d := range devices {
		deviceMap[d.ID] = d
	}

	entityMap := make(map[string]EntityMeta, len(entities))
	for _, e := range entities {
		entityMap[e.EntityID] = e
	}

	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.areas = areaMap
	rc.devices = deviceMap
	rc.entities = entityMap
	rc.ready = true
}

// IsReady returns true once the first successful registry fetch has completed.
func (rc *RegistryCache) IsReady() bool {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return rc.ready
}

// Enrich returns area name, device name, manufacturer, and model for an entity.
// Returns empty strings for any data that is missing or unknown.
func (rc *RegistryCache) Enrich(entityID string) (areaName, deviceName, manufacturer, model string) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	meta, ok := rc.entities[entityID]
	if !ok {
		return
	}

	// Start with entity-level areaID; fall back to device-level areaID.
	resolvedAreaID := meta.AreaID

	if meta.DeviceID != "" {
		if dev, ok := rc.devices[meta.DeviceID]; ok {
			deviceName = dev.Name
			manufacturer = dev.Manufacturer
			model = dev.Model
			if resolvedAreaID == "" {
				resolvedAreaID = dev.AreaID
			}
		}
	}

	if resolvedAreaID != "" {
		if area, ok := rc.areas[resolvedAreaID]; ok {
			areaName = area.Name
		}
	}

	return
}
