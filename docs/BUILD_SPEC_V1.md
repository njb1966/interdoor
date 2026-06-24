# BUILD SPEC V1

> Status: Historical game-specific source material. The root numbered InterDoor docs are canonical for Phase 1 framework, protocol, operator, and architecture decisions. Do not use this file as Phase 1 framework authority unless it has been explicitly reconciled.

## Dornhaven: The Low — Minimum Viable Game

---

## What This Is

This is the v1 feature specification for Dornhaven: The Low. It defines the minimum playable game — the LORD-equivalent core. Everything in this document ships in v1. Everything not in this document does not, regardless of how good it is.

v1 must be:
- **Fun on day one.** A new player's first session must be engaging without any knowledge of the deeper game.
- **Habitual by day three.** The daily loop must create anticipation. Players should want to come back tomorrow.
- **Deep enough for two weeks.** A daily player should not exhaust v1 content in less than 14 days of play.
- **Simple enough to explain in two minutes.** The mechanics a player needs to understand on day one should be that concise.

**Status: v0.2 — Stage 5 revision.** This revision absorbs the decisions locked in
CORE_LOOP.md (Stage 1), CONTENT_ARCHITECTURE.md (Stage 2), and DATA_MODEL.md. It reflects
design *decisions*, not just intent. Numbers remain provisional (tunable in Phase B2);
the *scope and mechanics* below are locked.

### V1 scope lock (the Stage 5 questions, answered)

- **Districts in v1:** Threshold, Lanternmarket, Warrens, Rafters. (Gutters, Vaults,
  Archive, the Deep are post-v1.)
- **NPCs in v1:** the Lamplighter, the Broker, Maren, Old Thursen. (Floor of 4; more may
  be added in production if the world feels thin.)
- **Factions in v1:** present as **lore and characters only** — no reputation mechanics.
  The four factions exist in dialogue/flavor; alignment, rep, and faction quests are
  post-v1.
- **Endgame in v1:** **no.** v1 is the daily loop. The Archive, the Knowledge system, the
  Deep, and the four endgame paths are post-v1. Anomalies appear as pure flavor that
  seeds those later systems.
- **Progression in v1:** level 1–10 + depth record + gear + debt management. No
  reputation/standing/knowledge tracks.

### Decisions absorbed from Stage 1 (changes vs. this doc's v0.1)

1. **Hybrid action economy** — 12 main actions/day **plus a separate 2 attacks/day** for
   PvP, plus free activities. PvP is no longer "2 main actions."
2. **PvP does not kill in v1** — a defeated victim is looted, not killed (retention
   choice; revisit in B2, candidate sysop toggle).
3. **Free inter-district movement** — death's cost is lost carried gear + a progress
   penalty, not an action tax.
4. **Wake at full HP daily**; Rest heals 30% MaxHP mid-day.
5. **Trade shortfalls create a debt** — the favor/debt economy appears in the first trade.
6. **Victim-side PvP cap (3×/day)** — anti-pile-on rule.

The mechanics sections below are updated to match. See CORE_LOOP.md for exact formulas
and numbers.

---

## The Two-Minute Explanation

You woke up underground. Nobody above remembers you exist. You're in the Low — a vast, old, dangerous place beneath the city of Dornhaven. You need to survive.

Every day you get **12 actions** — spend them exploring the tunnels (find loot, fight creatures, discover things), trading at the market (gear up, sell scavenged goods, deal in favors), or working contracts. Talking to the locals, moving around, and messaging are free. Separately, you get a couple of **attacks** you can spend on other players while they sleep — or find out someone spent theirs on you.

Everything runs on favors and debts. No money. You help someone, they owe you. You need something, you owe them. Be careful what you owe. The Low keeps track.

Survive long enough, go deep enough, and you might start to understand why this place exists — and how to get out.

---

## V1 Features

### Player Account
- Create account (name, password)
- Login / logout
- Character persists between sessions
- Basic stats: Health, Strength, Defense, Luck
- Inventory: carried items (limited capacity) and banked items (stored in the Rafters)

### Daily Actions  *(see CORE_LOOP.md §1.1 for the full cost table)*
- **12 main actions/day** — explore, trade, contract work, rest
- **2 player-attacks/day** — a separate budget for offline PvP (so PvP never eats your exploration)
- **Free:** talk to NPCs, read lore, message players, check status/debt board/help, move between districts, accept/post contracts
- Reset at server midnight (configurable); **no rollover**; count does not scale with level
- You **wake at full HP**; **Rest** (1 action) heals 30% MaxHP mid-day

### Districts (v1 subset)

**The Threshold**
- New player arrival sequence (first session only)
- Brief tutorial: movement, interaction, basic mechanics
- Introduction by the Lamplighter NPC
- Transition to the Lanternmarket

**The Lanternmarket**
- Hub area: accessible from all other districts
- Buy/sell goods with NPC merchants
- View and accept contracts (NPC-posted)
- Check the debt board (outstanding obligations)
- View other players (who's active, basic info)
- Post messages to other players

**The Warrens**
- Primary exploration/combat area
- Shallow exploration (2 actions, lower risk/reward)
- Deep exploration (3 actions, higher risk/reward)
- Procedural encounters from tables (combat, discovery, hazard, NPC, anomaly)
- Loot drops from encounter tables
- Depth tracking (how far you've gone — personal record)

**The Rafters**
- Player housing (basic — a room to store banked items)
- Rest/recovery (1 action, heal partial HP)
- Access faction NPCs (but faction system is post-v1 — NPCs present as characters, not quest-givers)
- Ambient NPC presence and dialogue

### Combat
- Turn-based, player vs creature
- Simple action set: Attack, Defend, Use Item, Flee
- Outcome determined by stats + gear + randomness
- Creatures have defined stats, behavior (aggressive, defensive, evasive), and loot tables
- Combat narration with personality (tone matters here)
- Death on HP reaching zero (see Death section)

### PvP  *(see CORE_LOOP.md §1.5)*
- Attack **sleeping players** (offline PvP) — you fight their stored state
- Costs **1 of your 2 daily attacks** (not a main action)
- Outcome from both players' stats + gear + randomness (same combat engine)
- **Winner loots** a portion of the loser's **carried** inventory; **banked items are safe**
- **PvP does not kill in v1** — the loser is looted and left at low HP, not killed (retention choice; revisit in B2, candidate sysop toggle)
- Loser is notified at next login ("While you slept, [player] attacked you...")
- **Limits:** can't attack the same player two days running; can't attack players more than 3 levels below you; a player can be attacked at most 3×/day; new players are protected until level 2 / tutorial complete

### Economy  *(see CORE_LOOP.md §1.3)*
- **No currency.** Everything is barter, favors, and debts.
- **Goods:** Scavenged items carry an abstract **trade value** (`trade_weight`). NPC merchants buy low (~55%) and sell high (~130%) of it.
- **Trade shortfalls create a debt:** when your goods don't cover a purchase, the merchant fronts the difference as an obligation — so the favor/debt economy appears in your very first trade, with no extra UI.
- **Favors** = an obligation where you're owed; **Debts** = one you owe. Same record, two directions (DATA_MODEL.md §1.3). NPCs may **call in** a favor as a small errand. For v1, obligations are **NPC-only** (player-to-player favors are post-v1).
- **Debt pressure is friction, not punishment in v1:** high debt → merchant markup, the Broker's pointed comments, some services refused, flavor "Ledger attention." The mechanical Ledger-attention system is post-v1.

### Contracts
- NPC-posted daily contracts: go somewhere, find something, deliver something, survive something
- Accepting a contract costs 0 actions; completing it costs normal action costs for the activities involved
- Rewards: goods, gear, reduced debts, reputation (post-v1), lore hints
- Contracts expire if not completed within their timeframe
- 2-3 new contracts available per day

### NPCs (v1 cast)
- **The Lamplighter** — Guide figure. Orients new players. Cheerful, helpful, evasive about the past. Provides tutorial information and hints. Always in the Threshold and Lanternmarket.
- **The Broker** — Manages contracts and trade disputes. Scrupulously fair. Dry, precise, faintly menacing. Primary economic NPC. Lanternmarket.
- **Maren** — A Warrens scavenger. Tough, pragmatic, dark humor. Sells gear, buys scavenged goods. Gives hints about what's deeper in the Warrens. Lanternmarket / Warrens entrance.
- **Old Thursen** — A Rafters resident who's been Low for decades. Talkative, a bit unreliable, full of stories. Primary lore delivery NPC. Drops hints about the Charter, the Ledger, the factions. Rafters.

Additional NPCs may be added during content production if the game feels underpopulated. Four is the minimum for v1.

### Progression (v1 version)
- **Character level** — Simplified for v1. Level 1-10. Level determines access to deeper Warrens areas and harder contracts. Leveling through accumulated actions (explore, fight, complete contracts), not XP numbers. The game tracks it internally; the player sees their level and rough progress to next.
- **Depth record** — How deep you've gone in the Warrens. Bragging rights and access gate.
- **Debt load** — Total outstanding debts. High debt triggers narrative warnings from NPCs and minor gameplay friction (merchants charge more, the Broker makes pointed comments).

Full reputation/standing/knowledge systems are post-v1. In v1, progression is level + depth + gear + debt management.

### Death  *(see CORE_LOOP.md §1.6)*
- HP reaches zero (creature or hazard) → death. **PvP does not kill in v1.**
- **Carried inventory lost; banked inventory safe; debts persist**
- Wake up in the **Threshold** next day (still you, not a newcomer), **at full HP**
- **Movement is free**, so returning costs no actions — the real cost is the lost gear plus the penalty below
- Small **progress penalty** (−50% level progress for one cycle; no level loss)
- Notification of what killed you

### Lore Delivery (v1)
- No formal knowledge system in v1
- Lore is delivered through NPC dialogue (Old Thursen especially), environmental descriptions in the Warrens, and contract flavor text
- Anomalies appear in deep Warrens exploration — strange descriptions, numbers on walls, doors that shouldn't exist. Pure flavor in v1, but they foreshadow the systems that become mechanical in later phases.
- Players who pay attention will start to piece together that the Low isn't random — something is organizing it. This is setup for the faction/knowledge/endgame systems in later phases.

### ANSI Art
- Title screen (Dornhaven: The Low)
- District headers (Lanternmarket, Warrens, Rafters, Threshold)
- Death screen
- Combat encounter frames (2-3 reusable frames, not per-creature)
- PvP result screens
- Daily login summary screen
- Help screen borders/headers

Target: 10-15 ANSI art assets for v1. Quality over quantity. Every screen the player sees frequently should look good.

### Help System
- In-game help accessible from any screen
- Covers: actions, combat, trading, debts, PvP, districts, commands
- Brief, in-tone (help text is written in the game's voice, not technical documentation voice)

---

## Explicitly NOT in V1

These features exist in the full DESIGN.md but are deferred:

- Faction system (factions exist as lore, NPCs are present, but no reputation mechanics)
- Knowledge fragment system
- The Archive district
- The Deep district
- Endgame confrontation / Descent
- Player-to-player favor trading
- Player-posted contracts
- The Gutters district
- The Vaults district
- Settlement/communal building
- Ledger attention mechanic (beyond NPC dialogue flavor)
- Cross-node travel (federation Phase B3)
- Game extension / IGM system

These are not cut. They are phased. The design for each exists in DESIGN.md and will be specified in detail during Game Design Stages 3-4. They enter the game in post-v1 phases after the core loop is proven.

---

## Content Volume Targets

Minimum content required for v1 to sustain two weeks of daily play without obvious repetition:

| Content Type | Target Count |
|-------------|-------------|
| Warrens creature types | 15-20 |
| Warrens non-combat encounters | 10-15 |
| Warrens environmental hazards | 8-10 |
| Warrens anomaly descriptions | 5-8 |
| Scavengeable item types | 20-25 |
| Weapons | 6-8 (tiered by level) |
| Armor | 6-8 (tiered by level) |
| Consumable items | 5-8 |
| NPC dialogue lines per NPC | 20-30 (rotating daily) |
| Contract templates | 10-15 |
| ANSI art screens | 10-15 |
| Ambient district descriptions | 5-8 per district |

These are estimates. Playtesting in Phase B2 will reveal whether the volumes are sufficient.

### Content Inventory (Stage 5.2)

The table above is the v1 content inventory. Each line fills a template defined in
CONTENT_ARCHITECTURE.md, so production can begin asset-by-asset:

| Asset type | Count | Fills template |
|---|---|---|
| ANSI art screens | 10–15 | district headers (4), title, death, 2–3 combat frames, PvP result, daily summary, help borders (CONTENT_ARCH §0/§2.1) |
| NPC dialogue lines | 20–30 × 4 NPCs | `dialogue_pool` (CONTENT_ARCH §2.2) — rotating daily, short arcs |
| Creatures | 15–20 | `creature_catalog` (§2.3) |
| Non-combat encounters | 10–15 | discovery / walker templates (§2.3) |
| Environmental hazards | 8–10 | hazard template (§2.3) |
| Anomalies | 5–8 | anomaly template (§2.3) — seed tags for post-v1 |
| Scavengeable items | 20–25 | `item_catalog` trade/curio (§2.4) |
| Weapons / Armor / Consumables | 6–8 / 6–8 / 5–8 | `item_catalog` (§2.4) |
| Contract templates | 10–15 | `contract_template` (§2.5) |
| Ambient descriptions | 5–8 × 4 districts | `text_pool` (§2.1) |
| Tutorial / help text | ~3–5 pages | help system + Threshold onboarding (§2.1) |

Production order, voice rules, and human-vs-AI-drafting split are scheduled in
CONTENT_PLAN.md (Stage 6); tone is governed by VOICE_GUIDE.md (Stage 6).

---

## Success Criteria

v1 is successful if:

1. A new player completes their first session and returns the next day.
2. A daily player after one week can describe the game to someone else and make it sound interesting.
3. The daily session takes 10-15 minutes and feels like enough — not too short, not a chore.
4. Players attack each other and have opinions about it.
5. Players talk about the game outside the game (on BBSes, in chat, in forums).
6. The developer enjoys playing it.

---

*Document version 0.2 — Stage 5 revision. Scope and mechanics locked against CORE_LOOP.md, CONTENT_ARCHITECTURE.md, and DATA_MODEL.md. Numbers tunable in Phase B2.*
