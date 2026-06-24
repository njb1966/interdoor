# Phase 3 Game Development Prompt

## Task

Develop or adapt a game for InterDoor without confusing game-specific work with framework work.

## Required Context

Read:

- `00_Project_Charter.md`
- `01_Project_Rules.md`
- `05_Game_SDK.md`
- `06_Code_Modernization.md`

Then inspect the target game directory.

## Rules

- Preserve the game's intended style and cadence.
- Keep framework primitives in the engine or SDK, not buried in one game.
- Use global IDs for federated player references.
- Classify state before syncing it.
- Keep terminal compatibility in mind.

## Integration Target

Choose the smallest useful tier:

- Tier 1: listed.
- Tier 2: events and roster.
- Tier 3: travel, PvP, and obligations.

Do not jump to Tier 3 without a clear compatibility contract.
