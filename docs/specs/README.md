# Feature specs

One file per feature, describing what it should do. The project README describes what's *built*; these files describe what's *intended*, including decisions and open questions.

## How to use

- **New feature:** copy [_template.md](_template.md), or ask Claude to draft one from a rough description.
- **Change of mind:** update the spec (Behaviour, Rules, and a dated line under Decisions), then ask Claude to implement the difference.
- **Implementing:** work in slices, e.g. "implement the first two 'Done when' items in gather.md". Tick items off as they're done.
- **Finished:** set Status to DONE, fill in Implementation, and make sure the project README describes it.

Status values: **IDEA** (not yet shaped) → **PLANNED** (agreed, not started) → **IN PROGRESS** → **DONE**.

## Index

### Built

| Spec | Command | Status |
| --- | --- | --- |
| [command-handling.md](command-handling.md) | (all commands) | DONE |
| [ping.md](ping.md) | `!botc ping` | DONE |
| [register.md](register.md) | `!botc register` / `start` | DONE |
| [unregister.md](unregister.md) | `!botc unregister` / `end` | DONE |
| [sitrep.md](sitrep.md) | `!botc sitrep` | DONE |
| [map.md](map.md) | `!botc map` | DONE (with known gaps) |

### Planned and ideas

| Spec | Command | Status |
| --- | --- | --- |
| [gather.md](gather.md) | `!botc gather` | IN PROGRESS |
| [bedtime.md](bedtime.md) | `!botc bedtime` | PLANNED |
| [village-management.md](village-management.md) | TBD | PLANNED |
| [obs-integration.md](obs-integration.md) | `!botc obs ...` | PLANNED (detailed in [roadmap.md](../../roadmap.md)) |
| [vote-tracking.md](vote-tracking.md) | TBD | IDEA |
| [game-visualisation.md](game-visualisation.md) | TBD | IDEA |
