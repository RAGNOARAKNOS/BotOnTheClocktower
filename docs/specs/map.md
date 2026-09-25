# Map the village (`!botc map`)

Status: DONE (with known gaps)

## Goal

Find the village's voice channels so later commands can move players between them, and work out who is playing.

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
3. List everyone in Town Square voice except the Storyteller, **to the bot's console only**.

```text
!botc map
→ Admin channel (TTS): "Town Locations Mapped"
```

## Rules & edge cases

- Missing locations are skipped silently.
- If mapping fails (e.g. a Discord error), the bot replies with the error.

## Done when

- [x] Records IDs for the named village channels
- [ ] Records the players (currently printed to the console, then thrown away)
- [ ] Reports which locations were found or missing

## Open questions

- Should the player list be stored, and if so, who counts as a player: people in Town Square, or people with `BoTC-Player`?
- Should the result be posted to the admin channel, e.g. found/missing locations and player names?
- Keep the TTS announcement, or use a plain message?
- Should this run automatically as part of `register`?
- Where do the "Cottage-XX" channels fit, since `gather` and `bedtime` depend on them?

## Decisions

- (none yet)

## Implementation

- `mapRooms`, `getMapGuildChannels`, `mapPlayers`, `villageCodeLookup` in [internal/bot/bot.go](../../internal/bot/bot.go).
- Known gaps:
  - Matching ignores channel type. If a category or text channel shares a name, or two channels share a name, the recorded ID is unpredictable. `Rooms["TS"]` could differ from the game channel `register` chose.
  - `Players` is reset to empty but never filled, so `playerNameToId` always returns "".
