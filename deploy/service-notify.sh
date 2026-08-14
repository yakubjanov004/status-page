#!/bin/bash
# service-notify.sh — systemd xizmat o'chganda/ishga tushganda
# status page API'ga xabar yuboradi.
#
# Foydalanish:
#   /usr/local/bin/service-notify.sh up   datan.service
#   /usr/local/bin/service-notify.sh down datan.service
#
# Systemd drop-in sozlash:
#   /etc/systemd/system/<service>.service.d/notify.conf
#   [Service]
#   ExecStartPost=/usr/local/bin/service-notify.sh up %n
#   ExecStopPost=/usr/local/bin/service-notify.sh down %n

ACTION="${1:-up}"
SERVICE="${2:-unknown.service}"
STATUS_API="http://127.0.0.1:8880/api/internal/service-notify"
LOG_TAG="service-notify"

# JSON payload yuborish — xatolarni log qilamiz
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$STATUS_API" \
  -H "Content-Type: application/json" \
  -d "{\"action\":\"$ACTION\",\"service\":\"$SERVICE\"}" \
  --connect-timeout 3 \
  --max-time 5 2>&1)

HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')

if [ "$HTTP_CODE" = "200" ]; then
  logger -t "$LOG_TAG" "OK: $SERVICE -> $ACTION (HTTP $HTTP_CODE)"
else
  logger -t "$LOG_TAG" "ERROR: $SERVICE -> $ACTION (HTTP $HTTP_CODE) body: $BODY"
fi

# Xatolik bo'lsa ham xizmat ishga tushishiga halaqit bermaymiz
exit 0
