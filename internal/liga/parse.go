package liga

import (
	"encoding/json"
	"fmt"
	"image"
	"net/url"
	"regexp"
	"strconv"

	"opdeals/internal/game"
	"opdeals/internal/model"
)

var (
	cardsJSONRe = regexp.MustCompile(`(?s)var cardsjson = (\[.*?\]);`)
	acronymRe   = regexp.MustCompile(`"acronym":"([^"]*)"`)
	stockRe     = regexp.MustCompile(`(?s)var cards_stock = (\[.*?\]);`)
	storesRe    = regexp.MustCompile(`(?s)var cards_stores\s*=\s*(\{.*?\});`)
)

type rawCard struct {
	SSigla string `json:"sSigla"`
	SN     string `json:"sN"`
	NEN    string `json:"nEN"`
	NPT    string `json:"nPT"`
	P1a    string `json:"p1a"`
	P1b    string `json:"p1b"`
}

type rawStock struct {
	LjID       int    `json:"lj_id"`
	Num        string `json:"num"`
	Qualid     string `json:"qualid"`
	Idioma     string `json:"idioma"`
	QuantCss   string `json:"quantCss"`
	Preco      string `json:"preco"`
	PrecoFinal string `json:"precoFinal"`
	PrecoCss   string `json:"precoCss"`
}

type rawStore struct {
	Name   string `json:"lj_name"`
	Cidade string `json:"lj_cidade"`
	UF     string `json:"lj_uf"`
}

type StoreListing struct {
	StoreID    int
	StoreName  string
	StoreCity  string
	StoreUF    string
	Number     string
	Condition  string
	Language   string
	Quantity   int
	QtyKnown   bool
	PriceBRL   float64
	PriceKnown bool
}

func parseEditions(html []byte) []string {
	matches := acronymRe.FindAllSubmatch(html, -1)
	seen := make(map[string]struct{}, len(matches))
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		code := string(m[1])
		if code == "" {
			continue
		}
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}
		out = append(out, code)
	}
	return out
}

func parseCards(html []byte, setCode string, g game.Game) ([]model.BrazilListing, error) {
	m := cardsJSONRe.FindSubmatch(html)
	if m == nil {
		return nil, fmt.Errorf("cardsjson not found for %s", setCode)
	}
	var raws []rawCard
	if err := json.Unmarshal(m[1], &raws); err != nil {
		return nil, fmt.Errorf("cardsjson decode %s: %w", setCode, err)
	}
	out := make([]model.BrazilListing, 0, len(raws))
	for _, r := range raws {
		low := parsePrice(r.P1a)
		if low <= 0 {
			continue
		}
		number := model.NormalizeNumber(r.SN)
		out = append(out, model.BrazilListing{
			Number:  number,
			SetCode: r.SSigla,
			Name:    cardName(r),
			Variant: "Normal",
			Source:  g.SourceName,
			URL:     cardURL(g.LigaBaseURL, r.SSigla, number, cardName(r)),
			LowBRL:  low,
			AvgBRL:  parsePrice(r.P1b),
		})
	}
	return out, nil
}

func listingPrice(s rawStock, priceAtlas image.Image, classPos map[string][2]int) (float64, bool) {
	if s.PrecoFinal != "" {
		if v := parsePrice(s.PrecoFinal); v > 0 {
			return v, true
		}
	}
	if s.Preco != "" {
		if v := parsePrice(s.Preco); v > 0 {
			return v, true
		}
	}
	if s.PrecoCss != "" {
		if v, ok := decodePrice(priceAtlas, classPos, s.PrecoCss); ok {
			return v, true
		}
	}
	return 0, false
}

func cardURL(baseURL, setCode, number, name string) string {
	q := url.Values{}
	q.Set("view", "cards/card")
	q.Set("card", name)
	q.Set("ed", setCode)
	q.Set("num", number)
	return baseURL + "?" + q.Encode()
}

func pageHasStock(html []byte) bool {
	m := stockRe.FindSubmatch(html)
	if m == nil {
		return false
	}
	var entries []json.RawMessage
	if err := json.Unmarshal(m[1], &entries); err != nil {
		return false
	}
	return len(entries) > 0
}

func parseCardStock(html []byte, qtyAtlas, priceAtlas image.Image) ([]StoreListing, error) {
	sm := stockRe.FindSubmatch(html)
	if sm == nil {
		return nil, nil
	}
	var stock []rawStock
	if err := json.Unmarshal(sm[1], &stock); err != nil {
		return nil, fmt.Errorf("cards_stock decode: %w", err)
	}
	stores := map[string]rawStore{}
	if stm := storesRe.FindSubmatch(html); stm != nil {
		_ = json.Unmarshal(stm[1], &stores)
	}
	classPos := styleClassPositions(html)

	out := make([]StoreListing, 0, len(stock))
	for _, s := range stock {
		qty, known := decodeQuantity(qtyAtlas, classPos, s.QuantCss)
		store := stores[strconv.Itoa(s.LjID)]
		price, priceKnown := listingPrice(s, priceAtlas, classPos)
		out = append(out, StoreListing{
			StoreID:    s.LjID,
			StoreName:  store.Name,
			StoreCity:  store.Cidade,
			StoreUF:    store.UF,
			Number:     model.NormalizeNumber(s.Num),
			Condition:  s.Qualid,
			Language:   s.Idioma,
			Quantity:   qty,
			QtyKnown:   known,
			PriceBRL:   price,
			PriceKnown: priceKnown,
		})
	}
	return out, nil
}

func cardName(r rawCard) string {
	if r.NEN != "" {
		return r.NEN
	}
	return r.NPT
}

func parsePrice(s string) float64 {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v
}
