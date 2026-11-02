package postsale

import (
	"encoding/json"
	"testing"

	"opdeals/internal/conversations"
)

func TestHistoryMessagesAlternateRoles(t *testing.T) {
	msgs := historyMessages([]Turn{
		{Role: conversations.RoleAssistant, Text: "oferta"},
		{Role: conversations.RoleCustomer, Text: "oi"},
		{Role: conversations.RoleCustomer, Text: "tem zoro?"},
		{Role: conversations.RoleSystem, Text: "nota"},
		{Role: conversations.RoleOwner, Text: "tem sim"},
		{Role: conversations.RoleAssistant, Text: "link"},
	})
	roles := make([]string, 0, len(msgs))
	for _, m := range msgs {
		body, err := json.Marshal(m)
		if err != nil {
			t.Fatal(err)
		}
		var v struct {
			Role string `json:"role"`
		}
		if err := json.Unmarshal(body, &v); err != nil {
			t.Fatal(err)
		}
		roles = append(roles, v.Role)
	}
	for i := 1; i < len(roles); i++ {
		if roles[i] == roles[i-1] {
			t.Fatalf("consecutive %s at %d: %v", roles[i], i, roles)
		}
	}
	if roles[0] != "user" {
		t.Fatalf("first role %s", roles[0])
	}
}

func TestFormatBRL(t *testing.T) {
	for in, want := range map[float64]string{300: "300,00", 1234.5: "1.234,50", 9.99: "9,99", 1000000: "1.000.000,00"} {
		if got := FormatBRL(in); got != want {
			t.Errorf("FormatBRL(%v) = %q, want %q", in, got, want)
		}
	}
}
