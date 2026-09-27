package bot

import (
	"fmt"
	"strings"
)

// characterList replies with the grimoire: every village player's character,
// team, life state and whether it has been sent, with totals at the top.
func (b *Bot) characterList(req *request) {
	if len(b.game.Players) == 0 {
		req.reply("The village is empty. Use `!botc village create` first.")
		return
	}

	lines := []string{"Grimoire: " + grimoireSummary(b.game)}
	for i, id := range b.game.sortedPlayerIDs() {
		lines = append(lines, fmt.Sprintf("%d. %s: %s", i+1, b.game.Players[id].Name, grimoireLine(b.game.Players[id])))
	}

	for _, chunk := range chunkLines(lines, maxMessageLength) {
		req.reply(chunk)
	}
}

// grimoireLine describes one player, e.g. "Monk (Good) · Dead, ghost vote used · sent · death not announced",
// or "none" for a player without a character. A player who died with a character
// is still shown as dead once it's cleared.
func grimoireLine(p *Player) string {
	c := p.Character
	parts := []string{"none"}
	if c != nil {
		parts = []string{fmt.Sprintf("%s (%s)", c.Name, c.Team)}
	}

	switch {
	case !p.Alive:
		parts = append(parts, "Dead, ghost vote "+ghostVoteState(p))
	case c != nil:
		parts = append(parts, "Alive")
	}

	if c != nil {
		if c.Sent {
			parts = append(parts, "sent")
		} else {
			parts = append(parts, "not sent")
		}
		if c.Guidance != "" {
			parts = append(parts, "has guidance")
		}
	}

	switch {
	case !p.Alive && p.AnnouncedAlive:
		parts = append(parts, "death not announced")
	case p.Alive && !p.AnnouncedAlive:
		parts = append(parts, "revival not announced")
	}

	return strings.Join(parts, " · ")
}

// grimoireSummary totals the characters, e.g. "Alive 6/8 · Good 5 · Evil 3 · 1 change not yet announced".
func grimoireSummary(g *Game) string {
	alive, good, evil, pending := 0, 0, 0, 0
	for _, p := range g.Players {
		if p.Alive != p.AnnouncedAlive {
			pending++
		}
		if p.Character == nil {
			continue
		}
		if p.Alive {
			alive++
		}
		if p.Character.Team == TeamEvil {
			evil++
		} else {
			good++
		}
	}

	summary := fmt.Sprintf("Alive %d/%d · Good %d · Evil %d", alive, good+evil, good, evil)
	if unassigned := len(g.Players) - (good + evil); unassigned > 0 {
		summary += fmt.Sprintf(" · %d without a character", unassigned)
	}
	if pending > 0 {
		summary += fmt.Sprintf(" · %d change(s) not yet announced", pending)
	}
	return summary
}

// ghostVoteState describes a dead player's ghost vote.
func ghostVoteState(p *Player) string {
	if p.GhostVoteUsed {
		return "used"
	}
	return "available"
}
