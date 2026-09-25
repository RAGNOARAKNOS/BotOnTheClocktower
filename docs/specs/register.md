# Register a game (`!botc register`, alias `!botc start`)

Status: DONE

## Goal

Start a game on the server: set up the admin and game channels and record who the Storyteller is.

## Behaviour

- **Who can run it:** anyone. Whoever runs it becomes the Storyteller.
- **Where from:** any text channel on the server. That channel becomes the **admin channel**.
- **Needs a registered game:** no, and it's refused if one already exists.

Steps:

1. If a game is already registered, refuse and name the current Storyteller.
2. Find the voice channel named exactly `Town Square`. If there isn't one, refuse and say how to fix it. Nothing is registered.
3. Record the server, admin channel (this channel), game channel (Town Square) and Storyteller (the sender).
4. Post an announcement in Town Square's text chat.
5. Give the sender the `BoTC-StoryTeller` role.
6. Reply with confirmation, plus a warning if the role couldn't be given.

```text
!botc register
→ Game channel: "A new game has begun. @Alice is the Storyteller."
→ Reply: "Game registered. @Alice is the Storyteller. This is the admin channel; #Town Square is the game channel."

!botc start   (game already registered)
→ Reply: "A game is already registered, with @Alice as the Storyteller. This command will not execute"
```

## Rules & edge cases

- Refused for everyone while a game is registered, including the current Storyteller. End the game first with `!botc unregister`.
- If the `BoTC-StoryTeller` role is missing or above the bot's role, registration still succeeds and the reply includes a warning. The game can run without the role.
- The bot does not join voice.
- One game at a time, across all servers the bot is in.

## Done when

- [x] Refuses if a game is registered
- [x] Refuses if there's no `Town Square` voice channel
- [x] Records admin channel, game channel and Storyteller
- [x] Announces in Town Square and replies in the admin channel
- [x] Gives the `BoTC-StoryTeller` role, warning on failure

## Open questions

- Should only certain people (e.g. moderators, or members with a role) be allowed to register?
- Should registration also set up the village map (what `!botc map` does), so `map` isn't a separate step?

## Decisions

- 2026-09-25: The sender becomes Storyteller and is given the `BoTC-StoryTeller` role; role names are exact and case-sensitive.
- 2026-09-25: Refuse if a game is already registered, for everyone.
- 2026-09-25: The register channel is the admin channel; Town Square is the game channel, and must exist.
- 2026-09-25: A failure to give the role is a warning, not a failure, because the game works without it.
- 2026-09-25: The bot doesn't join voice; it had no reason to.
- 2026-09-25: `start` added as an alias.

## Implementation

- `register`, `findVoiceChannelID`, `assignStorytellerRole`, `findRoleID` in [internal/bot/bot.go](../../internal/bot/bot.go).
- State is in memory only, so a restart forgets the game (but not the roles it gave out).
