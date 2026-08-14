#!/usr/bin/env bash
# fix-dropins.sh - Fix misplaced OnFailure= in systemd drop-in notify.conf files.
#
# Problem: OnFailure= was placed under [Service] instead of [Unit].
# This script rewrites each notify.conf with the correct section layout.
#
# Usage:
#   bash fix-dropins.sh            # dry-run (shows what would change)
#   bash fix-dropins.sh --apply    # apply fixes + daemon-reload

set -euo pipefail

SYSTEMD_DIR="/etc/systemd/system"
DRY_RUN=true

if [ "${1:-}" = "--apply" ]; then
    DRY_RUN=false
fi

printf '\n'
printf '======================================================\n'
printf '  Fix notify.conf drop-ins (OnFailure -> [Unit])\n'
printf '======================================================\n'
if $DRY_RUN; then
    printf '  Mode : DRY RUN  (pass --apply to write)\n'
else
    printf '  Mode : APPLY\n'
fi
printf '======================================================\n\n'

FIXED=0
SKIPPED=0

for conf in "$SYSTEMD_DIR"/*.service.d/notify.conf; do
    [ -f "$conf" ] || continue

    unit_dir=$(basename "$(dirname "$conf")")

    # Check if OnFailure is under [Service] (the bug)
    if grep -q '^\[Service\]' "$conf" && \
       awk '/^\[Service\]/,/^\[/' "$conf" | grep -q '^OnFailure='; then

        printf '  ✗ BROKEN  %s\n' "$conf"

        if ! $DRY_RUN; then
            # Backup original
            cp "$conf" "${conf}.bak"

            # Rewrite with correct layout
            cat > "$conf" <<'EOF'
[Unit]
OnFailure=notify@%n.service

[Service]
ExecStartPost=/usr/local/bin/service-notify.sh up %n
ExecStopPost=/usr/local/bin/service-notify.sh down %n
EOF
            printf '    ✓ FIXED  (backup: %s.bak)\n' "$conf"
        fi
        FIXED=$((FIXED + 1))
    else
        printf '  ✓ OK      %s\n' "$conf"
        SKIPPED=$((SKIPPED + 1))
    fi
done

printf '\n------------------------------------------------------\n'
printf '  Fixed: %d   Already OK: %d\n' "$FIXED" "$SKIPPED"
printf '------------------------------------------------------\n\n'

if ! $DRY_RUN && [ "$FIXED" -gt 0 ]; then
    printf '  Running: systemctl daemon-reload\n'
    systemctl daemon-reload
    printf '  ✓ Done! daemon-reload completed.\n\n'
elif $DRY_RUN && [ "$FIXED" -gt 0 ]; then
    printf '  Run with --apply to fix:\n'
    printf '    bash %s --apply\n\n' "$0"
fi
