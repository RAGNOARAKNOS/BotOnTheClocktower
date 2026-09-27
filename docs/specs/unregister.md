# End a game (`!botc unregister`, alias `!botc end`)

Status: DONE

## Goal

End the current game, clean up the game roles, and let a new game be registered.

## Behaviour

- **Who can run it:** the Storyteller only
- **Where from:** the admin channel only
- **Needs a registered game:** yes; without one it's ignored

Access is checked before the command runs (see [command-handling.md](command-handling.md)).

Steps:

1. Remove `BoTC-StoryTeller` and `BoTC-Player` from **every** member who has them, including roles given out by hand.
2. Post an announcement in Town Square's text chat.
3. Clear the game state.
4. Reply with how many roles were removed, plus a warning listing any failures.

```text
!botc end
→ Game channel: "The game has ended. Thanks for playing!"
→ Reply: "Game ended. Removed 8 game role(s)."
```

## Rules & edge cases

- If some roles can't be removed (missing role, role above the bot's), the game still ends and the reply lists the problems. One failure doesn't stop the rest.
- Handles servers with more than 1000 members by paging through the member list.

## Done when

- [x] Storyteller-only
- [x] Removes both game roles from everyone who has them
- [x] Announces in Town Square, clears state, replies with a count and warnings

## Open questions

- If the Storyteller is unavailable, should moderators (e.g. members with Manage Roles) be able to end the game?
- After a restart no game is registered, so this command is ignored and roles can't be cleaned up by the bot. Should role cleanup work without a registered game?

## Decisions

- 2026-09-25: Storyteller-only; otherwise anyone could end the game.
- 2026-09-25: Strip roles from every member, not just ones the bot assigned, because `BoTC-Player` is given out by hand.
- 2026-09-25: `end` added as an alias.
- 2026-09-26: From the admin channel only, like every command (was: any channel on the registered server). The checks moved to the shared access check.

## Implementation

- `unregister` in [internal/bot/lifecycle.go](../../internal/bot/lifecycle.go); `removeGameRoles` in [internal/bot/roles.go](../../internal/bot/roles.go).
