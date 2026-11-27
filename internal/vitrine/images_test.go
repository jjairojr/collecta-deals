package vitrine

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var (
	pngBytes  = []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1A, '\n', 0, 0, 0, 13}
	jpgBytes  = []byte{0xFF, 0xD8, 0xFF, 0xE0, 0, 16, 'J', 'F', 'I', 'F'}
	webpBytes = []byte("RIFF\x10\x00\x00\x00WEBPVP8 ")
)

func TestSaveImageChecksBytesAndSize(t *testing.T) {
	s := NewStore(filepath.Join(t.TempDir(), "vitrine.json"))
	for name, data := range map[string][]byte{"png": pngBytes, "jpg": jpgBytes, "webp": webpBytes} {
		url, err := s.SaveImage(data)
		if err != nil || !IsImageURL(url) || !strings.HasSuffix(url, "."+name) {
			t.Fatalf("%s: url=%q err=%v", name, url, err)
		}
		again, _ := s.SaveImage(data)
		if again != url {
			t.Fatalf("%s: o mesmo arquivo devia ter a mesma URL", name)
		}
		file, err := s.ImageFile(strings.TrimPrefix(url, ImagePath))
		if err != nil || filepath.Dir(file) != s.imageDir() {
			t.Fatalf("%s: arquivo %q %v", name, file, err)
		}
	}
	if _, err := s.SaveImage([]byte("<svg onload=alert(1)>")); !errors.Is(err, ErrImageType) {
		t.Fatalf("SVG/texto devia ser recusado: %v", err)
	}
	if _, err := s.SaveImage([]byte("GIF89a......")); !errors.Is(err, ErrImageType) {
		t.Fatalf("GIF devia ser recusado: %v", err)
	}
	big := append(append([]byte{}, pngBytes...), make([]byte, MaxImageBytes)...)
	if _, err := s.SaveImage(big); !errors.Is(err, ErrImageTooBig) {
		t.Fatalf("acima de 1 MB devia ser recusado: %v", err)
	}
	for _, bad := range []string{"../vitrine.json", "..%2Fvitrine.json", "abc.png", "0123456789abcdef.svg", "0123456789abcdef.png/../x", "0123456789abcdef.png"} {
		if _, err := s.ImageFile(bad); !errors.Is(err, ErrImageMissing) {
			t.Fatalf("nome %q devia ser recusado ou não existir: %v", bad, err)
		}
	}
}

func img(t *testing.T, s *Store) Image {
	t.Helper()
	d, err := s.SaveImage(webpBytes)
	if err != nil {
		t.Fatal(err)
	}
	m, err := s.SaveImage(jpgBytes)
	if err != nil {
		t.Fatal(err)
	}
	return Image{Desktop: d, Mobile: m, Alt: "Pikachu e Luffy: presentes de Natal"}
}

func TestImagesValidation(t *testing.T) {
	s := NewStore(filepath.Join(t.TempDir(), "vitrine.json"))
	good := img(t, s)
	ok := Config{
		BarImages: []BarImage{{Image: good, Href: "/selado", Window: Window{From: "2026-12-01", Until: "2026-12-24"}}},
		Banner:    &Banner{Title: "Presentes de Natal", Href: "/selado", Image: &good},
	}
	if err := Validate(ok); err != nil {
		t.Fatalf("config com imagens devia passar: %v", err)
	}
	cases := map[string]func(c *Config){
		"imagem de fora": func(c *Config) { c.BarImages[0].Desktop = "https://evil.test/x.png" },
		"sem celular":    func(c *Config) { c.BarImages[0].Mobile = "" },
		"faixa sem link": func(c *Config) { c.BarImages[0].Href = "" },
		"link externo":   func(c *Config) { c.BarImages[0].Href = "https://evil.test" },
		"sem descrição":  func(c *Config) { c.BarImages[0].Alt = "" },
		"alt com frete":  func(c *Config) { c.BarImages[0].Alt = "Frete grátis no Natal" },
		"banner sem link": func(c *Config) {
			c.Banner.Href = ""
		},
		"datas trocadas": func(c *Config) { c.BarImages[0].Window = Window{From: "2026-12-24", Until: "2026-12-01"} },
	}
	for name, mutate := range cases {
		c := ok
		c.BarImages = append([]BarImage(nil), ok.BarImages...)
		b := *ok.Banner
		c.Banner = &b
		mutate(&c)
		if err := Validate(c); err == nil {
			t.Errorf("%s: devia recusar", name)
		}
	}
}

func TestLivePicksFirstBarImageAndPopupOfTheDay(t *testing.T) {
	s := NewStore(filepath.Join(t.TempDir(), "vitrine.json"))
	im := img(t, s)
	c := Config{
		BarImages: []BarImage{
			{Image: im, Href: "/selado", Window: Window{Until: "2026-12-01"}},
			{Image: Image{Desktop: im.Mobile, Mobile: im.Desktop, Alt: "segunda arte"}, Href: "/singles", Window: Window{From: "2026-12-02"}},
		},
		Popups: []Popup{
			{Enabled: true, SKU: "kit3", Title: "Vai levar só um?", Window: Window{From: "2026-11-27", Until: "2026-12-01"}},
			{Enabled: true, SKU: "gem2", Title: "Presente de colecionador", Window: Window{From: "2026-12-02", Until: "2026-12-08"}},
			{Enabled: false, SKU: "off", Title: "Desligado", Window: Window{From: "2026-12-09", Until: "2026-12-15"}},
			{Enabled: true, SKU: "sylveon", Title: "Presente pronto", Window: Window{From: "2026-12-09", Until: "2026-12-15"}},
		},
	}
	at := func(day string) Live {
		d, _ := time.ParseInLocation(dateLayout, day, goiania)
		return c.LiveAt(d.Add(15 * time.Hour))
	}
	if l := at("2026-11-30"); l.BarImage == nil || l.BarImage.Href != "/selado" || l.Popup == nil || l.Popup.SKU != "kit3" {
		t.Fatalf("30/11 = %+v", l)
	}
	if l := at("2026-12-05"); l.BarImage == nil || l.BarImage.Href != "/singles" || l.Popup == nil || l.Popup.SKU != "gem2" {
		t.Fatalf("05/12 = %+v", l)
	}
	if l := at("2026-12-10"); l.Popup == nil || l.Popup.SKU != "sylveon" {
		t.Fatalf("10/12 devia pular o desligado: %+v", l)
	}
	if l := at("2026-12-20"); l.Popup != nil {
		t.Fatalf("20/12 sem pop-up: %+v", l)
	}
}

func TestOldSinglePopupMigrates(t *testing.T) {
	s := NewStore(filepath.Join(t.TempDir(), "vitrine.json"))
	old := Config{Phrases: []Phrase{}, Popup: &Popup{Enabled: true, SKU: " kit3 ", Title: "Vai levar só um?"}}
	saved, err := s.Put(old, time.Date(2026, 11, 27, 12, 0, 0, 0, goiania))
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.Get()
	if err != nil || saved.Popup != nil || len(got.Popups) != 1 || got.Popups[0].SKU != "kit3" || got.Popup != nil {
		t.Fatalf("migração: saved=%+v got=%+v err=%v", saved, got, err)
	}
}
