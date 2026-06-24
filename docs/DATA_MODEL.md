# DATA MODEL

> Status: Historical game-specific source material. The root numbered InterDoor docs are canonical for Phase 1 framework, protocol, operator, and architecture decisions. Do not use this file as Phase 1 framework authority unless it has been explicitly reconciled.

## Game State Structures, Classified for Federation

*Stage 4 deliverable (per GAME_DESIGN_PROCESS.md). Status: v0.1.*

---

## Framing: this is the network's data contract, demonstrated

**The network is the project. The game is the proof.** This document is written from
that order of priorities. It is not "the game's database schema." It is a concrete
demonstration that the engine's generic state-classification contract — Local /
Synced / Shared-Reference, defined in NETWORK_REQUIREMENTS.md — is sufficient to
express a real, federation-ready game.

Two consequences follow, and they shape every decision below:

1. **Generic vs. game-specific is marked throughout.** Anything labeled *engine* is a
   structure or event any terminal game on the network would use (players, sessions,
   obligations, events, the daily turn). Anything labeled *game* is the reference
   game's own content. A second game built on this network reuses the engine
   structures and supplies its own game structures. The engine never learns what a
   "Warren" is.

2. **Names are placeholders.** The network is `<NETWORK>`; the reference game is
   `<GAME>`. Neither name is decided. Nothing here bakes one in. When a name becomes
   load-bearing (e.g., the `node_id` namespace, the game-module identifier registered
   with the engine, or the on-disk game-version string), this document flags it as a
   **NAME DECISION POINT** rather than inventing a value.

---

## The LoRD principle (why the player record is tiny)

The reference game is, by deliberate constraint, *very* LoRD-like: a daily-turn
survival RPG, ~12 actions/day, no rollover, 10–15 minute sessions, the world evolves
while you are logged off. If it gets complicated, people stop logging in daily, and a
game nobody plays daily cannot prove a federated daily-turn network works.

The data-model expression of that constraint:

> **Keep the per-player synced footprint LoRD-small. Put all depth and character in
> Shared-Reference content.**

In LoRD a player is a dozen fields (name, class, level, HP, weapon/armor tier, gold,
fights left, a handful of flags). The depth and *character* of LoRD live in its
content — Violet, Seth, the bartender, the dragon, the writing — not in a complex
player record. We copy that exactly. It is also what federation wants: a small synced
spine that is cheap to move between nodes, and a fat content catalog that ships with
the game version and never syncs at runtime.

If a proposed mechanic would make the player record big or chatty, that is a signal to
move its richness into content or into local state — not to grow the synced spine.

---

## The classification scheme (and one refinement)

NETWORK_REQUIREMENTS.md defines three categories. We use them exactly, with one
sub-distinction inside **Synced** that the data model must make explicit because the
two sub-modes have very different cost and behavior:

| Class | Sub-mode | Who sees it | When it moves | Example |
|---|---|---|---|---|
| **Local** | — | One node only | Never | Today's Warren seed, NPC dialogue cursor, actions remaining |
| **Synced** | **Broadcast** | Hub + all nodes | Continuously / on change / at turn boundary | Player roster, obligations (the Ledger), event log |
| **Synced** | **Snapshot** | Only the destination node | Only when a player travels | Full character state (stats, inventory) |
| **Shared-Reference** | — | All nodes (identical) | Only on game-version update | Item / creature / encounter / NPC / contract catalogs |

**Why the Snapshot sub-mode exists.** Character state is owned by the home node
(NETWORK_REQUIREMENTS.md, Requirement 2). It does not broadcast — that would be
enormous and pointless, since no other node needs your inventory until you actually
stand on their node. It transfers as a one-time snapshot at travel time, and the
visiting node does not permanently store it. Collapsing Snapshot into Broadcast would
misrepresent the player record as something every node must hold, which is false and
would make running a node look far more expensive than it is.

**The classification test** (applied to every structure below): (1) Does another node
ever need this? No → Local. (2) Does it change during play? No → Shared-Reference.
(3) Yes to both → Synced — then ask: does *every* node need it (Broadcast) or only a
node the player visits (Snapshot)?

---

# Part 1 — Entities (v1)

Identifiers used throughout:

- **Node ID** — short, stable, registered with the hub. **NAME DECISION POINT**: the
  node-ID namespace and the example value (docs currently use `haven01`) are not
  locked. Treated here as an opaque string `node_id`.
- **Global player ID** — `node_id:player_id`, e.g. `node_id:p_3f8a`. Assigned at
  creation, never changes, even across travel (NETWORK_REQUIREMENTS.md, Req 2).
- **Obligation ID** — `source_node:o_<id>`, globally unique (Req 5).
- **Event ID** — `source_node:seq`, where `seq` is per-node monotonic (Req 4).
- **Catalog ID** — stable string key, e.g. `wpn_rusted_shiv`, `crt_gutter_eel`. Ships
  with the game version; identical on every node running that version.
- **NPC ref** — `npc:<npc_id>` (NPCs are shared-reference; an NPC creditor/debtor in
  the Ledger is referenced by catalog ID, optionally `@node_id` if node-scoped).

---

## 1.1 Player identity — *engine, Synced (Broadcast)*

The minimal portable spine. This is what the hub roster holds and what other nodes can
see about a player without the player being present. Kept deliberately thin.

| Field | Type | Class | Notes |
|---|---|---|---|
| `global_id` | string (PK) | Broadcast | `node_id:player_id`. Immutable. |
| `name` | string | Broadcast | Display name. Uniqueness enforced by hub network-wide; per-node in single-node v1. |
| `home_node` | string | Broadcast | Node that owns this character's state. |
| `created_at` | timestamp (UTC) | Broadcast | |
| `status` | enum(`active`,`dead`,`traveling`) | Broadcast | Coarse liveness for cross-node display and PvP eligibility. |
| `level` | int (1–10) | Broadcast | Mirrored from `player_state.level`. Needed cross-node for PvP level-gap rules without transferring full state. |
| `standing` | int | Broadcast | Global notability. **0 / unused in v1** (post-v1 system). Present in the spine now so federation visibility never needs reworking. |
| `last_seen` | timestamp (UTC) | Broadcast | Drives "who's active" and stale-data flagging after a partition. |

Size: ~150–250 bytes/player. A 1,000-player network roster is well under 250 KB.

> **Identity vs. state split.** `level` and `standing` are duplicated into the
> Broadcast spine on purpose: they are the only character attributes other nodes need
> for routine display/eligibility, so we publish just those two rather than the whole
> Snapshot record. Everything else about the character lives in §1.2 and only moves on
> travel.

---

## 1.2 Player character state — *engine, Synced (Snapshot)*

Owned by `home_node`. Authoritative store lives there; transferred as a one-time
snapshot when the player travels (Phase B3). This is the LoRD-tiny record.

| Field | Type | Class | Notes |
|---|---|---|---|
| `global_id` | string (PK, FK→identity) | Snapshot | |
| `hp` | int | Snapshot | Current health. |
| `max_hp` | int | Snapshot | |
| `strength` | int | Snapshot | Core combat stat (offense). |
| `defense` | int | Snapshot | Core combat stat. |
| `luck` | int | Snapshot | Modifies crits, flee, scavenge, loot rolls. |
| `level` | int (1–10) | Snapshot | Source of truth; mirrored to identity. |
| `level_progress` | int | Snapshot | Internal counter toward next level (action/fight/contract driven, *not* an XP bar shown as a number — see BUILD_SPEC). |
| `weapon_item` | string (FK→item_catalog) \| null | Snapshot | Equipped weapon. |
| `weapon_condition` | int (0–`max_condition`) | Snapshot | Degradation. |
| `armor_item` | string (FK→item_catalog) \| null | Snapshot | Equipped armor. |
| `armor_condition` | int | Snapshot | Degradation. |
| `depth_record` | int | Snapshot | Deepest Warren depth reached. Bragging right + access gate. |
| `death_penalty_active` | bool | Snapshot | True for one cycle after death → slower `level_progress` gain (no level loss). |
| `flags` | json / bitset | Snapshot | Small set of one-shot booleans: `tutorial_done`, `first_market_visit`, etc. Keep this small — it is not a general key/value dumping ground. |

**Derived (not stored):**
- `debt_load` = Σ `weight` of obligations where `debtor_ref == global_id` AND
  `status == open`. Computed on read from §1.3. Drives NPC friction (merchants charge
  more, the Broker comments) per BUILD_SPEC.

Size: ~300–500 bytes + inventory (§1.4). Full character snapshot is a few KB.

---

## 1.3 Obligation (the Ledger) — *engine, Synced (Broadcast)*

The favor/debt economy. **One entity, two directions.** A row is a *favor* when you
are the `creditor_ref` and a *debt* when you are the `debtor_ref`; the UI renders the
same row from the viewer's perspective. This is the network's differentiator and its
cross-node centerpiece, so it is Broadcast and globally identified from day one even
though v1 obligations are NPC-only.

| Field | Type | Class | Notes |
|---|---|---|---|
| `obligation_id` | string (PK) | Broadcast | `source_node:o_<id>`. Globally unique (Req 5). |
| `source_node` | string | Broadcast | Authoritative origin. Conflicts resolve in favor of the hub's record (Req 5). |
| `creditor_ref` | string | Broadcast | Global player ID or `npc:<id>`. The party owed. |
| `debtor_ref` | string | Broadcast | Global player ID or `npc:<id>`. The party owing. |
| `kind` | enum(`favor`,`debt`,`contract`) | Broadcast | `favor`/`debt` are framing of the same obligation; `contract` is a contract-backed obligation (§1.7). v1 restricts at least one side to an NPC. |
| `terms` | string | Broadcast | What is owed. **Free text in v1** (e.g. "safe passage owed to Maren"). Structured/typed in a later phase — kept as text now to avoid premature schema. |
| `weight` | int | Broadcast | Magnitude. Favor weight scales with the creditor's importance; debt weight feeds `debt_load`. |
| `status` | enum(`open`,`resolved`,`transferred`,`defaulted`) | Broadcast | |
| `created_at` | timestamp (UTC) | Broadcast | |
| `resolved_at` | timestamp (UTC) \| null | Broadcast | |
| `resolution` | enum(`paid`,`forgiven`,`defaulted`) \| null | Broadcast | Set when `status` leaves `open`. |

**Cross-node reconciliation note.** A debt created on Node A and owed by a player on
Node B resolves when the creditor calls it in on their node (an atomic local action)
and the `debt.resolved` event syncs to B and the hub. Between those, B may still show
it open. This delay is a valid game state, not a bug (Req 5, Req 6). v1 is single-node
so the delay is zero, but the structure assumes it.

Size: ~250–400 bytes/obligation.

---

## 1.4 Inventory item instance — *engine, Synced (Snapshot)*

A player's concrete item. References a shared-reference template (§2.1) plus
per-instance condition. Travels with the character snapshot.

| Field | Type | Class | Notes |
|---|---|---|---|
| `instance_id` | string (PK) | Snapshot | Node-local unique; namespaced under owner on travel. |
| `owner_global_id` | string (FK→identity) | Snapshot | |
| `item_id` | string (FK→item_catalog) | Snapshot | Template. |
| `location` | enum(`carried`,`banked`,`equipped`) | Snapshot | **Carried** = at risk on death/PvP; **banked** = safe (Rafters); **equipped** = weapon/armor slot. |
| `condition` | int (0–`max_condition`) | Snapshot | Degradation; ignored if template `degrades=false`. |
| `qty` | int | Snapshot | For stackable consumables/trade goods; 1 for unique gear. |

**Capacity:** carried inventory is slot- or weight-limited (value locked in Stage 1 /
BUILD_SPEC revision); banked is large/effectively unlimited. Capacity limit is game
config, not a stored per-item field.

---

## 1.5 Turn state — *engine, Local*

The daily action economy and the "world evolves offline" clock. Local: no other node
needs your remaining actions.

| Field | Type | Notes |
|---|---|---|
| `owner_global_id` | string (PK) | |
| `actions_remaining` | int | Default 12. No rollover. |
| `last_reset` | timestamp (UTC) | Server-midnight reset logic (configurable). |
| `day_index` | int | Increments each reset. The heartbeat that drives offline-world evolution (contract refresh, market refresh, dialogue rotation, debt interest). |

> The daily turn is an **engine** concept — any game on the network gets a turn/action
> economy from the engine. The *number* (12) and the *cost table* are game config.

---

## 1.6 Local session & world state — *game, Local*

Everything a single node generates for play and never needs to share. Grouped; most of
this is small and some is in-memory only.

| Structure | Key fields | Persisted? | Notes |
|---|---|---|---|
| `npc_dialogue_state` | `owner_global_id`, `npc_id`, `dialogue_cursor`, `lines_seen` (bitset), `last_rotation_day` | Yes | Per-player, per-NPC. Drives daily-rotating dialogue and arc progress. |
| `market_state` | `node`, `merchant_id`, `item_id`, `price_terms`, `qty`, `day_index` | Yes | Today's merchant inventory. Refreshes on `day_index` change. |
| `daily_offers` | `contract_instance_id`, `template_id`, `params`, `posted_day`, `expires_day`, `status` | Yes | The 2–3 contracts available today (§1.7). |
| `warren_session` | `seed`, `depth`, `encounter_cursor`, `expedition_log` | In-memory (per session) | One Warren expedition. The seed makes the encounter sequence reproducible; only *results* that touch synced state (death, notable loot, a created debt) leave this structure as events. |
| `pvp_inbox` | `id`, `victim_global_id`, `attacker_global_id` (global), `outcome`, `loot_taken[]`, `resolved_day`, `seen` | Yes | "While you slept…" notifications. Resolved on the **victim's** home node. Attacker referenced by global ID so the same structure carries cross-node PvP in B3 (currently same-node only). |

---

## 1.7 Contract instance — *engine shell, game content*

A contract is an engine concept (an offer with an objective and a reward, possibly
producing an obligation); its *objectives and flavor* are game content (§2.6). v1
contracts are NPC-posted only.

| Field | Type | Class | Notes |
|---|---|---|---|
| `contract_instance_id` | string (PK) | Local (v1) | Same-node contracts are Local. Cross-node contracts become Broadcast in B3 and must then use global IDs + offer/accept events. |
| `template_id` | string (FK→contract_template) | Shared-Reference (the template) | |
| `poster_ref` | string | Local/Broadcast | NPC ref in v1. |
| `holder_global_id` | string \| null | Local | Player who accepted; null if open. |
| `params` | json | Local | Instantiated objective (which item, where, how many). |
| `posted_day` / `expires_day` | int | Local | Expiration by `day_index`. |
| `status` | enum(`open`,`accepted`,`completed`,`expired`,`failed`) | Local | |
| `reward_obligation_id` | string \| null | Broadcast (if used) | If the reward is a favor/debt, it is a §1.3 obligation. |

> **Federation seam, explicitly deferred.** Same-node contracts are Local. Cross-node
> contracts (B3) must be Broadcast, carry global IDs, and decompose into
> offer/sync/accept/sync events (NETWORK_REQUIREMENTS.md, Req 3). v1 does not build
> this; the entity is shaped so it does not need redesign when B3 arrives.

---

# Part 2 — Shared-Reference content (v1)

These are the **content catalogs**. They ship inside the game package, are identical on
every node running a given game version, and **never sync at runtime** — a node update
is the only thing that changes them. This is where the LoRD-style depth and character
live. Field lists below are the structural template (the "fields that define one"); the
actual content volume is the Stage 6 CONTENT_PLAN.md job, with targets already in
BUILD_SPEC_V1.md.

A **NAME DECISION POINT** applies to the package as a whole: the on-disk
`game_version` / game-module identifier string. Nodes must agree on it for
shared-reference state to align (Req 1). Left unspecified here.

## 2.1 `item_catalog`
`item_id`, `name`, `category`(`weapon`/`armor`/`consumable`/`trade`/`curio`),
`tier`(1–10 gate), `stat_mods`(json: str/def/luck/hp deltas), `trade_weight`(barter
value), `degrades`(bool), `max_condition`, `carry_cost`(slots/weight),
`description_ref`.

## 2.2 `creature_catalog`
`creature_id`, `name`, `hp`, `strength`, `defense`, `behavior`(`aggressive`/
`defensive`/`evasive`), `depth_band`, `loot_table_id`, `narration_pool_id`,
`description_ref`.

## 2.3 `loot_table`
`loot_table_id`, weighted entries `[{item_id, qty_min, qty_max, weight}]`. Luck
modifies rolls.

## 2.4 `encounter_table`
`table_id`, `depth_band`, weighted entries
`[{type: combat|discovery|hazard|npc|anomaly, ref, weight}]`. Selected by Warren depth;
weights shift with depth so danger/reward escalate. The "shift between days" is the
per-day `seed` (§1.6) re-rolling against these tables — mechanical at the seed level,
cosmetic to the player.

## 2.5 `npc_catalog` & `dialogue_pool`
- `npc_catalog`: `npc_id`, `name`, `role`, `district`, `dialogue_pool_id`,
  `services`(json: buys/sells/gives-contracts/heals), `tone_tags`.
- `dialogue_pool`: `pool_id`, `lines[]` each `{text_ref, weight, trigger, rotation,
  arc_step}`. Supports daily rotation and simple multi-day arcs without per-player
  storage beyond the §1.6 cursor.

v1 cast (BUILD_SPEC): the Lamplighter, the Broker, Maren, Old Thursen.

## 2.6 `contract_template`
`template_id`, `type`(fetch/deliver/survive/explore), `objective_spec`(json,
parameterized), `reward_spec`(json: items/gear/debt-relief/obligation/lore),
`duration_days`, `flavor_ref`.

## 2.7 `district_def` & `text_pool`
- `district_def`: `district_id`, `name`, `functions[]`, `nav_links[]`,
  `ambient_pool_id`, `header_art_ref`.
- `text_pool`: `pool_id`, `lines[]` of ambient/atmosphere text.

v1 districts: Threshold, Lanternmarket, Warrens, Rafters.

---

# Part 3 — Event Catalog (4.2)

Events are the federation data contract (NETWORK_REQUIREMENTS.md, Req 4). Every event
is **self-contained** (no callback to the source node needed), **idempotent**
(`event_id` dedup), **ordered** (`source_node` + `seq`), and **typed**. The engine
provides the emission interface; the federation layer (B3) delivers. **In v1 these are
emitted to a local log only** — federation consumes the same log later, so emitting
them correctly now is what makes B3 cheap.

Common envelope on every event:
`{ event_id: "src:seq", source_node, seq, type, timestamp, idempotency_key, payload }`

## 3.1 Engine / standard events (generic — any game emits these)

| Type | Fires when | Payload | Consumers |
|---|---|---|---|
| `player.created` | Character created | `{global_id, name, home_node, created_at}` | Hub, all nodes (roster) |
| `player.died` | HP reaches 0 (combat/hazard/PvP) | `{global_id, cause:{type, source_ref, district, depth}, timestamp}` | Hub, all nodes |
| `player.traveled` | Player moves to another node (B3) | `{global_id, src_node, dest_node, snapshot_hash, timestamp}` | Hub, src + dest nodes |
| `node.status` | Periodic heartbeat | `{node_id, player_count, uptime_s, game_version, seq}` | Hub |
| `debt.created` | Obligation opened | `{obligation_id, source_node, creditor_ref, debtor_ref, kind, terms, weight, created_at}` | Hub, all nodes |
| `debt.adjusted` | Obligation weight changed, such as partial repayment | `{obligation_id, old_weight, new_weight, delta, reason}` | Hub, all nodes |
| `debt.resolved` | Obligation paid/forgiven/defaulted | `{obligation_id, resolution, resolved_at}` | Hub, all nodes |
| `debt.transferred` | Obligation's creditor changes | `{obligation_id, new_creditor_ref, transferred_at}` | Hub, all nodes |

## 3.2 Game-defined events (reference game adds)

| Type | Fires when | Payload | Consumers | Notes |
|---|---|---|---|---|
| `pvp.resolved` | An attack on a (sleeping) player is resolved | `{attacker_global_id, victim_global_id, outcome, loot:[{item_id, qty}], victim_died:bool, day_index}` | Hub, attacker + victim nodes | v1: same-node, local-only. B3: resolved on victim's home node, result event synced back to attacker's node. If `victim_died`, a `player.died` is also emitted. |

**Not events (intentionally):** the daily turn reset and the offline-world tick
(contract/market refresh, dialogue rotation, Warren re-seed) are driven by local
`day_index` and never leave the node. Promoting them to events would add network
traffic for state no other node needs.

## 3.3 Idempotency & ordering rules
- `idempotency_key` defaults to `event_id`. Reprocessing a seen key is a no-op.
- Receiving nodes apply events in `(source_node, seq)` order; out-of-order events queue
  until the gap fills (Req 4).
- Obligation lifecycle events are commutative under last-write-wins *only* within a
  single `obligation_id`; the hub record is authoritative on conflict (Req 5).

---

# Part 4 — Sync Profile (4.3)

Worked against a deliberately generous node: **50 daily-active players**.

**Daily Broadcast volume (rough):**

| Source | Est. events/day | Avg size | Subtotal |
|---|---|---|---|
| Logins / roster touch | ~50 (folded into heartbeats) | — | ~negligible |
| Deaths (`player.died`) | 10–25 | ~250 B | ~6 KB |
| Obligations (`debt.*`) | ~2–4 per active player | ~300 B | ~45 KB |
| PvP (`pvp.resolved`) | ~15–30 | ~350 B | ~10 KB |
| Heartbeats (`node.status`) | ~24–288 (hourly→5-min) | ~120 B | ~3–35 KB |
| **Total** | | | **~65–100 KB/day** |

Roster Broadcast: ~50 × ~200 B ≈ **10 KB**, sent on change or with heartbeats.
Snapshot transfers: only on travel (rare in early federation), a few KB each.

**Conclusion: bandwidth is trivial.** A node's full daily federation traffic is on the
order of ~100 KB. This is the expected result of the LoRD-tiny + fat-content design:
the big data (content catalogs) never syncs; the synced spine is small.

**Recommended sync intervals:**
- Active nodes: optional real-time WebSocket for event broadcast; otherwise poll
  every 5–15 min.
- The natural and sufficient boundary is the **daily turn reset** (Req 3). Hourly
  polling is comfortable headroom.

**Partition behavior (Req 6):**

| Outage | Queued backlog (≈) | Reconnect cost |
|---|---|---|
| 1 hour | ~4–8 KB | Instant replay |
| 1 day | ~65–100 KB | Sub-second replay |
| 1 week | ~0.5–0.7 MB | Trivial replay |

During a partition: local play continues unaffected; events queue locally with their
`seq`; on reconnect the node replays missed events in order and reconciles
obligations against the hub (hub authoritative). Cross-node interactions initiated
against an unreachable node (e.g. a PvP attack) sit in a visible **pending** state
until connectivity returns. Stale roster/player data is flagged stale in the UI.

---

# Part 5 — Post-v1 entities (classification only)

Per the agreed scope: these are **not modeled** yet — their fields belong to design
stages not yet locked (factions, knowledge, standing, endgame). They are listed here
*only* with their classification, ID convention, and event hooks, so that adding them
later does not force a rework of federation conventions or the synced spine.

| Future entity | Class | ID / ref | Event hook(s) | Spine impact |
|---|---|---|---|---|
| Faction reputation (per player × faction) | Local (computed per node) | `global_id × faction_id` | `faction.shift` (node-level standing only) | None — per-player rep stays local; only node faction standing broadcasts. |
| Faction definition | Shared-Reference | `faction_id` | — | Ships with content. |
| Standing (global notability) | Synced (Broadcast) | field on identity | — | **Already reserved** as `identity.standing` (§1.1); no new spine field needed. |
| Knowledge fragment (per player) | Snapshot | `global_id × fragment_id` | — | Rides the character snapshot; no broadcast. |
| Knowledge fragment definition | Shared-Reference | `fragment_id` | — | Ships with content. |
| Ledger-attention level (per player) | Local | field on local player session | — | Derived from standing + research; never leaves node. |
| Endgame resolution | Synced (Broadcast) | event-only | `endgame.completed` `{global_id, path, consequences}` | One broadcast event; per-node consequences applied locally. |

The takeaway: every post-v1 system either rides an existing spine field (`standing`),
rides the character snapshot (knowledge), stays local (faction rep, Ledger attention),
or is purely an event (`endgame.completed`). **None of them requires growing the
Broadcast spine.** That is the property the v1 model is built to guarantee.

---

## Exit-criteria check (against GAME_DESIGN_PROCESS.md Stage 4)

- [x] Every v1 entity has field names, types, and Local/Synced(Broadcast|Snapshot)/
      Shared-Reference classification.
- [x] Primary keys and global identifiers defined; player and obligation references use
      global IDs, never local row IDs.
- [x] Relationships defined (identity↔state↔inventory; obligation refs; instances→
      catalog templates).
- [x] Event catalog: types, payloads, fire conditions, consumers, idempotency.
- [x] Sync profile: per-node/day volume, intervals, partition behavior.
- [ ] **Open**: numeric balance values (carry capacity, action costs, stat ranges,
      level curve) — these belong to the Stage 1 lock / BUILD_SPEC revision, not the
      data model. Noted, not blocking.
- [ ] **Open NAME DECISION POINTS**: `node_id` namespace, `game_version` /
      game-module identifier. Flagged, not invented.

---

*Document version 0.1 — First data model. Models v1 concretely; post-v1 classified
only. Will be revised as Stage 1–3 design decisions lock numeric values and as B3
reveals federation details.*
