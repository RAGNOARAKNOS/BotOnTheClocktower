# Command handling (all commands)

Status: DONE

## Goal

Rules shared by every command: how messages are recognised as commands, and how the bot stays stable while handling them.

## Behaviour

- A command is a message whose first word is `!botc` (any capitalisation) followed by at least one more word. The second word, lowercased, is the command name; further words are arguments.
- Messages from bots, including this one, are ignored.
- A message that is just `!botc` with nothing after it is ignored.
- Commands run one at a time, never concurrently.
- Unknown command → `Huh? WTF is that command?!`
- If a command hits an unexpected bug (a panic), the bot stays up and replies `Something went wrong running that command. Check the bot's logs.`
- Replies go to the channel the command was sent in.

## Rules & edge cases

- `pmove` and `cmove` are recognised but currently do nothing and send no reply.
- Commands work in any channel the bot can read, including DMs, though game commands need a server.

## Out of scope

- Slash commands (Discord's `/command` UI). Everything is plain-text `!botc`.

## Open questions

- Should unknown commands reply with a help list instead?
- Should game-management commands only work in the admin channel?
- Should `pmove`/`cmove` reply "not implemented yet", or be removed?

## Decisions

- 2026-09-25: Prefix must be exactly `!botc` (was: any word containing `!botc`). Command names are case-insensitive.
- 2026-09-25: Commands are serialised with a mutex, because discordgo runs handlers concurrently and two simultaneous `register`s could both succeed.
- 2026-09-25: Handlers must not `panic`; a `recover` in `newMessage` is the safety net so a bug can't crash the bot and lose the in-memory game.

## Implementation

- `newMessage` and `extractCommand` in [internal/bot/bot.go](../../internal/bot/bot.go).
- Every argument is printed to the console (debug output).
