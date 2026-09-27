package bot

import (
	"net/http"
	"slices"
	"strings"
	"testing"
)

func TestPendingLifeChanges(t *testing.T) {
	g := newGame("guild", "admin", "town", "st")
	g.Players = map[string]*Player{
		"1": {Name: "Alice", Alive: true, AnnouncedAlive: true},  // unchanged
		"2": {Name: "Bob", Alive: false, AnnouncedAlive: true},   // killed, not announced
		"3": {Name: "Carol", Alive: true, AnnouncedAlive: false}, // revived, not announced
		"4": {Name: "Dave", Alive: false, AnnouncedAlive: false}, // death already announced
		"5": {Name: "Erin", Alive: false, AnnouncedAlive: true},  // killed, then character cleared
		"6": {Name: "Frank", Alive: true, AnnouncedAlive: true},  // unchanged, no character
	}

	died, revived := g.PendingLifeChanges()
	if !slices.Equal(died, []string{"Bob", "Erin"}) || !slices.Equal(revived, []string{"Carol"}) {
		t.Errorf("got died %v revived %v, want [Bob Erin] and [Carol]", died, revived)
	}

	// Killing then reviving before an announcement leaves nothing to announce.
	g.Players["2"].Alive = true
	g.Players["3"].Alive = false
	g.Players["5"].Alive = true
	died, revived = g.PendingLifeChanges()
	if len(died) != 0 || len(revived) != 0 {
		t.Errorf("got died %v revived %v, want nothing pending", died, revived)
	}
}

// cannotDM is Discord's answer when a user doesn't accept DMs from the server.
const cannotDM = `{"code":50007,"message":"Cannot send messages to this user"}`

func TestCharacterSendReportsFailedDMs(t *testing.T) {
	h := newHarness(t)
	h.withGame(map[string]string{"1": "Alice", "2": "Bob", "3": "Carol"})
	h.runOne("!botc character assign <@1> Monk")
	h.runOne("!botc character assign <@2> evil Imp")
	h.fake.on(http.StatusForbidden, cannotDM, "POST /api/v9/users/@me/channels", `"recipient_id":"2"`)

	wantReply(t, h.runOne("!botc character send"),
		"Sent 1 character(s): Alice",
		"Failed: Bob (they don't accept DMs from this server)",
		"Village players with no character yet: Carol")

	// Only the failed character is still unsent, so sending again retries just Bob.
	wantReply(t, h.runOne("!botc grimoire"), "Alice: Monk (Good) · Alive · sent", "Bob: Imp (Evil) · Alive · not sent")
	wantReply(t, h.runOne("!botc character send"), "Sent 0 character(s)", "Failed: Bob")
}

func TestCharacterAnnounceKeepsChangesIfThePostFails(t *testing.T) {
	h := newHarness(t)
	h.withGame(map[string]string{"1": "Alice"})
	h.runOne("!botc character assign <@1> Monk")
	h.runOne("!botc character kill <@1>")
	h.fake.on(http.StatusForbidden, `{"code":50013,"message":"Missing Permissions"}`, "POST /api/v9/channels/"+testTownSquare+"/messages")

	wantReply(t, h.runOne("!botc character announce"), "Could not post in <#town>", "Nothing was marked as announced.")
	wantReply(t, h.runOne("!botc grimoire"), "death not announced")
}

func TestCharacterLifeCommands(t *testing.T) {
	h := newHarness(t)
	h.withGame(map[string]string{"1": "Alice", "2": "Bob"})
	h.runOne("!botc character assign <@1> Monk")

	wantReply(t, h.runOne("!botc character kill <@1> <@2>"),
		"Now dead: 1 player(s): Alice.", "Skipped: Bob (no character)", "1 change(s) waiting to be announced")
	wantReply(t, h.runOne("!botc character kill <@1>"), "Now dead: 0 player(s).", "Skipped: Alice (already dead)")
	wantReply(t, h.runOne("!botc character ghostvote <@1>"), "Alice: ghost vote used")
	wantReply(t, h.runOne("!botc character revive <@1>"), "Now alive: 1 player(s): Alice.", "Nothing is waiting to be announced.")
	wantReply(t, h.runOne("!botc character ghostvote <@1>"), "No ghost votes changed.", "Skipped: Alice (alive, so no ghost vote)")
	wantReply(t, h.runOne("!botc character team <@1> evil"), "Now Evil: 1 player(s): Alice.")
	wantReply(t, h.runOne("!botc character team <@1> EVIL"), "Now Evil: 0 player(s)", "Skipped: Alice (already Evil)")
	wantReply(t, h.runOne("!botc character clear <@1> <@2>"), "Cleared 1 character(s): Alice", "Skipped: Bob (no character)")

	if got := h.runOne("!botc grimoire"); strings.Contains(got, "Monk") {
		t.Errorf("cleared character still in the grimoire: %q", got)
	}
}
