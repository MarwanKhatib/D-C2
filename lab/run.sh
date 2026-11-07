#!/usr/bin/env bash
# Build the client and run the two-container lab: one target, one attacker.
# Usage: [GO=/path/to/go] lab/run.sh ["command" ...]
set -euo pipefail
cd "$(dirname "$0")/.."

GO="${GO:-go}"
mkdir -p bin

# The attacker posts as a webhook, so the operator allowlist would reject it.
# Use a throwaway copy of .env with the allowlist cleared for the lab.
LAB_ENV="$(mktemp)"
sed 's/^DISCORD_ALLOWED_USERS=.*/DISCORD_ALLOWED_USERS=/' .env > "$LAB_ENV"
chmod 600 "$LAB_ENV"

echo "[lab] building static client"
CGO_ENABLED=0 "$GO" build -trimpath -ldflags "-s -w" -o bin/dc2-linux .

echo "[lab] building images"
docker build -q -f deploy/Dockerfile.target -t dc2-target:lab .
docker build -q -f deploy/Dockerfile.attacker -t dc2-attacker:lab .

echo "[lab] preparing network"
docker network inspect dc2-lab >/dev/null 2>&1 || docker network create dc2-lab >/dev/null

docker rm -f dc2-target >/dev/null 2>&1 || true
trap 'rm -f "$LAB_ENV"; docker rm -f dc2-target >/dev/null 2>&1 || true' EXIT

echo "[lab] starting target"
docker run -d --name dc2-target \
  --hostname dc2-target \
  --network dc2-lab \
  --cap-drop ALL \
  --security-opt no-new-privileges \
  -v "$LAB_ENV:/app/.env:ro" \
  dc2-target:lab >/dev/null

sleep 3
docker logs --tail 20 dc2-target || true

echo "[lab] starting attacker"
docker run --rm --name dc2-attacker \
  --network dc2-lab \
  -v "$LAB_ENV:/app/.env:ro" \
  dc2-attacker:lab "$@"
