# Gather players for the Tribunal (`!botc gather`)

Status: IN PROGRESS (groundwork only)

## Goal

Move all players into Town Square in one command: at the end of the NIGHT phase (from their "Cottage-XX" channels) and when the town gathers for nominations (from wherever they've wandered to).

## Behaviour

- **Who can run it:** TBD (Storyteller only?)
- **Where from:** TBD (admin channel only?)
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

- Who counts as a player: everyone with `BoTC-Player`, everyone in a village or cottage channel, or the list from `!botc map`?
- Which channels are players gathered from: cottages, village locations, all voice channels?
- Is the Storyteller moved too?
- A player who isn't in voice at all can't be moved. Skip them silently, or report them?
- What should the reply say: a count, names, failures?
- Are `pmove` / `cmove` (single-player and single-channel moves?) part of this feature, or separate?

## Decisions

- (none yet)

## Implementation

Groundwork that exists:

- `register` identifies the Storyteller and Town Square.
- `map` records the village channel IDs.
- `moveUserToChannel` in [internal/bot/bot.go](../../internal/bot/bot.go) moves one player. It doesn't check that the room code exists and ignores errors; fix that before relying on it.
- The bot needs **Move Members** and **Connect** on the destination channel.
