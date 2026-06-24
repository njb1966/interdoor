# Phase 1 Architecture Prompt

## Task

Refine or implement InterDoor framework architecture.

## Required Context

Read:

- `00_Project_Charter.md`
- `01_Project_Rules.md`
- `02_Architecture.md`
- `03_Hub_Specification.md`
- `08_Decisions.md`

Then inspect the relevant Go packages under `games/ledgerofthelow/internal/`.

## Constraints

- Keep hub, engine, federation client, and game modules separate.
- Do not make speculative repository moves.
- Do not introduce Docker, orchestration, or heavy framework dependencies.
- Prefer store-interface-compatible changes.
- Record new architectural decisions in `08_Decisions.md`.

## Verification

Run `go test ./...` from `games/ledgerofthelow` after code changes.
