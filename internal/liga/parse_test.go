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
