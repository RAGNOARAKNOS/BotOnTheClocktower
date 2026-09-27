package bot

import (
	"errors"
	"slices"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// The /botc command mirrors the commands table: each command is a subcommand, and
// village and character are subcommand groups mirroring villageCommands and
// characterCommands. TestSlashParity keeps them in step.

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

func subcommand(name, description string, options ...*discordgo.ApplicationCommandOption) *discordgo.ApplicationCommandOption {
	return &discordgo.ApplicationCommandOption{Type: discordgo.ApplicationCommandOptionSubCommand, Name: name, Description: description, Options: options}
}

func subcommandGroup(name, description string, subcommands ...*discordgo.ApplicationCommandOption) *discordgo.ApplicationCommandOption {
	return &discordgo.ApplicationCommandOption{Type: discordgo.ApplicationCommandOptionSubCommandGroup, Name: name, Description: description, Options: subcommands}
}

// slashCommands returns the /botc command definition.
func slashCommands() []*discordgo.ApplicationCommand {
	minMinutes := float64(1)
	minutes := slashOption(discordgo.ApplicationCommandOptionInteger, "minutes", "Minutes until the move (default 60 seconds)", false)
	minutes.MinValue, minutes.MaxValue = &minMinutes, gatherMaxMinutes

	character := slashOption(discordgo.ApplicationCommandOptionString, "character", "The character's name (you can change it in the form)", false)
	character.MaxLength = maxCharacterName

	return []*discordgo.ApplicationCommand{{
		Name:        slashCommandName,
		Description: "Blood on the Clocktower: Storyteller commands",
		Contexts:    &[]discordgo.InteractionContextType{discordgo.InteractionContextGuild},
		Options: []*discordgo.ApplicationCommandOption{
			subcommand("ping", "Check the bot is online"),
			subcommand("register", "Start a game: you become the Storyteller and this becomes the admin channel"),
			subcommand("unregister", "End the game and remove the game roles"),
			subcommand("sitrep", "Report the game's server, channels and Storyteller"),
			subcommand("map", "Find the village's voice channels"),
			subcommand("grimoire", "Show every player's character"),
			subcommand("whisper", "DM a player a secret message (opens a form)", playerOption()),
			subcommand("gather", "Count down, then move the players to Town Square",
				minutes,
				slashOption(discordgo.ApplicationCommandOptionBoolean, "cancel", "Cancel the running countdown", false)),
			subcommandGroup("village", "Manage the village's players",
				subcommand("create", "Make everyone in Town Square the village"),
				subcommand("add", "Add players to the village", playersOption(true)),
				subcommand("remove", "Remove players from the village", playersOption(true)),
				subcommand("list", "List the village's players")),
			subcommandGroup("character", "Manage the players' characters",
				subcommand("assign", "Give a player a character (opens a form for guidance)", playerOption(), teamOption(false), character),
				subcommand("team", "Move characters to another team", playersOption(true), teamOption(true)),
				subcommand("kill", "Mark players dead", playersOption(true)),
				subcommand("revive", "Mark players alive", playersOption(true)),
				subcommand("ghostvote", "Switch dead players' ghost votes", playersOption(true)),
				subcommand("announce", "Announce deaths and revivals in Town Square"),
				subcommand("clear", "Remove players' characters", playersOption(true)),
				subcommand("list", "Show every player's character"),
				subcommand("send", "DM unsent characters, or resend to the players given", playersOption(false))),
		},
	}}
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

// Text box IDs in the forms.
const (
	fieldText      = "text"
	fieldCharacter = "character"
	fieldGuidance  = "guidance"
)

// whisperModal is the form `/botc whisper` opens.
func whisperModal(userID, playerName string) *discordgo.InteractionResponseData {
	return &discordgo.InteractionResponseData{
		CustomID: modalPrefix + "whisper|" + userID,
		Title:    truncate("Whisper to "+playerName, 45),
		Components: []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.TextInput{CustomID: fieldText, Label: "Message", Style: discordgo.TextInputParagraph, Required: true, MaxLength: maxTextInput},
		}}},
	}
}

// assignModal is the form `/botc character assign` opens, with the name filled in.
func assignModal(userID, playerName, team, name string) *discordgo.InteractionResponseData {
	return &discordgo.InteractionResponseData{
		CustomID: modalPrefix + "assign|" + userID + "|" + team,
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
