package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"opdeals/internal/trades"
)

func sellTrade(t *testing.T, s *Server, id, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/trades/"+id+"/sell?game=onepiece", strings.NewReader(body))
	req.SetPathValue("id", id)
	rec := httptest.NewRecorder()
	s.handleTradesSell(rec, req)
	return rec
}

func TestSellStoresChannelAndWriteOff(t *testing.T) {
	s, _, _ := twoGameServer(t)
	item := createTrade(t, s, "onepiece", `{"name":"Booster Box","kind":"sealed","qty":5,"buyBRL":300,"status":"holding"}`)

	rec := sellTrade(t, s, item.ID, `{"qty":2,"sellPrice":800,"sellCurrency":"BRL","sellDate":"2026-10-30","buyer":"Fulano","channel":"balcao"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("sell status = %d: %s", rec.Code, rec.Body.String())
	}
	var sold trades.Trade
	if err := json.Unmarshal(rec.Body.Bytes(), &sold); err != nil {
		t.Fatal(err)
	}
	if sold.Channel != trades.ChannelBalcao || sold.Buyer != "Fulano" {
		t.Fatalf("sold = channel %q buyer %q", sold.Channel, sold.Buyer)
	}

	rec = sellTrade(t, s, item.ID, `{"qty":1,"sellCurrency":"BRL","sellDate":"2026-10-30","writeOff":"aberto"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("write-off status = %d: %s", rec.Code, rec.Body.String())
	}
	var written trades.Trade
	if err := json.Unmarshal(rec.Body.Bytes(), &written); err != nil {
		t.Fatal(err)
	}
	if written.WriteOff != trades.WriteOffAberto || written.SellPrice != 0 {
		t.Fatalf("write-off = %q price %v", written.WriteOff, written.SellPrice)
	}
}

func TestSellRejectsUnknownChannelOrWriteOff(t *testing.T) {
	s, _, _ := twoGameServer(t)
	item := createTrade(t, s, "onepiece", `{"name":"Booster Box","kind":"sealed","qty":5,"buyBRL":300,"status":"holding"}`)
	if rec := sellTrade(t, s, item.ID, `{"qty":1,"sellPrice":1,"channel":"shopee"}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("unknown channel = %d, want 400", rec.Code)
	}
	if rec := sellTrade(t, s, item.ID, `{"qty":1,"writeOff":"roubo"}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("unknown write-off = %d, want 400", rec.Code)
	}
}

func TestCreateSoldTradeIsMarkedSemLastro(t *testing.T) {
	s, _, _ := twoGameServer(t)
	sold := createTrade(t, s, "onepiece", `{"name":"Luffy","number":"OP01-001","qty":1,"status":"sold","sellPrice":50,"sellDate":"2026-10-30","buyer":"Fulano — Liga #1","channel":"liga"}`)
	if !sold.SemLastro {
		t.Fatal("sold trade created without a lot must be semLastro")
	}
	held := createTrade(t, s, "onepiece", `{"name":"Zoro","number":"OP01-025","qty":1,"buyBRL":10,"status":"holding"}`)
	if held.SemLastro {
		t.Fatal("holding must not be semLastro")
	}
}
