# Install on a target

> Use only on hosts you own or have explicit written authorization to test.

## 1. Preflight

- [ ] Bot token and guild ID (see [`SERVER.md`](./SERVER.md)).
- [ ] A built binary for the target OS (see [`BUILD.md`](./BUILD.md)).
- [ ] `DISCORD_ALLOWED_USERS` set to the operator's user ID.
- [ ] Outbound HTTPS/WSS to `discord.com` allowed.
- [ ] You have a way to remove the client afterwards (see §7).

The client needs no inbound port, no listening service, and no admin rights for
core commands. Windows firewall/Defender commands and Defender exclusion require
elevation.

## 2. Linux

### 2.1 Scripted install (systemd)

```bash
scripts/build.sh linux
sudo scripts/install-linux.sh dist/dc2-linux-amd64 .env
systemctl status dc2
```

What the script does:

- copies the binary to `/usr/local/bin/dc2-client` (`0755`),
- copies `.env` to `/etc/dc2/.env` (`0600`, directory `0700`),
- writes `/etc/systemd/system/dc2.service`,
- enables and starts it, `Restart=on-failure`.

### 2.2 Manual install

```bash
sudo install -m 0755 dist/dc2-linux-amd64 /usr/local/bin/dc2-client
sudo install -d -m 0700 /etc/dc2
sudo install -m 0600 .env /etc/dc2/.env
cd /etc/dc2 && sudo /usr/local/bin/dc2-client
```

For a quick foreground run without root:

```bash
mkdir -p ~/.local/bin ~/.config/dc2
cp dist/dc2-linux-amd64 ~/.local/bin/dc2-client
cp .env ~/.config/dc2/.env
chmod 600 ~/.config/dc2/.env
cd ~/.config/dc2 && ~/.local/bin/dc2-client
```

### 2.3 Verify

```bash
systemctl status dc2
journalctl -u dc2 -n 50 --no-pager
```

Then check Discord: new category, `#hostname-username`, pinned banner, `/help`.
`/cmd command:"id"` returns a job ID and posts the output asynchronously;
`/jobs list` shows recent jobs; `/dev info` shows version and uptime.
Run `dc2-client --check` first to validate token, guild, permissions, audit
path, and ffmpeg without connecting.

## 3. Windows

Build on any OS:

```bash
EMBED_ENV=1 scripts/build.sh windows   # or omit EMBED_ENV and copy .env
```

Copy `dist/dc2-windows-amd64.exe` and `.env` to the target, then in an elevated
PowerShell:

```powershell
.\scripts\install-windows.ps1 -Binary .\dc2-windows-amd64.exe -EnvFile .env
```

The script copies to `C:\ProgramData\D-C2`, restricts `.env` to
Administrators/SYSTEM, and registers a logon task with highest privileges.

Manual run:

```powershell
.\dc2-windows-amd64.exe
```

Run elevated if you need `/win *` commands or the Defender exclusion.

## 4. macOS

```bash
scripts/build.sh darwin
chmod +x dist/dc2-darwin-arm64
mkdir -p ~/.config/dc2 && cp .env ~/.config/dc2/.env && chmod 600 ~/.config/dc2/.env
cd ~/.config/dc2 && ~/dist/dc2-darwin-arm64
```

For a launchd service, create `~/Library/LaunchAgents/local.dc2.plist` with
`ProgramArguments` pointing at the binary, `WorkingDirectory` pointing at the
config directory, and `RunAtLoad`/`KeepAlive` true. Load it with
`launchctl load -w`.

Media features on macOS are limited; command/filesystem/process work.

## 5. Configuration location

The client reads `.env` from its **current working directory**. The service
scripts set `WorkingDirectory` to the config directory, so keep the file next
to that directory rather than next to the binary.

Precedence: disk `.env` overrides process environment; embedded values fill
gaps only. See [`CONFIGURATION.md`](./CONFIGURATION.md).

## 6. Update

Linux:

```bash
sudo install -m 0755 dist/dc2-linux-amd64 /usr/local/bin/dc2-client
sudo systemctl restart dc2
```

Windows: stop the task, replace the exe, start it again. Keep the same `.env`.

## 7. Remove

Linux:

```bash
sudo systemctl disable --now dc2
sudo rm -f /etc/systemd/system/dc2.service /usr/local/bin/dc2-client
sudo rm -rf /etc/dc2
sudo systemctl daemon-reload
```

Windows (elevated):

```powershell
Unregister-ScheduledTask -TaskName "D-C2" -Confirm:$false
Remove-Item -Recurse -Force "$env:ProgramData\D-C2"
```

macOS: `launchctl unload -w ~/Library/LaunchAgents/local.dc2.plist` and remove
the plist, binary, and config directory.

Also revoke the bot token if the deployment is retired.

## 8. Troubleshooting

| Symptom | Check |
|---|---|
| Exits immediately | `.env` in the working directory, both required variables set. |
| `websocket: close 4014` | Enable **Message Content Intent** in the portal. |
| No channel created | Bot has `Manage Channels`; guild ID correct; bot is in the guild. |
| Commands missing | Wait for the background sync; one OS per guild; re-run `/help`. |
| `/win *` says unsupported | Client is not running on Windows. |
| Permission denied writing files | Service user cannot access the path; adjust ownership or run as a user with access. |
| No ffmpeg | Install ffmpeg or bundle it; auto-install only runs as root. |
| Discord shows "The application did not respond" | The client process is not running, or the command was used in a different channel. Start the client and use its own `#hostname-username` channel. |
