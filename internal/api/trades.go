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
	lookup, fx := s.priceLookup(gs)
	writeJSON(w, trades.BuildPortfolio(all, pct, fx, lookup))
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
	t.Number = model.NormalizeNumber(t.Number)
	if t.Number == "" && t.Name == "" {
		http.Error(w, "number or name required", http.StatusBadRequest)
		return
	}
	if t.RefUSD == 0 && t.Kind != "sealed" {
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
	created, err := gs.Trades.Add(t)
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
	var in trades.Trade
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	in.Number = model.NormalizeNumber(in.Number)
	updated, err := gs.Trades.Update(id, func(t *trades.Trade) {
		in.ID = t.ID
		in.CreatedAt = t.CreatedAt
		if in.RefUSD == 0 {
			in.RefUSD = t.RefUSD
		}
		if in.ManualBRL == 0 {
			in.ManualBRL = t.ManualBRL
		}
		if in.Kind == "" {
			in.Kind = t.Kind
		}
		if in.Status == "" {
			in.Status = t.Status
		}
		*t = in
	})
	if errors.Is(err, trades.ErrNotFound) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, updated)
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
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	sold, err := gs.Trades.Sell(r.PathValue("id"), trades.Sale{
		Qty:          in.Qty,
		SellPrice:    in.SellPrice,
		SellCurrency: in.SellCurrency,
		SellDate:     in.SellDate,
		Buyer:        in.Buyer,
	})
	if errors.Is(err, trades.ErrNotFound) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, sold)
}

func (s *Server) handleTradesDelete(w http.ResponseWriter, r *http.Request) {
	gs := s.stackFor(r.URL.Query())
	if gs.Trades == nil {
		http.Error(w, "trades store unavailable", http.StatusServiceUnavailable)
		return
	}
	err := gs.Trades.Delete(r.PathValue("id"))
	if errors.Is(err, trades.ErrNotFound) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
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
