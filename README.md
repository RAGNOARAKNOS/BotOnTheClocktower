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
| Send Messages | Reply to commands |
| Send TTS Messages | Announce "Town Locations Mapped" after `!botc map` |
| Read Message History | Reply directly to the command message |
| Connect | Join the game channel's voice on `!botc register` |
| Manage Roles | Give the `BOTC-StoryTeller` role on `!botc register` |
| Move Members | Needed for the planned commands that move players between voice channels |

When the bot joins, Discord automatically creates a role with the bot's name that holds these permissions. Don't delete it.

#### 2. Check the game roles exist

The bot uses two roles that must already exist on the server. It doesn't create them; it looks them up by name, and the names must match exactly, including capitals and the hyphen:

| Role | Purpose |
| --- | --- |
| `BOTC-StoryTeller` | Given to whoever runs `!botc register` |
| `BOTC-Player` | Marks players in the game. Not used by any command yet |

The bot doesn't need either role to have any permissions. What you give them is up to you. Useful options for `BOTC-StoryTeller` are Move Members, Mute Members, Deafen Members and Priority Speaker.

#### 3. Put the roles in the right order

Discord only lets a bot give out roles that sit **below its own highest role**. In **Server Settings → Roles**, drag the roles into this order (top of the list = highest):

```text
Admin / Moderator roles     <- keep these above the bot
BotOnTheClocktower          <- the bot's own role
BOTC-StoryTeller            <- must be below the bot's role
BOTC-Player                 <- must be below the bot's role
Other member roles
@everyone
```

- If `BOTC-StoryTeller` is above the bot's role, registration still succeeds, but the bot can't give out the role. It replies with a warning instead.
- Manage Roles lets the bot give out **any** role below its own. Keep moderator and admin roles above the bot's role so it can never hand them out.

#### 4. Check channel overrides

Per-channel permission overrides take priority over server-wide permissions. Make sure no override on the game's text channel or the village voice channels denies the bot View Channels, Send Messages, Connect or Move Members.

#### Who becomes Storyteller

- Whoever sends `!botc register` becomes the Storyteller for that game and is given the `BOTC-StoryTeller` role.
- The bot runs one game at a time. Once a game is registered, `!botc register` is refused for everyone, including the current Storyteller.
- To start a new game or change Storyteller, restart the bot. The bot never removes the `BOTC-StoryTeller` role, so a moderator has to take it off the previous Storyteller by hand.

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

Once connected, the bot prints `Bot is ready` to the console and listens for commands in every server it has been invited to. It doesn't post anything to Discord at startup. To set up a game, send `!botc register` in the channel you want to use (see [Commands](#commands)). Press Ctrl+C to stop the bot.

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

Every command starts with `!botc`, followed by the command name, e.g. `!botc ping`. Send them in a text channel on the server where the game is being played.

| Command | What it does |
| --- | --- |
| `!botc ping` | Replies `pong`. Use it to check the bot is online. |
| `!botc register` | Registers the current server and channel as the game's location, makes the sender the Storyteller and gives them the `BOTC-StoryTeller` role, and tries to join that channel's voice. Run this before any other game command. Refused if a game is already registered. |
| `!botc sitrep` | Reports whether a game is registered and, if so, the server, channel and Storyteller IDs. |
| `!botc map` | Finds the village's voice channels by name and posts "Town Locations Mapped". It then lists the players in Town Square, but only to the bot's console for now. Requires `register` first. |

Any other `!botc` command gets a "Huh? WTF is that command?!" reply.

For `!botc map`, the voice channels must use these exact names:

| Code | Channel name |
| --- | --- |
| `TS` | Town Square |
| `CA` | Cathedral |
| `CF` | Campfire |
| `PS` | Potion Shop |
| `TW` | Tower |
| `RS` | Riverside |
| `SC` | Storyteller's Corner |

Game state is kept in memory only. If the bot restarts, run `!botc register` and `!botc map` again. Restarting is also the only way to register a new game or change Storyteller.

## Features

(Ordered by development priority)

### Gathering Players for the Tribunal

Status: IN WORK

```shell
!botc gather
```

During the NIGHT phase, all players are placed into individually allocated "Cottage-XX" voice channels.  At the end of the night phase, the Storyteller needs the ability to draw all players into the "Town Square" voice channel for the DAY phase.

Also, at the end of the DAY phase when the town gathers for nominations - *some* players have the tendency to dilly dally in the side channels, this will forceably move the players into "Town Square".

Done so far: game registration with the sender as Storyteller (`!botc register`), mapping the village's voice channels (`!botc map`), and an internal helper for moving a player to a channel. Still to do: the `gather` command itself, recording the player list, restricting commands to the Storyteller, and "Cottage-XX" channels. The `pmove` and `cmove` commands are placeholders that currently do nothing.

### Sending Players to Sleep

Status: PLANNED

```shell
!botc bedtime
```

At the end of the DAY phase, all players need to be placed into their respective "Cottage-XX" voice channel.

### Village Creation & Destruction

Status: PLANNED

### OBS Integration

Status: PLANNED

Control a local OBS instance from Discord (scene switching, audio and source control, recording), and change scenes automatically as the game moves between phases. See [roadmap.md](roadmap.md) for the plan.

### Vote tracking?

Status: IDEA

### Integration with game visualisation system?

Status: IDEA

## Developer Notes

### Anatomy of a command

Messages are split into words with [`strings.Fields`](https://pkg.go.dev/strings#Fields). If the first word contains `!botc` and there is at least one more word, the second word is the command name, dispatched in `extractCommand` in [internal/bot/bot.go](internal/bot/bot.go). Any further words are available as arguments.

<https://www.educative.io/answers/how-to-split-a-string-in-golang>
