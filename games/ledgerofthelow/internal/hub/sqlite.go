package hub

import (
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type sqliteStore struct{ db *sql.DB }

const sqliteSchema = `
CREATE TABLE IF NOT EXISTS nodes (
    node_id          TEXT PRIMARY KEY,
    api_key_hash     TEXT NOT NULL,
    game_id          TEXT NOT NULL,
    game_title       TEXT NOT NULL DEFAULT '',
    game_version     TEXT NOT NULL,
    protocol_version TEXT NOT NULL,
    advertise_addr   TEXT NOT NULL DEFAULT '',
    player_count     INTEGER NOT NULL DEFAULT 0,
    uptime_s         INTEGER NOT NULL DEFAULT 0,
    last_heartbeat   INTEGER NOT NULL DEFAULT 0,
    status           TEXT NOT NULL DEFAULT 'active',
    created_at       INTEGER NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_nodes_apikey ON nodes(api_key_hash);
CREATE TABLE IF NOT EXISTS events (
    hub_seq     INTEGER PRIMARY KEY AUTOINCREMENT,
    event_id    TEXT NOT NULL UNIQUE,
    source_node TEXT NOT NULL,
    seq         INTEGER NOT NULL,
    type        TEXT NOT NULL,
    ts          INTEGER NOT NULL,
    payload     TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS roster (
    global_id  TEXT PRIMARY KEY,
    node_id    TEXT NOT NULL,
    name       TEXT NOT NULL,
    level      INTEGER NOT NULL DEFAULT 1,
    status     TEXT NOT NULL DEFAULT 'active',
    last_seen  INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_roster_node ON roster(node_id);
CREATE TABLE IF NOT EXISTS debts (
    obligation_id TEXT PRIMARY KEY,
    source_node   TEXT NOT NULL,
    creditor_ref  TEXT NOT NULL,
    debtor_ref    TEXT NOT NULL,
    kind          TEXT NOT NULL,
    terms         TEXT NOT NULL,
    weight        INTEGER NOT NULL,
    status        TEXT NOT NULL DEFAULT 'open',
    updated_at    INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_debts_debtor ON debts(debtor_ref, status);
CREATE TABLE IF NOT EXISTS pvp_requests (
    request_id       TEXT PRIMARY KEY,
    attacker_id      TEXT NOT NULL,
    victim_id        TEXT NOT NULL,
    victim_node      TEXT NOT NULL,
    attacker_payload TEXT NOT NULL,
    status           TEXT NOT NULL DEFAULT 'pending',
    error            TEXT NOT NULL DEFAULT '',
    updated_at       INTEGER NOT NULL DEFAULT 0,
    created_at       INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_pvp_victim ON pvp_requests(victim_node, status);
CREATE TABLE IF NOT EXISTS travel (
    travel_id  TEXT PRIMARY KEY,
    global_id  TEXT NOT NULL,
    home_node  TEXT NOT NULL,
    from_node  TEXT NOT NULL,
    dest_node  TEXT NOT NULL,
    snapshot   TEXT NOT NULL,
    status     TEXT NOT NULL DEFAULT 'pending',
    error      TEXT NOT NULL DEFAULT '',
    updated_at INTEGER NOT NULL DEFAULT 0,
    created_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_travel_dest ON travel(dest_node, status);
CREATE TABLE IF NOT EXISTS player_locations (
    global_id    TEXT PRIMARY KEY,
    current_node TEXT NOT NULL,
    home_node    TEXT NOT NULL,
    status       TEXT NOT NULL DEFAULT 'active',
    travel_id    TEXT,
    updated_at   INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_player_locations_node ON player_locations(current_node, status);
`

// OpenSQLite opens (creating if needed) the hub database. SQLite is the current
// Phase 1 backend; another production backend can implement Store later.
func OpenSQLite(path string) (Store, error) {
	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		return nil, err
	}
	if _, err := db.Exec(sqliteSchema); err != nil {
		return nil, err
	}
	if err := migrateSQLite(db); err != nil {
		return nil, err
	}
	return &sqliteStore{db: db}, nil
}

func migrateSQLite(db *sql.DB) error {
	for _, stmt := range []string{
		`ALTER TABLE pvp_requests ADD COLUMN error TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE pvp_requests ADD COLUMN updated_at INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE travel ADD COLUMN error TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE travel ADD COLUMN updated_at INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE nodes ADD COLUMN game_title TEXT NOT NULL DEFAULT ''`,
	} {
		if _, err := db.Exec(stmt); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
			return err
		}
	}
	return nil
}

func (s *sqliteStore) RegisterNode(n Node, apiKeyHash string) error {
	var exists int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM nodes WHERE node_id=?`, n.NodeID).Scan(&exists); err != nil {
		return err
	}
	if exists > 0 {
		return ErrNodeExists
	}
	now := time.Now().Unix()
	_, err := s.db.Exec(
		`INSERT INTO nodes(node_id,api_key_hash,game_id,game_title,game_version,protocol_version,advertise_addr,last_heartbeat,created_at)
		 VALUES(?,?,?,?,?,?,?,?,?)`,
		n.NodeID, apiKeyHash, n.GameID, n.GameTitle, n.GameVersion, n.ProtocolVersion, n.AdvertiseAddr, now, now)
	return err
}

func (s *sqliteStore) NodeByAPIKeyHash(apiKeyHash string) (*Node, error) {
	row := s.db.QueryRow(
		`SELECT node_id,game_id,COALESCE(NULLIF(game_title,''),game_id),game_version,protocol_version,advertise_addr,player_count,uptime_s,last_heartbeat,status
		 FROM nodes WHERE api_key_hash=?`, apiKeyHash)
	var n Node
	var hb int64
	if err := row.Scan(&n.NodeID, &n.GameID, &n.GameTitle, &n.GameVersion, &n.ProtocolVersion,
		&n.AdvertiseAddr, &n.PlayerCount, &n.UptimeS, &hb, &n.Status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	n.LastHeartbeat = time.Unix(hb, 0)
	return &n, nil
}

func (s *sqliteStore) NodeByID(nodeID string) (*Node, error) {
	row := s.db.QueryRow(
		`SELECT node_id,game_id,COALESCE(NULLIF(game_title,''),game_id),game_version,protocol_version,advertise_addr,player_count,uptime_s,last_heartbeat,status
		 FROM nodes WHERE node_id=?`, nodeID)
	var n Node
	var hb int64
	if err := row.Scan(&n.NodeID, &n.GameID, &n.GameTitle, &n.GameVersion, &n.ProtocolVersion,
		&n.AdvertiseAddr, &n.PlayerCount, &n.UptimeS, &hb, &n.Status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	n.LastHeartbeat = time.Unix(hb, 0)
	return &n, nil
}

func (s *sqliteStore) Heartbeat(nodeID string, playerCount, uptimeS int, gameVersion string) error {
	res, err := s.db.Exec(
		`UPDATE nodes SET last_heartbeat=?, player_count=?, uptime_s=?, game_version=? WHERE node_id=?`,
		time.Now().Unix(), playerCount, uptimeS, gameVersion, nodeID)
	if err != nil {
		return err
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *sqliteStore) AppendEvents(events []WireEvent) (int, int64, error) {
	accepted := 0
	for _, e := range events {
		res, err := s.db.Exec(
			`INSERT OR IGNORE INTO events(event_id,source_node,seq,type,ts,payload) VALUES(?,?,?,?,?,?)`,
			e.EventID, e.SourceNode, e.Seq, e.Type, e.TS, string(e.Payload))
		if err != nil {
			return accepted, 0, err
		}
		if n, _ := res.RowsAffected(); n > 0 {
			accepted++
		}
	}
	head, err := s.head()
	return accepted, head, err
}

func (s *sqliteStore) FeedSince(after int64, limit int, excludeNode string) ([]FeedEvent, int64, error) {
	q := `SELECT hub_seq,event_id,source_node,seq,type,ts,payload FROM events WHERE hub_seq>?`
	args := []any{after}
	if excludeNode != "" {
		q += ` AND source_node<>?`
		args = append(args, excludeNode)
	}
	q += ` ORDER BY hub_seq LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []FeedEvent
	for rows.Next() {
		var e FeedEvent
		var payload string
		if err := rows.Scan(&e.HubSeq, &e.EventID, &e.SourceNode, &e.Seq, &e.Type, &e.TS, &payload); err != nil {
			return nil, 0, err
		}
		e.Payload = json.RawMessage(payload)
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	head, err := s.head()
	return out, head, err
}

func (s *sqliteStore) UpsertRoster(nodeID string, entries []RosterEntry) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec(`DELETE FROM roster WHERE node_id=?`, nodeID); err != nil {
		return err
	}
	now := time.Now().Unix()
	for _, e := range entries {
		if _, err := tx.Exec(
			`INSERT INTO roster(global_id,node_id,name,level,status,last_seen,updated_at)
			 VALUES(?,?,?,?,?,?,?)`,
			e.GlobalID, nodeID, e.Name, e.Level, e.Status, e.LastSeen, now,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *sqliteStore) GetRoster(excludeNode string) ([]RosterEntry, error) {
	q := `SELECT global_id,node_id,name,level,status,last_seen FROM roster`
	var args []any
	if excludeNode != "" {
		q += ` WHERE node_id<>?`
		args = append(args, excludeNode)
	}
	q += ` ORDER BY name`
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []RosterEntry
	for rows.Next() {
		var e RosterEntry
		if err := rows.Scan(&e.GlobalID, &e.NodeID, &e.Name, &e.Level, &e.Status, &e.LastSeen); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *sqliteStore) QueuePvP(req PvPRequest) error {
	_, err := s.db.Exec(
		`INSERT INTO pvp_requests(request_id,attacker_id,victim_id,victim_node,attacker_payload,status,error,updated_at,created_at)
		 VALUES(?,?,?,?,?,'pending','',0,?)`,
		req.RequestID, req.AttackerID, req.VictimID, req.VictimNode,
		string(req.AttackerPayload), req.CreatedAt)
	return err
}

func (s *sqliteStore) PendingPvP(victimNode string) ([]PvPRequest, error) {
	rows, err := s.db.Query(
		`SELECT request_id,attacker_id,victim_id,victim_node,attacker_payload,status,error,updated_at,created_at
		 FROM pvp_requests WHERE victim_node=? AND status='pending' ORDER BY created_at`, victimNode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PvPRequest
	for rows.Next() {
		var r PvPRequest
		var payload string
		if err := rows.Scan(&r.RequestID, &r.AttackerID, &r.VictimID, &r.VictimNode,
			&payload, &r.Status, &r.Error, &r.UpdatedAt, &r.CreatedAt); err != nil {
			return nil, err
		}
		r.AttackerPayload = json.RawMessage(payload)
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *sqliteStore) CompletePvP(requestID, victimNode string) error {
	res, err := s.db.Exec(
		`UPDATE pvp_requests SET status='resolved', updated_at=?
		 WHERE request_id=? AND victim_node=? AND status='pending'`,
		time.Now().Unix(), requestID, victimNode)
	if err != nil {
		return err
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *sqliteStore) BlockPvP(requestID, victimNode, reason string) error {
	res, err := s.db.Exec(
		`UPDATE pvp_requests SET status='blocked', error=?, updated_at=?
		 WHERE request_id=? AND victim_node=? AND status='pending'`,
		reason, time.Now().Unix(), requestID, victimNode)
	if err != nil {
		return err
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *sqliteStore) MaterializeEvents(events []WireEvent) error {
	for _, e := range events {
		switch e.Type {
		case "debt.created":
			var p struct {
				ObligationID string `json:"obligation_id"`
				CreditorRef  string `json:"creditor_ref"`
				DebtorRef    string `json:"debtor_ref"`
				Kind         string `json:"kind"`
				Terms        string `json:"terms"`
				Weight       int    `json:"weight"`
			}
			if err := json.Unmarshal(e.Payload, &p); err != nil {
				continue // malformed payload; skip, don't break the push
			}
			_, _ = s.db.Exec(
				`INSERT OR IGNORE INTO debts
				 (obligation_id,source_node,creditor_ref,debtor_ref,kind,terms,weight,status,updated_at)
				 VALUES(?,?,?,?,?,?,?,'open',?)`,
				p.ObligationID, e.SourceNode, p.CreditorRef, p.DebtorRef,
				p.Kind, p.Terms, p.Weight, e.TS)
		case "debt.resolved":
			var p struct {
				ObligationID string `json:"obligation_id"`
			}
			if err := json.Unmarshal(e.Payload, &p); err != nil {
				continue
			}
			_, _ = s.db.Exec(
				`UPDATE debts SET status='resolved', updated_at=? WHERE obligation_id=?`,
				e.TS, p.ObligationID)
		case "debt.adjusted":
			var p struct {
				ObligationID string `json:"obligation_id"`
				NewWeight    int    `json:"new_weight"`
			}
			if err := json.Unmarshal(e.Payload, &p); err != nil {
				continue
			}
			_, _ = s.db.Exec(
				`UPDATE debts SET weight=?, updated_at=? WHERE obligation_id=? AND status='open'`,
				p.NewWeight, e.TS, p.ObligationID)
		}
	}
	return nil
}

func (s *sqliteStore) DebtsForDebtor(debtor string) ([]HubDebt, error) {
	rows, err := s.db.Query(
		`SELECT obligation_id,source_node,creditor_ref,debtor_ref,kind,terms,weight,status,updated_at
		 FROM debts WHERE debtor_ref=? ORDER BY updated_at DESC`, debtor)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []HubDebt
	for rows.Next() {
		var d HubDebt
		if err := rows.Scan(&d.ObligationID, &d.SourceNode, &d.CreditorRef, &d.DebtorRef,
			&d.Kind, &d.Terms, &d.Weight, &d.Status, &d.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (s *sqliteStore) SubmitTravel(req TravelRequest) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// Single-active invariant: a player may only have one pending travel request.
	var count int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM travel WHERE global_id=? AND status='pending'`, req.GlobalID).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return ErrTravelActive
	}

	currentNode, homeNode, err := s.currentLocation(tx, req.GlobalID)
	if err != nil {
		return err
	}
	if currentNode == "" {
		currentNode = victimNode(req.GlobalID)
		homeNode = currentNode
	}
	if req.HomeNode == "" {
		req.HomeNode = homeNode
	}
	if currentNode != req.FromNode {
		return ErrLocationMismatch
	}
	if req.DestNode == req.FromNode {
		return ErrLocationMismatch
	}

	now := time.Now().Unix()
	if _, err := tx.Exec(
		`INSERT INTO travel(travel_id,global_id,home_node,from_node,dest_node,snapshot,status,error,updated_at,created_at)
		 VALUES(?,?,?,?,?,?,'pending','',0,?)`,
		req.TravelID, req.GlobalID, req.HomeNode, req.FromNode, req.DestNode,
		string(req.Snapshot), req.CreatedAt); err != nil {
		return err
	}
	if _, err := tx.Exec(
		`INSERT INTO player_locations(global_id,current_node,home_node,status,travel_id,updated_at)
		 VALUES(?,?,?,?,?,?)
		 ON CONFLICT(global_id) DO UPDATE SET
		   current_node=excluded.current_node,
		   home_node=excluded.home_node,
		   status=excluded.status,
		   travel_id=excluded.travel_id,
		   updated_at=excluded.updated_at`,
		req.GlobalID, req.FromNode, req.HomeNode, "traveling", req.TravelID, now); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *sqliteStore) currentLocation(tx *sql.Tx, globalID string) (string, string, error) {
	var currentNode, homeNode string
	err := tx.QueryRow(
		`SELECT current_node,home_node FROM player_locations WHERE global_id=?`, globalID,
	).Scan(&currentNode, &homeNode)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", nil
		}
		return "", "", err
	}
	return currentNode, homeNode, nil
}

func (s *sqliteStore) FeedHead() (int64, error) { return s.head() }

func (s *sqliteStore) GetDirectory() ([]Node, error) {
	rows, err := s.db.Query(
		`SELECT node_id,game_id,COALESCE(NULLIF(game_title,''),game_id),game_version,protocol_version,advertise_addr,
		        player_count,uptime_s,last_heartbeat,status
		 FROM nodes WHERE status='active' ORDER BY node_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Node
	for rows.Next() {
		var n Node
		var hb int64
		if err := rows.Scan(&n.NodeID, &n.GameID, &n.GameTitle, &n.GameVersion, &n.ProtocolVersion,
			&n.AdvertiseAddr, &n.PlayerCount, &n.UptimeS, &hb, &n.Status); err != nil {
			return nil, err
		}
		n.LastHeartbeat = time.Unix(hb, 0)
		out = append(out, n)
	}
	return out, rows.Err()
}

func (s *sqliteStore) EventCount() (int64, error) {
	var n sql.NullInt64
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM events`).Scan(&n); err != nil {
		return 0, err
	}
	return n.Int64, nil
}

func (s *sqliteStore) PendingPvPCount(victimNode string) (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM pvp_requests WHERE victim_node=? AND status='pending'`, victimNode).Scan(&n)
	return n, err
}

func (s *sqliteStore) PendingTravelCount(destNode string) (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM travel WHERE dest_node=? AND status='pending'`, destNode).Scan(&n)
	return n, err
}

func (s *sqliteStore) PendingTravel(destNode string) ([]TravelRequest, error) {
	rows, err := s.db.Query(
		`SELECT travel_id,global_id,home_node,from_node,dest_node,snapshot,status,error,updated_at,created_at
		 FROM travel WHERE dest_node=? AND status='pending' ORDER BY created_at`, destNode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TravelRequest
	for rows.Next() {
		var r TravelRequest
		var snap string
		if err := rows.Scan(&r.TravelID, &r.GlobalID, &r.HomeNode, &r.FromNode,
			&r.DestNode, &snap, &r.Status, &r.Error, &r.UpdatedAt, &r.CreatedAt); err != nil {
			return nil, err
		}
		r.Snapshot = json.RawMessage(snap)
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *sqliteStore) ArriveTravel(travelID, destNode string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var globalID, homeNode string
	if err := tx.QueryRow(
		`SELECT global_id,home_node FROM travel
		 WHERE travel_id=? AND dest_node=? AND status='pending'`,
		travelID, destNode,
	).Scan(&globalID, &homeNode); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	res, err := tx.Exec(
		`UPDATE travel SET status='arrived', updated_at=?
		 WHERE travel_id=? AND dest_node=? AND status='pending'`,
		time.Now().Unix(), travelID, destNode)
	if err != nil {
		return err
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return ErrNotFound
	}
	if _, err := tx.Exec(
		`INSERT INTO player_locations(global_id,current_node,home_node,status,travel_id,updated_at)
		 VALUES(?,?,?,?,NULL,?)
		 ON CONFLICT(global_id) DO UPDATE SET
		   current_node=excluded.current_node,
		   home_node=excluded.home_node,
		   status=excluded.status,
		   travel_id=NULL,
		   updated_at=excluded.updated_at`,
		globalID, destNode, homeNode, "active", time.Now().Unix()); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *sqliteStore) BlockTravel(travelID, destNode, reason string) error {
	res, err := s.db.Exec(
		`UPDATE travel SET status='blocked', error=?, updated_at=?
		 WHERE travel_id=? AND dest_node=? AND status='pending'`,
		reason, time.Now().Unix(), travelID, destNode)
	if err != nil {
		return err
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *sqliteStore) head() (int64, error) {
	var h sql.NullInt64
	if err := s.db.QueryRow(`SELECT MAX(hub_seq) FROM events`).Scan(&h); err != nil {
		return 0, err
	}
	return h.Int64, nil
}

func (s *sqliteStore) Close() error { return s.db.Close() }
