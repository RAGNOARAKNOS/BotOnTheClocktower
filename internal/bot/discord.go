package bot

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

// Discord's limits.
const (
	// maxMessageLength is the limit on a message's content.
	maxMessageLength = 2000
	// maxEmbedDescription is the limit on an embed's description.
	maxEmbedDescription = 4096
	// maxTextInput is the limit on a form's text box.
	maxTextInput = 4000
)

// truncate shortens s to at most n characters.
func truncate(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n])
	}
	return s
}

// lookupMember returns known if it has user details, otherwise the member from
// the state cache or, failing that, from Discord. It returns nil if all of those fail.
func (b *Bot) lookupMember(guildID, userID string, known *discordgo.Member) *discordgo.Member {
	if known != nil && known.User != nil {
		return known
	}
	if member, err := b.discord.State.Member(guildID, userID); err == nil && member.User != nil {
		return member
	}
	if member, err := b.discord.GuildMember(guildID, userID); err == nil && member.User != nil {
		return member
	}
	return nil
}

// memberDisplayName returns the member's server nickname, global name or username,
// or the user ID if the member couldn't be found.
func memberDisplayName(member *discordgo.Member, userID string) string {
	if member == nil || member.User == nil {
		return userID
	}
	return member.DisplayName()
}

// findVoiceChannelID returns the ID of the server's first voice channel with exactly this name.
func (b *Bot) findVoiceChannelID(guildID, channelName string) (string, error) {
	channels, err := b.discord.GuildChannels(guildID)
	if err != nil {
		return "", err
	}

	for _, ch := range channels {
		if ch.Type == discordgo.ChannelTypeGuildVoice && ch.Name == channelName {
			return ch.ID, nil
		}
	}

	return "", fmt.Errorf("no voice channel named %q in this server", channelName)
}
