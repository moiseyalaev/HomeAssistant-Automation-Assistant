package ha

// websocket.go — Phase 2 implementation
// Will hold the persistent HA WebSocket connection for registry events.
// Currently a placeholder; entity data is fetched via REST polling in client.go.
//
// Planned:
//   - Connect to ws://HA_URL/api/websocket
//   - Authenticate with HA_TOKEN
//   - Subscribe to state_changed events → update cache in real time
//   - Subscribe to entity/area registry → enrich entity metadata
//   - Reconnect loop with exponential backoff
