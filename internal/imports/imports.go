package imports

import (
	"sort"
	"strings"

	"opdeals/internal/compare"
	"opdeals/internal/model"
	"opdeals/internal/tracking"
)

const undercutBRL = 0.01

type Deal struct {
	Game         string  `json:"game"`
	Set          string  `json:"set"`
	Number       string  `json:"number"`
	Name         string  `json:"name"`
	TCGName      string  `json:"tcgName"`
	Rarity       string  `json:"rarity"`
	Variant      string  `json:"variant"`
	LigaURL      string  `json:"ligaUrl"`
	TCGURL       string  `json:"tcgUrl"`
	ProductID    int     `json:"productId"`
	CostUSD      float64 `json:"costUSD"`
	CostBRL      float64 `json:"costBRL"`
	FloorBRL     float64 `json:"floorBRL"`
	SellBRL      float64 `json:"sellBRL"`
	ProfitBRL    float64 `json:"profitBRL"`
	MarginPct    float64 `json:"marginPct"`
	Sellers      int     `json:"sellers"`
	FloorCopies  int     `json:"floorCopies"`
	Supply       int     `json:"supply"`
	OwnQty       int     `json:"ownQty"`
	OwnPriceBRL  float64 `json:"ownPriceBRL"`
	USListings   int     `json:"usListings"`
	USQty        int     `json:"usQty"`
	USVariants   int     `json:"usVariants"`
	USLive       bool    `json:"usLive"`
	TrackingDate string  `json:"trackingDate"`
}

type BuildOptions struct {
	Game   string
	FXRate float64
}

func Build(supply []tracking.ImportSupply, prices []model.USPrice, m compare.Matcher, opts BuildOptions) []Deal {
	if opts.FXRate <= 0 {
		return []Deal{}
	}
	groups := compare.USDGroups(prices, m)
	out := make([]Deal, 0, len(supply))
	for _, s := range supply {
		group := groups[m.LookupKey(s.Number, s.Name, s.Set)]
		us, ok := compare.Dearest(group)
		if !ok {
			continue
		}
		costUSD := compare.EffectiveUSD(us)
		sell := s.FloorBRL - undercutBRL
		if costUSD <= 0 || sell <= 0 {
			continue
		}
		costBRL := costUSD / opts.FXRate
		profit := sell - costBRL
		d := Deal{
			Game:         opts.Game,
			Set:          s.Set,
			Number:       s.Number,
			Name:         s.Name,
			TCGName:      us.Name,
			Rarity:       us.Rarity,
			Variant:      us.Variant,
			LigaURL:      s.URL,
			TCGURL:       us.URL,
			ProductID:    us.ProductID,
			CostUSD:      costUSD,
			CostBRL:      costBRL,
			FloorBRL:     s.FloorBRL,
			SellBRL:      sell,
			ProfitBRL:    profit,
			MarginPct:    profit / costBRL * 100,
			Sellers:      s.Sellers,
			FloorCopies:  s.FloorCopies,
			Supply:       s.Supply,
			OwnQty:       s.OwnQty,
			OwnPriceBRL:  s.OwnPriceBRL,
			USListings:   us.LiveListings,
			USQty:        us.LiveQty,
			USVariants:   len(group),
			USLive:       us.LiveUSD > 0,
			TrackingDate: s.Date,
		}
		out = append(out, d)
	}
	return out
}

type Filter struct {
	Games      []string
	MinSellers int
	MinUSD     float64
	MinProfit  float64
	SortBy     string
	Limit      int
}

func Apply(deals []Deal, f Filter) []Deal {
	games := make(map[string]bool, len(f.Games))
	for _, g := range f.Games {
		games[g] = true
	}
	out := make([]Deal, 0, len(deals))
	for _, d := range deals {
		if len(games) > 0 && !games[d.Game] {
			continue
		}
		if d.Sellers < f.MinSellers || d.CostUSD < f.MinUSD || d.ProfitBRL < f.MinProfit {
			continue
		}
		out = append(out, d)
	}
	key := sortKey(f.SortBy)
	sort.SliceStable(out, func(i, j int) bool {
		ki, kj := key(out[i]), key(out[j])
		if ki != kj {
			return ki > kj
		}
		if out[i].Game != out[j].Game {
			return out[i].Game < out[j].Game
		}
		if out[i].Set != out[j].Set {
			return out[i].Set < out[j].Set
		}
		return out[i].Number < out[j].Number
	})
	if f.Limit > 0 && len(out) > f.Limit {
		out = out[:f.Limit]
	}
	return out
}

func sortKey(by string) func(Deal) float64 {
	if strings.EqualFold(by, "margin") {
		return func(d Deal) float64 { return d.MarginPct }
	}
	return func(d Deal) float64 { return d.ProfitBRL }
}
