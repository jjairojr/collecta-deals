package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strings"

	"opdeals/internal/compare"
	"opdeals/internal/model"
	"opdeals/internal/trades"
)

// priceLookup returns the market-price lookup and FX rate used to value a game's
// portfolio. Games with a deals snapshot value holdings from the US (TCGplayer)
// side in USD, so the FX rate converts to BRL. BR-only games have no US price,
// so the lookup is nil and FX is 1, making the valuation fall back to each
// trade's manual reference value (RefUSD, carried in BRL) — auto-filled from
// the Liga tracking floor at create time.
func (s *Server) priceLookup(gs *GameStack) (trades.PriceLookup, float64) {
	if gs.Deals == nil {
		return nil, 1
	}
	snap := gs.Deals.Snapshot()
	m := compare.MatcherFor(gs.Game)
	index := compare.USDIndex(snap.Prices, m)
	lookup := func(number, name, set string) (float64, string, bool) {
		p, ok := index[m.LookupKey(number, name, set)]
		if !ok {
			return 0, "", false
		}
		return compare.EffectiveUSD(p), p.URL, true
	}
	return lookup, snap.FXRate
}

// brFloor returns the current Liga floor (BRL) for a card in a given set from the
// game's latest tracking snapshot, used to seed a Pokémon trade's reference value.
func (s *Server) brFloor(gs *GameStack, set, number string) (float64, bool) {
	if gs.Track == nil || set == "" {
		return 0, false
	}
	day, ok, err := gs.Track.LatestDay(set)
	if err != nil || !ok {
		return 0, false
	}
	number = model.NormalizeNumber(number)
	for _, c := range day.Cards {
		if model.NormalizeNumber(c.Number) == number && c.LowBRL > 0 {
			return c.LowBRL, true
		}
	}
	return 0, false
}

// ledgersFor returns the trade ledgers a request may touch, in lookup order: the
// active game's first, then the shared accessories one. Accessories belong to no
// game, so they are stocked once and show up in every game's portfolio.
func (s *Server) ledgersFor(gs *GameStack) []*trades.Store {
	out := make([]*trades.Store, 0, 2)
	if gs.Trades != nil {
		out = append(out, gs.Trades)
	}
	if s.accessories != nil {
		out = append(out, s.accessories)
	}
	return out
}

// forEachLedger walks every trade ledger — one per game, then the shared
// accessories one under the "acessorios" pseudo-game — handing each to fn already
// valued. Accessories are visited exactly once: handleTradesList appends them to
// every game's list, so a caller fanning out over ?game= would count them as many
// times as there are games.
func (s *Server) forEachLedger(pct float64, fn func(id, name string, p trades.PortfolioResponse)) error {
	for _, id := range s.orderedGameIDs() {
		gs := s.games[id]
		if gs.Trades == nil {
			continue
		}
		all, err := gs.Trades.List()
		if err != nil {
			return err
		}
		lookup, fx := s.priceLookup(gs)
		fn(gs.Game.ID, gs.Game.Name, trades.BuildPortfolio(all, pct, fx, lookup))
	}
	if s.accessories == nil {
		return nil
	}
	all, err := s.accessories.List()
	if err != nil {
		return err
	}
	// Accessories carry their own manual value, so no price lookup and fx = 1.
	fn("acessorios", "Acessórios", trades.BuildPortfolio(all, pct, 1, nil))
	return nil
}

// onLedger runs a write against whichever ledger actually holds the id. Trade ids
// are unique across ledgers and every mutation reports ErrNotFound before it
// persists anything, so trying the game's ledger first is free: a miss just means
// the record is an accessory. Callers therefore never have to say which ledger a
// trade lives in.
func onLedger[T any](stores []*trades.Store, fn func(*trades.Store) (T, error)) (T, error) {
	var zero T
	if len(stores) == 0 {
		return zero, trades.ErrNotFound
	}
	for _, st := range stores[:len(stores)-1] {
		v, err := fn(st)
		if !errors.Is(err, trades.ErrNotFound) {
			return v, err
		}
	}
	return fn(stores[len(stores)-1])
}

func (s *Server) handleTradesList(w http.ResponseWriter, r *http.Request) {
	gs := s.stackFor(r.URL.Query())
	if gs.Trades == nil {
		http.Error(w, "trades store unavailable", http.StatusServiceUnavailable)
		return
	}
	pct := floatParam(r.URL.Query(), "pct", 90)
	all, err := gs.Trades.List()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if s.accessories != nil {
		acc, err := s.accessories.List()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		all = append(all, acc...)
		sort.SliceStable(all, func(i, j int) bool {
			return all[i].CreatedAt.After(all[j].CreatedAt)
		})
	}
	lookup, fx := s.priceLookup(gs)
	writeJSON(w, trades.BuildPortfolio(all, pct, fx, lookup))
}

// errNoIdentity rejects a trade that names no card at all: without a number or a
// name there is nothing to price, match or show.
var errNoIdentity = errors.New("number or name required")

// addTrade files a new trade under the right ledger for a game, seeding its
// reference value from whichever market the game has. Shared by the plain
// create endpoint and by adding an item to a purchase order.
func (s *Server) addTrade(gs *GameStack, t trades.Trade) (trades.Trade, error) {
	t.Number = model.NormalizeNumber(t.Number)
	if t.Number == "" && t.Name == "" {
		return trades.Trade{}, errNoIdentity
	}
	if t.RefUSD == 0 && t.Kind == "" {
		if gs.Deals != nil {
			if lookup, _ := s.priceLookup(gs); lookup != nil {
				if usd, _, ok := lookup(t.Number, t.Name, t.Set); ok {
					t.RefUSD = usd
				}
			}
		} else if brl, ok := s.brFloor(gs, t.Set, t.Number); ok {
			t.RefUSD = brl
		}
	}
	// The kind decides the ledger: an accessory is never filed under the game
	// that happened to be selected when it was logged.
	ledger := gs.Trades
	if t.Kind == trades.KindAccessory && s.accessories != nil {
		ledger = s.accessories
	}
	return ledger.Add(t)
}

func (s *Server) handleTradesCreate(w http.ResponseWriter, r *http.Request) {
	gs := s.stackFor(r.URL.Query())
	if gs.Trades == nil {
		http.Error(w, "trades store unavailable", http.StatusServiceUnavailable)
		return
	}
	var t trades.Trade
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if !trades.ValidChannel(t.Channel) || !trades.ValidWriteOff(t.WriteOff) {
		http.Error(w, "unknown channel or writeOff", http.StatusBadRequest)
		return
	}
	if t.Status == "sold" {
		t.SemLastro = true
	}
	created, err := s.addTrade(gs, t)
	if errors.Is(err, errNoIdentity) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err == nil {
		err = s.reallocateOrderShipping(created.PurchaseID)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, created)
}

func (s *Server) handleTradesUpdate(w http.ResponseWriter, r *http.Request) {
	gs := s.stackFor(r.URL.Query())
	if gs.Trades == nil {
		http.Error(w, "trades store unavailable", http.StatusServiceUnavailable)
		return
	}
	id := r.PathValue("id")
	var patch map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if _, err := patchTrade(trades.Trade{}, patch); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var patchErr error
	prevPurchase := ""
	mutate := func(t *trades.Trade) {
		prevPurchase = t.PurchaseID
		next, err := patchTrade(*t, patch)
		if err != nil {
			patchErr = err
			return
		}
		*t = next
	}
	updated, err := onLedger(s.ledgersFor(gs), func(st *trades.Store) (trades.Trade, error) {
		return st.Update(id, mutate)
	})
	if errors.Is(err, trades.ErrNotFound) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err == nil {
		err = patchErr
	}
	if err == nil && touchesOrderShipping(patch) {
		err = s.reallocateOrders(prevPurchase, updated.PurchaseID)
		if err == nil {
			updated, err = onLedger(s.ledgersFor(gs), func(st *trades.Store) (trades.Trade, error) {
				return st.Update(id, func(*trades.Trade) {})
			})
		}
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, updated)
}

func touchesOrderShipping(patch map[string]json.RawMessage) bool {
	for _, k := range []string{"qty", "buyBRL", "purchaseID", "status"} {
		if _, ok := patch[k]; ok {
			return true
		}
	}
	return false
}

func patchTrade(cur trades.Trade, patch map[string]json.RawMessage) (trades.Trade, error) {
	base, err := json.Marshal(cur)
	if err != nil {
		return trades.Trade{}, err
	}
	fields := map[string]json.RawMessage{}
	if err := json.Unmarshal(base, &fields); err != nil {
		return trades.Trade{}, err
	}
	for k, v := range patch {
		fields[k] = v
	}
	merged, err := json.Marshal(fields)
	if err != nil {
		return trades.Trade{}, err
	}
	var next trades.Trade
	if err := json.Unmarshal(merged, &next); err != nil {
		return trades.Trade{}, err
	}
	next.ID = cur.ID
	next.CreatedAt = cur.CreatedAt
	next.OrderShippingBRL = cur.OrderShippingBRL
	next.Number = model.NormalizeNumber(next.Number)
	if next.RefUSD == 0 {
		next.RefUSD = cur.RefUSD
	}
	if next.ManualBRL == 0 {
		next.ManualBRL = cur.ManualBRL
	}
	switch next.Kind {
	case "":
		next.Kind = cur.Kind
	case trades.KindSingle:
		next.Kind = ""
	}
	if next.PurchaseID == "" {
		next.PurchaseID = cur.PurchaseID
	}
	if next.Status == "" {
		next.Status = cur.Status
	}
	if next.Photos == nil {
		next.Photos = cur.Photos
	}
	if next.ImageURL == "" && len(next.Photos) > 0 {
		next.ImageURL = next.Photos[0]
	}
	return next, nil
}

func (s *Server) handleTradesSell(w http.ResponseWriter, r *http.Request) {
	gs := s.stackFor(r.URL.Query())
	if gs.Trades == nil {
		http.Error(w, "trades store unavailable", http.StatusServiceUnavailable)
		return
	}
	var in struct {
		Qty          int     `json:"qty"`
		SellPrice    float64 `json:"sellPrice"`
		SellCurrency string  `json:"sellCurrency"`
		SellDate     string  `json:"sellDate"`
		Buyer        string  `json:"buyer"`
		Channel      string  `json:"channel"`
		WriteOff     string  `json:"writeOff"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if !trades.ValidChannel(in.Channel) || !trades.ValidWriteOff(in.WriteOff) {
		http.Error(w, "unknown channel or writeOff", http.StatusBadRequest)
		return
	}
	if in.WriteOff != "" {
		in.SellPrice = 0
	}
	sale := trades.Sale{
		Qty:          in.Qty,
		SellPrice:    in.SellPrice,
		SellCurrency: in.SellCurrency,
		SellDate:     in.SellDate,
		Buyer:        in.Buyer,
		Channel:      in.Channel,
		WriteOff:     in.WriteOff,
	}
	sold, err := onLedger(s.ledgersFor(gs), func(st *trades.Store) (trades.Trade, error) {
		return st.Sell(r.PathValue("id"), sale)
	})
	if errors.Is(err, trades.ErrNotFound) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if errors.Is(err, trades.ErrAlreadySold) {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	game := gs.Game.ID
	if sold.Kind == trades.KindAccessory {
		game = accessoryGameID
	}
	s.queueStockExit(game, sold)
	writeJSON(w, sold)
}

func (s *Server) handleTradesUnsell(w http.ResponseWriter, r *http.Request) {
	gs := s.stackFor(r.URL.Query())
	if gs.Trades == nil {
		http.Error(w, "trades store unavailable", http.StatusServiceUnavailable)
		return
	}
	res, err := onLedger(s.ledgersFor(gs), func(st *trades.Store) (trades.Unsold, error) {
		return st.Unsell(r.PathValue("id"))
	})
	if errors.Is(err, trades.ErrNotFound) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if errors.Is(err, trades.ErrNotSold) || errors.Is(err, trades.ErrSiteSale) {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	game := gs.Game.ID
	if res.Sale.Kind == trades.KindAccessory {
		game = accessoryGameID
	}
	s.queueStockReturn(game, res.Sale)
	if err := s.reallocateOrderShipping(res.Restored.PurchaseID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, res.Restored)
}

func (s *Server) handleTradesDelete(w http.ResponseWriter, r *http.Request) {
	gs := s.stackFor(r.URL.Query())
	if gs.Trades == nil {
		http.Error(w, "trades store unavailable", http.StatusServiceUnavailable)
		return
	}
	owners, err := s.purchaseIDsOf([]string{r.PathValue("id")})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, err = onLedger(s.ledgersFor(gs), func(st *trades.Store) (struct{}, error) {
		return struct{}{}, st.Delete(r.PathValue("id"))
	})
	if errors.Is(err, trades.ErrNotFound) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err == nil {
		err = s.reallocateOrders(owners...)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

// handleTradesMerge collapses several holdings into one, summing quantity and
// shipping and averaging the buy price (see trades.Store.Merge). The selection
// must sit in a single ledger: game holdings merge with game holdings and
// accessories with accessories, never across the two.
func (s *Server) handleTradesMerge(w http.ResponseWriter, r *http.Request) {
	gs := s.stackFor(r.URL.Query())
	if gs.Trades == nil {
		http.Error(w, "trades store unavailable", http.StatusServiceUnavailable)
		return
	}
	var in struct {
		IDs []string `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	owners, err := s.purchaseIDsOf(in.IDs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	merged, err := onLedger(s.ledgersFor(gs), func(st *trades.Store) (trades.Trade, error) {
		return st.Merge(in.IDs)
	})
	if errors.Is(err, trades.ErrNotFound) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.reallocateOrders(owners...); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, merged)
}

type quoteMatch struct {
	Number     string  `json:"number"`
	Name       string  `json:"name"`
	Set        string  `json:"set"`
	Variant    string  `json:"variant,omitempty"`
	MarketUSD  float64 `json:"marketUSD"`
	MarketBRL  float64 `json:"marketBRL"`
	LigaLowBRL float64 `json:"ligaLowBRL,omitempty"`
	LigaAvgBRL float64 `json:"ligaAvgBRL,omitempty"`
	LigaURL    string  `json:"ligaUrl,omitempty"`
	ProductID  int     `json:"productID,omitempty"`
}

type quoteResponse struct {
	FXRate  float64      `json:"fxRate"`
	Matches []quoteMatch `json:"matches"`
}

func quoteRank(q, number, name string) int {
	ln := strings.ToLower(number)
	lm := strings.ToLower(name)
	switch {
	case ln == q || lm == q:
		return 0
	case strings.HasPrefix(ln, q) || strings.HasPrefix(lm, q):
		return 1
	case strings.Contains(ln, q):
		return 2
	default:
		return 3
	}
}

func quoteLimit(q map[string][]string) int {
	limit := intParam(q, "limit", 25)
	if limit < 1 {
		limit = 1
	}
	if limit > 100 {
		limit = 100
	}
	return limit
}

func (s *Server) handleTradesQuote(w http.ResponseWriter, r *http.Request) {
	gs := s.stackFor(r.URL.Query())
	q := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	limit := quoteLimit(r.URL.Query())
	resp := quoteResponse{Matches: []quoteMatch{}}
	if q == "" {
		writeJSON(w, resp)
		return
	}
	if r.URL.Query().Get("kind") == "sealed" {
		s.quoteSealed(gs, q, limit, &resp)
		writeJSON(w, resp)
		return
	}
	if gs.Deals == nil {
		s.quoteFromTracking(gs, q, limit, &resp)
		writeJSON(w, resp)
		return
	}
	snap := gs.Deals.Snapshot()
	resp.FXRate = snap.FXRate
	m := compare.MatcherFor(gs.Game)
	brIndex := compare.BRIndex(snap.Listings, m)
	brPrint := compare.BRPrintIndex(snap.Listings, m)
	for _, p := range snap.Prices {
		if !strings.Contains(strings.ToLower(p.Number), q) && !strings.Contains(strings.ToLower(p.Name), q) {
			continue
		}
		usd := compare.EffectiveUSD(p)
		if usd <= 0 {
			continue
		}
		brl := 0.0
		if snap.FXRate > 0 {
			brl = usd / snap.FXRate
		}
		set := p.SetCode
		if set == "" {
			set = setFromNumber(p.Number)
		}
		match := quoteMatch{
			Number:    p.Number,
			Name:      p.Name,
			Set:       set,
			Variant:   p.Variant,
			MarketUSD: usd,
			MarketBRL: brl,
			ProductID: p.ProductID,
		}
		l, ok := brPrint[m.PrintKey(p.Number, p.Name, p.SetCode)]
		if !ok {
			l, ok = brIndex[m.Key(p.Number, p.Name, p.SetCode)]
		}
		if ok {
			match.LigaLowBRL = l.LowBRL
			match.LigaAvgBRL = l.AvgBRL
			match.LigaURL = l.URL
		}
		resp.Matches = append(resp.Matches, match)
	}
	sort.SliceStable(resp.Matches, func(i, j int) bool {
		ri := quoteRank(q, resp.Matches[i].Number, resp.Matches[i].Name)
		rj := quoteRank(q, resp.Matches[j].Number, resp.Matches[j].Name)
		if ri != rj {
			return ri < rj
		}
		return resp.Matches[i].MarketUSD > resp.Matches[j].MarketUSD
	})
	if len(resp.Matches) > limit {
		resp.Matches = resp.Matches[:limit]
	}
	writeJSON(w, resp)
}

// quoteFromTracking answers a BR-only (Pokémon) quote from the game's latest
// tracking snapshots, returning the Liga floor in BRL. FXRate is 1 so the UI can
// treat MarketBRL/MarketUSD interchangeably for a BR-priced portfolio.
func (s *Server) quoteFromTracking(gs *GameStack, q string, limit int, resp *quoteResponse) {
	resp.FXRate = 1
	if gs.Track == nil {
		return
	}
	for _, set := range s.trackSets(gs) {
		day, ok, err := gs.Track.LatestDay(set)
		if err != nil || !ok {
			continue
		}
		for _, c := range day.Cards {
			if !strings.Contains(strings.ToLower(c.Number), q) && !strings.Contains(strings.ToLower(c.Name), q) {
				continue
			}
			if c.LowBRL <= 0 {
				continue
			}
			resp.Matches = append(resp.Matches, quoteMatch{
				Number:     c.Number,
				Name:       c.Name,
				Set:        set,
				MarketUSD:  c.LowBRL,
				MarketBRL:  c.LowBRL,
				LigaLowBRL: c.LowBRL,
				LigaURL:    c.URL,
			})
		}
	}
	sort.SliceStable(resp.Matches, func(i, j int) bool {
		ri := quoteRank(q, resp.Matches[i].Number, resp.Matches[i].Name)
		rj := quoteRank(q, resp.Matches[j].Number, resp.Matches[j].Name)
		if ri != rj {
			return ri < rj
		}
		return resp.Matches[i].MarketBRL > resp.Matches[j].MarketBRL
	})
	if len(resp.Matches) > limit {
		resp.Matches = resp.Matches[:limit]
	}
}

func (s *Server) quoteSealed(gs *GameStack, q string, limit int, resp *quoteResponse) {
	resp.FXRate = 1
	if gs.Track == nil {
		return
	}
	day, ok, err := gs.Track.LatestDay("SEALED")
	if err != nil || !ok {
		return
	}
	for _, c := range day.Cards {
		if !strings.Contains(strings.ToLower(c.Number), q) && !strings.Contains(strings.ToLower(c.Name), q) {
			continue
		}
		resp.Matches = append(resp.Matches, quoteMatch{
			Number:  c.Number,
			Name:    c.Name,
			Set:     "SEALED",
			LigaURL: c.URL,
		})
	}
	sort.SliceStable(resp.Matches, func(i, j int) bool {
		return quoteRank(q, resp.Matches[i].Number, resp.Matches[i].Name) < quoteRank(q, resp.Matches[j].Number, resp.Matches[j].Name)
	})
	if len(resp.Matches) > limit {
		resp.Matches = resp.Matches[:limit]
	}
}

func setFromNumber(number string) string {
	if i := strings.IndexByte(number, '-'); i > 0 {
		return number[:i]
	}
	return number
}
