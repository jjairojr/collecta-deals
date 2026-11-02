package postsale

import (
	"sort"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

func Fold(s string) string {
	var b strings.Builder
	for _, r := range norm.NFD.String(strings.ToLower(s)) {
		if !unicode.Is(unicode.Mn, r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func Search(items []Item, query string, limit int) ([]Item, []Item) {
	words := Words(Fold(query))
	type scored struct {
		item  Item
		score int
	}
	var hits []scored
	for _, it := range items {
		hay := Fold(it.Name + " " + it.Set + " " + it.Number + " " + it.Meta)
		n := 0
		for _, w := range words {
			if strings.Contains(hay, w) {
				n++
			}
		}
		if n > 0 {
			hits = append(hits, scored{it, n})
		}
	}
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].score != hits[j].score {
			return hits[i].score > hits[j].score
		}
		return hits[i].item.UnitBRL > hits[j].item.UnitBRL
	})
	exact := []Item{}
	similar := []Item{}
	for _, h := range hits {
		if h.score == len(words) && len(exact) < limit {
			exact = append(exact, h.item)
		} else if h.score < len(words) && len(similar) < limit {
			similar = append(similar, h.item)
		}
	}
	return exact, similar
}
