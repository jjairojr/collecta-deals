package api

import "testing"

// O Cadastro manda kind "single" no POST. O livro guarda carta avulsa com kind
// vazio (como o PUT já faz), senão ela some do site.
func TestCreateTradeSingleKindIsStoredEmpty(t *testing.T) {
	s, _, _ := twoGameServer(t)
	created := createTrade(t, s, "pokemon", `{"kind":"single","number":"170","name":"Squirtle","set":"SV3a","condition":"PSA 10","qty":1,"askBRL":3100,"status":"holding"}`)
	if created.Kind != "" {
		t.Fatalf("kind = %q, want vazio", created.Kind)
	}
	if created.RefUSD != 0 {
		t.Fatalf("slab recebeu preço de referência da carta crua: %.2f", created.RefUSD)
	}
}
