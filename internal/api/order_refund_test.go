package api

import (
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"opdeals/internal/mercadopago"
	"opdeals/internal/orders"
	"opdeals/internal/syncqueue"
	"opdeals/internal/trades"
)

var adminHeaders = map[string]string{"X-Admin-Token": "admin"}

func (f *checkoutFixture) paidOrder(t *testing.T, paymentID int64) orders.Order {
	t.Helper()
	o := f.createOrder(t)
	f.gw.payments[paymentID] = mercadopago.Payment{ID: paymentID, Status: "approved", ExternalReference: o.ID, AmountBRL: 512, Method: "pix"}
	if rec := f.webhook(t, strconv.FormatInt(paymentID, 10), testWebhookSecret); rec.Code != http.StatusOK {
		t.Fatalf("webhook: %d %s", rec.Code, rec.Body.String())
	}
	got, err := f.s.Checkout.Orders.Get(o.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != orders.StatusPaid || len(got.Lines[0].TradeIDs) != 2 {
		t.Fatalf("paid order = %+v", got)
	}
	return got
}

func (f *checkoutFixture) holdingUnits(t *testing.T) (int, int) {
	t.Helper()
	all, err := f.store.List()
	if err != nil {
		t.Fatal(err)
	}
	units, lots := 0, 0
	for _, tr := range all {
		if tr.Status == "holding" {
			units += tr.Qty
			lots++
		}
	}
	return units, lots
}

func TestRefundReturnsStockToOriginLots(t *testing.T) {
	f := checkoutServer(t)
	o := f.paidOrder(t, 10)
	rec := f.do(t, http.MethodPatch, "/api/orders/"+o.ID, map[string]string{"status": "refunded"}, adminHeaders)
	if rec.Code != http.StatusOK {
		t.Fatalf("refund: %d %s", rec.Code, rec.Body.String())
	}
	units, lots := f.holdingUnits(t)
	if units != 3 || lots != 1 {
		t.Fatalf("holding after refund = %d units in %d lots, want 3 merged into 1", units, lots)
	}
	all, _ := f.store.List()
	if cost := all[0].BuyBRL*float64(all[0].Qty) + all[0].ShippingBRL; cost < 289.99 || cost > 290.01 {
		t.Fatalf("merged cost = %.2f, want 290", cost)
	}
	if sold := f.soldTrades(t); len(sold) != 0 {
		t.Fatalf("sold after refund = %+v", sold)
	}
	got, _ := f.s.Checkout.Orders.Get(o.ID)
	if got.Status != orders.StatusRefunded || got.StockReturnedAt.IsZero() {
		t.Fatalf("order = %+v", got)
	}
	if len(got.Lines[0].TradeIDs) != 0 || len(got.Lines[0].ReturnedTradeIDs) != 2 {
		t.Fatalf("line ids = %+v", got.Lines[0])
	}
}

func TestRefundIsIdempotent(t *testing.T) {
	f := checkoutServer(t)
	o := f.paidOrder(t, 11)
	for i := 0; i < 2; i++ {
		if rec := f.do(t, http.MethodPatch, "/api/orders/"+o.ID, map[string]string{"status": "refunded"}, adminHeaders); rec.Code != http.StatusOK {
			t.Fatalf("refund %d: %d %s", i, rec.Code, rec.Body.String())
		}
	}
	f.gw.payments[11] = mercadopago.Payment{ID: 11, Status: "refunded", ExternalReference: o.ID, AmountBRL: 512}
	f.webhook(t, "11", testWebhookSecret)
	if units, _ := f.holdingUnits(t); units != 3 {
		t.Fatalf("holding units = %d, want 3", units)
	}
}

func TestWebhookRefundReturnsStock(t *testing.T) {
	f := checkoutServer(t)
	o := f.paidOrder(t, 12)
	f.gw.payments[12] = mercadopago.Payment{ID: 12, Status: "charged_back", ExternalReference: o.ID, AmountBRL: 512}
	if rec := f.webhook(t, "12", testWebhookSecret); rec.Code != http.StatusOK {
		t.Fatalf("webhook: %d", rec.Code)
	}
	got, _ := f.s.Checkout.Orders.Get(o.ID)
	if got.Status != orders.StatusRefunded || got.StockReturnedAt.IsZero() {
		t.Fatalf("order = %+v", got)
	}
	if units, _ := f.holdingUnits(t); units != 3 {
		t.Fatalf("holding units = %d, want 3", units)
	}
}

func TestCancelOversoldReturnsPartialStock(t *testing.T) {
	f := checkoutServer(t)
	o := f.paidOrder(t, 13)
	if _, err := f.s.Checkout.Orders.Update(o.ID, func(x *orders.Order) {
		x.Status = orders.StatusOversold
	}); err != nil {
		t.Fatal(err)
	}
	if rec := f.do(t, http.MethodPatch, "/api/orders/"+o.ID, map[string]string{"status": "cancelled"}, adminHeaders); rec.Code != http.StatusOK {
		t.Fatalf("cancel: %d %s", rec.Code, rec.Body.String())
	}
	if units, _ := f.holdingUnits(t); units != 3 {
		t.Fatalf("holding units = %d, want 3", units)
	}
	got, _ := f.s.Checkout.Orders.Get(o.ID)
	if got.Status != orders.StatusCancelled || got.StockReturnedAt.IsZero() {
		t.Fatalf("order = %+v", got)
	}
}

func TestRefundOfLigaListedItemQueuesRestore(t *testing.T) {
	f := checkoutServer(t)
	f.s.SyncQueue = syncqueue.NewStore(filepath.Join(t.TempDir(), "syncqueue.json"))
	if _, err := f.store.SetLigaState(map[string]trades.LigaState{
		f.older.ID: {Listed: true, Qty: 1},
		f.newer.ID: {Listed: true, Qty: 2},
	}); err != nil {
		t.Fatal(err)
	}
	o := f.paidOrder(t, 14)
	pending, _ := f.s.SyncQueue.List(syncqueue.StatusPending)
	for _, task := range pending {
		if _, err := f.s.SyncQueue.SetStatus(task.ID, syncqueue.StatusDone, ""); err != nil {
			t.Fatal(err)
		}
	}
	if rec := f.do(t, http.MethodPatch, "/api/orders/"+o.ID, map[string]string{"status": "refunded"}, adminHeaders); rec.Code != http.StatusOK {
		t.Fatalf("refund: %d %s", rec.Code, rec.Body.String())
	}
	pending, _ = f.s.SyncQueue.List(syncqueue.StatusPending)
	restored := 0
	for _, task := range pending {
		if task.Action == syncqueue.ActionRestore {
			restored += task.Qty
		}
	}
	if restored != 2 {
		t.Fatalf("restore tasks = %+v", pending)
	}
}

func TestOversoldToPaidSellsWhenStockExists(t *testing.T) {
	f := checkoutServer(t)
	o := f.createOrder(t)
	if _, err := f.s.Checkout.Orders.Update(o.ID, func(x *orders.Order) {
		x.Status = orders.StatusOversold
	}); err != nil {
		t.Fatal(err)
	}
	if rec := f.do(t, http.MethodPatch, "/api/orders/"+o.ID, map[string]string{"status": "paid"}, adminHeaders); rec.Code != http.StatusOK {
		t.Fatalf("paid: %d %s", rec.Code, rec.Body.String())
	}
	got, _ := f.s.Checkout.Orders.Get(o.ID)
	if got.Status != orders.StatusPaid || len(got.Lines[0].TradeIDs) != 2 {
		t.Fatalf("order = %+v", got)
	}
	sold := f.soldTrades(t)
	if len(sold) != 2 || trades.ChannelOf(sold[0]) != trades.ChannelSite {
		t.Fatalf("sold = %+v", sold)
	}
	if rec := f.do(t, http.MethodPatch, "/api/orders/"+o.ID, map[string]string{"status": "shipped"}, adminHeaders); rec.Code != http.StatusOK {
		t.Fatalf("shipped: %d %s", rec.Code, rec.Body.String())
	}
	if sold := f.soldTrades(t); len(sold) != 2 {
		t.Fatalf("shipping must not sell again: %+v", sold)
	}
}

func TestOversoldToPaidWithoutStockConflicts(t *testing.T) {
	f := checkoutServer(t)
	o := f.createOrder(t)
	if _, err := f.store.Sell(f.newer.ID, trades.Sale{Qty: 2, SellPrice: 400, SellCurrency: "BRL", Buyer: "balcão"}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.Checkout.Orders.Update(o.ID, func(x *orders.Order) {
		x.Status = orders.StatusOversold
	}); err != nil {
		t.Fatal(err)
	}
	rec := f.do(t, http.MethodPatch, "/api/orders/"+o.ID, map[string]string{"status": "paid"}, adminHeaders)
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "sem estoque") {
		t.Fatalf("paid without stock: %d %s", rec.Code, rec.Body.String())
	}
	got, _ := f.s.Checkout.Orders.Get(o.ID)
	if got.Status != orders.StatusOversold {
		t.Fatalf("status = %s, want oversold kept", got.Status)
	}
	if sold := f.soldTrades(t); len(sold) != 1 {
		t.Fatalf("sold = %+v", sold)
	}
}

func TestCancelPendingOrderLeavesStockAlone(t *testing.T) {
	f := checkoutServer(t)
	o := f.createOrder(t)
	if rec := f.do(t, http.MethodPatch, "/api/orders/"+o.ID, map[string]string{"status": "cancelled"}, adminHeaders); rec.Code != http.StatusOK {
		t.Fatalf("cancel: %d %s", rec.Code, rec.Body.String())
	}
	got, _ := f.s.Checkout.Orders.Get(o.ID)
	if got.Status != orders.StatusCancelled || !got.StockReturnedAt.IsZero() {
		t.Fatalf("order = %+v", got)
	}
	if units, lots := f.holdingUnits(t); units != 3 || lots != 2 {
		t.Fatalf("holding = %d in %d lots", units, lots)
	}
}
