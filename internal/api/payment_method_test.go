package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"opdeals/internal/orders"
	"opdeals/internal/trades"
)

func cardPricedServer(t *testing.T) *checkoutFixture {
	t.Helper()
	f := checkoutServer(t)
	card := 270.0
	ships := "2026-11-06"
	if _, err := f.store.SetListings(map[string]trades.Listing{
		f.newer.ID: {AskBRL: 250, Listed: true, CardBRL: &card, ShipsFrom: &ships},
		f.older.ID: {AskBRL: 260, Listed: true},
	}); err != nil {
		t.Fatal(err)
	}
	return f
}

func methodOrderBody(method string, unitCents, cardCents int64) map[string]interface{} {
	b := validOrderBody(1, unitCents)
	b["lines"] = []map[string]interface{}{{"sku": teachSKU, "qty": 1, "unitCents": unitCents, "cardCents": cardCents}}
	if method != "" {
		b["paymentMethod"] = method
	}
	return b
}

func (f *checkoutFixture) createWith(t *testing.T, body map[string]interface{}) orders.Order {
	t.Helper()
	rec := f.do(t, http.MethodPost, "/api/orders", body, storefrontHeaders())
	if rec.Code != http.StatusOK {
		t.Fatalf("create: status %d body %s", rec.Code, rec.Body.String())
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

func TestCardOrderChargesCardPriceInSixInstallments(t *testing.T) {
	f := cardPricedServer(t)
	o := f.createWith(t, methodOrderBody("card", 25000, 26000))
	if o.PaymentMethod != orders.PaymentCard || o.Lines[0].UnitBRL != 260 || o.Totals.TotalBRL != 272 {
		t.Fatalf("order = %+v lines %+v", o.Totals, o.Lines)
	}
	if !o.Lines[0].CardPriced || o.Lines[0].ShipsFrom != "2026-11-06" {
		t.Fatalf("line = %+v", o.Lines[0])
	}
	pref := f.gw.prefs[0]
	if pref.Method != "card" || pref.Installments != cardPricedInstallments || pref.Items[0].UnitBRL != 260 {
		t.Fatalf("pref = %+v", pref)
	}
	pub := publicView(o, o.CreatedAt, nil)
	if pub.PaymentMethod != orders.PaymentCard || pub.Lines[0].ShipsFrom != "2026-11-06" {
		t.Fatalf("public = %+v", pub)
	}
}

func TestPixOrderChargesPixPriceInOneInstallment(t *testing.T) {
	f := cardPricedServer(t)
	o := f.createWith(t, methodOrderBody("pix", 25000, 26000))
	if o.PaymentMethod != orders.PaymentPix || o.Lines[0].UnitBRL != 250 || o.Totals.TotalBRL != 262 {
		t.Fatalf("order = %+v lines %+v", o.Totals, o.Lines)
	}
	if pref := f.gw.prefs[0]; pref.Method != "pix" || pref.Installments != 1 {
		t.Fatalf("pref = %+v", pref)
	}
}

func TestLegacyOrderWithoutMethod(t *testing.T) {
	f := cardPricedServer(t)
	o := f.createWith(t, methodOrderBody("", 25000, 0))
	if o.PaymentMethod != orders.PaymentPix || f.gw.prefs[0].Installments != 1 {
		t.Fatalf("card-priced legacy order must fall back to pix: %+v %+v", o, f.gw.prefs[0])
	}

	plain := checkoutServer(t)
	o = plain.createWith(t, methodOrderBody("", 25000, 0))
	if o.PaymentMethod != orders.PaymentAny || o.Lines[0].CardPriced || plain.gw.prefs[0].Installments != maxInstallments {
		t.Fatalf("plain legacy order = %+v %+v", o, plain.gw.prefs[0])
	}
}

func TestCardPriceChangeAndBadMethodRejected(t *testing.T) {
	f := cardPricedServer(t)
	rec := f.do(t, http.MethodPost, "/api/orders", methodOrderBody("card", 25000, 25000), storefrontHeaders())
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), `"cardCents":26000`) {
		t.Fatalf("card price diff: %d %s", rec.Code, rec.Body.String())
	}
	rec = f.do(t, http.MethodPost, "/api/orders", methodOrderBody("boleto", 25000, 26000), storefrontHeaders())
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bad method: %d %s", rec.Code, rec.Body.String())
	}
}

func TestListingsKeepCardPriceWhenOmitted(t *testing.T) {
	f := cardPricedServer(t)
	rec := f.do(t, http.MethodPost, "/api/trades/listings", map[string]interface{}{
		"items": []map[string]interface{}{{"id": f.newer.ID, "askBRL": 240, "listed": true}},
	}, map[string]string{"X-Admin-Token": "admin"})
	if rec.Code != http.StatusOK {
		t.Fatalf("listings: %d %s", rec.Code, rec.Body.String())
	}
	rec = f.do(t, http.MethodGet, "/api/storefront", nil, nil)
	var resp storefrontResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, it := range resp.Items {
		if it.AskBRL == 240 {
			found = true
			if it.CardBRL != 270 || it.ShipsFrom != "2026-11-06" {
				t.Fatalf("item = %+v", it)
			}
		}
	}
	if !found {
		t.Fatalf("items = %+v", resp.Items)
	}
	rec = f.do(t, http.MethodPost, "/api/trades/listings", map[string]interface{}{
		"items": []map[string]interface{}{{"id": f.newer.ID, "askBRL": 240, "listed": true, "shipsFrom": "06/11"}},
	}, map[string]string{"X-Admin-Token": "admin"})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bad date: %d %s", rec.Code, rec.Body.String())
	}
}

func TestPublicOrderPhotoGetsAPIBase(t *testing.T) {
	o := publicOrder{Lines: []publicLine{{ImageURL: itemPhotosPath + "a.jpg"}, {ImageURL: "https://cdn.test/b.jpg"}, {ImageURL: "/img?x=1"}}}
	got := withPhotoBase(o, "https://api.test")
	if got.Lines[0].ImageURL != "https://api.test"+itemPhotosPath+"a.jpg" || got.Lines[1].ImageURL != "https://cdn.test/b.jpg" || got.Lines[2].ImageURL != "/img?x=1" {
		t.Fatalf("lines = %+v", got.Lines)
	}
}
