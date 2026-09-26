package bot

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

// lookupMember returns known if it has user details, otherwise the member from
// the state cache or, failing that, from Discord. It returns nil if all of those fail.
func (b *Bot) lookupMember(userID string, known *discordgo.Member) *discordgo.Member {
	if known != nil && known.User != nil {
		return known
	}
	if member, err := b.discord.State.Member(b.game.GuildID, userID); err == nil && member.User != nil {
		return member
	}
	if member, err := b.discord.GuildMember(b.game.GuildID, userID); err == nil && member.User != nil {
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

func (b *Bot) getMapGuildChannels(guildId string) (map[string]string, error) {
	channels, err := b.discord.GuildChannels(guildId)
	if err != nil {
		return nil, err
	}

	chanMap := make(map[string]string)
	for _, ch := range channels {
		chanMap[ch.ID] = ch.Name
	}

	return chanMap, nil
}
