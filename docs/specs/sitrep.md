# Situation report (`!botc sitrep`)

Status: DONE

## Goal

Show whether a game is running, and where.

## Behaviour

- **Who can run it:** anyone
- **Where from:** any channel
- **Needs a registered game:** no

```text
!botc sitrep   (no game)
→ "SITREP-Game is not initialised"

!botc sitrep   (game registered)
→ "SITREP-Game is initialised at guildid# 1234… admin channel #st-admin game channel #Town Square storyteller @Alice"
```

Channels and the Storyteller appear as clickable mentions. The server is shown as a raw ID.

## Open questions

- Should it also show mapped rooms and players, once player tracking exists?
- Should the wording be tidied (e.g. "Game in progress" rather than "initialised")?

## Implementation

- `sitrep` in [internal/bot/bot.go](../../internal/bot/bot.go).
