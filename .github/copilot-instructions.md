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

- `cmd/bot/main.go` — entry point: `config.Load()` then `bot.Run(settings)`.
- `internal/config/config.go` — loads `.env` (a missing file is fine), fills `bot.Settings` with the token and `"UNSET"` placeholders for guild/admin channel/game channel/storyteller IDs.
- `internal/bot/bot.go` — `Settings`, the `Bot` struct, Discord session setup, message handling, and most command implementations.
- `internal/bot/characters.go` — the `character` and `whisper` commands, raw-message parsing (`parseAssignment`, `parseWhisper`) and DM helpers; tests in `characters_test.go`.
- `docs/uml/` — Mermaid UML activity and sequence diagrams of how the code works at runtime, with a source map from Go functions to diagrams.

Note that `config` imports `bot` (for `bot.Settings`), so `bot` must not import `config`.

## Feature specs

Intended behaviour lives in `docs/specs/`, one file per feature (index and workflow in `docs/specs/README.md`, template in `_template.md`). The project README describes what's built; specs describe what's intended.

- Before working on a feature, read its spec. Treat "Open questions" as undecided: ask rather than guess.
- When the user decides something or changes their mind, update the spec: Behaviour/Rules, plus a dated line under Decisions.
- When implementing, tick off "Done when" items; when a feature is finished, set Status to DONE, fill in Implementation, update the project README, and update the UML diagrams.

## How the bot works

- `bot.Run` opens a discordgo session with `IntentsAll`, registers a `Ready` handler and `newMessage`, then blocks until SIGINT/SIGTERM.
- `newMessage` ignores messages from any bot (including itself), splits the content with `strings.Fields`, and dispatches when the first token equals `!botc` (case-insensitive) and there are at least two tokens. The second token, lowercased, is the command.
- `newMessage` holds `Bot.mu` for the whole command, so commands run one at a time (discordgo runs each handler in its own goroutine). It also recovers panics, logging the stack and replying with an error, so a bug doesn't kill the bot and lose the in-memory game. Don't call anything that re-enters the message handler while holding the lock.
- Access (`commandAllowed`, checked in `extractCommand` before the switch, for every command): with no game registered, only `register`/`start` and `ping` run, from any channel, and everything else (including unknown commands) is ignored silently. Once registered, every command must come from `StoryTellerId`, in `AdminChannelId`, in `GuildId`, except `ping`, which only needs to come from `StoryTellerId`; anything else gets a refusal reply. Command handlers don't repeat these checks.
- Commands (`extractCommand` switch):
  - `ping` → replies `pong`
  - `register` / `start` → refused if `GameRegistered` is already true, or if `findVoiceChannelID` can't find a voice channel named `villageCodeLookup["TS"]` (Town Square). Otherwise sets `AdminChannelId` to the message's channel and `GameChannelId` to Town Square, sets `StoryTellerId` to the sender, sets `GameRegistered`, posts a start announcement in the game channel, and gives the sender the `BoTC-StoryTeller` Discord role via `assignStorytellerRole` (looks the role up by exact name via `findRoleID`; role names are the `storytellerRoleName`/`playerRoleName` constants; if that fails, registration still succeeds and the reply includes a warning).
  - `unregister` / `end` → `removeGameRoles` pages through all guild members (`GuildMembers`, 1000 per page) and strips both game roles from anyone who has them, collecting errors with `errors.Join` rather than stopping. Then it posts an end announcement in the game channel, resets settings to `"UNSET"`/false/nil, and replies with the count plus any warnings.
  - `sitrep` → reports the guild, admin/game channels and Storyteller (its "not initialised" branch is unreachable behind `commandAllowed`)
- The bot never joins voice; it has no audio features, and moving members (`GuildMemberMove`) doesn't require it. Don't add `ChannelVoiceJoin` back.
- Channel routing: command replies go to the channel the command came from (so the admin channel, except for `register`, `ping` and refusals); admin output (e.g. `mapRooms`' TTS) goes to `AdminChannelId`; player-facing announcements go to `GameChannelId` (Town Square's text-in-voice chat).
  - `map` → `mapRooms` resolves channel IDs for the names in `villageCodeLookup` (codes `TS`, `CA`, `CF`, `PS`, `TW`, `RS`, `SC`) into `Settings.Rooms`. It doesn't touch players.
  - `village create|add|remove|list` → maintains `Settings.Players` (user ID → display name). `create` replaces the list with everyone in Town Square voice (`GameChannelId`, from the state cache's voice states) except the Storyteller and bots. `add`/`remove` take @mentions (`message.Mentions`). `setPlayerRole` gives or takes `BoTC-Player` to match; role failures only add a warning to the reply. Players dropped from the village lose their stored character.
  - `character assign|team|kill|revive|ghostvote|announce|clear|list|send` → stores `Settings.Characters` (user ID → `*Character{Name, Guidance, Team, Sent, Alive, AnnouncedAlive, GhostVoteUsed}`) for village players. `assign` parses the raw `message.Content` (not `strings.Fields`) so multi-line guidance and Markdown survive: an optional `good`/`evil` word (default Good; `splitTeam`, with `teamWordNames` for "Evil Twin") then the name follow the mention on the first line, and later lines are guidance. `send` DMs unsent characters as embeds (`sendDM`, title includes the team) and marks them sent; `send @player` resends; `team` marks a character unsent. A failed DM leaves the character unsent and is reported.
  - Deaths: `kill`/`revive` only change `Alive`; `AnnouncedAlive` holds the state at the last `announce`, and `pendingLifeChanges` (where they differ) drives `announce` (posts to `GameChannelId`) and the grimoire. `revive` resets `GhostVoteUsed`.
  - `grimoire` is an alias for `character list`: `grimoireSummary` totals plus a `grimoireLine` per player, split into 2000-character messages by `chunkLines`.
  - `whisper @player <text>` → DMs a village player straight away; nothing is stored.
  - `pmove`, `cmove` → empty stubs
  - anything else (reaching the switch) → "Huh? WTF is that command?!"
- Helpers `moveUserToChannel` (uses `GuildMemberMove` with a room code) and `playerNameToId` exist but aren't wired to commands yet.

## Things to know before changing code

- **Keep the README in step.** Every command is `!botc <command>`. When you add or change a command, update the README's Commands table (and its Features section if the feature's status changes).
- **Game state is in-memory only** in `Bot.settings` and is lost on restart. Only touch it from within command handling, where `Bot.mu` is held; add locking if you ever read it from another handler or goroutine.
- **Access is global:** `commandAllowed` restricts every command to the Storyteller in the admin channel, so new commands need no access check of their own. `register`/`start` and `ping` are the only commands that run with no game registered, and `ping` is the only one the Storyteller can send outside the admin channel; add any other exception to `commandAllowed`.
- **Incomplete pieces:** `villageCodeLookup` has no "Cottage-XX" entries even though the planned features rely on them.
- **Error handling:** don't `panic`; return errors and reply to the channel. Many Discord call return values (mostly message sends) are still ignored.
- **UML diagrams:** when you change a function listed in the source map in `docs/uml/README.md`, update the affected diagrams in the same change; when you add a command, add its diagram and a source-map row. The steps, including how to validate the Mermaid, are in that README's *Sync procedure*; `/uml-sync` (Claude Code) and the `uml-sync` prompt (Copilot) run it.
- Lots of `fmt.Print*` debug output — there's no structured logging yet.
- The module path is `github.com/RAGNOARAKNOS/BotOnTheClocktower` (uppercase), even though the local checkout directory is lowercase. Use the module path in imports.

## Roadmap: OBS integration

`roadmap.md` plans OBS control through obs-websocket v5, using `github.com/andreykaipov/goobs`. None of it is built yet. The plan:

- A new `internal/obs` package (`client.go` for the connection and reconnects, `actions.go` for scenes, sources, audio and recording) that must never import `internal/bot`. `Bot` gets an `*obs.Client` field.
- New env vars `OBS_HOST` (default `localhost:4455`), `OBS_PASSWORD`, `OBS_ENABLED` (default `false`), and later `OBS_SCENE_*` and `OBS_ROSTER_SOURCE`. The roadmap also plans a `.env.example` file.
- Commands live under `!botc obs ...` (`ping`, `scene`, `scenes`, `current`, `mute`/`unmute`, `show`/`hide`, `record start|stop|status`).
- OBS is optional. If it's missing, OBS commands log a warning and do nothing, and Discord commands keep working. OBS errors must never crash the bot. Reconnects use exponential backoff (1s doubling, capped at 30s).
- Phases run in order: 1 connect + `obs ping`, 2 scene commands, 3 automatic scene changes on game phase, 4 source/audio/roster overlay, 5 recording.

Where the roadmap doesn't match the current code:

- It says "add OBS config fields to Settings in config.go", but `Settings` is defined in `internal/bot`.
- It says to gate commands behind a "Storyteller check (same pattern as `register`)", but `register` has no such check. No per-command check is needed: `commandAllowed` already restricts every command, including `obs` ones, to the Storyteller in the admin channel.
- Phase 3 hooks into bedtime/wake commands that don't exist yet.

## CI/CD

- `.github/workflows/release.yml`: on push of a `v*` tag, runs `go test ./...`, cross-compiles linux/windows amd64 binaries (`CGO_ENABLED=0`), and creates a GitHub Release with them.
- `.github/workflows/container.yml`: on a published release, builds the `Dockerfile` and pushes to `ghcr.io/<owner>/botontheclocktower` tagged with the version, `major.minor`, and `latest`.
- `.github/workflows/docs.yml`: on pushes to `main` and PRs touching Go code or `docs/uml/`, renders every Mermaid diagram with mermaid-cli (a syntax error fails the build) and warns if Go code changed without `docs/uml/` changing.
- Go version comes from `go.mod` (1.27.1); the Dockerfile uses `golang:1.27-alpine`. Update both together.
