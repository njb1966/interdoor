# PROJECT PLAN

> Status: Historical game-specific source material. The root numbered InterDoor docs are canonical for Phase 1 framework, protocol, operator, and architecture decisions. Do not use this file as Phase 1 framework authority unless it has been explicitly reconciled.

## Federated Terminal Game Network

---

## What This Project Is

This project is a **federated terminal game network** — an open protocol and reference implementation that allows independently operated nodes to host terminal-based multiplayer games with shared state, cross-node player identity, and synchronized gameplay across the network.

The project has a working title. It needs a real name. Throughout these documents, the network itself is referred to as **the Network** until a name is chosen.

The network's first game — its reference implementation and primary draw — is **Dornhaven: The Low**, a daily-turn survival RPG set in the underground of a fictional Hanseatic trading city. Dornhaven is designed to be compelling as a standalone single-node game, with federation adding depth and scale. It is the proof that the network works, and the reason people join.

**The project is not Dornhaven.** Dornhaven is the first game built on the network. The network is designed so that other games — a trading game, a strategy game, an entirely different genre — could be built against the same engine API and participate in the same federated infrastructure. Dornhaven is the killer app. The network is the product.

---

## Why This Project Exists

The gap this fills: federated, inter-system multiplayer for terminal-based games does not exist in any modern, maintainable form.

- **BBSlink, DoorParty, Exodus** are centralized door game servers. One operator hosts everything. If that server goes down, the games go down. No federation, no peer-to-peer, no ability to run your own node.
- **Old InterBBS leagues** were federated but used fragile file-based sync over FidoNet. The infrastructure is dead and the tools are unmaintained.
- **Modern door game recreations** (twclone, Legend of the Green Dragon, Synchronet JS ports) are standalone. No inter-system networking.

Nobody has built a modern, documented, open protocol for federated terminal game networking. This project does that, and ships a game worth playing on top of it.

---

## Project Phases

The project has two major tracks that run in sequence, with some overlap:

### Track A: Game Design (Dornhaven)

Design the game completely before writing production code. The game design must account for network requirements at every decision point — every mechanic, every system, every data structure must be designed with federation in mind, even though federation is built later.

Track A produces:
- Complete game design documentation (world, mechanics, content, balance)
- A v1 feature specification (the minimal playable game — LORD-equivalent scope)
- Data model definitions (what state exists, what syncs, what doesn't)
- Content requirements (writing, ANSI art, NPC dialogue, encounter tables)

Track A is complete when: the game could be handed to any competent developer with the documentation and they could build it without asking design questions.

### Track B: Network Engineering

Build the federated terminal game network, with Dornhaven as the reference game built against it.

Track B has four phases:

**Phase B1: Engine and Connection Server**
Build the game engine API and connection server. Implement Dornhaven v1 against the engine. Result: a playable single-node game accessible via SSH.

**Phase B2: Single-Node Polish**
Open to testers. Iterate on gameplay, balance, writing, ANSI art. Fix what's broken. Add what's missing from the v1 spec. Result: a game that's genuinely fun to play daily on a single node.

**Phase B3: Federation**
Design and implement the federation protocol. Node registration, authentication, state sync, cross-node identity, event broadcast. Test between personal nodes first. Result: a working multi-node network with Dornhaven running across nodes.

**Phase B4: Public Network Launch**
Documentation, sysop setup guide, node directory, hub infrastructure. Open for other operators to join. Result: a live federated game network that other people run nodes on.

---

## Phase Detail

### Track A: Game Design

This track follows the process defined in GAME_DESIGN_PROCESS.md. Summary of stages:

**A1. Core Mechanics Lock**
Finalize the daily action loop, combat system, economy (favor/debt model), progression tracks, and death/consequence mechanics. Every mechanic must pass the network compatibility check defined in NETWORK_REQUIREMENTS.md.

**A2. Content Design**
Design the districts, NPCs, faction systems, encounter tables, item/gear catalog, and contract system. Write representative samples of all content types (exploration text, NPC dialogue, combat narration, market interactions) to establish voice and tone.

**A3. Endgame Design**
Design the knowledge/investigation system, the Archive mechanic, the Deep, and all four endgame paths. Define how endgame choices affect game state — locally and across federated nodes.

**A4. Data Model**
Define every data structure: player state, game world state, NPC state, economy state, faction state, knowledge state. For each structure, define: what is local-only, what syncs across nodes, what is derived/computed. This directly feeds the federation protocol design.

**A5. V1 Specification**
Produce BUILD_SPEC_V1.md — the subset of the full design that constitutes the minimum playable game. LORD-equivalent scope. Everything a player needs to have a fun 10-15 minute daily session with zero knowledge of the deeper systems.

**A6. Content Production Plan**
Inventory all content that must be written for v1: ANSI art screens, NPC dialogue trees, exploration encounter text, item descriptions, tutorial text, help screens. Estimate scope. This runs parallel to early Phase B1 work.

Exit criteria for Track A: DESIGN.md is complete and versioned. BUILD_SPEC_V1.md is complete. DATA_MODEL.md is complete. CONTENT_PLAN.md is complete. All documents have been reviewed against NETWORK_REQUIREMENTS.md for compatibility.

---

### Phase B1: Engine and Connection Server

**Duration estimate:** 4-8 weeks

**Deliverables:**
- Game Engine API (Go interfaces) — documented, tested
- Connection server (SSH, terminal handling, player auth)
- SQLite persistence layer
- Event emission system (local — federation consumes these later)
- Dornhaven v1 implemented against the engine API
- Playable by the developer on localhost

**Key decisions:**
- Go module structure and package layout
- SSH library selection (e.g., golang.org/x/crypto/ssh)
- Terminal rendering approach (raw ANSI vs. a TUI library like tcell/bubbletea)
- SQLite schema based on DATA_MODEL.md
- Event format and serialization (JSON, protobuf, or custom)

**Build order within B1:**
1. Connection server skeleton — accept SSH connections, display text, handle input
2. Player authentication — create account, login, persist credentials
3. Game engine API interfaces — define the contracts
4. Dornhaven game module — implement against the API
5. Core loop — Warrens exploration, Lanternmarket, NPCs, daily actions
6. Combat system
7. Economy (favor/debt)
8. PvP (attack sleeping players)
9. Death and recovery
10. ANSI art integration (title screen, district headers)

**Exit criteria:** A single player can SSH into the server, create a character, play a full daily session (explore, trade, fight, interact with NPCs), log out, and return the next day with state preserved. The game is fun enough that the developer wants to play it.

---

### Phase B2: Single-Node Polish

**Duration estimate:** 4-8 weeks (overlaps with ongoing content work)

**Deliverables:**
- Bug fixes and balance adjustments based on testing
- Remaining v1 content (full NPC dialogues, encounter variety, item catalog)
- Player-to-player interactions (messaging, favor trades, PvP polish)
- Contract system (player-posted jobs)
- Quality ANSI art for all key screens
- Help system and in-game documentation
- Sysop admin tools (basic — player management, game resets, config)

**Process:**
- Invite 5-10 testers (BBS community contacts, OffGrid Holdout users, friends)
- Collect feedback on daily play sessions
- Iterate on balance (action economy, combat difficulty, progression pacing)
- Iterate on writing (tone consistency, NPC voice, encounter variety)
- Identify any mechanics that don't work and redesign

**Exit criteria:** Testers play daily without developer prompting. The game is fun. The balance feels right. The writing is consistent. The sysop can manage the server without touching the database directly.

---

### Phase B3: Federation

**Duration estimate:** 6-12 weeks

**Deliverables:**
- Federation protocol specification (FEDERATION_PROTOCOL.md)
- Hub server implementation
- Node registration and authentication
- State sync engine (REST API + optional WebSocket)
- Cross-node player identity
- Cross-node debt/favor tracking
- Event broadcast system
- Inter-node travel mechanic in Dornhaven
- Conflict resolution for concurrent state changes
- Federation admin dashboard (web or terminal)

**Build order within B3:**
1. Protocol specification — document before implementing
2. Hub server skeleton — registration endpoint, auth
3. Node-to-hub heartbeat and status reporting
4. Player roster sync — who exists where
5. Game state delta format — what changed, when, by whom
6. State sync engine — push/pull deltas on schedule
7. Conflict resolution — last-write-wins with manual override for edge cases
8. Cross-node debt tracking — the Ledger goes distributed
9. Event broadcast — notable events propagate across nodes
10. Inter-node travel in Dornhaven — players move between nodes
11. Testing between developer's own nodes (minimum 2-3 nodes)
12. Protocol documentation for third-party implementers

**Exit criteria:** Two or more nodes running independently sync state correctly. A player on Node A can interact with the game state on Node B. Debts cross node boundaries. Events propagate. A new node can join the network using only the documentation.

---

### Phase B4: Public Network Launch

**Duration estimate:** 2-4 weeks

**Deliverables:**
- Sysop setup guide (from zero to running node)
- Single-binary release builds (linux/amd64, linux/arm64)
- Hub infrastructure (hosted, monitored, backed up)
- Node directory (web page listing active nodes)
- Network status dashboard
- Project website with documentation
- Announcement to BBS community, retro computing community, small web community

**Exit criteria:** Someone who has never spoken to the developer can download the binary, follow the guide, stand up a node, register with the hub, and have players connecting within an hour.

---

## What Comes After Launch

These are not phases with timelines. They're ongoing work that begins after B4 and continues indefinitely:

- **Dornhaven depth expansion** — Factions, the Archive, the Knowledge system, endgame paths, the Deep. The full DESIGN.md vision, rolled out incrementally.
- **Game extension system** — An API for community-contributed content (the IGM equivalent). Custom encounters, NPCs, items, even new districts.
- **Additional games** — Other games built against the engine API. A trading game, a strategy game, community-developed games. The network supports multiple games, not just Dornhaven.
- **Protocol evolution** — Versioned protocol updates as the network grows and new requirements emerge.
- **Community management** — Running the hub, approving nodes, managing the network, engaging with sysops and players.

---

## Tools and Infrastructure

**Development:**
- Language: Go
- Editor: VSCodium
- AI assistance: Claude Code (primary), local models for review
- Version control: Git (GitHub)
- Testing: Go standard testing + manual playtesting

**Runtime:**
- OS: Debian Linux
- Database: SQLite (per-node), PostgreSQL (hub)
- Deployment: Single binary, systemd service
- Monitoring: Grafana (existing infrastructure)

**Network:**
- Hub hosting: Developer's infrastructure (debian13 or Contabo VPS)
- DNS: deSEC
- TLS: Caddy reverse proxy for web dashboard; SSH handles its own encryption
- Backup: Borg (existing infrastructure)

---

## Risk Register

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Game design scope creep delays engineering | High | High | V1 spec is LORD-equivalent. Full design is the bible, not the build target. |
| Federation protocol too complex for sysops | Medium | High | Hub-and-spoke, not peer-to-peer. Single binary. Setup in under an hour. |
| No players / no other node operators | Medium | High | Game must be fun single-node. Federation is a bonus, not a requirement. |
| Writing quality inconsistent | Medium | Medium | Establish voice guide with samples. AI-assisted drafting with human editing. |
| Go unfamiliar to developer | Low | Medium | AI-assisted development. Go is deliberately simple. Strong community docs. |
| Burnout / interest fade | High | Critical | Ship v1 fast. Get testers early. External accountability through public repo. |

The burnout risk is the real one. Mitigation: the game must be playable and fun as early as possible. Don't polish in isolation. Get other people playing it before it's "ready."

---

## Document Index

| Document | Purpose | Status |
|----------|---------|--------|
| PROJECT_PLAN.md (this document) | Master plan, phases, milestones | v0.1 |
| NETWORK_REQUIREMENTS.md | Constraints the network imposes on game design | v0.1 |
| FEDERATION_READINESS.md | Living tracker: current code vs. federation needs (updated each slice) | v0.1 |
| GAME_DESIGN_PROCESS.md | Step-by-step process for completing game design | v0.1 |
| DESIGN.md | Dornhaven world bible (setting, lore, full vision) | v0.1 |
| BUILD_SPEC_V1.md | V1 game feature set (LORD-equivalent scope) | v0.2 |
| CORE_LOOP.md | Stage 1 core loop lock (mechanics + numbers) | v0.1 |
| CONTENT_ARCHITECTURE.md | Stage 2 content structures (districts, NPCs, encounters, items, contracts) | v0.1 |
| DATA_MODEL.md | Game state data structures and sync classification | v0.1 |
| CONTENT_PLAN.md | Content inventory and production schedule | v0.1 |
| FEDERATION_PROTOCOL.md | Federation protocol specification | v0.1 |
| VOICE_GUIDE.md | Writing style guide with tone samples | v0.1 |

---

*Document version 0.1 — Initial plan. Subject to revision as game design progresses.*
