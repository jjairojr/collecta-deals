package trades

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

func slugify(s string) string {
	var b strings.Builder
	dash := false
	for _, r := range norm.NFD.String(strings.ToLower(s)) {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			dash = false
			continue
		}
		if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.TrimSuffix(b.String(), "-")
}

func SlugBase(t Trade) string {
	parts := make([]string, 0, 4)
	for _, p := range []string{t.Name, t.Number, t.Variant, t.Condition} {
		if p != "" {
			parts = append(parts, p)
		}
	}
	return slugify(strings.Join(parts, "-"))
}

func Slug(gameID string, t Trade) string {
	base := t.Slug
	if base == "" {
		base = SlugBase(t)
	}
	return slugify(gameID + "-" + base)
}
