package bot

import (
	"strings"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
)

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

func TestUsageReplies(t *testing.T) {
	h := newHarness(t)
	h.withGame(nil)
	for _, content := range []string{"!botc village", "!botc village dance", "!botc character", "!botc CHARACTER Dance"} {
		wantReply(t, h.runOne(content), "Usage: ")
	}
	if got := h.runOne("!botc village LIST"); !strings.Contains(got, "The village is empty") {
		t.Errorf("subcommand names should ignore capitals; got %q", got)
	}
}

// slashInteraction builds a /botc command from author in the admin channel.
func slashInteraction(author string, options ...*discordgo.ApplicationCommandInteractionDataOption) *discordgo.InteractionCreate {
	return &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		ID: "interaction", Token: "token", AppID: "app", Type: discordgo.InteractionApplicationCommand,
		GuildID: testGuild, ChannelID: testAdmin,
		Member: &discordgo.Member{User: &discordgo.User{ID: author}},
		Data:   discordgo.ApplicationCommandInteractionData{Name: slashCommandName, Options: options},
	}}
}

// TestSlashAcknowledgedWhileLocked checks a slash command is acknowledged at once,
// even while another command holds Bot.mu, and a form isn't.
func TestSlashAcknowledgedWhileLocked(t *testing.T) {
	const deferred, modal = `"type":5`, `"type":9`
	sub := discordgo.ApplicationCommandOptionSubCommand
	tests := []struct {
		name      string
		options   []*discordgo.ApplicationCommandInteractionDataOption
		wantEarly bool   // acknowledged while Bot.mu is held
		wantFirst string // the first response's type
	}{
		{"command", []*discordgo.ApplicationCommandInteractionDataOption{option("ping", sub, nil)}, true, deferred},
		{"form", []*discordgo.ApplicationCommandInteractionDataOption{option("whisper", sub, nil,
			option("player", discordgo.ApplicationCommandOptionUser, "1"))}, false, modal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(t)
			h.withGame(map[string]string{"1": "Alice"})

			h.b.mu.Lock()
			done := make(chan struct{})
			go func() {
				defer close(done)
				h.b.interaction(h.b.discord, slashInteraction(testStoryteller, tt.options...))
			}()
			early := h.fake.waitFor(200*time.Millisecond, "/callback")
			h.b.mu.Unlock()
			<-done

			if early != tt.wantEarly {
				t.Errorf("answered while Bot.mu was held: %v, want %v", early, tt.wantEarly)
			}
			if !h.fake.sent("/callback", tt.wantFirst) {
				t.Errorf("no response of %s; requests: %q", tt.wantFirst, h.fake.requests)
			}
		})
	}
}
