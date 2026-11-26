package trades

import "testing"

func openFixture(t *testing.T) (*Store, Trade) {
	t.Helper()
	s := newTestStore(t)
	parent, err := s.Add(Trade{Name: "Case OP18", Set: "SEALED", Kind: "sealed", Qty: 5, BuyBRL: 6192, ShippingBRL: 50, PurchaseID: "p1", BuyDate: "2026-10-07"})
	if err != nil {
		t.Fatal(err)
	}
	return s, parent
}

func boxTemplate() Trade {
	return Trade{Name: "Box OP18", Set: "SEALED", Kind: "sealed", AskBRL: 714.49, Listed: true}
}

func TestOpenMovesCostFromPackToContent(t *testing.T) {
	s, parent := openFixture(t)
	child, err := s.Open(parent.ID, 1, 12, boxTemplate())
	if err != nil {
		t.Fatal(err)
	}
	if child.Qty != 12 || child.BuyBRL != 516 || child.ShippingBRL != 10 || child.Status != "holding" {
		t.Fatalf("child = %+v", child)
	}
	if child.PurchaseID != "p1" || child.BuyDate != "2026-10-07" || !child.Listed || child.AskBRL != 714.49 || child.SplitFrom != "" {
		t.Fatalf("child must inherit the purchase and keep the offer: %+v", child)
	}
	all, _ := s.List()
	cost := 0.0
	for _, tr := range all {
		cost += float64(tr.Qty)*tr.BuyBRL + tr.ShippingBRL
		if tr.ID == parent.ID && (tr.Qty != 4 || tr.ShippingBRL != 40) {
			t.Fatalf("parent = %+v", tr)
		}
	}
	if len(all) != 2 || round2(cost) != 5*6192+50 {
		t.Fatalf("records %d cost %v", len(all), cost)
	}
}

func TestOpenAddsToTheSameLotAndDropsAnEmptyPack(t *testing.T) {
	s, parent := openFixture(t)
	first, err := s.Open(parent.ID, 2, 12, boxTemplate())
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.Open(parent.ID, 3, 12, boxTemplate())
	if err != nil {
		t.Fatal(err)
	}
	if second.ID != first.ID || second.Qty != 60 {
		t.Fatalf("second open = %+v", second)
	}
	all, _ := s.List()
	if len(all) != 1 || all[0].ID != first.ID || all[0].ShippingBRL != 50 {
		t.Fatalf("emptied pack must leave the ledger: %+v", all)
	}
}

func TestOpenRejectsBadRequests(t *testing.T) {
	s, parent := openFixture(t)
	if _, err := s.Open(parent.ID, 6, 12, boxTemplate()); err != ErrInvalidOpen {
		t.Fatalf("beyond stock: %v", err)
	}
	if _, err := s.Open(parent.ID, 1, 1, boxTemplate()); err != ErrInvalidOpen {
		t.Fatalf("one unit: %v", err)
	}
	if _, err := s.Open("nope", 1, 12, boxTemplate()); err != ErrNotFound {
		t.Fatalf("missing: %v", err)
	}
	if _, err := s.Sell(parent.ID, Sale{SellPrice: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Open(parent.ID, 1, 12, boxTemplate()); err != ErrAlreadySold {
		t.Fatalf("sold: %v", err)
	}
}
