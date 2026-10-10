package imports

import (
	"math"
	"sort"
	"strings"

	"opdeals/internal/compare"
	"opdeals/internal/model"
	"opdeals/internal/tracking"
)

const (
	undercutUSD = 0.01
	copiesCap   = 40.0
)

type ExportDeal struct {
	Game         string                `json:"game"`
	Set          string                `json:"set"`
	Number       string                `json:"number"`
	Name         string                `json:"name"`
	TCGName      string                `json:"tcgName"`
	Rarity       string                `json:"rarity"`
	Variant      string                `json:"variant"`
	LigaURL      string                `json:"ligaUrl"`
	TCGURL       string                `json:"tcgUrl"`
	ProductID    int                   `json:"productId"`
	CostBRL      float64               `json:"costBRL"`
	CostUSD      float64               `json:"costUSD"`
	SellUSD      float64               `json:"sellUSD"`
	SellBRL      float64               `json:"sellBRL"`
	ProfitUSD    float64               `json:"profitUSD"`
	ProfitBRL    float64               `json:"profitBRL"`
	MarginPct    float64               `json:"marginPct"`
	Score        float64               `json:"score"`
	FloorCopies  int                   `json:"floorCopies"`
	Sellers      int                   `json:"sellers"`
	Supply       int                   `json:"supply"`
	OwnQty       int                   `json:"ownQty"`
	OwnPriceBRL  float64               `json:"ownPriceBRL"`
	FloorStores  []tracking.FloorStore `json:"floorStores"`
	USListings   int                   `json:"usListings"`
	USQty        int                   `json:"usQty"`
	USVariants   int                   `json:"usVariants"`
	SuggestedQty int                   `json:"suggestedQty"`
	USLive       bool                  `json:"usLive"`
	TrackingDate string                `json:"trackingDate"`
}

func cheapest(group []model.USPrice) (model.USPrice, bool) {
	var best model.USPrice
	found := false
	for _, p := range group {
		if !found || compare.EffectiveUSD(p) < compare.EffectiveUSD(best) {
			best = p
			found = true
		}
	}
	return best, found
}

func BuildExport(supply []tracking.ImportSupply, prices []model.USPrice, m compare.Matcher, opts BuildOptions) []ExportDeal {
	if opts.FXRate <= 0 {
		return []ExportDeal{}
	}
	groups := compare.USDGroups(prices, m)
	out := make([]ExportDeal, 0, len(supply))
	for _, s := range supply {
		group := groups[m.LookupKey(s.Number, s.Name, s.Set)]
		us, ok := cheapest(group)
		if !ok || s.FloorBRL <= 0 {
			continue
		}
		sell := compare.EffectiveUSD(us) - undercutUSD
		if sell <= 0 {
			continue
		}
		costUSD := s.FloorBRL * opts.FXRate
		profitUSD := sell - costUSD
		stores := s.FloorStores
		if stores == nil {
			stores = []tracking.FloorStore{}
		}
		d := ExportDeal{
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
			CostBRL:      s.FloorBRL,
			CostUSD:      costUSD,
			SellUSD:      sell,
			SellBRL:      sell / opts.FXRate,
			ProfitUSD:    profitUSD,
			ProfitBRL:    profitUSD / opts.FXRate,
			MarginPct:    profitUSD / costUSD * 100,
			Score:        profitUSD / opts.FXRate * math.Min(float64(s.FloorCopies), copiesCap),
			FloorCopies:  s.FloorCopies,
			Sellers:      s.Sellers,
			Supply:       s.Supply,
			OwnQty:       s.OwnQty,
			OwnPriceBRL:  s.OwnPriceBRL,
			FloorStores:  stores,
			USListings:   us.LiveListings,
			USQty:        us.LiveQty,
			USVariants:   len(group),
			SuggestedQty: int(math.Max(1, float64(s.FloorCopies))),
			USLive:       us.LiveUSD > 0,
			TrackingDate: s.Date,
		}
		out = append(out, d)
	}
	return out
}

type ExportFilter struct {
	Games         []string
	MinSellers    int
	MinUSListings int
	MinUSD        float64
	MinProfit     float64
	SortBy        string
	Limit         int
}

func ApplyExport(deals []ExportDeal, f ExportFilter) []ExportDeal {
	games := make(map[string]bool, len(f.Games))
	for _, g := range f.Games {
		games[g] = true
	}
	out := make([]ExportDeal, 0, len(deals))
	for _, d := range deals {
		if len(games) > 0 && !games[d.Game] {
			continue
		}
		if d.Sellers < f.MinSellers || d.USListings < f.MinUSListings || d.SellUSD < f.MinUSD || d.ProfitBRL < f.MinProfit {
			continue
		}
		out = append(out, d)
	}
	key := exportSortKey(f.SortBy)
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

func exportSortKey(by string) func(ExportDeal) float64 {
	switch strings.ToLower(by) {
	case "profit":
		return func(d ExportDeal) float64 { return d.ProfitBRL }
	case "margin":
		return func(d ExportDeal) float64 { return d.MarginPct }
	case "copies":
		return func(d ExportDeal) float64 { return float64(d.FloorCopies) }
	default:
		return func(d ExportDeal) float64 { return d.Score }
	}
}
