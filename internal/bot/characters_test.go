package bot

import (
	"slices"
	"testing"
)

func TestPendingLifeChanges(t *testing.T) {
	players := map[string]string{"1": "Alice", "2": "Bob", "3": "Carol", "4": "Dave", "5": "Erin"}
	chars := map[string]*Character{
		"1": {Name: "Imp", Alive: true, AnnouncedAlive: true},   // unchanged
		"2": {Name: "Monk", Alive: false, AnnouncedAlive: true}, // killed, not announced
		"3": {Name: "Chef", Alive: true, AnnouncedAlive: false}, // revived, not announced
		"4": {Name: "Spy", Alive: false, AnnouncedAlive: false}, // death already announced
		// Erin has no character
	}

	died, revived := pendingLifeChanges(players, chars)
	if !slices.Equal(died, []string{"Bob"}) || !slices.Equal(revived, []string{"Carol"}) {
		t.Errorf("got died %v revived %v, want [Bob] and [Carol]", died, revived)
	}

	// Killing then reviving before an announcement leaves nothing to announce.
	chars["2"].Alive = true
	chars["3"].Alive = false
	died, revived = pendingLifeChanges(players, chars)
	if len(died) != 0 || len(revived) != 0 {
		t.Errorf("got died %v revived %v, want nothing pending", died, revived)
	}
}
