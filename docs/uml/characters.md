# Characters, deaths and whispers

[← UML index](README.md)

The Storyteller gives each village player a secret character and team, sends them by DM, and tracks deaths and ghost votes. Covers [internal/bot/characters.go](../../internal/bot/characters.go), [internal/bot/grimoire.go](../../internal/bot/grimoire.go), [internal/bot/whisper.go](../../internal/bot/whisper.go) and [internal/bot/parse.go](../../internal/bot/parse.go). Spec: [characters.md](../specs/characters.md).

Like every command, `character`, `grimoire` and `whisper` only run for the Storyteller in the admin channel; [`allowed`](command-dispatch.md#activity-allowed) checks that before they're called. A character (`Game.Characters`, user ID → `*Character`) holds these fields:

| Field | Meaning |
| --- | --- |
| `Name`, `Guidance`, `Team` | What the player is told. `Team` is Good or Evil. |
| `Sent` | Whether the player has been sent the current version. `assign` and `team` set it to false, and `send` sets it to true. |
| `Alive` | Changed by `kill` and `revive`. Not public until `announce`. |
| `AnnouncedAlive` | `Alive` as of the last `announce`. Where it differs from `Alive`, a change is waiting to be announced. |
| `GhostVoteUsed` | Whether a dead player has used their ghost vote. `revive` resets it. |

## Activity: `character assign`

`assign` parses the raw message text rather than splitting it into words, so multi-line guidance and Markdown survive exactly as typed.

```mermaid
flowchart TD
    start((" ")):::initial --> mention("splitAtMention: normalise line endings,<br/>find the mention on the first line")
    mention --> d1{" "}
    d1 -->|"[no mention, or more than one]"| bad("Reply: Could not read that, with the usage")
    d1 -->|"[exactly one mention]"| d2{" "}
    d2 -->|"[nothing after the mention]"| bad
    d2 -->|"[text follows]"| tw{" "}
    tw -->|"[whole name is in teamWordNames, e.g. Evil Twin]"| named("Team from teamWordNames,<br/>keep the whole name")
    tw -->|"[otherwise]"| first{" "}
    first -->|"[first word is not good or evil]"| good("Team = Good, name unchanged")
    first -->|"[first word is good or evil]"| rest{" "}
    rest -->|"[nothing after the team word]"| bad
    rest -->|"[name follows]"| split("Team = that word, name = the rest")
    named & good & split --> lim{" "}
    lim -->|"[name over 200, or guidance<br/>over 4096 characters]"| bad
    lim -->|"[within limits]"| vil{" "}
    vil -->|"[player not in the village]"| notv("Reply: isn't in the village,<br/>use village add first")
    vil -->|"[in the village]"| prev{" "}
    prev -->|"[already had a character]"| keep("Carry over Alive, AnnouncedAlive<br/>and GhostVoteUsed")
    prev -->|"[first character]"| fresh("Alive = AnnouncedAlive = true")
    keep & fresh --> store("Store the character with Sent = false")
    store --> reply("Reply: X will be the Y (team).<br/>Not sent yet.")
    bad --> done(((" "))):::final
    notv --> done
    reply --> done

    classDef initial fill:#000,stroke:#666
    classDef final fill:#000,stroke:#666
```

## Activity: `character kill` / `revive`

Only `Alive` changes. Nothing is posted publicly until `announce`.

```mermaid
flowchart TD
    start((" ")):::initial --> d0{" "}
    d0 -->|"[no mentions]"| usage("Reply: Mention the players, with the usage")
    d0 -->|"[one or more mentions]"| next{" "}
    next -->|"[next mention]"| has{" "}
    has -->|"[no character]"| skip("Record as skipped, with the reason")
    has -->|"[has a character]"| same{" "}
    same -->|"[already dead / alive]"| skip
    same -->|"[state changes]"| set("Set Alive")
    set --> rv{" "}
    rv -->|"[revive]"| ghost("GhostVoteUsed = false")
    rv -->|"[kill]"| next
    ghost --> next
    skip --> next
    next -->|"[no more mentions]"| pend("Game.PendingLifeChanges: players whose<br/>Alive differs from AnnouncedAlive")
    pend --> reply("Reply: Now dead / alive: N player(s), who was skipped,<br/>and how many changes are waiting to be announced")
    usage --> done(((" "))):::final
    reply --> done

    classDef initial fill:#000,stroke:#666
    classDef final fill:#000,stroke:#666
```

## Activity: `character ghostvote`

```mermaid
flowchart TD
    start((" ")):::initial --> d0{" "}
    d0 -->|"[no mentions]"| usage("Reply: Mention the dead players")
    d0 -->|"[one or more mentions]"| next{" "}
    next -->|"[next mention: no character]"| skip("Record as skipped, with the reason")
    next -->|"[next mention: alive]"| skip
    next -->|"[next mention: dead]"| toggle("Switch GhostVoteUsed<br/>between used and available")
    skip --> next
    toggle --> next
    next -->|"[no more mentions]"| reply("Reply: each player's ghost vote,<br/>and who was skipped")
    usage --> done(((" "))):::final
    reply --> done

    classDef initial fill:#000,stroke:#666
    classDef final fill:#000,stroke:#666
```

## Activity: `character team`

Changing a team marks the character unsent, so the next `character send` tells the player.

```mermaid
flowchart TD
    start((" ")):::initial --> d0{" "}
    d0 -->|"[no good/evil word, or no mentions]"| usage("Reply: Mention the players and give the team")
    d0 -->|"[team and mentions given]"| next{" "}
    next -->|"[next mention: no character]"| skip("Record as skipped, with the reason")
    next -->|"[next mention: already on that team]"| skip
    next -->|"[next mention: on the other team]"| set("Team = new team<br/>Sent = false")
    skip --> next
    set --> next
    next -->|"[no more mentions]"| reply("Reply: Now Good / Evil: N player(s).<br/>Use character send to tell them.")
    usage --> done(((" "))):::final
    reply --> done

    classDef initial fill:#000,stroke:#666
    classDef final fill:#000,stroke:#666
```

## Activity: `character announce`

```mermaid
flowchart TD
    start((" ")):::initial --> pend("Game.PendingLifeChanges")
    pend --> d1{" "}
    d1 -->|"[nothing pending]"| none("Reply: Nothing to announce")
    d1 -->|"[deaths or revivals]"| build("Build the lines, sorted by name:<br/>X has died. / Y has returned to life.")
    build --> post("Post the lines in Town Square")
    post --> d2{" "}
    d2 -->|"[post failed]"| fail("Reply: Could not post,<br/>nothing was marked as announced")
    d2 -->|"[posted]"| mark("For every character:<br/>AnnouncedAlive = Alive")
    mark --> ok("Reply: Announced, with the text")
    none --> done(((" "))):::final
    fail --> done
    ok --> done

    classDef initial fill:#000,stroke:#666
    classDef final fill:#000,stroke:#666
```

## Sequence: `character announce`

```mermaid
sequenceDiagram
    autonumber
    actor ST as Storyteller
    participant Bot as Bot.characterAnnounce
    participant REST as Discord REST API
    actor Town as Town Square chat

    ST->>Bot: !botc character announce
    Note over Bot: Game.PendingLifeChanges()
    alt nothing pending
        Bot->>REST: Reply "Nothing to announce..."
    else deaths or revivals pending
        Bot->>REST: ChannelMessageSend(Town Square, "X has died." ...)
        alt post failed
            REST-->>Bot: Error
            Bot->>REST: Reply "Could not post... Nothing was marked as announced."
        else posted
            REST-->>Town: Deaths and revivals
            Note over Bot: AnnouncedAlive = Alive for every character
            Bot->>REST: Reply "Announced in Town Square: ..."
        end
    end
    REST-->>ST: Reply in the admin channel
```

## Sequence: `character send`

With no mentions, `send` DMs every character not yet sent. With mentions, it resends to just those players. A failed DM leaves the character unsent, so running `send` again retries it.

```mermaid
sequenceDiagram
    autonumber
    actor ST as Storyteller
    participant Bot as Bot.characterSend
    participant State as discordgo State cache
    participant REST as Discord REST API
    actor P as Player

    ST->>Bot: !botc character send (optionally @player...)
    alt players mentioned
        Note over Bot: Targets = the mentioned players with a character<br/>(mentionedCharacters: the others are skipped, with no character)
    else no mentions
        Note over Bot: Targets = village players with an unsent character, by name
    end
    loop each target
        Bot->>State: Guild(GuildID), for the embed footer (dmEmbed)
        Bot->>REST: UserChannelCreate(player), in sendDM
        REST-->>Bot: DM channel
        Bot->>REST: ChannelMessageSendEmbed(DM, "Your character: Name (Team)", guidance)
        alt delivered
            REST-->>P: Character DM
            Note over Bot: Game.MarkSent: Sent = true
        else either call failed
            REST-->>Bot: Error
            Note over Bot: Still unsent. dmErrorReason explains,<br/>e.g. they don't accept DMs from this server.
        end
    end
    Bot->>REST: Reply "Sent N character(s)...", then Failed, Skipped,<br/>and Village players with no character yet
    REST-->>ST: Reply in the admin channel
```

## Sequence: `grimoire` / `character list`

```mermaid
sequenceDiagram
    autonumber
    actor ST as Storyteller
    participant Bot as Bot.characterList
    participant REST as Discord REST API

    ST->>Bot: !botc grimoire (or character list)
    alt village empty
        Bot->>REST: Reply "The village is empty..."
    else players in the village
        Note over Bot: First line: grimoireSummary, e.g.<br/>"Alive 6/8 · Good 5 · Evil 3 · 1 change(s) not yet announced"
        Note over Bot: Then one grimoireLine per player, by name, e.g.<br/>"Monk (Good) · Dead, ghost vote used · sent"
        Note over Bot: chunkLines splits the lines into messages<br/>of at most 2000 characters
        loop each chunk
            Bot->>REST: ChannelMessageSendReply(admin channel, chunk)
        end
    end
    REST-->>ST: The grimoire, in the admin channel
```

## Sequence: `whisper`

`whisper` DMs a village player straight away. Nothing is stored.

```mermaid
sequenceDiagram
    autonumber
    actor ST as Storyteller
    participant Bot as Bot.whisper
    participant REST as Discord REST API
    actor P as Player

    ST->>Bot: !botc whisper @player text (can span several lines)
    alt parseWhisper fails: no mention, more than one, no text, or over 4096 characters
        Bot->>REST: Reply "Could not read that..."
    else the player isn't in the village
        Bot->>REST: Reply "isn't in the village..."
    else ok
        Bot->>REST: UserChannelCreate(player), then<br/>ChannelMessageSendEmbed(DM, "A message from the Storyteller", text)
        alt delivered
            REST-->>P: Whisper DM
            Bot->>REST: Reply "Whispered to Name."
        else failed
            REST-->>Bot: Error
            Bot->>REST: Reply "Could not whisper to Name (reason)."
        end
    end
    REST-->>ST: Reply in the admin channel
```

---

Last checked against code: 2026-09-27 (f7831d7)
