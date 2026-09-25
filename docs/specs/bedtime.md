# Send players to sleep (`!botc bedtime`)

Status: PLANNED

## Goal

At the end of the DAY phase, move every player into their own "Cottage-XX" voice channel for the NIGHT phase.

## Behaviour

- **Who can run it:** TBD (Storyteller only?)
- **Where from:** TBD
- **Needs a registered game:** yes
- TBD

```text
!botc bedtime
→ TBD
```

## Rules & edge cases

- TBD

## Done when

- [ ] TBD

## Out of scope

- TBD

## Open questions

- How are cottage channels named and numbered (`Cottage-01`? `Cottage 1`?), and does the bot create them or expect them to exist? See [village-management.md](village-management.md).
- Does each player keep the same cottage every night, or are they assigned fresh each time? If fixed, how is the assignment decided?
- What if there are more players than cottages?
- Who counts as a player? This should match [gather.md](gather.md).
- Is the Storyteller moved (e.g. to Storyteller's Corner)?

## Decisions

- (none yet)

## Implementation

Not started. Depends on the same groundwork as [gather.md](gather.md).
