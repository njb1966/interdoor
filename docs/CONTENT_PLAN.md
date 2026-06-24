# CONTENT PLAN — Stage 6

> Status: Historical game-specific source material. The root numbered InterDoor docs are canonical for Phase 1 framework, protocol, operator, and architecture decisions. Do not use this file as Phase 1 framework authority unless it has been explicitly reconciled.

## What to Write, In What Order, and How

*Stage 6 deliverable (per GAME_DESIGN_PROCESS.md §6.2). Status: v0.1.*

---

## What this document is

The production schedule for all v1 content. It takes the inventory locked in
BUILD_SPEC_V1.md §5.2 (which fills the templates in CONTENT_ARCHITECTURE.md, in the voice
of VOICE_GUIDE.md) and answers four questions: **what** to produce, **in what order**,
**how long** each takes, and **who drafts it** (human vs. AI-assisted). The goal is that
any content task can be pulled off this plan and executed without further planning.

**Sequencing principle:** content is produced in two passes, aligned to the engineering
phases in PROJECT_PLAN.md. Pass 1 (the **Vertical Slice**) produces the *minimum* content
to make the core loop testable in Phase B1. Pass 2 (the **Full v1 Volume**) fills out to
the BUILD_SPEC targets during Phase B2 polish. We do **not** write all the content before
the game runs — that's the burnout risk the project plan warns about. Write enough to
play, get it playing, then deepen.

---

## 1. The two production passes

| | **Pass 1 — Vertical Slice** | **Pass 2 — Full v1 Volume** |
|---|---|---|
| Aligns to | Phase B1 (engine + playable localhost) | Phase B2 (single-node polish) |
| Goal | Core loop is testable end-to-end | No obvious repetition over 2 weeks of daily play |
| Volume | ~25–35% of targets | 100% of BUILD_SPEC targets |
| Quality bar | On-tone, rough edges OK | Edited, voice-consistent, final |

The slice is deliberately thin but **complete in shape**: at least one of every content
*type*, so every system can be exercised. Breadth before depth.

---

## 2. Asset inventory, passes, and effort

Counts: **Slice → Full** (Full = BUILD_SPEC §5.2 target). Effort is per-asset, *drafted
and voice-edited*, and assumes AI-assisted drafting where noted (§3). Ranges are
estimates for one writer; they exist for sequencing, not contracts.

| Asset type | Slice → Full | Draft source | Effort / asset | Notes |
|---|---|---|---|---|
| District headers (ANSI) | 4 → 4 | Hand (§4) | 2–4 h | All 4 needed in the slice — every district is visited early. |
| Title / Death / Daily-summary (ANSI) | 1 → 3 | Hand (§4) | 2–4 h | Title for slice; death + summary in Pass 2. |
| Combat frames / help borders (ANSI) | 1 → 3–4 | Generated + hand-finish (§4) | 0.5–1 h | Reusable; not per-creature. |
| PvP result screen (ANSI) | 0 → 1 | Generated + hand-finish | 0.5–1 h | Pass 2 (PvP polish). |
| NPC dialogue lines | 6×4 → 25×4 | AI draft + human voice-edit | 3–8 min/line | The 4 anchor lines/NPC in VOICE_GUIDE ship as-is. |
| NPC arcs (multi-day) | 0 → 1–2 short arcs | **Human** | 1–2 h/arc | Tone-critical; Thursen's contradictions especially. |
| Creatures | 5 → 15–20 | AI draft + human edit | 20–30 min | Stats (T) + name + description + small narration pool. |
| Discovery / Walker encounters | 3 → 10–15 | AI draft + human edit | 15–25 min | Walkers carry the favor economy — edit those by hand. |
| Hazards | 2 → 8–10 | AI draft + human edit | 10–20 min | |
| Anomalies | 2 → 5–8 | **Human** | 20–30 min | Lore seeds; must be precise. Small count, high care. |
| Weapons / Armor | 2+1 → 6–8 / 6–8 | AI draft + human edit | 10–15 min | Stats (T); one-line descriptions. |
| Consumables | 2 → 5–8 | AI draft + human edit | 10–15 min | |
| Scavengeable / trade / curio items | 5 → 20–25 | AI draft + human edit | 5–10 min | Bulk; light touch. |
| Contract templates | 2 → 10–15 | AI draft + human edit | 15–25 min | Objective + reward + flavor. |
| Ambient district text | 2×4 → 5–8×4 | AI draft + human edit | 5–10 min/line | Rotating per `day_index`. |
| Tutorial / onboarding (Threshold) | 1 pass → final | **Human** | 4–8 h total | First impression; written and edited by hand. |
| Help text | 1 page → 3–5 pages | **Human** | 3–6 h total | In-voice, not a manual (VOICE_GUIDE §4). |

**Rough totals (sequencing only):** Pass 1 slice ≈ a few focused days of content work
running parallel to B1 engineering. Pass 2 ≈ spread across the B2 polish window,
interleaved with playtest feedback.

---

## 3. AI-assisted vs. human-written

The split follows one rule: **AI drafts the volume; humans own the voice anchors and the
lore.** Everything AI-drafted gets a human voice-edit pass against VOICE_GUIDE §9 — AI is
the first draft, never the last.

**Fully human-written (no AI first draft):**
- The 4 NPCs' **anchor** lines and their multi-day **arcs** (the voice lives or dies here).
- **Tutorial / onboarding** and **help** text (first impression; must be flawless tone).
- **Anomalies** (lore seeds for post-v1 — wrong wording here misleads future design).
- The **title** and **death** screen copy (the lines players see most, at their most
  emotionally loaded moments).

**AI-assisted drafting + mandatory human voice-edit:**
- Bulk creatures, items, hazards, discoveries, ambient lines, contract flavor, and the
  *rotating* (non-anchor) NPC lines.
- Process: prompt with VOICE_GUIDE §1–8 + the template + 2–3 on-tone exemplars → draft
  → human edit for specificity, jargon, and winking → accept.

**AI failure modes to watch (and edit out):** atmosphere adjectives instead of specific
details; gamer jargon leaking into prose; lines that wink; whimsy that costs nothing;
sameness across a batch (every creature "unknots itself"). The human pass exists
primarily to kill these.

---

## 4. ANSI art: hand vs. generated

All art is **80×24, base-16 ANSI** (NETWORK_REQUIREMENTS.md Req 8); it must read on a
monochrome terminal (color is atmosphere, never information). Target 10–15 assets
(BUILD_SPEC).

**Hand-crafted (or hand-finished):**
- **Title screen** — the single most important asset; the front door. Hand-made.
- **The 4 district headers** — identity of each space; hand-made or heavily hand-finished.
- **Death screen** — high emotional weight; hand-made.

**Generated/templated, then hand-finished:**
- Combat frames (2–3 reusable), help borders/headers, PvP result frame, daily-summary
  frame. These are structural/repeating and tolerate a templated base with a cleanup
  pass.

**Tooling note:** hand work in a standard ANSI editor (e.g., a PabloDraw-class tool);
generated bases cleaned to the 16-color, 80×24 constraint. A public ANSI-art
contribution toolkit is a *Phase 5 / community* item (PROJECT_PLAN), not v1.

---

## 5. Priority order (the production queue)

Sequenced so content unblocks the B1 build order (PROJECT_PLAN §B1) — each item is ready
when the engineering that needs it comes online.

**Pass 1 — Vertical Slice (parallel to B1):**
1. **Title + 4 district headers** (ANSI) — needed the moment screens render.
2. **Lamplighter tutorial + onboarding** (human) — the first thing a tester experiences.
3. **5 creatures + 1 combat frame + narration pool** — unblocks the combat loop.
4. **Starting gear + ~8 items** (2 weapons, 1 armor, 2 consumables, ~3 trade goods) —
   unblocks inventory/economy.
5. **6 lines × 4 NPCs** (anchors first) + **2 ambient lines × 4 districts** — unblocks
   NPC/hub feel.
6. **2 contract templates** — unblocks the contract loop.
7. **2 hazards + 2 discoveries/walkers + 2 anomalies** — rounds out encounter variety.
8. **1 help page** — minimum playability.

→ At the end of Pass 1, a tester can complete a full daily session and every system has
content to exercise.

**Pass 2 — Full v1 Volume (during B2, feedback-driven):**
9. Fill creatures → 15–20, encounters → targets, items → targets, contracts → 10–15.
10. NPC pools → 25–30 lines each + 1–2 short arcs; ambient → 5–8 per district.
11. Death + daily-summary + PvP ANSI screens; remaining combat/help frames.
12. Help → 3–5 pages; tutorial polish from onboarding feedback.

→ At the end of Pass 2, the game sustains two weeks of daily play without obvious
repetition (BUILD_SPEC success criterion).

---

## 6. Definition of done (per asset)

A content asset is done when:
- It fills its CONTENT_ARCHITECTURE template completely (all required fields).
- It passes the VOICE_GUIDE §9 checklist.
- For mechanical content (creatures, items, contracts), its numbers are present (marked
  **(T)** if provisional) and reference the CORE_LOOP curve.
- It has had a human voice-edit pass (even if AI-drafted).
- For lore-bearing content (anomalies, Thursen arcs), its `knowledge_seed` / arc tag is
  set, so post-v1 systems can find it.

---

## 7. What this plan deliberately excludes

- **Post-v1 content** (factions, knowledge fragments, the Archive/Deep, endgame copy) —
  not written until those systems are designed (Stage 3) and scheduled.
- **A community content toolkit / IGM-style extension content** — Phase 5.
- **Localization** — English only for v1.

---

## Exit-criteria check (GAME_DESIGN_PROCESS.md Stage 6)

- [x] **6.1 VOICE_GUIDE.md** — tone, narrator, per-NPC voices, combat style, humor
      rules, word list. *(Sibling document.)*
- [x] **6.2 CONTENT_PLAN.md** — full v1 asset inventory, priority/queue order, per-type
      effort estimates, AI-vs-human split, ANSI hand-vs-generated split.
- [x] A writer can pull any task from §5, reference the voice guide and the template,
      and produce on-tone content without further guidance.

---

*Document version 0.1 — Stage 6 Content Plan. Two-pass production (slice in B1, full
volume in B2). Effort estimates are for sequencing, not commitments. Revise as B1
reveals real per-asset costs.*
