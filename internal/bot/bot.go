package bot

import (
	"log"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"sync"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

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

	discord.Identify.Intents = discordgo.IntentsAll

	b := &Bot{discord: discord}

	discord.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Println("Bot is ready")
	})
	discord.AddHandler(b.newMessage)

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

	b.mu.Lock()
	defer b.mu.Unlock()

	// A panic in a handler would otherwise crash the whole bot and lose the game.
	defer func() {
		if r := recover(); r != nil {
			// Log the command name only: the rest of the message can hold game secrets.
			log.Printf("Recovered from panic handling %q: %v\n%s", msgContents[1], r, debug.Stack())
			b.reply(message, "Something went wrong running that command. Check the bot's logs.")
		}
	}()

	b.extractCommand(message, msgContents)
}
