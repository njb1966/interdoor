package engine

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	_ "modernc.org/sqlite" // pure-Go SQLite driver, registers "sqlite"
)

// Store is the node's persistence layer. It owns the engine-generic tables
// (players, turn_state, events) and exposes the raw *sql.DB so game modules can
// manage their own tables. Writes are serialized through a mutex; combined with
// WAL this keeps SQLite happy under many concurrent sessions.
type Store struct {
	db       *sql.DB
	nodeID   string
	mu       sync.Mutex // serializes writes
	seq      int64      // monotonic per-node event sequence
	oseq     int64      // monotonic per-node obligation sequence
	handlers map[string][]EventHandler
}

// Open opens (creating if needed) the node database with WAL and a busy timeout.
func Open(path, nodeID string) (*Store, error) {
	dsn := path + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1) // single writer; simplest correct choice for B1
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return &Store{db: db, nodeID: nodeID, handlers: map[string][]EventHandler{}}, nil
}

// DB exposes the underlying handle for game-module tables.
func (s *Store) DB() *sql.DB { return s.db }

// NodeID returns this node's identifier.
func (s *Store) NodeID() string { return s.nodeID }

// Close closes the database.
func (s *Store) Close() error { return s.db.Close() }

const engineSchema = `
CREATE TABLE IF NOT EXISTS players (
    global_id  TEXT PRIMARY KEY,
    name       TEXT NOT NULL UNIQUE COLLATE NOCASE,
    home_node  TEXT NOT NULL,
    pw_hash    TEXT NOT NULL,
    level      INTEGER NOT NULL DEFAULT 1,
    standing   INTEGER NOT NULL DEFAULT 0,
    status     TEXT NOT NULL DEFAULT 'active',
    created_at INTEGER NOT NULL,
    last_seen  INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS turn_state (
    global_id  TEXT PRIMARY KEY REFERENCES players(global_id),
    actions    INTEGER NOT NULL,
    attacks    INTEGER NOT NULL,
    last_reset INTEGER NOT NULL,
    day_index  INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS events (
    event_id    TEXT PRIMARY KEY,
    source_node TEXT NOT NULL,
    seq         INTEGER NOT NULL,
    type        TEXT NOT NULL,
    ts          INTEGER NOT NULL,
    payload     TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS event_handler_state (
    event_id      TEXT NOT NULL,
    handler_index INTEGER NOT NULL,
    status        TEXT NOT NULL,
    error         TEXT NOT NULL DEFAULT '',
    updated_at    INTEGER NOT NULL,
    PRIMARY KEY(event_id, handler_index)
);
CREATE TABLE IF NOT EXISTS obligations (
    obligation_id TEXT PRIMARY KEY,
    source_node   TEXT NOT NULL,
    creditor_ref  TEXT NOT NULL,
    debtor_ref    TEXT NOT NULL,
    kind          TEXT NOT NULL,
    terms         TEXT NOT NULL,
    weight        INTEGER NOT NULL,
    status        TEXT NOT NULL DEFAULT 'open',
    created_at    INTEGER NOT NULL,
    resolved_at   INTEGER,
    resolution    TEXT
);
CREATE INDEX IF NOT EXISTS idx_obligations_debtor ON obligations(debtor_ref, status);
CREATE TABLE IF NOT EXISTS sync_state (
    id              INTEGER PRIMARY KEY CHECK (id = 1),
    hub_cursor      INTEGER NOT NULL DEFAULT 0,
    last_pushed_seq INTEGER NOT NULL DEFAULT 0,
    api_key         TEXT NOT NULL DEFAULT ''
);
INSERT OR IGNORE INTO sync_state(id) VALUES (1);
CREATE TABLE IF NOT EXISTS remote_roster (
    global_id  TEXT PRIMARY KEY,
    home_node  TEXT NOT NULL,
    name       TEXT NOT NULL,
    level      INTEGER NOT NULL DEFAULT 1,
    status     TEXT NOT NULL DEFAULT 'active',
    last_seen  INTEGER NOT NULL
);
`

// Migrate creates the engine-generic schema and initializes the event sequence.
func (s *Store) Migrate() error {
	if _, err := s.db.Exec(engineSchema); err != nil {
		return err
	}
	// Resume the event sequence from whatever's already stored.
	var max sql.NullInt64
	if err := s.db.QueryRow(`SELECT MAX(seq) FROM events WHERE source_node = ?`, s.nodeID).Scan(&max); err != nil {
		return err
	}
	if max.Valid {
		s.seq = max.Int64
	}
	// Resume the obligation sequence from the highest existing local o_N id.
	var omax sql.NullInt64
	if err := s.db.QueryRow(
		`SELECT MAX(CAST(substr(obligation_id, instr(obligation_id, ':o_') + 3) AS INTEGER))
		 FROM obligations WHERE source_node = ?`, s.nodeID).Scan(&omax); err != nil {
		return err
	}
	if omax.Valid {
		s.oseq = omax.Int64
	}
	return nil
}

// ---- Players ----

// InsertPlayer persists a new player and their initial turn state in one tx.
func (s *Store) InsertPlayer(p *Player, pwHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec(
		`INSERT INTO players(global_id,name,home_node,pw_hash,level,standing,status,created_at,last_seen)
		 VALUES(?,?,?,?,?,?,?,?,?)`,
		p.GlobalID, p.Name, p.HomeNode, pwHash, p.Level, p.Standing, p.Status,
		p.CreatedAt.Unix(), p.LastSeen.Unix(),
	); err != nil {
		return err
	}
	now := time.Now()
	if _, err := tx.Exec(
		`INSERT INTO turn_state(global_id,actions,attacks,last_reset,day_index) VALUES(?,?,?,?,?)`,
		p.GlobalID, MainActionsPerDay, AttacksPerDay, now.Unix(), 0,
	); err != nil {
		return err
	}
	return tx.Commit()
}

// PlayerByName loads a player and their password hash, case-insensitively.
func (s *Store) PlayerByName(name string) (*Player, string, error) {
	row := s.db.QueryRow(
		`SELECT global_id,name,home_node,pw_hash,level,standing,status,created_at,last_seen
		 FROM players WHERE name = ? COLLATE NOCASE`, name)
	var p Player
	var pwHash string
	var created, seen int64
	if err := row.Scan(&p.GlobalID, &p.Name, &p.HomeNode, &pwHash, &p.Level,
		&p.Standing, &p.Status, &created, &seen); err != nil {
		return nil, "", err
	}
	p.CreatedAt = time.Unix(created, 0)
	p.LastSeen = time.Unix(seen, 0)
	return &p, pwHash, nil
}

// SetPlayerStatus updates a player's status field (e.g. "active", "traveling").
func (s *Store) SetPlayerStatus(globalID, status string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`UPDATE players SET status=? WHERE global_id=?`, status, globalID)
	return err
}

// TouchLastSeen records that a player just connected.
func (s *Store) TouchLastSeen(globalID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`UPDATE players SET last_seen=? WHERE global_id=?`, time.Now().Unix(), globalID)
	return err
}

// LocalPlayers returns only players whose home node is this node (no remote_roster).
// Used by the federation syncer when pushing the local roster to the hub — we must
// never claim another node's players as our own.
func (s *Store) LocalPlayers() ([]Player, error) {
	rows, err := s.db.Query(
		`SELECT global_id,name,home_node,level,standing,status,created_at,last_seen
		 FROM players WHERE home_node=? ORDER BY name`, s.nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Player
	for rows.Next() {
		var p Player
		var created, seen int64
		if err := rows.Scan(&p.GlobalID, &p.Name, &p.HomeNode, &p.Level,
			&p.Standing, &p.Status, &created, &seen); err != nil {
			return nil, err
		}
		p.CreatedAt = time.Unix(created, 0)
		p.LastSeen = time.Unix(seen, 0)
		out = append(out, p)
	}
	return out, rows.Err()
}

// Players returns all players except excludeID — local players unioned with the
// remote roster pulled from the hub. Home node is set for all entries so the
// wanderers screen can show "(from nodeXX)" for visitors.
func (s *Store) Players(excludeID string) ([]Player, error) {
	rows, err := s.db.Query(
		`SELECT global_id,name,home_node,level,standing,status,created_at,last_seen
		   FROM players WHERE global_id <> ?
		 UNION ALL
		 SELECT global_id,name,home_node,level,0,status,0,last_seen
		   FROM remote_roster
		 ORDER BY name`, excludeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Player
	for rows.Next() {
		var p Player
		var created, seen int64
		if err := rows.Scan(&p.GlobalID, &p.Name, &p.HomeNode, &p.Level,
			&p.Standing, &p.Status, &created, &seen); err != nil {
			return nil, err
		}
		p.CreatedAt = time.Unix(created, 0)
		p.LastSeen = time.Unix(seen, 0)
		out = append(out, p)
	}
	return out, rows.Err()
}

// UpsertRemoteRoster replaces the full remote roster with the entries pulled
// from the hub. Called by the federation syncer after every pull cycle.
func (s *Store) UpsertRemoteRoster(entries []Player) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec(`DELETE FROM remote_roster`); err != nil {
		return err
	}
	for _, p := range entries {
		if _, err := tx.Exec(
			`INSERT INTO remote_roster(global_id,home_node,name,level,status,last_seen)
			 VALUES(?,?,?,?,?,?)`,
			p.GlobalID, p.HomeNode, p.Name, p.Level, p.Status, p.LastSeen.Unix(),
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ---- Turn economy ----

// Turn is a snapshot of a player's daily budget.
type Turn struct {
	Actions   int
	Attacks   int
	LastReset time.Time
	DayIndex  int
}

// LoadTurn reads the current turn state.
func (s *Store) LoadTurn(globalID string) (Turn, error) {
	var t Turn
	var reset int64
	err := s.db.QueryRow(
		`SELECT actions,attacks,last_reset,day_index FROM turn_state WHERE global_id=?`, globalID).
		Scan(&t.Actions, &t.Attacks, &reset, &t.DayIndex)
	t.LastReset = time.Unix(reset, 0)
	return t, err
}

// ResetIfNewDay refreshes the daily budget if the last reset was on an earlier
// calendar day (server-local). Returns true if a reset occurred. No rollover.
func (s *Store) ResetIfNewDay(globalID string) (bool, error) {
	t, err := s.LoadTurn(globalID)
	if err != nil {
		return false, err
	}
	now := time.Now()
	if sameDay(t.LastReset, now) {
		return false, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err = s.db.Exec(
		`UPDATE turn_state SET actions=?,attacks=?,last_reset=?,day_index=day_index+1 WHERE global_id=?`,
		MainActionsPerDay, AttacksPerDay, now.Unix(), globalID)
	return err == nil, err
}

// SpendActions decrements the main action budget. It fails if insufficient.
func (s *Store) SpendActions(globalID string, n int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.Exec(
		`UPDATE turn_state SET actions=actions-? WHERE global_id=? AND actions>=?`,
		n, globalID, n)
	if err != nil {
		return err
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return fmt.Errorf("not enough actions")
	}
	return nil
}

// SpendAttack decrements the separate daily PvP attack budget (CORE_LOOP §1.1).
func (s *Store) SpendAttack(globalID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.Exec(
		`UPDATE turn_state SET attacks=attacks-1 WHERE global_id=? AND attacks>=1`, globalID)
	if err != nil {
		return err
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return fmt.Errorf("no attacks left")
	}
	return nil
}

// ---- Events (DATA_MODEL.md Part 3) ----

// Emit appends a self-contained, sequenced event to the local log. Federation
// (Phase B3) consumes this same log; for now it is local only.
func (s *Store) Emit(typ string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	eventID := fmt.Sprintf("%s:%d", s.nodeID, s.seq)
	_, err = s.db.Exec(
		`INSERT INTO events(event_id,source_node,seq,type,ts,payload) VALUES(?,?,?,?,?,?)`,
		eventID, s.nodeID, s.seq, typ, time.Now().Unix(), string(body))
	return err
}

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}
