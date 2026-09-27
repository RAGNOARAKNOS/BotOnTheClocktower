package bot

import (
	"fmt"
	"slices"
	"strings"

	"github.com/bwmarrin/discordgo"
)

const villageUsage = "Usage: `!botc village create`, `!botc village add @player...`, `!botc village remove @player...`, `!botc village list`"

// villageCommands maps each village subcommand to its handler.
var villageCommands = map[string]func(b *Bot, req *request){
	"create": (*Bot).villageCreate,
	"add":    (*Bot).villageAdd,
	"remove": (*Bot).villageRemove,
	"list":   (*Bot).villageList,
}

// village dispatches the village subcommands.
func (b *Bot) village(req *request) {
	b.dispatch(req, villageCommands, villageUsage)
}

// villageCreate replaces the player list with everyone in Town Square voice,
// except the Storyteller and bots, and makes BoTC-Player match the new list.
func (b *Bot) villageCreate(req *request) {
	guild, err := b.discord.State.Guild(b.game.GuildID)
	if err != nil {
		reply := fmt.Sprintf("Could not read who is in <#%s> (%v). This command will not execute", b.game.GameChannelID, err)
		req.reply(reply)
		return
	}

	// Copy what we need while holding the state lock; looking members up takes it again.
	type listener struct {
		userID string
		member *discordgo.Member
	}
	var inTownSquare []listener
	b.discord.State.RLock()
	for _, voice := range guild.VoiceStates {
		if voice.ChannelID == b.game.GameChannelID && voice.UserID != b.game.StorytellerID {
			inTownSquare = append(inTownSquare, listener{voice.UserID, voice.Member})
		}
	}
	b.discord.State.RUnlock()

	players := make(map[string]string)
	for _, l := range inTownSquare {
		member := b.lookupMember(l.userID, l.member)
		if member != nil && member.User != nil && member.User.Bot {
			continue
		}
		players[l.userID] = memberDisplayName(member, l.userID)
	}

	dropped := b.game.ReplacePlayers(players)
	// Give the role to everyone in the new list, not just newcomers, so an earlier failure gets fixed.
	roleErr := b.setPlayerRole(players, dropped)

	var reply string
	if len(players) == 0 {
		reply = fmt.Sprintf("Village created, but it is empty: nobody except the Storyteller is in <#%s>.", b.game.GameChannelID)
	} else {
		reply = fmt.Sprintf("Village created with %d player(s): %s", len(players), strings.Join(sortedNames(players), ", "))
	}
	if len(dropped) > 0 {
		reply += fmt.Sprintf("\nNo longer in the village: %s", strings.Join(sortedNames(dropped), ", "))
	}
	b.replyWithRoleWarning(req, reply, roleErr)
}

// villageAdd adds each mentioned user to the village and gives them BoTC-Player.
func (b *Bot) villageAdd(req *request) {
	if len(req.mentions) == 0 {
		req.reply("Mention the players to add. " + villageUsage)
		return
	}

	added := make(map[string]string)
	var skipped []string
	for _, user := range req.mentions {
		switch {
		case user.Bot:
			skipped = append(skipped, user.Username+" (bot)")
		case user.ID == b.game.StorytellerID:
			skipped = append(skipped, user.Username+" (the Storyteller)")
		case b.game.Players[user.ID] != "":
			skipped = append(skipped, b.game.Players[user.ID]+" (already in the village)")
		default:
			added[user.ID] = memberDisplayName(b.lookupMember(user.ID, nil), user.ID)
		}
	}

	roleErr := b.setPlayerRole(added, nil)
	for id, name := range added {
		b.game.Players[id] = name
	}

	reply := fmt.Sprintf("Added %d player(s)", len(added))
	if len(added) > 0 {
		reply += ": " + strings.Join(sortedNames(added), ", ")
	}
	if len(skipped) > 0 {
		reply += "\nSkipped: " + strings.Join(skipped, ", ")
	}
	b.replyWithRoleWarning(req, reply, roleErr)
}

// villageRemove removes each mentioned user from the village and takes BoTC-Player away.
func (b *Bot) villageRemove(req *request) {
	if len(req.mentions) == 0 {
		req.reply("Mention the players to remove. " + villageUsage)
		return
	}

	removed := make(map[string]string)
	var skipped []string
	for _, user := range req.mentions {
		name, ok := b.game.Players[user.ID]
		if !ok {
			skipped = append(skipped, user.Username+" (not in the village)")
			continue
		}
		removed[user.ID] = name
	}

	roleErr := b.setPlayerRole(nil, removed)
	for id := range removed {
		b.game.RemovePlayer(id)
	}

	reply := fmt.Sprintf("Removed %d player(s)", len(removed))
	if len(removed) > 0 {
		reply += ": " + strings.Join(sortedNames(removed), ", ")
	}
	if len(skipped) > 0 {
		reply += "\nSkipped: " + strings.Join(skipped, ", ")
	}
	b.replyWithRoleWarning(req, reply, roleErr)
}

// villageList replies with the current players, numbered and sorted by name.
func (b *Bot) villageList(req *request) {
	if len(b.game.Players) == 0 {
		req.reply("The village is empty. Use `!botc village create` or `!botc village add @player`.")
		return
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "The village has %d player(s):", len(b.game.Players))
	for i, name := range sortedNames(b.game.Players) {
		fmt.Fprintf(&sb, "\n%d. %s", i+1, name)
	}
	req.reply(sb.String())
}

// sortedNames returns the names (values) of an ID-to-name map in alphabetical order.
func sortedNames(players map[string]string) []string {
	names := make([]string, 0, len(players))
	for _, name := range players {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}
