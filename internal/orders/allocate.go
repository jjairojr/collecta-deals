package orders

import (
	"sort"

	"opdeals/internal/trades"
)

type Pick struct {
	TradeID string
	Qty     int
}

func Eligible(ts []trades.Trade) []trades.Trade {
	out := make([]trades.Trade, 0, len(ts))
	for _, t := range ts {
		if t.Status != "holding" || !t.Listed || t.AskBRL <= 0 {
			continue
		}
		if t.Qty <= 0 {
			t.Qty = 1
		}
		out = append(out, t)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].BuyDate != out[j].BuyDate {
			if out[i].BuyDate == "" {
				return false
			}
			if out[j].BuyDate == "" {
				return true
			}
			return out[i].BuyDate < out[j].BuyDate
		}
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})
	return out
}

func Available(ts []trades.Trade) int {
	n := 0
	for _, t := range Eligible(ts) {
		n += t.Qty
	}
	return n
}

func UnitPrice(ts []trades.Trade) (float64, bool) {
	price := 0.0
	found := false
	for _, t := range Eligible(ts) {
		if !found || t.AskBRL < price {
			price = t.AskBRL
			found = true
		}
	}
	return price, found
}

func Allocate(ts []trades.Trade, qty int) ([]Pick, bool) {
	if qty <= 0 {
		return nil, false
	}
	picks := make([]Pick, 0, 2)
	left := qty
	for _, t := range Eligible(ts) {
		take := t.Qty
		if take > left {
			take = left
		}
		picks = append(picks, Pick{TradeID: t.ID, Qty: take})
		left -= take
		if left == 0 {
			return picks, true
		}
	}
	return nil, false
}
