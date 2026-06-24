package fed

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"interdoor.net/interdoor/internal/engine"
	"interdoor.net/interdoor/internal/game"
	"interdoor.net/interdoor/internal/hub"
)

func openNode(t *testing.T, id string) *engine.Store {
	t.Helper()
	s, err := engine.Open(filepath.Join(t.TempDir(), id+".db"), id)
	if err != nil {
		t.Fatalf("open %s: %v", id, err)
	}
	if err := s.Migrate(); err != nil {
		t.Fatalf("migrate %s: %v", id, err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func register(t *testing.T, baseURL, nodeID string) *Client {
	t.Helper()
	c := &Client{BaseURL: baseURL}
	key, err := c.Register("regsecret", nodeID, "ledger_of_the_low", "1.0.0", "")
	if err != nil {
		t.Fatalf("register %s: %v", nodeID, err)
	}
	c.APIKey = key
	return c
}

func reserveAddr(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve addr: %v", err)
	}
	addr := ln.Addr().String()
	if err := ln.Close(); err != nil {
		t.Fatalf("close reserved addr: %v", err)
	}
	return addr
}

func startHubProcess(t *testing.T, addr, dbPath string) (baseURL string, stop func()) {
	t.Helper()
	moduleRoot := filepath.Join("..", "..")
	bin := filepath.Join(t.TempDir(), "interdoor-hub-test")
	build := exec.Command("go", "build", "-o", bin, "./cmd/hub")
	build.Dir = moduleRoot
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build hub process binary: %v\n%s", err, out)
	}

	ctx, cancel := context.WithCancel(context.Background())
	var logs bytes.Buffer
	cmd := exec.CommandContext(ctx, bin,
		"-addr", addr,
		"-db", dbPath,
		"-reg-token", "regsecret",
		"-ssh-addr", "",
	)
	cmd.Stdout = &logs
	cmd.Stderr = &logs
	if err := cmd.Start(); err != nil {
		cancel()
		t.Fatalf("start hub process: %v", err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	baseURL = "http://" + addr
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case err := <-done:
			cancel()
			t.Fatalf("hub process exited early: %v\n%s", err, logs.String())
		default:
		}
		resp, err := http.Get(baseURL + "/v1/status")
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return baseURL, func() {
					cancel()
					select {
					case <-done:
					case <-time.After(5 * time.Second):
						t.Fatalf("hub process did not stop after cancel:\n%s", logs.String())
					}
				}
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
	}
	t.Fatalf("hub process did not become ready:\n%s", logs.String())
	return "", func() {}
}

func newFederatedHub(t *testing.T) (*httptest.Server, hub.Store) {
	t.Helper()
	hubStore, err := hub.OpenSQLite(filepath.Join(t.TempDir(), "hub.db"))
	if err != nil {
		t.Fatalf("hub open: %v", err)
	}
	ts := httptest.NewServer(hub.NewServer(hubStore, "regsecret").Handler())
	t.Cleanup(func() { ts.Close(); _ = hubStore.Close() })
	return ts, hubStore
}

func newLedgerNode(t *testing.T, id string) (*engine.Store, *game.Ledger) {
	t.Helper()
	s := openNode(t, id)
	s.RegisterDebtHandlers()
	g := game.New(id)
	if err := g.Migrate(s.DB()); err != nil {
		t.Fatalf("migrate game %s: %v", id, err)
	}
	return s, g
}

func playerCreatedPayload(nodeID, playerID, name string) map[string]any {
	return map[string]any{
		"global_id":  nodeID + ":" + playerID,
		"name":       name,
		"home_node":  nodeID,
		"created_at": int64(1),
	}
}

// TestPartitionTolerance verifies FEDERATION_PROTOCOL.md §11:
// - Tick fails gracefully when the hub is unreachable (no panic, no data loss).
// - Events accumulate in the local store during the partition.
// - On reconnect the backlog is pushed and the peer receives it (idempotent replay).
func TestPartitionTolerance(t *testing.T) {
	hubStore, err := hub.OpenSQLite(filepath.Join(t.TempDir(), "hub.db"))
	if err != nil {
		t.Fatalf("hub open: %v", err)
	}
	defer hubStore.Close()
	ts := httptest.NewServer(hub.NewServer(hubStore, "regsecret").Handler())
	defer ts.Close()

	a := openNode(t, "node01")
	b := openNode(t, "node02")
	sa := &Syncer{Store: a, Client: register(t, ts.URL, "node01"), NodeID: "node01"}
	sb := &Syncer{Store: b, Client: register(t, ts.URL, "node02"), NodeID: "node02"}

	// A emits an event before the partition.
	if err := a.Emit("player.created", playerCreatedPayload("node01", "p_x", "Rook")); err != nil {
		t.Fatalf("emit: %v", err)
	}

	// Simulate partition: point A at an unreachable address.
	realURL := sa.Client.BaseURL
	sa.Client.BaseURL = "http://127.0.0.1:1"

	// Tick must return an error without panicking or advancing any cursors.
	if err := sa.Tick(); err == nil {
		t.Fatal("expected error during partition, got nil")
	}

	// Event must still be in A's local store (not lost or consumed).
	events, err := a.EventsSince("node01", 0)
	if err != nil {
		t.Fatalf("EventsSince: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("event count during partition: want 1, got %d", len(events))
	}

	// Restore connection and replay the backlog.
	sa.Client.BaseURL = realURL
	if err := sa.Tick(); err != nil {
		t.Fatalf("A reconnect tick: %v", err)
	}
	if err := sb.Tick(); err != nil {
		t.Fatalf("B pull tick: %v", err)
	}

	got, err := b.EventsSince("node01", 0)
	if err != nil {
		t.Fatalf("B read: %v", err)
	}
	if len(got) != 1 || got[0].Type != "player.created" {
		t.Fatalf("event not propagated after reconnect: %+v", got)
	}

	// Idempotency: a third B tick must not duplicate the event.
	if err := sb.Tick(); err != nil {
		t.Fatalf("B tick 2: %v", err)
	}
	again, _ := b.EventsSince("node01", 0)
	if len(again) != 1 {
		t.Fatalf("duplicate applied on B after reconnect: got %d", len(again))
	}
}

// TestEventPropagation is the core B3.2 proof: an event emitted on node A reaches
// node B's event log via push→hub→pull, exactly once.
func TestEventPropagation(t *testing.T) {
	hubStore, err := hub.OpenSQLite(filepath.Join(t.TempDir(), "hub.db"))
	if err != nil {
		t.Fatalf("hub open: %v", err)
	}
	defer hubStore.Close()
	ts := httptest.NewServer(hub.NewServer(hubStore, "regsecret").Handler())
	defer ts.Close()

	a := openNode(t, "node01")
	b := openNode(t, "node02")
	sa := &Syncer{Store: a, Client: register(t, ts.URL, "node01"), NodeID: "node01"}
	sb := &Syncer{Store: b, Client: register(t, ts.URL, "node02"), NodeID: "node02"}

	// Node A emits an event.
	if err := a.Emit("player.created", playerCreatedPayload("node01", "p_x", "Rook")); err != nil {
		t.Fatalf("emit: %v", err)
	}

	// A pushes; B pulls and applies.
	if err := sa.Tick(); err != nil {
		t.Fatalf("A tick: %v", err)
	}
	if err := sb.Tick(); err != nil {
		t.Fatalf("B tick: %v", err)
	}

	got, err := b.EventsSince("node01", 0)
	if err != nil {
		t.Fatalf("B read: %v", err)
	}
	if len(got) != 1 || got[0].Type != "player.created" || got[0].EventID != "node01:1" {
		t.Fatalf("event not propagated to B: %+v", got)
	}

	// Idempotency: a second B tick applies nothing new (cursor advanced).
	if err := sb.Tick(); err != nil {
		t.Fatalf("B tick 2: %v", err)
	}
	if again, _ := b.EventsSince("node01", 0); len(again) != 1 {
		t.Fatalf("duplicate applied on B: %d", len(again))
	}

	// exclude_self: A pulling does not re-receive its own pushed event.
	if err := sa.Tick(); err != nil {
		t.Fatalf("A tick 2: %v", err)
	}
	if own, _ := a.EventsSince("node01", 0); len(own) != 1 {
		t.Fatalf("A self-event count drifted: %d", len(own))
	}
}

func TestPvPResolveFailureLeavesRequestPending(t *testing.T) {
	hubStore, err := hub.OpenSQLite(filepath.Join(t.TempDir(), "hub.db"))
	if err != nil {
		t.Fatalf("hub open: %v", err)
	}
	defer hubStore.Close()
	ts := httptest.NewServer(hub.NewServer(hubStore, "regsecret").Handler())
	defer ts.Close()

	a := openNode(t, "node01")
	b := openNode(t, "node02")
	ca := register(t, ts.URL, "node01")
	cb := register(t, ts.URL, "node02")

	reqID, err := ca.QueuePvP("node01:p_a", "node02:p_v", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("queue pvp: %v", err)
	}
	sb := &Syncer{
		Store:  b,
		Client: cb,
		NodeID: "node02",
		PvPResolve: func(*engine.Store, string, string, string, json.RawMessage) error {
			return errors.New("resolver down")
		},
	}
	if err := sb.Tick(); err != nil {
		t.Fatalf("failed resolver tick should keep syncing: %v", err)
	}
	pending, err := cb.PendingPvP()
	if err != nil {
		t.Fatalf("pending pvp: %v", err)
	}
	if len(pending) != 1 || pending[0].RequestID != reqID {
		t.Fatalf("request should remain pending after resolver failure: %+v", pending)
	}

	sb.PvPResolve = func(*engine.Store, string, string, string, json.RawMessage) error { return nil }
	if err := sb.Tick(); err != nil {
		t.Fatalf("successful resolver tick: %v", err)
	}
	pending, _ = cb.PendingPvP()
	if len(pending) != 0 {
		t.Fatalf("request should resolve after successful resolver: %+v", pending)
	}
	_ = a
}

func TestTravelImportFailureLeavesArrivalPending(t *testing.T) {
	hubStore, err := hub.OpenSQLite(filepath.Join(t.TempDir(), "hub.db"))
	if err != nil {
		t.Fatalf("hub open: %v", err)
	}
	defer hubStore.Close()
	ts := httptest.NewServer(hub.NewServer(hubStore, "regsecret").Handler())
	defer ts.Close()

	openNode(t, "node01")
	b := openNode(t, "node02")
	ca := register(t, ts.URL, "node01")
	cb := register(t, ts.URL, "node02")

	travelID, err := ca.SubmitTravel("node01:p_x", "node01", "node02", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("submit travel: %v", err)
	}
	sb := &Syncer{
		Store:  b,
		Client: cb,
		NodeID: "node02",
		TravelImport: func(*engine.Store, *engine.Snapshot, string, string) error {
			return errors.New("import failed")
		},
	}
	if err := sb.Tick(); err != nil {
		t.Fatalf("failed import tick should keep syncing: %v", err)
	}
	pending, err := cb.PendingTravel()
	if err != nil {
		t.Fatalf("pending travel: %v", err)
	}
	if len(pending) != 1 || pending[0].TravelID != travelID {
		t.Fatalf("arrival should remain pending after import failure: %+v", pending)
	}

	sb.TravelImport = func(*engine.Store, *engine.Snapshot, string, string) error { return nil }
	if err := sb.Tick(); err != nil {
		t.Fatalf("successful import tick: %v", err)
	}
	pending, _ = cb.PendingTravel()
	if len(pending) != 0 {
		t.Fatalf("arrival should resolve after successful import: %+v", pending)
	}
}

func TestCrossNodeDebtAdjustedPropagates(t *testing.T) {
	ts, _ := newFederatedHub(t)
	a, _ := newLedgerNode(t, "node01")
	b, _ := newLedgerNode(t, "node02")
	sa := &Syncer{Store: a, Client: register(t, ts.URL, "node01"), NodeID: "node01"}
	sb := &Syncer{Store: b, Client: register(t, ts.URL, "node02"), NodeID: "node02"}

	if _, err := a.CreateObligation("npc:npc_maren", "node02:p_b", "debt", "remote debt", 30); err != nil {
		t.Fatalf("create obligation: %v", err)
	}
	if err := sa.Tick(); err != nil {
		t.Fatalf("A push debt.created: %v", err)
	}
	if err := sb.Tick(); err != nil {
		t.Fatalf("B pull debt.created: %v", err)
	}
	if got, err := b.DebtLoad("node02:p_b"); err != nil || got != 30 {
		t.Fatalf("B debt after create: got %d err=%v, want 30", got, err)
	}

	if applied, err := a.PayDebt("node02:p_b", 7); err != nil || applied != 7 {
		t.Fatalf("partial payment: applied=%d err=%v", applied, err)
	}
	if err := sa.Tick(); err != nil {
		t.Fatalf("A push debt.adjusted: %v", err)
	}
	if err := sb.Tick(); err != nil {
		t.Fatalf("B pull debt.adjusted: %v", err)
	}
	if got, err := b.DebtLoad("node02:p_b"); err != nil || got != 23 {
		t.Fatalf("B debt after adjustment: got %d err=%v, want 23", got, err)
	}
}

func TestTravelReturnHomeAndDuplicateAcrossNodes(t *testing.T) {
	ts, _ := newFederatedHub(t)
	a, gA := newLedgerNode(t, "node01")
	b, gB := newLedgerNode(t, "node02")
	ca := register(t, ts.URL, "node01")
	cb := register(t, ts.URL, "node02")

	p, err := a.CreateAccount("Traveler", "roadpass")
	if err != nil {
		t.Fatalf("create traveler: %v", err)
	}
	if err := gA.NewCharacter(a.DB(), p); err != nil {
		t.Fatalf("new character: %v", err)
	}
	snap, err := a.ExportPlayer(p.GlobalID, gA)
	if err != nil {
		t.Fatalf("export outbound snapshot: %v", err)
	}
	snapJSON, err := json.Marshal(snap)
	if err != nil {
		t.Fatalf("marshal outbound snapshot: %v", err)
	}
	travelID, err := ca.SubmitTravel(p.GlobalID, p.HomeNode, "node02", snapJSON)
	if err != nil {
		t.Fatalf("submit outbound travel: %v", err)
	}
	if _, err := ca.SubmitTravel(p.GlobalID, p.HomeNode, "node02", snapJSON); err == nil {
		t.Fatalf("duplicate pending travel should be rejected")
	}
	if err := cb.ArriveTravel(travelID); err != nil {
		t.Fatalf("destination arrival: %v", err)
	}
	if err := b.ImportPlayer(snap, gB); err != nil {
		t.Fatalf("import on destination: %v", err)
	}
	if err := a.SetPlayerStatus(p.GlobalID, "traveling"); err != nil {
		t.Fatalf("mark origin traveling: %v", err)
	}
	if _, err := a.Authenticate("Traveler", "roadpass"); err != nil {
		t.Fatalf("credential should still authenticate before server login status gate: %v", err)
	}

	returnSnap, err := b.ExportPlayer(p.GlobalID, gB)
	if err != nil {
		t.Fatalf("export return snapshot: %v", err)
	}
	returnJSON, err := json.Marshal(returnSnap)
	if err != nil {
		t.Fatalf("marshal return snapshot: %v", err)
	}
	if _, err := ca.SubmitTravel(p.GlobalID, p.HomeNode, "node01", returnJSON); err == nil {
		t.Fatalf("home node should not be able to submit return while player is located at node02")
	}
	returnID, err := cb.SubmitTravel(p.GlobalID, p.HomeNode, "node01", returnJSON)
	if err != nil {
		t.Fatalf("submit return travel from current node: %v", err)
	}
	if err := ca.ArriveTravel(returnID); err != nil {
		t.Fatalf("home arrival: %v", err)
	}
	if err := a.ImportPlayer(returnSnap, gA); err != nil {
		t.Fatalf("import return on home: %v", err)
	}
	if err := a.SetPlayerStatus(p.GlobalID, "active"); err != nil {
		t.Fatalf("mark returned active: %v", err)
	}
	returned, err := a.Authenticate("Traveler", "roadpass")
	if err != nil {
		t.Fatalf("returned traveler should authenticate at home: %v", err)
	}
	if returned.Status != "active" {
		t.Fatalf("returned status: want active, got %q", returned.Status)
	}
}

func TestBothNodesReplayBacklogAfterHubPartition(t *testing.T) {
	ts, _ := newFederatedHub(t)
	a := openNode(t, "node01")
	b := openNode(t, "node02")
	sa := &Syncer{Store: a, Client: register(t, ts.URL, "node01"), NodeID: "node01"}
	sb := &Syncer{Store: b, Client: register(t, ts.URL, "node02"), NodeID: "node02"}

	realAURL := sa.Client.BaseURL
	realBURL := sb.Client.BaseURL
	sa.Client.BaseURL = "http://127.0.0.1:1"
	sb.Client.BaseURL = "http://127.0.0.1:1"

	if err := a.Emit("player.created", playerCreatedPayload("node01", "p_a", "A")); err != nil {
		t.Fatalf("emit A: %v", err)
	}
	if err := b.Emit("player.created", playerCreatedPayload("node02", "p_b", "B")); err != nil {
		t.Fatalf("emit B: %v", err)
	}
	if err := sa.Tick(); err == nil {
		t.Fatalf("A tick should fail during partition")
	}
	if err := sb.Tick(); err == nil {
		t.Fatalf("B tick should fail during partition")
	}

	sa.Client.BaseURL = realAURL
	sb.Client.BaseURL = realBURL
	if err := sa.Tick(); err != nil {
		t.Fatalf("A reconnect push: %v", err)
	}
	if err := sb.Tick(); err != nil {
		t.Fatalf("B reconnect push/pull: %v", err)
	}
	if err := sa.Tick(); err != nil {
		t.Fatalf("A reconnect pull: %v", err)
	}

	aGot, err := a.EventsSince("node02", 0)
	if err != nil {
		t.Fatalf("A read remote backlog: %v", err)
	}
	bGot, err := b.EventsSince("node01", 0)
	if err != nil {
		t.Fatalf("B read remote backlog: %v", err)
	}
	if len(aGot) != 1 || aGot[0].EventID != "node02:1" {
		t.Fatalf("A did not receive B backlog after reconnect: %+v", aGot)
	}
	if len(bGot) != 1 || bGot[0].EventID != "node01:1" {
		t.Fatalf("B did not receive A backlog after reconnect: %+v", bGot)
	}
}

func TestRemoteRosterCacheSurvivesPartitionForStaleDisplay(t *testing.T) {
	ts, _ := newFederatedHub(t)
	a := openNode(t, "node01")
	b := openNode(t, "node02")
	sa := &Syncer{Store: a, Client: register(t, ts.URL, "node01"), NodeID: "node01"}
	sb := &Syncer{Store: b, Client: register(t, ts.URL, "node02"), NodeID: "node02"}

	remote, err := b.CreateAccount("RemoteWanderer", "roadpass")
	if err != nil {
		t.Fatalf("create remote player: %v", err)
	}
	oldSeen := time.Now().Add(-20 * time.Minute).Unix()
	if _, err := b.DB().Exec(`UPDATE players SET last_seen=? WHERE global_id=?`, oldSeen, remote.GlobalID); err != nil {
		t.Fatalf("age remote player last_seen: %v", err)
	}

	if err := sb.Tick(); err != nil {
		t.Fatalf("B push roster: %v", err)
	}
	if err := sa.Tick(); err != nil {
		t.Fatalf("A pull roster: %v", err)
	}
	before, err := a.Players("")
	if err != nil {
		t.Fatalf("A players before partition: %v", err)
	}
	if len(before) != 1 || before[0].GlobalID != remote.GlobalID || before[0].HomeNode != "node02" {
		t.Fatalf("A remote roster before partition: %+v", before)
	}
	if before[0].LastSeen.Unix() != oldSeen {
		t.Fatalf("A remote last_seen before partition: got %d want %d", before[0].LastSeen.Unix(), oldSeen)
	}

	realURL := sa.Client.BaseURL
	sa.Client.BaseURL = "http://127.0.0.1:1"
	if err := sa.Tick(); err == nil {
		t.Fatalf("A tick should fail during partition")
	}
	sa.Client.BaseURL = realURL

	after, err := a.Players("")
	if err != nil {
		t.Fatalf("A players after partition: %v", err)
	}
	if len(after) != 1 || after[0].GlobalID != remote.GlobalID || after[0].HomeNode != "node02" {
		t.Fatalf("A remote roster should remain cached after partition: %+v", after)
	}
	if after[0].LastSeen.Unix() != oldSeen {
		t.Fatalf("A remote last_seen after partition: got %d want %d", after[0].LastSeen.Unix(), oldSeen)
	}
	if time.Since(after[0].LastSeen) <= 15*time.Minute {
		t.Fatalf("cached remote roster entry should be old enough for stale display: %s", after[0].LastSeen)
	}
}

func TestDeployedShapeExternalHubPartitionTravelAndDebt(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "hub.db")
	addr := reserveAddr(t)
	baseURL, stopHub := startHubProcess(t, addr, dbPath)

	a, gA := newLedgerNode(t, "node01")
	b, gB := newLedgerNode(t, "node02")
	ca := register(t, baseURL, "node01")
	cb := register(t, baseURL, "node02")
	sa := &Syncer{Store: a, Client: ca, NodeID: "node01"}
	sb := &Syncer{Store: b, Client: cb, NodeID: "node02"}

	p, err := a.CreateAccount("ProcessTraveler", "roadpass")
	if err != nil {
		t.Fatalf("create traveler: %v", err)
	}
	if err := gA.NewCharacter(a.DB(), p); err != nil {
		t.Fatalf("new character: %v", err)
	}
	if _, err := a.CreateObligation("npc:npc_maren", "node02:p_b", "debt", "remote debt", 30); err != nil {
		t.Fatalf("create obligation: %v", err)
	}
	if err := sa.Tick(); err != nil {
		t.Fatalf("A push debt.created: %v", err)
	}
	if err := sb.Tick(); err != nil {
		t.Fatalf("B pull debt.created: %v", err)
	}
	if applied, err := a.PayDebt("node02:p_b", 7); err != nil || applied != 7 {
		t.Fatalf("partial payment: applied=%d err=%v", applied, err)
	}
	if err := sa.Tick(); err != nil {
		t.Fatalf("A push debt.adjusted: %v", err)
	}
	if err := sb.Tick(); err != nil {
		t.Fatalf("B pull debt.adjusted: %v", err)
	}
	if got, err := b.DebtLoad("node02:p_b"); err != nil || got != 23 {
		t.Fatalf("B debt after adjustment: got %d err=%v, want 23", got, err)
	}

	snap, err := a.ExportPlayer(p.GlobalID, gA)
	if err != nil {
		t.Fatalf("export outbound snapshot: %v", err)
	}
	snapJSON, err := json.Marshal(snap)
	if err != nil {
		t.Fatalf("marshal outbound snapshot: %v", err)
	}
	travelID, err := ca.SubmitTravel(p.GlobalID, p.HomeNode, "node02", snapJSON)
	if err != nil {
		t.Fatalf("submit outbound travel: %v", err)
	}
	if _, err := ca.SubmitTravel(p.GlobalID, p.HomeNode, "node02", snapJSON); err == nil {
		t.Fatalf("duplicate pending travel should be rejected")
	}
	if err := a.SetPlayerStatus(p.GlobalID, "traveling"); err != nil {
		t.Fatalf("mark origin traveling: %v", err)
	}
	player, err := a.Authenticate("ProcessTraveler", "roadpass")
	if err != nil {
		t.Fatalf("origin player credential check: %v", err)
	}
	if player.Status != "traveling" {
		t.Fatalf("origin player status: want traveling, got %q", player.Status)
	}
	if err := cb.ArriveTravel(travelID); err != nil {
		t.Fatalf("destination arrival: %v", err)
	}
	if err := b.ImportPlayer(snap, gB); err != nil {
		t.Fatalf("import on destination: %v", err)
	}

	stopHub()
	if err := a.Emit("player.created", playerCreatedPayload("node01", "p_partition_a", "A")); err != nil {
		t.Fatalf("emit A during partition: %v", err)
	}
	if err := b.Emit("player.created", playerCreatedPayload("node02", "p_partition_b", "B")); err != nil {
		t.Fatalf("emit B during partition: %v", err)
	}
	if err := sa.Tick(); err == nil || !strings.Contains(err.Error(), "connect") {
		t.Fatalf("A tick should fail during stopped-hub partition, got %v", err)
	}
	if err := sb.Tick(); err == nil || !strings.Contains(err.Error(), "connect") {
		t.Fatalf("B tick should fail during stopped-hub partition, got %v", err)
	}

	baseURL, stopHub = startHubProcess(t, addr, dbPath)
	defer stopHub()
	sa.Client.BaseURL = baseURL
	sb.Client.BaseURL = baseURL
	if err := sa.Tick(); err != nil {
		t.Fatalf("A reconnect push: %v", err)
	}
	if err := sb.Tick(); err != nil {
		t.Fatalf("B reconnect push/pull: %v", err)
	}
	if err := sa.Tick(); err != nil {
		t.Fatalf("A reconnect pull: %v", err)
	}
	if got, err := a.EventsSince("node02", 0); err != nil || len(got) != 1 || got[0].EventID != "node02:1" {
		t.Fatalf("A did not receive B backlog after process restart: events=%+v err=%v", got, err)
	}
	if got, err := b.EventsSince("node01", 0); err != nil || len(got) < 2 {
		t.Fatalf("B did not retain/pull A events after process restart: events=%+v err=%v", got, err)
	}

	returnSnap, err := b.ExportPlayer(p.GlobalID, gB)
	if err != nil {
		t.Fatalf("export return snapshot: %v", err)
	}
	returnJSON, err := json.Marshal(returnSnap)
	if err != nil {
		t.Fatalf("marshal return snapshot: %v", err)
	}
	if _, err := ca.SubmitTravel(p.GlobalID, p.HomeNode, "node01", returnJSON); err == nil {
		t.Fatalf("home node should not be able to submit return while hub location is node02")
	}
	returnID, err := cb.SubmitTravel(p.GlobalID, p.HomeNode, "node01", returnJSON)
	if err != nil {
		t.Fatalf("submit return travel from current node: %v", err)
	}
	if err := ca.ArriveTravel(returnID); err != nil {
		t.Fatalf("home arrival: %v", err)
	}
	if err := a.ImportPlayer(returnSnap, gA); err != nil {
		t.Fatalf("import return on home: %v", err)
	}
	if err := a.SetPlayerStatus(p.GlobalID, "active"); err != nil {
		t.Fatalf("mark returned active: %v", err)
	}
	returned, err := a.Authenticate("ProcessTraveler", "roadpass")
	if err != nil {
		t.Fatalf("returned traveler credential check: %v", err)
	}
	if returned.Status != "active" {
		t.Fatalf("returned traveler status: want active, got %q", returned.Status)
	}
}
