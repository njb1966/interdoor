# InterDoor Architecture

## Shape

InterDoor is hub-and-spoke.

```text
players -> SSH node -> local engine/game/storage
                    -> HTTP JSON sync -> hub -> other nodes
```

Nodes serve games. The hub coordinates discovery, event relay, and cross-node queues.

## Node

A node is an independently operated SSH game server. In the current reference implementation it includes:

- SSH listener and terminal session handling.
- Player account creation and authentication.
- Local SQLite database.
- Generic engine tables for players, turns, events, obligations, sync cursors, and remote roster.
- A game module implementing the engine interface.
- Federation sync loop when configured with a hub URL.

Nodes can run standalone by omitting hub configuration.

## Hub

The hub is the federation coordinator. It provides:

- Node registration and API-key authentication.
- Heartbeat intake and online/offline status.
- Public directory and status endpoints.
- Append-only event feed with hub sequence numbers.
- Roster projection.
- Debt projection from `debt.*` events.
- PvP request queue.
- Travel request queue.
- Optional SSH portal for browsing the network directory.

The current hub backend is SQLite. The hub package uses a `Store` interface so a later production backend can be added without changing the HTTP layer.

## Engine

The engine owns framework-level primitives:

- Player identity spine.
- Password authentication.
- Session lifecycle.
- Daily turn budgets.
- Event emission and idempotent event apply.
- Obligation records.
- Snapshot export/import.
- Federation hooks for PvP and travel.

The engine must remain game-generic. It may know about players, sessions, events, turns, obligations, and snapshots. It must not know about a specific game's locations, factions, content, or economy flavor.

## Game Module

A game module owns gameplay:

- Screens and menus.
- Game-specific persistence tables.
- Combat and economy rules.
- Content catalogs.
- Snapshot serialization for game-specific state.
- Event handlers for game-specific federated effects.

The current reference game demonstrates the interface but should not define the framework's boundaries by accident.

## SDK Surface

The SDK is the stable integration contract for games. In Go, the current SDK is the engine package plus federation hooks. For non-Go games, the SDK is the HTTP protocol.

Expected SDK concepts:

- Register and heartbeat.
- Push and pull events.
- Push and pull roster entries.
- Export and import travel snapshots.
- Queue and resolve cross-node PvP.
- Emit and apply obligation events.

## Storage

Node storage currently uses SQLite with these framework tables:

- `players`
- `turn_state`
- `events`
- `obligations`
- `sync_state`
- `remote_roster`

Hub storage currently uses SQLite with these framework tables:

- `nodes`
- `events`
- `roster`
- `debts`
- `pvp_requests`
- `travel`

Game modules may create their own tables in the node database, keyed by global player ID where player-owned state is involved.

## Sync Loop

The reference sync loop performs these steps:

1. Heartbeat to the hub.
2. Read local events after `last_pushed_seq`.
3. Push local events to the hub.
4. Advance the push cursor after a successful push.
5. Pull hub events after `hub_cursor`.
6. Apply each event locally.
7. Advance the hub cursor after each successful apply.
8. Drain PvP requests for local victims.
9. Drain travel arrivals for this node.
10. Push local roster.
11. Pull and cache remote roster.

This loop is poll-based and partition-tolerant. Failed ticks leave cursors unchanged for retry, except for known hardening gaps listed in `07_Backlog.md`.

## Deployment Shape

The default deployment target is simple:

- One static node binary on a Debian host.
- One SQLite database per node.
- One SSH listen port for players.
- Optional hub URL and registration token.
- A systemd unit to keep the node running.

The hub is similarly simple in Phase 1:

- One static hub binary.
- One SQLite database.
- HTTP API for nodes and public status.
- Optional SSH portal for directory browsing.

## Systemd Examples

These examples are documentation defaults, not a required packaging format. Adjust paths, users,
ports, and environment files to match the actual host.

Node service:

```ini
[Unit]
Description=InterDoor node
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=interdoor
Group=interdoor
ExecStart=/usr/local/bin/interdoor-node -config /etc/interdoor/node.json
Restart=on-failure
RestartSec=5s
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=full
ReadWritePaths=/var/lib/interdoor-node /etc/interdoor

[Install]
WantedBy=multi-user.target
```

Hub service:

```ini
[Unit]
Description=InterDoor federation hub
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=interdoor-hub
Group=interdoor-hub
EnvironmentFile=-/etc/interdoor/hub.env
ExecStart=/usr/local/bin/interdoor-hub \
  -db /var/lib/interdoor-hub/hub.db \
  -addr ${INTERDOOR_HUB_ADDR} \
  -reg-token-file ${INTERDOOR_REG_TOKEN_FILE}
Restart=on-failure
RestartSec=5s
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=full
ReadWritePaths=/var/lib/interdoor-hub /etc/interdoor

[Install]
WantedBy=multi-user.target
```

Root-owned node config files should be mode `0600` or tighter when they contain a registration
token. Nodes use `hub_reg_token` only for first registration when no API key is already stored in
the node database. After registration, remove the token from `/etc/interdoor/node.json` and restart
the service. The hub registration-token file should be issued and revoked with
`interdoor-hub-admin` rather than edited casually.
