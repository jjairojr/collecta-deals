#!/usr/bin/env bash
# One-time: seed a fresh prod volume with the full local data/ — deals snapshots,
# tracking dirs, AND the initial trades*/quotes* so the portfolio starts
# populated. After this, use sync-up.sh (scraper output) and sync-down.sh
# (portfolio). Skips *.bak*/*.old*/*.tmp and the local .sync-bak backups.
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/_common.sh"

cd "$DATA_DIR"

items=()
for f in snapshot*.json trades*.json quotes*.json; do
	case "$f" in
		*.bak*|*.old*|*.tmp) continue ;;
	esac
	[ -e "$f" ] && items+=("$f")
done
for d in tracking tracking-pkm tracking-rft tracking-lor tracking-gnd; do
	[ -d "$d" ] && items+=("$d")
done

if [ ${#items[@]} -eq 0 ]; then
	echo "error: $DATA_DIR has nothing to seed" >&2
	exit 1
fi

echo "seeding volume $RAILWAY_VOLUME (/app/data) with:"
printf '  %s\n' "${items[@]}"
read -r -p "This uploads/overwrites these on the prod volume. Continue? [y/N] " ans
case "$ans" in y|Y|yes) ;; *) echo "aborted."; exit 1 ;; esac

for it in "${items[@]}"; do
	echo ">> $it"
	vf_upload "$it" "/$it"
done

reload_prod
echo "seed done. Verify: railway volume files -v $RAILWAY_VOLUME list /"
