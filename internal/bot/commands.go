package bot

import (
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// command is one `!botc <name>` command. By default a command only runs while a game
// is registered, for the Storyteller, in the admin channel; the flags relax that.
type command struct {
	run func(b *Bot, message *discordgo.MessageCreate, words []string)
	// beforeGame lets anyone run the command, from any channel, while no game is registered.
	beforeGame bool
	// anyChannel lets the Storyteller run the command outside the admin channel.
	anyChannel bool
}

// commands maps each command name, and each alias, to its command.
var commands = map[string]command{
	"ping":       {run: (*Bot).ping, beforeGame: true, anyChannel: true},
	"register":   {run: (*Bot).register, beforeGame: true},
	"start":      {run: (*Bot).register, beforeGame: true},
	"unregister": {run: (*Bot).unregister},
	"end":        {run: (*Bot).unregister},
	"sitrep":     {run: (*Bot).sitrep},
	"map":        {run: (*Bot).mapCommand},
	"village":    {run: (*Bot).village},
	"character":  {run: (*Bot).character},
	"grimoire":   {run: (*Bot).characterList},
	"whisper":    {run: (*Bot).whisper},
}

// extractCommand runs the command named by the second word, if the sender may run it.
func (b *Bot) extractCommand(message *discordgo.MessageCreate, words []string) {
	name := strings.ToLower(words[1])
	cmd, known := commands[name]

	ok, refusal := allowed(b.game, cmd, message.Author.ID, message.GuildID, message.ChannelID)
	if !ok {
		if refusal == "" {
			log.Printf("Ignoring %q: no game registered", name)
		} else {
			b.reply(message, refusal)
		}
		return
	}

	if !known {
		b.send(message.ChannelID, "Huh? WTF is that command?!")
		return
	}
	cmd.run(b, message, words)
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
func (b *Bot) ping(message *discordgo.MessageCreate, _ []string) {
	b.send(message.ChannelID, "pong")
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
