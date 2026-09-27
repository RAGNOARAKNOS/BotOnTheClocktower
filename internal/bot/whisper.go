package bot

import (
	"errors"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
)

// whisper DMs a village player a secret message from the Storyteller straight away.
func (b *Bot) whisper(req *request) {
	userID, text, err := parseWhisper(req.content)
	if err != nil {
		req.reply(fmt.Sprintf("Could not read that (%v). %s", err, whisperUsage))
		return
	}

	playerName, ok := b.game.Players[userID]
	if !ok {
		req.reply(fmt.Sprintf("<@%s> isn't in the village. This command will not execute", userID))
		return
	}

	if err := b.sendDM(userID, b.dmEmbed("A message from the Storyteller", text)); err != nil {
		log.Printf("Could not whisper to %s: %v", userID, err)
		req.reply(fmt.Sprintf("Could not whisper to %s (%s).", playerName, dmErrorReason(err)))
		return
	}

	req.reply(fmt.Sprintf("Whispered to %s.", playerName))
}

// sendDM sends an embed to the user in a direct message.
func (b *Bot) sendDM(userID string, embed *discordgo.MessageEmbed) error {
	channel, err := b.discord.UserChannelCreate(userID)
	if err != nil {
		return err
	}

	_, err = b.discord.ChannelMessageSendEmbed(channel.ID, embed)
	return err
}

// dmEmbed builds a DM embed, with the game's server named in the footer.
func (b *Bot) dmEmbed(title, description string) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{Title: title, Description: description}
	if guild, err := b.discord.State.Guild(b.game.GuildID); err == nil {
		embed.Footer = &discordgo.MessageEmbedFooter{Text: "Blood on the Clocktower on " + guild.Name}
	}
	return embed
}

// dmErrorReason turns a failed DM into a short reason for the Storyteller.
func dmErrorReason(err error) string {
	var restErr *discordgo.RESTError
	if errors.As(err, &restErr) && restErr.Message != nil && restErr.Message.Code == discordgo.ErrCodeCannotSendMessagesToThisUser {
		return "they don't accept DMs from this server"
	}
	return err.Error()
}
