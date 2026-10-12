package shipping

import (
	"fmt"
	"regexp"
)

type Item struct {
	Kind    string
	Name    string
	Qty     int
	UnitBRL float64
}

type box struct {
	width, height, length int
	weight                float64
}

var (
	singlesEnvelope = box{11, 16, 2, 0.05}
	singleWeight    = 0.01
	sealedDefault   = box{15, 20, 10, 0.5}
	accessoryBox    = box{11, 16, 5, 0.25}
	playmatTube     = box{8, 36, 8, 0.35}
)

var sealedBoxes = []struct {
	re  *regexp.Regexp
	box box
}{
	{regexp.MustCompile(`(?i)elite trainer box|\betb\b`), box{14, 21, 9, 0.7}},
	{regexp.MustCompile(`(?i)booster bundle`), box{12, 17, 6, 0.4}},
	{regexp.MustCompile(`(?i)caixa de booster|booster box|box display`), box{12, 16, 9, 0.9}},
	{regexp.MustCompile(`(?i)blister`), box{11, 16, 3, 0.1}},
	{regexp.MustCompile(`(?i)\bdeck\b`), box{11, 16, 4, 0.2}},
}

var playmatRe = regexp.MustCompile(`(?i)playmat`)

func Pack(items []Item) []Product {
	out := make([]Product, 0, len(items)+1)
	singles := 0
	singlesValue := 0.0
	for i, it := range items {
		qty := it.Qty
		if qty <= 0 {
			qty = 1
		}
		switch it.Kind {
		case "sealed":
			b := sealedBox(it.Name)
			out = append(out, product(fmt.Sprintf("sealed-%d", i), b, it.UnitBRL, qty))
		case "accessory":
			b := accessoryBox
			if playmatRe.MatchString(it.Name) {
				b = playmatTube
			}
			out = append(out, product(fmt.Sprintf("accessory-%d", i), b, it.UnitBRL, qty))
		default:
			singles += qty
			singlesValue += it.UnitBRL * float64(qty)
		}
	}
	if singles > 0 {
		b := singlesEnvelope
		b.weight += singleWeight * float64(singles)
		out = append(out, product("singles", b, singlesValue, 1))
	}
	return out
}

func sealedBox(name string) box {
	for _, c := range sealedBoxes {
		if c.re.MatchString(name) {
			return c.box
		}
	}
	return sealedDefault
}

func product(id string, b box, insurance float64, qty int) Product {
	return Product{
		ID:             id,
		Width:          b.width,
		Height:         b.height,
		Length:         b.length,
		Weight:         b.weight,
		InsuranceValue: insurance,
		Quantity:       qty,
	}
}
