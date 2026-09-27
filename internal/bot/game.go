package bot

import (
	"maps"
	"slices"
	"strings"

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

	gather *gatherCountdown // the running `gather` countdown, or nil
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

// AddPlayer puts a player in the village.
func (g *Game) AddPlayer(userID, name string) {
	g.Players[userID] = name
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

// ClearCharacter removes a player's character; they stay in the village.
func (g *Game) ClearCharacter(userID string) {
	delete(g.Characters, userID)
}

// MarkSent records that a player has been sent their character.
func (g *Game) MarkSent(userID string) {
	g.Characters[userID].Sent = true
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

// PendingLifeChanges returns the names of the players who have died or come back
// since the last announcement, each sorted by name.
func (g *Game) PendingLifeChanges() (died, revived []string) {
	for id, name := range g.Players {
		c, ok := g.Characters[id]
		if !ok || c.Alive == c.AnnouncedAlive {
			continue
		}
		if c.Alive {
			revived = append(revived, name)
		} else {
			died = append(died, name)
		}
	}
	slices.Sort(died)
	slices.Sort(revived)
	return died, revived
}

// MarkAnnounced records every character's current life state as announced.
func (g *Game) MarkAnnounced() {
	for _, c := range g.Characters {
		c.AnnouncedAlive = c.Alive
	}
}

// villageChannels returns the village's voice channel IDs: Town Square first, then
// the mapped rooms in room-code order, each once.
func (g *Game) villageChannels() []string {
	channels := []string{g.GameChannelID}
	for _, code := range slices.Sorted(maps.Keys(g.Rooms)) {
		if id := g.Rooms[code]; !slices.Contains(channels, id) {
			channels = append(channels, id)
		}
	}
	return channels
}

// playerName returns the player's village name, or fallback if they aren't in the village.
func (g *Game) playerName(userID, fallback string) string {
	if name, ok := g.Players[userID]; ok {
		return name
	}
	return fallback
}

// sortedPlayerIDs returns the village players' IDs, ordered by display name.
func (g *Game) sortedPlayerIDs() []string {
	return slices.SortedFunc(maps.Keys(g.Players), func(a, b string) int {
		return strings.Compare(g.Players[a], g.Players[b])
	})
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

// villageRooms maps each room code to the first voice channel with the room's name,
// the same rule findVoiceChannelID uses, so Town Square matches the game channel.
func villageRooms(channels []*discordgo.Channel) map[string]string {
	rooms := make(map[string]string)
	for _, ch := range channels {
		if ch.Type != discordgo.ChannelTypeGuildVoice {
			continue
		}
		for code, name := range villageCodeLookup {
			if _, found := rooms[code]; !found && ch.Name == name {
				rooms[code] = ch.ID
			}
		}
	}
	return rooms
}
