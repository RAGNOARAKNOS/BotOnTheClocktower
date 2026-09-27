package bot

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"sync"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

const panicReply = "Something went wrong running that command. Check the bot's logs."

type Bot struct {
	// mu serialises command handling. discordgo runs each event handler in its
	// own goroutine, so without it two commands could read and write the game at once.
	mu sync.Mutex
	// game is nil while no game is registered.
	game    *Game
	discord *discordgo.Session
}

// Run connects to Discord with the bot token and handles commands until SIGINT or SIGTERM.
func Run(token string) error {
	discord, err := discordgo.New("Bot " + token)
	if err != nil {
		return err
	}

	// Server Members and Message Content are privileged: enable them in the developer portal.
	discord.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMembers | discordgo.IntentsGuildVoiceStates |
		discordgo.IntentsGuildMessages | discordgo.IntentsDirectMessages | discordgo.IntentsMessageContent

	b := &Bot{discord: discord}

	discord.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Println("Bot is ready")
	})
	discord.AddHandler(b.newMessage)
	discord.AddHandler(b.interaction)
	discord.AddHandler(b.registerSlashCommands)

	if err := discord.Open(); err != nil {
		return err
	}
	defer discord.Close()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-stop

	return nil
}

func (b *Bot) newMessage(discord *discordgo.Session, message *discordgo.MessageCreate) {
	// Ignore other bots, including this one.
	if message.Author == nil || message.Author.Bot {
		return
	}

	msgContents := strings.Fields(message.Content)
	if len(msgContents) < 2 || !strings.EqualFold(msgContents[0], "!botc") {
		return
	}

	b.locked(fmt.Sprintf("command %q", msgContents[1]),
		func() { b.reply(message, panicReply) },
		func() { b.runCommand(b.messageRequest(message, msgContents)) })
}

// locked runs a command or timer step under Bot.mu, recovering from a panic so the
// game isn't lost. Only name is logged, as the rest can hold game secrets.
func (b *Bot) locked(name string, onPanic func(), run func()) {
	b.mu.Lock()
	defer b.mu.Unlock()

	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in %s: %v\n%s", name, r, debug.Stack())
			if onPanic != nil {
				onPanic()
			}
		}
	}()
	run()
}

// messageRequest turns a `!botc` message into a request. Replies are threaded to the message.
func (b *Bot) messageRequest(message *discordgo.MessageCreate, words []string) *request {
	return &request{
		authorID:  message.Author.ID,
		guildID:   message.GuildID,
		channelID: message.ChannelID,
		words:     words,
		content:   message.Content,
		mentions:  message.Mentions,
		reply:     func(text string) { b.reply(message, text) },
		say:       func(text string) { b.send(message.ChannelID, text) },
	}
}
