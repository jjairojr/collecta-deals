package tracking

import "sort"

type MarketPrice struct {
	LowBRL    float64 `json:"lowBRL"`
	MedianBRL float64 `json:"medianBRL"`
	Date      string  `json:"date"`
	Days      int     `json:"days"`
}

func CardMarket(days []DaySnapshot, number, lang string, excludeStore int) (MarketPrice, bool) {
	var floors []float64
	var last MarketPrice
	for _, d := range days {
		var match []CardDay
		for _, c := range d.Cards {
			if c.Number == number {
				match = append(match, c)
			}
		}
		if len(match) > 1 {
			return MarketPrice{}, false
		}
		if len(match) == 0 {
			continue
		}
		levels, _, _ := nmPriceLevelsExcluding(match[0], []string{lang}, excludeStore)
		if len(levels) == 0 {
			continue
		}
		floors = append(floors, levels[0].price)
		last = MarketPrice{LowBRL: levels[0].price, Date: d.Date}
	}
	if len(floors) == 0 {
		return MarketPrice{}, false
	}
	sorted := append([]float64(nil), floors...)
	sort.Float64s(sorted)
	mid := len(sorted) / 2
	last.MedianBRL = sorted[mid]
	if len(sorted)%2 == 0 {
		last.MedianBRL = (sorted[mid-1] + sorted[mid]) / 2
	}
	last.Days = len(floors)
	return last, true
}
