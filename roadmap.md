# OBS WebSocket Integration Roadmap

## Overview

Integrate the bot with a locally-running OBS instance via the obs-websocket protocol (v5, built into OBS 28+). This lets the bot drive OBS scene changes, source visibility, and audio muting in sync with game phase transitions — for example, cutting to a "Night Phase" overlay when players are sent to sleep, or displaying a player roster scene during Town Square.

---

## Background: obs-websocket Protocol

OBS 28+ ships with obs-websocket v5 built in. It exposes a local WebSocket server (default `ws://localhost:4455`) protected by an optional password.

The recommended Go client is **`github.com/andreykaipov/goobs`**, which provides a typed API over the raw JSON protocol and handles the authentication handshake automatically.

---

## Architecture

### New Package: `internal/obs`

Add a dedicated package that owns the OBS connection lifecycle and exposes a clean interface to the rest of the bot.

```
internal/
├── bot/bot.go          (existing — add OBS client field to Bot struct)
├── config/config.go    (existing — add OBS config fields to Settings)
└── obs/
    ├── client.go       (connect, disconnect, reconnect logic)
    └── actions.go      (scene switching, source toggle, audio mute helpers)
```

The `Bot` struct gains an `*obs.Client` field. Commands call methods on it; the OBS package never imports `internal/bot`.

### Connection Model

The OBS client connects at bot startup (after Discord session opens) and reconnects automatically on drop. OBS is a local dependency, not a remote service, so the bot should tolerate OBS not being present and degrade gracefully — Discord commands still work, OBS commands log a warning and no-op.

---

## Configuration Changes

Add three new environment variables (loaded in `internal/config/config.go`):

| Variable | Default | Purpose |
|---|---|---|
| `OBS_HOST` | `localhost:4455` | OBS WebSocket address |
| `OBS_PASSWORD` | *(empty)* | obs-websocket password (optional) |
| `OBS_ENABLED` | `false` | Feature flag — skips connection if OBS not in use |

The `.env.example` file (to be created) should document these.

---

## Implementation Phases

### Phase 1 — Foundation

**Goal:** Establish and maintain a connection to OBS. No game logic yet.

- [ ] Add `github.com/andreykaipov/goobs` dependency (`go get`)
- [ ] Create `internal/obs/client.go`
  - `Connect(host, password string) (*Client, error)` — dials obs-websocket, returns typed client
  - Wrap `goobs.New()` and store the session
  - `Close()` — graceful teardown
  - `IsConnected() bool` — health check for degraded-mode guard
- [ ] Update `internal/config/config.go` to read `OBS_HOST`, `OBS_PASSWORD`, `OBS_ENABLED`
- [ ] Update `Settings` struct with `OBSHost`, `OBSPassword`, `OBSEnabled` fields
- [ ] In `internal/bot/bot.go`, add `obs *obs.Client` to `Bot` struct
- [ ] In `Bot.Run()`, conditionally connect to OBS after Discord session opens; log but don't fatal on failure
- [ ] Add `!botc obs ping` command — queries OBS version and prints it to Discord, confirms end-to-end wiring

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
- [ ] Gate all OBS commands behind Storyteller check (same pattern as `register`)

**Deliverable:** Storyteller can switch OBS scenes from Discord.

---

### Phase 3 — Game Phase Integration

**Goal:** Automate OBS scene/overlay changes when game phase transitions happen.

Define a set of named OBS scenes (configurable, not hardcoded) that the bot switches to automatically:

| Game Event | OBS Scene (default name) |
|---|---|
| Game registered | `BOTC - Lobby` |
| Night begins (bedtime) | `BOTC - Night` |
| Day begins (wake) | `BOTC - Day` |
| Nomination open | `BOTC - Nomination` |
| Game over | `BOTC - End` |

- [ ] Add `OBS_SCENE_*` env vars for each scene name (so OBS scene names don't need to match exactly)
- [ ] Hook into existing phase transition points in `bot.go`:
  - After `register()` succeeds → switch to lobby scene
  - When bedtime command fires → switch to night scene
  - When wake command fires → switch to day scene
- [ ] All hooks wrapped in `IsConnected()` guard — missing OBS is a warning, not an error

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
- [ ] Player roster overlay: when `mapPlayers()` completes, optionally push player names to a named OBS text source (opt-in via `OBS_ROSTER_SOURCE` env var)

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
- On connection drop, attempt reconnect with exponential backoff (1s, 2s, 4s, cap 30s) in a background goroutine. Commands during this window return "OBS not connected, retrying...".
- `IsConnected()` is checked at the top of every OBS command handler.

---

## Testing Approach

- Unit tests for `internal/obs/client.go` can use a mock `goobs` interface (define a minimal `OBSSession` interface the real client satisfies).
- Integration smoke test: a `cmd/obs-check/main.go` standalone binary that connects to OBS, lists scenes, and exits — useful for local verification without running the full bot.
- Phase 1 `!botc obs ping` command is itself the integration test for wiring.

---

## Dependencies to Add

```
go get github.com/andreykaipov/goobs@latest
```

This library handles:
- WebSocket dial and TLS
- obs-websocket v5 authentication (SHA-256 challenge/response)
- Typed request/response structs for all OBS API calls
- Automatic versioning negotiation

---

## OBS Setup Requirements (Operator Checklist)

1. OBS 28 or later (obs-websocket v5 is built in)
2. Enable WebSocket server: **Tools → WebSocket Server Settings → Enable**
3. Note the port (default 4455) and set a password if desired
4. Create scenes with the names matching `OBS_SCENE_*` env vars (or use defaults)
5. Set `OBS_ENABLED=true` in `.env`

---

## Milestone Summary

| Phase | Deliverable | Complexity |
|---|---|---|
| 1 | Connect + `!botc obs ping` | Low |
| 2 | Scene switching commands | Low |
| 3 | Automatic phase-driven scene changes | Medium |
| 4 | Source/audio control + roster overlay | Medium |
| 5 | Recording management | Low |
