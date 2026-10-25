package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"opdeals/internal/orders"
)

func promoBody(qty int, unitCents int64, code string) map[string]interface{} {
	b := validOrderBody(qty, unitCents)
	b["collectaUserId"] = "u1"
	b["promoCode"] = code
	return b
}

func (f *checkoutFixture) createPromoOrder(t *testing.T, body map[string]interface{}) orders.Order {
	t.Helper()
	rec := f.do(t, http.MethodPost, "/api/orders", body, storefrontHeaders())
	if rec.Code != http.StatusOK {
		t.Fatalf("create with promo: status %d body %s", rec.Code, rec.Body.String())
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

func (f *checkoutFixture) markPaid(t *testing.T, id string) {
	t.Helper()
	if _, err := f.s.Checkout.Orders.Update(id, func(o *orders.Order) { o.Status = orders.StatusPaid }); err != nil {
		t.Fatal(err)
	}
}

func TestPromoAppliesPercentOnItems(t *testing.T) {
	f := checkoutServer(t)
	o := f.createPromoOrder(t, promoBody(2, 25000, " bemvindo "))
	if o.Promo == nil || o.Promo.Code != "BEMVINDO" || o.Promo.PercentBps != 500 || o.Promo.ValueBRL != 25 {
		t.Fatalf("promo = %+v", o.Promo)
	}
	if o.Totals.ItemsBRL != 500 || o.Totals.DiscountBRL != 25 || o.Totals.TotalBRL != 487 {
		t.Fatalf("totals = %+v", o.Totals)
	}
	if len(o.Coupons) != 0 {
		t.Fatalf("promo must not be stored as app coupon: %+v", o.Coupons)
	}
	pref := f.gw.prefs[0]
	if itemsSumCents(pref.Items) != cents(o.Totals.TotalBRL) {
		t.Fatalf("items sum %d != total %d", itemsSumCents(pref.Items), cents(o.Totals.TotalBRL))
	}
	rec := f.do(t, http.MethodGet, "/api/orders/"+o.ID, nil, nil)
	var pub publicOrder
	if err := json.Unmarshal(rec.Body.Bytes(), &pub); err != nil {
		t.Fatal(err)
	}
	if len(pub.Coupons) != 1 || pub.Coupons[0].Code != "BEMVINDO" || pub.Coupons[0].ValueBRL != 25 {
		t.Fatalf("public coupons = %+v", pub.Coupons)
	}
}

func TestPromoInputValidation(t *testing.T) {
	f, _ := couponServer(t)
	body := validOrderBody(1, 25000)
	body["promoCode"] = "BEMVINDO"
	rec := f.do(t, http.MethodPost, "/api/orders", body, storefrontHeaders())
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "login") {
		t.Fatalf("promo without user: %d %s", rec.Code, rec.Body.String())
	}
	stacked := promoBody(1, 25000, "BEMVINDO")
	stacked["couponIds"] = []string{"c1"}
	rec = f.do(t, http.MethodPost, "/api/orders", stacked, storefrontHeaders())
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), promoNoStacking) {
		t.Fatalf("stacked: %d %s", rec.Code, rec.Body.String())
	}
	rec = f.do(t, http.MethodPost, "/api/orders", promoBody(1, 25000, "NADA"), storefrontHeaders())
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), promoUnknown) {
		t.Fatalf("unknown: %d %s", rec.Code, rec.Body.String())
	}
	if all, _ := f.s.Checkout.Orders.List(); len(all) != 0 {
		t.Fatalf("rejected promo must not persist orders: %+v", all)
	}
}

func TestPromoFirstOrderOnly(t *testing.T) {
	f := checkoutServer(t)
	first := f.createPromoOrder(t, promoBody(1, 25000, "BEMVINDO"))
	f.markPaid(t, first.ID)

	rec := f.do(t, http.MethodPost, "/api/orders", promoBody(1, 25000, "BEMVINDO"), storefrontHeaders())
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), promoNotFirst) {
		t.Fatalf("second order same account: %d %s", rec.Code, rec.Body.String())
	}

	other := promoBody(1, 25000, "BEMVINDO")
	other["collectaUserId"] = "u2"
	rec = f.do(t, http.MethodPost, "/api/orders", other, storefrontHeaders())
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), promoNotFirst) {
		t.Fatalf("new account same CPF: %d %s", rec.Code, rec.Body.String())
	}

	rec = f.do(t, http.MethodPost, "/api/promos/check", map[string]string{"code": "BEMVINDO", "collectaUserId": "u1"}, storefrontHeaders())
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), promoNotFirst) {
		t.Fatalf("check after purchase: %d %s", rec.Code, rec.Body.String())
	}
}

func TestPromoReusableAfterCancelledOrder(t *testing.T) {
	f := checkoutServer(t)
	first := f.createPromoOrder(t, promoBody(1, 25000, "BEMVINDO"))
	second := f.createPromoOrder(t, promoBody(1, 25000, "BEMVINDO"))
	got, _ := f.s.Checkout.Orders.Get(first.ID)
	if got.Status != orders.StatusCancelled || second.Promo == nil {
		t.Fatalf("retry by same buyer must supersede: first %+v second %+v", got, second)
	}
}

func TestPromoDiscountRoundsDown(t *testing.T) {
	if got := promoDiscountCents(50066, promos["BEMVINDO"]); got != 2503 {
		t.Fatalf("discount = %d", got)
	}
}

func TestPromoCheck(t *testing.T) {
	f := checkoutServer(t)
	rec := f.do(t, http.MethodPost, "/api/promos/check", map[string]string{"code": "bemvindo", "collectaUserId": "u1"}, storefrontHeaders())
	if rec.Code != http.StatusOK {
		t.Fatalf("check: %d %s", rec.Code, rec.Body.String())
	}
	var out struct {
		Code       string `json:"code"`
		PercentBps int64  `json:"percentBps"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.Code != "BEMVINDO" || out.PercentBps != 500 {
		t.Fatalf("check = %+v", out)
	}
	rec = f.do(t, http.MethodPost, "/api/promos/check", map[string]string{"code": "BEMVINDO"}, storefrontHeaders())
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("check without user: %d", rec.Code)
	}
	rec = f.do(t, http.MethodPost, "/api/promos/check", map[string]string{"code": "BEMVINDO", "collectaUserId": "u1"}, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("check without token: %d", rec.Code)
	}
}
