# CORE LOOP — Stage 1 Lock

> Status: Historical game-specific source material. The root numbered InterDoor docs are canonical for Phase 1 framework, protocol, operator, and architecture decisions. Do not use this file as Phase 1 framework authority unless it has been explicitly reconciled.

## The Daily Heartbeat (Mechanics + Numbers)

*Stage 1 deliverable (per GAME_DESIGN_PROCESS.md). Status: v0.1.*

---

## What this document is

This is the **Core Loop Lock**: the fundamental thing every player does every session,
specified precisely enough to implement, with numbers defined. Per the design process,
this feeds the Stage 5 revision of BUILD_SPEC_V1.md and the numeric gaps left open in
DATA_MODEL.md.

**Reading conventions:**
- **(T)** marks a *tunable* value — a balance guess that will move during Phase B2
  playtesting. Its *presence* is locked; its *number* is provisional.
- **(S)** marks a *structural* decision — changing it changes the design, not just the
  balance.
- **Network check** lines confirm each mechanic against NETWORK_REQUIREMENTS.md and
  point to the relevant DATA_MODEL.md structure.
- Names remain placeholders; nothing here is a NAME DECISION POINT.

**Guiding constraint (unchanged):** very LoRD-like. Simple to play, fast (10–15 min/day),
the world evolves while you are gone. Depth and character come from *content*, not from
mechanical complexity. If a rule below could be simpler and still fun, that is a bug.

---

## The two-minute explanation (updated for the hybrid action model)

You woke up underground. Nobody above remembers you. You're in the Low, beneath the
city. Survive.

Each day you get **15 expeditions** into the Warrens (find loot, fight things, go deeper).
**Town is free** — trade, rest, see the bonesetter, read the news, take stock, eye the
other wanderers. Separately, you get **3 attacks** you can make on other players while
they sleep — and you'll find out if someone hit you.

No money. Everything runs on goods, favors, and debts. Help someone, they owe you. Take
something on credit, you owe them. The Low keeps track. Survive long enough, go deep
enough, and you might start to understand why this place exists.

---

## 1.1 Action Economy *(S: split-pool model — REVISED after B2 playtest)*

> **Playtest revision (2026-06-22):** the original hybrid model (12 shared "actions",
> explore costing 2–3) ran dry far too fast and felt constrained — a player did ~4 fights
> and was done. Replaced with a **split-pool** model closer to real LoRD: a generous
> dedicated expedition budget, a small separate PvP budget, and **free town actions**.
> Town healing is split into free Rest (limited) + a paid Bonesetter (goods/debt).

### The three buckets

| Bucket | Daily amount | Refills | Spent on |
|---|---|---|---|
| **Warren expeditions** | **15** (T) | Server midnight (configurable), no rollover | One Warren expedition (shallow or deep) each |
| **Player attacks** | **3** (T) | Server midnight, no rollover | Offline PvP only (§1.5) |
| **Free (town)** | unlimited | — | Trade, Rest, Bonesetter, News, character sheet, other wanderers, menus, messaging |

Separate budgets mean exploration and PvP never cannibalize each other, and a player can
always *do town things* without burning their fighting. This is the LoRD rhythm.

### Action cost table

| Activity | Cost | Bucket |
|---|---|---|
| Explore Warrens — shallow | 1 | Expedition |
| Explore Warrens — deep | 1 (harder band, richer) | Expedition |
| Attack a sleeping player | 1 | Attacks (3/day) |
| Trade at Maren's stall | 0 | Free |
| Rest (heals ~30% HP) | 0 — but limited to **3/day** | Free (capped) |
| The Bonesetter (full heal) | 0 actions — costs **goods, or a debt** | Free |
| News / character sheet / other wanderers / menus | 0 | Free |
| Message another player | 0 | Free |

*(Contracts deferred post-v1; their costs are removed here until that system lands.)*

> One **trade session** = one action, not one transaction. Sit at a merchant, buy and
> sell as much as you like, leave. This prevents action-nickel-and-diming and keeps
> trading fast. (T)

### Standing rules

- **No rollover.** Unspent actions vanish at reset. This is what drives the daily habit.
- **Action count is fixed at 12 in v1 — it does NOT scale with level.** (S) Keeps
  balance flat and the game LoRD-simple. Extra-action *consumables/curios* are a clean
  future lever; not in v1.
- **Injuries do not reduce actions.** (S) Being hurt is a *risk you manage* (you might
  die exploring at low HP), not an action tax. You may Rest to heal, or push your luck.
- **You wake at full HP each day.** (T, S) The daily reset restores HP to max. Mid-day,
  **Rest** (1 action) heals **30% of MaxHP** (T). This is generous on purpose — LoRD
  starts you fresh each day, and daily-retention games should not punish yesterday's
  bad fight into today.

**Network check:** All action/turn state is **Local** (`turn_state` in DATA_MODEL.md:
`actions_remaining`, `last_reset`, `day_index`; a parallel `attacks_remaining`). No
other node needs it. The daily turn is an *engine* concept; the number 12 and the cost
table are *game* config. ✓

---

## 1.2 Exploration / Combat *(core of the loop)*

### Stat model (v1) *(S)*

Four stats, LoRD-tiny: **HP/MaxHP, Strength, Defense, Luck.** Levels **1–10**.

Base stats by level (before gear):

| Lv | MaxHP | Str | Def | Luck |
|----|------|-----|-----|------|
| 1  | 20   | 10  | 4   | 5 |
| 2  | 30   | 13  | 6   | 5 |
| 3  | 42   | 16  | 8   | 6 |
| 4  | 56   | 20  | 11  | 6 |
| 5  | 72   | 24  | 14  | 7 |
| 6  | 90   | 29  | 17  | 7 |
| 7  | 110  | 34  | 21  | 8 |
| 8  | 132  | 40  | 25  | 8 |
| 9  | 156  | 46  | 29  | 9 |
| 10 | 182  | 53  | 34  | 9 |

All values **(T)**. Gear adds on top (weapons → effective attack, armor → effective
defense, some curios → Luck/HP).

**Leveling:** by accumulated `level_progress`, earned from exploration, combat wins, and
contract completion — **not** a visible XP number (BUILD_SPEC). The player sees their
level and a rough bar. Progress-per-level curve and event values are **(T)**, defined
in the BUILD_SPEC revision.

### One action = one encounter *(S, LoRD-faithful)*

A single explore action resolves **one encounter** from the depth-appropriate table,
then returns you to the menu. No multi-room "expeditions" in v1 — that keeps sessions
fast and mirrors LoRD's one-fight-per-forest-action rhythm.

### Depth *(S, simple)*

Depth is a single integer band.
- **Shallow explore (2):** encounter from band ≈ your level. Lower risk/reward.
- **Deep explore (3):** encounter from band = your current `depth_record` + 1 — pushes
  deeper, higher risk/reward. Surviving a new-record band increments `depth_record`.
- Depth gates content and brags. Banding math is **(T)**.

### Encounter type frequency

Weighted roll per explore action (T):

| Type | Shallow | Deep |
|---|---|---|
| Combat (creature) | 55% | 60% |
| Discovery (scavenge site) | 20% | 15% |
| Hazard (environmental) | 10% | 12% |
| Walker (NPC encounter) | 10% | 8% |
| Anomaly (Ledger flavor) | 5% | 5% |

Encounter content lives in **Shared-Reference** tables (`encounter_table`,
`creature_catalog`, `loot_table` — DATA_MODEL.md §2). This is where variety and
character live; targets are in BUILD_SPEC content volumes.

### Combat *(S: round-by-round, four verbs)*

Turn-based, one creature, you choose each round from: **Attack, Defend, Use Item, Flee.**
Fast and a little swingy — LoRD, not a tactics game.

Per-round resolution:

```
attack_power  = Strength + weapon_bonus
defense_power = Defense  + armor_bonus
raw           = max(1, attack_power - defense_power / 2)
variance      = random 0.8 .. 1.2
damage        = max(1, round(raw * variance))
crit          = (random < Luck%)  ->  damage *= 1.5
```

- **Attack:** deal `damage` to the creature; creature then attacks you (same formula,
  its stats).
- **Defend:** incoming damage this round is halved; you do not attack (or attack at
  half — pick one in implementation; default: no attack). (T)
- **Use Item:** consume a consumable (heal/buff) this round; creature still attacks.
- **Flee:** success = **40% + Luck×2% − depth_penalty** (T). Success → leave combat,
  keep what you carry, the action is already spent. Failure → creature gets one free
  hit and combat continues.
- **Crit chance = Luck × 1%** (Luck 5 → 5%); crit = **1.5× damage** (T).

**Tuning intent (sanity-checked, all T):** a player fighting at-level in **shallow**
wins in ~2–3 rounds taking modest damage; **deep** fights are genuinely dangerous for
the under-leveled or under-geared — that danger *is* the depth gate. A careful player
dies rarely (<~once/week); a greedy deep-diver dies sometimes. Target: ~5–10% of
at-level deep dives kill an under-geared player.

### Loot

Combat win rolls the creature's `loot_table`; discovery rolls a discovery table. **Luck
biases rarity.** Drops are mostly trade goods, with occasional gear/consumables.
Rarity distribution is **(T)** per `loot_table`.

### "The Warrens shift between days" *(S: mechanical at the seed, cosmetic to the player)*

No persistent map in v1. Each day's `day_index` reseeds the encounter RNG; descriptions
vary so it *feels* like the place rearranged. Anomaly flavor leans into this. No stored
geometry.

**Network check:** An expedition's encounter sequence and loot rolls are **Local**
(`warren_session`, in-memory, seed-reproducible). Only results that touch synced state
emit events: a death → `player.died`; a notable loot or trade-induced debt is handled by
its own system. Encounter/creature/loot tables are **Shared-Reference**. ✓

---

## 1.3 Trading / Economy *(the differentiator, in the daily loop)*

### No currency — barter on trade-weight

Items carry a **`trade_weight`** (abstract barter value, Shared-Reference). There is no
coin. Merchants:
- **Buy** scavenged goods at ~**55%** of `trade_weight` (T).
- **Sell** gear/consumables at ~**130%** of `trade_weight` (T).
- The spread is their margin and the sink that keeps the scavenge→trade loop alive.

### How debt enters routine play *(S — elegant integration)*

Barter rarely matches exactly, and exact-change haggling is tedious. So: **when you come
up short, the merchant covers the difference as a debt.** Buy gear worth 50, hand over
goods worth 40 → you walk out with the gear and a **10-weight debt** (an `obligation`,
DATA_MODEL.md §1.3). Conversely, overpay and you're owed a small **favor**.

This is how the favor/debt economy — the network's centerpiece — shows up in the *very
first* trade a player makes, without any extra UI. v1 obligations remain **NPC-only**
(BUILD_SPEC); player-to-player favors are post-v1.

### Other ways debt is incurred

- **Healing on credit** (a healer NPC service) when you can't pay in goods.
- **Gear advanced** by Maren before you've scavenged enough.
- **Contract advances** (materials fronted for a job).
- **Safe passage / services** from NPCs.

### Tracking & resolution

- All obligations appear on the **debt board** (free to check) and feed `debt_load`
  (derived; DATA_MODEL.md §1.2).
- **Resolve** by: repaying in goods, completing a **called-in favor** (an NPC asks you
  to do a task to clear it), or **forgiveness** (rare, narrative).
- NPCs may **call in** a favor they hold over you — turning an abstract debt into a small
  errand/contract. This is a key content hook for the Broker and Maren.

### Unpaid-debt consequences *(v1: friction only, no hard Ledger punishment)*

Escalating with `debt_load` (thresholds T):
- **Merchant markup** scales with `debt_load` (you're a credit risk).
- **The Broker comments** — pointed, dry, escalating (pure content/voice).
- Above a high threshold, **some NPC services are refused** until you pay down.
- **Flavor "Ledger attention"** hints in dialogue/ambient text.

Per BUILD_SPEC, the *mechanical* Ledger-attention system is explicitly **post-v1**. v1
debt pressure is economic friction + voice, nothing that hard-punishes a daily player.

**Network check:** Obligations are **Synced (Broadcast)** with global IDs and emit
`debt.created` / `debt.resolved` / `debt.transferred` (DATA_MODEL.md §1.3, Part 3) — even
though v1 is single-node, so federation needs zero rework. Local market inventory and
merchant prices are **Local** (`market_state`); the item catalog and `trade_weight` are
**Shared-Reference**. ✓

---

## 1.4 NPC Interaction *(S: menu-driven)*

### Format

**Menu-driven**, LoRD-style: arrive at a location, see a short menu of who's there and
what you can do; pick a verb. No free-text parser, no keyword hunting — fast, 80×24
friendly, and the right amount of "game show host" energy for the tone. Talking is
**Free** (no action cost).

### v1 cast (4)

| NPC | Where | Role | System hooks |
|---|---|---|---|
| **The Lamplighter** | Threshold, Lanternmarket | Guide / tutorial / hints | Onboarding; rotating hints; evasive lore |
| **The Broker** | Lanternmarket | Contracts, trade disputes, debt board | Posts contracts; comments on `debt_load`; calls in favors |
| **Maren** | Market / Warrens entrance | Scavenger merchant | Buys/sells gear & goods; advances gear on credit; deep-Warren hints |
| **Old Thursen** | Rafters | Lore | Primary Charter/Ledger/faction foreshadowing |

(4 is the floor; more may be added in production if the world feels thin — BUILD_SPEC.)

### Dialogue model

- Each NPC has a **dialogue pool** (Shared-Reference). On each new `day_index`, the
  rotation advances — the player gets fresh lines daily (1–3 new) so talking is worth
  doing every day. (T: lines/day)
- **Short multi-day arcs** via `arc_step`: a handful of NPCs (esp. Old Thursen) reveal a
  sequence across successive days or on triggers (e.g., you hit a depth milestone). Arcs
  are short in v1.
- **Quests/contracts** are delivered as **contracts** (the Broker and Maren) or, for Old
  Thursen, mostly as lore + the occasional "go look at X" pointer (flavor, not a
  tracked quest in v1).
- **Lore/hint delivery:** Old Thursen (primary), Lamplighter (hints), anomalies +
  ambient text (secondary). The drip that sets up the post-v1 faction/knowledge systems.

**Network check:** NPC dialogue progress and arc cursors are **Local**
(`npc_dialogue_state`). NPC definitions and dialogue pools are **Shared-Reference**. No
sync implications. ✓

---

## 1.5 PvP *(S: offline-only, separate 2/day budget)*

### How it works

You attack another player's **stored state while they're offline** (LoRD model; required
by NETWORK_REQUIREMENTS.md Req 3). Costs **1 of your 2 daily attacks** — not a main
action. Resolution uses the §1.2 combat engine: attacker chooses actions; the defender
fights with their stats+gear on **auto-defense**.

### Stakes

- **Win → loot.** Attacker takes a portion of the victim's **carried** inventory:
  ~**25–50%** of carried trade goods, plus a **chance** at one carried gear item (T).
  **Banked items are always safe** (DATA_MODEL.md §1.4 `location`).
- **(S) PvP does not kill in v1.** A defeated victim is left at low HP and looted, not
  killed. **Rationale:** getting killed in your sleep and losing everything is a classic
  quit-trigger; for a daily-retention game we loot rather than execute. This is a
  deliberate divergence from LoRD's lethal PvP — **revisit in B2 playtest**; it could
  become a sysop toggle.
- **Notification:** the victim sees it at next login — *"While you slept, [player]
  attacked you…"* (`pvp_inbox`, DATA_MODEL.md §1.6).

### Limits *(anti-grief)*

- **Cooldown:** can't attack the **same** player two days running (BUILD_SPEC).
- **Level floor:** can't attack players more than **3 levels below** you (BUILD_SPEC).
  Punching **up** is allowed (and risky).
- **Victim cap:** a player can be attacked at most **3×/day** (T) — prevents pile-ons.
  *(New rule, flagged for BUILD_SPEC revision.)*
- **New-player grace:** can't be attacked until **level 2** / tutorial complete (T).
- **Debt interaction:** v1 has no player-to-player obligations, so "can you attack
  someone who holds your debt?" is **moot in v1**. When player favors arrive (post-v1),
  attacking a creditor you owe should carry a penalty/default — deferred.

**Network check:** v1 PvP is same-node → **Local** resolution, but it already emits
`pvp.resolved` (game-defined event) and references both players by **global ID**, so B3
cross-node PvP (attack syncs to victim's home node → resolves there → result syncs back,
tolerating delay) needs no redesign. If a future toggle re-enables PvP kills, a
`player.died` rides alongside. ✓

---

## 1.6 Death and Consequences

### Trigger
HP reaches 0 from a **creature or hazard** (PvP does not kill in v1, §1.5).

### Effects

| What | Outcome |
|---|---|
| **Carried inventory** | **Lost** (scavenged by what killed you / absorbed by the Warrens). |
| **Banked inventory** | **Safe** (Rafters / future Vaults). |
| **Debts** | **Persist.** Death does not clear the Ledger. |
| **Respawn** | Wake in the **Threshold** next day — still you, still known, not a newcomer. |
| **Return cost** | Movement is free, so returning costs no actions. The real cost is the lost carried gear + the penalty below. *(Diverges from BUILD_SPEC's "walk back costs actions" — see Decisions below.)* |
| **Progress penalty** | `death_penalty_active` → **−50% `level_progress` gain for one cycle** (T). No level loss. |
| **Reputation** | No effect in v1 (reputation is post-v1; the BUILD_SPEC "small rep hit" lands with that system). |
| **Notification** | You're told what killed you and where. |

### Frequency target *(T)*
Death stings without being devastating. Cautious play: rare (<~once/week). Greedy
deep-diving while under-geared: occasional. Balanced so the threat is real but never a
quit-trigger.

**Network check:** Death is **Local** state change + a **Broadcast** `player.died` event
(self-contained: cause, district, depth). ✓

---

## Exit-criteria check (GAME_DESIGN_PROCESS.md Stage 1)

- [x] **1.1 Action economy** — buckets, counts, cost table, rollover, free activities,
      level/injury effects all specified with numbers.
- [x] **1.2 Exploration/combat** — stat model, level curve, encounter generation,
      combat formula + four verbs, depth, loot, daily shift.
- [x] **1.3 Trading/economy** — barter model, debt-on-shortfall, incur/track/resolve,
      unpaid consequences (friction-only in v1).
- [x] **1.4 NPC interaction** — menu-driven, 4-NPC cast, daily rotation, short arcs,
      quest/lore delivery.
- [x] **1.5 PvP** — offline attacks, separate budget, stakes, anti-grief limits,
      cross-node-ready.
- [x] **1.6 Death** — losses/persistence, respawn, penalty, frequency target.
- [x] Every mechanic passes its **network check** and maps to a DATA_MODEL.md structure.
- [x] Core loop explainable in **under two minutes** (above).
- [x] All numbers defined (each guess marked **(T)** for B2 rebalance).

---

## Decisions & divergences from prior docs (for the Stage 5 BUILD_SPEC revision)

These are deliberate choices made during the lock that the BUILD_SPEC revision should
absorb. Per the project's "stop and report if the plan changed" rule, they are called
out rather than buried:

1. **Hybrid action economy** *(your decision)* — PvP moves to a separate **2 attacks/day**
   budget; it is **no longer "2 main actions"** as BUILD_SPEC currently says. Update the
   action table accordingly.
2. **PvP does not kill in v1** — loots instead. Diverges from LoRD/BUILD_SPEC implication.
   Flagged for B2 playtest; candidate sysop toggle.
3. **Free inter-district movement; death's cost is gear + penalty, not an action tax** —
   reconciles BUILD_SPEC's "walk back costs actions." Simpler and more LoRD-like.
4. **Wake at full HP daily; Rest heals 30% MaxHP mid-day** — extends BUILD_SPEC; a
   retention-friendly default.
5. **Trade-shortfall creates a debt** — concrete mechanism that puts the favor/debt
   differentiator into the very first trade. Consistent with BUILD_SPEC ("debts via
   trade"), now specified.
6. **Victim-side PvP cap (3×/day)** — new anti-grief rule not in BUILD_SPEC.

---

*Document version 0.1 — Stage 1 Core Loop Lock. Structure locked; numbers provisional
and marked (T). Will be rebalanced in Phase B2 and folded into the BUILD_SPEC revision
at Stage 5.*
