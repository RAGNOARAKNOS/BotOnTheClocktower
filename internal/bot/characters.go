package bot

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/bwmarrin/discordgo"
)

// Team is the side a character plays for.
type Team string

const (
	TeamGood Team = "Good"
	TeamEvil Team = "Evil"
)

// Character is the game character the Storyteller has given a player, sent to them by DM.
type Character struct {
	Name     string
	Guidance string // optional; kept exactly as typed
	Team     Team
	Sent     bool

	Alive bool
	// AnnouncedAlive is Alive as of the last `character announce`. While the two
	// differ, a death or revival is waiting to be announced.
	AnnouncedAlive bool
	// GhostVoteUsed records whether a dead player has spent their one remaining vote.
	GhostVoteUsed bool
}

const (
	// maxEmbedDescription is Discord's limit on an embed's description.
	maxEmbedDescription = 4096
	// maxCharacterName keeps "Your character: <name> (<team>)" within Discord's 256-character embed title limit.
	maxCharacterName = 200
	// maxMessageLength is Discord's limit on a message's content.
	maxMessageLength = 2000

	characterUsage = "Usage: `!botc character assign @player [good|evil] <Character>` (guidance on the following lines), " +
		"`!botc character team @player good|evil`, `!botc character kill @player...`, `!botc character revive @player...`, " +
		"`!botc character ghostvote @player...`, `!botc character announce`, `!botc character clear @player...`, " +
		"`!botc character list` (or `!botc grimoire`), `!botc character send [@player...]`"
	whisperUsage = "Usage: `!botc whisper @player <text>` (the text can span several lines)"
)

// mentionPattern matches a user mention as it appears in raw message content.
var mentionPattern = regexp.MustCompile(`<@!?(\d+)>`)

// teamWordNames are characters whose names start with a team word, so that
// `assign @player Evil Twin` isn't read as team Evil, character "Twin".
var teamWordNames = map[string]Team{
	"evil twin": TeamEvil,
}

// parseTeam reads "good" or "evil", ignoring capitals.
func parseTeam(word string) (Team, bool) {
	switch strings.ToLower(word) {
	case "good":
		return TeamGood, true
	case "evil":
		return TeamEvil, true
	}
	return "", false
}

// splitTeam takes the optional team word off the front of a character name.
// Without one, the character is Good.
func splitTeam(name string) (Team, string, error) {
	if team, ok := teamWordNames[strings.ToLower(name)]; ok {
		return team, name, nil
	}

	first, rest := name, ""
	if i := strings.IndexFunc(name, unicode.IsSpace); i >= 0 {
		first, rest = name[:i], strings.TrimSpace(name[i:])
	}

	team, isTeam := parseTeam(first)
	if !isTeam {
		return TeamGood, name, nil
	}
	if rest == "" {
		return "", "", errors.New("put the character's name after the team")
	}
	return team, rest, nil
}

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

// parseAssignment reads `!botc character assign @player [good|evil] <Character>`
// plus optional guidance on the following lines.
func parseAssignment(content string) (userID string, team Team, name, guidance string, err error) {
	userID, name, guidance, err = splitAtMention(content)
	if err != nil {
		return "", "", "", "", err
	}
	if name == "" {
		return "", "", "", "", errors.New("put the character's name after the mention, on the same line")
	}

	team, name, err = splitTeam(name)
	if err != nil {
		return "", "", "", "", err
	}

	guidance = strings.TrimSpace(guidance)
	switch {
	case utf8.RuneCountInString(name) > maxCharacterName:
		return "", "", "", "", fmt.Errorf("the character's name is longer than %d characters", maxCharacterName)
	case utf8.RuneCountInString(guidance) > maxEmbedDescription:
		return "", "", "", "", fmt.Errorf("the guidance is longer than Discord's limit of %d characters", maxEmbedDescription)
	}

	return userID, team, name, guidance, nil
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

// character dispatches the character subcommands.
func (b *Bot) character(message *discordgo.MessageCreate, rawText []string) {
	if len(rawText) < 3 {
		b.reply(message, characterUsage)
		return
	}

	switch strings.ToLower(rawText[2]) {
	case "assign":
		b.characterAssign(message)
	case "team":
		b.characterTeam(message, rawText)
	case "kill":
		b.characterSetAlive(message, false)
	case "revive":
		b.characterSetAlive(message, true)
	case "ghostvote":
		b.characterGhostVote(message)
	case "announce":
		b.characterAnnounce(message)
	case "clear":
		b.characterClear(message)
	case "list":
		b.characterList(message)
	case "send":
		b.characterSend(message)
	default:
		b.reply(message, characterUsage)
	}
}

// characterAssign stores (or replaces) a player's character. It isn't sent until `character send`.
func (b *Bot) characterAssign(message *discordgo.MessageCreate) {
	userID, team, name, guidance, err := parseAssignment(message.Content)
	if err != nil {
		b.reply(message, fmt.Sprintf("Could not read that (%v). %s", err, characterUsage))
		return
	}

	playerName, ok := b.settings.Players[userID]
	if !ok {
		b.reply(message, fmt.Sprintf("<@%s> isn't in the village. Add them with `!botc village add` first.", userID))
		return
	}

	if b.settings.Characters == nil {
		b.settings.Characters = make(map[string]*Character)
	}
	previous := b.settings.Characters[userID]
	c := &Character{Name: name, Guidance: guidance, Team: team, Alive: true, AnnouncedAlive: true}
	if previous != nil {
		// A new character doesn't bring anyone back to life.
		c.Alive, c.AnnouncedAlive, c.GhostVoteUsed = previous.Alive, previous.AnnouncedAlive, previous.GhostVoteUsed
	}
	b.settings.Characters[userID] = c

	reply := fmt.Sprintf("%s will be the %s (%s", playerName, name, team)
	if guidance != "" {
		reply += fmt.Sprintf(", with %d line(s) of guidance", strings.Count(guidance, "\n")+1)
	}
	reply += ")."
	if previous != nil {
		reply += fmt.Sprintf(" This replaces %s.", previous.Name)
	}
	reply += " Not sent yet; use `!botc character send`."
	b.reply(message, reply)
}

// characterTeam moves the mentioned players' characters to another team and marks them unsent.
func (b *Bot) characterTeam(message *discordgo.MessageCreate, rawText []string) {
	var team Team
	for _, word := range rawText[3:] {
		if t, ok := parseTeam(word); ok {
			team = t
		}
	}
	if team == "" || len(message.Mentions) == 0 {
		b.reply(message, "Mention the players and give the team, e.g. `!botc character team @player evil`.")
		return
	}

	ids, skipped := b.mentionedCharacters(message)
	var changed []string
	for _, id := range ids {
		c := b.settings.Characters[id]
		if c.Team == team {
			skipped = append(skipped, fmt.Sprintf("%s (already %s)", b.settings.Players[id], team))
			continue
		}
		c.Team = team
		c.Sent = false
		changed = append(changed, b.settings.Players[id])
	}

	reply := fmt.Sprintf("Now %s: %d player(s)", team, len(changed))
	if len(changed) > 0 {
		reply += ": " + strings.Join(changed, ", ") + ". Not told yet; use `!botc character send` to send them their updated character."
	}
	if len(skipped) > 0 {
		reply += "\nSkipped: " + strings.Join(skipped, ", ")
	}
	b.reply(message, reply)
}

// characterSetAlive kills or revives the mentioned players. Nothing is posted
// publicly until `character announce`.
func (b *Bot) characterSetAlive(message *discordgo.MessageCreate, alive bool) {
	if len(message.Mentions) == 0 {
		b.reply(message, "Mention the players. "+characterUsage)
		return
	}

	state := "dead"
	if alive {
		state = "alive"
	}

	ids, skipped := b.mentionedCharacters(message)
	var changed []string
	for _, id := range ids {
		c := b.settings.Characters[id]
		if c.Alive == alive {
			skipped = append(skipped, fmt.Sprintf("%s (already %s)", b.settings.Players[id], state))
			continue
		}
		c.Alive = alive
		if alive {
			c.GhostVoteUsed = false
		}
		changed = append(changed, b.settings.Players[id])
	}

	reply := fmt.Sprintf("Now %s: %d player(s)", state, len(changed))
	if len(changed) > 0 {
		reply += ": " + strings.Join(changed, ", ")
	}
	reply += "."
	if len(skipped) > 0 {
		reply += "\nSkipped: " + strings.Join(skipped, ", ")
	}
	died, revived := pendingLifeChanges(b.settings.Players, b.settings.Characters)
	if pending := len(died) + len(revived); pending > 0 {
		reply += fmt.Sprintf("\n%d change(s) waiting to be announced; use `!botc character announce`.", pending)
	} else {
		reply += "\nNothing is waiting to be announced."
	}
	b.reply(message, reply)
}

// characterGhostVote switches the mentioned dead players' ghost votes between used and available.
func (b *Bot) characterGhostVote(message *discordgo.MessageCreate) {
	if len(message.Mentions) == 0 {
		b.reply(message, "Mention the dead players whose ghost vote to change. "+characterUsage)
		return
	}

	ids, skipped := b.mentionedCharacters(message)
	var changed []string
	for _, id := range ids {
		c := b.settings.Characters[id]
		if c.Alive {
			skipped = append(skipped, b.settings.Players[id]+" (alive, so no ghost vote)")
			continue
		}
		c.GhostVoteUsed = !c.GhostVoteUsed
		changed = append(changed, fmt.Sprintf("%s: ghost vote %s", b.settings.Players[id], ghostVoteState(c)))
	}

	reply := "No ghost votes changed."
	if len(changed) > 0 {
		reply = strings.Join(changed, "\n")
	}
	if len(skipped) > 0 {
		reply += "\nSkipped: " + strings.Join(skipped, ", ")
	}
	b.reply(message, reply)
}

// characterAnnounce posts the deaths and revivals since the last announcement in
// the game channel, then marks them announced.
func (b *Bot) characterAnnounce(message *discordgo.MessageCreate) {
	died, revived := pendingLifeChanges(b.settings.Players, b.settings.Characters)
	if len(died)+len(revived) == 0 {
		b.reply(message, "Nothing to announce: no deaths or revivals since the last announcement.")
		return
	}

	var lines []string
	for _, name := range died {
		lines = append(lines, name+" has died.")
	}
	for _, name := range revived {
		lines = append(lines, name+" has returned to life.")
	}
	announcement := strings.Join(lines, "\n")

	if _, err := b.discord.ChannelMessageSend(b.settings.GameChannelId, announcement); err != nil {
		log.Printf("Could not post the announcement: %v", err)
		b.reply(message, fmt.Sprintf("Could not post in <#%s> (%v). Nothing was marked as announced.", b.settings.GameChannelId, err))
		return
	}

	for _, c := range b.settings.Characters {
		c.AnnouncedAlive = c.Alive
	}
	b.reply(message, fmt.Sprintf("Announced in <#%s>:\n%s", b.settings.GameChannelId, announcement))
}

// characterClear removes the stored characters of the mentioned players.
func (b *Bot) characterClear(message *discordgo.MessageCreate) {
	if len(message.Mentions) == 0 {
		b.reply(message, "Mention the players to clear. "+characterUsage)
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
	b.reply(message, reply)
}

// characterList replies with the grimoire: every village player's character,
// team, life state and whether it has been sent, with totals at the top.
func (b *Bot) characterList(message *discordgo.MessageCreate) {
	if len(b.settings.Players) == 0 {
		b.reply(message, "The village is empty. Use `!botc village create` first.")
		return
	}

	lines := []string{"Grimoire: " + grimoireSummary(b.settings.Players, b.settings.Characters)}
	for i, id := range b.sortedPlayerIDs() {
		line := fmt.Sprintf("%d. %s: ", i+1, b.settings.Players[id])
		if c, ok := b.settings.Characters[id]; ok {
			line += grimoireLine(c)
		} else {
			line += "none"
		}
		lines = append(lines, line)
	}

	for _, chunk := range chunkLines(lines, maxMessageLength) {
		b.reply(message, chunk)
	}
}

// grimoireLine describes one character, e.g. "Monk (Good) · Dead, ghost vote used · sent · death not announced".
func grimoireLine(c *Character) string {
	parts := []string{fmt.Sprintf("%s (%s)", c.Name, c.Team)}

	if c.Alive {
		parts = append(parts, "Alive")
	} else {
		parts = append(parts, "Dead, ghost vote "+ghostVoteState(c))
	}

	if c.Sent {
		parts = append(parts, "sent")
	} else {
		parts = append(parts, "not sent")
	}

	if c.Guidance != "" {
		parts = append(parts, "has guidance")
	}

	switch {
	case !c.Alive && c.AnnouncedAlive:
		parts = append(parts, "death not announced")
	case c.Alive && !c.AnnouncedAlive:
		parts = append(parts, "revival not announced")
	}

	return strings.Join(parts, " · ")
}

// grimoireSummary totals the characters, e.g. "Alive 6/8 · Good 5 · Evil 3 · 1 change not yet announced".
func grimoireSummary(players map[string]string, chars map[string]*Character) string {
	alive, good, evil, pending := 0, 0, 0, 0
	for id := range players {
		c, ok := chars[id]
		if !ok {
			continue
		}
		if c.Alive {
			alive++
		}
		if c.Team == TeamEvil {
			evil++
		} else {
			good++
		}
		if c.Alive != c.AnnouncedAlive {
			pending++
		}
	}

	summary := fmt.Sprintf("Alive %d/%d · Good %d · Evil %d", alive, good+evil, good, evil)
	if unassigned := len(players) - (good + evil); unassigned > 0 {
		summary += fmt.Sprintf(" · %d without a character", unassigned)
	}
	if pending > 0 {
		summary += fmt.Sprintf(" · %d change(s) not yet announced", pending)
	}
	return summary
}

// pendingLifeChanges returns the names of village players who have died or
// come back to life since the last announcement, each sorted by name.
func pendingLifeChanges(players map[string]string, chars map[string]*Character) (died, revived []string) {
	for id, name := range players {
		c, ok := chars[id]
		if !ok || c.Alive == c.AnnouncedAlive {
			continue
		}
		if c.Alive {
			revived = append(revived, name)
		} else {
			died = append(died, name)
		}
	}
	slices.Sort(died)
	slices.Sort(revived)
	return died, revived
}

// ghostVoteState describes a dead player's ghost vote.
func ghostVoteState(c *Character) string {
	if c.GhostVoteUsed {
		return "used"
	}
	return "available"
}

// chunkLines joins lines with newlines into messages no longer than limit.
// A single line longer than limit is cut.
func chunkLines(lines []string, limit int) []string {
	var chunks []string
	var sb strings.Builder
	for _, line := range lines {
		if len(line) > limit {
			line = line[:limit]
		}
		if sb.Len() > 0 && sb.Len()+1+len(line) > limit {
			chunks = append(chunks, sb.String())
			sb.Reset()
		}
		if sb.Len() > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(line)
	}
	if sb.Len() > 0 {
		chunks = append(chunks, sb.String())
	}
	return chunks
}

// mentionedCharacters returns the IDs of the mentioned players who have a
// character, in mention order, and a note for each mention without one.
func (b *Bot) mentionedCharacters(message *discordgo.MessageCreate) (ids, skipped []string) {
	for _, user := range message.Mentions {
		if _, ok := b.settings.Characters[user.ID]; ok {
			ids = append(ids, user.ID)
		} else {
			skipped = append(skipped, b.playerName(user.ID, user.Username)+" (no character)")
		}
	}
	return ids, skipped
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
		embed := b.dmEmbed(fmt.Sprintf("Your character: %s (%s)", c.Name, c.Team), c.Guidance)
		name := b.playerName(id, id)
		if err := b.sendDM(id, embed); err != nil {
			log.Printf("Could not DM %s their character: %v", id, err)
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
	b.reply(message, reply)
}

// whisper DMs a village player a secret message from the Storyteller straight away.
func (b *Bot) whisper(message *discordgo.MessageCreate) {
	userID, text, err := parseWhisper(message.Content)
	if err != nil {
		b.reply(message, fmt.Sprintf("Could not read that (%v). %s", err, whisperUsage))
		return
	}

	playerName, ok := b.settings.Players[userID]
	if !ok {
		b.reply(message, fmt.Sprintf("<@%s> isn't in the village. This command will not execute", userID))
		return
	}

	if err := b.sendDM(userID, b.dmEmbed("A message from the Storyteller", text)); err != nil {
		log.Printf("Could not whisper to %s: %v", userID, err)
		b.reply(message, fmt.Sprintf("Could not whisper to %s (%s).", playerName, dmErrorReason(err)))
		return
	}

	b.reply(message, fmt.Sprintf("Whispered to %s.", playerName))
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
