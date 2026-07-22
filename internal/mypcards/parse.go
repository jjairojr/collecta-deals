package mypcards

import (
	"html"
	"regexp"
	"strconv"
	"strings"

	"opdeals/internal/game"
	"opdeals/internal/model"
)

var (
	editionLinkRe = regexp.MustCompile(`<a\s[^>]*class="edicao-link"[^>]*>`)

	streamItemRe = regexp.MustCompile(`(?s)<li class="stream-item" data-key="(\d+)">(.*?)</li>`)
	gaRe         = regexp.MustCompile(`data-ga-item-id="([^"]*)"`)
	productRe    = regexp.MustCompile(`class="card-img-link" href="([^"]*)"`)
	nameRe       = regexp.MustCompile(`<h3 title="([^"]*)"`)
	edicaoRe     = regexp.MustCompile(`class="card-edicao"[^>]*>([^<]+)<`)
	qtyRe        = regexp.MustCompile(`class="quantidade-num">\s*([0-9.]+)\s*<`)
	precoRe      = regexp.MustCompile(`(?s)class="card-preco moeda"[^>]*>\s*(.*?)\s*</span>`)

	printSuffixRe = regexp.MustCompile(`p\d+$`)
)

type listingItem struct {
	listing model.BrazilListing
	key     string
}

func parseEditions(htmlBytes []byte, gameSlug string) []string {
	doc := string(htmlBytes)
	hrefRe := regexp.MustCompile(`href="/` + regexp.QuoteMeta(gameSlug) + `/([a-z0-9-]+)"`)
	seen := map[string]struct{}{"edicoes": {}}
	var slugs []string
	for _, tag := range editionLinkRe.FindAllString(doc, -1) {
		m := hrefRe.FindStringSubmatch(tag)
		if m == nil {
			continue
		}
		slug := m[1]
		if _, ok := seen[slug]; ok {
			continue
		}
		seen[slug] = struct{}{}
		slugs = append(slugs, slug)
	}
	return slugs
}

func parseListing(htmlBytes []byte, cfg game.MyP) []listingItem {
	doc := string(htmlBytes)
	var out []listingItem
	for _, m := range streamItemRe.FindAllStringSubmatch(doc, -1) {
		key, inner := m[1], m[2]

		ga := firstSubmatch(gaRe, inner)
		number := numberFromGA(ga, cfg)
		if number == "" {
			continue
		}
		price := parseBRL(firstSubmatch(precoRe, inner))
		if price <= 0 {
			continue
		}
		qty := parseQty(firstSubmatch(qtyRe, inner))

		out = append(out, listingItem{
			key: key,
			listing: model.BrazilListing{
				Number:       number,
				SetCode:      cfg.SetCode(setCode(ga, firstSubmatch(edicaoRe, inner))),
				Name:         html.UnescapeString(strings.TrimSpace(firstSubmatch(nameRe, inner))),
				Variant:      "Normal",
				Source:       sourceName,
				URL:          productURL(firstSubmatch(productRe, inner), cfg.Slug),
				LowBRL:       price,
				StockChecked: true,
				InStock:      qty > 0,
			},
		})
	}
	return out
}

// numberFromGA pulls the card number out of a data-ga-item-id such as
// "one_op16_op16-080p1" or "riftbound_unl_059a/219". Listing grids mix in
// cross-sell cards from other games, so a mismatched prefix is skipped rather
// than parsed. Denominators and letter suffixes are left for the game's Matcher
// to normalize — stripping them here would fabricate matches.
func numberFromGA(ga string, cfg game.MyP) string {
	parts := strings.Split(ga, "_")
	if len(parts) < 3 || cfg.GAPrefix == "" || parts[0] != cfg.GAPrefix {
		return ""
	}
	num := strings.Join(parts[2:], "_")
	if cfg.StripPrintSuffix {
		num = printSuffixRe.ReplaceAllString(num, "")
	}
	return model.NormalizeNumber(num)
}

func setCode(ga, edicao string) string {
	if e := strings.TrimSpace(edicao); e != "" {
		return strings.ToUpper(e)
	}
	if parts := strings.Split(ga, "_"); len(parts) >= 2 {
		return strings.ToUpper(parts[1])
	}
	return ""
}

func productURL(href, gameSlug string) string {
	href = html.UnescapeString(strings.TrimSpace(href))
	if href == "" {
		return baseURL + "/" + gameSlug
	}
	if strings.HasPrefix(href, "http") {
		return href
	}
	return baseURL + href
}

func parseBRL(s string) float64 {
	s = html.UnescapeString(s)
	var b strings.Builder
	for _, r := range s {
		if (r >= '0' && r <= '9') || r == '.' || r == ',' {
			b.WriteRune(r)
		}
	}
	t := b.String()
	t = strings.ReplaceAll(t, ".", "")
	t = strings.ReplaceAll(t, ",", ".")
	v, err := strconv.ParseFloat(t, 64)
	if err != nil {
		return 0
	}
	return v
}

func parseQty(s string) int {
	s = strings.ReplaceAll(strings.TrimSpace(s), ".", "")
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}

func firstSubmatch(re *regexp.Regexp, s string) string {
	m := re.FindStringSubmatch(s)
	if m == nil {
		return ""
	}
	return m[1]
}
