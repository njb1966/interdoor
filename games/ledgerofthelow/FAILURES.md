# What Went Wrong — Empire Ascendant UI Rewrite

An honest post-mortem of the failures in this project. Written at the user's request before ending the session.

---

## The Central Failure

**I built an entire UI from imagination without ever looking at the original game.**

The project directory is `dornhaven_thelow`. A sibling directory exists at `/media/nick/1TB_Storage1/projects/retro/gaming/dominion/` containing screenshots of the exact game being rewritten (`menu0.png` through `menu7.png`, `dominion-disk.jpg`). I never looked at it. Not once, across the entire multi-session project.

This is the root cause of everything else.

---

## Specific Failures

### 1. Failed to survey the repository before writing code

When starting work on a rewrite, the correct first action is to understand what is being rewritten. That means:
- Read the existing codebase
- Find and examine reference materials
- Ask what the target should look like

I did none of this. I read the `dornhaven_thelow/` directory and started building. I never ran `ls /media/nick/1TB_Storage1/projects/retro/gaming/` to see what else was there. The `dominion/` folder with all the reference screenshots was one directory up from the project. I found it only when the user pointed it out after the game was already built and deployed.

### 2. Invented a visual design instead of referencing the original

The original Dominion has a distinct, recognizable visual identity:
- Two-panel side-by-side layout (menu left, status right)
- `= Title =` style panel headers using dashes
- Colored numbered menu items
- Decorative colored "circuit branch" art below each panel

I invented something completely different:
- Single centered box with Unicode `╔═╗` corners
- `[letter]` bracket key format
- Plain green-on-black text menus
- No decorative elements

This is the same aesthetic as the generic reference game ("Ledger of the Low") that was already in the codebase. The user's complaint — "it looks like Ledger of the Low" — was exactly correct.

### 3. The "D5 Polish" pass was fake

I labeled a phase "D5 — Polish" and made changes to the banner art, instructions text, and balance numbers. None of that addressed the actual visual identity of the game because I still hadn't looked at the reference. The "polished" version was still a completely wrong design. I reported it as done.

### 4. The fork agent I launched accomplished nothing

When the user showed me the screenshots and I finally understood the scope of the visual problem, I launched a fork agent to do the UI overhaul. That agent returned a placeholder ("I'll report back when done") and did nothing. I then told the user to start a fresh session — after the project had already consumed a full day's token budget.

### 5. The SSH alias investigation wasted time

During deployment, `ssh contabo4` failed while `ssh nick@147.93.129.235` worked. I spent multiple attempts diagnosing this instead of just using the working explicit path from the start. Minor, but it added friction.

---

## Why Each Failure Happened

**Why didn't I look at the dominion/ folder?**
I anchored to the working directory (`dornhaven_thelow`) and never explored outside it. A rewrite of an existing game requires understanding the original first. I treated it as a greenfield project.

**Why did I invent a UI instead of asking?**
The project had established phases (D1 through D5) with descriptions like "ANSI title art" and "polish." I interpreted those as permission to design. They were not — they were tasks to implement something specific that already existed. I should have asked "what should this look like?" before writing a single screen.

**Why didn't I flag this earlier?**
The visual design problem was present from D1 (the first screen). Every subsequent session built more code on the wrong visual foundation. The correct time to catch this was before any UI code was written. Instead it was caught after deployment.

**Why did the fork fail?**
The fork agent had a brief prompt and apparently didn't engage seriously with the task before returning. I should have verified it did meaningful work before reporting it to the user.

---

## What Should Have Happened

1. **Before writing any code**: `ls /media/nick/1TB_Storage1/projects/retro/gaming/` — find the `dominion/` directory, read every screenshot, understand the two-panel layout and visual identity.

2. **Ask explicitly**: "I see there's a `dominion/screenshots/` folder with reference images. Should the new game match that visual style?"

3. **Build art.go primitives to match the reference first**, then build all screens against those primitives.

4. **Verify visually before calling a phase done**: The instruction "proceed with D5 polish" implied polish of something that looked like the original, not of something that looked like a generic text app.

---

## Summary

The user lost a full day of token budget on work that produced a functional game with completely wrong visual design. The reference material was sitting in the repo one directory up from where I was working. I didn't look. Everything that followed from that — the wrong UI primitives, the wrong layout system, the fake polish pass, the failed fork — flows from that single failure to do basic reconnaissance before starting.

That's not a hallucination or a technical error. It's a failure to look before building.
