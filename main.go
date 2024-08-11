package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

// type for storing all game params
type DiscordBotSettings struct {
	ApiToken       string
	GuildId        string // Discord API calls servers "Guilds" for historical reasons
	ChannelId      string
	GameRegistered bool // true when a server's id and Channel have been stored in the two variables above
	StoryTellerId  string
	Players        map[string]string //index UserId data UserName
	Rooms          map[string]string //index RoomShortCode data channelId
}

// define an enum for room types
type LocationType int64

const (
	StoryTellerRoom LocationType = iota // value = position in array
	TribunalRoom
	VillageRoom
	PrivateRoom
)

type DiscordChannelData struct {
	ChannelId   string
	ChannelName string
	ChannelType LocationType
}

// globals

var GameSettings DiscordBotSettings

// Maintain a list of 2-char reference codes to the intended room
var villageCodeLookup = map[string]string{
	"TS": "Town Square",
	"CA": "Cathedral",
	"CF": "Campfire",
	"PS": "Potion Shop",
	"TW": "Tower",
	"RS": "Riverside",
	"SC": "Storyteller's Corner",
}

// improvement notes
// - look into registering "slash commands" in discord api rather than clumsy text parsing
// - look into variable player count
// - look into dynamically creating the rooms, rather than 'binding' to existing ones

// generic nil check logger
func checkNilError(e error) {
	if e != nil {
		log.Fatalf("Error message! Something has gone wrong... %v", e)
	}
}

func newMessage(discord *discordgo.Session, message *discordgo.MessageCreate) {
	// prevent the bot from reacting to its own messages
	if message.Author.ID == discord.State.User.ID {
		fmt.Println("dont talk to myself")
		return
	}

	// Deconstruct any new messages into paramaterised strings
	msgContents := strings.Fields(message.Content)

	// If the first param is a recognised command process it, otherwise return
	if strings.Contains(msgContents[0], "!botc") {
		// protect against accidental command misfire (aka no parameters)
		if len(msgContents) >= 2 {
			extractCommand(discord, message, msgContents, &GameSettings)
		}
	}
}

func extractCommand(discord *discordgo.Session, message *discordgo.MessageCreate, rawText []string, bs *DiscordBotSettings) {

	// Spit out the cmd segments (debug aide)
	for i := 0; i < len(rawText); i++ {
		fmt.Printf("MessageParam# %d MessageParamContent %q \n", i, rawText[i])
	}

	switch rawText[1] {
	case "ping": // simple liveness check "!botc ping"
		discord.ChannelMessageSend(message.ChannelID, "pong")
	case "register": // register a storyteller, guild, and bot-chat channel "!botc register"
		messy(discord, message)
	case "sitrep": // respond with a high level summary (debug) "!botc sitrep"
		sitrep(discord, message, bs)
	case "map": // map players to roles/slots - map channels to village zones "!botc map"
		if bs.GameRegistered {
			mapRooms(discord, bs)
			mapPlayers(discord, bs)
		} else {
			// no game registered to map
			fmt.Println("No game registered")
			discord.ChannelMessageSendReply(message.ChannelID, "No game registered, this command will not execute", message.Reference())
		}
	case "pmove": // move a player to a destination room "!botc pmove PLAYERNAME DESTINATIONCHANNELCODE"
	case "cmove": // move an entire channel's users to a destination room "!botc cmove SOURCECHANNELCODE DESTINATIONCHANNELCODE"
	default: // unparsed command rx
		fmt.Println("default fall thru")
		discord.ChannelMessageSend(message.ChannelID, "Huh? WTF is that command?!")
	}

}

// Identify and initialise the "rooms" (channel IDs) used within the game
func mapRooms(discord *discordgo.Session, bs *DiscordBotSettings) {

	fmt.Print(villageCodeLookup)

	bs.Rooms = make(map[string]string)

	// get all channels on the guild instance
	allChans := getMapGuildChannels(discord, bs.GuildId)

	for index, ch := range allChans {
		// filter the channels to BotC reserved channels only
		// note allChans is [id]name indexed i.e. id is the INDEX
		fmt.Printf("index %s, data %s", index, ch)

		// Iterate through the 'known rooms' list
		for code, room := range villageCodeLookup {
			if room == ch {
				// Add the channelId and the channel shortcode
				bs.Rooms[code] = index
			}
		}
	}

	fmt.Print(bs.Rooms)
	discord.ChannelMessageSendTTS(bs.ChannelId, "Town Locations Mapped")
}

// constructs a map of channel id and names for a given GuildId
func getMapGuildChannels(discord *discordgo.Session, guildId string) map[string]string {
	channels, err := discord.GuildChannels(guildId)
	checkNilError(err)

	chanMap := make(map[string]string)

	for _, ch := range channels {
		chanMap[ch.ID] = ch.Name
	}

	return chanMap
}

// Identify the players, and allocate to player slots, and configure village permissions?
func mapPlayers(discord *discordgo.Session, bs *DiscordBotSettings) {
	// https://discord.com/channels/118456055842734083/155361364909621248/1271200650499194951
	// Right, the API is weird - the VoiceStates is only updated in cache data, after an update call

	bs.Players = make(map[string]string) // init the player info

	stateGuildData, err := discord.State.Guild(bs.GuildId) // get the cache guild data (force an update)
	checkNilError(err)

	channelData, err := discord.Channel(bs.Rooms["TS"]) // get the channel data (for debugging)
	checkNilError(err)

	voiceData := stateGuildData.VoiceStates

	fmt.Println("*****")
	fmt.Println(channelData.Name)
	fmt.Println("VOICES")
	fmt.Print(voiceData)
	fmt.Println("*****")

	// List all players in the town square
	// ST does not get indexed
	var usersInChannel []string

	for _, voice := range voiceData {
		if voice.ChannelID == bs.Rooms["TS"] {
			if voice.UserID != bs.StoryTellerId {
				usersInChannel = append(usersInChannel, voice.UserID)
			}
		}
	}

	fmt.Print(usersInChannel)
}

func moveUserToChannel(discord *discordgo.Session, bs *DiscordBotSettings, playerId string, destinationChannelCode string) {

	destChannel := bs.Rooms[destinationChannelCode]

	discord.GuildMemberMove(bs.GuildId, playerId, &destChannel)
}

// From a given player name, search the players list, return ID or Blank
func playerNameToId(bs *DiscordBotSettings, playerName string) string {
	for key, value := range bs.Players {
		if value == playerName {
			return key
		}
	}

	return "" // couldnt find the name
}

// spit out a summary of the tracked states within the app to chat
func sitrep(discord *discordgo.Session, message *discordgo.MessageCreate, bs *DiscordBotSettings) {
	var sitrep string
	var serverstate string

	if bs.GameRegistered {
		serverstate = fmt.Sprintf("Game is initialised at guildid# %s channel id# %s", bs.GuildId, bs.ChannelId)
	} else {
		serverstate = "Game is not initialised"
	}

	sitrep = fmt.Sprintf("SITREP-%s", serverstate)
	discord.ChannelMessageSend(message.ChannelID, sitrep)
}

// "register game" tells the bot which server and channel the game is running via
func messy(discord *discordgo.Session, message *discordgo.MessageCreate) {
	GameSettings.GuildId = message.GuildID
	GameSettings.ChannelId = message.ChannelID

	GameSettings.GameRegistered = true

	discord.ChannelVoiceJoin(GameSettings.GuildId, GameSettings.ChannelId, false, false)

	fmt.Printf("The game has been registered at %s channel %s \n", GameSettings.GuildId, GameSettings.ChannelId)

	for i := 0; i < len(discord.State.Guilds); i++ {
		var tempGuild = discord.State.Guilds[i]
		fmt.Printf("Guild# %d GuildID %q GuildName %q \n", i, tempGuild.ID, tempGuild.Name)
	}

	var tempGuild, errGuild = discord.State.Guild(GameSettings.GuildId)
	checkNilError(errGuild)
	fmt.Println("Guild")
	fmt.Println(tempGuild.Name)

	var tempChannel, err = discord.Channel(GameSettings.ChannelId)
	checkNilError(err)
	fmt.Println("Channel")
	fmt.Println(tempChannel.Name)

	var tempStateChannel, errStateChannel = discord.State.Channel(GameSettings.ChannelId)
	checkNilError(errStateChannel)
	fmt.Println("Channel State")
	fmt.Println(tempStateChannel)

	var members = discord.State.Guilds[0].Members
	fmt.Println("Members")
	fmt.Println(members)

	var vstest = tempGuild.VoiceStates
	fmt.Print(vstest)

	// for i := 0; i < len(vstest); i++ {
	// 	fmt.Printf("voice state %d, text %q", i, vstest[i].Member.Nick)
	// }

	// var voicestate, vsError = discord.State.VoiceState(GameSettings.GuildId, message.Author.ID)
	// checkNilError(vsError)
	// fmt.Println("VOICE STATES by ID")
	// fmt.Println(voicestate)

	fmt.Println("debug here")
	//discord.GuildMemberMove()

	// https: //discord.com/channels/118456055842734083/155361364909621248/1173370114939297903  - looks like maintaing a list of who is in each chasnnel is going to be quite involved
	// https://discord.com/channels/118456055842734083/155361364909621248/821905418578165770
}

// Initialise all parameters for the bot, such as API key, and stage game management data structures
func initBotSettings() (DiscordBotSettings, error) {
	var newSettings DiscordBotSettings

	// Load configurable parameters via GOENV lib from .env file
	envErr := godotenv.Load()
	checkNilError(envErr)
	if envErr != nil {
		return newSettings, envErr
	}

	// get discord API token from ENVVARs
	newSettings.ApiToken = os.Getenv("BOTAPIKEY")

	// set fields to known value (for testing)
	newSettings.GuildId = "UNSET"
	newSettings.ChannelId = "UNSET"
	newSettings.StoryTellerId = "UNSET"
	newSettings.GameRegistered = false

	return newSettings, nil
}

func main() {

	// Collate game and bot parameters
	loadSettings, initErr := initBotSettings()
	checkNilError(initErr)

	GameSettings = loadSettings // discovered that Go allows "shadow" overrides of global variables!

	// configure discord API client
	discord, discordErr := discordgo.New("Bot " + GameSettings.ApiToken)
	checkNilError(discordErr)

	// register the bot 'intents' - make sure it matches settings
	// in the developer portal
	discord.Identify.Intents = discordgo.IntentsAll

	// register bot when Discord client ready event fires
	discord.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		fmt.Println("Bot is ready")
	})

	// register the funtion to process new messages
	discord.AddHandler(newMessage)

	// open the session
	discord.Open()
	defer discord.Close() // close session when function terminates

	// await the ctl+c interrupt to exit the program (indefinite loop)
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-stop
}
