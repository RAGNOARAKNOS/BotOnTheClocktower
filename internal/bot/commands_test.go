package bot

import (
	"os"
	"strings"
	"testing"
)

// mustFind returns the command at path (a name, then any subcommand's name).
func mustFind(t *testing.T, path ...string) command {
	t.Helper()
	cmd, used, known := resolve(path)
	if !known || used != len(path) {
		t.Fatalf("no command %q", path)
	}
	return cmd
}

func TestAllowed(t *testing.T) {
	game := newGame("guild", "admin", "town", "st")

	const (
		inAdmin   = "admin"
		elsewhere = "general"
	)
	defaultCmd := command{}
	register := mustFind(t, "register")
	ping := mustFind(t, "ping")

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

func TestResolve(t *testing.T) {
	tests := []struct {
		names     string
		wantName  string
		wantUsed  int
		wantKnown bool
	}{
		{"ping", "ping", 1, true},
		{"START", "register", 1, true},
		{"end now", "unregister", 1, true},
		{"gather 5", "gather", 1, true},
		{"village add <@1>", "add", 2, true},
		{"Character KILL <@1>", "kill", 2, true},
		{"village", "village", 1, true},
		{"village dance", "village", 1, true},
		{"dance", "", 0, false},
		{"", "", 0, false},
	}
	for _, tt := range tests {
		cmd, used, known := resolve(strings.Fields(tt.names))
		if cmd.name != tt.wantName || used != tt.wantUsed || known != tt.wantKnown {
			t.Errorf("resolve(%q) = %q, %d, %v; want %q, %d, %v", tt.names, cmd.name, used, known, tt.wantName, tt.wantUsed, tt.wantKnown)
		}
	}
}

// eachCommand calls f for every command in the table, with its path, groups included.
func eachCommand(f func(path string, c command)) {
	var walk func(prefix string, cmds []command)
	walk = func(prefix string, cmds []command) {
		for _, c := range cmds {
			path := strings.TrimSpace(prefix + " " + c.name)
			f(path, c)
			walk(path, c.subcommands)
		}
	}
	walk("", commands)
}

// TestCommandTable checks each entry is complete: a leaf runs something and says
// how to write it, a group only holds subcommands, and no name is used twice.
func TestCommandTable(t *testing.T) {
	eachCommand(func(path string, c command) {
		if len(c.subcommands) > 0 {
			if c.run != nil || c.usage != "" || len(c.options) > 0 || c.form != nil {
				t.Errorf("group %q has its own run, usage, options or form; its subcommands should", path)
			}
		} else {
			if c.run == nil {
				t.Errorf("command %q has no run function", path)
			}
			if !strings.HasPrefix(c.usage, "`!botc "+path) {
				t.Errorf("command %q's usage %q should start with `!botc %s", path, c.usage, path)
			}
		}
	})

	var check func(level string, cmds []command)
	check = func(level string, cmds []command) {
		seen := make(map[string]bool)
		for _, c := range cmds {
			for _, name := range append([]string{c.name}, c.aliases...) {
				if seen[name] {
					t.Errorf("%s: %q is used twice", level, name)
				}
				seen[name] = true
			}
			check(c.name, c.subcommands)
		}
	}
	check("top level", commands)
}

// TestREADMEListsEveryCommand checks the README's Commands table covers every command and alias.
func TestREADMEListsEveryCommand(t *testing.T) {
	readme, err := os.ReadFile("../../README.md")
	if err != nil {
		t.Fatal(err)
	}
	_, section, _ := strings.Cut(string(readme), "\n## Commands\n")
	section, _, _ = strings.Cut(section, "\n## ")
	eachCommand(func(path string, c command) {
		if len(c.subcommands) > 0 {
			return
		}
		paths := []string{path}
		for _, alias := range c.aliases {
			paths = append(paths, strings.TrimSuffix(path, c.name)+alias)
		}
		for _, p := range paths {
			// Named in full, e.g. "`!botc start`" or "`!botc character revive @player...`".
			if !strings.Contains(section, "`!botc "+p+"`") && !strings.Contains(section, "`!botc "+p+" ") {
				t.Errorf("README.md's Commands section doesn't list `!botc %s`", p)
			}
		}
	})
}
