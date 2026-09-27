package bot

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
)

const (
	// gatherDefault is the countdown when no time is given.
	gatherDefault = 60 * time.Second
	// gatherMaxMinutes is the longest countdown the Storyteller can set.
	gatherMaxMinutes = 10
	// gatherWarning is how long before the move the second warning goes out.
	gatherWarning = 30 * time.Second
)

// gatherCountdown is a running `gather`: the warning and the move, waiting to fire.
type gatherCountdown struct {
	ends time.Time
	warn *time.Timer
	move *time.Timer
}

// stop stops both timers. A callback that has already started finds the countdown
// is no longer the game's and does nothing (see gatherFire).
func (c *gatherCountdown) stop() {
	c.warn.Stop()
	c.move.Stop()
}

// parseGather reads the words after `gather`: nothing for the default countdown,
// "cancel", or a whole number of minutes from 1 to gatherMaxMinutes.
func parseGather(args []string) (d time.Duration, cancel bool, err error) {
	switch {
	case len(args) == 0:
		return gatherDefault, false, nil
	case len(args) > 1:
		return 0, false, errors.New("too many words")
	case strings.EqualFold(args[0], "cancel"):
		return 0, true, nil
	}
	minutes, err := strconv.Atoi(args[0])
	if err != nil || minutes < 1 || minutes > gatherMaxMinutes {
		return 0, false, fmt.Errorf("%q isn't a whole number of minutes from 1 to %d", args[0], gatherMaxMinutes)
	}
	return time.Duration(minutes) * time.Minute, false, nil
}

// formatCountdown writes a countdown to the second: "60 seconds" up to a minute,
// then "5 minutes" or "2 minutes 17 seconds".
func formatCountdown(d time.Duration) string {
	d = d.Round(time.Second)
	if d <= time.Minute {
		return plural(int(d/time.Second), "second")
	}
	out := plural(int(d/time.Minute), "minute")
	if s := int(d % time.Minute / time.Second); s > 0 {
		out += " " + plural(s, "second")
	}
	return out
}

// gather starts a countdown to move the players to Town Square, or cancels one.
func (b *Bot) gather(req *request) {
	d, cancel, err := parseGather(req.args)
	if err != nil {
		req.reply(fmt.Sprintf("Could not read that (%v). %s", err, req.usage))
		return
	}
	if cancel {
		b.gatherCancel(req)
		return
	}

	game := b.game
	if game.gather != nil {
		left := time.Until(game.gather.ends).Round(time.Second)
		req.reply(fmt.Sprintf("A gathering is already counting down, with %s left. Use `!botc gather cancel` to stop it. This command will not execute", formatCountdown(left)))
		return
	}

	c := &gatherCountdown{ends: time.Now().Add(d)}
	game.gather = c
	failures := b.gatherAnnounce(fmt.Sprintf("The Storyteller will be bringing everyone back to Town Square in %s.", formatCountdown(d)))
	c.warn = time.AfterFunc(d-gatherWarning, func() { b.gatherFire(game, c, "warning", b.gatherWarn) })
	c.move = time.AfterFunc(d, func() { b.gatherFire(game, c, "move", b.gatherMove) })

	reply := fmt.Sprintf("Gathering the players in Town Square in %s.", formatCountdown(d))
	if len(game.Rooms) == 0 {
		reply += " Only Town Square was told: run `!botc map` so the other rooms hear it too."
	}
	req.reply(reply + failureList(failures))
}

// gatherCancel stops the running countdown and tells everyone it's off.
func (b *Bot) gatherCancel(req *request) {
	if b.game.gather == nil {
		req.reply("No gathering is counting down.")
		return
	}
	b.game.gather.stop()
	b.game.gather = nil
	failures := b.gatherAnnounce("The Storyteller has called off the gathering in Town Square.")
	req.reply("Gathering cancelled." + failureList(failures))
}

// gatherFire runs a countdown step from its timer, like a command, and does nothing
// if the game was unregistered or replaced, or the countdown cancelled.
func (b *Bot) gatherFire(game *Game, c *gatherCountdown, step string, run func()) {
	b.locked("the gather "+step, nil, func() {
		if b.game == game && game.gather == c {
			run()
		}
	})
}

// gatherWarn sends the second warning, reporting any failures in the admin channel.
func (b *Bot) gatherWarn() {
	failures := b.gatherAnnounce(fmt.Sprintf("%s until everyone is brought back to Town Square.", formatCountdown(gatherWarning)))
	if len(failures) > 0 {
		b.send(b.game.AdminChannelID, "Gathering warning sent."+failureList(failures))
	}
}

// gatherMove moves every village player in voice, except the Storyteller, into
// Town Square, and reports the result in the admin channel.
func (b *Bot) gatherMove() {
	game := b.game
	game.gather = nil

	var moved int
	var notInVoice, failed []string
	for _, id := range b.game.sortedPlayerIDs() {
		if id == game.StorytellerID {
			continue
		}
		name := game.Players[id].Name
		state, err := b.discord.State.VoiceState(game.GuildID, id)
		switch {
		case err != nil || state.ChannelID == "":
			notInVoice = append(notInVoice, name)
			continue
		case state.ChannelID == game.GameChannelID:
			continue
		}
		if err := b.discord.GuildMemberMove(game.GuildID, id, &game.GameChannelID); err != nil {
			log.Printf("Could not move %s to Town Square: %v", id, err)
			failed = append(failed, fmt.Sprintf("%s (%v)", name, err))
			continue
		}
		moved++
	}

	report := fmt.Sprintf("Gathered %d player(s) in Town Square.", moved)
	report += listLine("Not in voice, so not moved", notInVoice) + listLine("Could not move", failed)
	b.send(game.AdminChannelID, report)
}

// gatherAnnounce sends text as a TTS message to each village voice channel's text
// chat and as a DM to each village player. It returns a line for each failure.
func (b *Bot) gatherAnnounce(text string) (failures []string) {
	for _, channelID := range b.game.villageChannels() {
		if _, err := b.discord.ChannelMessageSendTTS(channelID, text); err != nil {
			log.Printf("Could not post the gathering announcement in channel %s: %v", channelID, err)
			failures = append(failures, fmt.Sprintf("could not post in <#%s> (%v)", channelID, err))
		}
	}
	for _, id := range b.game.sortedPlayerIDs() {
		if err := b.sendDM(id, b.dmEmbed("Gathering in Town Square", text)); err != nil {
			log.Printf("Could not DM the gathering announcement to %s: %v", id, err)
			failures = append(failures, fmt.Sprintf("could not DM %s (%s)", b.game.Players[id].Name, dmErrorReason(err)))
		}
	}
	return failures
}
