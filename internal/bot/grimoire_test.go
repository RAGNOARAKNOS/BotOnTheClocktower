package bot

import "testing"

func TestGrimoireSummary(t *testing.T) {
	g := newGame("guild", "admin", "town", "st")
	g.Players = map[string]*Player{
		"1": {Name: "Alice", Alive: true, AnnouncedAlive: true, Character: &Character{Team: TeamEvil}},
		"2": {Name: "Bob", Alive: false, AnnouncedAlive: true, Character: &Character{Team: TeamGood}},
		"3": {Name: "Carol", Alive: true, AnnouncedAlive: true, Character: &Character{Team: TeamGood}},
		"4": {Name: "Dave", Alive: true, AnnouncedAlive: true},
	}

	got := grimoireSummary(g)
	want := "Alive 2/3 · Good 2 · Evil 1 · 1 without a character · 1 change(s) not yet announced"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestGrimoireLine(t *testing.T) {
	monk := func() *Character { return &Character{Name: "Monk", Team: TeamGood} }
	tests := []struct {
		name   string
		player Player
		want   string
	}{
		{"alive, unsent", Player{Alive: true, AnnouncedAlive: true, Character: monk()}, "Monk (Good) · Alive · not sent"},
		{"sent, with guidance", Player{Alive: true, AnnouncedAlive: true, Character: &Character{Name: "Imp", Team: TeamEvil, Sent: true, Guidance: "Kill"}},
			"Imp (Evil) · Alive · sent · has guidance"},
		{"dead, not announced", Player{AnnouncedAlive: true, GhostVoteUsed: true, Character: monk()},
			"Monk (Good) · Dead, ghost vote used · not sent · death not announced"},
		{"revived, not announced", Player{Alive: true, Character: monk()}, "Monk (Good) · Alive · not sent · revival not announced"},
		{"no character", Player{Alive: true, AnnouncedAlive: true}, "none"},
		{"no character, died with one", Player{AnnouncedAlive: true}, "none · Dead, ghost vote available · death not announced"},
	}
	for _, tt := range tests {
		if got := grimoireLine(&tt.player); got != tt.want {
			t.Errorf("%s: got %q, want %q", tt.name, got, tt.want)
		}
	}
}
