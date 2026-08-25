package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"sort"

	"opdeals/internal/purchases"
	"opdeals/internal/trades"
)

// purchaseRow is one bought item carrying the game it belongs to. A buy order
// routinely mixes games — the same seller ships One Piece and Pokémon in one box
// — so the purchases page reads every ledger at once instead of following the
// game switcher.
type purchaseRow struct {
	trades.TradeView
	Game     string `json:"game"`
	GameName string `json:"gameName"`
}

// purchaseView is a buy order with the items filed under it and what it cost.
// TotalBRL adds the order's own freight to the items, which already carry any
// per-item freight of their own.
type purchaseView struct {
	purchases.Purchase
	Items        []purchaseRow `json:"items"`
	Units        int           `json:"units"`
	ItemsCostBRL float64       `json:"itemsCostBRL"`
	TotalBRL     float64       `json:"totalBRL"`
	Sold         int           `json:"sold"`
	ProceedsBRL  float64       `json:"proceedsBRL"`
}

type purchasesResponse struct {
	Purchases []purchaseView `json:"purchases"`
	// Unassigned are items bought before orders existed, or logged straight into a
	// game's portfolio. They are listed so nothing is invisible here and so they
	// can be attached to an order.
	Unassigned []purchaseRow `json:"unassigned"`
}

// allPurchaseRows reads every trade of every ledger once, valued, tagged with its
// game.
func (s *Server) allPurchaseRows(pct float64) ([]purchaseRow, error) {
	rows := []purchaseRow{}
	err := s.forEachLedger(pct, func(id, name string, p trades.PortfolioResponse) {
		for _, v := range p.Trades {
			rows = append(rows, purchaseRow{TradeView: v, Game: id, GameName: name})
		}
	})
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func sortPurchaseRows(rows []purchaseRow) {
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].BuyDate != rows[j].BuyDate {
			return rows[i].BuyDate > rows[j].BuyDate
		}
		return rows[i].CreatedAt.After(rows[j].CreatedAt)
	})
}

// handlePurchases returns every buy order with its items, plus the items that
// belong to no order. Sold items stay listed: a sale never unmakes the purchase
// that paid for it.
func (s *Server) handlePurchases(w http.ResponseWriter, r *http.Request) {
	if s.purchases == nil {
		http.Error(w, "purchases store unavailable", http.StatusServiceUnavailable)
		return
	}
	pct := floatParam(r.URL.Query(), "pct", 90)
	rows, err := s.allPurchaseRows(pct)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	orders, err := s.purchases.List()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	byOrder := make(map[string][]purchaseRow, len(orders))
	resp := purchasesResponse{Purchases: []purchaseView{}, Unassigned: []purchaseRow{}}
	known := make(map[string]bool, len(orders))
	for _, p := range orders {
		known[p.ID] = true
	}
	for _, row := range rows {
		// An item pointing at a deleted order reads as loose rather than vanishing.
		if row.PurchaseID != "" && known[row.PurchaseID] {
			byOrder[row.PurchaseID] = append(byOrder[row.PurchaseID], row)
			continue
		}
		resp.Unassigned = append(resp.Unassigned, row)
	}
	sortPurchaseRows(resp.Unassigned)
	for _, p := range orders {
		items := byOrder[p.ID]
		sortPurchaseRows(items)
		view := purchaseView{Purchase: p, Items: items, TotalBRL: p.ShippingBRL}
		for _, it := range items {
			view.Units += it.Qty
			view.ItemsCostBRL += it.CostBRL
			if it.Realized {
				view.Sold++
				view.ProceedsBRL += it.ValueBRL
			}
		}
		view.TotalBRL += view.ItemsCostBRL
		if view.Items == nil {
			view.Items = []purchaseRow{}
		}
		resp.Purchases = append(resp.Purchases, view)
	}
	writeJSON(w, resp)
}

func (s *Server) handlePurchasesCreate(w http.ResponseWriter, r *http.Request) {
	if s.purchases == nil {
		http.Error(w, "purchases store unavailable", http.StatusServiceUnavailable)
		return
	}
	var in purchases.Purchase
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	created, err := s.purchases.Add(in)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, created)
}

func (s *Server) handlePurchasesUpdate(w http.ResponseWriter, r *http.Request) {
	if s.purchases == nil {
		http.Error(w, "purchases store unavailable", http.StatusServiceUnavailable)
		return
	}
	var in purchases.Purchase
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	updated, err := s.purchases.Update(r.PathValue("id"), in)
	if errors.Is(err, purchases.ErrNotFound) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, updated)
}

// handlePurchasesDelete removes an order header. It refuses while items still
// point at it: deleting the order must never look like a way to delete stock.
func (s *Server) handlePurchasesDelete(w http.ResponseWriter, r *http.Request) {
	if s.purchases == nil {
		http.Error(w, "purchases store unavailable", http.StatusServiceUnavailable)
		return
	}
	id := r.PathValue("id")
	rows, err := s.allPurchaseRows(90)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for _, row := range rows {
		if row.PurchaseID == id {
			http.Error(w, "purchase still has items", http.StatusConflict)
			return
		}
	}
	if err := s.purchases.Delete(id); errors.Is(err, purchases.ErrNotFound) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

// handlePurchaseItemCreate logs an item straight into a buy order: the trade is
// created in the ledger of the game given by ?game=, already pointing at the
// order, and inherits the order's date and seller when the item leaves them out.
func (s *Server) handlePurchaseItemCreate(w http.ResponseWriter, r *http.Request) {
	if s.purchases == nil {
		http.Error(w, "purchases store unavailable", http.StatusServiceUnavailable)
		return
	}
	gs := s.stackFor(r.URL.Query())
	if gs.Trades == nil {
		http.Error(w, "trades store unavailable", http.StatusServiceUnavailable)
		return
	}
	id := r.PathValue("id")
	order, err := s.purchaseByID(id)
	if errors.Is(err, purchases.ErrNotFound) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var t trades.Trade
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	t.PurchaseID = id
	if t.BuyDate == "" {
		t.BuyDate = order.Date
	}
	if t.Store == "" {
		t.Store = order.Store
	}
	created, err := s.addTrade(gs, t)
	if errors.Is(err, errNoIdentity) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, created)
}

func (s *Server) purchaseByID(id string) (purchases.Purchase, error) {
	all, err := s.purchases.List()
	if err != nil {
		return purchases.Purchase{}, err
	}
	for _, p := range all {
		if p.ID == id {
			return p, nil
		}
	}
	return purchases.Purchase{}, purchases.ErrNotFound
}

type itemRef struct {
	ID   string `json:"id"`
	Game string `json:"game"`
}

// reassign points existing items at an order, or at none when target is empty.
// Each item says which game it came from because the ids live in different
// ledgers; "acessorios" is the shared one.
func (s *Server) reassign(w http.ResponseWriter, r *http.Request, target string) {
	if s.purchases == nil {
		http.Error(w, "purchases store unavailable", http.StatusServiceUnavailable)
		return
	}
	if target != "" {
		if _, err := s.purchaseByID(target); errors.Is(err, purchases.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		} else if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	var in struct {
		Items []itemRef `json:"items"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	updated := 0
	for _, ref := range in.Items {
		ledger := s.accessories
		if gs, ok := s.games[ref.Game]; ok && gs.Trades != nil {
			ledger = gs.Trades
		}
		if ledger == nil {
			continue
		}
		_, err := ledger.Update(ref.ID, func(t *trades.Trade) { t.PurchaseID = target })
		if errors.Is(err, trades.ErrNotFound) {
			continue
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		updated++
	}
	writeJSON(w, map[string]int{"updated": updated})
}

func (s *Server) handlePurchaseAttach(w http.ResponseWriter, r *http.Request) {
	s.reassign(w, r, r.PathValue("id"))
}

func (s *Server) handlePurchaseDetach(w http.ResponseWriter, r *http.Request) {
	s.reassign(w, r, "")
}
