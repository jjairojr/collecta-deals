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

# Acessórios na vitrine e no dashboard (2026-07-27)

Plan: ~/.claude/plans/agile-doodling-cook.md — novo `Trade.Kind == "accessory"` reusando os trilhos do sealed.

- [x] Go: guards de kind manual (valuation.go, trades.go, storefront.go, main.go heldCardKeys)
- [x] Dashboard: tipo kind em api.ts
- [x] Dashboard: seção Acessórios no PortfolioPage (toggle, filtro, form generalizado, tabela)
- [x] Dashboard: estrela de destaque para acessórios no StockPage
- [x] Vitrine: types.ts + catalog.ts (bucket accessories, mapAccessory, detail)
- [x] Vitrine: rotas /acessorios e /acessorio/[slug]
- [x] Vitrine: generalizar SealedCard/Grid/DetailView
- [x] Vitrine: aba ACESSÓRIOS no TabBar (condicional)
- [x] Vitrine: seção Acessórios na home
- [x] Vitrine: carrinho hue + sitemap
- [x] Verificação: go build/vet/test, npm build web e storefront, e2e com server descartável

## Review
- Acessório = `Trade.Kind == "accessory"` nos mesmos trilhos do sealed: nenhum endpoint ou campo
  novo no Go. Os guards `Kind != "sealed"` viraram `Kind == ""` (kind vazio = single), então
  qualquer kind manual pula lookup de mercado, vale `ManualBRL × Qty`, sai do live-pricing
  (heldCardKeys) e depende só do `imageURL` do vendedor na vitrine.
- Dashboard: Portfolio ganhou toggle Singles/Sealed/Accessories; `AddSealedForm` virou
  `AddManualForm` (prop kind; acessório esconde a busca no catálogo Liga e grava `set:"ACESSORIO"`);
  `SealedTable/Row` viraram `ManualTable/Row` (compartilhadas); flag `sealed` do EditTradeForm
  renomeada `manual`. Estrela de destaque no Estoque agora aceita sealed E accessory.
- Vitrine: `Accessory`/`AccessoryDetail` espelham Sealed via Omit; bucket `accessories` no
  catalog (dedupSealed generalizado em `dedupProducts<T>`); SealedCard/Grid/DetailView aceitam
  a união e derivam href (/selado vs /acessorio), badge (LACRADO vs ACESSORIO) e wording
  (CAIXAS vs UNIDADES) do kind. Aba global ACESSORIOS no TabBar (só quando há estoque, decisão
  do usuário), página /acessorios agregando todos os jogos, detalhe /acessorio/[slug] com
  JSON-LD próprio, seção na home (destaques via estrela, fallback 4 primeiros), sitemap.
- Verificado: go build/vet/test limpos; npm build limpo em web/ e storefront/; e2e com server
  descartável no scratchpad (POST /api/trades kind accessory → valueBRL 200 = 40×5 sem lookup;
  /api/storefront expõe kind/askBRL/featured sem custo; vitrine dev renderizou aba, lista,
  detalhe "RESTAM 5 UNIDADES" e home). /selado e /singles sem regressão.

# Lote Ikenson refeito para R$ 13.000 (2026-07-27)

- [x] `ikenson-novo.csv`: preços proporcionais (fator 13000/19316 = 0,673017), % vs Liga recalculado
- [x] 135 lotes adicionados ao portfolio Riftbound (`store: Ikenson`, buyDate 2026-07-27)

- **Resultado:** soma exata R$ 13.000,00 (365 cartas). Arredondamento a 2 casas deixava residual de
  -R$ 0,43, corrigido com ajustes de 1 centavo nas linhas de maior erro de arredondamento, então
  `sum(qtd × unit)` fecha em 13.000,00 sem depender de linha de total.
- **Verificação:** `GET /api/trades?game=riftbound` → 135 holdings Ikenson, qty 365, costBRL 13000,0,
  marketKnown 135/135 (todos casaram com preço TCGplayer, refUSD preenchido). Portfolio total foi de
  60 para 195 holdings.
- **Gotcha:** o id do jogo na API é `riftbound`, não `rft` (o `rft` é só o sufixo do arquivo). Com
  `?game=rft` o `stackFor` cai silenciosamente no jogo default (One Piece) e grava no portfolio errado.
- [x] Merge dos 14 lotes que duplicavam cartas já existentes (195 → 181 holdings). Custo médio
  ponderado preserva a base; drift de arredondamento do `round2` no unitário: −R$ 0,01 no total.
  O primário do merge é o lote **mais antigo**, então esses 14 perderam a tag `store: Ikenson` e a
  buyDate virou 2026-07-21 (restam 121 lotes marcados Ikenson, R$ 11.038,11).
- [x] Vitrine: 135 lotes listados a **piso Liga EN × 1,10** (R$ 25.200,30 pedido, 395 unidades).
  Fonte do piso: `data/tracking-rft/<SET>/2026-07-25T00.json` `lowBRL` (NM + idioma 2 + em estoque,
  via `liga.FloorNM`) — **não** o `LowBRL` das listings do snapshot de deals, que inclui idioma 10
  (chinês) e por isso subestima o piso em 29 das 135 cartas (ex.: Zenith Blade OGN 262 → 49,90
  chinês vs 69,00 EN). A coluna `Liga_unit_BRL` do CSV do Ikenson vem dessa fonte contaminada.
- [x] **Replicado no PROD** (Railway `collecta-deals`) via API, não por arquivo. O prod estava muito
  à frente do local (14 vendas, 32 listagens, correções de unitário, 4 lotes inéditos), então
  `sync-up` dos trades teria destruído tudo isso — o runbook já marca `trades*.json` como
  prod-owned justamente por isso. Sequência: 135 POST /api/trades → 14 POST /api/trades/merge →
  122 itens em POST /api/trades/listings.
  Decisão do usuário: nas 14 duplicadas, **preservar o preço já definido no prod** (piso cravado);
  só as 121 novas + Pyke (que não tinha preço) entraram a piso×1,10. O merge preserva isso sozinho,
  porque o primário é o lote mais antigo — verificado lote a lote, 0 preços antigos alterados.
  Vitrine prod: 221 → 343 itens (riftbound 31 → 153). One Piece e Pokémon intocados.
  Usuário optou por NÃO rodar sync-down, então local e prod seguem divergentes de propósito.
- [x] Prod: Thrill of the Hunt mesclado (3+8=11 a R$ 100). O lote antigo estava com **número errado**
  (`UNL 114` = Nidalee - Cat Form), por isso não casava com o do Ikenson (`UNL 184`) na dedupe por
  (set, número) e era avaliado contra a carta errada (US$ 0,70 em vez de US$ 2,10). Corrigi o número
  via PUT antes de mesclar — o merge mantém a identidade do primário, então mesclar antes teria
  perpetuado o `114`.
- **Não mesclados (não são duplicatas):** os 6 grupos de runas repetem só o NOME — cada set é uma
  impressão distinta com productID e preço próprios (Fury Rune Alt: OGN 007A pid 652778 US$3,16 /
  SFD R01A pid 678224 US$3,42 / UNL R01A pid 692932 US$5,96). Mesclar destruiria a identidade de set.
- **Pendente de decisão:** lote `OGN-PR 58 Lillia - Protector of Dreams` (R$ 65/un, sem preço e sem
  arte). OGN-PR não tem número 58 na Liga; o TCG só tem Lillia em UNL 058/058A e **ROPP 058** (duas
  impressões promo, US$ 6,29 e US$ 118,66). É provável erro de set, mas o preço pago descarta que
  seja a UNL 058 (piso R$ 10,82) — precisa o usuário dizer qual promo é.

---

# Acessórios "não apareceram" — diagnóstico (2026-07-27)

Sintoma: usuário não via a aba de Acessórios no dashboard nem na vitrine, apesar do commit
b8a44ce ("acessorios no portfolio/estoque e na vitrine").

## Causa
- **Dashboard:** `localhost:8080` é o container Docker, e o Dockerfile assa `web/dist` na imagem
  (só `./data` é volume). O container estava de pé havia 45h servindo `index-D0YpYjbW.js`
  (0 ocorrências de "Add accessory"); o build local `index-Pmm29pWl.js` já tinha a aba.
  Fix: `docker compose up -d --build opdeals` → :8080 passou a servir o bundle novo.
- **Vitrine:** já estava no ar (`/acessorios` responde 200 em www.collectatcg.com.br, com
  "NENHUM ACESSORIO DISPONIVEL"). A aba ACESSORIOS no TabBar é condicional
  (`showAccessories = accessories.length > 0`) e o prod Railway serve 338 itens — 326 singles,
  12 selados, **0 acessórios** —, então a aba fica escondida até existir acessório listado.

## Verificação end-to-end (local, com limpeza)
- POST /api/trades (kind=accessory) → POST /api/trades/listings (askBRL, listed, featured, imageURL)
  → GET /api/storefront devolveu o item com `kind: "accessory"`, `featured: true`, askUSD convertido.
- Item de teste deletado depois (DELETE /api/trades/{id}); `data/trades.json` limpo, 0 acessórios.

## Pendente (decisão do usuário)
- Cadastrar os acessórios reais no Portfolio → Accessories e listá-los no Estoque.
- Replicar no prod (o prod é dono dos trades — usar API, não sync de arquivo) para a aba aparecer
  na vitrine pública.

---

# Acessórios independentes de jogo (2026-07-27)

Antes: um acessório era gravado no ledger do jogo que estivesse selecionado
(`data/trades*.json`), então sleeves cadastradas em One Piece sumiam na aba do Pokémon.

## Modelo novo
- Ledger único `data/accessories.json` (flag `-accessories`), fora do `GameStack`,
  guardado em `Server.accessories`.
- **Roteamento por kind, não por parâmetro:** `POST /api/trades` com `kind:"accessory"`
  grava no ledger compartilhado. O front não precisou de nenhum parâmetro novo.
- **Escritas por id:** `onLedger()` tenta o ledger do jogo e cai para o de acessórios
  quando dá `ErrNotFound` — ids são únicos e toda mutação (Update/Sell/Delete/Merge)
  devolve ErrNotFound *antes* de persistir, então a tentativa é gratuita. Editar/vender/
  excluir um acessário funciona a partir de qualquer jogo.
- `GET /api/trades?game=X` devolve o ledger do jogo **+** os acessórios (ordenado por
  data), então Portfolio e Estoque mostram os mesmos acessórios em todos os jogos.
- `POST /api/trades/listings` grava nos dois ledgers (cada um ignora ids que não são
  seus) — o Estoque salva cartas e acessórios num batch só.
- Merge é intra-ledger: cartas com cartas, acessórios com acessórios.
- `/api/storefront` publica os acessórios sob game `acessorios` / label `Acessórios`.
- `/api/portfolio/all` ganhou o campo `accessories` (contado uma vez, fora dos jogos).
- Migração no boot (`migrateAccessories`): tira `kind == accessory` de cada ledger de
  jogo e move para o compartilhado preservando id, preço e estado de listagem.

## Vitrine
- `mapAccessory` usa o set do produto (ou "Acessório") no meta, não mais o nome do jogo.
- Detalhe do acessório: breadcrumb sem o jogo, textos/SEO sem "acessório de Acessórios".

## Verificação
- `go build/vet/test ./...` limpo; `tsc --noEmit` limpo nos dois fronts.
- Novo `internal/api/accessories_test.go` (5 casos): grava fora do ledger do jogo,
  aparece em todos os jogos, edita/exclui de outro jogo, storefront sob `acessorios`,
  listings nos dois ledgers.
- Servidor isolado em :8099 com um acessório legado plantado em `trades.json`:
  log "moved 1 accessory holdings…", id/preço/listed/featured preservados; criar pelo
  Pokémon não criou `trades-pkm.json`; vender/excluir por Lorcana/Gundam funcionou.
- Container rebuildado; smoke test no :8080 (criar pelo Pokémon → visível em onepiece/
  riftbound/gundam, `trades-pkm.json` intacto) e teste removido no fim.

## Pendente
- `data/accessories.json` é **prod-owned**: sync-down/seed-prod e RAILWAY.md já cobrem,
  mas o prod só passa a ter o arquivo depois do próximo deploy do binário.

---

# Redesign do backoffice/dashboard — manual da marca (2026-07-28)

Plano: `~/.claude/plans/snug-wobbling-orbit.md`.
Base: `web/manual-marca.pdf` (74p; a linguagem de layout está nas p.56–58) +
o handoff da vitrine. Supera o brief antigo em `tasks/dashboard-redesign-prompt.md`,
que é anterior ao manual.

Decisões do usuário: **Outfit** no lugar da Nohemi (que não está no Google Fonts) ·
tema **claro** como o manual · **paleta da vitrine mantida** (#F6559B / #FDC4E5 /
#1355B3 / #0b0b0c; a paleta nova do manual — indigo/laranja/roxo — fica de fora) ·
janelas retrô "forte com moderação".

## O que mudou

- **`web/src/theme.css` (novo)** — tokens claros (`page` #e8e8e8, `panel` #fff,
  `cream` #efe8d6, `ink` #1b1b1b), helpers `.win` / `.win-bar` / `.pill` /
  `.bg-grid`, sticker e keyframes. O dashboard **parou de importar
  `shared/arcade.css`**: a vitrine continua dark e Baloo, os dois apps agora
  compartilham o vocabulário de classes, não os tokens.
- **Rampa `slate-*` invertida** dentro do `@theme`. É a alavanca do redesign:
  vira ~400 utilitários herdados do build escuro de uma vez, sem tocar nas 10.000
  linhas de página. `slate-900` é creme (não branco) de propósito — quase todo uso
  é em alpha baixo para linhas expandidas, e branco-sobre-branco sumiria.
- **`--color-brand-deep` #a3305f** para texto. O pink da marca dá 2.9:1 no branco
  e reprova AA abaixo de 24px, então virou exclusivamente preenchimento; os 54
  micro-labels passaram para o pink escuro (6.4:1).
- **Sinais escurecidos** para fundo claro: ganho #0d7a4e · perda #b81d2b ·
  live #a15c00. A perda começou em #c2264a e foi trocada porque lia como acento
  rosa, não como aviso.
- **Primitivos**: `Card` virou `.win`; novo `Window` com barra de título (três
  bolinhas + ✕ desenhados em pseudo-elemento); botões e toggles viraram pílulas;
  cabeçalho de tabela virou faixa creme com `font-pixel`; tooltip de gráfico virou
  mini-janela; `chartColors.sky` (que era pink) virou `.brand`.
- **Shell**: sidebar creme com item ativo em pílula pink, topbar branca, `.bg-grid`
  no fundo, e os **6 `<h1>` duplicados** removidos (cada página repetia o título
  que o `PageHeader` do shell já renderiza).
- **`EmptyState` novo** com o mascote — substituiu 11 painéis vazios repetidos.
- **Export do ShareList**: a paleta do canvas foi para creme/branco/ink/pink e cada
  carta ganhou contorno ink. `C_ASK` era `#8c86ff`, o último indigo fora de marca.

Adoção (antes → depois): `sticker` 11 → 53 · `font-pixel` 23 → 33 · `pill` 0 → 12 ·
`win-bar` 0 → 5 · `shadow-xl`/`bg-black`/`text-faint` → 0.

## Verificação

- `npm run build` (tsc + vite) limpo.
- Captura de tela no Vite (:5174) de Deals, Browse, Tracking, Sealed, Portfolio,
  All Games, Estoque, Orçamento e Buyout, mais o modal de imagem do Estoque.
- `git diff shared storefront` vazio — a vitrine não foi tocada.
- **Não verificado visualmente:** o PNG exportado do ShareList (conferir exige
  disparar um download). É troca de constantes + um `strokeRect`; vale olhar um
  export antes de mandar para cliente.

## Pendente

- **Nohemi**: hoje é Outfit. Com os arquivos licenciados, é trocar
  `--font-display`/`--font-body` em `theme.css` e o import em `main.tsx`.
- **Rebuild do container** `:8080` (`docker compose up -d --build opdeals`) —
  não mexi nele; a revisão toda rodou no Vite.
- **Mascote**: `web/src/assets/mascot.png` veio do handoff e carrega fundo
  padronado (o README do handoff já avisava). Vale pedir o PNG transparente.

## Correção: de volta pro escuro (mesma sessão)

O tema claro foi testado e rejeitado — "o fundo escuro realmente é melhor". Como
o redesign foi feito na camada de tokens, a volta custou só isso:

- **Três renomes** antes de virar a chave, porque os nomes iam passar a mentir no
  escuro: `text-ink` → `text-fg` (37) · `text-brand-deep` → `text-brand-label` (54)
  · `bg-cream` → `bg-raised` (47). Agora as classes descrevem a **função**, não a
  claridade — é o que torna a paleta trocável sem tocar em componente.
- **`theme.css`** com valores escuros, `chart.tsx` e a paleta de canvas do
  ShareList junto.

**O ajuste que importa:** com a página em #141416 e o contorno em #0b0b0c (os
valores da vitrine), a borda grossa e a sombra dura **somem** — cada painel virava
um borrão escuro e a linguagem sticker se perdia. Então os degraus do dashboard
ficaram um passo mais claros que os da vitrine, e o contorno virou preto puro:
página #17171b · painel #24242a · raised #2f2f36 · outline #000. Aí os três níveis
(painel > borda/sombra < página) voltam a ler. **Não "alinhar" isso de volta com a
vitrine** — é proposital.

`--color-faint`/`slate-500` também subiu para #82828b, para AA no texto miúdo.

Tudo que era estrutura ficou: janelas com barra de título, pílulas, `EmptyState`
com mascote, títulos dedupados, faixa de cabeçalho nas tabelas, tooltip de gráfico
como mini-janela, Outfit.

Verificado de novo: build limpo + Deals, Portfolio, All Games, Estoque e o modal
de imagem. A vitrine continua intocada.

## Importar EUA (2026-09-23) — deals EUA→BR

Plano: ~/.claude/plans/steady-moseying-abelson.md

- [x] tracking: `LatestDayWithDepth`, `imports.go` (ImportSupplyFor, ladder sem Collecta) + testes
- [x] compare: `USDGroups`, `Dearest`, `ImportCandidates` + testes
- [x] pipeline: live-price dos candidatos inversos (ImportMinUSD/ImportMax, flag `-import-live`)
- [x] internal/imports: Build/Apply/Deal + testes
- [x] api: GET /api/imports com cache por jogo (chave = data tracking + updatedAt + days); main: `-liga-store-id` (866280)
- [x] web: api.ts, brand/App, ImportsPage (Negócio › Importar EUA)
- [x] verificação: go build/vet/test verdes, `tsc && vite build` limpo, curl (80 linhas nos filtros padrão, 2,7 s frio / 10 ms cache), tela conferida no Chrome

### Review

- Ace OP16-118 (topo da lista) conferido à mão: piso EN NM R$ 198,90 (PK Center, tracking 16/08) e
  US$ 16,51 no snapshot de 22/09. A conta bate; o que não bate é a data: tracking local está 5
  semanas atrás do snapshot de deals. Rodar captura antes de comprar por essa lista.
- Custo "tabela" em quase tudo de OP porque o snapshot atual ainda não passou pelo pipeline com
  `ImportMax`; só o próximo refresh precifica ao vivo os candidatos inversos.
- Riftbound: Normal e Foil do TCGCSV caem na mesma chave — custo sai pelo foil (conservador),
  badge "ambíguo ×N" na tela. Separar por `extras` do tracking fica para uma v2.

## Exportar BR (2026-09-23) — sentido BR→EUA na mesma página

- [x] tracking: `FloorStores` no ImportSupply + teste
- [x] internal/imports/export.go: BuildExport/ApplyExport (venda pela variante mais barata) + testes
- [x] api: cache guarda o supply (chave data|days); `?direction=export`, `minUSListings`
- [x] web: getExports, toggle EUA→BR / BR→EUA (`?dir=export`), tabela export com lojas no piso, lista de compra
- [x] verificação: go build/vet/test verdes, `tsc && vite build` limpo, curl (100 linhas), Chrome nos dois sentidos

### Review
- Topo do BR→EUA: Teemo GG EZ (ROPP 197B) R$ 5.399 → US$ 1.430 ao vivo (8 anúncios), lucro US$ 376.
  Exatamente o que a página Ofertas mostra, agora com a loja onde está a cópia (CHQ Cards ×1).
- Container `:8080` ainda NÃO tem o sentido BR→EUA: rebuild só depois da captura em andamento
  terminar (uma faixa só da Liga).

## Sem giro (2026-09-23, pedido do dono)
- Removida a inferência de vendas dos dois sentidos: sem janela, sem min giro, sem sugerido=1,5×giro, sem score por giro.
  EUA→BR ordena por lucro/un (ou margem); BR→EUA mantém score = lucro × cópias no piso. Handler lê só a captura
  mais recente com profundidade (sem LoadRange), cache por data.
- Deploy: container local rebuildado e `railway up` feito; prod serve o bundle novo.

## Reprecificação Liga 2026-09-23 (rft + pkm)
- [x] bin/floors pkm+rft em paralelo (cfgate já rodando)
- [x] análise piso−0,01 (scripts/liga-reprecificar/analyze.py)
- [x] gravar no admin: rft 135/135, pkm 20/20 (verificado no reload)
- [x] sync dashboard (listings + flag Liga): rft 135, pkm 20
- [ ] PENDENTE gravar 2ª passada (custo unitário corrigido): rft 16 + pkm 3 — apply2-*.json no scratchpad; classificador barrou o browser_batch
- [ ] dono decidir: 42 rft + 9 pkm abaixo do custo unitário; 14 rft + 17 pkm com folga para subir
- [ ] 12 itens ligaListed sem oferta COLLECTA na Liga → liga-vendas/liga-conferir

## 2026-10-03 — Estoque da Liga → dashboard (singles + selados)

- [x] Exportar Liga (fetch in-page, clipboard): OP 182 linhas, PKM 603, RFT 354, Produtos 23; Lorcana/Gundam sem estoque
- [x] Rodar conferir.py por jogo e classificar as linhas da Liga sem registro no dashboard
- [x] Plano em scratchpad/plan.json: 121 criações (OP 4+1 selado, RFT 18, PKM 98), 92 flags, 10 askBRL, 55 pulos (gêmeos JP, R$0, GEM Pack)
- [x] POST /api/trades (buyBRL 0, nota "Cadastrado a partir do estoque da LigaMagic 2026-10-03") e guardar ids
- [x] PUT askBRL nos casados sem preço
- [x] POST /api/trades/liga com as flags (inclui os criados)
- [x] Verificar no GET e relatar

Revisão: 121 registros criados (ids em tasks/liga-sync-2026-10-03/created.json), 213 flags gravadas, 10 askBRL preenchidos.
Pulados (tasks/liga-sync-2026-10-03/skips.txt): 35 gêmeos JP no Pokémon (mesma carta cadastrada 2× na Liga), 18 linhas RFT q6 R$0 (Overnumbered/Signature VEN), GEM Pack Vol.4/Vol.6 (dashboard tem Vol.2/Vol.3).

## 2026-10-07 — Foto por item nas Compras

- [x] Backend: `POST/DELETE /api/trades/{id}/photo?game=` (multipart `photo`, 10 MB, jpeg/png/webp), `GET /api/item-photos/{name}`, flag `-item-photos` (default `data/item-photos`, no volume); upload grava `imageURL` relativo e apaga a foto anterior
- [x] `/api/storefront` absolutiza `imageURL` relativo (vitrine na Vercel); `ShareList` usa foto própria sem proxy
- [x] Frontend: botão de câmera na linha do item (Compras/Itens/Avulsos) e Enviar/Trocar/Remover foto no editar; `web/src/photo.ts` encolhe no browser (1600 px, jpeg 0.85)
- [x] Testado: curl (upload/troca/415/404/delete/traversal) + Vite contra server de teste (thumb troca, remover limpa arquivo)
- [x] Container local rebuildado
- [ ] Publicar em prod (pendente, dono roda)

Revisão: foto fica em `data/item-photos/<tradeID>-<rand>.jpg`; `imageURL` do trade = `/api/item-photos/...`. Limpar a URL pelo StockPage não apaga o arquivo.

## 2026-10-09 — collectatcg.com.br servindo catálogo MOCK

Causa: a recriação do ledger em 2026-10-07 (plan.py/post.py do liga-sync) gravou `askBRL` mas nunca `listed`.
`/api/storefront` em prod responde `count:0` e a vitrine caía no mock (Umbreon VMAX, FOTO DA CARTA, preços falsos).

- [x] `storefront/lib/catalog.ts`: mock só sem `STOREFRONT_API`; API vazia = catálogo vazio (`live:true`); erro/HTTP≠200 lança (Next mantém a última página boa)
- [x] `app/page.tsx`: "Singles em destaque" some quando vazio (igual selados)
- [x] tsc limpo; testado com backend local real (462 itens) e com backend fake vazio (hero + "Nenhuma carta encontrada", sem mock)
- [ ] DADOS (dono roda, classificador bloqueou escrita em prod): `python3 scripts/storefront-list-all.py` — marca `listed:true` em todo holding com askBRL>0 (OP 192, PKM ~262, RFT ~245 + acessórios), preservando imageURL/featured
- [ ] Deploy da vitrine (Vercel) com o catalog.ts novo

### Imagens (2026-10-09, continuação)
- [x] Script de listagem rodado pelo dono: `/api/storefront` count 664, site mostra estoque real
- [x] cfgate estava preso num IPv6 que o host não tem mais (prefixo rotacionou; "bind: can't assign requested address") → reiniciado, pinou 2804:3d90:82b1:...
- [x] 221 singles sem arte resolvidas localmente (105 via tracking, 116 via `url=` explícita). 35 prints erradas (Liga ignora `num` sem o sufixo "(#num/total)") corrigidas: 21 pelo padrão `cd/<setID>/<cardID>_<num>.jpg` do export, 13 refazendo com `Nome (#num/total)`, Caterpie sv2a + 2 GX (`.57.jpg`) à mão. Auditoria: sufixo da URL == número em 100% das pkm
- [x] OP: `DON/DON-001` e `OP13/OP13-057-3A` semeadas no cache (atalho por número servia o DON do EB03)
- [x] 23 produtos sem foto: 20 pela busca por nome na Liga (`cards/search&card=<nome> searchprod=1&category=products` cai na página do produto, imagem `up/prod/`), 3 promos JP pelo og:image do Mercado Livre → `tasks/storefront-images-2026-10-09/product-images.json`
- [x] `scripts/sync-up.sh -i` (só images.json + reload) e `scripts/storefront-product-images.py`
- [ ] DONO: `scripts/sync-up.sh -i` · `python3 scripts/storefront-product-images.py` · deploy Vercel
- [x] 2026-10-10 auditoria 216/216 pkm contra o tile da Liga: 36 corrigidas (Eevee TWM 188 era a comum 135). Agora cache == imagem que a Liga mostra para a print exata
- [x] Eevee ainda errada no site após o fix: backend prod já servia a certa; `/img` fica 24 h em 3 caches (data cache do Next sobrevive a deploy, CDN Vercel, browser). `catalog.ts` agora carimba `v=<VERCEL_DEPLOYMENT_ID>` nas URLs de arte → cada deploy renova. Precisa do deploy da vitrine

## 2026-10-10 — Arte da vitrine/dashboard vinha do TCGplayer, não da Liga

Chopper `OP01-015-RP` (Offline Regional 2024 Vol. 3, R$ 1.150) aparecia com a arte do Chopper comum (produto TCGplayer 454533). Causa: `/api/card-image` tentava o `productID` do TCGplayer ANTES da arte da Liga, e o matcher junta todas as impressões do mesmo número (18 singles OP com arte de outra impressão). Regra do dono: a Liga é a fonte de verdade da vitrine.

- [x] `internal/api/api.go` `handleCardImage`: arte da Liga (cache/página) primeiro; TCGplayer só quando a Liga não tem nada; 404 só se nenhum dos dois
- [x] `web/src/components/CardArt.tsx`: ordem das fontes = foto própria → card-image (Liga, com productID só como fallback no servidor) → CDN TCGplayer direto; `imgVersion` 2→3 (cache de 24 h no browser)
- [x] Verificado local (:8099): OP01-015-RP com productID=454533 serve a arte Regionals da Liga; VEN sem arte na Liga cai no TCGplayer; card inexistente 404; go test api/cardimg ok; tsc limpo
- [x] Cobertura em prod sem productID: OP 137/137, PKM, LOR, GND 100%; RFT 200 sem arte da Liga (156 VEN — set não está em `DefaultTrackSets`; 44 OGN/SFD/UNL/ROPP com página mas sem cache em prod → `scripts/sync-up.sh -i` após warm local)
- [ ] DONO: `railway up` (backend + web/dist) e redeploy da vitrine na Vercel (o `/img` fica 24 h em cache; `ART_VERSION` renova por deploy)
- [ ] Opcional: refUSD do dashboard ainda casa promo com a base (PrintKey existe em compare.go, priceLookup não usa) — só valuation, não afeta mais a arte
- [x] 2026-10-10: Railway deploya do GitHub (main). Backend acumulado commitado (f3ee96b): fotos, importar EUA, cfgate, Liga antes da TCGplayer, artVersion
- [x] Riftbound 200/200 singles resolvidas pelo tile da Liga (num exato, ex. 112A, R06A); OP cobre tudo pelo cache+byNum; PKM 216/216 auditadas
- [ ] DONO: `bash scripts/sync-up.sh -i` (sobe images.json pkm/op/rft + reload) e `python3 scripts/storefront-product-images.py` se ainda não rodou

---

# Gateway de pagamento na vitrine (Mercado Pago Checkout Pro + Melhor Envio) — 2026-10-12

Plano completo: ~/.claude/plans/luminous-hopping-lemur.md

## Plan
- [x] 1. Identidade + ledger (Go): trades.SKU, internal/orders (store + allocate), sku e reserva no /api/storefront
- [x] 2. Clientes externos (Go): internal/mercadopago, internal/shipping (Melhor Envio + embalagem), flags/env no main
- [x] 3. Endpoints (Go): internal/api/orders.go (quote, create, get, webhook, list, patch) + testes com fakes
- [x] 4. Vitrine: sku no catálogo/carrinho, rotas server, /checkout, /pedido/[id], CartView CTA, analytics purchase, robots
- [x] 5. Dashboard: aba Pedidos
- [ ] 6. Deploy/config (dono): envs Railway + Vercel, webhook MP, token ME, E2E sandbox

## Review

**Verificado:** `go vet ./...` + `go test ./...` verdes (21 testes novos: SKU, ledger de pedidos, alocação FIFO,
embalagem/parse do Melhor Envio, create/webhook/idempotência/concorrência/oversold/auth/patch); `tsc` + `next build`
da vitrine e `tsc` + `npm run build` do dashboard limpos. E2E local no Chrome com `CHECKOUT_FAKE=1`: carrinho →
/checkout (ViaCEP preencheu, 3 fretes cotados, total atualizado) → pedido criado → /pedido/{id} PAGO via `?payment_id`
→ carrinho zerado → `trades.json` com split vendido (`Ana Souza Teste — Site #1`, sellPrice 57) → vitrine com qty
descontada → aba Pedidos listou, "Marcar enviado" gravou rastreio e a visão pública mostrou o código. Mobile 390px
(iframe) ok no checkout e no carrinho.

**Decisões fora do plano:** `sku` sem porte do slugify (chave normalizada própria; catálogo agrupa por sku, com
fallback `legacy|slug` se o backend ainda não mandar); `CHECKOUT_FAKE=1` (`internal/checkoutfake`) para testar sem
credenciais; `GET/PATCH /api/orders` liberados quando ADMIN_TOKEN está vazio (dev), exigidos quando existe;
`MELHORENVIO_CONTACT` para o User-Agent que o Melhor Envio exige; cupom mock ARCADE10 removido do carrinho.

**Pendente do dono:** credenciais MP + webhook (`${PUBLIC_API_URL}/api/orders/mp-webhook`, segredo em
MP_WEBHOOK_SECRET), token Melhor Envio, envs na Railway (ver memória opdeals-checkout-mercadopago) e
`STOREFRONT_TOKEN` na Vercel; deploy do Go antes da vitrine; teste com cartões de sandbox; nada commitado.

---

# Importar Japão — Card Rush × Liga JP/PT (2026-10-17)

## Plano
- [x] `internal/jpimport`: tabela de coleções JP→ocidental (Mega + Escarlate/Violeta), leitura do `data/cardrush-pkm.json`, cálculo (piso JP/PT/EN, par ocidental por nome no bloco IR/SIR, giro 14 dias, preço real = min(JP, PT)) + testes
- [x] `tracking.Capturer.SetCardFilter` + capturador `pokemon jp import capture` (só cartas secretas, `data/jpimport-pkm`), flag `-pkm-jp-capture-once`, entra no scheduler
- [x] `fx.RateJPY` (frankfurter JPY→BRL) com cache de 6 h na API
- [x] `GET /api/jp-import` com cache por mtime da captura + data do tracking + câmbio
- [x] Página "Importar Japão" (grupo Negócio): KPIs, filtros (era, coleção, AR/SAR, com/sem PT, × mínimo, busca, só com estoque), tabela ordenável, links CR/JP/PT
- [x] `scripts/cardrush-capture.js` + `scripts/cardrush-import.py` + skill `cardrush-capturar`
- [x] Primeira captura da Card Rush (766 AR/SAR) e da Liga das SV

## Review
- Números conferidos contra a análise manual: Gladion 2,8×, Wally 4,6×, Golisopod 8,5×, Lucario M1L→MEG 179, Pikachu 30C gêmeos 149/150.
- O giro do tracking (`TopSoldCards`) ignora a loja que vende a última cópia; com 1–2 cópias por anúncio isso zerava tudo. `jpimport.SoldByLang` conta esse caso e pula capturas ocas.
- Card Rush: as páginas por raridade não listam tudo; a busca `AR <coleção>` em `/product-list` é a fonte completa.
- Prod (Railway) não recebe `data/cardrush-pkm.json` nem `data/jpimport-pkm` ainda — sincronizar se quiser a página lá.

---

# Baixa automática de vendas (Liga + MyP + Site → dashboard e canais) — 2026-10-23

## Contexto levantado
- `liga-vendas` já faz Liga → dashboard (dedup `Liga #N`) e, desde 20/10, tira o item vendido do MyP.
- MyP: `/CollectaTCG/vendas` lista pedidos (Aguard. envio / Enviado / Outros Status); detalhe em
  `/pedido/<id>` com cada item trazendo nome, edição, ga-id (`one_op15_eb04-007`), condição, qty, unitário.
  Hoje a baixa do MyP é manual (1 único sold com buyer "MYP Cards"); pedido #660256 (23/10, 8 itens) ainda sem baixa.
- Vendas do Site (`Site #N`) já entram no dashboard pelo webhook; falta só tirar da Liga e do MyP.
- Mac precisa estar acordado, com o Chrome aberto e a extensão conectada; cloud routine não serve (sessões logadas, cf_clearance ligado ao IPv6).

## Plano
- [x] Skill `myp-vendas`: ler `/CollectaTCG/vendas` + `/pedido/<id>`, contar só pedido pago (Aguard. envio / Enviado / Concluído),
      casar item pelo ga-id + nome + preço com o ledger, `POST /sell` com buyer `"<nick> — MyP #<pedido>"`, dedup por `MyP #N`
      e checagem do sold manual "MYP Cards"; depois reduzir/remover o anúncio na Liga.
- [x] Skill `baixa-vendas` (orquestradora): roda Liga → MyP → Site em sequência, cada venda nova vira baixa no dashboard
      + propagação para os outros canais; modo `--auto` sem perguntas: ambiguidade/sem lastro só vai pro relatório.
- [x] Relatório de cada execução em `data/baixa-vendas/<data-hora>.md` + push no ntfy com resumo e pendências.
- [x] Lock (`data/baixa-vendas/.lock`) para não sobrepor execuções.
- [x] `scripts/baixa-vendas.sh` chamando `claude -p "/baixa-vendas --auto" --chrome --allowedTools ...` (sem bypass total).
- [ ] (dono instala — bloqueado pelo classificador) `~/Library/LaunchAgents/com.collecta.baixa-vendas.plist` com `StartCalendarInterval` nos horários escolhidos.
- [ ] Verificação: rodar à mão em modo interativo com o pedido MyP #660256, conferir prod + MyP + Liga; depois uma
      execução headless pelo script e conferir o relatório e o push.

## Review
- MyP #660256 baixado em prod (6 registros `zapiti — MyP #660256`, total R$ 353,64); Liga já estava zerada à mão
  para os 5 itens esgotados e S-Snake em 3; `ligaQty` do lote restante de S-Snake 4 → 2. Nenhuma venda dupla nos
  pedidos da Liga de 22–23/10.
- Gravação de quantidade no admin da Liga (Fase 5 da myp-vendas) documentada mas ainda não exercitada — o caso real
  não precisou.
- Pendente na Liga: 5 pedidos pagos de 22–23/10 (#12297573, #12299752, #12301915, #12305730, #12305781) — primeira
  rodada da baixa-vendas pega.
- Falta: instalar o plist e rodar `scripts/baixa-vendas.sh` uma vez com o dono olhando (headless `-p --chrome`).

# SEO collectatcg.com.br (2026-10-24)

## Contexto levantado
- Search Console 90d: 386 cliques, 10,8 mil impressões, CTR 3,6%, posição 7,1. 437 indexadas, 270 não (41 404, 137 rastreada-não-indexada, 88 detectada).
- `/carta/pokemon-mega-darkrai-ex-101-084-nm` tinha ~60% das impressões (pos 3,4) e hoje é 404: esgotou e o slug mudou no recadastro de 07/10 (agora `...-101-nm`).
- Listagens do comerciante: 8 válidas, todas sem `shippingDetails`. Merchant Center 5831736937 com feed de 614 itens + 4 placeholders do GBP sem preço; 2 blisters reprovados por overlay promocional.
- `GET /api/trades` em prod sem autenticação expõe custo.
- Frete grátis já existe via env `FREE_SHIPPING_MIN_BRL`. Tracking em prod parado em 23/09.

## Plano
### Fase 0 — segurança
- [x] Token nos endpoints privados/escrita do Go; dashboard e scripts das skills mandam o token.
### Fase 1 — recuperar ranking
- [x] Página de esgotado (200, OutOfStock, outras versões, "Avise-me no WhatsApp" + evento), gerada das vendas do ledger; fora do feed.
- [x] 301 de slugs antigos (404 do Search Console + backup prewipe) e slug congelado no primeiro cadastro.
- [x] Título `{Nome} {nº} – Preço R$ X | Collecta`, descrição com preço/estoque/envio, H1 nome+número+variante.
### Fase 2 — dados estruturados e Merchant Center
- [x] `shippingDetails` (R$ 25, grátis acima de R$ 300, 1–2 + 2–10 dias), devolução na organização, `sku` único, marca = fabricante, Breadcrumb/ItemList nas categorias.
- [ ] `FREE_SHIPPING_MIN_BRL=300` no Railway + faixa no AnnouncementBar.
- [ ] Merchant Center: apagar 4 placeholders, políticas de frete/devolução; imagem limpa dos 2 blisters.
- [x] Bing Webmaster: importar do Search Console + sitemap.
### Fase 3 — categorias e conteúdo
- [x] `/jogo/[g]` 301 → `/singles/[g]` com intro, coleções e FAQ; H1 de coleção com nome.
- [x] Bloco "Preço de mercado no Brasil" (sem fonte, some com dado > 14 dias), só cartas do catálogo.
- [ ] `sync-up.sh` diário dos snapshots no launchd.
### Fase 4 — performance e confiança
- [x] `/img` WebP redimensionado, catálogo fatiado no cliente, sitemap com lastModified + imagens.
- [x] Privacidade, Termos, Frete, Sobre (razão social + CNPJ); remover HighScores mock, specs inventadas de selado, texto "finalize pelo WhatsApp".
### Dono
- Credenciamento de loja oficial de torneio; link do site em todos os perfis.

## Review (2026-10-24)
- Fase 0: gate de `X-Admin-Token` em toda `/api` privada (`internal/api/api.go` `gate`), públicas: storefront(+market), card-image, image, item-photos, orders, shipping, admin. Dashboard instala wrapper de fetch e pede o token num card quando recebe 401. Scripts Python, `cmd/floors` e 8 skills mandam o header (token de `scripts/deploy.env`). Teste local: sem token 401, com token 200, vitrine 200.
- Fase 1: `Trade.Slug` congelado (Add grava, Update preserva), `/api/storefront` devolve `slug` + `soldOut` (vendidos listados e itens zerados por reserva). Vitrine: página de esgotado 200 + OutOfStock + "Avise-me no WhatsApp", 31 redirects 308 de slugs pré-wipe (`lib/legacy-slugs.json`, inclui o Darkrai) e 464 páginas de esgotado do backup pré-wipe (`lib/legacy-sold.json`; arte só quando não é variante, `noArt`). Gerador em `tasks/seo-2026-10-24/legacy.py`. Slug Go = slug JS nos 623 itens de prod.
- Fase 2: JSON-LD com `shippingDetails` (R$ 25 / grátis ≥ R$ 300 / 1–2 + 2–10 dias), devolução também na Organization, `sku` = id do feed, marca = fabricante, legalName/taxID. Bing: sitemap enviado (site já existia). Merchant Center já tinha frete por peso e devolução 7 dias em 618 produtos; 611 aprovados, 140 cliques/28d.
- Fase 3: `/jogo/:g` 308 → `/singles/:g` (next.config), nav sem "VER TUDO", intro + coleções + FAQ + Breadcrumb/ItemList em `/singles/[g]` e coleções, H1 de coleção com nome, só o jogo atual vai para o cliente. Endpoint público `/api/storefront/market` (piso NM por idioma sem a Collecta, mediana 30d, some > 14 dias, recusa número ambíguo) + bloco na carta. `sync-up.sh --recent N` + `scripts/com.collecta.sync-up.plist` (8h30).
- Fase 4: `/img` em WebP ≤ 600px (sharp fixado), sitemap com lastModified/imagens/esgotados e sem /jogo, páginas /frete /privacidade /termos /sobre no rodapé, HighScores e specs inventadas removidos, título padrão curto, og sem herdar a home.
- Pendente do dono: setar `FREE_SHIPPING_MIN_BRL=300` no Railway ANTES do deploy da vitrine; commit/push (backend sobe junto com mudanças antigas da árvore); instalar o plist; rodar captura local (tracking parado em 19/10) e um `sync-up.sh --recent 31`; depois do env, regra de frete grátis no Merchant Center. Não apagados: 4 "produtos" do Perfil da Empresa no MC (são a vitrine do GBP).

---

# Dashboard: estoque, compras e vendas (2026-10-30)

Decidido no grilling de 30/10. Usuários: dono + sócio, desktop, uso pesado em lançar compra e dar baixa. Fonte da verdade: Liga manda em qty e preço de venda; dash manda em custo, compra e lucro. Cada fase: plano aprovado antes de codar, testes, "pronto pra commit?", commit separado. Backup do volume de prod antes de toda migração.

### Fase 1 — proteções
- [x] `PUT /api/trades/{id}` vira patch: só altera campos enviados (hoje `*t = in` apaga Liga/vitrine/venda num corpo parcial).
- [x] Confirmação ao excluir item e compra em Compras (hoje sem confirm: `PurchasesPage.tsx` delete do item e da compra).


- Nota: vendas antigas não foram reescritas; `trades.ChannelOf`/`WriteOffOf` resolvem na leitura. Baixa sem venda legada = buyer com "aberto/aberta" (a regra antiga de R$ 1/un. pegava venda real de bulk a R$ 0,27). `semLastro` só nas novas: em prod 448 vendas têm buyBRL 0 por causa do wipe de 07/10, então o critério não é seguro para o legado.

### Fase 3 — categorias e conteúdo
- [x] `/jogo/[g]` 301 → `/singles/[g]` com intro, coleções e FAQ; H1 de coleção com nome.
- [x] Bloco "Preço de mercado no Brasil" (sem fonte, some com dado > 14 dias), só cartas do catálogo.
- [ ] `sync-up.sh` diário dos snapshots no launchd.
### Fase 4 — performance e confiança
- [x] `/img` WebP redimensionado, catálogo fatiado no cliente, sitemap com lastModified + imagens.
- [x] Privacidade, Termos, Frete, Sobre (razão social + CNPJ); remover HighScores mock, specs inventadas de selado, texto "finalize pelo WhatsApp".
### Dono
- Credenciamento de loja oficial de torneio; link do site em todos os perfis.

## Review (2026-10-24)
- Fase 0: gate de `X-Admin-Token` em toda `/api` privada (`internal/api/api.go` `gate`), públicas: storefront(+market), card-image, image, item-photos, orders, shipping, admin. Dashboard instala wrapper de fetch e pede o token num card quando recebe 401. Scripts Python, `cmd/floors` e 8 skills mandam o header (token de `scripts/deploy.env`). Teste local: sem token 401, com token 200, vitrine 200.
- Fase 1: `Trade.Slug` congelado (Add grava, Update preserva), `/api/storefront` devolve `slug` + `soldOut` (vendidos listados e itens zerados por reserva). Vitrine: página de esgotado 200 + OutOfStock + "Avise-me no WhatsApp", 31 redirects 308 de slugs pré-wipe (`lib/legacy-slugs.json`, inclui o Darkrai) e 464 páginas de esgotado do backup pré-wipe (`lib/legacy-sold.json`; arte só quando não é variante, `noArt`). Gerador em `tasks/seo-2026-10-24/legacy.py`. Slug Go = slug JS nos 623 itens de prod.
- Fase 2: JSON-LD com `shippingDetails` (R$ 25 / grátis ≥ R$ 300 / 1–2 + 2–10 dias), devolução também na Organization, `sku` = id do feed, marca = fabricante, legalName/taxID. Bing: sitemap enviado (site já existia). Merchant Center já tinha frete por peso e devolução 7 dias em 618 produtos; 611 aprovados, 140 cliques/28d.
- Fase 3: `/jogo/:g` 308 → `/singles/:g` (next.config), nav sem "VER TUDO", intro + coleções + FAQ + Breadcrumb/ItemList em `/singles/[g]` e coleções, H1 de coleção com nome, só o jogo atual vai para o cliente. Endpoint público `/api/storefront/market` (piso NM por idioma sem a Collecta, mediana 30d, some > 14 dias, recusa número ambíguo) + bloco na carta. `sync-up.sh --recent N` + `scripts/com.collecta.sync-up.plist` (8h30).
- Fase 4: `/img` em WebP ≤ 600px (sharp fixado), sitemap com lastModified/imagens/esgotados e sem /jogo, páginas /frete /privacidade /termos /sobre no rodapé, HighScores e specs inventadas removidos, título padrão curto, og sem herdar a home.
- Pendente do dono: setar `FREE_SHIPPING_MIN_BRL=300` no Railway ANTES do deploy da vitrine; commit/push (backend sobe junto com mudanças antigas da árvore); instalar o plist; rodar captura local (tracking parado em 19/10) e um `sync-up.sh --recent 31`; depois do env, regra de frete grátis no Merchant Center. Não apagados: 4 "produtos" do Perfil da Empresa no MC (são a vitrine do GBP).

---

# Dashboard: estoque, compras e vendas (2026-10-30)

Decidido no grilling de 30/10. Usuários: dono + sócio, desktop, uso pesado em lançar compra e dar baixa. Fonte da verdade: Liga manda em qty e preço de venda; dash manda em custo, compra e lucro. Cada fase: plano aprovado antes de codar, testes, "pronto pra commit?", commit separado. Backup do volume de prod antes de toda migração.

### Fase 1 — proteções
- [x] `PUT /api/trades/{id}` vira patch: só altera campos enviados (hoje `*t = in` apaga Liga/vitrine/venda num corpo parcial).
- [x] Confirmação ao excluir item e compra em Compras (hoje sem confirm: `PurchasesPage.tsx` delete do item e da compra).

### Fase 2 — canal da venda
- [x] Campo `channel` na venda: `liga` | `myp` | `site` | `balcao` | `outros`; select na baixa; `buyer` fica só com o nome.
- [x] `balcao` = toda venda direta (loja física, WhatsApp, pagou por fora); `myp` só pedido do MyPCards.
- [x] Migração das vendas antigas pelo texto do buyer (`Liga #`, `MyP #`, `Site #`, `balcão`, `MYP` pago por fora → balcao); vazio → outros. (feito na leitura, sem reescrever prod)
- [x] "Baixa sem venda" com motivo (aberto, perda, brinde, uso próprio): custa, não conta como receita/venda no Financeiro; migrar as baixas simbólicas de R$ 1/un. (desvio: legado só vira "aberto" com "aberto/aberta" no comprador; a regra de R$ 1 pegava ~90 vendas reais de bulk)
- [x] Marca `semLastro` nas vendas criadas sem item pela liga-vendas + filtro "Vendas sem lastro" em Vendas para vincular a uma compra. (desvio: só vendas novas; 448 de 499 vendas de prod têm custo 0 pelo wipe de 07/10)
- [x] `ligasales.ts`/`finance.ts` usam `channel` em vez de regex no buyer; pacotes por nº de pedido de Liga, MyP e Site.
- [x] Skills liga-vendas, myp-vendas, baixa-vendas, dashboard-cadastrar mandam `channel` (dashboard-cadastrar não dá baixa, ficou igual); liga-precificar deixa de supor frete dentro do `buyBRL`.

### Fase 3 — fila de propagação
- [x] Venda com canal ≠ liga em item `ligaListed` gera pendência "tirar X un. da Liga" (`internal/syncqueue`, `data/syncqueue.json`; cobre `/sell` e o pagamento do site; baixa sem venda também gera). O Trade não tem flag de MyP, então a tarefa é só da Liga e a skill acerta o MyP pelas mesmas unidades. `/sell` em lote já vendido agora dá 409.
- [x] Desfazer venda de item que estava na Liga gera pendência inversa "devolver X un." (Fase 4: cancela a remoção ainda aberta em vez de gerar o par tirar/devolver).
- [x] Tela Sincronizar no grupo Negócio (pendentes/erro/feitas, contador no menu, marcar feita, voltar para pendente) + `GET /api/syncqueue`, `POST /api/syncqueue/{id}/done|error|retry`; skill baixa-vendas consome a fila na etapa 4. Servidor nunca mexe na Liga.
- [ ] Worker automático que executa a fila sem o Claude: pendente de decisão do dono (Railway com cookie da Liga vs mini PC na loja).
- [x] Vincular venda sem lastro a uma compra (attach) limpa `semLastro`.

### Fase 4 — lote vendido
- [x] Lápis em lote vendido: nome, tipo, custo, foto (não a venda nem a qtd). Jogo entra na Fase 5.
- [x] "Desfazer venda" (`POST /api/trades/{id}/unsell`): volta ao estoque e junta de volta ao lote de origem (`splitFrom` nas vendas novas; antigas casam por compra + SKU). Bloqueado para canal `site` (caminho é Pedidos → reembolsado/cancelado).
- [x] Editor da aba Vendas perde `qty` (fica preço, data, comprador, canal) e ganha Desfazer venda.

### Fase 5 — trocar jogo
- [x] Campo Jogo no form de edição: move o item de ledger preservando id, slug, custo, compra, venda e fotos (`POST /api/trades/{id}/move`).
- [x] Bloqueado com anúncio ativo ou pedido do site aberto, com aviso. Slug base mantido, produto aparece no jogo novo na vitrine (a URL pública ganha o prefixo do jogo novo). Também bloqueia acessório e item com pendência aberta na fila de sincronização.

### Fase 6 — rateio do frete do pedido
- [x] `Purchase.ShippingBRL` rateado nos itens proporcional ao valor pago, gravado em `orderShippingBRL` (separado do frete digitado no item); recalcula a cada mudança de frete/itens; lote vendido não é recalculado.

### Fase 7 — estorno do pedido do site
- [x] Reembolsado/cancelado depois de pago devolve as unidades ao estoque (lote novo ou junta ao de origem).
- [x] Mudança manual de "sem estoque" para pago/enviado dá a baixa que hoje não acontece.

### Fase 8 — juntar telas
- [x] Portfólio some (`PortfolioPage.tsx` e `AllPortfolioPage.tsx` apagados); `?tab=portfolio` e `?tab=allportfolio` abrem Compras, `?tab=acessorios` abre Compras com `?jogo=acessorios`; `Kpi`/`Chip` em `ui/kpi.tsx` e `ui/chip.tsx`.
- [x] Compras vira a casa do item: "Novo item avulso" nas abas Itens/Avulsos (compra opcional "Sem compra (avulso)" no form de item, com loja e data), filtro de acessórios via jogo, resumo por jogo no topo (clique filtra o jogo), Exportar CSV na aba Itens.
- [x] Form de edição único: ganha valor atual R$ (selado/acessório), referência de mercado (carta) e GTIN (selado/acessório).
- [x] Estoque fica só com preço, vitrine e Liga; a "Lista em inglês" (variante portfolio do ShareList, para comprador dos EUA) mudou do Portfólio para o Estoque, ao lado de Compartilhar.

## Review (2026-10-30)
- Fase 1 `0a16340`: PUT do item é patch (`patchTrade`); vazio/zero em kind, refUSD, manualBRL, purchaseID, status, photos mantém o valor. Confirmação ao excluir item e compra.
- Fase 2 `6b030e6`: `channel`, `writeOff`, `semLastro` no Trade; canal das vendas antigas deduzido do comprador na leitura (`trades.ChannelOf`), sem reescrever prod. Front e skills usam o canal.
- Fase 3 `0a4ad41`: fila `internal/syncqueue` + tela Sincronizar + etapa 4 da baixa-vendas. Revenda de lote vendido dá 409.
- Fase 4 `3dc0bbe`: `POST /api/trades/{id}/unsell`, `splitFrom` nas vendas parciais, lápis em lote vendido, editor de Vendas sem qtd.
- Fase 5 `41347d7`: `POST /api/trades/{id}/move`; a URL pública da vitrine muda de prefixo ao trocar o jogo; mover só o lote vendido e depois desfazer a venda deixa o lote no jogo novo, separado da origem.
- Fase 6 `182734b`: `orderShippingBRL` rateado pelos ledgers de jogo + acessórios; `POST /api/purchases/reallocate` para aplicar nos dados antigos.
- Fase 7 `9c15b09`: reembolso/cancelamento (PATCH e webhook refunded/charged_back) devolve o estoque via `UnsellSiteSale`, idempotente por `ReturnedTradeIDs`/`StockReturnedAt`; oversold → pago dá a baixa ou 409 sem estoque.
- Fase 8: Portfólio apagado, Compras como casa do item, form único, Estoque só preço/vitrine/Liga.
- Desvios do plano: regras de legado da Fase 2 (acima); rateio grava campo próprio em vez de `shippingBRL`; worker da Liga fora do escopo.
- Pendências: (1) worker que executa a fila na Liga sem o Claude — decidir Railway (teste do admin via `railway ssh` deu 403 com UA do wget; falta repetir com UA de Chrome) ou mini PC na loja; (2) depois do deploy, backup do volume e `POST /api/purchases/reallocate` em prod (o frete inteiro de cada compra vai para os itens ainda em estoque); (3) custos dos 591 lotes sem custo em `custos-pendentes-2026-10-30.csv`; (4) a flag nova `-syncqueue` tem padrão `data/syncqueue.json`, dentro do volume de prod; (5) nada foi testado clicando na tela.
