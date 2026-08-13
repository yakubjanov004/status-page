#!/usr/bin/env bash
# test-event.sh — Full smoke-test for the webhook receiver.
#
# Runs a complete down→wait→up cycle for one service,
# then verifies all API endpoints return correct data.
#
# Usage:
#   HOOK_TOKEN=secret ./deploy/test-event.sh [options]
#   bash deploy/test-event.sh -t secret -u http://127.0.0.1:8080
#
# Options:
#   -u URL    Webhook base URL  (default: http://127.0.0.1:8080)
#   -t TOKEN  X-Hook-Token      (default: $HOOK_TOKEN env var)
#   -s NAME   Service name      (default: AlfaConnect)
#   -w SECS   Wait between down/up (default: 5)
#   -q        Quiet — suppress curl body output, only print PASS/FAIL
#
# Exit codes:
#   0  All checks passed
#   1  One or more checks failed

# NOTE: set -e intentionally NOT used — we want to continue after
#       individual failures and print a full summary at the end.
set -uo pipefail

# ---------- defaults ----------
BASE_URL="${WEBHOOK_URL:-http://127.0.0.1:8080}"
# strip trailing /api/v1/webhook from WEBHOOK_URL if set that way
BASE_URL="${BASE_URL%/api/v1/webhook}"

HOOK_TOKEN="${HOOK_TOKEN:-}"
SERVICE="AlfaConnect"
WAIT_SECS=5
QUIET=false

# ---------- arg parsing ----------
while getopts "u:t:s:w:q" opt; do
    case $opt in
        u) BASE_URL="$OPTARG" ;;
        t) HOOK_TOKEN="$OPTARG" ;;
        s) SERVICE="$OPTARG" ;;
        w) WAIT_SECS="$OPTARG" ;;
        q) QUIET=true ;;
        *) echo "Unknown option: $opt"; exit 1 ;;
    esac
done

if [ -z "$HOOK_TOKEN" ]; then
    echo "ERROR: HOOK_TOKEN is required (set env var or use -t TOKEN)"
    exit 1
fi

API="${BASE_URL}/api/v1"
PASS=0
FAIL=0

# ---------- color helpers ----------
color_green() { printf '\033[32m%s\033[0m\n' "$*"; }
color_red()   { printf '\033[31m%s\033[0m\n' "$*"; }
color_yellow(){ printf '\033[33m%s\033[0m\n' "$*"; }
color_bold()  { printf '\033[1m%s\033[0m\n'  "$*"; }

# ---------- check helper ----------
# Never exits — always increments PASS or FAIL.
check() {
    local label="$1"
    local result="$2"
    local expected="$3"
    if echo "$result" | grep -q "$expected" 2>/dev/null; then
        PASS=$((PASS+1))
        $QUIET || color_green "  ✓ $label"
    else
        FAIL=$((FAIL+1))
        color_red "  ✗ $label"
        color_red "    expected to contain: $expected"
        color_red "    got: $(echo "$result" | head -3)"
    fi
}

# ---------- curl wrappers ----------
# Always return output; never exit on curl error (connection refused etc.)
do_curl() {
    local method="$1"; shift
    local url="$1"; shift
    curl -s -m 10 -X "$method" "$@" "$url" 2>/dev/null || echo "__CURL_FAILED__"
}

webhook_post() {
    local action="$1"
    local ts="$2"
    do_curl POST "${API}/webhook" \
        -H "Content-Type: application/json" \
        -H "X-Hook-Token: ${HOOK_TOKEN}" \
        -d "{\"service\":\"${SERVICE}\",\"action\":\"${action}\",\"time\":\"${ts}\",\"meta\":{\"source\":\"test-event.sh\"}}"
}

# ---------- wait-for-server ----------
wait_for_server() {
    local url="${BASE_URL}/healthz"
    local max_attempts=15
    local attempt=1
    color_yellow "  Waiting for server at ${BASE_URL} ..."
    while [ $attempt -le $max_attempts ]; do
        resp=$(curl -s -m 3 "$url" 2>/dev/null || true)
        if echo "$resp" | grep -q '"status":"ok"' 2>/dev/null; then
            color_green "  Server is up (attempt $attempt)"
            return 0
        fi
        printf "  attempt %d/%d: not ready yet, retrying in 2s...\n" "$attempt" "$max_attempts"
        sleep 2
        attempt=$((attempt+1))
    done
    color_red "  ERROR: Server did not become ready after $((max_attempts*2))s"
    color_red "  Check: journalctl -u status-webhook -n 30 --no-pager"
    color_red "  Check: systemctl status status-webhook"
    return 1
}

# ---------- header + server wait ----------
echo ""
color_bold "═══════════════════════════════════════════════"
color_bold "  Webhook Receiver Smoke Test"
color_bold "  Service : $SERVICE"
color_bold "  URL     : $BASE_URL"
color_bold "  Wait    : ${WAIT_SECS}s between down→up"
color_bold "═══════════════════════════════════════════════"
echo ""

if ! wait_for_server; then
    echo ""
    color_red "Cannot reach server. Aborting smoke test."
    color_yellow "Quick diagnostics:"
    color_yellow "  systemctl status status-webhook"
    color_yellow "  journalctl -u status-webhook -n 30 --no-pager"
    color_yellow "  ss -tlnp | grep 8080"
    exit 1
fi
echo ""

# ── 1. Health check ──────────────────────────────────────
echo "── 1. Health check"
resp=$(do_curl GET "${BASE_URL}/healthz")
check "GET /healthz returns ok" "$resp" '"status":"ok"'
$QUIET || echo "     $resp"
echo ""

# ── 2. Auth check ────────────────────────────────────────
echo "── 2. Auth check (wrong token → 401)"
http_code=$(curl -s -m 10 -o /dev/null -w "%{http_code}" \
    -X POST "${API}/webhook" \
    -H "Content-Type: application/json" \
    -H "X-Hook-Token: wrong-token" \
    -d '{"service":"AlfaConnect","action":"down","time":"2026-01-01T00:00:00Z"}' \
    2>/dev/null || echo "000")
check "Wrong token returns 401" "$http_code" "401"
echo ""

# ── 3. Unknown service → 400 ─────────────────────────────
echo "── 3. Unknown service name → 400"
resp_and_code=$(curl -s -m 10 -X POST "${API}/webhook" \
    -H "Content-Type: application/json" \
    -H "X-Hook-Token: ${HOOK_TOKEN}" \
    -d '{"service":"DOES_NOT_EXIST","action":"down","time":"2026-01-01T00:00:00Z"}' \
    -w "\n%{http_code}" 2>/dev/null || echo -e "\n000")
http_code=$(echo "$resp_and_code" | tail -1)
body=$(echo "$resp_and_code" | head -n -1)
check "Unknown service returns 400" "$http_code" "400"
check "Error body mentions service name" "$body" "DOES_NOT_EXIST"
$QUIET || echo "     $body"
echo ""

# ── 4. DOWN event ────────────────────────────────────────
DOWN_TS=$(date -u +%Y-%m-%dT%H:%M:%SZ)
echo "── 4. POST down event at $DOWN_TS"
resp=$(webhook_post "down" "$DOWN_TS")
check "down event: status ok"      "$resp" '"status":"ok"'
check "down event: has request_id" "$resp" '"request_id"'
$QUIET || echo "     $resp"

# Check X-Request-Id response header
req_id=$(curl -s -m 10 -D - -o /dev/null \
    -X POST "${API}/webhook" \
    -H "Content-Type: application/json" \
    -H "X-Hook-Token: ${HOOK_TOKEN}" \
    -d "{\"service\":\"${SERVICE}\",\"action\":\"down\",\"time\":\"${DOWN_TS}\"}" \
    2>/dev/null | grep -i "^x-request-id:" | tr -d '\r\n' | awk '{print $2}' || echo "")
check "X-Request-Id response header present" "$req_id" "wh-"
$QUIET || echo "     X-Request-Id: ${req_id:-<not found>}"
echo ""

# ── 5. Service list shows "down" ─────────────────────────
echo "── 5. GET /services — $SERVICE should be down"
resp=$(do_curl GET "${API}/services")
check "services list: contains $SERVICE" "$resp" "\"$SERVICE\""
check "services list: last_status=down"  "$resp" '"last_status":"down"'
$QUIET || echo "     $resp"
echo ""

# ── 6. Incidents list shows open incident ────────────────
echo "── 6. GET /services/$SERVICE/incidents — should have open incident"
resp=$(do_curl GET "${API}/services/${SERVICE}/incidents?limit=5&offset=0")
check "incidents: has total field"  "$resp" '"total":'
check "incidents: status open"      "$resp" '"status":"open"'
$QUIET || echo "     $resp"
echo ""

# ── 7. Wait, then send UP ────────────────────────────────
echo "── 7. Waiting ${WAIT_SECS}s, then sending UP event..."
sleep "$WAIT_SECS"

UP_TS=$(date -u +%Y-%m-%dT%H:%M:%SZ)
echo "     POST up event at $UP_TS"
resp=$(webhook_post "up" "$UP_TS")
check "up event: status ok" "$resp" '"status":"ok"'
$QUIET || echo "     $resp"
echo ""

# ── 8. Incident should be closed with duration ───────────
echo "── 8. GET /services/$SERVICE/incidents — incident should be closed"
resp=$(do_curl GET "${API}/services/${SERVICE}/incidents?limit=5&offset=0")
check "incidents: status closed"       "$resp" '"status":"closed"'
check "incidents: end_time set"        "$resp" '"end_time":'
check "incidents: duration_seconds set" "$resp" '"duration_seconds":'
$QUIET || echo "     $resp"
echo ""

# ── 9. Uptime endpoint ───────────────────────────────────
echo "── 9. GET /services/$SERVICE/uptime?window=24h"
resp=$(do_curl GET "${API}/services/${SERVICE}/uptime?window=24h")
check "uptime: has uptime_percent"        "$resp" '"uptime_percent":'
check "uptime: has total_downtime_seconds" "$resp" '"total_downtime_seconds":'
check "uptime: window=24h"                "$resp" '"window":"24h"'
$QUIET || echo "     $resp"
echo ""

# ── 10. Duplicate down — should be gracefully ignored ────
echo "── 10. Duplicate down (no open incident now) → ignored, not an error"
resp=$(webhook_post "down" "$(date -u +%Y-%m-%dT%H:%M:%SZ)")
check "duplicate down returns ok (ignored)" "$resp" '"status":"ok"'
echo ""

# ── 11. HTML status page ─────────────────────────────────
echo "── 11. GET /status — HTML page"
ct=$(curl -s -m 10 -o /dev/null -w "%{content_type}" \
    "${BASE_URL}/status" 2>/dev/null || echo "")
check "status page content-type is text/html" "$ct" "text/html"
echo ""

# ── Summary ──────────────────────────────────────────────
echo "═══════════════════════════════════════════════"
total=$((PASS+FAIL))
if [ $FAIL -eq 0 ]; then
    color_green "  PASSED $PASS/$total checks ✓"
    echo ""
    color_bold "  Next steps:"
    echo "  1. Install service-notify.sh drop-ins for your systemd units:"
    echo "     bash deploy/generate-dropins.sh"
    echo "  2. If Tokpoint runs as Docker: systemctl enable --now dockerwatch"
    echo "  3. Put nginx/caddy in front with TLS"
else
    color_red "  FAILED $FAIL/$total checks ✗  (passed: $PASS)"
    echo ""
    color_yellow "  Debug tips:"
    echo "  journalctl -u status-webhook -n 50 --no-pager"
    echo "  ss -tlnp | grep 8080"
    echo "  cat /opt/status-webhook/.env"
fi
echo "═══════════════════════════════════════════════"
echo ""

exit $FAIL
