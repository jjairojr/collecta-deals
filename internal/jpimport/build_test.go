package jpimport

import (
	"fmt"
	"testing"

	"opdeals/internal/tracking"
)

func card(num int, name string, total int, stores ...tracking.StoreQty) tracking.CardDay {
	return tracking.CardDay{
		Number: fmt.Sprintf("%03d", num),
		Name:   fmt.Sprintf("%s (#%03d/%03d)", name, num, total),
		URL:    fmt.Sprintf("https://liga/%s/%d", name, num),
		Stores: stores,
	}
}

func listing(lang string, price float64, qty int) tracking.StoreQty {
	return tracking.StoreQty{StoreID: int(price * 100), Language: lang, Condition: "2", Quantity: qty, Known: true, PriceBRL: price, PriceKnown: true}
}

func day(set string, cards ...tracking.CardDay) SetData {
	return SetData{Day: tracking.DaySnapshot{Set: set, Cards: cards}}
}

func findRow(t *testing.T, rows []Row, set string, num int) Row {
	t.Helper()
	for _, r := range rows {
		if r.Set == set && r.Number == num {
			return r
		}
	}
	t.Fatalf("row %s %d not found", set, num)
	return Row{}
}

func TestBuildMatchesWesternBlocksAndPricesAgainstPortuguese(t *testing.T) {
	in := Input{
		JPYBRL: 0.03,
		Capture: Capture{Items: []Item{
			{Set: "M1L", Number: 88, Rarity: "SAR", Yen: 8000},
			{Set: "M1L", Number: 65, Rarity: "AR", Yen: 1000},
			{Set: "M1L", Number: 91, Rarity: "SAR", Yen: 20000},
		}},
		Sets: map[string]SetData{
			"M1L": day("M1L",
				card(65, "Ivysaur", 63, listing(LangJP, 75, 3)),
				card(88, "Mega Lucario ex", 63, listing(LangJP, 600, 2)),
				card(91, "Lillie's Determination", 63),
			),
			"MEG": day("MEG",
				card(10, "Filler", 132),
				card(11, "Filler 2", 132),
				card(134, "Ivysaur", 132, listing(LangPT, 78, 5)),
				card(160, "Mega Lucario ex", 132, listing(LangPT, 90, 9)),
				card(179, "Mega Lucario ex", 132, listing(LangPT, 465, 4), listing(LangEN, 500, 1)),
				card(188, "Mega Lucario ex", 132, listing(LangPT, 900, 1)),
			),
		},
	}
	rows := Build(in)

	sar := findRow(t, rows, "M1L", 88)
	if sar.West != "MEG" || sar.WestNumber != 179 {
		t.Fatalf("SAR matched %s %d, want MEG 179", sar.West, sar.WestNumber)
	}
	if sar.PT.FloorBRL != 465 || sar.EN.FloorBRL != 500 || sar.JP.FloorBRL != 600 {
		t.Fatalf("floors jp=%v pt=%v en=%v", sar.JP.FloorBRL, sar.PT.FloorBRL, sar.EN.FloorBRL)
	}
	if sar.CostBRL != 240 || sar.RealBRL != 465 || sar.RealX != 1.94 {
		t.Fatalf("cost=%v real=%v x=%v", sar.CostBRL, sar.RealBRL, sar.RealX)
	}

	ar := findRow(t, rows, "M1L", 65)
	if ar.WestNumber != 134 || ar.RealBRL != 75 {
		t.Fatalf("AR matched %d real %v, want 134 and 75", ar.WestNumber, ar.RealBRL)
	}

	lillie := findRow(t, rows, "M1L", 91)
	if lillie.WestStatus != WestMissing || lillie.RealX != 0 {
		t.Fatalf("lillie status=%s x=%v", lillie.WestStatus, lillie.RealX)
	}
}

func TestBuildUsesSIROverrideAndPairsTwinsInOrder(t *testing.T) {
	in := Input{
		JPYBRL: 0.03,
		Capture: Capture{Items: []Item{
			{Set: "M6a", Number: 126, Rarity: "SAR", Yen: 6000},
			{Set: "m6A", Number: 127, Rarity: "SAR", Yen: 8000},
			{Set: "M6a", Number: 128, Rarity: "SAR", Yen: 5000},
			{Set: "M6", Number: 110, Rarity: "SAR", Yen: 30000},
		}},
		Sets: map[string]SetData{
			"M6a": day("M6a",
				card(126, "Pikachu ex", 103),
				card(127, "Pikachu ex", 103),
				card(128, "Mewtwo ex", 103),
			),
			"30C": day("30C",
				card(1, "Filler", 128),
				card(2, "Filler 2", 128),
				card(149, "Pikachu ex", 128, listing(LangPT, 309, 1)),
				card(150, "Pikachu ex", 128, listing(LangPT, 294, 1)),
				card(151, "Mewtwo ex", 128, listing(LangPT, 275, 1)),
				card(157, "Mewtwo ex", 128, listing(LangPT, 999, 1)),
			),
			"M6": day("M6", card(110, "Mega Rayquaza ex", 81, listing(LangJP, 3000, 1))),
		},
	}
	rows := Build(in)

	if r := findRow(t, rows, "M6a", 126); r.WestNumber != 149 {
		t.Fatalf("first pikachu -> %d, want 149", r.WestNumber)
	}
	if r := findRow(t, rows, "M6a", 127); r.WestNumber != 150 {
		t.Fatalf("second pikachu -> %d, want 150", r.WestNumber)
	}
	if r := findRow(t, rows, "M6a", 128); r.WestNumber != 151 {
		t.Fatalf("mewtwo -> %d, want 151", r.WestNumber)
	}
	if r := findRow(t, rows, "M6", 110); r.WestStatus != WestUnreleased || r.RealBRL != 3000 {
		t.Fatalf("rayquaza status=%s real=%v", r.WestStatus, r.RealBRL)
	}
}

func TestIsSecret(t *testing.T) {
	if !IsSecret("Lillie's Determination (#091/063)", "091") {
		t.Fatal("091/063 should be secret")
	}
	if !IsSecret("Genesect ex (#172JP/086)", "172JP") {
		t.Fatal("172JP/086 should be secret")
	}
	if IsSecret("Pineco (#001/098)", "001") {
		t.Fatal("001/098 should not be secret")
	}
}

func TestSoldByLangCountsSellOutsAndSkipsHollowCaptures(t *testing.T) {
	snap := func(stores ...tracking.StoreQty) tracking.DaySnapshot {
		return tracking.DaySnapshot{Cards: []tracking.CardDay{{Number: "186", Stores: stores}}}
	}
	days := []tracking.DaySnapshot{
		snap(listing(LangPT, 164, 2), listing(LangJP, 145, 1)),
		snap(),
		snap(listing(LangPT, 164, 1)),
	}
	sold := SoldByLang(days)
	if sold["186"][LangPT] != 1 || sold["186"][LangJP] != 1 {
		t.Fatalf("sold = %v, want pt 1 and jp 1", sold["186"])
	}
}
