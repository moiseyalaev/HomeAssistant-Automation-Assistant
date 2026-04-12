package ha

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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
	ticker := time.NewTicker(10 * time.Second)
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

// FetchVersion returns the running Home Assistant version string (e.g. "2024.4.1").
// It calls GET /api/ which returns {"message": "...", "version": "..."}.
func (c *Client) FetchVersion() (string, error) {
	req, err := http.NewRequest("GET", c.baseURL+"/api/", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HA returned %d on /api/", resp.StatusCode)
	}

	var result struct {
		Version string `json:"version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.Version, nil
}

// WriteAutomation writes an automation to automations.yaml via HA's config REST API
// and triggers a reload. The id is used as the automation's YAML key and becomes
// the entity_id suffix (e.g. id "my-auto" → automation.my_auto).
func (c *Client) WriteAutomation(id string, config map[string]interface{}) error {
	alias, _ := config["alias"].(string)
	log.Printf("write automation: id=%q alias=%q", id, alias)

	body, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to encode automation config: %w", err)
	}
	log.Printf("write automation: payload=%s", body)

	url := c.baseURL + "/api/config/automation/config/" + id
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		log.Printf("write automation: request failed: %v", err)
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	log.Printf("write automation: HA responded %d — %s", resp.StatusCode, respBody)

	if resp.StatusCode >= 300 {
		return fmt.Errorf("HA returned %d when writing automation %q: %s", resp.StatusCode, id, respBody)
	}

	return c.reloadAutomations()
}

func (c *Client) reloadAutomations() error {
	log.Printf("reloading automations")
	req, err := http.NewRequest("POST", c.baseURL+"/api/services/automation/reload", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		log.Printf("reload automations: request failed: %v", err)
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		log.Printf("reload automations: HA returned %d — %s", resp.StatusCode, respBody)
		return fmt.Errorf("HA reload returned %d: %s", resp.StatusCode, respBody)
	}
	log.Printf("reload automations: done (%d)", resp.StatusCode)
	return nil
}
