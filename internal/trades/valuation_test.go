package trades

import (
	"math"
	"testing"
)

func approx(a, b float64) bool {
	return math.Abs(a-b) < 0.5
}

func TestHoldingValuation(t *testing.T) {
	lookup := func(number, name, set string) (float64, string, bool) {
		if number == "ST26-005" {
			return 748, "https://www.tcgplayer.com/product/123", true
		}
		return 0, "", false
	}
	trade := Trade{Number: "ST26-005", Name: "Monkey.D.Luffy (SP)", Qty: 1, BuyBRL: 2500, ShippingBRL: 71.97, Status: "holding"}
	p := BuildPortfolio([]Trade{trade}, 90, 0.1925, lookup)
	v := p.Trades[0]
	if !approx(v.CostBRL, 2571.97) {
		t.Errorf("cost = %.2f, want 2571.97", v.CostBRL)
	}
	if !approx(v.ValueBRL, 3497.14) {
		t.Errorf("value = %.2f, want 3497.14", v.ValueBRL)
	}
	if !approx(v.ProfitBRL, 925.17) {
		t.Errorf("profit = %.2f, want 925.17", v.ProfitBRL)
	}
	if !approx(p.Summary.UnrealizedBRL, 925.17) || p.Summary.Holdings != 1 {
		t.Errorf("summary unrealized = %.2f holdings=%d", p.Summary.UnrealizedBRL, p.Summary.Holdings)
	}
	if v.TCGURL != "https://www.tcgplayer.com/product/123" {
		t.Errorf("tcgURL = %q, want the catalog product URL", v.TCGURL)
	}
}

func TestRefUSDFallbackWhenNotInCatalog(t *testing.T) {
	lookup := func(number, name, set string) (float64, string, bool) { return 0, "", false }
	trade := Trade{Number: "OP99-999", Qty: 1, BuyBRL: 100, RefUSD: 100, Status: "holding"}
	v := BuildPortfolio([]Trade{trade}, 100, 0.1925, lookup).Trades[0]
	if !v.MarketKnown || v.MarketUSD != 100 {
		t.Errorf("expected RefUSD fallback, got known=%v usd=%.2f", v.MarketKnown, v.MarketUSD)
	}
	if !approx(v.ValueBRL, 100/0.1925) {
		t.Errorf("value = %.2f, want %.2f", v.ValueBRL, 100/0.1925)
	}
}

func TestSealedValuation(t *testing.T) {
	lookup := func(number, name, set string) (float64, string, bool) {
		t.Errorf("lookup must not be called for sealed trades, got %q", number)
		return 0, "", false
	}
	held := Trade{Kind: "sealed", Number: "135948", Name: "OP-16 Booster Box", Set: "SEALED", Qty: 2, BuyBRL: 400, ShippingBRL: 20, ManualBRL: 500, Status: "holding"}
	p := BuildPortfolio([]Trade{held}, 90, 0.1925, lookup)
	v := p.Trades[0]
	if !approx(v.CostBRL, 820) {
		t.Errorf("cost = %.2f, want 820", v.CostBRL)
	}
	if !approx(v.ValueBRL, 1000) {
		t.Errorf("value = %.2f, want 1000 (manual value, no pct/fx)", v.ValueBRL)
	}
	if !approx(v.ProfitBRL, 180) {
		t.Errorf("profit = %.2f, want 180", v.ProfitBRL)
	}
	if v.MarketKnown || v.MarketUSD != 0 || v.TCGURL != "" {
		t.Errorf("sealed must have no market lookup, got known=%v usd=%.2f url=%q", v.MarketKnown, v.MarketUSD, v.TCGURL)
	}

	sold := Trade{Kind: "sealed", Number: "135948", Name: "OP-16 Booster Box", Qty: 1, BuyBRL: 500, ManualBRL: 600, Status: "sold", SellPrice: 700, SellCurrency: "BRL"}
	sv := BuildPortfolio([]Trade{sold}, 90, 0.1925, lookup).Trades[0]
	if !sv.Realized || !approx(sv.ValueBRL, 700) {
		t.Errorf("sold sealed value = %.2f realized=%v, want 700 realized", sv.ValueBRL, sv.Realized)
	}
}

func TestMixedCardAndSealedSummary(t *testing.T) {
	lookup := func(number, name, set string) (float64, string, bool) {
		if number == "OP01-001" {
			return 100, "https://www.tcgplayer.com/product/1", true
		}
		return 0, "", false
	}
	card := Trade{Number: "OP01-001", Qty: 1, BuyBRL: 300, Status: "holding"}
	sealed := Trade{Kind: "sealed", Number: "135948", Qty: 1, BuyBRL: 500, ManualBRL: 650, Status: "holding"}
	p := BuildPortfolio([]Trade{card, sealed}, 100, 0.2, lookup)
	if p.Summary.Holdings != 2 {
		t.Errorf("holdings = %d, want 2", p.Summary.Holdings)
	}
	if !approx(p.Summary.InvestedBRL, 800) {
		t.Errorf("invested = %.2f, want 800", p.Summary.InvestedBRL)
	}
	if !approx(p.Summary.MarketBRL, 100/0.2+650) {
		t.Errorf("market = %.2f, want %.2f", p.Summary.MarketBRL, 100/0.2+650)
	}
}

func TestSoldRealizedUSDAndBRL(t *testing.T) {
	usd := Trade{Number: "X", Qty: 1, BuyBRL: 2000, Status: "sold", SellPrice: 499, SellCurrency: "USD"}
	brl := Trade{Number: "Y", Qty: 1, BuyBRL: 2000, Status: "sold", SellPrice: 2600, SellCurrency: "BRL"}
	p := BuildPortfolio([]Trade{usd, brl}, 90, 0.1925, nil)
	var vusd, vbrl TradeView
	for _, v := range p.Trades {
		if v.Number == "X" {
			vusd = v
		} else {
			vbrl = v
		}
	}
	if !vusd.Realized || !approx(vusd.ValueBRL, 499/0.1925) {
		t.Errorf("usd sold value = %.2f, want %.2f", vusd.ValueBRL, 499/0.1925)
	}
	if !approx(vbrl.ValueBRL, 2600) {
		t.Errorf("brl sold value = %.2f, want 2600", vbrl.ValueBRL)
	}
	if p.Summary.Sold != 2 {
		t.Errorf("sold count = %d, want 2", p.Summary.Sold)
	}
	wantRealized := (499/0.1925 - 2000) + (2600 - 2000)
	if !approx(p.Summary.RealizedBRL, wantRealized) {
		t.Errorf("realized = %.2f, want %.2f", p.Summary.RealizedBRL, wantRealized)
	}
}
