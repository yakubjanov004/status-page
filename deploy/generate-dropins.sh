#!/usr/bin/env bash
# generate-dropins.sh — Generate systemd drop-in files for service status notifications.
#
# This script scans the repository's server's/system directory for known service files
# and generates drop-in configuration files that hook into ExecStartPost/ExecStopPost
# to send webhook notifications on service state changes.
#
# Usage:
#   ./generate-dropins.sh [--dry-run]
#
# Options:
#   --dry-run   Print what would be done without writing any files (default behavior)
#   --apply     Actually generate the drop-in files and print install commands
#
# Environment:
#   SYSTEMD_DIR — Path to scan for service files (default: server's/system)
#   OUTPUT_DIR  — Where to write generated drop-ins (default: ./generated-dropins)

set -euo pipefail

DRY_RUN=true
if [ "${1:-}" = "--apply" ]; then
    DRY_RUN=false
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
SYSTEMD_DIR="${SYSTEMD_DIR:-$REPO_ROOT/server's/system}"
OUTPUT_DIR="${OUTPUT_DIR:-$REPO_ROOT/generated-dropins}"

# Service mapping: pattern -> canonical name
declare -A SERVICE_MAP=(
    ["alfaconnect"]="AlfaConnect"
    ["mehmonxona"]="Mehmonxona"
    ["odimrepo"]="Odimrepo"
    ["datan"]="Datan"
)

# Note: Tokpoint units are NOT included here — Tokpoint uses docker-watcher instead.
# However, non-Docker Tokpoint services can still be tracked via systemd.

echo "=========================================="
echo "  systemd Drop-in Generator"
echo "=========================================="
echo ""
echo "Scanning: $SYSTEMD_DIR"
echo "Mode:     $(if $DRY_RUN; then echo 'DRY RUN (use --apply to generate files)'; else echo 'APPLY'; fi)"
echo ""

if [ ! -d "$SYSTEMD_DIR" ]; then
    echo "ERROR: Directory $SYSTEMD_DIR does not exist"
    exit 1
fi

UNITS_FOUND=()
INSTALL_COMMANDS=()

# Scan for matching service files
for pattern in "${!SERVICE_MAP[@]}"; do
    canonical="${SERVICE_MAP[$pattern]}"

    # Find all matching .service files
    while IFS= read -r -d '' svc_file; do
        unit_name=$(basename "$svc_file")

        # Skip non-.service files
        [[ "$unit_name" != *.service ]] && continue

        echo "  ✓ Found: $unit_name → $canonical"
        UNITS_FOUND+=("$unit_name")

        # Generate drop-in content
        DROPIN_CONTENT="[Service]
ExecStartPost=/usr/local/bin/service-notify.sh up %n
ExecStopPost=/usr/local/bin/service-notify.sh down %n
OnFailure=notify@%n.service
"

        if ! $DRY_RUN; then
            # Create output directory
            dropin_dir="$OUTPUT_DIR/${unit_name}.d"
            mkdir -p "$dropin_dir"
            echo "$DROPIN_CONTENT" > "$dropin_dir/notify.conf"
            echo "    → Written: $dropin_dir/notify.conf"
        fi

        # Collect install commands
        INSTALL_COMMANDS+=("sudo mkdir -p /etc/systemd/system/${unit_name}.d")
        INSTALL_COMMANDS+=("sudo cp $OUTPUT_DIR/${unit_name}.d/notify.conf /etc/systemd/system/${unit_name}.d/notify.conf")

    done < <(find "$SYSTEMD_DIR" -maxdepth 1 -name "${pattern}*" -print0 2>/dev/null)
done

echo ""
echo "=========================================="
echo "  Found ${#UNITS_FOUND[@]} unit(s)"
echo "=========================================="

if [ ${#UNITS_FOUND[@]} -eq 0 ]; then
    echo ""
    echo "No matching service files found in $SYSTEMD_DIR"
    echo "Expected patterns: alfaconnect-*, mehmonxona*, odimrepo-*, datan*"
    exit 0
fi

echo ""
echo "--- Installation Commands ---"
echo ""

if ! $DRY_RUN; then
    echo "Generated drop-in files are in: $OUTPUT_DIR"
    echo ""
    echo "Run the following commands to install:"
else
    echo "After running with --apply, run these commands to install:"
fi

echo ""
for cmd in "${INSTALL_COMMANDS[@]}"; do
    echo "  $cmd"
done

echo ""
echo "  sudo systemctl daemon-reload"
echo ""

echo "--- Prerequisite ---"
echo ""
echo "  sudo cp $(dirname "$0")/service-notify.sh /usr/local/bin/service-notify.sh"
echo "  sudo chmod +x /usr/local/bin/service-notify.sh"
echo "  sudo cp $(dirname "$0")/notify@.service /etc/systemd/system/notify@.service"
echo ""
echo "  Ensure HOOK_TOKEN and WEBHOOK_URL are set in the environment"
echo "  or in /etc/default/service-notify:"
echo ""
echo '  WEBHOOK_URL="http://127.0.0.1:8080/api/v1/webhook"'
echo '  HOOK_TOKEN="your-secret-token"'
echo ""
echo "=========================================="
