package metaevents

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestHashIsLowerHexSHA256(t *testing.T) {
	if got := Hash("abc"); got != "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad" {
		t.Fatalf("Hash = %s", got)
	}
}

func TestNormalization(t *testing.T) {
	cases := []struct{ name, got, want string }{
		{"email", Email("  John_Smith@Gmail.com "), "john_smith@gmail.com"},
		{"email inválido", Email("joao arroba gmail"), ""},
		{"celular com DDD", Phone("(62) 99339-9980"), "5562993399980"},
		{"já com 55", Phone("+55 62 99339-9980"), "5562993399980"},
		{"zero à esquerda", Phone("062 3333-4444"), "556233334444"},
		{"curto demais", Phone("12345"), ""},
		{"cidade", Plain("Goiânia"), "goiania"},
		{"cidade com espaço", Plain("São Paulo"), "saopaulo"},
		{"estado", Plain("GO"), "go"},
		{"cep", Digits("74000-000"), "74000000"},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s: %q, quer %q", c.name, c.got, c.want)
		}
	}
	if f, l := SplitName("  Maria das Graças  Souza "); f != "maria" || l != "souza" {
		t.Errorf("SplitName = %q %q", f, l)
	}
}

// O evento sai só com hash nos campos de identidade: nenhum texto do cliente.
func TestUserHasOnlyHashes(t *testing.T) {
	p := Person{Name: "Maria Souza", Email: "maria@x.com", Phone: "62993399980", City: "Goiânia", State: "GO", Zip: "74000-000"}
	data, _ := json.Marshal(p.User())
	s := string(data)
	for _, raw := range []string{"maria", "Maria", "993399980", "goiania", "74000", `"go"`} {
		if strings.Contains(s, raw) {
			t.Errorf("dado em texto %q no evento: %s", raw, s)
		}
	}
	u := p.User()
	if u.Email[0] != Hash("maria@x.com") || u.Phone[0] != Hash("5562993399980") || u.Country[0] != Hash("br") || u.State[0] != Hash("go") {
		t.Fatalf("hashes = %+v", u)
	}
	if !u.Matchable() || (Person{Name: "Só Nome"}).User().Matchable() {
		t.Fatal("Matchable errado")
	}
	row, ok := p.Row()
	if !ok || len(row) != len(CustomerSchema) || row[0] != Hash("maria@x.com") || row[7] != Hash("br") {
		t.Fatalf("linha = %v", row)
	}
	if _, ok := (Person{Name: "Sem Contato"}).Row(); ok {
		t.Fatal("sem e-mail nem telefone não entra na lista")
	}
}
