# BotOnTheClocktower

<https://0x2142.com/how-to-discordgo-bot/>

<https://medium.com/@mssandeepkamath/building-a-simple-discord-bot-using-go-12bfca31ad5d>

<https://dev.to/aurelievache/learning-go-by-examples-part-4-create-a-bot-for-discord-in-go-43cf>

<https://github.com/scraly/learning-go-by-examples/tree/main/go-gopher-bot-discord>

## Purpose

This bot is designed to facilitate the needs of the Storyteller during a game of "Blood on the Clocktower" via Discord.  

This bot will provide chat commands that can be executed by the Storyteller to manage the Players throughout the relevant phases of play.

## Disclaimer

This is a personal project, and is no way affiliated with "The Pandemonium Institute" who created "Blood On The Clocktower".  I love the game, and recommend you use the official app or better yet buy a physical copy [here](https://bloodontheclocktower.com/).

## Notable libraries used

1. <https://github.com/bwmarrin/discordgo> - A low-level Go wrapper to the Discord API
1. <https://github.com/joho/godotenv> - Provides a framework for defining and consuming application parameters in a lightweight config file.

## Running the Bot

### Discord Setup

Register the bot application within your Discord developer page [here](https://discord.com/developers/applications) and make a note of your Discord BOT API token (in the BOT page).

Assign the Bot the following permissions:
- 

### Run the executable

Download the release zip from the GitHub releases page, or build it from source.

Place application and ENV file into a working directory.

Ammend the ENV file with your Discord API token, you will need to generate this yourself.  Remember to *NOT* store your key in the public domain.

Windows

```powershell
BotOnTheClocktower.exe 
```

Linux

```shell
.\BotOnTheClockTower
```

The bot will register with your configured Discord channel, and post a message confirming it has initialised and is ready to receive commands.

#### (Alternative) Run the container

```shell
docker run -d <TBD>
```

## Features

(Ordered by development priority)

### Gathering Players for the Tribunal

Status: IN WORK

```shell
!gather
```

During the NIGHT phase, all players are placed into individually allocated "Cottage-XX" voice channels.  At the end of the night phase, the Storyteller needs the ability to draw all players into the "Town Square" voice channel for the DAY phase.

Also, at the end of the DAY phase when the town gathers for nominations - *some* players have the tendency to dilly dally in the side channels, this will forceably move the players into "Town Square".

### Sending Players to Sleep

Status: PLANNED

```shell
!bedtime
```

At the end of the DAY phase, all players need to be placed into their respective "Cottage-XX" voice channel.

### Village Creation & Destruction

Status: PLANNED

### Vote tracking?

Status: IDEA

### Integration with game visualisation system?

Status: IDEA

## Developer Notes

### Anatomy of a command

<https://www.educative.io/answers/how-to-split-a-string-in-golang>

or String "fields" methods <https://pkg.go.dev/strings#Fields>
