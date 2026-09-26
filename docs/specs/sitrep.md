# Situation report (`!botc sitrep`)

Status: DONE

## Goal

Show whether a game is running, and where.

## Behaviour

- **Who can run it:** the Storyteller only
- **Where from:** the admin channel only
- **Needs a registered game:** yes; without one it's ignored (see [command-handling.md](command-handling.md))

```text
!botc sitrep
→ "SITREP-Game is initialised at guildid# 1234… admin channel #st-admin game channel #Town Square storyteller @Alice"
```

Channels and the Storyteller appear as clickable mentions. The server is shown as a raw ID.

## Open questions

- Should it also show mapped rooms and players, once player tracking exists?
- Should the wording be tidied (e.g. "Game in progress" rather than "initialised")?
- With no game registered, `sitrep` is now ignored, so there's no way to ask the bot whether a game exists. Should `sitrep` be allowed before registration?

## Decisions

- 2026-09-26: Storyteller only, from the admin channel, like every command (was: anyone, anywhere, with or without a game).

## Implementation

- `sitrep` in [internal/bot/bot.go](../../internal/bot/bot.go).
- It has no "no game" branch: the access check ignores it when no game is registered.
