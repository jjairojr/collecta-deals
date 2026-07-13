package game

import "testing"

func contains(langs []string, want string) bool {
	for _, l := range langs {
		if l == want {
			return true
		}
	}
	return false
}

func TestBuyoutFloorLangs(t *testing.T) {
	// One Piece sets no BuyoutLangs, so buyout falls back to the English-only floor.
	op := OnePiece().BuyoutFloorLangs()
	if len(op) != 1 || op[0] != "2" {
		t.Errorf("OnePiece buyout langs = %v, want [2]", op)
	}

	// Pokémon's floor is multi-language (JP counts), but buyout/snipe must exclude
	// Japanese (6) while keeping English (2) and Portuguese (8).
	pk := Pokemon().BuyoutFloorLangs()
	if !contains(pk, "2") || !contains(pk, "8") {
		t.Errorf("Pokémon buyout langs = %v, want to include 2 and 8", pk)
	}
	if contains(pk, "6") {
		t.Errorf("Pokémon buyout langs = %v, must NOT include Japanese (6)", pk)
	}
	// The tracking floor itself still counts every language for Pokémon.
	if !Pokemon().FloorLangAllowed("6") {
		t.Errorf("Pokémon floor should still count Japanese listings")
	}
}

func TestRiftboundGroupSet(t *testing.T) {
	cases := []struct {
		abbrev string
		want   string
		ok     bool
	}{
		{"OGN", "OGN", true},
		{"OGS", "OGS", true},
		{"SFD", "SFD", true},
		{"UNL", "UNL", true},
		{"VEN", "VEN", true},
		{"OPP", "ROPP", true},
		{"PR", "OGN-PR", true},
		{"JDG", "JDG", true},
		{"RWB", "RWB", true},
		{"", "", false},
	}
	for _, c := range cases {
		got, ok := riftboundGroupSet(c.abbrev, "")
		if got != c.want || ok != c.ok {
			t.Errorf("riftboundGroupSet(%q) = %q,%v want %q,%v", c.abbrev, got, ok, c.want, c.ok)
		}
	}
}

func TestLorcanaGroupSet(t *testing.T) {
	cases := []struct {
		abbrev string
		want   string
		ok     bool
	}{
		{"1", "LOR1", true},
		{"9", "LOR9", true},
		{"13", "LOR13", true},
		{"Q1", "Q1", true},
		{"Q2", "Q2", true},
		{"D100", "D100", true},
		{"DLPC", "DLPC", true},
		{"Q3", "Q3", true},
		{"", "", false},
	}
	for _, c := range cases {
		got, ok := lorcanaGroupSet(c.abbrev, "")
		if got != c.want || ok != c.ok {
			t.Errorf("lorcanaGroupSet(%q) = %q,%v want %q,%v", c.abbrev, got, ok, c.want, c.ok)
		}
	}
}

func TestOnePieceGroupSet(t *testing.T) {
	if set, ok := onePieceGroupSet("", "Premium Booster -The Best-"); !ok || set != "PRB" {
		t.Errorf("PRB group = %q,%v", set, ok)
	}
	if set, ok := onePieceGroupSet("", "Premium Booster -The Best- Vol. 2"); !ok || set != "PRB2" {
		t.Errorf("PRB2 group = %q,%v", set, ok)
	}
	if set, ok := onePieceGroupSet("OP01", "Romance Dawn"); !ok || set != "" {
		t.Errorf("regular OP group = %q,%v, want \"\",true", set, ok)
	}
}

func TestGundamGroupSet(t *testing.T) {
	cases := []struct {
		abbrev string
		want   string
		ok     bool
	}{
		{"GD01", "GD01", true},
		{"GD04", "GD04", true},
		{"ST01", "ST01", true},
		{"EB01", "EB01", true},
		{"EXBP", "EXBP", true},
		{"RP", "RP", true},
		{"GD01_b", "BETA", true},
		{"", "", false},
	}
	for _, c := range cases {
		got, ok := gundamGroupSet(c.abbrev, "")
		if got != c.want || ok != c.ok {
			t.Errorf("gundamGroupSet(%q) = %q,%v want %q,%v", c.abbrev, got, ok, c.want, c.ok)
		}
	}
}

func TestHasDeals(t *testing.T) {
	if !OnePiece().HasDeals() || !Riftbound().HasDeals() || !Lorcana().HasDeals() || !Gundam().HasDeals() {
		t.Errorf("OnePiece/Riftbound/Lorcana/Gundam must have deals markets")
	}
	if Pokemon().HasDeals() {
		t.Errorf("Pokémon must stay BR-only (nil Market)")
	}
	if Riftbound().Market.TCGCSVCategoryID != 89 || Lorcana().Market.TCGCSVCategoryID != 71 ||
		OnePiece().Market.TCGCSVCategoryID != 68 || Gundam().Market.TCGCSVCategoryID != 86 {
		t.Errorf("wrong TCGCSV category ids")
	}
	// Gundam matches by globally-unique number like One Piece: no set-scoping.
	if Gundam().Market.SetScopedKey || Gundam().UniqueCardNumbers != true {
		t.Errorf("Gundam should use globally-unique numbers, no set-scoped key")
	}
}
