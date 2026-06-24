# Game SDK

## Purpose

The InterDoor SDK is the way games connect to the framework. It has two forms:

- The Go engine and federation client used by the reference implementation.
- The HTTP JSON protocol for games written in any other language.

The SDK should make federation possible without forcing games to share fiction, mechanics, storage schemas, or implementation language.

## Integration Tiers

### Tier 1: Listed

Minimum integration:

- Register with the hub.
- Heartbeat with game version, player count, and advertise address.

Result: the game appears in the directory and portal.

### Tier 2: Network-Aware

Adds:

- Event push.
- Event pull.
- Roster push.
- Roster pull.

Result: players can see cross-node presence and network events.

### Tier 3: Fully Integrated

Adds:

- Travel snapshot export/import.
- Cross-node PvP queue and victim-side resolution.
- Shared obligation events.
- Game-specific handlers for federated effects.

Result: characters and interactions can cross node boundaries.

Phase 1 Tier 3 mechanics require matching `game_id` on both nodes. Directory and roster data may show unlike games, but PvP, travel, and shared obligations are rejected across incompatible games until an explicit compatibility map exists.

## Engine Interface Expectations

A game module should provide:

- Stable game ID, human-readable title, and version.
- Migration for its game-specific tables.
- Terminal screens and input handling.
- Player state export/import for snapshots.
- Optional event handlers for federated events.
- Optional PvP resolver for incoming cross-node attacks.
- Optional travel UI using the engine travel hook.

The engine provides:

- Account and session handling.
- Terminal base.
- Player identity.
- Turn budgets.
- Event log.
- Obligation storage.
- Snapshot wrapper.
- Federation hook points.

## Event Hooks

Games should emit events for facts other nodes may need:

- Character creation and death.
- Public achievements.
- Debt creation and resolution.
- PvP results.
- Travel arrival.

Handlers should be idempotent. They should be safe to run once per accepted event.
The current framework records handler application status per event and retries failed handlers on replay; handlers still need idempotent local writes because a failure can happen after partial local work.

## Roster Hooks

Games should keep the roster small:

- ID.
- Name.
- Home node.
- Level or comparable eligibility field.
- Status.
- Last seen.

Do not put private character state or large game-specific data in the roster.

## Travel Hooks

Travel requires:

- Exporting a self-contained snapshot.
- Importing that snapshot on another node.
- Preserving global ID and home node.
- Blocking login on the origin node while the character is in transit.
- Marking a returning home character active after import.
- Treating failed imports as retryable by returning an error without acknowledging arrival.

The snapshot must include enough game-specific state for a visiting node to run the game without asking the origin node for more data.

## PvP Hooks

Cross-node PvP requires:

- Attacker-side payload containing only data needed to resolve the attack.
- Victim-side resolution against local victim state.
- A result event containing attacker ID, victim ID, request ID, outcome, loot, and death status.
- Attacker-side result handler for crediting rewards.

The victim's node is the authority for the victim's state. Attacker nodes must not mutate remote victims.

## Obligation Hooks

Games that use the shared obligation ledger should:

- Create obligations through the engine so `debt.created` is emitted.
- Partially repay obligations through the engine so `debt.adjusted` is emitted.
- Resolve obligations through the engine so `debt.resolved` is emitted.
- Reference players by global ID and NPCs by stable refs.

## Non-Go Games

Non-Go games do not need the reference engine. They need:

- Their own SSH or terminal server.
- Local player/account storage.
- An HTTP client for the hub endpoints.
- Event, roster, PvP, travel, and obligation code for whichever tier they implement.

Tier 1 and Tier 2 should remain small enough for a traditional door game port to implement without adopting the Go engine.

Non-Go HTTP examples live under `games/sdk/`:

- `tier1_http.md` shows registration and heartbeat.
- `tier2_http.md` shows event and roster push/pull.
