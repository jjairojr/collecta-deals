package trades

import "strings"

func SKU(gameID string, t Trade) string {
	kind := t.Kind
	if kind == "" {
		kind = "single"
	}
	parts := []string{gameID, t.Name, t.Number, t.Variant, t.Condition}
	for i := range parts {
		parts[i] = strings.Join(strings.Fields(strings.ToLower(parts[i])), " ")
	}
	return kind + "|" + strings.Join(parts, "|")
}
