package packs

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"opdeals/internal/trades"
)

const (
	caseSKU    = "sealed|onepiece|case|||"
	boxSKU     = "sealed|onepiece|box|||"
	boosterSKU = "sealed|onepiece|booster|||"
)

func chain() []Link {
	return []Link{
		{ID: "a", Game: "onepiece", ParentSKU: caseSKU, ChildSKU: boxSKU, Units: 12},
		{ID: "b", Game: "onepiece", ParentSKU: boxSKU, ChildSKU: boosterSKU, Units: 24},
	}
}

func holding(id, name string, qty int, listed bool) trades.Trade {
	t := trades.Trade{ID: id, Name: name, Kind: "sealed", Qty: qty, BuyBRL: 1200, Status: "holding", BuyDate: "2026-10-01", CreatedAt: time.Unix(1, 0)}
	if listed {
		t.Listed = true
		t.AskBRL = 10
	}
	return t
}

func qtyOf(ts []trades.Trade, sku string) int {
	n := 0
	for _, t := range ts {
		if t.Status == "holding" && trades.SKU("onepiece", t) == sku {
			n += t.Qty
		}
	}
	return n
}

func TestStoreRejectsBadLinks(t *testing.T) {
	s := NewStore(filepath.Join(t.TempDir(), "packs.json"))
	for _, l := range chain() {
		if _, err := s.Add(l); err != nil {
			t.Fatal(err)
		}
	}
	bad := []Link{
		{Game: "onepiece", ParentSKU: caseSKU, ChildSKU: caseSKU, Units: 12},
		{Game: "onepiece", ParentSKU: "x", ChildSKU: "y", Units: 1},
		{Game: "onepiece", ParentSKU: boosterSKU, ChildSKU: caseSKU, Units: 2},
	}
	for _, l := range bad {
		if _, err := s.Add(l); !errors.Is(err, ErrInvalid) {
			t.Fatalf("%+v: %v", l, err)
		}
	}
	if _, err := s.Add(Link{Game: "onepiece", ParentSKU: "other", ChildSKU: boxSKU, Units: 6}); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("second parent: %v", err)
	}
	if _, err := s.Add(Link{Game: "pokemon", ParentSKU: caseSKU, ChildSKU: boxSKU, Units: 6}); err != nil {
		t.Fatalf("another game is another family: %v", err)
	}
	all, _ := s.List()
	if err := s.Delete(all[0].ID); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete("nope"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing: %v", err)
	}
}

func TestExpandCountsOnlyTheDirectPack(t *testing.T) {
	all := []trades.Trade{
		holding("c", "Case", 2, false),
		holding("x", "Box", 3, true),
		holding("o", "Booster", 5, true),
	}
	stock, origins := Expand("onepiece", all, chain())
	if got := qtyOf(stock, boxSKU); got != 3+24 {
		t.Fatalf("boxes = %d", got)
	}
	if got := qtyOf(stock, boosterSKU); got != 5+3*24 {
		t.Fatalf("boosters must not see the cases: %d", got)
	}
	if got := qtyOf(stock, caseSKU); got != 2 {
		t.Fatalf("cases = %d", got)
	}
	o, ok := origins[VirtualPrefix+"c"]
	if !ok || o.ParentID != "c" || o.Units != 12 {
		t.Fatalf("origins = %+v", origins)
	}
	for _, tr := range stock {
		if tr.ID != VirtualPrefix+"c" {
			continue
		}
		if tr.Name != "Box" || !tr.Listed || tr.AskBRL != 10 || tr.BuyDate != "" || tr.BuyBRL != 100 {
			t.Fatalf("virtual box = %+v", tr)
		}
	}
}

func TestExpandNeedsAContentRecord(t *testing.T) {
	stock, origins := Expand("onepiece", []trades.Trade{holding("c", "Case", 2, false)}, chain())
	if len(stock) != 1 || len(origins) != 0 {
		t.Fatalf("stock = %+v", stock)
	}
	sold := holding("s", "Box", 1, true)
	sold.Status = "sold"
	stock, _ = Expand("onepiece", []trades.Trade{holding("c", "Case", 2, false), sold}, chain())
	if got := qtyOf(stock, boxSKU); got != 24 {
		t.Fatalf("a sold box still carries the offer: %d", got)
	}
}

func TestClaimsSpreadReservationsAcrossTheFamily(t *testing.T) {
	all := []trades.Trade{
		holding("c", "Case", 2, true),
		holding("x", "Box", 3, true),
		holding("o", "Booster", 5, true),
	}
	got := Claims("onepiece", all, chain(), map[string]int{boxSKU: 5, caseSKU: 1})
	if got[caseSKU] != 2 {
		t.Fatalf("2 boxes beyond the loose ones hold one more case: %+v", got)
	}
	if got[boxSKU] != 5+12 {
		t.Fatalf("a held case takes its 12 boxes off the shelf: %+v", got)
	}
	if got[boosterSKU] != 5*24 {
		t.Fatalf("held boxes take their boosters: %+v", got)
	}
	got = Claims("onepiece", all, chain(), map[string]int{boosterSKU: 5 + 24*3 + 1})
	if got[boxSKU] != 4 || got[caseSKU] != 1 {
		t.Fatalf("booster overflow climbs box then case: %+v", got)
	}
}
