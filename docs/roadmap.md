# OBS WebSocket Integration Roadmap

## Overview

Integrate the bot with an OBS instance running on the same PC, through the obs-websocket protocol (v5, built into OBS 28+). This lets the bot drive OBS scene changes, source visibility, and audio muting in sync with game phase transitions — for example, cutting to a "Night Phase" overlay at nightfall, or displaying a player roster scene during Town Square.

---

## Background: obs-websocket Protocol

OBS 28+ ships with obs-websocket v5 built in. It runs a WebSocket server (default port 4455) protected by an optional password. It has no TLS, and it listens on every network interface.

The recommended Go client is **`github.com/andreykaipov/goobs`** (v1.10.0 at the time of writing), which provides a typed API over the raw JSON protocol and handles the authentication handshake automatically.

The connection is direct: the bot process dials OBS's WebSocket server itself. Discord only carries the Storyteller's commands (`!botc obs ...`); no OBS traffic goes through Discord's API.

---

## Architecture

### New Package: `internal/obs`

Add a dedicated package that owns the OBS connection lifecycle and exposes a clean interface to the rest of the bot.

```text
cmd/bot/main.go          (existing — builds the OBS client when enabled and passes it to bot.Run)
internal/
├── bot/bot.go           (existing — Bot gains an OBS field; Run starts and closes the client)
├── bot/commands.go      (existing — request gains `after`)
├── config/config.go     (existing — add OBS fields to Config)
└── obs/
    ├── client.go        (connect, disconnect, reconnect logic)
    └── actions.go       (scene switching, source toggle, audio mute helpers)
```

The OBS package never imports `internal/bot`, and `config` and `bot` still don't import each other: `main` builds the `obs.Client` from the config and passes it to `bot.Run`.

### Connection Model

- `obs.New(host, password)` returns a client without connecting. `Start()` runs the connection loop in its own goroutine, and `Close()` stops it and disconnects. `bot.Run` calls `Start` after the Discord session opens, and `Close` when it returns.
- The loop dials with `goobs.New(host, goobs.WithPassword(password), goobs.WithResponseTimeoutDuration(3*time.Second))`, logs the OBS version, then blocks in `client.Listen(...)`. goobs closes `IncomingEvents` when the connection drops, so `Listen` returns. The client marks itself disconnected and dials again after a backoff: 1s, doubling, capped at 30s, and reset once a connection succeeds.
- A failed connection, including the first, is a log line, never fatal. OBS may be started after the bot, and Discord commands keep working without it.
- A command never dials. When the client isn't connected (`Connected()`), its methods return an error at once rather than waiting: after a drop, a goobs request waits for its full response timeout (10 seconds by default, 3 here).
- `obs.Client` serialises its requests with its own mutex, so two OBS commands can't interleave on the connection.

### Keeping OBS calls outside `Bot.mu`

`Bot.mu` is held for the whole command, and OBS calls must never be made while holding it (see "One command at a time" in `.github/copilot-instructions.md`). So an OBS command runs in two steps:

- `request` gains `after func()`: work to run once `Bot.mu` is released. An `obs` handler reads its arguments under the lock (`runCommand` has already checked access), and sets `req.after` to make the OBS call and reply.
- `newMessage` and `interaction` run `after` as soon as `locked` returns, in the same goroutine, recovering from a panic as `locked` does. Because it's the same goroutine, `slashResponder` (which has no locking) is still used by one goroutine at a time.
- `runSlash` doesn't send its "Done." fallback when `after` is set; `after` always replies.
- `after` never reads or writes `b.game`. Anything it needs is copied into the closure under the lock.
- Automatic scene changes (phase 3) that start from a timer run in their own goroutine, and report failures to the admin channel through `b.send`, using the channel ID read under the lock.

`Bot` holds the client behind a small interface (the methods the handlers use), so the harness tests can use a fake OBS. `bot.Run` only sets it when the client isn't nil, so that a nil `*obs.Client` doesn't become a non-nil interface. No client means OBS is disabled, and `obs` commands reply saying so.

---

## Configuration Changes

Add three new environment variables, read in `internal/config/config.go` into `config.Config` as `OBSEnabled`, `OBSHost` and `OBSPassword`:

| Variable | Default | Purpose |
| --- | --- | --- |
| `OBS_HOST` | `localhost:4455` | OBS WebSocket address. From a container in Docker Desktop on the same PC, use `host.docker.internal:4455` |
| `OBS_PASSWORD` | *(empty)* | obs-websocket password. Keep authentication on in OBS and set this |
| `OBS_ENABLED` | `false` | Feature flag, read with `strconv.ParseBool` (a value it can't read is a config error). When false, the bot doesn't connect to OBS |

`OBS_PASSWORD` is a secret, like `BOTAPIKEY`: never log it.

The `.env.example` file (to be created) should document these.

---

## Implementation Phases

### Phase 1 — Foundation

**Goal:** Establish and maintain a connection to OBS. No game logic yet.

- [ ] Add the `github.com/andreykaipov/goobs` dependency (`go get`; v1.10.0 at the time of writing)
- [ ] Create `internal/obs/client.go` (see Connection Model)
  - `New(host, password string) *Client` — returns a client without connecting
  - `Start()` — runs the connection loop in its own goroutine
  - `Close()` — stops the loop and disconnects
  - `Connected() bool` — whether the connection is up, so commands fail fast
  - `Version() (string, error)` — calls `client.General.GetVersion()`
- [ ] Add `OBSEnabled`, `OBSHost` and `OBSPassword` to `config.Config`, read from `OBS_ENABLED`, `OBS_HOST` and `OBS_PASSWORD`
- [ ] In `cmd/bot/main.go`, build the client when `OBSEnabled` is true and pass it to `bot.Run(token, obsClient)`
- [ ] Add `after` to `request` (`commands.go`), run it in `newMessage` (`bot.go`) and `interaction` (`interactions.go`), and skip the "Done." fallback in `runSlash` when it's set (see Keeping OBS calls outside `Bot.mu`)
- [ ] Add an `obs` group with a `ping` subcommand to the `commands` table (`commandtable.go`). It replies with the OBS and obs-websocket versions, or says OBS isn't enabled or isn't connected
- [ ] Add `obs ping` to the README's Commands table
- [ ] Add a UML diagram for `obs ping`, including the `after` step, and a source-map row in `docs/uml/README.md`
- [ ] Add a fake OBS to the harness tests, and test `obs ping` connected, disconnected and disabled

**Deliverable:** `!botc obs ping` returns OBS version string in Discord.

---

### Phase 2 — Scene Control

**Goal:** Let the Storyteller switch OBS scenes from Discord commands.

- [ ] Create `internal/obs/actions.go`
  - `SwitchScene(name string) error` — calls `SetCurrentProgramScene`
  - `GetCurrentScene() (string, error)` — calls `GetCurrentProgramScene`
  - `ListScenes() ([]string, error)` — calls `GetSceneList`
- [ ] Add Discord commands:
  - `!botc obs scene <name>` — switch to named scene
  - `!botc obs scenes` — list available scenes
  - `!botc obs current` — report current scene name
- [ ] Scene names can contain spaces (such as `BOTC - Night`): `obs scene` joins `req.args` with single spaces. Its `/botc` `scene` string option needs a case in `slashWords` (`slash.go`)

Access needs nothing extra: like every command, the `obs` subcommands only run for the Storyteller, in the admin channel, with a game registered.

**Deliverable:** Storyteller can switch OBS scenes from Discord.

---

### Phase 3 — Game Phase Integration

**Goal:** Automate OBS scene/overlay changes when game phase transitions happen.

Define a set of named OBS scenes (configurable, not hardcoded) that the bot switches to automatically:

| Game Event | OBS Scene (default name) |
| --- | --- |
| Game registered | `BOTC - Lobby` |
| Night begins | `BOTC - Night` |
| Day begins (wake) | `BOTC - Day` |
| Nomination open | `BOTC - Nomination` |
| Game over | `BOTC - End` |

- [ ] Add `OBS_SCENE_*` env vars for each scene name (so OBS scene names don't need to match exactly)
- [ ] Hook into the points where the game changes phase:
  - After `register` succeeds (`lifecycle.go`) → switch to lobby scene
  - Night begins → switch to night scene (trigger undecided: the planned `bedtime` command was dropped)
  - Day begins → switch to day scene (undecided whether this is `gather`'s move, in `gather.go`)
  - Nomination opens → switch to nomination scene (no command marks this yet)
  - After `unregister` (`lifecycle.go`) → switch to end scene
- [ ] Every hook runs outside `Bot.mu`: through `req.after` from a command, or in its own goroutine from a timer. Missing OBS is a warning, not an error, and the command still succeeds

**Deliverable:** OBS changes scene automatically with game phase transitions.

---

### Phase 4 — Source & Audio Control

**Goal:** Finer-grained OBS control for presentation polish.

- [ ] Add to `internal/obs/actions.go`:
  - `SetSourceVisible(scene, source string, visible bool) error` — shows/hides a named source
  - `SetInputMuted(inputName string, muted bool) error` — mutes/unmutes an audio input
  - `SetTextContent(sourceName, text string) error` — updates a GDI+ text source
- [ ] New Discord commands:
  - `!botc obs mute <input>` / `!botc obs unmute <input>` — audio control
  - `!botc obs show <source>` / `!botc obs hide <source>` — source visibility
- [ ] Player roster overlay: when the village changes (`village create`, `add` and `remove`, in `village.go`), optionally push the players' names to a named OBS text source (opt-in via `OBS_ROSTER_SOURCE` env var). Names only, never characters: the overlay is on stream

**Deliverable:** Audio and source-level OBS control; optional live player roster overlay.

---

### Phase 5 — Recording & Streaming Helpers

**Goal:** Session management — start/stop recording from Discord.

- [ ] Add to `actions.go`:
  - `StartRecording() error`
  - `StopRecording() error`
  - `GetRecordingStatus() (bool, error)`
- [ ] New commands (Storyteller-only):
  - `!botc obs record start`
  - `!botc obs record stop`
  - `!botc obs record status`

---

## Error Handling Strategy

- OBS WebSocket errors must never crash the Discord bot. Wrap all OBS calls so failures return an error that the command handler logs and replies to Discord with a human-readable message.
- On connection drop, the background loop reconnects with exponential backoff (1s, 2s, 4s, cap 30s). Commands during this window return "OBS not connected, retrying..." at once, without waiting.
- Requests time out after 3 seconds (`goobs.WithResponseTimeoutDuration`), not goobs' default of 10.
- No OBS call is made while holding `Bot.mu` (see Keeping OBS calls outside `Bot.mu`).
- Never log `OBS_PASSWORD`.

---

## Testing Approach

- Command tests in `internal/bot` use the harness (`harness_test.go`) with a fake OBS behind `Bot`'s interface: no OBS needed.
- The connection loop in `internal/obs/client.go` takes its dial function as a parameter, so tests can check reconnects and backoff without OBS. A minimal session interface over goobs covers the rest.
- Integration smoke test: a `cmd/obs-check/main.go` standalone binary that connects to OBS, lists scenes, and exits — useful for local verification without running the full bot.
- Phase 1 `!botc obs ping` command is itself the integration test for wiring.

---

## Dependencies to Add

```shell
go get github.com/andreykaipov/goobs@latest
```

This library handles:

- WebSocket dial
- obs-websocket v5 authentication (SHA-256 challenge/response)
- Typed request/response structs for all OBS API calls
- Automatic versioning negotiation

---

## OBS Setup Requirements (Operator Checklist)

The bot runs on the same PC as OBS.

1. OBS 28 or later (obs-websocket v5 is built in)
2. Enable WebSocket server: **Tools → WebSocket Server Settings → Enable**
3. Keep **Enable Authentication** ticked. Copy the password (**Show Connect Info**) into `OBS_PASSWORD` in `.env`
4. Leave the port at 4455, or set `OBS_HOST` to match
5. Running the bot binary: `OBS_HOST` defaults to `localhost:4455`. Running it in Docker Desktop: set `OBS_HOST=host.docker.internal:4455`
6. Don't add a Windows Firewall rule that lets other machines reach port 4455. The bot only needs it from this PC, and obs-websocket has no TLS
7. Create scenes with the names matching `OBS_SCENE_*` env vars (or use defaults)
8. Set `OBS_ENABLED=true` in `.env`

---

## Milestone Summary

| Phase | Deliverable | Complexity |
| --- | --- | --- |
| 1 | Connect + `!botc obs ping` | Low |
| 2 | Scene switching commands | Low |
| 3 | Automatic phase-driven scene changes | Medium |
| 4 | Source/audio control + roster overlay | Medium |
| 5 | Recording management | Low |
