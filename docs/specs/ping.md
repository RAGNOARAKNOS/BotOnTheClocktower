# Ping (`!botc ping`)

Status: DONE

## Goal

Let anyone check the bot is online and reading commands.

## Behaviour

- **Who can run it:** anyone
- **Where from:** any channel
- **Needs a registered game:** no

```text
!botc ping
→ "pong"
```

The reply is a plain message in the same channel, not a threaded reply.

## Implementation

- `extractCommand` in [internal/bot/bot.go](../../internal/bot/bot.go).
