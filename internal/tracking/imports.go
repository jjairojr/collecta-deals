package tracking

import (
	"sort"
	"strings"
)

type ImportSupply struct {
	Set         string
	Number      string
	Name        string
	URL         string
	Date        string
	FloorBRL    float64
	FloorCopies int
	Supply      int
	Sellers     int
	OwnQty      int
	OwnPriceBRL float64
	FloorStores []FloorStore
}

type FloorStore struct {
	StoreID   int    `json:"storeId"`
	StoreName string `json:"storeName"`
	Qty       int    `json:"qty"`
}

func ImportSupplyFor(day DaySnapshot, langs []string, excludeStore int) []ImportSupply {
	out := make([]ImportSupply, 0, len(day.Cards))
	for _, c := range day.Cards {
		if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(c.Number)), "DON-") {
			continue
		}
		levels, supply, _ := nmPriceLevelsExcluding(c, langs, excludeStore)
		if len(levels) == 0 {
			continue
		}
		stores := map[int]bool{}
		for _, lv := range levels {
			for id := range lv.stores {
				stores[id] = true
			}
		}
		s := ImportSupply{
			Set:         day.Set,
			Number:      c.Number,
			Name:        c.Name,
			URL:         c.URL,
			Date:        day.Date,
			FloorBRL:    levels[0].price,
			FloorCopies: levels[0].qty,
			Supply:      supply,
			Sellers:     len(stores),
		}
		s.FloorStores = floorStores(c, langs, excludeStore, levels[0].price)
		s.OwnQty, s.OwnPriceBRL = ownListing(c, langs, excludeStore)
		out = append(out, s)
	}
	return out
}

func floorStores(c CardDay, langs []string, excludeStore int, floor float64) []FloorStore {
	byStore := map[int]*FloorStore{}
	for _, s := range c.Stores {
		if s.Condition != conditionNM || !langAllowed(s.Language, langs) || !s.PriceKnown || s.PriceBRL != floor || s.Quantity <= 0 {
			continue
		}
		if excludeStore != 0 && s.StoreID == excludeStore {
			continue
		}
		fs := byStore[s.StoreID]
		if fs == nil {
			fs = &FloorStore{StoreID: s.StoreID, StoreName: storeLabel(s)}
			byStore[s.StoreID] = fs
		}
		fs.Qty += s.Quantity
	}
	out := make([]FloorStore, 0, len(byStore))
	for _, fs := range byStore {
		out = append(out, *fs)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Qty != out[j].Qty {
			return out[i].Qty > out[j].Qty
		}
		return out[i].StoreID < out[j].StoreID
	})
	return out
}

func ownListing(c CardDay, langs []string, store int) (int, float64) {
	if store == 0 {
		return 0, 0
	}
	qty, price := 0, 0.0
	for _, s := range c.Stores {
		if s.StoreID != store || s.Condition != conditionNM || !langAllowed(s.Language, langs) || s.Quantity <= 0 {
			continue
		}
		qty += s.Quantity
		if s.PriceKnown && s.PriceBRL > 0 && (price == 0 || s.PriceBRL < price) {
			price = s.PriceBRL
		}
	}
	return qty, price
}
