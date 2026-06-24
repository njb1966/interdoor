// Package engine is the generic InterDOOR terminal-game engine. It knows about
// players, sessions, the daily turn economy, persistence, and events — never about
// any specific game. Games implement the Game interface and are registered with a
// node; the engine never imports a game package.
package engine

import "time"

// Daily turn economy (engine concept; specific numbers are the reference game's).
// The reference game frames MainActionsPerDay as "Warren expeditions" (split-pool
// model, CORE_LOOP §1.1 revised after playtest). Town actions are free.
const (
	MainActionsPerDay = 15
	AttacksPerDay     = 3
)

// Player is the engine-generic identity spine (DATA_MODEL.md §1.1, "Synced/Broadcast").
// Game-specific character state (stats, inventory) lives in the game module's own
// tables, keyed by GlobalID.
type Player struct {
	GlobalID  string // node_id:player_id — globally unique, never changes
	Name      string
	HomeNode  string
	Level     int
	Standing  int
	Status    string // active | dead | traveling
	CreatedAt time.Time
	LastSeen  time.Time
}
