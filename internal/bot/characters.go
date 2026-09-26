package bot

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/bwmarrin/discordgo"
)

// Character is the game character the Storyteller has given a player, sent to them by DM.
type Character struct {
	Name     string
	Guidance string // optional; kept exactly as typed
	Sent     bool
}

const (
	// maxEmbedDescription is Discord's limit on an embed's description.
	maxEmbedDescription = 4096
	// maxCharacterName keeps "Your character: <name>" within Discord's 256-character embed title limit.
	maxCharacterName = 200

	characterUsage = "Usage: `!botc character assign @player <Character>` (guidance on the following lines), `!botc character clear @player...`, `!botc character list`, `!botc character send [@player...]`"
	whisperUsage   = "Usage: `!botc whisper @player <text>` (the text can span several lines)"
)

// mentionPattern matches a user mention as it appears in raw message content.
var mentionPattern = regexp.MustCompile(`<@!?(\d+)>`)

// splitAtMention finds the first user mention on the first line of a raw command
// message. It returns the mentioned user's ID, the rest of that line, and the
// lines after it, untouched.
func splitAtMention(content string) (userID, restOfLine, laterLines string, err error) {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	firstLine, laterLines, _ := strings.Cut(content, "\n")

	match := mentionPattern.FindStringSubmatchIndex(firstLine)
	if match == nil {
		return "", "", "", errors.New("mention the player on the first line")
	}

	restOfLine = firstLine[match[1]:]
	if mentionPattern.MatchString(restOfLine) {
		return "", "", "", errors.New("mention exactly one player")
	}

	return firstLine[match[2]:match[3]], strings.TrimSpace(restOfLine), laterLines, nil
}

// parseAssignment reads `!botc character assign @player <Character>` plus optional
// guidance on the following lines.
func parseAssignment(content string) (userID, name, guidance string, err error) {
	userID, name, guidance, err = splitAtMention(content)
	if err != nil {
		return "", "", "", err
	}

	guidance = strings.TrimSpace(guidance)
	switch {
	case name == "":
		return "", "", "", errors.New("put the character's name after the mention, on the same line")
	case utf8.RuneCountInString(name) > maxCharacterName:
		return "", "", "", fmt.Errorf("the character's name is longer than %d characters", maxCharacterName)
	case utf8.RuneCountInString(guidance) > maxEmbedDescription:
		return "", "", "", fmt.Errorf("the guidance is longer than Discord's limit of %d characters", maxEmbedDescription)
	}

	return userID, name, guidance, nil
}

// parseWhisper reads `!botc whisper @player <text>`, where the text starts after
// the mention and can carry on over the following lines.
func parseWhisper(content string) (userID, text string, err error) {
	userID, restOfLine, laterLines, err := splitAtMention(content)
	if err != nil {
		return "", "", err
	}

	text = strings.TrimSpace(restOfLine + "\n" + laterLines)
	switch {
	case text == "":
		return "", "", errors.New("there's nothing to whisper")
	case utf8.RuneCountInString(text) > maxEmbedDescription:
		return "", "", fmt.Errorf("the message is longer than Discord's limit of %d characters", maxEmbedDescription)
	}

	return userID, text, nil
}

// character dispatches the character subcommands. All of them are Storyteller-only, from the admin channel.
func (b *Bot) character(message *discordgo.MessageCreate, rawText []string) {
	if !b.requireStorytellerInAdmin(message) {
		return
	}

	if len(rawText) < 3 {
		b.discord.ChannelMessageSendReply(message.ChannelID, characterUsage, message.Reference())
		return
	}

	switch strings.ToLower(rawText[2]) {
	case "assign":
		b.characterAssign(message)
	case "clear":
		b.characterClear(message)
	case "list":
		b.characterList(message)
	case "send":
		b.characterSend(message)
	default:
		b.discord.ChannelMessageSendReply(message.ChannelID, characterUsage, message.Reference())
	}
}

// characterAssign stores (or replaces) a player's character. It isn't sent until `character send`.
func (b *Bot) characterAssign(message *discordgo.MessageCreate) {
	userID, name, guidance, err := parseAssignment(message.Content)
	if err != nil {
		b.discord.ChannelMessageSendReply(message.ChannelID, fmt.Sprintf("Could not read that (%v). %s", err, characterUsage), message.Reference())
		return
	}

	playerName, ok := b.settings.Players[userID]
	if !ok {
		b.discord.ChannelMessageSendReply(message.ChannelID, fmt.Sprintf("<@%s> isn't in the village. Add them with `!botc village add` first.", userID), message.Reference())
		return
	}

	if b.settings.Characters == nil {
		b.settings.Characters = make(map[string]*Character)
	}
	previous := b.settings.Characters[userID]
	b.settings.Characters[userID] = &Character{Name: name, Guidance: guidance}

	reply := fmt.Sprintf("%s will be the %s", playerName, name)
	if guidance != "" {
		reply += fmt.Sprintf(" (with %d line(s) of guidance)", strings.Count(guidance, "\n")+1)
	}
	reply += "."
	if previous != nil {
		reply += fmt.Sprintf(" This replaces %s.", previous.Name)
	}
	reply += " Not sent yet; use `!botc character send`."
	b.discord.ChannelMessageSendReply(message.ChannelID, reply, message.Reference())
}

// characterClear removes the stored characters of the mentioned players.
func (b *Bot) characterClear(message *discordgo.MessageCreate) {
	if len(message.Mentions) == 0 {
		b.discord.ChannelMessageSendReply(message.ChannelID, "Mention the players to clear. "+characterUsage, message.Reference())
		return
	}

	var cleared, skipped []string
	for _, user := range message.Mentions {
		if _, ok := b.settings.Characters[user.ID]; !ok {
			skipped = append(skipped, user.Username)
			continue
		}
		delete(b.settings.Characters, user.ID)
		cleared = append(cleared, b.playerName(user.ID, user.Username))
	}

	reply := fmt.Sprintf("Cleared %d character(s)", len(cleared))
	if len(cleared) > 0 {
		reply += ": " + strings.Join(cleared, ", ")
	}
	if len(skipped) > 0 {
		reply += "\nNo character to clear: " + strings.Join(skipped, ", ")
	}
	b.discord.ChannelMessageSendReply(message.ChannelID, reply, message.Reference())
}

// characterList replies with every village player's character and whether it has been sent.
func (b *Bot) characterList(message *discordgo.MessageCreate) {
	if len(b.settings.Players) == 0 {
		b.discord.ChannelMessageSendReply(message.ChannelID, "The village is empty. Use `!botc village create` first.", message.Reference())
		return
	}

	var sb strings.Builder
	sb.WriteString("Characters:")
	for i, id := range b.sortedPlayerIDs() {
		fmt.Fprintf(&sb, "\n%d. %s: ", i+1, b.settings.Players[id])

		c, ok := b.settings.Characters[id]
		if !ok {
			sb.WriteString("none")
			continue
		}
		sb.WriteString(c.Name)
		if c.Guidance != "" {
			sb.WriteString(" (+ guidance)")
		}
		if c.Sent {
			sb.WriteString(", sent")
		} else {
			sb.WriteString(", not sent")
		}
	}
	b.discord.ChannelMessageSendReply(message.ChannelID, sb.String(), message.Reference())
}

// characterSend DMs every unsent character, or, when players are mentioned,
// resends to just those players.
func (b *Bot) characterSend(message *discordgo.MessageCreate) {
	var targets, skipped []string
	if len(message.Mentions) > 0 {
		for _, user := range message.Mentions {
			if _, ok := b.settings.Characters[user.ID]; ok {
				targets = append(targets, user.ID)
			} else {
				skipped = append(skipped, user.Username)
			}
		}
	} else {
		for _, id := range b.sortedPlayerIDs() {
			if c, ok := b.settings.Characters[id]; ok && !c.Sent {
				targets = append(targets, id)
			}
		}
	}

	var sent, failed []string
	for _, id := range targets {
		c := b.settings.Characters[id]
		embed := b.dmEmbed("Your character: "+c.Name, c.Guidance)
		name := b.playerName(id, id)
		if err := b.sendDM(id, embed); err != nil {
			fmt.Printf("Could not DM %s their character: %v \n", id, err)
			failed = append(failed, fmt.Sprintf("%s (%s)", name, dmErrorReason(err)))
			continue
		}
		c.Sent = true
		sent = append(sent, name)
	}

	var missing []string
	for _, id := range b.sortedPlayerIDs() {
		if _, ok := b.settings.Characters[id]; !ok {
			missing = append(missing, b.settings.Players[id])
		}
	}

	var reply string
	if len(targets) == 0 && len(skipped) == 0 {
		reply = "Nothing to send: every assigned character has already been sent. Use `!botc character send @player` to resend."
	} else {
		reply = fmt.Sprintf("Sent %d character(s)", len(sent))
		if len(sent) > 0 {
			reply += ": " + strings.Join(sent, ", ")
		}
	}
	if len(failed) > 0 {
		reply += "\nFailed: " + strings.Join(failed, ", ") + ". Fix the problem and run `!botc character send` again."
	}
	if len(skipped) > 0 {
		reply += "\nNo character assigned: " + strings.Join(skipped, ", ")
	}
	if len(missing) > 0 {
		reply += "\nVillage players with no character yet: " + strings.Join(missing, ", ")
	}
	b.discord.ChannelMessageSendReply(message.ChannelID, reply, message.Reference())
}

// whisper DMs a village player a secret message from the Storyteller straight away.
func (b *Bot) whisper(message *discordgo.MessageCreate) {
	if !b.requireStorytellerInAdmin(message) {
		return
	}

	userID, text, err := parseWhisper(message.Content)
	if err != nil {
		b.discord.ChannelMessageSendReply(message.ChannelID, fmt.Sprintf("Could not read that (%v). %s", err, whisperUsage), message.Reference())
		return
	}

	playerName, ok := b.settings.Players[userID]
	if !ok {
		b.discord.ChannelMessageSendReply(message.ChannelID, fmt.Sprintf("<@%s> isn't in the village. This command will not execute", userID), message.Reference())
		return
	}

	if err := b.sendDM(userID, b.dmEmbed("A message from the Storyteller", text)); err != nil {
		fmt.Printf("Could not whisper to %s: %v \n", userID, err)
		b.discord.ChannelMessageSendReply(message.ChannelID, fmt.Sprintf("Could not whisper to %s (%s).", playerName, dmErrorReason(err)), message.Reference())
		return
	}

	b.discord.ChannelMessageSendReply(message.ChannelID, fmt.Sprintf("Whispered to %s.", playerName), message.Reference())
}

// sendDM sends an embed to the user in a direct message.
func (b *Bot) sendDM(userID string, embed *discordgo.MessageEmbed) error {
	channel, err := b.discord.UserChannelCreate(userID)
	if err != nil {
		return err
	}

	_, err = b.discord.ChannelMessageSendEmbed(channel.ID, embed)
	return err
}

// dmEmbed builds a DM embed, with the game's server named in the footer.
func (b *Bot) dmEmbed(title, description string) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{Title: title, Description: description}
	if guild, err := b.discord.State.Guild(b.settings.GuildId); err == nil {
		embed.Footer = &discordgo.MessageEmbedFooter{Text: "Blood on the Clocktower on " + guild.Name}
	}
	return embed
}

// dmErrorReason turns a failed DM into a short reason for the Storyteller.
func dmErrorReason(err error) string {
	var restErr *discordgo.RESTError
	if errors.As(err, &restErr) && restErr.Message != nil && restErr.Message.Code == discordgo.ErrCodeCannotSendMessagesToThisUser {
		return "they don't accept DMs from this server"
	}
	return err.Error()
}

// playerName returns the village name for the user, or fallback if they aren't in the village.
func (b *Bot) playerName(userID, fallback string) string {
	if name, ok := b.settings.Players[userID]; ok {
		return name
	}
	return fallback
}

// sortedPlayerIDs returns the village players' IDs, ordered by display name.
func (b *Bot) sortedPlayerIDs() []string {
	ids := make([]string, 0, len(b.settings.Players))
	for id := range b.settings.Players {
		ids = append(ids, id)
	}
	slices.SortFunc(ids, func(a, c string) int {
		return strings.Compare(b.settings.Players[a], b.settings.Players[c])
	})
	return ids
}
