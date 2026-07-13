# opdeals — build log

## Plan
- [x] Define model types + source interfaces (`internal/model`)
- [x] Shared HTTP client with browser UA, cookie jar, retry/backoff (`internal/httpx`)
- [x] TCGCSV US price source — groups → concurrent products+prices (`internal/tcgcsv`)
- [x] LigaOnePiece BR source + cardsjson/editions parser (`internal/liga`)
- [x] FX (BRL→USD), compare (match+margin+filter+sort), report (table)
- [x] CLI wiring + unit tests + end-to-end verification

## Review

**Result:** Working CLI. `go build/vet/test ./...` all clean. Full scan fetches all 78 Liga
sets (zero skips) and matches against TCGCSV US prices; spot-checked deals match verified
source prices and the margin math.

**Key decisions / fixes during build:**
- US side uses **TCGCSV** (free JSON mirror) instead of scraping TCGPlayer (Cloudflare + SPA).
- Cards match by **official number**; for numbers with multiple US printings, the **cheapest
  (base) print** is used. This fixed early bogus 10,000%+ margins caused by matching cheap
  Brazilian base cards against expensive Parallel/promo prints.
- **Dedup by number, keeping the cheapest Brazilian price** — the same card appears across
  base + promo Liga editions.
- LigaOnePiece **rate-limits hard (HTTP 429)**. Tuned to concurrency 2 + 500ms throttle with
  cooldown retry rounds for stragglers → zero skips. Per-set failures are non-fatal.

**Deferred (documented in README):** MyPCards (Cloudflare Turnstile → needs headless browser),
foil/alt-art variant matching, per-language pricing, real-time US prices (JustTCG), fees/shipping.

---

# MyP Cards — 2nd Brazilian source (in progress)

## Investigation findings (verified via chromedp probe)
- `mypcards.com` whole origin behind Cloudflare **JS challenge** (`cf-mitigated: challenge`).
  Headless Chrome (chromedp) **passes it** and harvests `cf_clearance`. Plain Go/curl + the
  cookie still 403s — clearance is bound to Chrome's TLS fingerprint, so **all fetching must go
  through the browser** (no harvest-then-fast-fetch shortcut).
- Server-rendered, PHP/Yii. Singles listing: `/onepiece/{slug}?page=N`, ~30 real singles/page,
  **real pagination** (page1∩page2 = 0), terminates when a page yields 0 `one_` singles.
- Editions: `/onepiece/edicoes?page=N&per-page=48` → set slugs (`the-time-of-battle`) + titles.
- Each product = `<li class="stream-item" data-key="{productId}">` with:
  - `data-ga-item-id="one_op16_op16-080p1"` → game `one` / set `op16` / number `OP16-080` / print `p1`.
    **Filter to `ga` prefix `one_`** (drops `mp_` featured booster + `op_{edid}_` sealed/sidebar +
    other-game `yugioh_…` that bleed onto overflow pages).
  - `<h3 title="Marshall.D.Teach (080) (Alternate Art)">` → name (carries variant in parens).
  - `card-edicao title="The Time of Battle">OP16` → set code.
  - `quantidade-num">10` → **stock count inline** (all listed products are in stock).
  - `card-preco moeda"> R$ 7.000,00` → lowest BRL price (BR format: `.`=thousands `,`=decimal).
- **No separate stock check needed** (unlike Liga): listing = in-stock + price. Set
  `StockChecked=true, InStock=(qty>0)` at parse time.

Decision: MyP **off by default**, opt in with `--mypcards`, fail-soft if Chrome missing.

## Plan
- [x] Add `github.com/chromedp/chromedp` dep (user-approved).
- [x] `internal/mypcards/browser.go` — chromedp lifecycle: one ExecAllocator (anti-detection
      flags, real UA), solve challenge once on `/`, then per-tab `fetch(url)`. **Gotcha fixed:**
      running the first nav on a `context.WithTimeout` child killed the browser when that child
      cancelled — solve runs on `b.ctx` directly; fetch uses a `time.AfterFunc` watchdog on the tab.
- [x] `internal/mypcards/parse.go` (pure) + `parse_test.go` against captured testdata HTML.
- [x] `internal/mypcards/mypcards.go` — `Client` implements `model.BrazilSource` (Name="mypcards"):
      editions → per-set paginate until a page yields 0 new `one_` singles. Fail-soft.
- [x] `internal/model`: `StockVerifier` interface (Liga satisfies it as-is); `Deal.Source` added,
      `Deal.LigaURL`→`Deal.BuyURL` (BR side now has two possible sources).
- [x] `internal/pipeline`: fetch from `[]BrazilSource` concurrently, merge; stock-verify only
      sources implementing `StockVerifier`, scoped to their own listings.
- [x] `internal/compare`: populate `Deal.Source` + `Deal.BuyURL` from the winning listing.
- [x] Display: `report.go` SRC column + `BUY (BR)`; `web/src/api.ts` (`buyUrl`,`source`) +
      `DealsTable.tsx` labels BR link by source.
- [x] Flags: `--mypcards` in `cmd/opdeals` + `cmd/server`.
- [x] `go build/vet/test ./...` clean; live one-set integration test (`MYP_LIVE=1`) passes
      (158 singles, all in-stock, OP16 numbers). Full `--mypcards` e2e scan + README updated.

## Also done this session (user request)
- [x] TCGPlayer sell price now uses **Near Mint only** (`isSellableGrade` = NM, no Lightly Played),
      with the existing fallback to overall-lowest only when a card has no NM listing.

## Review
- chromedp passes Cloudflare reliably from a residential IP; cf_clearance is TLS-bound so a
  harvest-then-curl shortcut is impossible — all fetching is browser-driven (the slow part).
- MyP listing HTML is self-describing (set code, number via `data-ga-item-id`, variant via name,
  inline stock + lowest price), so MyP needs no separate stock round-trip unlike Liga.
- Cross-source "buy from whoever's cheaper" falls out of the existing `cheapestListings` merge;
  the only new wiring was per-source stock verification + source tagging on the Deal.

---

# Sales by snapshot — per-interval sold cards (in progress)

Goal: tracking tab shows a "Sales by snapshot" timeline so the user can check, per
specific snapshot interval, which cards were sold (currently only whole-range totals exist).

## Backend (Go)
- [x] types.go: add `SnapshotSales{Date, PrevDate, Units, RevenueBRL, Cards []CardSale}`
- [x] analysis.go: extract `salesBetween(prev, cur)` from the TopSoldCards loop
- [x] analysis.go: add `mergeSales(groups)` and rewrite `TopSoldCards` to reuse it (no behavior change)
- [x] analysis.go: add `SalesBySnapshot(days) []SnapshotSales` (newest-first, includes 0-unit intervals)
- [x] api.go: route `GET /api/tracking/sold-by-snapshot` + handler (per-set only)

## Frontend (React/TS)
- [x] api.ts: `SnapshotSales` type + `getSalesBySnapshot(set, from, to)`
- [x] components/SalesBySnapshot.tsx: collapsible timeline rows (time + units + revenue → sold cards)
- [x] TrackingPage.tsx: mount under the store-selling section, gated `!isAll && dates>=2`, using from/to

## Verify
- [x] `go test ./...`, `go build ./...`, `go vet ./...` — all clean
- [x] `npm run build` (web) — tsc + vite clean
- [x] Real-data check: OP-16's 8 snapshots → 7 intervals newest-first, per-interval
      units/revenue/cards incl. zero-sale intervals, seller breakdowns populated

---

# Sealed-product sales tracking (in progress)

Goal: track sales + price trends of sealed products (booster boxes/packs, collector boxes,
starter decks) alongside singles. Scope confirmed: boxes + boosters + sealed decks (no accessories).

## Discovery (live-probed Liga)
- Sealed catalog: `?view=cards/search&card=categ=<ID>%20searchprod=1&category=products` → HTML tiles
  `prod/view&pcode=<N>&prod=<name>`. Categories: 10=Booster Box, 21=Booster Pack, 28=Collector Box,
  36=Starter Deck (4=Sleeves, 38=Kit excluded).
- Detail `?view=prod/view&pcode=<N>`: `var prod_stock`/`var prod_stores` + imgunid/imgnum atlases —
  IDENTICAL sprite mechanism to singles → full decode reuse. (See memory: opdeals-sealed-tracking.)
- US sealed prices: in TCGCSV cat 68 but dropped by `number==""` filter (tcgcsv.go) → un-filter to value.

## Phase 1 — scraper (DONE, verified live)
- [x] `internal/liga/sealed.go`: `SealedListings()` (4 categories) + `SealedDetail()` + `parseProductStock`
      (reuses rawStock/StoreListing/sprite decode). Live test: 137 products (40 box/40 pack/16
      collector/41 deck), OP-16 box decoded 34 stores w/ qty+price.

## Phase 2 — capture wiring (DONE, build+test pass)
- [x] `capture.go`: split into captureSingles + captureSealed; sealed saved as pseudo-set `SEALED`
      in a DaySnapshot (pcode as Number, min store price as LowBRL). `-track-sealed` flag (default on).
      Gets Sales-by-snapshot + price-movers for free by selecting SEALED in Tracking.
- [x] end-to-end capture verified: live capture wrote SEALED snapshot, 89/137 products with
      decoded per-store stock (OP-11 box R$8850, PRB-01 box R$6999 4 stores/7 units, etc.).

## Phase 3 — US sealed prices (TODO)
- [ ] tcgcsv: keep sealed products (no Number), tag ProductType=sealed, expose for valuation.

## Phase 4 — matching + UI (TODO)
- [ ] name-match BR sealed ↔ US sealed (no shared ID); sealed valuation; UI polish (product images,
      type grouping, maybe surface in Portfolio).

## Separate pages (DONE)
- [x] TrackingPage takes `mode: singles|sealed`. Singles tab excludes SEALED (dropdown + backend
      `trackSets()`/ALL pooling via `withoutSealed`). Sealed tab locks to SEALED, no dropdown/ALL,
      retitled "Sealed products". New "Sealed" tab in App.tsx between Tracking and Portfolio.
- [x] `-capture-sealed-once` flag: sealed-only capture, exits (skips singles scan). Ran live into
      data/tracking/SEALED (89 products, R$1.25M market value) — visible in the running server.

## Global snapshot indicator (DONE)
- [x] `GET /api/tracking/snapshots?limit=N` — distinct recent slot keys across all sets (dir reads
      only, no file loads). `SnapshotIndicator.tsx` in the Header (every tab): pill shows latest
      capturedAt + time-ago; click → dropdown of recent snapshot times, newest tagged "latest".
      Verified live: 9 slots returned, latest = SEALED 2026-07-04 21:24.

## Scheduler (every 6h)
- Existing infra: `-refresh-interval` (deals) + `-track-schedule`/`-track-interval` (singles+sealed
  via capture()). Run: `go build -o opdeals ./cmd/server` then
  `caffeinate -is ./opdeals -track-schedule -track-interval=6h -refresh-interval=6h`.

# Portfolio / trade P&L tracker (new feature)

- New `internal/trades` package: `Trade` model + JSON store (`data/trades.json`, mutex, atomic
  write, CRUD) + pure `BuildPortfolio` valuation (holdings @ target % of live TCG price; sold =
  realized proceeds; USD/BRL per sale). Fallback to `RefUSD` captured at entry if catalog lacks it.
- API: `GET/POST /api/trades`, `PUT/DELETE /api/trades/{id}`, `GET /api/trades/quote` (card
  autofill). Price lookup reuses `compare.USDIndex/MatchKey/EffectiveUSD` over the live snapshot;
  FX from `Snapshot.FXRate` (USD/BRL). Wired store into `api.New` + `cmd/server` (`-trades` flag).
- Frontend: new **Portfolio** tab — KPI strip (Invested/Value/Unrealized/Realized/Total P&L),
  global 85/90/95/100% toggle, add-trade form with card-number/name autofill, holdings + sold
  tables with per-card P&L, margin, sell/delete.
- Seeded `data/trades.json` with the user's 6 real buys.
- Verified: `go build/vet/test`, web build clean; full lifecycle (quote/create/list/sell/delete)
  tested against a throwaway server; seeded portfolio values to +R$2.778 (+41%) at 90%, all 6
  cards matched LIVE catalog prices — matches the earlier manual analysis exactly.
- Decisions (user-confirmed): autofill entry, holdings valued at configurable % of TCG, sells in
  BRL or USD per sale.

# Tracking page UX/UI consolidation

- KPI strip added (revenue moved / units sold / stores selling / top mover); derived client-side
  from the snapshot + trends data so it always matches the sections.
- One shared time-range control (24h/7d/30d/All) now drives sales, movers, and the store
  leaderboard. Removed the buried per-section range picker and the movers' own Daily/Weekly/Monthly
  toggle (range maps to daily/weekly/monthly for the price baseline).
- Merged the duplicate "Hottest cards" + "Sales by snapshot" into one **What's selling** section
  with a Totals | By snapshot toggle (Totals derived client-side via `mergeSnapshotCards`).
- Extracted a shared `SoldCardTile`; deleted `MarketPulse.tsx` (split into `KpiStrip`,
  `SalesSection`, `PriceMovers`); removed unused `getTopSold` client.
- Backend: pooled ALL sold-by-snapshot now honors `from/to` so the shared range applies to All too.
- Verified: `go build/vet/test`, web `tsc + vite build` all clean; API shapes spot-checked against
  the live server for OP-16 (7 intervals w/ sellers) and trends (prevDate + top mover). No browser
  screenshot — the Chrome extension wasn't connected.

# Sales by snapshot — per-interval sold cards

## Review
- Refactored the per-pair sale logic out of `TopSoldCards` into `salesBetween`; the whole-range
  aggregate now = `mergeSales` over each interval's result, so `SalesBySnapshot` and the existing
  "Hottest cards" totals share one source of truth (no double-counting, no behavior drift).
- New endpoint is per-set only (returns empty for `ALL`), matching how the store-selling and new
  timeline sections only render when a single collection is selected.
- Timeline respects the existing from/to range picker; zero-sale intervals show as non-expandable
  "no sales" rows so gaps in activity are visible, not hidden.

---

# Riftbound + Lorcana — full One Piece parity incl. US deals (in progress)

Plan: ~/.claude/plans/recursive-frolicking-hellman.md

- [ ] 1. internal/game/market.go: Market config + group-set mapping funcs (OP/RFT/LOR)
- [ ] 2. internal/game/game.go: Riftbound()/Lorcana() constructors, image regexes, ByID, tests
- [ ] 3. internal/compare: Matcher (zero value = OP), Key/LookupKey, thread through, tests
- [ ] 4. internal/tcgcsv: category + group mapping from game.Market
- [ ] 5. internal/pipeline: Options.Game, per-game sources/matcher
- [ ] 6. internal/api + internal/trades: GameStack.Deals, game-scoped deals routes, hasDeals, gates
- [ ] 7. cmd/server + cmd/opdeals: -rft-*/-lor-* flags, deals stores, staggered refresh, stacks
- [ ] 8. web/src: hasDeals-driven Deals tab, game= on deals calls, brands, Footer, labels
- [ ] 9. Verify: go build/vet/test, npm build, OP regression diff, live captures, deals refresh

## Progress
- [x] 1. internal/game/market.go — Market config + group mappers (identity-default, future-proof)
- [x] 2. game.go — Riftbound()/Lorcana(), image regexes, ByID + tests
- [x] 3. compare Matcher (zero value = OP; set-scoped keys; suffix variants A/S/*) + tests
- [x] 4. tcgcsv — category + abbreviation mapping from game.Market
- [x] 5. pipeline — Options.Game, per-game sources/matcher, mypcards gated OP
- [x] 6. api/trades — GameStack.Deals, game-scoped deals routes, hasDeals, deals-gated USD paths
- [x] 7. cmd/server (-rft-*/-lor-* flags, staggered scheduleDealsRefresh, dealsFX) + cmd/opdeals -game
- [x] 8. web — hasDeals-driven UI, game= on deals calls, brands, Footer, hints, gameHasDeals cache
- [x] 9a. go build/vet/test + npm build green; OP deals output byte-identical vs running server
- [ ] 9b. live verify: rft/lor deals refresh, captures, match spot-checks
- [x] 9b. live verified (2026-07-09):
  - Riftbound deals: 1114 BR listings, 1581 US prices, 1060 matched (95%); 162A/308/308S/117A all resolve to exact TCGCSV products; ROPP↔OPP and OGN-PR↔PR alias mapping works
  - Lorcana deals: 2988 BR listings, 5653 US prices, 2707 matched (91%); Enchanted/Epic + cross-set number disambiguation verified against TCGCSV
  - Riftbound capture: 6 sets + sealed (8 products); sprite decode 98.6–100% qty+price (decoder reused verbatim)
  - Riftbound buyout: fxRate 0.19394 + per-candidate sellUSD/tcgUrl; quote/portfolio in USD
  - Pokemon unchanged: buyout fxRate 0 no sellUSD, quote fxRate 1 from tracking floors
  - OP deals output byte-identical vs running server across 4 filter combos + search
  - Lorcana capture running (400/987 detail cards; Liga throttling heavily — ~1 card/45s with cooldown retries)

## Review
- Deals pipeline generalized via game.Market (TCGCSV category, group→set mapping, number
  normalization) + compare.Matcher whose zero value reproduces OP behavior exactly — legacy
  callers unchanged, guarded by TestMatcherZeroValueIsOnePiece.
- Set-scoped match keys were the crux: OP numbers are globally unique (OP01-025) but
  Riftbound/Lorcana use per-set bare digits, and their variants live in the NUMBER
  (162A alt, 308S signature, Lorcana enchanted 205+), so both new games use an empty variant
  vocabulary — number+set does all the work. Liga 308S ↔ TCG 308*/298 handled by suffix map.
- Per-game deals stores hang off GameStack.Deals (nil = BR-only); all USD gates switched from
  ID=="onepiece" to Deals!=nil, so Pokemon behavior is preserved structurally, and -rft-deals=false
  degrades Riftbound to a Pokemon-style BR-only game.
- Deals refreshes staggered 0/10/20min to avoid three simultaneous full scans.
- Operational note: Liga rate-limits hard with 4 games scraping — captures that took minutes
  for OP take hours for new games when concurrent with deals refreshes. Consider spreading
  capture schedules.
- [x] Lorcana capture done 2026-07-09 12:15: 16 sets + SEALED (48 products), 20,247 store rows, 99.2% decode (DLPC1 outlier 76.7%); LOR13 pre-release skip + DLPC2 empty handled gracefully

# Deals set filter (2026-07-09)
- [x] compare.Options.Set (server-side, exact case-insensitive) + matchesQuery extended to SetCode substring
- [x] dealsResponse.Sets (distinct listing SetCodes) on /api/deals + /api/search
- [x] FE: Set dropdown in Deals filters (All sets + resp.sets), set reset on game switch, set shown beside numbers in table/grid
- [x] Verified: rft set=OGN 347 deals all-OGN, OP set=OP-16 154 deals, OP unfiltered count unchanged (3894), lorcana q=lor9 matches by set code, pokemon zero-resp includes sets:[]

# Fix wrong card images for per-set-number games (2026-07-09)
- Root cause: two number-keyed set-insensitive fallbacks built for OP's globally-unique numbers:
  cardimg byNum cache fallback + api cardPageURL -> tracking.PageURLByNumber cross-set index.
  For LOR/RFT/PKM the same number names a different card per set -> wrong art served AND wrong
  URLs persisted (audited: 14/64 lor, 4/76 rft poisoned entries).
- Fix: game.UniqueCardNumbers (true only for OnePiece) gates both fallbacks
  (cardimg.NewStore param + cardPageURL guard). Deals-grid requests still resolve via the
  validated url query param; misses now 404 (missing art) instead of wrong art.
- Cleanup: deleted poisoned data/tracking-lor|rft/images.json; restarted :8080 with fixed build.
- Verified: LOR9/238 vs LOR10/238 vs LOR11/238 (the user's screenshot case) now three distinct
  correct images; regenerated cache audit 0 mismatches; OP image path unchanged (200).

# Unified update scheduler — single Liga lane (2026-07-09)
- New flags: -schedule (unified lane), -schedule-interval (6h cycle), -schedule-gap (2m between jobs).
- 7 jobs/cycle, strictly sequential: OP deals -> OP capture -> RFT deals -> RFT capture -> LOR deals
  -> LOR capture -> PKM capture. Deals jobs skip when snapshot younger than the cycle interval.
- With -schedule on: legacy per-store refresh tickers and -track-schedule are suppressed (Load only).
- Rationale: Liga rate limit is shared across its sites; 7 independent throttled clients overlapping
  caused 429 cascades (rft capture 9h vs ~15min solo). One scan at a time = full polite budget each.
- Verified: dry-run showed sequential order + gaps + freshness skips + cycle summary; legacy path
  unchanged; production :8080 restarted with -schedule (run: caffeinate -is tmp/server -web= -schedule).

# Dockerized deployment (2026-07-09)
- Dockerfile: 3-stage (node:26-alpine web build -> golang:1.26-alpine CGO_ENABLED=0 -> alpine:3
  + ca-certificates + tzdata); serves baked web/dist; ENTRYPOINT opdeals-server, CMD -web=web/dist -schedule.
- docker-compose.yml: port 8080, ./data bind mount, TZ=America/Sao_Paulo, restart unless-stopped.
- .dockerignore keeps data/tmp/node_modules out of the build context.
- Verified: image builds; smoke test on temp port (API+FE 200); switched production to the container
  (host tmp/server stopped) — scheduler cycle running inside, real snapshots loaded (6283 listings).

# Feature Orçamento — quote builder de compra (2026-07-10)

Plan: ~/.claude/plans/concurrent-swinging-pretzel.md

- [x] 1. internal/quotes/quotes.go — tipos Quote/Item + store JSON (clone trades)
- [x] 2. internal/compare/compare.go — BRIndex exportado
- [x] 3. internal/api/trades.go — ligaLowBRL/ligaAvgBRL no quoteMatch + join
- [x] 4. internal/api/quotes.go — CRUD handlers
- [x] 5. api.go (GameStack.Quotes + rotas) e main.go (flags + buildStack)
- [x] 6. web/src/api.ts — tipos + fetchers
- [x] 7. web/src/components/QuotePage.tsx — lista + editor + WhatsApp
- [x] 8. web/src/App.tsx — aba Orçamento
- [x] 9. Verificação: go build/vet/test, npm run build, curl CRUD, UI walkthrough

## Review
- Search reuses GET /api/trades/quote with additive ligaLowBRL/ligaAvgBRL (join via
  compare.BRIndex + Matcher.Key = deals semantics; alt-art TCG rows land on Liga alt listings).
  Verified live: OP06-118 variants each carry distinct Liga floors; Portfolio autofill untouched.
- Quotes are offer documents: item prices frozen at add-time, unitBRL user-editable, pct on the
  quote; totals client-side. Store = trades.Store clone (atomic tmp+rename), files data/quotes*.json
  per game via -quotes/-pkm-quotes/-rft-quotes/-lor-quotes flags.
- CRUD verified via curl: create (number normalized), list (UpdatedAt desc), full-replace PUT
  preserving createdAt, delete, 404 on missing id, 400 on empty number+name item.
- UI: aba Orçamento (?tab=orcamento) — lista de orçamentos salvos + editor com autocomplete
  (padrão AddTradeForm), qty/unitário editáveis, presets 50/60/70/100%, footer total→oferta,
  Copiar WhatsApp em PT. BR-only games escondem a coluna TCG.
- go build/vet/test + npm run build (tsc) limpos. Browser walkthrough pendente (extensão Chrome
  desconectada); servidor de teste em :8123 com o build novo pra validação manual.
- Busca reordenada por relevância (quoteRank: exato=0 → prefixo=1 → contains número=2 → contains
  nome=3; USD desc só como desempate) + ?limit= (default 25, cap 100; QuotePage pede 50) — corrige
  carta barata sumindo do dropdown quando o termo batia em 25+ cartas caras ("uta" → ST23-001 agora
  em 2º). Portfolio inalterado (default 25, ganha o ranking de graça).
- Imagens no orçamento via TCGplayer CDN (product-images.tcgplayer.com/fit-in/200x279/{productID}.jpg):
  quoteMatch/QuoteItem carregam productID; CardArt ganhou cadeia de fontes productID→liga→caixa.
  Corrige (a) carta sem imagem — variantes TCG-only e cartas fora do tracking (< track-min-price,
  DON sem número) e (b) imagem ERRADA — o fallback byNum da Liga servia arte da versão base
  (ou SAMPLE) pra variantes SP/AA que compartilham o número. ProductID é único por variante.
- Links por item do orçamento: pills "Liga" (quoteMatch/Item.LigaURL, da listagem joinada — também
  populado no path BR-only via c.URL) e "TCG" (tcgProductURL(productID)); TCG oculto em BR-only,
  pill só aparece quando a fonte existe. Itens salvos antes do campo não têm ligaUrl (re-adicionar
  a carta popula).
- Join de precisão na busca (Matcher.PrintKey + BRPrintIndex, fallback pro Key): prints especiais
  fora do VariantVocab ("Premium Card Collection -Best Selection-", "Anniversary Set", promos)
  caíam na listagem BASE da Liga (link+preço errados — caso Atmos OP08-040 vs OP08-040-BS).
  PrintKey soma os parênteses extras do nome (traços normalizados: TCG escreve "-Best Selection-",
  Liga sem traços; ignora números puros e card codes tipo OP08-040-BS). Deals pipeline intocado;
  só handleTradesQuote usa. Verificado: Atmos BS → R$62,90; OP06-118 SJ/AA/RE agora acham as
  listagens suffixadas próprias da Liga.
- Follow-up (mesmo dia): seletor "Mercado base" (TCGplayer | Liga) por orçamento — Quote.Market
  ("tcg" default, clamp no normalize), seedUnitBRL escolhe a fonte do unitário sugerido e trocar
  o mercado re-seeda todas as linhas (fallback pra outra fonte quando falta preço). BR-only trava
  em liga e esconde o toggle. Orçamentos antigos sem o campo viram "tcg" no próximo save.

# Phantom deals fix — verified-only by default (2026-07-10)
- Bug: deals list showed cards with zero Liga sellers ("Ops! Nenhum item encontrado" on click).
  Root cause: the set-grid cardsjson p1a/p1b/p1c are marketplace SALE stats that exist even with
  no current listings (phantom signature p1a==p1b==p1c, e.g. LOR6/217 Gadget Enchanted R$185,89);
  the grid has no stock field, and only candidates selling >= verify-floor ($100) were ever
  stock-verified, so everything below $100 passed through unverified at the phantom price.
- Fix: verify-floor default 100 -> 10 (cmd/server + cmd/opdeals; candidate counts at $10:
  OP ~240, RFT ~50, LOR ~31 — a few extra minutes in the serialized Liga lane); API
  requireInStock default false -> true (matches cmd/opdeals, which already required in-stock
  when verification ran); FE verifiedOnly default true ("Verified stock only" toggle remains
  to opt out).
- Verified: go build/vet/test clean; web build clean; compare.Deals on the cached LOR snapshot
  at minPrice=20 returned 13 deals ALL sellers=0 before vs 0 after; container rebuilt+deployed,
  API confirms phantoms hidden. Sub-$100 verified deals repopulate on the next successful scan
  (Liga was blocking at deploy time; scheduler retries each 6h cycle).

# Sidebar shell + black/accent re-theme (2026-07-12)

Plan: ~/.claude/plans/cheeky-twirling-snowglobe.md

- [x] 1. index.css — @theme palette remap (neutral slate + accent family), body bg/text, fade-in, scrollbar
- [x] 2. brand.tsx — unified-accent brand config + typed nav groups/items
- [x] 3. components/ui/sidebar.tsx — provider + primitives (collapsible rail, mobile drawer, tooltip)
- [x] 4. components/ui/button.tsx — primary variant emerald → accent
- [x] 5. components/PageHeader.tsx — shared page heading
- [x] 6. components/AppSidebar.tsx — brand + game switcher + grouped nav + footer status
- [x] 7. components/TopBar.tsx — hamburger + status pills + snapshot + refresh
- [x] 8. App.tsx — SidebarProvider/AppSidebar/SidebarInset/TopBar shell
- [x] 9. Verify: npm run build clean + headless-Chrome screenshots (desktop/mobile/collapsed/deals/portfolio)

## Review
- Two-column shell: collapsible icon-rail sidebar (grouped nav Discover/Market/Inventory, brand +
  game switcher + live-status footer, localStorage-persisted collapse) + off-canvas mobile drawer;
  slim sticky TopBar (FX/Updated pills + snapshot + refresh); shared PageHeader per view. Zero view
  internals changed — only App.tsx shell + 4 new components + brand/nav config extracted from App.
- Re-theme done centrally in index.css via Tailwind v4 @theme: neutralized the slate ramp to a
  near-black canvas (#08080a) with white/neutral ink, defined an electric-indigo `accent` family,
  and pointed sky/violet/fuchsia/indigo at it so all 4 per-game colors + interactive accents unify
  with no per-file edits. emerald(gain)/rose(loss)/amber(live) kept as data semantics. Accent is a
  one-line swap (5 --color-accent-* hexes). Only component color edit: button primary emerald→accent.
- Verified: tsc+vite build clean; headless screenshots confirm desktop (expanded+collapsed), mobile
  drawer topbar, Deals (search/filters/grid), Portfolio (KPIs+holdings) all cohesive. Chrome MCP
  extension was disconnected so used `Google Chrome --headless --screenshot`.
- Known minor: PageHeader "Portfolio" sits above the PortfolioPage's own "Portfolio" heading (app
  title + section subtitle). Left as-is per "view internals untouched"; tr­im in a follow-up if wanted.
