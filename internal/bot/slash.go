package bot

import (
	"errors"
	"slices"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// The /botc command is built from the commands table: each command is a subcommand,
// and each group (village, character) is a subcommand group.

const (
	slashCommandName = "botc"
	// modalPrefix starts the CustomID of every form the bot opens.
	modalPrefix = "botc|"
)

// slashOption builds an option; the helpers below cover the kinds /botc uses.
func slashOption(kind discordgo.ApplicationCommandOptionType, name, description string, required bool) *discordgo.ApplicationCommandOption {
	return &discordgo.ApplicationCommandOption{Type: kind, Name: name, Description: description, Required: required}
}

func playersOption(required bool) *discordgo.ApplicationCommandOption {
	return slashOption(discordgo.ApplicationCommandOptionString, "players", "The players, as @mentions", required)
}

func playerOption() *discordgo.ApplicationCommandOption {
	return slashOption(discordgo.ApplicationCommandOptionUser, "player", "The player", true)
}

func teamOption(required bool) *discordgo.ApplicationCommandOption {
	o := slashOption(discordgo.ApplicationCommandOptionString, "team", "Good or Evil", required)
	o.Choices = []*discordgo.ApplicationCommandOptionChoice{{Name: "Good", Value: "good"}, {Name: "Evil", Value: "evil"}}
	return o
}

func minutesOption() *discordgo.ApplicationCommandOption {
	minMinutes := float64(1)
	o := slashOption(discordgo.ApplicationCommandOptionInteger, "minutes", "Minutes until the move (default 60 seconds)", false)
	o.MinValue, o.MaxValue = &minMinutes, gatherMaxMinutes
	return o
}

func cancelOption() *discordgo.ApplicationCommandOption {
	return slashOption(discordgo.ApplicationCommandOptionBoolean, "cancel", "Cancel the running countdown", false)
}

// characterOption is the character's name, which only fills in the form's box.
func characterOption() *discordgo.ApplicationCommandOption {
	o := slashOption(discordgo.ApplicationCommandOptionString, "character", "The character's name (you can change it in the form)", false)
	o.MaxLength = maxCharacterName
	return o
}

func subcommand(name, description string, options ...*discordgo.ApplicationCommandOption) *discordgo.ApplicationCommandOption {
	return &discordgo.ApplicationCommandOption{Type: discordgo.ApplicationCommandOptionSubCommand, Name: name, Description: description, Options: options}
}

func subcommandGroup(name, description string, subcommands ...*discordgo.ApplicationCommandOption) *discordgo.ApplicationCommandOption {
	return &discordgo.ApplicationCommandOption{Type: discordgo.ApplicationCommandOptionSubCommandGroup, Name: name, Description: description, Options: subcommands}
}

// slashCommands returns the /botc command definition, built from the commands table.
func slashCommands() []*discordgo.ApplicationCommand {
	return []*discordgo.ApplicationCommand{{
		Name:        slashCommandName,
		Description: "Blood on the Clocktower: Storyteller commands",
		Contexts:    &[]discordgo.InteractionContextType{discordgo.InteractionContextGuild},
		Options:     slashSubcommands(commands),
	}}
}

// slashSubcommands turns commands into /botc subcommands, and groups into subcommand groups.
func slashSubcommands(cmds []command) []*discordgo.ApplicationCommandOption {
	options := make([]*discordgo.ApplicationCommandOption, 0, len(cmds))
	for _, c := range cmds {
		if len(c.subcommands) > 0 {
			options = append(options, subcommandGroup(c.name, c.description, slashSubcommands(c.subcommands)...))
		} else {
			options = append(options, subcommand(c.name, c.description, c.options...))
		}
	}
	return options
}

// slashOptions returns a /botc command's path ("/botc", subcommand group if any,
// subcommand) and the options given to the subcommand, by name.
func slashOptions(data discordgo.ApplicationCommandInteractionData) (path []string, options map[string]*discordgo.ApplicationCommandInteractionDataOption) {
	path = []string{"/" + slashCommandName}
	options = make(map[string]*discordgo.ApplicationCommandInteractionDataOption)
	if len(data.Options) == 0 {
		return path, options
	}
	opt := data.Options[0]
	path = append(path, opt.Name)
	if opt.Type == discordgo.ApplicationCommandOptionSubCommandGroup && len(opt.Options) > 0 {
		opt = opt.Options[0]
		path = append(path, opt.Name)
	}
	for _, o := range opt.Options {
		options[o.Name] = o
	}
	return path, options
}

// stringOption returns the named string option's value, or "" if it wasn't given.
func stringOption(options map[string]*discordgo.ApplicationCommandInteractionDataOption, name string) string {
	if o := options[name]; o != nil {
		return o.StringValue()
	}
	return ""
}

// slashWords turns a /botc command's options into the words `!botc` would have,
// plus the IDs of the users it names. Options are read by name, in a fixed order.
func slashWords(data discordgo.ApplicationCommandInteractionData) (words, mentionIDs []string) {
	words, options := slashOptions(data)
	if o := options["player"]; o != nil {
		mentionIDs = append(mentionIDs, o.Value.(string))
	}
	if o := options["players"]; o != nil {
		mentionIDs = append(mentionIDs, mentionIDsIn(o.StringValue())...)
	}
	if o := options["team"]; o != nil {
		words = append(words, o.StringValue())
	}
	if o := options["minutes"]; o != nil {
		words = append(words, strconv.FormatInt(o.IntValue(), 10))
	}
	if o := options["cancel"]; o != nil && o.BoolValue() {
		words = append(words, "cancel")
	}
	return words, mentionIDs
}

// mentionIDsIn returns the IDs of the users mentioned in text, each once, in order.
func mentionIDsIn(text string) []string {
	var ids []string
	for _, m := range mentionPattern.FindAllStringSubmatch(text, -1) {
		if !slices.Contains(ids, m[1]) {
			ids = append(ids, m[1])
		}
	}
	return ids
}

// Forms. The CustomID carries what the form is for: "botc|whisper|<user ID>" or
// "botc|assign|<user ID>|<team or empty>".

// formOpener builds the form a /botc command answers with, for the player it names,
// from the command's options.
type formOpener func(userID, playerName string, options map[string]*discordgo.ApplicationCommandInteractionDataOption) *discordgo.InteractionResponseData

// opensForm reports whether the /botc command with this path (see slashOptions)
// answers with a form rather than running straight away.
func opensForm(path []string) bool {
	cmd, _, _ := resolve(path[1:])
	return cmd.form != nil
}

// Text box IDs in the forms.
const (
	fieldText      = "text"
	fieldCharacter = "character"
	fieldGuidance  = "guidance"
)

// whisperModal is the form `/botc whisper` opens.
func whisperModal(userID, playerName string, _ map[string]*discordgo.ApplicationCommandInteractionDataOption) *discordgo.InteractionResponseData {
	return &discordgo.InteractionResponseData{
		CustomID: modalPrefix + "whisper|" + userID,
		Title:    truncate("Whisper to "+playerName, 45),
		Components: []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.TextInput{CustomID: fieldText, Label: "Message", Style: discordgo.TextInputParagraph, Required: true, MaxLength: maxTextInput},
		}}},
	}
}

// assignModal is the form `/botc character assign` opens, with the character's
// name filled in. The team option is carried in the CustomID.
func assignModal(userID, playerName string, options map[string]*discordgo.ApplicationCommandInteractionDataOption) *discordgo.InteractionResponseData {
	name := stringOption(options, "character")
	return &discordgo.InteractionResponseData{
		CustomID: modalPrefix + "assign|" + userID + "|" + stringOption(options, "team"),
		Title:    truncate("Character for "+playerName, 45),
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{Components: []discordgo.MessageComponent{
				discordgo.TextInput{CustomID: fieldCharacter, Label: "Character", Style: discordgo.TextInputShort, Value: name, Required: true, MaxLength: maxCharacterName},
			}},
			discordgo.ActionsRow{Components: []discordgo.MessageComponent{
				discordgo.TextInput{CustomID: fieldGuidance, Label: "Guidance (optional)", Style: discordgo.TextInputParagraph, MaxLength: maxTextInput},
			}},
		},
	}
}

// modalCommand turns a submitted form into the words and raw text `!botc` would
// have, so the same handler and parser (parse.go) run.
func modalCommand(customID string, fields map[string]string) (words []string, content, userID string, err error) {
	parts := strings.Split(strings.TrimPrefix(customID, modalPrefix), "|")
	switch {
	case len(parts) == 2 && parts[0] == "whisper":
		userID = parts[1]
		return []string{"/" + slashCommandName, "whisper"}, "<@" + userID + "> " + fields[fieldText], userID, nil
	case len(parts) == 3 && parts[0] == "assign":
		userID = parts[1]
		first := strings.TrimSpace(parts[2] + " " + fields[fieldCharacter])
		return []string{"/" + slashCommandName, "character", "assign"}, "<@" + userID + "> " + first + "\n" + fields[fieldGuidance], userID, nil
	}
	return nil, "", "", errors.New("unknown form")
}

// modalFields returns the text in each box of a submitted form, by box ID.
func modalFields(data discordgo.ModalSubmitInteractionData) map[string]string {
	fields := make(map[string]string)
	for _, c := range data.Components {
		row, ok := c.(*discordgo.ActionsRow)
		if !ok {
			continue
		}
		for _, rc := range row.Components {
			if input, ok := rc.(*discordgo.TextInput); ok {
				fields[input.CustomID] = input.Value
			}
		}
	}
	return fields
}
