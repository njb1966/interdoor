# Phase 0 Project Setup Prompt

Use this prompt to orient a fresh agent before implementation work.

## Task

Prepare the InterDoor repository for framework-focused work without rewriting existing games or legacy docs.

## Instructions

- Read the root numbered docs first, especially `00_Project_Charter.md`, `01_Project_Rules.md`, and `07_Backlog.md`.
- Treat InterDoor as the framework. Ledger of the Low is the Phase 1 reference node; other games are future clients only after explicit design approval.
- Inspect current code behavior before changing documentation or implementation.
- Preserve `docs/`, `games/ledgerofthelow/`, and `games/empireascendant/` unless the task explicitly targets them. Do not treat `games/empireascendant/` as Phase 1 implementation authority.
- Do not reorganize the Go module as part of setup.

## Expected Output

- A concise repo status summary.
- Any setup blockers.
- A minimal plan for the requested phase.
