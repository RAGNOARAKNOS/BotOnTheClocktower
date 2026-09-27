# Register a game (`!botc register`, alias `!botc start`)

Status: DONE

## Goal

Start a game on the server: set up the admin and game channels, find the village's voice channels, and record who the Storyteller is.

## Behaviour

- **Who can run it:** anyone, while no game is registered. Whoever runs it becomes the Storyteller.
- **Where from:** any text channel on the server. That channel becomes the **admin channel**, and from then on every command only works there (see [command-handling.md](command-handling.md)).
- **Needs a registered game:** no, and it's refused if one already exists. It and `ping` are the only commands that run without one.

Steps:

1. If a game is already registered, refuse and name the current Storyteller.
2. Find the village's voice channels by their exact names. If any is missing, refuse, name the missing ones and say how to fix it. Nothing is registered.

   | Code | Channel name |
   | --- | --- |
   | `TS` | Town Square |
   | `CA` | Cathedral |
   | `CF` | Campfire |
   | `PS` | Potion Shop |
   | `TW` | Tower |
   | `RS` | Riverside |
   | `SC` | Storyteller's Corner |

3. Record the server, admin channel (this channel), game channel (Town Square), the other rooms, and the Storyteller (the sender).
4. Post an announcement in Town Square's text chat.
5. Give the sender the `BoTC-StoryTeller` role.
6. Check the bot's permissions in the admin channel and every village voice channel.
7. Reply with confirmation, plus a warning if the role couldn't be given, and a warning listing each channel's missing permissions.

```text
!botc register
→ Game channel: "A new game has begun. @Alice is the Storyteller."
→ Reply: "Game registered. @Alice is the Storyteller. This is the admin channel; #Town Square is the game channel."

!botc register   (the server has no Campfire or Tower voice channel)
→ Reply: "Could not find these village voice channels: Campfire, Tower. Create voice channels with exactly these names, then try again. This command will not execute"

!botc register   (the bot can't see the village channels)
→ Reply: "Game registered. …
   Warning: the bot is missing permissions it needs in these channels:
   - #Town Square: can't see this channel at all (needs View Channels first)
   - #Cathedral: …"

!botc start   (game already registered; Storyteller, in the admin channel)
→ Reply: "A game is already registered, with @Alice as the Storyteller. This command will not execute"

!botc start   (game already registered; anyone else, or another channel)
→ Reply: "Commands only work for the Storyteller (@Alice), in the admin channel #st-admin. This command will not execute"
```

## Rules & edge cases

- Refused for everyone while a game is registered, including the current Storyteller. End the game first with `!botc unregister`. Who sent it decides which refusal they see.
- If the `BoTC-StoryTeller` role is missing or above the bot's role, registration still succeeds and the reply includes a warning. The game can run without the role.
- Missing channel permissions are a warning, not a failure: registration still succeeds. Without View Channels the bot ignores commands in that channel, so the warning is the only sign until someone notices it's silent.
- Only voice channels count: a text channel or category with a room's name is ignored. If several voice channels share a name, the first is used.
- The rooms are found once, at registration. A room created or renamed later isn't picked up until the game is registered again.
- Permissions needed: admin channel: View Channels, Send Messages, Read Message History. Town Square and the other village rooms: View Channels, Send Messages, Send TTS Messages, Connect, Move Members. Administrator covers them all.
- The bot does not join voice.
- One game at a time, across all servers the bot is in.

## Done when

- [x] Refuses if a game is registered
- [x] Refuses if any village voice channel is missing, naming them
- [x] Records every village room, so `gather` announces in all of them
- [x] Records admin channel, game channel and Storyteller
- [x] Announces in Town Square and replies in the admin channel
- [x] Gives the `BoTC-StoryTeller` role, warning on failure
- [x] Warns about missing channel permissions in the admin channel and village channels

## Open questions

- Should only certain people (e.g. moderators, or members with a role) be allowed to register?

## Decisions

- 2026-09-25: The sender becomes Storyteller and is given the `BoTC-StoryTeller` role; role names are exact and case-sensitive.
- 2026-09-25: Refuse if a game is already registered, for everyone.
- 2026-09-25: The register channel is the admin channel; Town Square is the game channel, and must exist.
- 2026-09-25: A failure to give the role is a warning, not a failure, because the game works without it.
- 2026-09-25: The bot doesn't join voice; it had no reason to.
- 2026-09-25: `start` added as an alias.
- 2026-09-27: `register` checks the bot's channel permissions and warns about any that are missing, after the bot silently ignored commands in game channels it couldn't view.
- 2026-09-26: `register` is the only command that runs with no game registered; every other command is ignored until then, and afterwards only works for the Storyteller in the admin channel.
- 2026-09-27 (from the removed `map` spec): Only voice channels are matched, and the first with each name wins. The village is just the rooms in `villageCodeLookup`; there are no "Cottage-XX" channels, and the planned `bedtime` command, which relied on them, is dropped.
- 2026-09-27: `!botc map` is removed. `register` finds the village's voice channels itself, and refuses unless all seven exist, naming any that are missing. So every game has every room, and `gather` always announces in all of them.

## Implementation

- `register` in [internal/bot/lifecycle.go](../../internal/bot/lifecycle.go); `newGame`, `villageCodeLookup`, `villageRooms` (voice channels by name, tested in `game_test.go`) and `missingRooms` in [internal/bot/game.go](../../internal/bot/game.go); the rooms are kept in `Game.Rooms`, and `register` is tested in `lifecycle_test.go`; `assignStorytellerRole`, `findRoleID` in [internal/bot/roles.go](../../internal/bot/roles.go); `channelAccessWarning`, `missingPermissions` in [internal/bot/permissions.go](../../internal/bot/permissions.go).
- State is in memory only, so a restart forgets the game (but not the roles it gave out).
