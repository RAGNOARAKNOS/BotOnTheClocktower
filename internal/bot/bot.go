package bot

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

type Settings struct {
	ApiToken       string
	GuildId        string
	ChannelId      string
	GameRegistered bool
	StoryTellerId  string
	Players        map[string]string
	Rooms          map[string]string
}

var villageCodeLookup = map[string]string{
	"TS": "Town Square",
	"CA": "Cathedral",
	"CF": "Campfire",
	"PS": "Potion Shop",
	"TW": "Tower",
	"RS": "Riverside",
	"SC": "Storyteller's Corner",
}

type Bot struct {
	settings Settings
	discord  *discordgo.Session
}

func Run(settings Settings) error {
	discord, err := discordgo.New("Bot " + settings.ApiToken)
	if err != nil {
		return err
	}

	discord.Identify.Intents = discordgo.IntentsAll

	b := &Bot{
		settings: settings,
		discord:  discord,
	}

	discord.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		fmt.Println("Bot is ready")
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
	if discord.State != nil && discord.State.User != nil && message.Author != nil && message.Author.ID == discord.State.User.ID {
		fmt.Println("dont talk to myself")
		return
	}

	msgContents := strings.Fields(message.Content)
	if len(msgContents) < 2 {
		return
	}

	if strings.Contains(msgContents[0], "!botc") {
		b.extractCommand(message, msgContents)
	}
}

func (b *Bot) extractCommand(message *discordgo.MessageCreate, rawText []string) {
	for i := 0; i < len(rawText); i++ {
		fmt.Printf("MessageParam# %d MessageParamContent %q \n", i, rawText[i])
	}

	switch rawText[1] {
	case "ping":
		b.discord.ChannelMessageSend(message.ChannelID, "pong")
	case "register":
		b.register(message)
	case "sitrep":
		b.sitrep(message)
	case "map":
		if b.settings.GameRegistered {
			b.mapRooms()
			b.mapPlayers()
		} else {
			fmt.Println("No game registered")
			b.discord.ChannelMessageSendReply(message.ChannelID, "No game registered, this command will not execute", message.Reference())
		}
	case "pmove":
	case "cmove":
	default:
		fmt.Println("default fall thru")
		b.discord.ChannelMessageSend(message.ChannelID, "Huh? WTF is that command?!")
	}
}

func (b *Bot) mapRooms() {
	fmt.Print(villageCodeLookup)

	b.settings.Rooms = make(map[string]string)

	allChans := b.getMapGuildChannels(b.settings.GuildId)
	for index, ch := range allChans {
		fmt.Printf("index %s, data %s", index, ch)

		for code, room := range villageCodeLookup {
			if room == ch {
				b.settings.Rooms[code] = index
			}
		}
	}

	fmt.Print(b.settings.Rooms)
	b.discord.ChannelMessageSendTTS(b.settings.ChannelId, "Town Locations Mapped")
}

func (b *Bot) getMapGuildChannels(guildId string) map[string]string {
	channels, err := b.discord.GuildChannels(guildId)
	if err != nil {
		panic(err)
	}

	chanMap := make(map[string]string)
	for _, ch := range channels {
		chanMap[ch.ID] = ch.Name
	}

	return chanMap
}

func (b *Bot) mapPlayers() {
	b.settings.Players = make(map[string]string)

	stateGuildData, err := b.discord.State.Guild(b.settings.GuildId)
	if err != nil {
		panic(err)
	}

	channelData, err := b.discord.Channel(b.settings.Rooms["TS"])
	if err != nil {
		panic(err)
	}

	voiceData := stateGuildData.VoiceStates

	var userIdsInChannel []string
	var userNamesInChannel []string

	for _, voice := range voiceData {
		if voice.ChannelID == b.settings.Rooms["TS"] {
			if voice.UserID != b.settings.StoryTellerId {
				userIdsInChannel = append(userIdsInChannel, voice.UserID)

				displayName := ""
				if voice.Member != nil {
					displayName = voice.Member.Nick
					if displayName == "" && voice.Member.User != nil {
						displayName = voice.Member.User.Username
					}
				}
				userNamesInChannel = append(userNamesInChannel, displayName)
			}
		}
	}

	fmt.Println("VOICES")
	fmt.Println(channelData.Name)
	fmt.Print(userIdsInChannel)
	fmt.Print(userNamesInChannel)
}

func (b *Bot) moveUserToChannel(playerID string, destinationChannelCode string) {
	destChannel := b.settings.Rooms[destinationChannelCode]
	b.discord.GuildMemberMove(b.settings.GuildId, playerID, &destChannel)
}

func (b *Bot) playerNameToId(playerName string) string {
	for key, value := range b.settings.Players {
		if value == playerName {
			return key
		}
	}

	return ""
}

func (b *Bot) sitrep(message *discordgo.MessageCreate) {
	var serverstate string

	if b.settings.GameRegistered {
		serverstate = fmt.Sprintf("Game is initialised at guildid# %s channel id# %s storyteller id# %s", b.settings.GuildId, b.settings.ChannelId, b.settings.StoryTellerId)
	} else {
		serverstate = "Game is not initialised"
	}

	sitrep := fmt.Sprintf("SITREP-%s", serverstate)
	b.discord.ChannelMessageSend(message.ChannelID, sitrep)
}

func (b *Bot) register(message *discordgo.MessageCreate) {
	b.settings.GuildId = message.GuildID
	b.settings.ChannelId = message.ChannelID
	b.settings.GameRegistered = true

	b.discord.ChannelVoiceJoin(b.settings.GuildId, b.settings.ChannelId, false, false)

	fmt.Printf("The game has been registered at %s channel %s \n", b.settings.GuildId, b.settings.ChannelId)

	for i := 0; i < len(b.discord.State.Guilds); i++ {
		tempGuild := b.discord.State.Guilds[i]
		fmt.Printf("Guild# %d GuildID %q GuildName %q \n", i, tempGuild.ID, tempGuild.Name)
	}

	tempGuild, err := b.discord.State.Guild(b.settings.GuildId)
	if err != nil {
		panic(err)
	}
	fmt.Println("Guild")
	fmt.Println(tempGuild.Name)

	tempChannel, err := b.discord.Channel(b.settings.ChannelId)
	if err != nil {
		panic(err)
	}
	fmt.Println("Channel")
	fmt.Println(tempChannel.Name)

	tempStateChannel, err := b.discord.State.Channel(b.settings.ChannelId)
	if err != nil {
		panic(err)
	}
	fmt.Println("Channel State")
	fmt.Println(tempStateChannel)

	members := b.discord.State.Guilds[0].Members
	fmt.Println("Members")
	fmt.Println(members)

	vstest := tempGuild.VoiceStates
	fmt.Print(vstest)

	fmt.Println("debug here")
}
