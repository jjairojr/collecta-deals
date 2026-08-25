package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"opdeals/internal/logx"
	"opdeals/internal/purchases"
	"opdeals/internal/trades"
)

func purchaseServer(t *testing.T) (*Server, *trades.Store, *trades.Store) {
	t.Helper()
	s, op, acc := twoGameServer(t)
	s.purchases = purchases.NewStore(filepath.Join(t.TempDir(), "purchases.json"), logx.New(io.Discard))
	return s, op, acc
}

func newOrder(t *testing.T, s *Server, body string) purchases.Purchase {
	t.Helper()
	rec := httptest.NewRecorder()
	s.handlePurchasesCreate(rec, httptest.NewRequest(http.MethodPost, "/api/purchases", strings.NewReader(body)))
	if rec.Code != http.StatusOK {
		t.Fatalf("create purchase status = %d: %s", rec.Code, rec.Body.String())
	}
	var out purchases.Purchase
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func addItem(t *testing.T, s *Server, orderID, gameID, body string) trades.Trade {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/purchases/"+orderID+"/items?game="+gameID, strings.NewReader(body))
	req.SetPathValue("id", orderID)
	rec := httptest.NewRecorder()
	s.handlePurchaseItemCreate(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("add item status = %d: %s", rec.Code, rec.Body.String())
	}
	var out trades.Trade
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func purchasesOf(t *testing.T, s *Server) purchasesResponse {
	t.Helper()
	rec := httptest.NewRecorder()
	s.handlePurchases(rec, httptest.NewRequest(http.MethodGet, "/api/purchases", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("purchases status = %d: %s", rec.Code, rec.Body.String())
	}
	var resp purchasesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	return resp
}

// One order holds items of different games, and its total is the items plus the
// order's own freight.
func TestPurchaseGroupsItemsAcrossGames(t *testing.T) {
	s, _, _ := purchaseServer(t)
	order := newOrder(t, s, `{"date":"2026-08-20","store":"Legends","shippingBRL":30}`)

	addItem(t, s, order.ID, "onepiece", `{"number":"OP16-080","name":"Teach","set":"OP16","qty":2,"buyBRL":100}`)
	addItem(t, s, order.ID, "pokemon", `{"number":"107","name":"Samurott","set":"SSP","qty":1,"buyBRL":50,"shippingBRL":5}`)

	resp := purchasesOf(t, s)
	if len(resp.Purchases) != 1 {
		t.Fatalf("purchases = %d, want 1", len(resp.Purchases))
	}
	p := resp.Purchases[0]
	if len(p.Items) != 2 || p.Units != 3 {
		t.Fatalf("items = %d units = %d, want 2 items / 3 units", len(p.Items), p.Units)
	}
	// 2×100 + (50 + 5) = 255 of items, plus 30 of order freight.
	if p.ItemsCostBRL != 255 || p.TotalBRL != 285 {
		t.Fatalf("items cost = %v total = %v, want 255 / 285", p.ItemsCostBRL, p.TotalBRL)
	}
	games := map[string]bool{p.Items[0].Game: true, p.Items[1].Game: true}
	if !games["onepiece"] || !games["pokemon"] {
		t.Fatalf("items lost their game: %+v", p.Items)
	}
	if len(resp.Unassigned) != 0 {
		t.Fatalf("unassigned = %d, want 0", len(resp.Unassigned))
	}
}

// An item logged into an order inherits the order's date and seller, so a whole
// order does not have to be retyped item by item.
func TestPurchaseItemInheritsOrderDateAndStore(t *testing.T) {
	s, _, _ := purchaseServer(t)
	order := newOrder(t, s, `{"date":"2026-08-20","store":"Legends"}`)
	item := addItem(t, s, order.ID, "onepiece", `{"number":"OP16-080","name":"Teach","set":"OP16","qty":1,"buyBRL":100}`)

	if item.BuyDate != "2026-08-20" || item.Store != "Legends" {
		t.Fatalf("item = date %q store %q, want the order's", item.BuyDate, item.Store)
	}
	if item.PurchaseID != order.ID {
		t.Fatalf("item purchaseID = %q, want %q", item.PurchaseID, order.ID)
	}
}

// Holdings logged before orders existed are listed as loose, can be attached to
// an order and detached again.
func TestAttachAndDetachExistingHoldings(t *testing.T) {
	s, op, acc := purchaseServer(t)
	card, _ := op.Add(trades.Trade{Number: "OP15-119", Name: "Luffy", Set: "OP15", Qty: 1, BuyBRL: 60, Status: "holding"})
	sleeves, _ := acc.Add(trades.Trade{Kind: trades.KindAccessory, Name: "Sleeves", Qty: 2, BuyBRL: 25, Status: "holding"})

	if got := purchasesOf(t, s); len(got.Unassigned) != 2 {
		t.Fatalf("unassigned = %d, want the 2 pre-existing holdings", len(got.Unassigned))
	}

	order := newOrder(t, s, `{"date":"2026-08-21","store":"Balcão"}`)
	body := `{"items":[{"id":"` + card.ID + `","game":"onepiece"},{"id":"` + sleeves.ID + `","game":"acessorios"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/purchases/"+order.ID+"/attach", strings.NewReader(body))
	req.SetPathValue("id", order.ID)
	rec := httptest.NewRecorder()
	s.handlePurchaseAttach(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("attach status = %d: %s", rec.Code, rec.Body.String())
	}

	got := purchasesOf(t, s)
	if len(got.Unassigned) != 0 || len(got.Purchases[0].Items) != 2 {
		t.Fatalf("after attach: unassigned %d, items %d", len(got.Unassigned), len(got.Purchases[0].Items))
	}

	rec = httptest.NewRecorder()
	s.handlePurchaseDetach(rec, httptest.NewRequest(http.MethodPost, "/api/purchases/detach",
		strings.NewReader(`{"items":[{"id":"`+sleeves.ID+`","game":"acessorios"}]}`)))
	if rec.Code != http.StatusOK {
		t.Fatalf("detach status = %d: %s", rec.Code, rec.Body.String())
	}
	got = purchasesOf(t, s)
	if len(got.Unassigned) != 1 || got.Unassigned[0].Name != "Sleeves" {
		t.Fatalf("after detach: unassigned %+v", got.Unassigned)
	}
	if len(got.Purchases[0].Items) != 1 {
		t.Fatalf("after detach the order should keep 1 item, got %d", len(got.Purchases[0].Items))
	}
}

// Deleting an order must never look like a way to delete stock.
func TestDeletePurchaseRefusesWhileItHasItems(t *testing.T) {
	s, _, _ := purchaseServer(t)
	order := newOrder(t, s, `{"date":"2026-08-20","store":"Legends"}`)
	addItem(t, s, order.ID, "onepiece", `{"number":"OP16-080","name":"Teach","set":"OP16","qty":1,"buyBRL":100}`)

	del := func() int {
		req := httptest.NewRequest(http.MethodDelete, "/api/purchases/"+order.ID, nil)
		req.SetPathValue("id", order.ID)
		rec := httptest.NewRecorder()
		s.handlePurchasesDelete(rec, req)
		return rec.Code
	}
	if code := del(); code != http.StatusConflict {
		t.Fatalf("delete with items = %d, want 409", code)
	}

	rec := httptest.NewRecorder()
	s.handlePurchaseDetach(rec, httptest.NewRequest(http.MethodPost, "/api/purchases/detach",
		strings.NewReader(`{"items":[{"id":"`+purchasesOf(t, s).Purchases[0].Items[0].ID+`","game":"onepiece"}]}`)))
	if code := del(); code != http.StatusOK {
		t.Fatalf("delete after detach = %d, want 200", code)
	}
	if got := purchasesOf(t, s); len(got.Purchases) != 0 || len(got.Unassigned) != 1 {
		t.Fatalf("after delete: purchases %d, unassigned %d", len(got.Purchases), len(got.Unassigned))
	}
}

// Editing an item through the plain trade endpoint must not silently drop the
// order it belongs to.
func TestUpdateKeepsPurchaseLink(t *testing.T) {
	s, _, _ := purchaseServer(t)
	order := newOrder(t, s, `{"date":"2026-08-20","store":"Legends"}`)
	item := addItem(t, s, order.ID, "onepiece", `{"number":"OP16-080","name":"Teach","set":"OP16","qty":1,"buyBRL":100}`)

	req := httptest.NewRequest(http.MethodPut, "/api/trades/"+item.ID+"?game=onepiece",
		strings.NewReader(`{"number":"OP16-080","name":"Teach","set":"OP16","qty":2,"buyBRL":110}`))
	req.SetPathValue("id", item.ID)
	rec := httptest.NewRecorder()
	s.handleTradesUpdate(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("update status = %d: %s", rec.Code, rec.Body.String())
	}
	var out trades.Trade
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.PurchaseID != order.ID {
		t.Fatalf("purchaseID = %q after edit, want %q", out.PurchaseID, order.ID)
	}
}
