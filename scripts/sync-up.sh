#!/usr/bin/env bash
# Push locally scraped output (deals snapshots + tracking dirs) onto the prod
# /app/data volume, then reload prod. Never touches prod-owned trades/quotes.
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/_common.sh"

cd "$DATA_DIR"

items=()
for f in snapshot*.json; do
	case "$f" in
		*.bak*|*.old*|*.tmp) continue ;;
	esac
	[ -e "$f" ] && items+=("$f")
done
for d in tracking tracking-pkm tracking-rft tracking-lor tracking-gnd; do
	[ -d "$d" ] && items+=("$d")
done

if [ ${#items[@]} -eq 0 ]; then
	echo "error: nothing to push (no snapshot*.json or tracking* dirs in $DATA_DIR)" >&2
	exit 1
fi

echo "pushing to $RAILWAY_SERVICE:/app/data ->"
printf '  %s\n' "${items[@]}"

tar czf - --exclude='*.bak*' --exclude='*.old*' --exclude='*.tmp' "${items[@]}" \
	| railway ssh --service "$RAILWAY_SERVICE" -- 'tar xzf - -C /app/data'

reload_prod
echo "sync-up done."
