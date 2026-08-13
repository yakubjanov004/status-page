#!/usr/bin/env bash
# service-notify.sh — Send a webhook notification for systemd service state changes.
#
# Usage: service-notify.sh <up|down> <unit-name>
# Example: service-notify.sh up alfaconnect-bot.service
#
# Environment variables (loaded from /etc/default/service-notify if present):
#   WEBHOOK_URL   — URL of the webhook endpoint (default: http://127.0.0.1:8080/api/v1/webhook)
#   HOOK_TOKEN    — Authentication token for X-Hook-Token header (required)
#   HMAC_SECRET   — Optional. If set, adds X-Hub-Signature-256 HMAC-SHA256 header.
#
# This script is non-blocking (timeout 5s) and always exits 0.

# Load credentials file if it exists
if [ -f /etc/default/service-notify ]; then
    # shellcheck source=/dev/null
    . /etc/default/service-notify
fi

ACTION="${1:-}"
UNIT_NAME="${2:-}"

if [ -z "$ACTION" ] || [ -z "$UNIT_NAME" ]; then
    echo "Usage: service-notify.sh <up|down> <unit-name>"
    exit 0
fi

WEBHOOK_URL="${WEBHOOK_URL:-http://127.0.0.1:8080/api/v1/webhook}"
HOOK_TOKEN="${HOOK_TOKEN:-}"
HMAC_SECRET="${HMAC_SECRET:-}"

if [ -z "$HOOK_TOKEN" ]; then
    echo "WARNING: HOOK_TOKEN not set, cannot send webhook"
    exit 0
fi

# Map systemd unit name to human-readable service name
map_service_name() {
    local unit="$1"
    unit="${unit%.service}"
    case "$unit" in
        alfaconnect-*|alfaconnect) echo "AlfaConnect" ;;
        mehmonxona*)               echo "Mehmonxona"  ;;
        odimrepo-*|odimrepo)       echo "Odimrepo"    ;;
        tokpoint-*|tokpoint)       echo "Tokpoint"    ;;
        datan*)                    echo "Datan"        ;;
        # Capitalize first letter as fallback
        *)                         echo "${unit^}"    ;;
    esac
}

SERVICE_NAME=$(map_service_name "$UNIT_NAME")
TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build compact JSON payload (no heredoc — CRLF safe)
PAYLOAD=$(printf '{"service":"%s","action":"%s","time":"%s","meta":{"unit":"%s","source":"systemd","hostname":"%s"}}' \
    "$SERVICE_NAME" "$ACTION" "$TIMESTAMP" "$UNIT_NAME" "$(hostname)")

# Build curl arguments
CURL_ARGS="-s -m 5 -X POST"
CURL_ARGS="$CURL_ARGS -H Content-Type:application/json"
CURL_ARGS="$CURL_ARGS -H X-Hook-Token:${HOOK_TOKEN}"

# Add HMAC-SHA256 signature if HMAC_SECRET is set
if [ -n "$HMAC_SECRET" ]; then
    SIG=$(printf '%s' "$PAYLOAD" | openssl dgst -sha256 -hmac "$HMAC_SECRET" 2>/dev/null \
        | awk '{print "sha256="$2}')
    if [ -n "$SIG" ]; then
        CURL_ARGS="$CURL_ARGS -H X-Hub-Signature-256:${SIG}"
    fi
fi

# Send webhook (non-blocking, timeout 5s, always exit 0)
curl $CURL_ARGS -d "$PAYLOAD" "${WEBHOOK_URL}" >/dev/null 2>&1 || true

exit 0
