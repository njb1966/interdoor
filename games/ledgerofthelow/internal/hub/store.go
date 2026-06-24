// Package hub is the InterDOOR federation coordinator (FEDERATION_PROTOCOL.md).
// It is hub-and-spoke and hub-authoritative. Storage is behind the Store interface
// so the current SQLite backend can be replaced later without touching the HTTP
// layer.
package hub

import (
	"encoding/json"
	"errors"
	"time"
)

// ProtocolVersion is the wire-protocol version this hub speaks.
const ProtocolVersion = "1"

var (
	// ErrNodeExists is returned when registering an already-known node id.
	ErrNodeExists = errors.New("node already registered")
	// ErrNotFound is returned when a lookup matches nothing.
	ErrNotFound = errors.New("not found")
	// ErrIncompatibleGame is returned when a Tier 3 feature crosses game IDs.
	ErrIncompatibleGame = errors.New("incompatible game")
	// ErrLocationMismatch is returned when a node tries to move a player it does
	// not currently hold.
	ErrLocationMismatch = errors.New("player is not located at this node")
	// ErrTravelActive is returned when a player already has a pending travel request
	// (single-active invariant, FEDERATION_PROTOCOL.md §10).
	ErrTravelActive = errors.New("player already has an active travel request")
)

// Node is a registered spoke.
type Node struct {
	NodeID          string
	GameID          string
	GameTitle       string
	GameVersion     string
	ProtocolVersion string
	AdvertiseAddr   string
	PlayerCount     int
	UptimeS         int
	LastHeartbeat   time.Time
	Status          string
}

func (n Node) DisplayGameTitle() string {
	if n.GameTitle != "" {
		return n.GameTitle
	}
	return n.GameID
}

// WireEvent is an event as it crosses the network (FEDERATION_PROTOCOL.md §5).
type WireEvent struct {
	EventID    string          `json:"event_id"`
	SourceNode string          `json:"source_node"`
	Seq        int64           `json:"seq"`
	Type       string          `json:"type"`
	TS         int64           `json:"ts"`
	Payload    json.RawMessage `json:"payload"`
}

// FeedEvent is a WireEvent positioned in the hub's total order. The embedded
// WireEvent fields flatten into the same JSON object as hub_seq.
type FeedEvent struct {
	HubSeq int64 `json:"hub_seq"`
	WireEvent
}

// RosterEntry is one player as seen by the hub roster (§6 of FEDERATION_PROTOCOL.md).
// It carries only the display fields needed by the wanderers screen — no credentials.
type RosterEntry struct {
	GlobalID string `json:"global_id"`
	NodeID   string `json:"node_id"`
	Name     string `json:"name"`
	Level    int    `json:"level"`
	Status   string `json:"status"`
	LastSeen int64  `json:"last_seen"` // unix timestamp
}

// HubDebt is an obligation record in the hub's cross-node Ledger index.
type HubDebt struct {
	ObligationID string `json:"obligation_id"`
	SourceNode   string `json:"source_node"`
	CreditorRef  string `json:"creditor_ref"`
	DebtorRef    string `json:"debtor_ref"`
	Kind         string `json:"kind"`
	Terms        string `json:"terms"`
	Weight       int    `json:"weight"`
	Status       string `json:"status"`
	UpdatedAt    int64  `json:"updated_at"`
}

// TravelRequest is a player snapshot in transit between nodes (FEDERATION_PROTOCOL.md §9).
type TravelRequest struct {
	TravelID  string          `json:"travel_id"`
	GlobalID  string          `json:"global_id"`
	HomeNode  string          `json:"home_node"`
	FromNode  string          `json:"from_node"`
	DestNode  string          `json:"dest_node"`
	Snapshot  json.RawMessage `json:"snapshot"`
	Status    string          `json:"status"`
	Error     string          `json:"error,omitempty"`
	CreatedAt int64           `json:"created_at"`
	UpdatedAt int64           `json:"updated_at,omitempty"`
}

// PvPRequest is a queued cross-node attack (FEDERATION_PROTOCOL.md §8).
type PvPRequest struct {
	RequestID       string          `json:"request_id"`
	AttackerID      string          `json:"attacker_id"`
	VictimID        string          `json:"victim_id"`
	VictimNode      string          `json:"victim_node"`
	AttackerPayload json.RawMessage `json:"attacker"`
	Status          string          `json:"status"` // pending | resolved | blocked
	Error           string          `json:"error,omitempty"`
	CreatedAt       int64           `json:"created_at"`
	UpdatedAt       int64           `json:"updated_at,omitempty"`
}

// Store is the hub's persistence contract. The development backend is SQLite
// (OpenSQLite); a Postgres backend implements the same interface for production.
type Store interface {
	// RegisterNode records a new node with its API-key hash. Returns ErrNodeExists
	// if the node id is taken.
	RegisterNode(n Node, apiKeyHash string) error
	// NodeByAPIKeyHash authenticates a node by its API-key hash.
	NodeByAPIKeyHash(apiKeyHash string) (*Node, error)
	// NodeByID returns a registered node by id.
	NodeByID(nodeID string) (*Node, error)
	// Heartbeat updates liveness/metrics for a node.
	Heartbeat(nodeID string, playerCount, uptimeS int, gameVersion string) error
	// AppendEvents stores events idempotently (dedup by event_id), assigning each
	// a monotonic hub_seq. Returns how many were newly accepted and the feed head.
	AppendEvents(events []WireEvent) (accepted int, head int64, err error)
	// FeedSince returns feed events with hub_seq greater than after, ordered and
	// capped at limit. If excludeNode is non-empty, that source's events are omitted.
	FeedSince(after int64, limit int, excludeNode string) ([]FeedEvent, int64, error)
	// UpsertRoster replaces all roster entries for nodeID with the supplied list.
	UpsertRoster(nodeID string, entries []RosterEntry) error
	// GetRoster returns all roster entries except those from excludeNode.
	GetRoster(excludeNode string) ([]RosterEntry, error)
	// QueuePvP adds a cross-node attack request. The hub generates the request_id.
	QueuePvP(req PvPRequest) error
	// PendingPvP returns queued attacks whose victim lives on victimNode.
	PendingPvP(victimNode string) ([]PvPRequest, error)
	// CompletePvP marks an attack request resolved. Only the victim node may
	// complete a pending request.
	CompletePvP(requestID, victimNode string) error
	// BlockPvP marks a malformed pending attack blocked. Only the victim node may
	// block its own pending request.
	BlockPvP(requestID, victimNode, reason string) error
	// MaterializeEvents parses debt.* events and maintains the hub Ledger index.
	// Call after AppendEvents for every push batch. Idempotent (INSERT OR IGNORE).
	MaterializeEvents(events []WireEvent) error
	// DebtsForDebtor returns all obligations (any status) owed by a global player ID.
	DebtsForDebtor(debtor string) ([]HubDebt, error)
	// SubmitTravel queues a player snapshot for relay to the destination node.
	// Returns ErrTravelActive if the player already has a pending request.
	SubmitTravel(req TravelRequest) error
	// PendingTravel returns travel requests destined for destNode with status 'pending'.
	PendingTravel(destNode string) ([]TravelRequest, error)
	// ArriveTravel marks a travel request arrived. Only the destination node may
	// complete a pending arrival.
	ArriveTravel(travelID, destNode string) error
	// BlockTravel marks a malformed pending arrival blocked. Only the destination
	// node may block its own pending request.
	BlockTravel(travelID, destNode, reason string) error
	// FeedHead returns the highest hub_seq in the event feed (0 if empty).
	FeedHead() (int64, error)
	// PendingPvPCount returns the count of pending PvP requests targeting victimNode.
	PendingPvPCount(victimNode string) (int, error)
	// PendingTravelCount returns the count of pending travel arrivals for destNode.
	PendingTravelCount(destNode string) (int, error)
	// GetDirectory returns all registered nodes (used for the public node directory).
	GetDirectory() ([]Node, error)
	// EventCount returns the total number of events stored in the hub feed.
	EventCount() (int64, error)
	Close() error
}
