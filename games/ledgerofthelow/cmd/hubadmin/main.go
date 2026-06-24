// Command interdoor-hub-admin provides local Phase 1 hub operator tooling.
package main

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

var dbPath = flag.String("db", "/var/lib/interdoor-hub/hub.db", "hub SQLite database path")

const auditSchema = `
CREATE TABLE IF NOT EXISTS hub_admin_audit (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at INTEGER NOT NULL,
    operator   TEXT NOT NULL,
    command    TEXT NOT NULL,
    target     TEXT NOT NULL,
    detail     TEXT NOT NULL
);`

func main() {
	flag.Usage = usage
	flag.Parse()
	if flag.NArg() < 1 {
		usage()
		os.Exit(2)
	}

	cmd := flag.Arg(0)
	args := flag.Args()[1:]
	var err error
	switch cmd {
	case "nodes":
		err = withDB(func(db *sql.DB) error { return listNodes(db) })
	case "node-suspend":
		err = nodeStatus(args, "suspended")
	case "node-activate":
		err = nodeStatus(args, "active")
	case "node-title":
		err = nodeTitle(args)
	case "node-remove":
		err = nodeRemove(args)
	case "queues":
		err = queues(args)
	case "queue-retry":
		err = queueRetry(args)
	case "backup":
		err = backup(args)
	case "rotate-key":
		err = rotateKey(args)
	case "token-issue":
		err = tokenIssue(args)
	case "token-revoke":
		err = tokenRevoke(args)
	default:
		err = fmt.Errorf("unknown command %q", cmd)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintf(flag.CommandLine.Output(), `Usage:
  interdoor-hub-admin -db /var/lib/interdoor-hub/hub.db nodes
  interdoor-hub-admin -db /var/lib/interdoor-hub/hub.db node-suspend [--execute] NODE_ID
  interdoor-hub-admin -db /var/lib/interdoor-hub/hub.db node-activate [--execute] NODE_ID
  interdoor-hub-admin -db /var/lib/interdoor-hub/hub.db node-title [--execute] NODE_ID "Game Title"
  interdoor-hub-admin -db /var/lib/interdoor-hub/hub.db node-remove [--execute] NODE_ID
  interdoor-hub-admin -db /var/lib/interdoor-hub/hub.db queues [--node NODE_ID]
  interdoor-hub-admin -db /var/lib/interdoor-hub/hub.db queue-retry [--execute] pvp|travel ID
  interdoor-hub-admin -db /var/lib/interdoor-hub/hub.db backup OUT.db
  interdoor-hub-admin -db /var/lib/interdoor-hub/hub.db rotate-key [--execute] NODE_ID
  interdoor-hub-admin token-issue --file /etc/interdoor-hub/reg-token.txt --execute
  interdoor-hub-admin token-revoke --file /etc/interdoor-hub/reg-token.txt --execute

Mutating commands default to dry-run. Pass --execute to apply changes.
`)
}

func withDB(fn func(*sql.DB) error) error {
	db, err := sql.Open("sqlite", *dbPath+"?_pragma=busy_timeout(5000)")
	if err != nil {
		return err
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		return err
	}
	return fn(db)
}

func listNodes(db *sql.DB) error {
	titleExpr := "game_id"
	if ok, err := columnExists(db, "nodes", "game_title"); err != nil {
		return err
	} else if ok {
		titleExpr = "COALESCE(NULLIF(game_title,''),game_id)"
	}
	rows, err := db.Query(`SELECT node_id,game_id,` + titleExpr + `,game_version,advertise_addr,status,player_count,last_heartbeat FROM nodes ORDER BY node_id`)
	if err != nil {
		return err
	}
	defer rows.Close()
	fmt.Println("node_id\tstatus\tgame_id\tgame_title\tversion\tplayers\tlast_heartbeat\tadvertise_addr")
	for rows.Next() {
		var nodeID, gameID, gameTitle, version, advertise, status string
		var players int
		var heartbeat int64
		if err := rows.Scan(&nodeID, &gameID, &gameTitle, &version, &advertise, &status, &players, &heartbeat); err != nil {
			return err
		}
		fmt.Printf("%s\t%s\t%s\t%s\t%s\t%d\t%s\t%s\n", nodeID, status, gameID, gameTitle, version, players, formatUnix(heartbeat), advertise)
	}
	return rows.Err()
}

func nodeStatus(args []string, status string) error {
	fs := flag.NewFlagSet("node-status", flag.ContinueOnError)
	execute := fs.Bool("execute", false, "apply the change")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("node id required")
	}
	nodeID := fs.Arg(0)
	return withDB(func(db *sql.DB) error {
		if !*execute {
			fmt.Printf("dry-run: would set node %q status to %q\n", nodeID, status)
			return describeNode(db, nodeID)
		}
		res, err := db.Exec(`UPDATE nodes SET status=? WHERE node_id=?`, status, nodeID)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return fmt.Errorf("node %q not found", nodeID)
		}
		if err := auditAction(db, "node-"+status, nodeID, "status="+status); err != nil {
			return err
		}
		fmt.Printf("node %q status set to %q\n", nodeID, status)
		return nil
	})
}

func nodeTitle(args []string) error {
	fs := flag.NewFlagSet("node-title", flag.ContinueOnError)
	execute := fs.Bool("execute", false, "apply the title")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() < 2 {
		return errors.New("node id and game title required")
	}
	nodeID := fs.Arg(0)
	title := strings.TrimSpace(strings.Join(fs.Args()[1:], " "))
	if title == "" {
		return errors.New("game title cannot be empty")
	}
	return withDB(func(db *sql.DB) error {
		ok, err := columnExists(db, "nodes", "game_title")
		if err != nil {
			return err
		}
		if !ok {
			return errors.New("nodes.game_title column is not migrated; deploy/current hub migration before node-title")
		}
		if !*execute {
			fmt.Printf("dry-run: would set node %q game_title to %q\n", nodeID, title)
			return describeNode(db, nodeID)
		}
		res, err := db.Exec(`UPDATE nodes SET game_title=? WHERE node_id=?`, title, nodeID)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return fmt.Errorf("node %q not found", nodeID)
		}
		if err := auditAction(db, "node-title", nodeID, "game_title="+title); err != nil {
			return err
		}
		fmt.Printf("node %q game_title set to %q\n", nodeID, title)
		return nil
	})
}

func describeNode(db *sql.DB, nodeID string) error {
	titleExpr := "game_id"
	if ok, err := columnExists(db, "nodes", "game_title"); err != nil {
		return err
	} else if ok {
		titleExpr = "COALESCE(NULLIF(game_title,''),game_id)"
	}
	var gameID, gameTitle, version, advertise, status string
	var players int
	var heartbeat int64
	err := db.QueryRow(
		`SELECT game_id,`+titleExpr+`,game_version,advertise_addr,status,player_count,last_heartbeat FROM nodes WHERE node_id=?`,
		nodeID,
	).Scan(&gameID, &gameTitle, &version, &advertise, &status, &players, &heartbeat)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("node %q not found", nodeID)
		}
		return err
	}
	fmt.Printf("node_id=%s status=%s game_id=%s game_title=%q version=%s players=%d last_heartbeat=%s advertise_addr=%s\n",
		nodeID, status, gameID, gameTitle, version, players, formatUnix(heartbeat), advertise)
	return nil
}

func nodeRemove(args []string) error {
	fs := flag.NewFlagSet("node-remove", flag.ContinueOnError)
	execute := fs.Bool("execute", false, "delete rows for this node")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("node id required")
	}
	nodeID := fs.Arg(0)
	return withDB(func(db *sql.DB) error {
		counts, err := removalCounts(db, nodeID)
		if err != nil {
			return err
		}
		fmt.Printf("affected rows for %q:\n", nodeID)
		for _, c := range counts {
			fmt.Printf("  %-16s %d\n", c.name, c.count)
		}
		if !*execute {
			fmt.Println("dry-run: no rows deleted; run again with --execute after taking a backup")
			return nil
		}
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()
		stmts := []string{
			`DELETE FROM roster WHERE node_id=?`,
			`DELETE FROM player_locations WHERE current_node=? OR home_node=?`,
			`DELETE FROM pvp_requests WHERE victim_node=?`,
			`DELETE FROM travel WHERE from_node=? OR dest_node=? OR home_node=?`,
			`DELETE FROM debts WHERE source_node=?`,
			`DELETE FROM events WHERE source_node=?`,
			`DELETE FROM nodes WHERE node_id=?`,
		}
		argsFor := [][]any{
			{nodeID}, {nodeID, nodeID}, {nodeID}, {nodeID, nodeID, nodeID}, {nodeID}, {nodeID}, {nodeID},
		}
		for i, stmt := range stmts {
			if _, err := tx.Exec(stmt, argsFor[i]...); err != nil {
				return err
			}
		}
		if err := auditActionTx(tx, "node-remove", nodeID, formatCounts(counts)); err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		fmt.Printf("node %q removed from hub tables\n", nodeID)
		return nil
	})
}

type rowCount struct {
	name  string
	count int
}

func removalCounts(db *sql.DB, nodeID string) ([]rowCount, error) {
	checks := []struct {
		name string
		sql  string
		args []any
	}{
		{"nodes", `SELECT COUNT(*) FROM nodes WHERE node_id=?`, []any{nodeID}},
		{"roster", `SELECT COUNT(*) FROM roster WHERE node_id=?`, []any{nodeID}},
		{"events", `SELECT COUNT(*) FROM events WHERE source_node=?`, []any{nodeID}},
		{"debts", `SELECT COUNT(*) FROM debts WHERE source_node=?`, []any{nodeID}},
		{"pvp_pending", `SELECT COUNT(*) FROM pvp_requests WHERE victim_node=? AND status='pending'`, []any{nodeID}},
		{"travel_pending", `SELECT COUNT(*) FROM travel WHERE (from_node=? OR dest_node=? OR home_node=?) AND status='pending'`, []any{nodeID, nodeID, nodeID}},
		{"locations", `SELECT COUNT(*) FROM player_locations WHERE current_node=? OR home_node=?`, []any{nodeID, nodeID}},
	}
	out := make([]rowCount, 0, len(checks))
	for _, c := range checks {
		var n int
		if err := db.QueryRow(c.sql, c.args...).Scan(&n); err != nil {
			return nil, err
		}
		out = append(out, rowCount{c.name, n})
	}
	return out, nil
}

func queues(args []string) error {
	fs := flag.NewFlagSet("queues", flag.ContinueOnError)
	nodeID := fs.String("node", "", "filter by node id")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return withDB(func(db *sql.DB) error {
		if err := listPvPQueue(db, *nodeID); err != nil {
			return err
		}
		return listTravelQueue(db, *nodeID)
	})
}

func listPvPQueue(db *sql.DB, nodeID string) error {
	errorExpr := `''`
	if ok, err := columnExists(db, "pvp_requests", "error"); err != nil {
		return err
	} else if ok {
		errorExpr = "error"
	}
	q := `SELECT request_id,attacker_id,victim_id,victim_node,status,` + errorExpr + `,created_at FROM pvp_requests`
	args := []any{}
	if nodeID != "" {
		q += ` WHERE victim_node=?`
		args = append(args, nodeID)
	}
	q += ` ORDER BY created_at`
	rows, err := db.Query(q, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	fmt.Println("pvp_requests:")
	for rows.Next() {
		var id, attacker, victim, victimNode, status, errText string
		var created int64
		if err := rows.Scan(&id, &attacker, &victim, &victimNode, &status, &errText, &created); err != nil {
			return err
		}
		fmt.Printf("  %s status=%s age=%s victim_node=%s attacker=%s victim=%s%s\n",
			id, status, age(created), victimNode, attacker, victim, formatError(errText))
	}
	return rows.Err()
}

func listTravelQueue(db *sql.DB, nodeID string) error {
	errorExpr := `''`
	if ok, err := columnExists(db, "travel", "error"); err != nil {
		return err
	} else if ok {
		errorExpr = "error"
	}
	q := `SELECT travel_id,global_id,home_node,from_node,dest_node,status,` + errorExpr + `,created_at FROM travel`
	args := []any{}
	if nodeID != "" {
		q += ` WHERE from_node=? OR dest_node=? OR home_node=?`
		args = append(args, nodeID, nodeID, nodeID)
	}
	q += ` ORDER BY created_at`
	rows, err := db.Query(q, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	fmt.Println("travel:")
	for rows.Next() {
		var id, globalID, homeNode, fromNode, destNode, status, errText string
		var created int64
		if err := rows.Scan(&id, &globalID, &homeNode, &fromNode, &destNode, &status, &errText, &created); err != nil {
			return err
		}
		fmt.Printf("  %s status=%s age=%s player=%s from=%s dest=%s home=%s%s\n",
			id, status, age(created), globalID, fromNode, destNode, homeNode, formatError(errText))
	}
	return rows.Err()
}

func queueRetry(args []string) error {
	fs := flag.NewFlagSet("queue-retry", flag.ContinueOnError)
	execute := fs.Bool("execute", false, "reset a blocked queue item to pending")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 2 {
		return errors.New("queue type and id required")
	}
	queueType, id := fs.Arg(0), fs.Arg(1)
	if queueType != "pvp" && queueType != "travel" {
		return errors.New("queue type must be pvp or travel")
	}
	return withDB(func(db *sql.DB) error {
		table := "pvp_requests"
		idCol := "request_id"
		if queueType == "travel" {
			table = "travel"
			idCol = "travel_id"
		}
		for _, col := range []string{"error", "updated_at"} {
			ok, err := columnExists(db, table, col)
			if err != nil {
				return err
			}
			if !ok {
				return fmt.Errorf("%s queue schema is not migrated; deploy/current hub migration before queue-retry", queueType)
			}
		}
		var status, errText string
		q := fmt.Sprintf(`SELECT status,error FROM %s WHERE %s=?`, table, idCol)
		if err := db.QueryRow(q, id).Scan(&status, &errText); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("%s queue item %q not found", queueType, id)
			}
			return err
		}
		if !*execute {
			fmt.Printf("dry-run: would retry %s queue item %q status=%s%s\n", queueType, id, status, formatError(errText))
			return nil
		}
		if status != "blocked" {
			return fmt.Errorf("%s queue item %q is %q, not blocked", queueType, id, status)
		}
		stmt := fmt.Sprintf(`UPDATE %s SET status='pending', error='', updated_at=? WHERE %s=? AND status='blocked'`, table, idCol)
		res, err := db.Exec(stmt, time.Now().Unix(), id)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return fmt.Errorf("%s queue item %q not retried", queueType, id)
		}
		if err := auditAction(db, "queue-retry", queueType+":"+id, "status=pending"); err != nil {
			return err
		}
		fmt.Printf("%s queue item %q reset to pending\n", queueType, id)
		return nil
	})
}

func columnExists(db *sql.DB, table, column string) (bool, error) {
	rows, err := db.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, typ string
		var notNull int
		var def any
		var pk int
		if err := rows.Scan(&cid, &name, &typ, &notNull, &def, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}

func backup(args []string) error {
	if len(args) != 1 {
		return errors.New("backup output path required")
	}
	out := args[0]
	if _, err := os.Stat(out); err == nil {
		return fmt.Errorf("backup output already exists: %s", out)
	}
	if err := os.MkdirAll(filepath.Dir(out), 0750); err != nil {
		return err
	}
	return withDB(func(db *sql.DB) error {
		if _, err := db.Exec("VACUUM INTO '" + sqlQuote(out) + "'"); err != nil {
			return err
		}
		if err := os.Chmod(out, 0600); err != nil {
			return err
		}
		if err := auditAction(db, "backup", out, "vacuum-into"); err != nil {
			return err
		}
		fmt.Printf("backup written to %s\n", out)
		return nil
	})
}

func rotateKey(args []string) error {
	fs := flag.NewFlagSet("rotate-key", flag.ContinueOnError)
	execute := fs.Bool("execute", false, "apply the key rotation")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("node id required")
	}
	nodeID := fs.Arg(0)
	return withDB(func(db *sql.DB) error {
		if !*execute {
			fmt.Printf("dry-run: would rotate API key for node %q; old key would stop authenticating immediately\n", nodeID)
			return describeNode(db, nodeID)
		}
		key, hash, err := newSecret()
		if err != nil {
			return err
		}
		res, err := db.Exec(`UPDATE nodes SET api_key_hash=? WHERE node_id=?`, hash, nodeID)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return fmt.Errorf("node %q not found", nodeID)
		}
		if err := auditAction(db, "rotate-key", nodeID, "api-key-hash-rotated"); err != nil {
			return err
		}
		fmt.Printf("new API key for %s:\n%s\n", nodeID, key)
		fmt.Println("store this in the node database/configuration before restarting the node")
		return nil
	})
}

func tokenIssue(args []string) error {
	fs := flag.NewFlagSet("token-issue", flag.ContinueOnError)
	file := fs.String("file", "", "registration token file")
	execute := fs.Bool("execute", false, "write the token file")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *file == "" {
		return errors.New("--file is required")
	}
	token, _, err := newSecret()
	if err != nil {
		return err
	}
	if !*execute {
		fmt.Printf("dry-run: would write a new registration token to %s\n", *file)
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(*file), 0750); err != nil {
		return err
	}
	if err := os.WriteFile(*file, []byte(token+"\n"), 0600); err != nil {
		return err
	}
	fmt.Printf("new registration token written to %s:\n%s\n", *file, token)
	return nil
}

func tokenRevoke(args []string) error {
	fs := flag.NewFlagSet("token-revoke", flag.ContinueOnError)
	file := fs.String("file", "", "registration token file")
	execute := fs.Bool("execute", false, "truncate the token file")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *file == "" {
		return errors.New("--file is required")
	}
	if !*execute {
		fmt.Printf("dry-run: would clear registration token file %s\n", *file)
		return nil
	}
	if err := os.WriteFile(*file, []byte{}, 0600); err != nil {
		return err
	}
	fmt.Printf("registration token file cleared: %s\n", *file)
	return nil
}

func newSecret() (key, hash string, err error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", "", err
	}
	key = base64.RawURLEncoding.EncodeToString(b[:])
	sum := sha256.Sum256([]byte(key))
	return key, hex.EncodeToString(sum[:]), nil
}

func sqlQuote(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

func auditAction(db *sql.DB, command, target, detail string) error {
	if _, err := db.Exec(auditSchema); err != nil {
		return err
	}
	_, err := db.Exec(
		`INSERT INTO hub_admin_audit(created_at,operator,command,target,detail) VALUES(?,?,?,?,?)`,
		time.Now().Unix(), operatorName(), command, target, detail,
	)
	return err
}

func auditActionTx(tx *sql.Tx, command, target, detail string) error {
	if _, err := tx.Exec(auditSchema); err != nil {
		return err
	}
	_, err := tx.Exec(
		`INSERT INTO hub_admin_audit(created_at,operator,command,target,detail) VALUES(?,?,?,?,?)`,
		time.Now().Unix(), operatorName(), command, target, detail,
	)
	return err
}

func operatorName() string {
	if v := os.Getenv("USER"); v != "" {
		return v
	}
	if v := os.Getenv("LOGNAME"); v != "" {
		return v
	}
	return "unknown"
}

func formatCounts(counts []rowCount) string {
	parts := make([]string, 0, len(counts))
	for _, c := range counts {
		parts = append(parts, fmt.Sprintf("%s=%d", c.name, c.count))
	}
	return strings.Join(parts, " ")
}

func formatUnix(ts int64) string {
	if ts == 0 {
		return "never"
	}
	return time.Unix(ts, 0).UTC().Format(time.RFC3339)
}

func age(ts int64) string {
	if ts == 0 {
		return "unknown"
	}
	return time.Since(time.Unix(ts, 0)).Round(time.Second).String()
}

func formatError(errText string) string {
	if strings.TrimSpace(errText) == "" {
		return ""
	}
	return " error=" + strconv.Quote(errText)
}
