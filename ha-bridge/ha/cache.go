package ha

import (
	"sync"
	"time"
)

// Entity mirrors the relevant fields from HA's /api/states response.
type Entity struct {
	EntityID   string                 `json:"entity_id"`
	State      string                 `json:"state"`
	Attributes map[string]interface{} `json:"attributes"`
	LastChanged time.Time             `json:"last_changed"`
}

// Cache holds the entity list with a TTL and concurrent-read-safe access.
type Cache struct {
	mu        sync.RWMutex
	entities  map[string]Entity
	cachedAt  time.Time
	ttl       time.Duration
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
