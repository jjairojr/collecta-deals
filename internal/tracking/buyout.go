package tracking

import (
	"math"
	"sort"
	"strings"
)

const conditionNM = "2"

// MarketConfig carries the per-game knobs the buyout/snipe analysis needs: which
// listing languages count toward a card's floor ladder (nil = all languages) and
// the main-character name tokens for the `chars=main` filter.
type MarketConfig struct {
	Langs      []string
	MainTokens []string
}

func isMainCharacter(name string, tokens []string) bool {
	n := strings.ToLower(name)
	for _, t := range tokens {
		if strings.Contains(n, t) {
			return true
		}
	}
	return false
}

// isSP reports whether a Liga card name carries the "(SP)" variant marker, e.g.
// "Perona (SP)". SP is a One Piece-only parallel; other games never match.
func isSP(name string) bool {
	return strings.Contains(strings.ToLower(name), "(sp)")
}

func langAllowed(lang string, langs []string) bool {
	if len(langs) == 0 {
		return true
	}
	for _, l := range langs {
		if l == lang {
			return true
		}
	}
	return false
}

type BuyoutCandidate struct {
	Set           string  `json:"set,omitempty"`
	Number        string  `json:"number"`
	Name          string  `json:"name"`
	URL           string  `json:"url"`
	Floor         float64 `json:"floor"`
	NextFloor     float64 `json:"nextFloor"`
	LiftPct       float64 `json:"liftPct"`
	BuyoutCost    float64 `json:"buyoutCost"`
	ShippingCost  float64 `json:"shippingCost"`
	StoreCount    int     `json:"storeCount"`
	CopiesToClear int     `json:"copiesToClear"`
	ProfitBRL     float64 `json:"profitBRL"`
	NMSupply      int     `json:"nmSupply"`
	Sellers       int     `json:"sellers"`
	Score         float64 `json:"score"`
	// SellUSD and TCGURL are the card's US resale price and TCGPlayer link. They
	// are left zero by Buyout (which only sees Brazil data) and populated by the
	// API layer, which has the deals snapshot to join against.
	SellUSD float64 `json:"sellUSD,omitempty"`
	TCGURL  string  `json:"tcgUrl,omitempty"`
}

type priceLevel struct {
	price  float64
	qty    int
	stores map[int]bool
}

// Buyout ranks cards by how far a buyout of `budget` reais can push the NM floor.
// For each card it clears the cheapest NM price levels from the bottom while the
// cumulative cost stays within budget; the new floor is the next uncleared level.
// A thick cheap tail naturally scores low because clearing it blows the budget
// for little lift. Only cards with floor >= minFloor are considered.
func Buyout(day DaySnapshot, budget, minFloor, shipping float64, mainCharsOnly, spOnly bool, sortBy string, top int, cfg MarketConfig) []BuyoutCandidate {
	out := make([]BuyoutCandidate, 0, len(day.Cards))
	for _, c := range day.Cards {
		if mainCharsOnly && !isMainCharacter(c.Name, cfg.MainTokens) {
			continue
		}
		if spOnly && !isSP(c.Name) {
			continue
		}
		cand, ok := buyoutForCard(c, budget, minFloor, shipping, cfg.Langs)
		if ok {
			out = append(out, cand)
		}
	}
	return RankBuyout(out, sortBy, top)
}

// Snipe hunts for a single underpriced listing per card: one where the cheapest
// English NM price is at least minGapPct below the next distinct price (from any
// store). The "play" is to buy every copy at that cheapest price and flip near
// the next price. Unlike Buyout it is threshold-driven, not budget-bounded. Only
// cards with floor >= minFloor and gap >= minGapPct are returned. Candidates use
// the BuyoutCandidate shape so the ranking, US enrichment, and UI are shared.
func Snipe(day DaySnapshot, minFloor, minGapPct, shipping float64, mainCharsOnly, spOnly bool, sortBy string, top int, cfg MarketConfig) []BuyoutCandidate {
	out := make([]BuyoutCandidate, 0, len(day.Cards))
	for _, c := range day.Cards {
		if mainCharsOnly && !isMainCharacter(c.Name, cfg.MainTokens) {
			continue
		}
		if spOnly && !isSP(c.Name) {
			continue
		}
		cand, ok := snipeForCard(c, minFloor, minGapPct, shipping, cfg.Langs)
		if ok {
			out = append(out, cand)
		}
	}
	return RankBuyout(out, sortBy, top)
}

func snipeForCard(c CardDay, minFloor, minGapPct, shipping float64, langs []string) (BuyoutCandidate, bool) {
	levels, supply, sellers := nmPriceLevels(c, langs)
	if len(levels) < 2 {
		return BuyoutCandidate{}, false
	}

	floor := levels[0].price
	if floor < minFloor {
		return BuyoutCandidate{}, false
	}
	ceiling := levels[1].price
	gap := (ceiling - floor) / floor * 100
	if gap < minGapPct {
		return BuyoutCandidate{}, false
	}

	copies := levels[0].qty
	stores := len(levels[0].stores)
	cost := floor * float64(copies)
	shipCost := float64(stores) * shipping
	profit := float64(copies)*ceiling - cost - shipCost

	// Same corner-ability score as Buyout so the shared ranking stays consistent:
	// big gap, on a thin/low-supply card with enough sellers for liquidity.
	demand := float64(sellers) / 8
	if demand > 1 {
		demand = 1
	}
	corner := 1.0 / (1.0 + float64(supply)/40.0)
	effort := 1.0 / math.Sqrt(float64(copies))
	score := gap * demand * corner * effort

	return BuyoutCandidate{
		Number:        c.Number,
		Name:          c.Name,
		URL:           c.URL,
		Floor:         floor,
		NextFloor:     ceiling,
		LiftPct:       gap,
		BuyoutCost:    cost,
		ShippingCost:  shipCost,
		StoreCount:    stores,
		CopiesToClear: copies,
		ProfitBRL:     profit,
		NMSupply:      supply,
		Sellers:       sellers,
		Score:         score,
	}, true
}

// RankBuyout sorts candidates by the chosen mode (descending in desirability,
// set+number as tie-break) and truncates to top. It is used both per-set and
// when merging candidates from several sets into an "all collections" ranking.
func RankBuyout(out []BuyoutCandidate, sortBy string, top int) []BuyoutCandidate {
	key := buyoutKey(sortBy)
	sort.SliceStable(out, func(i, j int) bool {
		ki, kj := key(out[i]), key(out[j])
		if ki != kj {
			return ki > kj
		}
		if out[i].Set != out[j].Set {
			return out[i].Set < out[j].Set
		}
		return out[i].Number < out[j].Number
	})
	if top > 0 && len(out) > top {
		out = out[:top]
	}
	return out
}

// buyoutKey maps a sort mode to a ranking key where a higher value is better.
// "best" rewards a big floor lift and profit achieved with the fewest copies to
// clear (efficiency per copy bought), directly favoring thin, high-upside walls.
func buyoutKey(sortBy string) func(BuyoutCandidate) float64 {
	switch sortBy {
	case "best":
		return func(c BuyoutCandidate) float64 {
			if c.CopiesToClear <= 0 {
				return 0
			}
			return c.LiftPct * c.ProfitBRL / float64(c.CopiesToClear)
		}
	case "lift":
		return func(c BuyoutCandidate) float64 { return c.LiftPct }
	case "profit":
		return func(c BuyoutCandidate) float64 { return c.ProfitBRL }
	case "copies":
		return func(c BuyoutCandidate) float64 { return -float64(c.CopiesToClear) }
	default:
		return func(c BuyoutCandidate) float64 { return c.Score }
	}
}

// nmPriceLevels groups a card's eligible listings (English NM, in stock, with a
// known price) into distinct price levels sorted ascending, and reports the
// total NM supply and seller count. Both Buyout and Snipe build on this ladder.
func nmPriceLevels(c CardDay, langs []string) (levels []*priceLevel, supply, sellers int) {
	byPrice := map[float64]*priceLevel{}
	for _, s := range c.Stores {
		if s.Condition != conditionNM || !langAllowed(s.Language, langs) || !s.PriceKnown || s.PriceBRL <= 0 || s.Quantity <= 0 {
			continue
		}
		lv := byPrice[s.PriceBRL]
		if lv == nil {
			lv = &priceLevel{price: s.PriceBRL, stores: map[int]bool{}}
			byPrice[s.PriceBRL] = lv
		}
		lv.qty += s.Quantity
		lv.stores[s.StoreID] = true
		supply += s.Quantity
		sellers++
	}
	levels = make([]*priceLevel, 0, len(byPrice))
	for _, lv := range byPrice {
		levels = append(levels, lv)
	}
	sort.Slice(levels, func(i, j int) bool { return levels[i].price < levels[j].price })
	return levels, supply, sellers
}

func buyoutForCard(c CardDay, budget, minFloor, shipping float64, langs []string) (BuyoutCandidate, bool) {
	levels, supply, sellers := nmPriceLevels(c, langs)
	if len(levels) < 2 {
		return BuyoutCandidate{}, false
	}

	floor := levels[0].price
	if floor < minFloor {
		return BuyoutCandidate{}, false
	}

	// Clear the cheapest English NM price levels from the bottom. Each distinct
	// store adds one shipping fee, so the running total (cards + shipping) must
	// stay within budget. A store that appears at several levels is charged once.
	cardCost := 0.0
	copies := 0
	cleared := map[int]bool{}
	bestCardCost, bestCopies, bestNext, bestStores := 0.0, 0, 0.0, 0
	for i := 0; i+1 < len(levels); i++ {
		cardCost += levels[i].price * float64(levels[i].qty)
		copies += levels[i].qty
		for id := range levels[i].stores {
			cleared[id] = true
		}
		if cardCost+float64(len(cleared))*shipping > budget {
			break
		}
		bestCardCost, bestCopies, bestNext, bestStores = cardCost, copies, levels[i+1].price, len(cleared)
	}
	if bestNext <= floor || bestCopies == 0 {
		return BuyoutCandidate{}, false
	}

	bestShipping := float64(bestStores) * shipping
	lift := (bestNext - floor) / floor * 100
	profit := float64(bestCopies)*bestNext - bestCardCost - bestShipping

	// Score rewards a big floor lift achieved with FEW copies (thin wall) on a
	// card with LOW total supply (corner-able), gated by a minimum of sellers for
	// liquidity. A grindy card (many copies / high supply) scores far lower even
	// with a comparable headline lift.
	demand := float64(sellers) / 8
	if demand > 1 {
		demand = 1
	}
	corner := 1.0 / (1.0 + float64(supply)/40.0)   // high total supply -> harder to hold the price
	effort := 1.0 / math.Sqrt(float64(bestCopies)) // more copies to clear -> more capital & competing resale
	score := lift * demand * corner * effort

	return BuyoutCandidate{
		Number:        c.Number,
		Name:          c.Name,
		URL:           c.URL,
		Floor:         floor,
		NextFloor:     bestNext,
		LiftPct:       lift,
		BuyoutCost:    bestCardCost,
		ShippingCost:  bestShipping,
		StoreCount:    bestStores,
		CopiesToClear: bestCopies,
		ProfitBRL:     profit,
		NMSupply:      supply,
		Sellers:       sellers,
		Score:         score,
	}, true
}
