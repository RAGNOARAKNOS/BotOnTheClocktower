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

// The game's rules. These change only the Game, never Discord, so they can be
// unit-tested; the command handlers do the Discord side and build the replies.

// ReplacePlayers makes players the whole village and returns who was dropped.
// Dropped players lose their characters.
func (g *Game) ReplacePlayers(players map[string]string) (dropped map[string]string) {
	dropped = make(map[string]string)
	for id, name := range g.Players {
		if _, ok := players[id]; !ok {
			dropped[id] = name
			delete(g.Characters, id)
		}
	}
	g.Players = players
	return dropped
}

// RemovePlayer takes a player out of the village, with their character.
func (g *Game) RemovePlayer(userID string) {
	delete(g.Players, userID)
	delete(g.Characters, userID)
}

// Assign gives a village player a character, unsent, and returns the character it
// replaces, if any. A new character starts alive; a replacement keeps the player's
// life and ghost vote, because a new character doesn't bring anyone back to life.
func (g *Game) Assign(userID string, c *Character) (previous *Character) {
	previous = g.Characters[userID]
	c.Sent = false
	if previous != nil {
		c.Alive, c.AnnouncedAlive, c.GhostVoteUsed = previous.Alive, previous.AnnouncedAlive, previous.GhostVoteUsed
	} else {
		c.Alive, c.AnnouncedAlive, c.GhostVoteUsed = true, true, false
	}
	g.Characters[userID] = c
	return previous
}

// SetTeam moves a player's character to team and marks it unsent, so `send` tells
// them. It returns false if the character was already on that team.
func (g *Game) SetTeam(userID string, team Team) bool {
	c := g.Characters[userID]
	if c.Team == team {
		return false
	}
	c.Team = team
	c.Sent = false
	return true
}

// SetAlive kills or revives a player's character; reviving gives back the ghost vote.
// It returns false if the character was already in that state. Nothing is public
// until MarkAnnounced.
func (g *Game) SetAlive(userID string, alive bool) bool {
	c := g.Characters[userID]
	if c.Alive == alive {
		return false
	}
	c.Alive = alive
	if alive {
		c.GhostVoteUsed = false
	}
	return true
}

// ToggleGhostVote switches a dead player's ghost vote between used and available.
// It returns false, changing nothing, if the player is alive.
func (g *Game) ToggleGhostVote(userID string) bool {
	c := g.Characters[userID]
	if c.Alive {
		return false
	}
	c.GhostVoteUsed = !c.GhostVoteUsed
	return true
}

// PendingLifeChanges returns who has died or come back since the last announcement.
func (g *Game) PendingLifeChanges() (died, revived []string) {
	return pendingLifeChanges(g.Players, g.Characters)
}

// MarkAnnounced records every character's current life state as announced.
func (g *Game) MarkAnnounced() {
	for _, c := range g.Characters {
		c.AnnouncedAlive = c.Alive
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

func (b *Bot) register(message *discordgo.MessageCreate, _ []string) {
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
func (b *Bot) unregister(message *discordgo.MessageCreate, _ []string) {
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
func (b *Bot) sitrep(message *discordgo.MessageCreate, _ []string) {
	b.send(message.ChannelID, fmt.Sprintf("SITREP-Game is initialised at guildid# %s admin channel <#%s> game channel <#%s> storyteller <@%s>", b.game.GuildID, b.game.AdminChannelID, b.game.GameChannelID, b.game.StorytellerID))
}

// mapCommand runs `!botc map`, replying if the channels couldn't be read.
func (b *Bot) mapCommand(message *discordgo.MessageCreate, _ []string) {
	if err := b.mapRooms(); err != nil {
		log.Printf("Could not map rooms: %v", err)
		b.reply(message, fmt.Sprintf("Could not map the town's channels (%v)", err))
	}
}

// mapRooms records the IDs of the channels named in villageCodeLookup in Game.Rooms.
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
