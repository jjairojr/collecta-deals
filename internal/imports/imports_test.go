package imports

import (
	"math"
	"testing"

	"opdeals/internal/compare"
	"opdeals/internal/game"
	"opdeals/internal/model"
	"opdeals/internal/tracking"
)

func near(a, b float64) bool { return math.Abs(a-b) < 1e-6 }

func TestBuildUsesDearestVariantAndUndercutsFloor(t *testing.T) {
	supply := []tracking.ImportSupply{
		{Set: "OGN", Number: "001", Name: "Jinx (001)", URL: "liga", Date: "2026-08-16", FloorBRL: 100, FloorCopies: 2, Supply: 7, Sellers: 6},
		{Set: "OGN", Number: "002", Name: "Vi (002)", FloorBRL: 5, Sellers: 1},
		{Set: "OGN", Number: "003", Name: "No US", FloorBRL: 50, Sellers: 3},
	}
	prices := []model.USPrice{
		{Number: "001/298", Name: "Jinx", SetCode: "OGN", LowUSD: 4, LiveUSD: 5, ProductID: 1, Rarity: "R"},
		{Number: "001/298", Name: "Jinx", SetCode: "OGN", LowUSD: 8, ProductID: 2, Rarity: "R", Variant: "Foil"},
		{Number: "002/298", Name: "Vi", SetCode: "OGN", LowUSD: 3, ProductID: 3},
	}
	got := Build(supply, prices, compare.MatcherFor(game.Riftbound()), BuildOptions{Game: "riftbound", FXRate: 0.2})
	if len(got) != 2 {
		t.Fatalf("want 2 deals, got %+v", got)
	}
	d := got[0]
	if d.ProductID != 2 || d.USVariants != 2 || d.CostUSD != 8 || d.USLive {
		t.Errorf("cost must come from the dearest (table-priced) variant: %+v", d)
	}
	if !near(d.CostBRL, 40) || !near(d.SellBRL, 99.99) || !near(d.ProfitBRL, 59.99) || !near(d.MarginPct, 59.99/40*100) {
		t.Errorf("math: cost=%.2f sell=%.2f profit=%.2f margin=%.2f", d.CostBRL, d.SellBRL, d.ProfitBRL, d.MarginPct)
	}
	if d.Game != "riftbound" || d.TrackingDate != "2026-08-16" || d.LigaURL != "liga" {
		t.Errorf("metadata: %+v", d)
	}
	if got[1].ProfitBRL >= 0 {
		t.Errorf("loss-making row still built (Apply gates it): %+v", got[1])
	}
}

func TestApplyFiltersAndSorts(t *testing.T) {
	deals := []Deal{
		{Game: "onepiece", Number: "A", Sellers: 6, CostUSD: 3, ProfitBRL: 10, MarginPct: 50},
		{Game: "onepiece", Number: "B", Sellers: 2, CostUSD: 9, ProfitBRL: 90, MarginPct: 10},
		{Game: "riftbound", Number: "C", Sellers: 9, CostUSD: 20, ProfitBRL: 40, MarginPct: 200},
	}
	got := Apply(deals, Filter{MinSellers: 6, MinUSD: 3, MinProfit: 10, SortBy: "profit"})
	if len(got) != 2 || got[0].Number != "C" || got[1].Number != "A" {
		t.Fatalf("profit sort with thresholds: %+v", got)
	}
	got = Apply(deals, Filter{SortBy: "margin", Limit: 1})
	if len(got) != 1 || got[0].Number != "C" {
		t.Errorf("margin sort + limit: %+v", got)
	}
	got = Apply(deals, Filter{SortBy: "profit", Games: []string{"onepiece"}})
	if len(got) != 2 || got[0].Number != "B" {
		t.Errorf("profit sort scoped to onepiece: %+v", got)
	}
}
