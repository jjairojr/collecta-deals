#!/usr/bin/env bash
# Push locally scraped output (deals snapshots + tracking dirs) onto the prod
# volume, then reload prod. Never touches prod-owned trades/quotes.
#
# Tracking dirs are re-uploaded whole (railway volume files has no delta sync);
# they're the bulk of the transfer. Snapshots are tiny and are what the portfolio
# pricing needs fresh, so this always sends them. Pass -s / --snapshots-only to
# push just the snapshots (fast) and skip the heavy tracking re-upload.
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/_common.sh"

snapshots_only=0
case "${1:-}" in
	-s|--snapshots-only) snapshots_only=1 ;;
	"") ;;
	*) echo "usage: sync-up.sh [-s|--snapshots-only]" >&2; exit 1 ;;
esac

cd "$DATA_DIR"

items=()
for f in snapshot*.json; do
	case "$f" in
		*.bak*|*.old*|*.tmp) continue ;;
	esac
	[ -e "$f" ] && items+=("$f")
done
if [ "$snapshots_only" -eq 0 ]; then
	for d in tracking tracking-pkm tracking-rft tracking-lor tracking-gnd; do
		[ -d "$d" ] && items+=("$d")
	done
fi

if [ ${#items[@]} -eq 0 ]; then
	echo "error: nothing to push (no snapshot*.json in $DATA_DIR)" >&2
	exit 1
fi

echo "pushing to volume $RAILWAY_VOLUME (/app/data):"
for it in "${items[@]}"; do
	echo ">> $it"
	vf_upload "$it" "/$it"
done

reload_prod
echo "sync-up done."
