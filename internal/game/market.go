package game

type Market struct {
	TCGCSVCategoryID     int
	GroupSetCode         func(abbrev, name string) (string, bool)
	SetScopedKey         bool
	StripDenominator     bool
	StripLeadingZeros    bool
	NumberSuffixVariants map[string]string
	VariantVocab         map[string]string
	DON                  bool
}

var donSetGroups = map[string]string{
	"Premium Booster -The Best-":        "PRB",
	"Premium Booster -The Best- Vol. 2": "PRB2",
}

func onePieceGroupSet(_, name string) (string, bool) {
	return donSetGroups[name], true
}

func riftboundGroupSet(abbrev, _ string) (string, bool) {
	switch abbrev {
	case "":
		return "", false
	case "OPP":
		return "ROPP", true
	case "PR":
		return "OGN-PR", true
	}
	return abbrev, true
}

func lorcanaGroupSet(abbrev, _ string) (string, bool) {
	if abbrev == "" {
		return "", false
	}
	if isDigits(abbrev) {
		return "LOR" + abbrev, true
	}
	return abbrev, true
}

// gundamGroupSet maps a TCGCSV group to its Liga set code. Codes are identical on
// both sides (GD01, ST01, EB01, EXBP…) except TCGCSV's "GD01_b" beta printing,
// which Liga lists as "BETA". Since Gundam numbers are globally unique the set
// code is only used for display/filtering, not the match key.
func gundamGroupSet(abbrev, _ string) (string, bool) {
	switch abbrev {
	case "":
		return "", false
	case "GD01_b":
		return "BETA", true
	}
	return abbrev, true
}

func isDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return s != ""
}

var onePieceVariants = map[string]string{
	"alternate art":    "Alt Art",
	"alt art":          "Alt Art",
	"sp":               "SP",
	"manga":            "Manga",
	"gold":             "Gold",
	"parallel":         "Parallel",
	"full art":         "Full Art",
	"pirate foil":      "Pirate Foil",
	"wanted poster":    "Wanted Poster",
	"jolly roger foil": "Jolly Roger",
	"dash pack":        "Dash Pack",
	"reprint":          "Reprint",
	"box topper":       "Box Topper",
}

func onePieceMarket() *Market {
	return &Market{
		TCGCSVCategoryID: 68,
		GroupSetCode:     onePieceGroupSet,
		VariantVocab:     onePieceVariants,
		DON:              true,
	}
}

func riftboundMarket() *Market {
	return &Market{
		TCGCSVCategoryID:     89,
		GroupSetCode:         riftboundGroupSet,
		SetScopedKey:         true,
		StripDenominator:     true,
		StripLeadingZeros:    true,
		NumberSuffixVariants: map[string]string{"A": "alt", "S": "sig", "*": "sig"},
	}
}

func lorcanaMarket() *Market {
	return &Market{
		TCGCSVCategoryID:  71,
		GroupSetCode:      lorcanaGroupSet,
		SetScopedKey:      true,
		StripDenominator:  true,
		StripLeadingZeros: true,
	}
}

// gundamMarket mirrors One Piece: globally-unique set-prefixed numbers matched
// whole (no set-scoping, no leading-zero/denominator stripping). Variant vocab is
// left empty for now — the cheapest-print dedup keeps matches conservative until a
// Gundam variant vocabulary is needed.
func gundamMarket() *Market {
	return &Market{
		TCGCSVCategoryID: 86,
		GroupSetCode:     gundamGroupSet,
	}
}

func (g Game) HasDeals() bool {
	return g.Market != nil
}
