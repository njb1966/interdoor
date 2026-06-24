# FEDERATION PROTOCOL

> Status: Historical game-specific source material. The root numbered InterDoor docs are canonical for Phase 1 framework, protocol, operator, and architecture decisions. Do not use this file as Phase 1 framework authority unless it has been explicitly reconciled.

## The InterDOOR Network Wire Contract

*Phase B3 deliverable (per PROJECT_PLAN.md). Status: v0.1 — spec before code.*

---

## 0. What this document is

This is the protocol that lets independently-operated **InterDOOR nodes** share state:
cross-node player identity, a distributed favor/debt Ledger, cross-node PvP and travel,
and a network-wide event feed. It is the spec the **hub** and the **node sync client**
are built against in Phase B3.

It is deliberately grounded in what already exists in the single-node engine — the
protocol is mostly *plumbing the primitives we already built across the wire*:

| Already built (single-node) | Its role in federation |
|---|---|
| `Store.Emit` / append-only `events` (`source_node:seq`) | The unit of propagation |
| `Store.EventsSince(node, afterSeq)` | The **outbound** push source |
| `Store.ApplyEvent` (idempotent, dedup by `event_id`, `OnEvent` handlers) | The **inbound** apply path |
| Obligations (`source_node:o_N`, `debt.*` events) | The distributed Ledger |
| `ExportPlayer`/`ImportPlayer` snapshots (vouched credential) | Cross-node travel |
| Daily turn reset | The sync boundary (Req 3) |
| Global IDs everywhere (`node:p_xxx`) | Portable identity (Req 2) |

If a behavior here isn't backed by one of those, it's called out as new work.

---

## 1. Architecture & trust model

**Hub-and-spoke** (PROJECT_PLAN; NETWORK_REQUIREMENTS implies it). One **hub** is the
authoritative coordinator; **nodes** are the game servers players connect to.

```
        ┌────────────── hub store ─────────────┐
        │  master event feed · roster · Ledger  │
        │  pvp queue · travel queue · locations │
        └───▲────────────▲─────────────▲────────┘
            │ REST/HTTPS  │             │
        ┌───┴───┐     ┌───┴───┐     ┌───┴───┐
        │ node01│     │ node02│     │ node03│   (each: SQLite + game)
        └───────┘     └───────┘     └───────┘
```

- **Nodes are independent.** They run the full game locally and never depend on the hub
  to serve a player (Req 6). The hub is for *sharing*, not *serving*.
- **The hub is authoritative** for cross-node disputes (Req 5): the master event feed,
  the master Ledger index, and the player-location projection used by travel.
- **Trust:** the hub trusts a registered node for events whose `source_node` is that
  node, and nothing else (§13). Nodes trust the hub's relay. End-to-end node signing of
  snapshots/events is a future hardening (§18).
- **Not peer-to-peer.** Nodes never talk directly; all coordination is via the hub. This
  is the deliberate choice that keeps a node operator's setup to "point at a hub URL."

---

## 2. Identifiers & versioning

| Identifier | Format | Source |
|---|---|---|
| Node ID | lowercase alnum, network-unique (e.g. `node01`) | operator-chosen, registered |
| Player global ID | `node_id:p_<base32>` | engine, immutable |
| Obligation ID | `source_node:o_<n>` | engine |
| Event ID | `source_node:<seq>` | engine, per-source monotonic |
| Hub sequence | `hub_seq` (int64, hub-assigned) | hub, total order of the feed |

**Versioning fields** (sent at registration and heartbeat):
- `protocol_version` — this document's version. Hub rejects incompatible nodes.
- `game_id` + `game_version` — Shared-Reference identity (Req 1). Roster and directory
  visibility may cross games. Tier 3 interactions that move or mutate character state
  require matching `game_id`; `game_version` is recorded for compatibility policy and
  operator visibility.

Base URL: `https://<hub-host>/v1/` (hub host is node config). All bodies are JSON.
Auth: `Authorization: Bearer <api_key>` on every call except `POST /register`.

---

## 3. State classification on the wire

Straight from DATA_MODEL.md, mapped to transport:

| Class | What | How it crosses |
|---|---|---|
| **Synced — Broadcast** | roster, obligations (Ledger), events | event feed (§6) + roster sync (§7) |
| **Synced — Snapshot** | full character state | travel transfer (§10), point-to-point via hub |
| **Local** | turn budget, today's Warren seed, NPC cursors, market | **never crosses** |
| **Shared-Reference** | item/creature/NPC/encounter catalogs | ships with the game build; identical per `game_version`; never synced at runtime |

The single most important rule: **Local state never leaves the node, and
Shared-Reference state never travels at runtime.** Only Broadcast (small) and Snapshot
(on travel only) move.

---

## 4. Registration & authentication

**`POST /v1/register`** (unauthenticated)
```json
{ "node_id": "node02", "registration_token": "<operator-issued>",
  "game_id": "ledger_of_the_low", "game_version": "1.0.0",
  "protocol_version": "1", "advertise_addr": "ssh://node02.example:2323" }
```
→ `200 { "api_key": "<secret>", "hub_seq_head": 0 }` or `409` (node_id taken) /
`401` (bad token) / `426` (incompatible protocol).

- The hub operator issues a `registration_token` out of band. On success the
  hub stores the node and returns a long-lived `api_key` (the node persists it).
- `advertise_addr` is how the hub/other players can reach the node (for the node
  directory in B4); optional.

**Heartbeat** keeps liveness and surfaces pending work:

**`POST /v1/heartbeat`**
```json
{ "node_id": "node02", "player_count": 12, "uptime_s": 86400, "game_version": "1.0.0" }
```
→ `200 { "hub_seq_head": 91827, "pending": { "events": 14, "pvp": 1, "travel": 0 } }`

This is the `node.status` event made into a request/response. Cadence: every 30–60 s.
`pending` lets a node decide whether to pull now rather than wait for its poll tick.

---

## 5. The event feed (the core of federation)

Everything Broadcast-worthy is an event. Federation = propagating the append-only log.

### 5.1 Push (outbound)

A node pushes its locally-emitted events — exactly `EventsSince(myNode, lastPushedSeq)`.

**`POST /v1/events`**
```json
{ "events": [
  { "event_id": "node02:41", "source_node": "node02", "seq": 41,
    "type": "debt.created", "ts": 1750600000,
    "payload": { "obligation_id": "node02:o_7", "creditor_ref": "node02:p_a",
                 "debtor_ref": "node01:p_b", "kind": "debt", "weight": 22 } }
] }
```
→ `200 { "accepted": 1, "duplicates": 0, "last_hub_seq": 91828 }`

- **The hub rejects any event whose `source_node` ≠ the authenticated node** (`403`).
  A node may only speak for itself. This is the spine of the trust model.
- The hub **dedups by `event_id`** (idempotent — re-pushing after a flaky connection is
  safe) and assigns a monotonic `hub_seq` to each newly accepted event.
- The node advances `last_pushed_seq` to the highest `seq` it has confirmed delivered.
- Nodes SHOULD push in `seq` order per source.

### 5.2 Pull (inbound)

**`GET /v1/events?after=<hub_cursor>&limit=500&exclude_self=true`**
→
```json
{ "head": 91900,
  "events": [ { "hub_seq": 91829, "event_id": "node01:55", "source_node": "node01",
                "seq": 55, "type": "player.died", "ts": 1750600100,
                "payload": { "...": "..." } } ] }
```

- Returns feed events with `hub_seq > hub_cursor`, ordered by `hub_seq`, capped at
  `limit`. `exclude_self` omits the caller's own events (it already has them).
- The node applies each via **`ApplyEvent`** (idempotent; `event_id` dedup) and advances
  its `hub_cursor` to the last processed `hub_seq`. Event receipt and handler application
  are tracked separately so failed handlers retry on replay without duplicating the event
  row.

### 5.3 Ordering & idempotency guarantees

- **Idempotent everywhere** (Req 4): dedup by `event_id` on push (hub) and apply (node).
  At-least-once delivery is safe. Handler status tracking prevents rerunning handlers
  already marked applied; handlers must still make their own writes idempotent.
- **Total order** of the feed is `hub_seq`. **Per-source order** is preserved because a
  node pushes its own events in `seq` order. Consumers must tolerate cross-source
  interleaving (events are self-contained facts).
- **Gaps:** a node pulls contiguously by `hub_seq`, so it never has feed gaps. Strict
  per-source gap-queuing (if a node ever pushed out of order) is a hub-side option, not
  required for v1.

### 5.4 Handlers that materialize foreign state

Applying a foreign event must update local Broadcast tables so local players see it.
Nodes register `OnEvent` handlers in B3:

| Event | Handler effect on the receiving node |
|---|---|
| `player.created` / `player.died` | upsert roster spine (also covered by §7) |
| `debt.created` | if `debtor_ref` or `creditor_ref` is a local player → insert the obligation locally (§8) |
| `debt.adjusted` | update the local mirror weight for a partial repayment |
| `debt.resolved` / `debt.transferred` | update the local mirror of that obligation |
| `pvp.resolved` | if `attacker_global_id` is a local player → credit the looted goods (§9) |
| `endgame.completed` (post-v1) | apply node-wide consequence locally |

Handlers are game/engine code; this table is their contract.

---

## 6. Roster sync

The roster is Broadcast identity spine (DATA_MODEL §1.1): who exists, where, their level
(for PvP eligibility) and standing. `player.created`/`died` events cover lifecycle; a
periodic roster push keeps mutable spine fields fresh.

**`POST /v1/roster`** — a node uploads the spine of players it owns (`home_node == self`):
```json
{ "players": [ { "global_id": "node02:p_a", "name": "Maren", "home_node": "node02",
                 "level": 4, "standing": 0, "status": "active", "last_seen": 1750600000 } ] }
```
**`GET /v1/roster?since=<ts>`** → the network roster (optionally only rows changed since
`ts`). The hub merges by latest `last_seen`. Nodes use it to render cross-node PvP target
lists and "who's out there." Rows older than a freshness threshold are shown **stale**.

---

## 7. Cross-node obligations (the distributed Ledger)

The favor/debt economy is the network's differentiator, and it already crosses cleanly
because obligations are globally IDed and event-backed.

- A debt created on node A where `debtor_ref` is a **node-B** player: A emits
  `debt.created` → feed → B's handler inserts the obligation into B's local
  `obligations` table, so the B player sees their debt and `DebtLoad` reflects it.
- Partial repayment and resolution propagate the same way (`debt.adjusted` /
  `debt.resolved`).
- **The hub maintains a master obligations index** by applying `debt.*` events to its own
  table. On any divergence, **the hub's record is authoritative** (Req 5). A node that
  believes a debt is resolved but the hub hasn't seen the resolution treats it as still
  open until confirmed — the documented eventual-consistency window.
- Cross-node resolution has **no real-time coordination**: the creditor calls it in
  locally (atomic), emits `debt.resolved`, and the debtor's node updates on next pull.
  Minutes-to-hours of delay is a valid state, not a bug (Req 5/6).

---

## 8. Cross-node PvP

PvP already resolves against **stored, offline** victim state and emits `pvp.resolved`
(B1.3). Across nodes, resolution must happen on the **victim's home node**, so the attack
is routed through the hub.

**Flow:**
1. Attacker on node A picks a target who is a node-B player (visible via roster) and
   spends 1 attack locally.
2. **`POST /v1/pvp`** — A submits the attack with the attacker's combat-relevant state:
   ```json
   { "attacker_id": "nodeA:p_x", "victim_id": "node02:p_y",
     "attacker": { "name":"Rook","level":4,"hp":40,"strength":24,"defense":11,"luck":7,
                   "weapon_bonus":9,"armor_bonus":1 } }
   ```
   → `200 { "request_id": "...", "status": "queued" }`. The hub validates the victim's
   home node and queues the request for node B.
3. **`GET /v1/pvp/pending`** — node B pulls queued attacks on its players, resolves each
   **locally** against the stored victim (reusing the B1.3 bout engine + the supplied
   attacker stats), deducts loot from the victim, writes the victim's "while you slept"
   inbox, and **emits `pvp.resolved`** (carrying `attacker_global_id`, `victim_global_id`,
   `loot`, `victim_died`).
4. **`POST /v1/pvp/{request_id}/result`** — B closes the request at the hub.
5. The `pvp.resolved` event propagates via the feed to A, whose handler **credits the
   looted goods** to the attacker. Cross-node loot conservation = victim node removes +
   attacker node adds, tied together by the single idempotent event.

**Offline victim node:** the request simply waits in the hub queue until B next pulls
(Req 6). The attacker sees a `pending` state until the `pvp.resolved` event returns.
Anti-grief limits (cooldown, level floor, victim cap, blooded-grace) are enforced on the
**victim's** node at resolution time, using roster level + local history.

---

## 9. Cross-node travel

Travel moves a **Snapshot** (B1.H2) point-to-point via the hub, with the hub enforcing
that a character is active on exactly one node at a time.

**Flow:**
1. Player on current node A chooses to travel to node B. The hub requires matching
   `game_id` for travel unless a future compatibility map allows otherwise.
2. A `ExportPlayer` → snapshot, sets the player `status = traveling` (locks local login),
   and **`POST /v1/travel`** `{ snapshot, dest_node: "node02" }` → `{ travel_id }`. The
   hub records the character as **in transit** (current location = none).
3. **`GET /v1/travel/pending`** — node B pulls arrivals, `ImportPlayer` (visiting; does
   **not** take home ownership — `home_node` stays A), then
   **`POST /v1/travel/{id}/arrived`**. If import fails, the request remains pending.
   The hub sets current location = B and emits `player.traveled`.
4. The visiting player plays on B with a **local** turn budget (turn state is Local). B
   owns only a copy; on departure B exports the updated snapshot back via `POST /v1/travel`
   with `dest_node = home`, and A re-imports. The hub flips location back to A.

**Single-active invariant:** the hub's player-location projection is the source of truth. While
in transit or visiting, the home node refuses local login (`status = traveling`). This
prevents the character being played on two nodes — the one place travel could corrupt
state.

---

## 10. Conflict resolution

Hub-authoritative, **last-write-wins per object** (Req 5), with object-specific rules:

| Object | Rule |
|---|---|
| Events | No conflict — append-only, idempotent dedup by `event_id`. |
| Obligations | Hub master index authoritative. Lifecycle is monotonic (`open → resolved`); a `debt.resolved` wins once the hub has it. |
| Roster spine | LWW by `last_seen` (latest push wins) per field. |
| Character location (travel) | Hub player-location projection is absolute; conflicting claims rejected. |

No two nodes ever own the same mutable object simultaneously: obligations have a single
`source_node`; a character has a single current node. That ownership discipline is
what makes LWW safe here.

---

## 11. Partition tolerance (Req 6)

- A node **fully operates offline** — local play is unaffected.
- **Outbound** events accumulate in the local `events` table; on reconnect the node
  pushes everything since `last_pushed_seq`.
- **Inbound** — on reconnect the node pulls from its `hub_cursor` and applies the backlog
  in order (idempotent).
- **Cross-node requests to an offline node** (pvp/travel) queue at the hub until it
  returns. The initiator sees a visible `pending` state ("their node is unreachable").
- **Stale data** (roster rows past a freshness threshold) is flagged stale in-game.

Worked sizes (DATA_MODEL sync profile): a 1-week partition for a 50-DAU node is ~0.7 MB
of backlog — sub-second to replay.

---

## 12. Transport, security, TLS

- **Transport:** REST/JSON over HTTPS is the baseline (partition-friendly, trivial to
  operate). **Optional WebSocket** from hub→node pushes a "you have pending work" nudge to
  active nodes to cut poll latency; it never carries authoritative state — the node still
  pulls via REST. A node that only polls REST is fully functional.
- **TLS:** Caddy reverse proxy terminates TLS at the hub (PROJECT_PLAN). SSH (player↔node)
  handles its own encryption separately.
- **Node auth:** bearer `api_key` per node; hub stores only a hash. Revocable.
- **Event authenticity:** `source_node` must equal the authenticated node (§5.1) — a node
  cannot forge another's events.
- **Snapshots:** the home node vouches via the credential hash inside the snapshot; the
  hub relays. Cryptographic node-signing of snapshots/events is **future hardening** (§17)
  — until then trust is "registered node + hub relay."
- **Rate limiting** per node on push/pull/pvp/travel; abuse → throttle or deregister.

---

## 13. Hub data model

| Table | Key columns |
|---|---|
| `nodes` | `node_id` PK, `api_key_hash`, `game_id`, `game_version`, `protocol_version`, `advertise_addr`, `last_heartbeat`, `status` |
| `events` | `hub_seq` BIGSERIAL PK, `event_id` UNIQUE, `source_node`, `seq`, `type`, `ts`, `payload` JSONB |
| `roster` | `global_id` PK, `name`, `home_node`, `level`, `standing`, `status`, `last_seen` |
| `obligations` | `obligation_id` PK, `source_node`, `creditor_ref`, `debtor_ref`, `kind`, `weight`, `status`, `updated_at` (master index, applied from `debt.*`) |
| `pvp_requests` | `request_id` PK, `attacker_id`, `victim_id`, `victim_node`, `attacker_payload` JSONB, `status`, `created_at` |
| `travel` | `travel_id` PK, `global_id`, `from_node`, `to_node`, `snapshot` JSONB, `status`, `created_at` |
| `player_locations` | `global_id` PK, `current_node`, `home_node`, `status`, `travel_id`, `updated_at` |

The `events` table is the spine; `roster`/`obligations` are projections the hub maintains
by applying events (plus roster pushes), so they can always be rebuilt from the log.

---

## 14. Node-side sync client (implementation shape)

New in B3 on each node (engine, generic):
- A `sync_state` table: `hub_cursor` (last applied `hub_seq`), `last_pushed_seq`, `api_key`.
- A sync goroutine loop (interval = config, default 15–30 s; or WebSocket-nudged):
  1. **Heartbeat** → read `pending`.
  2. **Push** `EventsSince(self, last_pushed_seq)`; advance on ack.
  3. **Pull** `GET /events?after=hub_cursor`; `ApplyEvent` each; advance cursor.
  4. **Roster** push (own players) + pull (network) on a slower cadence.
  5. **Drain** `pvp/pending` and `travel/pending`; resolve/import locally.
- Registered `OnEvent` handlers (§5.4) to materialize foreign Broadcast state.
- All of this is **engine-generic** — a different game on InterDOOR reuses it unchanged.

---

## 15. API reference (summary)

| Method | Path | Auth | Purpose |
|---|---|---|---|
| POST | `/v1/register` | token | Join the network, get `api_key` |
| POST | `/v1/heartbeat` | key | Liveness + pending counts |
| POST | `/v1/events` | key | Push own events (source must match) |
| GET | `/v1/events?after=…` | key | Pull the feed since a cursor |
| POST | `/v1/roster` | key | Upload owned-player spine |
| GET | `/v1/roster?since=…` | key | Download network roster |
| POST | `/v1/pvp` | key | Submit a cross-node attack |
| GET | `/v1/pvp/pending` | key | Pull attacks on my players |
| POST | `/v1/pvp/{id}/result` | key | Report resolution |
| POST | `/v1/travel` | key | Send a character snapshot to a node |
| GET | `/v1/travel/pending` | key | Pull arriving characters |
| POST | `/v1/travel/{id}/arrived` | key | Confirm arrival |

---

## 16. Sync cadence & bandwidth

- Heartbeat: 30–60 s. Event push/pull: 15–30 s (or WebSocket-nudged). Roster: 1–5 min.
- The **daily turn reset is the natural minimum** sync boundary (Req 3); everything above
  it is latency comfort, not correctness.
- Bandwidth is trivial (DATA_MODEL profile: ~100 KB/day for a 50-DAU node). The big data
  (content catalogs) never syncs.

---

## 17. Open questions / future hardening

- **Cryptographic signing** of events and snapshots (node keypairs) so the hub/peers can
  verify authenticity without fully trusting the relay. v1 uses registered-node + hub
  trust.
- **Multi-game hub:** the protocol is game-agnostic; a hub could federate multiple games
  keyed by `game_id`. Roster/Ledger/PvP/travel are all `game_id`-scoped.
- **Hub federation / failover:** v1 is a single hub. Multiple cooperating hubs or hub HA
  is post-launch.
- **Strict per-source gap-queuing** if out-of-order push ever becomes a real concern.
- **Game-version migration** of in-flight snapshots across incompatible catalogs.

---

## 18. B3 build order (maps to PROJECT_PLAN)

1. **This spec** ✅ (v0.1).
2. Hub skeleton — `/register`, `/heartbeat`, store schema (§13).
3. Event feed — `/events` push+pull; node sync client push/pull loop (§5, §14).
4. Roster sync (§6).
5. Cross-node obligations — `debt.*` handlers + hub master index (§7).
6. Cross-node PvP — pvp queue + handlers (§8).
7. Cross-node travel — snapshot relay + single-active invariant (§9).
8. Conflict resolution + partition tests between ≥2 local nodes (§10–11).
9. Third-party node onboarding doc (→ Phase B4).

---

*Document version 0.1 — Federation protocol. Built on the single-node primitives already
shipped (idempotent event log, globally-IDed Ledger, portable snapshots, daily-turn
boundary). Revised as the hub implementation reveals details.*
