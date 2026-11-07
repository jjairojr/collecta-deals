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
	first, err := s.Add(sample(10000))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Add(sample(6000)); !errors.Is(err, ErrOverBudget) {
		t.Fatalf("passou do teto: %v", err)
	}
	if _, err := s.Decide(first.ID, first.Token, false); err != nil {
		t.Fatal(err)
	}
	second, err := s.Add(sample(15000))
	if err != nil {
		t.Fatalf("rejeitado devia liberar a verba: %v", err)
	}
	if avail, _ := s.Available(); avail != 0 {
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
