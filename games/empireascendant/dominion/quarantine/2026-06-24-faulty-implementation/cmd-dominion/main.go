// Command dominion runs an Empire Ascendant node: an SSH server hosting the game.
package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/crypto/ssh"

	"interdoor.net/interdoor/internal/dominion"
	"interdoor.net/interdoor/internal/engine"
	"interdoor.net/interdoor/internal/fed"
)

func main() {
	cfg := loadConfig()

	store, err := engine.Open(cfg.DB, cfg.Node)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer store.Close()
	if err := store.Migrate(); err != nil {
		log.Fatalf("migrate engine: %v", err)
	}

	store.RegisterDebtHandlers()

	g := dominion.New(cfg.Node)
	if err := g.Migrate(store.DB()); err != nil {
		log.Fatalf("migrate game: %v", err)
	}
	g.RegisterGameHandlers(store)

	signer, err := loadOrCreateHostKey(cfg.HostKey)
	if err != nil {
		log.Fatalf("host key: %v", err)
	}

	srv := engine.NewServer(store, g, engine.Options{
		Addr:        cfg.Addr,
		HostKey:     signer,
		NodeID:      cfg.Node,
		MaxSessions: cfg.MaxSessions,
		IdleTimeout: time.Duration(cfg.IdleTimeout) * time.Second,
	})
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Fatalf("serve: %v", err)
		}
	}()
	log.Printf("Empire Ascendant node %q on %s — game %q (max %d sessions, idle %ds)",
		cfg.Node, cfg.Addr, g.Title(), cfg.MaxSessions, cfg.IdleTimeout)

	// Federation: register with the hub (first run) and run the sync loop.
	if cfg.HubURL != "" {
		client := &fed.Client{BaseURL: cfg.HubURL}
		_, _, apiKey, err := store.SyncState()
		if err != nil {
			log.Fatalf("sync state: %v", err)
		}
		if apiKey == "" {
			if cfg.HubRegToken == "" {
				log.Fatalf("hub %s set but no stored API key and no -hub-reg-token to register", cfg.HubURL)
			}
			apiKey, err = client.Register(cfg.HubRegToken, cfg.Node, g.ID(), dominion.Version, cfg.AdvertiseAddr)
			if err != nil {
				log.Fatalf("hub register: %v", err)
			}
			if err := store.SetAPIKey(apiKey); err != nil {
				log.Fatalf("store api key: %v", err)
			}
			log.Printf("registered node %q with hub %s", cfg.Node, cfg.HubURL)
		}
		client.APIKey = apiKey
		syncer := &fed.Syncer{Store: store, Client: client, NodeID: cfg.Node, GameVersion: dominion.Version}
		syncer.PvPResolve = g.IncomingPvP
		syncer.TravelImport = func(st *engine.Store, snap *engine.Snapshot) error {
			if err := st.ImportPlayer(snap, g); err != nil {
				return err
			}
			if snap.Player.HomeNode == cfg.Node {
				_ = st.SetPlayerStatus(snap.Player.GlobalID, "active")
			}
			return st.Emit("player.traveled", map[string]any{
				"global_id": snap.Player.GlobalID,
				"dest_node": cfg.Node,
			})
		}
		srv.SetCrossNodeAttack(func(attackerID, victimID string, payload json.RawMessage) (string, error) {
			return client.QueuePvP(attackerID, victimID, payload)
		})
		srv.SetTravelFn(func(globalID, destNode string) error {
			snap, err := store.ExportPlayer(globalID, g)
			if err != nil {
				return err
			}
			snapJSON, err := json.Marshal(snap)
			if err != nil {
				return err
			}
			if err := store.SetPlayerStatus(globalID, "traveling"); err != nil {
				return err
			}
			if _, err := client.SubmitTravel(globalID, snap.Player.HomeNode, destNode, json.RawMessage(snapJSON)); err != nil {
				_ = store.SetPlayerStatus(globalID, "active")
				return err
			}
			return nil
		})
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go syncer.Run(ctx, time.Duration(cfg.SyncInterval)*time.Second)
		log.Printf("federation sync active -> %s (every %ds)", cfg.HubURL, cfg.SyncInterval)
	}

	// Graceful shutdown.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	log.Printf("shutting down")
	_ = srv.Shutdown()
}

// loadOrCreateHostKey reads an ed25519 host key, generating and persisting one on
// first run.
func loadOrCreateHostKey(path string) (ssh.Signer, error) {
	pemBytes, err := os.ReadFile(path)
	if err == nil {
		return ssh.ParsePrivateKey(pemBytes)
	}
	if !os.IsNotExist(err) {
		return nil, err
	}
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return nil, err
	}
	pemBytes = pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	if err := os.WriteFile(path, pemBytes, 0o600); err != nil {
		return nil, err
	}
	return ssh.ParsePrivateKey(pemBytes)
}
