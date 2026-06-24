# InterDoor Project Charter

## Purpose

InterDoor is a federated networking framework for terminal games. It lets independently operated SSH game nodes register with a hub, publish their presence, exchange roster and event data, and, for deeper integrations, coordinate cross-node travel, PvP, and shared obligations.

The network is the product. Individual games are proofs and clients of the framework.

## Three-Part Framing

InterDoor has three related but distinct parts:

1. The federation framework: hub, node sync client, wire protocol, event feed, roster, queues, and operational rules.
2. The game engine and SDK surface: reusable terminal-game primitives such as identity, auth, turns, events, snapshots, and integration hooks.
3. Reference and future games: Ledger of the Low as the Phase 1 reference node, plus future custom games or license-compatible open-source games only after their own design approval.

The canonical root documentation describes parts 1 and 2. Game-specific fiction, balance, content, and implementation notes remain in the existing game folders and legacy docs.

## Phase 1 Scope

Phase 1 is the framework phase. It is not a content-complete game phase.

Phase 1 includes:

- Hub-and-spoke federation over HTTP JSON.
- Node registration, bearer-token authentication, heartbeat, and public directory/status.
- Append-only event propagation with hub-assigned total ordering.
- Roster sync for cross-node player visibility.
- Shared obligation ledger projection driven by `debt.*` events.
- Hub queues for cross-node PvP and travel.
- A reusable Go engine surface for terminal games.
- Documentation and prompts for follow-on implementation work.
- Tests that prove the framework behavior and identify remaining hardening gaps.

Phase 1 does not include:

- Rewriting current games.
- Adding new game implementations to the live InterDoor network before design review and approval.
- Declaring Ledger of the Low or Dornhaven to be the project itself.
- A production Postgres backend unless explicitly scheduled.
- Replacing SSH terminal gameplay with a web-first system.
- Real-time multiplayer.
- Peer-to-peer node communication.

## Current Reference Implementation

The current implementation lives under `games/ledgerofthelow/`. Despite the directory name, it contains the working InterDoor reference node, hub, federation client, generic engine, and example game module.

Important implementation facts:

- Node and hub are Go programs.
- Node storage and hub storage currently use SQLite.
- Hub storage is behind an interface intended to allow a later Postgres backend.
- Nodes serve players over SSH and sync with the hub over HTTP JSON.
- The generic engine does not import the reference game module.
- The current action economy is 15 main actions per day and 3 PvP attacks per day in code.

## Documentation Policy

The root numbered documents are the canonical framework docs. Existing documents under `docs/` and game folders are preserved as source material, design history, and game-specific context.

When old docs conflict with code, code is treated as the factual baseline. Conflicts that need design decisions are recorded in `07_Backlog.md` or `08_Decisions.md`.
