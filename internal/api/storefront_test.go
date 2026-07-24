package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"opdeals/internal/game"
	"opdeals/internal/logx"
	"opdeals/internal/trades"
)

func storefrontServer(t *testing.T) (*Server, *trades.Store) {
	t.Helper()
	store := trades.NewStore(filepath.Join(t.TempDir(), "trades.json"), logx.New(io.Discard))
	gs := &GameStack{Game: game.Game{ID: "onepiece", Name: "One Piece"}, Trades: store}
	s := &Server{games: map[string]*GameStack{"onepiece": gs}, defaultGame: "onepiece"}
	return s, store
}

func TestStorefrontShowsOnlyListedHoldings(t *testing.T) {
	s, store := storefrontServer(t)

	listed, err := store.Add(trades.Trade{Number: "OP16-080", Name: "Teach", Set: "OP16", Qty: 2, BuyBRL: 100, Status: "holding"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.SetListings(map[string]trades.Listing{listed.ID: {AskBRL: 250, Listed: true}}); err != nil {
		t.Fatal(err)
	}
	// Unlisted holding, a sold card, and a listed-but-priceless card must all be hidden.
	if _, err := store.Add(trades.Trade{Number: "OP01-001", Name: "Roronoa", Set: "OP01", Qty: 1, BuyBRL: 50, Status: "holding"}); err != nil {
		t.Fatal(err)
	}
	sold, _ := store.Add(trades.Trade{Number: "OP02-013", Name: "Ace", Set: "OP02", Qty: 1, BuyBRL: 80, Status: "sold"})
	store.SetListings(map[string]trades.Listing{sold.ID: {AskBRL: 500, Listed: true}})

	rec := httptest.NewRecorder()
	s.handleStorefront(rec, httptest.NewRequest(http.MethodGet, "/api/storefront", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	var resp storefrontResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Count != 1 || len(resp.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", resp.Count)
	}
	it := resp.Items[0]
	if it.Number != "OP16-080" || it.AskBRL != 250 || it.Qty != 2 || it.GameLabel != "One Piece" {
		t.Fatalf("unexpected item: %+v", it)
	}
}

func TestStorefrontNeverLeaksCost(t *testing.T) {
	s, store := storefrontServer(t)
	held, _ := store.Add(trades.Trade{Number: "OP16-080", Name: "Teach", Set: "OP16", Qty: 1, BuyBRL: 999, ShippingBRL: 50, Status: "holding"})
	store.SetListings(map[string]trades.Listing{held.ID: {AskBRL: 250, Listed: true}})

	rec := httptest.NewRecorder()
	s.handleStorefront(rec, httptest.NewRequest(http.MethodGet, "/api/storefront", nil))
	body := rec.Body.String()

	for _, banned := range []string{"buyBRL", "costBRL", "profit", "shippingBRL", "999", "marginPct"} {
		if strings.Contains(body, banned) {
			t.Fatalf("storefront leaked %q in body: %s", banned, body)
		}
	}
}
