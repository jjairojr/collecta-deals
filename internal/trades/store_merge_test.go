package trades

import "testing"

func TestMergeCombinesHoldings(t *testing.T) {
	s := newTestStore(t)
	a, err := s.Add(Trade{Number: "OP16-080", Name: "Teach", Set: "OP16", Qty: 2, BuyBRL: 100, ShippingBRL: 10, BuyDate: "2026-07-10", Delivered: true, Notes: "lote1", AskBRL: 250, Listed: true})
	if err != nil {
		t.Fatalf("add a: %v", err)
	}
	b, err := s.Add(Trade{Number: "OP16-080", Name: "Teach", Set: "OP16", Qty: 3, BuyBRL: 50, ShippingBRL: 5, BuyDate: "2026-07-05", Delivered: false, Notes: "lote2"})
	if err != nil {
		t.Fatalf("add b: %v", err)
	}

	merged, err := s.Merge([]string{a.ID, b.ID})
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	if merged.ID != a.ID {
		t.Fatalf("primary should keep the oldest holding's id: got %s want %s", merged.ID, a.ID)
	}
	if merged.Qty != 5 {
		t.Fatalf("qty: got %d want 5", merged.Qty)
	}
	if merged.BuyBRL != 70 {
		t.Fatalf("weighted buy: got %v want 70", merged.BuyBRL)
	}
	if merged.ShippingBRL != 15 {
		t.Fatalf("shipping: got %v want 15", merged.ShippingBRL)
	}
	if merged.Delivered {
		t.Fatal("merged must not be delivered when one part is in transit")
	}
	if merged.BuyDate != "2026-07-05" {
		t.Fatalf("buy date should be earliest: got %s", merged.BuyDate)
	}
	if merged.Notes != "lote1; lote2" {
		t.Fatalf("notes should concatenate: got %q", merged.Notes)
	}
	if merged.AskBRL != 250 || !merged.Listed {
		t.Fatalf("merged should keep the primary's listing state: %+v", merged)
	}

	all, err := s.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("want a single trade after merge, got %d", len(all))
	}

	costA := 2*100.0 + 10
	costB := 3*50.0 + 5
	costMerged := float64(merged.Qty)*merged.BuyBRL + merged.ShippingBRL
	if round2(costMerged) != round2(costA+costB) {
		t.Fatalf("cost basis not conserved: %v vs %v", costMerged, costA+costB)
	}
}

func TestMergeRejectsSold(t *testing.T) {
	s := newTestStore(t)
	a := seed(t, s, 1, 100, 0)
	b, err := s.Add(Trade{Name: "Kuzan", Set: "SEALED", Qty: 1, BuyBRL: 50, Status: "sold"})
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if _, err := s.Merge([]string{a.ID, b.ID}); err == nil {
		t.Fatal("merging a sold trade must fail")
	}
	all, _ := s.List()
	if len(all) != 2 {
		t.Fatalf("failed merge must not mutate the store, got %d trades", len(all))
	}
}

func TestMergeNeedsTwo(t *testing.T) {
	s := newTestStore(t)
	a := seed(t, s, 1, 100, 0)
	if _, err := s.Merge([]string{a.ID}); err == nil {
		t.Fatal("merging fewer than two holdings must fail")
	}
}

func TestMergeMissing(t *testing.T) {
	s := newTestStore(t)
	a := seed(t, s, 1, 100, 0)
	if _, err := s.Merge([]string{a.ID, "nope"}); err != ErrNotFound {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}
