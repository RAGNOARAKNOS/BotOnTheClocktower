# UML diagrams

How the bot works at runtime, drawn as UML **activity diagrams** (the steps and decisions inside one piece of code) and **sequence diagrams** (who calls whom, and in what order: the Storyteller, the bot, the Discord API and the players).

These diagrams describe the code as it is now. For what the bot *does*, see the [project README](../../README.md). For what features are *meant* to do, including features not built yet, see the [feature specs](../specs/README.md). Planned features (`gather`, `bedtime`, OBS) have no diagrams until they're built.

All diagrams are written in [Mermaid](https://mermaid.js.org/), which GitHub renders directly in Markdown. To preview them locally, use a Markdown previewer with Mermaid support, e.g. the VS Code extension *Markdown Preview Mermaid Support*.

## Diagrams

| File | Activity diagrams | Sequence diagrams |
| --- | --- | --- |
| [startup.md](startup.md) | Startup and shutdown | Startup and shutdown |
| [command-dispatch.md](command-dispatch.md) | Handling a message, `requireStorytellerInAdmin` | Handling a message |
| [game-lifecycle.md](game-lifecycle.md) | `register`, `unregister` | `register`, `unregister`, `map`, `sitrep` |
| [village.md](village.md) | `village` dispatch, `create`, `add`, `remove` | `village create` |
| [characters.md](characters.md) | `character assign`, `kill`/`revive`, `ghostvote`, `team`, `announce` | `announce`, `send`, `grimoire`, `whisper` |

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

Replies to a command always go to the channel the command came from. For the Storyteller-only commands, that's the admin channel.

## Source map

Which diagram covers which code. When you change one of these functions, update the diagrams listed next to it in the same change. The `/uml-sync` command (Claude Code) and the `uml-sync` prompt (Copilot) use this table to find diagrams that are out of date.

| Code | Diagram |
| --- | --- |
| `cmd/bot/main.go` `main`, `config.Load`, `bot.Run` | [startup.md](startup.md) |
| `newMessage`, `extractCommand` | [command-dispatch.md](command-dispatch.md) |
| `requireStorytellerInAdmin` | [command-dispatch.md](command-dispatch.md) |
| `register`, `findVoiceChannelID`, `assignStorytellerRole`, `findRoleID` | [game-lifecycle.md](game-lifecycle.md) |
| `unregister`, `removeGameRoles` | [game-lifecycle.md](game-lifecycle.md) |
| `mapRooms`, `getMapGuildChannels`, `villageCodeLookup` | [game-lifecycle.md](game-lifecycle.md) |
| `sitrep` | [game-lifecycle.md](game-lifecycle.md) |
| `village`, `villageCreate`, `villageAdd`, `villageRemove` | [village.md](village.md) |
| `setPlayerRole`, `replyWithRoleWarning`, `lookupMember`, `memberDisplayName` | [village.md](village.md) |
| `character`, `characterAssign`, `parseAssignment`, `splitAtMention`, `splitTeam`, `parseTeam`, `teamWordNames` | [characters.md](characters.md) |
| `characterSetAlive`, `characterGhostVote`, `characterTeam`, `mentionedCharacters` | [characters.md](characters.md) |
| `characterAnnounce`, `pendingLifeChanges` | [characters.md](characters.md) |
| `characterSend`, `sendDM`, `dmEmbed`, `dmErrorReason` | [characters.md](characters.md) |
| `characterList`, `grimoireSummary`, `grimoireLine`, `chunkLines` | [characters.md](characters.md) |
| `whisper`, `parseWhisper` | [characters.md](characters.md) |

**Not diagrammed:** these are too simple to need a diagram, or aren't used yet. If one of them becomes part of a command's flow, add it to the table above.

- `villageList`, `characterClear`: a single loop and a reply.
- `sortedNames`, `sortedPlayerIDs`, `playerName`, `ghostVoteState`: formatting helpers.
- `moveUserToChannel`, `playerNameToId`: not used by any command yet.

## Keeping the diagrams up to date

- Change a function in the source map → update its diagrams in the same commit.
- Add a command → add its diagrams (to an existing file, or a new file listed under [Diagrams](#diagrams)) and add a row to the source map.
- Each diagram file ends with a `Last checked against code:` line giving a date and commit. `/uml-sync` updates this line after checking the file.
- Run `/uml-sync` (Claude Code) or the `uml-sync` prompt (Copilot Chat) to find and fix stale diagrams. Pass a git ref to compare against, e.g. `/uml-sync main`.
- CI ([docs.yml](../../.github/workflows/docs.yml)) renders every diagram, so a Mermaid syntax error fails the build. It also warns when Go code changes without any change to `docs/uml/`.
