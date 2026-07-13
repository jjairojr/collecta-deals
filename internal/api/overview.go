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

		resp.Total.Holdings += p.Summary.Holdings
		resp.Total.InvestedBRL += p.Summary.InvestedBRL
		resp.Total.MarketBRL += p.Summary.MarketBRL
		resp.Total.Sold += p.Summary.Sold
		resp.Total.CostOfSoldBRL += p.Summary.CostOfSoldBRL
		resp.Total.ProceedsBRL += p.Summary.ProceedsBRL
	}
	resp.Total.UnrealizedBRL = resp.Total.MarketBRL - resp.Total.InvestedBRL
	resp.Total.RealizedBRL = resp.Total.ProceedsBRL - resp.Total.CostOfSoldBRL
	resp.Total.TotalPnLBRL = resp.Total.RealizedBRL + resp.Total.UnrealizedBRL
	writeJSON(w, resp)
}
