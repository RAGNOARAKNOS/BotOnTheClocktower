# OBS integration (`!botc obs ...`)

Status: PLANNED

## Goal

Control a local OBS instance from Discord (scenes, sources, audio, recording), and change scenes automatically as the game moves between phases.

## Behaviour

The detailed plan lives in [roadmap.md](../roadmap.md): architecture, configuration, five implementation phases with checklists, error handling and testing. Keep that file as the source of truth, and record decisions here or there, but not both.

## Open questions

- Phase 3 changes scene at nightfall and daybreak. The planned `bedtime` command was dropped on 2026-09-27, so nothing marks nightfall. What should trigger the night scene, and is daybreak `gather`'s move?

## Decisions

- 2026-09-27: The bot connects directly to OBS's WebSocket server (obs-websocket v5, through `goobs`). No OBS traffic goes through Discord's API; Discord only carries the Storyteller's `!botc obs` commands.
- 2026-09-27: The bot runs on the same PC as OBS and connects over loopback (`localhost:4455`, or `host.docker.internal:4455` from Docker Desktop). OBS keeps authentication on, and port 4455 isn't opened in the firewall.
- 2026-09-27: Every `obs` command, `obs ping` included, keeps the default access: Storyteller only, from the admin channel, with a game registered. None runs before a game is registered.
- 2026-09-27: No OBS call is made while holding `Bot.mu`. A background loop keeps the connection up, and commands never dial; each OBS command makes its call once the lock is released, through `request.after` (see the roadmap's *Keeping OBS calls outside `Bot.mu`*).

## Implementation

Not started.
