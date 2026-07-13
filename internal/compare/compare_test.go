package compare

import (
	"testing"

	"opdeals/internal/game"
	"opdeals/internal/model"
)

func TestDeals(t *testing.T) {
	listings := []model.BrazilListing{
		{Number: "OP01-001", Name: "Zoro", LowBRL: 100},
		{Number: "OP01-002", Name: "Cheap US", LowBRL: 10},
		{Number: "OP01-099", Name: "No US match", LowBRL: 5},
	}
	prices := []model.USPrice{
		{Number: "OP01-001", Variant: "Normal", Rarity: "L", MarketUSD: 40, LowUSD: 35},
		{Number: "OP01-002", Variant: "Normal", MarketUSD: 1.5, LowUSD: 1.0},
	}

	deals := Deals(listings, prices, Options{
		FXRate:    0.2,
		MinMargin: 30,
		MinPrice:  1,
		UsePrice:  "market",
		SortBy:    "margin",
	})

	if len(deals) != 1 {
		t.Fatalf("want 1 deal, got %d: %+v", len(deals), deals)
	}
	d := deals[0]
	if d.Number != "OP01-001" {
		t.Fatalf("want OP01-001, got %s", d.Number)
	}
	if d.MarginPct != 100 {
		t.Fatalf("margin = %v, want 100", d.MarginPct)
	}
	if d.Rarity != "L" {
		t.Fatalf("rarity = %q, want L", d.Rarity)
	}
	if d.ProfitUSD != 20 {
		t.Fatalf("profit = %v, want 20", d.ProfitUSD)
	}
}

func TestDealsMatchesCheapestPrintForNumberCollision(t *testing.T) {
	listings := []model.BrazilListing{
		{Number: "OP01-024", Name: "Monkey.D.Luffy", LowBRL: 6.90},
	}
	prices := []model.USPrice{
		{Number: "OP01-024", Variant: "Parallel", MarketUSD: 201.43},
		{Number: "OP01-024", Variant: "Normal", MarketUSD: 15.48},
	}

	deals := Deals(listings, prices, Options{FXRate: 0.2, MinMargin: 0, MinPrice: 1, UsePrice: "market", SortBy: "margin"})
	if len(deals) != 1 {
		t.Fatalf("want 1 deal, got %d", len(deals))
	}
	if deals[0].SellUSD != 15.48 {
		t.Fatalf("want cheapest base print (15.48) matched, got %v (variant %s)", deals[0].SellUSD, deals[0].Variant)
	}
}

func TestDealsExcludesVerifiedOutOfStock(t *testing.T) {
	listings := []model.BrazilListing{
		{Number: "OP02-013-MA", Name: "Portgas.D.Ace (Manga) (OP02-013-MA)", LowBRL: 2150, StockChecked: true, InStock: false},
		{Number: "OP01-024", Name: "Monkey.D.Luffy (OP01-024)", LowBRL: 6.90, StockChecked: true, InStock: true},
	}
	prices := []model.USPrice{
		{Number: "OP02-013", Name: "Portgas.D.Ace (Manga)", MarketUSD: 1000},
		{Number: "OP01-024", Name: "Monkey.D.Luffy (024)", MarketUSD: 15.48},
	}

	deals := Deals(listings, prices, Options{FXRate: 0.2, MinMargin: 0, MinPrice: 1, UsePrice: "market", SortBy: "margin"})
	for _, d := range deals {
		if d.Number == "OP02-013-MA" {
			t.Fatalf("out-of-stock card should be excluded, but it was returned")
		}
	}
	if len(deals) != 1 || deals[0].Number != "OP01-024" {
		t.Fatalf("want only the in-stock deal, got %+v", deals)
	}
}

func TestDealsRequireInStockHidesUnverified(t *testing.T) {
	listings := []model.BrazilListing{
		{Number: "OP05-060", Name: "Monkey.D.Luffy (SPR) (OP05-060)", LowBRL: 54599.9},
		{Number: "OP01-024", Name: "Monkey.D.Luffy (OP01-024)", LowBRL: 6.90, StockChecked: true, InStock: true},
	}
	prices := []model.USPrice{
		{Number: "OP05-060", Name: "Monkey.D.Luffy (SPR)", MarketUSD: 13999},
		{Number: "OP01-024", Name: "Monkey.D.Luffy (024)", MarketUSD: 15.48},
	}

	deals := Deals(listings, prices, Options{FXRate: 0.2, MinMargin: -1e9, MinPrice: 0, UsePrice: "market", SortBy: "margin", RequireInStock: true})
	if len(deals) != 1 || deals[0].Number != "OP01-024" {
		t.Fatalf("want only the verified in-stock deal, got %+v", deals)
	}
}

func TestDealsMatchesVariantsByName(t *testing.T) {
	listings := []model.BrazilListing{
		{Number: "OP11-106", Name: "Zeus (OP11-106)", LowBRL: 9.40},
		{Number: "OP11-106-AA", Name: "Zeus (Alternate Art) (OP11-106-AA)", LowBRL: 329.99},
		{Number: "OP11-106-SP", Name: "Zeus (SP) (OP11-106-SP)", LowBRL: 2700},
	}
	prices := []model.USPrice{
		{Number: "OP11-106", Name: "Zeus", Variant: "Foil", MarketUSD: 1.72},
		{Number: "OP11-106", Name: "Zeus (Alternate Art)", Variant: "Foil", MarketUSD: 48.27},
		{Number: "OP11-106", Name: "Zeus (SP)", Variant: "Foil", MarketUSD: 498},
	}

	deals := Deals(listings, prices, Options{FXRate: 0.2, MinMargin: -1e9, MinPrice: 0, UsePrice: "market", SortBy: "margin"})
	got := map[string]model.Deal{}
	for _, d := range deals {
		got[d.Number] = d
	}
	if len(deals) != 3 {
		t.Fatalf("want 3 variant deals, got %d", len(deals))
	}
	if got["OP11-106-SP"].SellUSD != 498 {
		t.Fatalf("SP should match $498, got %v", got["OP11-106-SP"].SellUSD)
	}
	if got["OP11-106-AA"].SellUSD != 48.27 {
		t.Fatalf("AA should match $48.27, got %v", got["OP11-106-AA"].SellUSD)
	}
	if got["OP11-106"].SellUSD != 1.72 {
		t.Fatalf("base should match $1.72, got %v", got["OP11-106"].SellUSD)
	}
	if got["OP11-106-SP"].Variant != "SP" {
		t.Fatalf("SP variant label, got %q", got["OP11-106-SP"].Variant)
	}
}

func TestVariantTokenDistinguishesMultiMarker(t *testing.T) {
	m := Matcher{}.norm()
	if m.variantToken("Monkey.D.Luffy (119) (SP)", "") == m.variantToken("Monkey.D.Luffy (119) (SP) (Gold)", "") {
		t.Fatal("(SP) and (SP) (Gold) must produce different variant tokens")
	}
	// order-independent
	if m.variantToken("X (SP) (Gold)", "") != m.variantToken("X (Gold) (SP)", "") {
		t.Fatal("variant token should be order-independent")
	}
}

func TestMatcherRiftboundKeys(t *testing.T) {
	m := MatcherFor(game.Riftbound())
	cases := []struct {
		ligaNum, ligaName, ligaSet string
		tcgNum, tcgName, tcgSet    string
	}{
		{"162", "Miss Fortune - Captain (162)", "OGN", "162/298", "Miss Fortune - Captain", "OGN"},
		{"162A", "Miss Fortune - Captain (Alternate Art) (162A)", "OGN", "162a/298", "Miss Fortune - Captain (Alternate Art)", "OGN"},
		{"308S", "Viktor - Herald of the Arcane (Signature) (308S)", "OGN", "308*/298", "Viktor - Herald of the Arcane (Signature)", "OGN"},
		{"308", "Viktor - Herald of the Arcane (Overnumbered) (308)", "OGN", "308/298", "Viktor - Herald of the Arcane (Overnumbered)", "OGN"},
		{"1", "Blazing Scorcher (1)", "OGN", "001/298", "Blazing Scorcher", "OGN"},
	}
	for _, c := range cases {
		lk := m.Key(c.ligaNum, c.ligaName, c.ligaSet)
		tk := m.Key(c.tcgNum, c.tcgName, c.tcgSet)
		if lk != tk {
			t.Errorf("liga %q/%q -> %q but tcg %q/%q -> %q", c.ligaNum, c.ligaName, lk, c.tcgNum, c.tcgName, tk)
		}
	}
	if m.Key("162", "Miss Fortune - Captain (162)", "OGN") == m.Key("162A", "Miss Fortune - Captain (Alternate Art) (162A)", "OGN") {
		t.Error("base and alt-art must not collide")
	}
	if m.Key("308", "x (Overnumbered)", "OGN") == m.Key("308S", "x (Signature)", "OGN") {
		t.Error("overnumbered and signature must not collide")
	}
	if m.Key("1", "Blazing Scorcher (1)", "OGN") == m.Key("1", "Fleeting Squire (1)", "SFD") {
		t.Error("same number in different sets must not collide")
	}
}

func TestMatcherLorcanaKeys(t *testing.T) {
	m := MatcherFor(game.Lorcana())
	if m.Key("10", "Mulan - Free Spirit (10)", "LOR9") != m.Key("10/204", "Mulan - Free Spirit", "LOR9") {
		t.Error("liga and tcg base numbers must align")
	}
	if m.Key("235", "Mulan - Considerate Diplomat (Enchanted) (235)", "LOR9") != m.Key("235/204", "Mulan - Considerate Diplomat (Enchanted)", "LOR9") {
		t.Error("enchanted numbers must align")
	}
	if m.Key("10", "Mulan", "LOR9") == m.Key("10", "HeiHei", "LOR10") {
		t.Error("same number in different sets must not collide")
	}
	if m.Key("010/204", "x", "LOR9") != m.Key("10", "x", "LOR9") {
		t.Error("leading zeros must be stripped")
	}
}

func TestMatcherZeroValueIsOnePiece(t *testing.T) {
	zero := Matcher{}
	op := MatcherFor(game.OnePiece())
	cases := [][3]string{
		{"OP11-106", "Zeus (Alternate Art)", ""},
		{"OP01-024", "Monkey.D.Luffy (024)", "OP-01"},
		{"DON-PRB30-G", "DON!! Card (Zoro) (Gold) (DON-PRB30-G)", "PRB"},
		{"OP05-119-SP", "Monkey.D.Luffy (119) (SP)", ""},
	}
	for _, c := range cases {
		if zero.Key(c[0], c[1], c[2]) != op.Key(c[0], c[1], c[2]) {
			t.Errorf("zero matcher diverges from OnePiece matcher for %v", c)
		}
	}
	if op.Key("OP11-106-AA", "Zeus (Alternate Art) (OP11-106-AA)", "") != op.Key("OP11-106", "Zeus (Alternate Art)", "") {
		t.Error("OP variant matching must join Liga -AA suffix with TCG name tag")
	}
	if op.LookupKey("OP11-106-AA", "Zeus (Alternate Art)", "OP-11") != op.Key("OP11-106-AA", "Zeus (Alternate Art)", "") {
		t.Error("OP LookupKey must drop the set")
	}
}

// DON!! cards have no card number on the US side, so they match by set-scoped name.
// The same character's Gold DON exists in both premium boosters, so the set must
// keep them apart: a PRB Zoro must take the PRB Zoro US price, not the PRB2 one.
func TestDealsMatchesDONBySetScopedName(t *testing.T) {
	listings := []model.BrazilListing{
		{Number: "DON-PRB30-G", SetCode: "PRB", Name: "DON!! Card (Zoro) (Gold) (DON-PRB30-G)", LowBRL: 1200},
		{Number: "DON-018-G", SetCode: "PRB2", Name: "DON!! Card (Nami) (Gold) (DON-018-G)", LowBRL: 1500},
	}
	prices := []model.USPrice{
		{Name: "DON!! Card (Zoro) (Gold)", SetCode: "PRB", Rarity: "DON!!", MarketUSD: 300},
		{Name: "DON!! Card (Zoro) (Gold)", SetCode: "PRB2", Rarity: "DON!!", MarketUSD: 999}, // same char, other set
		{Name: "DON!! Card (Nami) (Gold)", SetCode: "PRB2", Rarity: "DON!!", MarketUSD: 250},
	}
	deals := Deals(listings, prices, Options{FXRate: 0.2, MinMargin: -1e9, MinPrice: 0, UsePrice: "market", SortBy: "margin"})
	got := map[string]model.Deal{}
	for _, d := range deals {
		got[d.Number] = d
	}
	if len(deals) != 2 {
		t.Fatalf("want 2 DON deals, got %d: %+v", len(deals), deals)
	}
	if got["DON-PRB30-G"].SellUSD != 300 {
		t.Fatalf("PRB Zoro must match the PRB US price $300, got %v", got["DON-PRB30-G"].SellUSD)
	}
	if got["DON-018-G"].SellUSD != 250 {
		t.Fatalf("PRB2 Nami must match $250, got %v", got["DON-018-G"].SellUSD)
	}
	if got["DON-PRB30-G"].Variant != "Gold" {
		t.Fatalf("DON variant label should be Gold, got %q", got["DON-PRB30-G"].Variant)
	}
}

func TestDealsDoesNotMatchSPtoSPGold(t *testing.T) {
	listings := []model.BrazilListing{
		{Number: "OP05-119-SP", Name: "Monkey.D.Luffy (119) (SP)", LowBRL: 18999.9},
	}
	prices := []model.USPrice{
		{Number: "OP05-119", Name: "Monkey.D.Luffy (119) (SP) (Gold)", MarketUSD: 13999},
	}
	deals := Deals(listings, prices, Options{FXRate: 0.2, MinMargin: -1e9, MinPrice: 0, UsePrice: "market", SortBy: "margin"})
	if len(deals) != 0 {
		t.Fatalf("plain (SP) must not match (SP) (Gold); got %+v", deals)
	}
}

func TestDealsPicksCheapestSourceAndTagsIt(t *testing.T) {
	listings := []model.BrazilListing{
		{Number: "OP16-080", Name: "Marshall.D.Teach (Alternate Art)", Source: "ligaonepiece", URL: "https://liga/x", LowBRL: 500},
		{Number: "OP16-080", Name: "Marshall.D.Teach (Alternate Art)", Source: "mypcards", URL: "https://myp/x", LowBRL: 397.90, StockChecked: true, InStock: true},
	}
	prices := []model.USPrice{
		{Number: "OP16-080", Name: "Marshall.D.Teach (Alternate Art)", MarketUSD: 200},
	}

	deals := Deals(listings, prices, Options{FXRate: 0.2, MinMargin: -1e9, MinPrice: 0, UsePrice: "market", SortBy: "margin"})
	if len(deals) != 1 {
		t.Fatalf("want 1 deal, got %d: %+v", len(deals), deals)
	}
	d := deals[0]
	if d.LowBRL != 397.90 {
		t.Fatalf("want cheapest BR price across sources (397.90), got %v", d.LowBRL)
	}
	if d.Source != "mypcards" {
		t.Fatalf("want source mypcards (the cheaper one), got %q", d.Source)
	}
	if d.BuyURL != "https://myp/x" {
		t.Fatalf("want the winning source's URL, got %q", d.BuyURL)
	}
}

func TestDealsDedupesSameNumberKeepingCheapestBR(t *testing.T) {
	listings := []model.BrazilListing{
		{Number: "OP01-003", SetCode: "OP-01", Name: "Luffy", LowBRL: 28.99},
		{Number: "OP01-003", SetCode: "OP-01-PR", Name: "Luffy", LowBRL: 27.00},
	}
	prices := []model.USPrice{
		{Number: "OP01-003", Variant: "Normal", Rarity: "L", MarketUSD: 19.14},
	}

	deals := Deals(listings, prices, Options{FXRate: 0.2, MinMargin: 0, MinPrice: 1, UsePrice: "market", SortBy: "margin"})
	if len(deals) != 1 {
		t.Fatalf("want 1 deduped deal, got %d", len(deals))
	}
	if deals[0].LowBRL != 27.00 {
		t.Fatalf("want cheapest BR price (27.00), got %v", deals[0].LowBRL)
	}
}
