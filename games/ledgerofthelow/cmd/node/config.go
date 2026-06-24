package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
)

// Config is a node's operator configuration. Precedence: built-in defaults, then
// an optional JSON config file (-config), then any explicitly-set command flags.
type Config struct {
	Addr         string `json:"addr"`
	DB           string `json:"db"`
	HostKey      string `json:"hostkey"`
	Node         string `json:"node"`
	MaxSessions  int    `json:"max_sessions"`
	IdleTimeout  int    `json:"idle_timeout_sec"`
	HubURL       string `json:"hub_url"`        // "" = standalone (no federation)
	HubRegToken  string `json:"hub_reg_token"`  // used once to register if no api key yet
	AdvertiseAddr string `json:"advertise_addr"` // public SSH address shown in the node directory, e.g. ssh://mynode.example.com:2323
	SyncInterval int    `json:"sync_interval_sec"`
}

func loadConfig() Config {
	cfg := Config{
		Addr: ":2323", DB: "interdoor.db", HostKey: "hostkey", Node: "node01",
		MaxSessions: 64, IdleTimeout: 600, SyncInterval: 20,
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
	advertise := flag.String("advertise", cfg.AdvertiseAddr, "public SSH address for the node directory, e.g. ssh://host:2323")
	syncInt := flag.Int("sync-interval", cfg.SyncInterval, "federation sync interval in seconds")
	flag.Parse()

	// A config file overrides the defaults (JSON only sets the fields it contains).
	if *configPath != "" {
		data, err := os.ReadFile(*configPath)
		if err != nil {
			log.Fatalf("config: %v", err)
		}
		if err := json.Unmarshal(data, &cfg); err != nil {
			log.Fatalf("config: %v", err)
		}
	}

	// Explicitly-set flags win over the file and defaults.
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
