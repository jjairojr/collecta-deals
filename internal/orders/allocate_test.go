package orders

import (
	"testing"
	"time"

	"opdeals/internal/trades"
)

func holding(id, buyDate string, qty int, ask float64) trades.Trade {
	return trades.Trade{ID: id, Status: "holding", Listed: true, AskBRL: ask, Qty: qty, BuyDate: buyDate, CreatedAt: time.Now()}
}

func TestAllocateFIFO(t *testing.T) {
	ts := []trades.Trade{
		holding("newer", "2026-09-01", 2, 100),
		holding("older", "2026-08-01", 1, 120),
		{ID: "sold", Status: "sold", Listed: true, AskBRL: 100, Qty: 5, BuyDate: "2026-01-01"},
		{ID: "unlisted", Status: "holding", Listed: false, AskBRL: 100, Qty: 5, BuyDate: "2026-01-01"},
	}
	cases := []struct {
		name string
		qty  int
		want []Pick
		ok   bool
	}{
		{"one from oldest", 1, []Pick{{"older", 1}}, true},
		{"spans two trades", 2, []Pick{{"older", 1}, {"newer", 1}}, true},
		{"everything", 3, []Pick{{"older", 1}, {"newer", 2}}, true},
		{"insufficient", 4, nil, false},
		{"zero", 0, nil, false},
	}
	for _, c := range cases {
		got, ok := Allocate(ts, c.qty)
		if ok != c.ok {
			t.Fatalf("%s: ok = %v want %v", c.name, ok, c.ok)
		}
		if len(got) != len(c.want) {
			t.Fatalf("%s: picks = %+v want %+v", c.name, got, c.want)
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Fatalf("%s: picks = %+v want %+v", c.name, got, c.want)
			}
		}
	}
	if n := Available(ts); n != 3 {
		t.Fatalf("available = %d want 3", n)
	}
	if p, ok := UnitPrice(ts); !ok || p != 100 {
		t.Fatalf("unit price = %v %v want 100", p, ok)
	}
}

func TestUnitPriceNoneEligible(t *testing.T) {
	if _, ok := UnitPrice([]trades.Trade{{ID: "x", Status: "sold", Listed: true, AskBRL: 10}}); ok {
		t.Fatal("expected no price")
	}
}
