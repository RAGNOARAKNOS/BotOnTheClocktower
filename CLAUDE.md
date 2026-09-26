# CLAUDE.md

Guidance for Claude Code when working in this repository.

> **Keep in sync with [.github/copilot-instructions.md](.github/copilot-instructions.md).** This file and the Copilot instructions share the same project guidance. Whenever you change one, check the other and apply the equivalent update so they don't drift apart. Review both at the start of any session that touches project structure, commands, workflow, or conventions.

## What this is

A Discord bot (Go, [discordgo](https://github.com/bwmarrin/discordgo)) that helps the Storyteller run a game of *Blood on the Clocktower* over Discord — mainly by moving players between voice channels ("Town Square", cottages, etc.) as the game switches between DAY and NIGHT phases. It's an early-stage personal project; most features in the README are PLANNED or IN WORK.

## Commands

```shell
go run ./cmd/bot                                   # run locally (needs BOTAPIKEY)
go build -o BotOnTheClocktower.exe ./cmd/bot       # build a binary
go test ./...                                      # tests (none exist yet)
go vet ./...
docker build -t botontheclocktower .               # container image (distroless, nonroot)
```

Configuration: `BOTAPIKEY` (Discord bot token), read from the environment or from an optional `.env` file in the working directory. `.env` is gitignored — never commit it or print the token.

## Layout

- [cmd/bot/main.go](cmd/bot/main.go) — entry point: `config.Load()` then `bot.Run(settings)`.
- [internal/config/config.go](internal/config/config.go) — loads `.env` (a missing file is fine), fills `bot.Settings` with the token and `"UNSET"` placeholders for guild/admin channel/game channel/storyteller IDs.
- [internal/bot/bot.go](internal/bot/bot.go) — everything else: `Settings`, the `Bot` struct, Discord session setup, message handling, and command implementations.

Note that `config` imports `bot` (for `bot.Settings`), so `bot` must not import `config`.

## Feature specs

Intended behaviour lives in [docs/specs/](docs/specs/), one file per feature (index and workflow in [docs/specs/README.md](docs/specs/README.md), template in `_template.md`). The project README describes what's built; specs describe what's intended.

- Before working on a feature, read its spec. Treat "Open questions" as undecided: ask rather than guess.
- When the user decides something or changes their mind, update the spec: Behaviour/Rules, plus a dated line under Decisions.
- When implementing, tick off "Done when" items; when a feature is finished, set Status to DONE, fill in Implementation, and update the project README.

## How the bot works

- `bot.Run` opens a discordgo session with `IntentsAll`, registers a `Ready` handler and `newMessage`, then blocks until SIGINT/SIGTERM.
- `newMessage` ignores messages from any bot (including itself), splits the content with `strings.Fields`, and dispatches when the first token equals `!botc` (case-insensitive) and there are at least two tokens. The second token, lowercased, is the command.
- `newMessage` holds `Bot.mu` for the whole command, so commands run one at a time (discordgo runs each handler in its own goroutine). It also recovers panics, logging the stack and replying with an error, so a bug doesn't kill the bot and lose the in-memory game. Don't call anything that re-enters the message handler while holding the lock.
- Commands (`extractCommand` switch):
  - `ping` → replies `pong`
  - `register` / `start` → refused if `GameRegistered` is already true, or if `findVoiceChannelID` can't find a voice channel named `villageCodeLookup["TS"]` (Town Square). Otherwise sets `AdminChannelId` to the message's channel and `GameChannelId` to Town Square, sets `StoryTellerId` to the sender, sets `GameRegistered`, posts a start announcement in the game channel, and gives the sender the `BoTC-StoryTeller` Discord role via `assignStorytellerRole` (looks the role up by exact name via `findRoleID`; role names are the `storytellerRoleName`/`playerRoleName` constants; if that fails, registration still succeeds and the reply includes a warning).
  - `unregister` / `end` → Storyteller only, from the registered guild. `removeGameRoles` pages through all guild members (`GuildMembers`, 1000 per page) and strips both game roles from anyone who has them, collecting errors with `errors.Join` rather than stopping. Then it posts an end announcement in the game channel, resets settings to `"UNSET"`/false/nil, and replies with the count plus any warnings.
  - `sitrep` → reports whether a game is registered, plus the admin/game channels and Storyteller
- The bot never joins voice; it has no audio features, and moving members (`GuildMemberMove`) doesn't require it. Don't add `ChannelVoiceJoin` back.
- Channel routing: command replies go to the channel the command came from; admin output (e.g. `mapRooms`' TTS) goes to `AdminChannelId`; player-facing announcements go to `GameChannelId` (Town Square's text-in-voice chat).
  - `map` → (requires `register` first) `mapRooms` resolves channel IDs for the names in `villageCodeLookup` (codes `TS`, `CA`, `CF`, `PS`, `TW`, `RS`, `SC`) into `Settings.Rooms`. It doesn't touch players.
  - `village create|add|remove|list` → Storyteller only, from the admin channel (`requireStorytellerInAdmin`). Maintains `Settings.Players` (user ID → display name). `create` replaces the list with everyone in Town Square voice (`GameChannelId`, from the state cache's voice states) except the Storyteller and bots. `add`/`remove` take @mentions (`message.Mentions`). `setPlayerRole` gives or takes `BoTC-Player` to match; role failures only add a warning to the reply.
  - `pmove`, `cmove` → empty stubs
  - anything else → "Huh? WTF is that command?!"
- Helpers `moveUserToChannel` (uses `GuildMemberMove` with a room code) and `playerNameToId` exist but aren't wired to commands yet.

## Things to know before changing code

- **Command syntax differs from the README.** The README documents `!gather` / `!bedtime`; the code actually expects `!botc <command>`. Keep them in sync when adding commands.
- **Game state is in-memory only** in `Bot.settings` and is lost on restart. Only touch it from within command handling, where `Bot.mu` is held; add locking if you ever read it from another handler or goroutine.
- **Incomplete pieces:** only `unregister` and `village` are restricted to the Storyteller; `villageCodeLookup` has no "Cottage-XX" entries even though the planned features rely on them.
- **Error handling:** don't `panic`; return errors and reply to the channel. Many Discord call return values (mostly message sends) are still ignored.
- Lots of `fmt.Print*` debug output — there's no structured logging yet.
- The module path is `github.com/RAGNOARAKNOS/BotOnTheClocktower` (uppercase), even though the local checkout directory is lowercase. Use the module path in imports.

## Roadmap: OBS integration

[roadmap.md](roadmap.md) plans OBS control through obs-websocket v5, using `github.com/andreykaipov/goobs`. None of it is built yet. The plan:

- A new `internal/obs` package (`client.go` for the connection and reconnects, `actions.go` for scenes, sources, audio and recording) that must never import `internal/bot`. `Bot` gets an `*obs.Client` field.
- New env vars `OBS_HOST` (default `localhost:4455`), `OBS_PASSWORD`, `OBS_ENABLED` (default `false`), and later `OBS_SCENE_*` and `OBS_ROSTER_SOURCE`. The roadmap also plans a `.env.example` file.
- Commands live under `!botc obs ...` (`ping`, `scene`, `scenes`, `current`, `mute`/`unmute`, `show`/`hide`, `record start|stop|status`).
- OBS is optional. If it's missing, OBS commands log a warning and do nothing, and Discord commands keep working. OBS errors must never crash the bot. Reconnects use exponential backoff (1s doubling, capped at 30s).
- Phases run in order: 1 connect + `obs ping`, 2 scene commands, 3 automatic scene changes on game phase, 4 source/audio/roster overlay, 5 recording.

Where the roadmap doesn't match the current code:

- It says "add OBS config fields to Settings in config.go", but `Settings` is defined in `internal/bot`.
- It says to gate commands behind a "Storyteller check (same pattern as `register`)", but `register` has no such check. Use `requireStorytellerInAdmin` (used by `village`) or the inline check in `unregister`.
- Phase 3 hooks into bedtime/wake commands that don't exist yet.

## CI/CD

- [.github/workflows/release.yml](.github/workflows/release.yml): on push of a `v*` tag, runs `go test ./...`, cross-compiles linux/windows amd64 binaries (`CGO_ENABLED=0`), and creates a GitHub Release with them.
- [.github/workflows/container.yml](.github/workflows/container.yml): on a published release, builds the [Dockerfile](Dockerfile) and pushes to `ghcr.io/<owner>/botontheclocktower` tagged with the version, `major.minor`, and `latest`.
- Go version comes from `go.mod` (1.27.1); the Dockerfile uses `golang:1.27-alpine`. Update both together.
