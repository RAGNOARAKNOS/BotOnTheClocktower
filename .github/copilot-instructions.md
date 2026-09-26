# Project instructions

The single source of project guidance for AI coding agents. GitHub Copilot reads this file directly, and `CLAUDE.md` imports it for Claude Code, so edit this file rather than `CLAUDE.md`. Paths are relative to the repository root.

## Language: British English

Write all prose in British English. This covers code comments, doc comments, Markdown documentation and commit messages.

- Use `-ise` / `-isation`, not `-ize` / `-ization`: initialise, serialise, organise, recognise, normalisation.
- Use `-our`: behaviour, colour, favour, honour.
- Use `-re`: centre, metre.
- Double the final `l` before a suffix: cancelled, travelling, modelled, labelled.
- Other common forms: licence (noun) / license (verb), practice (noun) / practise (verb), defence, catalogue, grey, programme (a schedule; "program" for software).

**Don't change code to match.** Keep American spelling where it's part of code, not prose:

- Identifiers, and names from Go's standard library or third-party packages (e.g. `Color`, `Initialize`, `discordgo` types and methods)
- String literals, config keys, JSON fields, command names, and Discord role or channel names
- Quoted text from external sources, such as API documentation or error messages

## Commit messages

Commit messages follow `.github/commit-message-instructions.md` (Copilot's commit message generator reads it through `.vscode/settings.json`).

## What this is

A Discord bot (Go, [discordgo](https://github.com/bwmarrin/discordgo)) that helps the Storyteller run a game of *Blood on the Clocktower* over Discord — mainly by moving players between voice channels ("Town Square", cottages, etc.) as the game switches between DAY and NIGHT phases. It's an early-stage personal project; most features in the README are PLANNED or IN WORK.

## Commands

```shell
go run ./cmd/bot                                   # run locally (needs BOTAPIKEY)
go build -o BotOnTheClocktower.exe ./cmd/bot       # build a binary
go test ./...                                      # tests (parser tests in internal/bot)
go vet ./...
docker build -t botontheclocktower .               # container image (distroless, nonroot)
```

Configuration: `BOTAPIKEY` (Discord bot token), read from the environment or from an optional `.env` file in the working directory. `.env` is gitignored — never commit it or print the token.

## Layout

- `cmd/bot/main.go` — entry point: `config.Load()` then `bot.Run(cfg.Token)`.
- `internal/config/config.go` — loads `.env` (a missing file is fine) and returns a `Config` holding the bot token.
- `internal/bot/` — one package, one file per feature. Keep files under about 300 lines; add a new file for a new feature.
  - `bot.go` — the `Bot` struct, `Run` (session setup) and `newMessage` (lock, recover).
  - `commands.go` — the `commands` table (name and alias → handler plus access flags), `extractCommand` (dispatch), `allowed` (access), `ping`, and the `reply`/`send` helpers.
  - `game.go` — `Game` (the registered game's state), `register`, `unregister`, `sitrep`, `map`.
  - `village.go` — the `village` commands. `roles.go` — Discord role lookups and changes. `discord.go` — member and channel lookups.
  - `characters.go` — the `character` commands and `Character`. `grimoire.go` — `character list`/`grimoire` output. `whisper.go` — `whisper` and the DM helpers.
  - `parse.go` — parsing raw message content (mentions, assignments, whispers). Tests sit next to the code: `parse_test.go`, `grimoire_test.go`, `characters_test.go`.
- `docs/uml/` — Mermaid UML activity and sequence diagrams of how the code works at runtime, with a source map from Go functions to diagrams.

`config` and `bot` don't import each other; `main` passes the token from one to the other.

## Feature specs

Intended behaviour lives in `docs/specs/`, one file per feature (index and workflow in `docs/specs/README.md`, template in `_template.md`). The project README describes what's built; specs describe what's intended.

- Before working on a feature, read its spec. Treat "Open questions" as undecided: ask rather than guess.
- When the user decides something or changes their mind, update the spec: Behaviour/Rules, plus a dated line under Decisions.
- When implementing, tick off "Done when" items; when a feature is finished, set Status to DONE, fill in Implementation, update the project README, and update the UML diagrams.

## How the bot works

This section covers the architecture and the rules to keep. For what each command does, see the README's Commands table; for step-by-step flows, see `docs/uml/`; for intended behaviour, see `docs/specs/`. Don't re-describe individual commands here.

- **Flow:** `bot.Run` opens the discordgo session and blocks until SIGINT/SIGTERM. `newMessage` ignores bots, needs `!botc` (any case) plus a command word, takes `Bot.mu` for the whole command, and recovers panics. `extractCommand` looks the command up in the `commands` table, checks `allowed`, then calls the handler.
- **One command at a time:** `Bot.mu` is held for the whole command, because discordgo runs each handler in its own goroutine. Don't call anything that re-enters the message handler while holding it.
- **Access** is decided once, by `allowed`, from the command's table entry. With no game registered, only `beforeGame` commands (`register`/`start`, `ping`) run, and everything else is ignored silently. With a game, only the Storyteller in the admin channel may run commands (`anyChannel` commands, such as `ping`, from anywhere); anyone else gets a refusal.
- **Channel routing:** replies go to the channel the command came from, via `b.reply`/`b.send`. Admin output goes to `Game.AdminChannelID`. Player-facing announcements go to `Game.GameChannelID` (Town Square's text-in-voice chat). Secrets (characters, whispers) go by DM.
- **Parsing:** commands that take free text (`character assign`, `whisper`) parse the raw `message.Content` in `parse.go`, not the `strings.Fields` words, so multi-line text and Markdown survive. Players are named by @mention only.
- **Voice:** the bot never joins voice. Moving members (`GuildMemberMove`) doesn't need it. Don't add `ChannelVoiceJoin` back.
- **Discord roles** (`BoTC-StoryTeller`, `BoTC-Player`) are looked up by exact name. A role failure is a warning in the reply, never a failed command.

## Things to know before changing code

- **Keep the README in step.** Every command is `!botc <command>`. When you add or change a command, update the README's Commands table (and its Features section if the feature's status changes).
- **Game rules are `Game` methods** in `game.go` (`ReplacePlayers`, `RemovePlayer`, `Assign`, `SetTeam`, `SetAlive`, `ToggleGhostVote`, `MarkAnnounced`, `PendingLifeChanges`). They change only the `Game`, never Discord, and are tested in `game_test.go`. Handlers do the Discord side and build replies; put new rules on `Game`, not in a handler.
- **Game state is in-memory only** in `Bot.game` (`nil` = no game registered) and is lost on restart. Only touch it from within command handling, where `Bot.mu` is held; add locking if you ever read it from another handler or goroutine.
- **Adding a command** is one entry in the `commands` table in `commands.go`. By default it's Storyteller-only, from the admin channel, with a game registered; set `beforeGame` or `anyChannel` only when the spec says so, and add a case to `TestAllowed` if the access is new. Handlers need no access check of their own.
- **Incomplete pieces:** `villageCodeLookup` has no "Cottage-XX" entries even though the planned features rely on them.
- **Error handling:** don't `panic`; return errors and reply to the channel. Send replies with `b.reply` (threaded) or `b.send` (plain), which log send failures; don't call `ChannelMessageSend*` directly without checking the error.
- **UML diagrams:** when you change a function listed in the source map in `docs/uml/README.md`, update the affected diagrams in the same change; when you add a command, add its diagram and a source-map row. The steps, including how to validate the Mermaid, are in that README's *Sync procedure*; `/uml-sync` (Claude Code) and the `uml-sync` prompt (Copilot) run it.
- **Logging:** use the standard `log` package. Never log message content: whispers and character guidance are secrets.
- The module path is `github.com/RAGNOARAKNOS/BotOnTheClocktower` (uppercase), even though the local checkout directory is lowercase. Use the module path in imports.

## Roadmap: OBS integration

`docs/roadmap.md` plans OBS control through obs-websocket v5, using `github.com/andreykaipov/goobs`. None of it is built yet. The plan:

- A new `internal/obs` package (`client.go` for the connection and reconnects, `actions.go` for scenes, sources, audio and recording) that must never import `internal/bot`. `Bot` gets an `*obs.Client` field.
- New env vars `OBS_HOST` (default `localhost:4455`), `OBS_PASSWORD`, `OBS_ENABLED` (default `false`), and later `OBS_SCENE_*` and `OBS_ROSTER_SOURCE`. The roadmap also plans a `.env.example` file.
- Commands live under `!botc obs ...` (`ping`, `scene`, `scenes`, `current`, `mute`/`unmute`, `show`/`hide`, `record start|stop|status`).
- OBS is optional. If it's missing, OBS commands log a warning and do nothing, and Discord commands keep working. OBS errors must never crash the bot. Reconnects use exponential backoff (1s doubling, capped at 30s).
- Phases run in order: 1 connect + `obs ping`, 2 scene commands, 3 automatic scene changes on game phase, 4 source/audio/roster overlay, 5 recording.

Where the roadmap doesn't match the current code:

- It says "add OBS config fields to Settings in config.go"; there's no `Settings` any more. Add them to `config.Config` and pass them to the bot from `main`.
- It says to gate commands behind a "Storyteller check (same pattern as `register`)". No per-command check is needed: an `obs` entry in the `commands` table is Storyteller-only, from the admin channel, by default.
- Phase 3 hooks into bedtime/wake commands that don't exist yet.

## CI/CD

- `.github/workflows/release.yml`: on push of a `v*` tag, runs `go test ./...`, cross-compiles linux/windows amd64 binaries (`CGO_ENABLED=0`), and creates a GitHub Release with them.
- `.github/workflows/container.yml`: on a published release, builds the `Dockerfile` and pushes to `ghcr.io/<owner>/botontheclocktower` tagged with the version, `major.minor`, and `latest`.
- `.github/workflows/docs.yml`: on pushes to `main` and PRs touching Go code or `docs/uml/`, renders every Mermaid diagram with mermaid-cli (a syntax error fails the build) and warns if Go code changed without `docs/uml/` changing.
- Go version comes from `go.mod` (1.27.1); the Dockerfile uses `golang:1.27-alpine`. Update both together.
