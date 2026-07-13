#!/usr/bin/env bash
#
# backup-data.sh — timestamped, rotated backups of the opdeals data/ directory.
#
# Usage:
#   scripts/backup-data.sh                 # backup ./data -> ./backups, keep last 14
#   DATA_DIR=/path/to/data scripts/backup-data.sh
#   BACKUP_DIR=/mnt/disk/opdeals-backups KEEP=30 scripts/backup-data.sh
#
# Schedule it (macOS cron example, every 6h aligned with the scan cycle):
#   0 */6 * * * cd /Users/jjairo/Documents/www/opdeals && scripts/backup-data.sh >> backups/backup.log 2>&1

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DATA_DIR="${DATA_DIR:-$REPO_ROOT/data}"
BACKUP_DIR="${BACKUP_DIR:-$REPO_ROOT/backups}"
KEEP="${KEEP:-14}"

if [ ! -d "$DATA_DIR" ]; then
  echo "backup-data: data dir not found: $DATA_DIR" >&2
  exit 1
fi

mkdir -p "$BACKUP_DIR"

STAMP="$(date +%Y-%m-%dT%H%M%S)"
ARCHIVE="$BACKUP_DIR/data-$STAMP.tar.gz"

# Archive to a temp file first, then rename — a partial/failed run never leaves
# a truncated archive that looks valid.
TMP="$ARCHIVE.tmp"
tar -czf "$TMP" -C "$(dirname "$DATA_DIR")" "$(basename "$DATA_DIR")"
mv "$TMP" "$ARCHIVE"

SIZE="$(du -h "$ARCHIVE" | cut -f1)"
echo "backup-data: wrote $ARCHIVE ($SIZE)"

# Rotation: keep the newest $KEEP archives, delete the rest.
ls -1t "$BACKUP_DIR"/data-*.tar.gz 2>/dev/null | tail -n "+$((KEEP + 1))" | while IFS= read -r f; do
  rm -f "$f"
  echo "backup-data: rotated out $(basename "$f")"
done
