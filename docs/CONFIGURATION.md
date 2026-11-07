# Configuration

All runtime configuration is read from environment variables, optionally loaded
from a `.env` file.

## 1. Files

| Path | Purpose | Committed? |
|---|---|---|
| `.env` | Real credentials and runtime settings | **No** (`.gitignore`) |
| `.env.example` | Template with placeholder values | Yes |
| `env_embed.go` | Embeds `.env` when built with `-tags envembed` | Yes |
| `env_embed_stub.go` | No-op stub for default builds | Yes |

Create your local file:

```bash
cp .env.example .env
chmod 600 .env
$EDITOR .env
```

## 2. Variables

| Variable | Required | Default | Meaning |
|---|---|---|---|
| `DISCORD_BOT_TOKEN` | yes | — | Bot token from the Discord Developer Portal. Do **not** include the `Bot ` prefix. |
| `DISCORD_GUILD_ID` | yes | — | Numeric ID of the guild (server) that hosts device channels. |
| `DISCORD_ALLOWED_USERS` | no | empty | Comma-separated Discord user IDs allowed to control devices. Empty means any member who can post in the device channel. Get an ID with Developer Mode → right-click user → **Copy User ID**. |
| `DC2_CHANNEL_NAME` | lab only | `dc2-target-operator` | Channel the lab attacker script looks for. Not read by the client. |

The client itself only reads the first three.

## 3. Precedence

1. Embedded payload (only when built with `-tags envembed`): applied first, and
   only for keys not already present in the process environment.
2. `.env` file in the working directory: applied next and **overwrites** values.
3. Process environment: used for keys that neither source set.

Because the on-disk file wins over the environment, do not ship a stale `.env`
next to a binary that is configured through container/systemd environment
variables. Either remove the file or use `envembed`.

The lab script works around this by mounting a temporary `.env` with
`DISCORD_ALLOWED_USERS` cleared, so the webhook-based attacker is not rejected
by the operator allowlist.

## 4. Getting the values

1. Create the application at <https://discord.com/developers/applications>.
2. **Bot** → **Reset Token** → copy the token.
3. **Bot** → **Privileged Gateway Intents** → enable **Message Content Intent**.
   Without it the emoji/text shortcuts receive empty message content.
4. **OAuth2 → URL Generator** → scopes `bot`, `applications.commands`; permissions
   `Manage Channels`, `Manage Webhooks`, `Send Messages`, `Attach Files`,
   `Read Message History`, `Add Reactions`.
5. Invite the bot and copy the guild ID (Discord → right-click server → **Copy
   Server ID** with Developer Mode enabled).

Verify the token and list guilds without printing the token:

```bash
set -a; . ./.env; set +a
curl -s -H "Authorization: Bot $DISCORD_BOT_TOKEN" \
  https://discord.com/api/v10/users/@me | jq '{id,username}'
curl -s -H "Authorization: Bot $DISCORD_BOT_TOKEN" \
  https://discord.com/api/v10/users/@me/guilds | jq '.[] | {id,name}'
```

## 5. Where keys are stored

- Local development: `.env` at the repository root, mode `600`, never committed.
- CI/CD: the platform's secret store (GitHub Actions secrets, GitLab masked
  variables, etc.). Write them to `.env` at job time or inject them as
  environment variables.
- Containers: mount `.env` read-only, or pass environment variables with
  `--env-file`. The lab mounts `.env` read-only into both containers.
- Single-file builds: `-tags envembed` bakes `.env` into the binary. Treat that
  binary as a secret; rotating the token requires a rebuild.

Never put tokens in `README.md`, docs, issue trackers, or chat.

## 6. Rotating a leaked token

1. Discord Developer Portal → **Bot** → **Reset Token**.
2. Update `.env` (and any secret store / embedded build).
3. Restart every client using the old token.
4. Check audit logs for unexpected actions during the exposure window.

## 7. Build tags

| Tag | Effect |
|---|---|
| *(none)* | Reads `.env` from disk at startup. |
| `envembed` | Embeds `.env` into the binary at build time. Requires the file to exist. |
| `ffmpegembed` | Embeds `internal/media/ffmpeg/windows/ffmpeg.exe` (Windows builds). |

```bash
go build -o dc2-client .                                  # disk .env
go build -tags envembed -o dc2-client .                   # embedded .env
go build -tags "envembed ffmpegembed" -o dc2.exe .        # both
```
