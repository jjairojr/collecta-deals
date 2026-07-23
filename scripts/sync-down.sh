#!/usr/bin/env bash
# Pull prod-owned portfolio files (trades*.json + quotes*.json) down to local.
# Never touches local snapshots or tracking. Backs up local copies first.
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/_common.sh"

mkdir -p "$DATA_DIR"

# List the volume root and keep only the portfolio files. Read with a while loop
# rather than mapfile/readarray — macOS ships bash 3.2, which lacks both.
files=()
while IFS= read -r name; do
	[ -n "$name" ] && files+=("$name")
done < <(vf list / --json 2>/dev/null | python3 -c '
import sys, json, re
try:
    fs = json.load(sys.stdin).get("files", [])
except Exception:
    fs = []
for f in fs:
    if f.get("type") == "file" and re.match(r"^(trades|quotes).*\.json$", f.get("name", "")):
        print(f["name"])
')

if [ ${#files[@]} -eq 0 ]; then
	echo "nothing to pull (prod has no trades*/quotes* files yet)."
	exit 0
fi

# Back up whatever we're about to overwrite.
bak="$DATA_DIR/.sync-bak/portfolio-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$bak"
( cd "$DATA_DIR" && cp -p trades*.json quotes*.json "$bak"/ 2>/dev/null || true )

for f in "${files[@]}"; do
	echo ">> $f"
	vf_download "/$f" "$DATA_DIR/$f"
done

echo "pulled ${#files[@]} file(s) into $DATA_DIR (local copies backed up to $bak)."
echo "sync-down done."
