package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"opdeals/internal/trades"
)

func putTrade(t *testing.T, s *Server, id, gameID, body string) (int, trades.Trade) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPut, "/api/trades/"+id+"?game="+gameID, strings.NewReader(body))
	req.SetPathValue("id", id)
	rec := httptest.NewRecorder()
	s.handleTradesUpdate(rec, req)
	var out trades.Trade
	if rec.Code == http.StatusOK {
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
	}
	return rec.Code, out
}

func TestUpdateWithPartialBodyKeepsOtherFields(t *testing.T) {
	s, _, _ := purchaseServer(t)
	order := newOrder(t, s, `{"date":"2026-10-22","store":"Copag"}`)
	item := addItem(t, s, order.ID, "onepiece", `{"number":"OP16-080","name":"Teach","set":"OP16","qty":2,"buyBRL":100}`)

	if rec := sellTrade(t, s, item.ID, `{"sellPrice":300,"sellCurrency":"BRL","sellDate":"2026-10-25","buyer":"Fulano — Liga #1"}`); rec.Code != http.StatusOK {
		t.Fatalf("sell = %d", rec.Code)
	}
	full := `{"number":"OP16-080","name":"Teach","set":"OP16","qty":2,"buyBRL":100,"askBRL":150,"listed":true,` +
		`"ligaListed":true,"ligaQty":2,"ligaPriceBRL":149.99,"status":"sold","sellPrice":300,"sellCurrency":"BRL",` +
		`"sellDate":"2026-10-25","buyer":"Fulano — Liga #1","notes":"vitrine","photos":["https://x/a.jpg"]}`
	if code, _ := putTrade(t, s, item.ID, "onepiece", full); code != http.StatusOK {
		t.Fatalf("full update status = %d", code)
	}

	code, got := putTrade(t, s, item.ID, "onepiece", `{"qty":3}`)
	if code != http.StatusOK {
		t.Fatalf("partial update status = %d", code)
	}
	if got.Qty != 3 {
		t.Fatalf("qty = %d, want 3", got.Qty)
	}
	if got.Name != "Teach" || got.Number != "OP16-080" || got.BuyBRL != 100 {
		t.Fatalf("identity lost: %+v", got)
	}
	if !got.LigaListed || got.LigaQty != 2 || got.LigaPriceBRL != 149.99 {
		t.Fatalf("liga state lost: %+v", got)
	}
	if !got.Listed || got.AskBRL != 150 {
		t.Fatalf("storefront state lost: %+v", got)
	}
	if got.Status != "sold" || got.SellPrice != 300 || got.Buyer != "Fulano — Liga #1" || got.SellDate != "2026-10-25" {
		t.Fatalf("sale lost: %+v", got)
	}
	if got.Notes != "vitrine" || len(got.Photos) != 1 || got.PurchaseID != order.ID {
		t.Fatalf("notes/photos/purchase lost: %+v", got)
	}
	if got.ID != item.ID || !got.CreatedAt.Equal(item.CreatedAt) || got.Slug != item.Slug {
		t.Fatalf("id/createdAt/slug changed: %+v", got)
	}
}

func TestUpdateIgnoresComputedFieldsAndNormalizesNumber(t *testing.T) {
	s, _, _ := purchaseServer(t)
	order := newOrder(t, s, `{"date":"2026-10-22","store":"Copag"}`)
	item := addItem(t, s, order.ID, "onepiece", `{"number":"OP16-080","name":"Teach","set":"OP16","qty":1,"buyBRL":100}`)

	code, got := putTrade(t, s, item.ID, "onepiece", `{"number":" op16-080 ","costBRL":999,"valueBRL":1,"realized":true,"id":"forged"}`)
	if code != http.StatusOK {
		t.Fatalf("update status = %d", code)
	}
	if got.ID != item.ID {
		t.Fatalf("id = %q, want %q", got.ID, item.ID)
	}
	if want := item.Number; got.Number != want {
		t.Fatalf("number = %q, want normalized %q", got.Number, want)
	}
}

func TestUpdateRejectsMalformedBody(t *testing.T) {
	s, _, _ := purchaseServer(t)
	order := newOrder(t, s, `{"date":"2026-10-22","store":"Copag"}`)
	item := addItem(t, s, order.ID, "onepiece", `{"number":"OP16-080","name":"Teach","set":"OP16","qty":1,"buyBRL":100}`)

	if code, _ := putTrade(t, s, item.ID, "onepiece", `{"qty":"três"}`); code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", code)
	}
}
