# OBS integration (`!botc obs ...`)

Status: PLANNED

## Goal

Control a local OBS instance from Discord (scenes, sources, audio, recording), and change scenes automatically as the game moves between phases.

## Behaviour

The detailed plan lives in [roadmap.md](../../roadmap.md): architecture, configuration, five implementation phases with checklists, error handling and testing. Keep that file as the source of truth, and record decisions here or there, but not both.

## Open questions

Points where the roadmap doesn't match the current code:

- It says to add OBS settings in `config.go`, but the `Settings` struct lives in `internal/bot`.
- It says to gate commands behind a "Storyteller check (same pattern as `register`)". No command other than `unregister` checks for the Storyteller yet.
- Phase 3 hooks into `bedtime` and a "wake" command, which don't exist yet. Is "wake" the same as `gather`?

## Decisions

- (none yet)

## Implementation

Not started.
