package bot

import (
	"errors"
	"fmt"
	"log"
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

func (b *Bot) extractCommand(message *discordgo.MessageCreate, rawText []string) {
	command := strings.ToLower(rawText[1])
	if !b.commandAllowed(message, command) {
		return
	}

	switch command {
	case "ping":
		b.send(message.ChannelID, "pong")
	case "register", "start":
		b.register(message)
	case "unregister", "end":
		b.unregister(message)
	case "sitrep":
		b.sitrep(message)
	case "map":
		if err := b.mapRooms(); err != nil {
			log.Printf("Could not map rooms: %v", err)
			b.reply(message, fmt.Sprintf("Could not map the town's channels (%v)", err))
		}
	case "village":
		b.village(message, rawText)
	case "character":
		b.character(message, rawText)
	case "grimoire":
		b.characterList(message)
	case "whisper":
		b.whisper(message)
	default:
		b.send(message.ChannelID, "Huh? WTF is that command?!")
	}
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

func (b *Bot) mapRooms() error {
	allChans, err := b.getMapGuildChannels(b.settings.GuildId)
	if err != nil {
		return err
	}

	b.settings.Rooms = make(map[string]string)
	for index, ch := range allChans {
		for code, room := range villageCodeLookup {
			if room == ch {
				b.settings.Rooms[code] = index
			}
		}
	}

	if _, err := b.discord.ChannelMessageSendTTS(b.settings.AdminChannelId, "Town Locations Mapped"); err != nil {
		log.Printf("Could not post in channel %s: %v", b.settings.AdminChannelId, err)
	}
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

// village dispatches the village subcommands.
func (b *Bot) village(message *discordgo.MessageCreate, rawText []string) {
	if len(rawText) < 3 {
		b.reply(message, villageUsage)
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
		b.reply(message, villageUsage)
	}
}

// villageCreate replaces the player list with everyone in Town Square voice,
// except the Storyteller and bots, and makes BoTC-Player match the new list.
func (b *Bot) villageCreate(message *discordgo.MessageCreate) {
	guild, err := b.discord.State.Guild(b.settings.GuildId)
	if err != nil {
		reply := fmt.Sprintf("Could not read who is in <#%s> (%v). This command will not execute", b.settings.GameChannelId, err)
		b.reply(message, reply)
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
		b.reply(message, "Mention the players to add. "+villageUsage)
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
		b.reply(message, "Mention the players to remove. "+villageUsage)
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
		b.reply(message, "The village is empty. Use `!botc village create` or `!botc village add @player`.")
		return
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "The village has %d player(s):", len(b.settings.Players))
	for i, name := range sortedNames(b.settings.Players) {
		fmt.Fprintf(&sb, "\n%d. %s", i+1, name)
	}
	b.reply(message, sb.String())
}

// commandAllowed reports whether a command may run, and is checked before every command.
// With no game registered, only register/start and ping run, from any channel (register's
// channel becomes the admin channel), and every other command is ignored without a reply.
// Once a game is registered, every command must come from the Storyteller, in the
// admin channel of the registered server, except ping, which the Storyteller can send
// from anywhere; anything else gets a refusal.
func (b *Bot) commandAllowed(message *discordgo.MessageCreate, command string) bool {
	if !b.settings.GameRegistered {
		if command == "register" || command == "start" || command == "ping" {
			return true
		}
		log.Printf("Ignoring %q: no game registered", command)
		return false
	}

	if command == "ping" && message.Author.ID == b.settings.StoryTellerId {
		return true
	}

	if message.GuildID != b.settings.GuildId || message.Author.ID != b.settings.StoryTellerId || message.ChannelID != b.settings.AdminChannelId {
		reply := fmt.Sprintf("Commands only work for the Storyteller (<@%s>), in the admin channel <#%s>. This command will not execute", b.settings.StoryTellerId, b.settings.AdminChannelId)
		b.reply(message, reply)
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
		log.Printf("Problems updating the %s role: %v", playerRoleName, roleErr)
		reply += fmt.Sprintf("\nWarning: the %q role could not be fully updated (%v). Check the role exists and sits below the bot's role.", playerRoleName, roleErr)
	}
	b.reply(message, reply)
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

// sitrep reports where the game is running. It only runs while a game is registered.
func (b *Bot) sitrep(message *discordgo.MessageCreate) {
	b.send(message.ChannelID, fmt.Sprintf("SITREP-Game is initialised at guildid# %s admin channel <#%s> game channel <#%s> storyteller <@%s>", b.settings.GuildId, b.settings.AdminChannelId, b.settings.GameChannelId, b.settings.StoryTellerId))
}

func (b *Bot) register(message *discordgo.MessageCreate) {
	if b.settings.GameRegistered {
		reply := fmt.Sprintf("A game is already registered, with <@%s> as the Storyteller. This command will not execute", b.settings.StoryTellerId)
		b.reply(message, reply)
		return
	}

	townSquareName := villageCodeLookup["TS"]
	gameChannelID, err := b.findVoiceChannelID(message.GuildID, townSquareName)
	if err != nil {
		reply := fmt.Sprintf("Could not find the game channel (%v). Create a voice channel named %q, then try again. This command will not execute", err, townSquareName)
		b.reply(message, reply)
		return
	}

	b.settings.GuildId = message.GuildID
	b.settings.AdminChannelId = message.ChannelID
	b.settings.GameChannelId = gameChannelID
	b.settings.StoryTellerId = message.Author.ID
	b.settings.GameRegistered = true

	log.Printf("The game has been registered at %s admin channel %s game channel %s storyteller %s", b.settings.GuildId, b.settings.AdminChannelId, b.settings.GameChannelId, b.settings.StoryTellerId)

	b.send(b.settings.GameChannelId, fmt.Sprintf("A new game has begun. <@%s> is the Storyteller.", b.settings.StoryTellerId))

	reply := fmt.Sprintf("Game registered. <@%s> is the Storyteller. This is the admin channel; <#%s> is the game channel.", b.settings.StoryTellerId, b.settings.GameChannelId)
	if err := b.assignStorytellerRole(b.settings.GuildId, b.settings.StoryTellerId); err != nil {
		log.Printf("Could not assign the %s role: %v", storytellerRoleName, err)
		reply += fmt.Sprintf(" Warning: could not assign the %q role (%v). Check the role exists and sits below the bot's role.", storytellerRoleName, err)
	}
	b.reply(message, reply)
}

// unregister ends the current game: it removes the game roles from everyone
// and resets the game state so a new game can be registered.
func (b *Bot) unregister(message *discordgo.MessageCreate) {
	guildID := b.settings.GuildId

	removed, roleErr := b.removeGameRoles(guildID)

	b.send(b.settings.GameChannelId, "The game has ended. Thanks for playing!")

	b.settings.GuildId = "UNSET"
	b.settings.AdminChannelId = "UNSET"
	b.settings.GameChannelId = "UNSET"
	b.settings.StoryTellerId = "UNSET"
	b.settings.GameRegistered = false
	b.settings.Players = nil
	b.settings.Characters = nil
	b.settings.Rooms = nil

	log.Printf("The game at %s has been unregistered, %d game roles removed", guildID, removed)

	reply := fmt.Sprintf("Game ended. Removed %d game role(s).", removed)
	if roleErr != nil {
		log.Printf("Problems removing game roles: %v", roleErr)
		reply += fmt.Sprintf(" Warning: some roles could not be removed (%v). Check the %q and %q roles exist and sit below the bot's role.", roleErr, storytellerRoleName, playerRoleName)
	}
	b.reply(message, reply)
}
