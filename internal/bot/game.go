package bot

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
)

// Game is the state of the registered game. It's held in memory only, so a restart loses it.
type Game struct {
	GuildID        string
	AdminChannelID string // channel register was sent from; commands and admin output go here
	GameChannelID  string // Town Square voice channel; game announcements go to its text chat
	StorytellerID  string
	Players        map[string]string     // village: user ID → display name
	Characters     map[string]*Character // user ID → character; only for players in the village
	Rooms          map[string]string     // room code (see villageCodeLookup) → channel ID; filled by map
}

// newGame starts a game with empty player, character and room lists.
func newGame(guildID, adminChannelID, gameChannelID, storytellerID string) *Game {
	return &Game{
		GuildID:        guildID,
		AdminChannelID: adminChannelID,
		GameChannelID:  gameChannelID,
		StorytellerID:  storytellerID,
		Players:        make(map[string]string),
		Characters:     make(map[string]*Character),
		Rooms:          make(map[string]string),
	}
}

var villageCodeLookup = map[string]string{
	"TS": "Town Square",
	"CA": "Cathedral",
	"CF": "Campfire",
	"PS": "Potion Shop",
	"TW": "Tower",
	"RS": "Riverside",
	"SC": "Storyteller's Corner",
}

func (b *Bot) register(message *discordgo.MessageCreate) {
	if b.game != nil {
		reply := fmt.Sprintf("A game is already registered, with <@%s> as the Storyteller. This command will not execute", b.game.StorytellerID)
		b.reply(message, reply)
		return
	}

	townSquareName := villageCodeLookup["TS"]
	gameChannelID, err := b.findVoiceChannelID(message.GuildID, townSquareName)
	if err != nil {
		reply := fmt.Sprintf("Could not find the game channel (%v). Create a voice channel named %q, then try again. This command will not execute", err, townSquareName)
		b.reply(message, reply)
		return
	}

	b.game = newGame(message.GuildID, message.ChannelID, gameChannelID, message.Author.ID)

	log.Printf("The game has been registered at %s admin channel %s game channel %s storyteller %s", b.game.GuildID, b.game.AdminChannelID, b.game.GameChannelID, b.game.StorytellerID)

	b.send(b.game.GameChannelID, fmt.Sprintf("A new game has begun. <@%s> is the Storyteller.", b.game.StorytellerID))

	reply := fmt.Sprintf("Game registered. <@%s> is the Storyteller. This is the admin channel; <#%s> is the game channel.", b.game.StorytellerID, b.game.GameChannelID)
	if err := b.assignStorytellerRole(b.game.GuildID, b.game.StorytellerID); err != nil {
		log.Printf("Could not assign the %s role: %v", storytellerRoleName, err)
		reply += fmt.Sprintf(" Warning: could not assign the %q role (%v). Check the role exists and sits below the bot's role.", storytellerRoleName, err)
	}
	b.reply(message, reply)
}

// unregister ends the current game: it removes the game roles from everyone
// and forgets the game so a new one can be registered.
func (b *Bot) unregister(message *discordgo.MessageCreate) {
	guildID := b.game.GuildID

	removed, roleErr := b.removeGameRoles(guildID)

	b.send(b.game.GameChannelID, "The game has ended. Thanks for playing!")

	b.game = nil

	log.Printf("The game at %s has been unregistered, %d game roles removed", guildID, removed)

	reply := fmt.Sprintf("Game ended. Removed %d game role(s).", removed)
	if roleErr != nil {
		log.Printf("Problems removing game roles: %v", roleErr)
		reply += fmt.Sprintf(" Warning: some roles could not be removed (%v). Check the %q and %q roles exist and sit below the bot's role.", roleErr, storytellerRoleName, playerRoleName)
	}
	b.reply(message, reply)
}

// sitrep reports where the game is running. It only runs while a game is registered.
func (b *Bot) sitrep(message *discordgo.MessageCreate) {
	b.send(message.ChannelID, fmt.Sprintf("SITREP-Game is initialised at guildid# %s admin channel <#%s> game channel <#%s> storyteller <@%s>", b.game.GuildID, b.game.AdminChannelID, b.game.GameChannelID, b.game.StorytellerID))
}

func (b *Bot) mapRooms() error {
	allChans, err := b.getMapGuildChannels(b.game.GuildID)
	if err != nil {
		return err
	}

	b.game.Rooms = make(map[string]string)
	for index, ch := range allChans {
		for code, room := range villageCodeLookup {
			if room == ch {
				b.game.Rooms[code] = index
			}
		}
	}

	if _, err := b.discord.ChannelMessageSendTTS(b.game.AdminChannelID, "Town Locations Mapped"); err != nil {
		log.Printf("Could not post in channel %s: %v", b.game.AdminChannelID, err)
	}
	return nil
}
