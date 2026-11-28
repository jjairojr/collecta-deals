package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"opdeals/internal/conversations"
	"opdeals/internal/metaevents"
	"opdeals/internal/orders"
	"opdeals/internal/trades"
)

type metaFix struct {
	*checkoutFixture
	ids map[string]string
}

func metaFixture(t *testing.T) *metaFix {
	t.Helper()
	f := &metaFix{checkoutFixture: checkoutServer(t), ids: map[string]string{}}
	f.s.PostSale.Conversations = conversations.NewStore(filepath.Join(t.TempDir(), "conversations.json"))
	now := time.Now()
	day := func(d int) string { return now.AddDate(0, 0, -d).In(saoPaulo).Format("2006-01-02") }
	mk := func(name string, status orders.Status, paid time.Time, ads *orders.AdsContext) {
		o, err := f.s.Checkout.Orders.Create(func(_ map[string]int, _ []orders.Order) (orders.Order, error) {
			return orders.Order{Ads: ads,
				Buyer:   orders.Buyer{Name: "Maria Souza", Email: "Maria@X.com", Phone: "(62) 99339-9980"},
				Address: orders.Address{City: "Goiânia", State: "GO", CEP: "74000-000"},
				Totals:  orders.Totals{TotalBRL: 650},
				Lines:   []orders.Line{{Qty: 2}}}, nil
		})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.s.Checkout.Orders.Update(o.ID, func(x *orders.Order) { x.Status, x.PaidAt = status, paid }); err != nil {
			t.Fatal(err)
		}
		f.ids[name] = o.ID
	}
	ua := &orders.AdsContext{UserAgent: "Mozilla/5.0 teste", FBP: "fb.1.1700000000.123"}
	mk("pago", orders.StatusPaid, now.Add(-2*time.Hour), ua)
	mk("enviado-sem-navegador", orders.StatusShipped, now.Add(-24*time.Hour), nil)
	mk("velho", orders.StatusDelivered, now.Add(-20*24*time.Hour), ua)
	mk("cancelado", orders.StatusCancelled, time.Time{}, ua)
	add := func(tr trades.Trade) {
		tr.Status, tr.SellCurrency, tr.Qty = "sold", "BRL", max(tr.Qty, 1)
		if _, err := f.store.Add(tr); err != nil {
			t.Fatal(err)
		}
	}
	add(trades.Trade{Number: "1", Name: "A", Set: "OP01", SellPrice: 100, SellDate: day(1), Buyer: "diego — Liga #111", Channel: "liga"})
	add(trades.Trade{Number: "2", Name: "B", Set: "OP01", SellPrice: 50, SellDate: day(1), Buyer: "diego — Liga #111", Channel: "liga"})
	add(trades.Trade{Number: "3", Name: "C", Set: "OP01", SellPrice: 70, SellDate: day(2), Buyer: "Liga #222", Channel: "liga"}) // sem conversa
	add(trades.Trade{Number: "4", Name: "D", Set: "OP01", SellPrice: 300, SellDate: day(0), Buyer: "João 62 98888-7777", Channel: "balcao"})
	add(trades.Trade{Number: "5", Name: "E", Set: "OP01", SellPrice: 90, SellDate: day(0), Buyer: "", Channel: "balcao"})
	add(trades.Trade{Number: "6", Name: "F", Set: "OP01", SellPrice: 0, SellDate: day(0), Buyer: "aberto", WriteOff: "aberto"})
	add(trades.Trade{Number: "7", Name: "G", Set: "OP01", SellPrice: 500, SellDate: day(60), Buyer: "Liga #333", Channel: "liga"})
	for code, phone := range map[string]string{"111": "5562977776666", "333": "5562955554444"} {
		if _, _, err := f.s.PostSale.Conversations.Create(func(_ []conversations.Conversation) (conversations.Conversation, bool) {
			return conversations.Conversation{Channel: conversations.ChannelLiga, Phone: phone, Name: "Diego", OrderID: "liga:" + code}, true
		}); err != nil {
			t.Fatal(err)
		}
	}
	return f
}

func metaGet(t *testing.T, f *metaFix, path string, out any) string {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("X-Admin-Token", "admin")
	f.s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("%s = %d: %s", path, rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), out); err != nil {
		t.Fatal(err)
	}
	return rec.Body.String()
}

func TestMetaEventsLastWeek(t *testing.T) {
	f := metaFixture(t)
	var resp metaEventsResponse
	body := metaGet(t, f, "/api/meta/events", &resp)
	byID := map[string]metaevents.Event{}
	for _, e := range resp.Events {
		byID[e.EventID] = e
	}
	if len(resp.Events) != 3 {
		t.Fatalf("eventos = %+v", resp.Events)
	}
	site := byID["pedido-"+f.ids["pago"]]
	if site.ActionSource != "website" || site.EventName != "Purchase" || site.CustomData.Value != 650 || site.EventSourceURL != "https://www.collectatcg.com.br/pedido/"+f.ids["pago"] ||
		site.UserData.UserAgent != "Mozilla/5.0 teste" || site.UserData.FBP != "fb.1.1700000000.123" || site.UserData.Email[0] != metaevents.Hash("maria@x.com") {
		t.Fatalf("site = %+v", site)
	}
	liga := byID["liga-111"]
	if liga.ActionSource != "other" || liga.CustomData.Value != 150 || liga.CustomData.NumItems != 2 || liga.UserData.Phone[0] != metaevents.Hash("5562977776666") {
		t.Fatalf("liga = %+v", liga)
	}
	balcao := resp.Events[0]
	for _, e := range resp.Events {
		if e.ActionSource == "physical_store" {
			balcao = e
		}
	}
	if balcao.ActionSource != "physical_store" || balcao.CustomData.Value != 300 || balcao.UserData.Phone[0] != metaevents.Hash("5562988887777") {
		t.Fatalf("balcão = %+v", balcao)
	}
	if resp.Skipped.SiteSemNavegador != 1 || resp.Skipped.LigaSemTelefone != 1 || resp.Skipped.BalcaoSemContato != 1 {
		t.Fatalf("fora = %+v", resp.Skipped)
	}
	for _, raw := range []string{"Maria", "maria@", "99339", "Diego", "977776666", "João", "98888", "Goi"} {
		if strings.Contains(body, raw) {
			t.Errorf("dado de cliente em texto na resposta: %q", raw)
		}
	}
	for _, e := range resp.Events {
		if time.Since(time.Unix(e.EventTime, 0)) > 7*24*time.Hour || e.EventTime > time.Now().Unix() {
			t.Errorf("%s fora da janela de 7 dias: %v", e.EventID, time.Unix(e.EventTime, 0))
		}
	}
}

func TestMetaCustomersHistory(t *testing.T) {
	f := metaFixture(t)
	var resp metaCustomersResponse
	body := metaGet(t, f, "/api/meta/customers?days=180", &resp)
	// Maria (site, 3 pedidos, uma pessoa só), Diego da Liga #111 e #333 (dois
	// telefones) e o João do balcão.
	if len(resp.Data) != 4 || resp.Counts.Site != 1 || resp.Counts.Liga != 2 || resp.Counts.Balcao != 1 {
		t.Fatalf("lista = %d linhas, %+v", len(resp.Data), resp.Counts)
	}
	if strings.Join(resp.Schema, ",") != "EMAIL,PHONE,FN,LN,CT,ST,ZIP,COUNTRY" {
		t.Fatalf("schema = %v", resp.Schema)
	}
	for _, raw := range []string{"Maria", "maria@", "Diego", "João"} {
		if strings.Contains(body, raw) {
			t.Errorf("dado de cliente em texto: %q", raw)
		}
	}
}

func TestMetaRoutesNeedAdmin(t *testing.T) {
	f := metaFixture(t)
	for _, path := range []string{"/api/meta/events", "/api/meta/customers"} {
		rec := httptest.NewRecorder()
		f.s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusUnauthorized && rec.Code != http.StatusForbidden {
			t.Errorf("%s sem token = %d", path, rec.Code)
		}
	}
}

func TestOrderKeepsAdsContext(t *testing.T) {
	a := (&orders.AdsContext{UserAgent: " UA ", FBP: "fb.1.2.3", FBC: "lixo"}).Clean()
	if a == nil || a.UserAgent != "UA" || a.FBP != "fb.1.2.3" || a.FBC != "" {
		t.Fatalf("Clean = %+v", a)
	}
	if (&orders.AdsContext{FBC: "x"}).Clean() != nil {
		t.Fatal("sem nada válido devia ser nil")
	}
}
