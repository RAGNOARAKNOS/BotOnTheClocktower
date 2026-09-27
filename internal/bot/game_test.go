package bot

import (
	"maps"
	"slices"
	"testing"

	"github.com/bwmarrin/discordgo"
)

// testGame returns a game with two village players, Alice (with a character) and Bob (without).
func testGame() *Game {
	g := newGame("guild", "admin", "town", "st")
	g.AddPlayer("a", "Alice")
	g.AddPlayer("b", "Bob")
	g.Assign("a", &Character{Name: "Monk", Team: TeamGood})
	return g
}

// playerNames returns the village as user ID → name.
func playerNames(g *Game) map[string]string {
	names := make(map[string]string)
	for id, p := range g.Players {
		names[id] = p.Name
	}
	return names
}

func TestReplacePlayers(t *testing.T) {
	g := testGame()
	g.Assign("b", &Character{Name: "Imp", Team: TeamEvil})
	g.SetAlive("b", false)
	dropped := g.ReplacePlayers(map[string]string{"b": "Bobby", "c": "Carol"})

	if want := map[string]string{"a": "Alice"}; !maps.Equal(dropped, want) {
		t.Errorf("dropped = %v, want %v", dropped, want)
	}
	if want := map[string]string{"b": "Bobby", "c": "Carol"}; !maps.Equal(playerNames(g), want) {
		t.Errorf("Players = %v, want %v", playerNames(g), want)
	}
	if b := g.Players["b"]; b.Character == nil || b.Character.Name != "Imp" || b.Alive {
		t.Errorf("staying player = %+v, want still the Imp and dead", *b)
	}
	if c := g.Players["c"]; c.Character != nil || !c.Alive || !c.AnnouncedAlive {
		t.Errorf("new player = %+v, want alive with no character", *c)
	}
}

func TestRemovePlayer(t *testing.T) {
	g := testGame()
	g.RemovePlayer("a")
	if _, ok := g.Players["a"]; ok {
		t.Error("player still in the village")
	}
}

func TestAddPlayerAndClearCharacter(t *testing.T) {
	g := testGame()
	g.AddPlayer("c", "Carol")
	if c := g.Players["c"]; c.Name != "Carol" || !c.Alive || c.Character != nil {
		t.Errorf("Players[c] = %+v, want Carol, alive, with no character", *c)
	}

	g.ClearCharacter("a")
	if a, ok := g.Players["a"]; !ok || a.Character != nil {
		t.Error("clearing a character should keep the player and remove the character")
	}

	// Adding a player again only changes their name.
	g.Assign("a", &Character{Name: "Monk"})
	g.AddPlayer("a", "Alicia")
	if a := g.Players["a"]; a.Name != "Alicia" || a.Character == nil {
		t.Errorf("re-added player = %+v, want Alicia with the Monk", *a)
	}
}

func TestMarkSent(t *testing.T) {
	g := testGame()
	g.MarkSent("a")
	if !g.Players["a"].Character.Sent {
		t.Error("character not marked sent")
	}
}

func TestSortedPlayerIDs(t *testing.T) {
	g := testGame()
	g.AddPlayer("c", "Aaron")
	if got, want := g.sortedPlayerIDs(), []string{"c", "a", "b"}; !slices.Equal(got, want) {
		t.Errorf("sortedPlayerIDs() = %v, want %v", got, want)
	}
}

func TestAssign(t *testing.T) {
	g := testGame()
	a := g.Players["a"]
	c := a.Character
	if !a.Alive || !a.AnnouncedAlive || c.Sent || a.GhostVoteUsed {
		t.Fatalf("new player = %+v with %+v, want alive, announced alive, unsent, ghost vote unused", *a, *c)
	}

	c.Sent = true
	g.SetAlive("a", false)
	g.ToggleGhostVote("a")
	previous := g.Assign("a", &Character{Name: "Imp", Team: TeamEvil})
	if previous != c {
		t.Errorf("Assign returned %v, want the replaced character", previous)
	}
	if got := a.Character; got.Name != "Imp" || got.Sent {
		t.Errorf("replacement = %+v, want Imp, unsent", *got)
	}
	if a.Alive || !a.AnnouncedAlive || !a.GhostVoteUsed {
		t.Errorf("after reassigning: %+v, want still dead (not yet announced), ghost vote still used", *a)
	}
}

// TestNewCharacterDoesNotRevive checks a dead player stays dead when their character
// is cleared and they're given another (decided 2026-09-27, characters.md).
func TestNewCharacterDoesNotRevive(t *testing.T) {
	g := testGame()
	g.SetAlive("a", false)
	g.ClearCharacter("a")
	if died, _ := g.PendingLifeChanges(); !slices.Equal(died, []string{"Alice"}) {
		t.Errorf("pending deaths after clearing = %v, want [Alice]", died)
	}

	g.Assign("a", &Character{Name: "Imp", Team: TeamEvil})
	if g.Players["a"].Alive {
		t.Error("a new character brought a dead player back to life")
	}
}

func TestSetTeam(t *testing.T) {
	g := testGame()
	g.Players["a"].Character.Sent = true

	if g.SetTeam("a", TeamGood) {
		t.Error("SetTeam to the same team reported a change")
	}
	if !g.SetTeam("a", TeamEvil) {
		t.Fatal("SetTeam to a new team reported no change")
	}
	if c := g.Players["a"].Character; c.Team != TeamEvil || c.Sent {
		t.Errorf("after SetTeam: %+v, want Evil and unsent", *c)
	}
}

func TestSetAliveAndGhostVote(t *testing.T) {
	g := testGame()

	if g.SetAlive("a", true) {
		t.Error("reviving a living player reported a change")
	}
	if g.ToggleGhostVote("a") {
		t.Error("a living player's ghost vote changed")
	}

	if !g.SetAlive("a", false) {
		t.Fatal("killing a living player reported no change")
	}
	if !g.ToggleGhostVote("a") || !g.Players["a"].GhostVoteUsed {
		t.Fatal("a dead player's ghost vote wasn't used")
	}
	if died, _ := g.PendingLifeChanges(); len(died) != 1 || died[0] != "Alice" {
		t.Errorf("pending deaths = %v, want [Alice]", died)
	}

	if !g.SetAlive("a", true) {
		t.Fatal("reviving a dead player reported no change")
	}
	if g.Players["a"].GhostVoteUsed {
		t.Error("reviving didn't give the ghost vote back")
	}

	g.SetAlive("a", false)
	g.MarkAnnounced()
	if died, revived := g.PendingLifeChanges(); len(died)+len(revived) != 0 {
		t.Errorf("after MarkAnnounced: died %v, revived %v, want nothing pending", died, revived)
	}
}

func TestVillageChannels(t *testing.T) {
	g := testGame()
	if got, want := g.villageChannels(), []string{"town"}; !slices.Equal(got, want) {
		t.Errorf("unmapped: villageChannels() = %v, want %v", got, want)
	}

	g.Rooms = map[string]string{"TS": "town", "TW": "tower", "CA": "cathedral"}
	if got, want := g.villageChannels(), []string{"town", "cathedral", "tower"}; !slices.Equal(got, want) {
		t.Errorf("mapped: villageChannels() = %v, want %v", got, want)
	}
}

func TestVillageRooms(t *testing.T) {
	voice, text, category := discordgo.ChannelTypeGuildVoice, discordgo.ChannelTypeGuildText, discordgo.ChannelTypeGuildCategory
	channels := []*discordgo.Channel{
		{ID: "cat", Name: "Tower", Type: category},
		{ID: "chat", Name: "Tower", Type: text},
		{ID: "town", Name: "Town Square", Type: voice},
		{ID: "tower", Name: "Tower", Type: voice},
		{ID: "tower2", Name: "Tower", Type: voice},
		{ID: "general", Name: "General", Type: voice},
	}
	want := map[string]string{"TS": "town", "TW": "tower"}
	if got := villageRooms(channels); !maps.Equal(got, want) {
		t.Errorf("villageRooms() = %v, want %v", got, want)
	}
}
