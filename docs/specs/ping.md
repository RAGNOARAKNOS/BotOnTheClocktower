# Ping (`!botc ping`)

Status: DONE

## Goal

Check the bot is online and reading commands.

## Behaviour

- **Who can run it:** anyone while no game is registered; once a game is registered, the Storyteller only (anyone else is refused, see [command-handling.md](command-handling.md))
- **Where from:** any channel, including DMs
- **Needs a registered game:** no

```text
!botc ping
→ "pong"
```

The reply is a plain message in the same channel, not a threaded reply.

## Decisions

- 2026-09-26: Storyteller only, from the admin channel, like every command (was: anyone, anywhere, with or without a game).
- 2026-09-26: Changed: `ping` always answers the Storyteller, from any channel. With no game registered anyone can ping, so the bot can be checked before registering.

## Implementation

- `runCommand` in [internal/bot/commands.go](../../internal/bot/commands.go).
