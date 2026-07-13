package liga

import "testing"

func row(cond, lang string, price float64, qty int) StoreListing {
	return StoreListing{
		Condition:  cond,
		Language:   lang,
		PriceBRL:   price,
		PriceKnown: price > 0,
		Quantity:   qty,
		QtyKnown:   true,
	}
}

// The cheapest English listing is a non-NM (qualid 3) card; the NM floor must
// skip it and price against the cheapest English Near Mint instead.
func TestFloorNMSkipsLowerGrades(t *testing.T) {
	rows := []StoreListing{
		row("3", LangEnglish, 400, 1), // Excellent/SP, cheaper — must be ignored
		row(condNM, LangEnglish, 680, 2),
		row(condNM, LangEnglish, 720, 1),
	}
	if got, has := FloorNM(rows, []string{LangEnglish}); !has || got != 680 {
		t.Fatalf("FloorNM = %.2f has=%v, want 680 true", got, has)
	}
	if got, _ := Floor(rows, []string{LangEnglish}); got != 400 {
		t.Fatalf("plain Floor = %.2f, want 400 (any grade)", got)
	}
}

func TestFloorNMExcludesNonEnglishNM(t *testing.T) {
	rows := []StoreListing{
		row(condNM, "6", 300, 1), // Japanese NM — wrong language
		row(condNM, LangEnglish, 500, 1),
	}
	if got, has := FloorNM(rows, []string{LangEnglish}); !has || got != 500 {
		t.Fatalf("FloorNM = %.2f has=%v, want 500 true", got, has)
	}
}

func TestFloorNMNoNMStockReportsOut(t *testing.T) {
	rows := []StoreListing{
		row("3", LangEnglish, 400, 1),
		row("4", LangEnglish, 350, 2),
	}
	if got, has := FloorNM(rows, []string{LangEnglish}); has || got != 0 {
		t.Fatalf("FloorNM = %.2f has=%v, want 0 false (no NM English)", got, has)
	}
}

func TestFloorNMSkipsOutOfStockNM(t *testing.T) {
	rows := []StoreListing{
		row(condNM, LangEnglish, 500, 0), // NM but zero qty
		row(condNM, LangEnglish, 620, 1),
	}
	if got, has := FloorNM(rows, []string{LangEnglish}); !has || got != 620 {
		t.Fatalf("FloorNM = %.2f has=%v, want 620 true", got, has)
	}
}

func storeRow(store int, cond, lang string, price float64, qty int) StoreListing {
	r := row(cond, lang, price, qty)
	r.StoreID = store
	return r
}

// floorDepth counts copies sitting exactly at the floor and distinct stores with
// English NM stock — ignoring lower grades, other languages, and out-of-stock rows.
func TestFloorDepthCountsCopiesAndSellers(t *testing.T) {
	rows := []StoreListing{
		storeRow(1, condNM, LangEnglish, 680, 2), // at floor: +2 copies
		storeRow(2, condNM, LangEnglish, 680, 3), // at floor: +3 copies, distinct store
		storeRow(2, condNM, LangEnglish, 720, 1), // above floor: same store, not counted as copies
		storeRow(3, condNM, LangEnglish, 900, 1), // above floor: distinct store
		storeRow(4, "3", LangEnglish, 400, 5),    // cheaper but not NM: excluded entirely
		storeRow(5, condNM, "6", 500, 4),         // Japanese NM: excluded
		storeRow(6, condNM, LangEnglish, 680, 0), // at floor price but zero qty: excluded
	}
	floor, has := FloorNM(rows, []string{LangEnglish})
	if !has || floor != 680 {
		t.Fatalf("FloorNM = %.2f has=%v, want 680 true", floor, has)
	}
	copies, sellers := floorDepth(rows, []string{LangEnglish}, floor)
	if copies != 5 {
		t.Errorf("copies = %d, want 5 (2+3 at the 680 floor)", copies)
	}
	if sellers != 3 {
		t.Errorf("sellers = %d, want 3 (stores 1,2,3 with English NM stock)", sellers)
	}
}

func TestFloorDepthUnknownFloorIsZero(t *testing.T) {
	rows := []StoreListing{storeRow(1, "3", LangEnglish, 400, 1)}
	if copies, sellers := floorDepth(rows, []string{LangEnglish}, 0); copies != 0 || sellers != 0 {
		t.Errorf("floorDepth on unknown floor = %dc/%ds, want 0/0", copies, sellers)
	}
}
