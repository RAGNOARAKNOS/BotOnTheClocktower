# Command dispatch

[← UML index](README.md)

How a `!botc` message or a `/botc` slash command becomes a command. Covers `newMessage`, `locked` and `messageRequest` in [internal/bot/bot.go](../../internal/bot/bot.go); `runCommand`, `resolve`, `findCommand`, `command.help` and `allowed` in [internal/bot/commands.go](../../internal/bot/commands.go), with the `commands` table in [internal/bot/commandtable.go](../../internal/bot/commandtable.go); and `interaction`, `slashCommand`, `modalSubmit`, `runSlash` and `slashResponder` in [internal/bot/interactions.go](../../internal/bot/interactions.go), with `slashWords` and `modalCommand` in [internal/bot/slash.go](../../internal/bot/slash.go).

Both forms build a `request` (the sender, server, channel, command words, raw text, mentioned users, and how to reply) and run the same handler through `runCommand`, so handlers never know which form a command came from.

## Activity: handling a message

discordgo calls `newMessage` in its own goroutine for every message the bot can see. It runs the command through `locked`, which holds `Bot.mu` so only one command runs at a time, and defers a `recover` so a bug in one command doesn't crash the bot and lose the game in memory.

Every command is one entry in the `commands` table ([internal/bot/commandtable.go](../../internal/bot/commandtable.go)): its name and aliases, its handler, who may run it, its usage, and its `/botc` description and options. A group, such as `village` or `character`, holds its subcommands instead of a handler. `resolve` follows the words through the table (a name or alias, then a subcommand's name, ignoring capitals), and `allowed` checks the entry it reaches before the handler runs, so the handlers make no access checks of their own.

```mermaid
flowchart TD
    start((" ")):::initial --> recv("Receive MessageCreate")
    recv --> d1{" "}
    d1 -->|"[author missing, or a bot]"| ignored(((" "))):::final
    d1 -->|"[human author]"| split("Split the content into words<br/>(strings.Fields)")
    split --> d2{" "}
    d2 -->|"[fewer than 2 words, or the first isn't !botc]"| ignored
    d2 -->|"[!botc command]"| lock("locked: lock Bot.mu<br/>Defer recover")
    lock --> build("messageRequest: build a request<br/>(replies threaded to the message)")
    build --> lookup("runCommand: resolve the words through the<br/>commands table (unknown: no flags)")
    lookup --> allowed("allowed(game, command, sender, server, channel)")
    allowed --> dok{" "}
    dok -->|"[not allowed, no refusal]"| ign("Log: Ignoring, no game registered")
    dok -->|"[not allowed, with a refusal]"| refuse("Reply with the refusal")
    dok -->|"[allowed]"| cmd{" "}
    cmd -->|"[unknown command]"| wtf("Reply: Huh? WTF is that command?!")
    cmd -->|"[group, with no subcommand it knows]"| usage("Reply with the group's usage:<br/>each subcommand's")
    cmd -->|"[command]"| args("req.args = the words after its name,<br/>req.usage = its usage")
    args --> run("Run its handler: see the diagram<br/>for each command")
    ign & refuse & wtf & usage & run --> merged{" "}
    merged -->|"[the command panicked]"| rec("Log the stack trace<br/>Reply: Something went wrong")
    merged -->|"[no panic]"| unlock("Unlock Bot.mu")
    rec --> unlock
    unlock --> done(((" "))):::final

    classDef initial fill:#000,stroke:#666
    classDef final fill:#000,stroke:#666
```

Each command is detailed in [game-lifecycle.md](game-lifecycle.md), [village.md](village.md), [characters.md](characters.md) and [gather.md](gather.md).

## Sequence: handling a message

```mermaid
sequenceDiagram
    autonumber
    actor User
    participant Gateway as Discord Gateway
    participant Session as discordgo Session
    participant Bot as Bot.newMessage
    participant Handler as Command handler
    participant REST as Discord REST API

    User->>Gateway: Posts "!botc command args"
    Gateway->>Session: MessageCreate event
    Session-)Bot: newMessage(session, message), in its own goroutine
    alt author is a bot, or not a !botc command
        Note over Bot: Return without replying
    else !botc command
        critical Bot.mu held for the whole command
            Note over Bot: messageRequest(message, words)
            Bot->>+Handler: runCommand(request)
            Note over Handler: Look up the command, then allowed(...)
            alt no game registered, and the command isn't marked beforeGame
                Note over Handler: Ignore, log to the console only
            else game registered, and not the Storyteller in the admin channel (or anywhere, for anyChannel)
                Handler->>REST: Reply "Commands only work for the Storyteller..."
            else allowed
                Handler->>REST: Any Discord calls the command needs
                REST-->>Handler: Results
                Handler->>REST: ChannelMessageSendReply(source channel, reply)
            end
            Handler-->>-Bot: Return
        option the handler panics
            Note over Bot: recover(), print the stack trace
            Bot->>REST: Reply "Something went wrong running that command..."
        end
        opt a reply was sent
            REST-->>User: Reply appears in the channel
        end
    end
```

## Activity: handling a slash command

discordgo calls `interaction` for every interaction. Discord needs an answer within 3 seconds and every answer is ephemeral, which `slashResponder` handles: `acknowledge` shows "thinking…", and each `reply` fills it in or adds a follow-up. Another command can hold `Bot.mu` for longer than 3 seconds while it talks to Discord, so `interaction` acknowledges before waiting for the lock. `whisper` and `character assign`, whose table entries have a `form` (`opensForm`), answer with a form instead, which must be the first response, so they aren't acknowledged early: under the lock they check `allowed` themselves before opening it. The submitted form comes back as a second interaction (see the form sequence below).

```mermaid
flowchart TD
    start((" ")):::initial --> d1{" "}
    d1 -->|"[not in a server, not /botc, or not one of the bot's forms]"| ignored(((" "))):::final
    d1 -->|"[/botc command or bot form]"| early{" "}
    early -->|"[opensForm: whisper or character assign]"| lock("locked: lock Bot.mu<br/>Defer recover")
    early -->|"[any other command, or a submitted form]"| ack("acknowledge: thinking…<br/>before waiting for Bot.mu")
    ack --> dack{" "}
    dack -->|"[Discord couldn't be told]"| ignored
    dack -->|"[acknowledged]"| lock
    lock --> kind{" "}
    kind -->|"[slash command]"| words("slashWords: options to words<br/>and mentioned user IDs")
    words --> form{" "}
    form -->|"[its table entry has a form:<br/>whisper, character assign]"| fok{" "}
    fok -->|"[not allowed]"| frefuse("Reply: the refusal,<br/>or No game registered")
    fok -->|"[player not in the village]"| fnot("Reply: isn't in the village")
    fok -->|"[allowed]"| open("Open the form<br/>(whisperModal or assignModal)")
    form -->|"[any other command]"| resolve("resolveUsers: mentioned IDs to users")
    kind -->|"[form submitted]"| mc("modalCommand: form to words<br/>and raw text for parse.go")
    mc --> resolve
    resolve --> ext("runCommand(request):<br/>allowed, then the handler")
    ext --> d3{" "}
    d3 -->|"[ignored: no game registered]"| nogame("Reply: No game registered")
    d3 -->|"[ran or refused]"| d4{" "}
    nogame --> d4
    d4 -->|"[nothing replied]"| doneReply("Reply: Done.")
    d4 -->|"[replied]"| merged{" "}
    doneReply --> merged
    frefuse & fnot & open --> merged
    merged -->|"[panicked]"| rec("Log the stack trace<br/>Reply: Something went wrong")
    merged -->|"[no panic]"| unlock("Unlock Bot.mu")
    rec --> unlock
    unlock --> done(((" "))):::final

    classDef initial fill:#000,stroke:#666
    classDef final fill:#000,stroke:#666
```

## Sequence: handling a slash command

```mermaid
sequenceDiagram
    autonumber
    actor ST as Storyteller
    participant Gateway as Discord Gateway
    participant Bot as Bot.interaction
    participant Handler as Command handler
    participant REST as Discord REST API

    ST->>Gateway: /botc village add players:@Alice @Bob
    Gateway-)Bot: InteractionCreate, in its own goroutine
    Bot->>REST: InteractionRespond(deferred, ephemeral), before waiting for Bot.mu
    REST-->>ST: thinking… (only the Storyteller sees it)
    critical Bot.mu held for the whole command
        Note over Bot: slashWords: /botc village add, mentions Alice and Bob
        Bot->>+Handler: runCommand(request)
        Handler->>REST: Any Discord calls the command needs
        Handler->>REST: InteractionResponseEdit(reply)
        opt more replies, e.g. a long grimoire
            Handler->>REST: FollowupMessageCreate(reply, ephemeral)
        end
        Handler-->>-Bot: Return
        opt nothing replied
            Bot->>REST: InteractionResponseEdit(Done.)
        end
    end
    REST-->>ST: Reply, visible only to the Storyteller
```

## Sequence: a command with a form

```mermaid
sequenceDiagram
    autonumber
    actor ST as Storyteller
    participant Bot as Bot.interaction
    participant Handler as Bot.whisper
    participant REST as Discord REST API
    actor Player

    ST->>Bot: /botc whisper player:@Alice
    critical Bot.mu
        Note over Bot: allowed(...), and Alice is in the village
        Bot->>REST: InteractionRespond(modal Whisper to Alice)
    end
    REST-->>ST: Form with a message box
    ST->>Bot: Submits the form (a new interaction)
    Bot->>REST: InteractionRespond(deferred, ephemeral), before waiting for Bot.mu
    critical Bot.mu
        Note over Bot: modalCommand: words /botc whisper,<br/>text is Alice's mention then the message
        Bot->>+Handler: runCommand(request), which checks allowed again
        Handler->>REST: DM Alice the message
        REST-->>Player: DM
        Handler->>REST: InteractionResponseEdit(Whispered to Alice.)
        Handler-->>-Bot: Return
    end
```

## Activity: `allowed`

A pure function: it makes no Discord calls, so `commands_test.go` covers every case. It uses two flags from the command's table entry:

- `beforeGame` (`register`, `start`, `ping`): anyone may run it, from any channel, while no game is registered.
- `anyChannel` (`ping`): the Storyteller may run it outside the admin channel.

`runCommand` logs an ignored command and replies with a refusal. For a slash command, an ignored command gets a "No game registered" reply instead, because Discord needs every slash command answered.

```mermaid
flowchart TD
    start((" ")):::initial --> d1{" "}
    d1 -->|"[no game registered]"| d2{" "}
    d2 -->|"[beforeGame]"| ok("Return true")
    d2 -->|"[not beforeGame]"| r1("Return false, no refusal<br/>(ignore)")
    d1 -->|"[game registered]"| d3{" "}
    d3 -->|"[Storyteller, and in the admin channel<br/>of the registered server, or anyChannel]"| ok
    d3 -->|"[anyone else, or anywhere else]"| r2("Return false, refusal:<br/>Commands only work for the Storyteller,<br/>in the admin channel")
    r1 --> refused(((" "))):::final
    r2 --> refused
    ok --> allowedEnd(((" "))):::final

    classDef initial fill:#000,stroke:#666
    classDef final fill:#000,stroke:#666
```

---

Last checked against code: 2026-09-27 (85bccfb)
