package bot

import "testing"

func TestAllowed(t *testing.T) {
	game := newGame("guild", "admin", "town", "st")

	const (
		inAdmin   = "admin"
		elsewhere = "general"
	)
	defaultCmd := command{}
	register := commands["register"]
	ping := commands["ping"]

	tests := []struct {
		name        string
		game        *Game
		cmd         command
		author      string
		guild       string
		channel     string
		wantOK      bool
		wantRefusal bool
	}{
		{"no game: register from anyone, anywhere", nil, register, "anyone", "guild", elsewhere, true, false},
		{"no game: ping from anyone, anywhere", nil, ping, "anyone", "", "dm", true, false},
		{"no game: other command ignored", nil, defaultCmd, "st", "guild", inAdmin, false, false},

		{"game: Storyteller in admin", game, defaultCmd, "st", "guild", inAdmin, true, false},
		{"game: Storyteller elsewhere", game, defaultCmd, "st", "guild", elsewhere, false, true},
		{"game: Storyteller in another server", game, defaultCmd, "st", "other", inAdmin, false, true},
		{"game: player in admin", game, defaultCmd, "player", "guild", inAdmin, false, true},
		{"game: register refused for others", game, register, "player", "guild", elsewhere, false, true},
		{"game: register by Storyteller in admin reaches register", game, register, "st", "guild", inAdmin, true, false},

		{"game: Storyteller pings from anywhere", game, ping, "st", "", "dm", true, false},
		{"game: player ping refused", game, ping, "player", "guild", inAdmin, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, refusal := allowed(tt.game, tt.cmd, tt.author, tt.guild, tt.channel)
			if ok != tt.wantOK || (refusal != "") != tt.wantRefusal {
				t.Errorf("allowed() = %v, %q; want ok %v, refusal %v", ok, refusal, tt.wantOK, tt.wantRefusal)
			}
		})
	}
}

func TestCommandTable(t *testing.T) {
	for name, cmd := range commands {
		if cmd.run == nil {
			t.Errorf("command %q has no run function", name)
		}
	}
	for _, alias := range [][2]string{{"register", "start"}, {"unregister", "end"}} {
		a, b := commands[alias[0]], commands[alias[1]]
		if a.beforeGame != b.beforeGame || a.anyChannel != b.anyChannel {
			t.Errorf("alias %q has different access from %q", alias[1], alias[0])
		}
	}
}
