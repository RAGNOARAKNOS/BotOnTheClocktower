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
	Players        map[string]*Player // the village: user ID → player
	Rooms          map[string]string  // room code (see villageCodeLookup) → channel ID; found by register, every room

	gather *gatherCountdown // the running `gather` countdown, or nil
}

// Player is a player in the village. Everything the game knows about a player
// lives here, so taking them out of the village forgets all of it.
type Player struct {
	Name string // display name

	// Alive belongs to the player, not the character: a new character doesn't bring
	// anyone back to life.
	Alive bool
	// AnnouncedAlive is Alive as of the last `character announce`. While the two
	// differ, a death or revival is waiting to be announced.
	AnnouncedAlive bool
	// GhostVoteUsed records whether a dead player has spent their one remaining vote.
	GhostVoteUsed bool

	// Character is the character the Storyteller has given the player, or nil.
	Character *Character
}

// newGame starts a game with empty player and room lists.
func newGame(guildID, adminChannelID, gameChannelID, storytellerID string) *Game {
	return &Game{
		GuildID:        guildID,
		AdminChannelID: adminChannelID,
		GameChannelID:  gameChannelID,
		StorytellerID:  storytellerID,
		Players:        make(map[string]*Player),
		Rooms:          make(map[string]string),
	}
}

// The game's rules. These change only the Game, never Discord, so they can be
// unit-tested; the command handlers do the Discord side and build the replies.
// The rules that take a user ID expect a village player; the handlers check first.

// ReplacePlayers makes players (user ID → display name) the whole village and returns
// who was dropped. Players who stay keep their character and life state.
func (g *Game) ReplacePlayers(players map[string]string) (dropped map[string]string) {
	dropped = make(map[string]string)
	for id, p := range g.Players {
		if _, ok := players[id]; !ok {
			dropped[id] = p.Name
			delete(g.Players, id)
		}
	}
	for id, name := range players {
		g.AddPlayer(id, name)
	}
	return dropped
}

// AddPlayer puts a player in the village, alive and without a character. A player
// already in the village only takes the new name.
func (g *Game) AddPlayer(userID, name string) {
	if p, ok := g.Players[userID]; ok {
		p.Name = name
		return
	}
	g.Players[userID] = &Player{Name: name, Alive: true, AnnouncedAlive: true}
}

// RemovePlayer takes a player out of the village, with their character.
func (g *Game) RemovePlayer(userID string) {
	delete(g.Players, userID)
}

// Assign gives a player a character, unsent, and returns the character it replaces,
// if any. The player's life and ghost vote don't change.
func (g *Game) Assign(userID string, c *Character) (previous *Character) {
	p := g.Players[userID]
	previous = p.Character
	c.Sent = false
	p.Character = c
	return previous
}

// ClearCharacter removes a player's character; they stay in the village, alive or dead.
func (g *Game) ClearCharacter(userID string) {
	if p, ok := g.Players[userID]; ok {
		p.Character = nil
	}
}

// MarkSent records that a player has been sent their character.
func (g *Game) MarkSent(userID string) {
	g.Players[userID].Character.Sent = true
}

// SetTeam moves a player's character to team and marks it unsent, so `send` tells
// them. It returns false if the character was already on that team.
func (g *Game) SetTeam(userID string, team Team) bool {
	c := g.Players[userID].Character
	if c.Team == team {
		return false
	}
	c.Team = team
	c.Sent = false
	return true
}

// SetAlive kills or revives a player; reviving gives back the ghost vote. It returns
// false if the player was already in that state. Nothing is public until MarkAnnounced.
func (g *Game) SetAlive(userID string, alive bool) bool {
	p := g.Players[userID]
	if p.Alive == alive {
		return false
	}
	p.Alive = alive
	if alive {
		p.GhostVoteUsed = false
	}
	return true
}

// ToggleGhostVote switches a dead player's ghost vote between used and available.
// It returns false, changing nothing, if the player is alive.
func (g *Game) ToggleGhostVote(userID string) bool {
	p := g.Players[userID]
	if p.Alive {
		return false
	}
	p.GhostVoteUsed = !p.GhostVoteUsed
	return true
}

// PendingLifeChanges returns the names of the players who have died or come back
// since the last announcement, each sorted by name.
func (g *Game) PendingLifeChanges() (died, revived []string) {
	for _, p := range g.Players {
		switch {
		case p.Alive == p.AnnouncedAlive:
		case p.Alive:
			revived = append(revived, p.Name)
		default:
			died = append(died, p.Name)
		}
	}
	slices.Sort(died)
	slices.Sort(revived)
	return died, revived
}

// MarkAnnounced records every player's current life state as announced.
func (g *Game) MarkAnnounced() {
	for _, p := range g.Players {
		p.AnnouncedAlive = p.Alive
	}
}

// villageChannels returns the village's voice channel IDs: Town Square first, then
// the other rooms in room-code order, each once.
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
	if p, ok := g.Players[userID]; ok {
		return p.Name
	}
	return fallback
}

// sortedPlayerIDs returns the village players' IDs, ordered by display name.
func (g *Game) sortedPlayerIDs() []string {
	return slices.SortedFunc(maps.Keys(g.Players), func(a, b string) int {
		return strings.Compare(g.Players[a].Name, g.Players[b].Name)
	})
}

// villageCodeLookup is every room in the village, by code: the voice channels the
// server must have, with exactly these names, before a game can be registered.
var villageCodeLookup = map[string]string{
	"TS": "Town Square",
	"CA": "Cathedral",
	"CF": "Campfire",
	"PS": "Potion Shop",
	"TW": "Tower",
	"RS": "Riverside",
	"SC": "Storyteller's Corner",
}

// villageRooms maps each room code to the first voice channel with the room's name.
// Text channels and categories with a room's name don't count.
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

// missingRooms returns the names of the village rooms that rooms (from villageRooms)
// has no channel for, sorted.
func missingRooms(rooms map[string]string) []string {
	var missing []string
	for code, name := range villageCodeLookup {
		if _, ok := rooms[code]; !ok {
			missing = append(missing, name)
		}
	}
	slices.Sort(missing)
	return missing
}
