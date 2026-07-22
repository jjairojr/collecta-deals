#!/usr/bin/env bash
# Pull prod-owned portfolio files (trades*.json + quotes*.json) down to local.
# Never touches local snapshots or tracking. Backs up the local copies first.
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/_common.sh"

mkdir -p "$DATA_DIR"
tmp="$(mktemp -t opdeals-portfolio.XXXXXX).tgz"
trap 'rm -f "$tmp"' EXIT

echo "pulling trades*.json + quotes*.json from $RAILWAY_SERVICE:/app/data ..."
railway ssh --service "$RAILWAY_SERVICE" -- \
	sh -c 'cd /app/data && files=$(ls trades*.json quotes*.json 2>/dev/null) && [ -n "$files" ] && tar czf - $files || true' \
	> "$tmp"

if [ ! -s "$tmp" ]; then
	echo "nothing to pull (prod has no trades*/quotes* files yet)."
	exit 0
fi

# Back up whatever we're about to overwrite.
bak="$DATA_DIR/.sync-bak/portfolio-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$bak"
( cd "$DATA_DIR" && cp -p trades*.json quotes*.json "$bak"/ 2>/dev/null || true )

tar xzf "$tmp" -C "$DATA_DIR"
echo "pulled into $DATA_DIR (local copies backed up to $bak):"
tar tzf "$tmp" | sed 's/^/  /'
echo "sync-down done."
