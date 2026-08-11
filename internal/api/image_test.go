package api

import (
	"io"
	"path/filepath"
	"testing"

	"opdeals/internal/game"
	"opdeals/internal/logx"
	"opdeals/internal/trades"
)

func TestAllowedImageURL(t *testing.T) {
	cases := []struct {
		url  string
		want bool
	}{
		{"https://repositorio.sbrauble.com/arquivos/in/pokemon/cd/1/abc.jpg", true},
		{"https://tcgplayer-cdn.tcgplayer.com/product/684395_in_1000x1000.jpg", true},
		{"http://repositorio.sbrauble.com/arquivos/in/pokemon/cd/1/abc.jpg", false},
		{"https://evil.example.com/x.jpg", false},
		{"https://169.254.169.254/latest/meta-data", false},
		{"", false},
	}
	for _, c := range cases {
		if got := allowedImageURL(c.url); got != c.want {
			t.Errorf("allowedImageURL(%q) = %v, want %v", c.url, got, c.want)
		}
	}
}

func TestLedgerImageURL(t *testing.T) {
	store := trades.NewStore(filepath.Join(t.TempDir(), "trades.json"), logx.New(io.Discard))
	gs := &GameStack{Game: game.Game{ID: "onepiece"}, Trades: store}
	s := &Server{games: map[string]*GameStack{"onepiece": gs}, defaultGame: "onepiece"}

	held, err := store.Add(trades.Trade{Number: "OP16-080", Name: "Teach", Set: "OP16", Qty: 1, Status: "holding"})
	if err != nil {
		t.Fatal(err)
	}
	art := "https://loja-qualquer.com.br/img/teach.jpg"
	if _, err := store.SetListings(map[string]trades.Listing{held.ID: {AskBRL: 250, Listed: true, ImageURL: art}}); err != nil {
		t.Fatal(err)
	}

	if !s.ledgerImageURL(gs, art) {
		t.Errorf("art saved on a holding must be proxyable")
	}
	if s.ledgerImageURL(gs, "https://loja-qualquer.com.br/img/outra.jpg") {
		t.Errorf("URL in no ledger must be refused")
	}
	if s.ledgerImageURL(gs, "http://loja-qualquer.com.br/img/teach.jpg") {
		t.Errorf("non-https must be refused")
	}
}
