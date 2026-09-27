package bot

import (
	"encoding/json"
	"maps"
	"net/http"
	"slices"
	"testing"

	"github.com/bwmarrin/discordgo"
)

// serveChannels makes the fake Discord list these channels for the test server.
func serveChannels(t *testing.T, h *harness, channels []*discordgo.Channel) {
	t.Helper()
	body, err := json.Marshal(channels)
	if err != nil {
		t.Fatal(err)
	}
	h.fake.on(http.StatusOK, string(body), "GET /api/v9/guilds/"+testGuild+"/channels")
}

// villageVoiceChannels returns a voice channel, with ID "room-<code>", for each
// village room except those named in skip.
func villageVoiceChannels(skip ...string) []*discordgo.Channel {
	var channels []*discordgo.Channel
	for code, name := range villageCodeLookup {
		if !slices.Contains(skip, name) {
			channels = append(channels, &discordgo.Channel{ID: "room-" + code, Name: name, Type: discordgo.ChannelTypeGuildVoice})
		}
	}
	return channels
}

func TestRegisterRecordsEveryRoom(t *testing.T) {
	h := newHarness(t)
	// A text channel with a room's name doesn't count, even listed first.
	channels := append([]*discordgo.Channel{{ID: "chat", Name: "Town Square", Type: discordgo.ChannelTypeGuildText}}, villageVoiceChannels()...)
	serveChannels(t, h, channels)

	wantReply(t, h.runOne("!botc register"), "Game registered.", "<#room-TS> is the game channel")
	g := h.b.game
	if g == nil {
		t.Fatal("no game registered")
	}
	want := make(map[string]string)
	for code := range villageCodeLookup {
		want[code] = "room-" + code
	}
	if g.GameChannelID != "room-TS" || !maps.Equal(g.Rooms, want) {
		t.Errorf("game channel %q, rooms %v; want room-TS and %v", g.GameChannelID, g.Rooms, want)
	}
	if !h.fake.sent("POST /api/v9/channels/room-TS/messages", "A new game has begun") {
		t.Error("no announcement in Town Square")
	}
}

func TestRegisterRefusedWithoutEveryRoom(t *testing.T) {
	h := newHarness(t)
	// Tower exists only as a text channel, which doesn't count.
	channels := append(villageVoiceChannels("Campfire", "Tower"), &discordgo.Channel{ID: "tower-chat", Name: "Tower", Type: discordgo.ChannelTypeGuildText})
	serveChannels(t, h, channels)

	wantReply(t, h.runOne("!botc register"),
		"Could not find these village voice channels: Campfire, Tower.", "This command will not execute")
	if h.b.game != nil {
		t.Error("a game was registered")
	}
	if h.fake.sent("POST /api/v9/channels/") {
		t.Error("something was posted for a game that wasn't registered")
	}
}

func TestRegisterRefusedIfChannelsCantBeRead(t *testing.T) {
	h := newHarness(t)
	h.fake.on(http.StatusForbidden, `{"code":50001,"message":"Missing Access"}`, "GET /api/v9/guilds/"+testGuild+"/channels")

	wantReply(t, h.runOne("!botc register"), "Could not read the server's channels", "This command will not execute")
	if h.b.game != nil {
		t.Error("a game was registered")
	}
}
