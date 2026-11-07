#!/usr/bin/env bash
# Install a built client on a Linux target and run it as a systemd service.
# Run as root on a host you own.
#
# Usage:
#   sudo scripts/install-linux.sh dist/dc2-linux-amd64 [path/to/.env]
set -euo pipefail

if [[ $# -lt 1 ]]; then
  echo "usage: $0 <binary> [env-file]" >&2
  exit 2
fi

BIN="$1"
ENV_FILE="${2:-.env}"
PREFIX="${PREFIX:-/usr/local/bin}"
CONF_DIR="${CONF_DIR:-/etc/dc2}"
SERVICE="${SERVICE:-dc2}"
UNIT="/etc/systemd/system/${SERVICE}.service"

[[ -f "$BIN" ]] || { echo "binary not found: $BIN" >&2; exit 1; }
[[ -f "$ENV_FILE" ]] || { echo "env file not found: $ENV_FILE" >&2; exit 1; }
[[ "$(id -u)" -eq 0 ]] || { echo "run as root" >&2; exit 1; }

install -m 0755 "$BIN" "${PREFIX}/dc2-client"
install -d -m 0700 "$CONF_DIR"
install -m 0600 "$ENV_FILE" "${CONF_DIR}/.env"

cat > "$UNIT" <<EOF
[Unit]
Description=D-C2 client
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=${CONF_DIR}
ExecStart=${PREFIX}/dc2-client
Restart=on-failure
RestartSec=10
# The client needs to run commands and touch files; add hardening only if your
# use case tolerates it, for example:
# NoNewPrivileges=true
# PrivateTmp=true

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable --now "$SERVICE"

echo "installed ${SERVICE}:"
systemctl --no-pager --lines=10 status "$SERVICE" || true
