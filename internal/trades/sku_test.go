package trades

import "testing"

func TestSKU(t *testing.T) {
	cases := []struct {
		name string
		game string
		in   Trade
		want string
	}{
		{"single", "onepiece", Trade{Name: "Monkey.D.Luffy", Number: "OP01-003", Variant: "SP", Condition: "NM"}, "single|onepiece|monkey.d.luffy|op01-003|sp|nm"},
		{"spaces and case collapse", "onepiece", Trade{Name: "  Monkey  D  LUFFY ", Number: "op01-003"}, "single|onepiece|monkey d luffy|op01-003||"},
		{"sealed", "pokemon", Trade{Kind: "sealed", Name: "(ING) Booster Box - Prismatic"}, "sealed|pokemon|(ing) booster box - prismatic|||"},
		{"accessory", "acessorios", Trade{Kind: KindAccessory, Name: "Sleeves Dragon Shield"}, "accessory|acessorios|sleeves dragon shield|||"},
	}
	for _, c := range cases {
		if got := SKU(c.game, c.in); got != c.want {
			t.Errorf("%s: got %q want %q", c.name, got, c.want)
		}
	}
}
