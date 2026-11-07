# Security notes

> Operational guidance for running this client on an authorized engagement is in
> [`OPSEC.md`](./OPSEC.md).

## 1. Threat model

The client is a remote-control agent. Anyone who can post in the device channel
can execute commands on the host. The primary assets are:

- the Discord bot token (full control of the guild),
- the host the client runs on,
- files transferred through the channel.

## 2. Controls in place

| Control | Where |
|---|---|
| Optional user allowlist (`DISCORD_ALLOWED_USERS`) | `internal/config`, `internal/bot`, `internal/commands` |
| Channel scoping (only the device channel is accepted) | `internal/bot/handler.go`, `internal/commands/router.go` |
| Ignore the bot's own messages | `internal/bot/handler.go` |
| `.env` gitignored and mode `600` | `.gitignore`, setup docs |
| Token never logged | `internal/config`, startup logs |
| HTTP timeouts for attachment transfer | `internal/httpx` |
| Local JSONL audit trail (job, operator, command, exit, duration) | `internal/audit`, `internal/jobs` |
| Extracted ffmpeg dir `0700`, binary `0700` | `internal/media/media.go` |
| Unprivileged, capability-dropped lab container | `deploy/Dockerfile.target`, `lab/run.sh` |
| Dependency upgrades (no known reachable CVEs) | `go.mod` |

## 3. Audit results (2026-11-07)

Tools:

- `staticcheck` (latest) — **clean**
- `gosec` v2.30.0 — 131 findings, all triaged
- `govulncheck` v1.9.0 — **no reachable vulnerabilities**

Fixed during the audit:

- `GO-2026-6278` (weak PRNG in `gorilla/websocket`) — upgraded to v1.5.3.
- Upgraded `discordgo` v0.26.1 → v0.29.0 and `kbinani/screenshot` to the
  2025-06 release (also restores the `darwin` build).
- `.env` embedding made opt-in so builds do not fail when the file is absent.
- Embedded ffmpeg directory permissions reduced from `0755` to `0700`.
- Attachment downloads moved to an `http.Client` with a 5-minute timeout.
- Added `DISCORD_ALLOWED_USERS` so control can be restricted to known operators.

Accepted gosec findings (by design for a C2):

| Rule | Count | Why accepted |
|---|---|---|
| `G702` (taint → exec) | 5 | The product executes operator-supplied commands; there is no fixed command set. |
| `G204` (variable subprocess) | 11 | Command and ffmpeg arguments are operator-supplied by design. |
| `G304` (variable file path) | 10 | File transfer and recording paths are operator-supplied by design. |
| `G104` (unhandled error) | 98 | Mostly Discord API send/reaction calls where failure is non-actionable. |
| `G706` (log injection) | 7 | Log lines may contain operator input; logs stay local unless `/log on`. |

These are documented rather than suppressed so a future review can re-evaluate
them.

## 4. Residual risks

- **Default is open.** With `DISCORD_ALLOWED_USERS` empty, any guild member who
  can post in the device channel can control the host. Set the allowlist in any
  shared guild.
- **No command confirmation.** Destructive commands execute immediately.
- **8 MB transfer cap** with no resumable/chunked transfer.
- **Audit log contains operator input.** It is written mode `0600`; protect or
  rotate it if commands may contain sensitive values.
- **Discord is the control plane.** Compromise of the bot token implies
  compromise of every connected host.
- **Windows security commands** require an elevated client.

## 5. Hardening checklist

- [ ] Rotate the bot token if it was ever pasted into chat, an issue, or a
      terminal recording.
- [ ] Set `DISCORD_ALLOWED_USERS` to your user ID.
- [ ] Keep the bot in a dedicated guild with no other members.
- [ ] Run the client as an unprivileged user; drop capabilities in containers.
- [ ] Keep `.env` at mode `600`; never commit it.
- [ ] Prefer `-tags envembed` only for single-file builds you control.
- [ ] Leave `/log off` outside troubleshooting.
- [ ] Re-run `govulncheck`, `staticcheck`, and `gosec` before releases.
