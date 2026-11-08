# Control

Everything is driven from the per-device channel that the client creates under
its OS category (`windows`, `linux`, `mac`, `other`).

## 1. Access model

- The channel name is derived from `hostname-username`, normalized to lowercase
  alphanumerics and dashes, max 90 characters.
- The client ignores messages from other channels and from itself.
- If `DISCORD_ALLOWED_USERS` is set, only those user IDs may run commands.
  Everyone else gets `⛔ You are not authorized to control this device.`
- If `DISCORD_ALLOWED_USERS` is empty, any guild member who can post in the
  device channel can control the host. The client logs a warning at startup.

## 2. Execution model

Arbitrary commands (`/cmd`, the 🏃‍♂️ shortcut, and the quick-info buttons) run as
**asynchronous jobs**:

1. The client replies immediately with a job ID: `🆔 job abc12345 started`.
2. The command runs with a timeout, concurrency limit, and bounded output.
3. When it finishes, the client posts the status, exit code, duration, and
   output as one or more fenced messages.

Defaults: 120 s timeout, 200 KB output, 8 concurrent jobs. Configure with
`DC2_JOB_TIMEOUT`, `DC2_MAX_OUTPUT`, and `DC2_MAX_JOBS`. Every job is written to
the local audit log (`DC2_AUDIT_LOG`) and the job store (`DC2_JOB_STORE`), so
`/jobs` survives restarts.

If `DC2_REQUIRE_APPROVAL=true`, commands are not executed immediately. The
client posts an approval prompt with **Approve**/**Deny** buttons; a second
operator (listed in `DC2_APPROVERS`, or any allowlisted user when the list is
empty) must approve. The requester cannot approve their own command. Requests
expire after 10 minutes. Requests, approvals, and denials are written to the
audit log.

On shutdown (SIGINT/SIGTERM) the client cancels running jobs and waits up to
five seconds for them to exit.

Fixed commands (`/fs`, `/sys`, media) stay synchronous and reply inline.

## 3. Slash commands

| Command | Arguments | Action |
|---|---|---|
| `/help` | — | Ephemeral, OS-aware summary of available controls. |
| `/cmd` | `command` | Start a shell command job (`cmd.exe /c` on Windows, `/bin/bash -c` elsewhere). |
| `/jobs list` | — | Recent jobs with status, duration, and command. |
| `/jobs show` | `id` | Job output (chunked). |
| `/jobs kill` | `id` | Cancel a running job. |
| `/media screenshot` | — | Capture all displays and upload PNGs. |
| `/media camera` | — | Capture a still from the default camera (ffmpeg). |
| `/media mic` | `duration` 1–300 s, `path` | Record microphone audio. Default 10 s. |
| `/media record` | `source` (`camera`/`screen`), `max_size_mb`, `duration_minutes`, `max_parts`, `quality` | Segmented video recording sized for Discord. |
| `/media recordstop` | — | Stop the active recording. |
| `/fs cwd` | — | Print the working directory. |
| `/fs ls` | `path` | List a directory. |
| `/fs download` | `source`, `destination` | Send a file from the host. Files over `DC2_CHUNK_SIZE` are split into parts plus `manifest.json`. |
| `/fs upload` | `source` attachment, `destination` | Write an attachment to the host. |
| `/sys ps` | — | List processes (`ps aux` / `tasklist`). |
| `/sys pidinfo` | `pid` | Inspect one process. |
| `/sys log` | `mode` (`on`/`off`) | Stream client logs into the channel. |
| `/win sec` | — | Windows firewall + Defender status. |
| `/win firewall` | `mode` (`on`/`off`) | Toggle Windows firewall. |
| `/win defender` | `mode` (`on`/`off`) | Toggle Defender real-time protection. |
| `/win exclude` | `path` | Add a Defender exclusion (defaults to the current executable). |
| `/dev restart` | — | Re-exec the client process. |
| `/dev info` | — | Version, commit, OS/arch, Go version, uptime, running jobs. |

Commands are grouped in the Discord picker: `media`, `fs`, `sys`, `win`, `dev`.
Discord scopes commands to the guild, not the channel, so the `win` group is
registered only when the client runs on Windows. Run one OS per guild (or one
bot per OS) for a mixed fleet; otherwise the Windows group appears/disappears
with whichever client last synced commands.

Windows-only subcommands also reply with an "unsupported" message if invoked on
another platform.

## 4. Message shortcuts
Send these in the device channel:

| Message | Action |
|---|---|
| `🏃‍♂️ <command>` | Run a shell command. |
| `help` | Post the quick-help message with buttons. |
| `📸` | Screenshot. |
| `🎙` | 10-second microphone recording. |
| `📷` | Camera snapshot. |
| `👇 <path>` | Download a file from the host, split when over the chunk size. |
| `☝️ <path>` + attachment | Upload the attachment to `<path>`. |
| `🔁` | Restart the client. |
| `💀` | Close the session and exit. |

## 5. Buttons and modals

The quick-help message exposes buttons for `whoami`, `id`, `hostname`, `ip info`,
`cwd`, and screenshots. `Run command` and `ls` open modals.

## 6. Limits and behavior

- Command output is capped per job (`DC2_MAX_OUTPUT`, default 200 KB) and then
  chunked into Discord-sized code blocks; long output becomes multiple messages.
- Jobs time out (`DC2_JOB_TIMEOUT`, default 120 s) and are limited to
  `DC2_MAX_JOBS` concurrent processes (default 8).
- File transfers are capped at Discord's 8 MB attachment limit per message;
  larger downloads are split into parts plus a `manifest.json` (each part is a
  separate message, so large files are rate-limited by Discord).
- Uploads from Discord remain bounded by Discord's own attachment limit.
- Only one segmented recording can run at a time (`media.videoState`).
- Log streaming is queued and rate-limited to 5 messages per second; excess
  lines are dropped and summarized. Leave it off outside troubleshooting.

## 7. Non-Discord control (lab)

`lab/attacker.sh` posts commands through a channel webhook, which the client
accepts because webhook messages are not authored by the bot itself. See
[`LAB.md`](./LAB.md).
