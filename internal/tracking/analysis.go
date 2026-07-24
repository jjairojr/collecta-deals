package tracking

import (
	"sort"
	"strconv"
)

type saleKey struct {
	Number    string
	StoreID   int
	Condition string
	Language  string
}

func PriceTrends(today, prev DaySnapshot) []CardTrend {
	prevLow := make(map[string]float64, len(prev.Cards))
	for _, c := range prev.Cards {
		prevLow[c.Number] = c.LowBRL
	}
	out := make([]CardTrend, 0, len(today.Cards))
	for _, c := range today.Cards {
		t := CardTrend{
			Number:  c.Number,
			Name:    c.Name,
			LowBRL:  c.LowBRL,
			PrevBRL: prevLow[c.Number],
			URL:     c.URL,
		}
		if t.PrevBRL > 0 {
			t.DeltaPct = (c.LowBRL - t.PrevBRL) / t.PrevBRL * 100
		}
		out = append(out, t)
	}
	return out
}

func CardHistory(days []DaySnapshot, number string) []PricePoint {
	var out []PricePoint
	for _, d := range days {
		for _, c := range d.Cards {
			if c.Number == number {
				out = append(out, PricePoint{Date: d.Date, LowBRL: c.LowBRL})
				break
			}
		}
	}
	return out
}

func Leaderboard(days []DaySnapshot) []StoreStat {
	type cardAgg struct {
		name    string
		units   int
		revenue float64
	}
	storeCards := map[int]map[string]*cardAgg{}
	names := map[int]string{}

	for i := 1; i < len(days); i++ {
		prevQ := indexQuantities(days[i-1])
		curQ := indexQuantities(days[i])
		curLow := indexLow(days[i])
		curName := indexCardName(days[i])
		for key, pq := range prevQ {
			cq, ok := curQ[key]
			if !ok {
				continue
			}
			if !pq.Known || !cq.Known {
				continue
			}
			drop := pq.Quantity - cq.Quantity
			if drop <= 0 {
				continue
			}
			if cq.StoreName != "" {
				names[key.StoreID] = cq.StoreName
			}
			cm := storeCards[key.StoreID]
			if cm == nil {
				cm = map[string]*cardAgg{}
				storeCards[key.StoreID] = cm
			}
			ca := cm[key.Number]
			if ca == nil {
				ca = &cardAgg{name: curName[key.Number]}
				cm[key.Number] = ca
			}
			ca.units += drop
			ca.revenue += float64(drop) * salePriceBRL(pq, cq, curLow[key.Number])
		}
	}

	out := make([]StoreStat, 0, len(storeCards))
	for id, cm := range storeCards {
		cards := make([]CardSale, 0, len(cm))
		units := 0
		revenue := 0.0
		for num, ca := range cm {
			cards = append(cards, CardSale{Number: num, Name: ca.name, Units: ca.units, RevenueBRL: ca.revenue})
			units += ca.units
			revenue += ca.revenue
		}
		sort.SliceStable(cards, func(i, j int) bool {
			if cards[i].Units != cards[j].Units {
				return cards[i].Units > cards[j].Units
			}
			if cards[i].RevenueBRL != cards[j].RevenueBRL {
				return cards[i].RevenueBRL > cards[j].RevenueBRL
			}
			return cards[i].Number < cards[j].Number
		})
		out = append(out, StoreStat{
			StoreID:    id,
			StoreName:  names[id],
			UnitsSold:  units,
			RevenueBRL: revenue,
			Cards:      cards,
		})
	}
	sortByUnits(out)
	return out
}

func TopSoldCards(days []DaySnapshot) []CardSale {
	groups := make([][]CardSale, 0, len(days))
	for i := 1; i < len(days); i++ {
		groups = append(groups, salesBetween(days[i-1], days[i]))
	}
	out := mergeSales(groups)
	SortCardSales(out)
	return out
}

// SalesBySnapshot reports the cards sold in each individual snapshot interval
// (days[i-1] -> days[i]), newest interval first. Intervals with no inferred
// sales are included with an empty Cards slice.
func SalesBySnapshot(days []DaySnapshot) []SnapshotSales {
	out := make([]SnapshotSales, 0, len(days))
	for i := 1; i < len(days); i++ {
		cards := salesBetween(days[i-1], days[i])
		SortCardSales(cards)
		units, revenue := 0, 0.0
		for _, c := range cards {
			units += c.Units
			revenue += c.RevenueBRL
		}
		out = append(out, SnapshotSales{
			Date:           days[i].Date,
			PrevDate:       days[i-1].Date,
			CapturedAt:     days[i].CapturedAt,
			PrevCapturedAt: days[i-1].CapturedAt,
			Units:          units,
			RevenueBRL:     revenue,
			Cards:          cards,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Date > out[j].Date
	})
	return out
}

// PoolSnapshotSales merges the per-set snapshot timelines into one combined
// timeline, grouping intervals by their target Date so every collection's sales
// for the same capture time appear together. Cards must already carry their Set
// so they stay distinct across collections.
func PoolSnapshotSales(perSet [][]SnapshotSales) []SnapshotSales {
	byDate := map[string]*SnapshotSales{}
	for _, snaps := range perSet {
		for _, s := range snaps {
			agg := byDate[s.Date]
			if agg == nil {
				agg = &SnapshotSales{Date: s.Date, PrevDate: s.PrevDate, Cards: []CardSale{}}
				byDate[s.Date] = agg
			}
			if s.PrevDate > agg.PrevDate {
				agg.PrevDate = s.PrevDate
			}
			// Pooled sets are captured within moments of each other; surface the
			// latest real capture time so the label matches the header, not the
			// storage slot key.
			if s.CapturedAt.After(agg.CapturedAt) {
				agg.CapturedAt = s.CapturedAt
			}
			if s.PrevCapturedAt.After(agg.PrevCapturedAt) {
				agg.PrevCapturedAt = s.PrevCapturedAt
			}
			agg.Units += s.Units
			agg.RevenueBRL += s.RevenueBRL
			agg.Cards = append(agg.Cards, s.Cards...)
		}
	}
	out := make([]SnapshotSales, 0, len(byDate))
	for _, agg := range byDate {
		SortCardSales(agg.Cards)
		out = append(out, *agg)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Date > out[j].Date
	})
	return out
}

// salesBetween infers, for a single snapshot transition, the cards whose
// per-store stock dropped, attributing the drop as units sold at the current
// floor price.
func salesBetween(prev, cur DaySnapshot) []CardSale {
	type agg struct {
		name    string
		url     string
		units   int
		rev     float64
		sellers map[int]*CardSeller
		langs   map[string]*LangSale
	}
	prevQ := indexQuantities(prev)
	curQ := indexQuantities(cur)
	curLow := indexLow(cur)
	curName := indexCardName(cur)
	curURL := indexCardURL(cur)
	byCard := map[string]*agg{}
	for key, pq := range prevQ {
		cq, ok := curQ[key]
		if !ok || !pq.Known || !cq.Known {
			continue
		}
		drop := pq.Quantity - cq.Quantity
		if drop <= 0 {
			continue
		}
		revenue := float64(drop) * salePriceBRL(pq, cq, curLow[key.Number])
		a := byCard[key.Number]
		if a == nil {
			a = &agg{
				name:    curName[key.Number],
				url:     curURL[key.Number],
				sellers: map[int]*CardSeller{},
				langs:   map[string]*LangSale{},
			}
			byCard[key.Number] = a
		}
		a.units += drop
		a.rev += revenue
		addLangSale(a.langs, key.Language, drop, revenue)
		seller := a.sellers[key.StoreID]
		if seller == nil {
			seller = &CardSeller{StoreID: key.StoreID, StoreName: storeLabel(cq)}
			a.sellers[key.StoreID] = seller
		}
		seller.Units += drop
		seller.RevenueBRL += revenue
	}
	out := make([]CardSale, 0, len(byCard))
	for num, a := range byCard {
		sellers := make([]CardSeller, 0, len(a.sellers))
		for _, se := range a.sellers {
			se.PriceBRL = perUnit(se.RevenueBRL, se.Units)
			sellers = append(sellers, *se)
		}
		sortSellers(sellers)
		out = append(out, CardSale{Number: num, Name: a.name, URL: a.url, Units: a.units, RevenueBRL: a.rev, Sellers: sellers, Languages: flattenLangs(a.langs)})
	}
	return out
}

// mergeSales combines per-interval sales into one aggregate per card, summing
// units and revenue and merging each card's sellers by store and units by language.
func mergeSales(groups [][]CardSale) []CardSale {
	type agg struct {
		name    string
		url     string
		units   int
		rev     float64
		sellers map[int]*CardSeller
		langs   map[string]*LangSale
	}
	byCard := map[string]*agg{}
	for _, group := range groups {
		for _, c := range group {
			a := byCard[c.Number]
			if a == nil {
				a = &agg{name: c.Name, url: c.URL, sellers: map[int]*CardSeller{}, langs: map[string]*LangSale{}}
				byCard[c.Number] = a
			}
			a.units += c.Units
			a.rev += c.RevenueBRL
			for _, l := range c.Languages {
				addLangSale(a.langs, l.Code, l.Units, l.RevenueBRL)
			}
			for _, se := range c.Sellers {
				seller := a.sellers[se.StoreID]
				if seller == nil {
					seller = &CardSeller{StoreID: se.StoreID, StoreName: se.StoreName}
					a.sellers[se.StoreID] = seller
				}
				seller.Units += se.Units
				seller.RevenueBRL += se.RevenueBRL
			}
		}
	}
	out := make([]CardSale, 0, len(byCard))
	for num, a := range byCard {
		sellers := make([]CardSeller, 0, len(a.sellers))
		for _, se := range a.sellers {
			se.PriceBRL = perUnit(se.RevenueBRL, se.Units)
			sellers = append(sellers, *se)
		}
		sortSellers(sellers)
		out = append(out, CardSale{Number: num, Name: a.name, URL: a.url, Units: a.units, RevenueBRL: a.rev, Sellers: sellers, Languages: flattenLangs(a.langs)})
	}
	return out
}

// addLangSale accumulates one language's units and revenue for a card.
func addLangSale(langs map[string]*LangSale, code string, units int, revenue float64) {
	l := langs[code]
	if l == nil {
		l = &LangSale{Code: code}
		langs[code] = l
	}
	l.Units += units
	l.RevenueBRL += revenue
}

// flattenLangs orders a card's language split biggest-first so the UI can show
// the dominant printing without re-sorting.
func flattenLangs(langs map[string]*LangSale) []LangSale {
	out := make([]LangSale, 0, len(langs))
	for _, l := range langs {
		out = append(out, *l)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Units != out[j].Units {
			return out[i].Units > out[j].Units
		}
		return out[i].Code < out[j].Code
	})
	return out
}

// salePriceBRL estimates what a store actually sold copies for: its own listed
// price on the day the copies were still in stock (prev), falling back to the
// current-day listed price, then to the card's floor when no price was decoded.
func salePriceBRL(prev, cur StoreQty, floor float64) float64 {
	if prev.PriceKnown && prev.PriceBRL > 0 {
		return prev.PriceBRL
	}
	if cur.PriceKnown && cur.PriceBRL > 0 {
		return cur.PriceBRL
	}
	return floor
}

// perUnit is the average per-unit price for an aggregated seller (revenue/units).
func perUnit(revenue float64, units int) float64 {
	if units <= 0 {
		return 0
	}
	return revenue / float64(units)
}

func sortSellers(sellers []CardSeller) {
	sort.SliceStable(sellers, func(i, j int) bool {
		if sellers[i].Units != sellers[j].Units {
			return sellers[i].Units > sellers[j].Units
		}
		return sellers[i].StoreName < sellers[j].StoreName
	})
}

func SortCardSales(cards []CardSale) {
	sort.SliceStable(cards, func(i, j int) bool {
		if cards[i].Units != cards[j].Units {
			return cards[i].Units > cards[j].Units
		}
		if cards[i].RevenueBRL != cards[j].RevenueBRL {
			return cards[i].RevenueBRL > cards[j].RevenueBRL
		}
		if cards[i].Set != cards[j].Set {
			return cards[i].Set < cards[j].Set
		}
		return cards[i].Number < cards[j].Number
	})
}

func SortLeaderboard(stats []StoreStat, by string) {
	if by == "revenue" {
		sort.SliceStable(stats, func(i, j int) bool {
			if stats[i].RevenueBRL != stats[j].RevenueBRL {
				return stats[i].RevenueBRL > stats[j].RevenueBRL
			}
			return stats[i].UnitsSold > stats[j].UnitsSold
		})
		return
	}
	sortByUnits(stats)
}

func sortByUnits(stats []StoreStat) {
	sort.SliceStable(stats, func(i, j int) bool {
		if stats[i].UnitsSold != stats[j].UnitsSold {
			return stats[i].UnitsSold > stats[j].UnitsSold
		}
		return stats[i].RevenueBRL > stats[j].RevenueBRL
	})
}

func Inventory(day DaySnapshot, topStores, topCards, topHolders int) InventorySummary {
	type agg struct {
		name      string
		units     int
		cards     int
		value     float64
		topBRL    float64
		topNumber string
		topName   string
	}
	stores := map[int]*agg{}
	totalUnits := 0
	totalValue := 0.0

	for _, c := range day.Cards {
		for _, s := range c.Stores {
			if !s.Known || s.Quantity <= 0 {
				continue
			}
			a := stores[s.StoreID]
			if a == nil {
				a = &agg{name: storeLabel(s)}
				stores[s.StoreID] = a
			}
			a.units += s.Quantity
			a.cards++
			a.value += float64(s.Quantity) * c.LowBRL
			if c.LowBRL > a.topBRL {
				a.topBRL = c.LowBRL
				a.topNumber = c.Number
				a.topName = c.Name
			}
			totalUnits += s.Quantity
			totalValue += float64(s.Quantity) * c.LowBRL
		}
	}

	statList := make([]StoreInventoryStat, 0, len(stores))
	for id, a := range stores {
		statList = append(statList, StoreInventoryStat{
			StoreID:       id,
			StoreName:     a.name,
			Units:         a.units,
			Cards:         a.cards,
			ValueBRL:      a.value,
			TopCardNumber: a.topNumber,
			TopCardName:   a.topName,
			TopCardBRL:    a.topBRL,
		})
	}
	sort.SliceStable(statList, func(i, j int) bool {
		if statList[i].ValueBRL != statList[j].ValueBRL {
			return statList[i].ValueBRL > statList[j].ValueBRL
		}
		return statList[i].StoreID < statList[j].StoreID
	})
	if topStores > 0 && len(statList) > topStores {
		statList = statList[:topStores]
	}

	return InventorySummary{
		Date:         day.Date,
		ActiveStores: len(stores),
		TotalUnits:   totalUnits,
		TotalValue:   totalValue,
		Stores:       statList,
		Expensive:    expensiveCards(day, topCards, topHolders),
	}
}

func expensiveCards(day DaySnapshot, topCards, topHolders int) []ExpensiveCard {
	cards := make([]CardDay, len(day.Cards))
	copy(cards, day.Cards)
	sort.SliceStable(cards, func(i, j int) bool {
		if cards[i].LowBRL != cards[j].LowBRL {
			return cards[i].LowBRL > cards[j].LowBRL
		}
		return cards[i].Number < cards[j].Number
	})
	if topCards > 0 && len(cards) > topCards {
		cards = cards[:topCards]
	}
	out := make([]ExpensiveCard, 0, len(cards))
	for _, c := range cards {
		holders := make([]CardHolder, 0, len(c.Stores))
		total := 0
		for _, s := range c.Stores {
			if !s.Known || s.Quantity <= 0 {
				continue
			}
			holders = append(holders, CardHolder{StoreID: s.StoreID, StoreName: storeLabel(s), Quantity: s.Quantity})
			total += s.Quantity
		}
		stores := len(holders)
		sort.SliceStable(holders, func(i, j int) bool {
			if holders[i].Quantity != holders[j].Quantity {
				return holders[i].Quantity > holders[j].Quantity
			}
			return holders[i].StoreID < holders[j].StoreID
		})
		if topHolders > 0 && len(holders) > topHolders {
			holders = holders[:topHolders]
		}
		out = append(out, ExpensiveCard{
			Number:   c.Number,
			Name:     c.Name,
			LowBRL:   c.LowBRL,
			TotalQty: total,
			Stores:   stores,
			Holders:  holders,
		})
	}
	return out
}

func storeLabel(s StoreQty) string {
	if s.StoreName != "" {
		return s.StoreName
	}
	return "Store " + strconv.Itoa(s.StoreID)
}

func indexQuantities(d DaySnapshot) map[saleKey]StoreQty {
	out := make(map[saleKey]StoreQty)
	for _, c := range d.Cards {
		for _, s := range c.Stores {
			key := saleKey{Number: c.Number, StoreID: s.StoreID, Condition: s.Condition, Language: s.Language}
			existing, ok := out[key]
			if !ok {
				out[key] = s
				continue
			}
			existing.Quantity += s.Quantity
			existing.Known = existing.Known && s.Known
			if existing.StoreName == "" {
				existing.StoreName = s.StoreName
			}
			out[key] = existing
		}
	}
	return out
}

func indexLow(d DaySnapshot) map[string]float64 {
	out := make(map[string]float64, len(d.Cards))
	for _, c := range d.Cards {
		out[c.Number] = c.LowBRL
	}
	return out
}

func indexCardName(d DaySnapshot) map[string]string {
	out := make(map[string]string, len(d.Cards))
	for _, c := range d.Cards {
		out[c.Number] = c.Name
	}
	return out
}

func indexCardURL(d DaySnapshot) map[string]string {
	out := make(map[string]string, len(d.Cards))
	for _, c := range d.Cards {
		out[c.Number] = c.URL
	}
	return out
}
