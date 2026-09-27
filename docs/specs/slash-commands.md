# Slash commands (`/botc ...`)

Status: DONE

## Goal

Let the Storyteller run every command through Discord's `/botc` slash-command UI, with Discord's autocomplete and option checking, as well as by typing `!botc`.

## Behaviour

- **Who can run it:** the same as the `!botc` form of each command (see [command-handling.md](command-handling.md)); `allowed` checks both.
- **Where from:** the same as `!botc`.
- **Needs a registered game:** the same as `!botc`.
- One command, `/botc`, whose subcommands mirror the `!botc` commands:
  - `/botc ping`, `register`, `unregister`, `sitrep`, `grimoire`
  - `/botc gather [minutes:1–10] [cancel:True]`
  - `/botc village create|add|remove|list`. `add` and `remove` take a `players` text option holding @mentions, e.g. `@Alice @Bob`.
  - `/botc character assign|team|kill|revive|ghostvote|announce|clear|list|send`. `team`, `kill`, `revive`, `ghostvote`, `clear` and `send` take `players`, and `team` also takes `team:good|evil`.
  - `/botc whisper player:@x` opens a form with a multi-line box for the message.
  - `/botc character assign player:@x [team:good|evil] [character:Name]` opens a form with the character's name (filled in from `character`) and a multi-line box for the guidance.
- Every slash reply, including refusals, is **ephemeral**: only the person who ran it sees it. Everything else a command posts is unchanged: Town Square announcements, DMs, `gather`'s TTS messages and its admin-channel report.
- With no game registered, a slash command other than `register` or `ping` gets an ephemeral "No game registered" reply. Discord needs every slash command answered, so it can't be silently ignored as `!botc` is.
- A slash command that finishes without replying gets an ephemeral "Done." (Every command replies today; this is a safety net.)

```text
/botc gather minutes:2
→ Ephemeral reply: "Gathering the players in Town Square in 2 minutes."
→ Town Square, rooms and DMs: as for !botc gather 2

/botc whisper player:@Alice
→ Form: "Whisper to Alice" with a message box
→ On submit: as for !botc whisper @Alice <message>
```

## Rules & edge cases

- **Parity:** every `!botc` command and subcommand has a `/botc` equivalent and the other way round, with the same access, effects and reply text. Aliases (`start`, `end`) are `!botc` only. A test enforces parity.
- Slash commands are built into the same words and text that `!botc` would produce and run through the same handlers, so the two can't drift apart.
- Slash commands are registered on each server the bot is in, when it starts and when it joins a server. This needs the bot to be invited with the `applications.commands` scope.
- `/botc` shows in everyone's command list. Discord can't hide it from all but one person; `allowed` refuses anyone who isn't the Storyteller.
- A form (whisper, character assign) is only opened once `allowed` has passed. The submitted form is checked again, since the game can change while it's open.
- Discord needs an answer within 3 seconds, so the bot acknowledges each slash command at once ("thinking…", ephemeral) and fills in the reply when the command finishes.

## Done when

- [x] Handlers take a `request` that either form can build
- [x] `/botc` is registered on every server the bot is in
- [x] Every command runs from `/botc`, with ephemeral replies
- [x] Whisper and character assign use a form for multi-line text
- [x] A parity test covers commands and subcommands
- [x] README (commands and invite scope), agent guidance, [command-handling.md](command-handling.md) and UML updated

## Out of scope

- Removing `!botc`: both forms stay permanently.
- Hiding `/botc` from non-Storytellers.
- Making `!botc` replies private: the admin channel's permissions already do that.

## Open questions

- None.

## Decisions

- 2026-09-27: Both `!botc` and `/botc` are kept permanently and in parity.
- 2026-09-27: Slash replies are visible only to the person who ran the command. `/botc` stays visible in everyone's command list, and `!botc` replies are unchanged.

## Implementation

- Handlers take a `request` ([internal/bot/commands.go](../../internal/bot/commands.go)). `newMessage` builds one from a message, and the slash side from an interaction.
- [internal/bot/slash.go](../../internal/bot/slash.go):
  - `slashCommands` builds `/botc` from the `commands` table ([internal/bot/commandtable.go](../../internal/bot/commandtable.go)): each entry's name, description and options, with groups as subcommand groups (`slashSubcommands`).
  - `slashWords` turns options into `!botc` words plus mentioned user IDs; `players` text is read with `mentionPattern`.
  - `whisperModal` and `assignModal` build the forms; each is the `form` of its command's table entry. `modalCommand` turns a submitted form into the raw text `parse.go` expects.
- [internal/bot/interactions.go](../../internal/bot/interactions.go):
  - `registerSlashCommands` runs on every `GuildCreate` and uses `ApplicationCommandBulkOverwrite`, so the servers always have the current definition.
  - `interaction` acknowledges the command before taking `Bot.mu` (except a command with a form, which must answer with the form), so a slow command already running can't make it miss Discord's 3-second deadline. It then holds `Bot.mu` and recovers from panics.
  - `slashResponder` acknowledges within 3 seconds, fills in the reply, and sends later replies as follow-ups, all ephemeral.
- Because `/botc` is built from the same table as `!botc`, the two can't drift apart. `TestSlashDefinition` checks the definition keeps to Discord's limits (a bad one is refused at registration, which is only logged), and `TestSlashOptionsRead` checks `slashWords` reads every option.
- Known gaps:
  - Not yet tried on Discord. In particular, check that picking players from the @ autocomplete in the `players` box sends `<@id>` mentions.
  - Usage and hint texts still show the `!botc` form.
