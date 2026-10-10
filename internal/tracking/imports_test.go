package tracking

import (
	"testing"
)

const ownStore = 866280

func TestImportSupplyExcludesOwnStore(t *testing.T) {
	d := day("2026-08-16",
		card("OGN-1", 10,
			nmPriced(ownStore, "COLLECTA", 3, 10),
			nmPriced(2, "B", 2, 14),
			nmPriced(2, "B", 1, 18),
			nmPriced(3, "C", 4, 20),
			priced(4, 5, 5, conditionNM, "8"),
			priced(5, 5, 6, "3", "2"),
		),
		card("OGN-2", 30, nmPriced(ownStore, "COLLECTA", 1, 30)),
		card("DON-001", 5, nmPriced(2, "B", 1, 5)),
	)
	got := ImportSupplyFor(d, []string{"2"}, ownStore)
	if len(got) != 1 {
		t.Fatalf("want only OGN-1, got %+v", got)
	}
	s := got[0]
	if s.FloorBRL != 14 || s.FloorCopies != 2 {
		t.Errorf("floor = %.2f x%d, want 14 x2 (own store and PT/played excluded)", s.FloorBRL, s.FloorCopies)
	}
	if s.Sellers != 2 || s.Supply != 7 {
		t.Errorf("sellers=%d supply=%d, want 2 distinct stores / 7 copies", s.Sellers, s.Supply)
	}
	if s.OwnQty != 3 || s.OwnPriceBRL != 10 {
		t.Errorf("own = %d @ %.2f, want 3 @ 10", s.OwnQty, s.OwnPriceBRL)
	}
	if len(s.FloorStores) != 1 || s.FloorStores[0].StoreID != 2 || s.FloorStores[0].Qty != 2 {
		t.Errorf("floor stores = %+v, want only B x2", s.FloorStores)
	}
	if s.Date != "2026-08-16" || s.Set != "OP-16" {
		t.Errorf("date/set = %s/%s", s.Date, s.Set)
	}
}

func TestLatestDayWithDepthSkipsHollowCapture(t *testing.T) {
	st := NewStore(t.TempDir(), nil)
	deep := day("2026-08-15T18", card("A", 10, nmPriced(2, "B", 1, 10)))
	hollow := day("2026-08-16T18", CardDay{Number: "A", Name: "A", LowBRL: 12})
	for _, d := range []DaySnapshot{deep, hollow} {
		if err := st.SaveDay(d); err != nil {
			t.Fatal(err)
		}
	}
	got, ok, err := st.LatestDayWithDepth("OP-16", 8)
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if got.Date != "2026-08-15T18" {
		t.Errorf("date = %s, want the deep capture", got.Date)
	}
	if _, ok, _ := st.LatestDayWithDepth("OP-16", 1); ok {
		t.Errorf("maxBack=1 should not reach the deep capture")
	}
}
