# Gather players for the Tribunal (`!botc gather`)

Status: IN PROGRESS (groundwork only)

## Goal

Move all players into Town Square in one command: at the end of the NIGHT phase (from their "Cottage-XX" channels) and when the town gathers for nominations (from wherever they've wandered to).

## Behaviour

- **Who can run it:** the Storyteller only (every command is; see [command-handling.md](command-handling.md))
- **Where from:** the admin channel only
- **Needs a registered game:** yes
- TBD

```text
!botc gather
→ TBD
```

## Rules & edge cases

- TBD

## Done when

- [ ] TBD

## Out of scope

- TBD

## Open questions

- Which channels are players gathered from: cottages, village locations, all voice channels?
- Is the Storyteller moved too?
- A player who isn't in voice at all can't be moved. Skip them silently, or report them?
- What should the reply say: a count, names, failures?
- Are `pmove` / `cmove` (single-player and single-channel moves?) part of this feature, or separate?

## Decisions

- 2026-09-26: The players are the village list from `!botc village` ([village-management.md](village-management.md)).

## Implementation

Groundwork that exists:

- `register` identifies the Storyteller and Town Square.
- `map` records the village channel IDs.
- `village` records the players in `Game.Players`.
- An unused `moveUserToChannel` helper was removed on 2026-09-27. Moving a member is one `GuildMemberMove(guild, user, &channelID)` call; check the room code exists in `Rooms` and handle the error.
- The bot needs **Move Members** and **Connect** on the destination channel.
