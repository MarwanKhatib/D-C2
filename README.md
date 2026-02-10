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

- Language: Go 1.19+
- Protocol: Discord HTTP + WebSocket via [github.com/bwmarrin/discordgo](https://github.com/bwmarrin/discordgo)
- Platforms:
  - Windows: full feature set (filesystem, commands, media, Windows security).
  - Linux/macOS: command, filesystem, process, and basic system features.
- Media:
  - Screenshots via [github.com/kbinani/screenshot](https://github.com/kbinani/screenshot).
  - Audio/video capture via ffmpeg (system or bundled/embedded binary).

---

## 2. Project Structure Map

Repository root:

```text
discord-c2-server/
├── actions/
│   ├── media.go
│   ├── ffmpeg_embed_windows.go
│   └── ffmpeg_embed_stub.go
├── categories.go
├── client.go
├── env.go
├── handler.go
├── interactions.go
├── interactions_cmd.go
├── interactions_control.go
├── interactions_fs.go
├── interactions_help.go
├── interactions_help_test.go
├── interactions_log.go
├── interactions_media.go
├── interactions_system.go
├── interactions_windows.go
├── go.mod
├── go.sum
├── .gitignore
└── README.md
```

**Directory and file descriptions**

- `actions/`
  - `media.go`  
    Implements all media‑related actions:
    - `Screenshot`: captures all displays and uploads PNG images.
    - `CameraSnapshot`: Windows camera still image via ffmpeg/DirectShow.
    - `MicRecord`: Windows microphone recording to audio file (size‑checked).
    - `VideoRecord` / `VideoStop`: segmented video recording using ffmpeg, with bitrate‑based segmentation algorithm (see below).
    - ffmpeg availability logic:
      - Finds ffmpeg in `PATH`.
      - Falls back to bundled `ffmpeg` folder.
      - On Linux, can install ffmpeg via `apt-get`, `dnf`, `yum`, `zypper`, or `pacman`.
  - `ffmpeg_embed_windows.go`  
    Windows‑only, when built with `-tags ffmpegembed`. Uses `//go:embed` to bake `ffmpeg/windows/ffmpeg.exe` into the binary as `embeddedFFmpegWin`.
  - `ffmpeg_embed_stub.go`  
    Stub for non‑embed builds or non‑Windows OS. Defines an empty `embeddedFFmpegWin` so the code compiles everywhere.

- `client.go`  
  Main entrypoint:
  - Loads configuration via `loadEnvFile(".env")`.
  - Creates Discord session and registers:
    - Slash commands (`/cmd`, `/screenshot`, `/camera`, `/mic`, `/record`, `/recordstop`, `/log`, `/cwd`, `/ls`, `/ps`, `/pidinfo`, `/winsec`, `/winfirewall`, `/windefender`, `/windefenderexclude`, `/download`, `/upload`, `/restart`).
  - Detects OS, user, hostname, IP.
  - Creates/ensures OS category and device channel using helpers in `categories.go`.
  - Posts the initial banner and pins it.
  - Sets `log.SetOutput` to a custom writer that can stream logs into Discord.
  - Automatically:
    - Ensures the current executable is excluded from Defender (Windows).
    - Enables log streaming to the device channel.
    - Checks ffmpeg availability and logs the result.

- `env.go`  
  Configuration and environment embedding:
  - Uses `//go:embed .env` to optionally embed `.env` at build time.
  - `loadEnvFile`:
    - Reads from embedded `.env` if present.
    - Otherwise parses `.env` from disk.
  - Sets process‑wide env vars such as:
    - `DISCORD_BOT_TOKEN`
    - `DISCORD_GUILD_ID`

- `categories.go`  
  Discord channel / category helpers:
  - `getOrCreateCategory`: creates `windows`, `linux`, `mac`, or `other`.
  - `getOrCreateDeviceChannel`: creates or reuses a channel per device.
  - `makeChannelName`: normalizes `hostname-username` into a valid channel name.

- `handler.go`  
  Core Discord message handler and log streaming:
  - `discordLogWriter`:
    - Wraps an underlying writer (stdout) and optionally forwards log lines to a device channel as ```log``` blocks.
  - `enableLogStreaming` / `disableLogStreaming`:
    - Toggle streaming of Go logs into a specific channel.
  - `handler`:
    - Processes messages in the device channel.
    - Supports emoji shortcuts:
      - `📸` – screenshots.
      - `📷` – camera snapshot.
      - `🎙` – 10‑second microphone recording.
      - `👇 <path>` – download file (8 MB limit).
      - `☝️ <path>` + attachment – upload file.
      - `🔁` – restart client.
      - `💀` – terminate client.

- `interactions.go`  
  Slash command dispatcher:
  - Routes each slash command to the corresponding `handleSlash*` function in the specialized files below.

- `interactions_cmd.go`  
  - `/cmd <command>` – run arbitrary shell command:
    - Windows: `cmd.exe /c`.
    - Linux/macOS: `/bin/bash -c`.

- `interactions_fs.go`  
  Filesystem commands:
  - `/cwd` – show current working directory (`cd` on Windows, `pwd` elsewhere).
  - `/ls [path]` – list directory (`dir` on Windows, `ls` elsewhere).
  - `/download <source> [destination]` – download file (≤ 8 MB) as attachment.
  - `/upload <source attachment> <destination>` – write Discord attachment to a path on the client, resolving `~/` and directories.

- `interactions_media.go`  
  Media commands:
  - `/screenshot` – capture all active displays and upload PNGs.
  - `/camera` – Windows camera snapshot (DirectShow via ffmpeg).
  - `/mic [duration] [path]`:
    - Default duration: 10 seconds.
    - Range: 1–300 seconds enforced.
    - Optional `path` (supports `~/` expansion); if omitted, uses temp directory.
  - `/record <source> [max_size_mb] [duration_minutes] [max_parts] [quality]`:
    - `source`: `camera` or `screen` (Windows only).
    - `max_size_mb`: default 2, clamped 1–8 MB per file.
    - `duration_minutes`: optional total duration.
    - `max_parts`: optional; if omitted, computed based on bitrate estimate.
    - `quality`: `low`, `medium` (default), or `high`.
    - Uses a simple bitrate‑based segmentation algorithm:
      - Estimate total bytes as `duration * bitrate`.
      - Divide by `max_size_mb` to derive `max_parts` (capped at 30).
      - Enforces minimum 5 seconds per segment.
  - `/recordstop` – stop current video recording (if any).

- `interactions_system.go`  
  System information:
  - `/ps` – list processes:
    - `tasklist` on Windows.
    - `ps aux` on Unix‑like systems.
  - `/pidinfo <pid>` – show details for a specific process:
    - Uses `tasklist /FI "PID eq <pid>"` on Windows.
    - Uses `ps -p <pid> ...` on Unix‑like systems.

- `interactions_windows.go`  
  Windows‑specific security controls:
  - `/winsec`:
    - Reports firewall state (enabled/disabled/mixed/unknown) via `netsh advfirewall`.
    - Reports Defender real‑time protection via `Get-MpComputerStatus`.
  - `/winfirewall <mode>`:
    - `mode`: `on` or `off`.
    - Uses `netsh advfirewall set allprofiles state <mode>`.
    - Logs results and returns `netsh` output (trimmed) to the operator.
  - `/windefender <mode>`:
    - `mode`: `on` or `off`.
    - Uses PowerShell `Set-MpPreference -DisableRealtimeMonitoring`.
  - `/windefenderexclude [path]`:
    - Adds Defender exclusion for a path.
    - If `path` omitted, uses the current executable.
    - Internally uses `Get-MpPreference` and `Add-MpPreference`.
  - Startup behavior:
    - `ensureCurrentExeDefenderExcluded`:
      - Checks if the current executable is already excluded.
      - Adds exclusion if needed and logs the outcome.

- `interactions_help.go` / `interactions_help_test.go`  
  - `/help` – sends an ephemeral help message summarizing:
    - Slash commands.
    - Emoji shortcuts.
    - Quick‑action buttons (`whoami`, `id`, `hostname`, `ip info`, `cwd`).
  - Test file exercises the help handler in isolation.

- `interactions_log.go`  
  - `/log <mode>`:
    - `mode`: `on` or `off`.
    - Enables/disables streaming of Go logs into the current channel.

- `interactions_control.go`  
  - `/restart` – restart the client process using platform‑specific re‑exec logic.

- `interactions.go`  
  - Central interaction handler that:
    - Inspects `InteractionCreate` payloads.
    - Dispatches to the correct `handleSlash*` function.

- `go.mod`, `go.sum`  
  - Go module definition and dependency lockfile.

- `.gitignore`  
  - Standard Git ignore rules (build artifacts, local environment files, etc.).

- `README.md`  
  - This documentation.

---

## 3. Installation Requirements

### 3.1 Supported platforms

- Windows 10/11 (x64) – full capabilities.
- Linux (modern distributions; x64) – non‑media features fully supported; media features depend on ffmpeg and display/audio stack.
- macOS (x64/ARM64) – basic command/filesystem/process features; media features are currently Windows‑focused.

### 3.2 Software prerequisites

- **Go**: 1.19 or newer (for building from source).
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
```

- `DISCORD_BOT_TOKEN` – Bot token from the Discord Developer Portal (without the `Bot ` prefix).
- `DISCORD_GUILD_ID` – Numeric ID of the Discord server to manage.

The file is **optionally embedded** into the binary at build time via `//go:embed .env`. If embedded, you do not need to ship `.env` separately unless you want to override values via environment variables.

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

To build a **single exe** containing both configuration and ffmpeg:

1. Ensure `.env` exists in the repo root. It is automatically embedded by `env.go`.
2. Place `ffmpeg.exe` for embedding at:

   ```text
   actions/ffmpeg/windows/ffmpeg.exe
   ```

3. Build with the `ffmpegembed` tag:

   ```powershell
   go build -tags ffmpegembed -ldflags "-s -w" -o dc2-client.exe .
   ```

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

- Runs a shell command on the device:
  - Windows: `cmd.exe /c <command>`.
  - Linux/macOS: `/bin/bash -c <command>`.
- Output is sent back to the channel. Long output may be chunked or uploaded as a file.

Example:

```text
/cmd command="ipconfig /all"
```

#### `/screenshot`

- Captures all active displays using the screenshot library.
- Uploads one PNG per display to the device channel.

#### `/camera`

- Windows only.
- Captures a single frame from the default camera using ffmpeg/DirectShow:
  - Uses `DC2_CAMERA_DEVICE` env var if set.
  - Otherwise auto‑detects a DirectShow video device.
- Uploads a JPEG snapshot to the channel.

#### `/mic [duration] [path]`

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
/mic duration=60 path="~/recordings/meeting.m4a"
```

#### `/record <source> [max_size_mb] [duration_minutes] [max_parts] [quality]`

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
  - Recording is stopped via `/recordstop`.

#### `/recordstop`

- Signals the recording goroutine to stop current recording.
- Gracefully terminates ffmpeg and finalizes the last segment.

#### `/log <mode>`

- `mode`: `on` or `off`.
- When `on`:
  - All Go log lines are forwarded to the device channel as ```log``` blocks.
- When `off`:
  - Logging continues only to stdout/stderr, not Discord.

Note: on startup, the client automatically enables log streaming for the device channel and logs ffmpeg and Windows security checks.

#### `/cwd`

- Shows current working directory for the client:
  - Runs `cd` on Windows, `pwd` elsewhere, via `runCommandString`.

#### `/ls [path]`

- Lists directory contents:
  - `ls` on Unix.
  - `dir` on Windows.
- `path` optional; defaults to current directory.

#### `/ps`

- Lists running processes:
  - `tasklist` on Windows.
  - `ps aux` on Unix‑like systems.

#### `/pidinfo <pid>`

- Displays details for a specific process:
  - Windows: `tasklist /FI "PID eq <pid>"`.
  - Unix‑like: `ps -p <pid> -o ...`.

#### `/download <source> [destination]`

- Streams a small local file back as an attachment.
- Enforces 8 MB limit.
- `source`:
  - File path on the device.
  - `~/` is expanded.
- `destination`:
  - Optional label for your own tracking; does not affect path on disk.

#### `/upload <source attachment> <destination>`

- Downloads a Discord attachment and writes it to the device.
- `destination`:
  - May be a directory; the attachment filename is appended.
  - Supports `~/` expansion.

#### `/restart`

- Triggers a local restart of the client process:
  - Windows: spawns a new process with same args and exits.
  - Unix‑like: replaces current process via `syscall.Exec`.

#### Windows security commands

##### `/winsec`

- Shows:
  - Firewall state (enabled/disabled/mixed/unknown).
  - Defender real‑time protection state.

##### `/winfirewall <mode>`

- `mode`: `on` or `off`.
- Applies to all profiles via `netsh advfirewall set allprofiles state <mode>`.
- Returns reported status and truncated `netsh` output.
- Recommended to run the client with sufficient privileges.

##### `/windefender <mode>`

- `mode`: `on` or `off`.
- Uses PowerShell `Set-MpPreference -DisableRealtimeMonitoring`.
- Reports the final Defender status via `Get-MpComputerStatus`.

##### `/windefenderexclude [path]`

- Adds a Defender exclusion:
  - `path` optional; if omitted, uses current executable path.
  - Path is normalized to an absolute path.
- Uses `Add-MpPreference -ExclusionPath`.

---

### 5.2 Emoji shortcuts

Available in the device channel:

- `📸` – `/screenshot`
- `📷` – `/camera` (Windows)
- `🎙` – `/mic` (10 seconds default)
- `👇 <path>` – `/download`
- `☝️ <path>` + attachment – `/upload`
- `🔁` – `/restart`
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
  - Keep Discord interaction routing in `interactions*.go`.
  - Keep platform‑specific logic isolated (e.g., `interactions_windows.go`).
  - Keep media‑specific logic in `actions/media.go`.

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
- Check logs (via `/log on` or console) for:
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

- `/mic`, `/camera`, `/record` fail with “ffmpeg unavailable” or detailed ffmpeg error output.

Checks:

- Confirm ffmpeg is installed:

  ```powershell
  where ffmpeg
  ```

- Or confirm bundled/embedded binary:
  - `ffmpeg/windows/ffmpeg.exe` (bundled).
  - `ffmpeg/linux/ffmpeg` (Linux).
  - Or build with `-tags ffmpegembed` and ensure `actions/ffmpeg/windows/ffmpeg.exe` is present at build time.

### 8.4 Windows firewall/Defender commands fail

- Ensure the client is running with administrative privileges.
- Check error messages returned by `/winfirewall` or `/windefender`:
  - They include truncated `netsh` or PowerShell error text.
- Verify:
  - PowerShell is available (`powershell` in `PATH`).
  - `Get-MpComputerStatus` and `Set-MpPreference` are supported on the OS version.

### 8.5 Log streaming not visible

- `/log on` enables log streaming for the channel.
- On startup, the client already enables logging for its device channel; if you moved the client between channels or guilds, rerun `/log on`.

---

## 9. FAQ

**Q: Is this a full C2 framework?**  
A: No. This is a single client designed to connect to Discord as its transport. It does not include a separate, complex server—Discord is the “controller” UI.

**Q: Can I use this for unauthorized penetration testing?**  
A: No. Use is restricted to environments where you have explicit authorization. See the legal section.

**Q: Can I add new actions (features)?**  
A: Yes. Add new slash command definitions in `client.go`, route them in `interactions.go`, and implement handlers in a dedicated `interactions_*.go` or `actions/*.go` file.

**Q: Does this support multiple operators?**  
A: Yes. Any Discord user with access to the device channel can use the slash commands and emoji shortcuts.

**Q: How big can uploads/downloads be?**  
A: The code enforces an 8 MB limit for files sent to Discord to comply with standard Discord file limits.

---

## 10. Changelog (summary)

Recent notable changes:

- Refactored core logic into modular `interactions_*.go` and `actions/media.go`.
- Added comprehensive slash commands for:
  - Media (screenshot, camera, mic, record, recordstop).
  - Filesystem (cwd, ls, download, upload).
  - Processes (ps, pidinfo).
  - Windows security (winsec, winfirewall, windefender, windefenderexclude).
  - Logging (/log) and restart (/restart).
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
