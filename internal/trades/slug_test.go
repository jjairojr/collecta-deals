package trades

import (
	"io"
	"path/filepath"
	"testing"

	"opdeals/internal/logx"
)

func TestSlugMatchesStorefront(t *testing.T) {
	cases := map[string]Trade{
		"pokemon-mega-darkrai-ex-101-nm":            {Name: "Mega Darkrai ex", Number: "101", Condition: "NM"},
		"pokemon-pokemon-de-n-s-zoroark-ex-185-en": {Name: "Pokémon de N's Zoroark ex", Number: "185", Variant: "EN"},
		"pokemon-boss-s-orders-tg24":                {Name: "Boss's Orders", Number: "TG24"},
		"pokemon-ceu-escuro-101-084-nm":             {Name: "Céu Escuro", Number: "101/084", Condition: "NM"},
	}
	for want, tr := range cases {
		if got := Slug("pokemon", tr); got != want {
			t.Errorf("Slug(%+v) = %q, want %q", tr, got, want)
		}
	}
}

func TestSlugFrozenAcrossEdits(t *testing.T) {
	st := NewStore(filepath.Join(t.TempDir(), "trades.json"), logx.New(io.Discard))
	added, err := st.Add(Trade{Name: "Mega Darkrai ex", Number: "101/084", Condition: "NM"})
	if err != nil {
		t.Fatal(err)
	}
	updated, err := st.Update(added.ID, func(tr *Trade) {
		*tr = Trade{Name: "Mega Darkrai ex", Number: "101", Condition: "NM"}
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := Slug("pokemon", updated); got != "pokemon-mega-darkrai-ex-101-084-nm" {
		t.Fatalf("slug moved after edit: %q", got)
	}
}
