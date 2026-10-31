package api

import (
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"testing"

	"opdeals/internal/logx"
	"opdeals/internal/orders"
	"opdeals/internal/syncqueue"
	"opdeals/internal/trades"
)

func moveTrade(t *testing.T, s *Server, id, body string) *http.Response {
	t.Helper()
	rec := queueCall(t, http.MethodPost, "/api/trades/"+id+"/move", id, body, s.handleTradesMove)
	return rec.Result()
}

func TestMoveCarriesTheRecordToTheOtherGame(t *testing.T) {
	s, op, _ := twoGameServer(t)
	pkm := s.games["pokemon"].Trades
	item := createTrade(t, s, "onepiece", `{"kind":"sealed","name":"Double Pack 30 Anos","qty":19,"buyBRL":66.49,"shippingBRL":10,"store":"COPAG","buyDate":"2026-10-22","purchaseID":"p1","notes":"x","photos":["/a.jpg"],"status":"holding"}`)

	res := moveTrade(t, s, item.ID, `{"from":"onepiece","to":"pokemon"}`)
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("move status = %d: %s", res.StatusCode, body)
	}
	var out trades.Trade
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.ID != item.ID {
		t.Fatalf("id = %q, want %q", out.ID, item.ID)
	}
	if _, ok := ledgerTrades(t, op)[item.ID]; ok {
		t.Fatal("item still in the source ledger")
	}
	got, ok := ledgerTrades(t, pkm)[item.ID]
	if !ok {
		t.Fatal("item missing from the target ledger")
	}
	if got.Slug != item.Slug || !got.CreatedAt.Equal(item.CreatedAt) || got.PurchaseID != "p1" || got.Kind != "sealed" ||
		got.Qty != 19 || got.BuyBRL != 66.49 || got.ShippingBRL != 10 || got.Notes != "x" || len(got.Photos) != 1 {
		t.Fatalf("moved record lost fields: %+v", got)
	}
}

func TestMoveTakesASoldLot(t *testing.T) {
	s, op, _ := twoGameServer(t)
	pkm := s.games["pokemon"].Trades
	item := createTrade(t, s, "onepiece", `{"kind":"sealed","name":"Coleção com Pôster","qty":3,"buyBRL":110,"status":"holding"}`)
	sellTrade(t, s, item.ID, `{"qty":1,"sellPrice":140,"channel":"balcao","buyer":"Fulano"}`)
	sold := soldIDOther(t, op, item.ID)

	if res := moveTrade(t, s, sold, `{"from":"onepiece","to":"pokemon"}`); res.StatusCode != http.StatusOK {
		t.Fatalf("move sold status = %d", res.StatusCode)
	}
	got := ledgerTrades(t, pkm)[sold]
	if got.Status != "sold" || got.SellPrice != 140 || got.Buyer != "Fulano" || got.Channel != "balcao" || got.SplitFrom != item.ID {
		t.Fatalf("sale fields lost on move: %+v", got)
	}
	if _, ok := ledgerTrades(t, op)[item.ID]; !ok {
		t.Fatal("the holding lot moved with the sale")
	}
}

func TestMoveRefusals(t *testing.T) {
	cases := []struct {
		name string
		body string
		want int
	}{
		{"same game", `{"from":"onepiece","to":"onepiece"}`, http.StatusBadRequest},
		{"unknown target", `{"from":"onepiece","to":"magic"}`, http.StatusBadRequest},
		{"unknown source", `{"from":"magic","to":"pokemon"}`, http.StatusBadRequest},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s, _, _ := twoGameServer(t)
			item := createTrade(t, s, "onepiece", `{"name":"Teach","number":"OP16-080","qty":1,"buyBRL":10,"status":"holding"}`)
			if res := moveTrade(t, s, item.ID, c.body); res.StatusCode != c.want {
				t.Fatalf("status = %d, want %d", res.StatusCode, c.want)
			}
		})
	}

	t.Run("not found", func(t *testing.T) {
		s, _, _ := twoGameServer(t)
		if res := moveTrade(t, s, "nope", `{"from":"onepiece","to":"pokemon"}`); res.StatusCode != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", res.StatusCode)
		}
	})
}

func TestMoveRefusesAnAccessory(t *testing.T) {
	s, _, acc := twoGameServer(t)
	item := createTrade(t, s, "onepiece", `{"kind":"accessory","name":"Playmat","qty":1,"buyBRL":80,"status":"holding"}`)
	if res := moveTrade(t, s, item.ID, `{"from":"onepiece","to":"pokemon"}`); res.StatusCode != http.StatusConflict {
		t.Fatalf("status = %d, want 409", res.StatusCode)
	}
	if _, ok := ledgerTrades(t, acc)[item.ID]; !ok {
		t.Fatal("accessory left the shared ledger")
	}
}

func TestMoveRefusesAListedHolding(t *testing.T) {
	mark := map[string]func(*trades.Store, string) error{
		"vitrine": func(st *trades.Store, id string) error {
			_, err := st.SetListings(map[string]trades.Listing{id: {AskBRL: 20, Listed: true}})
			return err
		},
		"liga": func(st *trades.Store, id string) error {
			_, err := st.SetLigaState(map[string]trades.LigaState{id: {Listed: true, Qty: 1, PriceBRL: 20}})
			return err
		},
	}
	for name, set := range mark {
		t.Run(name, func(t *testing.T) {
			s, op, _ := twoGameServer(t)
			item := createTrade(t, s, "onepiece", `{"name":"Teach","qty":1,"buyBRL":10,"status":"holding"}`)
			if err := set(op, item.ID); err != nil {
				t.Fatal(err)
			}
			if res := moveTrade(t, s, item.ID, `{"from":"onepiece","to":"pokemon"}`); res.StatusCode != http.StatusConflict {
				t.Fatalf("status = %d, want 409", res.StatusCode)
			}
			if _, ok := ledgerTrades(t, op)[item.ID]; !ok {
				t.Fatal("refused item left the source ledger")
			}
		})
	}
}

func TestMoveRefusesAnItemOnAnOpenSiteOrder(t *testing.T) {
	s, _, _ := twoGameServer(t)
	s.Checkout.Orders = orders.NewStore(filepath.Join(t.TempDir(), "orders.json"), logx.New(io.Discard))
	item := createTrade(t, s, "onepiece", `{"name":"Teach","number":"OP16-080","qty":1,"buyBRL":10,"status":"holding"}`)
	if _, err := s.Checkout.Orders.Create(func(map[string]int, []orders.Order) (orders.Order, error) {
		return orders.Order{Lines: []orders.Line{{SKU: trades.SKU("onepiece", item), Game: "onepiece", Name: "Teach", Qty: 1}}}, nil
	}); err != nil {
		t.Fatal(err)
	}
	if res := moveTrade(t, s, item.ID, `{"from":"onepiece","to":"pokemon"}`); res.StatusCode != http.StatusConflict {
		t.Fatalf("status = %d, want 409", res.StatusCode)
	}
}

func TestMoveRefusesAnItemWithAnOpenSyncTask(t *testing.T) {
	s, _, _ := twoGameServer(t)
	s.SyncQueue = syncqueue.NewStore(filepath.Join(t.TempDir(), "syncqueue.json"))
	item := createTrade(t, s, "onepiece", `{"name":"Teach","number":"OP16-080","qty":1,"buyBRL":10,"status":"holding"}`)
	if _, err := s.SyncQueue.Add(syncqueue.Task{Game: "onepiece", TradeID: item.ID, Name: "Teach", Target: syncqueue.TargetLiga, Action: syncqueue.ActionRemove, Qty: 1}); err != nil {
		t.Fatal(err)
	}
	if res := moveTrade(t, s, item.ID, `{"from":"onepiece","to":"pokemon"}`); res.StatusCode != http.StatusConflict {
		t.Fatalf("status = %d, want 409", res.StatusCode)
	}
}
