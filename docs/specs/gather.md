# Gather players in Town Square (`!botc gather`)

Status: DONE

## Goal

Bring every player back to Town Square with a countdown: warn them in text and TTS, give them time to finish their conversations, then move anyone still elsewhere. Used at the end of the NIGHT phase and when the town gathers for nominations.

## Behaviour

- **Who can run it:** the Storyteller only (every command is; see [command-handling.md](command-handling.md))
- **Where from:** the admin channel only
- **Needs a registered game:** yes
- `!botc gather` starts a 60-second countdown. `!botc gather <minutes>` starts a countdown of that many minutes, from 1 to 10.
- When the command is entered, the bot announces that the Storyteller will be bringing the players back to Town Square, and when:
  - a TTS message in the text chat of each village voice channel: Town Square and the other rooms `register` found, and
  - a DM to every player in the village.
- 30 seconds before the countdown ends, it sends a second warning to the same channels and players, again with TTS in the channels.
- When the countdown ends, it moves every village player who is in a voice channel other than Town Square into Town Square. The Storyteller isn't moved.
- The Storyteller gets a reply when the countdown starts, and a report in the admin channel when the players have been moved. The report lists players who couldn't be moved because they aren't in voice, and any moves that failed.
- `!botc gather cancel` stops a running countdown and tells the same channels and players that the gathering is off.

```text
!botc gather
→ Reply: "Gathering the players in Town Square in 60 seconds."
→ Each voice channel (TTS): "The Storyteller will be bringing everyone back to Town Square in 60 seconds."
→ Each player (DM): "The Storyteller will be bringing everyone back to Town Square in 60 seconds."
   … 30 seconds later …
→ Each voice channel (TTS) and each player (DM): "30 seconds until everyone is brought back to Town Square."
   … 30 seconds later …
→ Players moved to Town Square
→ Admin channel: "Gathered 7 players in Town Square."

!botc gather 5
→ Same, with "in 5 minutes"; the second warning comes at 4 minutes 30 seconds.

!botc gather cancel
→ Reply: "Gathering cancelled."
→ Each voice channel (TTS) and each player (DM): "The Storyteller has called off the gathering in Town Square."
```

## Rules & edge cases

- The time is a whole number of minutes, from 1 to 10. Anything else gets a usage reply and starts nothing.
- Only one countdown runs at a time. `gather` while one is running is refused, with the time left.
- `gather cancel` with no countdown running just says so.
- A player who isn't in voice can't be moved, and is named in the report.
- Players already in Town Square aren't moved.
- The countdown runs after the command has finished, so it mustn't hold `Bot.mu` while waiting. When it fires (warning or move), it takes `Bot.mu` and checks that the game it was started for is still the registered one; if the game was unregistered or replaced, it does nothing.
- A failure to send one message, DM or move doesn't stop the others. Failures are listed in the admin channel report.
- Discord's TTS reads a message aloud to people who have that text chat open with TTS turned on; it doesn't play into the voice call. The bot never joins voice (see [command-handling.md](command-handling.md)), so TTS messages are the only spoken announcement.
- The bot needs **Send Messages** and **Send TTS Messages** in every voice channel it announces in, and **Move Members** and **Connect** on Town Square.

## Done when

- [x] `!botc gather` and `!botc gather <minutes>` start a countdown, with a usage reply for a bad time
- [x] Opening announcement: TTS in each voice channel's text chat and a DM to each player
- [x] Second warning 30 seconds before the end, to the same places
- [x] Players are moved to Town Square when the countdown ends, with a report in the admin channel
- [x] A second `gather` while a countdown runs is refused
- [x] `gather cancel` stops the countdown and announces it
- [x] The countdown does nothing if the game is unregistered or replaced before it fires
- [x] `permissions.go` checks for Send Messages and Send TTS Messages in the channels gather announces in
- [x] README Commands table and Features section updated
- [x] UML diagrams in [docs/uml/](../uml/README.md) updated (`/uml-sync`), including a source-map row for any new function

## Out of scope

- Joining voice to speak the announcement.
- Moving players anywhere other than Town Square.

## Open questions

- None.

## Decisions

- 2026-09-26: The players are the village list from `!botc village` ([village-management.md](village-management.md)).
- 2026-09-27: The earlier plan is scrapped. `gather` is now a countdown: announce in text and TTS in the voice channels and by DM, warn again 30 seconds before the end, then force-move the players to Town Square. The default is 60 seconds; the Storyteller can give a time in minutes.
- 2026-09-27: Players are moved from whichever voice channel they're in, not only the village rooms.
- 2026-09-27: `!botc map` is removed; `register` requires and records every village room, so every gathering is announced in all of them (it used to reach only Town Square until `map` had been run).
- 2026-09-27: Announcements go to the village voice channels only, not every voice channel on the server. One countdown at a time: a second `gather` is refused. `gather cancel` stops it and tells everyone. The maximum is 10 minutes. The Storyteller isn't moved. The report lists players who aren't in voice.

## Implementation

- [internal/bot/gather.go](../../internal/bot/gather.go):
  - `gather` is the handler, and `parseGather` reads the time.
  - `gatherCancel` handles `gather cancel`.
  - `gatherAnnounce` sends the TTS messages and DMs.
  - `gatherWarn` and `gatherMove` are the timed steps, run through `gatherFire`, which takes `Bot.mu` and checks the game and countdown are unchanged.
- The running countdown is `Game.gather` (a `gatherCountdown` holding both timers). `unregister` stops it.
- `Game.villageChannels` lists Town Square and the other village rooms, each once.
- `permissions.go` checks Send Messages and Send TTS Messages in Town Square and the village rooms.
- Where players are comes from discordgo's voice-state cache, so a player who joined voice while the bot was offline is still found once the gateway has sent the guild's voice states.
