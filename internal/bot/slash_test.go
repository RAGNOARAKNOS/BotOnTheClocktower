package bot

import (
	"maps"
	"slices"
	"testing"

	"github.com/bwmarrin/discordgo"
)

// TestSlashParity checks every `!botc` command and subcommand has a `/botc` one, and the other way round.
func TestSlashParity(t *testing.T) {
	var textNames []string
	for name, cmd := range commands {
		if !cmd.alias {
			textNames = append(textNames, name)
		}
	}
	slashNames := make(map[string]*discordgo.ApplicationCommandOption)
	for _, o := range slashCommands()[0].Options {
		slashNames[o.Name] = o
	}
	if got, want := slices.Sorted(maps.Keys(slashNames)), slices.Sorted(slices.Values(textNames)); !slices.Equal(got, want) {
		t.Errorf("/botc subcommands = %v, want the commands table's %v", got, want)
	}

	groups := map[string]map[string]func(*Bot, *request){"village": villageCommands, "character": characterCommands}
	for name, subcommands := range groups {
		group := slashNames[name]
		if group == nil || group.Type != discordgo.ApplicationCommandOptionSubCommandGroup {
			t.Errorf("/botc %s is not a subcommand group", name)
			continue
		}
		var got []string
		for _, o := range group.Options {
			got = append(got, o.Name)
		}
		if want := slices.Sorted(maps.Keys(subcommands)); !slices.Equal(slices.Sorted(slices.Values(got)), want) {
			t.Errorf("/botc %s subcommands = %v, want %v", name, got, want)
		}
	}
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
