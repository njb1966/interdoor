# InterDoor Project Rules

## Operating Rules

- Preserve the framework-first architecture.
- Keep the hub, engine, SDK, and games conceptually separate.
- Prefer small, inspectable changes over broad rewrites.
- Treat legacy terminal and BBS constraints as intentional unless proven otherwise.
- Do not add dependencies unless they solve a current, concrete problem.
- Document behavior that exists before documenting behavior that is merely desired.
- Keep old docs and game folders intact unless a task explicitly targets them.
- Treat test hardening as continuous work. Federation bugs, security-sensitive changes,
  queue behavior changes, and protocol contract changes need matching regression tests
  before they are considered closed, or a documented reason automation is not yet practical.

## Federation-First Principles

- Nodes are independent game servers. They must continue serving local players while disconnected from the hub.
- The hub coordinates federation. It does not host gameplay sessions or own player credentials.
- Nodes never talk directly to each other in Phase 1.
- Cross-node behavior is asynchronous and queue-based.
- Events must be self-contained enough to replay without querying the source node.
- Delivery can be at least once; effects must be idempotent.
- Global identifiers are mandatory for cross-node objects.

## State Classification

Every piece of state must be classified before it participates in federation:

- Local: owned by one node and never synced.
- Broadcast: small shared facts replicated through events or roster sync.
- Snapshot: full character state moved only for travel.
- Shared reference: catalogs and static game data shipped with the game version.

Ambiguous state should be redesigned or explicitly kept local.

## Terminal Compatibility

InterDoor targets SSH terminal games:

- 80x24 must remain a usable baseline.
- Base 16 ANSI color is the portability target.
- Keyboard input is the control model.
- UTF-8 may be used only when the game can degrade cleanly or the target game explicitly requires it.
- No mouse, truecolor, SyncTERM-only, or browser-only assumptions belong in the framework.

## Design Constraints

- Federation is turn/session-scale, not sub-second real-time.
- Cross-node PvP resolves on the victim side.
- Travel moves snapshots through the hub and blocks login while the character is in transit.
- Nodes may only push events whose `source_node` matches their authenticated node ID.
- Roster pushes are scoped to the authenticated node.
- Hub projections are derived from accepted events and queue tables.

## Non-Goals

- Kubernetes, containers, orchestration, or cloud-native deployment as a default.
- A required web frontend for operators or players.
- Centralized hosting of all games.
- Replacing existing retro protocols or terminal workflows.
- Broad modernization of legacy games before their behavior is understood.
- Building a universal MMO framework.

## Compatibility Rule

If a change makes it harder for a single Debian host to run a node with a static binary, SQLite database, systemd service, and public SSH port, the change needs an explicit architectural decision before implementation.

## Game Implementation Rule

Future games are not added to the Phase 1 build or release surface until their design has been
reviewed and explicitly approved. Historical or faulty game code may be preserved as source
material only when it is quarantined outside active module paths and clearly labeled.
