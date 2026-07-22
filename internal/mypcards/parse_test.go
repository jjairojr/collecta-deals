package mypcards

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"opdeals/internal/game"
	"opdeals/internal/model"
)

func readTestdata(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return b
}

func TestParseEditions(t *testing.T) {
	slugs := parseEditions(readTestdata(t, "editions.html"), "onepiece")
	if len(slugs) < 40 {
		t.Fatalf("expected many edition slugs, got %d", len(slugs))
	}
	want := map[string]bool{"the-time-of-battle": false, "set-sail-deck-set": false}
	for _, s := range slugs {
		if _, ok := want[s]; ok {
			want[s] = true
		}
		if strings.ContainsAny(s, " /?&") {
			t.Errorf("slug has invalid chars: %q", s)
		}
	}
	for slug, found := range want {
		if !found {
			t.Errorf("expected edition slug %q not parsed", slug)
		}
	}
}

func findByNumber(items []listingItem, number string) (model.BrazilListing, bool) {
	for _, it := range items {
		if it.listing.Number == number {
			return it.listing, true
		}
	}
	return model.BrazilListing{}, false
}

func TestParseListing(t *testing.T) {
	items := parseListing(readTestdata(t, "set_op16_p1.html"), *game.OnePiece().MyP)
	if len(items) < 20 {
		t.Fatalf("expected ~30 singles, got %d", len(items))
	}

	for _, it := range items {
		l := it.listing
		if l.Source != sourceName {
			t.Errorf("%s: source=%q", l.Number, l.Source)
		}
		if l.LowBRL <= 0 {
			t.Errorf("%s: non-positive price %.2f", l.Number, l.LowBRL)
		}
		if !l.InStock || !l.StockChecked {
			t.Errorf("%s: expected in-stock/checked", l.Number)
		}
		if !strings.HasPrefix(l.URL, baseURL) {
			t.Errorf("%s: bad url %q", l.Number, l.URL)
		}
		// Featured booster / sealed / other-game noise must be filtered out.
		if strings.Contains(strings.ToLower(l.Name), "one piece op16 -") {
			t.Errorf("featured booster leaked into listings: %q", l.Name)
		}
	}

	marshall, ok := findByNumber(items, "OP16-080")
	if !ok {
		t.Fatal("OP16-080 (Marshall.D.Teach) not found")
	}
	if !strings.Contains(marshall.Name, "Alternate Art") {
		t.Errorf("OP16-080 name = %q, want Alternate Art", marshall.Name)
	}
	if marshall.LowBRL != 397.90 {
		t.Errorf("OP16-080 price = %.2f, want 397.90", marshall.LowBRL)
	}
	if marshall.SetCode != "OP16" {
		t.Errorf("OP16-080 setcode = %q, want OP16", marshall.SetCode)
	}

	borsalino, ok := findByNumber(items, "OP16-073")
	if !ok {
		t.Fatal("OP16-073 (Borsalino Manga) not found")
	}
	if borsalino.LowBRL != 7000.00 {
		t.Errorf("OP16-073 price = %.2f, want 7000.00 (thousands separator)", borsalino.LowBRL)
	}
}

func TestParseListingOverflowIsEmpty(t *testing.T) {
	items := parseListing(readTestdata(t, "set_overflow.html"), *game.OnePiece().MyP)
	if len(items) != 0 {
		t.Fatalf("overflow page should yield 0 one_ singles, got %d", len(items))
	}
}

func TestParseBRL(t *testing.T) {
	cases := map[string]float64{
		"R$&nbsp;398,00":  398.00,
		"R$ 7.000,00":     7000.00,
		"R$ 0,50":         0.50,
		"R$ 1.234.567,89": 1234567.89,
		"":                0,
	}
	for in, want := range cases {
		if got := parseBRL(in); got != want {
			t.Errorf("parseBRL(%q) = %.2f, want %.2f", in, got, want)
		}
	}
}

func TestNumberFromGA(t *testing.T) {
	cases := map[string]string{
		"one_op16_op16-080p1": "OP16-080",
		"one_op16_op16-073p2": "OP16-073",
		"one_eb01_eb01-046":   "EB01-046",
		"mp_331390":           "",
		"yugioh_mp25-en021":   "",
		"op_8775_228435":      "",
	}
	for in, want := range cases {
		if got := numberFromGA(in, *game.OnePiece().MyP); got != want {
			t.Errorf("numberFromGA(%q) = %q, want %q", in, got, want)
		}
	}
}

// Riftbound numbers keep their denominator and letter suffix for the Matcher to
// normalize, and One Piece's print-suffix strip must not apply: "234p" is a
// distinct printing, not "234". Cross-sell cards from other games are dropped.
func TestNumberFromGARiftbound(t *testing.T) {
	cfg := *game.Riftbound().MyP
	cases := map[string]string{
		"riftbound_unl_234/219":  "234/219",
		"riftbound_unl_059a/219": "059A/219",
		"riftbound_opp_197b/298": "197B/298",
		"riftbound_unl_234p/219": "234P/219",
		"one_op16_op16-080p1":    "",
		"mp_331390":              "",
		"op_8775_228435":         "",
	}
	for in, want := range cases {
		if got := numberFromGA(in, cfg); got != want {
			t.Errorf("numberFromGA(%q) = %q, want %q", in, got, want)
		}
	}
}

// MyP transposes Spiritforged to SPF and uses TCGCSV's OPP/PR promo abbrevs;
// all three must land on the Liga codes or set-scoped matching drops the set.
func TestRiftboundSetAliases(t *testing.T) {
	cfg := game.Riftbound().MyP
	cases := map[string]string{
		"SPF": "SFD", "OPP": "ROPP", "PR": "OGN-PR",
		"UNL": "UNL", "OGN": "OGN", "VEN": "VEN", "OGS": "OGS", "JDG": "JDG",
	}
	for in, want := range cases {
		if got := cfg.SetCode(in); got != want {
			t.Errorf("SetCode(%q) = %q, want %q", in, got, want)
		}
	}
}
