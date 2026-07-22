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

# reload_prod swaps the pushed deals snapshots into prod memory (zero downtime).
reload_prod() {
	echo "reloading prod deals snapshots..."
	curl -fsS -X POST -H "X-Admin-Token: $ADMIN_TOKEN" "$OPDEALS_URL/api/admin/reload"
	echo
}
