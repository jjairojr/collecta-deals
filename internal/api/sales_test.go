package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"opdeals/internal/trades"
)

func salesOf(t *testing.T, s *Server) []saleRow {
	t.Helper()
	rec := httptest.NewRecorder()
	s.handleSales(rec, httptest.NewRequest(http.MethodGet, "/api/sales", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("sales status = %d: %s", rec.Code, rec.Body.String())
	}
	var resp salesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	return resp.Sales
}

func TestSalesSpanEveryGameAndSkipHoldings(t *testing.T) {
	s, op, _ := twoGameServer(t)
	pkm := s.games["pokemon"].Trades

	op.Add(trades.Trade{Number: "OP15-119", Name: "Luffy", Set: "OP15", Qty: 2, BuyBRL: 50, Status: "sold", SellPrice: 310, SellCurrency: "BRL", SellDate: "2026-08-03", Buyer: "diego — Liga #11576384"})
	op.Add(trades.Trade{Number: "OP16-080", Name: "Teach", Set: "OP16", Qty: 1, BuyBRL: 100, Status: "holding"})
	pkm.Add(trades.Trade{Number: "107", Name: "Samurott", Set: "SSP", Qty: 1, BuyBRL: 118.63, Status: "sold", SellPrice: 175, SellCurrency: "BRL", SellDate: "2026-07-30", Buyer: "Liga #11541716"})

	got := salesOf(t, s)
	if len(got) != 2 {
		t.Fatalf("sales = %d rows, want the 2 sold trades: %+v", len(got), got)
	}
	// Newest first, and each row says which game it came from.
	if got[0].Game != "onepiece" || got[0].Name != "Luffy" {
		t.Fatalf("first sale = %+v, want the One Piece one (2026-08-03)", got[0])
	}
	if got[1].Game != "pokemon" || got[1].GameName != "Pokémon" {
		t.Fatalf("second sale = %+v, want the Pokémon one", got[1])
	}
	// CostBRL and ProfitBRL come from the shared valuation, so a package's profit
	// is just a sum of these.
	if got[0].CostBRL != 100 || got[0].ValueBRL != 310 || got[0].ProfitBRL != 210 {
		t.Fatalf("valuation = cost %v value %v profit %v", got[0].CostBRL, got[0].ValueBRL, got[0].ProfitBRL)
	}
}

// An accessory sale lives in the shared ledger, which handleTradesList appends to
// every game. Counting it once per game would multiply the revenue by the number
// of games.
func TestSoldAccessoryIsCountedOnce(t *testing.T) {
	s, _, acc := twoGameServer(t)
	acc.Add(trades.Trade{Kind: trades.KindAccessory, Name: "Sleeves", Qty: 1, BuyBRL: 25, Status: "sold", SellPrice: 49.9, SellCurrency: "BRL", SellDate: "2026-08-05"})

	got := salesOf(t, s)
	if len(got) != 1 {
		t.Fatalf("sales = %d rows, want 1 (one accessory, two games): %+v", len(got), got)
	}
	if got[0].Game != "acessorios" {
		t.Fatalf("accessory sale filed under %q", got[0].Game)
	}
}

func TestSalesConvertUSDProceeds(t *testing.T) {
	s, op, _ := twoGameServer(t)
	op.Add(trades.Trade{Number: "OP13-031", Name: "Law", Set: "OP13", Qty: 1, BuyBRL: 695, Status: "sold", SellPrice: 100, SellCurrency: "USD", SellDate: "2026-07-27"})

	got := salesOf(t, s)
	if len(got) != 1 {
		t.Fatalf("sales = %d rows, want 1", len(got))
	}
	// No deals snapshot means fx = 1, so a USD sale keeps its face value instead of
	// silently reading as zero.
	if got[0].ValueBRL != 100 {
		t.Fatalf("USD proceeds = %v, want 100", got[0].ValueBRL)
	}
}
