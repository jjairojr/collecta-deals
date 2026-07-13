# opdeals

Cross-border One Piece TCG arbitrage finder. It compares the lowest **Brazilian**
marketplace price (LigaOnePiece, plus optionally MyP Cards) against the **US** reference
price (TCGPlayer, via the free TCGCSV mirror), converts BRL→USD, and prints the cards with
the biggest price gap — i.e. cards cheap to buy in Brazil and worth more in the US. When
more than one Brazilian source is enabled, each card is bought from whichever source is
cheaper.

## How it works

```
LigaOnePiece ─┐
MyP Cards   ──┤─► cheapest BR per card ─► match by card number ─► FX (BRL→USD) ─► margin ─► table
TCGCSV (US) ──┘   (--mypcards, opt-in)
```

- **US prices** start from [TCGCSV](https://tcgcsv.com) — a free, no-auth JSON mirror of
  TCGPlayer (One Piece `categoryId 68`) used for discovery/matching across the whole catalog.
  TCGCSV's `marketPrice` (recent-sales average) and `lowPrice` are both unreliable for illiquid
  high-value cards (market runs high; low can be a stale outlier). So for candidate deals the real
  **current lowest listing** is fetched live from TCGPlayer's marketplace API
  (`mp-search-api.tcgplayer.com/.../listings`) — see live pricing below.
- **Brazil prices** come from LigaOnePiece's server-rendered set pages (`var cardsjson`),
  one request per set. A browser `User-Agent` is enough (no headless browser).
- **MyP Cards** (`--mypcards`, opt-in) is a second Brazilian source. Its whole site sits
  behind a Cloudflare JS challenge, so it is scraped through a **headless Chrome** (chromedp):
  Chrome solves the challenge once, then every One Piece edition is paginated and parsed from
  the rendered HTML. Each listed product already carries its lowest price **and** current stock
  inline, so — unlike LigaOnePiece — no separate stock check is needed. It needs Chrome
  installed and adds ~1–3 min to a scan, so it is off by default; it fails soft (a missing
  Chrome or an unsolved challenge logs and is skipped rather than failing the scan).
- Sources, plus the FX rate, are fetched **concurrently with goroutines** (bounded worker
  pools via `errgroup`). TCGCSV fans out one goroutine per set group; LigaOnePiece is
  rate-limited politely (it returns HTTP 429 under load) with straggler retry rounds.
- Cards are matched by **base card number + variant**. *All* variant markers (Alternate Art, SP,
  Manga, Parallel, Gold, …) are parsed from the spelled-out names on both sides, deduped and sorted
  into one token — so `(SP)` (`sp`) and `(SP) (Gold)` (`gold+sp`) are distinct and never matched to
  each other. LigaOnePiece also encodes variants as number suffixes (`OP11-106-SP`); TCGPlayer keeps
  the base number and names the product "Zeus (SP)". So each Brazilian variant matches its exact US
  counterpart (or nothing) rather than being collapsed or mismatched.
- Each card+variant is reported once, at its **cheapest** Brazilian price across all editions.
- **Live US pricing + stock verification (candidates only).** For each candidate deal selling above
  `--verify-floor` (default $100), two live checks run concurrently:
  - **TCGPlayer live price** — the card's current listings are fetched and the **lowest Near Mint
    (NM) listing** becomes the sell price (falling back to the overall-lowest listing only when a
    card has no NM listing at all). This fixes both TCGCSV failure modes, e.g. Enel (Manga) shows
    $945 (real floor) not $1106 (market), and a Luffy SP shows its true $13,999 floor instead of a
    stale $50,000 `lowPrice`.
  - **LigaOnePiece stock** — Liga's set-level price persists even with *no current sellers*, so
    every listing of a candidate card is checked and the deal is kept only if it has **current
    sellers** (`cards_stock` non-empty). Checking *every* listing (not just the cheapest) avoids a
    cascade where an excluded cheapest exposes an unverified, also-empty next-cheapest. This drops
    stale "price-only" cards (e.g. a Manga Ace at R$2150 with zero sellers). If a live check fails,
    it falls back to the sane TCGCSV price / keeps the card.

## Usage

```sh
# Top 20 deals with at least 100% margin on cards selling for $5+ (all sets)
go run ./cmd/opdeals --min-margin 100 --min-price 5 --limit 20

# Limit to specific Liga set codes (fast)
go run ./cmd/opdeals --sets OP-01,OP-02 --min-margin 0

# Override the exchange rate (offline / reproducible)
go run ./cmd/opdeals --fx 0.19 --sets ST30
```

### Flags

| Flag | Default | Meaning |
|---|---|---|
| `--min-margin` | `30` | Minimum margin percent to show |
| `--min-price` | `1.0` | Minimum US sell price (USD) to consider |
| `--fx` | `0` | BRL→USD rate override (`0` = fetch live from frankfurter.app) |
| `--sets` | _all_ | Comma-separated Liga set codes (e.g. `OP-01,ST30`) — filters LigaOnePiece only |
| `--mypcards` | `false` | Also scrape MyP Cards (2nd BR source; needs Chrome installed, adds ~1–3 min) |
| `--concurrency` | `8` | Max concurrent requests per source (Liga is internally capped) |
| `--us-price` | `market` | US price basis: `market` or `low` |
| `--sort` | `margin` | Sort by `margin` or `profit` |
| `--limit` | `0` | Max rows to print (`0` = all) |
| `--timeout` | `30s` | Per-request timeout |
| `--quiet` | `false` | Suppress progress logs |

Progress logs (stages, per-set/per-group counts, timing) are written to **stderr**, so the
results table on **stdout** stays clean and pipeable (`opdeals ... > deals.txt`). Use `--quiet`
to silence the logs.

Margin = `(US sell − Brazil low × FX) / (Brazil low × FX) × 100`. **It does not subtract
TCGPlayer seller fees or shipping** — it is a raw FX-adjusted price gap, so treat it as a
gross upper bound when judging real profit.

The last two columns are direct links to each card: **TCGPLAYER** (`product/{id}`, the exact
base print whose price is shown) and **LIGAONEPIECE** (the card's marketplace page). Most
terminals make them clickable; otherwise they are plain, copyable URLs. The link columns make
each row wide — use a wide terminal, or pipe to a file. The full URLs always resolve to the
right card.

## Web UI

A Vite + React + TypeScript + Tailwind front end backed by a Go HTTP API. The home page shows
the **best deals**; a search box lets you **compare any card** by name or number (showing the
Brazil vs US price even when it isn't profitable). Each row links straight to the card on
TCGPlayer and LigaOnePiece.

Because a full scan is rate-limited (~1–2 min), the server scans once, **caches the snapshot to
disk** (`data/snapshot.json`), and serves everything from memory. The UI shows a first-run
spinner until the first scan finishes, then live data; a **Refresh** button re-scans in the
background.

### Run it

```sh
# 1. build the front end
cd web && npm install && npm run build && cd ..

# 2. run the API + UI (serves web/dist, scans on first start)
go run ./cmd/server            # then open http://localhost:8080
```

Development (hot-reload front end + Go API on :8080, Vite proxies `/api`):

```sh
go run ./cmd/server &          # API on :8080
cd web && npm run dev          # UI on http://localhost:5173
```

Server flags: `--addr` (`:8080`), `--web` (`web/dist`, empty to serve API only),
`--cache` (`data/snapshot.json`), `--sets`, `--fx`, `--concurrency`,
`--mypcards` (`false` — add MyP Cards via headless Chrome),
`--verify-floor` (`100` — live-price + stock-check deals selling above this US$; `0` = skip),
`--live-prices` (`true` — use live TCGPlayer listing prices for candidates),
`--refresh-interval` (e.g. `6h`, `0` = off).

Live pricing + stock verification add ~1–2 min to a full scan (they hit TCGPlayer and LigaOnePiece
per high-value candidate, rate-limited, run concurrently). The home/API default to a **$100 minimum
sell price**.

### API

| Endpoint | Purpose |
|---|---|
| `GET /api/deals?minMargin=&minPrice=&sort=&limit=` | Best deals (home) |
| `GET /api/search?q=&limit=` | Compare cards matching a name/number (no margin filter) |
| `GET /api/status` | Cache readiness, last update, FX rate, counts |
| `POST /api/refresh` | Trigger a background re-scan |

## Layout

```
cmd/opdeals      CLI entry: flags, output
cmd/server       HTTP API + static UI server
internal/model   shared types + source interfaces + Snapshot
internal/httpx   shared HTTP client (browser UA, cookie jar, retry/backoff)
internal/tcgcsv  US catalog + baseline prices (TCGCSV)
internal/tcgplayer live lowest-listing prices (TCGPlayer marketplace API)
internal/liga    LigaOnePiece source + cardsjson/editions parser + stock check
internal/mypcards MyP Cards source (headless Chrome via chromedp) + listing parser
internal/fx      BRL→USD exchange rate
internal/pipeline concurrent fetch → Snapshot (shared by CLI and server)
internal/store   in-memory snapshot + disk cache + guarded refresh
internal/api     JSON API handlers + SPA serving
internal/compare matching, margin, filtering, sorting, search
internal/report  terminal table (text/tabwriter)
web/             Vite + React + TS + Tailwind front end
```

## Tests

```sh
go test ./...
```

Parser tests run against captured HTML in `internal/liga/testdata/`; compare tests cover the
number-collision, dedup, and margin logic.

## Known limitations

- **Language:** LigaOnePiece's set-level lowest can occasionally reflect a cheap Japanese
  listing rather than the English print. Brazilian One Piece is the English print, so it is
  usually correct; exact per-language pricing needs the single-card page, whose per-seller
  price is CSS-glyph obfuscated.
- **Variants:** base, Alternate Art, SP, Manga, Parallel, Gold, etc. are matched by their
  spelled-out name marker. A LigaOnePiece variant whose marker TCGPlayer doesn't write the same
  way (e.g. some region-specific promos) won't find a US price and is skipped, not mismatched.
- **Freshness:** the TCGCSV catalog is a daily snapshot, but candidate deals' US prices are fetched
  **live** from TCGPlayer at scan time. Cards below `--verify-floor` keep the daily TCGCSV price.
- **MyP Cards** (`--mypcards`) needs **Google Chrome installed** and drives it headless via
  chromedp to pass Cloudflare. Cloudflare can occasionally detect automation and not clear the
  challenge; when that happens the source is skipped (fail-soft) rather than erroring the scan.
  It also makes a scan noticeably slower (everything goes through the browser).
- **No fees/shipping** are deducted from the margin.
```
