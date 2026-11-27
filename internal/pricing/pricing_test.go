package pricing

import "testing"

func TestCard(t *testing.T) {
	cases := []struct{ pix, registered, want float64 }{
		{8300, 0, 9545},        // Charizard PSA 10 sem cardBRL
		{8300, 9222.22, 9545},  // cadastrado com +11%: sobe para +15%
		{8300, 9600, 9600},     // cadastrado acima do piso: mantém
		{289.9, 309.9, 333.39}, // Display Sylveon: sobe para o piso
		{600, 660, 690},        // Gem Pack: +10% sobe para +15%
		{3.47, 0, 3.99},
		{0, 0, 0},
	}
	for _, c := range cases {
		if got := Card(c.pix, c.registered); got != c.want {
			t.Errorf("Card(%v, %v) = %v, want %v", c.pix, c.registered, got, c.want)
		}
	}
}
