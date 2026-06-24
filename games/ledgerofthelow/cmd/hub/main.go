// Command hub runs the InterDOOR federation hub (FEDERATION_PROTOCOL.md).
package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"interdoor.net/interdoor/internal/hub"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address (terminate TLS at a reverse proxy)")
	dbPath := flag.String("db", "hub.db", "hub database path (SQLite dev backend)")
	regToken := flag.String("reg-token", "", "registration token nodes must present (required)")
	regTokenFile := flag.String("reg-token-file", "", "path to a file containing the registration token")
	sshAddr := flag.String("ssh-addr", ":2300", "SSH portal listen address (empty to disable)")
	sshHostKey := flag.String("ssh-hostkey", "", "path to SSH host key PEM file (enables the portal)")
	flag.Parse()

	if *regTokenFile != "" {
		b, err := os.ReadFile(*regTokenFile)
		if err != nil {
			log.Fatalf("read reg-token-file: %v", err)
		}
		*regToken = strings.TrimSpace(string(b))
	}
	if *regToken == "" {
		log.Fatal("-reg-token or -reg-token-file is required")
	}

	store, err := hub.OpenSQLite(*dbPath)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer store.Close()

	srv := &http.Server{Addr: *addr, Handler: hub.NewServer(store, *regToken).Handler()}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("serve: %v", err)
		}
	}()
	log.Printf("InterDOOR hub listening on %s (protocol v%s)", *addr, hub.ProtocolVersion)

	if *sshAddr != "" && *sshHostKey != "" {
		portal := &hub.Portal{Store: store}
		go func() {
			if err := portal.ListenAndServe(*sshAddr, *sshHostKey); err != nil {
				log.Fatalf("portal: %v", err)
			}
		}()
		log.Printf("InterDOOR hub SSH portal on %s", *sshAddr)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	log.Printf("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
