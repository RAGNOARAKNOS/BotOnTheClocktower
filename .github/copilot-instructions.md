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

- `bot.Run` opens a discordgo session with `IntentsAll`, registers a `Ready` handler and `newMessage`, then blocks until SIGINT/SIGTERM.
- `newMessage` ignores messages from any bot (including itself), splits the content with `strings.Fields`, and dispatches when the first token equals `!botc` (case-insensitive) and there are at least two tokens. The second token, lowercased, is the command.
- `newMessage` holds `Bot.mu` for the whole command, so commands run one at a time (discordgo runs each handler in its own goroutine). It also recovers panics, logging the stack and replying with an error, so a bug doesn't kill the bot and lose the in-memory game. Don't call anything that re-enters the message handler while holding the lock.
- Access (`allowed`, a pure function checked in `extractCommand` for every command, tested in `commands_test.go`): with no game registered, only commands marked `beforeGame` (`register`/`start`, `ping`) run, from any channel, and everything else (including unknown commands) is ignored silently. Once registered, every command must come from `StorytellerID`, in `AdminChannelID`, in `GuildID`; commands marked `anyChannel` (`ping`) only need `StorytellerID`. Anything else gets a refusal reply. Command handlers don't repeat these checks.
- Commands (entries in the `commands` table; every handler has the signature `func(b *Bot, message *discordgo.MessageCreate, words []string)`):
  - `ping` → replies `pong`
  - `register` / `start` → refused if `b.game` is already set, or if `findVoiceChannelID` can't find a voice channel named `villageCodeLookup["TS"]` (Town Square). Otherwise creates the game with `newGame` (admin channel = the message's channel, game channel = Town Square, Storyteller = the sender), posts a start announcement in the game channel, and gives the sender the `BoTC-StoryTeller` Discord role via `assignStorytellerRole` (looks the role up by exact name via `findRoleID`; role names are the `storytellerRoleName`/`playerRoleName` constants; if that fails, registration still succeeds and the reply includes a warning).
  - `unregister` / `end` → `removeGameRoles` pages through all guild members (`GuildMembers`, 1000 per page) and strips both game roles from anyone who has them, collecting errors with `errors.Join` rather than stopping. Then it posts an end announcement in the game channel, sets `b.game` to nil, and replies with the count plus any warnings.
  - `sitrep` → reports the guild, admin/game channels and Storyteller
- The bot never joins voice; it has no audio features, and moving members (`GuildMemberMove`) doesn't require it. Don't add `ChannelVoiceJoin` back.
- Channel routing: command replies go to the channel the command came from (so the admin channel, except for `register`, `ping` and refusals); admin output (e.g. `mapRooms`' TTS) goes to `AdminChannelID`; player-facing announcements go to `GameChannelID` (Town Square's text-in-voice chat).
  - `map` → `mapRooms` resolves channel IDs for the names in `villageCodeLookup` (codes `TS`, `CA`, `CF`, `PS`, `TW`, `RS`, `SC`) into `Game.Rooms`. It doesn't touch players.
  - `village create|add|remove|list` → maintains `Game.Players` (user ID → display name). `create` replaces the list with everyone in Town Square voice (`GameChannelID`, from the state cache's voice states) except the Storyteller and bots. `add`/`remove` take @mentions (`message.Mentions`). `setPlayerRole` gives or takes `BoTC-Player` to match; role failures only add a warning to the reply. Players dropped from the village lose their stored character.
  - `character assign|team|kill|revive|ghostvote|announce|clear|list|send` → stores `Game.Characters` (user ID → `*Character{Name, Guidance, Team, Sent, Alive, AnnouncedAlive, GhostVoteUsed}`) for village players. `assign` parses the raw `message.Content` (not `strings.Fields`) so multi-line guidance and Markdown survive: an optional `good`/`evil` word (default Good; `splitTeam`, with `teamWordNames` for "Evil Twin") then the name follow the mention on the first line, and later lines are guidance. `send` DMs unsent characters as embeds (`sendDM`, title includes the team) and marks them sent; `send @player` resends; `team` marks a character unsent. A failed DM leaves the character unsent and is reported.
  - Deaths: `kill`/`revive` only change `Alive`; `AnnouncedAlive` holds the state at the last `announce`, and `pendingLifeChanges` (where they differ) drives `announce` (posts to `GameChannelID`) and the grimoire. `revive` resets `GhostVoteUsed`.
  - `grimoire` is an alias for `character list`: `grimoireSummary` totals plus a `grimoireLine` per player, split into 2000-character messages by `chunkLines`.
  - `whisper @player <text>` → DMs a village player straight away; nothing is stored.
  - anything else (reaching the switch) → "Huh? WTF is that command?!"

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

`roadmap.md` plans OBS control through obs-websocket v5, using `github.com/andreykaipov/goobs`. None of it is built yet. The plan:

- A new `internal/obs` package (`client.go` for the connection and reconnects, `actions.go` for scenes, sources, audio and recording) that must never import `internal/bot`. `Bot` gets an `*obs.Client` field.
- New env vars `OBS_HOST` (default `localhost:4455`), `OBS_PASSWORD`, `OBS_ENABLED` (default `false`), and later `OBS_SCENE_*` and `OBS_ROSTER_SOURCE`. The roadmap also plans a `.env.example` file.
- Commands live under `!botc obs ...` (`ping`, `scene`, `scenes`, `current`, `mute`/`unmute`, `show`/`hide`, `record start|stop|status`).
- OBS is optional. If it's missing, OBS commands log a warning and do nothing, and Discord commands keep working. OBS errors must never crash the bot. Reconnects use exponential backoff (1s doubling, capped at 30s).
- Phases run in order: 1 connect + `obs ping`, 2 scene commands, 3 automatic scene changes on game phase, 4 source/audio/roster overlay, 5 recording.

Where the roadmap doesn't match the current code:

- It says "add OBS config fields to Settings in config.go"; there's no `Settings` any more. Add them to `config.Config` and pass them to the bot from `main`.
- It says to gate commands behind a "Storyteller check (same pattern as `register`)", but `register` has no such check. No per-command check is needed: an `obs` entry in the `commands` table is Storyteller-only, from the admin channel, by default.
- Phase 3 hooks into bedtime/wake commands that don't exist yet.

## CI/CD

- `.github/workflows/release.yml`: on push of a `v*` tag, runs `go test ./...`, cross-compiles linux/windows amd64 binaries (`CGO_ENABLED=0`), and creates a GitHub Release with them.
- `.github/workflows/container.yml`: on a published release, builds the `Dockerfile` and pushes to `ghcr.io/<owner>/botontheclocktower` tagged with the version, `major.minor`, and `latest`.
- `.github/workflows/docs.yml`: on pushes to `main` and PRs touching Go code or `docs/uml/`, renders every Mermaid diagram with mermaid-cli (a syntax error fails the build) and warns if Go code changed without `docs/uml/` changing.
- Go version comes from `go.mod` (1.27.1); the Dockerfile uses `golang:1.27-alpine`. Update both together.
