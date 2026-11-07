# Indicators of Compromise

These are indicators this project creates. They are deliberately published so
defenders can detect, tune, and deconflict authorized activity.

## Network

| Indicator | Notes |
|---|---|
| `discord.com/api/v10/*` | REST control plane (commands, files, channels). |
| `gateway.discord.gg` (WSS) | Gateway connection; long-lived TLS session. |
| `cdn.discordapp.com` | Attachment upload/download. |
| `discord.com/api/webhooks/*` | Webhook posts (lab attacker and many commodity implants). |
| Outbound 443 only | No inbound listener; no custom C2 port. |

Correlate destination with the initiating process. Discord traffic from a
browser or the Discord client is normal; from an unsigned binary in `%ProgramData%`
or `/usr/local/bin` it is not.

## Host — Windows

| Artifact | Value |
|---|---|
| Binary | `C:\ProgramData\D-C2\dc2-client.exe` (default installer path) |
| Config | `C:\ProgramData\D-C2\.env` containing `DISCORD_BOT_TOKEN` |
| Persistence | Scheduled task named `D-C2`, trigger AtLogOn |
| Audit | `%USERPROFILE%\.config\dc2\audit.jsonl` (JSON lines) |
| Temp | `%TEMP%\dc2-ffmpeg\ffmpeg.exe` when the embedded binary is used |
| Defender | Exclusion path added for the client executable (requires elevation) |
| Child processes | `cmd.exe`, `powershell.exe`, `ffmpeg.exe` spawned by the client |

## Host — Linux / macOS

| Artifact | Value |
|---|---|
| Binary | `/usr/local/bin/dc2-client` (installer) or a user-chosen path |
| Config | `/etc/dc2/.env` (installer) or the working directory |
| Service | `/etc/systemd/system/dc2.service` |
| Audit | `~/.config/dc2/audit.jsonl` |
| Temp | `$TMPDIR/dc2-ffmpeg/ffmpeg` |
| Child processes | `/bin/bash -c ...`, `ffmpeg` |

## Discord-side

| Artifact | Value |
|---|---|
| Category | `windows`, `linux`, `mac`, or `other` |
| Channel | `<hostname>-<username>` under that category |
| Pinned message | `🛰️ New device connected!` banner with session, IP, user, host, OS, arch, CWD |
| Slash commands | `media`, `fs`, `sys`, `win`, `dev`, `jobs` groups |
| Messages | `🆔 job <id> started`, `✅ job <id> done`, `🔄 reconnected` |
| Bot behavior | Creates channels, posts webhooks, bulk-overwrites application commands |

## TTP mapping

| Behavior | ATT&CK |
|---|---|
| Discord API as C2 | T1102.002 (Web Service) |
| Web protocols | T1071.001 |
| Command execution via shell | T1059.003 / T1059.004 |
| Scheduled task / systemd persistence | T1053.005 / T1543.002 |
| Screen capture / audio / video | T1113, T1125 |
| File transfer | T1041, T1105 |
| Impair defenses (Defender exclusion) | T1562.001 — only with explicit authorization |

## Hunting queries

Microsoft Defender XDR (device network events):

```kql
DeviceNetworkEvents
| where RemoteUrl has_any ("discord.com", "discordapp.com")
| where InitiatingProcessFileName !in~ ("discord.exe","chrome.exe","msedge.exe","firefox.exe","brave.exe","opera.exe")
| project Timestamp, DeviceName, InitiatingProcessFileName, InitiatingProcessCommandLine, RemoteUrl
| order by Timestamp desc
```

Sysmon process creation (EID 1):

```text
ParentImage ends with \dc2-client.exe
  OR (Image ends with \cmd.exe OR \powershell.exe) AND CommandLine contains "discord-c2"
```

Splunk (proxy or DNS):

```spl
index=proxy dest IN ("discord.com","discordapp.com")
| stats count by src, process_name, dest
| where process_name!="discord.exe" AND process_name!="chrome.exe"
```

File hunting:

```bash
grep -RIl --include='*.env' 'DISCORD_BOT_TOKEN' /etc /home /opt 2>/dev/null
grep -RIl 'dc2-audit.jsonl' /home /root 2>/dev/null
```

## Limitations

- Discord tokens are a generic pattern; a hit is a lead, not a verdict.
- Discord traffic is common in many environments. Process attribution is the
  discriminating signal.
- The binary can be renamed and rebuilt; hash IOCs age quickly. Prefer behavior.
- Published indicators are for defense and deconfliction, not a guarantee of
  completeness.
