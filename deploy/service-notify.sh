#!/usr/bin/env bash
# service-notify.sh — Send a webhook notification for systemd service state changes.
#
# Usage: service-notify.sh <up|down> <unit-name>
# Example: service-notify.sh up alfaconnect-bot.service
#
# Environment variables:
#   WEBHOOK_URL  — URL of the webhook endpoint (default: http://127.0.0.1:8080/api/v1/webhook)
#   HOOK_TOKEN   — Authentication token for X-Hook-Token header (required)
#
# This script is non-blocking (timeout 5s) and always exits 0.

set -euo pipefail

ACTION="${1:-}"
UNIT_NAME="${2:-}"

if [ -z "$ACTION" ] || [ -z "$UNIT_NAME" ]; then
    echo "Usage: service-notify.sh <up|down> <unit-name>"
    exit 0
fi

WEBHOOK_URL="${WEBHOOK_URL:-http://127.0.0.1:8080/api/v1/webhook}"
HOOK_TOKEN="${HOOK_TOKEN:-}"

if [ -z "$HOOK_TOKEN" ]; then
    echo "WARNING: HOOK_TOKEN not set, cannot send webhook"
    exit 0
fi

# Map systemd unit name to human-readable service name
map_service_name() {
    local unit="$1"
    # Remove .service suffix
    unit="${unit%.service}"

    case "$unit" in
        alfaconnect-*|alfaconnect)
            echo "AlfaConnect"
            ;;
        mehmonxona*)
            echo "Mehmonxona"
            ;;
        odimrepo-*|odimrepo)
            echo "Odimrepo"
            ;;
        tokpoint-*|tokpoint)
            echo "Tokpoint"
            ;;
        datan*)
            echo "Datan"
            ;;
        *)
            # Capitalize first letter as fallback
            echo "${unit^}"
            ;;
    esac
}

SERVICE_NAME=$(map_service_name "$UNIT_NAME")
TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build JSON payload
PAYLOAD=$(cat <<EOF
{
  "service": "${SERVICE_NAME}",
  "action": "${ACTION}",
  "time": "${TIMESTAMP}",
  "meta": {
    "unit": "${UNIT_NAME}",
    "source": "systemd",
    "hostname": "$(hostname)"
  }
}
EOF
)

# Send webhook (non-blocking, timeout 5s, always exit 0)
curl -s -m 5 \
    -X POST "${WEBHOOK_URL}" \
    -H "Content-Type: application/json" \
    -H "X-Hook-Token: ${HOOK_TOKEN}" \
    -d "${PAYLOAD}" \
    >/dev/null 2>&1 || true

exit 0
