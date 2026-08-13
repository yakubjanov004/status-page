#!/usr/bin/env bash
# generate-dropins.sh — Generate systemd drop-in files for webhook notifications.
#
# Usage:
#   bash deploy/generate-dropins.sh            # dry-run (default)
#   bash deploy/generate-dropins.sh --apply    # write files
#
# After --apply, run the printed install commands, then: systemctl daemon-reload

set -uo pipefail

DRY_RUN=true
if [ "${1:-}" = "--apply" ]; then
    DRY_RUN=false
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
SYSTEMD_DIR="${SYSTEMD_DIR:-$REPO_ROOT/server's/system}"
OUTPUT_DIR="${OUTPUT_DIR:-$REPO_ROOT/generated-dropins}"

# Map unit name to canonical service name.
# Returns empty string if unit should be skipped.
service_for_unit() {
    case "$1" in
        alfaconnect-bot.service)      echo "AlfaConnect" ;;
        alfaconnect-webapp.service)   echo "AlfaConnect" ;;
        mehmonxona.service)           echo "Mehmonxona" ;;
        odimrepo-backend.service)     echo "Odimrepo" ;;
        odimrepo-frontend.service)    echo "Odimrepo" ;;
        tokpoint-backend.service)     echo "Tokpoint" ;;
        tokpoint-frontend.service)    echo "Tokpoint" ;;
        tokpoint-worker.service)      echo "Tokpoint" ;;
        tokpoint-beat.service)        echo "Tokpoint" ;;
        *)                            echo "" ;;
    esac
    # Note: tokpoint-docker.service is excluded — handled by dockerwatch.
}

UNITS="alfaconnect-bot.service
alfaconnect-webapp.service
mehmonxona.service
odimrepo-backend.service
odimrepo-frontend.service
tokpoint-backend.service
tokpoint-frontend.service
tokpoint-worker.service
tokpoint-beat.service"

printf '\n'
printf '======================================================\n'
printf '  systemd Drop-in Generator\n'
printf '======================================================\n'
if $DRY_RUN; then
    printf '  Mode : DRY RUN  (pass --apply to write files)\n'
else
    printf '  Mode : APPLY\n'
fi
printf '  Scan : %s\n' "$SYSTEMD_DIR"
printf '  Out  : %s\n' "$OUTPUT_DIR"
printf '======================================================\n\n'

if [ ! -d "$SYSTEMD_DIR" ]; then
    printf 'ERROR: directory not found: %s\n' "$SYSTEMD_DIR"
    exit 1
fi

FOUND=0
SKIPPED=0

for unit in $UNITS; do
    canonical=$(service_for_unit "$unit")
    if [ -z "$canonical" ]; then
        continue
    fi

    if [ ! -f "$SYSTEMD_DIR/$unit" ]; then
        printf '  o SKIP   %s  (not in %s)\n' "$unit" "$SYSTEMD_DIR"
        SKIPPED=$((SKIPPED + 1))
        continue
    fi

    printf '  + FOUND  %s  ->  %s\n' "$unit" "$canonical"
    FOUND=$((FOUND + 1))

    if ! $DRY_RUN; then
        dropin_dir="$OUTPUT_DIR/${unit}.d"
        mkdir -p "$dropin_dir"
        cat > "$dropin_dir/notify.conf" <<DROPIN
[Service]
ExecStartPost=/usr/local/bin/service-notify.sh up %n
ExecStopPost=/usr/local/bin/service-notify.sh down %n
OnFailure=notify@%n.service
DROPIN
        printf '      Written: %s/notify.conf\n' "$dropin_dir"
    fi
done

printf '\n------------------------------------------------------\n'
printf '  Found: %d  |  Skipped: %d\n' "$FOUND" "$SKIPPED"
printf '------------------------------------------------------\n\n'

if [ "$FOUND" -eq 0 ]; then
    printf 'No matching units found. Check SYSTEMD_DIR.\n'
    exit 0
fi

printf '  Prerequisite — run once if not already done:\n\n'
printf '    cp %s/service-notify.sh /usr/local/bin/service-notify.sh\n' "$SCRIPT_DIR"
printf '    chmod +x /usr/local/bin/service-notify.sh\n'
printf '    cp %s/notify@.service /etc/systemd/system/notify@.service\n\n' "$SCRIPT_DIR"
printf '  /etc/default/service-notify contents:\n\n'
printf '    WEBHOOK_URL=http://127.0.0.1:8080/api/v1/webhook\n'
printf '    HOOK_TOKEN=<your-token>\n\n'

if $DRY_RUN; then
    printf '  Run with --apply to generate files, then install:\n\n'
    printf '    bash %s/generate-dropins.sh --apply\n\n' "$SCRIPT_DIR"
else
    printf '  Install commands:\n\n'
fi

for unit in $UNITS; do
    canonical=$(service_for_unit "$unit")
    [ -z "$canonical" ] && continue
    [ ! -f "$SYSTEMD_DIR/$unit" ] && continue
    printf '    mkdir -p /etc/systemd/system/%s.d\n' "$unit"
    printf '    cp %s/%s.d/notify.conf /etc/systemd/system/%s.d/notify.conf\n' \
        "$OUTPUT_DIR" "$unit" "$unit"
done

printf '    systemctl daemon-reload\n'
printf '\n======================================================\n\n'
