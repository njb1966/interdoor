# InterDoor

InterDoor is a federated networking framework for terminal games.

The network is the product. Individual games are nodes or reference clients of the framework.
Phase 1 is focused on the hub, node sync, protocol, SDK surface, operator tooling, tests, and
deployment shape for a small federated SSH game network.

## Current Status

Phase 1 is hardened for the current documented deployment shape:

- One public hub.
- Local-only hub administration.
- SQLite hub storage.
- Independently operated SSH game nodes.
- Ledger of the Low as the Phase 1 reference node.

No known critical Phase 1 production blocker remains for this deployment shape. Remaining items in
the backlog are conditional hardening or future scale work.

Empire Ascendant/Dominion is deferred to Phase 2. It must be designed and approved before any
implementation is added to InterDoor.

## Canonical Documentation

The root numbered documents are the source of truth for InterDoor Phase 1:

- [00_Project_Charter.md](00_Project_Charter.md) - purpose, scope, and Phase 1 boundaries.
- [01_Project_Rules.md](01_Project_Rules.md) - operating rules and design constraints.
- [02_Architecture.md](02_Architecture.md) - hub, node, engine, SDK, storage, and deployment shape.
- [03_Hub_Specification.md](03_Hub_Specification.md) - hub API, projections, local admin, and production readiness.
- [04_Federation_Protocol.md](04_Federation_Protocol.md) - HTTP JSON protocol and federation behavior.
- [05_Game_SDK.md](05_Game_SDK.md) - game integration tiers and SDK expectations.
- [06_Code_Modernization.md](06_Code_Modernization.md) - rules for future legacy/open-source ports.
- [07_Backlog.md](07_Backlog.md) - completed hardening and remaining conditional backlog.
- [08_Decisions.md](08_Decisions.md) - accepted decisions and unresolved decision points.
- [09_Testing.md](09_Testing.md) - test coverage, restore rehearsal, and verification expectations.

## Repository Layout

- `games/ledgerofthelow/` - current Go reference implementation containing the hub, node,
  federation client, generic engine, and Ledger of the Low reference game module.
- `games/sdk/` - HTTP examples for non-Go game integration.
- `games/empireascendant/` - Phase 2 historical/planning material only. Not active Phase 1 code.
- `docs/` - legacy design/source material. Root numbered docs win for Phase 1 framework decisions.
- `prompts/` - implementation and context-reset prompts.

## Important Boundaries

- LoRD, Usurper, and BRE are examples only, not planned implementations.
- Future games must be custom works or license-compatible open-source games that can legally be
  modified and redistributed.
- Do not extract a generic game-node framework from Ledger alone. Wait until a second approved game
  node exists for comparison.
- Do not add game implementations to the Phase 1 build or release surface without explicit approval.

## Verification

From the reference implementation directory:

```bash
cd games/ledgerofthelow
go test ./...
make build
```

For deployed hub acceptance checks, use the checklist in
[03_Hub_Specification.md](03_Hub_Specification.md).
