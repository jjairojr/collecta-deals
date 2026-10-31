package api

import (
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"opdeals/internal/logx"
	"opdeals/internal/mercadopago"
	"opdeals/internal/orders"
	"opdeals/internal/trades"
)

func TestOversoldRetrySellsOnlyTheMissingUnits(t *testing.T) {
	f := checkoutServer(t)
	o := f.createOrder(t)
	first, err := f.store.Sell(f.older.ID, trades.Sale{Qty: 1, SellPrice: 256, SellCurrency: "BRL", Buyer: "Fulano — Site #1", Channel: trades.ChannelSite})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.Checkout.Orders.Update(o.ID, func(x *orders.Order) {
		x.Status = orders.StatusOversold
		x.Lines[0].TradeIDs = []string{first.ID}
	}); err != nil {
		t.Fatal(err)
	}
	if rec := f.do(t, http.MethodPatch, "/api/orders/"+o.ID, map[string]string{"status": "paid"}, adminHeaders); rec.Code != http.StatusOK {
		t.Fatalf("paid: %d %s", rec.Code, rec.Body.String())
	}
	got, _ := f.s.Checkout.Orders.Get(o.ID)
	if got.Status != orders.StatusPaid || len(got.Lines[0].TradeIDs) != 2 || got.Lines[0].TradeIDs[0] != first.ID {
		t.Fatalf("order = %+v", got)
	}
	units := 0
	for _, s := range f.soldTrades(t) {
		units += s.Qty
	}
	if units != 2 {
		t.Fatalf("sold units = %d, want 2", units)
	}
}

func TestMyPSaleQueuesNothing(t *testing.T) {
	s, _ := queueServer(t)
	item := createTrade(t, s, "onepiece", ligaItem)
	if rec := sellTrade(t, s, item.ID, `{"qty":1,"sellPrice":150,"buyer":"nick — MyP #9","channel":"myp"}`); rec.Code != http.StatusOK {
		t.Fatalf("sell = %d", rec.Code)
	}
	if got := pendingTasks(t, s); len(got) != 0 {
		t.Fatalf("tasks = %+v, want none", got)
	}
}

func TestRefundAfterShippingKeepsStockUntilReturned(t *testing.T) {
	f := checkoutServer(t)
	o := f.paidOrder(t, 20)
	if rec := f.do(t, http.MethodPatch, "/api/orders/"+o.ID, map[string]string{"status": "shipped"}, adminHeaders); rec.Code != http.StatusOK {
		t.Fatalf("shipped: %d %s", rec.Code, rec.Body.String())
	}
	f.gw.payments[20] = mercadopago.Payment{ID: 20, Status: "charged_back", ExternalReference: o.ID, AmountBRL: 512}
	f.webhook(t, "20", testWebhookSecret)
	got, _ := f.s.Checkout.Orders.Get(o.ID)
	if got.Status != orders.StatusRefunded || !got.StockReturnedAt.IsZero() || !strings.Contains(got.Notes, "estoque não devolvido") {
		t.Fatalf("order = %+v", got)
	}
	if units, _ := f.holdingUnits(t); units != 1 {
		t.Fatalf("holding units = %d, want 1", units)
	}
	if rec := f.do(t, http.MethodPost, "/api/orders/"+o.ID+"/return-stock", nil, adminHeaders); rec.Code != http.StatusOK {
		t.Fatalf("return-stock: %d %s", rec.Code, rec.Body.String())
	}
	if units, _ := f.holdingUnits(t); units != 3 {
		t.Fatalf("holding units after return = %d, want 3", units)
	}
	if rec := f.do(t, http.MethodPost, "/api/orders/"+o.ID+"/return-stock", nil, adminHeaders); rec.Code != http.StatusConflict {
		t.Fatalf("second return-stock = %d, want 409", rec.Code)
	}
}

func TestManualRefundOfDeliveredOrderKeepsStock(t *testing.T) {
	f := checkoutServer(t)
	o := f.paidOrder(t, 21)
	f.do(t, http.MethodPatch, "/api/orders/"+o.ID, map[string]string{"status": "delivered"}, adminHeaders)
	if rec := f.do(t, http.MethodPatch, "/api/orders/"+o.ID, map[string]string{"status": "refunded"}, adminHeaders); rec.Code != http.StatusOK {
		t.Fatalf("refund: %d %s", rec.Code, rec.Body.String())
	}
	if units, _ := f.holdingUnits(t); units != 1 {
		t.Fatalf("holding units = %d, want 1", units)
	}
}

func TestEditingChannelKeepsLegacyWriteOff(t *testing.T) {
	s, _, _ := purchaseServer(t)
	item := createTrade(t, s, "onepiece", `{"name":"Booster OP16","set":"SEALED","kind":"sealed","qty":1,"buyBRL":20,"status":"sold","sellPrice":1,"buyer":"caixa aberta"}`)
	code, got := putTrade(t, s, item.ID, "onepiece", `{"channel":"balcao","sellDate":"2026-10-30"}`)
	if code != http.StatusOK {
		t.Fatalf("put = %d", code)
	}
	if got.WriteOff != trades.WriteOffAberto || got.SellPrice != 0 {
		t.Fatalf("after edit = %+v", got)
	}
}

func TestUpdateValidatesSaleFields(t *testing.T) {
	s, _, _ := purchaseServer(t)
	item := createTrade(t, s, "onepiece", `{"number":"OP16-080","name":"Teach","set":"OP16","qty":1,"buyBRL":100,"status":"holding"}`)
	sellTrade(t, s, item.ID, `{"sellPrice":300,"channel":"balcao"}`)
	for _, body := range []string{`{"channel":"shopee"}`, `{"writeOff":"sumiu"}`, `{"status":"holding"}`} {
		if code, _ := putTrade(t, s, item.ID, "onepiece", body); code != http.StatusBadRequest {
			t.Fatalf("put %s = %d, want 400", body, code)
		}
	}
	code, got := putTrade(t, s, item.ID, "onepiece", `{"writeOff":"perda","status":"sold"}`)
	if code != http.StatusOK || got.WriteOff != trades.WriteOffPerda || got.SellPrice != 0 || got.Status != "sold" {
		t.Fatalf("write-off put = %d %+v", code, got)
	}
}

func TestMoveRefusesALotOfADeliveredSiteOrder(t *testing.T) {
	s, _, _ := twoGameServer(t)
	s.Checkout.Orders = orders.NewStore(filepath.Join(t.TempDir(), "orders.json"), logx.New(io.Discard))
	item := createTrade(t, s, "onepiece", `{"name":"Teach","number":"OP16-080","qty":1,"buyBRL":10,"status":"holding"}`)
	o, err := s.Checkout.Orders.Create(func(map[string]int, []orders.Order) (orders.Order, error) {
		return orders.Order{Lines: []orders.Line{{SKU: "x", Game: "onepiece", Name: "Teach", Qty: 1, TradeIDs: []string{item.ID}}}}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Checkout.Orders.Update(o.ID, func(x *orders.Order) { x.Status = orders.StatusDelivered }); err != nil {
		t.Fatal(err)
	}
	if res := moveTrade(t, s, item.ID, `{"from":"onepiece","to":"pokemon"}`); res.StatusCode != http.StatusConflict {
		t.Fatalf("status = %d, want 409", res.StatusCode)
	}
}

func TestUnsellWholeLotDoesNotMergeIntoAnotherLot(t *testing.T) {
	s, op := queueServer(t)
	a := createTrade(t, s, "onepiece", `{"number":"OP16-080","name":"Teach","set":"OP16","qty":1,"buyBRL":50,"buyDate":"2026-08-01","status":"holding"}`)
	b := createTrade(t, s, "onepiece", `{"number":"OP16-080","name":"Teach","set":"OP16","qty":1,"buyBRL":80,"buyDate":"2026-10-01","status":"holding"}`)
	sellTrade(t, s, b.ID, `{"sellPrice":150,"channel":"balcao"}`)
	if rec := unsellTrade(t, s, b.ID); rec.Code != http.StatusOK {
		t.Fatalf("unsell = %d: %s", rec.Code, rec.Body.String())
	}
	got := ledgerTrades(t, op)
	if len(got) != 2 || got[a.ID].BuyBRL != 50 || got[a.ID].Qty != 1 || got[b.ID].BuyBRL != 80 || got[b.ID].Status != "holding" {
		t.Fatalf("ledger = %+v", got)
	}
}

func TestRepointingASoldLotDropsTheOldFreightShare(t *testing.T) {
	s, op, _ := purchaseServer(t)
	x := newOrder(t, s, `{"date":"2026-10-01","store":"X","shippingBRL":12}`)
	sold := addItem(t, s, x.ID, "onepiece", `{"number":"OP16-080","name":"Teach","set":"OP16","qty":1,"buyBRL":100}`)
	sellTrade(t, s, sold.ID, `{"sellPrice":150,"channel":"balcao"}`)
	y := newOrder(t, s, `{"date":"2026-10-02","store":"Y","shippingBRL":20}`)
	kept := addItem(t, s, y.ID, "onepiece", `{"number":"OP16-001","name":"Luffy","set":"OP16","qty":1,"buyBRL":50}`)
	if code, _ := putTrade(t, s, sold.ID, "onepiece", `{"purchaseID":"`+y.ID+`"}`); code != http.StatusOK {
		t.Fatalf("put = %d", code)
	}
	if got := orderShipOf(t, op, sold.ID); got != 0 {
		t.Fatalf("moved sold share = %v, want 0", got)
	}
	if got := orderShipOf(t, op, kept.ID); got != 20 {
		t.Fatalf("Y holding share = %v, want 20", got)
	}
}
