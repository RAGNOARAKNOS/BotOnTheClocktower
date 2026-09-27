package bot

import "github.com/bwmarrin/discordgo"

// commands is every `!botc` / `/botc` command, in the order /botc lists them.
// Adding a command is one entry here (or in a group's subcommands) and its handler:
// the /botc definition (slashCommands) and the usage replies are built from this
// table. A new kind of /botc option also needs reading in slashWords.
var commands = []command{
	{
		name: "ping", description: "Check the bot is online",
		usage: "`!botc ping`",
		run:   (*Bot).ping, beforeGame: true, anyChannel: true,
	},
	{
		name: "register", aliases: []string{"start"},
		description: "Start a game: you become the Storyteller and this becomes the admin channel",
		usage:       "`!botc register` (or `!botc start`)",
		run:         (*Bot).register, beforeGame: true,
	},
	{
		name: "unregister", aliases: []string{"end"},
		description: "End the game and remove the game roles",
		usage:       "`!botc unregister` (or `!botc end`)",
		run:         (*Bot).unregister,
	},
	{
		name: "sitrep", description: "Report the game's server, channels and Storyteller",
		usage: "`!botc sitrep`",
		run:   (*Bot).sitrep,
	},
	{
		name: "map", description: "Find the village's voice channels",
		usage: "`!botc map`",
		run:   (*Bot).mapCommand,
	},
	{
		name: "grimoire", description: "Show every player's character",
		usage: "`!botc grimoire`",
		run:   (*Bot).characterList,
	},
	{
		name: "whisper", description: "DM a player a secret message (opens a form)",
		usage:   "`!botc whisper @player <text>` (the text can span several lines)",
		run:     (*Bot).whisper,
		options: opts(playerOption()),
		form:    whisperModal,
	},
	{
		name: "gather", description: "Count down, then move the players to Town Square",
		usage:   "`!botc gather` (60 seconds), `!botc gather <minutes>` (1 to 10) or `!botc gather cancel`",
		run:     (*Bot).gather,
		options: opts(minutesOption(), cancelOption()),
	},
	{
		name: "village", description: "Manage the village's players",
		subcommands: []command{
			{
				name: "create", description: "Make everyone in Town Square the village",
				usage: "`!botc village create`",
				run:   (*Bot).villageCreate,
			},
			{
				name: "add", description: "Add players to the village",
				usage:   "`!botc village add @player...`",
				run:     (*Bot).villageAdd,
				options: opts(playersOption(true)),
			},
			{
				name: "remove", description: "Remove players from the village",
				usage:   "`!botc village remove @player...`",
				run:     (*Bot).villageRemove,
				options: opts(playersOption(true)),
			},
			{
				name: "list", description: "List the village's players",
				usage: "`!botc village list`",
				run:   (*Bot).villageList,
			},
		},
	},
	{
		name: "character", description: "Manage the players' characters",
		subcommands: []command{
			{
				name: "assign", description: "Give a player a character (opens a form for guidance)",
				usage:   "`!botc character assign @player [good|evil] <Character>` (guidance on the following lines)",
				run:     (*Bot).characterAssign,
				options: opts(playerOption(), teamOption(false), characterOption()),
				form:    assignModal,
			},
			{
				name: "team", description: "Move characters to another team",
				usage:   "`!botc character team @player good|evil`",
				run:     (*Bot).characterTeam,
				options: opts(playersOption(true), teamOption(true)),
			},
			{
				name: "kill", description: "Mark players dead",
				usage:   "`!botc character kill @player...`",
				run:     func(b *Bot, req *request) { b.characterSetAlive(req, false) },
				options: opts(playersOption(true)),
			},
			{
				name: "revive", description: "Mark players alive",
				usage:   "`!botc character revive @player...`",
				run:     func(b *Bot, req *request) { b.characterSetAlive(req, true) },
				options: opts(playersOption(true)),
			},
			{
				name: "ghostvote", description: "Switch dead players' ghost votes",
				usage:   "`!botc character ghostvote @player...`",
				run:     (*Bot).characterGhostVote,
				options: opts(playersOption(true)),
			},
			{
				name: "announce", description: "Announce deaths and revivals in Town Square",
				usage: "`!botc character announce`",
				run:   (*Bot).characterAnnounce,
			},
			{
				name: "clear", description: "Remove players' characters",
				usage:   "`!botc character clear @player...`",
				run:     (*Bot).characterClear,
				options: opts(playersOption(true)),
			},
			{
				name: "list", description: "Show every player's character",
				usage: "`!botc character list` (or `!botc grimoire`)",
				run:   (*Bot).characterList,
			},
			{
				name: "send", description: "DM unsent characters, or resend to the players given",
				usage:   "`!botc character send [@player...]`",
				run:     (*Bot).characterSend,
				options: opts(playersOption(false)),
			},
		},
	},
}

// opts lists a command's /botc options.
func opts(options ...*discordgo.ApplicationCommandOption) []*discordgo.ApplicationCommandOption {
	return options
}
