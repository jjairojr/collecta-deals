#!/usr/bin/env bash
# One-time: seed a fresh prod volume with the full local data/ (minus backups),
# including the initial trades*/quotes* so the portfolio starts populated.
# After this, use sync-up.sh (scraper output) and sync-down.sh (portfolio).
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/_common.sh"

if [ ! -d "$DATA_DIR" ]; then
	echo "error: $DATA_DIR not found" >&2
	exit 1
fi

echo "seeding $RAILWAY_SERVICE:/app/data with the full local data/ (excluding *.bak*/*.old*/*.tmp)..."
read -r -p "This overwrites matching files on the prod volume. Continue? [y/N] " ans
case "$ans" in y|Y|yes) ;; *) echo "aborted."; exit 1 ;; esac

tar czf - -C "$DATA_DIR" --exclude='*.bak*' --exclude='*.old*' --exclude='*.tmp' --exclude='.sync-bak' . \
	| railway ssh --service "$RAILWAY_SERVICE" -- 'tar xzf - -C /app/data'

reload_prod
echo "seed done. Verify: railway ssh --service $RAILWAY_SERVICE -- 'ls -la /app/data | head'"
