package main

import (
	"bytes"
	"database/sql"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"interdoor.net/interdoor/internal/hub"

	_ "modernc.org/sqlite"
)

func testDB(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "hub.db")
	store, err := hub.OpenSQLite(path)
	if err != nil {
		t.Fatalf("open hub db: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close hub db: %v", err)
	}
	old := *dbPath
	*dbPath = path
	t.Cleanup(func() { *dbPath = old })
	return path
}

func openSQL(t *testing.T, path string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", path+"?_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatalf("open sql: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Fatalf("ping sql: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func seedNode(t *testing.T, db *sql.DB, nodeID string) {
	t.Helper()
	now := time.Now().Unix()
	_, err := db.Exec(
		`INSERT INTO nodes(node_id,api_key_hash,game_id,game_version,protocol_version,advertise_addr,player_count,uptime_s,last_heartbeat,status,created_at)
		 VALUES(?,?,?,?,?,?,?,?,?,?,?)`,
		nodeID, strings.Repeat(nodeID, 8), "ledger_of_the_low", "1.0.0", "1", "ssh://"+nodeID+":2323", 1, 10, now, "active", now,
	)
	if err != nil {
		t.Fatalf("seed node %s: %v", nodeID, err)
	}
}

func countRows(t *testing.T, db *sql.DB, q string, args ...any) int {
	t.Helper()
	var n int
	if err := db.QueryRow(q, args...).Scan(&n); err != nil {
		t.Fatalf("count rows: %v", err)
	}
	return n
}

func auditCount(t *testing.T, db *sql.DB, command string) int {
	t.Helper()
	var n int
	err := db.QueryRow(`SELECT COUNT(*) FROM hub_admin_audit WHERE command=?`, command).Scan(&n)
	if err != nil {
		t.Fatalf("count audit rows for %s: %v", command, err)
	}
	return n
}

func captureStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	err = fn()
	_ = w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	_ = r.Close()
	return buf.String(), err
}

func TestNodeRemoveDryRunAndExecute(t *testing.T) {
	path := testDB(t)
	db := openSQL(t, path)
	seedNode(t, db, "node01")
	seedNode(t, db, "node02")
	now := time.Now().Unix()
	_, err := db.Exec(`
		INSERT INTO roster(global_id,node_id,name,level,status,last_seen,updated_at) VALUES('node01:p_1','node01','A',1,'active',?,?);
		INSERT INTO events(event_id,source_node,seq,type,ts,payload) VALUES('node01:1','node01',1,'player.created',?, '{}');
		INSERT INTO debts(obligation_id,source_node,creditor_ref,debtor_ref,kind,terms,weight,status,updated_at) VALUES('node01:o_1','node01','npc:maren','node02:p_1','debt','terms',5,'open',?);
		INSERT INTO pvp_requests(request_id,attacker_id,victim_id,victim_node,attacker_payload,status,created_at) VALUES('pvp1','node02:p_a','node01:p_1','node01','{}','pending',?);
		INSERT INTO travel(travel_id,global_id,home_node,from_node,dest_node,snapshot,status,created_at) VALUES('travel1','node01:p_1','node01','node01','node02','{}','pending',?);
		INSERT INTO player_locations(global_id,current_node,home_node,status,travel_id,updated_at) VALUES('node01:p_1','node01','node01','traveling','travel1',?);
	`, now, now, now, now, now, now, now)
	if err != nil {
		t.Fatalf("seed dependent rows: %v", err)
	}

	if _, err := captureStdout(t, func() error { return nodeRemove([]string{"node01"}) }); err != nil {
		t.Fatalf("dry-run node remove: %v", err)
	}
	if got := countRows(t, db, `SELECT COUNT(*) FROM nodes WHERE node_id='node01'`); got != 1 {
		t.Fatalf("dry-run removed node rows, got %d", got)
	}
	if got := countRows(t, db, `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='hub_admin_audit'`); got != 0 {
		t.Fatalf("dry-run should not create audit table, got %d", got)
	}

	if _, err := captureStdout(t, func() error { return nodeRemove([]string{"--execute", "node01"}) }); err != nil {
		t.Fatalf("execute node remove: %v", err)
	}
	for _, check := range []struct {
		name string
		q    string
	}{
		{"nodes", `SELECT COUNT(*) FROM nodes WHERE node_id='node01'`},
		{"roster", `SELECT COUNT(*) FROM roster WHERE node_id='node01'`},
		{"events", `SELECT COUNT(*) FROM events WHERE source_node='node01'`},
		{"debts", `SELECT COUNT(*) FROM debts WHERE source_node='node01'`},
		{"pvp", `SELECT COUNT(*) FROM pvp_requests WHERE victim_node='node01'`},
		{"travel", `SELECT COUNT(*) FROM travel WHERE from_node='node01' OR dest_node='node01' OR home_node='node01'`},
		{"locations", `SELECT COUNT(*) FROM player_locations WHERE current_node='node01' OR home_node='node01'`},
	} {
		if got := countRows(t, db, check.q); got != 0 {
			t.Fatalf("%s rows remain after node remove: %d", check.name, got)
		}
	}
	if got := countRows(t, db, `SELECT COUNT(*) FROM nodes WHERE node_id='node02'`); got != 1 {
		t.Fatalf("unrelated node should remain, got %d", got)
	}
	if got := auditCount(t, db, "node-remove"); got != 1 {
		t.Fatalf("node-remove audit rows: want 1, got %d", got)
	}
}

func TestNodeTitleDryRunAndExecute(t *testing.T) {
	path := testDB(t)
	db := openSQL(t, path)
	seedNode(t, db, "thelow")

	out, err := captureStdout(t, func() error { return nodeTitle([]string{"thelow", "Ledger", "of", "the", "Low"}) })
	if err != nil {
		t.Fatalf("dry-run node title: %v", err)
	}
	if !strings.Contains(out, `dry-run: would set node "thelow" game_title to "Ledger of the Low"`) {
		t.Fatalf("unexpected dry-run output:\n%s", out)
	}
	var title string
	if err := db.QueryRow(`SELECT game_title FROM nodes WHERE node_id='thelow'`).Scan(&title); err != nil {
		t.Fatalf("select game_title: %v", err)
	}
	if title != "" {
		t.Fatalf("dry-run changed title to %q", title)
	}

	if _, err := captureStdout(t, func() error { return nodeTitle([]string{"--execute", "thelow", "Ledger of the Low"}) }); err != nil {
		t.Fatalf("execute node title: %v", err)
	}
	if err := db.QueryRow(`SELECT game_title FROM nodes WHERE node_id='thelow'`).Scan(&title); err != nil {
		t.Fatalf("select updated game_title: %v", err)
	}
	if title != "Ledger of the Low" {
		t.Fatalf("game_title got %q", title)
	}
	if got := auditCount(t, db, "node-title"); got != 1 {
		t.Fatalf("audit node-title count got %d", got)
	}
}

func TestQueuesFilterAndBackupRestoreDrill(t *testing.T) {
	path := testDB(t)
	db := openSQL(t, path)
	seedNode(t, db, "node01")
	seedNode(t, db, "node02")
	now := time.Now().Unix()
	_, err := db.Exec(`
		INSERT INTO pvp_requests(request_id,attacker_id,victim_id,victim_node,attacker_payload,status,created_at) VALUES('pvp-node01','node02:p_a','node01:p_1','node01','{}','pending',?);
		INSERT INTO pvp_requests(request_id,attacker_id,victim_id,victim_node,attacker_payload,status,created_at) VALUES('pvp-node02','node01:p_a','node02:p_1','node02','{}','pending',?);
		INSERT INTO travel(travel_id,global_id,home_node,from_node,dest_node,snapshot,status,created_at) VALUES('travel-node01','node01:p_1','node01','node01','node02','{}','pending',?);
		INSERT INTO travel(travel_id,global_id,home_node,from_node,dest_node,snapshot,status,created_at) VALUES('travel-node02','node02:p_1','node02','node02','node03','{}','pending',?);
	`, now, now, now, now)
	if err != nil {
		t.Fatalf("seed queues: %v", err)
	}

	out, err := captureStdout(t, func() error { return queues([]string{"--node", "node01"}) })
	if err != nil {
		t.Fatalf("queues: %v", err)
	}
	for _, want := range []string{"pvp-node01", "travel-node01"} {
		if !strings.Contains(out, want) {
			t.Fatalf("queue output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "pvp-node02") || strings.Contains(out, "travel-node02") {
		t.Fatalf("queue filter included unrelated rows:\n%s", out)
	}

	outPath := filepath.Join(t.TempDir(), "backup.db")
	if _, err := captureStdout(t, func() error { return backup([]string{outPath}) }); err != nil {
		t.Fatalf("backup: %v", err)
	}
	info, err := os.Stat(outPath)
	if err != nil {
		t.Fatalf("backup missing: %v", err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("backup mode: want 0600, got %o", got)
	}
	backupDB := openSQL(t, outPath)
	if got := countRows(t, backupDB, `SELECT COUNT(*) FROM nodes`); got != 2 {
		t.Fatalf("backup node count: want 2, got %d", got)
	}
	if got := countRows(t, backupDB, `SELECT COUNT(*) FROM pvp_requests`); got != 2 {
		t.Fatalf("backup pvp count: want 2, got %d", got)
	}
	if got := countRows(t, backupDB, `SELECT COUNT(*) FROM travel`); got != 2 {
		t.Fatalf("backup travel count: want 2, got %d", got)
	}
	if got := auditCount(t, db, "backup"); got != 1 {
		t.Fatalf("backup audit rows: want 1, got %d", got)
	}
}

func TestQueueRetryDryRunAndExecute(t *testing.T) {
	path := testDB(t)
	db := openSQL(t, path)
	seedNode(t, db, "node01")
	now := time.Now().Unix()
	_, err := db.Exec(`
		INSERT INTO pvp_requests(request_id,attacker_id,victim_id,victim_node,attacker_payload,status,error,created_at,updated_at) VALUES('pvp-blocked','node02:p_a','node01:p_1','node01','{}','blocked','bad attacker payload',?,?);
		INSERT INTO travel(travel_id,global_id,home_node,from_node,dest_node,snapshot,status,error,created_at,updated_at) VALUES('travel-blocked','node02:p_1','node02','node02','node01','{}','blocked','bad snapshot',?,?);
	`, now, now, now, now)
	if err != nil {
		t.Fatalf("seed blocked queues: %v", err)
	}

	out, err := captureStdout(t, func() error { return queues([]string{"--node", "node01"}) })
	if err != nil {
		t.Fatalf("queues: %v", err)
	}
	for _, want := range []string{`pvp-blocked`, `error="bad attacker payload"`, `travel-blocked`, `error="bad snapshot"`} {
		if !strings.Contains(out, want) {
			t.Fatalf("queue output missing %q:\n%s", want, out)
		}
	}

	if _, err := captureStdout(t, func() error { return queueRetry([]string{"pvp", "pvp-blocked"}) }); err != nil {
		t.Fatalf("dry-run queue-retry: %v", err)
	}
	var status, errText string
	if err := db.QueryRow(`SELECT status,error FROM pvp_requests WHERE request_id='pvp-blocked'`).Scan(&status, &errText); err != nil {
		t.Fatalf("read dry-run pvp row: %v", err)
	}
	if status != "blocked" || errText != "bad attacker payload" {
		t.Fatalf("dry-run changed pvp row: status=%q error=%q", status, errText)
	}

	if _, err := captureStdout(t, func() error { return queueRetry([]string{"--execute", "pvp", "pvp-blocked"}) }); err != nil {
		t.Fatalf("execute pvp queue-retry: %v", err)
	}
	if err := db.QueryRow(`SELECT status,error FROM pvp_requests WHERE request_id='pvp-blocked'`).Scan(&status, &errText); err != nil {
		t.Fatalf("read retried pvp row: %v", err)
	}
	if status != "pending" || errText != "" {
		t.Fatalf("execute did not reset pvp row: status=%q error=%q", status, errText)
	}

	if _, err := captureStdout(t, func() error { return queueRetry([]string{"--execute", "travel", "travel-blocked"}) }); err != nil {
		t.Fatalf("execute travel queue-retry: %v", err)
	}
	if err := db.QueryRow(`SELECT status,error FROM travel WHERE travel_id='travel-blocked'`).Scan(&status, &errText); err != nil {
		t.Fatalf("read retried travel row: %v", err)
	}
	if status != "pending" || errText != "" {
		t.Fatalf("execute did not reset travel row: status=%q error=%q", status, errText)
	}
	if got := auditCount(t, db, "queue-retry"); got != 2 {
		t.Fatalf("queue-retry audit rows: want 2, got %d", got)
	}
	if _, err := captureStdout(t, func() error { return queueRetry([]string{"--execute", "pvp", "pvp-blocked"}) }); err == nil {
		t.Fatalf("retrying non-blocked pvp should fail")
	}
}

func TestQueuesSupportsLegacyQueueSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy-hub.db")
	db := openSQL(t, path)
	now := time.Now().Unix()
	_, err := db.Exec(`
		CREATE TABLE pvp_requests (
			request_id TEXT PRIMARY KEY,
			attacker_id TEXT NOT NULL,
			victim_id TEXT NOT NULL,
			victim_node TEXT NOT NULL,
			attacker_payload TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			created_at INTEGER NOT NULL
		);
		CREATE TABLE travel (
			travel_id TEXT PRIMARY KEY,
			global_id TEXT NOT NULL,
			home_node TEXT NOT NULL,
			from_node TEXT NOT NULL,
			dest_node TEXT NOT NULL,
			snapshot TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			created_at INTEGER NOT NULL
		);
		INSERT INTO pvp_requests(request_id,attacker_id,victim_id,victim_node,attacker_payload,status,created_at)
			VALUES('legacy-pvp','node02:p_a','node01:p_1','node01','{}','pending',?);
		INSERT INTO travel(travel_id,global_id,home_node,from_node,dest_node,snapshot,status,created_at)
			VALUES('legacy-travel','node01:p_1','node01','node01','node02','{}','pending',?);
	`, now, now)
	if err != nil {
		t.Fatalf("seed legacy queue schema: %v", err)
	}
	old := *dbPath
	*dbPath = path
	t.Cleanup(func() { *dbPath = old })

	out, err := captureStdout(t, func() error { return queues([]string{"--node", "node01"}) })
	if err != nil {
		t.Fatalf("legacy queues: %v", err)
	}
	for _, want := range []string{"legacy-pvp", "legacy-travel"} {
		if !strings.Contains(out, want) {
			t.Fatalf("legacy queue output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "error=") {
		t.Fatalf("legacy queue output should not show empty error text:\n%s", out)
	}
	if _, err := captureStdout(t, func() error { return queueRetry([]string{"pvp", "legacy-pvp"}) }); err == nil {
		t.Fatalf("legacy queue-retry should require migrated schema")
	}
}

func TestRotateKeyAndRegistrationTokenLifecycle(t *testing.T) {
	path := testDB(t)
	db := openSQL(t, path)
	seedNode(t, db, "node01")
	var oldHash string
	if err := db.QueryRow(`SELECT api_key_hash FROM nodes WHERE node_id='node01'`).Scan(&oldHash); err != nil {
		t.Fatalf("read old hash: %v", err)
	}

	if _, err := captureStdout(t, func() error { return rotateKey([]string{"node01"}) }); err != nil {
		t.Fatalf("dry-run rotate-key: %v", err)
	}
	var dryHash string
	_ = db.QueryRow(`SELECT api_key_hash FROM nodes WHERE node_id='node01'`).Scan(&dryHash)
	if dryHash != oldHash {
		t.Fatalf("dry-run changed api hash")
	}
	if got := countRows(t, db, `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='hub_admin_audit'`); got != 0 {
		t.Fatalf("dry-run rotate should not create audit table, got %d", got)
	}
	if _, err := captureStdout(t, func() error { return rotateKey([]string{"--execute", "node01"}) }); err != nil {
		t.Fatalf("execute rotate-key: %v", err)
	}
	var newHash string
	_ = db.QueryRow(`SELECT api_key_hash FROM nodes WHERE node_id='node01'`).Scan(&newHash)
	if newHash == "" || newHash == oldHash {
		t.Fatalf("rotate-key did not replace hash: old=%q new=%q", oldHash, newHash)
	}
	if got := auditCount(t, db, "rotate-key"); got != 1 {
		t.Fatalf("rotate-key audit rows: want 1, got %d", got)
	}

	tokenFile := filepath.Join(t.TempDir(), "reg-token.txt")
	if _, err := captureStdout(t, func() error {
		return tokenIssue([]string{"--file", tokenFile})
	}); err != nil {
		t.Fatalf("dry-run token issue: %v", err)
	}
	if _, err := os.Stat(tokenFile); !os.IsNotExist(err) {
		t.Fatalf("dry-run token issue created file")
	}
	if _, err := captureStdout(t, func() error {
		return tokenIssue([]string{"--file", tokenFile, "--execute"})
	}); err != nil {
		t.Fatalf("execute token issue: %v", err)
	}
	token, err := os.ReadFile(tokenFile)
	if err != nil {
		t.Fatalf("read token: %v", err)
	}
	if len(strings.TrimSpace(string(token))) == 0 {
		t.Fatalf("token file is empty after issue")
	}
	info, err := os.Stat(tokenFile)
	if err != nil {
		t.Fatalf("stat token: %v", err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("token mode: want 0600, got %o", got)
	}
	if _, err := captureStdout(t, func() error {
		return tokenRevoke([]string{"--file", tokenFile, "--execute"})
	}); err != nil {
		t.Fatalf("token revoke: %v", err)
	}
	revoked, err := os.ReadFile(tokenFile)
	if err != nil {
		t.Fatalf("read revoked token: %v", err)
	}
	if len(revoked) != 0 {
		t.Fatalf("token file should be empty after revoke, got %q", revoked)
	}
}

func TestNodeStatusWritesAuditOnExecute(t *testing.T) {
	path := testDB(t)
	db := openSQL(t, path)
	seedNode(t, db, "node01")

	if _, err := captureStdout(t, func() error { return nodeStatus([]string{"node01"}, "suspended") }); err != nil {
		t.Fatalf("dry-run node status: %v", err)
	}
	var status string
	if err := db.QueryRow(`SELECT status FROM nodes WHERE node_id='node01'`).Scan(&status); err != nil {
		t.Fatalf("read status: %v", err)
	}
	if status != "active" {
		t.Fatalf("dry-run changed status to %q", status)
	}

	if _, err := captureStdout(t, func() error { return nodeStatus([]string{"--execute", "node01"}, "suspended") }); err != nil {
		t.Fatalf("execute suspend: %v", err)
	}
	if err := db.QueryRow(`SELECT status FROM nodes WHERE node_id='node01'`).Scan(&status); err != nil {
		t.Fatalf("read status after suspend: %v", err)
	}
	if status != "suspended" {
		t.Fatalf("status after suspend: want suspended, got %q", status)
	}
	if got := auditCount(t, db, "node-suspended"); got != 1 {
		t.Fatalf("node-suspended audit rows: want 1, got %d", got)
	}
}
