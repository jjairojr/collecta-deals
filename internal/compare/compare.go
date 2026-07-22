package compare

import (
	"regexp"
	"sort"
	"strings"

	"opdeals/internal/game"
	"opdeals/internal/model"
)

type Options struct {
	FXRate         float64
	MinMargin      float64
	MinPrice       float64
	UsePrice       string
	SortBy         string
	Query          string
	Set            string
	Source         string
	Limit          int
	RequireInStock bool
	SPOnly         bool
	Matcher        Matcher
}

// IsSP reports whether a card name carries the "(SP)" variant marker. Both the
// LigaOnePiece and TCGPlayer names tag SP cards this way, e.g. "Perona (SP)".
func IsSP(name string) bool {
	for _, m := range parenRe.FindAllStringSubmatch(name, -1) {
		if strings.EqualFold(strings.TrimSpace(m[1]), "sp") {
			return true
		}
	}
	return false
}

var (
	parenRe = regexp.MustCompile(`\(([^)]*)\)`)
	wsRe    = regexp.MustCompile(`\s+`)
)

// Matcher builds the join key between a game's Brazil listings and its
// TCGplayer prices. The zero value reproduces One Piece semantics exactly, so
// legacy callers that never set Options.Matcher keep their historical output.
type Matcher struct {
	configured     bool
	setScoped      bool
	stripDenom     bool
	stripZeros     bool
	suffixVariants map[string]string
	vocab          map[string]string
	don            bool
}

func MatcherFor(g game.Game) Matcher {
	m := g.Market
	if m == nil {
		return MatcherFor(game.OnePiece())
	}
	return Matcher{
		configured:     true,
		setScoped:      m.SetScopedKey,
		stripDenom:     m.StripDenominator,
		stripZeros:     m.StripLeadingZeros,
		suffixVariants: m.NumberSuffixVariants,
		vocab:          m.VariantVocab,
		don:            m.DON,
	}
}

func (m Matcher) norm() Matcher {
	if !m.configured {
		return MatcherFor(game.OnePiece())
	}
	return m
}

// Key is the index-side match key for a card, always receiving its set code.
func (m Matcher) Key(number, name, set string) string {
	m = m.norm()
	if m.don && isDON(number, name) {
		return "DON|" + strings.ToUpper(strings.TrimSpace(set)) + "|" + donName(name)
	}
	base, suffixTok := m.normalizeNumber(number)
	key := base + "|" + m.variantToken(name, suffixTok)
	if m.setScoped {
		key = strings.ToUpper(strings.TrimSpace(set)) + "|" + key
	}
	return key
}

// LookupKey is the key used by lookup-side callers (buyout enrichment, held
// cards, portfolio quotes). One Piece historically joined without a set except
// for DON cards, so the OP matcher keeps passing the set only to the DON path;
// set-scoped games need the set on every key.
func (m Matcher) LookupKey(number, name, set string) string {
	m = m.norm()
	if m.setScoped {
		return m.Key(number, name, set)
	}
	return m.Key(number, name, "")
}

func (m Matcher) normalizeNumber(number string) (string, string) {
	n := strings.ToUpper(strings.TrimSpace(number))
	if m.stripDenom {
		if i := strings.Index(n, "/"); i >= 0 {
			n = n[:i]
		}
	}
	if len(m.suffixVariants) > 0 && len(n) > 1 {
		suffix := n[len(n)-1:]
		if tok, ok := m.suffixVariants[suffix]; ok && isDigits(n[:len(n)-1]) {
			return m.trimZeros(n[:len(n)-1]), tok
		}
	}
	if m.stripZeros && isDigits(n) {
		return m.trimZeros(n), ""
	}
	if !m.setScoped {
		return baseNumber(n), ""
	}
	return n, ""
}

func (m Matcher) trimZeros(n string) string {
	if !m.stripZeros {
		return n
	}
	trimmed := strings.TrimLeft(n, "0")
	if trimmed == "" {
		return "0"
	}
	return trimmed
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func Deals(listings []model.BrazilListing, prices []model.USPrice, opts Options) []model.Deal {
	m := opts.Matcher.norm()
	index := indexPrices(prices, m)
	query := strings.ToLower(strings.TrimSpace(opts.Query))
	deals := make([]model.Deal, 0)
	for key, l := range cheapestListings(listings, opts.RequireInStock, m, opts.Source) {
		if query != "" && !matchesQuery(l, query) {
			continue
		}
		if opts.Set != "" && !strings.EqualFold(l.SetCode, opts.Set) {
			continue
		}
		if opts.SPOnly && !IsSP(l.Name) {
			continue
		}
		us, ok := index[key]
		if !ok {
			continue
		}
		sell := effectivePrice(us)
		if strings.EqualFold(opts.UsePrice, "market") && us.MarketUSD > 0 {
			sell = us.MarketUSD
		}
		if sell <= 0 || sell < opts.MinPrice {
			continue
		}
		buy := l.LowBRL * opts.FXRate
		if buy <= 0 {
			continue
		}
		margin := (sell - buy) / buy * 100
		if margin < opts.MinMargin {
			continue
		}
		deals = append(deals, model.Deal{
			Number:     l.Number,
			Name:       l.Name,
			Set:        l.SetCode,
			Rarity:     us.Rarity,
			Variant:    m.displayVariant(us),
			Source:     l.Source,
			BuyURL:     l.URL,
			TCGURL:     us.URL,
			LowBRL:     l.LowBRL,
			BuyUSD:     buy,
			SellUSD:    sell,
			MarginPct:  margin,
			ProfitUSD:  sell - buy,
			Verified:   l.StockChecked && l.InStock,
			USListings: us.LiveListings,
			USQty:      us.LiveQty,
			BRCopies:   l.FloorCopies,
			BRSellers:  l.Sellers,
		})
	}
	sortDeals(deals, opts.SortBy)
	if opts.Limit > 0 && len(deals) > opts.Limit {
		deals = deals[:opts.Limit]
	}
	return deals
}

// USDIndex maps each card's match key to its best (lowest effective) US price.
// It lets callers outside this package — e.g. the tracking/buyout path, which
// has Brazil data but no US price — join a card to its TCGPlayer price using the
// same key the deals path uses.
func USDIndex(prices []model.USPrice, m Matcher) map[string]model.USPrice {
	return indexPrices(prices, m.norm())
}

// EffectiveUSD is the sell price used for a US price: the live lowest listing
// when present, otherwise the sane TCGCSV baseline.
func EffectiveUSD(p model.USPrice) float64 {
	return effectivePrice(p)
}

func indexPrices(prices []model.USPrice, m Matcher) map[string]model.USPrice {
	index := make(map[string]model.USPrice, len(prices))
	for _, p := range prices {
		if effectivePrice(p) <= 0 {
			continue
		}
		key := m.Key(p.Number, p.Name, p.SetCode)
		existing, ok := index[key]
		if !ok || effectivePrice(p) < effectivePrice(existing) {
			index[key] = p
		}
	}
	return index
}

func effectivePrice(p model.USPrice) float64 {
	if p.LiveUSD > 0 {
		return p.LiveUSD
	}
	return saneTCG(p)
}

func saneTCG(p model.USPrice) float64 {
	if p.MarketUSD > 0 && p.LowUSD > p.MarketUSD {
		return p.MarketUSD
	}
	if p.LowUSD > 0 {
		return p.LowUSD
	}
	return p.MarketUSD
}

func matchesQuery(l model.BrazilListing, query string) bool {
	return strings.Contains(strings.ToLower(l.Number), query) ||
		strings.Contains(strings.ToLower(l.Name), query) ||
		strings.Contains(strings.ToLower(l.SetCode), query)
}

// MatchesSource reports whether a listing's source satisfies a source filter.
// An empty filter accepts everything, preserving the historical cheapest-wins
// behaviour across sources. "liga" matches every per-game Liga source name
// (ligaonepiece, ligariftbound, …); anything else is an exact match.
func MatchesSource(source, want string) bool {
	want = strings.ToLower(strings.TrimSpace(want))
	if want == "" {
		return true
	}
	source = strings.ToLower(strings.TrimSpace(source))
	if want == "liga" {
		return strings.HasPrefix(source, "liga")
	}
	return source == want
}

func cheapestListings(listings []model.BrazilListing, requireInStock bool, m Matcher, source string) map[string]model.BrazilListing {
	best := make(map[string]model.BrazilListing, len(listings))
	for _, l := range listings {
		if l.LowBRL <= 0 {
			continue
		}
		if !MatchesSource(l.Source, source) {
			continue
		}
		if requireInStock {
			if !l.InStock {
				continue
			}
		} else if l.StockChecked && !l.InStock {
			continue
		}
		key := m.Key(l.Number, l.Name, l.SetCode)
		existing, ok := best[key]
		if !ok || l.LowBRL < existing.LowBRL {
			best[key] = l
		}
	}
	return best
}

func BRIndex(listings []model.BrazilListing, m Matcher) map[string]model.BrazilListing {
	return cheapestListings(listings, false, m, "")
}

var cardCodeRe = regexp.MustCompile(`^[a-z]{1,5}[0-9]{0,3}-[a-z0-9-]+$`)

// extraTokens are the parenthesized print markers Key ignores: anything that is
// not a variant-vocab token, a bare number, or a card code. Liga and TCGplayer
// write the same collection with different dashes ("-Best Selection Vol. 3-" vs
// "Best Selection Vol. 3"), so dashes normalize to spaces.
func (m Matcher) extraTokens(name string) []string {
	m = m.norm()
	var tokens []string
	seen := make(map[string]bool)
	for _, match := range parenRe.FindAllStringSubmatch(name, -1) {
		raw := strings.ToLower(strings.TrimSpace(match[1]))
		if raw == "" || isDigits(raw) || cardCodeRe.MatchString(raw) {
			continue
		}
		v := strings.TrimSpace(wsRe.ReplaceAllString(strings.ReplaceAll(raw, "-", " "), " "))
		if v == "alt art" {
			v = "alternate art"
		}
		if _, ok := m.vocab[v]; ok || v == "" || seen[v] {
			continue
		}
		seen[v] = true
		tokens = append(tokens, v)
	}
	sort.Strings(tokens)
	return tokens
}

// PrintKey extends Key with the extra print markers so distinct special prints
// of the same number (promo packs, premium collections) key separately. Callers
// should fall back to Key when a PrintKey lookup misses.
func (m Matcher) PrintKey(number, name, set string) string {
	key := m.Key(number, name, set)
	extras := m.extraTokens(name)
	if len(extras) == 0 {
		return key
	}
	return key + "|" + strings.Join(extras, "+")
}

func BRPrintIndex(listings []model.BrazilListing, m Matcher) map[string]model.BrazilListing {
	best := make(map[string]model.BrazilListing, len(listings))
	for _, l := range listings {
		if l.LowBRL <= 0 {
			continue
		}
		if l.StockChecked && !l.InStock {
			continue
		}
		key := m.PrintKey(l.Number, l.Name, l.SetCode)
		existing, ok := best[key]
		if !ok || l.LowBRL < existing.LowBRL {
			best[key] = l
		}
	}
	return best
}

func CandidateListings(listings []model.BrazilListing, prices []model.USPrice, fxRate, floor float64, m Matcher) ([]model.BrazilListing, []model.USPrice) {
	m = m.norm()
	index := indexPrices(prices, m)
	candidateKeys := make(map[string]bool)
	us := make([]model.USPrice, 0)
	for key, l := range cheapestListings(listings, false, m, "") {
		p, ok := index[key]
		if !ok {
			continue
		}
		sell := effectivePrice(p)
		if sell < floor {
			continue
		}
		buy := l.LowBRL * fxRate
		if buy <= 0 {
			continue
		}
		if (sell-buy)/buy*100 <= 0 {
			continue
		}
		candidateKeys[key] = true
		us = append(us, p)
	}
	brs := make([]model.BrazilListing, 0)
	for _, l := range listings {
		if l.LowBRL > 0 && candidateKeys[m.Key(l.Number, l.Name, l.SetCode)] {
			brs = append(brs, l)
		}
	}
	return brs, us
}

var donSuffixRe = regexp.MustCompile(`(?i)\s*\(DON-[A-Z0-9-]+\)\s*$`)

func isDON(number, name string) bool {
	return strings.HasPrefix(strings.ToUpper(strings.TrimSpace(number)), "DON-") ||
		strings.HasPrefix(name, "DON!! Card")
}

func donName(name string) string {
	stripped := donSuffixRe.ReplaceAllString(name, "")
	return wsRe.ReplaceAllString(strings.ToLower(strings.TrimSpace(stripped)), " ")
}

func baseNumber(number string) string {
	n := strings.ToUpper(strings.TrimSpace(number))
	for {
		i := strings.LastIndex(n, "-")
		if i <= 0 {
			return n
		}
		if !isAllLetters(n[i+1:]) {
			return n
		}
		n = n[:i]
	}
}

func isAllLetters(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < 'A' || r > 'Z' {
			return false
		}
	}
	return true
}

func (m Matcher) variantToken(name, suffixTok string) string {
	var tokens []string
	seen := make(map[string]bool)
	if suffixTok != "" {
		seen[suffixTok] = true
		tokens = append(tokens, suffixTok)
	}
	for _, match := range parenRe.FindAllStringSubmatch(name, -1) {
		v := wsRe.ReplaceAllString(strings.ToLower(strings.TrimSpace(match[1])), " ")
		if v == "alt art" {
			v = "alternate art"
		}
		if _, ok := m.vocab[v]; !ok || seen[v] {
			continue
		}
		seen[v] = true
		tokens = append(tokens, v)
	}
	sort.Strings(tokens)
	return strings.Join(tokens, "+")
}

func (m Matcher) displayVariant(p model.USPrice) string {
	token := m.variantToken(p.Name, "")
	if token == "" {
		return p.Variant
	}
	parts := strings.Split(token, "+")
	for i, t := range parts {
		if label, ok := m.vocab[t]; ok {
			parts[i] = label
		}
	}
	return strings.Join(parts, " ")
}

func sortDeals(deals []model.Deal, by string) {
	sort.SliceStable(deals, func(i, j int) bool {
		if strings.EqualFold(by, "profit") && deals[i].ProfitUSD != deals[j].ProfitUSD {
			return deals[i].ProfitUSD > deals[j].ProfitUSD
		}
		if deals[i].MarginPct != deals[j].MarginPct {
			return deals[i].MarginPct > deals[j].MarginPct
		}
		return deals[i].Number < deals[j].Number
	})
}
