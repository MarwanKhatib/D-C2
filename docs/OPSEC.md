# OPSEC for D‑C2

> This guide is for authorized lab work and contracted engagements. Use only on
> systems you own or have written permission to test.

D‑C2 is a Discord-based C2. There is no self-hosted team server to protect, but
the bot token is a bearer credential for every device, and Discord sees all
control traffic. OPSEC here is mostly **authorization, credential hygiene, and
data discipline** — not stealth. Assume the target, its EDR, and its proxy logs
can see what runs and where it connects.

## 1. Operating boundary (before anything runs)

Record and get sign-off on:

- authorized hosts, accounts, and networks; explicit exclusions;
- start/end times and time zone;
- permitted actions (read vs write vs destructive) and prohibited ones;
- data classes that may be collected, retention, and deletion;
- emergency stop phrase and 24/7 contacts;
- who holds the mapping from bot/guild to operator (deconfliction contact);
- teardown owner and date.

Stop conditions — pause and call the engagement lead if you:

- reach an out-of-scope host or tenant;
- encounter live sensitive data (PII, credentials, customer records);
- cause an unexpected service change or outage;
- see signs of a real incident (not your traffic);
- lose the agreed comms channel.

## 2. Operator OPSEC (while working)

### Identity and accounts

- Use a **dedicated Discord application and bot** for the engagement. Never a
  personal account or a bot reused across clients.
- Enable MFA on the Discord account that owns the application.
- Keep the token in a secret manager or `.env` mode `600`; never in shell
  history, screenshots, tickets, chat, or docs.
- Rotate the token at engagement start and immediately after any exposure.
  Resetting the token kills every deployed client until they are updated — plan
  for it.
- Do not reuse the guild, bot, or channel names across engagements.

### Infrastructure separation

- Dedicated guild with **only** the bot and the operator(s). No public invite.
- Set `DISCORD_ALLOWED_USERS` to the operator user IDs. An empty allowlist means
  anyone who can post in the device channel can run commands.
- Use a dedicated VM/container for the operator workstation; do not mix
  engagement credentials with personal sessions.
- If the engagement requires source-address control, route operator traffic
  through the approved egress and record those addresses in the RoE annex.

### Communications and evidence

- Discuss engagement details only on approved encrypted channels.
- Keep the audit log (`DC2_AUDIT_LOG`) with the engagement record. It contains
  commands, operator IDs, exit codes, and durations.
- Store evidence encrypted, restrict access, and delete on the agreed schedule.
- Prefer a test marker or access check over downloading real data.

### Pre-task check (log the answer)

1. Is this target in scope right now?
2. Which identity and source system am I using?
3. Is this a read, a write, an auth attempt, or remote execution?
4. What will it contact, change, or collect?
5. What evidence proves the result, and how do I stop it?

## 3. Target OPSEC (while the client runs)

### Host and process

- Run as an **unprivileged dedicated user**, not root/Administrator, unless the
  approved objective requires elevation (Windows firewall/Defender actions do).
- Use a dedicated VM or container. The lab uses `--cap-drop ALL` and
  `no-new-privileges`; keep that posture.
- Do not mount the Docker socket or host home directories into the container.
- Keep `DC2_MAX_JOBS`, `DC2_MAX_OUTPUT`, and `DC2_JOB_TIMEOUT` low enough that a
  mistake cannot exhaust the host.
- Run `/sys log off` outside troubleshooting. Log streaming posts raw client
  logs into the channel, including paths and errors.

### Files and data

- Transfer only what the objective needs. The 8 MB cap is a feature, not a
  limitation to work around casually.
- Remember that anything posted to Discord leaves the host and is stored by a
  third party. Treat channel history as external storage.
- Clean up staging paths and downloaded files at the end of each task.
- The audit log is local and `0600`; protect it like evidence.

### Build and artifact hygiene

- Do not bake the token into an artifact (`-tags envembed`) unless you control
  every copy and can rotate the token afterwards.
- Hash every artifact (`dist/SHA256SUMS`) and verify before execution.
- Keep builds private; do not upload engagement artifacts to public multi-scanner
  services.
- Build per engagement; do not reuse the same binary hash across clients.

## 4. What defenders can see (assume this)

Discord C2 is well known to blue teams. Expect:

- **Network:** endpoints talking to `discord.com` / `discordapp.com` API and
  gateway endpoints from a process that has no business doing so; webhook URLs
  in command lines or memory.
- **Process:** an unsigned binary spawning `cmd.exe`, `bash`, `powershell`, or
  ffmpeg; child processes of the client visible in EDR telemetry.
- **Host artifacts:** `.env` with a Discord token format, a JSONL audit log,
  staged files in temp directories, systemd units or scheduled tasks.
- **Discord side:** a new bot, guild, category, and channels; DMs and messages
  are visible to Discord and to anyone with the token.
- **Static:** Discord bot tokens match a known pattern and are extractable from
  binaries and configs.

This is why OPSEC here means **scope discipline and cleanup**, not hiding. Do not
disable logging, EDR, or audit controls, and do not attempt anti-forensics,
unless the RoE explicitly authorizes it and the client has approved the risk.

## 5. Abort and teardown

Kill switch order:

1. Stop the client service (Linux `systemctl stop dc2`; Windows
   `Unregister-ScheduledTask D-C2` / stop the process).
2. Revoke the bot token in the Discord Developer Portal — this invalidates every
   deployed copy at once.
3. Remove the guild or kick the bot from it.
4. Remove binaries, `.env`, audit logs, systemd units, scheduled tasks, and
   staged files from every target.
5. Export and hash the audit/evidence logs before deletion if the RoE requires
   retention.
6. Record the teardown (who, when, what) and confirm nothing is left running.

See [`INSTALL.md`](./INSTALL.md) §7 for per-OS removal commands.

## 6. Quick checklist

**Before**
- [ ] Signed RoE with scope, times, prohibited actions, stop phrase.
- [ ] Dedicated bot/app/guild; MFA; token in secret storage.
- [ ] `DISCORD_ALLOWED_USERS` set; guild locked down.
- [ ] Artifact built, hashed, verified; no token embedded.
- [ ] Teardown plan and kill switch rehearsed.

**During**
- [ ] Every action logged with operator, target, action, result.
- [ ] Least privilege; no destructive commands without separate approval.
- [ ] Log streaming off; job limits configured.
- [ ] No real sensitive data collected beyond the minimum.
- [ ] Stop conditions honored.

**After**
- [ ] Client stopped, token revoked, guild cleaned.
- [ ] Files/audit/logs removed or archived per RoE.
- [ ] Teardown recorded; deconfliction contact informed.
- [ ] Retro: what was noisy, what would you change next time.

## 7. Sources

- Red Team OPSEC handbook (scope, stop conditions, accountability, data handling) —
  <https://01xjb.github.io/handbook/operations/operational-security/>
- Authorized red-team infrastructure (ROE annex, deconfliction, teardown) —
  <https://hacktricks.wiki/en/privacy/authorized-red-team-infrastructure.html>
- Cobalt RoE checklist — <https://www.cobalt.io/learning-center/red-team-rules-of-engagement-comprehensive-checklist>
- OPSEC process for red teams — <https://forgeboundresearch.com/hacking-tutorials/opsec-for-red-teamers-ctf-guide/>
- Discord/IM C2 detection (what defenders look for) —
  <https://www.dragos.com/blog/how-to-detect-adversarial-c2-techniques-in-instant-messaging>
- Cloud messaging C2 detection and token YARA — <https://threathive.ai/blog/detecting-c2-over-slack-discord-telegram-github/>
- Discord token format reference — <https://discord.com/developers/docs/reference#authentication>
