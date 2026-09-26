# Map the village (`!botc map`)

Status: DONE (with known gaps)

## Goal

Find the village's voice channels so later commands can move players between them. (Working out who is playing is now `!botc village create`; see [village-management.md](village-management.md).)

## Behaviour

- **Who can run it:** anyone
- **Where from:** any channel
- **Needs a registered game:** yes

Steps:

1. Find the server's channels whose names exactly match the village locations, and record their IDs:

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
- If mapping fails (e.g. a Discord error), the bot replies with the error.

## Done when

- [x] Records IDs for the named village channels
- [ ] Reports which locations were found or missing

## Open questions

- Should the result be posted to the admin channel, e.g. found/missing locations?
- Keep the TTS announcement, or use a plain message?
- Should this run automatically as part of `register`?
- Where do the "Cottage-XX" channels fit, since `gather` and `bedtime` depend on them?

## Decisions

- 2026-09-26: `map` no longer lists or touches players. The player list is built by `!botc village create` ([village-management.md](village-management.md)), and `map` used to wipe it.

## Implementation

- `mapRooms`, `getMapGuildChannels`, `villageCodeLookup` in [internal/bot/bot.go](../../internal/bot/bot.go).
- Known gaps:
  - Matching ignores channel type. If a category or text channel shares a name, or two channels share a name, the recorded ID is unpredictable. `Rooms["TS"]` could differ from the game channel `register` chose.
