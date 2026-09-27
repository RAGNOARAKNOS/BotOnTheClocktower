# Startup and shutdown

[← UML index](README.md)

How the bot starts, connects to Discord, and stops. Covers `main` in [cmd/bot/main.go](../../cmd/bot/main.go), `config.Load` in [internal/config/config.go](../../internal/config/config.go), and `bot.Run` in [internal/bot/bot.go](../../internal/bot/bot.go).

## Activity: startup and shutdown

A missing `.env` file is fine, because the token can come straight from the environment (as in the container). An empty `BOTAPIKEY`, or any other error, ends the process through `log.Fatal`.

```mermaid
flowchart TD
    start((" ")):::initial --> load("config.Load: godotenv.Load")
    load --> d1{" "}
    d1 -->|"[error other than file not found]"| fatal("log.Fatal: print the error and exit")
    d1 -->|"[.env loaded, or no .env file]"| fill("Read BOTAPIKEY")
    fill --> dtok{" "}
    dtok -->|"[empty]"| fatal
    dtok -->|"[set]"| run("bot.Run(cfg.Token)")
    run --> newS("discordgo.New with the bot token")
    newS --> d2{" "}
    d2 -->|"[error]"| fatal
    d2 -->|"[session created]"| intents("Request the intents it uses: servers, members,<br/>voice states, messages, DMs, message content")
    intents --> handlers("Register handlers: Ready, newMessage,<br/>interaction and registerSlashCommands")
    handlers --> open("Open the gateway connection")
    open --> d3{" "}
    d3 -->|"[error, e.g. bad token or a privileged intent not enabled]"| fatal
    d3 -->|"[connected]"| wait("Handle commands until SIGINT or SIGTERM<br/>(see command-dispatch.md)")
    wait --> closeS("Close the session (deferred)")
    closeS --> stopOk(((" "))):::final
    fatal --> stopErr(((" "))):::final

    classDef initial fill:#000,stroke:#666
    classDef final fill:#000,stroke:#666
```

## Sequence: startup and shutdown

```mermaid
sequenceDiagram
    autonumber
    participant OS as Operating system
    participant Main as main
    participant Config as config.Load
    participant Bot as bot.Run
    participant Session as discordgo Session
    participant Gateway as Discord Gateway

    OS->>Main: Start the process
    Main->>+Config: Load()
    Note over Config: godotenv.Load() (a missing .env is ignored)<br/>Token = BOTAPIKEY, an error if it's empty
    Config-->>-Main: Config
    Main->>+Bot: Run(cfg.Token)
    Bot->>Session: discordgo.New("Bot " + token)
    Bot->>Session: Identify.Intents = Guilds, GuildMembers, GuildVoiceStates,<br/>GuildMessages, DirectMessages, MessageContent
    Bot->>Session: AddHandler for Ready, newMessage, interaction, registerSlashCommands
    Bot->>+Session: Open()
    Session->>Gateway: Connect and identify (token, intents)
    Gateway-->>Session: Ready event
    Session-)Bot: Ready handler logs "Bot is ready"
    loop each server the bot is in, and any it joins later
        Gateway-->>Session: GuildCreate event
        Session-)Bot: registerSlashCommands
        Bot->>Gateway: ApplicationCommandBulkOverwrite(app, server, /botc), over REST.<br/>slashCommands builds /botc from the commands table
    end
    Session-->>-Bot: nil
    Note over Bot: Blocks on the stop channel
    Note over Gateway,Bot: Commands arrive as MessageCreate or InteractionCreate events (see command-dispatch.md)
    OS-)Bot: SIGINT or SIGTERM (e.g. Ctrl+C)
    Bot->>Session: Close() (deferred)
    Session->>Gateway: Disconnect
    Bot-->>-Main: nil
    Main-->>OS: Exit
```

---

Last checked against code: 2026-09-27 (85bccfb)
