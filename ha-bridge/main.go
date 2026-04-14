package main

import (
	"log"
	"net/http"
	"os"

	"github.com/moiseyalaev/ha-bridge/api"
	"github.com/moiseyalaev/ha-bridge/ha"
)

func main() {
	haURL := os.Getenv("HA_URL")
	haToken := os.Getenv("HA_TOKEN")
	port := os.Getenv("BRIDGE_PORT")
	if port == "" {
		port = "8080"
	}

	registryCache := ha.NewRegistryCache()
	registryClient := ha.NewRegistryClient(haURL, haToken, registryCache)
	go registryClient.Run()

	cache := ha.NewCache()
	client := ha.NewClient(haURL, haToken, cache)

	go client.Run() // persistent WS connection + cache refresh

	h := api.NewHandler(cache, registryCache, client)
	mux := http.NewServeMux()
	mux.HandleFunc("/health", h.Health)
	mux.HandleFunc("/version", h.Version)
	mux.HandleFunc("/entities", h.Entities)
	mux.HandleFunc("/entities/", h.EntityByID)
	mux.HandleFunc("/automations", h.WriteAutomation)

	log.Printf("ha-bridge listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
