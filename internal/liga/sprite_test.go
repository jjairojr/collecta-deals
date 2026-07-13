package liga

import (
	"os"
	"testing"
)

func loadFixtureStock(t *testing.T) []StoreListing {
	t.Helper()
	html, err := os.ReadFile("testdata/card_op16_056aa.html")
	if err != nil {
		t.Fatalf("read html fixture: %v", err)
	}
	raw, err := os.ReadFile("testdata/card_op16_056aa_atlas.png")
	if err != nil {
		t.Fatalf("read atlas fixture: %v", err)
	}
	atlas, ok := decodeAtlas(raw)
	if !ok {
		t.Fatal("decode atlas fixture")
	}
	listings, err := parseCardStock(html, atlas, nil)
	if err != nil {
		t.Fatalf("parseCardStock: %v", err)
	}
	if len(listings) == 0 {
		t.Fatal("no listings parsed")
	}
	return listings
}

func findStore(listings []StoreListing, name string) (StoreListing, bool) {
	for _, l := range listings {
		if l.StoreName == name {
			return l, true
		}
	}
	return StoreListing{}, false
}

func TestParseCardStockGroundTruth(t *testing.T) {
	listings := loadFixtureStock(t)

	cases := []struct {
		store string
		qty   int
	}{
		{"Main Deck - Card Games", 3},
		{"Geex Hobby Store", 1},
		{"Geek Collections House", 2},
		{"MTG Brasil", 1},
	}
	for _, c := range cases {
		l, ok := findStore(listings, c.store)
		if !ok {
			t.Errorf("store %q not found", c.store)
			continue
		}
		if !l.QtyKnown {
			t.Errorf("store %q: quantity not decoded", c.store)
			continue
		}
		if l.Quantity != c.qty {
			t.Errorf("store %q: got qty %d, want %d", c.store, l.Quantity, c.qty)
		}
	}
}

func TestParseCardStockResolvesStoreNames(t *testing.T) {
	listings := loadFixtureStock(t)
	for _, l := range listings {
		if l.StoreName == "" {
			t.Errorf("store id %d has no resolved name", l.StoreID)
		}
		if l.Number == "" {
			t.Errorf("store %d has empty card number", l.StoreID)
		}
	}
}

func TestDecodeQuantityUnknownAtlas(t *testing.T) {
	html, err := os.ReadFile("testdata/card_op16_056aa.html")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	listings, err := parseCardStock(html, nil, nil)
	if err != nil {
		t.Fatalf("parseCardStock: %v", err)
	}
	for _, l := range listings {
		if l.QtyKnown {
			t.Fatalf("expected all quantities unknown without an atlas, got known for %q", l.StoreName)
		}
	}
}

func TestParseCardStockDecodesPrices(t *testing.T) {
	html, err := os.ReadFile("testdata/card_op16_015_prices.html")
	if err != nil {
		t.Fatalf("read html fixture: %v", err)
	}
	raw, err := os.ReadFile("testdata/card_op16_015_priceatlas.png")
	if err != nil {
		t.Fatalf("read price atlas fixture: %v", err)
	}
	priceAtlas, ok := decodeAtlas(raw)
	if !ok {
		t.Fatal("decode price atlas fixture")
	}
	listings, err := parseCardStock(html, nil, priceAtlas)
	if err != nil {
		t.Fatalf("parseCardStock: %v", err)
	}
	cases := []struct {
		store string
		price float64
	}{
		{"Cards Of Paradise", 4.88}, // plain precoFinal
		{"MTG Brasil", 4.95},        // decoded precoCss
	}
	for _, c := range cases {
		l, found := findStore(listings, c.store)
		if !found {
			t.Errorf("store %q not found", c.store)
			continue
		}
		if !l.PriceKnown {
			t.Errorf("store %q: price not decoded", c.store)
			continue
		}
		if l.PriceBRL != c.price {
			t.Errorf("store %q: got price %.2f, want %.2f", c.store, l.PriceBRL, c.price)
		}
	}
}
