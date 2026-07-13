# Pokémon (ligapokemon.com.br) — multi-game build

Goal: add Pokémon as a 2nd game alongside One Piece. Build every BR feature EXCEPT deals and the
TCGplayer/US integration: Tracking/Market-Pulse, Buyout/Snipe, Inventory, Sealed, Browse,
Export/Selection, Portfolio (BRL). One Piece stays the default game; its behavior is unchanged.

Platform verified identical (memory `opdeals-pokemon`). Game-specific bits only: Liga host,
image path (`pokemon_bkp`), all-language floor (not English-only), main-char tokens, set list.

## Backend
- [x] `internal/game/game.go` — Game config struct + OnePiece(), Pokemon(), ByID().
- [x] `internal/liga` — Client fields from Game (host/source/sealed cats/setCodeRe); add Floor(rows, langs).
- [x] `internal/cardimg` — per-game image regex (field on Store).
- [x] `internal/tracking/capture.go` — Capturer holds floorLangs; use liga.Floor.
- [x] `internal/tracking/buyout.go` — Buyout/Snipe take langs + mainTokens (nil langs = all).
- [x] `internal/api` — GameStack map, ?game= selector; buyout US-enrich only onepiece; both Liga hosts; portfolio per-game (BRL for pkm); /api/games route.
- [x] `cmd/server/main.go` — buildStack helper builds both stacks; -pkm-* flags; scheduler both; api.New(map, default).

## Frontend
- [x] `web/src/api.ts` — getGame/setGame/getGames + gp() appends game= to every tracking/buyout/inventory/cards/image/export/trades/quote/capture call.
- [x] `web/src/App.tsx` — GameSwitcher in header, game-aware branding + footer, hide Deals tab + FX/Refresh for Pokémon, content remounts on game change.
- [x] BuyoutGrid/BuyoutPage hide US strip when fxRate==0 (Pokémon); BrowsePage default set no longer hardcoded OP-16; PortfolioPage game-aware (BRL labels/format for Pokémon).

## Verify
- [x] go build ./... clean; go test ./... all pass; go vet clean.
- [x] Live Pokémon capture (DRI): 244 cards, **sprite decode 99.9% qty AND price** (1956/1957) — Liga font identical, templates reused verbatim. Language mix 1392 PT / 530 EN confirms all-language floor.
- [x] npm run build clean; server smoke test: /api/games lists both; pokemon inventory (271 stores, R$224k), buyout (40 plays, fxRate=0 → US strip hidden), card-image resolves pokemon_bkp art (49KB JPEG). OnePiece unaffected (117 stores, buyout fxRate 0.192).

## Review
Multi-game architecture is additive: One Piece is the default game so all existing routes/links/behavior are byte-for-byte unchanged (verified: OP sets, inventory, buyout US strip all intact). Pokémon is a pure BR tracking suite (no deals, no US/TCGplayer). The whole Liga scraping + sprite decode stack was reused unchanged — the two sites share the same platform and digit font. Game-specific config lives in one place (`internal/game`). Portfolio for Pokémon values holdings in BRL from the Liga floor (fx=1 trick) with a per-game ledger (`data/trades-pkm.json`).

Seeded Pokémon data: DRI + SV10/JTG/MEG/CRI/PFL + SEALED (data/tracking-pkm). Movers/sold need a 2nd daily snapshot (day-over-day), so those fill in tomorrow; inventory/buyout/browse/sealed work from day one.

Follow-ups (not blocking): buyout help text still says "English NM" (Pokémon uses all languages); default track list (17 sets) is broad — trim/expand via -pkm-track-set; Pokémon number collisions across sets are avoided in portfolio create (uses set+number) but the BR quote is number/name search only.
