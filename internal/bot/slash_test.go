package bot

import (
	"maps"
	"regexp"
	"slices"
	"testing"
	"unicode/utf8"

	"github.com/bwmarrin/discordgo"
)

// discordName is what Discord accepts as a command or option name.
var discordName = regexp.MustCompile(`^[-_a-z0-9]{1,32}$`)

// TestSlashDefinition checks the /botc definition keeps to Discord's rules. Discord
// refuses a bad definition when the bot registers it, which is only logged
// (registerSlashCommands), so /botc would silently stay out of date.
func TestSlashDefinition(t *testing.T) {
	defs := slashCommands()
	if len(defs) != 1 || defs[0].Name != slashCommandName {
		t.Fatalf("want one command, /%s", slashCommandName)
	}

	var check func(path string, options []*discordgo.ApplicationCommandOption)
	check = func(path string, options []*discordgo.ApplicationCommandOption) {
		if len(options) > 25 {
			t.Errorf("%s: %d options or subcommands, Discord allows 25", path, len(options))
		}
		optional := false
		for _, o := range options {
			p := path + " " + o.Name
			if !discordName.MatchString(o.Name) {
				t.Errorf("%s: name must be 1 to 32 lower-case letters, digits, - or _", p)
			}
			if n := utf8.RuneCountInString(o.Description); n < 1 || n > 100 {
				t.Errorf("%s: description is %d characters, Discord allows 1 to 100", p, n)
			}
			if o.Type == discordgo.ApplicationCommandOptionSubCommand || o.Type == discordgo.ApplicationCommandOptionSubCommandGroup {
				check(p, o.Options)
				continue
			}
			if o.Required && optional {
				t.Errorf("%s: a required option must come before the optional ones", p)
			}
			optional = optional || !o.Required
		}
	}
	check("/"+slashCommandName, defs[0].Options)
}

// TestSlashOptionsRead checks every /botc option is one slashWords (or a form) reads,
// so an option added to the commands table can't be silently ignored.
func TestSlashOptionsRead(t *testing.T) {
	read := []string{"player", "players", "team", "minutes", "cancel", "character"}
	eachCommand(func(path string, c command) {
		for _, o := range c.options {
			if !slices.Contains(read, o.Name) {
				t.Errorf("/botc %s option %q: add a case for it to slashWords, then add it to this test", path, o.Name)
			}
		}
	})
}

func option(name string, kind discordgo.ApplicationCommandOptionType, value any, options ...*discordgo.ApplicationCommandInteractionDataOption) *discordgo.ApplicationCommandInteractionDataOption {
	return &discordgo.ApplicationCommandInteractionDataOption{Name: name, Type: kind, Value: value, Options: options}
}

func TestSlashWords(t *testing.T) {
	sub, group := discordgo.ApplicationCommandOptionSubCommand, discordgo.ApplicationCommandOptionSubCommandGroup
	tests := []struct {
		name         string
		options      []*discordgo.ApplicationCommandInteractionDataOption
		wantWords    []string
		wantMentions []string
	}{
		{"plain", []*discordgo.ApplicationCommandInteractionDataOption{option("sitrep", sub, nil)}, []string{"/botc", "sitrep"}, nil},
		{"gather minutes", []*discordgo.ApplicationCommandInteractionDataOption{option("gather", sub, nil,
			option("minutes", discordgo.ApplicationCommandOptionInteger, float64(5)))}, []string{"/botc", "gather", "5"}, nil},
		{"gather cancel", []*discordgo.ApplicationCommandInteractionDataOption{option("gather", sub, nil,
			option("cancel", discordgo.ApplicationCommandOptionBoolean, true))}, []string{"/botc", "gather", "cancel"}, nil},
		{"gather cancel false", []*discordgo.ApplicationCommandInteractionDataOption{option("gather", sub, nil,
			option("cancel", discordgo.ApplicationCommandOptionBoolean, false))}, []string{"/botc", "gather"}, nil},
		{"group with players and team", []*discordgo.ApplicationCommandInteractionDataOption{option("character", group, nil,
			option("team", sub, nil,
				option("players", discordgo.ApplicationCommandOptionString, "<@1> and <@!2>, <@1>"),
				option("team", discordgo.ApplicationCommandOptionString, "evil")))},
			[]string{"/botc", "character", "team", "evil"}, []string{"1", "2"}},
		{"user option", []*discordgo.ApplicationCommandInteractionDataOption{option("whisper", sub, nil,
			option("player", discordgo.ApplicationCommandOptionUser, "3"))}, []string{"/botc", "whisper"}, []string{"3"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			words, mentions := slashWords(discordgo.ApplicationCommandInteractionData{Name: "botc", Options: tt.options})
			if !slices.Equal(words, tt.wantWords) || !slices.Equal(mentions, tt.wantMentions) {
				t.Errorf("slashWords() = %q, %q; want %q, %q", words, mentions, tt.wantWords, tt.wantMentions)
			}
		})
	}
}

// TestModalCommand checks a submitted form reaches the same parsers as `!botc`, with multi-line text intact.
func TestModalCommand(t *testing.T) {
	words, content, userID, err := modalCommand("botc|whisper|42", map[string]string{fieldText: "Line one\n**Line two**"})
	if err != nil || userID != "42" || !slices.Equal(words, []string{"/botc", "whisper"}) {
		t.Fatalf("whisper form: %q, %q, %v", words, userID, err)
	}
	if id, text, err := parseWhisper(content); err != nil || id != "42" || text != "Line one\n**Line two**" {
		t.Errorf("parseWhisper(form) = %q, %q, %v", id, text, err)
	}

	words, content, _, err = modalCommand("botc|assign|42|evil", map[string]string{fieldCharacter: "Imp", fieldGuidance: "You are the demon.\nKill at night."})
	if err != nil || !slices.Equal(words, []string{"/botc", "character", "assign"}) {
		t.Fatalf("assign form: %q, %v", words, err)
	}
	if id, team, name, guidance, err := parseAssignment(content); err != nil || id != "42" || team != TeamEvil || name != "Imp" || guidance != "You are the demon.\nKill at night." {
		t.Errorf("parseAssignment(form) = %q, %q, %q, %q, %v", id, team, name, guidance, err)
	}

	// No team given: the same as `!botc character assign @x Evil Twin`.
	_, content, _, _ = modalCommand("botc|assign|42|", map[string]string{fieldCharacter: "Evil Twin"})
	if _, team, name, guidance, err := parseAssignment(content); err != nil || team != TeamEvil || name != "Evil Twin" || guidance != "" {
		t.Errorf("parseAssignment(no team) = %q, %q, %q, %v", team, name, guidance, err)
	}

	if _, _, _, err := modalCommand("botc|other|1", nil); err == nil {
		t.Error("unknown form accepted")
	}
}

func TestModalFields(t *testing.T) {
	data := discordgo.ModalSubmitInteractionData{Components: []discordgo.MessageComponent{
		&discordgo.ActionsRow{Components: []discordgo.MessageComponent{&discordgo.TextInput{CustomID: fieldCharacter, Value: "Monk"}}},
		&discordgo.ActionsRow{Components: []discordgo.MessageComponent{&discordgo.TextInput{CustomID: fieldGuidance, Value: "Protect"}}},
	}}
	if got, want := modalFields(data), map[string]string{fieldCharacter: "Monk", fieldGuidance: "Protect"}; !maps.Equal(got, want) {
		t.Errorf("modalFields() = %v, want %v", got, want)
	}
}
