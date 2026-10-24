package tracking

import "testing"

func TestCardMarketNMLanguageFloorExcludingOwnStore(t *testing.T) {
	day := func(date string, stores ...StoreQty) DaySnapshot {
		return DaySnapshot{Date: date, Cards: []CardDay{{Number: "101", Stores: stores}}}
	}
	offer := func(store int, lang, cond string, price float64) StoreQty {
		return StoreQty{StoreID: store, Language: lang, Condition: cond, Quantity: 1, PriceBRL: price, PriceKnown: true}
	}
	days := []DaySnapshot{
		day("2026-10-01", offer(1, "8", "2", 50), offer(2, "2", "2", 30)),
		day("2026-10-05", offer(1, "8", "2", 46), offer(9, "8", "2", 40), offer(3, "8", "3", 20)),
		day("2026-10-09", offer(1, "8", "2", 44)),
	}
	got, ok := CardMarket(days, "101", "8", 9)
	if !ok {
		t.Fatal("expected a market price")
	}
	if got.LowBRL != 44 || got.MedianBRL != 46 || got.Date != "2026-10-09" || got.Days != 3 {
		t.Fatalf("got %+v", got)
	}
}

func TestCardMarketRefusesAmbiguousNumber(t *testing.T) {
	stores := []StoreQty{{StoreID: 1, Language: "2", Condition: "2", Quantity: 1, PriceBRL: 10, PriceKnown: true}}
	days := []DaySnapshot{{Date: "2026-10-09", Cards: []CardDay{{Number: "OP01-001", Stores: stores}, {Number: "OP01-001", Stores: stores}}}}
	if _, ok := CardMarket(days, "OP01-001", "2", 0); ok {
		t.Fatal("two cards share the number; the price must not be guessed")
	}
}
