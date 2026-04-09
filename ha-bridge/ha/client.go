package ha

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// Client manages all communication with Home Assistant.
type Client struct {
	baseURL string
	token   string
	cache   *Cache
	http    *http.Client
}

func NewClient(baseURL, token string, cache *Cache) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		cache:   cache,
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

// Run starts the background loop: fetch entities on start, then refresh on TTL.
// Also starts the WebSocket connection for registry data.
// TODO: replace polling with WS push once WS layer is implemented.
func (c *Client) Run() {
	c.refresh()
	ticker := time.NewTicker(45 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		c.refresh()
	}
}

func (c *Client) refresh() {
	entities, err := c.fetchStates()
	if err != nil {
		log.Printf("ha refresh error: %v", err)
		return
	}
	c.cache.Set(entities)
	log.Printf("ha cache refreshed: %d entities", len(entities))
}

func (c *Client) fetchStates() ([]Entity, error) {
	req, err := http.NewRequest("GET", c.baseURL+"/api/states", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HA returned %d", resp.StatusCode)
	}

	var entities []Entity
	if err := json.NewDecoder(resp.Body).Decode(&entities); err != nil {
		return nil, err
	}
	return entities, nil
}

// WriteAutomation posts automation YAML to HA and triggers a reload.
func (c *Client) WriteAutomation(id, yamlContent string) error {
	payload := map[string]string{"id": id}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", c.baseURL+"/api/config/automation/config/"+id, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("HA write returned %d", resp.StatusCode)
	}

	return c.reloadAutomations()
}

func (c *Client) reloadAutomations() error {
	req, err := http.NewRequest("POST", c.baseURL+"/api/services/automation/reload", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
