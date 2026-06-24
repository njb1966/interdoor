# CONTENT ARCHITECTURE — Stage 2

> Status: Historical game-specific source material. The root numbered InterDoor docs are canonical for Phase 1 framework, protocol, operator, and architecture decisions. Do not use this file as Phase 1 framework authority unless it has been explicitly reconciled.

## The Structures Content Fills

*Stage 2 deliverable (per GAME_DESIGN_PROCESS.md). Status: v0.1.*

---

## What this document is

Stage 2 defines the **content structures** for the v1 game — districts, NPCs,
encounters, items, contracts — as **fillable templates**, not finished content. Per the
process: *"Not writing the content yet, but defining the structures it will fill."*
A small number of in-tone **samples** are included to prove each template works and to
point the voice; the full content volume (BUILD_SPEC targets) and the formal voice
guide are Stage 6.

These templates are the concrete form of the **Shared-Reference** catalogs in
DATA_MODEL.md §2, populated to satisfy the mechanics locked in CORE_LOOP.md.

**Conventions:** **(T)** = tunable value. **(S)** = structural decision. Each section
ends with a **Network check**. Names remain placeholders; sample IDs (`crt_…`, `wpn_…`)
are catalog keys, not NAME DECISION POINTS.

**Guiding constraint (unchanged):** very LoRD-like. The templates are intentionally
small. *Depth and character live in the content that fills them*, not in the field
count. If a template grows fields the game doesn't need, that's a bug.

---

## 0. The standard screen & navigation *(S)*

Everything renders in **80×24, base-16 ANSI** (NETWORK_REQUIREMENTS.md Req 8). Every
district uses one screen template so the game feels consistent and is cheap to build:

```
+------------------------------------------------------------------------------+  rows
|                    [ ANSI DISTRICT HEADER ART  -- ~5 rows ]                   |  1-6
+------------------------------------------------------------------------------+
  <ambient atmosphere line, drawn from this district's text_pool>                 8
                                                                                  9
  Here you can:                                                                  10
    [E] Explore the Warrens (shallow)              -2 actions                    11
    [D] Explore the Warrens (deep)                 -3 actions                    ..
    [T] Talk to Maren                               free
    [M] Lanternmarket                               free  (move)
    ...
                                                                                  22
+------------------------------------------------------------------------------+
| HP 18/20   Actions 9/12   Attacks 2/2   Debt 12   Lv 2  >>                    |  24
+------------------------------------------------------------------------------+
```

- **Header art** = `district_def.header_art_ref` (Shared-Reference). One per district.
- **Ambient line** = a rotating pick from the district's `text_pool` (varies by
  `day_index` so the place feels alive). (T: how many lines per district)
- **Menu** = the district's available verbs + visible NPCs + movement options. Action
  costs shown inline; free verbs marked `free`.
- **Status line (row 24)** is global and always present: HP, main Actions, PvP Attacks,
  `debt_load`, Level. Color is atmosphere only — the line is readable in monochrome.

**Navigation graph (v1) — hub-and-spoke, movement is free (CORE_LOOP §1.1):**

```
        [Threshold]
             |  (arrival / respawn; little reason to return)
             v
      [Lanternmarket] <-----> [Rafters]
             ^
             |  (entrance)
             v
        [Warrens]   (depth is reached via Explore actions, not movement)
```

The **Lanternmarket is the hub**; all routine movement passes through it. Depth within
the Warrens is traversed by spending explore actions, not by menu movement.

---

## 2.1 District Specifications

Template (one per district; backs `district_def`):

```
district_id      : <key>
name             : <display>
depth / danger   : <from DESIGN.md>
functions[]      : <gameplay verbs available here>
nav_links[]      : <districts reachable from here (free movement)>
present_npcs[]   : <npc_ids visible here>
services[]       : <merchant/heal/contract/bank/etc.>
ambient_pool_id  : <text_pool key>
header_art_ref   : <ANSI asset key>
notes            : <atmosphere & special rules>
```

### The Threshold
```
district_id   : dst_threshold
functions     : guided onboarding (first session); respawn point (on death)
nav_links     : Lanternmarket
present_npcs  : npc_lamplighter
services      : none (it is a funnel, not a hub)
header_art    : art_hdr_threshold
notes         : Tutorial only on first session — teaches menu, movement, one combat,
                one trade, the debt concept, then routes to the Lanternmarket. Also the
                death respawn point (CORE_LOOP §1.6): you wake here, no actions spent,
                walk back free. Cramped, dark, dripping. (S) After onboarding it holds
                no recurring content — by design.
```

### The Lanternmarket  *(the hub)*
```
district_id   : dst_lanternmarket
functions     : trade session; view/accept contracts; check debt board; view active
                players; message players; talk to NPCs
nav_links     : Threshold, Warrens, Rafters
present_npcs  : npc_lamplighter, npc_broker, npc_maren
services      : merchant (Maren), contracts + debt board (Broker)
header_art    : art_hdr_lanternmarket
notes         : Safe hub under the Market Truce (lore: no violence here). PvP is offline
                anyway, but the Truce is why the social/economic game centers here.
                Vast plaza, lanterns on chains into a ceiling lost in shadow.
```

### The Warrens
```
district_id   : dst_warrens
functions     : Explore shallow (-2); Explore deep (-3); combat; scavenge; anomalies
nav_links     : Lanternmarket
present_npcs  : npc_maren (loiters near the entrance)
services      : none inside; Maren buys/sells at the mouth
header_art    : art_hdr_warrens
notes         : The exploration/combat engine (CORE_LOOP §1.2). One explore action =
                one encounter from the depth-appropriate table. Depth is an integer band
                pushed by deep dives. The place "shifts" daily via the day_index seed —
                cosmetic to the player. Twisting, unmapped, wrong.
```

### The Rafters
```
district_id   : dst_rafters
functions     : Rest/heal (-1); bank items (store/retrieve); talk to NPCs
nav_links     : Lanternmarket
present_npcs  : npc_thursen (+ ambient faction-flavored residents)
services      : banking (safe storage); rest/heal
header_art    : art_hdr_rafters
notes         : Player housing = a room that holds banked items (safe on death/PvP).
                Faction NPCs are present as CHARACTERS only — no reputation mechanics in
                v1 (BUILD_SPEC). Closest thing to normal; you can hear the city Above
                through the ceiling.
```

**Network check:** District definitions and ambient pools are **Shared-Reference**.
Per-day ambient selection uses the **Local** `day_index`. Banking changes
`inventory_item.location` (**Snapshot** state). No sync implications beyond existing
structures. ✓

---

## 2.2 NPC Profiles

Profile template (backs `npc_catalog` + `dialogue_pool`):

```
npc_id           : <key>
name             : <display>
district(s)      : <where present>
role             : <function in the game>
personality      : <1-2 line summary>
services         : <data flags: buys / sells / advances-credit / heals / gives-contracts
                   / lore / tutorial>
dialogue         : pool size target (T); rotation rule; arc? (steps + triggers)
tone samples     : 2-3 representative lines (voice proof)
lore function    : what deeper systems this NPC foreshadows
```

### The Lamplighter
```
npc_id       : npc_lamplighter
district(s)  : Threshold, Lanternmarket
role         : guide / tutorial / daily hints
personality  : Cheerful, genuinely helpful, gently evasive about himself. The kind of
               kind that you later realize never answered your question.
services     : tutorial; rotating hints; (no merchant)
dialogue     : ~20-30 lines (T); rotate 1-3/day; short arc: hints escalate as the
               player's depth_record / day count grows
lore function: First to imply the Low is *organized*, not random. Deflects every
               question about his own past, how long he's been here, whether he leaves.
tone samples :
  - "Welcome down. You'll have questions. I'll have answers to some of them, and very
     convincing non-answers to the rest. It's a service I provide."
  - "Avoid the third passage on the left. Not dangerous — it's perfectly safe. It just
     isn't there on Tuesdays, and you seem the sort to be halfway through when Tuesday
     arrives."
```

### The Broker
```
npc_id       : npc_broker
district(s)  : Lanternmarket
role         : contracts, trade disputes, the debt board
personality  : Scrupulously fair. Dry. The fairness is the menace.
services      : gives/posts contracts; debt board; comments on debt_load; calls in favors
dialogue     : ~20-30 lines (T); rotate 1-3/day; debt-load-triggered comment tiers
lore function: Understands the debt economy unsettlingly well — the first hint the
               Ledger is real, total, and watching. Never wrong about a number.
tone samples :
  - "You owe twelve. I know you know. I mention it not as pressure — pressure would be
     beneath us both — but because the Ledger and I share a hobby, and the hobby is
     remembering."
  - "A fair contract. I've made it fair myself, which means it is fair to me as well.
     Do read the part you'd rather skip."
```

### Maren
```
npc_id       : npc_maren
district(s)  : Lanternmarket, Warrens entrance
role         : scavenger merchant
personality  : Tough, pragmatic, gallows humor. Has buried friends; jokes about it.
services      : buys goods; sells gear; advances gear on credit (-> debt); deep hints
dialogue     : ~20-30 lines (T); rotate 1-3/day; hints keyed to player depth_record
lore function: Knows what's deeper and what doesn't come back. Practical, not
               theoretical — the counterweight to Thursen's stories.
tone samples :
  - "Take the hook. Pay me when you've got something to pay with — and you will, or the
     Warrens will, and the Warrens are a worse creditor than me."
  - "Deep's deep. Down there the rooms remember being other rooms. Bring oil. Bring
     less attitude."
```

### Old Thursen
```
npc_id       : npc_thursen
district(s)  : Rafters
role         : primary lore delivery
personality  : Talkative, warm, unreliable. Tells the same story three ways and means
               all of them.
services      : lore; occasional "go look at X" pointer (flavor, not a tracked quest)
dialogue     : ~25-30 lines (T); rotate 1-3/day; multi-day arcs that drip Charter /
               Ledger / signatory / faction lore
lore function: The deep-lore tap. (S) Some of his stories are *wrong* — and the
               contradictions between his versions are themselves a clue for the post-v1
               knowledge system. Players who pay attention start assembling the truth.
tone samples :
  - "The founders didn't sign the Charter. That's the part they get wrong Above. The
     founders *found* it. Signed-and-found, easy to muddle, very different consequences."
  - "I've been down here long enough to forget the year I came. Or I came down because
     I'd forgotten it. One of those. Sit — I'll tell it the other way and you decide."
```

**Network check:** NPC definitions and dialogue pools are **Shared-Reference**;
per-player dialogue cursor / arc step is **Local** (`npc_dialogue_state`). ✓

---

## 2.3 Encounter Design

The Warrens engine draws one entry per explore action from an `encounter_table` keyed by
depth band (CORE_LOOP §1.2). Five entry types, each pointing at a sub-template.

**Encounter-table entry:**
```
{ type: combat | discovery | hazard | walker | anomaly,
  ref: <id of the sub-content below>,
  weight: <int>,           // relative frequency within this band
  depth_band: <int> }
```
Type frequencies per band are locked in CORE_LOOP §1.2; `weight` tunes within a type.

### Combat — creature template (`creature_catalog`)
```
creature_id      : <key>
name             : <display>
depth_band       : <int>
hp / strength / defense
behavior         : aggressive | defensive | evasive
loot_table_id    : <ref>
narration_pool_id: <intro / on-hit / on-death lines>
description      : <1-2 lines>
```
**Behavior semantics (S):**
- *aggressive* — always attacks, hits above its band, low deterrence.
- *defensive* — high defense, spends some rounds defending; a slog, but safer.
- *evasive* — harder to land hits on; may itself flee (denying loot); rewards Luck.

Samples (stats illustrative, **T**, scaled to the CORE_LOOP curve):
```
crt_gutter_eel    band 1  hp 12 str 7  def 3   evasive    loot lt_shallow_low
  "Something long unknots itself from the black water, decides you are food, and is
   only partly wrong."
crt_brick_haunt   band 3  hp 34 str 14 def 11  defensive  loot lt_mid_std
  "A draught-shape in the bricked-over doorway. It does not want to fight. It wants you
   to leave. It will, regrettably, settle for the fight."
crt_tally_shade   band 6  hp 70 str 26 def 18  aggressive loot lt_deep_rich
  "It wears the outline of a person who was balanced out mid-sentence. The numbers on
   its skin are still trying to finish the equation."
```

### Discovery — scavenge-site template
```
discovery_id : <key>
description  : <the find>
loot_table_id: <ref>
risk         : <0-100 chance it's trapped/claimed> (T)
risk_outcome : <hazard_id or creature_id if risk triggers>
```
Sample:
```
dsc_counting_house  loot lt_mid_std  risk 20  risk_outcome haz_floor_give
  "A merchant's counting house. The abacus still works, which is more than can be said
   for the merchant — dead some three hundred years, skeleton fixed in an expression of
   profound irritation at the interruption."
```

### Hazard — environmental template
```
hazard_id    : <key>
description  : <the danger>
check_stat   : luck | defense
damage_on_fail: <int or % MaxHP> (T)
gear_bypass  : <item_id that negates it, or none>
```
Sample:
```
haz_gas_pocket  check luck  dmg 25% MaxHP  bypass con_lantern_oil(no-flame variant)
  "The air down here has opinions. Most of them are about your lungs."
```

### Walker — NPC mini-encounter template *(favor economy in the field)*
```
walker_id : <key>
description: <who you meet>
options[] : { verb, outcome }   // e.g. help/rob/trade/ignore
outcomes  : favor | debt | loot | combat | nothing
```
Sample:
```
wlk_lost_arrival  options: [Help, Rob, Ignore]
  Help  -> they owe you a favor (obligation, creditor = you)
  Rob   -> loot now, but a debt/grudge flag (and Maren hears about it) (T)
  Ignore-> nothing
  "A newcomer, fresh-subtracted, still expecting the lights to come back on. You know
   the look. You wore it."
```

### Anomaly — Ledger-flavor template
```
anomaly_id    : <key>
description   : <the wrongness>
minor_effect  : none (v1 default) | small flavor effect (T)
knowledge_seed: <tag foreshadowing a post-v1 Knowledge fragment>
```
Sample:
```
anm_tallied_wall  effect none  seed charter.terms
  "Someone has scratched a column of figures into the stone. It sums. You check it
   twice because you don't want it to, and it does."
```

### Scaling with depth
`encounter_table` is keyed by `depth_band`; deeper bands reference tougher creatures,
richer `loot_table`s, nastier hazards, and stranger anomalies. Band→stat/loot scaling
tracks the CORE_LOOP level curve (T). **Variety targets** (BUILD_SPEC): 15-20 creatures,
10-15 non-combat encounters, 8-10 hazards, 5-8 anomalies — enough that a daily player
sees no obvious repeats for two weeks.

**Network check:** All encounter sub-content is **Shared-Reference**. A resolved
encounter is **Local** (`warren_session`); only a death (`player.died`) or a created
obligation (`debt.created`, from a Walker) emits an event. ✓

---

## 2.4 Item and Gear Catalog

Item template (backs `item_catalog`, DATA_MODEL.md §2.1):
```
item_id        : <key>
name           : <display>
category       : weapon | armor | consumable | trade | curio
tier           : 1-10   (gates by level / depth)
stat_mods      : { attack | defense | maxhp | luck | heal | ... }
trade_weight   : <abstract barter value>
degrades       : true | false
max_condition  : <int, if degrades>
carry_cost     : <slots; usually 1>
rarity         : common | uncommon | rare | curio   (-> loot_table weighting)
description_ref: <flavor>
```

**Stat model (S):**
- **Weapon** → `attack` adds to `attack_power` (CORE_LOOP combat formula).
- **Armor** → `defense` adds to `defense_power`.
- **Consumable** → one-shot effect (heal, temp buff, flee/hazard aid).
- **Trade good** → no stats; pure `trade_weight` — the scavenge fuel that drives the
  economy and the debt loop.
- **Curio** → small passive (`luck`/`maxhp`) and/or a lore object; rare.

**Rarity → loot distribution (T):** common ~60% / uncommon ~28% / rare ~10% / curio ~2%,
biased by Luck. Tiers gate availability (a tier-6 weapon won't drop in band-1).

**Degradation (S):** weapons/armor degrade with use (`condition` per instance,
DATA_MODEL.md §1.4). At `condition 0` the item is *broken* — its bonus drops to a
fraction until **repaired** at a merchant (costs goods, or a debt). Consumables and
trade goods don't degrade.

**Storage (CORE_LOOP):** carried capacity **10 slots** (T), at risk on death/PvP;
banked storage large/effectively unlimited, safe.

**Starting gear (S, T):** `wpn_rusted_shiv` (tier1) + `arm_scavengers_wrap` (tier1) +
2× `con_poultice`.

Samples (a small tier ladder to prove the template):
```
wpn_rusted_shiv      weapon t1  attack +3   weight 4   degrades  rarity common
  "Rust, mostly, in the shape of a knife. The shape is the important part."
wpn_dock_hook        weapon t3  attack +9   weight 14  degrades  rarity uncommon
wpn_ledger_stylus    weapon t6  attack +18  weight 40  degrades  rarity rare
  "A clerk's stylus, sharpened past any clerical purpose. It writes one thing now."
arm_scavengers_wrap  armor  t1  defense +1  weight 3   degrades  rarity common
arm_vault_plate      armor  t5  defense +14 weight 35  degrades  rarity rare
con_poultice         consum t1  heal 30% MaxHP   weight 3   rarity common
con_lantern_oil      consum t2  hazard/flee aid (light)  weight 5  rarity uncommon
trd_brass_fittings   trade  t-  weight 8   degrades:false  rarity common
cur_lucky_tooth      curio  t-  luck +1 (passive)  weight 12  rarity curio
  "Whose tooth, and why lucky, are questions for someone with more teeth to spare."
```
**Catalog targets (BUILD_SPEC):** 6-8 weapons, 6-8 armor, 5-8 consumables, 20-25
scavengeable items, plus a handful of curios.

**Network check:** The catalog is **Shared-Reference**. A player's owned items are
**Snapshot** (`inventory_item`); condition/location changes ride the character snapshot,
not the broadcast spine. ✓

---

## 2.5 Contract System

Contract template (backs `contract_template`, DATA_MODEL.md §2.6):
```
template_id   : <key>
type          : fetch | deliver | survive | explore
objective_spec: <parameterized goal>   // params filled per daily instance
reward_spec   : { goods | gear | debt_relief | obligation | lore }
duration_days : <int>
flavor_ref    : <poster's framing>
```

**Types (v1) (S):**
- **fetch** — recover item X from the Warrens (param: item, min depth band).
- **deliver** — take item X to an NPC/location (param: item, destination).
- **survive** — complete N explores at depth ≥ band without dying (param: N, band).
- **explore** — reach a depth band / surface a specific anomaly (param: band/anomaly).

**Lifecycle (CORE_LOOP economy):**
- **Accept/post = 0 actions** (free). Completion costs the normal action costs of the
  activities involved (e.g., the explores you do for it).
- **2-3 new contracts/day** (BUILD_SPEC), posted by the Broker (general) and Maren
  (scavenging-flavored).
- **Verification** is event-driven against the `contract_instance.params`: a fetch
  completes when the target item enters inventory; survive/explore complete on the
  qualifying explore action; deliver on hand-off.
- **Reward** may include **debt_relief** (clears/reduces an obligation) — a clean tie
  back to the favor/debt loop — or grant an **obligation** (a favor the NPC now owes
  you, or a fronted-materials debt you owe).
- **Expiry / failure** by `expires_day`: a minor consequence — a Broker note, and if
  materials were fronted, the advance becomes a standing **debt**. No hard punishment
  (v1 keeps pressure economic, not punitive).

Samples:
```
ct_maren_hooks   type fetch   obj: bring 1x trd_brass_fittings from band>=2
  reward: gear (wpn_dock_hook) OR debt_relief 10   duration 3d
  flavor (Maren): "Fittings off the old pump-house. Brass, not the soft stuff. Bring
                   them and the hook's yours — or we call it even on what you owe."
ct_broker_census type survive obj: 3 explores at band>=3, no death
  reward: debt_relief 15 + lore(anm_tallied_wall hint)   duration 4d
  flavor (Broker): "A small accounting. Go down three times, come back three times. The
                    Ledger appreciates symmetry, and so, within reason, do I."
ct_broker_door   type explore obj: surface anomaly anm_tallied_wall
  reward: goods + lore(charter.terms seed)   duration 5d
```
**Catalog target (BUILD_SPEC):** 10-15 contract templates.

**Network check (S — federation seam, deferred):** Same-node contracts are **Local**
(`contract_instance`). A reward that creates/clears an obligation goes through the
**Broadcast** obligation system (`debt.*`). Cross-node contracts (B3) become Broadcast
with global IDs and offer/accept events — the template already supports it; v1 does not
build it. ✓

---

## Exit-criteria check (GAME_DESIGN_PROCESS.md Stage 2)

- [x] **2.1** Every v1 district has functions, navigation, on-screen presentation,
      NPCs/services, and ambient structure — on one shared 80×24 screen template.
- [x] **2.2** NPC profile template defined and filled for all 4 v1 NPCs, with services,
      dialogue/rotation/arc model, voice samples, and lore function.
- [x] **2.3** Encounter-table entry format + templates for all five encounter types
      (creature, discovery, hazard, walker, anomaly), depth scaling, variety targets.
- [x] **2.4** Item template, stat model, rarity/degradation/storage rules, starting
      gear, and a tiered sample ladder.
- [x] **2.5** Contract template, v1 types, lifecycle, verification, reward/expiry rules,
      samples.
- [x] **An encounter, an NPC, an item, and a contract can each be fully described by
      filling in a template** → content production (Stage 6) can begin.
- [x] Each section passes its network check and maps to a DATA_MODEL.md structure.

---

## Notes for downstream stages

- **Voice:** samples here point the register (Gaiman-meets-Adams, world plays it
  straight). The formal VOICE_GUIDE.md (Stage 6) will lock tone rules, the word list,
  and per-NPC voice; treat these samples as direction, not canon.
- **Numbers:** all creature/item/reward values are **(T)** and will be set against the
  CORE_LOOP curve during the BUILD_SPEC revision (Stage 5) and B2 playtest.
- **Lore seeds:** Walker grudges, anomaly `knowledge_seed` tags, and Thursen's
  contradictions are deliberate hooks for the post-v1 faction/knowledge systems — laid
  now so the later phases have soil to grow in.

---

*Document version 0.1 — Stage 2 Content Architecture. Templates locked; samples are
voice-direction and tunable. Feeds Stage 5 (BUILD_SPEC revision) and Stage 6
(CONTENT_PLAN / VOICE_GUIDE).*
