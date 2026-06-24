# InterDOOR

A federation protocol and network for SSH terminal games.

Independent game servers connect to a shared hub. Players on one server can see players on
other servers, attack them, trade with them, travel between games, and share a cross-node
economy. Each game defines its own mechanics, UI, and world -- the network is the layer that
connects them.

```
ssh -p 2300 hub.interdoor.net     # browse the network
```

---

## What this is (and is not)

**InterDOOR is a network protocol.** The hub at `hub.interdoor.net` is a directory and event
relay. It tracks which games are running, who is playing, and what is happening across the
network. It does not host games or manage player accounts.

**InterDOOR is not a single game.** Any SSH terminal game can connect to the network by
implementing a small HTTPS API client. The protocol does not require Go, the reference engine,
or any particular game mechanics.

**Ledger of the Low** is the reference implementation -- an early-stage daily-turn survival RPG
bundled with the node binary to prove the protocol works. It is functional but thin: the
gameplay loop, writing, and world all need significant development before it would be a game
sysops run for players by choice. It is a demonstration, not the product.

The network is the product.

---

## Browse the network

```bash
ssh -p 2300 hub.interdoor.net
```

The hub SSH portal shows the live node directory: which games are running, how many players,
and the connect command for each. No account required. Press 1--9 to select a node, R to
refresh, Q to quit.

The directory is also available as JSON:

```
https://hub.interdoor.net/v1/directory
https://hub.interdoor.net/v1/status
```

---

## Running a node (Ledger of the Low)

Full operator guide: [`files/SYSOP_GUIDE.md`](files/SYSOP_GUIDE.md)

Short version:

**1. Get the binary**

Download from the releases page or build from source (see below). Place at
`/usr/local/bin/interdoor-node`.

**2. Write the config**

```json
{
  "addr": ":2323",
  "db": "/var/lib/interdoor/node.db",
  "hostkey": "/etc/interdoor/hostkey",
  "node": "mynode",
  "max_sessions": 32,
  "idle_timeout_sec": 600,
  "hub_url": "https://hub.interdoor.net",
  "hub_reg_token": "TOKEN_FROM_HUB_OPERATOR",
  "advertise_addr": "ssh://mynode.example.com:2323",
  "sync_interval_sec": 20
}
```

Omit `hub_url` and `hub_reg_token` to run standalone (no federation).

**3. Run it**

```
interdoor-node -config /etc/interdoor/node.json
```

The SSH host key generates automatically on first start. After that, `hub_reg_token`
is no longer needed (the node stores its API key in the database).

**4. Connect**

```
ssh -p 2323 mynode.example.com
```

Players create accounts at the terminal. No shell access is granted.

---

## Connecting a different game to the network

Any SSH terminal game can join the network. The minimum is two HTTP calls (register + heartbeat).
A full integration adds event push/pull, roster sync, cross-node travel, and PvP.

Full guide: [`files/BUILDING_FOR_INTERDOOR.md`](files/BUILDING_FOR_INTERDOOR.md)

The short version of the three tiers:

| Tier | What you implement | What you get |
|---|---|---|
| 1 — Listed | Register + heartbeat | Game appears in the directory |
| 2 — Network-aware | + Events + roster | Cross-node players and event feed |
| 3 — Integrated | + Travel + PvP + ledger | Characters move between games |

You own your SSH server, player accounts, game mechanics, and hosting. InterDOOR is the
layer that connects independent servers to each other.

---

## Build from source

Requires Go 1.24+. No CGO -- the SQLite driver is pure Go.

```bash
git clone https://github.com/njb1966/interdoor
cd interdoor
make build        # builds bin/interdoor-node and bin/interdoor-hub
make test         # runs all tests
make release      # cross-compiles linux/amd64 + linux/arm64 into dist/
```

---

## Project layout

```
cmd/
  node/         -- node binary (SSH game server + federation client)
  hub/          -- hub binary (federation registry + event feed + SSH portal)
internal/
  engine/       -- generic game engine (auth, turns, events, obligations, federation hooks)
  game/         -- Ledger of the Low game module (implements engine.Game)
  hub/          -- hub server, SQLite store, ANSI SSH portal
  fed/          -- node-side federation client (sync, register, push/pull)
files/
  SYSOP_GUIDE.md              -- operator documentation
  BUILDING_FOR_INTERDOOR.md   -- how to connect any game to the network
  FEDERATION_PROTOCOL.md      -- wire protocol specification
  FEDERATION_READINESS.md     -- requirements tracker
  DESIGN.md                   -- network and game design
```

The engine never imports the game. The `engine.Game` interface is the only seam between them.
Any game that implements the interface runs on the same engine and federates with the network.

---

## Federation protocol

Hub-and-spoke over HTTPS. Nodes authenticate with a bearer API key issued at registration.

- **Event feed** -- nodes push local events; the hub assigns a global sequence number; nodes
  pull and apply events from other nodes via idempotent handlers.
- **Roster sync** -- nodes push their player list; wanderer screens show cross-node players.
- **Cross-node PvP** -- attacks queue at the hub; the victim's node resolves the bout and
  emits a result event; the attacker credits loot via the event feed.
- **Travel** -- snapshot exported from origin, relayed through hub, imported on destination.
  Login blocked while traveling. Single-active invariant enforced at hub.
- **Obligation ledger** -- debt events replicate across nodes; hub maintains a master index.
- **Partition tolerance** -- cursors advance only after confirmed push/pull; events accumulate
  locally during hub outage and replay on reconnect.

Full specification: [`files/FEDERATION_PROTOCOL.md`](files/FEDERATION_PROTOCOL.md)

---

## Hub endpoints

| Method | Path | Auth | Description |
|---|---|---|---|
| GET | `/v1/status` | none | Hub health and node/event counts |
| GET | `/v1/directory` | none | All registered nodes with online status |
| `SSH :2300` | — | none | ANSI terminal portal — live node directory |
| POST | `/v1/register` | reg-token | Register a new node, receive API key |
| POST | `/v1/heartbeat` | api-key | Node heartbeat; returns pending queue depth |
| POST | `/v1/events` | api-key | Push events from this node |
| GET | `/v1/events` | api-key | Pull events from other nodes (cursor-based) |
| POST | `/v1/roster` | api-key | Push this node's player roster |
| GET | `/v1/roster` | api-key | Pull combined roster from other nodes |
| GET | `/v1/debts` | api-key | Cross-node debt query by debtor global ID |
| POST | `/v1/pvp` | api-key | Submit cross-node PvP request |
| GET | `/v1/pvp/pending` | api-key | Drain incoming PvP requests for this node |
| POST | `/v1/pvp/{id}/result` | api-key | Report PvP resolution |
| POST | `/v1/travel` | api-key | Submit player travel request |
| GET | `/v1/travel/pending` | api-key | Drain incoming arrivals for this node |
| POST | `/v1/travel/{id}/arrived` | api-key | Confirm player arrived |

---

## Stack

- Go 1.25, `net/http` stdlib (no web framework)
- `golang.org/x/crypto/ssh` for SSH servers (game nodes and hub portal)
- `modernc.org/sqlite` -- pure-Go SQLite, CGO_ENABLED=0, single static binary
- Node database: SQLite per node
- Hub database: SQLite (designed for Postgres drop-in later)

Release binaries are statically linked (~10 MB) with no runtime dependencies.

---

## License

MIT
