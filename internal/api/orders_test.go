package api

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"opdeals/internal/game"
	"opdeals/internal/logx"
	"opdeals/internal/mercadopago"
	"opdeals/internal/orders"
	"opdeals/internal/promos"
	"opdeals/internal/shipping"
	"opdeals/internal/trades"
)

const (
	testStorefrontToken = "sf-token"
	testWebhookSecret   = "wh-secret"
)

type fakeGateway struct {
	mu       sync.Mutex
	prefs    []mercadopago.PreferenceInput
	payments map[int64]mercadopago.Payment
	fail     bool
}

func (g *fakeGateway) CreatePreference(_ context.Context, in mercadopago.PreferenceInput) (mercadopago.Preference, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.fail {
		return mercadopago.Preference{}, errors.New("mp down")
	}
	g.prefs = append(g.prefs, in)
	return mercadopago.Preference{ID: "pref-" + in.OrderID, InitPoint: "https://mp.test/" + in.OrderID}, nil
}

func (g *fakeGateway) GetPayment(_ context.Context, id int64) (mercadopago.Payment, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	p, ok := g.payments[id]
	if !ok {
		return mercadopago.Payment{}, errors.New("payment not found")
	}
	return p, nil
}

type fakeQuoter struct {
	products []shipping.Product
}

func (q *fakeQuoter) Quote(_ context.Context, _, _ string, products []shipping.Product) ([]shipping.Service, error) {
	q.products = products
	return []shipping.Service{
		{ID: 17, Name: "Mini Envios", Company: "Correios", PriceBRL: 12, DeliveryDays: 9},
		{ID: 2, Name: "SEDEX", Company: "Correios", PriceBRL: 35.1, DeliveryDays: 3},
	}, nil
}

type checkoutFixture struct {
	s      *Server
	store  *trades.Store
	gw     *fakeGateway
	quoter *fakeQuoter
	older  trades.Trade
	newer  trades.Trade
}

func checkoutServer(t *testing.T) *checkoutFixture {
	t.Helper()
	dir := t.TempDir()
	store := trades.NewStore(filepath.Join(dir, "trades.json"), logx.New(io.Discard))
	gs := &GameStack{Game: game.Game{ID: "onepiece", Name: "One Piece"}, Trades: store}
	gw := &fakeGateway{payments: map[int64]mercadopago.Payment{}}
	quoter := &fakeQuoter{}
	s := &Server{games: map[string]*GameStack{"onepiece": gs}, defaultGame: "onepiece", adminToken: "admin"}
	s.Checkout = Checkout{
		Orders:          orders.NewStore(filepath.Join(dir, "orders.json"), logx.New(io.Discard)),
		Promos:          promos.NewStore(filepath.Join(dir, "promos.json")),
		Gateway:         gw,
		Quoter:          quoter,
		StorefrontToken: testStorefrontToken,
		WebhookSecret:   testWebhookSecret,
		SiteURL:         "https://site.test",
		APIURL:          "https://api.test",
		FromCEP:         "01001000",
		Log:             logx.New(io.Discard),
	}
	newer, err := store.Add(trades.Trade{Number: "OP16-080", Name: "Teach", Set: "OP16", Qty: 2, BuyBRL: 100, BuyDate: "2026-09-01", Status: "holding"})
	if err != nil {
		t.Fatal(err)
	}
	older, err := store.Add(trades.Trade{Number: "OP16-080", Name: "Teach", Set: "OP16", Qty: 1, BuyBRL: 90, BuyDate: "2026-08-01", Status: "holding"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.SetListings(map[string]trades.Listing{
		newer.ID: {AskBRL: 250, Listed: true},
		older.ID: {AskBRL: 260, Listed: true},
	}); err != nil {
		t.Fatal(err)
	}
	return &checkoutFixture{s: s, store: store, gw: gw, quoter: quoter, older: older, newer: newer}
}

const teachSKU = "single|onepiece|teach|op16-080||"

func (f *checkoutFixture) do(t *testing.T, method, path string, body interface{}, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	f.s.Handler().ServeHTTP(rec, req)
	return rec
}

func storefrontHeaders() map[string]string {
	return map[string]string{"X-Storefront-Token": testStorefrontToken}
}

func validOrderBody(qty int, unitCents int64) map[string]interface{} {
	return map[string]interface{}{
		"lines": []map[string]interface{}{{"sku": teachSKU, "qty": qty, "unitCents": unitCents}},
		"buyer": map[string]string{"name": "Ana Souza", "email": "ana@example.com", "phone": "(11) 99999-8888", "cpf": "529.982.247-25"},
		"address": map[string]string{
			"cep": "04538-132", "street": "Av. Faria Lima", "number": "1000", "district": "Itaim", "city": "São Paulo", "state": "sp",
		},
		"shipping":    map[string]interface{}{"serviceID": 17, "priceBRL": 12},
		"backURLBase": "https://site.test",
	}
}

func (f *checkoutFixture) createOrder(t *testing.T) orders.Order {
	t.Helper()
	rec := f.do(t, http.MethodPost, "/api/orders", validOrderBody(2, 25000), storefrontHeaders())
	if rec.Code != http.StatusOK {
		t.Fatalf("create: status %d body %s", rec.Code, rec.Body.String())
	}
	var out struct {
		ID        string `json:"id"`
		Number    int    `json:"number"`
		InitPoint string `json:"initPoint"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.Number != 1 || out.InitPoint != "https://mp.test/"+out.ID {
		t.Fatalf("unexpected create response: %+v", out)
	}
	o, err := f.s.Checkout.Orders.Get(out.ID)
	if err != nil {
		t.Fatal(err)
	}
	return o
}

func signature(dataID, requestID, secret string) string {
	ts := "1704908010000"
	manifest := "id:" + dataID + ";request-id:" + requestID + ";ts:" + ts + ";"
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(manifest))
	return "ts=" + ts + ",v1=" + hex.EncodeToString(mac.Sum(nil))
}

func (f *checkoutFixture) webhook(t *testing.T, paymentID string, secret string) *httptest.ResponseRecorder {
	t.Helper()
	return f.do(t, http.MethodPost, "/api/orders/mp-webhook?type=payment&data.id="+paymentID,
		map[string]interface{}{"type": "payment", "action": "payment.updated", "data": map[string]string{"id": paymentID}},
		map[string]string{"x-signature": signature(paymentID, "req-1", secret), "x-request-id": "req-1"})
}

func (f *checkoutFixture) soldTrades(t *testing.T) []trades.Trade {
	t.Helper()
	all, err := f.store.List()
	if err != nil {
		t.Fatal(err)
	}
	out := make([]trades.Trade, 0)
	for _, tr := range all {
		if tr.Status == "sold" {
			out = append(out, tr)
		}
	}
	return out
}

func TestStorefrontCarriesSKUAndSubtractsReservations(t *testing.T) {
	f := checkoutServer(t)
	rec := f.do(t, http.MethodGet, "/api/storefront", nil, nil)
	var resp storefrontResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Items) != 2 || resp.Items[0].SKU != teachSKU || resp.Items[1].SKU != teachSKU {
		t.Fatalf("items = %+v", resp.Items)
	}
	f.createOrder(t)
	rec = f.do(t, http.MethodGet, "/api/storefront", nil, nil)
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	total := 0
	for _, it := range resp.Items {
		total += it.Qty
	}
	if total != 1 {
		t.Fatalf("storefront qty after reservation = %d want 1 (%+v)", total, resp.Items)
	}
}

func TestOrderCreateRejections(t *testing.T) {
	f := checkoutServer(t)
	cases := []struct {
		name   string
		body   map[string]interface{}
		header map[string]string
		code   int
		want   string
	}{
		{"missing token", validOrderBody(1, 25000), nil, http.StatusUnauthorized, ""},
		{"price changed", validOrderBody(1, 26000), storefrontHeaders(), http.StatusConflict, "price_changed"},
		{"out of stock", validOrderBody(4, 25000), storefrontHeaders(), http.StatusConflict, "out_of_stock"},
		{"bad cpf", func() map[string]interface{} {
			b := validOrderBody(1, 25000)
			b["buyer"] = map[string]string{"name": "Ana Souza", "email": "ana@example.com", "phone": "11999998888", "cpf": "111.111.111-11"}
			return b
		}(), storefrontHeaders(), http.StatusBadRequest, "cpf"},
		{"shipping changed", func() map[string]interface{} {
			b := validOrderBody(1, 25000)
			b["shipping"] = map[string]interface{}{"serviceID": 2, "priceBRL": 12}
			return b
		}(), storefrontHeaders(), http.StatusConflict, "shipping_changed"},
	}
	for _, c := range cases {
		rec := f.do(t, http.MethodPost, "/api/orders", c.body, c.header)
		if rec.Code != c.code || !strings.Contains(rec.Body.String(), c.want) {
			t.Fatalf("%s: status %d body %s", c.name, rec.Code, rec.Body.String())
		}
	}
	all, _ := f.s.Checkout.Orders.List()
	if len(all) != 0 {
		t.Fatalf("rejections must not persist orders: %+v", all)
	}
}

func TestOrderCreateBuildsPreference(t *testing.T) {
	f := checkoutServer(t)
	o := f.createOrder(t)
	if o.Totals.ItemsBRL != 500 || o.Totals.ShippingBRL != 12 || o.Totals.TotalBRL != 512 {
		t.Fatalf("totals = %+v", o.Totals)
	}
	if o.Buyer.CPF != "52998224725" || o.Buyer.Phone != "11999998888" || o.Address.CEP != "04538132" || o.Address.State != "SP" {
		t.Fatalf("normalization: %+v %+v", o.Buyer, o.Address)
	}
	if o.Lines[0].UnitBRL != 250 || o.Lines[0].Meta != "OP16 · OP16-080" || o.Lines[0].Kind != "single" {
		t.Fatalf("line = %+v", o.Lines[0])
	}
	if o.Lines[0].ImageURL != "" {
		t.Fatalf("line image = %q, want none for an OP listing without art", o.Lines[0].ImageURL)
	}
	pub := publicView(o, time.Now(), nil)
	if pub.Lines[0].ImageURL != "" {
		t.Fatalf("public image = %q", pub.Lines[0].ImageURL)
	}
	if o.Shipping.ServiceName != "Mini Envios" || o.MP.PreferenceID != "pref-"+o.ID {
		t.Fatalf("shipping/mp = %+v %+v", o.Shipping, o.MP)
	}
	pref := f.gw.prefs[0]
	if pref.ReturnURL != "https://site.test/pedido/"+o.ID || pref.NotificationURL != "https://api.test/api/orders/mp-webhook" {
		t.Fatalf("urls = %+v", pref)
	}
	if len(pref.Items) != 2 || pref.Items[0].UnitBRL != 250 || pref.Items[0].Qty != 2 || pref.Items[1].UnitBRL != 12 {
		t.Fatalf("items = %+v", pref.Items)
	}
	if !pref.ExpiresAt.Equal(o.ExpiresAt) {
		t.Fatalf("expires = %v want %v", pref.ExpiresAt, o.ExpiresAt)
	}
	if len(f.quoter.products) != 1 || f.quoter.products[0].ID != "singles" || f.quoter.products[0].InsuranceValue != 500 {
		t.Fatalf("packed products = %+v", f.quoter.products)
	}
}

func TestOrderCreateGatewayFailureCancels(t *testing.T) {
	f := checkoutServer(t)
	f.gw.fail = true
	rec := f.do(t, http.MethodPost, "/api/orders", validOrderBody(1, 25000), storefrontHeaders())
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status %d", rec.Code)
	}
	all, _ := f.s.Checkout.Orders.List()
	if len(all) != 1 || all[0].Status != orders.StatusCancelled {
		t.Fatalf("orders = %+v", all)
	}
	reserved, _ := f.s.Checkout.Orders.Reserved(time.Now())
	if len(reserved) != 0 {
		t.Fatalf("cancelled order still reserves: %v", reserved)
	}
}

func TestWebhookApprovedSellsFIFOOnce(t *testing.T) {
	f := checkoutServer(t)
	o := f.createOrder(t)
	f.gw.payments[777] = mercadopago.Payment{ID: 777, Status: "approved", ExternalReference: o.ID, AmountBRL: 512, Method: "pix"}

	for i := 0; i < 2; i++ {
		if rec := f.webhook(t, "777", testWebhookSecret); rec.Code != http.StatusOK {
			t.Fatalf("webhook %d: status %d body %s", i, rec.Code, rec.Body.String())
		}
	}
	got, _ := f.s.Checkout.Orders.Get(o.ID)
	if got.Status != orders.StatusPaid || got.MP.PaymentID != 777 || got.MP.Method != "pix" || got.PaidAt.IsZero() {
		t.Fatalf("order = %+v", got)
	}
	sold := f.soldTrades(t)
	if len(sold) != 2 {
		t.Fatalf("sold = %+v", sold)
	}
	byID := map[string]trades.Trade{}
	for _, s := range sold {
		byID[s.ID] = s
	}
	if len(got.Lines[0].TradeIDs) != 2 {
		t.Fatalf("tradeIDs = %v", got.Lines[0].TradeIDs)
	}
	first := byID[got.Lines[0].TradeIDs[0]]
	if first.ID != f.older.ID || first.Qty != 1 || first.SellPrice != 250 || first.Buyer != "Ana Souza — Site #1" || first.SellCurrency != "BRL" {
		t.Fatalf("oldest sold first: %+v", first)
	}
	second := byID[got.Lines[0].TradeIDs[1]]
	if second.Qty != 1 || second.SellPrice != 250 || second.ID == f.newer.ID {
		t.Fatalf("split of newer: %+v", second)
	}
	all, _ := f.store.List()
	for _, tr := range all {
		if tr.ID == f.newer.ID && (tr.Status != "holding" || tr.Qty != 1) {
			t.Fatalf("newer remainder = %+v", tr)
		}
	}
	if strings.Contains(got.Notes, "duplicado") {
		t.Fatalf("same payment replay must not note a duplicate: %q", got.Notes)
	}
}

func TestWebhookConcurrentDuplicatesSellOnce(t *testing.T) {
	f := checkoutServer(t)
	o := f.createOrder(t)
	f.gw.payments[1] = mercadopago.Payment{ID: 1, Status: "approved", ExternalReference: o.ID, AmountBRL: 512, Method: "pix"}
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			f.webhook(t, "1", testWebhookSecret)
		}()
	}
	wg.Wait()
	if sold := f.soldTrades(t); len(sold) != 2 {
		t.Fatalf("sold = %+v", sold)
	}
	got, _ := f.s.Checkout.Orders.Get(o.ID)
	if got.Status != orders.StatusPaid || len(got.Lines[0].TradeIDs) != 2 {
		t.Fatalf("order = %+v", got)
	}
}

func TestWebhookAmountMismatchFails(t *testing.T) {
	f := checkoutServer(t)
	o := f.createOrder(t)
	f.gw.payments[5] = mercadopago.Payment{ID: 5, Status: "approved", ExternalReference: o.ID, AmountBRL: 500}
	f.webhook(t, "5", testWebhookSecret)
	got, _ := f.s.Checkout.Orders.Get(o.ID)
	if got.Status != orders.StatusFailed || !strings.Contains(got.Notes, "difere") {
		t.Fatalf("order = %+v", got)
	}
	if sold := f.soldTrades(t); len(sold) != 0 {
		t.Fatalf("must not sell: %+v", sold)
	}
}

func TestWebhookStockGoneOversold(t *testing.T) {
	f := checkoutServer(t)
	o := f.createOrder(t)
	if _, err := f.store.Sell(f.newer.ID, trades.Sale{Qty: 2, SellPrice: 400, SellCurrency: "BRL", Buyer: "balcão"}); err != nil {
		t.Fatal(err)
	}
	f.gw.payments[9] = mercadopago.Payment{ID: 9, Status: "approved", ExternalReference: o.ID, AmountBRL: 512}
	if rec := f.webhook(t, "9", testWebhookSecret); rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	got, _ := f.s.Checkout.Orders.Get(o.ID)
	if got.Status != orders.StatusOversold || !strings.Contains(got.Notes, "sem estoque") {
		t.Fatalf("order = %+v", got)
	}
	if sold := f.soldTrades(t); len(sold) != 1 || sold[0].Buyer != "balcão" {
		t.Fatalf("site must not sell anything: %+v", sold)
	}
}

func TestWebhookRejectedThenApproved(t *testing.T) {
	f := checkoutServer(t)
	o := f.createOrder(t)
	f.gw.payments[2] = mercadopago.Payment{ID: 2, Status: "rejected", ExternalReference: o.ID, AmountBRL: 512, Method: "master"}
	f.webhook(t, "2", testWebhookSecret)
	got, _ := f.s.Checkout.Orders.Get(o.ID)
	if got.Status != orders.StatusFailed {
		t.Fatalf("order = %+v", got)
	}
	rec := f.do(t, http.MethodGet, "/api/orders/"+o.ID, nil, nil)
	var pub publicOrder
	if err := json.Unmarshal(rec.Body.Bytes(), &pub); err != nil {
		t.Fatal(err)
	}
	if pub.InitPoint == "" || pub.FirstName != "Ana" || pub.Status != orders.StatusFailed {
		t.Fatalf("public = %+v", pub)
	}
	f.gw.payments[3] = mercadopago.Payment{ID: 3, Status: "approved", ExternalReference: o.ID, AmountBRL: 512, Method: "pix"}
	rec = f.do(t, http.MethodGet, "/api/orders/"+o.ID+"?payment_id=3", nil, nil)
	pub = publicOrder{}
	if err := json.Unmarshal(rec.Body.Bytes(), &pub); err != nil {
		t.Fatal(err)
	}
	if pub.Status != orders.StatusPaid || pub.InitPoint != "" {
		t.Fatalf("public after sync = %+v", pub)
	}
	if body := rec.Body.String(); strings.Contains(body, "52998224725") || strings.Contains(body, "ana@example.com") {
		t.Fatalf("public view leaks PII: %s", body)
	}
}

func TestWebhookAuth(t *testing.T) {
	f := checkoutServer(t)
	o := f.createOrder(t)
	f.gw.payments[4] = mercadopago.Payment{ID: 4, Status: "approved", ExternalReference: o.ID, AmountBRL: 512}
	if rec := f.webhook(t, "4", "wrong"); rec.Code != http.StatusUnauthorized {
		t.Fatalf("bad signature: status %d", rec.Code)
	}
	rec := f.do(t, http.MethodPost, "/api/orders/mp-webhook?type=merchant_order&data.id=99", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("other topic: status %d", rec.Code)
	}
	got, _ := f.s.Checkout.Orders.Get(o.ID)
	if got.Status != orders.StatusPending {
		t.Fatalf("order = %+v", got)
	}
}

func TestOrdersAdminListAndPatch(t *testing.T) {
	f := checkoutServer(t)
	o := f.createOrder(t)
	if rec := f.do(t, http.MethodGet, "/api/orders", nil, nil); rec.Code != http.StatusUnauthorized {
		t.Fatalf("list without token: %d", rec.Code)
	}
	admin := map[string]string{"X-Admin-Token": "admin"}
	rec := f.do(t, http.MethodGet, "/api/orders", nil, admin)
	var all []orders.Order
	if err := json.Unmarshal(rec.Body.Bytes(), &all); err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 || all[0].Buyer.CPF != "52998224725" {
		t.Fatalf("list = %+v", all)
	}
	rec = f.do(t, http.MethodPatch, "/api/orders/"+o.ID, map[string]string{"status": "shipped"}, admin)
	if rec.Code != http.StatusConflict {
		t.Fatalf("shipping a pending order: %d", rec.Code)
	}
	f.gw.payments[6] = mercadopago.Payment{ID: 6, Status: "approved", ExternalReference: o.ID, AmountBRL: 512}
	f.webhook(t, "6", testWebhookSecret)
	rec = f.do(t, http.MethodPatch, "/api/orders/"+o.ID, map[string]string{"status": "shipped", "trackingCode": " AA123456789BR "}, admin)
	if rec.Code != http.StatusOK {
		t.Fatalf("patch: %d %s", rec.Code, rec.Body.String())
	}
	got, _ := f.s.Checkout.Orders.Get(o.ID)
	if got.Status != orders.StatusShipped || got.TrackingCode != "AA123456789BR" || got.ShippedAt.IsZero() {
		t.Fatalf("order = %+v", got)
	}
	rec = f.do(t, http.MethodPatch, "/api/orders/"+o.ID, map[string]string{"status": "delivered"}, admin)
	if rec.Code != http.StatusOK {
		t.Fatalf("deliver: %d %s", rec.Code, rec.Body.String())
	}
	got, _ = f.s.Checkout.Orders.Get(o.ID)
	if got.Status != orders.StatusDelivered || got.DeliveredAt.IsZero() || got.TrackingCode != "AA123456789BR" {
		t.Fatalf("order = %+v", got)
	}
	rec = f.do(t, http.MethodPatch, "/api/orders/"+o.ID, map[string]string{"status": "shipped"}, admin)
	if rec.Code != http.StatusOK {
		t.Fatalf("delivered back to shipped: %d %s", rec.Code, rec.Body.String())
	}
	got, _ = f.s.Checkout.Orders.Get(o.ID)
	if got.Status != orders.StatusShipped || !got.DeliveredAt.IsZero() || got.ShippedAt.IsZero() {
		t.Fatalf("order = %+v", got)
	}
	rec = f.do(t, http.MethodPatch, "/api/orders/"+o.ID, map[string]string{"status": "cancelled"}, admin)
	if rec.Code != http.StatusConflict {
		t.Fatalf("cancelling a shipped order: %d", rec.Code)
	}
	rec = f.do(t, http.MethodPatch, "/api/orders/"+o.ID, map[string]string{"status": "refunded"}, admin)
	if rec.Code != http.StatusOK {
		t.Fatalf("refund: %d %s", rec.Code, rec.Body.String())
	}
}

func TestShippingQuoteAppliesFreeShipping(t *testing.T) {
	f := checkoutServer(t)
	f.s.Checkout.FreeShippingMin = 300
	body := map[string]interface{}{"cep": "04538-132", "lines": []map[string]interface{}{{"sku": teachSKU, "qty": 2, "unitCents": 25000}}}
	rec := f.do(t, http.MethodPost, "/api/shipping/quote", body, storefrontHeaders())
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d %s", rec.Code, rec.Body.String())
	}
	var out struct {
		Services []shipping.Service `json:"services"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Services) != 4 || out.Services[0].PriceBRL != 0 || out.Services[1].PriceBRL != 35.1 || out.Services[2].ID != shipping.PickupServiceID || out.Services[3].ID != shipping.JoinServiceID {
		t.Fatalf("services = %+v", out.Services)
	}
}

func TestPickupOption(t *testing.T) {
	f := checkoutServer(t)
	quote := func(cep string) []shipping.Service {
		body := map[string]interface{}{"cep": cep, "lines": []map[string]interface{}{{"sku": teachSKU, "qty": 1, "unitCents": 25000}}}
		rec := f.do(t, http.MethodPost, "/api/shipping/quote", body, storefrontHeaders())
		var out struct {
			Services []shipping.Service `json:"services"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		return out.Services
	}
	local := quote("")
	if len(local) != 2 || local[0].ID != shipping.PickupServiceID || local[0].Company != "Goiânia" || local[1].ID != shipping.JoinServiceID {
		t.Fatalf("no-cep quote must list only local options: %+v", local)
	}
	full := quote("04538-132")
	if len(full) != 4 || full[2].ID != shipping.PickupServiceID || full[2].PriceBRL != 0 {
		t.Fatalf("services = %+v", full)
	}
	order := validOrderBody(1, 25000)
	delete(order, "address")
	order["shipping"] = map[string]interface{}{"serviceID": shipping.PickupServiceID, "priceBRL": 0}
	rec := f.do(t, http.MethodPost, "/api/orders", order, storefrontHeaders())
	if rec.Code != http.StatusOK {
		t.Fatalf("create: %d %s", rec.Code, rec.Body.String())
	}
	all, _ := f.s.Checkout.Orders.List()
	if all[0].Totals.TotalBRL != 250 || all[0].Shipping.ServiceName != "Retirar em mãos" || all[0].Address.CEP != "" {
		t.Fatalf("order = %+v", all[0])
	}
	if items := f.gw.prefs[len(f.gw.prefs)-1].Items; len(items) != 1 {
		t.Fatalf("pickup must not add a freight item: %+v", items)
	}
}

func TestJoinWithPaidOrder(t *testing.T) {
	f := checkoutServer(t)
	first := f.createOrder(t)
	join := func(number int) *httptest.ResponseRecorder {
		body := validOrderBody(1, 25000)
		delete(body, "address")
		body["shipping"] = map[string]interface{}{"serviceID": shipping.JoinServiceID, "priceBRL": 0, "joinOrder": number}
		return f.do(t, http.MethodPost, "/api/orders", body, storefrontHeaders())
	}
	if rec := join(first.Number); rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "join_not_found") {
		t.Fatalf("joining a pending order: %d %s", rec.Code, rec.Body.String())
	}
	f.gw.payments[8] = mercadopago.Payment{ID: 8, Status: "approved", ExternalReference: first.ID, AmountBRL: 512}
	f.webhook(t, "8", testWebhookSecret)
	if rec := join(99); rec.Code != http.StatusConflict {
		t.Fatalf("unknown number: %d", rec.Code)
	}
	rec := join(first.Number)
	if rec.Code != http.StatusOK {
		t.Fatalf("join: %d %s", rec.Code, rec.Body.String())
	}
	all, _ := f.s.Checkout.Orders.List()
	joined := all[0]
	if joined.Shipping.JoinOrder != first.Number || joined.Shipping.Company != "pedido #1" || joined.Totals.TotalBRL != 250 {
		t.Fatalf("joined = %+v", joined.Shipping)
	}
	if joined.Address != first.Address || joined.Address.CEP == "" {
		t.Fatalf("joined order must inherit the base address: %+v", joined.Address)
	}
}

func TestRetrySupersedesPendingOrder(t *testing.T) {
	f := checkoutServer(t)
	first := f.createOrder(t)
	rec := f.do(t, http.MethodPost, "/api/orders", validOrderBody(2, 25000), storefrontHeaders())
	if rec.Code != http.StatusOK {
		t.Fatalf("retry must not hit its own reservation: %d %s", rec.Code, rec.Body.String())
	}
	got, _ := f.s.Checkout.Orders.Get(first.ID)
	if got.Status != orders.StatusCancelled || !strings.Contains(got.Notes, "nova tentativa") {
		t.Fatalf("first order = %+v", got)
	}
	reserved, _ := f.s.Checkout.Orders.Reserved(time.Now())
	if reserved[teachSKU] != 2 {
		t.Fatalf("reserved = %v want only the retry", reserved)
	}
	f.gw.payments[11] = mercadopago.Payment{ID: 11, Status: "approved", ExternalReference: first.ID, AmountBRL: 512, Method: "pix"}
	f.webhook(t, "11", testWebhookSecret)
	got, _ = f.s.Checkout.Orders.Get(first.ID)
	if got.Status != orders.StatusPaid || len(f.soldTrades(t)) != 2 {
		t.Fatalf("a cancelled order that gets paid must still settle: %+v", got)
	}
}

func TestCancelPendingIgnoresOtherBuyers(t *testing.T) {
	f := checkoutServer(t)
	first := f.createOrder(t)
	other := validOrderBody(2, 25000)
	other["buyer"] = map[string]string{"name": "Bruno Lima", "email": "bruno@example.com", "phone": "11988887777", "cpf": "390.533.447-05"}
	rec := f.do(t, http.MethodPost, "/api/orders", other, storefrontHeaders())
	if rec.Code != http.StatusConflict {
		t.Fatalf("another buyer must still see the reservation: %d %s", rec.Code, rec.Body.String())
	}
	got, _ := f.s.Checkout.Orders.Get(first.ID)
	if got.Status != orders.StatusPending {
		t.Fatalf("first order = %+v", got)
	}
}

const listingArtURL = "https://repositorio.sbrauble.com/arquivos/in/onepiece/1/teach.jpg"

func TestOrderLineTakesListingArtFromAnyLot(t *testing.T) {
	f := checkoutServer(t)
	if _, err := f.store.SetListings(map[string]trades.Listing{
		f.newer.ID: {AskBRL: 250, Listed: true, ImageURL: listingArtURL},
	}); err != nil {
		t.Fatal(err)
	}
	o := f.createOrder(t)
	if o.Lines[0].ImageURL != listingArtURL {
		t.Fatalf("line image = %q, want the newer lot's listing art", o.Lines[0].ImageURL)
	}
}

func TestOrderLineReplacesGuessedListingArt(t *testing.T) {
	f := checkoutServer(t)
	o := f.createOrder(t)
	o.Lines[0].ImageURL = "/api/card-image?game=onepiece&number=OP16-080&productID=656656&set=OP16"
	if got := orderLineImage(o.Lines[0], f.s.listingArt()); got != "" {
		t.Fatalf("guess without listing art = %q, want none", got)
	}
	if _, err := f.store.SetListings(map[string]trades.Listing{
		f.older.ID: {AskBRL: 260, Listed: true, ImageURL: listingArtURL},
	}); err != nil {
		t.Fatal(err)
	}
	art := f.s.listingArt()
	if got := orderLineImage(o.Lines[0], art); got != listingArtURL {
		t.Fatalf("line image = %q, want listing art", got)
	}
	if got := publicView(o, time.Now(), art).Lines[0].ImageURL; got != listingArtURL {
		t.Fatalf("public image = %q", got)
	}
	o.Lines[0].ImageURL = "https://example.test/photo.jpg"
	if got := orderLineImage(o.Lines[0], art); got != "https://example.test/photo.jpg" {
		t.Fatalf("stored listing art replaced: %q", got)
	}
}

func TestOrderLineGuessKeptForOtherGames(t *testing.T) {
	l := orders.Line{Kind: "single", Game: "pokemon", Meta: "MEG · 086"}
	want := "/api/card-image?game=pokemon&number=086&set=MEG"
	if got := orderLineImage(l, nil); got != want {
		t.Fatalf("pokemon image = %q, want %q", got, want)
	}
}
