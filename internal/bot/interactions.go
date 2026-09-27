package bot

import (
	"fmt"
	"log"
	"runtime/debug"
	"strings"

	"github.com/bwmarrin/discordgo"
)

const noGameReply = "No game registered. Start one with `/botc register`."

// registerSlashCommands sets up /botc on a server. discordgo sends GuildCreate for
// every server when the bot connects, and again when it joins a new one.
func (b *Bot) registerSlashCommands(s *discordgo.Session, g *discordgo.GuildCreate) {
	// A bot's application ID is its user ID.
	if _, err := s.ApplicationCommandBulkOverwrite(s.State.User.ID, g.ID, slashCommands()); err != nil {
		log.Printf("Could not register /%s on server %s (was the bot invited with the applications.commands scope?): %v", slashCommandName, g.ID, err)
	}
}

// interaction handles /botc commands and the forms they open, like newMessage does
// for `!botc`: one at a time under Bot.mu, recovering from panics.
func (b *Bot) interaction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.GuildID == "" || i.Member == nil || i.Member.User == nil {
		return
	}
	var name string
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		if i.ApplicationCommandData().Name != slashCommandName {
			return
		}
		path, _ := slashOptions(i.ApplicationCommandData())
		name = strings.Join(path, " ")
	case discordgo.InteractionModalSubmit:
		if !strings.HasPrefix(i.ModalSubmitData().CustomID, modalPrefix) {
			return
		}
		name = "form"
	default:
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	r := &slashResponder{b: b, i: i.Interaction}
	defer func() {
		if rec := recover(); rec != nil {
			// Log the command name only: the options can hold game secrets.
			log.Printf("Recovered from panic handling %q: %v\n%s", name, rec, debug.Stack())
			r.reply("Something went wrong running that command. Check the bot's logs.")
		}
	}()

	if i.Type == discordgo.InteractionApplicationCommand {
		b.slashCommand(i, r)
	} else {
		b.modalSubmit(i, r)
	}
}

// slashCommand runs a /botc command through the same handlers as `!botc`.
func (b *Bot) slashCommand(i *discordgo.InteractionCreate, r *slashResponder) {
	data := i.ApplicationCommandData()
	words, mentionIDs := slashWords(data)
	if len(words) < 2 {
		r.reply("Choose a command.")
		return
	}
	req := slashRequest(i, r, words, "")

	// A form must be the first response, so check access before opening one.
	path, options := slashOptions(data)
	if key := strings.Join(path[1:], " "); key == "whisper" || key == "character assign" {
		ok, refusal := allowed(b.game, commands[path[1]], req.authorID, req.guildID, req.channelID)
		if !ok {
			r.reply(orNoGame(refusal))
			return
		}
		userID := mentionIDs[0]
		playerName, inVillage := b.game.Players[userID]
		if !inVillage {
			r.reply(fmt.Sprintf("<@%s> isn't in the village. This command will not execute", userID))
			return
		}
		if key == "whisper" {
			r.modal(whisperModal(userID, playerName))
		} else {
			r.modal(assignModal(userID, playerName, stringOption(options, "team"), stringOption(options, "character")))
		}
		return
	}

	if !r.acknowledge() {
		return
	}
	req.mentions = b.resolveUsers(i.GuildID, mentionIDs)
	b.runSlash(req, r)
}

// modalSubmit runs the command a submitted form was opened for.
func (b *Bot) modalSubmit(i *discordgo.InteractionCreate, r *slashResponder) {
	data := i.ModalSubmitData()
	words, content, userID, err := modalCommand(data.CustomID, modalFields(data))
	if err != nil {
		r.reply("Could not read that form.")
		return
	}
	if !r.acknowledge() {
		return
	}
	req := slashRequest(i, r, words, content)
	req.mentions = b.resolveUsers(i.GuildID, []string{userID})
	b.runSlash(req, r)
}

// runSlash runs a request, making sure the slash command always gets an answer.
func (b *Bot) runSlash(req *request, r *slashResponder) {
	if !b.extractCommand(req) {
		r.reply(noGameReply)
	}
	if !r.replied {
		r.reply("Done.")
	}
}

// slashRequest builds a request from an interaction; its replies go to r.
func slashRequest(i *discordgo.InteractionCreate, r *slashResponder, words []string, content string) *request {
	return &request{
		authorID:  i.Member.User.ID,
		guildID:   i.GuildID,
		channelID: i.ChannelID,
		words:     words,
		content:   content,
		reply:     r.reply,
		say:       r.reply,
	}
}

// orNoGame returns the refusal, or the no-game reply for a command ignored because
// no game is registered (a slash command can't be ignored silently).
func orNoGame(refusal string) string {
	if refusal == "" {
		return noGameReply
	}
	return refusal
}

// resolveUsers looks up the users with these IDs, from the state cache or Discord.
// A user who can't be found is kept, with their mention as the name.
func (b *Bot) resolveUsers(guildID string, ids []string) []*discordgo.User {
	users := make([]*discordgo.User, 0, len(ids))
	for _, id := range ids {
		if m, err := b.discord.State.Member(guildID, id); err == nil && m.User != nil {
			users = append(users, m.User)
		} else if m, err := b.discord.GuildMember(guildID, id); err == nil && m.User != nil {
			users = append(users, m.User)
		} else {
			users = append(users, &discordgo.User{ID: id, Username: "<@" + id + ">"})
		}
	}
	return users
}

// slashResponder answers one interaction. Everything it sends is ephemeral: only
// the person who ran the command sees it.
type slashResponder struct {
	b *Bot
	i *discordgo.Interaction
	// acknowledged: Discord has had its first response. replied: a reply has been shown.
	acknowledged, replied bool
}

// acknowledge tells Discord the command is being worked on ("thinking…"), which
// must happen within 3 seconds. It returns false if Discord couldn't be told.
func (r *slashResponder) acknowledge() bool {
	err := r.b.discord.InteractionRespond(r.i, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Flags: discordgo.MessageFlagsEphemeral},
	})
	if err != nil {
		log.Printf("Could not acknowledge a slash command in channel %s: %v", r.i.ChannelID, err)
		return false
	}
	r.acknowledged = true
	return true
}

// reply shows text to the person who ran the command: as the first response, by
// filling in the acknowledgement, or as a follow-up message.
func (r *slashResponder) reply(text string) {
	var err error
	switch {
	case !r.acknowledged:
		err = r.b.discord.InteractionRespond(r.i, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: text, Flags: discordgo.MessageFlagsEphemeral},
		})
		r.acknowledged = true
	case !r.replied:
		_, err = r.b.discord.InteractionResponseEdit(r.i, &discordgo.WebhookEdit{Content: &text})
	default:
		_, err = r.b.discord.FollowupMessageCreate(r.i, true, &discordgo.WebhookParams{Content: text, Flags: discordgo.MessageFlagsEphemeral})
	}
	r.replied = true
	if err != nil {
		log.Printf("Could not reply to a slash command in channel %s: %v", r.i.ChannelID, err)
	}
}

// modal answers the command by opening a form.
func (r *slashResponder) modal(data *discordgo.InteractionResponseData) {
	err := r.b.discord.InteractionRespond(r.i, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseModal, Data: data})
	r.acknowledged, r.replied = true, true
	if err != nil {
		log.Printf("Could not open a form in channel %s: %v", r.i.ChannelID, err)
	}
}
