# Secret characters (`!botc character ...`, `!botc whisper`)

Status: DONE (builds and parser tests pass; not yet tested on a live server)

## Goal

Once the village has been created, the Storyteller gives each player a character in secret. Every player gets a character name, and some also get guidance, which may be long. The bot stores the assignments, lets the Storyteller check them, and then sends each player theirs by direct message. The Storyteller can also send a player secret information later in the game.

"Character" here means the game role, to avoid confusion with the Discord roles `BoTC-StoryTeller` and `BoTC-Player`.

## Behaviour

- **Who can run it:** the Storyteller only
- **Where from:** the admin channel only
- **Needs a registered game:** yes, and the player must be in the village

| Command | Effect |
| --- | --- |
| `!botc character assign @player <Character>`, with optional guidance on the following lines (Shift+Enter) | Stores or replaces the player's character and marks it unsent. The reply echoes it back. |
| `!botc character clear @player...` | Removes the stored character. |
| `!botc character list` | Lists every village player with their character (or "none"), whether it has guidance, and whether it has been sent. |
| `!botc character send` | DMs every unsent character. The reply lists who was sent, who failed, and which village players have no character yet. |
| `!botc character send @player...` | Resends to those players, even if theirs was already sent. |
| `!botc whisper @player <text>` | DMs the text straight away. It can span several lines. Nothing is stored. |

A character DM is an embed titled `Your character: <Character>`, with the guidance as its description. A whisper DM is an embed titled "A message from the Storyteller". Both name the server in the footer.

```text
!botc character assign @Bob Fortune Teller
Each night, choose 2 players: you learn if either is a Demon.
There is a good player that registers as a Demon to you.
→ Reply: "Bob will be the Fortune Teller (with 2 line(s) of guidance). Not sent yet; use `!botc character send`."

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
- The character name is whatever follows the mention on the first line. An empty name is refused.
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

## Out of scope

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

## Implementation

- [internal/bot/characters.go](../../internal/bot/characters.go): `character` (dispatch), `characterAssign`, `characterClear`, `characterList`, `characterSend`, `whisper`.
- Parsing: `splitAtMention` finds the single mention on the first line (`<@id>` or `<@!id>`); `parseAssignment` and `parseWhisper` build on it and enforce the length limits. Tests are in `characters_test.go`.
- Delivery: `sendDM` (`UserChannelCreate` + `ChannelMessageSendEmbed`), `dmEmbed` (footer names the server), `dmErrorReason` (turns Discord error 50007 into "they don't accept DMs from this server").
- State: `Settings.Characters` (user ID → `*Character`). `villageCreate`/`villageRemove` delete dropped players' entries, and `unregister` clears them.
- Character names are limited to 200 characters to keep the embed title within Discord's 256-character limit.
