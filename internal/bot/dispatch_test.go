package bot

import (
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/bwmarrin/discordgo"
)

// fakeDiscord records the REST requests the bot makes and answers each with an empty message.
type fakeDiscord struct {
	mu       sync.Mutex
	requests []string // "METHOD path body"
}

func (f *fakeDiscord) RoundTrip(r *http.Request) (*http.Response, error) {
	body := ""
	if r.Body != nil {
		b, _ := io.ReadAll(r.Body)
		body = string(b)
	}
	f.mu.Lock()
	f.requests = append(f.requests, r.Method+" "+r.URL.Path+" "+body)
	f.mu.Unlock()
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"id":"1","channel_id":"c"}`)),
		Request:    r,
	}, nil
}

// testBot returns a Bot whose Discord session talks to a fakeDiscord.
func testBot(t *testing.T) (*Bot, *fakeDiscord) {
	t.Helper()
	session, err := discordgo.New("Bot test-token")
	if err != nil {
		t.Fatal(err)
	}
	fake := &fakeDiscord{}
	session.Client = &http.Client{Transport: fake}
	return &Bot{discord: session}, fake
}

func messageFrom(author, guild, channel, content string) *discordgo.MessageCreate {
	return &discordgo.MessageCreate{Message: &discordgo.Message{
		ID: "m", Content: content, ChannelID: channel, GuildID: guild,
		Author: &discordgo.User{ID: author},
	}}
}

func TestPingResponds(t *testing.T) {
	tests := []struct {
		name string
		game *Game
		msg  *discordgo.MessageCreate
	}{
		{"no game, anyone", nil, messageFrom("anyone", "guild", "general", "!botc ping")},
		{"game, Storyteller in admin", newGame("guild", "admin", "town", "st"), messageFrom("st", "guild", "admin", "!botc ping")},
		{"game, Storyteller elsewhere", newGame("guild", "admin", "town", "st"), messageFrom("st", "guild", "general", "!botc ping")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, fake := testBot(t)
			b.game = tt.game
			b.newMessage(b.discord, tt.msg)

			for _, req := range fake.requests {
				if strings.Contains(req, "/messages") && strings.Contains(req, "pong") {
					return
				}
			}
			t.Errorf("no pong sent; requests: %q", fake.requests)
		})
	}
}
