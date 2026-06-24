# Building for InterDOOR

> Status: Historical game-specific source material. The root numbered InterDoor docs are canonical for Phase 1 framework, protocol, operator, and architecture decisions. Do not use this file as Phase 1 framework authority unless it has been explicitly reconciled.


*How to connect a terminal game to the InterDOOR network.*

---

## What InterDOOR is (from a developer's perspective)

InterDOOR is a hub-and-spoke federation protocol for SSH terminal games. The hub keeps track of
which games are running, who is playing, and what is happening across the network. It does not
host games, manage player accounts, or care what language your game is written in.

Your game joins the network by making HTTPS calls to the hub API. That is the entire contract.

The hub at `hub.interdoor.net` is the public registry. Nodes that register with it appear in the
SSH portal (`ssh -p 2300 hub.interdoor.net`) and the web directory at `hub.interdoor.net`.

---

## What you get by joining

**At minimum (a few HTTP calls):**

- Your game appears in the hub portal and web directory — players browsing the network can find it
- The directory shows your node ID, game name, SSH address, and live player count

**With a bit more work:**

- Players in your game see players from other games on the network in a cross-node roster
- Events your game emits (kills, deaths, news, milestones) propagate to every other node —
  other games can display them, react to them, or ignore them as they choose

**With deep integration (shared data contracts required):**

- Players travel between your game and other games, carrying their character state
- Cross-node PvP: players attack each other across node boundaries
- Shared obligation ledger: debts accrue and replicate across the network

---

## Three tiers — pick your depth

### Tier 1 — Listed

**Four HTTP calls. An afternoon's work.**

Register once on first start. Send a heartbeat every 20 seconds. Your game appears in the
directory and stays marked online as long as it heartbeats.

```
POST /v1/register    — once; returns your API key
POST /v1/heartbeat   — every 20s; sends player count, keeps you online
```

That is it. Your game is on the network.

### Tier 2 — Network-aware

**Add event push and roster sync. A weekend's work.**

Push events when notable things happen in your game. Pull the cross-node roster to show
players from other games in your "who's online" or news screens. Players feel connected
to something larger without leaving your game.

```
POST /v1/events      — push events when things happen
GET  /v1/events      — pull events from other nodes (cursor-based, idempotent)
POST /v1/roster      — push your current player list
GET  /v1/roster      — pull the combined cross-node roster
```

Events are freeform: a `type` string and a JSON `payload`. You define your event types.
The hub relays them verbatim. Other games decide what to display or act on.

### Tier 3 — Fully integrated

**Cross-node travel, PvP, and the obligation ledger.**

This tier requires agreeing on shared data contracts: what a character snapshot looks like
when it moves between games, how a PvP bout is resolved, what an obligation record contains.
It works best between games in the same genre with compatible player models.

The reference Go engine handles all of this if you are writing in Go. For other languages,
you implement the contracts yourself against the hub's travel and PvP endpoints.

Most games should aim for Tier 2. Tier 3 is the long-term vision for games that want to
feel like one interconnected world.

---

## What you implement yourself

InterDOOR does not provide:

- Your SSH server (run your own however you like)
- Your player accounts and credentials
- Your game mechanics, UI, or data model
- Hosting or infrastructure

You own all of that. InterDOOR is the layer that connects independent game servers to each other.

---

## Implementation paths

### Writing a new game in Go

Use the reference engine (`internal/engine`) as a starting point. It handles the SSH server,
player authentication, event log, federation sync, and the `engine.Game` interface. You
implement the `Game` interface — the game logic, screens, and mechanics — and the engine
handles the rest.

`Ledger of the Low` (`internal/game`) is the working example of this path.

### Writing a new game in any other language

Implement the hub API client yourself. It is approximately 200 lines of HTTP calls in any
language. Run your own SSH server. Implement whichever tier of the protocol fits your goals.
The hub does not know or care what stack your game uses.

### Porting an existing SSH game

Add an HTTP client to your existing codebase. On startup, register with the hub and store
your API key. Run a background thread that heartbeats every 20 seconds. When significant
events happen (player logged in, player died, major kill), POST them to `/v1/events`. Your
existing game, SSH server, and player accounts stay exactly as they are.

---

## What kinds of games fit

**Good fit:**
- BBS door games and MUD-adjacent text RPGs
- Anything with persistent player characters and daily or session-based play
- Games with a social layer: leaderboards, news feeds, player-vs-player
- Games where "there are players on other servers" is an interesting fact

**Poor fit:**
- Real-time multiplayer (federation is async and poll-based, not sub-second)
- Games with no persistent player state
- Games that are not SSH-based

---

## Getting listed on the public hub

1. Run your game in standalone mode first. Confirm it works.
2. Open a [GitHub issue](https://github.com/njb1966/interdoor/issues/new?title=Node+registration+request&body=Node+ID%3A+%0AGame+name%3A+%0APublic+SSH+address%3A+)
   with your proposed node ID, game name, and public SSH address.
3. A registration token arrives by return. Add it to your config and restart.

The hub operator reviews requests to keep the directory clean. Tier 1 is enough to get listed.

---

## Further reading

- [`FEDERATION_PROTOCOL.md`](FEDERATION_PROTOCOL.md) — full wire protocol specification
- [`SYSOP_GUIDE.md`](SYSOP_GUIDE.md) — running the reference node binary
- `internal/fed/fed.go` — the Go federation client, if you want to see a working implementation
