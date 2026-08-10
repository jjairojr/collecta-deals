package api

import (
	"net/http"
	"sort"

	"opdeals/internal/trades"
)

// saleRow is one realized sale carrying the game it belongs to. The Liga packs a
// single order out of whatever the buyer picked, so an order mixes games — the
// sales page has to read every ledger at once instead of following the game
// switcher.
type saleRow struct {
	trades.TradeView
	Game     string `json:"game"`
	GameName string `json:"gameName"`
}

type salesResponse struct {
	Sales []saleRow `json:"sales"`
}

// handleSales returns every sold trade across all games, plus the shared
// accessories ledger counted once. handleTradesList appends accessories to each
// game's list, so a client fanning out over ?game= would count those sales as
// many times as there are games.
func (s *Server) handleSales(w http.ResponseWriter, r *http.Request) {
	pct := floatParam(r.URL.Query(), "pct", 90)
	resp := salesResponse{Sales: []saleRow{}}
	collect := func(id, name string, all []trades.Trade, fx float64, lookup trades.PriceLookup) {
		for _, v := range trades.BuildPortfolio(all, pct, fx, lookup).Trades {
			if !v.Realized {
				continue
			}
			resp.Sales = append(resp.Sales, saleRow{TradeView: v, Game: id, GameName: name})
		}
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
		collect(gs.Game.ID, gs.Game.Name, all, fx, lookup)
	}
	if s.accessories != nil {
		all, err := s.accessories.List()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		collect("acessorios", "Acessórios", all, 1, nil)
	}
	sort.SliceStable(resp.Sales, func(i, j int) bool {
		if resp.Sales[i].SellDate != resp.Sales[j].SellDate {
			return resp.Sales[i].SellDate > resp.Sales[j].SellDate
		}
		return resp.Sales[i].UpdatedAt.After(resp.Sales[j].UpdatedAt)
	})
	writeJSON(w, resp)
}
