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
	MaxBarImages = 6
	MaxPopups    = 8
	// MaxHeroSlides caps the home carousel.
	MaxHeroSlides     = 8
	maxSlidePhraseLen = 90
	maxBadgeLen       = 20
	maxAltLen         = 120
	MaxTiers          = 4
	maxPhraseLen      = 70
	maxTitleLen       = 60
	maxTextLen        = 160
	maxLabelLen       = 30
	maxCTALen         = 24
	dateLayout        = "2006-01-02"
	siteOrigin        = "https://www.collectatcg.com.br/"
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

// Image is a piece of art uploaded on the Vitrine screen: one file for desktop
// and one for the phone, both stored by this package (ImageURL), plus the text
// read aloud and shown when the image does not load.
type Image struct {
	Desktop string `json:"desktop"`
	Mobile  string `json:"mobile"`
	Alt     string `json:"alt"`
}

// BarImage replaces the text bar at the top of every page while it is live: the
// whole image is one link. Art sizes: desktop 2880x120 (24:1), phone 1170x180.
type BarImage struct {
	Image
	Href string `json:"href"`
	Window
}

type Banner struct {
	Title string `json:"title"`
	Text  string `json:"text,omitempty"`
	CTA   string `json:"cta,omitempty"`
	Href  string `json:"href,omitempty"`
	Tiers []Tier `json:"tiers,omitempty"`
	// Image, when set, is shown instead of the title and text, as a link to Href
	// (required then). Art sizes: desktop 2720x800 (3,4:1), phone 1080x1080.
	Image *Image `json:"image,omitempty"`
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

// Hero carousel slides (topo da home). A "produto" slide names a SKU (a kit too)
// and the site reads its photo, name and prices from the catalog; an "imagem"
// slide is uploaded art with a link. Slides are shown in list order.
const (
	SlideProduct = "produto"
	SlideImage   = "imagem"

	HeroStyleShowcase  = "vitrine"
	HeroStyleSpotlight = "destaque"
)

type HeroSlide struct {
	Enabled bool   `json:"enabled"`
	Kind    string `json:"kind"`
	SKU     string `json:"sku,omitempty"`
	Image   *Image `json:"image,omitempty"`
	// Badge is a short tag on the slide ("Kit", "Pré-venda", "Natal", "Novo").
	Badge  string `json:"badge,omitempty"`
	Phrase string `json:"phrase,omitempty"`
	// Href overrides where the slide leads (default: the product page). Required
	// for an image slide.
	Href string `json:"href,omitempty"`
	Window
}

type Config struct {
	Phrases   []Phrase   `json:"phrases"`
	BarImages []BarImage `json:"barImages,omitempty"`
	Banner    *Banner    `json:"banner,omitempty"`
	// Popups are tried in order: the site shows the first one switched on, inside
	// its dates and with its product on sale. A config saved before the list
	// existed had a single Popup; reading it moves that one into Popups.
	Popups []Popup `json:"popups,omitempty"`
	Popup  *Popup  `json:"popup,omitempty"`
	// HeroStyle picks the carousel look ("vitrine" or "destaque"); empty means
	// "vitrine". With no live slide the home keeps its usual hero.
	HeroStyle  string      `json:"heroStyle,omitempty"`
	HeroSlides []HeroSlide `json:"heroSlides,omitempty"`
	UpdatedAt  time.Time   `json:"updatedAt,omitempty"`
}

// Live is what the storefront shows right now: only the pieces inside their
// window, already filtered, so the site needs no date logic of its own.
type Live struct {
	Phrases  []string  `json:"phrases"`
	BarImage *BarImage `json:"barImage,omitempty"`
	Banner   *Banner   `json:"banner,omitempty"`
	Popup    *Popup    `json:"popup,omitempty"`
	// HeroSlides are the carousel slides switched on and inside their dates;
	// the API also drops product slides whose SKU is not on sale.
	HeroStyle  string      `json:"heroStyle,omitempty"`
	HeroSlides []HeroSlide `json:"heroSlides,omitempty"`
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

// pressureRe: words the store does not use to push a sale (combinado com o Jairo).
var pressureRe = regexp.MustCompile(`(?i)\bcorre\b|[uú]ltimas\s+unidades`)

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
	if len(c.BarImages) > MaxBarImages {
		return fmt.Errorf("no máximo %d imagens na faixa", MaxBarImages)
	}
	for i, bi := range c.BarImages {
		field := fmt.Sprintf("imagem %d da faixa", i+1)
		if err := checkImage(field, bi.Image); err != nil {
			return err
		}
		if err := checkHref(field, bi.Href, true); err != nil {
			return err
		}
		if err := checkWindow(field, bi.Window); err != nil {
			return err
		}
	}
	if b := c.Banner; b != nil {
		if err := checkText("título do banner", b.Title, 3, maxTitleLen); err != nil {
			return err
		}
		if b.Image != nil {
			if err := checkImage("imagem do banner", *b.Image); err != nil {
				return err
			}
			if err := checkHref("imagem do banner", b.Href, true); err != nil {
				return err
			}
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
	if err := validateHero(c); err != nil {
		return err
	}
	popups := c.popupList()
	if len(popups) > MaxPopups {
		return fmt.Errorf("no máximo %d pop-ups", MaxPopups)
	}
	for i, p := range popups {
		field := fmt.Sprintf("pop-up %d", i+1)
		if p.Enabled && strings.TrimSpace(p.SKU) == "" {
			return fmt.Errorf("%s ligado precisa do SKU do produto", field)
		}
		if p.Enabled || p.Title != "" {
			if err := checkText("título do "+field, p.Title, 3, maxTitleLen); err != nil {
				return err
			}
		}
		if err := checkText("texto do "+field, p.Text, 0, maxTextLen); err != nil {
			return err
		}
		if err := checkWindow(field, p.Window); err != nil {
			return err
		}
	}
	return nil
}

func validateHero(c Config) error {
	switch c.HeroStyle {
	case "", HeroStyleShowcase, HeroStyleSpotlight:
	default:
		return fmt.Errorf("estilo do carrossel precisa ser %q ou %q", HeroStyleShowcase, HeroStyleSpotlight)
	}
	if len(c.HeroSlides) > MaxHeroSlides {
		return fmt.Errorf("no máximo %d slides no carrossel", MaxHeroSlides)
	}
	for i, sl := range c.HeroSlides {
		field := fmt.Sprintf("slide %d", i+1)
		switch sl.Kind {
		case SlideProduct:
			if strings.TrimSpace(sl.SKU) == "" {
				return fmt.Errorf("%s: escolha o produto", field)
			}
		case SlideImage:
			if sl.Image == nil {
				return fmt.Errorf("%s: envie as imagens", field)
			}
			if err := checkImage(field, *sl.Image); err != nil {
				return err
			}
			if err := checkHref(field, sl.Href, true); err != nil {
				return err
			}
		default:
			return fmt.Errorf("%s: tipo precisa ser %q ou %q", field, SlideProduct, SlideImage)
		}
		if err := checkText("selo do "+field, sl.Badge, 0, maxBadgeLen); err != nil {
			return err
		}
		if err := checkText("frase do "+field, sl.Phrase, 0, maxSlidePhraseLen); err != nil {
			return err
		}
		for _, s := range []string{sl.Badge, sl.Phrase} {
			if pressureRe.MatchString(s) {
				return fmt.Errorf("%s: sem \"corre\" nem \"últimas unidades\"", field)
			}
		}
		if err := checkHref(field, sl.Href, false); err != nil {
			return err
		}
		if err := checkWindow(field, sl.Window); err != nil {
			return err
		}
	}
	return nil
}

// HeroSlidesOn lists, in order, the slides switched on and inside their dates on
// now's day; the caller drops product slides whose SKU is not on sale.
func (c Config) HeroSlidesOn(now time.Time) []HeroSlide {
	day := now.In(goiania).Format(dateLayout)
	var out []HeroSlide
	for _, sl := range c.HeroSlides {
		if sl.Enabled && sl.activeOn(day) {
			out = append(out, sl)
		}
	}
	return out
}

// HeroStyleOrDefault is the style the site should use.
func (c Config) HeroStyleOrDefault() string {
	if c.HeroStyle == "" {
		return HeroStyleShowcase
	}
	return c.HeroStyle
}

// checkImage: both files are required and must be ones uploaded here (never a
// link to another site), with a short description.
func checkImage(field string, im Image) error {
	if im.Desktop == "" || im.Mobile == "" {
		return fmt.Errorf("%s: envie as duas imagens, a do computador e a do celular", field)
	}
	for _, u := range []string{im.Desktop, im.Mobile} {
		if !IsImageURL(u) {
			return fmt.Errorf("%s: a imagem precisa ser enviada pela tela Vitrine", field)
		}
	}
	return checkText("descrição da "+field, im.Alt, 3, maxAltLen)
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
	for _, bi := range c.BarImages {
		if bi.activeOn(day) {
			cp := bi
			out.BarImage = &cp
			break
		}
	}
	if b := c.Banner; b != nil && b.activeOn(day) {
		cp := *b
		out.Banner = &cp
	}
	if ps := c.PopupsOn(now); len(ps) > 0 {
		cp := ps[0]
		out.Popup = &cp
	}
	if sl := c.HeroSlidesOn(now); len(sl) > 0 {
		out.HeroStyle = c.HeroStyleOrDefault()
		out.HeroSlides = sl
	}
	return out
}

// popupList is Popups, or the single Popup of a config saved before the list.
func (c Config) popupList() []Popup {
	if len(c.Popups) == 0 && c.Popup != nil {
		return []Popup{*c.Popup}
	}
	return c.Popups
}

// PopupsOn lists, in order, the pop-ups switched on and inside their dates on
// now's day; the caller shows the first whose product is on sale.
func (c Config) PopupsOn(now time.Time) []Popup {
	day := now.In(goiania).Format(dateLayout)
	var out []Popup
	for _, p := range c.popupList() {
		if p.Enabled && p.activeOn(day) {
			out = append(out, p)
		}
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
		if len(c.Popups) == 0 {
			c.Popups = []Popup{*c.Popup}
		}
		c.Popup = nil
	}
	for i := range c.Popups {
		c.Popups[i].SKU = strings.TrimSpace(c.Popups[i].SKU)
	}
	for i := range c.HeroSlides {
		sl := &c.HeroSlides[i]
		sl.SKU = strings.TrimSpace(sl.SKU)
		sl.Badge = strings.TrimSpace(sl.Badge)
		sl.Phrase = strings.TrimSpace(sl.Phrase)
		if sl.Kind == SlideProduct {
			sl.Image = nil
		}
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
