# GAME DESIGN PROCESS

> Status: Historical game-specific source material. The root numbered InterDoor docs are canonical for Phase 1 framework, protocol, operator, and architecture decisions. Do not use this file as Phase 1 framework authority unless it has been explicitly reconciled.

## How to Complete the Design of Dornhaven: The Low

---

## Purpose

This document defines the step-by-step process for completing Dornhaven's game design. Each stage produces specific deliverables. Each deliverable is validated against NETWORK_REQUIREMENTS.md before the stage is considered complete.

The goal is a game design complete enough that implementation requires no further design decisions — only engineering and content creation.

Reference documents:
- **DESIGN.md** — The world bible. Setting, lore, districts, factions, tone, endgame vision.
- **NETWORK_REQUIREMENTS.md** — Constraints every mechanic must satisfy.
- **BUILD_SPEC_V1.md** — The subset of the full design that constitutes the v1 release.

---

## Design Stages

### Stage 1: Core Loop Lock

**Goal:** Finalize the fundamental daily gameplay loop — the thing every player does every session. This is the heartbeat of the game. Nothing else matters if this isn't right.

**Deliverables:**

**1.1 Action Economy**
Define the exact action system:
- How many actions per day (starting value, balance rationale)
- Action costs for every activity type
- Whether unspent actions carry over (current design says no — confirm or revise)
- Whether any activities are free (current design: messaging and contract posting are free)
- How action count changes with progression (does it? should it?)
- How injuries affect available actions

*Network check:* Actions are local state. No sync implications. ✓

**1.2 Exploration / Combat**
Define the Warrens exploration system:
- How encounters are generated (random tables, weighted by depth, affected by gear/level)
- Combat mechanics: what stats matter, how damage is calculated, how the player makes combat decisions (fight/flee/use item — keep it simple)
- Encounter types and their relative frequency
- Loot tables: what you can find, rarity distribution
- Depth system: how "deep" works mechanically, how it affects difficulty and rewards
- How the Warrens "shift" between days (is this mechanical or cosmetic?)

*Network check:* Exploration is local state (the encounter sequence, loot rolls). Results that affect synced state (player death, notable loot) must emit events. Encounter tables are shared-reference state.

**1.3 Trading / Economy**
Define the Lanternmarket system:
- What goods/services are available for trade
- How the favor/debt system works mechanically (not just narratively)
- Minimum viable economy: what do players trade, what do they need, what creates demand
- NPC merchants: what they sell, what they buy, how prices work
- How debt is incurred, tracked, and resolved
- What happens when a debt goes unpaid (Ledger consequences at a basic level)

*Network check:* Debts are synced state. Debt creation/resolution must emit events. Debt IDs must be globally unique. Local market inventory is local state. Player-to-player trades are local if same-node, event-based if cross-node.

**1.4 NPC Interaction**
Define the NPC interaction model:
- How many NPCs exist in v1 (keep it small — 4-6 core NPCs)
- Interaction format: menu-driven? keyword-driven? conversational?
- Do NPCs have daily-rotating dialogue? Story arcs that advance across days?
- How do NPCs deliver quests/contracts?
- How do NPCs communicate lore and hints about the deeper game?

*Network check:* NPC state is local (dialogue progress, quest state). NPC definitions are shared-reference. No sync implications.

**1.5 PvP**
Define player-vs-player mechanics:
- Attack sleeping players (offline PvP): how it works, what's at stake, what the attacker risks
- How PvP interacts with the debt/favor economy (can you attack someone who holds your debt?)
- Limitations: cooldowns, level restrictions, faction protections
- Cross-node PvP (for federation): how attacking a player on another node works with async sync

*Network check:* Same-node PvP is local. Cross-node PvP must be expressible as an event (attack initiated → result calculated on target's home node → result event sent back). Both sides must tolerate delay.

**1.6 Death and Consequences**
Define exactly what happens when a player dies:
- What is lost (carried inventory — define what "carried" vs "banked" means)
- What persists (banked resources, debts, reputation, knowledge)
- Where the player respawns and what it costs to return to activity
- How death affects reputation/standing
- How frequently death should occur (balance target)

*Network check:* Player death emits an event. Death state changes are local but the event is synced (other nodes know it happened).

**Exit criteria for Stage 1:** Every mechanic above has a written specification precise enough to implement. No ambiguity. Numbers defined (even if they'll be rebalanced later). The core loop can be explained in under two minutes.

---

### Stage 2: Content Architecture

**Goal:** Design the content structures that populate the game world — districts, NPCs, encounters, items. Not writing the content yet, but defining the structures it will fill.

**Deliverables:**

**2.1 District Specifications**
For each district in the v1 game (Threshold, Lanternmarket, Warrens, Rafters at minimum):
- Exact gameplay functions available
- Navigation: how the player moves between districts (menu? map? directional?)
- Visual presentation: what the player sees on screen when in this district
- Available NPCs, merchants, and services
- Ambient text (descriptions, atmosphere) — structure and frequency

*Network check:* District definitions are shared-reference. District state (if any is dynamic) must be classified.

**2.2 NPC Profiles**
For each v1 NPC:
- Name, role, personality summary
- Dialogue structure: how many dialogue branches, how they rotate, what triggers new dialogue
- Relationship to game systems: what services they provide, what quests they give
- Tone reference: 2-3 sample lines that establish their voice
- Relationship to the deeper lore (how do they hint at the Charter/Ledger?)

**2.3 Encounter Design**
For the Warrens exploration system:
- Encounter template format (what fields define an encounter)
- Creature catalog: names, descriptions, stats, behavior patterns, loot tables
- Non-combat encounter types: discovery, environmental hazard, NPC, anomaly
- How encounters scale with depth
- Minimum encounter variety for v1 (target: enough that a player doesn't see obvious repeats in the first two weeks of daily play)

**2.4 Item and Gear Catalog**
- Item categories (weapons, armor, consumables, trade goods, curiosities)
- Stat model: what attributes items affect, how gear affects combat outcomes
- Rarity tiers and their distribution
- Item degradation (if any — current design says gear degrades)
- Starting gear for new players
- Item storage: what can be carried vs banked, capacity limits

**2.5 Contract System**
- Contract types available in v1
- NPC-posted contracts: format, reward structure, expiration
- Player-posted contracts: how they work, what can be offered/requested
- Contract completion and verification
- Failed/expired contract consequences

*Network check:* Contracts between same-node players are local. Cross-node contracts are synced state — they must use global player IDs and emit events on creation, acceptance, and completion.

**Exit criteria for Stage 2:** Every content structure has a defined format. An encounter, an NPC, an item, and a contract can each be fully described by filling in a template. Content production (Stage A6 in the project plan) can begin using these templates.

---

### Stage 3: Progression and Depth

**Goal:** Design the systems that keep players engaged beyond the first week — reputation, factions, the knowledge system, and the endgame path.

**Deliverables:**

**3.1 Reputation System**
- Reputation scale (numeric range, thresholds for tier names)
- How reputation is earned and lost with each faction
- What reputation thresholds unlock (specific content, access, NPCs, quests)
- Cross-faction reputation tension (does gaining rep with one lose it with another? always? sometimes?)
- How reputation is displayed to the player

**3.2 Standing System**
- How global standing is calculated (derived from actions, reputation, debt network, notable events)
- Standing tiers and their effects on gameplay
- The Ledger attention mechanic: how standing triggers increasing Ledger response
- How standing is displayed to the player and to other players

*Network check:* Standing is local state (each node calculates it for its own players). Standing may be included in player roster sync for cross-node visibility.

**3.3 Faction System**
For each faction (Ledgermen, Remnants, Tally, Ferrymen):
- Quest lines: structure, pacing, how many quests to faction alignment
- Alignment mechanic: when does "building reputation" become "aligned with"
- Alignment exclusivity: can you align with multiple factions, or must you choose?
- Faction-specific benefits and content
- Inter-faction conflict mechanics (territorial disputes, opposing quests)

**3.4 Knowledge System**
- Knowledge fragment categories (Charter Terms, Ledger Mechanisms, Signatories, The Deep, The Flaw)
- How fragments are acquired (Archive research, Warrens anomalies, faction quests, NPC dialogue)
- Fragment assembly: how many fragments per category, how they combine into understanding
- How knowledge is displayed to the player (a journal? a progression screen?)
- How knowledge gates endgame access

**3.5 Endgame Specification**
- Prerequisites for attempting the Descent (knowledge thresholds, faction alignment, resources)
- The Descent sequence: structure, duration (how many days/actions), what happens at each stage
- The four endgame paths: exact choice points, consequences, resolution
- Post-endgame state: what changes in the game world, what happens to the character
- Replay mechanic: how starting over works, what carries over (if anything)

*Network check:* Endgame choices are synced state — they emit events and may have consequences on other nodes. The specific consequences must be defined per-path. How does one node's endgame resolution affect another node's game state?

**Exit criteria for Stage 3:** The complete player arc from character creation to endgame completion is specified. A player's journey from "confused newcomer" to "confronting the Ledger" has no design gaps.

---

### Stage 4: Data Model

**Goal:** Translate all game design into concrete data structures, classified for federation compatibility.

**Deliverables:**

**4.1 DATA_MODEL.md**
For every entity in the game (player, NPC, item, debt, faction, encounter, knowledge fragment, etc.), define:
- Field names and types
- Which fields are local, synced, or shared-reference
- Relationships between entities
- Primary keys and global identifiers
- Size estimates (how much data per player, per node, for the hub)

**4.2 Event Catalog**
Complete list of event types the game emits, with:
- Event type identifier
- Payload schema (JSON)
- When it fires
- Who consumes it (hub, other nodes, both)
- Idempotency guarantee

**4.3 Sync Profile**
A summary document:
- Total estimated sync payload per node per day
- Bandwidth requirements for federation
- Recommended sync intervals for different network conditions
- What happens during a 1-hour partition, a 1-day partition, a 1-week partition

**Exit criteria for Stage 4:** A developer can look at DATA_MODEL.md and implement the database schema without asking design questions. The event catalog fully describes the federation data contract.

---

### Stage 5: V1 Specification

**Goal:** Draw the line. Define exactly what is in v1 and what is not.

**Deliverables:**

**5.1 BUILD_SPEC_V1.md (revision)**
Revise the v1 spec based on everything learned in Stages 1-4. The v1 spec must answer:
- Which districts are in v1?
- Which NPCs are in v1?
- Which factions are in v1? (Possibly none at launch — factions may be Phase 2 depth content)
- Is the endgame in v1? (Possibly not — v1 may be the daily loop only, with endgame added later)
- What is the minimum content inventory (encounters, items, dialogue) for v1 to not feel empty?
- What is explicitly deferred to later?

**5.2 Content Inventory**
A complete list of content assets required for v1:
- ANSI art screens (title, district headers, death, key events) — count and descriptions
- NPC dialogue lines — count per NPC
- Encounter descriptions — count by type
- Item descriptions — count by category
- Tutorial/help text — page count
- Ambient/atmospheric text — count by district

**Exit criteria for Stage 5:** BUILD_SPEC_V1.md is a complete, unambiguous implementation specification. It says exactly what to build, what not to build, and what content to produce.

---

### Stage 6: Content Production Plan

**Goal:** Plan the actual writing and art work.

**Deliverables:**

**6.1 VOICE_GUIDE.md**
The writing style guide for all game content:
- Tone definition with positive and negative examples
- Narrator voice: who is "speaking" the exploration text?
- NPC voice samples: 5-10 lines per NPC establishing their personality
- Combat narration style
- How humor works in the game (what's funny, what's off-limits)
- Word list: terms that belong to the game's vocabulary (and terms that don't)

**6.2 CONTENT_PLAN.md**
Production schedule:
- All content assets required for v1, grouped by type
- Priority order (what's needed first for testing)
- Estimated production time per asset type
- Which content is AI-assisted drafting (with human editing) vs fully human-written
- Which ANSI art is created by hand vs generated

**Exit criteria for Stage 6:** A writer (whether that's you, AI-assisted, or a contributor) can pick up any content task from the plan, reference the voice guide, and produce on-tone content without further guidance.

---

## After All Stages: Transition to Engineering

When Stages 1-6 are complete, the following documents exist:

| Document | Contains |
|----------|----------|
| DESIGN.md | World bible — setting, lore, full vision |
| NETWORK_REQUIREMENTS.md | Federation constraints (already written) |
| BUILD_SPEC_V1.md | Exact v1 implementation specification |
| DATA_MODEL.md | All data structures, classified for sync |
| VOICE_GUIDE.md | Writing style reference |
| CONTENT_PLAN.md | Content production schedule |

At this point, Track A (game design) is complete and Track B (engineering) begins as defined in PROJECT_PLAN.md Phase B1.

The game design documents remain living references — they will be updated as implementation and playtesting reveal needed changes. But the design *decisions* are made. Engineering is execution, not exploration.

---

## Working Method

Each stage should follow this process:

1. **Draft** — Work through the deliverables. Use AI assistance for brainstorming and drafting. Don't aim for perfection; aim for completeness.
2. **Network check** — Run every mechanic through the NETWORK_REQUIREMENTS.md compatibility checklist. Identify and resolve conflicts.
3. **Simplicity check** — For each mechanic, ask: "Could this be simpler and still be fun?" If yes, simplify. The player's experience should be simple. The underlying systems can be sophisticated. The difference is what the player has to understand vs. what the code handles silently.
4. **Voice check** — Read any written content aloud. Does it sound like the game? Does it hit the Gaiman-meets-Adams register? If it sounds generic, rewrite.
5. **Lock** — Once a stage's deliverables pass all checks, they're locked. Locked doesn't mean frozen forever — it means "we're not reopening this unless testing reveals a fundamental problem."

---

*Document version 0.1 — Process subject to refinement as early stages reveal what works.*
