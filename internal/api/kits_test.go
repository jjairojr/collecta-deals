package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"opdeals/internal/kits"
	"opdeals/internal/mercadopago"
	"opdeals/internal/orders"
	"opdeals/internal/trades"
)

const packSKU = "sealed|onepiece|(jap) booster 30 anos|||"

type kitFixture struct {
	*checkoutFixture
	oldLot trades.Trade
	newLot trades.Trade
	kit3   kitView
}

func kitServer(t *testing.T) *kitFixture {
	t.Helper()
	f := checkoutServer(t)
	f.s.Kits = kits.NewStore(filepath.Join(t.TempDir(), "kits.json"))
	pack := trades.Trade{Kind: "sealed", Name: "(JAP) Booster 30 Anos", Set: "SEALED", BuyBRL: 15, Status: "holding"}
	pack.Qty, pack.BuyDate = 6, "2026-10-01"
	newLot, err := f.store.Add(pack)
	if err != nil {
		t.Fatal(err)
	}
	pack.Qty, pack.BuyDate = 4, "2026-09-01"
	oldLot, err := f.store.Add(pack)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.store.SetListings(map[string]trades.Listing{
		newLot.ID: {AskBRL: 30, Listed: true},
		oldLot.ID: {AskBRL: 30, Listed: true},
	}); err != nil {
		t.Fatal(err)
	}
	k := &kitFixture{checkoutFixture: f, oldLot: oldLot, newLot: newLot}
	k.kit3 = k.createKit(t, 3, 80)
	return k
}

func (k *kitFixture) createKit(t *testing.T, units int, askBRL float64) kitView {
	t.Helper()
	rec := k.do(t, http.MethodPost, "/api/kits", map[string]interface{}{
		"tradeID": k.newLot.ID, "units": units, "askBRL": askBRL, "listed": true,
	}, adminHeaders)
	if rec.Code != http.StatusOK {
		t.Fatalf("create kit: status %d body %s", rec.Code, rec.Body.String())
	}
	var out kitView
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func kitOrderBody(buyer int, lines ...map[string]interface{}) map[string]interface{} {
	body := validOrderBody(1, 0)
	body["lines"] = lines
	cpfs := []string{"529.982.247-25", "111.444.777-35"}
	body["buyer"] = map[string]string{
		"name": "Ana Souza", "email": fmt.Sprintf("buyer%d@example.com", buyer), "phone": "(11) 99999-8888", "cpf": cpfs[buyer],
	}
	return body
}

func orderLine(sku string, qty int, unitCents int64) map[string]interface{} {
	return map[string]interface{}{"sku": sku, "qty": qty, "unitCents": unitCents}
}

func (k *kitFixture) storefront(t *testing.T) storefrontResponse {
	t.Helper()
	rec := k.do(t, http.MethodGet, "/api/storefront", nil, nil)
	var resp storefrontResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	return resp
}

func (k *kitFixture) packUnits(t *testing.T, status string) int {
	t.Helper()
	all, err := k.store.List()
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, tr := range all {
		if tr.Kind == "sealed" && tr.Status == status {
			n += tr.Qty
		}
	}
	return n
}

func (k *kitFixture) placeOrder(t *testing.T, body map[string]interface{}) orders.Order {
	t.Helper()
	rec := k.do(t, http.MethodPost, "/api/orders", body, storefrontHeaders())
	if rec.Code != http.StatusOK {
		t.Fatalf("create order: status %d body %s", rec.Code, rec.Body.String())
	}
	var out struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	o, err := k.s.Checkout.Orders.Get(out.ID)
	if err != nil {
		t.Fatal(err)
	}
	return o
}

func (k *kitFixture) pay(t *testing.T, o orders.Order) orders.Order {
	t.Helper()
	k.gw.payments[900] = mercadopago.Payment{ID: 900, Status: "approved", ExternalReference: o.ID, AmountBRL: o.Totals.TotalBRL, Method: "pix"}
	if rec := k.webhook(t, "900", testWebhookSecret); rec.Code != http.StatusOK {
		t.Fatalf("webhook: status %d body %s", rec.Code, rec.Body.String())
	}
	got, err := k.s.Checkout.Orders.Get(o.ID)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestKitRoutesNeedAnExistingHolding(t *testing.T) {
	k := kitServer(t)
	rec := k.do(t, http.MethodPost, "/api/kits", map[string]interface{}{"tradeID": "nope", "units": 2, "askBRL": 55}, adminHeaders)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown holding: status %d", rec.Code)
	}
	rec = k.do(t, http.MethodPost, "/api/kits", map[string]interface{}{"tradeID": k.newLot.ID, "units": 3, "askBRL": 70}, adminHeaders)
	if rec.Code != http.StatusConflict {
		t.Fatalf("duplicate size: status %d", rec.Code)
	}
	rec = k.do(t, http.MethodGet, "/api/kits", nil, adminHeaders)
	var list struct {
		Kits []kitView `json:"kits"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Kits) != 1 || list.Kits[0].BaseSKU != packSKU || len(list.Kits[0].TradeIDs) != 2 {
		t.Fatalf("kits = %+v", list.Kits)
	}
}

func TestStorefrontOffersKitWhileBaseStockCoversIt(t *testing.T) {
	k := kitServer(t)
	resp := k.storefront(t)
	if len(resp.Kits) != 1 || resp.Kits[0].SKU != k.kit3.SKU || resp.Kits[0].BaseSKU != packSKU || resp.Kits[0].Units != 3 || resp.Kits[0].AskBRL != 80 {
		t.Fatalf("kits = %+v", resp.Kits)
	}
	k.placeOrder(t, kitOrderBody(0, orderLine(packSKU, 8, 3000)))
	if resp = k.storefront(t); len(resp.Kits) != 0 {
		t.Fatalf("2 packs left cannot fill a kit of 3: %+v", resp.Kits)
	}
}

func TestKitOrderReservesUnitsOfTheBaseProduct(t *testing.T) {
	k := kitServer(t)
	o := k.placeOrder(t, kitOrderBody(0, orderLine(k.kit3.SKU, 2, 8000)))
	l := o.Lines[0]
	if l.Units != 3 || l.StockSKU != packSKU || l.Qty != 2 || l.UnitBRL != 80 || !strings.Contains(l.Name, "Kit 3 un") {
		t.Fatalf("line = %+v", l)
	}
	if o.Totals.ItemsBRL != 160 {
		t.Fatalf("items total = %v want 160", o.Totals.ItemsBRL)
	}
	if got := k.gw.prefs[0].Items[0]; got.Qty != 2 || got.UnitBRL != 80 {
		t.Fatalf("payment item = %+v", got)
	}
	if len(k.quoter.products) != 1 || k.quoter.products[0].Quantity != 6 {
		t.Fatalf("shipping must pack 6 units: %+v", k.quoter.products)
	}
	left := 0
	for _, it := range k.storefront(t).Items {
		if it.SKU == packSKU {
			left += it.Qty
		}
	}
	if left != 4 {
		t.Fatalf("packs left on the storefront = %d want 4", left)
	}
	rec := k.do(t, http.MethodPost, "/api/orders", kitOrderBody(1, orderLine(packSKU, 5, 3000)), storefrontHeaders())
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), `"available":4`) {
		t.Fatalf("second buyer: status %d body %s", rec.Code, rec.Body.String())
	}
	rec = k.do(t, http.MethodPost, "/api/orders", kitOrderBody(1, orderLine(k.kit3.SKU, 2, 8000)), storefrontHeaders())
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), `"available":1`) {
		t.Fatalf("second buyer kit: status %d body %s", rec.Code, rec.Body.String())
	}
}

func TestPaidKitSellsItsUnitsOldestLotFirst(t *testing.T) {
	k := kitServer(t)
	got := k.pay(t, k.placeOrder(t, kitOrderBody(0, orderLine(k.kit3.SKU, 2, 8000))))
	if got.Status != orders.StatusPaid {
		t.Fatalf("order = %s notes %q", got.Status, got.Notes)
	}
	if sold, held := k.packUnits(t, "sold"), k.packUnits(t, "holding"); sold != 6 || held != 4 {
		t.Fatalf("sold %d holding %d, want 6 and 4", sold, held)
	}
	all, _ := k.store.List()
	revenue := 0.0
	for _, tr := range all {
		if tr.Kind != "sealed" || tr.Status != "sold" {
			continue
		}
		revenue += tr.SellPrice
		if tr.ID == k.oldLot.ID && (tr.Qty != 4 || tr.SellPrice != 106.67) {
			t.Fatalf("oldest lot must go whole at the kit's per-unit price: %+v", tr)
		}
		if tr.Channel != trades.ChannelSite {
			t.Fatalf("channel = %q", tr.Channel)
		}
	}
	if round2(revenue) != 160 {
		t.Fatalf("ledger revenue = %v want 160", revenue)
	}
}

func TestKitAndSingleUnitsShareOneStock(t *testing.T) {
	k := kitServer(t)
	rec := k.do(t, http.MethodPost, "/api/orders", kitOrderBody(0, orderLine(packSKU, 8, 3000), orderLine(k.kit3.SKU, 1, 8000)), storefrontHeaders())
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "out_of_stock") {
		t.Fatalf("11 units of 10: status %d body %s", rec.Code, rec.Body.String())
	}
	got := k.pay(t, k.placeOrder(t, kitOrderBody(0, orderLine(packSKU, 7, 3000), orderLine(k.kit3.SKU, 1, 8000))))
	if got.Status != orders.StatusPaid {
		t.Fatalf("order = %s notes %q", got.Status, got.Notes)
	}
	if sold, held := k.packUnits(t, "sold"), k.packUnits(t, "holding"); sold != 10 || held != 0 {
		t.Fatalf("sold %d holding %d, want 10 and 0", sold, held)
	}
}

func TestKitChargesItsOwnPrice(t *testing.T) {
	k := kitServer(t)
	rec := k.do(t, http.MethodPost, "/api/orders", kitOrderBody(0, orderLine(k.kit3.SKU, 1, 9000)), storefrontHeaders())
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), `"unitCents":8000`) {
		t.Fatalf("stale kit price: status %d body %s", rec.Code, rec.Body.String())
	}
	if rec := k.do(t, http.MethodPut, "/api/kits/"+k.kit3.ID, map[string]interface{}{"listed": false}, adminHeaders); rec.Code != http.StatusOK {
		t.Fatalf("unlist: status %d body %s", rec.Code, rec.Body.String())
	}
	rec = k.do(t, http.MethodPost, "/api/orders", kitOrderBody(0, orderLine(k.kit3.SKU, 1, 8000)), storefrontHeaders())
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "out_of_stock") {
		t.Fatalf("unlisted kit: status %d body %s", rec.Code, rec.Body.String())
	}
}
