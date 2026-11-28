package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"opdeals/internal/adtestq"
)

func approvedTest(t *testing.T, f *fixture) adtestq.Test {
	t.Helper()
	if _, errText := callTool[proposeOutput](t, f, "propor_teste", proposeArgs(f.creatives(t), 150)); errText != "" {
		t.Fatal(errText)
	}
	approveAds(t, f)
	list, _ := f.ads.List(adtestq.StatusApproved)
	if len(list) != 1 {
		t.Fatalf("aprovados = %+v", list)
	}
	return list[0]
}

func (m *fakeMeta) campaigns() []*fakeObject {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []*fakeObject
	for _, o := range m.objects {
		if o.kind == "campaign" {
			out = append(out, o)
		}
	}
	return out
}

// Caso de 28/11 (teste 0ab7fc2cbdfff8e2): o painel respondeu 502 ao /started
// depois que a campanha estava no ar. Não pode nascer uma segunda campanha.
func TestStartedFailureDoesNotDuplicateCampaign(t *testing.T) {
	f, meta, ex := newAdFixture(t)
	ex.startedWait = time.Millisecond
	test := approvedTest(t, f)
	f.failStarted = startedTries // o painel fica fora a rodada inteira
	if err := ex.run(context.Background()); err == nil || !strings.Contains(err.Error(), "502") {
		t.Fatalf("a rodada devia acusar o 502: %v", err)
	}
	cs := meta.campaigns()
	if len(cs) != 1 || cs[0].status != "PAUSED" {
		t.Fatalf("uma campanha, pausada até registrar: %+v", cs)
	}
	if still, _ := f.ads.List(adtestq.StatusApproved); len(still) != 1 {
		t.Fatalf("sem registro, o teste segue aprovado: %+v", still)
	}
	// Próxima rodada, painel de volta: reaproveita a mesma campanha.
	if err := ex.run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if cs := meta.campaigns(); len(cs) != 1 || cs[0].status != "ACTIVE" {
		t.Fatalf("não pode criar outra; a mesma volta a ficar ativa: %+v", cs)
	}
	running, _ := f.ads.List(adtestq.StatusRunning)
	if len(running) != 1 || running[0].ID != test.ID || running[0].Meta.CampaignID != meta.campaigns()[0].id || len(running[0].Meta.Ads) != 3 || running[0].EndsAt.IsZero() {
		t.Fatalf("o painel devia registrar a campanha reaproveitada: %+v", running)
	}
	for i, ad := range running[0].Meta.Ads {
		if ad.Creative != i {
			t.Fatalf("criativo do anúncio %d = %d", i, ad.Creative)
		}
	}
	log, _ := os.ReadFile(filepath.Join(f.data, "anuncios.log"))
	if !strings.Contains(string(log), "pausada até a próxima rodada registrar") || !strings.Contains(string(log), "reaproveitada "+test.ID) {
		t.Fatalf("log = %s", log)
	}
}

func TestStartedRetrySucceedsInTheSameRound(t *testing.T) {
	f, meta, ex := newAdFixture(t)
	ex.startedWait = time.Millisecond
	approvedTest(t, f)
	f.failStarted = 1
	if err := ex.run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if cs := meta.campaigns(); len(cs) != 1 || cs[0].status != "ACTIVE" {
		t.Fatalf("campanhas = %+v", cs)
	}
	if running, _ := f.ads.List(adtestq.StatusRunning); len(running) != 1 {
		t.Fatalf("a segunda tentativa registra: %+v", running)
	}
}

// Uma subida interrompida no meio (sem conjunto) deixa uma campanha pausada: ela
// é pausada de novo como sobra e o teste sobe numa campanha nova e completa.
func TestIncompleteLeftoverIsPausedAndTestGoesLive(t *testing.T) {
	f, meta, ex := newAdFixture(t)
	ex.startedWait = time.Millisecond
	test := approvedTest(t, f)
	meta.mu.Lock()
	meta.objects = append(meta.objects, &fakeObject{id: "sobra1", kind: "campaign", name: campaignName(test), status: "ACTIVE"})
	meta.mu.Unlock()
	if err := ex.run(context.Background()); err != nil {
		t.Fatal(err)
	}
	cs := meta.campaigns()
	if len(cs) != 2 || cs[0].status != "PAUSED" || cs[1].status != "ACTIVE" {
		t.Fatalf("sobra pausada e campanha nova ativa: %+v %+v", cs[0], cs[len(cs)-1])
	}
	running, _ := f.ads.List(adtestq.StatusRunning)
	if len(running) != 1 || running[0].Meta.CampaignID == "sobra1" {
		t.Fatalf("o teste sobe na campanha nova: %+v", running)
	}
}

// Sem conseguir conferir no Meta se o teste já tem campanha, não cria.
func TestNoCreateWhenTheCheckFails(t *testing.T) {
	f, meta, ex := newAdFixture(t)
	approvedTest(t, f)
	meta.failOn = "act_1/campaigns"
	if err := ex.run(context.Background()); err == nil || !strings.Contains(err.Error(), "já tem campanha") {
		t.Fatalf("erro = %v", err)
	}
	if cs := meta.campaigns(); len(cs) != 0 {
		t.Fatalf("não podia criar: %+v", cs)
	}
	if still, _ := f.ads.List(adtestq.StatusApproved); len(still) != 1 {
		t.Fatal("o teste segue aprovado para a próxima rodada")
	}
}
