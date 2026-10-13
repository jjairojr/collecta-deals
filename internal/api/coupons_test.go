package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"opdeals/internal/collecta"
	"opdeals/internal/mercadopago"
	"opdeals/internal/orders"
	"opdeals/internal/trades"
)

type reserveCall struct {
	userID     string
	ids        []string
	totalCents int64
}

type fakeCoupons struct {
	mu         sync.Mutex
	reserves   []reserveCall
	released   [][]string
	events     []string
	reserveErr error
	releaseErr error
	value      int64
}

func (f *fakeCoupons) Reserve(_ context.Context, userID string, ids []string, totalCents int64) ([]collecta.Coupon, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.reserveErr != nil {
		return nil, f.reserveErr
	}
	f.reserves = append(f.reserves, reserveCall{userID: userID, ids: ids, totalCents: totalCents})
	f.events = append(f.events, "reserve")
	value := f.value
	if value == 0 {
		value = 1000
	}
	out := make([]collecta.Coupon, 0, len(ids))
	for _, id := range ids {
		out = append(out, collecta.Coupon{ID: id, Code: "IND-" + strings.ToUpper(id), ValueCents: value})
	}
	return out, nil
}

func (f *fakeCoupons) Release(_ context.Context, codes []string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.releaseErr != nil {
		return f.releaseErr
	}
	f.released = append(f.released, codes)
	f.events = append(f.events, "release")
	return nil
}

func (f *fakeCoupons) releasedCodes() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, 0)
	for _, codes := range f.released {
		out = append(out, codes...)
	}
	return out
}

func couponServer(t *testing.T) (*checkoutFixture, *fakeCoupons) {
	t.Helper()
	f := checkoutServer(t)
	fc := &fakeCoupons{}
	f.s.Checkout.Coupons = fc
	return f, fc
}

func couponBody(qty int, unitCents int64, ids ...string) map[string]interface{} {
	b := validOrderBody(qty, unitCents)
	b["collectaUserId"] = "u1"
	b["couponIds"] = ids
	return b
}

func (f *checkoutFixture) createCouponOrder(t *testing.T, ids ...string) orders.Order {
	t.Helper()
	rec := f.do(t, http.MethodPost, "/api/orders", couponBody(2, 25000, ids...), storefrontHeaders())
	if rec.Code != http.StatusOK {
		t.Fatalf("create with coupons: status %d body %s", rec.Code, rec.Body.String())
	}
	var out struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	o, err := f.s.Checkout.Orders.Get(out.ID)
	if err != nil {
		t.Fatal(err)
	}
	return o
}

func itemsSumCents(items []mercadopago.Item) int64 {
	var sum int64
	for _, it := range items {
		sum += int64(it.Qty) * cents(it.UnitBRL)
	}
	return sum
}

func TestOrderWithCouponsAppliesDiscount(t *testing.T) {
	f, fc := couponServer(t)
	o := f.createCouponOrder(t, "c1", "c2")
	if o.Totals.ItemsBRL != 500 || o.Totals.ShippingBRL != 12 || o.Totals.DiscountBRL != 20 || o.Totals.TotalBRL != 492 {
		t.Fatalf("totals = %+v", o.Totals)
	}
	if o.CollectaUserID != "u1" || len(o.Coupons) != 2 || o.Coupons[0].Code != "IND-C1" || o.Coupons[1].ValueBRL != 10 {
		t.Fatalf("order = %+v", o)
	}
	if len(fc.reserves) != 1 || fc.reserves[0].userID != "u1" || fc.reserves[0].totalCents != 50000 || len(fc.reserves[0].ids) != 2 {
		t.Fatalf("reserve calls = %+v", fc.reserves)
	}
	pref := f.gw.prefs[0]
	if len(pref.Items) != 2 || pref.Items[0].Qty != 2 || pref.Items[0].UnitBRL != 240 || pref.Items[1].ID != "frete" || pref.Items[1].UnitBRL != 12 {
		t.Fatalf("items = %+v", pref.Items)
	}
	if itemsSumCents(pref.Items) != cents(o.Totals.TotalBRL) {
		t.Fatalf("items sum %d != total %d", itemsSumCents(pref.Items), cents(o.Totals.TotalBRL))
	}
	rec := f.do(t, http.MethodGet, "/api/orders/"+o.ID, nil, nil)
	var pub publicOrder
	if err := json.Unmarshal(rec.Body.Bytes(), &pub); err != nil {
		t.Fatal(err)
	}
	if pub.Totals.DiscountBRL != 20 || len(pub.Coupons) != 2 || pub.Coupons[0].Code != "IND-C1" {
		t.Fatalf("public = %+v", pub)
	}
}

func TestCouponInputValidation(t *testing.T) {
	f, _ := couponServer(t)
	first := f.createOrder(t)
	body := validOrderBody(1, 25000)
	body["couponIds"] = []string{"c1"}
	rec := f.do(t, http.MethodPost, "/api/orders", body, storefrontHeaders())
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "login") {
		t.Fatalf("coupon without user: %d %s", rec.Code, rec.Body.String())
	}
	got, _ := f.s.Checkout.Orders.Get(first.ID)
	if got.Status != orders.StatusPending {
		t.Fatalf("validation failure must not cancel the pending order: %+v", got)
	}
	plain := checkoutServer(t)
	rec = plain.do(t, http.MethodPost, "/api/orders", couponBody(1, 25000, "c1"), storefrontHeaders())
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "coupon_invalid") {
		t.Fatalf("coupons disabled: %d %s", rec.Code, rec.Body.String())
	}
	if all, _ := plain.s.Checkout.Orders.List(); len(all) != 0 {
		t.Fatalf("disabled coupons must not persist orders: %+v", all)
	}
}

func TestCouponRejectedCancelsOrder(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		status int
		want   string
	}{
		{"not found", &collecta.APIError{Status: http.StatusNotFound, Message: "Cupom não encontrado"}, http.StatusConflict, "Cupom não encontrado"},
		{"unauthorized", &collecta.APIError{Status: http.StatusUnauthorized, Message: "unauthorized"}, http.StatusBadGateway, "indisponíveis"},
		{"network", errors.New("dial tcp: timeout"), http.StatusBadGateway, "indisponíveis"},
	}
	for _, c := range cases {
		f, fc := couponServer(t)
		fc.reserveErr = c.err
		rec := f.do(t, http.MethodPost, "/api/orders", couponBody(1, 25000, "c1"), storefrontHeaders())
		if rec.Code != c.status || !strings.Contains(rec.Body.String(), c.want) {
			t.Fatalf("%s: status %d body %s", c.name, rec.Code, rec.Body.String())
		}
		all, _ := f.s.Checkout.Orders.List()
		if len(all) != 1 || all[0].Status != orders.StatusCancelled || !strings.Contains(all[0].Notes, "cupom recusado") || len(all[0].Coupons) != 0 {
			t.Fatalf("%s: orders = %+v", c.name, all)
		}
		if len(f.gw.prefs) != 0 {
			t.Fatalf("%s: preference must not be created", c.name)
		}
	}
}

func TestCouponGatewayFailureReleases(t *testing.T) {
	f, fc := couponServer(t)
	f.gw.fail = true
	rec := f.do(t, http.MethodPost, "/api/orders", couponBody(1, 25000, "c1"), storefrontHeaders())
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status %d", rec.Code)
	}
	all, _ := f.s.Checkout.Orders.List()
	if len(all) != 1 || all[0].Status != orders.StatusCancelled || all[0].CouponsReleasedAt.IsZero() {
		t.Fatalf("orders = %+v", all)
	}
	if got := fc.releasedCodes(); len(got) != 1 || got[0] != "IND-C1" {
		t.Fatalf("released = %v", got)
	}
}

func TestRetryReleasesPreviousCouponsFirst(t *testing.T) {
	f, fc := couponServer(t)
	first := f.createCouponOrder(t, "c1")
	second := f.createCouponOrder(t, "c1")
	if strings.Join(fc.events, ",") != "reserve,release,reserve" {
		t.Fatalf("events = %v", fc.events)
	}
	got, _ := f.s.Checkout.Orders.Get(first.ID)
	if got.Status != orders.StatusCancelled || got.CouponsReleasedAt.IsZero() {
		t.Fatalf("first = %+v", got)
	}
	if second.Totals.TotalBRL != 502 || len(second.Coupons) != 1 {
		t.Fatalf("second = %+v", second.Totals)
	}
}

func TestRetryAfterRejectedPaymentReleasesCoupons(t *testing.T) {
	f, fc := couponServer(t)
	first := f.createCouponOrder(t, "c1")
	f.gw.payments[2] = mercadopago.Payment{ID: 2, Status: "rejected", ExternalReference: first.ID, AmountBRL: 502, Method: "master"}
	f.webhook(t, "2", testWebhookSecret)
	if got, _ := f.s.Checkout.Orders.Get(first.ID); got.Status != orders.StatusFailed || len(fc.releasedCodes()) != 0 {
		t.Fatalf("rejected must keep coupons while payable: %+v released %v", got, fc.releasedCodes())
	}
	f.createCouponOrder(t, "c1")
	got, _ := f.s.Checkout.Orders.Get(first.ID)
	if got.Status != orders.StatusCancelled || got.CouponsReleasedAt.IsZero() || len(fc.releasedCodes()) != 1 {
		t.Fatalf("retry must cancel the failed order and release: %+v released %v", got, fc.releasedCodes())
	}
}

func TestWebhookDiscountedAmountSettles(t *testing.T) {
	f, fc := couponServer(t)
	o := f.createCouponOrder(t, "c1")
	f.gw.payments[5] = mercadopago.Payment{ID: 5, Status: "approved", ExternalReference: o.ID, AmountBRL: 512}
	f.webhook(t, "5", testWebhookSecret)
	got, _ := f.s.Checkout.Orders.Get(o.ID)
	if got.Status != orders.StatusFailed || !strings.Contains(got.Notes, "difere") {
		t.Fatalf("full amount must not settle a discounted order: %+v", got)
	}
	if len(fc.releasedCodes()) != 0 {
		t.Fatalf("failed before expiry must keep coupons: %v", fc.releasedCodes())
	}
	f.gw.payments[6] = mercadopago.Payment{ID: 6, Status: "approved", ExternalReference: o.ID, AmountBRL: 502, Method: "pix"}
	f.webhook(t, "6", testWebhookSecret)
	got, _ = f.s.Checkout.Orders.Get(o.ID)
	if got.Status != orders.StatusPaid || len(got.Coupons) != 1 || strings.Contains(got.Notes, "liberados") {
		t.Fatalf("order = %+v", got)
	}
	if sold := f.soldTrades(t); len(sold) != 2 || sold[0].SellPrice != 250 {
		t.Fatalf("ledger keeps the gross unit price: %+v", sold)
	}
}

func TestOversoldReleasesCoupons(t *testing.T) {
	f, fc := couponServer(t)
	o := f.createCouponOrder(t, "c1")
	if _, err := f.store.Sell(f.newer.ID, trades.Sale{Qty: 2, SellPrice: 400, SellCurrency: "BRL", Buyer: "balcão"}); err != nil {
		t.Fatal(err)
	}
	f.gw.payments[9] = mercadopago.Payment{ID: 9, Status: "approved", ExternalReference: o.ID, AmountBRL: 502}
	f.webhook(t, "9", testWebhookSecret)
	got, _ := f.s.Checkout.Orders.Get(o.ID)
	if got.Status != orders.StatusOversold || got.CouponsReleasedAt.IsZero() || len(fc.releasedCodes()) != 1 {
		t.Fatalf("order = %+v released %v", got, fc.releasedCodes())
	}
}

func TestExpiredReleasesAfterGraceAndLatePaymentNotes(t *testing.T) {
	f, fc := couponServer(t)
	o := f.createCouponOrder(t, "c1")
	f.s.Checkout.Orders.Update(o.ID, func(cur *orders.Order) { cur.ExpiresAt = time.Now().Add(-time.Minute) })
	f.s.sweepCoupons(context.Background())
	if len(fc.releasedCodes()) != 0 {
		t.Fatalf("expired inside the grace window must not release: %v", fc.releasedCodes())
	}
	f.s.Checkout.Orders.Update(o.ID, func(cur *orders.Order) { cur.ExpiresAt = time.Now().Add(-couponReleaseGrace - time.Minute) })
	f.s.sweepCoupons(context.Background())
	got, _ := f.s.Checkout.Orders.Get(o.ID)
	if got.CouponsReleasedAt.IsZero() || len(fc.releasedCodes()) != 1 {
		t.Fatalf("expired past grace must release: %+v released %v", got, fc.releasedCodes())
	}
	f.s.sweepCoupons(context.Background())
	if len(fc.releasedCodes()) != 1 {
		t.Fatalf("release must be idempotent: %v", fc.releasedCodes())
	}
	f.gw.payments[7] = mercadopago.Payment{ID: 7, Status: "approved", ExternalReference: o.ID, AmountBRL: 502, Method: "pix"}
	f.webhook(t, "7", testWebhookSecret)
	got, _ = f.s.Checkout.Orders.Get(o.ID)
	if got.Status != orders.StatusPaid || !strings.Contains(got.Notes, "cupons liberados antes do pagamento") {
		t.Fatalf("late payment = %+v", got)
	}
}

func TestSweepRetriesFailedRelease(t *testing.T) {
	f, fc := couponServer(t)
	o := f.createCouponOrder(t, "c1")
	fc.releaseErr = errors.New("api down")
	admin := map[string]string{"X-Admin-Token": "admin"}
	if rec := f.do(t, http.MethodPatch, "/api/orders/"+o.ID, map[string]string{"status": "cancelled"}, admin); rec.Code != http.StatusOK {
		t.Fatalf("patch: %d %s", rec.Code, rec.Body.String())
	}
	got, _ := f.s.Checkout.Orders.Get(o.ID)
	if got.Status != orders.StatusCancelled || !got.CouponsReleasedAt.IsZero() {
		t.Fatalf("failed release must stay pending: %+v", got)
	}
	fc.releaseErr = nil
	f.s.sweepCoupons(context.Background())
	got, _ = f.s.Checkout.Orders.Get(o.ID)
	if got.CouponsReleasedAt.IsZero() || len(fc.releasedCodes()) != 1 {
		t.Fatalf("sweep must retry: %+v released %v", got, fc.releasedCodes())
	}
}

func TestReleaseSkipsCodesHeldByAnotherOrder(t *testing.T) {
	f, fc := couponServer(t)
	first := f.createCouponOrder(t, "c1")
	second := f.createCouponOrder(t, "c1")
	fc.mu.Lock()
	fc.released = nil
	fc.mu.Unlock()
	f.s.Checkout.Orders.Update(first.ID, func(cur *orders.Order) { cur.CouponsReleasedAt = time.Time{} })
	f.s.sweepCoupons(context.Background())
	got, _ := f.s.Checkout.Orders.Get(first.ID)
	if got.CouponsReleasedAt.IsZero() || len(fc.releasedCodes()) != 0 {
		t.Fatalf("code held by pending order %s must not be released: %+v released %v", second.ID, got, fc.releasedCodes())
	}
}

func TestDiscountedItemsSplitExactly(t *testing.T) {
	line := func(sku string, qty int, unit float64) orders.Line {
		return orders.Line{SKU: sku, Name: sku, Qty: qty, UnitBRL: unit}
	}
	cases := []struct {
		name     string
		lines    []orders.Line
		discount int64
		want     []mercadopago.Item
	}{
		{"even", []orders.Line{line("a", 2, 250)}, 2000, []mercadopago.Item{{ID: "a", Title: "a", Qty: 2, UnitBRL: 240}}},
		{"remainder", []orders.Line{line("a", 3, 10)}, 1000, []mercadopago.Item{{ID: "a", Title: "a", Qty: 1, UnitBRL: 6.66}, {ID: "a", Title: "a", Qty: 2, UnitBRL: 6.67}}},
		{"proportional", []orders.Line{line("a", 1, 100), line("b", 1, 50)}, 1000, []mercadopago.Item{{ID: "a", Title: "a", Qty: 1, UnitBRL: 93.33}, {ID: "b", Title: "b", Qty: 1, UnitBRL: 46.67}}},
		{"unit zero", []orders.Line{line("a", 5, 0.02)}, 7, []mercadopago.Item{{ID: "a", Title: "a", Qty: 1, UnitBRL: 0.03}}},
		{"no discount", []orders.Line{line("a", 2, 250)}, 0, []mercadopago.Item{{ID: "a", Title: "a", Qty: 2, UnitBRL: 250}}},
	}
	for _, c := range cases {
		got := discountedItems(c.lines, c.discount)
		if len(got) != len(c.want) {
			t.Fatalf("%s: items = %+v", c.name, got)
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Fatalf("%s: item %d = %+v want %+v", c.name, i, got[i], c.want[i])
			}
		}
		if itemsSumCents(got) != itemsCents(c.lines)-c.discount {
			t.Fatalf("%s: sum %d want %d", c.name, itemsSumCents(got), itemsCents(c.lines)-c.discount)
		}
	}
}

func TestDiscountCoveringOrderIsRejected(t *testing.T) {
	f, fc := couponServer(t)
	fc.value = 25000
	rec := f.do(t, http.MethodPost, "/api/orders", couponBody(1, 25000, "c1"), storefrontHeaders())
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "inteiro") {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	all, _ := f.s.Checkout.Orders.List()
	if len(all) != 1 || all[0].Status != orders.StatusCancelled || all[0].CouponsReleasedAt.IsZero() || len(fc.releasedCodes()) != 1 {
		t.Fatalf("orders = %+v released %d", all, len(fc.releasedCodes()))
	}
	if len(f.gw.prefs) != 0 {
		t.Fatal("preference must not be created")
	}
}

func TestUserOrdersList(t *testing.T) {
	f, _ := couponServer(t)
	mine := f.createCouponOrder(t, "c1")
	other := validOrderBody(1, 25000)
	other["buyer"] = map[string]string{"name": "Bruno Lima", "email": "bruno@example.com", "phone": "11988887777", "cpf": "390.533.447-05"}
	if rec := f.do(t, http.MethodPost, "/api/orders", other, storefrontHeaders()); rec.Code != http.StatusOK {
		t.Fatalf("other buyer: %d %s", rec.Code, rec.Body.String())
	}
	if rec := f.do(t, http.MethodGet, "/api/orders?collectaUserId=u1", nil, nil); rec.Code != http.StatusUnauthorized {
		t.Fatalf("no token: %d", rec.Code)
	}
	if rec := f.do(t, http.MethodGet, "/api/orders?collectaUserId=u1", nil, map[string]string{"X-Admin-Token": "admin"}); rec.Code != http.StatusUnauthorized {
		t.Fatalf("admin token alone: %d", rec.Code)
	}
	rec := f.do(t, http.MethodGet, "/api/orders?collectaUserId=u1", nil, storefrontHeaders())
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d %s", rec.Code, rec.Body.String())
	}
	var list []publicOrder
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ID != mine.ID || list[0].Totals.DiscountBRL != 10 || len(list[0].Coupons) != 1 || list[0].FirstName != "Ana" {
		t.Fatalf("list = %+v", list)
	}
	if body := rec.Body.String(); strings.Contains(body, "52998224725") || strings.Contains(body, "ana@example.com") {
		t.Fatalf("user list leaks PII: %s", body)
	}
	rec = f.do(t, http.MethodGet, "/api/orders?collectaUserId=nobody", nil, storefrontHeaders())
	if strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Fatalf("unknown user = %s", rec.Body.String())
	}
}
