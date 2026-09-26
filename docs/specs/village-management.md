# Village creation & management (`!botc village ...`)

Status: DONE (builds; not yet tested on a live server)

## Goal

The act of creating a village is essentially creating a list of players who will be participating in the game. The plan is to have a command that lists every player currently in the voice channel known as Town Square and add them to the player list, obviously omitting the storyteller. There will also need to be a function to add or remove players in an ad hoc manner to or from that list

## Behaviour

- **Who can run it:** The Storyteller
- **Where from:** The admin channel
- **Needs a registered game:** Yes

Subcommands:

- `create`: builds the village from everyone in Town Square voice, leaving out the Storyteller and bots. It **replaces** any existing list. New players get `BoTC-Player`, and anyone dropped from the list loses it.
- `add @user...`: adds each mentioned user and gives them `BoTC-Player`.
- `remove @user...`: removes each mentioned user and takes `BoTC-Player` away.
- `list`: shows the current players.

```text
!botc village create
→ Reply: "Village created with 3 player(s): Alice, Bob, Carol"

!botc village add @Dave @Erin
→ Reply: "Added 2 player(s): Dave, Erin"

!botc village remove @Bob
→ Reply: "Removed 1 player(s): Bob"

!botc village list
→ Reply: "The village has 4 player(s): 1. Alice 2. Carol 3. Dave 4. Erin"
```

## Rules & edge cases

- `create` reads Town Square's voice state (the game channel found by `register`), so `map` doesn't need to run first.
- If Town Square is empty, `create` still succeeds and says the village is empty.
- Players are named by @mention only. `add`/`remove` without any mentions replies with usage help.
- `add` skips (and reports) bots, the Storyteller, and anyone already in the village. `remove` skips (and reports) anyone not in the village.
- If `BoTC-Player` can't be given or taken away (missing role, role above the bot's), the list is still updated and the reply includes a warning.
- `end` clears the village and strips `BoTC-Player` from everyone.

## Done when

- [x] A game has been registered by a storyteller
- [x] Storyteller only, from the admin channel only
- [x] `create` builds the list from Town Square, replacing any existing list
- [x] `add` / `remove` take one or more @mentions
- [x] `list` shows the players
- [x] `BoTC-Player` follows the list, and failures produce a warning
- [x] `map` no longer wipes the player list

## Out of scope

- Adding or removing players by typed name
- Keeping the village across a bot restart

## Open questions

- (none)

## Decisions

- 2026-09-26: Commands are subcommands: `!botc village create | add | remove | list`.
- 2026-09-26: The bot gives and takes away `BoTC-Player` as players join and leave the village.
- 2026-09-26: Running `create` again replaces the list, and anyone dropped loses the role.
- 2026-09-26: Players are named by @mention only; one command can take several mentions.

## Implementation

- `village`, `villageCreate`, `villageAdd`, `villageRemove`, `villageList` in [internal/bot/bot.go](../../internal/bot/bot.go).
- Access is checked before any command runs, by `commandAllowed` (see [command-handling.md](command-handling.md)).
- Helpers: `setPlayerRole` (looks the role up once, keeps going past failures), `replyWithRoleWarning`, `lookupMember` (voice state → state cache → Discord API), `memberDisplayName`, `sortedNames`.
- Players are stored in `Settings.Players` (user ID → display name).
- `create` gives `BoTC-Player` to everyone in the new list, not just newcomers, so running it again fixes any earlier role failure.
- The old `mapPlayers` (console only) has been removed.
