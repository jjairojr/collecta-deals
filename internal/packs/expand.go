package packs

import (
	"sort"
	"strings"

	"opdeals/internal/trades"
)

const VirtualPrefix = "pack:"

type Origin struct {
	ParentID string
	Units    int
}

func IsVirtual(id string) bool {
	return strings.HasPrefix(id, VirtualPrefix)
}

func Template(game string, all []trades.Trade, sku string) (trades.Trade, bool) {
	var best trades.Trade
	found := false
	for _, t := range all {
		if IsVirtual(t.ID) || trades.SKU(game, t) != sku {
			continue
		}
		if !found || better(t, best) {
			best = t
			found = true
		}
	}
	return best, found
}

func better(a, b trades.Trade) bool {
	if (a.Status == "holding") != (b.Status == "holding") {
		return a.Status == "holding"
	}
	return a.CreatedAt.After(b.CreatedAt)
}

func Expand(game string, all []trades.Trade, links []Link) ([]trades.Trade, map[string]Origin) {
	out := append(make([]trades.Trade, 0, len(all)+2), all...)
	origins := make(map[string]Origin)
	for _, l := range ForGame(links, game) {
		template, ok := Template(game, all, l.ChildSKU)
		if !ok {
			continue
		}
		for _, p := range all {
			if p.Status != "holding" || trades.SKU(game, p) != l.ParentSKU {
				continue
			}
			qty := p.Qty
			if qty <= 0 {
				qty = 1
			}
			v := template
			v.ID = VirtualPrefix + p.ID
			v.Qty = qty * l.Units
			v.BuyBRL = p.BuyBRL / float64(l.Units)
			v.ShippingBRL = 0
			v.OrderShippingBRL = 0
			v.BuyDate = ""
			v.PurchaseID = p.PurchaseID
			v.Status = "holding"
			v.SellPrice = 0
			v.SellCurrency = ""
			v.SellDate = ""
			v.Buyer = ""
			v.Channel = ""
			v.WriteOff = ""
			v.SemLastro = false
			v.SplitFrom = ""
			v.CreatedAt = p.CreatedAt
			out = append(out, v)
			origins[v.ID] = Origin{ParentID: p.ID, Units: l.Units}
		}
	}
	return out, origins
}

func Claims(game string, all []trades.Trade, links []Link, reserved map[string]int) map[string]int {
	out := make(map[string]int, len(reserved))
	for k, v := range reserved {
		out[k] = v
	}
	own := make(map[string]int)
	for _, t := range all {
		if IsVirtual(t.ID) || t.Status != "holding" || !t.Listed || t.AskBRL <= 0 {
			continue
		}
		qty := t.Qty
		if qty <= 0 {
			qty = 1
		}
		own[trades.SKU(game, t)] += qty
	}
	scoped := leafFirst(ForGame(links, game))
	opened := make(map[string]int, len(scoped))
	for _, l := range scoped {
		excess := reserved[l.ChildSKU] + opened[l.ChildSKU] - own[l.ChildSKU]
		if excess > 0 {
			opened[l.ParentSKU] += (excess + l.Units - 1) / l.Units
		}
	}
	for _, l := range scoped {
		out[l.ParentSKU] += opened[l.ParentSKU]
		out[l.ChildSKU] += reserved[l.ParentSKU] * l.Units
	}
	return out
}

func leafFirst(links []Link) []Link {
	childOf := make(map[string]string, len(links))
	for _, l := range links {
		childOf[l.ParentSKU] = l.ChildSKU
	}
	height := func(l Link) int {
		n := 0
		for sku, ok := childOf[l.ChildSKU]; ok && n <= len(links); sku, ok = childOf[sku] {
			n++
		}
		return n
	}
	out := append([]Link(nil), links...)
	sort.SliceStable(out, func(i, j int) bool { return height(out[i]) < height(out[j]) })
	return out
}
