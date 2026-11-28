package adtestq

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func sample(budget int64) Test {
	c := Creative{Path: "/x/a.mp4", SHA256: "abc", Size: 10, Kind: "video", Thumb: "/x/a.png", Text: "Reinado Delta chegou", Hook: "preço"}
	return Test{Objective: "seguidores", Audience: "goiania", Creatives: []Creative{c, c, c}, BudgetCents: budget, Days: 4}
}

func newStore(t *testing.T, now *time.Time) *Store {
	t.Helper()
	return NewStoreWithClock(filepath.Join(t.TempDir(), "adtests.json"), func() time.Time { return *now })
}

func TestWeeklyCap(t *testing.T) {
	now := time.Date(2026, 11, 6, 10, 0, 0, 0, time.UTC)
	s := newStore(t, &now)
	// Teto de vendas/seguidores: R$ 950 por 7 dias (3 testes de R$ 300 cabem juntos).
	first, err := s.Add(sample(30000))
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if _, err := s.Add(sample(30000)); err != nil {
			t.Fatalf("3 testes de R$ 300 devem caber: %v", err)
		}
	}
	if _, err := s.Add(sample(WeeklyCapCents - 90000 + 100)); !errors.Is(err, ErrOverBudget) {
		t.Fatalf("passou do teto: %v", err)
	}
	if _, err := s.Decide(first.ID, first.Token, false); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Add(sample(MaxBudgetCents + 100)); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("acima do teto por teste devia ser recusado: %v", err)
	}
	second, err := s.Add(sample(30000))
	if err != nil {
		t.Fatalf("rejeitado devia liberar a verba: %v", err)
	}
	if avail, _ := s.Available(); avail != WeeklyCapCents-90000 {
		t.Fatalf("disponível = %d", avail)
	}
	now = now.Add(BudgetWindow + time.Hour)
	if avail, _ := s.Available(); avail != WeeklyCapCents {
		t.Fatalf("depois de 7 dias = %d", avail)
	}
	if second.Status != StatusPending || second.Token == "" {
		t.Fatalf("teste = %+v", second)
	}
}

func TestValidation(t *testing.T) {
	now := time.Now()
	s := newStore(t, &now)
	bad := []func(*Test){
		func(x *Test) { x.Objective = "alcance" },
		func(x *Test) { x.Audience = "mundo" },
		func(x *Test) { x.BudgetCents = 1000 },
		func(x *Test) { x.Days = 10 },
		func(x *Test) { x.Creatives = x.Creatives[:1] },
		func(x *Test) { x.Objective = "vendas" },
		func(x *Test) { x.Creatives[0].Thumb = "" },
	}
	for i, change := range bad {
		x := sample(5000)
		x.Creatives = append([]Creative(nil), x.Creatives...)
		change(&x)
		if _, err := s.Add(x); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("caso %d aceito: %v", i, err)
		}
	}
}

func TestLifecycle(t *testing.T) {
	now := time.Date(2026, 11, 6, 10, 0, 0, 0, time.UTC)
	s := newStore(t, &now)
	x, err := s.Add(sample(5000))
	if err != nil {
		t.Fatal(err)
	}
	meta := Meta{CampaignID: "c", AdSetID: "s", Ads: []MetaAd{{ID: "a1"}, {ID: "a2", Creative: 1}}}
	if _, err := s.Started(x.ID, meta, now.Add(96*time.Hour)); !errors.Is(err, ErrWrongStatus) {
		t.Fatalf("subiu sem aprovação: %v", err)
	}
	if _, err := s.Decide(x.ID, "errado", true); !errors.Is(err, ErrBadToken) {
		t.Fatal(err)
	}
	if _, err := s.Decide(x.ID, x.Token, true); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Decide(x.ID, x.Token, false); !errors.Is(err, ErrNotPending) {
		t.Fatal("decidiu duas vezes")
	}
	running, err := s.Started(x.ID, meta, now.Add(96*time.Hour))
	if err != nil || running.Status != StatusRunning || running.Token != "" {
		t.Fatalf("started = %+v %v", running, err)
	}
	meta.Ads[0].Paused = true
	if _, err := s.SetMeta(x.ID, meta); err != nil {
		t.Fatal(err)
	}
	done, err := s.Finished(x.ID, "vencedor: a2")
	if err != nil || done.Status != StatusDone || !done.Meta.Ads[0].Paused {
		t.Fatalf("finished = %+v %v", done, err)
	}
	if _, err := s.Failed(x.ID, "x", nil); !errors.Is(err, ErrWrongStatus) {
		t.Fatal("falhou depois de terminar")
	}
}

func messages(budget int64) Test {
	t := sample(budget)
	t.Objective = "mensagens"
	return t
}

func TestMessagesBudgetIsSeparateWeeklyAndMonthly(t *testing.T) {
	now := time.Date(2026, 12, 1, 12, 0, 0, 0, saoPaulo)
	s := newStore(t, &now)
	for _, b := range []int64{30000, 30000, 30000, WeeklyCapCents - 90000} { // vendas: semana cheia (R$ 950)
		if _, err := s.Add(sample(b)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.Add(sample(MinBudgetCents)); !errors.Is(err, ErrOverBudget) {
		t.Fatalf("vendas devia estar cheia: %v", err)
	}
	if _, err := s.Add(messages(25000)); err != nil {
		t.Fatalf("mensagens tem verba própria: %v", err)
	}
	if _, err := s.Add(messages(MinBudgetCents)); !errors.Is(err, ErrOverBudget) {
		t.Fatalf("mensagens passou de R$ 250 na semana: %v", err)
	}
	for i, day := range []int{8, 15, 22} { // R$ 250 por semana até R$ 1.000 no mês
		now = time.Date(2026, 12, day, 13+i, 0, 0, 0, saoPaulo) // um pouco depois dos 7 dias
		if _, err := s.Add(messages(25000)); err != nil {
			t.Fatalf("dia %d: %v", day, err)
		}
	}
	now = time.Date(2026, 12, 30, 12, 0, 0, 0, saoPaulo)
	if left, _ := s.AvailableFor("mensagens"); left != 0 {
		t.Fatalf("mês cheio devia deixar 0, sobra %d", left)
	}
	if _, err := s.Add(messages(MinBudgetCents)); !errors.Is(err, ErrOverBudget) {
		t.Fatalf("mensagens passou de R$ 1.000 no mês: %v", err)
	}
	now = time.Date(2027, 1, 2, 12, 0, 0, 0, saoPaulo)
	if _, err := s.Add(messages(25000)); err != nil {
		t.Fatalf("mês novo libera a verba: %v", err)
	}
}

func TestValidAcceptsLookalikeAndBuyers(t *testing.T) {
	for _, audience := range []string{"semelhante", "clientes"} {
		tt := Test{Objective: "seguidores", Audience: audience, BudgetCents: MinBudgetCents, Days: MinDays, Creatives: []Creative{
			{Kind: "imagem", Path: "a.png", SHA256: "x", Text: "t"}, {Kind: "imagem", Path: "b.png", SHA256: "y", Text: "t"},
		}}
		if !Valid(tt) {
			t.Fatalf("%s devia ser válido", audience)
		}
	}
	if Valid(Test{Objective: "seguidores", Audience: "lookalike", BudgetCents: MinBudgetCents, Days: MinDays}) {
		t.Fatal("público desconhecido passou")
	}
}

func TestValidExistingPost(t *testing.T) {
	post := Test{Objective: "seguidores", Audience: "brasil", BudgetCents: MinBudgetCents, Days: MinDays, Creatives: []Creative{
		{Kind: "post", MediaID: "1790", Hook: "a"}, {Kind: "post", MediaID: "1791", Hook: "b"},
	}}
	if !Valid(post) {
		t.Fatal("post existente sem arquivo nem texto devia ser válido")
	}
	post.Creatives[1].MediaID = ""
	if Valid(post) {
		t.Fatal("post sem id da mídia passou")
	}
}

func TestValidSalesEventAndOpenAudience(t *testing.T) {
	base := Test{Objective: "vendas", Audience: "aberto", Link: "https://x", BudgetCents: MinBudgetCents, Days: MinDays, Creatives: []Creative{
		{Kind: "imagem", Path: "a.png", SHA256: "x", Text: "t"}, {Kind: "imagem", Path: "b.png", SHA256: "y", Text: "t"},
	}}
	for _, ev := range []string{"", "compra", "checkout", "carrinho"} {
		tt := base
		tt.Event = ev
		if !Valid(tt) {
			t.Fatalf("evento %q devia valer", ev)
		}
	}
	bad := base
	bad.Event = "lead"
	if Valid(bad) {
		t.Fatal("evento desconhecido passou")
	}
	follow := base
	follow.Objective, follow.Link, follow.Event = "seguidores", "", "checkout"
	if Valid(follow) {
		t.Fatal("evento fora de vendas passou")
	}
}

func TestHeldUsesRealSpendOnceDone(t *testing.T) {
	spent := int64(1234)
	cases := []struct {
		name string
		t    Test
		want int64
	}{
		{"ativo conta a verba inteira", Test{Status: StatusRunning, BudgetCents: 15000}, 15000},
		{"aprovado conta a verba inteira", Test{Status: StatusApproved, BudgetCents: 15000}, 15000},
		{"terminado com gasto gravado", Test{Status: StatusDone, BudgetCents: 15000, SpentCents: &spent}, 1234},
		{"resumo do executor", Test{Status: StatusDone, BudgetCents: 12000, Summary: "vendas no site, Brasil, interesse em TCG: R$ 116.29 gastos de R$ 120,00, medido em visitas"}, 11629},
		// Os dois resumos escritos à mão em 28/11 (A38A04 e F1A434).
		{"resumo à mão A38A04", Test{Status: StatusDone, BudgetCents: 15000, Summary: `Encerrado pelo Jairo em 27/11/2026 ("desliga pra mim mano"): campanha 120252903149630671 pausada pelo Instalador às 22:05 pela API do Meta. Gasto real segundo o Meta: R$ 75,06 (24 a 27/11).`}, 7506},
		{"resumo à mão F1A434", Test{Status: StatusDone, BudgetCents: 30000, Summary: `Encerrado em 28/11/2026 às 12:56 pelo Instalador: campanha 120252966561440671 pausada pela API do Meta. Gasto real segundo o Meta: R$ 0,13 (24 impressões, 27 a 28/11).`}, 13},
		{"milhar", Test{Status: StatusDone, BudgetCents: 30000, Summary: "Gasto real segundo o Meta: R$ 1.250,50"}, 30000},
		{"terminado sem gasto conhecido", Test{Status: StatusDone, BudgetCents: 15000, Summary: "vencedor: a2"}, 15000},
	}
	for _, c := range cases {
		if got := Held(c.t); got != c.want {
			t.Errorf("%s: %d, quer %d", c.name, got, c.want)
		}
	}
}

// Caso de 28/11: A38A04 (R$ 75,06 de R$ 150) e F1A434 (R$ 0,13 de R$ 300)
// terminados seguravam R$ 450 e só sobravam R$ 50 na semana.
func TestAvailableFreesBudgetOfFinishedTests(t *testing.T) {
	now := time.Date(2026, 11, 28, 13, 0, 0, 0, saoPaulo)
	at := now.Add(-24 * time.Hour)
	a, f := int64(7506), int64(13)
	all := []Test{
		{Objective: "vendas", Status: StatusDone, BudgetCents: 15000, SpentCents: &a, CreatedAt: at},
		{Objective: "vendas", Status: StatusDone, BudgetCents: 30000, SpentCents: &f, CreatedAt: at},
		{Objective: "vendas", Status: StatusRunning, BudgetCents: 15000, CreatedAt: at},
		{Objective: "vendas", Status: StatusRunning, BudgetCents: 30000, CreatedAt: at},
		{Objective: "mensagens", Status: StatusRunning, BudgetCents: 15000, CreatedAt: at},
	}
	want := int64(WeeklyCapCents - (7506 + 13 + 15000 + 30000))
	if got := available(all, now, "vendas"); got != want {
		t.Fatalf("sobra = %d, quer %d", got, want)
	}
}

func TestFinishedSpentRecordsSpend(t *testing.T) {
	now := time.Date(2026, 11, 6, 10, 0, 0, 0, time.UTC)
	s := newStore(t, &now)
	x, err := s.Add(sample(5000))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Decide(x.ID, x.Token, true); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Started(x.ID, Meta{CampaignID: "c", AdSetID: "s", Ads: []MetaAd{{ID: "a1"}, {ID: "a2", Creative: 1}}}, now.Add(96*time.Hour)); err != nil {
		t.Fatal(err)
	}
	spent := int64(4321)
	done, err := s.FinishedSpent(x.ID, "fim", &spent)
	if err != nil || done.SpentCents == nil || *done.SpentCents != 4321 || Held(done) != 4321 {
		t.Fatalf("terminado = %+v %v", done, err)
	}
}
