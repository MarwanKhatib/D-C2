# Two-container lab

The lab runs one **target** container (the D-C2 client) and one **attacker**
container (an operator simulator). Both are Debian `bookworm-slim`.

```text
┌──────────────────── docker network: dc2-lab ────────────────────┐
│                                                                 │
│  dc2-target (debian)               dc2-attacker (debian)        │
│  /app/dc2-client                   attacker.sh                  │
│  user: operator                    curl + jq                    │
│  mounted: .env (ro)                mounted: .env (ro)           │
│         │                                  │                    │
└─────────┼──────────────────────────────────┼────────────────────┘
          │                                  │
          └──────────► discord.com ◄─────────┘
                 gateway + REST API
```

The attacker does not need a second bot: it creates a **webhook** in the device
channel and posts messages through it. The client only ignores messages authored
by its own bot user, so webhook messages are accepted.

## 1. Host prerequisites

- Linux host with Docker Engine.
- Go 1.21+ (only to build the binary).
- A Discord bot that is a member of your guild.

Debian/Ubuntu installation:

```bash
sudo apt-get update
sudo apt-get install -y docker.io
sudo systemctl enable --now docker
sudo usermod -aG docker "$USER"
newgrp docker          # or log out and back in
docker info
```

## 2. Configure credentials

```bash
cp .env.example .env
chmod 600 .env
$EDITOR .env
```

Required:

```env
DISCORD_BOT_TOKEN=...
DISCORD_GUILD_ID=...
DISCORD_ALLOWED_USERS=
```

Leave `DISCORD_ALLOWED_USERS` set for normal use: only that user may control
devices. The lab script copies `.env` to a temporary file and clears the
allowlist, because the attacker posts as a channel webhook rather than as your
Discord user.

The bot needs the **Message Content Intent** enabled in the Developer Portal,
otherwise `🏃‍♂️ <command>` messages arrive with empty content.

## 3. Run the lab

```bash
GO=$(command -v go) ./lab/run.sh
```

Custom commands:

```bash
GO=$(command -v go) ./lab/run.sh "id" "uname -a" "ls -la /"
```

What the script does:

1. Builds a static `bin/dc2-linux`.
2. Builds `dc2-target:lab` and `dc2-attacker:lab`.
3. Creates the `dc2-lab` bridge network.
4. Starts `dc2-target` as unprivileged user `operator`, with `--cap-drop ALL`
   and `no-new-privileges`, mounting a temporary copy of `.env` (allowlist
   cleared) read-only.
5. Runs `dc2-attacker`, which waits for `#dc2-target-operator`, creates a
   webhook, posts each command, and prints the client's replies.
6. Removes the target container on exit.

Expected output:

```text
[attacker] channel id: 1568439067085906083
----- echo dc2-lab-ok -----
```bash
dc2-lab-ok
```
[OK]
----- hostname -----
```bash
dc2-target
```
[OK]
----- id -----
```bash
uid=1000(operator) gid=37(operator) groups=37(operator)
```
[OK]
[attacker] all commands succeeded
```

## 4. Manual control

Start only the target and drive it from Discord:

```bash
docker run -d --name dc2-target --hostname dc2-target \
  --network dc2-lab \
  -v "$PWD/.env:/app/.env:ro" dc2-target:lab
docker logs -f dc2-target
```

Then open the guild, find `#dc2-target-operator` under the `linux` category, and
use `/cmd`, `/fs ls`, `/media screenshot`, or the emoji shortcuts.

## 5. Inspect and stop

```bash
docker logs dc2-target
docker exec -it dc2-target bash        # inside the target, as operator
docker rm -f dc2-target
docker network rm dc2-lab
docker rmi dc2-target:lab dc2-attacker:lab
```

## 6. Troubleshooting

| Symptom | Cause | Fix |
|---|---|---|
| `websocket: close 4014` | Privileged intent not allowed | Enable **Message Content Intent** in the portal. |
| Channel not found | Client not connected, or hostname/user mismatch | Check `docker logs dc2-target`; set `DC2_CHANNEL_NAME` for the attacker. |
| Webhook `403` | Bot lacks `Manage Webhooks` | Grant the permission or use a channel where it has it. |
| No reply | Message content intent missing, or allowlist set | Enable the intent; leave `DISCORD_ALLOWED_USERS` empty. |
| `Cannot connect to the Docker daemon` | User not in `docker` group | `sudo usermod -aG docker $USER` then re-login. |
| Target exits immediately | Missing `.env` values | `docker logs dc2-target` shows the config error. |

## 7. Safety

- This lab is for a guild and hosts you own.
- The target runs unprivileged; do not add `--privileged` or mount the Docker
  socket.
- The bot token grants full control of the guild. Keep `.env` at mode `600` and
  out of source control.
