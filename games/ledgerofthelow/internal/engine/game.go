package engine

import (
	"database/sql"
	"encoding/json"

	"interdoor.net/interdoor/internal/engine/term"
)

// CrossNodePvPFn submits a cross-node PvP attack to the hub. Returns a request ID.
// Injected by the server when a hub is configured; nil on standalone nodes.
type CrossNodePvPFn func(attackerID, victimID string, payload json.RawMessage) (requestID string, err error)

// PvPResolveFn resolves a queued cross-node attack that arrived at this node.
// The game module provides the implementation; the sync loop calls it.
type PvPResolveFn func(store *Store, reqID, attackerID, victimID string, payload json.RawMessage) error

// PvPPending is a queued cross-node attack from the hub, ready to resolve.
type PvPPending struct {
	RequestID       string
	AttackerID      string
	VictimID        string
	AttackerPayload json.RawMessage
}

// TravelFn initiates cross-node travel for a player. The closure captures the
// fed client and handles snapshot export + hub submission. Nil on standalone nodes.
type TravelFn func(globalID, destNode string) error

// TravelImportFn installs an arriving player snapshot. The node startup code
// provides the implementation; the sync loop calls it on each pending arrival.
type TravelImportFn func(store *Store, snap *Snapshot, fromNode, destNode string) error

// TravelPending is an arriving player snapshot from the hub, ready to import.
type TravelPending struct {
	TravelID string
	GlobalID string
	HomeNode string
	FromNode string
	DestNode string
	Snapshot json.RawMessage
}

// Context is everything a game needs to run one logged-in session. The engine
// builds it after authentication and passes it to Game.Run.
type Context struct {
	Player          *Player
	Term            *term.Terminal
	Store           *Store // generic persistence + events + turn economy
	NodeID          string
	CrossNodeAttack CrossNodePvPFn // nil if no hub configured
	Travel          TravelFn       // nil if no hub configured
}

// Game is the engine<->game contract (NETWORK_REQUIREMENTS Req 7). The engine
// handles SSH, terminals, accounts, persistence, the turn economy, and events;
// a Game supplies the world. The engine never imports a concrete game.
type Game interface {
	// ID is the stable module identifier (e.g. "ledger_of_the_low").
	ID() string
	// Title is the player-facing name shown by the engine.
	Title() string
	// Banner is the welcome/title screen content (may embed ANSI). The engine
	// renders it above the generic login gate; the game owns the vibe.
	Banner() string
	// Migrate creates the game's own tables. Called once at node startup.
	Migrate(db *sql.DB) error
	// NewCharacter seeds game-specific state for a freshly created player.
	NewCharacter(db *sql.DB, p *Player) error
	// Run drives the game UI for a logged-in session until the player quits.
	Run(ctx *Context) error
	// ExportState serializes a player's game-specific state for a travel snapshot.
	// The bytes are opaque to the engine.
	ExportState(db *sql.DB, globalID string) (json.RawMessage, error)
	// ImportState installs game-specific state for a visiting player from a snapshot.
	ImportState(db *sql.DB, p *Player, state json.RawMessage) error
}
