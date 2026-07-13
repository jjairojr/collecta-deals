package tracking

import (
	"testing"
	"time"
)

func TestStorageRoundTrip(t *testing.T) {
	st := NewStore(t.TempDir(), nil)
	d := DaySnapshot{
		Set:        "OP-16",
		Date:       "2026-07-02",
		CapturedAt: time.Date(2026, 7, 2, 3, 0, 0, 0, time.UTC),
		FXRate:     0.2,
		Cards:      []CardDay{card("OP16-001", 100, qty(1, "A", 5))},
	}
	if st.HasDay("OP-16", "2026-07-02") {
		t.Fatal("HasDay true before save")
	}
	if err := st.SaveDay(d); err != nil {
		t.Fatalf("SaveDay: %v", err)
	}
	if !st.HasDay("OP-16", "2026-07-02") {
		t.Fatal("HasDay false after save")
	}
	got, ok, err := st.LoadDay("OP-16", "2026-07-02")
	if err != nil || !ok {
		t.Fatalf("LoadDay: ok=%v err=%v", ok, err)
	}
	if len(got.Cards) != 1 || got.Cards[0].Stores[0].Quantity != 5 {
		t.Errorf("round trip mismatch: %+v", got)
	}
}

func TestListAndLoadRange(t *testing.T) {
	st := NewStore(t.TempDir(), nil)
	for _, date := range []string{"2026-07-03", "2026-07-01", "2026-07-02"} {
		if err := st.SaveDay(DaySnapshot{Set: "OP-16", Date: date}); err != nil {
			t.Fatal(err)
		}
	}
	dates, err := st.ListDates("OP-16")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"2026-07-01", "2026-07-02", "2026-07-03"}
	if len(dates) != 3 || dates[0] != want[0] || dates[2] != want[2] {
		t.Errorf("ListDates = %v, want sorted %v", dates, want)
	}
	rng, err := st.LoadRange("OP-16", "2026-07-02", "2026-07-03")
	if err != nil {
		t.Fatal(err)
	}
	if len(rng) != 2 {
		t.Errorf("LoadRange got %d days, want 2", len(rng))
	}
}

func TestListDatesMissingSet(t *testing.T) {
	st := NewStore(t.TempDir(), nil)
	dates, err := st.ListDates("OP-99")
	if err != nil {
		t.Fatalf("err on missing set: %v", err)
	}
	if len(dates) != 0 {
		t.Errorf("want no dates, got %v", dates)
	}
}
