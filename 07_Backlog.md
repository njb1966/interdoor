# InterDoor Backlog

## Phase 1 Closeout Status

No known critical Phase 1 production blocker remains for the current small-network deployment
shape: one public hub, local-only hub administration, SQLite hub storage, and independently
operated SSH game nodes.

The remaining items in this backlog are conditional hardening or future scale work. They should not
block Phase 2 planning unless the deployment shape changes before then.

## Completed Critical Phase 1 Hardening

### Event Handler Retry Semantics

Implemented: node stores now track handler application status by event ID and handler index. Failed handlers are retried on replay without duplicating the event row or rerunning handlers already marked applied.

Remaining hardening:

- Keep handlers idempotent where they can fail after partial local work.

### PvP Queue Completion On Failure

Implemented: node sync leaves a PvP request pending when victim-side resolution returns an error. Hub completion is scoped to the victim node and only pending requests can be completed.

Remaining hardening:

- Covered for Phase 1 by `interdoor-hub-admin queues`, which reports queue age, status, and blocked
  error text. Result-event linkage remains conditional future audit work.

### Travel Arrival Completion On Failure

Implemented: node sync leaves a travel request pending when import returns an error. Hub arrival confirmation is scoped to the destination node and only pending requests can be completed.
Malformed non-retryable snapshots may be marked `blocked` by the destination node and removed from
pending drain results with an operator-visible error.

Remaining hardening:

- Covered for Phase 1 by `interdoor-hub-admin queues` and `queue-retry` for blocked items. Direct
  travel payload repair remains manual/operator-guided and should not be automated without a
  specific recovery case.

### Travel Current-Location Invariant

Implemented: the hub keeps an explicit `player_locations` projection. A travel submission is accepted only from the node that currently holds the player. If no location row exists yet, the player's global-ID source prefix is the initial current node.

Remaining hardening:

- Add broader deployed-node coverage if future Phase 1 deployment shape starts running multiple
  node binaries under test, not just a real hub process with real node stores.

### Partial Debt Repayment Federation Gap

Implemented: partial repayment emits `debt.adjusted` with old weight, new weight, delta, and reason. Hub and node handlers materialize the new open weight.

Remaining hardening:

- Add result/event audit tooling if operators need to trace individual debt projection changes
  from the hub database.

### Cross-Node PvP Victim-Side Enforcement

Implemented: only the victim node can complete a pending PvP request, repeated completion is rejected, and PvP queueing requires registered same-game victim nodes.
Malformed non-retryable attacker payloads may be marked `blocked` by the victim node and removed
from pending drain results with an operator-visible error.

Remaining hardening:

- Add result-event audit linkage to completion.

### Same-Game Tier 3 Compatibility

Implemented: PvP, travel, `debt.created`, and `pvp.resolved` are rejected across registered nodes with different `game_id` values. Directory and roster visibility remain network-wide.

Remaining hardening:

- Design a future explicit compatibility map if unlike games should share Tier 3 mechanics.

## Remaining Phase 1 Hardening

### Dominion Build Surface Cleanup

Implemented: the faulty Dominion/Empire Ascendant code and local build artifacts were quarantined
under `games/empireascendant/dominion/quarantine/` and removed from the active Phase 1 Makefile
build/release targets.

Remaining hardening:

- Keep Empire Ascendant out of the active module and production deployment until Phase 2 design
  review and approval.
- Do not extract generic game-node framework guidance from Ledger alone; wait until Ledger and a
  properly designed second game can be compared.

### Action-Economy Documentation Drift

Older docs describe 12 actions per day. Current code defines 15 main actions and 3 attacks per day.

Needed:

- Treat code as factual for now: 15 main actions, 3 attacks.
- Update or quarantine game-specific docs when that work is in scope.
- Keep framework docs from promising game balance numbers unless necessary.

## Framework Backlog

- Add an SDK compatibility test suite that can be run against any hub implementation.
- Add optional machine-readable error codes to the structured error response if client needs justify it.
- Add pagination/cursor support for non-event list endpoints if Phase 1 scale outgrows unpaginated lists.
- Revisit delegated moderation/admin workflows only if node volume or submission volume makes local CLI administration insufficient.

Recently covered:

- Stale remote roster display is now tested against the 15-minute Phase 1 rule.
- Remote roster cache persistence after a hub partition is now tested so stale presentation has the
  required data after reconnect failures.
- Phase 1 standard event payload schemas are now validated before events enter the hub feed.
- Malformed non-retryable PvP and travel queue payloads now have an owner-scoped `blocked`
  lifecycle and local `queue-retry` operator command.
- Deployed-shape E2E coverage now runs a real hub process against SQLite and verifies reconnect
  replay, cross-node debt adjustment, duplicate travel rejection, return-home travel, and login
  status while traveling.
- Raw HTTP conformance coverage now exercises Tier 1/Tier 2 endpoints without the Go federation
  client.
- Production-style restore rehearsal completed 2026-06-24 using a copied real hub SQLite online
  backup from `contabo2`.
- `interdoor-hub-admin` is installed on the production hub host `contabo2`; read-only `nodes` and
  `queues` plus dry-run `node-suspend thelow` were verified without restarting the hub.
- The production hub binary on `contabo2` is upgraded to the current Phase 1 hub. The blocked queue
  endpoints and SQLite queue schema migration are live; rollback DB and binary backups were
  preserved on the host.
- The production hub SSH portal detail screen now explicitly says the hub lists nodes and does not
  launch game sessions, then shows "Open a new terminal and run:" with the node SSH command.
- Hub-admin regression tests now cover node-removal dry-run/execute behavior, queue filtering,
  blocked queue retry, backup creation and mode, API-key rotation, and registration-token
  issue/revoke.
- Hub-admin tests now verify local backup restore-readback and successful audit rows for
  database-backed mutating commands.
- Non-Go Tier 1 and Tier 2 HTTP examples now live under `games/sdk/`.
- Root architecture docs now include Debian/systemd examples for hub and node services.

## Future Game/SDK Backlog

- Create `games/sdk` as a stable package or documentation set.
- Define the approval checklist for adding a new game to InterDoor.
- Document the licensing rule: future games must be custom works or open-source/license-compatible games that can legally be modified and redistributed.
- Define whether approved future games use native code plus HTTP client, wrappers, or reimplementations.
- Define game compatibility tiers for travel and PvP between unlike games.

## Documentation Backlog

- Audit old `docs/` for game-specific claims that conflict with current InterDoor framing.
- Move reusable operator material into root docs without deleting historical files.
- Mark historical game-specific docs clearly where they can mislead Phase 1 operators.
- Defer generic game-node framework extraction until a second approved game node exists.
- Add diagrams after protocol and invariants settle.
- Add a glossary for node, hub, engine, game, roster, snapshot, obligation, and event.
