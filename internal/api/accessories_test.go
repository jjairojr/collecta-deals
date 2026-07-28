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

// twoGameServer builds a server with two game ledgers plus the shared accessories
// one, which is the setup that proves accessories are not tied to a game.
func twoGameServer(t *testing.T) (*Server, *trades.Store, *trades.Store) {
	t.Helper()
	dir := t.TempDir()
	log := logx.New(io.Discard)
	op := trades.NewStore(filepath.Join(dir, "trades.json"), log)
	pkm := trades.NewStore(filepath.Join(dir, "trades-pkm.json"), log)
	acc := trades.NewStore(filepath.Join(dir, "accessories.json"), log)
	s := &Server{
		games: map[string]*GameStack{
			"onepiece": {Game: game.Game{ID: "onepiece", Name: "One Piece"}, Trades: op},
			"pokemon":  {Game: game.Game{ID: "pokemon", Name: "Pokémon"}, Trades: pkm},
		},
		defaultGame: "onepiece",
		accessories: acc,
	}
	return s, op, acc
}

func createTrade(t *testing.T, s *Server, gameID, body string) trades.Trade {
	t.Helper()
	rec := httptest.NewRecorder()
	s.handleTradesCreate(rec, httptest.NewRequest(http.MethodPost, "/api/trades?game="+gameID, strings.NewReader(body)))
	if rec.Code != http.StatusOK {
		t.Fatalf("create status = %d: %s", rec.Code, rec.Body.String())
	}
	var out trades.Trade
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func portfolioOf(t *testing.T, s *Server, gameID string) trades.PortfolioResponse {
	t.Helper()
	rec := httptest.NewRecorder()
	s.handleTradesList(rec, httptest.NewRequest(http.MethodGet, "/api/trades?game="+gameID, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d", rec.Code)
	}
	var resp trades.PortfolioResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	return resp
}

func TestAccessoryIsStoredOutsideTheGameLedger(t *testing.T) {
	s, op, acc := twoGameServer(t)

	created := createTrade(t, s, "onepiece", `{"kind":"accessory","name":"Sleeves Ultra Pro","set":"ACESSORIO","qty":3,"buyBRL":25,"manualBRL":45,"status":"holding"}`)

	inGame, err := op.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(inGame) != 0 {
		t.Fatalf("accessory leaked into the One Piece ledger: %+v", inGame)
	}
	inShared, err := acc.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(inShared) != 1 || inShared[0].ID != created.ID {
		t.Fatalf("shared ledger = %+v, want the created accessory", inShared)
	}
}

func TestAccessoryShowsUpUnderEveryGame(t *testing.T) {
	s, _, _ := twoGameServer(t)
	createTrade(t, s, "onepiece", `{"kind":"accessory","name":"Deckbox","qty":1,"buyBRL":30,"manualBRL":50,"status":"holding"}`)
	createTrade(t, s, "onepiece", `{"number":"OP16-080","name":"Teach","set":"OP16","qty":1,"buyBRL":100,"status":"holding"}`)

	for _, id := range []string{"onepiece", "pokemon"} {
		var names []string
		for _, tv := range portfolioOf(t, s, id).Trades {
			if tv.Kind == trades.KindAccessory {
				names = append(names, tv.Name)
			}
		}
		if len(names) != 1 || names[0] != "Deckbox" {
			t.Fatalf("%s portfolio accessories = %v, want [Deckbox]", id, names)
		}
	}
	// The card, in contrast, stays with its own game.
	if got := len(portfolioOf(t, s, "pokemon").Trades); got != 1 {
		t.Fatalf("pokemon portfolio = %d trades, want only the shared accessory", got)
	}
}

func TestAccessoryEditsResolveWithoutAGame(t *testing.T) {
	s, _, acc := twoGameServer(t)
	created := createTrade(t, s, "onepiece", `{"kind":"accessory","name":"Playmat","qty":1,"buyBRL":80,"manualBRL":120,"status":"holding"}`)

	// Edited from another game's screen: the id alone must find the record.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/trades/"+created.ID+"?game=pokemon", strings.NewReader(`{"kind":"accessory","name":"Playmat XL","qty":2,"buyBRL":80,"manualBRL":130,"status":"holding"}`))
	req.SetPathValue("id", created.ID)
	s.handleTradesUpdate(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("update status = %d: %s", rec.Code, rec.Body.String())
	}

	all, err := acc.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 || all[0].Name != "Playmat XL" || all[0].Qty != 2 {
		t.Fatalf("shared ledger after update = %+v", all)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/api/trades/"+created.ID+"?game=pokemon", nil)
	req.SetPathValue("id", created.ID)
	s.handleTradesDelete(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete status = %d: %s", rec.Code, rec.Body.String())
	}
	if all, _ := acc.List(); len(all) != 0 {
		t.Fatalf("accessory survived delete: %+v", all)
	}
}

func TestStorefrontPublishesAccessoriesWithoutAGame(t *testing.T) {
	s, op, acc := twoGameServer(t)
	card, _ := op.Add(trades.Trade{Number: "OP16-080", Name: "Teach", Set: "OP16", Qty: 1, BuyBRL: 100, Status: "holding"})
	sleeves, _ := acc.Add(trades.Trade{Kind: trades.KindAccessory, Name: "Sleeves", Set: "ACESSORIO", Qty: 5, BuyBRL: 25, Status: "holding"})
	op.SetListings(map[string]trades.Listing{card.ID: {AskBRL: 250, Listed: true}})
	acc.SetListings(map[string]trades.Listing{sleeves.ID: {AskBRL: 49.9, Listed: true}})

	rec := httptest.NewRecorder()
	s.handleStorefront(rec, httptest.NewRequest(http.MethodGet, "/api/storefront", nil))
	var resp storefrontResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Count != 2 {
		t.Fatalf("count = %d, want 2", resp.Count)
	}
	var found bool
	for _, it := range resp.Items {
		if it.Kind != trades.KindAccessory {
			continue
		}
		found = true
		if it.Game != accessoryGameID || it.GameLabel != accessoryGameLabel {
			t.Fatalf("accessory published under %q/%q", it.Game, it.GameLabel)
		}
		if it.AskBRL != 49.9 || it.Qty != 5 {
			t.Fatalf("unexpected accessory item: %+v", it)
		}
	}
	if !found {
		t.Fatal("accessory missing from the storefront")
	}
}

func TestListingsWriteToBothLedgers(t *testing.T) {
	s, op, acc := twoGameServer(t)
	card, _ := op.Add(trades.Trade{Number: "OP16-080", Name: "Teach", Set: "OP16", Qty: 1, BuyBRL: 100, Status: "holding"})
	sleeves, _ := acc.Add(trades.Trade{Kind: trades.KindAccessory, Name: "Sleeves", Qty: 5, BuyBRL: 25, Status: "holding"})

	body := `{"items":[{"id":"` + card.ID + `","askBRL":250,"listed":true},{"id":"` + sleeves.ID + `","askBRL":49.9,"listed":true,"featured":true}]}`
	rec := httptest.NewRecorder()
	s.handleTradesListings(rec, httptest.NewRequest(http.MethodPost, "/api/trades/listings?game=onepiece", strings.NewReader(body)))
	if rec.Code != http.StatusOK {
		t.Fatalf("listings status = %d: %s", rec.Code, rec.Body.String())
	}
	var out map[string]int
	json.Unmarshal(rec.Body.Bytes(), &out)
	if out["updated"] != 2 {
		t.Fatalf("updated = %d, want 2", out["updated"])
	}
	saved, _ := acc.List()
	if len(saved) != 1 || saved[0].AskBRL != 49.9 || !saved[0].Listed || !saved[0].Featured {
		t.Fatalf("accessory listing not saved: %+v", saved)
	}
}
