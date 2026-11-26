package trades

import "testing"

func TestIsGraded(t *testing.T) {
	for cond, want := range map[string]bool{
		"PSA 10": true, "psa 9": true, "BGS 9.5": true, "CGC 10": true, "SGC 10": true, "TAG 10": true, "PSA10": true,
		"NM": false, "": false, "LP": false, "NM sem grade": false, "PSA": false, "Played": false,
	} {
		if got := IsGraded(cond); got != want {
			t.Errorf("IsGraded(%q) = %v, want %v", cond, got, want)
		}
	}
}

func TestGradedSlabIgnoresRawMarket(t *testing.T) {
	lookup := func(number, name, set string) (float64, string, bool) { return 90, "https://www.tcgplayer.com/product/1", true }
	slab := Trade{Number: "170", Name: "Squirtle", Condition: "PSA 10", Qty: 1, BuyBRL: 1500, RefUSD: 90, ManualBRL: 2800, Status: "holding"}
	v := BuildPortfolio([]Trade{slab}, 90, 0.1925, lookup).Trades[0]
	if v.MarketKnown || v.MarketUSD != 0 {
		t.Errorf("slab usou o preço da carta crua: known=%v usd=%.2f", v.MarketKnown, v.MarketUSD)
	}
	if !approx(v.ValueBRL, 2800) {
		t.Errorf("value = %.2f, want o valor manual 2800", v.ValueBRL)
	}
	slab.ManualBRL = 0
	if v := BuildPortfolio([]Trade{slab}, 90, 0.1925, lookup).Trades[0]; v.ValueBRL != 0 {
		t.Errorf("slab sem valor manual = %.2f, want 0 (sem avaliação)", v.ValueBRL)
	}
	raw := Trade{Number: "170", Name: "Squirtle", Condition: "NM", Qty: 1, Status: "holding"}
	if v := BuildPortfolio([]Trade{raw}, 100, 0.1925, lookup).Trades[0]; !v.MarketKnown || !approx(v.ValueBRL, 90/0.1925) {
		t.Errorf("carta crua continua pelo mercado: known=%v value=%.2f", v.MarketKnown, v.ValueBRL)
	}
}
