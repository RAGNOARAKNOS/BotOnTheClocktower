# Map the village (`!botc map`)

Status: DONE (with known gaps)

## Goal

Find the village's voice channels so later commands can move players between them. (Working out who is playing is now `!botc village create`; see [village-management.md](village-management.md).)

## Behaviour

- **Who can run it:** the Storyteller only
- **Where from:** the admin channel only
- **Needs a registered game:** yes; without one it's ignored (see [command-handling.md](command-handling.md))

Steps:

1. Find the server's voice channels whose names exactly match the village locations, and record their IDs (the first, if several share a name):

   | Code | Channel name |
   | --- | --- |
   | `TS` | Town Square |
   | `CA` | Cathedral |
   | `CF` | Campfire |
   | `PS` | Potion Shop |
   | `TW` | Tower |
   | `RS` | Riverside |
   | `SC` | Storyteller's Corner |

2. Post "Town Locations Mapped" as a text-to-speech message in the admin channel.

```text
!botc map
→ Admin channel (TTS): "Town Locations Mapped"
```

## Rules & edge cases

- Missing locations are skipped silently.
- Only voice channels count: a text channel or category with a location's name is ignored.
- If mapping fails (e.g. a Discord error), the bot replies with the error.

## Done when

- [x] Records IDs for the named village channels
- [ ] Reports which locations were found or missing

## Open questions

- Should the result be posted to the admin channel, e.g. found/missing locations?
- Keep the TTS announcement, or use a plain message?
- Should this run automatically as part of `register`?

## Decisions

- 2026-09-26: `map` no longer lists or touches players. The player list is built by `!botc village create` ([village-management.md](village-management.md)), and `map` used to wipe it.
- 2026-09-26: Storyteller only, from the admin channel, like every command (was: anyone, anywhere). With no game registered it's now ignored rather than replying "No game registered".
- 2026-09-27: No "Cottage-XX" channels. The village is just the rooms in `villageCodeLookup`, and the planned `bedtime` command, which relied on cottages, is dropped.
- 2026-09-27: Only voice channels are matched, and the first with each name wins, the same rule `register` uses to find Town Square. So `Rooms["TS"]` is always the game channel.

## Implementation

- `mapRooms` in [internal/bot/lifecycle.go](../../internal/bot/lifecycle.go); `villageRooms` (the matching, tested in `game_test.go`) and `villageCodeLookup` in [internal/bot/game.go](../../internal/bot/game.go).
- Known gap: it doesn't report which locations were found or missing (see Open questions).
