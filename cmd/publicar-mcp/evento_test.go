package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"opdeals/internal/adtestq"
)

func salesArgs(f *fixture, t *testing.T, event string) map[string]any {
	args := proposeArgs(f.creatives(t), 100)
	args["objetivo"] = "vendas"
	args["link"] = "https://www.collectatcg.com.br/selado/pokemon"
	if event != "" {
		args["evento"] = event
	}
	return args
}

// O evento de otimização é escolhido por teste de vendas; sem ele, segue compra.
func TestSalesOptimizationEventPerTest(t *testing.T) {
	for _, c := range []struct{ event, stored, pixel, label string }{
		{"", "", "PURCHASE", "Otimiza: compra"},
		{"compra", "", "PURCHASE", "Otimiza: compra"},
		{"checkout", "checkout", "INITIATED_CHECKOUT", "Otimiza: início do checkout"},
		{"Carrinho", "carrinho", "ADD_TO_CART", "Otimiza: adição ao carrinho"},
	} {
		f, meta, ex := newAdFixture(t)
		if _, errText := callTool[proposeOutput](t, f, "propor_teste", salesArgs(f, t, c.event)); errText != "" {
			t.Fatalf("%q: %s", c.event, errText)
		}
		list, _ := f.ads.List("")
		if len(list) != 1 || list[0].Event != c.stored {
			t.Fatalf("%q: evento guardado = %+v", c.event, list)
		}
		if p := f.pushed(); len(p) != 1 || !strings.Contains(p[0].message, c.label) {
			t.Fatalf("%q: aviso = %+v", c.event, p)
		}
		approveAds(t, f)
		if err := ex.run(context.Background()); err != nil {
			t.Fatal(err)
		}
		set := meta.find("act_1/adsets")
		if len(set) != 1 || !strings.Contains(set[0].form["promoted_object"], `"custom_event_type":"`+c.pixel+`"`) {
			t.Fatalf("%q: conjunto = %+v", c.event, set)
		}
	}
}

func TestSalesEventRefused(t *testing.T) {
	f, _, _ := newAdFixture(t)
	follow := proposeArgs(f.creatives(t), 100)
	follow["evento"] = "checkout"
	if _, errText := callTool[proposeOutput](t, f, "propor_teste", follow); !strings.Contains(errText, "só existe em vendas") {
		t.Fatalf("seguidores com evento: %q", errText)
	}
	if _, errText := callTool[proposeOutput](t, f, "propor_teste", salesArgs(f, t, "lead")); !strings.Contains(errText, "não existe") {
		t.Fatalf("evento desconhecido: %q", errText)
	}
	if list, _ := f.ads.List(""); len(list) != 0 {
		t.Fatalf("nada devia ser criado: %+v", list)
	}
}

// Otimizando para checkout, o teste é julgado pelos checkouts, mas o resumo
// continua com as compras e o custo por compra.
func TestCheckoutTestKeepsCostPerPurchase(t *testing.T) {
	var rows []insightRow
	if err := json.Unmarshal([]byte(`[
		{"ad_id":"a","spend":"40","clicks":"90","actions":[{"action_type":"omni_initiated_checkout","value":"8"},{"action_type":"omni_purchase","value":"2"}]},
		{"ad_id":"b","spend":"20","clicks":"50","actions":[{"action_type":"omni_initiated_checkout","value":"1"}]}]`), &rows); err != nil {
		t.Fatal(err)
	}
	tt := adtestq.Test{Objective: "vendas", Audience: "brasil", Event: "checkout", BudgetCents: 6000,
		Creatives: []adtestq.Creative{{Hook: "preço"}, {Hook: "raridade"}}}
	metric, count := measureTest(tt, rows)
	if metric != "checkouts iniciados" || count(rows[0]) != 8 {
		t.Fatalf("métrica = %q %v", metric, count(rows[0]))
	}
	results := []adResult{
		{index: 0, spend: 40, results: count(rows[0]), purchases: purchasesOf(rows[0])},
		{index: 1, spend: 20, results: count(rows[1]), purchases: purchasesOf(rows[1])},
	}
	meta := adtestq.Meta{Ads: []adtestq.MetaAd{{ID: "a", Creative: 0}, {ID: "b", Creative: 1}}}
	s := summarize(tt, meta, results, metric)
	if !strings.Contains(s, "medido em checkouts iniciados") || !strings.Contains(s, "Compras: 2, R$ 30.00 por compra.") || !strings.Contains(s, "Vencedor: 1. preço") {
		t.Fatalf("resumo = %s", s)
	}
	// Sem checkout nenhum, cai na régua de sempre (compras, visitas, cliques).
	if metric, _ := measureTest(tt, []insightRow{{AdID: "a", Spend: "10", Clicks: "5"}}); metric != "cliques" {
		t.Fatalf("sem checkout = %q", metric)
	}
}

func TestOpenAudience(t *testing.T) {
	f, _, ex := newAdFixture(t)
	tg, err := ex.targeting(context.Background(), "aberto")
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(tg)
	s := string(data)
	for _, want := range []string{`"countries":["BR"]`, `"advantage_audience":1`, `"age_min":18`, `"age_max":65`,
		`"publisher_platforms":["facebook","instagram"]`, `"instagram_positions":["stream","story","reels"]`} {
		if !strings.Contains(s, want) {
			t.Errorf("falta %s em %s", want, s)
		}
	}
	for _, not := range []string{"flexible_spec", "custom_audiences", "custom_locations"} {
		if strings.Contains(s, not) {
			t.Errorf("aberto não leva %s: %s", not, s)
		}
	}
	args := proposeArgs(f.creatives(t), 100)
	args["publico"] = "aberto"
	if _, errText := callTool[proposeOutput](t, f, "propor_teste", args); errText != "" {
		t.Fatalf("aberto não precisa de id no anuncios.env: %s", errText)
	}
	if p := f.pushed(); len(p) != 1 || !strings.Contains(p[0].message, "Aberto (Brasil, Advantage+ público)") {
		t.Fatalf("aviso = %+v", p)
	}
}
