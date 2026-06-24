# Phase 5 Integration Prompt

## Task

Integrate a game, SDK change, or protocol feature across node, hub, and tests.

## Required Context

Read:

- `02_Architecture.md`
- `03_Hub_Specification.md`
- `04_Federation_Protocol.md`
- `05_Game_SDK.md`
- `09_Testing.md`

## Implementation Rules

- Make additive protocol changes where possible.
- Keep hub projections derived from accepted events or explicit queue operations.
- Keep node gameplay available during hub failures.
- Add tests at the lowest level that proves the behavior.
- Add E2E coverage for cross-node behavior when feasible.

## Verification

- Run relevant package tests during development.
- Run `go test ./...` from `games/ledgerofthelow` before finishing.
- Recheck docs for drift if behavior changes.
