# Federation Protocol

## Wire Contract

InterDoor Phase 1 uses HTTP JSON. Nodes authenticate with bearer API keys after registration. The protocol version is currently `1`.

Events, roster entries, travel snapshots, and PvP requests use global IDs. A node ID is the prefix before `:` in player and obligation IDs.

Tier 3 mechanics that mutate or move character state across nodes require matching `game_id` unless a future compatibility map explicitly allows otherwise. Directory and roster visibility remain network-wide.

## Response Rules

Successful responses are JSON objects. Failed responses use the Phase 1 structured error shape:

```json
{
  "error": "human-readable error"
}
```

Future protocol revisions may add stable machine-readable fields such as `code` and `detail`, but
Phase 1 clients must tolerate the current single-field error object.

## Pagination and Cursors

`GET /v1/events` is cursor-based:

- `after` is the last observed `hub_seq`; omitted means `0`.
- `limit` defaults to `500`.
- `limit < 1` is treated as `1`.
- `limit > 1000` is capped at `1000`.
- Results are ordered by ascending `hub_seq`.

Phase 1 directory, roster, debt, PvP-pending, and travel-pending endpoints are unpaginated. They are
expected to remain small during Phase 1. If any of those lists grow enough to need pagination, the
endpoint must add cursor rules before it is considered production-ready at that scale.

## Identifiers

| Object | Format |
|---|---|
| Node | Registered lowercase identifier |
| Player | `<node_id>:p_<id>` |
| Obligation | `<source_node>:o_<n>` |
| Event | `<source_node>:<seq>` |
| Hub sequence | Integer `hub_seq` assigned by the hub |

## Registration

Request:

```json
{
  "node_id": "node01",
  "registration_token": "operator-token",
  "game_id": "ledger_of_the_low",
  "game_title": "Ledger of the Low",
  "game_version": "1.0.0",
  "protocol_version": "1",
  "advertise_addr": "ssh://node01.example:2323"
}
```

`game_id` is the stable protocol identifier used for compatibility checks. It should be a
lowercase machine identifier and should not be changed for display polish. `game_title` is optional
human-readable directory metadata; when omitted, hubs display `game_id`.

Response:

```json
{
  "api_key": "secret",
  "hub_seq_head": 0
}
```

## Heartbeat

Heartbeat updates liveness and returns hub-side queue information.

```json
{
  "node_id": "node01",
  "player_count": 4,
  "uptime_s": 3600,
  "game_version": "1.0.0"
}
```

Response includes `hub_seq_head` and pending counts for PvP and travel. The current response shape includes an `events` field in the pending object, but current code does not populate it.

## Event Feed

Events are the core broadcast mechanism.

Event shape:

```json
{
  "event_id": "node01:42",
  "source_node": "node01",
  "seq": 42,
  "type": "debt.created",
  "ts": 1750600000,
  "payload": {}
}
```

Event envelope fields:

| Field | Type | Required | Notes |
|---|---|---|---|
| `event_id` | string | yes | `<source_node>:<seq>` |
| `source_node` | string | yes | Must match the authenticated node on push |
| `seq` | integer | yes | Monotonic within the source node |
| `type` | string | yes | Standard or game-defined event type |
| `ts` | integer | yes | Unix timestamp |
| `payload` | object | yes | Event-specific JSON object |

Push:

```text
POST /v1/events
```

Pull:

```text
GET /v1/events?after=<hub_cursor>&limit=500&exclude_self=true
```

Rules:

- Nodes may only push events for their authenticated source node.
- Payloads must be JSON objects.
- Phase 1 standard event payloads are schema-validated before acceptance.
- Hub deduplicates by `event_id`.
- Hub assigns `hub_seq`.
- Pull returns events ordered by `hub_seq`.
- Nodes apply events idempotently.
- Event receipt and event-handler application are tracked separately on nodes so failed handlers can be retried without inserting duplicate events.

Handlers must remain idempotent because a handler can fail after doing some local work before the framework records it as applied.

## Roster

Roster sync publishes the small player identity spine needed for cross-node visibility.

Push:

```text
POST /v1/roster
```

Pull:

```text
GET /v1/roster?exclude_self=true
```

The roster is not a character snapshot. It should stay small and should not carry inventory, credentials, private state, or game-specific blobs.

Roster entry shape:

| Field | Type | Required | Notes |
|---|---|---|---|
| `global_id` | string | yes | `<node_id>:p_<id>` |
| `node_id` | string | server-set | Hub overwrites this with the authenticated node on push |
| `name` | string | yes | Display name |
| `level` | integer | yes | Minimal eligibility/display field |
| `status` | string | yes | `active`, `dead`, `traveling`, or game-compatible extension |
| `last_seen` | integer | yes | Unix timestamp |

Stale roster display rule: clients must not present remote roster data as definitely current.
Default Phase 1 behavior is to mark a remote player stale when `last_seen` is more than 15 minutes
old, and to show the last-seen time when available. Games may choose stricter display rules.

## Obligations

Obligations are broadcast through `debt.*` events and projected by the hub.

Current standard events:

- `debt.created`
- `debt.adjusted`
- `debt.resolved`

Expected future events:

- `debt.transferred` if obligation transfer becomes part of Phase 2.

The hub is authoritative for the cross-node projection, but source nodes still emit the facts.

`debt.adjusted` is used for partial repayment and carries `obligation_id`, `old_weight`, `new_weight`, `delta`, and `reason`.

Standard debt payloads:

| Event | Payload fields |
|---|---|
| `debt.created` | `obligation_id`, `creditor_ref`, `debtor_ref`, `kind`, `terms`, `weight` |
| `debt.adjusted` | `obligation_id`, `old_weight`, `new_weight`, `delta`, `reason` |
| `debt.resolved` | `obligation_id`, `resolution`, `resolved_at` |

## PvP Queue

Cross-node PvP is asynchronous:

1. Attacker node queues a request at the hub.
2. Hub stores request for the victim node.
3. Victim node drains pending requests.
4. Victim node resolves against local victim state.
5. Victim node emits a `pvp.resolved` event.
6. Attacker node observes the event and credits any result.
7. Victim node marks the request complete at the hub.

The attacker node must not resolve combat for a remote victim.

If victim-side resolution fails, the request remains pending. Only the victim node may complete a pending request.
If the victim node receives a malformed attacker payload that cannot be retried successfully, it may
mark the request `blocked`.

Queue request payload:

| Field | Type | Required |
|---|---|---|
| `attacker_id` | string | yes |
| `victim_id` | string | yes |
| `attacker` | object | yes |

Pending request fields include `request_id`, `attacker_id`, `victim_id`, `victim_node`,
`attacker`, `status`, `error`, `updated_at`, and `created_at`.

Completion:

```text
POST /v1/pvp/{id}/result
```

Blocked:

```text
POST /v1/pvp/{id}/blocked
```

Blocked request body:

```json
{"error":"malformed attacker payload"}
```

Only the victim node may block a PvP request. Blocked requests are not returned by
`GET /v1/pvp/pending`. Operators may reset a blocked item to `pending` after repair through local
hub tooling.

## Travel

Travel is snapshot relay:

1. Origin node exports a player snapshot.
2. Origin node submits it to the hub with destination node.
3. Origin node marks the player `traveling`.
4. Destination node drains pending arrivals.
5. Destination node imports the snapshot.
6. Destination node marks the hub request arrived.
7. If the player returns home, local status is set `active`.

The hub enforces one pending travel request per global player ID and maintains the authoritative current-location projection. Travel submission is accepted only from the node that currently holds the player. If the hub has no location row for a player, the global-ID source prefix is the initial current node.

If destination import fails, the request remains pending. Only the destination node may mark a pending travel request arrived.
If the destination node receives a malformed snapshot payload that cannot be retried successfully,
it may mark the request `blocked`.

Travel submit payload:

| Field | Type | Required |
|---|---|---|
| `global_id` | string | yes |
| `home_node` | string | no; hub/default may derive it |
| `dest_node` | string | yes |
| `snapshot` | object | yes |

Pending arrival fields include `travel_id`, `global_id`, `home_node`, `from_node`, `dest_node`,
`snapshot`, `status`, `error`, `updated_at`, and `created_at`.

Arrival:

```text
POST /v1/travel/{id}/arrived
```

Blocked:

```text
POST /v1/travel/{id}/blocked
```

Blocked request body:

```json
{"error":"invalid travel snapshot"}
```

Only the destination node may block a travel request. Blocked arrivals are not returned by
`GET /v1/travel/pending`. Operators may reset a blocked item to `pending` after repair through
local hub tooling.

## Conflict and Partition Rules

- Nodes continue local play during hub outages.
- Push and pull cursors advance only after successful network operations.
- Replayed events must be safe.
- Stale roster data should be displayed as stale by clients.
- Cross-node actions may remain pending while a target node is offline.
- The hub rejects source-node spoofing.
- The hub should be the final authority for cross-node queue state and materialized projections.
- PvP/travel queue completion is owner-scoped and must not acknowledge failed local processing.
- Non-retryable malformed PvP/travel payloads may be marked `blocked` only by the victim or
  destination node that owns the pending work.

## Required Standard Events

Phase 1 framework events:

- `player.created`
- `player.died`
- `player.traveled`
- `debt.created`
- `debt.adjusted`
- `debt.resolved`
- `pvp.resolved`

Games may define additional events. Additional events must remain self-contained and typed.
The hub requires all event payloads to be JSON objects. It validates the standard Phase 1 event
schemas below before accepting them into the feed. Unknown game-defined event types may include
additional object fields but must still be self-contained.

Standard event payloads:

| Event | Payload fields |
|---|---|
| `player.created` | `global_id`, `name`, `home_node`, `created_at` |
| `player.died` | `global_id`, `cause`, `timestamp` |
| `player.traveled` | `global_id`, `src_node`, `dest_node`, `snapshot_hash`, `timestamp` |
| `pvp.resolved` | `request_id`, `attacker_global_id`, `victim_global_id`, `winner_global_id`, `result_text`, `resolved_at` |

Payloads may include additional game-specific fields, but standard consumers must be able to ignore
unknown fields safely.
