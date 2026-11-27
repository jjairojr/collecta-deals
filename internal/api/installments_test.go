package api

import (
	"testing"

	"opdeals/internal/orders"
)

// Até 6x sem juros no cartão em todo pedido, com ou sem preço no cartão; Pix 1x.
func TestInstallmentsCapAtSix(t *testing.T) {
	card := orders.Order{PaymentMethod: orders.PaymentCard, Lines: []orders.Line{{SKU: "a", Qty: 1, UnitBRL: 900}}}
	cardPriced := orders.Order{PaymentMethod: orders.PaymentCard, Lines: []orders.Line{{SKU: "b", Qty: 1, UnitBRL: 660, CardPriced: true}}}
	for name, o := range map[string]orders.Order{
		"cartão":                   card,
		"cartão com preço próprio": cardPriced,
		"qualquer":                 {PaymentMethod: orders.PaymentAny},
	} {
		if got := installments(o); got != 6 {
			t.Errorf("%s: %d parcelas, quero 6", name, got)
		}
	}
	if got := installments(orders.Order{PaymentMethod: orders.PaymentPix}); got != 1 {
		t.Errorf("pix: %d parcelas, quero 1", got)
	}
}
