# Hub Specification

## Responsibilities

The hub coordinates federation. It is responsible for:

- Registering node IDs.
- Issuing and validating node API keys.
- Tracking node liveness.
- Maintaining the public node directory.
- Accepting node events and assigning `hub_seq`.
- Serving an ordered event feed to nodes.
- Maintaining roster and debt projections.
- Queueing cross-node PvP and travel requests.
- Maintaining the Tier 3 player-location projection used by travel.

The hub is not responsible for:

- Serving player sessions.
- Owning player passwords.
- Running game logic.
- Resolving combat directly.
- Editing game-specific snapshots.

## Authentication

Registration uses an operator-issued registration token. Successful registration returns a long-lived API key. All non-public endpoints use:

```text
Authorization: Bearer <api_key>
```

The hub stores API-key hashes, not plaintext API keys.

Only nodes with `nodes.status='active'` may authenticate. Suspending a node preserves its row and
history but rejects its existing API key for protected endpoints.

Error responses use the Phase 1 JSON shape:

```json
{"error":"human-readable error"}
```

## Public Endpoints

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/` | Minimal hub index |
| `GET` | `/v1/status` | Hub health, protocol version, active node counts, event count |
| `GET` | `/v1/directory` | Active registered nodes with online status and advertise address |
| SSH | portal port | Optional ANSI directory browser |

Online status is derived from recent heartbeat time. Current code treats a node as online when its last heartbeat is within five minutes. Suspended nodes are excluded from public directory and status responses.

Directory entries include both `game_id` and `game_title`. `game_id` is the stable protocol
identifier used for same-game compatibility checks. `game_title` is human-readable display text for
the public directory and SSH portal. If a node omits `game_title`, the hub falls back to `game_id`.

## Authenticated Endpoints

| Method | Path | Purpose |
|---|---|---|
| `POST` | `/v1/register` | Register node and receive API key |
| `POST` | `/v1/heartbeat` | Update liveness and return pending queue counts |
| `POST` | `/v1/events` | Push local events |
| `GET` | `/v1/events` | Pull hub-ordered events |
| `POST` | `/v1/roster` | Replace this node's roster projection |
| `GET` | `/v1/roster` | Pull network roster |
| `GET` | `/v1/debts?debtor=<global_id>` | Query debt projection |
| `POST` | `/v1/pvp` | Queue cross-node PvP request |
| `GET` | `/v1/pvp/pending` | Drain PvP requests for this node's players |
| `POST` | `/v1/pvp/{id}/result` | Mark PvP request resolved |
| `POST` | `/v1/pvp/{id}/blocked` | Mark malformed pending PvP request blocked |
| `POST` | `/v1/travel` | Queue travel snapshot |
| `GET` | `/v1/travel/pending` | Drain travel arrivals for this node |
| `POST` | `/v1/travel/{id}/arrived` | Mark travel request arrived |
| `POST` | `/v1/travel/{id}/blocked` | Mark malformed pending travel request blocked |

## Event Feed

The hub accepts batches of events. For each new event:

- `source_node` must match the authenticated node.
- Event envelope fields must be present and valid.
- Phase 1 standard events must match their documented payload schemas.
- `event_id` is deduplicated.
- The hub assigns a monotonic `hub_seq`.
- Events are returned by `hub_seq` order.

`GET /v1/events` uses `after` as the last seen `hub_seq`, defaults `limit` to `500`, and caps
`limit` at `1000`. Other Phase 1 list endpoints are currently unpaginated and must stay small
enough for that to remain operationally safe.

The hub does not interpret most game-defined event payloads beyond requiring a JSON object payload.
It validates Phase 1 standard event schemas before accepting them into the feed, then materializes
`debt.created`, `debt.adjusted`, and `debt.resolved` into the debt projection.

## Roster Projection

Roster pushes are full replacement for the authenticated node. The hub sets each pushed entry's node ID to the authenticated node, regardless of client input.

Roster entries are intentionally small:

- Global player ID.
- Node ID.
- Name.
- Level.
- Status.
- Last seen timestamp.

## Debt Projection

The hub debt table is a projection from accepted `debt.*` events. Current behavior:

- `debt.created` inserts an open debt with weight and terms.
- `debt.adjusted` changes the open debt weight after partial repayment.
- `debt.resolved` marks the debt resolved.
- Malformed standard debt payloads are rejected before they enter the event feed.

Tier 3 debt events that reference registered player nodes must stay within the same `game_id` unless a future compatibility map explicitly permits the combination. NPC refs such as `npc:<id>` are game-local refs and are not treated as node IDs.

## PvP Queue

PvP requests are queued at the hub and drained by the victim's node.

Current hub validation:

- Attacker and victim IDs are required.
- Attacker ID must belong to the authenticated node.
- Victim node is derived from the victim global ID prefix.
- Victim node must be registered and active.
- Attacker and victim nodes must have matching `game_id`.

The victim node resolves the attack and emits the result event. Only the victim node may mark a pending request resolved. Missing, already resolved, and wrong-owner completion attempts are rejected.

Queue statuses:

- `pending`: normal retryable work. Resolver failure leaves the request pending.
- `resolved`: victim node successfully processed and completed the request.
- `blocked`: victim node found a malformed non-retryable queue payload. The request is removed
  from pending drain results and retains an operator-visible error string.

Only the victim node may block a pending PvP request. Blocking is for protocol/payload problems
that retrying will not fix, not for transient local resolver errors.

## Travel Queue

Travel requests move a player snapshot through the hub.

Current hub validation:

- Global ID, destination node, and snapshot are required.
- Destination node must be registered and active.
- Origin and destination nodes must have matching `game_id`.
- Only one pending travel request per global player ID is allowed.
- The hub stores origin node, home node, destination node, snapshot, and status.

The hub maintains `player_locations` as the current-location authority. A travel submission is accepted only from the node that currently holds the player. If the hub has no location row yet, the player's source-prefix node is treated as the initial current node. The destination node imports the snapshot and then marks the travel request arrived. Only the destination node may mark the pending request arrived.

Queue statuses:

- `pending`: normal retryable arrival. Import failure leaves the request pending.
- `arrived`: destination node successfully imported the snapshot and completed the request.
- `blocked`: destination node found a malformed non-retryable snapshot payload. The request is
  removed from pending drain results and retains an operator-visible error string.

Only the destination node may block a pending travel request. Blocking is for payloads that cannot
be decoded as a valid snapshot object, not for transient destination import errors.

## Storage Projections

The hub database is not the whole network state. It is a set of projections and queues:

- `nodes`: registration and liveness.
- `events`: accepted global feed.
- `roster`: latest roster entries per node.
- `debts`: materialized ledger view.
- `pvp_requests`: pending/resolved/blocked cross-node PvP queue, including last error text.
- `travel`: pending/arrived/blocked travel snapshots, including last error text.
- `player_locations`: current node, home node, travel state, and pending travel ID per traveling player.

## Phase 1 Production Readiness

The current Phase 1 deployment target is a small InterDoor network: one public hub, local-only hub
administration, SQLite hub storage, and independently operated SSH game nodes.

SQLite is acceptable for this target when these operating rules are followed:

- Keep hub administration local to the hub host; do not expose admin endpoints.
- Keep the hub database readable only by the hub service account and privileged operators.
- Use `interdoor-hub-admin backup` or SQLite online backup for routine backups.
- Verify restore drills against copied production data after schema or queue changes.
- Watch `/v1/status`, `/v1/directory`, hub service logs, and `interdoor-hub-admin queues`.
- Revisit a server database backend if node count, write volume, delegated administration, or
  recovery requirements exceed SQLite's operational comfort zone.

Final Phase 1 acceptance checks for a deployed hub:

```bash
curl -sS https://hub.interdoor.net/v1/status
curl -sS https://hub.interdoor.net/v1/directory
ssh -p 2300 hub.interdoor.net
interdoor-hub-admin -db /var/lib/interdoor-hub/hub.db nodes
interdoor-hub-admin -db /var/lib/interdoor-hub/hub.db queues
interdoor-hub-admin -db /var/lib/interdoor-hub/hub.db backup /var/lib/interdoor-hub/hub.db.acceptance-YYYYMMDD-HHMM.db
interdoor-hub-admin -db /var/lib/interdoor-hub/hub.db node-suspend NODE_ID
interdoor-hub-admin -db /var/lib/interdoor-hub/hub.db node-remove NODE_ID
```

The final two commands are dry-runs unless `--execute` is supplied.

## Operator Node Removal

Phase 1 does not expose a public admin endpoint. Hub administration is local-only through
`interdoor-hub-admin` against the hub SQLite database and registration-token file.

`nodes.status='active'` is the only state visible in the public directory and allowed to
authenticate against protected hub endpoints. Suspending a node hides it from `/v1/directory`,
removes it from `/v1/status` node counts, and rejects its existing bearer token for sync calls.

Use suspension before removal when an operator needs to stop a node from participating without
destroying hub records:

```bash
interdoor-hub-admin -db /var/lib/interdoor-hub/hub.db node-suspend NODE_ID
interdoor-hub-admin -db /var/lib/interdoor-hub/hub.db node-suspend --execute NODE_ID
```

Use the local title command when the human-readable directory title needs correction without
changing the stable protocol `game_id`:

```bash
interdoor-hub-admin -db /var/lib/interdoor-hub/hub.db node-title NODE_ID "Game Title"
interdoor-hub-admin -db /var/lib/interdoor-hub/hub.db node-title --execute NODE_ID "Game Title"
```

When a live node must be removed from the network, the hub operator must preserve state first and
then remove only the affected node projections:

1. Stop and disable the node service at the node host so it stops heartbeating and cannot re-push
   roster or events during cleanup.
2. Preserve the node's binary, config, database, host key, and service unit in a dated backup or
   quarantine path. Do not destroy the only copy of a node database during an incident.
3. Back up the hub database before editing it.
4. Inspect the hub rows tied to the node ID:
   - `nodes.node_id`
   - `roster.node_id`
   - `events.source_node`
   - `debts.source_node`
   - `pvp_requests.victim_node`
   - `travel.from_node`, `travel.dest_node`, and `travel.home_node`
   - `player_locations.current_node` and `player_locations.home_node`
5. If there are pending travel or PvP rows for the node, decide whether they can be safely deleted,
   repaired, or need player-facing recovery. Do not blindly delete pending queues for a valid game
   node with active users.
6. Delete only rows tied to the removed node, in one transaction, and leave unrelated nodes and
   projections untouched.
7. Verify `/v1/directory`, `/v1/status`, the hub logs, and the remaining node services.
8. Update the infrastructure record with what was removed, where backups were placed, and what hub
   rows were changed.

The local admin command provides dry-run counts before deleting:

```bash
interdoor-hub-admin -db /var/lib/interdoor-hub/hub.db node-remove NODE_ID
interdoor-hub-admin -db /var/lib/interdoor-hub/hub.db backup /var/lib/interdoor-hub/hub.db.pre-remove-NODE_ID.db
interdoor-hub-admin -db /var/lib/interdoor-hub/hub.db node-remove --execute NODE_ID
```

Related local operator commands:

- `nodes`: list all registered nodes, including suspended nodes.
- `queues [--node NODE_ID]`: inspect PvP and travel queues with age and blocked error text.
- `queue-retry pvp|travel ID`: dry-run reset of a blocked queue item back to `pending`;
  add `--execute` after the payload or local node state has been repaired.
- `rotate-key NODE_ID`: dry-run API-key rotation; `--execute` prints the new one-time key.
- `token-issue --file PATH --execute`: write a fresh registration token.
- `token-revoke --file PATH --execute`: clear the registration-token file.

Database-backed mutating commands record successful execution in the local `hub_admin_audit`
table. Audit rows include timestamp, local operator name, command, target, and non-secret detail.
Dry-runs do not create audit rows. Registration-token file operations are filesystem operations and
must be covered by normal host logging and file permissions.

Future admin tooling may add delegated moderator workflows if node volume requires it, but Phase 1
keeps administration local to the hub host.
