package bot

import (
	"errors"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// permission is a Discord permission the bot needs, with the name Discord's settings show.
type permission struct {
	bit  int64
	name string
}

var (
	viewChannel        = permission{discordgo.PermissionViewChannel, "View Channels"}
	sendMessages       = permission{discordgo.PermissionSendMessages, "Send Messages"}
	sendTTSMessages    = permission{discordgo.PermissionSendTTSMessages, "Send TTS Messages"}
	readMessageHistory = permission{discordgo.PermissionReadMessageHistory, "Read Message History"}
	connect            = permission{discordgo.PermissionVoiceConnect, "Connect"}
	moveMembers        = permission{discordgo.PermissionVoiceMoveMembers, "Move Members"}
)

// What the bot needs in each kind of game channel.
var (
	// The admin channel: commands, replies (threaded, so history) and map's TTS announcement.
	adminChannelNeeds = []permission{viewChannel, sendMessages, readMessageHistory, sendTTSMessages}
	// Town Square: announcements in its text chat, and moving players in and out.
	gameChannelNeeds = []permission{viewChannel, sendMessages, connect, moveMembers}
	// The other village rooms: moving players in and out.
	roomNeeds = []permission{viewChannel, connect, moveMembers}
)

// missingPermissions returns the names of the permissions in needs that have lacks.
// Administrator grants everything.
func missingPermissions(have int64, needs []permission) []string {
	if have&discordgo.PermissionAdministrator != 0 {
		return nil
	}
	var missing []string
	for _, p := range needs {
		if have&p.bit == 0 {
			missing = append(missing, p.name)
		}
	}
	return missing
}

// channelAccessWarning checks the bot's permissions in the admin channel, Town Square
// and the other village voice channels that exist. It returns an empty string if
// nothing is missing, otherwise a warning listing each channel's missing permissions.
func (b *Bot) channelAccessWarning(guildID, adminChannelID, gameChannelID string) string {
	botID, err := b.botUserID()
	if err != nil {
		return fmt.Sprintf("\nWarning: could not check the bot's channel permissions (%v).", err)
	}

	type check struct {
		channelID string
		needs     []permission
	}
	checks := []check{{adminChannelID, adminChannelNeeds}, {gameChannelID, gameChannelNeeds}}
	if channels, err := b.discord.GuildChannels(guildID); err == nil {
		for _, ch := range channels {
			if ch.Type == discordgo.ChannelTypeGuildVoice && ch.ID != gameChannelID && isVillageRoom(ch.Name) {
				checks = append(checks, check{ch.ID, roomNeeds})
			}
		}
	}

	var problems []string
	for _, c := range checks {
		have, err := b.discord.UserChannelPermissions(botID, c.channelID)
		var restErr *discordgo.RESTError
		switch {
		case errors.As(err, &restErr) && restErr.Message != nil && restErr.Message.Code == discordgo.ErrCodeMissingAccess:
			// Discord won't describe a channel to a bot that can't view it.
			problems = append(problems, fmt.Sprintf("- <#%s>: can't see this channel at all (needs %s first)", c.channelID, viewChannel.name))
			continue
		case err != nil:
			problems = append(problems, fmt.Sprintf("- <#%s>: couldn't check (%v)", c.channelID, err))
			continue
		}
		if missing := missingPermissions(have, c.needs); len(missing) > 0 {
			problems = append(problems, fmt.Sprintf("- <#%s>: %s", c.channelID, strings.Join(missing, ", ")))
		}
	}
	if len(problems) == 0 {
		return ""
	}
	return "\nWarning: the bot is missing permissions it needs in these channels:\n" + strings.Join(problems, "\n") +
		"\nAllow them for the bot's role on the game channels' category (see Server Setup in the README). In a channel the bot can't view, it ignores commands."
}

// botUserID returns the bot's own user ID, from the session state or, failing that, from Discord.
func (b *Bot) botUserID() (string, error) {
	if b.discord.State != nil && b.discord.State.User != nil {
		return b.discord.State.User.ID, nil
	}
	me, err := b.discord.User("@me")
	if err != nil {
		return "", err
	}
	return me.ID, nil
}

// isVillageRoom reports whether a channel name is one of the village locations.
func isVillageRoom(name string) bool {
	for _, room := range villageCodeLookup {
		if room == name {
			return true
		}
	}
	return false
}
