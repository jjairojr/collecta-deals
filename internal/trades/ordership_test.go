package trades

import "testing"

func TestAllocateOrderShippingByValue(t *testing.T) {
	got := AllocateOrderShipping(30, []Trade{
		{ID: "a", Qty: 2, BuyBRL: 100},
		{ID: "b", Qty: 1, BuyBRL: 100},
	})
	if got["a"] != 20 || got["b"] != 10 {
		t.Fatalf("shares = %v, want a 20 / b 10", got)
	}
}

func TestAllocateOrderShippingByQtyWhenNothingWasPaid(t *testing.T) {
	got := AllocateOrderShipping(30, []Trade{
		{ID: "a", Qty: 2},
		{ID: "b", Qty: 1},
	})
	if got["a"] != 20 || got["b"] != 10 {
		t.Fatalf("shares = %v, want a 20 / b 10", got)
	}
}

func TestAllocateOrderShippingClosesTheCents(t *testing.T) {
	got := AllocateOrderShipping(10, []Trade{
		{ID: "a", Qty: 1, BuyBRL: 10},
		{ID: "b", Qty: 1, BuyBRL: 10},
		{ID: "c", Qty: 1, BuyBRL: 10},
	})
	sum := 0.0
	for _, v := range got {
		sum += v
	}
	if round2(sum) != 10 {
		t.Fatalf("shares = %v sum %v, want 10", got, sum)
	}
}

func TestAllocateOrderShippingZeroResetsEveryLot(t *testing.T) {
	got := AllocateOrderShipping(0, []Trade{{ID: "a", Qty: 1, BuyBRL: 10}})
	if v, ok := got["a"]; !ok || v != 0 {
		t.Fatalf("shares = %v, want a 0", got)
	}
}

func TestSellSplitsOrderShipping(t *testing.T) {
	s := NewStore(t.TempDir()+"/t.json", nil)
	lot, err := s.Add(Trade{Name: "Box", Qty: 4, BuyBRL: 10, OrderShippingBRL: 8})
	if err != nil {
		t.Fatal(err)
	}
	sold, err := s.Sell(lot.ID, Sale{Qty: 1, SellPrice: 20})
	if err != nil {
		t.Fatal(err)
	}
	if sold.OrderShippingBRL != 2 {
		t.Fatalf("sold share = %v, want 2", sold.OrderShippingBRL)
	}
	all, _ := s.List()
	for _, tr := range all {
		if tr.ID == lot.ID && tr.OrderShippingBRL != 6 {
			t.Fatalf("remaining share = %v, want 6", tr.OrderShippingBRL)
		}
	}
	back, err := s.Unsell(sold.ID)
	if err != nil {
		t.Fatal(err)
	}
	if back.Restored.OrderShippingBRL != 8 {
		t.Fatalf("restored share = %v, want 8", back.Restored.OrderShippingBRL)
	}
}
