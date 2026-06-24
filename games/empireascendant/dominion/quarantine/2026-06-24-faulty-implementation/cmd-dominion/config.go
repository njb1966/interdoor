package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
)

// Config is the operator configuration for an Empire Ascendant node.
// Precedence: built-in defaults, then optional JSON file (-config), then
// explicitly-set command flags.
type Config struct {
	Addr          string `json:"addr"`
	DB            string `json:"db"`
	HostKey       string `json:"hostkey"`
	Node          string `json:"node"`
	MaxSessions   int    `json:"max_sessions"`
	IdleTimeout   int    `json:"idle_timeout_sec"`
	HubURL        string `json:"hub_url"`
	HubRegToken   string `json:"hub_reg_token"`
	AdvertiseAddr string `json:"advertise_addr"`
	SyncInterval  int    `json:"sync_interval_sec"`
}

func loadConfig() Config {
	cfg := Config{
		Addr: ":2324", DB: "dominion.db", HostKey: "dominion-hostkey",
		Node: "dominion01", MaxSessions: 64, IdleTimeout: 600, SyncInterval: 20,
	}

	configPath := flag.String("config", "", "optional JSON config file")
	addr := flag.String("addr", cfg.Addr, "SSH listen address")
	db := flag.String("db", cfg.DB, "SQLite database path")
	hostkey := flag.String("hostkey", cfg.HostKey, "SSH host key path (created if absent)")
	node := flag.String("node", cfg.Node, "this node's identifier")
	maxSess := flag.Int("max-sessions", cfg.MaxSessions, "max concurrent sessions (0 = unlimited)")
	idle := flag.Int("idle-timeout", cfg.IdleTimeout, "idle session timeout in seconds (0 = none)")
	hubURL := flag.String("hub", cfg.HubURL, "federation hub base URL (empty = standalone)")
	hubReg := flag.String("hub-reg-token", cfg.HubRegToken, "hub registration token (first run only)")
	advertise := flag.String("advertise", cfg.AdvertiseAddr, "public SSH address, e.g. ssh://host:2324")
	syncInt := flag.Int("sync-interval", cfg.SyncInterval, "federation sync interval in seconds")
	flag.Parse()

	if *configPath != "" {
		data, err := os.ReadFile(*configPath)
		if err != nil {
			log.Fatalf("config: %v", err)
		}
		if err := json.Unmarshal(data, &cfg); err != nil {
			log.Fatalf("config: %v", err)
		}
	}

	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "addr":
			cfg.Addr = *addr
		case "db":
			cfg.DB = *db
		case "hostkey":
			cfg.HostKey = *hostkey
		case "node":
			cfg.Node = *node
		case "max-sessions":
			cfg.MaxSessions = *maxSess
		case "idle-timeout":
			cfg.IdleTimeout = *idle
		case "hub":
			cfg.HubURL = *hubURL
		case "hub-reg-token":
			cfg.HubRegToken = *hubReg
		case "advertise":
			cfg.AdvertiseAddr = *advertise
		case "sync-interval":
			cfg.SyncInterval = *syncInt
		}
	})
	return cfg
}
