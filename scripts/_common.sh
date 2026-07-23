#!/usr/bin/env bash
# Shared setup for the deploy/sync scripts. Sourced, not run directly.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
DATA_DIR="$REPO_DIR/data"
ENV_FILE="$SCRIPT_DIR/deploy.env"

if [ ! -f "$ENV_FILE" ]; then
	echo "error: $ENV_FILE not found. Copy scripts/deploy.env.example to scripts/deploy.env and fill it in." >&2
	exit 1
fi
# shellcheck disable=SC1090
set -a; . "$ENV_FILE"; set +a

: "${OPDEALS_URL:?set OPDEALS_URL in scripts/deploy.env}"
: "${ADMIN_TOKEN:?set ADMIN_TOKEN in scripts/deploy.env}"
: "${RAILWAY_SERVICE:=opdeals}"

if ! command -v railway >/dev/null 2>&1; then
	echo "error: railway CLI not found. Install it: brew install railway" >&2
	exit 1
fi

# Resolve the Railway volume mounted at /app/data (override with RAILWAY_VOLUME).
: "${RAILWAY_VOLUME:=}"
if [ -z "$RAILWAY_VOLUME" ]; then
	RAILWAY_VOLUME="$(railway volume list --json 2>/dev/null | python3 -c '
import sys, json
try:
    vols = json.load(sys.stdin).get("volumes", [])
except Exception:
    vols = []
pick = next((v for v in vols if v.get("mountPath") == "/app/data"), None) or (vols[0] if vols else None)
print(pick["name"] if pick else "")
')"
fi
: "${RAILWAY_VOLUME:?could not determine the Railway volume; set RAILWAY_VOLUME in scripts/deploy.env}"

# vf wraps the volume-files CLI with the resolved volume. Remote paths are
# relative to the volume root, which is the /app/data mount (so "/snapshot.json"
# is /app/data/snapshot.json on the running service).
vf() { railway volume files -v "$RAILWAY_VOLUME" "$@"; }

# The volume-files transport occasionally flakes with a transient
# "SSH authentication failed" / SFTP timeout; retry a few times before giving up.
# Replacing prod's copy is the whole point of a push, so --overwrite is required:
# without it the CLI aborts as soon as the remote path exists (i.e. every sync
# after the initial seed).
vf_upload() {
	local src="$1" dst="$2" n=0
	until vf upload "$src" "$dst" --overwrite; do
		n=$((n + 1))
		if [ "$n" -ge 4 ]; then
			echo "error: upload $src -> $dst failed after $n attempts" >&2
			return 1
		fi
		echo "  transient failure; retry $n for $src..." >&2
		sleep 3
	done
}

# Pulls prod's portfolio over the local copy on purpose (sync-down.sh backs the
# local files up first), so --overwrite is required — without it the CLI aborts
# whenever the local file already exists.
vf_download() {
	local src="$1" dst="$2" n=0
	until vf download "$src" "$dst" --overwrite; do
		n=$((n + 1))
		if [ "$n" -ge 4 ]; then
			echo "error: download $src -> $dst failed after $n attempts" >&2
			return 1
		fi
		echo "  transient failure; retry $n for $src..." >&2
		sleep 3
	done
}

# reload_prod swaps the pushed deals snapshots into prod memory (zero downtime).
reload_prod() {
	echo "reloading prod deals snapshots..."
	curl -fsS -X POST -H "X-Admin-Token: $ADMIN_TOKEN" "$OPDEALS_URL/api/admin/reload"
	echo
}
