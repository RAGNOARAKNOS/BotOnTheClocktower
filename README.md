# BotOnTheClocktower

<https://0x2142.com/how-to-discordgo-bot/>

<https://medium.com/@mssandeepkamath/building-a-simple-discord-bot-using-go-12bfca31ad5d>

<https://dev.to/aurelievache/learning-go-by-examples-part-4-create-a-bot-for-discord-in-go-43cf>

<https://github.com/scraly/learning-go-by-examples/tree/main/go-gopher-bot-discord>

## Purpose

This bot is designed to facilitate the needs of the Storyteller during a game of "Blood on the Clocktower" via Discord.  

This bot will provide chat commands that can be executed by the Storyteller to manage the Players throughout the relevant phases of play.

## Disclaimer

This is a personal project, and is no way affiliated with "The Pandemonium Institute" who created "Blood On The Clocktower".  I love the game, and recommend you use the official app or better yet buy a physical copy [of the game here](https://bloodontheclocktower.com/).

## Notable libraries used

1. <https://github.com/bwmarrin/discordgo> - A low-level Go wrapper to the Discord API
1. <https://github.com/joho/godotenv> - Provides a framework for defining and consuming application parameters in a lightweight config file.

## Running the Bot

### Discord Setup

Register the bot application within your Discord developer page [on the dev page](https://discord.com/developers/applications) and make a note of your Discord BOT API token (in the BOT page).

The bot requests all gateway intents, so on the BOT page you must enable all three **Privileged Gateway Intents** (Presence, Server Members and Message Content). If you don't, Discord refuses the connection, and without Message Content the bot can't read commands.

### Server Setup (for Discord moderators)

These steps need someone with the **Manage Server** and **Manage Roles** permissions on the Discord server.

#### 1. Invite the bot with these permissions

In the developer portal, open **OAuth2 → URL Generator**, tick the `bot` scope, then tick these permissions and use the generated link to invite the bot:

| Permission | Why the bot needs it |
| --- | --- |
| View Channels | See the game's text and voice channels |
| Send Messages | Reply to commands, and post game announcements in Town Square's text chat |
| Send TTS Messages | Announce "Town Locations Mapped" in the admin channel after `!botc map` |
| Read Message History | Reply directly to the command message |
| Connect | Needed alongside Move Members: Discord only lets the bot move someone into a voice channel it could connect to itself. The bot never joins voice |
| Manage Roles | Give the `BoTC-StoryTeller` role on `!botc register`, give and take `BoTC-Player` with the `!botc village` commands, and remove the game roles on `!botc unregister` |
| Move Members | Needed for the planned commands that move players between voice channels |

When the bot joins, Discord automatically creates a role with the bot's name that holds these permissions. Don't delete it. These are server-wide permissions: if your game channels are private, the bot also needs access to them (step 4).

#### 2. Check the game roles exist

The bot uses two roles that must already exist on the server. It doesn't create them; it looks them up by name, and the names must match exactly, including capitals and the hyphen:

| Role | Purpose |
| --- | --- |
| `BoTC-StoryTeller` | Given to whoever runs `!botc register`, removed on `!botc unregister` |
| `BoTC-Player` | Marks players in the village. Given and taken by `!botc village create`/`add`/`remove`; removed from everyone on `!botc unregister` |

The bot doesn't need either role to have any permissions. What you give them is up to you. Useful options for `BoTC-StoryTeller` are Move Members, Mute Members, Deafen Members and Priority Speaker.

#### 3. Put the roles in the right order

Discord only lets a bot give out roles that sit **below its own highest role**. In **Server Settings → Roles**, drag the roles into this order (top of the list = highest):

```text
Admin / Moderator roles     <- keep these above the bot
BotOnTheClocktower          <- the bot's own role
BoTC-StoryTeller            <- must be below the bot's role
BoTC-Player                 <- must be below the bot's role
Other member roles
@everyone
```

- If `BoTC-StoryTeller` is above the bot's role, registration still succeeds, but the bot can't give out the role. It replies with a warning instead.
- Likewise, if `BoTC-Player` is above the bot's role, the `!botc village` commands still update the player list but reply with a warning, and `!botc unregister` still ends the game but can't remove that role and replies with a warning.
- Manage Roles lets the bot give out **any** role below its own. Keep moderator and admin roles above the bot's role so it can never hand them out.

#### 4. Give the bot access to the game channels

Server-wide permissions aren't enough if the game channels are private. A channel or category that hides itself from `@everyone` also hides itself from the bot, unless the bot is let in. Discord doesn't send a bot messages from channels it can't view, so **the bot silently ignores commands there**, even `!botc ping`.

The simplest reliable setup is one category for the whole game, with a permission override for the bot's role:

1. Put every game channel in one category: the admin text channel (where the Storyteller sends `!botc register`, e.g. `botc-storyteller-notes`), any player text channel (e.g. `botc-chat`), and the village voice channels (`Town Square`, `Cathedral`, `Campfire`, `Potion Shop`, `Tower`, `Riverside`, `Storyteller's Corner`).
2. Open the category's **Edit Category → Permissions**, add the **`BotOnTheClocktower`** role, and allow:

   | Permission | Why |
   | --- | --- |
   | View Channels | See commands and the channels it posts in |
   | Send Messages | Reply to commands and post announcements |
   | Read Message History | Reply to a command as a threaded reply |
   | Connect | Required alongside Move Members for voice channels |
   | Move Members | Move players between voice channels (planned commands) |

3. In each channel's **Permissions** tab, check it says **Synced with category**. If a channel isn't synced, click **Sync Now**, or add the same override to that channel by hand.

Use the bot's **role**, not the bot's user, in the override, so it keeps working if the bot is removed and re-invited. New channels added to the category inherit the override automatically. Don't give the bot Administrator: it would work, but it's far more access than the bot needs.

`!botc register` checks these permissions and lists any that are missing, channel by channel.

**If the bot doesn't respond at all:** send `!botc ping` in a channel the bot can certainly see (e.g. a public text channel). If it answers there but not in a game channel, that channel is hiding itself from the bot; fix its permissions as above. If it answers nowhere, check the bot is running and that it's in the server.

#### Who becomes Storyteller

- Whoever sends `!botc register` becomes the Storyteller for that game and is given the `BoTC-StoryTeller` role.
- The bot runs one game at a time. Once a game is registered, `!botc register` is refused for everyone, including the current Storyteller.
- To end the game, the Storyteller sends `!botc unregister` (or `!botc end`). This removes `BoTC-StoryTeller` and `BoTC-Player` from every member who has them, not just the ones the bot gave out. After that, anyone can register a new game.
- Only the Storyteller can end the game, or run any other command, and only from the admin channel. If they're unavailable, restart the bot and have a moderator remove the game roles by hand.

### Run the executable

Download the binary for your platform (Linux or Windows, amd64) from the GitHub releases page, or build it from source.

If you are building from source, use Go 1.27.1 or newer.

Place the application and a `.env` file into a working directory.

Amend the `.env` file with your Discord API token, you will need to generate this yourself.  Remember to *NOT* store your key in the public domain.

Current environment variables:

```dotenv
BOTAPIKEY=your_discord_bot_token
```

- `BOTAPIKEY`: the Discord bot token used when the application connects to the Discord API

Windows

```powershell
go run .\cmd\bot
```

Linux

```shell
go run ./cmd/bot
```

Once connected, the bot prints `Bot is ready` to the console and listens for commands in every server it has been invited to. It doesn't post anything to Discord at startup. To set up a game, the Storyteller sends `!botc register` in the channel they want to use as the admin channel (see [Game channels](#game-channels)). Press Ctrl+C to stop the bot.

To build a binary instead of running from source:

```powershell
go build -o BotOnTheClocktower.exe ./cmd/bot
```

#### (Alternative) Run the container

Container images are published to the GitHub Container Registry (ghcr.io) for
every GitHub Release. Supply the `BOTAPIKEY` as an environment variable (no
`.env` file is required inside the container):

```shell
docker run -d -e BOTAPIKEY=your_discord_bot_token ghcr.io/ragnoaraknos/botontheclocktower:latest
```

Replace `latest` with a specific version tag (e.g. `v1.0`) to pin a release.

## Releases & CI/CD

This repository uses GitHub Actions to automate builds and container packaging:

- **Release build** (`.github/workflows/release.yml`): triggered when a tag
  matching `v*` (e.g. `v1.0`) is pushed. It runs the tests, cross-compiles the
  Linux and Windows (amd64) binaries, and publishes a GitHub Release with those
  binaries attached as assets.
- **Container image** (`.github/workflows/container.yml`): triggered separately
  when a Release is published. It builds the container image and pushes it to
  `ghcr.io/<owner>/botontheclocktower`, tagged with the release version and
  `latest`.

To cut a release:

```shell
git tag v1.0
git push origin v1.0
```

Pushing the tag publishes the GitHub Release (with binaries), which in turn
triggers the container image build and push.

## Commands

Every command starts with `!botc`, followed by the command name, e.g. `!botc ping`. Capitals don't matter (`!BotC Start` works). Messages from other bots are ignored.

Commands only work from the admin channel, and only for the Storyteller:

- **No game registered:** only `!botc register` (or `start`) and `!botc ping` work, from any channel. The channel `register` is sent from becomes the admin channel. Every other command is ignored, with no reply.
- **Game registered:** every command must come from the Storyteller, in the admin channel, except `!botc ping`, which the Storyteller can send from any channel. Anyone else, or a command sent anywhere else, gets a reply naming the Storyteller and the admin channel, and the command doesn't run.

| Command | What it does |
| --- | --- |
| `!botc ping` | Replies `pong`. Use it to check the bot is online. Works for anyone before a game is registered, and for the Storyteller from any channel once one is. |
| `!botc register` or `!botc start` | Starts a game. Makes the current channel the admin channel and the Town Square voice channel the game channel, and posts a "new game" announcement in Town Square's text chat. Makes the sender the Storyteller and gives them the `BoTC-StoryTeller` role. Apart from `ping`, the only command that works before a game is registered. Refused if a game is already registered, or if there's no voice channel named `Town Square`. Warns if the bot is missing permissions in the admin channel or the village voice channels (see [Server Setup](#4-give-the-bot-access-to-the-game-channels)). |
| `!botc unregister` or `!botc end` | Ends the game. Removes `BoTC-StoryTeller` and `BoTC-Player` from every member who has them, posts a "game ended" announcement in Town Square, and clears the game state so a new game can be registered. |
| `!botc sitrep` | Reports the game's server, the admin and game channels, and the Storyteller. |
| `!botc map` | Finds the village's voice channels by name and posts "Town Locations Mapped" in the admin channel. |
| `!botc village create` | Makes everyone in Town Square voice (except the Storyteller and bots) the village's players, replacing any existing list. Gives them `BoTC-Player` and takes it from anyone dropped. |
| `!botc village add @player...` | Adds the mentioned users to the village and gives them `BoTC-Player`. |
| `!botc village remove @player...` | Removes the mentioned users from the village and takes `BoTC-Player` away. |
| `!botc village list` | Lists the village's players. |
| `!botc character assign @player [good\|evil] <Character>` | Stores a village player's character and team (Good if left out). Any lines after the first (Shift+Enter) are guidance, kept exactly as typed. Nothing is sent yet. |
| `!botc character team @player good\|evil` | Moves a character to the other team and marks it unsent, so `send` tells the player. |
| `!botc character kill @player...` / `revive @player...` | Marks players Dead or Alive. Nothing is posted publicly until `announce`. Reviving gives back the ghost vote. |
| `!botc character ghostvote @player...` | Switches a dead player's ghost vote between used and available. |
| `!botc character announce` | Posts the deaths and revivals since the last announcement in Town Square's text chat. |
| `!botc character clear @player...` | Removes the stored characters. |
| `!botc character list` or `!botc grimoire` | Shows the grimoire: each village player's character, team, Alive/Dead, ghost vote, whether it has been sent and any unannounced change, with totals. |
| `!botc character send [@player...]` | DMs every unsent character to its player, or resends to the mentioned players. Reports failures and players with no character. |
| `!botc whisper @player <text>` | DMs a village player a secret message straight away. The text can span several lines. |

Replies to a command always go to the channel the command was sent in, which is the admin channel except for `register`, `ping` and refusals.

Any other `!botc` command from the Storyteller in the admin channel gets a "Huh? WTF is that command?!" reply.

### Game channels

A game uses two channels:

- **Admin channel:** the channel the Storyteller sends `!botc register` from. All later commands must be sent here, and admin output, such as the `!botc map` announcement, goes here. A private text channel only the Storyteller and moderators can see works well.
- **Game channel:** the `Town Square` voice channel. The bot posts game announcements (game started, game ended) in its text chat. It doesn't join the voice channel. It must exist before `!botc register`.

For `!botc register` and `!botc map`, the voice channels must use these exact names:

| Code | Channel name |
| --- | --- |
| `TS` | Town Square |
| `CA` | Cathedral |
| `CF` | Campfire |
| `PS` | Potion Shop |
| `TW` | Tower |
| `RS` | Riverside |
| `SC` | Storyteller's Corner |

Game state, including the village's player list and characters, is kept in memory only. If the bot restarts, run `!botc register`, `!botc map` and `!botc village create` again, and reassign the characters. A restart doesn't remove anyone's game roles; run `!botc unregister` first if you can.

## Features

(Ordered by development priority)

This section summarises each feature. The detailed intended behaviour, open questions and decisions for each one are in its spec under [docs/specs/](docs/specs/) (see [Feature specs](#feature-specs)). For how the built features work at runtime, see the [UML diagrams](#uml-diagrams).

### Gathering Players for the Tribunal

Status: IN PROGRESS

Spec: [gather.md](docs/specs/gather.md)

```shell
!botc gather
```

During the NIGHT phase, all players are placed into individually allocated "Cottage-XX" voice channels.  At the end of the night phase, the Storyteller needs the ability to draw all players into the "Town Square" voice channel for the DAY phase.

Also, at the end of the DAY phase when the town gathers for nominations - *some* players have the tendency to dilly dally in the side channels, this will forceably move the players into "Town Square".

Done so far: game registration with the sender as Storyteller (`!botc register`), mapping the village's voice channels (`!botc map`), and the player list (`!botc village`). Still to do: moving players, the `gather` command itself and "Cottage-XX" channels.

### Sending Players to Sleep

Status: PLANNED

Spec: [bedtime.md](docs/specs/bedtime.md)

```shell
!botc bedtime
```

At the end of the DAY phase, all players need to be placed into their respective "Cottage-XX" voice channel.

### Village Creation & Management

Status: DONE

Spec: [village-management.md](docs/specs/village-management.md)

```shell
!botc village create
!botc village add @player
!botc village remove @player
!botc village list
```

The village is the list of players in the game. The Storyteller builds it from everyone in Town Square voice, then adds or removes players by hand. Players in the village have the `BoTC-Player` role. See [Commands](#commands).

### Secret Characters

Status: DONE

Spec: [characters.md](docs/specs/characters.md)

```shell
!botc character assign @player Fortune Teller
Each night, choose 2 players: you learn if either is a Demon.
!botc character assign @player evil Poisoner
!botc grimoire
!botc character send
!botc whisper @player Your number tonight is 1.
!botc character kill @player
!botc character announce
```

The Storyteller gives each village player a character and a team (Good unless `evil` is given), optionally with guidance on the following lines. They check the grimoire, then send them all at once. Each player gets theirs by direct message, including their team. `whisper` sends a player secret information during the game.

During the game the Storyteller records deaths with `kill` and `revive`, and dead players' ghost votes with `ghostvote`. Nothing is made public until the Storyteller runs `announce`, which posts only what has changed since the last announcement. `!botc grimoire` shows the whole state in the admin channel.

Players must allow direct messages from server members (**Server → Privacy Settings → Direct Messages**). If a DM can't be delivered, the bot names the player so the Storyteller can fix it and send again.

### OBS Integration

Status: PLANNED

Spec: [obs-integration.md](docs/specs/obs-integration.md)

Control a local OBS instance from Discord (scene switching, audio and source control, recording), and change scenes automatically as the game moves between phases. See [docs/roadmap.md](docs/roadmap.md) for the plan.

### Vote tracking?

Status: IDEA

Spec: [vote-tracking.md](docs/specs/vote-tracking.md)

### Integration with game visualisation system?

Status: IDEA

Spec: [game-visualisation.md](docs/specs/game-visualisation.md)

## Developer Notes

### Feature specs

Each feature has a spec file in [docs/specs/](docs/specs/) describing what it should do. This README describes what the bot does *now*; the specs describe what's *intended*, and record how that changes over time.

- **Where to start:** [docs/specs/README.md](docs/specs/README.md) lists every spec with its status (IDEA → PLANNED → IN PROGRESS → DONE).
- **What a spec contains:** each follows [_template.md](docs/specs/_template.md):
  - the goal
  - behaviour with example commands and replies
  - rules and edge cases
  - a "Done when" checklist
  - what's out of scope
  - open questions
  - a dated log of decisions
  - once built, notes on the implementation

Workflow:

1. **New feature:** copy the template, or describe the feature roughly and have it drafted into a spec. Fill in the behaviour, edge cases and "Done when" list, and leave anything undecided under Open questions.
2. **Build in slices:** implement a few "Done when" items at a time and tick them off.
3. **Changed your mind:** update the spec first (Behaviour or Rules, plus a dated line under Decisions), then change the code to match.
4. **Finished:** set the status to DONE, fill in Implementation, update this README's [Commands](#commands) and [Features](#features) sections, and update the [UML diagrams](#uml-diagrams) (`/uml-sync`).

The AI agent instructions ([.github/copilot-instructions.md](.github/copilot-instructions.md), which Claude Code reads through [CLAUDE.md](CLAUDE.md)) tell agents to read the relevant spec before working on a feature, to ask about open questions rather than guess, and to record decisions in the spec as they're made.

The detailed plan for OBS integration lives in [docs/roadmap.md](docs/roadmap.md), which its spec links to.

### UML diagrams

[docs/uml/](docs/uml/README.md) shows how the bot works at runtime, as UML activity diagrams (the steps and decisions in each command) and sequence diagrams (the messages between the Storyteller, the bot, Discord and the players). They're written in Mermaid, so GitHub renders them in place.

| Diagrams | Covers |
| --- | --- |
| [startup.md](docs/uml/startup.md) | Loading the configuration, connecting to Discord, shutting down |
| [command-dispatch.md](docs/uml/command-dispatch.md) | How a message becomes a command, locking, panic recovery, the Storyteller check |
| [game-lifecycle.md](docs/uml/game-lifecycle.md) | `register`, `unregister`, `map`, `sitrep` |
| [village.md](docs/uml/village.md) | `village create`, `add`, `remove` |
| [characters.md](docs/uml/characters.md) | `character` subcommands, `grimoire`, `whisper` |

Keeping them current:

- When a change alters a command's flow, update its diagram in the same commit. The source map in [docs/uml/README.md](docs/uml/README.md) lists which diagram covers which function.
- `/uml-sync [git ref]` in Claude Code, or the `uml-sync` prompt in Copilot Chat, finds the diagrams affected by code changes since a ref, updates them, and checks that they render.
- The [Docs workflow](.github/workflows/docs.yml) renders every diagram on each push and pull request, so a Mermaid syntax error fails CI. It also warns when Go code changes without any change to `docs/uml/`.

### Anatomy of a command

Messages are split into words with [`strings.Fields`](https://pkg.go.dev/strings#Fields). If the first word is `!botc` (ignoring case) and there is at least one more word, the second word is the command name, dispatched in `extractCommand` in [internal/bot/commands.go](internal/bot/commands.go). Any further words are available as arguments. [command-dispatch.md](docs/uml/command-dispatch.md) shows the whole flow.

<https://www.educative.io/answers/how-to-split-a-string-in-golang>
