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

## 2. Slash commands

| Command | Arguments | Action |
|---|---|---|
| `/help` | — | Ephemeral, OS-aware summary of available controls. |
| `/cmd` | `command` | Run a shell command (`cmd.exe /c` on Windows, `/bin/bash -c` elsewhere). |
| `/media screenshot` | — | Capture all displays and upload PNGs. |
| `/media camera` | — | Capture a still from the default camera (ffmpeg). |
| `/media mic` | `duration` 1–300 s, `path` | Record microphone audio. Default 10 s. |
| `/media record` | `source` (`camera`/`screen`), `max_size_mb`, `duration_minutes`, `max_parts`, `quality` | Segmented video recording sized for Discord. |
| `/media recordstop` | — | Stop the active recording. |
| `/fs cwd` | — | Print the working directory. |
| `/fs ls` | `path` | List a directory. |
| `/fs download` | `source`, `destination` | Upload a file from the host (max 8 MB). |
| `/fs upload` | `source` attachment, `destination` | Write an attachment to the host. |
| `/sys ps` | — | List processes (`ps aux` / `tasklist`). |
| `/sys pidinfo` | `pid` | Inspect one process. |
| `/sys log` | `mode` (`on`/`off`) | Stream client logs into the channel. |
| `/win sec` | — | Windows firewall + Defender status. |
| `/win firewall` | `mode` (`on`/`off`) | Toggle Windows firewall. |
| `/win defender` | `mode` (`on`/`off`) | Toggle Defender real-time protection. |
| `/win exclude` | `path` | Add a Defender exclusion (defaults to the current executable). |
| `/dev restart` | — | Re-exec the client process. |

Commands are grouped in the Discord picker: `media`, `fs`, `sys`, `win`, `dev`.
Discord scopes commands to the guild, not the channel, so the `win` group is
registered only when the client runs on Windows. Run one OS per guild (or one
bot per OS) for a mixed fleet; otherwise the Windows group appears/disappears
with whichever client last synced commands.

Windows-only subcommands also reply with an "unsupported" message if invoked on
another platform.

## 3. Message shortcuts

Send these in the device channel:

| Message | Action |
|---|---|
| `🏃‍♂️ <command>` | Run a shell command. |
| `help` | Post the quick-help message with buttons. |
| `📸` | Screenshot. |
| `🎙` | 10-second microphone recording. |
| `📷` | Camera snapshot. |
| `👇 <path>` | Download a file from the host (max 8 MB). |
| `☝️ <path>` + attachment | Upload the attachment to `<path>`. |
| `🔁` | Restart the client. |
| `💀` | Close the session and exit. |

## 4. Buttons and modals

The quick-help message exposes buttons for `whoami`, `id`, `hostname`, `ip info`,
`cwd`, and screenshots. `Run command` and `ls` open modals.

## 5. Limits and behavior

- Command output is chunked into 1800-character code blocks; long output becomes
  multiple messages.
- File transfers are capped at Discord's 8 MB attachment limit.
- Only one segmented recording can run at a time (`media.videoState`).
- Log streaming forwards each log line as a separate message; leave it off
  outside troubleshooting to avoid rate limits.

## 6. Non-Discord control (lab)

`lab/attacker.sh` posts commands through a channel webhook, which the client
accepts because webhook messages are not authored by the bot itself. See
[`LAB.md`](./LAB.md).
