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

# JSON payload yuborish
curl -s -X POST "$STATUS_API" \
  -H "Content-Type: application/json" \
  -d "{\"action\":\"$ACTION\",\"service\":\"$SERVICE\"}" \
  --connect-timeout 3 \
  --max-time 5 \
  > /dev/null 2>&1 || true

# Xatolik bo'lsa ham xizmat ishga tushishiga halaqit bermaymiz (|| true)
