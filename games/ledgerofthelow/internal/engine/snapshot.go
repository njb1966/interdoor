package engine

import (
	"encoding/json"
	"fmt"
	"time"
)

// Snapshot is a portable, self-contained character bundle for cross-node travel
// (DATA_MODEL.md §1.2 "Synced-Snapshot" + Req 2). The home node vouches for the
// player by including the credential hash, so a visiting node can authenticate
// without contacting home. Game-specific state is opaque to the engine.
type Snapshot struct {
	Version   int             `json:"version"`
	Player    Player          `json:"player"`
	PwHash    string          `json:"pw_hash"`
	GameID    string          `json:"game_id"`
	GameState json.RawMessage `json:"game_state"`
}

const snapshotVersion = 1

// ExportPlayer builds a portable snapshot of a player owned by this node.
func (s *Store) ExportPlayer(globalID string, g Game) (*Snapshot, error) {
	p, hash, err := s.playerByID(globalID)
	if err != nil {
		return nil, err
	}
	state, err := g.ExportState(s.db, globalID)
	if err != nil {
		return nil, err
	}
	return &Snapshot{
		Version: snapshotVersion, Player: *p, PwHash: hash,
		GameID: g.ID(), GameState: state,
	}, nil
}

// ImportPlayer installs a snapshot received from another node. Identity (with the
// vouched credential) and game state are written; a fresh LOCAL turn budget is
// created if the visiting player has none (turn state is node-local). HomeNode is
// preserved — the visiting node does not take ownership of the character.
func (s *Store) ImportPlayer(snap *Snapshot, g Game) error {
	if snap.Version != snapshotVersion {
		return fmt.Errorf("unsupported snapshot version %d", snap.Version)
	}
	if snap.GameID != g.ID() {
		return fmt.Errorf("snapshot game %q != node game %q", snap.GameID, g.ID())
	}
	p := snap.Player
	s.mu.Lock()
	_, err := s.db.Exec(
		`INSERT INTO players(global_id,name,home_node,pw_hash,level,standing,status,created_at,last_seen)
		 VALUES(?,?,?,?,?,?,?,?,?)
		 ON CONFLICT(global_id) DO UPDATE SET
		   name=excluded.name, home_node=excluded.home_node, pw_hash=excluded.pw_hash,
		   level=excluded.level, standing=excluded.standing, status=excluded.status,
		   last_seen=excluded.last_seen`,
		p.GlobalID, p.Name, p.HomeNode, snap.PwHash, p.Level, p.Standing, p.Status,
		p.CreatedAt.Unix(), p.LastSeen.Unix())
	if err == nil {
		_, err = s.db.Exec(
			`INSERT OR IGNORE INTO turn_state(global_id,actions,attacks,last_reset,day_index) VALUES(?,?,?,?,?)`,
			p.GlobalID, MainActionsPerDay, AttacksPerDay, time.Now().Unix(), 0)
	}
	s.mu.Unlock()
	if err != nil {
		return err
	}
	return g.ImportState(s.db, &p, snap.GameState)
}

// playerByID loads a player and credential hash by global ID.
func (s *Store) playerByID(globalID string) (*Player, string, error) {
	row := s.db.QueryRow(
		`SELECT global_id,name,home_node,pw_hash,level,standing,status,created_at,last_seen
		 FROM players WHERE global_id=?`, globalID)
	var p Player
	var hash string
	var created, seen int64
	if err := row.Scan(&p.GlobalID, &p.Name, &p.HomeNode, &hash, &p.Level,
		&p.Standing, &p.Status, &created, &seen); err != nil {
		return nil, "", err
	}
	p.CreatedAt = time.Unix(created, 0)
	p.LastSeen = time.Unix(seen, 0)
	return &p, hash, nil
}
