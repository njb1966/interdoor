# NETWORK REQUIREMENTS

> Status: Historical game-specific source material. The root numbered InterDoor docs are canonical for Phase 1 framework, protocol, operator, and architecture decisions. Do not use this file as Phase 1 framework authority unless it has been explicitly reconciled.

## Constraints on Game Design for Federation Compatibility

---

## Purpose

This document defines the technical and architectural constraints that the federated terminal game network imposes on any game built for it — including Dornhaven. Every game design decision must be checked against these requirements. A mechanic that works beautifully on a single node but breaks under federation is a mechanic that needs redesign before implementation.

These requirements exist so that federation isn't bolted on after the fact. The game is designed for federation from the start, even though federation is built later.

---

## Requirement 1: All State Must Be Classifiable

Every piece of game state must belong to exactly one of three categories:

### Local State
State that exists only on one node and never syncs. Examples: NPC dialogue progress for a specific player, Warrens map seed for today, local market inventory.

**Rule:** Local state can be anything. No constraints on format, frequency of change, or complexity.

### Synced State
State that must be consistent across all nodes in the network. Examples: player identity (name, credentials), cross-node debt obligations, network-wide event log, endgame consequences that affect all nodes.

**Rule:** Synced state must be serializable to JSON. It must have a timestamp and a source node identifier. It must be expressible as a delta (what changed) rather than requiring full state transfer. It must tolerate eventual consistency — two nodes may have slightly different versions of synced state for a short period, and the game must handle that gracefully.

### Shared-Reference State
State that all nodes need to agree on but that doesn't change during play. Examples: the game's item catalog, faction definitions, base NPC attributes, encounter templates.

**Rule:** Shared-reference state is distributed as part of the game package. All nodes running the same game version have identical shared-reference state. It changes only with game updates, not during runtime.

### The Classification Test

For every data structure in the game design, ask:
1. Does another node ever need to know about this? If no → **Local.**
2. Does it change during gameplay? If no → **Shared-Reference.**
3. If yes to both → **Synced.** Define the delta format.

If a mechanic produces state that is ambiguously classifiable, the mechanic needs redesign.

---

## Requirement 2: Player Identity Must Be Portable

A player's identity must work across nodes. This means:

- **Unique player ID** — A globally unique identifier assigned at character creation. Format: `node_id:player_id` (e.g., `haven01:p_3f8a`). This never changes even if the player travels between nodes.
- **Credential portability** — A player can authenticate on any node in the network. This requires either a central auth service (hub-managed) or a token-based system where the home node vouches for the player.
- **Character state ownership** — A player's character state (stats, inventory, progression) is owned by their home node. When they travel to another node, a snapshot of relevant state is transferred. The visiting node does not permanently store the character.

### Design Implication
No game mechanic should assume the player was created on this node. No mechanic should reference a player by local database row ID. All player references use the global ID format.

---

## Requirement 3: Time Must Be Turn-Based, Not Real-Time

The federation protocol syncs state on a schedule — not instantly. This means:

- **No real-time multiplayer mechanics.** Two players on different nodes cannot engage in simultaneous, tick-by-tick interaction. (Two players on the SAME node can, but cross-node interactions must tolerate latency.)
- **Daily turns as the sync boundary.** The natural sync point is the daily turn reset. State accumulated during a day is synced at or around turn reset. Mechanics that require sub-day cross-node consistency are not supported.
- **Actions are atomic.** A player's action (explore, trade, fight) produces a complete state change that can be serialized as a single event. No action should require mid-execution coordination with another node.

### Design Implication
Combat between players on different nodes must be asynchronous — you attack their stored state, not their live session. This is already the natural model (attack sleeping players, à la LORD), but it must be a design constraint, not just a default.

Cross-node contracts and trades must be structured as offer/accept, not as real-time negotiation. Player A posts an offer. It syncs to other nodes. Player B accepts. The acceptance syncs back. Each step is a complete action.

---

## Requirement 4: Events Must Be Self-Contained

The federation protocol broadcasts events across nodes. An event is a record of something that happened — a player action, a game state change, a milestone. Events are the primary mechanism by which nodes learn about what's happening elsewhere in the network.

An event must be:
- **Self-contained.** Reading the event must not require querying the source node for additional context. All necessary information is in the event payload.
- **Idempotent.** Processing the same event twice must produce the same result as processing it once. Events may be delivered more than once due to network issues.
- **Ordered.** Events carry a timestamp and a sequence number from their source node. Receiving nodes process events in order. Out-of-order events are queued.
- **Typed.** Every event has a type identifier that tells receiving nodes what kind of state change it represents and how to process it.

### Standard Event Types (minimum set)

| Event Type | Description | Synced Data |
|------------|-------------|-------------|
| `player.created` | New player registered | Player ID, name, home node |
| `player.died` | Player death | Player ID, cause, location |
| `player.traveled` | Player moved to another node | Player ID, source node, dest node, character snapshot |
| `debt.created` | New debt obligation | Creditor ID, debtor ID, terms, amount |
| `debt.resolved` | Debt paid or forgiven | Debt ID, resolution type |
| `debt.transferred` | Debt sold/traded to new creditor | Debt ID, new creditor ID |
| `faction.shift` | Faction power change on a node | Node ID, faction ID, old standing, new standing |
| `endgame.completed` | Player completed endgame | Player ID, path chosen, consequences |
| `node.status` | Periodic heartbeat | Node ID, player count, uptime, game version |

Games may define additional event types. The engine API provides the event emission interface; the federation layer handles delivery.

### Design Implication
Every game mechanic that produces a state change visible to other nodes must emit an event. If a mechanic can't be expressed as a self-contained event, it either needs redesign or must be classified as local-only state.

---

## Requirement 5: The Economy Must Support Cross-Node Transactions

The favor/debt economy is a core mechanic of Dornhaven and a key differentiator of the network. Debts must be trackable across node boundaries. This means:

- **Every debt has a globally unique ID.** Format: `source_node:debt_id`.
- **Debts reference players by global ID,** not local ID.
- **Debt creation, resolution, and transfer each emit events** that sync across nodes.
- **The hub maintains a master debt index.** Individual nodes maintain their own debt tables for local performance, but the hub is authoritative for cross-node debts.
- **Debt conflicts resolve in favor of the hub's record.** If Node A says a debt is resolved but the hub hasn't received the resolution event, the debt is still active until the hub confirms.

### Design Implication
The in-game economy must be designed so that cross-node debt resolution doesn't require real-time coordination. A debt created on Node A and owed by a player on Node B resolves when the creditor calls it in (an atomic action on the creditor's node) and the resolution event syncs to Node B and the hub. The debtor's state updates on next sync. There may be a delay of minutes to hours between the creditor calling in the debt and the debtor seeing the effect.

---

## Requirement 6: Game State Must Tolerate Partitions

Nodes will go offline. The network will partition. A node that was syncing every hour might go dark for three days and come back. The game must handle this gracefully.

**Rules:**
- A node operates independently when disconnected from the hub. All local gameplay continues normally.
- When a node reconnects, it replays missed events in order and reconciles state.
- Events generated during a partition are queued locally and transmitted on reconnection.
- Cross-node interactions initiated during a partition (e.g., attacking a player whose node is offline) are queued and resolve when connectivity is restored.
- Stale player data (from before the partition) is marked as stale in the UI so players know the information may be outdated.

### Design Implication
No game mechanic should fail or produce errors because another node is unreachable. Every cross-node interaction must have a queued/pending state that is visible to the player. "Your attack on [player] from [node] is pending — their node is currently unreachable" is a valid game state.

---

## Requirement 7: The Game Engine API Must Be Generic

The engine API is not Dornhaven-specific. It defines interfaces that any terminal-based game could implement. This means:

- **No Dornhaven-specific concepts in the engine.** The engine knows about players, sessions, actions, state, and events. It does not know about the Warrens, the Lanternmarket, favors, or the Ledger. Those are Dornhaven's implementation of the engine's interfaces.
- **Games register with the engine as modules.** A node could theoretically run multiple games, each implemented as a separate module against the same engine API.
- **The connection server is game-agnostic.** It handles SSH, authentication, and session management. It passes control to whichever game module the player selects.

### Design Implication
When designing Dornhaven's mechanics, distinguish between "this is a game mechanic" and "this is an engine feature." If a mechanic could apply to any game (e.g., daily action limits, player-to-player messaging, leaderboards), it should be an engine feature, not a Dornhaven-specific implementation. If it's specific to Dornhaven's setting and fiction (e.g., Ledger attention, faction reputation), it belongs in the game module.

---

## Requirement 8: Terminal Rendering Must Be Standard

The game must be playable on any terminal emulator that supports:
- ANSI escape codes (colors, cursor positioning, basic formatting)
- 80 columns × 24 rows minimum
- UTF-8 (optional — game must degrade gracefully to ASCII)

**No assumptions about:**
- 256-color or truecolor support (use the base 16 ANSI colors)
- Mouse input
- Terminal size beyond 80×24
- Specific terminal emulator features (no SyncTERM-specific extensions)

### Design Implication
All UI design, ANSI art, and screen layouts must fit in 80×24. Color is used for atmosphere and readability, not for conveying essential information (a player on a monochrome terminal should still be able to play). All input is keyboard-based.

---

## Compatibility Checklist

For every game mechanic, run through this checklist:

- [ ] Can the state it produces be classified as Local, Synced, or Shared-Reference?
- [ ] Does it reference players by global ID (not local database ID)?
- [ ] Does it work with daily-turn sync boundaries (no real-time cross-node needs)?
- [ ] Can its state changes be expressed as self-contained, idempotent events?
- [ ] Does it tolerate another node being offline for days?
- [ ] Does it belong in the engine (generic) or the game module (Dornhaven-specific)?
- [ ] Does it render correctly in 80×24 with base ANSI colors?

If any answer is "no," the mechanic needs redesign or must be explicitly scoped as local-only (single-node feature that doesn't participate in federation).

---

*Document version 0.1 — These requirements will be refined as game design progresses and federation implementation reveals additional constraints.*
