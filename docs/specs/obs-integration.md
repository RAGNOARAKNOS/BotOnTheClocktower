# OBS integration (`!botc obs ...`)

Status: PLANNED

## Goal

Control a local OBS instance from Discord (scenes, sources, audio, recording), and change scenes automatically as the game moves between phases.

## Behaviour

The detailed plan lives in [roadmap.md](../roadmap.md): architecture, configuration, five implementation phases with checklists, error handling and testing. Keep that file as the source of truth, and record decisions here or there, but not both.

## Open questions

Points where the roadmap doesn't match the current code:

- It says to add OBS settings to a `Settings` struct in `config.go`. That struct no longer exists: add them to `config.Config` and pass them to the bot from `main`.
- It says to gate commands behind a "Storyteller check (same pattern as `register`)". That's no longer needed: every command is already restricted to the Storyteller in the admin channel.
- Phase 3 hooks into "bedtime" and "wake" commands. The planned `bedtime` command was dropped on 2026-09-27, so nothing marks nightfall. What should trigger the night scene, and is "wake" the same as `gather`?

## Decisions

- (none yet)

## Implementation

Not started.
