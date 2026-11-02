package mercadopago

import (
	"slices"
	"testing"
)

func excludedTypes(method Method, installments int) ([]string, int) {
	pm := paymentMethods(method, installments)
	out := make([]string, 0, len(pm.ExcludedPaymentTypes))
	for _, t := range pm.ExcludedPaymentTypes {
		out = append(out, t.ID)
	}
	return out, pm.Installments
}

func TestPaymentMethods(t *testing.T) {
	cases := []struct {
		method       Method
		installments int
		excluded     []string
		allowed      []string
		want         int
	}{
		{MethodPix, 6, []string{"ticket", "credit_card", "debit_card"}, []string{"bank_transfer"}, 1},
		{MethodCard, 6, []string{"ticket", "bank_transfer"}, []string{"credit_card"}, 6},
		{"", 12, []string{"ticket"}, []string{"bank_transfer", "credit_card"}, 12},
	}
	for _, c := range cases {
		got, n := excludedTypes(c.method, c.installments)
		for _, id := range c.excluded {
			if !slices.Contains(got, id) {
				t.Fatalf("%q: %s not excluded (%v)", c.method, id, got)
			}
		}
		for _, id := range c.allowed {
			if slices.Contains(got, id) {
				t.Fatalf("%q: %s excluded (%v)", c.method, id, got)
			}
		}
		if n != c.want {
			t.Fatalf("%q: installments %d want %d", c.method, n, c.want)
		}
	}
}
