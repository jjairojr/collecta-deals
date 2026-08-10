package liga

import (
	"os"
	"path/filepath"
	"testing"

	"opdeals/internal/game"
)

func TestParseEditions(t *testing.T) {
	html := readTestdata(t, "editions.html")
	sets := parseEditions(html)
	if len(sets) < 50 {
		t.Fatalf("expected many editions, got %d", len(sets))
	}
	if !contains(sets, "OP-01") {
		t.Fatalf("expected OP-01 among editions, first few: %v", sets[:5])
	}
}

func TestParseCards(t *testing.T) {
	html := readTestdata(t, "search_op01.html")
	cards, err := parseCards(html, "OP-01", game.OnePiece())
	if err != nil {
		t.Fatal(err)
	}
	if len(cards) == 0 {
		t.Fatal("expected cards, got 0")
	}
	found := false
	for _, c := range cards {
		if c.Number == "OP01-001" {
			found = true
			if c.LowBRL != 104.50 {
				t.Fatalf("OP01-001 LowBRL = %v, want 104.50", c.LowBRL)
			}
			if c.Name == "" {
				t.Fatal("OP01-001 name is empty")
			}
		}
	}
	if !found {
		t.Fatal("OP01-001 not found among parsed cards")
	}
}

func readTestdata(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

// Liga sends extras/sellType as a bare number on some pages and as a quoted
// string on others; decoding must survive both, and absence must stay unknown
// rather than collapsing to 0 (which means "Normal").
func TestNumOrStringIntAcceptsBothEncodings(t *testing.T) {
	cases := []struct {
		raw  string
		want *int
	}{
		{`2`, intPtr(2)},
		{`"2"`, intPtr(2)},
		{`0`, intPtr(0)},
		{`"0"`, intPtr(0)},
		{`null`, nil},
		{`""`, nil},
		{``, nil},
		{`"abc"`, nil},
	}
	for _, c := range cases {
		got := numOrStringInt([]byte(c.raw))
		switch {
		case c.want == nil && got != nil:
			t.Errorf("numOrStringInt(%q) = %d, want nil", c.raw, *got)
		case c.want != nil && got == nil:
			t.Errorf("numOrStringInt(%q) = nil, want %d", c.raw, *c.want)
		case c.want != nil && got != nil && *got != *c.want:
			t.Errorf("numOrStringInt(%q) = %d, want %d", c.raw, *got, *c.want)
		}
	}
}

func TestParseCardStockReadsExtrasAsString(t *testing.T) {
	html := []byte(`var cards_stock = [{"lj_id":1,"num":"60","qualid":"2","idioma":"2","extras":"2","sellType":1,"precoFinal":"179.00"},` +
		`{"lj_id":2,"num":"60","qualid":"2","idioma":"2","extras":0,"precoFinal":"169.00"}];`)
	out, err := parseCardStock(html, nil, nil)
	if err != nil {
		t.Fatalf("parseCardStock: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("got %d listings, want 2", len(out))
	}
	if out[0].Extras == nil || *out[0].Extras != 2 {
		t.Errorf("string-encoded extras = %v, want 2", out[0].Extras)
	}
	if out[1].Extras == nil || *out[1].Extras != 0 {
		t.Errorf("number-encoded extras = %v, want 0", out[1].Extras)
	}
	if out[1].SellType != nil {
		t.Errorf("absent sellType = %d, want nil (unknown)", *out[1].SellType)
	}
}

func intPtr(v int) *int { return &v }
