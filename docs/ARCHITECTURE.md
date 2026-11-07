# Architecture

D‑C2 is a single Go binary that connects to a Discord guild and exposes the
local host through one text channel per device.

## Startup sequence

```text
main
 └─ config.Load(embeddedEnv, ".env")
     └─ bot.Run(cfg)
         ├─ discordgo.New("Bot <token>")
         ├─ register message + interaction handlers
         ├─ dg.Open()
         ├─ register missing slash commands
         ├─ resolve OS category → device channel
         ├─ commands.SetChannel(id)
         ├─ logstream.Enable(session, id)
         ├─ media.EnsureFFmpegAvailable (background)
         ├─ send + pin connection banner
         └─ block on SIGINT/SIGTERM
```

## Packages

| Package | Owns | May import |
|---|---|---|
| `main` | configuration wiring | `config`, `bot` |
| `internal/bot` | session, registration, channel provisioning, message shortcuts | `commands`, `config`, `logstream`, `media`, `winsec`, `process` |
| `internal/commands` | slash commands, buttons, modals | `jobs`, `shell`, `media`, `winsec`, `logstream`, `process` |
| `internal/jobs` | async execution, timeouts, output cap, cancellation | `audit` |
| `internal/audit` | JSONL audit trail | stdlib |
| `internal/transfer` | chunked downloads and manifests | stdlib, `discordgo` |
| `internal/preflight` | `--check` validation | `config`, `audit`, `transfer` |
| `internal/version` | build metadata | stdlib |
| `internal/shell` | command execution and output chunking | stdlib, `discordgo` |
| `internal/media` | capture/recording, ffmpeg discovery | stdlib, `discordgo`, `screenshot` |
| `internal/winsec` | firewall/Defender queries and controls | stdlib |
| `internal/logstream` | `io.Writer` that mirrors logs into Discord | stdlib, `discordgo` |
| `internal/process` | self re-exec | stdlib |
| `internal/config` | `.env` parsing and validation | stdlib |

No leaf package imports `bot` or `commands`, so the dependency graph is acyclic.

## Shared state

Two values are process-wide because Discord handlers receive no custom context:

- `bot.channelID` — the device channel; the message handler ignores everything
  else.
- `commands.channelID` — set once via `commands.SetChannel`; the interaction
  router ignores interactions from other channels.

`media` keeps its own mutex-guarded recording state (`videoState`) because only
one segmented recording can run per process. `jobs.Manager` holds the job table;
it retains the last 200 snapshots and calls a notifier when a job finishes.

## Job lifecycle

```text
/cmd | 🏃‍♂️ | quick button
   └─ jobs.Manager.Start
        ├─ reject if running >= DC2_MAX_JOBS
        ├─ context.WithTimeout(DC2_JOB_TIMEOUT)
        ├─ exec.CommandContext → bounded buffer (DC2_MAX_OUTPUT)
        ├─ audit.write(job.start)
        ├─ on finish: status/exit/duration, audit.write(job.finish)
        └─ notifier → post result chunks to the device channel
```

`/jobs list|show|kill` reads and controls the same manager.

## Reconnect handling

`internal/bot/reconnect.go` registers `Ready`, `Resumed`, and `Disconnect`
handlers. The first `Ready` is ignored because the startup path already posts the
banner. Later events re-ensure the OS category and device channel, rebind the
router and log stream, and post a short `🔄 reconnected` notice. Reconciles are
serialized and debounced for 5 seconds.

## Interaction path
```text
Discord event
 ├─ MessageCreate  → bot.handler        → shell.Run / media.* / process.Restart
 ├─ ApplicationCmd → commands.Handle    → handleSlashCommand → handler file
 ├─ Component      → commands.Handle    → handleComponent   → media/shell
 └─ ModalSubmit    → commands.Handle    → handleModal       → shell.Run
```

`internal/commands/definitions.go` is the single source of truth for the slash
commands advertised to Discord; handlers are looked up by name in
`router.go:handleSlashCommand`.

## Platform behavior

- `internal/winsec` and `internal/media` compile on every platform.
- Behaviour is selected at runtime with `runtime.GOOS`; Windows-only commands
  reply with an "unsupported" ephemeral message elsewhere.
- The `ffmpegembed` build tag embeds a Windows `ffmpeg.exe`; the `envembed` tag
  embeds `.env`. Both are optional and off by default.

## Known constraints

- The `github.com/kbinani/screenshot` version pinned in `go.mod` does not build
  for `darwin`; macOS builds fail in that dependency. Windows and Linux build
  cleanly.
- File transfers are capped at Discord's 8 MB attachment limit; there is no
  chunked transfer yet.
- Authorization is channel-based: any guild member who can post in the device
  channel can drive the host.
