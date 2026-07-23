# Deploying opdeals to Railway (serve-only)

Prod runs **one** service (`collecta-deals`) that only serves — it never scrapes.
Scraping needs the heavy local setup (FlareSolverr for Cloudflare, Chrome for
MyPCards) and shares a Liga IP reputation a stray prod scrape could get banned, so
it stays on your machine. You scrape locally, then push the results onto the prod
volume with the scripts in `scripts/`.

No FlareSolverr service is deployed — prod does no Cloudflare fetching. Card art
still works because `images.json` holds resolved CDN URLs and the bytes come from
`repositorio.sbrauble.com` (a non-Cloudflare host), which prod can fetch directly.

**Data ownership** (the two sides never touch each other's files):

- **Prod-owned** — edited live via the UI: `trades*.json`, `quotes*.json`.
  Pulled **down** with `scripts/sync-down.sh`.
- **Local-owned** — produced by your scrapes: `snapshot*.json`, `tracking*` dirs
  (day snapshots + each dir's `images.json`). Pushed **up** with `scripts/sync-up.sh`.

---

## One-time setup

```sh
cd /Users/jjairo/Documents/www/opdeals

# 1. CLI + auth
brew install railway
railway login
railway link                       # pick the existing project + service

# 2. Register an SSH key (required for volume file transfer)
railway ssh keys add               # registers ~/.ssh/id_*.pub

# 3. Create the persistent volume mounted at /app/data
railway volume add -m /app/data

# 4. Env vars on the service
railway variable set "TZ=America/Sao_Paulo"
railway variable set "ADMIN_TOKEN=$(openssl rand -hex 24)"   # save this value
```

`PORT` is injected by Railway; the app reads it from the env var (the start
command in `railway.toml` passes no `-addr`, so no shell `$PORT` expansion is
needed). The domain's target port must match the app's listen port (8080 by
default — leave it unless you set a `PORT` var).

Then fill `scripts/deploy.env` (copy from `scripts/deploy.env.example`):

```sh
OPDEALS_URL=https://<your-app>.up.railway.app   # railway domain
ADMIN_TOKEN=<same secret you set above>
RAILWAY_SERVICE=collecta-deals
RAILWAY_VOLUME=                                  # blank = auto-detect /app/data volume
```

## Deploy + seed

```sh
railway up                         # or push to the linked GitHub repo
```

Watch logs for `cache loaded: …` (per game), `scheduler off …`, and
`listening on :8080`. There must be **no** "starting initial scan" line.

Then seed the volume with your current local data (one-time):

```sh
scripts/seed-prod.sh
```

This uploads `snapshot*.json`, the `tracking*` dirs, and your initial
`trades*/quotes*` (minus `*.bak*`/`*.old*`), then reloads. Verify:

```sh
railway volume files -v <volume> list /
```

Hit the domain — the SPA loads, the portfolio is populated and editable.

---

## Ongoing workflow (every time you scrape)

1. Scrape locally as usual (`docker compose up` — runs `-schedule` with
   FlareSolverr, unchanged).
2. **Optional, nicer prices:** `scripts/sync-down.sh` first, so local held-card
   live-pricing reflects the real prod portfolio. (Not required — a pushed
   snapshot already carries the full price index and values any held card.)
3. `scripts/sync-up.sh` — uploads `snapshot*.json` + `tracking*` dirs, then calls
   `POST /api/admin/reload` (guarded by `ADMIN_TOKEN`). That reload swaps the deals
   snapshots **and** the card-image cache in memory — no restart, no session blip.
   Tracking day data is read from disk per request, so it's live immediately.
   - `scripts/sync-up.sh -s` pushes only the (tiny) snapshots and skips the heavy
     tracking re-upload — use it when you just want fresh deals/portfolio prices.

`scripts/sync-down.sh` pulls `trades*.json` + `quotes*.json` back down (backing up
the local copies to `data/.sync-bak/` first).

---

## Transport notes / gotchas

- **`railway volume files` needs a registered SSH key** (`railway ssh keys add`)
  and occasionally flakes with a transient `SSH authentication failed` /
  SFTP timeout — the scripts retry uploads/downloads a few times automatically.
- **Remote paths are relative to the volume root**, which is the `/app/data`
  mount — i.e. `/snapshot.json` on the volume is `/app/data/snapshot.json` in the
  container.
- **Every upload/download needs `--overwrite`.** Without it the CLI aborts as soon
  as the destination exists, i.e. on every sync after the initial seed.
- **Upload targets the volume ROOT (`/`), not `/<name>`.** With `--overwrite` the
  CLI has `cp -r` semantics, so uploading a directory onto an existing one nests it
  (`/tracking/tracking`) — which then shows up as a phantom set in the UI. The
  scripts always pass `/` for this reason. If a phantom appears, remove it with
  `railway volume files delete --volume <vol> /<dir>/<dir>` (agents are blocked
  from deleting, so this one is manual).
- **macOS bash is 3.2** — no `mapfile`/`readarray`. The scripts stick to
  3.2-compatible constructs.
- **Single replica only.** Never scale past 1 — the app owns one `/app/data`
  volume; a second replica would fight over the writes (see `railway.toml`).
- **Missing snapshot → no scan.** In serve-only, a missing `snapshot*.json` is not
  auto-scraped (by design). If a game shows no deals/portfolio prices, its snapshot
  never got seeded — re-run `seed-prod.sh` or `sync-up.sh`.
