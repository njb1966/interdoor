# VOICE GUIDE — Stage 6

> Status: Historical game-specific source material. The root numbered InterDoor docs are canonical for Phase 1 framework, protocol, operator, and architecture decisions. Do not use this file as Phase 1 framework authority unless it has been explicitly reconciled.

## How the Low Sounds

*Stage 6 deliverable (per GAME_DESIGN_PROCESS.md §6.1). Status: v0.1.*

---

## What this document is

The writing-style reference for **all** game content: exploration text, NPC dialogue,
combat narration, item and encounter descriptions, help text. If it's words the player
reads, this document governs it. A writer (human or AI-assisted) should be able to pick
up any content task, read the relevant section here, and produce on-tone copy without
further guidance — that's the Stage 6 exit bar.

**Note on names:** the *game's title* is undecided and never appears in content yet. But
the **setting vocabulary is canon** and in active use: *Dornhaven* (the city), *the Low*,
*the Ledger*, *the Charter*, and the district names are established in DESIGN.md and are
correct to write. See the Word List below for what's in and what's out.

---

## 1. The register

**Gaiman meets Adams.** Two writers, two contributions, held in tension:

- **From Gaiman:** the conviction that there is an old, layered, rule-bound world hidden
  under the ordinary one, and that it runs on bargains nobody fully remembers making.
  Wonder and dread share a room. The Low is *real*, *old*, and takes itself seriously.
- **From Adams:** deadpan. Cosmic and bureaucratic horrors described in the same flat,
  precise, slightly weary voice you'd use for a parking ticket. The comedy is in the
  understatement and the misapplied procedure, never in a joke being *told*.

The collision of the two **is** the voice: an impossible world, narrated like
paperwork.

> **The one-line test:** *Does it describe something strange or dangerous in a calm,
> specific, slightly dry voice that takes the world seriously?* If yes, it's on-tone.

---

## 2. The five rules

1. **The world plays it straight.** The Low is never in on the joke. Characters are not
   quirky-for-the-camera; they are people who have adapted to impossible circumstances
   and now find them ordinary. Their *normalcy about the abnormal* is the effect.

2. **Humor comes from juxtaposition, never from jokes.** Funny = the mundane and the
   impossible given equal weight (an abacus that still works; a skeleton that's
   *irritated*; a debt the Ledger "remembers" as a hobby). Not funny = puns as the
   point, zingers, anything that exists only to be a joke.

3. **Specificity over atmosphere words.** Don't say "eerie," "eldritch," "ancient,"
   "mysterious." Show one exact, slightly-wrong detail and let the reader feel it. *"The
   numbers on its skin are still trying to finish the equation"* beats any adjective.

4. **The impossible is logistics.** Treat supernatural problems as practical ones —
   things with rules, costs, schedules, and workarounds. A passage that isn't there on
   Tuesdays is a *scheduling issue*. This is the spine of the comedy and the horror at
   once.

5. **Never wink.** No fourth-wall breaks, no "as you well know, player," no nudging the
   reader to notice how clever the line is. The voice does not perform. It observes.

---

## 3. Positive vs. negative examples

| ✗ Off-tone | ✓ On-tone |
|---|---|
| "You enter a spooky, ancient counting house!" | "The tunnel opens into what was once a merchant's counting house. The abacus on the desk still works, which is more than can be said for the merchant." |
| "A terrifying monster leaps out! Prepare for battle!" | "Something long unknots itself from the black water, decides you are food, and is only partly wrong." |
| "You leveled up! +5 STR! Awesome!" | "Something in you has settled. The dark holds fewer surprises than it did. Not none. Fewer." |
| "The mysterious Ledger watches you ominously." | "Your debts shifted by one overnight. You didn't move them. Someone did the arithmetic." |
| "Haha, classic Tuesday problem, am I right?" | "It isn't there on Tuesdays, and you seem the sort to be halfway through when Tuesday arrives." |

The off-tone column fails on: hype punctuation, labelled emotion, gamer vocabulary in
prose, telling instead of showing, and winking.

---

## 4. The narrator (exploration & system text)

**Who is speaking:** an unnamed observer who has watched a great many people wake up in
the Low. Second person, present tense. Dry, unhurried, faintly fond of you and entirely
unwilling to lie to make you feel better. It knows more about the Low than it tells —
and it never tells more than the moment needs.

- **Person/tense:** "You" / present. *"You step down. The water is colder than it has
  any business being."*
- **Stance:** calm under all circumstances. It does not gasp, warn in capitals, or
  editorialize. Danger is reported, not sold.
- **Knowledge:** omniscient about texture, coy about meaning. It will tell you the wall
  sums correctly; it will not tell you why that matters.
- **Restraint:** short sentences earn their length. One precise wrong detail per beat is
  plenty.

Sample exploration beats:
- "A door. Newer than the wall around it, which is the wrong way for doors to be."
- "The passage you came in by is still there. You check. It's good to check."
- "Three hundred years of dust, and one set of footprints leaving. None arriving."

---

## 5. Combat narration

Fast, physical, a little swingy — matches the round-by-round engine (CORE_LOOP §1.2).
Keep it to one or two clauses per round so fights stay quick (10–15 min sessions). The
status line carries the numbers; **the prose never says "HP," "damage," or "crit."**

- **Player hit:** "The shiv goes in where the shiv goes. The eel reconsiders its plans."
- **Player miss / creature defends:** "It folds around the blow like it expected it.
  Maybe it did."
- **Creature hits player:** "Something cold closes on your arm. You'll feel that
  tomorrow, assuming the usual number of tomorrows."
- **Crit (no jargon):** "You find the seam. Whatever holds it together stops."
- **Flee success:** "You leave. Dignity is for the surface."
- **Death blow:** "The dark does the rest. It's good at the rest."

Each creature has a small narration pool (`narration_pool_id`) so repeat fights vary.

---

## 6. NPC voices (5–10 lines each)

Expanding the anchors in CONTENT_ARCHITECTURE.md §2.2. These lines establish the voice;
the full rotating pools are produced in CONTENT_PLAN.md.

### The Lamplighter — *cheerful, helpful, evasive about himself*
Warmth that never quite answers the question. Helpful in the specifics, fog in the
personal.
- "Welcome down. You'll have questions. I'll have answers to some of them, and very convincing non-answers to the rest."
- "First rule: the Low is fair. It is not *kind*. People confuse the two on the way down and it costs them."
- "Don't thank me. Thanks is a small favor, and you don't yet know what favors cost here. Learn that first, then thank me."
- "How long have *I* been here? Long enough to stop asking that question. You'll get there too. Lovely down here once you do."
- "Take the lantern. No, it's not a gift — nothing is. Consider it a loan against your good sense, repayable in surviving the week."
- "The market's that way. Mind the Truce. It's the one promise down here that everyone keeps, which should tell you how much it cost to make."

### The Broker — *scrupulously fair; the fairness is the menace*
Never raises a voice, never rounds a number, never threatens — because the facts do it
for him.
- "A fair contract. I've made it fair to *you* as well, which I mention because people forget that's optional."
- "You owe twelve. I say so not as pressure — pressure would be beneath us both — but because the Ledger and I share a hobby, and the hobby is remembering."
- "I don't bend the terms. If I bent them for you, you'd never again trust that I hadn't bent them for someone else. My rigidity is the kindest thing about me."
- "Pay when you can. 'Can' is doing a great deal of work in that sentence, and we both know it."
- "I have never been wrong about a number. It's not a boast. It's the job. The day I'm wrong, you'll have larger problems than the number."
- "Default if you like. I won't stop you. I'll simply write it down, and down here, written down is a kind of weather."

### Maren — *tough, pragmatic, gallows humor*
Has buried friends and jokes about it because the alternative is worse. Practical to the
bone.
- "Take the hook. Pay me when you've got something to pay with — and you will, or the Warrens will, and they're a worse creditor than me."
- "Deep's deep. Down there the rooms remember being other rooms. Bring oil. Bring less attitude."
- "That armor? Came off someone who didn't need it anymore. Don't make a face. Everything down here came off someone."
- "You want my advice or my prices? They're different services and only one of them's free."
- "Rule of the dark: if it's easy to reach, someone left it for a reason. Usually a bad one."
- "Come back alive and we'll call the debt small. Come back rich and we'll call it smaller. Don't come back and, well — I'll have your hook back, won't I."

### Old Thursen — *warm, talkative, unreliable; contradicts himself on purpose*
The lore tap. His versions don't match, and the gaps are the point.
- "The founders didn't *sign* the Charter. That's the part they get wrong Above. They *found* it. Signed-and-found, easy to muddle, very different consequences."
- "I've been down here long enough to forget the year I came. Or I came down because I'd forgotten it. One of those."
- "There were five signatories. Or three. I've told it both ways and meant both. The truth's somewhere I can't reach without a lamp I don't have."
- "The Ledger doesn't punish, lad. Punishing's personal, and it isn't a person. It *adjusts*. Like rain adjusts a coastline. Nothing personal in it, and you drown all the same."
- "Sit. I'll tell you the one about the Ferryman who rowed past the end of the water. It's not true. Most of the true things aren't, down here."
- "You'll want to write down what I say. Then write down where it disagrees with itself. That second list — that's the one worth keeping."

---

## 7. How humor works (and where it stops)

**Funny here:**
- Bureaucracy applied to the impossible (debts as weather, balancing as arithmetic).
- Deadpan understatement of danger ("only partly wrong").
- Specificity that implies a whole sad story in six words (the irritated skeleton).
- Characters being *reasonable* about unreasonable things.

**Not funny here — off-limits:**
- Puns or wordplay as the payload of a line.
- Fourth-wall breaks, player-address, "achievement unlocked" energy.
- Pop-culture or anachronistic references (no memes, no modern brands).
- Cruelty played for a laugh; jokes that deflate a genuine scare.
- Emoji, exclamation-point hype, ALL-CAPS excitement.
- Whimsy for its own sake. The whimsy must cost something or reveal something.

Rule of thumb: **if a line would survive being read aloud, slowly, by a tired clerk, it's
on-tone. If it needs a rimshot, cut it.**

---

## 8. Word list

### In-vocabulary (canon — use freely)
- **Places:** Dornhaven, the Low, Above, the Threshold, the Lanternmarket, the Rafters,
  the Warrens. *(Post-v1: the Gutters, the Vaults, the Archive, the Deep.)*
- **Systems-as-fiction:** the Ledger, the Charter, the Signatories, the Market Truce,
  the debt board. **balanced out** / **subtracted** (how people arrive in the Low),
  **favor**, **debt**, **owe**, **call in**.
- **People:** **Walkers** (those who explore the Warrens), newcomers, residents, the
  factions by name (Ledgermen, Remnants, Tally, Ferrymen — as lore in v1).
- **Texture words that carry the world:** lantern, brick, vault, cellar, counting house,
  abacus, tally, ledger, current, draught, seam, arithmetic, terms, accounting.

### Out-of-vocabulary (do not use in player-facing prose)
- **Gamer jargon in narration:** XP, level-up, loot, buff/debuff, HP, damage, crit,
  spawn, grind, DPS. *(These may appear only in the status line / UI labels, never in
  prose. "Respawn" → "wake.")*
- **Fantasy boilerplate:** adventurer, hero, quest (→ *contract* / *errand*), dungeon,
  monster (→ something specific, or *thing*), magic, spell, enchanted, dragon, "ye
  olde."
- **Labelled atmosphere:** eerie, eldritch, Lovecraftian, ancient, mysterious,
  ominous, blood-curdling, sinister. *Show the detail instead.*
- **Hype & wink:** epic, legendary (as hype), awesome, "am I right," any emoji, any
  fourth-wall address.

---

## 9. Quick checklist (tape to the monitor)

Before a line ships:
- [ ] Does the world take itself seriously? (No winking.)
- [ ] Is the humor from juxtaposition, not a joke being told?
- [ ] One *specific* wrong detail, not an atmosphere adjective?
- [ ] Is the impossible treated as logistics?
- [ ] No gamer jargon in the prose?
- [ ] Would it survive a tired clerk reading it aloud?

---

*Document version 0.1 — Stage 6 Voice Guide. The register is locked; the line samples
are canon-quality reference. Per-NPC pools and all v1 copy are produced per
CONTENT_PLAN.md against this guide.*
