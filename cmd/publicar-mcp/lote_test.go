package main

import (
	"strings"
	"testing"
	"time"

	"opdeals/internal/adtestq"
)

const lotePhrase = "Aprovo a verba da semana de 02/11 a 08/11: R$ 950. Venda só de Pokémon. CPA máximo: R$ 80 por compra (e R$ 15 por conversa na compra de cartas)"

func TestParseLote(t *testing.T) {
	now := time.Date(2026, 11, 6, 10, 0, 0, 0, saoPaulo) // sexta
	l, err := parseLote(lotePhrase, now)
	if err != nil {
		t.Fatal(err)
	}
	if l.From != "2026-11-02" || l.To != "2026-11-08" || l.BudgetReai != 950 || l.Product != "Pokémon" || l.CPAReais != 80 || l.ChatReais != 15 {
		t.Fatalf("lote = %+v", l)
	}
	bad := map[string]string{
		"Aprovo a verba da semana de 03/11 a 09/11: R$ 500. Venda só de Pokémon. CPA máximo: R$ 80 por compra":                       "segunda a domingo",
		"Aprovo a verba da semana de 26/10 a 01/11: R$ 500. Venda só de Pokémon. CPA máximo: R$ 80 por compra":                       "já passou",
		"Aprovo a verba da semana de 02/11 a 08/11: R$ 1.000. Venda só de Pokémon. CPA máximo: R$ 80 por compra":                     "teto da semana",
		"Aprovo a verba da semana de 02/11 a 08/11: R$ 500. Venda só de Pokémon.":                                                    "CPA máximo",
		"Aprovo a verba da semana de 02/11 a 08/11: R$ 500. CPA máximo: R$ 80 por compra":                                            "Venda só de",
		"Não aprovo a verba da semana de 02/11 a 08/11: R$ 500. Venda só de Pokémon. CPA máximo: R$ 80 por compra":                   "negação",
		"Aprovo a verba da semana de 02/11 a 08/11: R$ 500 e aprovo a verba da semana de 09/11 a 15/11: R$ 400. Venda só de Pokémon": "mais de uma semana",
		"aprova aí a semana": "formato",
	}
	for phrase, want := range bad {
		if _, err := parseLote(phrase, now); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: devia recusar com %q: %v", phrase, want, err)
		}
	}
	// Em dezembro, "04/01" é janeiro do ano seguinte.
	if l, err := parseLote("Aprovo a verba da semana de 04/01 a 10/01: R$ 300. Venda só de One Piece. CPA máximo: R$ 50,00 por compra", time.Date(2026, 12, 30, 9, 0, 0, 0, saoPaulo)); err != nil || l.From != "2027-01-04" {
		t.Fatalf("virada do ano = %+v %v", l, err)
	}
}

func gravaLote(t *testing.T, f *fixture, db *chatDB, phrase string) loteOutput {
	t.Helper()
	db.jairo(t, phrase)
	out, errText := callTool[loteOutput](t, f, "aprovar_lote_semanal", nil)
	if errText != "" {
		t.Fatal(errText)
	}
	return out
}

func TestLoteNeedsJairosOwnPhrase(t *testing.T) {
	f, _, _ := newAdFixture(t)
	db := newChatDB(t, f)
	db.ceo(t, lotePhrase)
	db.add(t, "t-ceo", "user", "routine", "routine", lotePhrase)
	db.add(t, "t-social", "user", "", "user", lotePhrase)
	if _, errText := callTool[loteOutput](t, f, "aprovar_lote_semanal", nil); !strings.Contains(errText, "não achei frase de lote do Jairo") {
		t.Fatalf("agente, rotina ou outro chat não valem: %s", errText)
	}
	db.jairo(t, strings.Replace(lotePhrase, "R$ 950", "R$ 1.200", 1))
	if _, errText := callTool[loteOutput](t, f, "aprovar_lote_semanal", nil); !strings.Contains(errText, "teto da semana") {
		t.Fatalf("acima do teto devia recusar: %s", errText)
	}
	db.jairo(t, lotePhrase)
	chk, errText := callTool[loteOutput](t, f, "aprovar_lote_semanal", map[string]any{"so_conferir": true})
	if errText != "" || chk.Gravado || chk.Verba != "R$ 950,00" {
		t.Fatalf("só conferir: %+v %s", chk, errText)
	}
	if l, _ := f.app.currentLote(); l != nil {
		t.Fatal("só conferir não grava o lote")
	}
	if _, errText := callTool[loteOutput](t, f, "aprovar_lote_semanal", map[string]any{"mensagem_id": "m1"}); !strings.Contains(errText, "não é do Jairo") {
		t.Fatalf("mensagem do agente pelo id não vale: %s", errText)
	}
	out := gravaLote(t, f, db, lotePhrase)
	if !out.Gravado || out.Verba != "R$ 950,00" || out.Sobra != "R$ 950,00" || out.De != "02/11/2026" || out.Ate != "08/11/2026" || out.Produto != "Pokémon" {
		t.Fatalf("lote = %+v", out)
	}
}

func TestLoteApprovesTestsThatFit(t *testing.T) {
	f, _, _ := newAdFixture(t)
	db := newChatDB(t, f)
	gravaLote(t, f, db, strings.Replace(lotePhrase, "R$ 950", "R$ 500", 1))

	first, errText := callTool[proposeOutput](t, f, "propor_teste", proposeArgs(f.creatives(t), 150))
	if errText != "" || !first.PeloLote || !strings.Contains(first.Proximo, "lote da semana") {
		t.Fatalf("150 cabe no lote: %+v %s", first, errText)
	}
	second, _ := callTool[proposeOutput](t, f, "propor_teste", proposeArgs(f.creatives(t), 300))
	if !second.PeloLote {
		t.Fatalf("150 + 300 cabe em 500: %+v", second)
	}
	third, _ := callTool[proposeOutput](t, f, "propor_teste", proposeArgs(f.creatives(t), 100))
	if third.PeloLote || third.Codigo == "" || !strings.Contains(third.Proximo, "passa do lote") {
		t.Fatalf("550 passa de 500 e pede código: %+v", third)
	}
	approved, _ := f.ads.List(adtestq.StatusApproved)
	pending, _ := f.ads.List(adtestq.StatusPending)
	if len(approved) != 2 || len(pending) != 1 || pending[0].ID != third.ID {
		t.Fatalf("aprovados %d, pendentes %+v", len(approved), pending)
	}
	var lotePushes int
	for _, p := range f.pushed() {
		if p.title == "Teste aprovado pelo lote" {
			lotePushes++
		}
	}
	if lotePushes != 2 {
		t.Fatalf("aviso ao dono de cada aprovação pelo lote: %+v", f.pushed())
	}
	w, errText := callTool[waitingOutput](t, f, "esperando_jairo", nil)
	if errText != "" || w.Lote == nil || w.Lote.Usado != "R$ 450,00" || w.Lote.Sobra != "R$ 50,00" || len(w.PeloLote) != 2 || len(w.Itens) != 1 || w.Itens[0].Codigo != third.Codigo {
		t.Fatalf("esperando_jairo = %+v %s", w, errText)
	}
}

func TestLoteKeepsLimits(t *testing.T) {
	f, _, _ := newAdFixture(t)
	db := newChatDB(t, f)
	gravaLote(t, f, db, lotePhrase) // R$ 950
	for i := 0; i < 3; i++ {
		if out, _ := callTool[proposeOutput](t, f, "propor_teste", proposeArgs(f.creatives(t), 300)); !out.PeloLote {
			t.Fatalf("teste %d devia caber: %+v", i+1, out)
		}
	}
	fourth, _ := callTool[proposeOutput](t, f, "propor_teste", proposeArgs(f.creatives(t), 30))
	if fourth.PeloLote || !strings.Contains(fourth.Proximo, "3 testes aprovados ou no ar") {
		t.Fatalf("o quarto ao mesmo tempo pede código: %+v", fourth)
	}
	msg := proposeArgs(f.creatives(t), 50)
	msg["objetivo"] = "mensagens"
	if out, _ := callTool[proposeOutput](t, f, "propor_teste", msg); out.PeloLote || !strings.Contains(out.Proximo, "verba própria") {
		t.Fatalf("compra de cartas não entra no lote: %+v", out)
	}
}

func TestNoLoteStillAsksCode(t *testing.T) {
	f, _, _ := newAdFixture(t)
	newChatDB(t, f)
	out, errText := callTool[proposeOutput](t, f, "propor_teste", proposeArgs(f.creatives(t), 150))
	if errText != "" || out.PeloLote || out.Codigo == "" || strings.Contains(out.Proximo, "lote") {
		t.Fatalf("sem lote, segue o código: %+v %s", out, errText)
	}
	// Lote de outra semana não vale para esta.
	db := newChatDB(t, f)
	gravaLote(t, f, db, "Aprovo a verba da semana de 09/11 a 15/11: R$ 500. Venda só de Pokémon. CPA máximo: R$ 80 por compra")
	if out, _ := callTool[proposeOutput](t, f, "propor_teste", proposeArgs(f.creatives(t), 100)); out.PeloLote {
		t.Fatalf("lote da semana que vem não aprova teste de hoje: %+v", out)
	}
}
