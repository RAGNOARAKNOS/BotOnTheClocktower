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
		line := fmt.Sprintf("%d. %s: ", i+1, b.game.Players[id])
		if c, ok := b.game.Characters[id]; ok {
			line += grimoireLine(c)
		} else {
			line += "none"
		}
		lines = append(lines, line)
	}

	for _, chunk := range chunkLines(lines, maxMessageLength) {
		req.reply(chunk)
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
func grimoireSummary(g *Game) string {
	alive, good, evil, pending := 0, 0, 0, 0
	for id := range g.Players {
		c, ok := g.Characters[id]
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
	if unassigned := len(g.Players) - (good + evil); unassigned > 0 {
		summary += fmt.Sprintf(" · %d without a character", unassigned)
	}
	if pending > 0 {
		summary += fmt.Sprintf(" · %d change(s) not yet announced", pending)
	}
	return summary
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
		line = truncate(line, limit)
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
