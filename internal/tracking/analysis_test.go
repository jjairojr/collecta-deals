package tracking

import "testing"

func day(date string, cards ...CardDay) DaySnapshot {
	return DaySnapshot{Set: "OP-16", Date: date, Cards: cards}
}

func card(number string, low float64, stores ...StoreQty) CardDay {
	return CardDay{Number: number, Name: number, LowBRL: low, Stores: stores}
}

func qty(id int, name string, n int) StoreQty {
	return StoreQty{StoreID: id, StoreName: name, Condition: "2", Language: "2", Quantity: n, Known: true}
}

func pricedQty(id int, name string, n int, price float64) StoreQty {
	return StoreQty{StoreID: id, StoreName: name, Condition: "2", Language: "2", Quantity: n, Known: true, PriceBRL: price, PriceKnown: true}
}

// TestSalesUsePerStorePrice verifies revenue and the seller price reflect what
// each store was actually listing at, not the card's floor, when prices are known.
func TestSalesUsePerStorePrice(t *testing.T) {
	// Floor is 100, but store A lists at 180 and store B at 250.
	d1 := day("2026-07-01", card("OP16-050", 100, pricedQty(1, "A", 3, 180), pricedQty(2, "B", 2, 250)))
	d2 := day("2026-07-02", card("OP16-050", 100, pricedQty(1, "A", 1, 180), pricedQty(2, "B", 0, 250)))
	snaps := SalesBySnapshot([]DaySnapshot{d1, d2})
	if len(snaps) != 1 {
		t.Fatalf("intervals = %d, want 1", len(snaps))
	}
	c := snaps[0].Cards[0]
	// A sold 2 @180 = 360, B sold 2 @250 = 500 -> 860 total (not 4*100 floor).
	if c.RevenueBRL != 860 {
		t.Errorf("revenue = %v, want 860 (per-store prices, not floor)", c.RevenueBRL)
	}
	byStore := map[int]CardSeller{}
	for _, s := range c.Sellers {
		byStore[s.StoreID] = s
	}
	if byStore[1].PriceBRL != 180 || byStore[2].PriceBRL != 250 {
		t.Errorf("seller prices = A:%v B:%v, want 180 / 250", byStore[1].PriceBRL, byStore[2].PriceBRL)
	}
	stats := Leaderboard([]DaySnapshot{d1, d2})
	s, _ := statFor(stats, 2)
	if s.RevenueBRL != 500 {
		t.Errorf("store B leaderboard revenue = %v, want 500 (2 @250)", s.RevenueBRL)
	}
}

func statFor(stats []StoreStat, id int) (StoreStat, bool) {
	for _, s := range stats {
		if s.StoreID == id {
			return s, true
		}
	}
	return StoreStat{}, false
}

func TestLeaderboardCountsDrops(t *testing.T) {
	d1 := day("2026-07-01", card("OP16-001", 100, qty(1, "A", 5)))
	d2 := day("2026-07-02", card("OP16-001", 110, qty(1, "A", 2)))
	stats := Leaderboard([]DaySnapshot{d1, d2})
	s, ok := statFor(stats, 1)
	if !ok {
		t.Fatal("store A missing")
	}
	if s.UnitsSold != 3 {
		t.Errorf("units = %d, want 3", s.UnitsSold)
	}
	if s.RevenueBRL != 3*110 {
		t.Errorf("revenue = %v, want %v", s.RevenueBRL, 3*110.0)
	}
	if len(s.Cards) != 1 || s.Cards[0].Number != "OP16-001" || s.Cards[0].Units != 3 || s.Cards[0].RevenueBRL != 330 {
		t.Errorf("per-card breakdown = %+v, want 1 card OP16-001 units 3 rev 330", s.Cards)
	}
}

func TestLeaderboardPerCardBreakdown(t *testing.T) {
	d1 := day("2026-07-01",
		card("OP16-065-MA", 9500, qty(1, "A", 2)),
		card("OP16-001", 100, qty(1, "A", 10)),
	)
	d2 := day("2026-07-02",
		card("OP16-065-MA", 9500, qty(1, "A", 1)),
		card("OP16-001", 100, qty(1, "A", 4)),
	)
	stats := Leaderboard([]DaySnapshot{d1, d2})
	s, _ := statFor(stats, 1)
	if s.UnitsSold != 7 {
		t.Fatalf("units = %d, want 7 (1 MA + 6 common)", s.UnitsSold)
	}
	// cards sorted by units desc: OP16-001 (6) before OP16-065-MA (1)
	if len(s.Cards) != 2 || s.Cards[0].Number != "OP16-001" || s.Cards[0].Units != 6 {
		t.Errorf("cards[0] = %+v, want OP16-001 units 6", s.Cards)
	}
	if s.Cards[1].Number != "OP16-065-MA" || s.Cards[1].RevenueBRL != 9500 {
		t.Errorf("cards[1] = %+v, want OP16-065-MA rev 9500", s.Cards)
	}
}

func TestLeaderboardIgnoresIncreaseNewUnknownAndVanished(t *testing.T) {
	d1 := day("2026-07-01",
		card("OP16-001", 100, qty(1, "A", 2)), // increase case
		card("OP16-002", 50, qty(2, "B", 4)),  // vanished next day
		card("OP16-003", 10, StoreQty{StoreID: 3, StoreName: "C", Condition: "2", Language: "2", Quantity: 5, Known: false}), // unknown
	)
	d2 := day("2026-07-02",
		card("OP16-001", 100, qty(1, "A", 5)), // went up
		card("OP16-002", 50),                  // B vanished
		card("OP16-003", 10, StoreQty{StoreID: 3, StoreName: "C", Condition: "2", Language: "2", Quantity: 2, Known: false}), // unknown
		card("OP16-004", 20, qty(4, "D", 3)), // new key
	)
	stats := Leaderboard([]DaySnapshot{d1, d2})
	if len(stats) != 0 {
		t.Errorf("expected no sales inferred, got %+v", stats)
	}
}

func TestLeaderboardSumsDuplicateFullKey(t *testing.T) {
	d1 := day("2026-07-01", card("OP16-001", 100, qty(1, "A", 4), qty(1, "A", 6)))
	d2 := day("2026-07-02", card("OP16-001", 100, qty(1, "A", 3), qty(1, "A", 2)))
	stats := Leaderboard([]DaySnapshot{d1, d2})
	s, _ := statFor(stats, 1)
	if s.UnitsSold != 5 {
		t.Errorf("units = %d, want 5 (10 -> 5)", s.UnitsSold)
	}
}

func TestPriceTrends(t *testing.T) {
	prev := day("2026-07-01", card("OP16-001", 100))
	today := day("2026-07-02", card("OP16-001", 120), card("OP16-002", 50))
	trends := PriceTrends(today, prev)
	var got *CardTrend
	for i := range trends {
		if trends[i].Number == "OP16-001" {
			got = &trends[i]
		}
	}
	if got == nil {
		t.Fatal("OP16-001 trend missing")
	}
	if got.DeltaPct != 20 {
		t.Errorf("delta = %v, want 20", got.DeltaPct)
	}
	for _, tr := range trends {
		if tr.Number == "OP16-002" && tr.DeltaPct != 0 {
			t.Errorf("new card delta = %v, want 0", tr.DeltaPct)
		}
	}
}

func TestInventory(t *testing.T) {
	day := day("2026-07-02",
		card("OP16-065-MA", 9500, qty(1, "Brandao", 1)),
		card("OP16-001", 100, qty(1, "Brandao", 5), qty(2, "Volume Co", 40)),
	)
	inv := Inventory(day, 10, 5, 5)
	if inv.ActiveStores != 2 {
		t.Errorf("active stores = %d, want 2", inv.ActiveStores)
	}
	if inv.TotalUnits != 46 {
		t.Errorf("total units = %d, want 46", inv.TotalUnits)
	}
	// Brandao: 1*9500 + 5*100 = 10000; Volume Co: 40*100 = 4000
	if inv.Stores[0].StoreName != "Brandao" || inv.Stores[0].ValueBRL != 10000 {
		t.Errorf("top store = %+v, want Brandao 10000", inv.Stores[0])
	}
	if inv.Stores[0].TopCardNumber != "OP16-065-MA" {
		t.Errorf("top card = %s, want OP16-065-MA", inv.Stores[0].TopCardNumber)
	}
	if inv.Stores[1].StoreName != "Volume Co" || inv.Stores[1].Units != 40 {
		t.Errorf("second store = %+v, want Volume Co 40 units", inv.Stores[1])
	}
	if len(inv.Expensive) == 0 || inv.Expensive[0].Number != "OP16-065-MA" || inv.Expensive[0].TotalQty != 1 {
		t.Errorf("expensive[0] = %+v, want OP16-065-MA qty 1", inv.Expensive)
	}
}

func TestSalesBySnapshotPerInterval(t *testing.T) {
	d1 := day("2026-07-01", card("OP16-001", 100, qty(1, "A", 5)))
	d2 := day("2026-07-02", card("OP16-001", 110, qty(1, "A", 2)), card("OP16-002", 50, qty(2, "B", 4)))
	d3 := day("2026-07-03", card("OP16-001", 110, qty(1, "A", 2)), card("OP16-002", 55, qty(2, "B", 1)))
	snaps := SalesBySnapshot([]DaySnapshot{d1, d2, d3})
	if len(snaps) != 2 {
		t.Fatalf("intervals = %d, want 2", len(snaps))
	}
	// newest first: 07-03 (B sold 3), then 07-02 (A sold 3)
	if snaps[0].Date != "2026-07-03" || snaps[0].Units != 3 || snaps[0].RevenueBRL != 3*55 {
		t.Errorf("snaps[0] = %+v, want 07-03 units 3 rev 165", snaps[0])
	}
	if len(snaps[0].Cards) != 1 || snaps[0].Cards[0].Number != "OP16-002" {
		t.Errorf("snaps[0] cards = %+v, want only OP16-002", snaps[0].Cards)
	}
	if snaps[1].Date != "2026-07-02" || snaps[1].PrevDate != "2026-07-01" || snaps[1].Units != 3 {
		t.Errorf("snaps[1] = %+v, want 07-02 from 07-01 units 3", snaps[1])
	}
}

func TestSalesBySnapshotIncludesZeroIntervals(t *testing.T) {
	d1 := day("2026-07-01", card("OP16-001", 100, qty(1, "A", 5)))
	d2 := day("2026-07-02", card("OP16-001", 100, qty(1, "A", 5)))
	snaps := SalesBySnapshot([]DaySnapshot{d1, d2})
	if len(snaps) != 1 || snaps[0].Units != 0 || len(snaps[0].Cards) != 0 {
		t.Errorf("snaps = %+v, want one zero-unit interval", snaps)
	}
}

func TestTopSoldMatchesSnapshotTotals(t *testing.T) {
	d1 := day("2026-07-01", card("OP16-001", 100, qty(1, "A", 5)))
	d2 := day("2026-07-02", card("OP16-001", 110, qty(1, "A", 2)))
	d3 := day("2026-07-03", card("OP16-001", 120, qty(1, "A", 0)))
	top := TopSoldCards([]DaySnapshot{d1, d2, d3})
	if len(top) != 1 || top[0].Number != "OP16-001" {
		t.Fatalf("top = %+v, want single OP16-001", top)
	}
	// 3 units @110 + 2 units @120 = 330 + 240 = 570, merged into one seller
	if top[0].Units != 5 || top[0].RevenueBRL != 3*110+2*120 {
		t.Errorf("top[0] = %+v, want units 5 rev 570", top[0])
	}
	if len(top[0].Sellers) != 1 || top[0].Sellers[0].Units != 5 {
		t.Errorf("sellers = %+v, want one seller with 5 units", top[0].Sellers)
	}
}

func TestSortLeaderboardByRevenue(t *testing.T) {
	stats := []StoreStat{
		{StoreID: 1, UnitsSold: 10, RevenueBRL: 100},
		{StoreID: 2, UnitsSold: 3, RevenueBRL: 900},
	}
	SortLeaderboard(stats, "revenue")
	if stats[0].StoreID != 2 {
		t.Errorf("revenue sort: top = %d, want 2", stats[0].StoreID)
	}
	SortLeaderboard(stats, "units")
	if stats[0].StoreID != 1 {
		t.Errorf("units sort: top = %d, want 1", stats[0].StoreID)
	}
}
