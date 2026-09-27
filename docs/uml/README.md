# UML diagrams

How the bot works at runtime, drawn as UML **activity diagrams** (the steps and decisions inside one piece of code) and **sequence diagrams** (who calls whom, and in what order: the Storyteller, the bot, the Discord API and the players).

These diagrams describe the code as it is now. For what the bot *does*, see the [project README](../../README.md). For what features are *meant* to do, including features not built yet, see the [feature specs](../specs/README.md). Planned features (OBS) have no diagrams until they're built.

All diagrams are written in [Mermaid](https://mermaid.js.org/), which GitHub renders directly in Markdown. To preview them locally, use a Markdown previewer with Mermaid support, e.g. the VS Code extension *Markdown Preview Mermaid Support*.

## Diagrams

| File | Activity diagrams | Sequence diagrams |
| --- | --- | --- |
| [startup.md](startup.md) | Startup and shutdown | Startup and shutdown |
| [command-dispatch.md](command-dispatch.md) | Handling a message, `allowed` | Handling a message |
| [game-lifecycle.md](game-lifecycle.md) | `register`, `unregister` | `register`, `unregister`, `map`, `sitrep` |
| [village.md](village.md) | `village` dispatch, `create`, `add`, `remove` | `village create` |
| [characters.md](characters.md) | `character assign`, `kill`/`revive`, `ghostvote`, `team`, `announce` | `announce`, `send`, `grimoire`, `whisper` |
| [gather.md](gather.md) | `gather` | `gather` countdown, `gather cancel` |

## Notation

Mermaid has no dedicated UML activity diagram, so activity diagrams are flowcharts drawn with UML's notation:

| Element | Drawn as |
| --- | --- |
| Initial node | Small filled circle |
| Activity final node | Filled double circle. An activity can have several. |
| Action | Rounded rectangle, named after the Go function or the user-visible reply |
| Decision / merge | Diamond. The conditions are on the outgoing edges. |
| Guard | Condition in square brackets on an edge, e.g. `[not the Storyteller]` |

Activity diagrams don't use swimlanes, because Mermaid can't keep subgraphs in lane order and the result is hard to read. Each activity diagram follows one Go function. The sequence diagram next to it shows which party (Storyteller, bot, Discord, player) does each step.

Sequence diagrams use UML's usual notation:

| Element | Meaning |
| --- | --- |
| Stick figure (actor) | A person: the Storyteller or a player |
| Solid arrow | A call or request, e.g. a Discord REST call |
| Dashed arrow | A reply or return value |
| Open arrow (`-)`) | An asynchronous event, e.g. discordgo calling a handler in its own goroutine |
| `alt` / `else` | Alternatives: exactly one branch runs |
| `opt` | Runs only if its condition holds |
| `loop` | Repeats, e.g. once per player |
| `critical` | Runs while holding `Bot.mu` |
| Note | Internal state changes, and the Go function that does the work |

Participants in the sequence diagrams:

- **Discord Gateway**: the websocket that delivers events such as `MessageCreate` to the bot.
- **discordgo State cache**: discordgo's in-memory copy of guilds, members and voice states.
- **Discord REST API**: every call the bot makes, such as `ChannelMessageSend`, `GuildRoles`, `GuildMemberRoleAdd` or `UserChannelCreate`.

Replies to a command always go to the channel the command came from. Apart from `register` and `ping`, every command only runs for the Storyteller in the admin channel (`allowed`), so that's where replies go, apart from `register`, `ping` and refusals.

## Source map

Which diagram covers which code. When you change one of these functions, update the diagrams listed next to it in the same change. The `/uml-sync` command (Claude Code) and the `uml-sync` prompt (Copilot) use this table to find diagrams that are out of date.

| Code | Diagram |
| --- | --- |
| `cmd/bot/main.go` `main`, `config.Load`, `bot.Run` | [startup.md](startup.md) |
| `newMessage`, `extractCommand`, `commands` | [command-dispatch.md](command-dispatch.md) |
| `allowed` | [command-dispatch.md](command-dispatch.md) |
| `mapCommand` | [game-lifecycle.md](game-lifecycle.md) |
| `register`, `newGame`, `findVoiceChannelID`, `assignStorytellerRole`, `findRoleID`, `channelAccessWarning` | [game-lifecycle.md](game-lifecycle.md) |
| `unregister`, `removeGameRoles` | [game-lifecycle.md](game-lifecycle.md) |
| `mapRooms`, `getMapGuildChannels`, `villageCodeLookup` | [game-lifecycle.md](game-lifecycle.md) |
| `sitrep` | [game-lifecycle.md](game-lifecycle.md) |
| `village`, `villageCreate`, `villageAdd`, `villageRemove`, `Game.ReplacePlayers`, `Game.RemovePlayer` | [village.md](village.md) |
| `setPlayerRole`, `replyWithRoleWarning`, `lookupMember`, `memberDisplayName` | [village.md](village.md) |
| `character`, `characterAssign`, `parseAssignment`, `splitAtMention`, `splitTeam`, `parseTeam`, `teamWordNames` | [characters.md](characters.md) |
| `characterSetAlive`, `characterGhostVote`, `characterTeam`, `mentionedCharacters`, `Game.Assign`, `Game.SetAlive`, `Game.ToggleGhostVote`, `Game.SetTeam` | [characters.md](characters.md) |
| `characterAnnounce`, `pendingLifeChanges`, `Game.PendingLifeChanges`, `Game.MarkAnnounced` | [characters.md](characters.md) |
| `characterSend`, `sendDM`, `dmEmbed`, `dmErrorReason` | [characters.md](characters.md) |
| `characterList`, `grimoireSummary`, `grimoireLine`, `chunkLines` | [characters.md](characters.md) |
| `whisper`, `parseWhisper` | [characters.md](characters.md) |
| `gather`, `parseGather`, `gatherCancel`, `gatherFire`, `gatherWarn`, `gatherMove`, `gatherAnnounce`, `gatherCountdown`, `Game.villageChannels` | [gather.md](gather.md) |

**Not diagrammed:** these are too simple to need a diagram, or aren't used yet. If one of them becomes part of a command's flow, add it to the table above.

- `villageList`, `characterClear`: a single loop and a reply.
- `sortedNames`, `sortedPlayerIDs`, `playerName`, `ghostVoteState`, `formatCountdown`, `plural`, `failureList`: formatting helpers.
- `reply`, `send`: send a message and log any failure. `missingPermissions`, `isVillageRoom`, `botUserID`: small helpers for `channelAccessWarning`. `ping`: sends `pong`.

## Keeping the diagrams up to date

- Change a function in the source map → update its diagrams in the same commit.
- Add a command → add its diagrams (to an existing file, or a new file listed under [Diagrams](#diagrams)) and add a row to the source map.
- Each diagram file ends with a `Last checked against code:` line giving a date and commit.
- To find and fix stale diagrams, follow the [sync procedure](#sync-procedure). `/uml-sync [git ref]` in Claude Code, and the `uml-sync` prompt in Copilot Chat, run it for you.
- CI ([docs.yml](../../.github/workflows/docs.yml)) renders every diagram, so a Mermaid syntax error fails the build. It also warns when Go code changes without any change to `docs/uml/`.

### Sync procedure

1. **Choose the base ref.** Use the ref you were given. Otherwise use the oldest commit named on the `Last checked against code:` lines of `docs/uml/*.md`, or `main` if none has one.
2. **Find the changed code.** Run `git diff --name-only <base> -- '*.go' ':!*_test.go'`. This compares with the working tree, so uncommitted changes are included. If nothing changed, the diagrams are current and you can stop.
3. **List what changed.** For each changed file, read `git diff <base> -- <file>` and list the functions added, removed, renamed or changed. Include package-level values that drive a flow, such as `villageCodeLookup` or `teamWordNames`.
4. **Find the affected diagrams.** Look each item up in the [source map](#source-map). Read every affected diagram file and the current code of every function it covers.
5. **Update the diagrams.** Change only what the code change affects: new or removed branches, guards, Discord calls, state changes and reply text. Follow the [notation](#notation). A change that doesn't alter a flow (a renamed local variable, debug output) needs no diagram change.
6. **Deal with code missing from the source map.** A new command, or a function that is now part of a command's flow, gets a diagram (in the file that fits, or in a new file added to [Diagrams](#diagrams)) and a source-map row. A trivial helper goes on the *Not diagrammed* list. If a function was removed or renamed, update or remove its rows.
7. **Update `Last checked against code:`** in every diagram file you reviewed: today's date and `git rev-parse --short HEAD`.
8. **Validate.** Render each file you changed, in Docker (preferred, so mermaid-cli isn't installed on the host):
   - `docker run --rm -v "$PWD/docs/uml:/data" minlag/mermaid-cli -i /data/<file>.md -o /tmp/<file>.md` (on Git Bash for Windows, put `MSYS_NO_PATHCONV=1` in front).
   - Only if Docker isn't available and the user agrees: `npx -y @mermaid-js/mermaid-cli -i docs/uml/<file>.md -o <temp dir>/<file>.md`.

   Fix any parse errors. Common causes: a node ID called `end`, double quotes or `<`/`>` inside a label, or a `;` in a sequence-diagram message.
9. **Keep the other docs in step.** If a command was added or removed, check that the project README's Commands table and `.github/copilot-instructions.md` agree.
10. **Report** which diagrams changed and why, which rows were added to the source map, and anything you weren't sure about.
