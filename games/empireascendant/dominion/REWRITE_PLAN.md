# Empire Ascendant — Rewrite Plan for InterDOOR

> Status: Superseded historical planning material. Empire Ascendant/Dominion is deferred until
> Phase 2 or later and requires a fresh design review before implementation. Paths, Makefile
> targets, deployment notes, and binary names in this document are not approved current
> instructions.

## What the original source tells us

Before designing anything: the Pascal source is **partially complete**. The data model, player
creation, config system, and menus are all there. The actual gameplay (`Play_Game`) was never
written — it's a stub with an empty `begin/end`. So this isn't a port; it's a
**design-informed green-field rewrite**. The `.PAS` files give us the data model and intent.
We design the actual gameplay loop from scratch, staying true to the spirit (TradeWars +
Usurper + LoRD).

---

## Architecture

This section is superseded. The earlier plan placed Empire Ascendant directly in the active
InterDoor module and release pipeline. That is no longer approved.

Future Empire Ascendant architecture must be decided during Phase 2 after design review. It may
reuse InterDoor engine/protocol components, but no package path, binary name, port, deployment
host, or Makefile target in this historical plan should be treated as current instruction.

---

## Game identity

- **Name:** Empire Ascendant
- **Theme:** Space empire, sci-fi. Players are rulers of planetary empires competing across
  the galaxy.
- **Player identifier:** Empire Name + World Name (both from original).
- **Daily turns:** 15 (from CONFIG.DOM).
- **Session length target:** 10–20 minutes.

---

## Data model

Derived directly from `One_Player` in DOMINION.PAS, modernized to SQLite. Scoped for v1 —
full tech tree is 50+ fields; we stage it.

### `empires` table

| Field | Type | Notes |
|---|---|---|
| `id` | text PK | global ID (username@node) |
| `username`, `password_hash` | text | auth |
| `world_name`, `empire_name` | text | unique per node |
| `turns_left` | int | resets to 15 daily |
| `turn_day` | int | Unix day number (`epoch/86400`) for reset detection — not day-of-year (avoids year-rollover bug) |
| `money`, `money_bank` | int | |
| `population` | int | grows daily with food surplus |
| `food`, `food_storage` | int | |
| `energy` | int | |
| `research_pts`, `building_pts` | int | accumulated, spent on upgrades |

### `empire_regions` table

One row per region type per empire.

| Field | Notes |
|---|---|
| `empire_id`, `region_type` | PK |
| `quantity` | how many owned |
| `activated` | how many are producing |
| `activate_cost` | scales with quantity |

Region types: Agricultural, Industrial, Desert, Urban, River, Ocean, Volcanic, Wasteland.

### `empire_tech` table

Boolean flags per tech per empire. v1 tech tree — 3 tiers per category:

| Category | v1 tiers |
|---|---|
| Energy | Fossil (default), Fission, Fusion |
| Soldiers | Normal Human (default), SuperHuman, MegaHuman |
| Vehicles | Tank, Hovercraft |
| Ballistic | Nuclear, Antimatter |
| Defense | Ground Turrets, Orbital Satellites, Global Shield |
| Espionage | Intelligence Building (unlocks spies/terrorists) |
| Travel | Hyperdrive (unlocks cross-node warp; requires Fusion energy) |

Full tree (BioBots, HyperHuman, Radiation/Food/Population bombs, etc.) deferred to v2.

### `empire_military` table

Integer counts of each unit type per empire: soldiers (by tier), vehicles, ballistic weapons,
recon drones, defense structures.

### `empire_mines` table

| Field | Notes |
|---|---|
| `empire_id`, `mine_type` | PK |
| `num_mines` | owned |
| `miners_assigned` | from Miners Guild workforce |
| `mineral_left` | depletes over time; mine is exhausted at 0 |

Mine types: Gold, Silver, Iron, Nickel, Copper.

When `mineral_left` hits 0 the mine is inert (still owned, yields nothing). Players can
purchase new mines via the Develop Empire menu (turn action + money cost; see turn actions
table). This is the primary money sink for established players.

### `empire_buildings` table

Per-empire counts/flags: Miners Guild, Fishing Guild, Construction Factories, Research Labs,
Intelligence Building, Lottery.

Fishing Guild owns a `fishers_assigned` integer (workforce headcount, not a boolean). Each
fisher contributes 50 food/day. The guild building must be constructed before fishers can be
hired (turn action: "Hire Fishers", costs money per head). Firing fishers is free and
immediate.

### `pvp_log` table

Incoming attack results, drained on login ("Galactic Dispatches" inbox).

### `attack_queue` table

Outgoing ballistic strikes queued for async processing.

---

## Game mechanics

### Daily production (fires on first login each day)

1. **Food production:** `activated_agricultural * 500 + activated_river * 300 + fishers_assigned * 50`
2. **Food consumption:** `population * 0.1` per day (subtracted from food_storage after production is added)
3. **Population growth:** if `food_storage > 0` after consumption, pop grows 1%; if `food_storage < 0`
   (deficit), pop shrinks 0.5% and `food_storage` is clamped to 0
4. **Energy:** Fossil 100/plant, Fission 500/plant, Fusion 2000/plant
5. **Money from mines:** `miners_assigned * mineral_grade * mine_yield` per type; `mineral_left`
   depletes by `miners_assigned` each day; mine becomes inert at 0
6. **Research Points:** `activated_labs * 10` per day
7. **Building Points:** `activated_construction * 5` per day

### Turn actions (each costs 1 turn unless noted)

| Action | Cost | Effect |
|---|---|---|
| Develop Region | 1 turn + money | Activate a region for production |
| Recruit Soldiers | 1 turn + money | Add units (requires tech) |
| Hire Fishers | 1 turn + money | Assign workers to Fishing Guild (guild req.) |
| Buy Mine | 1 turn + money | Purchase a new mine of chosen type; starts with full `mineral_left` |
| Build Structure | 1 turn + Building Pts | Construct guild/factory/lab |
| Research Tech | 1 turn + Research Pts | Unlock tech tree node |
| Launch Attack | 1 turn (ground) | Attack another empire's army (max 3× same target/day) |
| Fire Missile | 1 turn (ballistic) | Damage enemy resources/pop (max 2× same target/day) |
| Deploy Spy | 1 turn (Intel Bldg req.) | Scout or sabotage target (max 1× same target/day) |
| Warp to Galaxy | 2 turns (Hyperdrive req.) | Travel to another node; warp back costs 2 turns |
| Bank/Withdraw | free | Move money to/from bank |
| Sell Minerals | free | Sell stored minerals for money (no buy side in v1) |

Maintenance (empire report, rankings, news, messages) is always free.

**Attack limits** are per-target per calendar day and enforced server-side. They prevent a
dominant player from grinding the same empire into nothing in a single session. Cross-node
attacks share the same limits.

### Combat resolution (ground assault)

```
attack_power  = sum(unit_count * unit_strength) for each attacker unit type
defense_power = sum(unit_count * unit_strength)
              + (satellites * 50) + (turrets * 20) + (shield_gens * 200)

attacker_wins = attack_power > defense_power * rand(0.8, 1.2)
```

The random multiplier is applied to defense — at equal power the attacker wins ~50% of the
time (when rand < 1.0 in a uniform 0.8–1.2 range, ~50%). This gives defenders a slight
structural edge, which is intentional: attackers initiate risk; defenders have home advantage.

Win: attacker loots 25% of defender's cash. Casualties on both sides proportional to power
ratio.
Lose: attacker loses force entirely, defender takes minor casualties.

Defender sees the result in a "Galactic Dispatches" inbox on next login.

### Ballistic strike

Costs 1 turn + 1 missile. Same-node targeting is available in D3. Cross-node ballistic
targeting is enabled in D4 when the hub queue is wired — the architecture is designed for it
from the start, but the hub transport isn't connected until D4.

| Missile | Effect |
|---|---|
| Nuclear | Kills population, destroys 1 random region |
| Antimatter | Destroys military units |

Intercepted by Orbital Satellites (% chance per satellite). Global Shield blocks all
ballistic while active; costs 1000 energy per day (deducted during the daily production
tick). If energy reserves drop to 0 during the tick, the shield goes offline automatically
until re-activated.

### Spy mission (requires Intelligence Building)

Cost: 1 turn + 1 spy (consumed on failure).

- **Scout:** reveals target's army composition and money (no damage)
- **Sabotage:** destroys 1 random building or tech installation

### Empire score (leaderboard)

```
score = population * 1
      + military_power * 10
      + money * 0.01
      + tech_tier_sum * 500
```

**Balance note:** at these weights, money contributes very little relative to tech and
military (1M money = 10k score; one tier-3 tech = 500). This risks a dominant "ignore
money, stack military/tech" strategy. D5 balance pass should revisit the `money` coefficient
— tentatively `money * 0.1` — and validate against simulated end-state empires.

Rankings visible locally and cross-node via InterDOOR roster.

---

## UI structure

```
  [Title screen: EMPIRE ASCENDANT block art]

  MAIN MENU
    [E] Enter Your Empire
    [R] Rankings -- Top Empires
    [N] Galactic News
    [S] Story / Instructions
    [Q] Quit

  EMPIRE HQ  (after login)
    [T] Your Empire Report   (resources, regions, tech, army summary)
    [D] Develop Empire       (build, research, recruit -- uses turns)
    [A] Attack Menu          (ground assault, missile strike, spy)
    [I] Intel Report         (recon drone results, spy reports)
    [M] Messages             (player-to-player mail)
    [W] Wanderers            (cross-node player list via InterDOOR roster)
    [Q] Quit to Main
```

Each menu option that costs turns shows `[15 turns remaining]` in the header.

---

## InterDOOR integration

| Tier | What | Phase |
|---|---|---|
| 1 -- Listed | Register + heartbeat | D1 |
| 2 -- Network-aware | Events + roster | D4 |
| Cross-node PvP | Attacks via hub queue | D4 |

### Events emitted

| Event type | Trigger | Payload |
|---|---|---|
| `dominion.empire_founded` | new player | `{world_name, empire_name, node}` |
| `dominion.attack_resolved` | any attack result | `{attacker, defender, outcome, loot, casualties}` |
| `dominion.missile_strike` | missile lands | `{attacker, target, missile_type, damage}` |
| `dominion.tech_breakthrough` | player researches tier-3 tech | `{empire, tech}` |
| `dominion.galactic_news` | sysop-broadcast or milestone | `{headline}` |

### Galactic News

The `[N]` feed shows local events + pulled cross-node events. A `dominion.attack_resolved`
on node A shows in the news ticker on node B:
`"[Galactic Dispatch] The Omega Collective of Proxima IV defeated The Iron Fist of New Carthage"`

### Cross-node attacks

Same hub queue mechanism as LoL PvP. Attacker submits to hub; victim node drains on next
`Syncer.Tick()` and resolves locally. Result emitted as `dominion.attack_resolved` and
propagates to all nodes.

### Roster

`[W] Wanderers` shows cross-node empires: Empire Name, World Name, Score, Node. Selecting a
wanderer offers an attack option.

---

## Phase breakdown

### D1 — Walking skeleton
- Define approved package path, binary name, port, database path, and config shape after Phase 2
  design review.
- Schema: `empires` table only
- Player creation: World Name + Empire Name (duplicate check)
- Main menu + Empire HQ shell (stubs)
- `[T] Empire Report` — shows current resources
- Daily turn reset
- SSH connect, login, works end-to-end

### D2 — Economy
- `empire_regions`, `empire_buildings`, `empire_mines`, `empire_tech` tables
- Daily production tick (food, pop growth, energy, research/building pts, mine yield)
- `[D] Develop Empire`: activate regions, build labs/factories, research energy tech
- Bank system
- Mineral market (sell output for money)

### D3 — Military
- `empire_military` table
- Recruit menu (soldiers, vehicles)
- Build defense (turrets, orbital satellites)
- Research weapons tech
- Local ground assault (same-node PvP)
- Ballistic strike (same-node)
- "Galactic Dispatches" inbox — offline attack results
- Spy missions

### D4 — InterDOOR integration
- Federation: register, heartbeat, events, roster
- `Syncer.Tick()` drains cross-node attack queue
- Cross-node PvP (ground + ballistic vs any empire in Wanderers)
- Cross-node ballistic strikes
- Galactic News feed (local + pulled events)
- Cross-node rankings
- **Cross-node travel:** `[W] Warp to Galaxy` option — player "travels" to a connected node,
  temporarily playing as a visitor on that node's instance. Requires Hyperdrive tech (D3 tech
  tree addition). While visiting: can attack local empires, trade minerals, send messages.
  Cannot build or research. Warp costs 2 turns; return costs 2 turns. Travel state is
  persisted (player is absent from home node until they warp back).
- **Inactive empire purge:** empires with 0 logins for 14 days are marked inactive and
  hidden from rankings and Wanderers. After 30 days they are deleted. A warning message is
  shown on login if the player is at risk (12 days inactive).

### D5 — Polish
- ANSI title art (block characters, "EMPIRE ASCENDANT")
- Story text / lore
- Instructions screen
- Balance pass: production numbers, combat ratios, score formula (especially money
  coefficient — tentatively raise from 0.01 to 0.1)

---

## Deployment

Superseded. Empire Ascendant is not deployed and has been removed from the live network. A future
deployment target, port, binary path, database path, firewall rule, and systemd unit must be chosen
during the approved Phase 2 implementation work.

---

## Resolved decisions

1. **Port:** `:2324` on contabo4. **Decided.**
2. **Tech tree scope:** 3-tier v1 as planned. Full tree deferred to v2. **Decided.**
3. **Name:** "Empire Ascendant". **Decided.**
4. **Cross-node travel:** Kept. Implemented as Hyperdrive tech + Warp action in D4.
   InterDOOR's value depends on cross-node interaction; travel is the deepest form of that.
   **Decided.**
5. **Minerals:** Keep all 5 mine types (Gold, Silver, Iron, Nickel, Copper). **Decided.**
