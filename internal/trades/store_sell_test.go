package trades

import (
	"path/filepath"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	return NewStore(filepath.Join(t.TempDir(), "trades.json"), nil)
}

func seed(t *testing.T, s *Store, qty int, buy, ship float64) Trade {
	t.Helper()
	trade, err := s.Add(Trade{Name: "Kuzan", Set: "SEALED", Kind: "sealed", Qty: qty, BuyBRL: buy, ShippingBRL: ship, ManualBRL: 99.9})
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	return trade
}

func TestSellPartialSplits(t *testing.T) {
	s := newTestStore(t)
	base := seed(t, s, 3, 89.91, 5.99)

	sold, err := s.Sell(base.ID, Sale{Qty: 2, SellPrice: 260, SellCurrency: "BRL", SellDate: "2026-07-15", Buyer: "Wallace"})
	if err != nil {
		t.Fatalf("sell: %v", err)
	}
	if sold.ID == base.ID {
		t.Fatal("sold split must be a new trade, not the original")
	}
	if sold.Qty != 2 || sold.Status != "sold" || sold.SellPrice != 260 {
		t.Fatalf("unexpected sold split: %+v", sold)
	}

	all, err := s.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("want 2 trades after split, got %d", len(all))
	}

	var rem *Trade
	for i := range all {
		if all[i].ID == base.ID {
			rem = &all[i]
		}
	}
	if rem == nil {
		t.Fatal("remaining holding not found")
	}
	if rem.Qty != 1 || rem.Status != "holding" {
		t.Fatalf("remaining should be 1 holding, got qty=%d status=%s", rem.Qty, rem.Status)
	}
	if rem.SellPrice != 0 || rem.Buyer != "" {
		t.Fatalf("remaining holding must not carry sale fields: %+v", rem)
	}

	if got := round2(rem.ShippingBRL + sold.ShippingBRL); got != 5.99 {
		t.Fatalf("shipping not conserved: %v + %v = %v", rem.ShippingBRL, sold.ShippingBRL, got)
	}
}

func TestSellFullMarksSold(t *testing.T) {
	s := newTestStore(t)
	base := seed(t, s, 3, 89.91, 5.99)

	sold, err := s.Sell(base.ID, Sale{Qty: 3, SellPrice: 390, SellCurrency: "BRL"})
	if err != nil {
		t.Fatalf("sell: %v", err)
	}
	if sold.ID != base.ID {
		t.Fatal("full sale must mutate the original trade, not split")
	}
	all, err := s.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(all) != 1 || all[0].Status != "sold" || all[0].Qty != 3 {
		t.Fatalf("want single sold trade of qty 3, got %+v", all)
	}
}

func TestSellQtyBeyondHoldingIsFull(t *testing.T) {
	s := newTestStore(t)
	base := seed(t, s, 2, 50, 0)

	if _, err := s.Sell(base.ID, Sale{Qty: 5, SellPrice: 100, SellCurrency: "BRL"}); err != nil {
		t.Fatalf("sell: %v", err)
	}
	all, _ := s.List()
	if len(all) != 1 || all[0].Status != "sold" {
		t.Fatalf("over-qty sale should mark whole trade sold, got %+v", all)
	}
}

func TestSellMissingTrade(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Sell("nope", Sale{Qty: 1, SellPrice: 10}); err != ErrNotFound {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}
