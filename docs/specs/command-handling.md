# Command handling (all commands)

Status: DONE

## Goal

Rules shared by every command: how messages are recognised as commands, and how the bot stays stable while handling them.

## Behaviour

- A command is a message whose first word is `!botc` (any capitalisation) followed by at least one more word. The second word, lowercased, is the command name; further words are arguments.
- Every command can also be run as a `/botc` slash command, with the same access and results; see [slash-commands.md](slash-commands.md). The rules below apply to both, except where that spec says otherwise.
- Messages from bots, including this one, are ignored.
- A message that is just `!botc` with nothing after it is ignored.
- Commands run one at a time, never concurrently.
- **Access, checked before every command:**
  - No game registered: only `register` / `start` and `ping` run, from any channel; `register`'s channel becomes the admin channel. Every other command, including unknown ones, is ignored with no reply (logged to the console).
  - Game registered: every command must come from the Storyteller, in the admin channel of the registered server. The exception is `ping`, which the Storyteller can send from any channel. Anything else is refused: `Commands only work for the Storyteller (@Alice), in the admin channel #st-admin. This command will not execute`
- Unknown command (from the Storyteller, in the admin channel) → `Huh? WTF is that command?!`
- A group command (`village`, `character`) with no subcommand, or one it doesn't know → the usage of each of its subcommands. A command that can't be read (for example, missing its mentions) replies with its own usage.
- If a command hits an unexpected bug (a panic), the bot stays up and replies `Something went wrong running that command. Check the bot's logs.`
- Replies go to the channel the command was sent in. Because of the access rule, that's the admin channel, except for `register`, `ping` and refusals.

## Rules & edge cases

- A refusal is sent in the channel the command came from, which can be a public channel or a DM, so it names the Storyteller and the admin channel.
- `register` while a game is registered is caught by the access rule unless it comes from the Storyteller in the admin channel, in which case `register` itself refuses it.

## Out of scope

- Slash-command details: see [slash-commands.md](slash-commands.md).

## Open questions

- Should unknown commands reply with a help list instead?

## Decisions

- 2026-09-25: Prefix must be exactly `!botc` (was: any word containing `!botc`). Command names are case-insensitive.
- 2026-09-25: Commands are serialised with a mutex, because discordgo runs handlers concurrently and two simultaneous `register`s could both succeed.
- 2026-09-25: Handlers must not `panic`; a `recover` in `newMessage` is the safety net so a bug can't crash the bot and lose the in-memory game.
- 2026-09-26: Commands only work from the admin channel, and only for the Storyteller. With no game registered only `register` / `start` runs (from any channel, which becomes the admin channel) and everything else is ignored silently. Once registered, commands from anyone else or anywhere else get a refusal. This replaces the per-command checks (`requireStorytellerInAdmin`, and `unregister`'s own), so `ping`, `sitrep` and `map` are no longer open to everyone.
- 2026-09-26: Exception for `ping`: anyone can ping while no game is registered, and the Storyteller can ping from any channel once one is.
- 2026-09-27: The `pmove`/`cmove` stubs are removed; they now get the unknown-command reply. Moving players returns with `gather`.
- 2026-09-27: Stop printing every command word to the console, as whispers and guidance are secret.
- 2026-09-27: Commands also work as `/botc` slash commands, permanently and in parity with `!botc` ([slash-commands.md](slash-commands.md)).

## Implementation

- `newMessage` in [internal/bot/bot.go](../../internal/bot/bot.go), and `interaction` for slash commands in [internal/bot/interactions.go](../../internal/bot/interactions.go). Both build a `request`. `runCommand`, `resolve` and `allowed` (the access check) are in [internal/bot/commands.go](../../internal/bot/commands.go).
- Every command is declared once, in the `commands` table in [internal/bot/commandtable.go](../../internal/bot/commandtable.go): name, aliases, handler, access flags, usage, and its `/botc` description, options and form. Groups hold their subcommands. `resolve` walks the table (names are case-insensitive), and the handler gets the words after its name in `req.args` and its usage in `req.usage`.
- Each command's table entry sets its access (a subcommand's are its own, not its group's): `beforeGame` (runs for anyone, anywhere, with no game: `register`, `start`, `ping`) and `anyChannel` (the Storyteller can run it anywhere: `ping`). `allowed` is pure and tested in `commands_test.go`.
- Message content is never logged, because whispers and character guidance are secret. Send failures are logged by the `reply`/`send` helpers.
