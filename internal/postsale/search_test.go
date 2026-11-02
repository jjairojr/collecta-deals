package postsale

import "testing"

func TestSearchRanksExactBeforeSimilar(t *testing.T) {
	items := []Item{
		{SKU: "a", Name: "Shanks", Set: "OP09", Number: "OP09-004", Meta: "OP09 · Manga", UnitBRL: 900, Available: 1},
		{SKU: "b", Name: "Shanks", Set: "OP01", Number: "OP01-120", Meta: "OP01", UnitBRL: 40, Available: 2},
		{SKU: "c", Name: "Nami", Set: "OP01", Number: "OP01-016", Meta: "OP01", UnitBRL: 15, Available: 1},
	}
	exact, similar := Search(items, "Shanks manga", 8)
	if len(exact) != 1 || exact[0].SKU != "a" {
		t.Fatalf("exact = %+v, want only a", exact)
	}
	if len(similar) != 1 || similar[0].SKU != "b" {
		t.Fatalf("similar = %+v, want only b", similar)
	}
}

func TestSearchFoldsAccents(t *testing.T) {
	items := []Item{{SKU: "s", Name: "Coleção Premium Pikachu", Kind: "sealed", UnitBRL: 300, Available: 1}}
	for _, q := range []string{"colecao premium", "COLEÇÃO premium", "colecão"} {
		exact, _ := Search(items, q, 8)
		if len(exact) != 1 {
			t.Fatalf("%q: exact = %+v, want 1 hit", q, exact)
		}
	}
}

func TestSearchByCardNumber(t *testing.T) {
	items := []Item{
		{SKU: "n", Name: "Nami", Set: "OP01", Number: "OP01-016", UnitBRL: 15, Available: 1},
		{SKU: "z", Name: "Zoro", Set: "OP01", Number: "OP01-025", UnitBRL: 30, Available: 1},
	}
	exact, _ := Search(items, "op01-016", 8)
	if len(exact) != 1 || exact[0].SKU != "n" {
		t.Fatalf("exact = %+v, want n", exact)
	}
}

func TestSearchRespectsLimitAndPriceOrder(t *testing.T) {
	items := []Item{
		{SKU: "1", Name: "Pikachu", UnitBRL: 10, Available: 1},
		{SKU: "2", Name: "Pikachu", UnitBRL: 50, Available: 1},
		{SKU: "3", Name: "Pikachu", UnitBRL: 30, Available: 1},
	}
	exact, _ := Search(items, "pikachu", 2)
	if len(exact) != 2 || exact[0].SKU != "2" || exact[1].SKU != "3" {
		t.Fatalf("exact = %+v, want 2 then 3", exact)
	}
}

func TestSearchShortWordsMatchNothing(t *testing.T) {
	items := []Item{{SKU: "1", Name: "Ex Pikachu", UnitBRL: 10, Available: 1}}
	exact, similar := Search(items, "ex", 8)
	if len(exact) != 0 || len(similar) != 0 {
		t.Fatalf("got exact=%v similar=%v, want nothing", exact, similar)
	}
}
