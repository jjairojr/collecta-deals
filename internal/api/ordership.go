package api

import (
	"errors"
	"net/http"

	"opdeals/internal/purchases"
	"opdeals/internal/trades"
)

func (s *Server) allLedgers() []*trades.Store {
	out := make([]*trades.Store, 0, len(s.games)+1)
	for _, id := range s.orderedGameIDs() {
		if gs := s.games[id]; gs.Trades != nil {
			out = append(out, gs.Trades)
		}
	}
	if s.accessories != nil {
		out = append(out, s.accessories)
	}
	return out
}

func (s *Server) reallocateOrderShipping(purchaseID string) error {
	if s.purchases == nil || purchaseID == "" {
		return nil
	}
	order, err := s.purchaseByID(purchaseID)
	if errors.Is(err, purchases.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	ledgers := s.allLedgers()
	holding := make([][]trades.Trade, len(ledgers))
	var lots []trades.Trade
	remaining := order.ShippingBRL
	for i, st := range ledgers {
		all, err := st.List()
		if err != nil {
			return err
		}
		for _, t := range all {
			if t.PurchaseID != purchaseID {
				continue
			}
			if t.Status == "sold" {
				remaining -= t.OrderShippingBRL
				continue
			}
			holding[i] = append(holding[i], t)
			lots = append(lots, t)
		}
	}
	shares := trades.AllocateOrderShipping(max(remaining, 0), lots)
	for i, st := range ledgers {
		mine := make(map[string]float64, len(holding[i]))
		for _, t := range holding[i] {
			mine[t.ID] = shares[t.ID]
		}
		if _, err := st.SetOrderShipping(mine); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) reallocateOrders(ids ...string) error {
	seen := make(map[string]bool, len(ids))
	for _, id := range ids {
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		if err := s.reallocateOrderShipping(id); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) purchaseIDsOf(ids []string) ([]string, error) {
	want := make(map[string]bool, len(ids))
	for _, id := range ids {
		want[id] = true
	}
	var out []string
	for _, st := range s.allLedgers() {
		all, err := st.List()
		if err != nil {
			return nil, err
		}
		for _, t := range all {
			if want[t.ID] && t.PurchaseID != "" {
				out = append(out, t.PurchaseID)
			}
		}
	}
	return out, nil
}

func (s *Server) handlePurchasesReallocate(w http.ResponseWriter, _ *http.Request) {
	if s.purchases == nil {
		http.Error(w, "purchases store unavailable", http.StatusServiceUnavailable)
		return
	}
	orders, err := s.purchases.List()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for _, p := range orders {
		if err := s.reallocateOrderShipping(p.ID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	writeJSON(w, map[string]int{"purchases": len(orders)})
}
