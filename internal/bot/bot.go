package bot

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"slices"
	"strings"
	"sync"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

type Settings struct {
	ApiToken       string
	GuildId        string
	AdminChannelId string // channel register was sent from; admin output goes here
	GameChannelId  string // Town Square voice channel; game announcements go to its text chat
	GameRegistered bool
	StoryTellerId  string
	Players        map[string]string     // village: user ID → display name
	Characters     map[string]*Character // user ID → character; only for players in the village
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
	// playerRoleName marks players in the village. Given and taken by the village
	// commands, and removed from everyone on unregister.
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
			}
		} else {
			fmt.Println("No game registered")
			b.discord.ChannelMessageSendReply(message.ChannelID, "No game registered, this command will not execute", message.Reference())
		}
	case "village":
		b.village(message, rawText)
	case "character":
		b.character(message, rawText)
	case "grimoire":
		if b.requireStorytellerInAdmin(message) {
			b.characterList(message)
		}
	case "whisper":
		b.whisper(message)
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

const villageUsage = "Usage: `!botc village create`, `!botc village add @player...`, `!botc village remove @player...`, `!botc village list`"

// village dispatches the village subcommands. All of them are Storyteller-only, from the admin channel.
func (b *Bot) village(message *discordgo.MessageCreate, rawText []string) {
	if !b.requireStorytellerInAdmin(message) {
		return
	}

	if len(rawText) < 3 {
		b.discord.ChannelMessageSendReply(message.ChannelID, villageUsage, message.Reference())
		return
	}

	switch strings.ToLower(rawText[2]) {
	case "create":
		b.villageCreate(message)
	case "add":
		b.villageAdd(message)
	case "remove":
		b.villageRemove(message)
	case "list":
		b.villageList(message)
	default:
		b.discord.ChannelMessageSendReply(message.ChannelID, villageUsage, message.Reference())
	}
}

// villageCreate replaces the player list with everyone in Town Square voice,
// except the Storyteller and bots, and makes BoTC-Player match the new list.
func (b *Bot) villageCreate(message *discordgo.MessageCreate) {
	guild, err := b.discord.State.Guild(b.settings.GuildId)
	if err != nil {
		reply := fmt.Sprintf("Could not read who is in <#%s> (%v). This command will not execute", b.settings.GameChannelId, err)
		b.discord.ChannelMessageSendReply(message.ChannelID, reply, message.Reference())
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
		if voice.ChannelID == b.settings.GameChannelId && voice.UserID != b.settings.StoryTellerId {
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

	dropped := make(map[string]string)
	for id, name := range b.settings.Players {
		if _, ok := players[id]; !ok {
			dropped[id] = name
		}
	}

	// Give the role to everyone in the new list, not just newcomers, so an earlier failure gets fixed.
	roleErr := b.setPlayerRole(players, dropped)
	b.settings.Players = players
	for id := range dropped {
		delete(b.settings.Characters, id)
	}

	var reply string
	if len(players) == 0 {
		reply = fmt.Sprintf("Village created, but it is empty: nobody except the Storyteller is in <#%s>.", b.settings.GameChannelId)
	} else {
		reply = fmt.Sprintf("Village created with %d player(s): %s", len(players), strings.Join(sortedNames(players), ", "))
	}
	if len(dropped) > 0 {
		reply += fmt.Sprintf("\nNo longer in the village: %s", strings.Join(sortedNames(dropped), ", "))
	}
	b.replyWithRoleWarning(message, reply, roleErr)
}

// villageAdd adds each mentioned user to the village and gives them BoTC-Player.
func (b *Bot) villageAdd(message *discordgo.MessageCreate) {
	if len(message.Mentions) == 0 {
		b.discord.ChannelMessageSendReply(message.ChannelID, "Mention the players to add. "+villageUsage, message.Reference())
		return
	}

	if b.settings.Players == nil {
		b.settings.Players = make(map[string]string)
	}

	added := make(map[string]string)
	var skipped []string
	for _, user := range message.Mentions {
		switch {
		case user.Bot:
			skipped = append(skipped, user.Username+" (bot)")
		case user.ID == b.settings.StoryTellerId:
			skipped = append(skipped, user.Username+" (the Storyteller)")
		case b.settings.Players[user.ID] != "":
			skipped = append(skipped, b.settings.Players[user.ID]+" (already in the village)")
		default:
			added[user.ID] = memberDisplayName(b.lookupMember(user.ID, nil), user.ID)
		}
	}

	roleErr := b.setPlayerRole(added, nil)
	for id, name := range added {
		b.settings.Players[id] = name
	}

	reply := fmt.Sprintf("Added %d player(s)", len(added))
	if len(added) > 0 {
		reply += ": " + strings.Join(sortedNames(added), ", ")
	}
	if len(skipped) > 0 {
		reply += "\nSkipped: " + strings.Join(skipped, ", ")
	}
	b.replyWithRoleWarning(message, reply, roleErr)
}

// villageRemove removes each mentioned user from the village and takes BoTC-Player away.
func (b *Bot) villageRemove(message *discordgo.MessageCreate) {
	if len(message.Mentions) == 0 {
		b.discord.ChannelMessageSendReply(message.ChannelID, "Mention the players to remove. "+villageUsage, message.Reference())
		return
	}

	removed := make(map[string]string)
	var skipped []string
	for _, user := range message.Mentions {
		name, ok := b.settings.Players[user.ID]
		if !ok {
			skipped = append(skipped, user.Username+" (not in the village)")
			continue
		}
		removed[user.ID] = name
	}

	roleErr := b.setPlayerRole(nil, removed)
	for id := range removed {
		delete(b.settings.Players, id)
		delete(b.settings.Characters, id)
	}

	reply := fmt.Sprintf("Removed %d player(s)", len(removed))
	if len(removed) > 0 {
		reply += ": " + strings.Join(sortedNames(removed), ", ")
	}
	if len(skipped) > 0 {
		reply += "\nSkipped: " + strings.Join(skipped, ", ")
	}
	b.replyWithRoleWarning(message, reply, roleErr)
}

// villageList replies with the current players, numbered and sorted by name.
func (b *Bot) villageList(message *discordgo.MessageCreate) {
	if len(b.settings.Players) == 0 {
		b.discord.ChannelMessageSendReply(message.ChannelID, "The village is empty. Use `!botc village create` or `!botc village add @player`.", message.Reference())
		return
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "The village has %d player(s):", len(b.settings.Players))
	for i, name := range sortedNames(b.settings.Players) {
		fmt.Fprintf(&sb, "\n%d. %s", i+1, name)
	}
	b.discord.ChannelMessageSendReply(message.ChannelID, sb.String(), message.Reference())
}

// requireStorytellerInAdmin replies and returns false unless a game is registered and
// the message is from the Storyteller, in the admin channel of the registered server.
func (b *Bot) requireStorytellerInAdmin(message *discordgo.MessageCreate) bool {
	if !b.settings.GameRegistered {
		b.discord.ChannelMessageSendReply(message.ChannelID, "No game registered, this command will not execute", message.Reference())
		return false
	}

	if message.GuildID != b.settings.GuildId || message.Author.ID != b.settings.StoryTellerId || message.ChannelID != b.settings.AdminChannelId {
		reply := fmt.Sprintf("Only the Storyteller (<@%s>) can run this, from the admin channel <#%s>. This command will not execute", b.settings.StoryTellerId, b.settings.AdminChannelId)
		b.discord.ChannelMessageSendReply(message.ChannelID, reply, message.Reference())
		return false
	}

	return true
}

// setPlayerRole gives playerRoleName to everyone in add and takes it from everyone
// in remove (both map user ID to display name). It keeps going past individual failures.
func (b *Bot) setPlayerRole(add, remove map[string]string) error {
	if len(add) == 0 && len(remove) == 0 {
		return nil
	}

	roleID, err := b.findRoleID(b.settings.GuildId, playerRoleName)
	if err != nil {
		return err
	}

	var errs []error
	for id, name := range add {
		if err := b.discord.GuildMemberRoleAdd(b.settings.GuildId, id, roleID); err != nil {
			errs = append(errs, fmt.Errorf("giving role to %s: %w", name, err))
		}
	}
	for id, name := range remove {
		if err := b.discord.GuildMemberRoleRemove(b.settings.GuildId, id, roleID); err != nil {
			errs = append(errs, fmt.Errorf("removing role from %s: %w", name, err))
		}
	}

	return errors.Join(errs...)
}

// replyWithRoleWarning sends reply, adding a warning if the player role couldn't be updated.
func (b *Bot) replyWithRoleWarning(message *discordgo.MessageCreate, reply string, roleErr error) {
	if roleErr != nil {
		fmt.Printf("Problems updating the %s role: %v \n", playerRoleName, roleErr)
		reply += fmt.Sprintf("\nWarning: the %q role could not be fully updated (%v). Check the role exists and sits below the bot's role.", playerRoleName, roleErr)
	}
	b.discord.ChannelMessageSendReply(message.ChannelID, reply, message.Reference())
}

// lookupMember returns known if it has user details, otherwise the member from
// the state cache or, failing that, from Discord. It returns nil if all of those fail.
func (b *Bot) lookupMember(userID string, known *discordgo.Member) *discordgo.Member {
	if known != nil && known.User != nil {
		return known
	}
	if member, err := b.discord.State.Member(b.settings.GuildId, userID); err == nil && member.User != nil {
		return member
	}
	if member, err := b.discord.GuildMember(b.settings.GuildId, userID); err == nil && member.User != nil {
		return member
	}
	return nil
}

// memberDisplayName returns the member's server nickname, global name or username,
// or the user ID if the member couldn't be found.
func memberDisplayName(member *discordgo.Member, userID string) string {
	if member == nil || member.User == nil {
		return userID
	}
	return member.DisplayName()
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

	fmt.Printf("The game has been registered at %s admin channel %s game channel %s storyteller %s \n", b.settings.GuildId, b.settings.AdminChannelId, b.settings.GameChannelId, b.settings.StoryTellerId)

	b.discord.ChannelMessageSend(b.settings.GameChannelId, fmt.Sprintf("A new game has begun. <@%s> is the Storyteller.", b.settings.StoryTellerId))

	reply := fmt.Sprintf("Game registered. <@%s> is the Storyteller. This is the admin channel; <#%s> is the game channel.", b.settings.StoryTellerId, b.settings.GameChannelId)
	if err := b.assignStorytellerRole(b.settings.GuildId, b.settings.StoryTellerId); err != nil {
		fmt.Printf("Could not assign the %s role: %v \n", storytellerRoleName, err)
		reply += fmt.Sprintf(" Warning: could not assign the %q role (%v). Check the role exists and sits below the bot's role.", storytellerRoleName, err)
	}
	b.discord.ChannelMessageSendReply(message.ChannelID, reply, message.Reference())
}

// unregister ends the current game: it removes the game roles from everyone
// and resets the game state so a new game can be registered.
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

	b.discord.ChannelMessageSend(b.settings.GameChannelId, "The game has ended. Thanks for playing!")

	b.settings.GuildId = "UNSET"
	b.settings.AdminChannelId = "UNSET"
	b.settings.GameChannelId = "UNSET"
	b.settings.StoryTellerId = "UNSET"
	b.settings.GameRegistered = false
	b.settings.Players = nil
	b.settings.Characters = nil
	b.settings.Rooms = nil

	fmt.Printf("The game at %s has been unregistered, %d game roles removed \n", guildID, removed)

	reply := fmt.Sprintf("Game ended. Removed %d game role(s).", removed)
	if roleErr != nil {
		fmt.Printf("Problems removing game roles: %v \n", roleErr)
		reply += fmt.Sprintf(" Warning: some roles could not be removed (%v). Check the %q and %q roles exist and sit below the bot's role.", roleErr, storytellerRoleName, playerRoleName)
	}
	b.discord.ChannelMessageSendReply(message.ChannelID, reply, message.Reference())
}
