package bot

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"sync"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

type Settings struct {
	ApiToken       string
	GuildId        string
	AdminChannelId string // channel register was sent from; admin output goes here
	GameChannelId  string // Town Square voice channel; the bot joins it and posts game announcements there
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

// Discord roles the server must already have. The bot looks them up by exact name.
const (
	// storytellerRoleName is given to the user who registers the game.
	storytellerRoleName = "BoTC-StoryTeller"
	// playerRoleName marks players in the game. Only removed (on unregister), never assigned, by the bot.
	playerRoleName = "BoTC-Player"
)

type Bot struct {
	// mu serialises command handling. discordgo runs each event handler in its
	// own goroutine, so without it two commands could read and write settings at once.
	mu       sync.Mutex
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
			fmt.Printf("Recovered from panic handling %q: %v\n%s", message.Content, r, debug.Stack())
			b.discord.ChannelMessageSendReply(message.ChannelID, "Something went wrong running that command. Check the bot's logs.", message.Reference())
		}
	}()

	b.extractCommand(message, msgContents)
}

func (b *Bot) extractCommand(message *discordgo.MessageCreate, rawText []string) {
	for i := 0; i < len(rawText); i++ {
		fmt.Printf("MessageParam# %d MessageParamContent %q \n", i, rawText[i])
	}

	switch strings.ToLower(rawText[1]) {
	case "ping":
		b.discord.ChannelMessageSend(message.ChannelID, "pong")
	case "register", "start":
		b.register(message)
	case "unregister", "end":
		b.unregister(message)
	case "sitrep":
		b.sitrep(message)
	case "map":
		if b.settings.GameRegistered {
			if err := b.mapRooms(); err != nil {
				fmt.Printf("Could not map rooms: %v \n", err)
				b.discord.ChannelMessageSendReply(message.ChannelID, fmt.Sprintf("Could not map the town's channels (%v)", err), message.Reference())
				return
			}
			if err := b.mapPlayers(); err != nil {
				fmt.Printf("Could not map players: %v \n", err)
				b.discord.ChannelMessageSendReply(message.ChannelID, fmt.Sprintf("Could not map the players (%v)", err), message.Reference())
			}
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

func (b *Bot) mapRooms() error {
	fmt.Print(villageCodeLookup)

	allChans, err := b.getMapGuildChannels(b.settings.GuildId)
	if err != nil {
		return err
	}

	b.settings.Rooms = make(map[string]string)
	for index, ch := range allChans {
		fmt.Printf("index %s, data %s", index, ch)

		for code, room := range villageCodeLookup {
			if room == ch {
				b.settings.Rooms[code] = index
			}
		}
	}

	fmt.Print(b.settings.Rooms)
	b.discord.ChannelMessageSendTTS(b.settings.AdminChannelId, "Town Locations Mapped")
	return nil
}

func (b *Bot) getMapGuildChannels(guildId string) (map[string]string, error) {
	channels, err := b.discord.GuildChannels(guildId)
	if err != nil {
		return nil, err
	}

	chanMap := make(map[string]string)
	for _, ch := range channels {
		chanMap[ch.ID] = ch.Name
	}

	return chanMap, nil
}

func (b *Bot) mapPlayers() error {
	b.settings.Players = make(map[string]string)

	stateGuildData, err := b.discord.State.Guild(b.settings.GuildId)
	if err != nil {
		return err
	}

	channelData, err := b.discord.Channel(b.settings.Rooms["TS"])
	if err != nil {
		return err
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
	return nil
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

// findVoiceChannelID returns the ID of the server's first voice channel with exactly this name.
func (b *Bot) findVoiceChannelID(guildID, channelName string) (string, error) {
	channels, err := b.discord.GuildChannels(guildID)
	if err != nil {
		return "", err
	}

	for _, ch := range channels {
		if ch.Type == discordgo.ChannelTypeGuildVoice && ch.Name == channelName {
			return ch.ID, nil
		}
	}

	return "", fmt.Errorf("no voice channel named %q in this server", channelName)
}

// findRoleID returns the ID of the server's role with exactly this name.
func (b *Bot) findRoleID(guildID, roleName string) (string, error) {
	roles, err := b.discord.GuildRoles(guildID)
	if err != nil {
		return "", err
	}

	for _, role := range roles {
		if role.Name == roleName {
			return role.ID, nil
		}
	}

	return "", fmt.Errorf("no role named %q in this server", roleName)
}

// assignStorytellerRole gives the user the server's existing storytellerRoleName role.
func (b *Bot) assignStorytellerRole(guildID, userID string) error {
	roleID, err := b.findRoleID(guildID, storytellerRoleName)
	if err != nil {
		return err
	}

	return b.discord.GuildMemberRoleAdd(guildID, userID, roleID)
}

// removeGameRoles takes the Storyteller and Player roles off every member of the
// server who has them, including roles given out by hand. It returns how many
// roles were removed, and keeps going past individual failures.
func (b *Bot) removeGameRoles(guildID string) (int, error) {
	var errs []error

	gameRoleIDs := make(map[string]bool)
	for _, roleName := range []string{storytellerRoleName, playerRoleName} {
		roleID, err := b.findRoleID(guildID, roleName)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		gameRoleIDs[roleID] = true
	}

	if len(gameRoleIDs) == 0 {
		return 0, errors.Join(errs...)
	}

	// Discord returns at most 1000 members per request, so page through them.
	const pageSize = 1000
	removed := 0
	after := ""
	for {
		members, err := b.discord.GuildMembers(guildID, after, pageSize)
		if err != nil {
			errs = append(errs, err)
			break
		}

		for _, member := range members {
			for _, roleID := range member.Roles {
				if !gameRoleIDs[roleID] {
					continue
				}
				if err := b.discord.GuildMemberRoleRemove(guildID, member.User.ID, roleID); err != nil {
					errs = append(errs, fmt.Errorf("removing role from %s: %w", member.User.Username, err))
					continue
				}
				removed++
			}
		}

		if len(members) < pageSize {
			break
		}
		after = members[len(members)-1].User.ID
	}

	return removed, errors.Join(errs...)
}

func (b *Bot) sitrep(message *discordgo.MessageCreate) {
	var serverstate string

	if b.settings.GameRegistered {
		serverstate = fmt.Sprintf("Game is initialised at guildid# %s admin channel <#%s> game channel <#%s> storyteller <@%s>", b.settings.GuildId, b.settings.AdminChannelId, b.settings.GameChannelId, b.settings.StoryTellerId)
	} else {
		serverstate = "Game is not initialised"
	}

	sitrep := fmt.Sprintf("SITREP-%s", serverstate)
	b.discord.ChannelMessageSend(message.ChannelID, sitrep)
}

func (b *Bot) register(message *discordgo.MessageCreate) {
	if b.settings.GameRegistered {
		reply := fmt.Sprintf("A game is already registered, with <@%s> as the Storyteller. This command will not execute", b.settings.StoryTellerId)
		b.discord.ChannelMessageSendReply(message.ChannelID, reply, message.Reference())
		return
	}

	townSquareName := villageCodeLookup["TS"]
	gameChannelID, err := b.findVoiceChannelID(message.GuildID, townSquareName)
	if err != nil {
		reply := fmt.Sprintf("Could not find the game channel (%v). Create a voice channel named %q, then try again. This command will not execute", err, townSquareName)
		b.discord.ChannelMessageSendReply(message.ChannelID, reply, message.Reference())
		return
	}

	b.settings.GuildId = message.GuildID
	b.settings.AdminChannelId = message.ChannelID
	b.settings.GameChannelId = gameChannelID
	b.settings.StoryTellerId = message.Author.ID
	b.settings.GameRegistered = true

	if _, err := b.discord.ChannelVoiceJoin(b.settings.GuildId, b.settings.GameChannelId, false, false); err != nil {
		fmt.Printf("Could not join the game channel voice: %v \n", err)
	}

	fmt.Printf("The game has been registered at %s admin channel %s game channel %s storyteller %s \n", b.settings.GuildId, b.settings.AdminChannelId, b.settings.GameChannelId, b.settings.StoryTellerId)

	b.discord.ChannelMessageSend(b.settings.GameChannelId, fmt.Sprintf("A new game has begun. <@%s> is the Storyteller.", b.settings.StoryTellerId))

	reply := fmt.Sprintf("Game registered. <@%s> is the Storyteller. This is the admin channel; <#%s> is the game channel.", b.settings.StoryTellerId, b.settings.GameChannelId)
	if err := b.assignStorytellerRole(b.settings.GuildId, b.settings.StoryTellerId); err != nil {
		fmt.Printf("Could not assign the %s role: %v \n", storytellerRoleName, err)
		reply += fmt.Sprintf(" Warning: could not assign the %q role (%v). Check the role exists and sits below the bot's role.", storytellerRoleName, err)
	}
	b.discord.ChannelMessageSendReply(message.ChannelID, reply, message.Reference())
}

// unregister ends the current game: it removes the game roles from everyone,
// leaves voice, and resets the game state so a new game can be registered.
func (b *Bot) unregister(message *discordgo.MessageCreate) {
	if !b.settings.GameRegistered {
		b.discord.ChannelMessageSendReply(message.ChannelID, "No game registered, this command will not execute", message.Reference())
		return
	}

	if message.GuildID != b.settings.GuildId || message.Author.ID != b.settings.StoryTellerId {
		reply := fmt.Sprintf("Only the Storyteller (<@%s>) can end the game, from the server it was registered in. This command will not execute", b.settings.StoryTellerId)
		b.discord.ChannelMessageSendReply(message.ChannelID, reply, message.Reference())
		return
	}

	guildID := b.settings.GuildId

	removed, roleErr := b.removeGameRoles(guildID)

	if vc, ok := b.discord.VoiceConnections[guildID]; ok {
		if err := vc.Disconnect(); err != nil {
			fmt.Printf("Could not leave voice: %v \n", err)
		}
	}

	b.discord.ChannelMessageSend(b.settings.GameChannelId, "The game has ended. Thanks for playing!")

	b.settings.GuildId = "UNSET"
	b.settings.AdminChannelId = "UNSET"
	b.settings.GameChannelId = "UNSET"
	b.settings.StoryTellerId = "UNSET"
	b.settings.GameRegistered = false
	b.settings.Players = nil
	b.settings.Rooms = nil

	fmt.Printf("The game at %s has been unregistered, %d game roles removed \n", guildID, removed)

	reply := fmt.Sprintf("Game ended. Removed %d game role(s).", removed)
	if roleErr != nil {
		fmt.Printf("Problems removing game roles: %v \n", roleErr)
		reply += fmt.Sprintf(" Warning: some roles could not be removed (%v). Check the %q and %q roles exist and sit below the bot's role.", roleErr, storytellerRoleName, playerRoleName)
	}
	b.discord.ChannelMessageSendReply(message.ChannelID, reply, message.Reference())
}
