package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"opdeals/internal/trades"
)

func orderShipOf(t *testing.T, st *trades.Store, id string) float64 {
	t.Helper()
	all, err := st.List()
	if err != nil {
		t.Fatal(err)
	}
	for _, tr := range all {
		if tr.ID == id {
			return tr.OrderShippingBRL
		}
	}
	t.Fatalf("trade %s not found", id)
	return 0
}

func updateOrder(t *testing.T, s *Server, id, body string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPut, "/api/purchases/"+id, strings.NewReader(body))
	req.SetPathValue("id", id)
	rec := httptest.NewRecorder()
	s.handlePurchasesUpdate(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("update purchase = %d: %s", rec.Code, rec.Body.String())
	}
}

func TestOrderShippingSpreadsAcrossGamesAndAccessories(t *testing.T) {
	s, op, acc := purchaseServer(t)
	order := newOrder(t, s, `{"date":"2026-10-22","store":"Copag","shippingBRL":40}`)
	a := addItem(t, s, order.ID, "onepiece", `{"name":"Double Pack","kind":"sealed","qty":2,"buyBRL":100}`)
	b := addItem(t, s, order.ID, "pokemon", `{"name":"Box","kind":"sealed","qty":1,"buyBRL":100}`)
	c := addItem(t, s, order.ID, "onepiece", `{"name":"Sleeves","kind":"accessory","qty":1,"buyBRL":100}`)

	pkm := s.games["pokemon"].Trades
	if got := orderShipOf(t, op, a.ID); got != 20 {
		t.Fatalf("onepiece share = %v, want 20", got)
	}
	if got := orderShipOf(t, pkm, b.ID); got != 10 {
		t.Fatalf("pokemon share = %v, want 10", got)
	}
	if got := orderShipOf(t, acc, c.ID); got != 10 {
		t.Fatalf("accessory share = %v, want 10", got)
	}

	resp := purchasesOf(t, s)
	p := resp.Purchases[0]
	if p.ItemsCostBRL != 400 || p.TotalBRL != 440 {
		t.Fatalf("items = %v total = %v, want 400 / 440", p.ItemsCostBRL, p.TotalBRL)
	}
	for _, it := range p.Items {
		if it.ID == a.ID && it.CostBRL != 220 {
			t.Fatalf("lot cost = %v, want 220 with its freight share", it.CostBRL)
		}
	}

	updateOrder(t, s, order.ID, `{"date":"2026-10-22","store":"Copag","shippingBRL":80}`)
	if got := orderShipOf(t, op, a.ID); got != 40 {
		t.Fatalf("after freight change share = %v, want 40", got)
	}
}

func TestOrderShippingLeavesSoldLotsAndDiscountsThem(t *testing.T) {
	s, op, _ := purchaseServer(t)
	order := newOrder(t, s, `{"date":"2026-10-22","shippingBRL":30}`)
	a := addItem(t, s, order.ID, "onepiece", `{"name":"A","kind":"sealed","qty":1,"buyBRL":100}`)
	b := addItem(t, s, order.ID, "onepiece", `{"name":"B","kind":"sealed","qty":2,"buyBRL":100}`)

	if _, err := op.Sell(a.ID, trades.Sale{SellPrice: 150}); err != nil {
		t.Fatal(err)
	}
	updateOrder(t, s, order.ID, `{"date":"2026-10-22","shippingBRL":60}`)
	if got := orderShipOf(t, op, a.ID); got != 10 {
		t.Fatalf("sold share = %v, want untouched 10", got)
	}
	if got := orderShipOf(t, op, b.ID); got != 50 {
		t.Fatalf("holding share = %v, want 60 - 10 = 50", got)
	}
}

func TestOrderShippingFollowsAttachAndDetach(t *testing.T) {
	s, op, _ := purchaseServer(t)
	first := newOrder(t, s, `{"date":"2026-10-22","shippingBRL":20}`)
	second := newOrder(t, s, `{"date":"2026-10-23","shippingBRL":10}`)
	a := addItem(t, s, first.ID, "onepiece", `{"name":"A","kind":"sealed","qty":1,"buyBRL":100}`)
	b := addItem(t, s, first.ID, "onepiece", `{"name":"B","kind":"sealed","qty":1,"buyBRL":100}`)
	c := addItem(t, s, second.ID, "onepiece", `{"name":"C","kind":"sealed","qty":1,"buyBRL":100}`)

	req := httptest.NewRequest(http.MethodPost, "/api/purchases/"+second.ID+"/attach",
		strings.NewReader(`{"items":[{"id":"`+b.ID+`","game":"onepiece"}]}`))
	req.SetPathValue("id", second.ID)
	s.handlePurchaseAttach(httptest.NewRecorder(), req)
	if got := orderShipOf(t, op, a.ID); got != 20 {
		t.Fatalf("origin share = %v, want 20", got)
	}
	if got := orderShipOf(t, op, b.ID) + orderShipOf(t, op, c.ID); got != 10 {
		t.Fatalf("target shares = %v, want 10", got)
	}

	rec := httptest.NewRecorder()
	s.handlePurchaseDetach(rec, httptest.NewRequest(http.MethodPost, "/api/purchases/detach",
		strings.NewReader(`{"items":[{"id":"`+b.ID+`","game":"onepiece"}]}`)))
	if got := orderShipOf(t, op, b.ID); got != 0 {
		t.Fatalf("detached share = %v, want 0", got)
	}
	if got := orderShipOf(t, op, c.ID); got != 10 {
		t.Fatalf("target after detach = %v, want 10", got)
	}
}

func TestOrderShippingRecalculatesOnUnsellAndDelete(t *testing.T) {
	s, op, _ := purchaseServer(t)
	order := newOrder(t, s, `{"date":"2026-10-22","shippingBRL":30}`)
	a := addItem(t, s, order.ID, "onepiece", `{"name":"A","kind":"sealed","qty":2,"buyBRL":100}`)
	b := addItem(t, s, order.ID, "onepiece", `{"name":"B","kind":"sealed","qty":1,"buyBRL":100}`)

	sold, err := op.Sell(a.ID, trades.Sale{Qty: 1, SellPrice: 150})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/trades/"+sold.ID+"/unsell?game=onepiece", nil)
	req.SetPathValue("id", sold.ID)
	rec := httptest.NewRecorder()
	s.handleTradesUnsell(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("unsell = %d: %s", rec.Code, rec.Body.String())
	}
	if got := orderShipOf(t, op, a.ID); got != 20 {
		t.Fatalf("after unsell share = %v, want 20", got)
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/trades/"+b.ID+"?game=onepiece", nil)
	req.SetPathValue("id", b.ID)
	s.handleTradesDelete(httptest.NewRecorder(), req)
	if got := orderShipOf(t, op, a.ID); got != 30 {
		t.Fatalf("after delete share = %v, want 30", got)
	}
}

func TestOrderShippingRecalculatesWhenQtyChanges(t *testing.T) {
	s, op, _ := purchaseServer(t)
	order := newOrder(t, s, `{"date":"2026-10-22","shippingBRL":30}`)
	a := addItem(t, s, order.ID, "onepiece", `{"name":"A","kind":"sealed","qty":1,"buyBRL":100}`)
	b := addItem(t, s, order.ID, "onepiece", `{"name":"B","kind":"sealed","qty":1,"buyBRL":100}`)

	req := httptest.NewRequest(http.MethodPut, "/api/trades/"+a.ID+"?game=onepiece", strings.NewReader(`{"qty":2,"orderShippingBRL":999}`))
	req.SetPathValue("id", a.ID)
	s.handleTradesUpdate(httptest.NewRecorder(), req)
	if got := orderShipOf(t, op, a.ID); got != 20 {
		t.Fatalf("share = %v, want 20", got)
	}
	if got := orderShipOf(t, op, b.ID); got != 10 {
		t.Fatalf("share = %v, want 10", got)
	}
}

func TestReallocateAllOrders(t *testing.T) {
	s, op, _ := purchaseServer(t)
	order := newOrder(t, s, `{"date":"2026-10-22","shippingBRL":30}`)
	a := addItem(t, s, order.ID, "onepiece", `{"name":"A","kind":"sealed","qty":1,"buyBRL":100}`)
	if _, err := op.Update(a.ID, func(tr *trades.Trade) { tr.OrderShippingBRL = 0 }); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	s.handlePurchasesReallocate(rec, httptest.NewRequest(http.MethodPost, "/api/purchases/reallocate", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("reallocate = %d: %s", rec.Code, rec.Body.String())
	}
	if got := orderShipOf(t, op, a.ID); got != 30 {
		t.Fatalf("share = %v, want 30", got)
	}
}
