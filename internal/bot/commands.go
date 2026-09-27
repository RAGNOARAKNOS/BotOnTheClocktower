package bot

import (
	"fmt"
	"log"
	"slices"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// request is one command, from a `!botc` message or a `/botc` slash command, in a
// form the handlers can use without knowing which.
type request struct {
	authorID, guildID, channelID string
	// words is the command as words: "!botc" (or "/botc"), the command name, then its arguments.
	words []string
	// args are the words after the command's name, and after the subcommand's if it has one.
	args []string
	// usage shows how to write the command, starting "Usage: ", for replies to a command
	// that couldn't be read.
	usage string
	// content is the raw command text, for the commands that parse free text (parse.go).
	content string
	// mentions are the users the command names.
	mentions []*discordgo.User
	// reply answers the command: a threaded reply to a message, or the slash command's
	// response, visible only to the sender. say is the same, but not threaded.
	reply func(text string)
	say   func(text string)
}

// command is one `!botc` / `/botc` command, or a group of subcommands such as
// `village`. Everything about a command is declared in its entry in the commands
// table (commandtable.go): how it runs, who may run it, its help and its /botc
// options. By default a command only runs while a game is registered, for the
// Storyteller, in the admin channel; the flags relax that. A subcommand's access
// is its own: it doesn't inherit its group's flags.
type command struct {
	name string
	// aliases are other names for the command. They work as `!botc` words only;
	// /botc has just the name.
	aliases []string
	// description is the command's /botc help text, 1 to 100 characters.
	description string
	// usage shows how to write the command, starting with "`!botc <name>", e.g.
	// "`!botc character team @player good|evil`". The handler gets it in req.usage.
	usage string
	// run handles the command. A group has none: its subcommands run instead.
	run func(b *Bot, req *request)
	// subcommands make the command a group; the next word names one of them.
	subcommands []command
	// options are the command's /botc options. slashWords turns them into the words
	// `!botc` would have, so the handler reads them from req.args and req.mentions.
	options []*discordgo.ApplicationCommandOption
	// form, if set, makes /botc answer with a form (see slashCommand); submitting
	// it runs the command.
	form formOpener

	// beforeGame lets anyone run the command, from any channel, while no game is registered.
	beforeGame bool
	// anyChannel lets the Storyteller run the command outside the admin channel.
	anyChannel bool
}

// runCommand runs the command the words name, if the sender may run it.
// It returns false if the command was ignored without a reply.
func (b *Bot) runCommand(req *request) (answered bool) {
	cmd, used, known := resolve(req.words[1:])

	ok, refusal := allowed(b.game, cmd, req.authorID, req.guildID, req.channelID)
	if !ok {
		if refusal == "" {
			log.Printf("Ignoring %q: no game registered", strings.ToLower(req.words[1]))
			return false
		}
		req.reply(refusal)
		return true
	}

	switch {
	case !known:
		req.say("Huh? WTF is that command?!")
	case cmd.run == nil:
		// A group, without a subcommand it knows.
		req.reply(cmd.help())
	default:
		req.args = req.words[1+used:]
		req.usage = cmd.help()
		cmd.run(b, req)
	}
	return true
}

// resolve follows names (a command's name or alias, then a subcommand's name) through
// the commands table. It returns the command they lead to and how many names that
// took. For a group whose subcommand is missing or unknown, that's the group itself.
// known is false if the first name isn't a command.
func resolve(names []string) (cmd command, used int, known bool) {
	if len(names) == 0 {
		return command{}, 0, false
	}
	if cmd, known = findCommand(commands, names[0]); !known {
		return command{}, 0, false
	}
	for used = 1; used < len(names) && len(cmd.subcommands) > 0; used++ {
		sub, ok := findCommand(cmd.subcommands, names[used])
		if !ok {
			break
		}
		cmd = sub
	}
	return cmd, used, true
}

// findCommand returns the command in cmds with this name or alias, ignoring capitals.
func findCommand(cmds []command, name string) (command, bool) {
	matches := func(s string) bool { return strings.EqualFold(s, name) }
	for _, c := range cmds {
		if matches(c.name) || slices.ContainsFunc(c.aliases, matches) {
			return c, true
		}
	}
	return command{}, false
}

// help returns how to write the command, starting "Usage: ": its own usage, or for a
// group, each of its subcommands'.
func (c command) help() string {
	if len(c.subcommands) == 0 {
		return "Usage: " + c.usage
	}
	forms := make([]string, len(c.subcommands))
	for i, sub := range c.subcommands {
		forms[i] = sub.usage
	}
	return "Usage: " + strings.Join(forms, ", ")
}

// allowed decides whether a command may run. With no game registered, only commands
// marked beforeGame run, for anyone, from anywhere; anything else is ignored, which
// allowed signals with an empty refusal. Once a game is registered, commands only run
// for the Storyteller, in the admin channel of the registered server (anyChannel
// commands skip the server and channel check); anything else gets the refusal text.
// Unknown commands are checked like any other command, with no flags set.
func allowed(game *Game, cmd command, authorID, guildID, channelID string) (ok bool, refusal string) {
	if game == nil {
		return cmd.beforeGame, ""
	}

	isStoryteller := authorID == game.StorytellerID
	inAdmin := guildID == game.GuildID && channelID == game.AdminChannelID
	if isStoryteller && (inAdmin || cmd.anyChannel) {
		return true, ""
	}

	return false, fmt.Sprintf("Commands only work for the Storyteller (<@%s>), in the admin channel <#%s>. This command will not execute", game.StorytellerID, game.AdminChannelID)
}

// ping lets anyone check the bot is online and reading commands.
func (b *Bot) ping(req *request) {
	req.say("pong")
}

// reply answers a command as a threaded reply in the channel it came from.
// Send failures are logged, as there's nowhere else to report them.
func (b *Bot) reply(message *discordgo.MessageCreate, text string) {
	if _, err := b.discord.ChannelMessageSendReply(message.ChannelID, text, message.Reference()); err != nil {
		log.Printf("Could not reply in channel %s: %v", message.ChannelID, err)
	}
}

// send posts a plain message in a channel, logging any failure.
func (b *Bot) send(channelID, text string) {
	if _, err := b.discord.ChannelMessageSend(channelID, text); err != nil {
		log.Printf("Could not post in channel %s: %v", channelID, err)
	}
}
