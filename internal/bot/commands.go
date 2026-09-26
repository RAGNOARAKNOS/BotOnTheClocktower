package bot

import (
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) extractCommand(message *discordgo.MessageCreate, rawText []string) {
	command := strings.ToLower(rawText[1])
	if !b.commandAllowed(message, command) {
		return
	}

	switch command {
	case "ping":
		b.send(message.ChannelID, "pong")
	case "register", "start":
		b.register(message)
	case "unregister", "end":
		b.unregister(message)
	case "sitrep":
		b.sitrep(message)
	case "map":
		if err := b.mapRooms(); err != nil {
			log.Printf("Could not map rooms: %v", err)
			b.reply(message, fmt.Sprintf("Could not map the town's channels (%v)", err))
		}
	case "village":
		b.village(message, rawText)
	case "character":
		b.character(message, rawText)
	case "grimoire":
		b.characterList(message)
	case "whisper":
		b.whisper(message)
	default:
		b.send(message.ChannelID, "Huh? WTF is that command?!")
	}
}

// commandAllowed reports whether a command may run, and is checked before every command.
// With no game registered, only register/start and ping run, from any channel (register's
// channel becomes the admin channel), and every other command is ignored without a reply.
// Once a game is registered, every command must come from the Storyteller, in the
// admin channel of the registered server, except ping, which the Storyteller can send
// from anywhere; anything else gets a refusal.
func (b *Bot) commandAllowed(message *discordgo.MessageCreate, command string) bool {
	if b.game == nil {
		if command == "register" || command == "start" || command == "ping" {
			return true
		}
		log.Printf("Ignoring %q: no game registered", command)
		return false
	}

	if command == "ping" && message.Author.ID == b.game.StorytellerID {
		return true
	}

	if message.GuildID != b.game.GuildID || message.Author.ID != b.game.StorytellerID || message.ChannelID != b.game.AdminChannelID {
		reply := fmt.Sprintf("Commands only work for the Storyteller (<@%s>), in the admin channel <#%s>. This command will not execute", b.game.StorytellerID, b.game.AdminChannelID)
		b.reply(message, reply)
		return false
	}

	return true
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
