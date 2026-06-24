# Phase 2 Protocol Prompt

## Task

Work on the InterDoor federation protocol, event contracts, hub endpoints, or client sync behavior.

## Required Context

Read:

- `03_Hub_Specification.md`
- `04_Federation_Protocol.md`
- `07_Backlog.md`
- `09_Testing.md`

Inspect:

- `games/ledgerofthelow/internal/hub/`
- `games/ledgerofthelow/internal/fed/`
- `games/ledgerofthelow/internal/engine/eventlog.go`
- `games/ledgerofthelow/internal/engine/obligations.go`

## Priority Issues

- Event handler retry semantics.
- PvP completion only after successful victim-side resolution.
- Travel arrival only after successful import.
- Partial debt repayment federation events.
- Victim-side PvP enforcement.

## Verification

- Add or update tests for protocol behavior.
- Run `go test ./...` from `games/ledgerofthelow`.
- Update `07_Backlog.md`, `08_Decisions.md`, or `09_Testing.md` if the work changes known gaps.
