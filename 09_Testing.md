# Testing

## Required Test Categories

InterDoor needs tests at several levels:

- Engine unit tests for auth, turns, events, obligations, snapshots, terminal constraints, and generic hooks.
- Hub unit tests for registration, auth, directory, events, roster, debts, PvP, travel, and heartbeat counts.
- Federation client tests for push/pull cursors, retries, partition behavior, and queue drains.
- Game integration tests for handlers, travel import/export, PvP resolution, and obligation effects.
- End-to-end tests with at least two nodes and one hub.
- Documentation checks for local links and stale references.

Test hardening is continuous Phase 1 work. A federation bug fix, security-sensitive change,
queue-state change, or protocol contract change is not complete until it includes a regression
test at the lowest useful level, or this document records why automation is not yet practical.

## Existing Coverage

Current Go tests cover useful Phase 1 behavior, including:

- Event apply idempotency.
- Event handler failure retry after the event row is already stored.
- Event handler failure retry after process restart.
- Obligation creation and resolution events.
- Partial debt repayment emitting `debt.adjusted`.
- Partial debt repayment propagating across two federated nodes.
- Turn reset and no rollover.
- Server login rejects a player whose status is `traveling`.
- PvP attack budget.
- Snapshot export/import.
- Hub registration/auth and endpoint behavior.
- Event feed push/pull and deduplication.
- Same-game validation for Tier 3 debt events.
- Roster sync.
- Travel conflict prevention.
- Hub current-location enforcement for travel submit/arrival.
- Return-home travel and duplicate-travel rejection across two federated nodes.
- Owner-scoped PvP and travel completion.
- Heartbeat pending counts.
- Partition tolerance.
- Two nodes replay local backlogs after a simulated hub partition and reconnect.
- Remote roster cache survives a hub partition with `last_seen` intact so stale presentation can be
  shown after reconnect failures.
- PvP resolver failure leaving a queue item pending.
- Travel import failure leaving an arrival pending.
- Blocked queue lifecycle for malformed non-retryable PvP and travel payloads.
- Suspended nodes are hidden from public directory/status and cannot authenticate.
- Phase 1 standard event schema validation and malformed standard-event rejection.
- Stale remote roster entries are marked after the 15-minute Phase 1 freshness window.
- Hub-admin CLI dry-run and execute behavior for node removal.
- Hub-admin queue filtering, blocked queue retry, backup creation, API-key rotation, and
  registration-token issue/revoke.
- Hub-admin backup restore-readback against a copied SQLite database.
- Hub-admin audit rows for database-backed mutating commands.
- Deployed-shape E2E with a real hub process, copied SQLite state, reconnect replay, debt
  adjustment, duplicate travel rejection, return-home travel, and traveling-player status.
- Raw HTTP protocol conformance across Tier 1/Tier 2 endpoints without the Go federation client.
- Non-Go Tier 1 and Tier 2 HTTP examples under `games/sdk/`.

These tests live under `games/ledgerofthelow/`.

## Missing Or Weak Coverage

No currently documented Phase 1 production blocker is missing automated coverage.

Broader full-node SSH E2E coverage can be added later if Phase 1 needs tests that run multiple
complete node binaries. Current deployed-shape coverage already runs a real hub process against
SQLite and real node stores for the federation invariants.

## Production Restore Rehearsal

Completed 2026-06-24 against a copied real production hub DB from `contabo2`.

Rehearsal shape:

- Source: `/var/lib/interdoor-hub/hub.db` on `contabo2`, copied through SQLite `.backup` as the
  `interdoor-hub` service user while production stayed online.
- Local restored copy: `/tmp/interdoor-hub-restored-20260624.sqlite`.
- Verification: current `interdoor-hub` binary opened the restored copy on loopback
  `127.0.0.1:18282`; `/v1/status` returned `events_total=5`, `nodes_total=1`, `nodes_online=1`;
  `/v1/directory` returned only active node `thelow`.
- Operator readback: current `interdoor-hub-admin` listed `thelow` as active and showed empty
  PvP/travel queues.

No production service was stopped or modified. The rehearsal also verified that current schema
migrations can open the copied production database.

## Federation E2E Scenarios

Minimum two-node scenarios:

1. Register node A and node B with the hub.
2. Create one player on each node.
3. Verify both rosters appear through sync.
4. Emit event on node A and verify node B applies it.
5. Create debt on node A involving node B player and verify hub projection and node B mirror.
6. Partially repay debt and verify federated adjustment.
7. Queue PvP from A to B and verify only B resolves it.
8. Force B resolver failure and verify request remains pending.
9. Travel player from A to B and verify login blocked on A while pending.
10. Force B import failure and verify request is not marked arrived.
11. Complete travel and verify the hub current-location projection.
12. Disconnect hub, create local events, reconnect, and verify backlog replay.

## Documentation Verification

For the new root documentation:

- Verify expected files exist with `find`.
- Use `rg` over new Markdown files for local Markdown links.
- Check that root canonical docs do not depend on Ledger/Dornhaven as the product identity.
- Confirm resolved critical issues are documented as implemented and remaining hardening is still tracked in backlog/testing/decisions.

## Current Verification Command

From the reference implementation directory:

```bash
go test ./...
```

This confirms documentation additions did not disturb the Go implementation.
