package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"opdeals/internal/trades"
)

func TestLigaFlagWritesToBothLedgers(t *testing.T) {
	s, op, acc := twoGameServer(t)
	card, _ := op.Add(trades.Trade{Number: "OP16-080", Name: "Teach", Set: "OP16", Qty: 2, BuyBRL: 100, Status: "holding"})
	sleeves, _ := acc.Add(trades.Trade{Kind: trades.KindAccessory, Name: "Sleeves", Qty: 5, BuyBRL: 25, Status: "holding"})

	body := `{"items":[{"id":"` + card.ID + `","ligaListed":true,"ligaQty":2,"ligaPriceBRL":250},` +
		`{"id":"` + sleeves.ID + `","ligaListed":true,"ligaQty":5,"ligaPriceBRL":49.9}]}`
	rec := httptest.NewRecorder()
	s.handleTradesLiga(rec, httptest.NewRequest(http.MethodPost, "/api/trades/liga?game=onepiece", strings.NewReader(body)))
	if rec.Code != http.StatusOK {
		t.Fatalf("liga status = %d: %s", rec.Code, rec.Body.String())
	}
	var out map[string]int
	json.Unmarshal(rec.Body.Bytes(), &out)
	if out["updated"] != 2 {
		t.Fatalf("updated = %d, want 2", out["updated"])
	}

	saved, _ := acc.List()
	if len(saved) != 1 || !saved[0].LigaListed || saved[0].LigaQty != 5 || saved[0].LigaPriceBRL != 49.9 {
		t.Fatalf("accessory liga state not saved: %+v", saved)
	}
	cards, _ := op.List()
	if len(cards) != 1 || !cards[0].LigaListed || cards[0].LigaQty != 2 {
		t.Fatalf("card liga state not saved: %+v", cards)
	}
}

// The Liga flag and the storefront listing are different columns of the same
// ledger, written by different endpoints. Marking one must never disturb the
// other — that separation is what keeps a flag toggle from wiping a price.
func TestLigaFlagDoesNotTouchTheAskingPrice(t *testing.T) {
	s, op, _ := twoGameServer(t)
	card, _ := op.Add(trades.Trade{Number: "OP16-080", Name: "Teach", Set: "OP16", Qty: 1, BuyBRL: 100, Status: "holding"})

	rec := httptest.NewRecorder()
	s.handleTradesListings(rec, httptest.NewRequest(http.MethodPost, "/api/trades/listings?game=onepiece",
		strings.NewReader(`{"items":[{"id":"`+card.ID+`","askBRL":250,"listed":true}]}`)))
	if rec.Code != http.StatusOK {
		t.Fatalf("listings status = %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	s.handleTradesLiga(rec, httptest.NewRequest(http.MethodPost, "/api/trades/liga?game=onepiece",
		strings.NewReader(`{"items":[{"id":"`+card.ID+`","ligaListed":true,"ligaQty":1,"ligaPriceBRL":250}]}`)))
	if rec.Code != http.StatusOK {
		t.Fatalf("liga status = %d", rec.Code)
	}

	saved, _ := op.List()
	if saved[0].AskBRL != 250 || !saved[0].Listed {
		t.Fatalf("marking the Liga flag clobbered the listing: %+v", saved[0])
	}
	if !saved[0].LigaListed {
		t.Fatalf("liga flag not set: %+v", saved[0])
	}
}
