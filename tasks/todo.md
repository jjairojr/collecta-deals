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

# MyP re-populate + prod deploy (2026-07-24)

Today's snapshots were re-scraped by the local `-schedule` Docker container → all MyP dropped
(OP + RFT both 0 mypcards). Task: run MyP for both, then single prod deploy of enriched snapshots.

- [x] Build host `opdeals-server` binary to scratch
- [x] Throwaway FlareSolverr on host :8191 (existing FS not host-published)
- [x] Run 1 — OP (~16m): 9877 listings = 5822 ligaonepiece + 4055 mypcards
- [x] Run 2 — RFT (~16m): 2366 listings = 1307 ligariftbound + 1059 mypcards (OP held idle via ready-copy)
- [x] Verified both scratch snapshots: liga + mypcards sources, fresh updatedAt (16:30 / 16:47)
- [x] Backed up (`.bak-nomyp-1648`) + atomically swapped into data/snapshot.json + data/snapshot-rft.json
- [x] `scripts/sync-up.sh -s` → uploaded all 4 snapshots to prod volume + reload `{"reloaded":4}`
- [x] Verified prod: OP deals 100% mypcards; RFT deals 74 mypcards + 26 liga
- [x] Cleanup: dev servers killed, `opdeals-fs-dev` removed
- [ ] (optional, NOT done — user's running local :8080) `docker restart opdeals` so local also serves MyP

# Vitrine pública (storefront) — plano (2026-07-24)

Objetivo: um site público que o usuário manda o link e a pessoa vê o estoque
(baseado no portfolio/holdings), monta um carrinho e finaliza no WhatsApp.

## Decisões (fechadas com o usuário)
- Site **separado em Next.js**, hospedado na **Vercel** (isolamento total do admin;
  SSR dá preview de link bonito no WhatsApp).
- Go continua **fonte da verdade**: expõe API read-only, Next só lê (sem migração de DB;
  "compartilhar o DB" = consumir a API, não acoplar no storage JSON).
- **Catálogo unificado** (todos os jogos num link, filtrável por jogo/set/busca).
- Comprador: **carrinho → WhatsApp** (marca várias, manda tudo de uma vez).
- Limite de segurança: o JSON público **nunca** carrega custo/lucro.

## Fase 1 — Backend Go: `GET /api/storefront`
- [ ] `internal/api/storefront.go` — handler espelhando `handlePortfolioAll`
      (itera `orderedGameIDs()`, `gs.Trades.List()`, filtra `Status=="holding" && Qty>0`,
      `priceLookup(gs)` p/ market). Preço venda = market × pct (default 90).
- [ ] Struct de resposta **dedicada** (NÃO `TradeView` — ele tem `costBRL`/lucro):
      `storefrontItem{game,gameLabel,number,name,set,variant,condition,qty,productID,
      imageURL,askUSD,askBRL,marketKnown}` + `{fxRate,updatedAt,pct,items[]}`.
- [ ] Arte: URL pública direta — productID → product-images.tcgplayer.com; fallback
      `{GO}/api/card-image?...` (img cross-origin carrega sem CORS).
- [ ] Rota + middleware CORS (env `STOREFRONT_ORIGIN`) em /api/storefront (+ /api/card-image).
- [ ] Funciona em serve-only (GET). Teste garantindo ZERO campo de custo na resposta.

## Fase 2 — Frontend Next.js (`storefront/`)
- [ ] Scaffold Next App Router + TS + Tailwind. Env: API URL + número WhatsApp.
- [ ] `/` Server Component (revalidate 60): grid + filtros (jogo/set/busca) + toggle US$/R$.
- [ ] Carrinho (context + localStorage) → botão "Finalizar no WhatsApp" (wa.me pré-preenchido).
- [ ] OG meta tags (preview no WhatsApp). Design dark alinhado ao app.

## Fase 3 — Deploy + verificação
- [ ] Deploy Vercel apontando pro Railway; setar STOREFRONT_ORIGIN no Railway.
- [ ] Verificar catálogo/filtros/carrinho/WhatsApp/preview e que NENHUM custo vaza (Network).

## Fora do escopo v1
- Pedido gravando no banco, reserva/checkout/pagamento, preço por-carta international em US$ (v1 guarda R$, deriva US$ pelo câmbio).

## Review (2026-07-24)
- **Modelo de preço mudou** (pedido do usuário): preços definidos manualmente numa
  aba "Estoque" no app atual, não % global automático. Campos `AskBRL`+`Listed` no
  `trades.Trade`; uma carta entra na vitrine só quando `holding && Listed && AskBRL>0`.
- **Backend:** `internal/api/storefront.go` — `GET /api/storefront` (struct dedicada
  storefrontItem, SEM custo/lucro — testado em storefront_test.go) agrega holdings de
  todos os jogos, deriva askUSD por um FX único (primeiro jogo com deals). Arte: tcg CDN
  via productID do tcgUrl, senão fallback card-image. `POST /api/trades/listings` (batch)
  + `trades.Store.SetListings`. cors global já é `*`, sem trabalho extra.
- **Aba Estoque:** `web/src/components/StockPage.tsx` (nav "Estoque" em brand.tsx/App.tsx).
  Tabela de holdings: preço R$ editável, toggle à venda, "Sugerir %" (do TCG/floor),
  "Listar/Ocultar todas", Salvar em batch. api.ts: askBRL/listed em Trade + setListings().
- **Vitrine Collecta:** `storefront/` — Next 15 + React 19 + Tailwind v4, marca Collecta
  (rosa #F6559B, azul #1355B3, rosa-claro #FDC4E5, fundo #141416). SSR fetch server-only
  (`STOREFRONT_API`, host do Go NÃO vaza no browser); `/img` proxy pra arte fallback.
  Catálogo unificado filtrável (jogo/set/busca), toggle R$/US$, carrinho localStorage-less
  (client state) → "Finalizar no WhatsApp" (`NEXT_PUBLIC_WHATSAPP` placeholder). OG tags.
- **Verificado:** go build/vet/test ./... limpo; web `npm run build` limpo; storefront
  `next build` limpo (5 rotas). Screenshot ao vivo pendente (precisa marcar cartas à venda
  + rodar os 2 servers; não mexi nos trades reais do usuário).
- **Pendente pra ir ao ar:** deploy `storefront/` na Vercel (env STOREFRONT_API →
  Railway, NEXT_PUBLIC_WHATSAPP → número real), e o usuário marca as cartas na aba Estoque.

# Collecta Marketplace — arcade redesign (design handoff) (2026-07-24)

Recriar as 5 telas do handoff `storefront/design/design_handoff_collecta_marketplace`
(Home, Browse singles, Página da carta, Selado, Carrinho) no stack existente.
Decisões (fechadas com o usuário): **design-first tudo mock**, **rotas reais Next
(App Router)**, checkout continua **WhatsApp**.

## Fundação
- [x] globals.css — tokens arcade exatos (#0b0b0c outline, #1f1f22 surface, text sec/ter,
      on-light-pink), helpers sticker/arcade-press/hard-shadow, keyframes blink/bob/rise,
      scanlines, prefers-reduced-motion, overflow-x guard
- [x] next/font — Baloo 2, Press Start 2P, DM Sans como CSS vars
- [x] lib/mock.ts (games, featured singles, selados, sellers, price history, high scores),
      extend lib/types.ts, lib/cart.tsx (context + localStorage), money() em centavos
- [x] layout.tsx — chrome persistente: AnnouncementBar, Header (busca+SALDO+CARRINHO),
      TabBar (links de rota), Footer, Scanlines

## Entrega 1 (comece aqui — Home + card de single) — FEITO
- [x] Componentes: SingleCard, SealedCard, Stepper, ArtPlaceholder, SectionHead, Container,
      GameCabinet, HighScores, ComingSoon (placeholder das rotas ainda-não-feitas)
- [x] Home (/): hero, game select, singles em destaque, produto selado, HIGH SCORES
- [x] Verificado: next build limpo (8 rotas), screenshot desktop 1440 + small 500 sem overflow
      (headless piso em 500px; <390 usa flex-wrap). Storefront.tsx antigo removido.

## Telas restantes — FEITO
- [x] /singles (SinglesBrowse): sidebar de filtros (JOGO/TIPO/ESTADO/IDIOMA/RARIDADE + track
      de PREÇO), toolbar com sort (RELEVANCIA/MENOR PRECO/NOVIDADES), grid 4-col, paginação.
      JOGO/ESTADO/IDIOMA/busca/sort funcionam; RARIDADE+PREÇO são visuais (design-first).
      useSearchParams (jogo/q) sob <Suspense>. Filtros viram sheet inline no mobile (botão FILTROS).
- [x] /carta/[slug] (SingleDetailView): galeria (frame rosa + 4 thumbs), badges, buy box
      (preço+parcelas+stepper+ADICIONAR+♥), seller bar, histórico de preço (6 barras), outros
      vendedores. generateStaticParams sobre CATALOG (16 cartas). Fallback buildSingleDetail
      pra qualquer slug.
- [x] /selado (index grid) + /selado/[slug] (SealedDetailView): frame rosa-claro, badges
      PRE-VENDA/LACRADO, buy box com barra de estoque, spec grid 2×2. Fallback buildSealedDetail.
- [x] /carrinho (CartView): line items + stepper + remover, nudge de frete (< R$250), resumo
      (subtotal/frete/cupom ARCADE10 -10%/total), botão double-ring INSERIR FICHA → WhatsApp.
      Carrinho persiste em localStorage (lib/cart).

## Verificação final
- [x] next build limpo — 5 rotas + 16 /carta + 4 /selado prerender (SSG), lint ok
- [x] Screenshots desktop 1440 de todas as telas; cart com itens + cupom aplicado (−R$158,30);
      barras do histórico renderizam (fix: % height precisa de flex item com altura definida)
- [x] Checkout = WhatsApp handoff (decisão do usuário); cupom/frete/pagamento são visuais

## Ligar catálogo REAL de singles (2026-07-24) — FEITO
- [x] `lib/catalog.ts` (server-only): fetch `${STOREFRONT_API}/api/storefront`, mapeia
      storefrontItem→Single (askBRL reais → centavos, slug estável game-name-number-variant-cond,
      dedup somando qty, grade derivada de condition PSA*), **fallback pro mock** se a API não
      responder JSON (prod ainda serve o SPA em /api/storefront → cai no mock). `loadSingleDetail`
      (curated → build sobre catálogo real → 404).
- [x] `lib/games.ts`: hue/pixel generalizados p/ 5 jogos (pokemon/onepiece/riftbound/lorcana/gundam);
      `GameId` virou `string`. Removidos GAMES/GAME_LABEL/GAME_PIXEL/singleBySlug (mortos).
- [x] Home + /singles (server) fazem loadCatalog (revalidate 60); jogos derivados dos dados
      (eyebrow "N UNIVERSOS", cabinets por jogo em estoque). /carta/[slug] agora **dinâmico**.
      SinglesBrowse recebe catalog+games por prop; ESTADO derivado das condições reais, IDIOMA
      só aparece se houver idioma (backend não manda), JOGO por jogo em estoque.
- [x] Selado/sellers/histórico/high-scores **continuam mock** (backend não fornece).
- [x] Verificado com backend fake local (shape exato do Go): 12 itens/5 jogos → Home mostra
      "5 UNIVERSOS" + 5 cabinets com contagem; Browse ESTADO=LP/MP/NM/PSA10/PSA9 (MP veio dos
      dados, não do hardcode); /carta/lorcana-... e /carta/gundam-... resolvem (5º jogo ok);
      preço em centavos correto; imagem via /img proxy carrega. next build limpo com env real.
- **Pendente pra ver dados reais no ar:** deploy do backend Go com a rota /api/storefront (prod
  atual é build antigo, serve o SPA) — aí o site mostra o estoque real automaticamente.

# Reskin arcade do dashboard (web/) (2026-07-24)

Reskin visual do dashboard interno (`web/`, React+Vite+Tailwind v4) para a linguagem
"arcade" da vitrine (`storefront/`). É RESKIN: preservar 100% dos dados, chamadas de API
e lógica — só a camada visual muda.

## Decisões (aprovadas com o usuário)
- Shell: manter sidebar+topbar (reskin, não reescrever navegação).
- Escopo: fundação + todas as views.
- Tabelas densas: SUTIL — sticker completo só em KPI/botões/painéis; linhas com borda fina 2–3px, sombra ~0.
- Tokens: EXTRAIR para arquivo CSS comum importado pelos dois apps (evita divergência).

## Regras de estilo
- Press Start 2P só em rótulos curtos e ACCENT-FREE (KPI caption, eyebrow, badge). Nunca em
  números/células/texto longo, nunca em label com acento ("Orçamento" fica em Baloo/DM Sans).
- Sticker = cor sólida + borda 4–5px `#0b0b0c` + sombra dura `Xpx Ypx 0` (sem blur), via `--sh`.
- Semânticas (verde/ganho, vermelho/perda, âmbar/live) MANTIDAS distinguíveis, ganhando borda ink.
- Focus rings nunca removidos. Respeitar prefers-reduced-motion. Scanlines OFF por padrão.

## Fase 0 — Fontes + tokens compartilhados
- [ ] `@fontsource/baloo-2` (600/700/800), `@fontsource/dm-sans` (400/500/700),
      `@fontsource/press-start-2p` (400) no `web/package.json`; importar pesos em `web/src/main.tsx`.
- [ ] Criar `shared/arcade.css` na raiz: `@theme` (brand/brand-soft/royal/ink/surface/outline/
      muted/faint/on-soft + aliases de fonte) + classes à mão (sticker, sticker-5, arcade-press
      +hover/active, font-display, font-pixel, logo/hero-shadow, art-stripes-*, scanlines,
      keyframes blink/bob/rise + animate-*, focus-visible, scrollbar).
- [ ] Refatorar `storefront/app/globals.css` p/ importar o comum (remover bloco duplicado).
- [ ] Reescrever `web/src/index.css`: importar tailwindcss + comum; vars cruas de fonte;
      remapear `accent-*`→rosa e `slate-*`→neutros arcade (mantendo aliases sky/violet/fuchsia);
      body→DM Sans; canvas→ink.

## Fase 1 — Primitivos (cascateiam p/ todas as views)
- [ ] `ui/card.tsx` sticker · `ui/button.tsx` arcade-press · `ui/badge.tsx` chip 3px
- [ ] `ui/input.tsx` + `ui/select.tsx` sticker flat · `ui/table.tsx` sutil
- [ ] `ui/tabs.tsx` · `ui/toggle-group.tsx` · `ui/checkbox.tsx` arcade

## Fase 2 — Shell
- [ ] `ui/sidebar.tsx` · `AppSidebar.tsx` · `TopBar.tsx` · `PageHeader.tsx` · `App.tsx` Footer

## Fase 3 — Polish por view
- [ ] `KpiStrip` · Deals (Page/Table/Grid/Depth) · Tracking (Sales/Movers/Leaderboard/TrendTable/
      InventoryPanel) · Portfolio + AllPortfolio · Quote · Stock · `ui/chart.tsx` chartColors ·
      restos (CardArt, SelectionTray, Browse, Buyout, SnapshotIndicator, ShareList, SoldCardTile).

## Fase 4 — Verificação
- [ ] `npm run build` (web) limpo · `npm run build` (storefront) limpo · screenshots headless de
      cada view · sem comentários · não commitar.

## Review (2026-07-24)
- **Fundação:** `shared/arcade.css` na raiz (tokens `@theme` brand/royal/ink/surface/outline +
  aliases de fonte + classes sticker/arcade-press/art-stripes/scanlines/keyframes) — importado
  por `storefront/app/globals.css` (bloco duplicado removido) e `web/src/index.css`. Fontes no
  dashboard via `@fontsource` (baloo-2 600/700/800, dm-sans 400/500/700, press-start-2p 400)
  importadas em `main.tsx`. `web/src/index.css` remapeia as rampas legadas `accent-*`→rosa e
  `slate-*`→neutros arcade (mantendo aliases sky/violet/fuchsia→accent), então TODA classe
  existente (`bg-slate-900`, `text-slate-400`, `bg-accent-500`…) já renderiza arcade sem editar
  componente. emerald(ganho)/rose(perda)/amber(live) preservados como os 3 sinais de dado.
- **Primitivos** (cascateiam p/ tudo): Card (sticker sutil), Button (arcade-press por variante),
  Badge (chip borda ink), Input/Select (sticker flat), Table (sutil: header muted, linha borda
  fina), Tabs/ToggleGroup/Checkbox (ativo rosa), Sidebar (borda ink, item ativo rosa).
- **Shell:** AppSidebar (logo Baloo + tile royal, game switcher pixel, labels DISCOVER/MARKET/
  INVENTORY em pixel), TopBar (pills arcade + Refresh rosa), PageHeader (título Baloo + tile
  sticker), Footer borda ink.
- **Views:** KpiStrip + KPIs de Portfolio/Stock/Inventory viraram sticker completo (tile rosa,
  caption pixel EN / DM Sans uppercase p/ pt-BR com acento, número tabular). Charts (`ui/chart.tsx`)
  recoloridos (série rosa, mantendo emerald/rose; tooltip sticker). Barras de margem/lift, badges
  de delta, chips de link e tabelas cruas (Portfolio/Quote/Stock) reskinnados sutis. Regra da
  Press Start 2P respeitada: pixel só em rótulos curtos accent-free.
- **Verificação:** `npm run build` (tsc+vite) limpo em `web/` E `storefront/`. Screenshots headless
  1440px de Deals/Tracking/Portfolio/Estoque/Orçamento/Buyout/AllGames — shell arcade coeso, dados
  reais (via vite :5173 → :8080). Sem comentários no código; não commitado.
- **Nota:** `marginTier.ring`/`liftTier.ring` viraram campos de objeto não-usados (não são
  variáveis soltas; TS não acusa) — deixados colocados com o resto da config de tier.
