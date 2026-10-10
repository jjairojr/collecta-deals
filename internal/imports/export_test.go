package imports

import (
	"testing"

	"opdeals/internal/compare"
	"opdeals/internal/game"
	"opdeals/internal/model"
	"opdeals/internal/tracking"
)

func TestBuildExportUsesCheapestVariant(t *testing.T) {
	supply := []tracking.ImportSupply{
		{Set: "OGN", Number: "001", Name: "Jinx (001)", URL: "liga", Date: "2026-08-16", FloorBRL: 20, FloorCopies: 5, Supply: 9, Sellers: 4,
			FloorStores: []tracking.FloorStore{{StoreID: 1, StoreName: "A", Qty: 3}, {StoreID: 2, StoreName: "B", Qty: 2}}},
		{Set: "OGN", Number: "003", Name: "No US", FloorBRL: 50, Sellers: 3},
	}
	prices := []model.USPrice{
		{Number: "001/298", Name: "Jinx", SetCode: "OGN", LowUSD: 4, LiveUSD: 10, LiveListings: 6, LiveQty: 20, ProductID: 1, Rarity: "R"},
		{Number: "001/298", Name: "Jinx", SetCode: "OGN", LowUSD: 30, ProductID: 2, Variant: "Foil"},
	}
	got := BuildExport(supply, prices, compare.MatcherFor(game.Riftbound()), BuildOptions{Game: "riftbound", FXRate: 0.2})
	if len(got) != 1 {
		t.Fatalf("want 1 deal, got %+v", got)
	}
	d := got[0]
	if d.ProductID != 1 || d.USVariants != 2 || !d.USLive || d.USListings != 6 {
		t.Errorf("sell side must be the cheapest live variant: %+v", d)
	}
	if !near(d.CostUSD, 4) || !near(d.SellUSD, 9.99) || !near(d.ProfitUSD, 5.99) || !near(d.ProfitBRL, 29.95) || !near(d.MarginPct, 5.99/4*100) {
		t.Errorf("math: cost=%.2f sell=%.2f profitUSD=%.2f profitBRL=%.2f margin=%.2f", d.CostUSD, d.SellUSD, d.ProfitUSD, d.ProfitBRL, d.MarginPct)
	}
	if d.SuggestedQty != 5 || !near(d.Score, 29.95*5) || len(d.FloorStores) != 2 {
		t.Errorf("qty=%d score=%.2f stores=%d", d.SuggestedQty, d.Score, len(d.FloorStores))
	}
}

func TestApplyExportGatesAndSorts(t *testing.T) {
	deals := []ExportDeal{
		{Game: "onepiece", Number: "A", Sellers: 6, USListings: 3, SellUSD: 20, ProfitBRL: 10, MarginPct: 50, FloorCopies: 1, Score: 10},
		{Game: "onepiece", Number: "B", Sellers: 6, USListings: 0, SellUSD: 50, ProfitBRL: 90, MarginPct: 10, FloorCopies: 9, Score: 810},
		{Game: "riftbound", Number: "C", Sellers: 9, USListings: 8, SellUSD: 5, ProfitBRL: 40, MarginPct: 200, FloorCopies: 3, Score: 120},
	}
	got := ApplyExport(deals, ExportFilter{MinUSListings: 2, MinUSD: 3, MinProfit: 10, SortBy: "score"})
	if len(got) != 2 || got[0].Number != "C" || got[1].Number != "A" {
		t.Fatalf("table-priced row must be gated out, score order: %+v", got)
	}
	got = ApplyExport(deals, ExportFilter{SortBy: "copies", Limit: 1})
	if len(got) != 1 || got[0].Number != "B" {
		t.Errorf("copies sort + limit: %+v", got)
	}
	got = ApplyExport(deals, ExportFilter{SortBy: "margin", Games: []string{"onepiece"}})
	if len(got) != 2 || got[0].Number != "A" {
		t.Errorf("margin sort scoped: %+v", got)
	}
}
