package bot

import (
	"net/http"
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestVillageAddAndRemove(t *testing.T) {
	h := newHarness(t)
	h.withGame(map[string]string{"1": "Alice"})
	h.users["8"] = &discordgo.User{ID: "8", Username: "Robot", Bot: true}
	h.member("3", "Carol")

	wantReply(t, h.runOne("!botc village add <@1> <@9> <@8> <@3>"),
		"Added 1 player(s): Carol",
		"Skipped: Alice (already in the village), 9 (the Storyteller), Robot (bot)")
	if !h.fake.sent("PUT /api/v9/guilds/guild/members/3/roles/role-player") {
		t.Error("Carol wasn't given the player role")
	}
	if h.fake.sent("PUT /api/v9/guilds/guild/members/1/roles/role-player") {
		t.Error("Alice was given the player role again")
	}
	wantReply(t, h.runOne("!botc village list"), "The village has 2 player(s):\n1. Alice\n2. Carol")

	wantReply(t, h.runOne("!botc village remove <@1> <@7>"), "Removed 1 player(s): Alice", "Skipped: 7 (not in the village)")
	if !h.fake.sent("DELETE /api/v9/guilds/guild/members/1/roles/role-player") {
		t.Error("Alice's player role wasn't taken away")
	}
	wantReply(t, h.runOne("!botc village list"), "The village has 1 player(s):\n1. Carol")
}

func TestVillageAddWarnsIfTheRoleIsMissing(t *testing.T) {
	h := newHarness(t)
	h.withGame(nil)
	h.fake.routes = nil // the server has no roles
	h.fake.on(http.StatusOK, `[]`, "GET /api/v9/guilds/guild/roles")

	wantReply(t, h.runOne("!botc village add <@3>"), "Added 1 player(s): 3", `Warning: the "BoTC-Player" role could not be fully updated`)
}
