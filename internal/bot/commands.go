package bot

import (
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// request is one command, from a `!botc` message or a `/botc` slash command, in a
// form the handlers can use without knowing which.
type request struct {
	authorID, guildID, channelID string
	// words is the command as words: "!botc" (or "/botc"), the command name, then its arguments.
	words []string
	// content is the raw command text, for the commands that parse free text (parse.go).
	content string
	// mentions are the users the command names.
	mentions []*discordgo.User
	// reply answers the command: a threaded reply to a message, or the slash command's
	// response, visible only to the sender. say is the same, but not threaded.
	reply func(text string)
	say   func(text string)
}

// command is one `!botc <name>` / `/botc <name>` command. By default a command only
// runs while a game is registered, for the Storyteller, in the admin channel; the flags relax that.
type command struct {
	run func(b *Bot, req *request)
	// beforeGame lets anyone run the command, from any channel, while no game is registered.
	beforeGame bool
	// anyChannel lets the Storyteller run the command outside the admin channel.
	anyChannel bool
	// alias marks another name for a command listed under its own name. Aliases work
	// as `!botc` words only; slash commands have just the main name.
	alias bool
}

// commands maps each command name, and each alias, to its command.
var commands = map[string]command{
	"ping":       {run: (*Bot).ping, beforeGame: true, anyChannel: true},
	"register":   {run: (*Bot).register, beforeGame: true},
	"start":      {run: (*Bot).register, beforeGame: true, alias: true},
	"unregister": {run: (*Bot).unregister},
	"end":        {run: (*Bot).unregister, alias: true},
	"sitrep":     {run: (*Bot).sitrep},
	"map":        {run: (*Bot).mapCommand},
	"village":    {run: (*Bot).village},
	"character":  {run: (*Bot).character},
	"grimoire":   {run: (*Bot).characterList},
	"whisper":    {run: (*Bot).whisper},
	"gather":     {run: (*Bot).gather},
}

// extractCommand runs the command named by the second word, if the sender may run it.
// It returns false if the command was ignored without a reply.
func (b *Bot) extractCommand(req *request) (answered bool) {
	name := strings.ToLower(req.words[1])
	cmd, known := commands[name]

	ok, refusal := allowed(b.game, cmd, req.authorID, req.guildID, req.channelID)
	if !ok {
		if refusal == "" {
			log.Printf("Ignoring %q: no game registered", name)
			return false
		}
		req.reply(refusal)
		return true
	}

	if !known {
		req.say("Huh? WTF is that command?!")
		return true
	}
	cmd.run(b, req)
	return true
}

// dispatch runs the subcommand named by the third word from subcommands, or replies with usage.
func (b *Bot) dispatch(req *request, subcommands map[string]func(b *Bot, req *request), usage string) {
	if len(req.words) < 3 {
		req.reply(usage)
		return
	}
	run, ok := subcommands[strings.ToLower(req.words[2])]
	if !ok {
		req.reply(usage)
		return
	}
	run(b, req)
}

// allowed decides whether a command may run. With no game registered, only commands
// marked beforeGame run, for anyone, from anywhere; anything else is ignored, which
// allowed signals with an empty refusal. Once a game is registered, commands only run
// for the Storyteller, in the admin channel of the registered server (anyChannel
// commands skip the server and channel check); anything else gets the refusal text.
// Unknown commands are checked like any other command, with no flags set.
func allowed(game *Game, cmd command, authorID, guildID, channelID string) (ok bool, refusal string) {
	if game == nil {
		return cmd.beforeGame, ""
	}

	isStoryteller := authorID == game.StorytellerID
	inAdmin := guildID == game.GuildID && channelID == game.AdminChannelID
	if isStoryteller && (inAdmin || cmd.anyChannel) {
		return true, ""
	}

	return false, fmt.Sprintf("Commands only work for the Storyteller (<@%s>), in the admin channel <#%s>. This command will not execute", game.StorytellerID, game.AdminChannelID)
}

// ping lets anyone check the bot is online and reading commands.
func (b *Bot) ping(req *request) {
	req.say("pong")
}

// reply answers a command as a threaded reply in the channel it came from.
// Send failures are logged, as there's nowhere else to report them.
func (b *Bot) reply(message *discordgo.MessageCreate, text string) {
	if _, err := b.discord.ChannelMessageSendReply(message.ChannelID, text, message.Reference()); err != nil {
		log.Printf("Could not reply in channel %s: %v", message.ChannelID, err)
	}
}

// send posts a plain message in a channel, logging any failure.
func (b *Bot) send(channelID, text string) {
	if _, err := b.discord.ChannelMessageSend(channelID, text); err != nil {
		log.Printf("Could not post in channel %s: %v", channelID, err)
	}
}
