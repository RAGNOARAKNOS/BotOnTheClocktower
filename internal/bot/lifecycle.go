package bot

import (
	"fmt"
	"log"
	"strings"
)

// register starts a game: the sender becomes the Storyteller, this channel the admin
// channel, and the village's voice channels (villageCodeLookup) are recorded. It's
// refused if any of those channels is missing.
func (b *Bot) register(req *request) {
	if b.game != nil {
		reply := fmt.Sprintf("A game is already registered, with <@%s> as the Storyteller. This command will not execute", b.game.StorytellerID)
		req.reply(reply)
		return
	}

	channels, err := b.discord.GuildChannels(req.guildID)
	if err != nil {
		req.reply(fmt.Sprintf("Could not read the server's channels (%v). This command will not execute", err))
		return
	}
	rooms := villageRooms(channels)
	if missing := missingRooms(rooms); len(missing) > 0 {
		req.reply(fmt.Sprintf("Could not find these village voice channels: %s. Create voice channels with exactly these names, then try again. This command will not execute",
			strings.Join(missing, ", ")))
		return
	}

	b.game = newGame(req.guildID, req.channelID, rooms["TS"], req.authorID)
	b.game.Rooms = rooms

	log.Printf("The game has been registered at %s admin channel %s game channel %s storyteller %s", b.game.GuildID, b.game.AdminChannelID, b.game.GameChannelID, b.game.StorytellerID)

	b.send(b.game.GameChannelID, fmt.Sprintf("A new game has begun. <@%s> is the Storyteller.", b.game.StorytellerID))

	reply := fmt.Sprintf("Game registered. <@%s> is the Storyteller. This is the admin channel; <#%s> is the game channel.", b.game.StorytellerID, b.game.GameChannelID)
	if err := b.assignStorytellerRole(b.game.GuildID, b.game.StorytellerID); err != nil {
		log.Printf("Could not assign the %s role: %v", storytellerRoleName, err)
		reply += fmt.Sprintf(" Warning: could not assign the %q role (%v). Check the role exists and sits below the bot's role.", storytellerRoleName, err)
	}
	// Missing channel permissions make the bot silently ignore commands, so warn now.
	if warning := b.channelAccessWarning(b.game.AdminChannelID, b.game.villageChannels()); warning != "" {
		log.Printf("The bot is missing channel permissions: %s", warning)
		reply += warning
	}
	req.reply(reply)
}

// unregister ends the current game: it removes the game roles from everyone
// and forgets the game so a new one can be registered.
func (b *Bot) unregister(req *request) {
	guildID := b.game.GuildID

	removed, roleErr := b.removeGameRoles(guildID)

	b.send(b.game.GameChannelID, "The game has ended. Thanks for playing!")

	if b.game.gather != nil {
		b.game.gather.stop()
	}
	b.game = nil

	log.Printf("The game at %s has been unregistered, %d game roles removed", guildID, removed)

	reply := fmt.Sprintf("Game ended. Removed %d game role(s).", removed)
	if roleErr != nil {
		log.Printf("Problems removing game roles: %v", roleErr)
		reply += fmt.Sprintf(" Warning: some roles could not be removed (%v). Check the %q and %q roles exist and sit below the bot's role.", roleErr, storytellerRoleName, playerRoleName)
	}
	req.reply(reply)
}

// sitrep reports where the game is running. It only runs while a game is registered.
func (b *Bot) sitrep(req *request) {
	req.say(fmt.Sprintf("SITREP-Game is initialised at guildid# %s admin channel <#%s> game channel <#%s> storyteller <@%s>", b.game.GuildID, b.game.AdminChannelID, b.game.GameChannelID, b.game.StorytellerID))
}
