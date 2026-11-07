# Server and control plane

## 1. There is no separate C2 server

D‑C2 has no backend process you host. The control plane is **Discord**:

```text
operator (Discord app / web / mobile)
        │  slash commands, emoji, buttons
        ▼
Discord bot application + guild  ◄── HTTPS/WSS ──►  client on each target
                                                     (one process per device)
```

The bot application and the guild are the "server". Each target runs one client
process that connects outbound to Discord and creates its own device channel.

If you want a durable control box, run the client on a host you own and use it
as a jump box. There is nothing else to install server-side.

## 2. Create the Discord application

1. Open <https://discord.com/developers/applications> → **New Application**.
2. **Bot** → **Reset Token** → copy the token once. Do not paste it into docs.
3. **Bot** → **Privileged Gateway Intents**:
   - Enable **Message Content Intent** (required for `🏃‍♂️ <command>` and
     `help`; slash commands work without it).
   - **Server Members Intent** is not required.
4. **OAuth2 → URL Generator**:
   - Scopes: `bot`, `applications.commands`
   - Permissions: `View Channels`, `Send Messages`, `Attach Files`,
     `Read Message History`, `Add Reactions`, `Manage Channels`,
     `Manage Webhooks`
5. Open the generated URL and invite the bot to a guild you own.

Equivalent invite URL (replace `APP_ID`):

```text
https://discord.com/api/oauth2/authorize?client_id=APP_ID&permissions=536972368&scope=bot%20applications.commands
```

## 3. Configure the guild

1. Enable **Developer Mode** in Discord settings.
2. Right-click your server → **Copy Server ID**.
3. Optionally copy your own user ID (right-click yourself → **Copy User ID**).
4. Put both in `.env` next to the binary (see
   [`CONFIGURATION.md`](./CONFIGURATION.md)):

```env
DISCORD_BOT_TOKEN=...
DISCORD_GUILD_ID=...
DISCORD_ALLOWED_USERS=...
```

## 4. Start a client

```bash
scripts/build.sh linux            # produces dist/dc2-linux-amd64 + SHA256SUMS
./dist/dc2-linux-amd64            # reads .env from the current directory
```

On startup the client:

1. Opens the gateway connection.
2. Finds/creates the OS category (`windows`, `linux`, `mac`, `other`).
3. Finds/creates `#<hostname>-<username>` under that category.
4. Enables log streaming for that channel.
5. Posts and pins a **New device connected** banner.
6. Syncs the grouped slash commands (bulk overwrite) in the background.

## 5. Verify

- The guild shows the new category and device channel.
- The pinned banner lists session, IP, user, hostname, OS, arch, CWD, and
  firewall/Defender state.
- `/help` responds ephemerally and lists the groups valid for that OS.
- `/cmd command:"id"` returns command output.

## 6. Run it as a service

`scripts/install-linux.sh` installs the binary and a systemd unit:

```bash
sudo scripts/install-linux.sh dist/dc2-linux-amd64 .env
systemctl status dc2
journalctl -u dc2 -f
```

Windows: `scripts/install-windows.ps1` (elevated PowerShell) copies the binary,
restricts `.env` ACLs, and registers a logon task.

Keep the unit simple. The client must be able to run commands and touch files;
aggressive systemd hardening (`ProtectHome`, `ProtectSystem=strict`) will break
filesystem and media commands.

## 7. Operating notes

- **One OS per guild** for clean command visibility. Discord scopes slash
  commands to the guild, not to a channel; the `win` group is registered only by
  a Windows client.
- **Set `DISCORD_ALLOWED_USERS`.** With it empty, any guild member who can post
  in the device channel can run commands on the host.
- **Token = full control of the guild.** Store it in `.env` mode `600`, in a
  secret manager, or embed it with `-tags envembed` only for artifacts you
  control. Rotate immediately if it leaks. See [`OPSEC.md`](./OPSEC.md).
- **Log streaming** (`/sys log on`) is useful for troubleshooting and noisy for
  long-running operation.
- **Multiple operators**: add their user IDs to `DISCORD_ALLOWED_USERS`
  (comma-separated).
