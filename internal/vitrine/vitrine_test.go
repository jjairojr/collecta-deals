package vitrine

import (
	"errors"
	"os"
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

func TestHeroSlidesValidation(t *testing.T) {
	ok := Config{HeroStyle: HeroStyleSpotlight, HeroSlides: []HeroSlide{
		{Enabled: true, Kind: SlideProduct, SKU: "sealed|pokemon|x|||", Badge: "Kit", Phrase: "Cada pacote sai mais barato no kit"},
	}}
	if err := Validate(ok); err != nil {
		t.Fatal(err)
	}
	bad := map[string]HeroSlide{
		"sem sku":          {Enabled: true, Kind: SlideProduct},
		"tipo":             {Enabled: true, Kind: "video", SKU: "a"},
		"imagem sem link":  {Enabled: true, Kind: SlideImage, Image: &Image{Desktop: "/api/storefront/vitrine/img/0123456789abcdef.webp", Mobile: "/api/storefront/vitrine/img/0123456789abcdef.webp", Alt: "Natal"}},
		"imagem externa":   {Enabled: true, Kind: SlideImage, Href: "/selado", Image: &Image{Desktop: "https://x.com/a.png", Mobile: "https://x.com/a.png", Alt: "Natal"}},
		"frete":            {Enabled: true, Kind: SlideProduct, SKU: "a", Phrase: "Frete grátis acima de 600"},
		"corre":            {Enabled: true, Kind: SlideProduct, SKU: "a", Phrase: "Corre que acaba"},
		"ultimas":          {Enabled: true, Kind: SlideProduct, SKU: "a", Badge: "Últimas unidades"},
		"selo longo":       {Enabled: true, Kind: SlideProduct, SKU: "a", Badge: strings.Repeat("x", 21)},
		"frase longa":      {Enabled: true, Kind: SlideProduct, SKU: "a", Phrase: strings.Repeat("x", 91)},
		"link de fora":     {Enabled: true, Kind: SlideProduct, SKU: "a", Href: "https://outro.site/x"},
		"datas invertidas": {Enabled: true, Kind: SlideProduct, SKU: "a", Window: Window{From: "2026-12-10", Until: "2026-12-01"}},
	}
	for name, sl := range bad {
		if err := Validate(Config{HeroSlides: []HeroSlide{sl}}); err == nil {
			t.Errorf("%s: devia recusar", name)
		}
	}
	if err := Validate(Config{HeroStyle: "neon"}); err == nil {
		t.Error("estilo desconhecido devia recusar")
	}
	many := make([]HeroSlide, MaxHeroSlides+1)
	for i := range many {
		many[i] = HeroSlide{Kind: SlideProduct, SKU: "a"}
	}
	if err := Validate(Config{HeroSlides: many}); err == nil {
		t.Error("slides demais devia recusar")
	}
}

func TestHeroSlidesLiveKeepOrderAndDates(t *testing.T) {
	c := Config{HeroSlides: []HeroSlide{
		{Enabled: true, Kind: SlideProduct, SKU: "a", Window: Window{Until: "2026-12-01"}},
		{Enabled: false, Kind: SlideProduct, SKU: "desligado"},
		{Enabled: true, Kind: SlideProduct, SKU: "b"},
		{Enabled: true, Kind: SlideProduct, SKU: "c", Window: Window{From: "2026-12-02"}},
	}}
	got := func(day string) string {
		var ids []string
		for _, sl := range c.LiveAt(at(day)).HeroSlides {
			ids = append(ids, sl.SKU)
		}
		return strings.Join(ids, ",")
	}
	if g := got("2026-12-01"); g != "a,b" {
		t.Fatalf("01/12 = %q", g)
	}
	if g := got("2026-12-02"); g != "b,c" {
		t.Fatalf("02/12 = %q", g)
	}
	if live := c.LiveAt(at("2026-12-01")); live.HeroStyle != HeroStyleShowcase {
		t.Fatalf("estilo padrão = %q", live.HeroStyle)
	}
	if live := (Config{}).LiveAt(at("2026-12-01")); live.HeroStyle != "" || len(live.HeroSlides) != 0 {
		t.Fatalf("sem slides o site mantém o hero de hoje: %+v", live)
	}
}

func TestOldConfigWithoutHeroStillLoads(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vitrine.json")
	if err := os.WriteFile(path, []byte(`{"phrases":[{"text":"Para chegar até 24/12, peça até 09/12"}],"popup":{"enabled":true,"sku":"a","title":"Leve"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := NewStore(path).Get()
	if err != nil || len(c.HeroSlides) != 0 || len(c.Popups) != 1 || len(c.Phrases) != 1 {
		t.Fatalf("config antiga: %+v %v", c, err)
	}
}
