package bot

import (
	"maps"
	"testing"
)

// testGame returns a game with two village players, Alice (with a character) and Bob (without).
func testGame() *Game {
	g := newGame("guild", "admin", "town", "st")
	g.Players["a"] = "Alice"
	g.Players["b"] = "Bob"
	g.Assign("a", &Character{Name: "Monk", Team: TeamGood})
	return g
}

func TestReplacePlayers(t *testing.T) {
	g := testGame()
	dropped := g.ReplacePlayers(map[string]string{"b": "Bob", "c": "Carol"})

	if want := map[string]string{"a": "Alice"}; !maps.Equal(dropped, want) {
		t.Errorf("dropped = %v, want %v", dropped, want)
	}
	if _, ok := g.Characters["a"]; ok {
		t.Error("a dropped player kept their character")
	}
	if want := map[string]string{"b": "Bob", "c": "Carol"}; !maps.Equal(g.Players, want) {
		t.Errorf("Players = %v, want %v", g.Players, want)
	}
}

func TestRemovePlayer(t *testing.T) {
	g := testGame()
	g.RemovePlayer("a")
	if _, ok := g.Players["a"]; ok {
		t.Error("player still in the village")
	}
	if _, ok := g.Characters["a"]; ok {
		t.Error("removed player kept their character")
	}
}

func TestAssign(t *testing.T) {
	g := testGame()

	c := g.Characters["a"]
	if !c.Alive || !c.AnnouncedAlive || c.Sent || c.GhostVoteUsed {
		t.Fatalf("new character = %+v, want alive, announced alive, unsent, ghost vote unused", *c)
	}

	c.Sent = true
	g.SetAlive("a", false)
	g.ToggleGhostVote("a")
	previous := g.Assign("a", &Character{Name: "Imp", Team: TeamEvil})
	if previous != c {
		t.Errorf("Assign returned %v, want the replaced character", previous)
	}
	got := g.Characters["a"]
	if got.Name != "Imp" || got.Sent || got.Alive || !got.AnnouncedAlive || !got.GhostVoteUsed {
		t.Errorf("replacement = %+v, want Imp, unsent, still dead (not yet announced), ghost vote still used", *got)
	}
}

func TestSetTeam(t *testing.T) {
	g := testGame()
	g.Characters["a"].Sent = true

	if g.SetTeam("a", TeamGood) {
		t.Error("SetTeam to the same team reported a change")
	}
	if !g.SetTeam("a", TeamEvil) {
		t.Fatal("SetTeam to a new team reported no change")
	}
	if c := g.Characters["a"]; c.Team != TeamEvil || c.Sent {
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
	if !g.ToggleGhostVote("a") || !g.Characters["a"].GhostVoteUsed {
		t.Fatal("a dead player's ghost vote wasn't used")
	}
	if died, _ := g.PendingLifeChanges(); len(died) != 1 || died[0] != "Alice" {
		t.Errorf("pending deaths = %v, want [Alice]", died)
	}

	if !g.SetAlive("a", true) {
		t.Fatal("reviving a dead player reported no change")
	}
	if g.Characters["a"].GhostVoteUsed {
		t.Error("reviving didn't give the ghost vote back")
	}

	g.SetAlive("a", false)
	g.MarkAnnounced()
	if died, revived := g.PendingLifeChanges(); len(died)+len(revived) != 0 {
		t.Errorf("after MarkAnnounced: died %v, revived %v, want nothing pending", died, revived)
	}
}
