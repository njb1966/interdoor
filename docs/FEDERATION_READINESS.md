# FEDERATION READINESS TRACKER

> Status: Historical game-specific source material. The root numbered InterDoor docs are canonical for Phase 1 framework, protocol, operator, and architecture decisions. Do not use this file as Phase 1 framework authority unless it has been explicitly reconciled.


*Living document. Status: v0.1. Updated every build slice.*

---

## Why this file exists

The network is the product; the game is the proof. During the single-node build phase
it is easy to drift into "just making the game" and quietly let the federated network —
the actual point — fall off the radar. This tracker exists to prevent that.

**The rule:** every build slice ends with a federation-readiness check recorded here.
For anything new we add, we answer: does it emit the right events? does it reference
players/obligations by *global* ID? is its state correctly classified (Local / Synced-
Broadcast / Synced-Snapshot / Shared-Reference)? would it tolerate another node being
offline? If a slice can't answer those, it isn't done.

This keeps Phase B3 (federation) cheap: we are not retrofitting federation later, we are
keeping the foundation continuously B3-ready while the game grows on top of it.

---

## NETWORK_REQUIREMENTS.md vs. the current code

| Req | Requirement | Current reality in code | Status |
|---|---|---|---|
| 1 | All state classifiable | `players`/`events`/`obligations` = Broadcast; `char_state`/`inventory` = Snapshot (game-owned); `turn_state` = Local. Split holds. | ✅ Ready |
| 2 | Player identity portable | Global IDs `node:p_xxx`; `home_node` stored & preserved on import. `ExportPlayer`/`ImportPlayer` travel snapshot — credential hash vouched (auth works on the visiting node), game state via `Game.Export/ImportState`. Round-trip tested across two node ids. Live transfer transport = B3. | ✅ Ready |
| 3 | Turn-based, not real-time | Daily turn economy is the sync boundary; all actions are atomic state changes. Cross-node async not yet exercised. | ✅ Ready (design) |
| 4 | Events self-contained / idempotent / ordered / typed | `Emit` (produce) + `EventsSince` (read/replay) + `ApplyEvent` (idempotent intake: dedup by `event_id`, retryable handler application) + `OnEvent` registry. Out-of-order gap-queue is a B3 transport concern. | ✅ Ready (local guarantees); transport queue = B3 |
| 5 | Economy cross-node | Obligations globally IDed `node:o_N`, debtor/creditor refs, `debt.created`/`debt.adjusted`/`debt.resolved` emitted and materialized. | ✅ Ready |
| 6 | Tolerate partitions | Verified in `TestPartitionTolerance`: Tick() fails gracefully during partition, events accumulate locally (cursors NOT advanced), backlog pushed on reconnect, idempotent re-apply confirmed. Cross-node requests (pvp/travel) queue at hub until target node reconnects. | ✅ Ready |
| 7 | Generic engine vs game module | `internal/engine` never imports `internal/game`; the `Game` interface is the only seam. Verified. | ✅ Ready |
| 8 | Terminal standard (80×24, base-16) | `engine/term` enforces width and base-16; wraps long lines. | ✅ Ready |

---

## Federation-critical primitives (the real "tighten up" list)

These are the things B3 will distribute. They are foundation, not game content — and
they are the priority whenever we are not adding a network-relevant game feature.

- ✅ **Event read/replay API + idempotent apply** — `EventsSince` (outbound/inspect) +
  `ApplyEvent` (idempotent intake, dedup by `event_id`, retryable handler status) + `OnEvent`
  registry. Done; the inbound contract B3 will use.
- ✅ **Portable identity** — `ExportPlayer`/`ImportPlayer` travel snapshot: vouched
  credential (auth works on the visiting node) + game state via `Game.Export/ImportState`;
  home-node ownership preserved. Round-trip tested. Live transport = B3.
- ✅ **Graceful lifecycle & limits** — SIGINT/SIGTERM shutdown (clean DB close),
  per-session panic recovery, **max-sessions cap + idle timeout** (both verified).
  Graceful per-session drain on shutdown still future (minor).
- ✅ **Config** — JSON config file + flag overrides (`-config`; `addr/db/hostkey/node/
  max-sessions/idle-timeout`). Deeper admin tooling (player management, resets) still TODO.
- ✅ **The hub** — registration, event feed, roster sync, cross-node Ledger, cross-node
  PvP, travel, and conflict-resolution hardening are implemented for Phase 1.

---

## What every new game feature must preserve

A quick reference so game work never breaks federation-readiness:

- **Players** are referenced by `global_id` (`node:p_xxx`) — never a local row number.
- **Obligations** are referenced by `obligation_id` (`node:o_N`) and go through the
  engine Ledger (`CreateObligation`/`PayDebt`) so `debt.*` events fire.
- **Anything another node would care about emits an event** (`player.*`, `debt.*`,
  `pvp.resolved`, …) — self-contained, no callback to the source node required.
- **Cross-node interactions resolve on the target's home node** and tolerate delay —
  design them as offer→sync→accept / attack→resolve→result, never real-time.

---

## Road to B3 (federation proper)

Foundation is green. In rough order:

1. `FEDERATION_PROTOCOL.md` — spec before code. ✅ **Written (v0.1).**
2. Hub skeleton — registration endpoint + auth. ✅ **Built (`cmd/hub`, SQLite-backed `Store` interface).**
3. Node↔hub heartbeat (`node.status`). ✅ **Built & verified over real HTTP.**
4. Event feed — push/pull over the wire + node sync client. ✅ **Built & verified (live 2-node + hub: an event on node01 reached node02's log over real HTTP).**
5. **Player roster sync (who exists where).** ✅ **Done — hub `POST/GET /v1/roster`; node `remote_roster` table; `Players()` UNIONs local+remote; wanderers screen federation-ready.**
6. **Cross-node debt tracking (the Ledger goes distributed — the differentiator).** ✅ **Done — node-side `OnEvent` handlers for `debt.created`/`debt.adjusted`/`debt.resolved` replicate foreign obligations into local table; hub `debts` master index materialized from event push; `GET /v1/debts?debtor=<global_id>` cross-node query live.**
7. ✅ **Cross-node PvP.** Done — hub pvp queue; attack routed from attacker's node; resolved on victim's node; `pvp.resolved` event propagates via feed; attacker's node credits loot via `OnEvent` handler.
8. ✅ **Cross-node travel.** Done — hub `travel` table + 3 endpoints; `TravelFn`/`TravelImportFn` injected at startup; `Syncer.Tick()` drains arrivals; `ImportPlayer` + `SetPlayerStatus` handle arrival/return; `status='traveling'` blocks login; game `[V]` menu + depart/arrive screens.
9. Conflict resolution (hub-authoritative, last-write-wins per object).

---

## Slice log (append one row per build slice)

| Slice | What it added | Federation-readiness check |
|---|---|---|
| B1.1 walking skeleton | SSH, auth, term, turn economy, explore→combat, `player.created`/`player.died` | ✅ global IDs; events emitted; engine/game split clean; 80×24 |
| B1.2 economy | Obligation Ledger (engine), Maren's stall, loot/goods, live Debt, `debt.created`/`debt.resolved` | ✅ obligations globally IDed + engine-owned; events fire; NPC-only (per spec); cross-node-ready in shape |
| B1.3 PvP | Offline attacks (separate budget), bout resolution, loot transfer, "while you slept" inbox, anti-grief limits, `pvp.resolved` | ✅ both parties by global ID; resolves against **stored** (offline) victim state = the exact async, resolve-on-target's-home-node model B3 needs; event self-contained (ids+loot+`victim_died`); UTC-day window for cooldown/cap. **Cross-node TODO (B3):** route attack → victim's home node → resolve there → result event back. Note: `pvp_log`/inbox is game-local today; under federation it lives on the victim's node. |
| B1.H1 hardening | Event read/replay (`EventsSince`) + idempotent apply (`ApplyEvent`/`OnEvent`); graceful SIGINT/SIGTERM shutdown + per-session panic recovery | ✅ **Req 4 → Ready**: dedup by `event_id`, retryable handler status, foreign events readable. Node lifecycle robust; clean DB close verified (`integrity_check ok`). |
| B1.H2 hardening | Config file + flags (`-config`, max-sessions, idle-timeout); session cap + idle-timeout watchdog; portable-identity travel snapshot (`ExportPlayer`/`ImportPlayer` + `Game.Export/ImportState`) | ✅ **Req 2 → Ready**: snapshot round-trips across node ids, credential vouched (auth on visiting node), home-node ownership preserved. Session cap + idle close verified live. **Foundation complete — only the hub (B3) remains.** |
| B3.0 protocol | `FEDERATION_PROTOCOL.md` v0.1 — hub-and-spoke, REST/HTTPS, event-feed core, cross-node Ledger/PvP/travel, conflict + partition rules, endpoint reference | ✅ Spec before code; grounded in shipped primitives (`EventsSince`/`ApplyEvent`, obligations, snapshots, daily turn). |
| B3.1 hub skeleton | `cmd/hub` + `internal/hub` (SQLite behind `Store` iface): `/v1/register` (token + api-key), bearer auth, `/v1/heartbeat` | ✅ Verified over real HTTP (register→key, auth heartbeat updates metrics, 401/409 negatives). Alternative production backend can be added behind `Store` later. |
| B3.2 event feed | Hub `POST/GET /v1/events` (push: source-match + dedup by `event_id` + `hub_seq`; pull: cursor + `exclude_self`). Node sync client `internal/fed` (`Client`+`Syncer`): register, push `EventsSince`, pull + `ApplyEvent`, cursors in `sync_state`. Node auto-registers + runs sync loop when `-hub` set. | ✅ **Cross-node propagation works.** Go integration test (event A→B, idempotent, exclude_self) + **live 2-node+hub demo over real HTTP** (`player.created` on node01 appeared in node02's log via push→hub→pull). |
| B3.3 roster sync | Hub `POST/GET /v1/roster` (full-replace per node; `exclude_self` on pull). Node `remote_roster` table. `Store.UpsertRemoteRoster()`. `Players()` UNIONs local+remote (HomeNode set for origin display). `Syncer.Tick()` pushes local roster + pulls+caches remote after every event cycle. | ✅ **Wanderers screen now federation-aware.** Verified live: WandererAlpha (node01) appeared in node02's remote_roster and vice versa after first sync tick. PvP vs remote players deferred to B3.5. |
| B3.4 cross-node Ledger | Node `RegisterDebtHandlers()` — `OnEvent("debt.created"/"debt.adjusted"/"debt.resolved")` insert/update local `obligations` table from foreign events. Hub `debts` master index — `MaterializeEvents()` called on every push; `GET /v1/debts?debtor=<id>` cross-node query. | ✅ `debt.adjusted` covers partial repayment; same-game player refs enforced for Tier 3 debt events. |
| B3.5 cross-node PvP | Hub `pvp_requests` queue (`POST /v1/pvp`, `GET /v1/pvp/pending`, `POST /v1/pvp/{id}/result`). Attacker node routes remote attack via `CrossNodePvPFn` (injected at startup). Victim node drains pending in `Syncer.Tick()` via `PvPResolveFn` (`IncomingPvP`): runs bout with attacker payload stats, calls `stealLoot`, writes victim `pvp_log`, emits `pvp.resolved`. Attacker node's `RegisterGameHandlers` `OnEvent("pvp.resolved")` handler credits loot and updates pending `pvp_log` entry. `(remote)` tag in attack menu. `request_id=""` distinguishes local vs cross-node events to prevent double-credit. | ✅ Resolver failure leaves request pending; only the victim node can complete; same-game victim required. |
| B3.6 cross-node travel | Hub `travel` table + `player_locations` projection + `POST /v1/travel`, `GET /v1/travel/pending`, `POST /v1/travel/{id}/arrived`. `TravelFn`/`TravelImportFn`/`TravelPending` types in engine. `SetPlayerStatus` on Store. `SetTravelFn` on Server; blocks `status='traveling'` at login. `Syncer.TravelImport` drain loop. `TravelFn` closure in node commands: sets `traveling`, exports snapshot, submits to hub, and rolls status back if submit fails. On arrival: `ImportPlayer` + set `active` if returning home + emit `player.traveled`. | ✅ Import failure leaves request pending; only the destination node can mark arrival; same-game destination required. |
| B3.7 conflict resolution + partition tests | Hub `SubmitTravel` conflict guard (`ErrTravelActive` → 409, single-active invariant §10), explicit current-location authority, owner-scoped completion, heartbeat counts, and partition replay tests. | ✅ Build clean; all tests green. Req 6 now Ready. |
| B4 public network launch | `AdvertiseAddr` in node Config + `-advertise` flag; wired into hub registration so public SSH address appears correctly in the directory. Hub `GET /v1/directory` (public, no auth): all nodes with online/offline status, player count, advertise_addr. Hub `GET /v1/status` (public): hub health, node counts, event total. `GetDirectory()` + `EventCount()` added to Store interface + SQLite backend. `TestDirectoryAndStatus` in hub_test. `Makefile`: `make build` (local), `make release` (CGO_ENABLED=0, linux/amd64 + linux/arm64 for both binaries, ~10 MB static). `files/SYSOP_GUIDE.md`: zero-to-node in under one hour — install, config, first run, systemd, firewall, verification, hub setup section, troubleshooting. | ✅ Build clean; all tests green; release binaries confirmed. |
| *(network launch complete — B2 polish / game content next)* | | |

---

*Update this file at the end of every slice. If the slice log stops growing, the
network has fallen off the radar — which is exactly what this file is here to catch.*
