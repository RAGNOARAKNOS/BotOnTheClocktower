package bot

import (
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
)

// fakeDiscord stands in for Discord's REST API. It records every request and answers
// with the first route that matches, or with an empty message.
type fakeDiscord struct {
	mu       sync.Mutex
	requests []string // "METHOD path body"
	routes   []fakeRoute
}

// fakeRoute answers every request whose record ("METHOD path body") contains all of match.
type fakeRoute struct {
	match  []string
	status int
	body   string
}

// on answers the requests whose record contains all of match with status and body.
// Routes added first win.
func (f *fakeDiscord) on(status int, body string, match ...string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.routes = append(f.routes, fakeRoute{match, status, body})
}

func (f *fakeDiscord) RoundTrip(r *http.Request) (*http.Response, error) {
	body := ""
	if r.Body != nil {
		b, _ := io.ReadAll(r.Body)
		body = string(b)
	}
	record := r.Method + " " + r.URL.Path + " " + body

	f.mu.Lock()
	f.requests = append(f.requests, record)
	status, answer := http.StatusOK, `{"id":"1","channel_id":"c"}`
	for _, route := range f.routes {
		if containsAll(record, route.match) {
			status, answer = route.status, route.body
			break
		}
	}
	f.mu.Unlock()

	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(answer)),
		Request:    r,
	}, nil
}

// sent reports whether a request containing all of match has been made.
func (f *fakeDiscord) sent(match ...string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, record := range f.requests {
		if containsAll(record, match) {
			return true
		}
	}
	return false
}

// waitFor waits up to timeout for a request containing all of match.
func (f *fakeDiscord) waitFor(timeout time.Duration, match ...string) bool {
	for deadline := time.Now().Add(timeout); time.Now().Before(deadline); time.Sleep(5 * time.Millisecond) {
		if f.sent(match...) {
			return true
		}
	}
	return f.sent(match...)
}

func containsAll(s string, parts []string) bool {
	for _, part := range parts {
		if !strings.Contains(s, part) {
			return false
		}
	}
	return true
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

// The test server: its ID, admin channel, Town Square and Storyteller.
const (
	testGuild       = "guild"
	testAdmin       = "admin"
	testTownSquare  = "town"
	testStoryteller = "9"
)

// harness runs commands against a Bot whose Discord session talks to a fakeDiscord,
// and whose state cache holds the test server.
type harness struct {
	t    *testing.T
	b    *Bot
	fake *fakeDiscord
	// users are the users that mentions resolve to; anyone else is a plain user
	// whose username is their ID.
	users map[string]*discordgo.User
}

// newHarness returns a harness for the test server, which has both game roles.
func newHarness(t *testing.T) *harness {
	t.Helper()
	b, fake := testBot(t)
	if err := b.discord.State.GuildAdd(&discordgo.Guild{ID: testGuild, Name: "Test Server"}); err != nil {
		t.Fatal(err)
	}
	fake.on(http.StatusOK, `[{"id":"role-st","name":"BoTC-StoryTeller"},{"id":"role-player","name":"BoTC-Player"}]`,
		"GET /api/v9/guilds/"+testGuild+"/roles")
	return &harness{t: t, b: b, fake: fake, users: make(map[string]*discordgo.User)}
}

// withGame registers a game on the test server with these players (user ID → name).
func (h *harness) withGame(players map[string]string) *Game {
	h.b.game = newGame(testGuild, testAdmin, testTownSquare, testStoryteller)
	for id, name := range players {
		h.b.game.AddPlayer(id, name)
		h.member(id, name)
	}
	return h.b.game
}

// member puts a server member with this nickname in the state cache.
func (h *harness) member(userID, nick string) {
	h.t.Helper()
	user := h.user(userID)
	if err := h.b.discord.State.MemberAdd(&discordgo.Member{GuildID: testGuild, User: user, Nick: nick}); err != nil {
		h.t.Fatal(err)
	}
}

// user returns the user with this ID.
func (h *harness) user(id string) *discordgo.User {
	if u, ok := h.users[id]; ok {
		return u
	}
	return &discordgo.User{ID: id, Username: id}
}

// run runs a `!botc` command from the Storyteller in the admin channel and returns
// the replies. As in a real message, the mentions are the users mentioned in content.
func (h *harness) run(content string) []string {
	return h.runAs(testStoryteller, testAdmin, content)
}

// runAs runs a `!botc` command from author in channel and returns the replies.
func (h *harness) runAs(author, channel, content string) []string {
	var replies []string
	record := func(text string) { replies = append(replies, text) }
	var mentions []*discordgo.User
	for _, id := range mentionIDsIn(content) {
		mentions = append(mentions, h.user(id))
	}
	h.b.runCommand(&request{
		authorID: author, guildID: testGuild, channelID: channel,
		words: strings.Fields(content), content: content, mentions: mentions,
		reply: record, say: record,
	})
	return replies
}

// runOne runs a `!botc` command like run and returns its only reply, failing the
// test if there isn't exactly one.
func (h *harness) runOne(content string) string {
	h.t.Helper()
	replies := h.run(content)
	if len(replies) != 1 {
		h.t.Fatalf("%q: got %d replies %q, want 1", content, len(replies), replies)
	}
	return replies[0]
}

// wantReply fails the test unless reply contains every one of want.
func wantReply(t *testing.T, reply string, want ...string) {
	t.Helper()
	for _, w := range want {
		if !strings.Contains(reply, w) {
			t.Errorf("reply %q doesn't contain %q", reply, w)
		}
	}
}
