package bot

import (
	"fmt"
	"log"
	"strings"
)

// Team is the side a character plays for.
type Team string

const (
	TeamGood Team = "Good"
	TeamEvil Team = "Evil"
)

// Character is the game character the Storyteller has given a player, sent to them
// by DM. Whether the player is alive is kept on their Player.
type Character struct {
	Name     string
	Guidance string // optional; kept exactly as typed
	Team     Team
	Sent     bool
}

// maxCharacterName keeps "Your character: <name> (<team>)" within Discord's 256-character embed title limit.
const maxCharacterName = 200

// characterAssign stores (or replaces) a player's character. It isn't sent until `character send`.
func (b *Bot) characterAssign(req *request) {
	userID, team, name, guidance, err := parseAssignment(req.content)
	if err != nil {
		req.reply(fmt.Sprintf("Could not read that (%v). %s", err, req.usage))
		return
	}

	player, ok := b.game.Players[userID]
	if !ok {
		req.reply(fmt.Sprintf("<@%s> isn't in the village. Add them with `!botc village add` first.", userID))
		return
	}

	previous := b.game.Assign(userID, &Character{Name: name, Guidance: guidance, Team: team})

	reply := fmt.Sprintf("%s will be the %s (%s", player.Name, name, team)
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
	for _, word := range req.args {
		if t, ok := parseTeam(word); ok {
			team = t
		}
	}
	if team == "" || len(req.mentions) == 0 {
		req.reply("Mention the players and give the team, e.g. `!botc character team @player evil`.")
		return
	}

	changed, skipped := b.eachMentioned(req, func(id string, _ *Player) string {
		if !b.game.SetTeam(id, team) {
			return "already " + string(team)
		}
		return ""
	})

	reply := countLine("Now "+string(team)+":", "player(s)", changed)
	if len(changed) > 0 {
		reply += ". Not told yet; use `!botc character send` to send them their updated character."
	}
	req.reply(reply + listLine("Skipped", skipped))
}

// characterSetAlive kills or revives the mentioned players. Nothing is posted
// publicly until `character announce`.
func (b *Bot) characterSetAlive(req *request, alive bool) {
	if len(req.mentions) == 0 {
		req.reply("Mention the players. " + req.usage)
		return
	}

	state := "dead"
	if alive {
		state = "alive"
	}

	changed, skipped := b.eachMentioned(req, func(id string, _ *Player) string {
		if !b.game.SetAlive(id, alive) {
			return "already " + state
		}
		return ""
	})

	reply := countLine("Now "+state+":", "player(s)", changed) + "." + listLine("Skipped", skipped)
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
		req.reply("Mention the dead players whose ghost vote to change. " + req.usage)
		return
	}

	var lines []string
	_, skipped := b.eachMentioned(req, func(id string, p *Player) string {
		if !b.game.ToggleGhostVote(id) {
			return "alive, so no ghost vote"
		}
		lines = append(lines, fmt.Sprintf("%s: ghost vote %s", p.Name, ghostVoteState(p)))
		return ""
	})

	reply := "No ghost votes changed."
	if len(lines) > 0 {
		reply = strings.Join(lines, "\n")
	}
	req.reply(reply + listLine("Skipped", skipped))
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
		req.reply("Mention the players to clear. " + req.usage)
		return
	}

	cleared, skipped := b.eachMentioned(req, func(id string, _ *Player) string {
		b.game.ClearCharacter(id)
		return ""
	})
	req.reply(countLine("Cleared", "character(s)", cleared) + listLine("Skipped", skipped))
}

// eachMentioned applies change to each mentioned player who has a character, in
// mention order. change returns "" if it changed the player, or why it didn't (such
// as "already dead"). eachMentioned returns the names of the players changed, and a
// note for each mention skipped, including those without a character.
func (b *Bot) eachMentioned(req *request, change func(id string, p *Player) (skip string)) (changed, skipped []string) {
	ids, skipped := b.mentionedCharacters(req)
	for _, id := range ids {
		p := b.game.Players[id]
		if why := change(id, p); why != "" {
			skipped = append(skipped, p.Name+" ("+why+")")
			continue
		}
		changed = append(changed, p.Name)
	}
	return changed, skipped
}

// mentionedCharacters returns the IDs of the mentioned players who have a
// character, in mention order, and a note for each mention without one.
func (b *Bot) mentionedCharacters(req *request) (ids, skipped []string) {
	for _, user := range req.mentions {
		if p, ok := b.game.Players[user.ID]; ok && p.Character != nil {
			ids = append(ids, user.ID)
		} else {
			skipped = append(skipped, b.game.playerName(user.ID, user.Username)+" (no character)")
		}
	}
	return ids, skipped
}

// characterSend DMs every unsent character, or, when players are mentioned,
// resends to just those players.
func (b *Bot) characterSend(req *request) {
	var targets, skipped []string
	if len(req.mentions) > 0 {
		targets, skipped = b.mentionedCharacters(req)
	} else {
		for _, id := range b.game.sortedPlayerIDs() {
			if c := b.game.Players[id].Character; c != nil && !c.Sent {
				targets = append(targets, id)
			}
		}
	}

	var sent, failed []string
	for _, id := range targets {
		c := b.game.Players[id].Character
		embed := b.dmEmbed(fmt.Sprintf("Your character: %s (%s)", c.Name, c.Team), c.Guidance)
		name := b.game.Players[id].Name
		if err := b.sendDM(id, embed); err != nil {
			log.Printf("Could not DM %s their character: %v", id, err)
			failed = append(failed, fmt.Sprintf("%s (%s)", name, dmErrorReason(err)))
			continue
		}
		b.game.MarkSent(id)
		sent = append(sent, name)
	}

	var missing []string
	for _, id := range b.game.sortedPlayerIDs() {
		if p := b.game.Players[id]; p.Character == nil {
			missing = append(missing, p.Name)
		}
	}

	var reply string
	if len(targets) == 0 && len(skipped) == 0 {
		reply = "Nothing to send: every assigned character has already been sent. Use `!botc character send @player` to resend."
	} else {
		reply = countLine("Sent", "character(s)", sent)
	}
	if len(failed) > 0 {
		reply += listLine("Failed", failed) + ". Fix the problem and run `!botc character send` again."
	}
	req.reply(reply + listLine("Skipped", skipped) + listLine("Village players with no character yet", missing))
}
