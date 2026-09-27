# Secret characters (`!botc character ...`, `!botc whisper`)

Status: DONE (builds and unit tests pass; not yet tested on a live server)

## Goal

Once the village has been created, the Storyteller gives each player a character in secret. Every player gets a character name and a team (Good or Evil), and some also get guidance, which may be long. The bot stores the assignments, lets the Storyteller check them, and then sends each player theirs by direct message. The Storyteller can also send a player secret information later in the game.

During the game the Storyteller records deaths and revivals, and each dead player's single ghost vote. These are announced in Town Square only when the Storyteller chooses. The Storyteller can see the whole state (the grimoire) in the admin channel.

"Character" here means the game role, to avoid confusion with the Discord roles `BoTC-StoryTeller` and `BoTC-Player`.

## Behaviour

- **Who can run it:** the Storyteller only
- **Where from:** the admin channel only
- **Needs a registered game:** yes, and the player must be in the village

| Command | Effect |
| --- | --- |
| `!botc character assign @player [good\|evil] <Character>`, with optional guidance on the following lines (Shift+Enter) | Stores or replaces the player's character and team (Good if left out), and marks it unsent. A new character starts Alive with its ghost vote unused; reassigning keeps the life state. The reply echoes it back. |
| `!botc character team @player good\|evil` | Changes the team and marks the character unsent, so `send @player` tells them. |
| `!botc character clear @player...` | Removes the stored character. |
| `!botc character kill @player...` | Marks the players Dead. Nothing is posted publicly. |
| `!botc character revive @player...` | Marks the players Alive and gives their ghost vote back. |
| `!botc character ghostvote @player...` | Switches a dead player's ghost vote between used and available. |
| `!botc character announce` | Posts the deaths and revivals since the last announcement in Town Square's text chat. |
| `!botc character list` or `!botc grimoire` | Shows every village player's character, team, Alive/Dead, ghost vote, whether it has been sent and any unannounced change, plus totals. |
| `!botc character send` | DMs every unsent character. The reply lists who was sent, who failed, and which village players have no character yet. |
| `!botc character send @player...` | Resends to those players, even if theirs was already sent. |
| `!botc whisper @player <text>` | DMs the text straight away. It can span several lines. Nothing is stored. |

A character DM is an embed titled `Your character: <Character> (<Team>)`, with the guidance as its description. A whisper DM is an embed titled "A message from the Storyteller". Both name the server in the footer.

```text
!botc character assign @Bob Fortune Teller
Each night, choose 2 players: you learn if either is a Demon.
There is a good player that registers as a Demon to you.
→ Reply: "Bob will be the Fortune Teller (Good, with 2 line(s) of guidance). Not sent yet; use `!botc character send`."

!botc character assign @Carol evil Poisoner
→ Reply: "Carol will be the Poisoner (Evil). Not sent yet; use `!botc character send`."

!botc character kill @Bob
→ Reply: "Now dead: Bob. Not announced yet; use `!botc character announce`."

!botc character announce
→ Game channel: "Bob has died."

!botc grimoire
→ Reply:
Grimoire: Alive 4/5 · Good 3 · Evil 2
1. Alice: Imp (Evil) · Alive · sent
2. Bob: Fortune Teller (Good) · Dead, ghost vote available · sent
3. Dave: none

!botc character send
→ DM to each player: "Your character: ..."
→ Reply: "Sent 5 character(s): Alice, Bob, ... Failed: Carol (cannot send messages to this user). No character yet: Dave"

!botc whisper @Bob
Your number tonight is 1.
→ DM to Bob: "A message from the Storyteller" / "Your number tonight is 1."
→ Reply: "Whispered to Bob."
```

## Rules & edge cases

- `assign` and `whisper` take exactly one @mention. `clear` and `send` can take several.
- The character name is whatever follows the mention (and the optional team word) on the first line. An empty name is refused.
- The first word after the mention is the team only if it is `good` or `evil` (any capitals) and more words follow. A name that is only `good` or `evil` is refused.
- "Evil Twin" starts with a team word, so `assign @player Evil Twin` gives the Evil Twin on the Evil team; `assign @player evil Evil Twin` works too.
- The team given on `assign` always applies. Reassigning an Evil character without a team word makes it Good.
- `team` marks the character unsent, so plain `send` delivers the updated DM as well.
- `kill` on a dead player, or `revive` on a living one, is reported and skipped. `ghostvote` is refused for living players. `revive` gives the ghost vote back.
- The bot remembers each player's Alive/Dead state as of the last announcement. `announce` only posts players whose state differs from that, so a kill followed by a revive before the announcement posts nothing. With nothing to announce, the reply says so and nothing is posted.
- Players without a character can't be killed or revived.
- Guidance and whisper text are kept exactly as typed, including line breaks and Markdown, because the bot parses the raw message rather than splitting it into words.
- Guidance or whisper text longer than 4096 characters (Discord's embed limit) is refused.
- Assigning a character again replaces the previous one and marks it unsent.
- If a DM fails (for example the player doesn't accept DMs from server members), that character stays unsent and the reply names the player. Running `send` again retries only the unsent ones.
- Village players without a character are listed as a warning by `send`, not a blocker.
- A player removed from the village (by `village remove` or `village create`) loses their stored character. `end` clears all characters.

## Done when

- [x] `character assign` stores a name and optional guidance, parsed from the raw message
- [x] `character clear` and `character list`
- [x] `character send` DMs unsent characters and reports sent, failed and missing players
- [x] `character send @player` resends
- [x] `whisper` DMs text straight away
- [x] Characters are dropped with the village and cleared on `end`
- [x] Parser unit tests
- [x] Team on `assign` (Good by default) and `team` to change it; the DM shows the team
- [x] `kill` / `revive` and the ghost vote
- [x] `announce` posts only unannounced changes
- [x] `list` / `grimoire` shows the full state with totals
- [x] Tests for team parsing, pending changes and totals

## Out of scope

- Counting votes or using the ghost vote automatically (see [vote-tracking.md](vote-tracking.md))
- A public alive/dead list for players

- Checking character names against the official character list
- Slash commands and ephemeral replies
- Keeping characters across a bot restart
- Players replying to the Storyteller through the bot

## Open questions

- (none)

## Decisions

- 2026-09-26: Characters are delivered by direct message.
- 2026-09-26: Stage, then send: `assign` only stores; `send` delivers them all together, so mistakes can be fixed first.
- 2026-09-26: The first line holds the character; any following lines are guidance, kept as typed.
- 2026-09-26: Add `!botc whisper @player <text>` for secret information during the game.
- 2026-09-26: Each character has a team, Good or Evil, given as an optional word after the mention on `assign`; Good if left out.
- 2026-09-26: The character DM shows the team.
- 2026-09-26: The team can change during the game (`character team`), for effects that switch teams.
- 2026-09-26: Deaths are tracked with `kill` / `revive`, and each dead player's ghost vote is tracked.
- 2026-09-26: Deaths and revivals are announced in Town Square only when the Storyteller runs `announce`, which posts only what has changed since the last announcement.
- 2026-09-26: `character list` (alias `!botc grimoire`) is the full state view in the admin channel.

## Implementation

- [internal/bot/characters.go](../../internal/bot/characters.go): `character` (dispatch), `characterAssign`, `characterTeam`, `characterSetAlive` (kill and revive), `characterGhostVote`, `characterAnnounce`, `characterClear`, `characterSend`. `characterList` (the grimoire) is in [internal/bot/grimoire.go](../../internal/bot/grimoire.go), and `whisper` and the DM helpers in [internal/bot/whisper.go](../../internal/bot/whisper.go). `!botc grimoire` in `runCommand` calls `characterList`.
- Parsing: `splitAtMention` finds the single mention on the first line (`<@id>` or `<@!id>`); `parseAssignment` and `parseWhisper` build on it and enforce the length limits. `splitTeam` takes off the optional team word, with `teamWordNames` for names that start with one ("Evil Twin"). These are in [internal/bot/parse.go](../../internal/bot/parse.go), with tests in `parse_test.go`.
- Life state: `Alive` and `AnnouncedAlive` (the state at the last `announce`). `Game.PendingLifeChanges` lists the players where they differ; `announce` only marks them announced once the post succeeds.
- Grimoire: `grimoireSummary` (totals), `grimoireLine` (one per player), `chunkLines` (keeps each reply under Discord's 2000-character limit).
- Delivery: `sendDM` (`UserChannelCreate` + `ChannelMessageSendEmbed`), `dmEmbed` (footer names the server), `dmErrorReason` (turns Discord error 50007 into "they don't accept DMs from this server").
- State: `Game.Characters` (user ID → `*Character`). `villageCreate`/`villageRemove` delete dropped players' entries, and `unregister` clears them.
- Character names are limited to 200 characters to keep the embed title within Discord's 256-character limit.
