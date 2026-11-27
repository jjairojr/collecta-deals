package vitrine

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func at(day string) time.Time {
	t, err := time.ParseInLocation("2006-01-02 15:04", day+" 12:00", goiania)
	if err != nil {
		panic(err)
	}
	return t
}

func TestExampleIsValid(t *testing.T) {
	if err := Validate(Example()); err != nil {
		t.Fatal(err)
	}
}

func TestLiveFollowsTheDatesInGoiania(t *testing.T) {
	c := Example()
	cases := map[string][]string{
		"2026-12-01": {"Para chegar até 24/12, peça até 09/12"},
		"2026-12-09": {"Para chegar até 24/12, peça até 09/12"},
		"2026-12-10": {},
		"2026-12-16": {"De 16 a 24/12, retire em Goiânia"},
		"2026-12-24": {"De 16 a 24/12, retire em Goiânia"},
		"2026-12-25": {},
	}
	for day, want := range cases {
		got := c.LiveAt(at(day)).Phrases
		if strings.Join(got, "|") != strings.Join(want, "|") {
			t.Errorf("%s: frases = %q, want %q", day, got, want)
		}
	}
	if c.LiveAt(at("2026-12-24")).Banner == nil || c.LiveAt(at("2026-12-25")).Banner != nil {
		t.Error("banner devia valer até 24/12 e sumir em 25/12")
	}
	// 09/12 às 23:30 em Goiânia já é 10/12 em UTC: vale o dia de Goiânia.
	late := time.Date(2026, 12, 9, 23, 30, 0, 0, goiania).UTC()
	if got := c.LiveAt(late).Phrases; len(got) != 1 {
		t.Errorf("09/12 23:30 em Goiânia: frases = %q", got)
	}
}

func TestPopupOnlyWhenEnabledAndInWindow(t *testing.T) {
	c := Config{Popup: &Popup{Enabled: false, SKU: "x", Title: "Kit 3 do booster"}}
	if c.LiveAt(at("2026-12-01")).Popup != nil {
		t.Error("pop-up desligado apareceu")
	}
	c.Popup.Enabled = true
	c.Popup.Until = "2026-12-05"
	if c.LiveAt(at("2026-12-01")).Popup == nil || c.LiveAt(at("2026-12-06")).Popup != nil {
		t.Error("pop-up ligado devia seguir a janela")
	}
}

func TestValidateRejects(t *testing.T) {
	cases := map[string]Config{
		"data errada":       {Phrases: []Phrase{{Text: "Natal na Collecta", Window: Window{From: "09/12/2026"}}}},
		"fim antes":         {Phrases: []Phrase{{Text: "Natal na Collecta", Window: Window{From: "2026-12-10", Until: "2026-12-01"}}}},
		"frete grátis":      {Phrases: []Phrase{{Text: "Frete grátis no Natal"}}},
		"frete gratis":      {Banner: &Banner{Title: "Natal", Text: "com FRETE GRATIS"}},
		"frase longa":       {Phrases: []Phrase{{Text: strings.Repeat("a", 71)}}},
		"link de fora":      {Banner: &Banner{Title: "Natal", Tiers: []Tier{{Label: "Até R$ 100", Href: "https://outro.site/x"}}}},
		"link //":           {Banner: &Banner{Title: "Natal", Tiers: []Tier{{Label: "Até R$ 100", Href: "//outro.site"}}}},
		"bloco sem link":    {Banner: &Banner{Title: "Natal", Tiers: []Tier{{Label: "Até R$ 100"}}}},
		"botão sem link":    {Banner: &Banner{Title: "Natal", CTA: "Ver presentes"}},
		"pop-up sem sku":    {Popup: &Popup{Enabled: true, Title: "Kit 3 do booster"}},
		"pop-up sem título": {Popup: &Popup{Enabled: true, SKU: "x"}},
		"frases demais":     {Phrases: make([]Phrase, MaxPhrases+1)},
		"blocos demais":     {Banner: &Banner{Title: "Natal", Tiers: make([]Tier, MaxTiers+1)}},
	}
	for name, c := range cases {
		if err := Validate(c); err == nil {
			t.Errorf("%s: devia recusar", name)
		}
	}
	ok := Config{Banner: &Banner{Title: "Natal", CTA: "Ver", Href: "https://www.collectatcg.com.br/selado"}}
	if err := Validate(ok); err != nil {
		t.Errorf("link completo do site devia passar: %v", err)
	}
}

func TestStoreRoundTrip(t *testing.T) {
	s := NewStore(filepath.Join(t.TempDir(), "vitrine.json"))
	empty, err := s.Get()
	if err != nil || len(empty.Phrases) != 0 || empty.Banner != nil || empty.Popup != nil {
		t.Fatalf("sem arquivo devia vir vazio: %+v %v", empty, err)
	}
	now := at("2026-11-27")
	saved, err := s.Put(Example(), now)
	if err != nil || !saved.UpdatedAt.Equal(now) {
		t.Fatalf("put: %+v %v", saved, err)
	}
	got, err := s.Get()
	if err != nil || len(got.Phrases) != 2 || got.Banner == nil || len(got.Banner.Tiers) != 3 {
		t.Fatalf("get: %+v %v", got, err)
	}
	if _, err := s.Put(Config{Phrases: []Phrase{{Text: "Frete grátis"}}}, now); !errors.Is(err, ErrInvalid) {
		t.Fatalf("put inválido = %v", err)
	}
	if again, _ := s.Get(); len(again.Phrases) != 2 {
		t.Fatal("put inválido não pode apagar o que estava salvo")
	}
}
