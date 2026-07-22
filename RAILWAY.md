# Deploying opdeals to Railway (serve-only)

Prod runs **one** service that only serves — it never scrapes. Scraping needs the
heavy local setup (FlareSolverr for Cloudflare, Chrome for MyPCards) and shares a
Liga IP reputation a stray prod scrape could get banned, so it stays on your
machine. You scrape locally, then push the results onto the prod volume.

| Service   | Source                                     | Volume      | Public domain |
| --------- | ------------------------------------------ | ----------- | ------------- |
| `opdeals` | this repo (Dockerfile, via `railway.toml`) | `/app/data` | yes           |

No FlareSolverr service is deployed — prod does no Cloudflare fetching.

**Data ownership** (the two sides never touch each other's files):

- **Prod-owned** — edited live via the UI: `trades*.json`, `quotes*.json`.
  Pulled **down** to your machine with `scripts/sync-down.sh`.
- **Local-owned** — produced by your scrapes: `snapshot*.json`, `tracking*` dirs
  (day snapshots + each dir's `images.json`). Pushed **up** with `scripts/sync-up.sh`.

---

## 1. Create the project and link this repo

```sh
cd /Users/jjairo/Documents/www/opdeals
# Install the CLI once (macOS):  brew install railway
railway login
railway init            # create a new project (or: railway link to an existing one)
```

The `opdeals` service builds from the Dockerfile using `railway.toml` — no build
config needed in the dashboard. Its start command is already
`opdeals-server -web=web/dist -addr :$PORT -serve-only`.

## 2. Attach the data volume (critical)

On the **opdeals** service → **Settings → Volumes** → add a volume mounted at
**`/app/data`**, size **≥ 2 GB** (local data is ~450 MB and grows daily). Without
this, every redeploy wipes all history.

## 3. Set env vars on the opdeals service

- `TZ=America/Sao_Paulo` — the tracking calendar day depends on it.
- `ADMIN_TOKEN=<a long random secret>` — enables `POST /api/admin/reload`, which
  `sync-up.sh` calls to reload the pushed deals snapshots with zero downtime.
  Generate one with `openssl rand -hex 24`.

`PORT` is injected by Railway automatically; the start command reads it via `-addr :$PORT`.

## 4. Deploy

```sh
railway up
```

Watch logs for `cache loaded: …` (per game), `scheduler off …`, and
`listening on :<PORT>`. There must be **no** "starting initial scan" line — that
would mean a snapshot was missing (seed it, step 5).

## 5. Seed the volume (one-time)

`data/` is gitignored, so a fresh deploy starts empty. After the volume is
attached and the service is up, seed it once:

```sh
scripts/seed-prod.sh
```

This pushes your full local `data/` (minus `*.bak*`/`*.old*`) onto the volume,
then reloads. Verify:

```sh
railway ssh --service opdeals -- 'ls -la /app/data | head'
```

Hit the generated domain — the SPA loads, `/api/games` returns JSON, and the
portfolio is editable.

---

## Ongoing workflow (every time you scrape)

1. Scrape locally as usual (`docker compose up`, which runs `-schedule` with
   FlareSolverr — unchanged).
2. **Optional but nicer prices:** `scripts/sync-down.sh` first, so local
   held-card live-pricing reflects your real prod portfolio. (Not required — a
   pushed snapshot already carries the full price index and values any held card.)
3. `scripts/sync-up.sh` — pushes `snapshot*.json` + `tracking*` dirs to the
   volume and calls `/api/admin/reload`. Tracking is live immediately (read from
   disk per request); deals snapshots are swapped by the reload call. No restart.

`scripts/sync-down.sh` pulls `trades*.json` + `quotes*.json` back down whenever
you want your local scrape to price against the current shared portfolio.

All three scripts read `scripts/deploy.env` (gitignored):

```sh
OPDEALS_URL=https://<your-app>.up.railway.app
ADMIN_TOKEN=<same secret as the Railway env var>
RAILWAY_SERVICE=opdeals
```

---

## Known gotchas

- **No Brazil region.** Prod never scrapes Liga, so its US/EU IP is irrelevant —
  all Liga fetching happens locally.
- **Single replica only.** Never scale `opdeals` past 1 — the sync/reload assume
  one process owns `/app/data` (see the comment in `railway.toml`).
- **`railway ssh` stdin/stdout piping.** The sync scripts pipe tar over
  `railway ssh`. If your CLI is old and can't stream, `brew upgrade railway`.
  On the first `sync-down.sh`, eyeball the pulled files before trusting it.
- **Missing snapshot → no scan.** In serve-only, a missing `snapshot*.json` is
  NOT auto-scraped (by design). If a game shows no deals/portfolio prices, its
  snapshot never got seeded — re-run `seed-prod.sh` or `sync-up.sh`.
