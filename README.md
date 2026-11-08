# D‑C2 🛰️

Discord‑based command‑and‑control (C2) **client** for educational and defensive research in environments you own or are explicitly authorized to test.

> This project is provided **only** for lawful, defensive, and educational use (for example: blue‑team labs, adversary simulation in your own environment, and red‑team exercises with written approval).  
> **Do not** deploy it on systems or networks you do not own or explicitly control.

---

## 1. Project Overview

**D‑C2** turns a Discord server into a central command console for endpoints running this Go client.

When the client starts:

- It connects to a Discord bot using your token.
- It creates (or reuses) an OS‑specific category:
  - `windows`, `linux`, `mac`, or `other`.
- It ensures a **per‑device channel** named from the hostname and username.
- It posts a pinned **“New device connected”** banner with:
  - Session label (`hostname | username | ip`)
  - IP address
  - Username
  - Hostname
  - OS + architecture
  - Current working directory
  - Windows firewall / Defender status (on Windows)

From that device channel you can:

- Run shell commands.
- Inspect processes and system info.
- Browse, upload, and download files.
- Capture screenshots, camera snapshots, microphone audio, and segmented videos.
- Stream the client’s Go logs into the channel.
- Control Windows Defender and firewall (Windows only).

**Target audience**

- Security engineers, red‑teamers, and blue‑teamers building lab environments.
- Researchers exploring command‑and‑control patterns.
- Developers interested in Discord‑based automation and remote orchestration.

**Technical scope**

- Language: Go 1.21+
- Protocol: Discord HTTP + WebSocket via [github.com/bwmarrin/discordgo](https://github.com/bwmarrin/discordgo)
- Platforms:
  - Windows: full feature set (filesystem, commands, media, Windows security).
  - Linux/macOS: command, filesystem, process, and basic system features.
- Media:
  - Screenshots via [github.com/kbinani/screenshot](https://github.com/kbinani/screenshot).
  - Audio/video capture via ffmpeg (system or bundled/embedded binary).

---

## 1.1 Quick start

1. **Configure** — `cp .env.example .env && chmod 600 .env`, then set
   `DISCORD_BOT_TOKEN`, `DISCORD_GUILD_ID`, and `DISCORD_ALLOWED_USERS`.
2. **Build** — `scripts/build.sh linux` (or `windows`, `darwin`, `all`).
3. **Check** — `./dist/dc2-linux-amd64 --check` validates the token, guild,
   permissions, audit path, and ffmpeg before connecting.
4. **Run** — `./dist/dc2-linux-amd64` from the directory containing `.env`.
5. **Control** — in the new `#hostname-username` channel use `/help`, `/cmd`,
   `/media`, `/fs`, `/sys`, `/win`, `/dev`.
6. **Deploy** — `sudo scripts/install-linux.sh dist/dc2-linux-amd64 .env`
   (Windows: `scripts/install-windows.ps1`).

Full paths: [server setup](docs/SERVER.md) → [build](docs/BUILD.md) →
[install](docs/INSTALL.md) → [control](docs/CONTROL.md). Lab:
[docs/LAB.md](docs/LAB.md).

---

## 2. Project Structure Map

Detailed guides: [Configuration](docs/CONFIGURATION.md), [Control reference](docs/CONTROL.md), [Server setup](docs/SERVER.md), [Build artifacts](docs/BUILD.md), [Release](docs/RELEASE.md), [Install on a target](docs/INSTALL.md), [Stager](docs/STAGER.md), [OPSEC](docs/OPSEC.md), [Purple-team validation](docs/PURPLE.md), [Two-container lab](docs/LAB.md), [Security](docs/SECURITY.md), [Skills used](docs/SKILLS.md), [Architecture](docs/ARCHITECTURE.md).

Repository root:

```text
discord-c2/
├── main.go                     # entrypoint: load config, start bot
├── env_embed.go                # //go:embed .env (build tag: envembed)
├── env_embed_stub.go           # no-op stub for default builds
├── go.mod
├── go.sum
├── internal/
│   ├── bot/
│   │   ├── bot.go              # session startup, registration, banner
│   │   ├── category.go         # OS category + per-device channel helpers
│   │   ├── handler.go          # message/emoji shortcuts, quick help
│   │   ├── reconnect.go        # Ready/Resumed watchdog, channel rebind
│   │   └── process.go          # restart wrapper
│   ├── commands/
│   │   ├── router.go           # interaction router (slash, buttons, modals)
│   │   ├── definitions.go      # slash-command definitions (data only)
│   │   ├── help.go             # /help
│   │   ├── cmd.go              # /cmd
│   │   ├── fs.go               # /fs cwd /fs ls /fs download /fs upload
│   │   ├── log.go              # /sys log
│   │   ├── media.go            # /media screenshot /media camera /media mic /media record /media recordstop
│   │   ├── system.go           # /sys ps /sys pidinfo
│   │   ├── windows.go          # /win sec /win firewall /win defender /win exclude
│   │   ├── jobs.go             # /jobs list|show|kill
│   │   ├── info.go             # /dev info
│   │   ├── control.go          # /dev restart
│   │   └── help_test.go        # unit test for the help responder
│   ├── config/config.go        # .env loading + Config type
│   ├── audit/audit.go          # JSONL audit trail
│   ├── approvals/approvals.go  # two-person approval requests
│   ├── jobs/jobs.go            # async execution, timeouts, cancellation
│   ├── jobs/render.go          # job ack/result formatting
│   ├── version/version.go      # build metadata (-ldflags)
│   ├── logstream/logstream.go  # io.Writer that mirrors logs into Discord
│   ├── media/
│   │   ├── media.go            # capture/recording + ffmpeg discovery
│   │   ├── ffmpeg_embed_windows.go
│   │   └── ffmpeg_embed_stub.go
│   ├── process/process.go      # self re-exec (restart)
│   ├── shell/shell.go          # command execution + output chunking
│   ├── transfer/transfer.go    # chunked downloads + manifest
│   ├── preflight/preflight.go  # --check validation
│   └── winsec/winsec.go        # firewall/Defender status and controls
├── detections/
│   ├── yara/discord_c2.yar     # token/webhook/client strings
│   ├── sigma/                  # Discord API + shell-spawn rules
│   └── IOCS.md                 # indicators, TTP map, hunting queries
├── scripts/
│   ├── build.sh                # cross-compile artifacts + SHA256SUMS
│   ├── install-linux.sh        # systemd install
│   └── install-windows.ps1     # ProgramData install + logon task
├── lab/
│   ├── run.sh                  # build and run target + attacker containers
│   └── attacker.sh             # webhook-based operator simulator
├── deploy/
│   ├── Dockerfile.target
│   └── Dockerfile.attacker
├── docs/
│   ├── ARCHITECTURE.md         # package boundaries and data flow
│   ├── SERVER.md               # Discord control plane setup
│   ├── BUILD.md                # artifact generation and tags
│   ├── RELEASE.md              # tagging, workflow, verification, rollback
│   ├── INSTALL.md              # Linux/Windows/macOS deployment
│   ├── STAGER.md               # bootstrap installer patterns
│   ├── CONFIGURATION.md        # env vars, precedence, secret storage
│   ├── CONTROL.md              # commands, shortcuts, access model
│   ├── LAB.md                  # two-container lab
│   ├── SECURITY.md             # audit results and hardening
│   ├── OPSEC.md                # operator and runtime operational security
│   ├── PURPLE.md               # detection validation plan
│   └── SKILLS.md               # audit skills and sources
├── .gitignore
└── README.md
```

### 2.1 Package responsibilities

- `main` — parses configuration and calls `bot.Run`. No Discord logic.
- `internal/config` — applies `key=value` pairs from an optional embedded
  payload and `.env`, then validates `DISCORD_BOT_TOKEN` and `DISCORD_GUILD_ID`.
- `internal/bot` — owns the Discord session, command registration, channel
  provisioning, the connection banner, and the message shortcut handler.
- `internal/commands` — one file per command family plus a router. Handlers only
  format Discord responses and delegate work to leaf packages.
- `internal/jobs` — asynchronous execution: timeouts, concurrency limit, bounded
  output, cancellation, persistence, and result formatting.
- `internal/approvals` — two-person approval requests for commands.
- `internal/audit` — JSON-lines audit trail for command lifecycle events.
- `internal/version` — build metadata injected with `-ldflags`.
- `internal/shell` — synchronous runner for fixed info commands; chunks output
  into Discord-sized messages.
- `internal/logstream` — `io.Writer` wrapper used as `log.SetOutput`.
- `internal/media` — screenshots, camera, microphone, segmented video, and
  ffmpeg discovery (PATH → bundled → embedded → package-manager install).
- `internal/winsec` — Windows firewall/Defender helpers; safe on all platforms
  (returns `unsupported`).
- `internal/process` — re-execs the binary for `/dev restart` and `🔁`.

### 2.2 Dependency direction

```text
main → bot → commands → { shell, media, winsec, logstream, process }
                 └──────→ config (startup only)
```

Leaf packages never import `bot` or `commands`, so the graph stays acyclic.
`winsec` and `media` compile on every platform; platform behavior is selected at
runtime through `runtime.GOOS`.

---

## 3. Installation Requirements

### 3.1 Supported platforms

- Windows 10/11 (x64) – full capabilities.
- Linux (modern distributions; x64) – non‑media features fully supported; media features depend on ffmpeg and display/audio stack.
- macOS (x64/ARM64) – builds as of the 2025-06 `screenshot` dependency; basic command/filesystem/process features. Media features remain Windows‑focused.

### 3.2 Software prerequisites

- **Go**: 1.21 or newer (for building from source).
- **Discord**:
  - A Discord account.
  - A Discord application with a bot user and token.
  - A Discord server (guild) where the bot has:
    - `Manage Channels`
    - `Read Messages/View Channels`
    - `Send Messages`
    - `Attach Files`
    - `Add Reactions`
- **ffmpeg** (for media actions):
  - Windows:
    - Either ffmpeg installed and available on `PATH`, or
    - A bundled binary in `ffmpeg/windows/ffmpeg.exe`, or
    - An embedded ffmpeg when building with `-tags ffmpegembed`.
  - Linux:
    - ffmpeg installed via the system package manager **or**
    - A bundled binary in `ffmpeg/linux/ffmpeg`.

### 3.3 External Go dependencies

Declared in [`go.mod`](./go.mod):

- `github.com/bwmarrin/discordgo` – Discord client library.
- `github.com/kbinani/screenshot` – cross‑platform screenshot capture.
- Transitive dependencies for window system integration and crypto/syscalls.

### 3.4 Configuration via `.env`

Create a `.env` file in the repository root:

```env
DISCORD_BOT_TOKEN=YOUR_BOT_TOKEN_HERE
DISCORD_GUILD_ID=YOUR_GUILD_ID_HERE
DISCORD_ALLOWED_USERS=
```

- `DISCORD_BOT_TOKEN` – Bot token from the Discord Developer Portal (without the `Bot ` prefix).
- `DISCORD_GUILD_ID` – Numeric ID of the Discord server to manage.
- `DISCORD_ALLOWED_USERS` – Optional comma-separated user IDs allowed to control devices. Empty means any guild member in the device channel.

The file is **optionally embedded** into the binary at build time with `-tags envembed` (see `env_embed.go`). Default builds read `.env` from the working directory at startup; existing environment variables take precedence over embedded values.

> Never commit real tokens to a public repository. Rotate tokens immediately if they leak.

### 3.5 OS‑specific preparation (optional)

- **Windows: ffmpeg folder (non‑embedded builds)**  
  Place ffmpeg in:

  ```text
  ffmpeg/windows/ffmpeg.exe
  ```

  This allows the client to run media features even if ffmpeg is not installed globally.

- **Linux: ffmpeg installation**  
  If ffmpeg is not available, the client can try to install it via:

  - `apt-get install -y ffmpeg`
  - `dnf install -y ffmpeg`
  - `yum install -y ffmpeg`
  - `zypper install -y ffmpeg`
  - `pacman -Sy --noconfirm ffmpeg`

  Alternatively, you can place a static binary at:

  ```text
  ffmpeg/linux/ffmpeg
  ```

---

## 4. Build Instructions

All commands are run from the repository root.

### 4.1 Development build

Build a development binary with debug info:

```bash
go build -o dc2-client .
```

Run tests:

```bash
go test ./...
```

### 4.2 Production build (stripped binary)

Produce a smaller Windows binary:

```powershell
go build -ldflags "-s -w" -o dc2-client.exe .
```

Explanation:

- `-s` – strip symbol table and debug info.
- `-w` – omit DWARF debug information.

The resulting `dc2-client.exe` is smaller and contains less debug metadata but behaves identically.

### 4.3 Embedding `.env` and ffmpeg (Windows)

Default builds read `.env` at runtime. To build a **single exe** containing both
configuration and ffmpeg:

1. Ensure `.env` exists in the repo root. It is embedded only when the
   `envembed` build tag is set (see `env_embed.go`).
2. Place `ffmpeg.exe` for embedding at:

   ```text
   internal/media/ffmpeg/windows/ffmpeg.exe
   ```

3. Build with the `envembed` and `ffmpegembed` tags:

   ```powershell
   go build -tags "envembed ffmpegembed" -ldflags "-s -w" -o dc2-client.exe .
   ```

Without `envembed`, a missing `.env` is not a build failure; the client simply
requires the variables to be present in the process environment.

This produces a single `dc2-client.exe` that:

- Extracts ffmpeg to a temporary directory on first use if it is not in `PATH`.
- Uses that extracted binary for all media actions.

### 4.4 Cross‑compilation

Linux (x64) from any OS:

```bash
GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o dc2-client-linux .
```

Windows (x64) from any OS:

```bash
GOOS=windows GOARCH=amd64 go build -ldflags "-s -w" -o dc2-client-windows.exe .
```

### 4.5 Running the client

Windows:

```powershell
.\dc2-client.exe
```

Linux:

```bash
./dc2-client-linux
```

On startup, a device channel is created/updated and a “New device connected” banner is posted and pinned.

### 4.6 Two-container lab (Docker)

Run one Debian container as the target and one as the attacker:

```bash
GO=$(command -v go) ./lab/run.sh
```

The attacker posts commands to the device channel through a webhook, so no
second bot token is required. Full prerequisites, expected output, manual
control, and troubleshooting are in [`docs/LAB.md`](docs/LAB.md).

---

## 5. Usage Guidelines

All operations are driven from the **per‑device channel** in your Discord server.

### 5.1 Slash commands

#### `/help`

- Shows a concise help menu with:
  - Slash commands.
  - Emoji shortcuts.
  - Quick‑action buttons.
- Response is ephemeral (visible only to the caller).

#### `/cmd <command>`

- Starts an asynchronous shell command job on the device:
  - Windows: `cmd.exe /c <command>`.
  - Linux/macOS: `/bin/bash -c <command>`.
- Replies immediately with a job ID (for example `🆔 job 530f3f04 started`).
- When the command finishes, posts status, exit code, duration, and bounded
  output as one or more fenced messages.
- Defaults: 120 s timeout, 200 KB output, 8 concurrent jobs. Configure with
  `DC2_JOB_TIMEOUT`, `DC2_MAX_OUTPUT`, `DC2_MAX_JOBS`.
- Every job is appended to the local audit log (`DC2_AUDIT_LOG`).

Example:

```text
/cmd command="ipconfig /all"
```

#### `/jobs <list|show|kill>`

- `/jobs list` – recent jobs with status, duration, and command.
- `/jobs show <id>` – output for one job.
- `/jobs kill <id>` – cancel a running job.

#### `/dev info`

- Version, commit, OS/arch, Go version, uptime, running jobs, device channel ID.

#### `/media screenshot`

- Captures all active displays using the screenshot library.
- Uploads one PNG per display to the device channel.

#### `/media camera`

- Windows only.
- Captures a single frame from the default camera using ffmpeg/DirectShow:
  - Uses `DC2_CAMERA_DEVICE` env var if set.
  - Otherwise auto‑detects a DirectShow video device.
- Uploads a JPEG snapshot to the channel.

#### `/media mic [duration] [path]`

- Records audio from the microphone.
- Parameters:
  - `duration` (integer, optional):
    - Default: 10 seconds.
    - Range: 1–300 seconds; values outside this range are clamped.
  - `path` (string, optional):
    - Destination path on the device.
    - If omitted, a temp path is used.
    - `~/` is expanded to the user’s home directory.
- Uses ffmpeg to record with a fixed audio configuration optimized for speech.
- Uploads the resulting file if ≤ 8 MB; otherwise reports path and size.

Example:

```text
/media mic duration=60 path="~/recordings/meeting.m4a"
```

#### `/media record <source> [max_size_mb] [duration_minutes] [max_parts] [quality]`

Segmented video recording (Windows):

- `source` (required): `camera` or `screen`.
- `max_size_mb` (optional):
  - Default: 2.
  - Range: 1–8; values outside are clamped.
- `duration_minutes` (optional):
  - Total desired recording duration.
  - If 0, recording continues until manually stopped (subject to other limits).
- `max_parts` (optional):
  - If not provided:
    - Algorithm computes an estimated number of segments:
      - Estimates bitrate based on `quality`:
        - low ≈ 400 kbps
        - medium ≈ 800 kbps
        - high ≈ 1500 kbps
      - Computes `totalBytes = duration * bitrate`.
      - Divides by `max_size_mb` to choose a number of parts (1–30).
    - Ensures at least 5 seconds per segment.
- `quality` (optional):
  - `low`, `medium` (default), or `high`.

Internally, the client:

- For camera:
  - Uses `-f dshow` video/audio capture with ffmpeg.
- For screen:
  - Uses `-f gdigrab` desktop capture (Windows).
- Uses ffmpeg flags:
  - `libx264` encoder.
  - `veryfast` preset.
  - CRF based on `quality`.
  - `-fs` to limit segment size.
- Loops segment creation until:
  - `max_parts` reached, or
  - Requested duration covered, or
  - Recording is stopped via `/media recordstop`.

#### `/media recordstop`

- Signals the recording goroutine to stop current recording.
- Gracefully terminates ffmpeg and finalizes the last segment.

#### `/sys log <mode>`

- `mode`: `on` or `off`.
- When `on`:
  - All Go log lines are forwarded to the device channel as ```log``` blocks.
- When `off`:
  - Logging continues only to stdout/stderr, not Discord.

Note: on startup, the client automatically enables log streaming for the device channel and logs ffmpeg and Windows security checks.

#### `/fs cwd`

- Shows current working directory for the client:
  - Runs `cd` on Windows, `pwd` elsewhere, via `internal/shell`.

#### `/fs ls [path]`

- Lists directory contents:
  - `ls` on Unix.
  - `dir` on Windows.
- `path` optional; defaults to current directory.

#### `/sys ps`

- Lists running processes:
  - `tasklist` on Windows.
  - `ps aux` on Unix‑like systems.

#### `/sys pidinfo <pid>`

- Displays details for a specific process:
  - Windows: `tasklist /FI "PID eq <pid>"`.
  - Unix‑like: `ps -p <pid> -o ...`.

#### `/fs download <source> [destination]`

- Streams a local file back as an attachment.
- Files over `DC2_CHUNK_SIZE` (default 7 MiB) are split into numbered parts plus
  a `manifest.json` containing the whole-file SHA-256, part hashes, and sizes.
- Each part is a separate message; Discord rate-limits very large transfers.
- `source`:
  - File path on the device.
  - `~/` is expanded.
- `destination`:
  - Optional label for your own tracking; does not affect path on disk.

#### `/fs upload <source attachment> <destination>`

- Downloads a Discord attachment and writes it to the device.
- `destination`:
  - May be a directory; the attachment filename is appended.
  - Supports `~/` expansion.

#### `/dev restart`

- Triggers a local restart of the client process:
  - Windows: spawns a new process with same args and exits.
  - Unix‑like: replaces current process via `syscall.Exec`.

#### Windows security commands

##### `/win sec`

- Shows:
  - Firewall state (enabled/disabled/mixed/unknown).
  - Defender real‑time protection state.

##### `/win firewall <mode>`

- `mode`: `on` or `off`.
- Applies to all profiles via `netsh advfirewall set allprofiles state <mode>`.
- Returns reported status and truncated `netsh` output.
- Recommended to run the client with sufficient privileges.

##### `/win defender <mode>`

- `mode`: `on` or `off`.
- Uses PowerShell `Set-MpPreference -DisableRealtimeMonitoring`.
- Reports the final Defender status via `Get-MpComputerStatus`.

##### `/win exclude [path]`

- Adds a Defender exclusion:
  - `path` optional; if omitted, uses current executable path.
  - Path is normalized to an absolute path.
- Uses `Add-MpPreference -ExclusionPath`.

---

### 5.2 Emoji shortcuts

Available in the device channel:

- `📸` – `/media screenshot`
- `📷` – `/media camera` (Windows)
- `🎙` – `/media mic` (10 seconds default)
- `👇 <path>` – `/fs download`
- `☝️ <path>` + attachment – `/fs upload`
- `🔁` – `/dev restart`
- `💀` – terminate the client process

---

## 6. Legal Documentation

### 6.1 Disclaimer and prohibited use

This project is provided strictly for **defensive, educational, and research** purposes.  
You must comply with all applicable laws and obtain proper authorization before deploying it.

Examples of **prohibited** activities include (but are not limited to):

- Running the client on systems or networks you do **not** own or administer.
- Using the client to:
  - Bypass or weaken security controls on third‑party systems.
  - Exfiltrate data from unauthorized environments.
  - Perform espionage, surveillance, or stalking.
  - Deliver or control malware.
- Using this project in violation of:
  - Corporate or organizational policies.
  - Discord’s Terms of Service.
  - Local, national, or international laws.

By using this software, you acknowledge:

- You are solely responsible for its deployment and use.
- The author assumes **no liability** for misuse or damage.

### 6.2 License (MIT)

Copyright (c) 2026 Marwan Alkhatib

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

### 6.3 Author attribution

- **Author**: Marwan Alkhatib

No other human authors are claimed; third‑party libraries maintain their own licenses and attribution separately.

---

## 7. Contributing Guidelines

### 7.1 Code style and general rules

- Go code:
  - Follow idiomatic Go style (`gofmt`, `go vet`).
  - Keep functions small and focused.
  - Prefer explicit error handling and logging.
- Architecture:
  - Keep Discord interaction routing in `internal/commands/router.go`.
  - Add one file per command family under `internal/commands/` and register it
    in `handleSlashCommand`.
  - Keep platform‑specific logic isolated (for example `internal/winsec`).
  - Keep media logic in `internal/media/media.go`.
  - Leaf packages must not import `internal/bot` or `internal/commands`.

### 7.2 Branching and pull requests

1. Fork the repository.
2. Create a feature branch:

   ```bash
   git checkout -b feature/short-description
   ```

3. Make changes and ensure:

   ```bash
   go test ./...
   go build ./...
   ```

4. Submit a pull request with:
   - Clear description of the change.
   - Rationale (bug fix, new feature, refactor).
   - Any notable security or compatibility impacts.

### 7.3 Commit messages

- Use concise, descriptive messages:
  - `fix: handle Windows firewall error output`
  - `feat: add segmented video recording`
  - `refactor: split interactions into separate files`

### 7.4 Reporting issues

When opening an issue, include:

- Environment details:
  - OS and version.
  - Go version.
  - DiscordGo version (from `go.mod`).
- Reproduction steps:
  - Exact commands used (`go build`, `go run`, slash command).
  - Relevant log excerpts (scrub sensitive data).
  - Expected vs actual behavior.

---

## 8. Troubleshooting Guide

### 8.1 Client does not create a device channel

- Verify:
  - `DISCORD_BOT_TOKEN` and `DISCORD_GUILD_ID` in `.env` are correct.
  - The bot is invited to the guild.
  - Bot has `Manage Channels` permission.
- Check logs (via `/sys log on` or console) for:
  - `failed to create Discord session`
  - `failed to ensure category`
  - `failed to ensure device channel`

### 8.2 Slash commands do not appear

- Ensure the client has run at least once with valid credentials.
- Check logs for:
  - `failed to list existing application commands`
  - `failed to register command <name>`
- Discord may cache commands; wait up to a few minutes or re‑invite the bot.

### 8.3 ffmpeg errors

Symptoms:

- `/media mic`, `/media camera`, `/media record` fail with “ffmpeg unavailable” or detailed ffmpeg error output.

Checks:

- Confirm ffmpeg is installed:

  ```powershell
  where ffmpeg
  ```

- Or confirm bundled/embedded binary:
  - `ffmpeg/windows/ffmpeg.exe` (bundled).
  - `ffmpeg/linux/ffmpeg` (Linux).
  - Or build with `-tags ffmpegembed` and ensure `internal/media/ffmpeg/windows/ffmpeg.exe` is present at build time.

### 8.4 Windows firewall/Defender commands fail

- Ensure the client is running with administrative privileges.
- Check error messages returned by `/win firewall` or `/win defender`:
  - They include truncated `netsh` or PowerShell error text.
- Verify:
  - PowerShell is available (`powershell` in `PATH`).
  - `Get-MpComputerStatus` and `Set-MpPreference` are supported on the OS version.

### 8.5 Log streaming not visible

- `/sys log on` enables log streaming for the channel.
- On startup, the client already enables logging for its device channel; if you moved the client between channels or guilds, rerun `/sys log on`.

---

## 9. FAQ

**Q: Is this a full C2 framework?**  
A: No. This is a single client designed to connect to Discord as its transport. It does not include a separate, complex server—Discord is the “controller” UI.

**Q: Can I use this for unauthorized penetration testing?**  
A: No. Use is restricted to environments where you have explicit authorization. See the legal section.

**Q: Can I add new actions (features)?**  
A: Yes. Add the definition in `internal/commands/definitions.go`, route it in `handleSlashCommand` in `internal/commands/router.go`, and implement the handler in a new file under `internal/commands/`.

**Q: Does this support multiple operators?**  
A: Yes. Any Discord user with access to the device channel can use the slash commands and emoji shortcuts, unless `DISCORD_ALLOWED_USERS` is set, in which case only those user IDs are accepted.

**Q: How big can uploads/fs downloads be?**  
A: Downloads are split into parts of `DC2_CHUNK_SIZE` (default 7 MiB) with a manifest, so large files work. Uploads from Discord remain bounded by Discord's own attachment limit (8 MB in this client).

---

## 10. Changelog (summary)

Recent notable changes:

- Supply chain: Dependabot, dependency review, release provenance attestations,
  and branch protection on `main`.
- Approval requests are audited (`approval.request/approve/deny`); shutdown
  cancels running jobs and waits up to five seconds.
- Two-person approval (`DC2_REQUIRE_APPROVAL`, `DC2_APPROVERS`): commands wait
  for an Approve/Deny click from a second operator; requests expire in 10 min.
- Job history persists to `DC2_JOB_STORE`, so `/jobs list|show` survives restarts.
- Log streaming is rate-limited to 5 msg/s with a drop summary.
- Releases include a CycloneDX SBOM; Windows signing runs when the signing
  secrets are configured.
- Chunked downloads: files over `DC2_CHUNK_SIZE` (7 MiB default) are split into
  parts with a `manifest.json` and SHA-256 hashes.
- `--check` preflight validates token, guild, permissions, audit path, ffmpeg,
  and `.env` permissions without connecting.
- Published detection pack (`detections/`): YARA, Sigma, IOCs, ATT&CK mapping,
  hunting queries, and a purple-team validation plan.
- Asynchronous job engine: `/cmd`, 🏃‍♂️, and quick buttons return a job ID and post
  bounded output when done; `/jobs list|show|kill`; per-job timeouts and
  concurrency limits; local JSONL audit trail (`DC2_AUDIT_LOG`).
- `/dev info` reports version, commit, OS/arch, uptime, and running jobs;
  `scripts/build.sh` injects version metadata.
- Security audit: upgraded `gorilla/websocket` (GO-2026-6278), `discordgo`, and
  `screenshot`; added `DISCORD_ALLOWED_USERS`; attachment HTTP timeout; ffmpeg
  temp permissions tightened. `govulncheck` and `staticcheck` are clean.
- Added a two-container Docker lab (`lab/`, `deploy/`) and `docs/LAB.md`,
  `docs/CONFIGURATION.md`, `docs/CONTROL.md`, `docs/SECURITY.md`.

- Restructured into `main` + `internal/*` packages (`bot`, `commands`, `config`,
  `logstream`, `media`, `process`, `shell`, `winsec`); command definitions moved
  to `internal/commands/definitions.go`.
- Made `.env` embedding opt-in via `-tags envembed`, so default builds no longer
  fail when `.env` is absent.
- Renamed `interactions_windows.go` to a platform-neutral file so non-Windows
  builds compile.
- Added tests for config loading, command definitions, and Windows security
  helpers, plus a `Makefile` and `docs/ARCHITECTURE.md`.
- Refactored core logic into modular command files under `internal/commands/` and `internal/media/media.go`.
- Added comprehensive slash commands for:
  - Media (screenshot, camera, mic, record, recordstop).
  - Filesystem (cwd, ls, download, upload).
  - Processes (ps, pidinfo).
  - Windows security (`/win sec`, `/win firewall`, `/win defender`, `/win exclude`).
  - Logging (/sys log) and restart (/dev restart).
- Implemented ffmpeg bundling and optional embedding (`ffmpegembed` build tag).
- Added automatic Defender exclusion for the current executable.
- Enhanced initial device banner with firewall/Defender status (Windows).

---

## 11. Support and Resources

- **Issues & bugs**: Use the repository’s issue tracker.
- **Feature requests**: Open an issue with `[feature]` in the title and a clear description.
- **Security concerns**: Use responsible disclosure; do not post exploit details publicly.

**Related documentation and resources**

- Discord developer portal: <https://discord.com/developers/docs/intro>
- DiscordGo library: <https://github.com/bwmarrin/discordgo>
- FFmpeg documentation: <https://ffmpeg.org/documentation.html>
- Screenshot library: <https://github.com/kbinani/screenshot>

---

© 2026 Marwan Alkhatib. All rights reserved under the MIT License (see above).  
Use responsibly and only in environments where you have explicit authorization.
