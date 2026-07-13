package tracking

import (
	"testing"

	"opdeals/internal/game"
)

var opCfg = MarketConfig{Langs: []string{"2"}, MainTokens: game.OnePiece().MainCharTokens}

func nmPriced(id int, name string, qty int, price float64) StoreQty {
	return StoreQty{StoreID: id, StoreName: name, Condition: conditionNM, Language: "2", Quantity: qty, Known: true, PriceBRL: price, PriceKnown: true}
}

func priced(id int, qty int, price float64, cond, lang string) StoreQty {
	return StoreQty{StoreID: id, StoreName: "S", Condition: cond, Language: lang, Quantity: qty, Known: true, PriceBRL: price, PriceKnown: true}
}

func TestBuyoutShippingCost(t *testing.T) {
	day := day("2026-07-03",
		CardDay{Number: "SHIP", Name: "Ship", LowBRL: 25, Stores: []StoreQty{
			nmPriced(1, "A", 1, 25), nmPriced(2, "B", 1, 25), nmPriced(3, "C", 1, 25),
			nmPriced(4, "D", 5, 60),
		}},
	)
	res := Buyout(day, 200, 20, 15, false, false, "score", 10, opCfg)
	if len(res) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(res))
	}
	c := res[0]
	// 3 copies at R$25 across 3 stores: cards R$75 + 3×R$15 shipping = R$45.
	if c.StoreCount != 3 || c.ShippingCost != 45 || c.BuyoutCost != 75 {
		t.Errorf("stores=%d ship=%.0f cards=%.0f, want 3/45/75", c.StoreCount, c.ShippingCost, c.BuyoutCost)
	}
	// profit = 3×60 - 75 cards - 45 shipping = 60
	if c.ProfitBRL != 60 {
		t.Errorf("profit = %.0f, want 60 (net of shipping)", c.ProfitBRL)
	}
}

func TestBuyoutMainCharactersFilter(t *testing.T) {
	mk := func(number, name string) CardDay {
		return CardDay{Number: number, Name: name, LowBRL: 30, Stores: []StoreQty{
			nmPriced(1, "A", 1, 30), nmPriced(2, "B", 3, 70),
		}}
	}
	day := day("2026-07-03",
		mk("A1", "Monkey.D.Luffy (Alternate Art)"),
		mk("A2", "Trafalgar Law (Parallel)"),
		mk("A3", "Boa Hancock"),
		mk("A4", "Nico Robin"),
		mk("A5", "Charlotte Katakuri"),
		mk("A6", "Kaido"),
	)
	res := Buyout(day, 500, 20, 0, true, false, "score", 40, opCfg)
	got := map[string]bool{}
	for _, c := range res {
		got[c.Number] = true
	}
	for _, n := range []string{"A1", "A2", "A3", "A4"} {
		if !got[n] {
			t.Errorf("expected main character %s to be included", n)
		}
	}
	for _, n := range []string{"A5", "A6"} {
		if got[n] {
			t.Errorf("non-main character %s should be excluded", n)
		}
	}
}

func TestBuyoutEnglishOnly(t *testing.T) {
	day := day("2026-07-03",
		// cheapest copy is Japanese (idioma 6) at R$10 and must NOT set the floor;
		// English NM floor is R$30, next English level R$70.
		CardDay{Number: "LANG", Name: "Lang", LowBRL: 10, Stores: []StoreQty{
			priced(1, 1, 10, conditionNM, "6"),
			nmPriced(2, "B", 1, 30),
			nmPriced(3, "C", 3, 70),
		}},
	)
	res := Buyout(day, 500, 20, 0, false, false, "score", 10, opCfg)
	if len(res) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(res))
	}
	if res[0].Floor != 30 || res[0].NextFloor != 70 {
		t.Errorf("floor=%.0f next=%.0f, want 30/70 (Japanese R$10 ignored)", res[0].Floor, res[0].NextFloor)
	}
}

func TestBuyoutThinWallBeatsThickTail(t *testing.T) {
	day := day("2026-07-02",
		// thin wall: 2 copies at R$25, then jumps to R$60
		CardDay{Number: "THIN", Name: "Thin", LowBRL: 25, Stores: []StoreQty{
			nmPriced(1, "A", 1, 25), nmPriced(2, "B", 1, 25),
			nmPriced(3, "C", 5, 60), nmPriced(4, "D", 5, 60),
		}},
		// thick tail: 40 copies clustered at R$25
		CardDay{Number: "THICK", Name: "Thick", LowBRL: 25, Stores: []StoreQty{
			nmPriced(5, "E", 20, 25), nmPriced(6, "F", 20, 25),
			nmPriced(7, "G", 5, 60),
		}},
	)
	res := Buyout(day, 200, 20, 0, false, false, "score", 10, opCfg)
	if len(res) == 0 {
		t.Fatal("expected candidates")
	}
	// THIN: clear 2×25 = R$50 (within 200) -> floor 25->60 = +140%
	// THICK: clear 40×25 = R$1000 > 200 budget -> can't lift -> excluded/low
	if res[0].Number != "THIN" {
		t.Errorf("top candidate = %s, want THIN", res[0].Number)
	}
	thin := res[0]
	if thin.NextFloor != 60 || thin.BuyoutCost != 50 || thin.CopiesToClear != 2 {
		t.Errorf("thin: next=%.0f cost=%.0f copies=%d, want 60/50/2", thin.NextFloor, thin.BuyoutCost, thin.CopiesToClear)
	}
	if thin.LiftPct < 139 || thin.LiftPct > 141 {
		t.Errorf("thin lift = %.1f, want ~140", thin.LiftPct)
	}
	for _, c := range res {
		if c.Number == "THICK" {
			t.Errorf("THICK should be excluded (budget can't clear the R$25 wall), got %+v", c)
		}
	}
}

func TestSnipeGapAndProfit(t *testing.T) {
	day := day("2026-07-05",
		// snipe: seller A dumped 3 copies at R$10, next price is R$50 -> +400% gap.
		CardDay{Number: "SNIPE", Name: "Luffy", LowBRL: 10, Stores: []StoreQty{
			nmPriced(1, "A", 3, 10), nmPriced(2, "B", 2, 50),
		}},
		// only a +20% gap -> excluded when minGap is 100%.
		CardDay{Number: "FLAT", Name: "Flat", LowBRL: 30, Stores: []StoreQty{
			nmPriced(3, "C", 1, 30), nmPriced(4, "D", 1, 36),
		}},
	)
	res := Snipe(day, 5, 100, 15, false, false, "lift", 40, opCfg)
	if len(res) != 1 {
		t.Fatalf("expected 1 snipe (FLAT excluded by gap), got %d", len(res))
	}
	c := res[0]
	if c.Number != "SNIPE" {
		t.Fatalf("top snipe = %s, want SNIPE", c.Number)
	}
	if c.Floor != 10 || c.NextFloor != 50 {
		t.Errorf("floor=%.0f next=%.0f, want 10/50", c.Floor, c.NextFloor)
	}
	if c.LiftPct < 399 || c.LiftPct > 401 {
		t.Errorf("gap = %.1f, want ~400", c.LiftPct)
	}
	if c.CopiesToClear != 3 || c.StoreCount != 1 {
		t.Errorf("copies=%d stores=%d, want 3/1", c.CopiesToClear, c.StoreCount)
	}
	// profit = 3×50 - 3×10 cards - 1×15 ship = 150 - 30 - 15 = 105
	if c.ProfitBRL != 105 {
		t.Errorf("profit = %.0f, want 105 (3×50 - 30 cards - 15 ship)", c.ProfitBRL)
	}
}

func TestSnipeFloorAndEnglishFilter(t *testing.T) {
	day := day("2026-07-05",
		// floor 5 < minFloor 20 -> excluded even with a huge gap.
		CardDay{Number: "CHEAP", Name: "Cheap", LowBRL: 5, Stores: []StoreQty{
			nmPriced(1, "A", 1, 5), nmPriced(2, "B", 1, 90),
		}},
		// cheapest copy is Japanese and must not set the snipe floor.
		CardDay{Number: "LANG", Name: "Lang", LowBRL: 8, Stores: []StoreQty{
			priced(3, 1, 8, conditionNM, "6"),
			nmPriced(4, "B", 1, 40), nmPriced(5, "C", 1, 90),
		}},
	)
	res := Snipe(day, 20, 50, 0, false, false, "lift", 40, opCfg)
	if len(res) != 1 || res[0].Number != "LANG" {
		t.Fatalf("expected only LANG, got %+v", res)
	}
	if res[0].Floor != 40 || res[0].NextFloor != 90 {
		t.Errorf("floor=%.0f next=%.0f, want 40/90 (Japanese R$8 ignored)", res[0].Floor, res[0].NextFloor)
	}
}

func TestBuyoutFloorFilterAndCondition(t *testing.T) {
	day := day("2026-07-02",
		// below minFloor -> excluded
		CardDay{Number: "CHEAP", Name: "Cheap", LowBRL: 5, Stores: []StoreQty{
			nmPriced(1, "A", 1, 5), nmPriced(2, "B", 1, 30),
		}},
		// non-NM cheap copy must not set the floor
		CardDay{Number: "COND", Name: "Cond", LowBRL: 25, Stores: []StoreQty{
			{StoreID: 3, StoreName: "Damaged", Condition: "3", Quantity: 1, PriceBRL: 22, PriceKnown: true},
			nmPriced(4, "NMseller", 1, 25), nmPriced(5, "C", 3, 70),
		}},
	)
	res := Buyout(day, 500, 20, 0, false, false, "score", 10, opCfg)
	for _, c := range res {
		if c.Number == "CHEAP" {
			t.Errorf("CHEAP (floor 5 < 20) should be excluded")
		}
		if c.Number == "COND" && c.Floor != 25 {
			t.Errorf("COND floor = %.0f, want 25 (non-NM R$22 ignored)", c.Floor)
		}
	}
}
