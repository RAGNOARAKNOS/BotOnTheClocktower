package bot

import (
	"fmt"
	"log"
	"slices"
	"strings"
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

// characterCommands maps each character subcommand to its handler.
var characterCommands = map[string]func(b *Bot, req *request){
	"assign":    (*Bot).characterAssign,
	"team":      (*Bot).characterTeam,
	"kill":      func(b *Bot, req *request) { b.characterSetAlive(req, false) },
	"revive":    func(b *Bot, req *request) { b.characterSetAlive(req, true) },
	"ghostvote": (*Bot).characterGhostVote,
	"announce":  (*Bot).characterAnnounce,
	"clear":     (*Bot).characterClear,
	"list":      (*Bot).characterList,
	"send":      (*Bot).characterSend,
}

// character dispatches the character subcommands.
func (b *Bot) character(req *request) {
	b.dispatch(req, characterCommands, characterUsage)
}

// characterAssign stores (or replaces) a player's character. It isn't sent until `character send`.
func (b *Bot) characterAssign(req *request) {
	userID, team, name, guidance, err := parseAssignment(req.content)
	if err != nil {
		req.reply(fmt.Sprintf("Could not read that (%v). %s", err, characterUsage))
		return
	}

	playerName, ok := b.game.Players[userID]
	if !ok {
		req.reply(fmt.Sprintf("<@%s> isn't in the village. Add them with `!botc village add` first.", userID))
		return
	}

	previous := b.game.Assign(userID, &Character{Name: name, Guidance: guidance, Team: team})

	reply := fmt.Sprintf("%s will be the %s (%s", playerName, name, team)
	if guidance != "" {
		reply += fmt.Sprintf(", with %d line(s) of guidance", strings.Count(guidance, "\n")+1)
	}
	reply += ")."
	if previous != nil {
		reply += fmt.Sprintf(" This replaces %s.", previous.Name)
	}
	reply += " Not sent yet; use `!botc character send`."
	req.reply(reply)
}

// characterTeam moves the mentioned players' characters to another team and marks them unsent.
func (b *Bot) characterTeam(req *request) {
	var team Team
	for _, word := range req.words[3:] {
		if t, ok := parseTeam(word); ok {
			team = t
		}
	}
	if team == "" || len(req.mentions) == 0 {
		req.reply("Mention the players and give the team, e.g. `!botc character team @player evil`.")
		return
	}

	ids, skipped := b.mentionedCharacters(req)
	var changed []string
	for _, id := range ids {
		if !b.game.SetTeam(id, team) {
			skipped = append(skipped, fmt.Sprintf("%s (already %s)", b.game.Players[id], team))
			continue
		}
		changed = append(changed, b.game.Players[id])
	}

	reply := fmt.Sprintf("Now %s: %d player(s)", team, len(changed))
	if len(changed) > 0 {
		reply += ": " + strings.Join(changed, ", ") + ". Not told yet; use `!botc character send` to send them their updated character."
	}
	if len(skipped) > 0 {
		reply += "\nSkipped: " + strings.Join(skipped, ", ")
	}
	req.reply(reply)
}

// characterSetAlive kills or revives the mentioned players. Nothing is posted
// publicly until `character announce`.
func (b *Bot) characterSetAlive(req *request, alive bool) {
	if len(req.mentions) == 0 {
		req.reply("Mention the players. " + characterUsage)
		return
	}

	state := "dead"
	if alive {
		state = "alive"
	}

	ids, skipped := b.mentionedCharacters(req)
	var changed []string
	for _, id := range ids {
		if !b.game.SetAlive(id, alive) {
			skipped = append(skipped, fmt.Sprintf("%s (already %s)", b.game.Players[id], state))
			continue
		}
		changed = append(changed, b.game.Players[id])
	}

	reply := fmt.Sprintf("Now %s: %d player(s)", state, len(changed))
	if len(changed) > 0 {
		reply += ": " + strings.Join(changed, ", ")
	}
	reply += "."
	if len(skipped) > 0 {
		reply += "\nSkipped: " + strings.Join(skipped, ", ")
	}
	died, revived := b.game.PendingLifeChanges()
	if pending := len(died) + len(revived); pending > 0 {
		reply += fmt.Sprintf("\n%d change(s) waiting to be announced; use `!botc character announce`.", pending)
	} else {
		reply += "\nNothing is waiting to be announced."
	}
	req.reply(reply)
}

// characterGhostVote switches the mentioned dead players' ghost votes between used and available.
func (b *Bot) characterGhostVote(req *request) {
	if len(req.mentions) == 0 {
		req.reply("Mention the dead players whose ghost vote to change. " + characterUsage)
		return
	}

	ids, skipped := b.mentionedCharacters(req)
	var changed []string
	for _, id := range ids {
		if !b.game.ToggleGhostVote(id) {
			skipped = append(skipped, b.game.Players[id]+" (alive, so no ghost vote)")
			continue
		}
		changed = append(changed, fmt.Sprintf("%s: ghost vote %s", b.game.Players[id], ghostVoteState(b.game.Characters[id])))
	}

	reply := "No ghost votes changed."
	if len(changed) > 0 {
		reply = strings.Join(changed, "\n")
	}
	if len(skipped) > 0 {
		reply += "\nSkipped: " + strings.Join(skipped, ", ")
	}
	req.reply(reply)
}

// characterAnnounce posts the deaths and revivals since the last announcement in
// the game channel, then marks them announced.
func (b *Bot) characterAnnounce(req *request) {
	died, revived := b.game.PendingLifeChanges()
	if len(died)+len(revived) == 0 {
		req.reply("Nothing to announce: no deaths or revivals since the last announcement.")
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

	if _, err := b.discord.ChannelMessageSend(b.game.GameChannelID, announcement); err != nil {
		log.Printf("Could not post the announcement: %v", err)
		req.reply(fmt.Sprintf("Could not post in <#%s> (%v). Nothing was marked as announced.", b.game.GameChannelID, err))
		return
	}

	b.game.MarkAnnounced()
	req.reply(fmt.Sprintf("Announced in <#%s>:\n%s", b.game.GameChannelID, announcement))
}

// characterClear removes the stored characters of the mentioned players.
func (b *Bot) characterClear(req *request) {
	if len(req.mentions) == 0 {
		req.reply("Mention the players to clear. " + characterUsage)
		return
	}

	var cleared, skipped []string
	for _, user := range req.mentions {
		if _, ok := b.game.Characters[user.ID]; !ok {
			skipped = append(skipped, user.Username)
			continue
		}
		delete(b.game.Characters, user.ID)
		cleared = append(cleared, b.playerName(user.ID, user.Username))
	}

	reply := fmt.Sprintf("Cleared %d character(s)", len(cleared))
	if len(cleared) > 0 {
		reply += ": " + strings.Join(cleared, ", ")
	}
	if len(skipped) > 0 {
		reply += "\nNo character to clear: " + strings.Join(skipped, ", ")
	}
	req.reply(reply)
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

// mentionedCharacters returns the IDs of the mentioned players who have a
// character, in mention order, and a note for each mention without one.
func (b *Bot) mentionedCharacters(req *request) (ids, skipped []string) {
	for _, user := range req.mentions {
		if _, ok := b.game.Characters[user.ID]; ok {
			ids = append(ids, user.ID)
		} else {
			skipped = append(skipped, b.playerName(user.ID, user.Username)+" (no character)")
		}
	}
	return ids, skipped
}

// characterSend DMs every unsent character, or, when players are mentioned,
// resends to just those players.
func (b *Bot) characterSend(req *request) {
	var targets, skipped []string
	if len(req.mentions) > 0 {
		for _, user := range req.mentions {
			if _, ok := b.game.Characters[user.ID]; ok {
				targets = append(targets, user.ID)
			} else {
				skipped = append(skipped, user.Username)
			}
		}
	} else {
		for _, id := range b.sortedPlayerIDs() {
			if c, ok := b.game.Characters[id]; ok && !c.Sent {
				targets = append(targets, id)
			}
		}
	}

	var sent, failed []string
	for _, id := range targets {
		c := b.game.Characters[id]
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
		if _, ok := b.game.Characters[id]; !ok {
			missing = append(missing, b.game.Players[id])
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
	req.reply(reply)
}

// playerName returns the village name for the user, or fallback if they aren't in the village.
func (b *Bot) playerName(userID, fallback string) string {
	if name, ok := b.game.Players[userID]; ok {
		return name
	}
	return fallback
}

// sortedPlayerIDs returns the village players' IDs, ordered by display name.
func (b *Bot) sortedPlayerIDs() []string {
	ids := make([]string, 0, len(b.game.Players))
	for id := range b.game.Players {
		ids = append(ids, id)
	}
	slices.SortFunc(ids, func(a, c string) int {
		return strings.Compare(b.game.Players[a], b.game.Players[c])
	})
	return ids
}
