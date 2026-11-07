# Purple-team validation

Goal: prove that the blue team can see this C2, not that the C2 can hide.

> Run this only on hosts and networks you own or are authorized to test. Share
> the test window and source addresses with the deconfliction contact.

## 1. Setup

- One lab host running the client (VM or the Docker lab in [`LAB.md`](./LAB.md)).
- One collection point with:
  - Sysmon (EID 1 process create, 3 network, 11 file create, 13 registry),
  - Defender/EDR telemetry or an equivalent agent,
  - proxy/DNS logs,
  - a SIEM to run the queries from [`../detections/IOCS.md`](../detections/IOCS.md).
- The YARA and Sigma rules from [`../detections/`](../detections/) loaded in
  alert-only mode.

Do **not** disable the security stack for this test.

## 2. Baseline

1. Record 30 minutes of normal activity (no client).
2. Confirm there are no alerts for Discord traffic from non-browser processes.

## 3. Test cases

| # | Action | Expected telemetry | Expected detection |
|---|---|---|---|
| 1 | Start client | Process create, TLS to `discord.com` / `gateway.discord.gg` | Non-browser Discord API rule |
| 2 | Client creates category/channel | Discord audit log, bot API calls | Discord-side anomaly (bot creating channels) |
| 3 | `/cmd command:"whoami"` | Child `cmd.exe`/`bash`, audit JSONL | Shell-spawn rule |
| 4 | `/fs download` a test file | File read, attachment upload to CDN | Data transfer correlation |
| 5 | `/fs upload` a marker file | File write, CDN download | File create + network |
| 6 | `/media screenshot` | `ffmpeg`/capture activity, attachment upload | Media/child-process heuristic |
| 7 | Kill network for 30 s | Gateway disconnect/reconnect, `🔄 reconnected` | Reconnect behavior |
| 8 | `/dev info` | Version string, uptime | Asset inventory match |
| 9 | Stop client and remove files | Process exit, file deletions | Cleanup verification |

## 4. Measure

- Did the rule fire? (yes/no)
- Time from action to alert (MTTD).
- False positives in the 30-minute baseline.
- Gaps: which action produced no telemetry.

## 5. Tune

- Add environment-specific allowlists only for known-good tools (for example a
  CI agent that legitimately posts to Discord webhooks).
- Do not broaden a rule until it stops catching the test cases.
- Record every tuning change with the test case that justified it.

## 6. Report

Include:

- test window and scope;
- actions performed and timestamps (UTC);
- detections fired, with rule IDs and raw event references;
- detections missed and why (no telemetry, logic gap, tuning);
- recommended detection improvements;
- deconfliction statement: this activity was authorized and part of the test.

## 7. Cleanup

- Stop the client, revoke the bot token, remove the guild/channels.
- Remove test files, audit logs, scheduled tasks/units.
- Restore the security stack to its pre-test state and verify.
