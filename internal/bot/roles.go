package bot

import (
	"errors"
	"fmt"
	"log"
)

// Discord roles the server must already have. The bot looks them up by exact name.
const (
	// storytellerRoleName is given to the user who registers the game.
	storytellerRoleName = "BoTC-StoryTeller"
	// playerRoleName marks players in the village. Given and taken by the village
	// commands, and removed from everyone on unregister.
	playerRoleName = "BoTC-Player"
)

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

// setPlayerRole gives playerRoleName to everyone in add and takes it from everyone
// in remove (both map user ID to display name). It keeps going past individual failures.
func (b *Bot) setPlayerRole(add, remove map[string]string) error {
	if len(add) == 0 && len(remove) == 0 {
		return nil
	}

	roleID, err := b.findRoleID(b.game.GuildID, playerRoleName)
	if err != nil {
		return err
	}

	var errs []error
	for id, name := range add {
		if err := b.discord.GuildMemberRoleAdd(b.game.GuildID, id, roleID); err != nil {
			errs = append(errs, fmt.Errorf("giving role to %s: %w", name, err))
		}
	}
	for id, name := range remove {
		if err := b.discord.GuildMemberRoleRemove(b.game.GuildID, id, roleID); err != nil {
			errs = append(errs, fmt.Errorf("removing role from %s: %w", name, err))
		}
	}

	return errors.Join(errs...)
}

// replyWithRoleWarning sends reply, adding a warning if the player role couldn't be updated.
func (b *Bot) replyWithRoleWarning(req *request, reply string, roleErr error) {
	if roleErr != nil {
		log.Printf("Problems updating the %s role: %v", playerRoleName, roleErr)
		reply += fmt.Sprintf("\nWarning: the %q role could not be fully updated (%v). Check the role exists and sits below the bot's role.", playerRoleName, roleErr)
	}
	req.reply(reply)
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
