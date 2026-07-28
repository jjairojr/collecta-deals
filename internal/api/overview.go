package api

import (
	"net/http"
	"sort"

	"opdeals/internal/trades"
)

type gamePortfolio struct {
	Game    gameInfo       `json:"game"`
	Summary trades.Summary `json:"summary"`
}

type portfolioAllResponse struct {
	TargetPct float64         `json:"targetPct"`
	Total     trades.Summary  `json:"total"`
	Games     []gamePortfolio `json:"games"`
	// Accessories are shared by every game, so they are counted once here instead
	// of inside any game's row. Absent when there is no accessories ledger.
	Accessories *trades.Summary `json:"accessories,omitempty"`
}

func (s *Server) orderedGameIDs() []string {
	ids := make([]string, 0, len(s.games))
	for id := range s.games {
		if id != s.defaultGame {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	if _, ok := s.games[s.defaultGame]; ok {
		ids = append([]string{s.defaultGame}, ids...)
	}
	return ids
}

func (s *Server) handlePortfolioAll(w http.ResponseWriter, r *http.Request) {
	pct := floatParam(r.URL.Query(), "pct", 90)
	resp := portfolioAllResponse{
		TargetPct: pct,
		Total:     trades.Summary{TargetPct: pct},
		Games:     []gamePortfolio{},
	}
	addTotal := func(sum trades.Summary) {
		resp.Total.Holdings += sum.Holdings
		resp.Total.InvestedBRL += sum.InvestedBRL
		resp.Total.MarketBRL += sum.MarketBRL
		resp.Total.Sold += sum.Sold
		resp.Total.CostOfSoldBRL += sum.CostOfSoldBRL
		resp.Total.ProceedsBRL += sum.ProceedsBRL
	}
	for _, id := range s.orderedGameIDs() {
		gs := s.games[id]
		if gs.Trades == nil {
			continue
		}
		all, err := gs.Trades.List()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		lookup, fx := s.priceLookup(gs)
		p := trades.BuildPortfolio(all, pct, fx, lookup)
		resp.Games = append(resp.Games, gamePortfolio{Game: s.gameInfoFor(gs), Summary: p.Summary})
		addTotal(p.Summary)
	}
	if s.accessories != nil {
		all, err := s.accessories.List()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// Accessories carry their own manual value, so no price lookup and fx = 1.
		p := trades.BuildPortfolio(all, pct, 1, nil)
		resp.Accessories = &p.Summary
		addTotal(p.Summary)
	}
	resp.Total.UnrealizedBRL = resp.Total.MarketBRL - resp.Total.InvestedBRL
	resp.Total.RealizedBRL = resp.Total.ProceedsBRL - resp.Total.CostOfSoldBRL
	resp.Total.TotalPnLBRL = resp.Total.RealizedBRL + resp.Total.UnrealizedBRL
	writeJSON(w, resp)
}
