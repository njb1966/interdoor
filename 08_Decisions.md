# Decisions

## Accepted Decisions

### InterDoor Is The Framework

InterDoor is the federated terminal-game framework. Ledger of the Low is the Phase 1 reference node. Dornhaven, Dominion/Empire Ascendant, and future games are not part of Phase 1 unless explicitly approved as separate game work.

### Empire Ascendant Deferred

Empire Ascendant/Dominion is deferred to Phase 2 or later. It must be designed against the original source material and reviewed before being added to InterDoor as a game node. Existing Empire Ascendant artifacts are historical source material, not accepted Phase 1 architecture or implementation.

### Faulty Dominion Code Quarantined

The faulty Dominion/Empire Ascendant implementation was removed from the active Phase 1 Go module
and release targets. Preserved source and local build artifacts live under
`games/empireascendant/dominion/quarantine/` as historical material only. They must not be built,
released, deployed, or used as architecture authority without a fresh Phase 2 design approval.

### Licensed Door Games Are Examples Only

LORD, Usurper, and BRE are examples of the BBS door-game tradition InterDoor is inspired by. They are not planned implementations, ports, or integration targets. Future InterDoor games must be custom works or open-source/license-compatible games that can legally be modified and redistributed.

### Hub-And-Spoke For Phase 1

Phase 1 uses a hub-and-spoke architecture. Nodes do not talk directly to other nodes.

### SSH For Player Sessions

Player sessions are SSH terminal sessions. HTTP is used for node-to-hub federation, not for gameplay.

### HTTP JSON Protocol

The Phase 1 wire protocol is HTTP JSON with bearer-token authentication.

### Node Independence

Nodes must continue local gameplay when the hub is unreachable. Federation failures should degrade cross-node features, not local play.

### Engine/Game Separation

The engine must remain generic. Game modules implement game logic and own game-specific state.

### SQLite Is Accepted For Phase 1

SQLite is accepted for the Phase 1 public hub while InterDoor is operating as a small network with
local-only hub administration and low write volume. The hub must use online backup or service-aware
copy procedures for production backups, and restore drills must be verified against copied
production data.

A production hub backend such as Postgres may be added later behind the existing store interface if
node count, write volume, delegated administration, or operational recovery needs outgrow SQLite.

### Global IDs Are Required

Federated players, obligations, and events use source-prefixed global IDs.

### Victim-Side PvP Resolution

Cross-node PvP resolves on the victim's node. The attacker node queues a request and consumes the result.

### Snapshot-Based Travel

Travel moves a self-contained snapshot through the hub. The origin blocks login while the player is in transit.

### Hub Location Projection

The hub maintains an explicit player-location projection for Tier 3 travel. The projection records the current node, home node, active/traveling status, and pending travel ID. The travel queue is the state-transition source for departures and arrivals.

### Retryable Handler Application

Event receipt and handler application are tracked separately. A node may store an event and retry failed handlers on replay without duplicating the event row or rerunning handlers already marked applied.

### Partial Debt Adjustments

Partial repayments use `debt.adjusted`. The event carries `obligation_id`, `old_weight`, `new_weight`, `delta`, and `reason`. Hubs and nodes materialize the new open weight idempotently.

### Standard Event Schema Validation

The hub validates Phase 1 standard event payloads before accepting them into the global feed.
Malformed standard events are rejected with the Phase 1 error response instead of being stored and
silently skipped by projections. Unknown game-defined events are still allowed when their payload is
a JSON object.

### Queue Completion Ownership

Only the victim node may complete a pending PvP request. Only the destination node may mark a pending travel request arrived. Completion of missing, already completed, or wrong-owner requests is rejected.

### Queue Failure Handling

Node-side PvP resolver failures and travel import failures leave hub queue items pending for retry.
Malformed non-retryable PvP attacker payloads and travel snapshots may be marked `blocked` only by
the victim or destination node that owns the pending work. Blocked items are hidden from pending
drain endpoints, keep an operator-visible error string, and may be reset to `pending` only through
local hub operator tooling after repair.

### Same-Game Tier 3 Compatibility

Roster and directory visibility may cross all games. Tier 3 mechanics that move or mutate character state across nodes, including PvP and travel, require matching `game_id` unless a future compatibility map explicitly allows otherwise.

## Unresolved Decision Points

### Naming And Repository Layout

Need to decide whether framework code remains under `games/ledgerofthelow/` or moves to a root module later. No move should happen until the framework docs and tests stabilize.
