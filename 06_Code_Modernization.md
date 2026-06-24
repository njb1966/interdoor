# Code Modernization

## Goal

InterDoor will likely integrate old or open-source terminal games. Modernization should make those games maintainable and federatable without erasing their design or turning them into unrelated modern applications.

## Principles

- Preserve gameplay before improving internals.
- Keep terminal UX and pacing intact unless a change is intentional.
- Do not replace a simple working architecture with a heavier one.
- Avoid framework migrations as a first move.
- Add tests around existing behavior before changing risky code.
- Prefer small compatibility adapters over rewrites.
- Separate game-specific cleanup from federation integration.

## Porting Order

1. Build and run the original game or closest available source.
2. Document current behavior, data files, timing model, and persistence.
3. Identify player identity and state boundaries.
4. Add a minimal InterDoor Tier 1 adapter.
5. Add event and roster hooks for Tier 2.
6. Only then consider deeper changes for travel, PvP, or obligations.

## What To Preserve

- Daily/session cadence.
- Text UI style.
- Simple deployment.
- Original mechanics unless incompatible with federation.
- Data formats when they are understandable and sufficient.
- Sysop expectations for configuration and operation.

## What To Improve

- Memory safety and bounds checks.
- Build reproducibility on Debian.
- Clear configuration.
- Password storage.
- Logging.
- Testability around combat, economy, turns, and persistence.
- Federation-facing identifiers and event emission.

## Dependency Rule

Adding a dependency is justified only when it:

- Removes a real security or correctness risk.
- Replaces brittle custom code with a stable standard component.
- Is small enough for the project's deployment philosophy.
- Does not require Docker, orchestration, or a heavy runtime by default.

## Federation Adapter Pattern

For legacy ports, prefer an adapter layer:

- Translate local player IDs to InterDoor global IDs.
- Publish roster entries from existing player records.
- Emit events at existing action boundaries.
- Keep the original game loop intact.
- Store hub API key and sync cursors outside fragile legacy files when needed.

This keeps the original game understandable while letting it participate in the network.

## Rewrite Threshold

A rewrite should require explicit approval. It may be justified when:

- The original source cannot be legally or practically maintained.
- The build cannot be made reliable.
- The persistence model prevents safe federation.
- The code cannot be safely exposed to public network traffic.

Even then, the rewrite target is behavioral preservation plus federation, not novelty.
