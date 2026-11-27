// Package pricing holds the store's card-price rule, shared by the checkout, the
// kits and the storefront API so every place charges and shows the same value.
package pricing

import "math"

// CardMarkupPercent is how much more the card (installments) costs than Pix, at
// least. Jairo, 27/11/2026: "em vez de 10% coloca 15% a mais"; card ≥ Pix × 1,15.
const CardMarkupPercent = 15

// CardFloor is the lowest card price allowed for a Pix price, in reais, rounded
// to cents (R$ 8.300,00 → R$ 9.545,00). Computed in whole cents so a price like
// R$ 289,90 rounds the way a person would (R$ 333,39), not by float noise.
func CardFloor(pix float64) float64 {
	if pix <= 0 {
		return 0
	}
	pixCents := math.Round(pix * 100)
	return math.Round(pixCents*(100+CardMarkupPercent)/100) / 100
}

// Card is the card price actually charged: the registered card price when it is
// already at least the floor, otherwise the floor. An empty (0) registered card
// price means "not set" and falls to the floor too. Since every stock unit gets
// this floor, the cheapest card price is never below Pix × 1,15 of the cheapest
// Pix price.
func Card(pix, registered float64) float64 {
	return math.Max(registered, CardFloor(pix))
}
