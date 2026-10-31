package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"opdeals/internal/promos"
)

func (f *checkoutFixture) addPromo(t *testing.T, p promos.Promo) {
	t.Helper()
	if _, err := f.s.Checkout.Promos.Add(p); err != nil {
		t.Fatal(err)
	}
}

func (f *checkoutFixture) rejectPromo(t *testing.T, body map[string]interface{}, want string) {
	t.Helper()
	rec := f.do(t, http.MethodPost, "/api/orders", body, storefrontHeaders())
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), want) {
		t.Fatalf("want 409 %q, got %d %s", want, rec.Code, rec.Body.String())
	}
}

func TestPromoFixedDiscountOnItems(t *testing.T) {
	f := checkoutServer(t)
	f.addPromo(t, promos.Promo{Code: "MENOS30", Kind: promos.KindFixed, ValueCents: 3000, Active: true})
	o := f.createPromoOrder(t, promoBody(2, 25000, "menos30"))
	if o.Promo == nil || o.Promo.Code != "MENOS30" || o.Promo.ValueBRL != 30 {
		t.Fatalf("promo = %+v", o.Promo)
	}
	if o.Totals.DiscountBRL != 30 || o.Totals.TotalBRL != 482 {
		t.Fatalf("totals = %+v", o.Totals)
	}
	if itemsSumCents(f.gw.prefs[0].Items) != cents(o.Totals.TotalBRL) {
		t.Fatalf("preference items %d != total %d", itemsSumCents(f.gw.prefs[0].Items), cents(o.Totals.TotalBRL))
	}
}

func TestPromoRejections(t *testing.T) {
	f := checkoutServer(t)
	today := time.Now().In(saoPaulo)
	f.addPromo(t, promos.Promo{Code: "OFF", Kind: promos.KindPercent, PercentBps: 1000})
	f.addPromo(t, promos.Promo{Code: "VELHO", Kind: promos.KindPercent, PercentBps: 1000, Active: true, ValidUntil: today.AddDate(0, 0, -1).Format("2006-01-02")})
	f.addPromo(t, promos.Promo{Code: "FUTURO", Kind: promos.KindPercent, PercentBps: 1000, Active: true, ValidFrom: today.AddDate(0, 0, 2).Format("2006-01-02")})
	f.addPromo(t, promos.Promo{Code: "MIN", Kind: promos.KindPercent, PercentBps: 1000, Active: true, MinItemsCents: 100000})
	f.addPromo(t, promos.Promo{Code: "UMA", Kind: promos.KindPercent, PercentBps: 1000, Active: true, MaxUses: 1})

	f.rejectPromo(t, promoBody(1, 25000, "OFF"), promoInactive)
	f.rejectPromo(t, promoBody(1, 25000, "VELHO"), promoExpired)
	f.rejectPromo(t, promoBody(1, 25000, "FUTURO"), promoNotYet)
	f.rejectPromo(t, promoBody(1, 25000, "MIN"), "a partir de R$ 1000,00")

	first := f.createPromoOrder(t, promoBody(1, 25000, "UMA"))
	f.markPaid(t, first.ID)
	other := promoBody(1, 25000, "UMA")
	other["collectaUserId"] = "u2"
	other["buyer"] = map[string]string{"name": "Outra Pessoa", "email": "outra@x.com", "phone": "11999998888", "cpf": "52998224725"}
	f.rejectPromo(t, other, promoExhausted)

	rec := f.do(t, http.MethodPost, "/api/promos/check", map[string]string{"code": "VELHO", "collectaUserId": "u9"}, storefrontHeaders())
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), promoExpired) {
		t.Fatalf("check expired: %d %s", rec.Code, rec.Body.String())
	}
}

func TestPromosAdminCRUD(t *testing.T) {
	f := checkoutServer(t)
	rec := f.do(t, http.MethodGet, "/api/promos", nil, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("list without token: %d", rec.Code)
	}
	rec = f.do(t, http.MethodPost, "/api/promos", map[string]interface{}{"code": "natal10", "kind": "percent", "percentBps": 1000, "active": true}, adminHeaders)
	if rec.Code != http.StatusOK {
		t.Fatalf("create: %d %s", rec.Code, rec.Body.String())
	}
	rec = f.do(t, http.MethodPost, "/api/promos", map[string]interface{}{"code": "NATAL10", "kind": "percent", "percentBps": 500}, adminHeaders)
	if rec.Code != http.StatusConflict {
		t.Fatalf("duplicate: %d", rec.Code)
	}
	rec = f.do(t, http.MethodPost, "/api/promos", map[string]interface{}{"code": "x", "kind": "percent", "percentBps": 500}, adminHeaders)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid: %d", rec.Code)
	}
	rec = f.do(t, http.MethodPut, "/api/promos/NATAL10", map[string]interface{}{"active": false}, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("update without token: %d", rec.Code)
	}
	rec = f.do(t, http.MethodPut, "/api/promos/natal10", map[string]interface{}{"active": false}, adminHeaders)
	if rec.Code != http.StatusOK {
		t.Fatalf("update: %d %s", rec.Code, rec.Body.String())
	}
	got, _ := f.s.Checkout.Promos.Get("NATAL10")
	if got.Active || got.PercentBps != 1000 {
		t.Fatalf("patch must keep other fields: %+v", got)
	}

	o := f.createPromoOrder(t, promoBody(1, 25000, "BEMVINDO"))
	f.markPaid(t, o.ID)
	rec = f.do(t, http.MethodDelete, "/api/promos/BEMVINDO", nil, adminHeaders)
	if rec.Code != http.StatusConflict {
		t.Fatalf("delete used: %d", rec.Code)
	}
	rec = f.do(t, http.MethodDelete, "/api/promos/NATAL10", nil, adminHeaders)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete unused: %d %s", rec.Code, rec.Body.String())
	}

	rec = f.do(t, http.MethodGet, "/api/promos", nil, adminHeaders)
	var out struct {
		Promos []promoView `json:"promos"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Promos) != 1 || out.Promos[0].Code != "BEMVINDO" || out.Promos[0].Uses != 1 || out.Promos[0].DiscountBRL != 12.5 {
		t.Fatalf("list = %+v", out.Promos)
	}
}
