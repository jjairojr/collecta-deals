// Package vitrine keeps the storefront's editable campaign content: extra
// phrases for the top bar, one home banner with gift tiers and one exit pop-up
// with a product. The owner edits it on the dashboard's "Vitrine" screen, so a
// seasonal change (Natal) needs no deploy. Each piece has optional start and end
// dates, read as whole days in Goiânia's time.
package vitrine

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

var ErrInvalid = errors.New("vitrine inválida")

const (
	MaxPhrases   = 6
	MaxTiers     = 4
	maxPhraseLen = 70
	maxTitleLen  = 60
	maxTextLen   = 160
	maxLabelLen  = 30
	maxCTALen    = 24
	dateLayout   = "2006-01-02"
	siteOrigin   = "https://www.collectatcg.com.br/"
)

var goiania = loadLocation("America/Sao_Paulo")

func loadLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		return time.FixedZone("BRT", -3*60*60)
	}
	return loc
}

// Window limits a piece to the days From..Until, both inclusive and both
// optional (AAAA-MM-DD).
type Window struct {
	From  string `json:"from,omitempty"`
	Until string `json:"until,omitempty"`
}

// Phrase is one extra line of the top bar.
type Phrase struct {
	Text string `json:"text"`
	Window
}

// Tier is one gift block of the home banner ("Até R$ 100").
type Tier struct {
	Label string `json:"label"`
	Href  string `json:"href"`
}

type Banner struct {
	Title string `json:"title"`
	Text  string `json:"text,omitempty"`
	CTA   string `json:"cta,omitempty"`
	Href  string `json:"href,omitempty"`
	Tiers []Tier `json:"tiers,omitempty"`
	Window
}

// Popup is the exit-intent pop-up: one product, no coupon.
type Popup struct {
	Enabled bool   `json:"enabled"`
	SKU     string `json:"sku"`
	Title   string `json:"title"`
	Text    string `json:"text,omitempty"`
	Window
}

type Config struct {
	Phrases   []Phrase  `json:"phrases"`
	Banner    *Banner   `json:"banner,omitempty"`
	Popup     *Popup    `json:"popup,omitempty"`
	UpdatedAt time.Time `json:"updatedAt,omitempty"`
}

// Live is what the storefront shows right now: only the pieces inside their
// window, already filtered, so the site needs no date logic of its own.
type Live struct {
	Phrases []string `json:"phrases"`
	Banner  *Banner  `json:"banner,omitempty"`
	Popup   *Popup   `json:"popup,omitempty"`
}

// Example is the Natal content proposed on 27/11/2026. It is never written by
// the code: the owner fills the screen himself; tests and the PR use it.
func Example() Config {
	return Config{
		Phrases: []Phrase{
			{Text: "Para chegar até 24/12, peça até 09/12", Window: Window{Until: "2026-12-09"}},
			{Text: "De 16 a 24/12, retire em Goiânia", Window: Window{From: "2026-12-16", Until: "2026-12-24"}},
		},
		Banner: &Banner{
			Title: "Presentes de Natal",
			Text:  "Cartas e produtos originais para todo tipo de colecionador.",
			Tiers: []Tier{
				{Label: "Até R$ 100", Href: "/singles?preco=0-100"},
				{Label: "Até R$ 300", Href: "/singles?preco=100-300"},
				{Label: "Selados", Href: "/selado"},
			},
			Window: Window{Until: "2026-12-24"},
		},
	}
}

var freeShippingRe = regexp.MustCompile(`(?i)frete\s+(gr[aá]tis|gratuito|zero)`)

func checkText(field, s string, min, max int) error {
	n := utf8.RuneCountInString(strings.TrimSpace(s))
	if n < min || n > max {
		if min > 0 {
			return fmt.Errorf("%s precisa ter de %d a %d letras", field, min, max)
		}
		return fmt.Errorf("%s pode ter até %d letras", field, max)
	}
	if freeShippingRe.MatchString(s) {
		return fmt.Errorf("%s não pode falar de frete grátis: a regra do frete aparece sozinha no site quando está ligada no checkout", field)
	}
	return nil
}

func checkWindow(field string, w Window) error {
	for _, d := range []string{w.From, w.Until} {
		if d == "" {
			continue
		}
		if _, err := time.Parse(dateLayout, d); err != nil {
			return fmt.Errorf("%s: datas no formato AAAA-MM-DD", field)
		}
	}
	if w.From != "" && w.Until != "" && w.Until < w.From {
		return fmt.Errorf("%s: o fim vem antes do início", field)
	}
	return nil
}

// checkHref accepts a path on the site ("/singles?preco=0-100") or a full link to
// collectatcg.com.br, never another domain.
func checkHref(field, href string, required bool) error {
	if href == "" {
		if required {
			return fmt.Errorf("%s: falta o link", field)
		}
		return nil
	}
	if strings.HasPrefix(href, "/") && !strings.HasPrefix(href, "//") {
		return nil
	}
	if strings.HasPrefix(href, siteOrigin) {
		return nil
	}
	return fmt.Errorf("%s: o link precisa ser do site (começar com / ou %s)", field, siteOrigin)
}

func Validate(c Config) error {
	if len(c.Phrases) > MaxPhrases {
		return fmt.Errorf("no máximo %d frases na faixa", MaxPhrases)
	}
	for i, p := range c.Phrases {
		field := fmt.Sprintf("frase %d", i+1)
		if err := checkText(field, p.Text, 3, maxPhraseLen); err != nil {
			return err
		}
		if err := checkWindow(field, p.Window); err != nil {
			return err
		}
	}
	if b := c.Banner; b != nil {
		if err := checkText("título do banner", b.Title, 3, maxTitleLen); err != nil {
			return err
		}
		if err := checkText("texto do banner", b.Text, 0, maxTextLen); err != nil {
			return err
		}
		if err := checkText("botão do banner", b.CTA, 0, maxCTALen); err != nil {
			return err
		}
		if err := checkHref("banner", b.Href, b.CTA != ""); err != nil {
			return err
		}
		if len(b.Tiers) > MaxTiers {
			return fmt.Errorf("no máximo %d blocos de presente", MaxTiers)
		}
		for i, t := range b.Tiers {
			field := fmt.Sprintf("bloco %d", i+1)
			if err := checkText(field, t.Label, 2, maxLabelLen); err != nil {
				return err
			}
			if err := checkHref(field, t.Href, true); err != nil {
				return err
			}
		}
		if err := checkWindow("banner", b.Window); err != nil {
			return err
		}
	}
	if p := c.Popup; p != nil {
		if p.Enabled && strings.TrimSpace(p.SKU) == "" {
			return errors.New("pop-up ligado precisa do SKU do produto")
		}
		if p.Enabled || p.Title != "" {
			if err := checkText("título do pop-up", p.Title, 3, maxTitleLen); err != nil {
				return err
			}
		}
		if err := checkText("texto do pop-up", p.Text, 0, maxTextLen); err != nil {
			return err
		}
		if err := checkWindow("pop-up", p.Window); err != nil {
			return err
		}
	}
	return nil
}

func (w Window) activeOn(day string) bool {
	return (w.From == "" || day >= w.From) && (w.Until == "" || day <= w.Until)
}

// LiveAt filters the config to what is in its window on now's day in Goiânia.
func (c Config) LiveAt(now time.Time) Live {
	day := now.In(goiania).Format(dateLayout)
	out := Live{Phrases: []string{}}
	for _, p := range c.Phrases {
		if p.activeOn(day) {
			out.Phrases = append(out.Phrases, strings.TrimSpace(p.Text))
		}
	}
	if b := c.Banner; b != nil && b.activeOn(day) {
		cp := *b
		out.Banner = &cp
	}
	if p := c.Popup; p != nil && p.Enabled && p.activeOn(day) {
		cp := *p
		out.Popup = &cp
	}
	return out
}

func normalize(c Config) Config {
	if c.Phrases == nil {
		c.Phrases = []Phrase{}
	}
	for i := range c.Phrases {
		c.Phrases[i].Text = strings.TrimSpace(c.Phrases[i].Text)
	}
	if c.Popup != nil {
		c.Popup.SKU = strings.TrimSpace(c.Popup.SKU)
	}
	return c
}

type Store struct {
	path string
	mu   sync.RWMutex
}

func NewStore(path string) *Store {
	return &Store{path: path}
}

// Get returns the saved config; with no file yet, an empty one (nothing shows).
func (s *Store) Get() (Config, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	body, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return Config{Phrases: []Phrase{}}, nil
		}
		return Config{}, err
	}
	var c Config
	if err := json.Unmarshal(body, &c); err != nil {
		return Config{}, err
	}
	return normalize(c), nil
}

// Put validates and replaces the whole config.
func (s *Store) Put(c Config, now time.Time) (Config, error) {
	c = normalize(c)
	if err := Validate(c); err != nil {
		return Config{}, errors.Join(ErrInvalid, err)
	}
	c.UpdatedAt = now
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return Config{}, err
	}
	body, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return Config{}, err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, body, 0o644); err != nil {
		return Config{}, err
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return Config{}, err
	}
	return c, nil
}
