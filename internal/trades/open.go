package trades

import (
	"errors"
	"time"
)

var ErrInvalidOpen = errors.New("abertura inválida")

func (s *Store) Open(parentID string, qty, units int, template Trade) (Trade, error) {
	if qty <= 0 || units < 2 {
		return Trade{}, ErrInvalidOpen
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	all, err := s.load()
	if err != nil {
		return Trade{}, err
	}
	at := -1
	for i := range all {
		if all[i].ID == parentID {
			at = i
			break
		}
	}
	if at < 0 {
		return Trade{}, ErrNotFound
	}
	parent := all[at]
	if parent.Status == "sold" {
		return Trade{}, ErrAlreadySold
	}
	if parent.Qty <= 0 {
		parent.Qty = 1
	}
	if qty > parent.Qty {
		return Trade{}, ErrInvalidOpen
	}
	now := time.Now()
	ship := round2(parent.ShippingBRL * float64(qty) / float64(parent.Qty))
	orderShip := round2(parent.OrderShippingBRL * float64(qty) / float64(parent.Qty))
	buy := round2(parent.BuyBRL / float64(units))
	added := qty * units
	sku := SKU("", template)

	var child Trade
	target := -1
	for i := range all {
		t := all[i]
		if i == at || t.Status != "holding" || SKU("", t) != sku {
			continue
		}
		if t.PurchaseID == parent.PurchaseID && t.BuyBRL == buy {
			target = i
			break
		}
	}
	if target >= 0 {
		c := &all[target]
		if c.Qty <= 0 {
			c.Qty = 1
		}
		c.Qty += added
		c.ShippingBRL = round2(c.ShippingBRL + ship)
		c.OrderShippingBRL = round2(c.OrderShippingBRL + orderShip)
		c.UpdatedAt = now
		child = *c
	} else {
		child = template
		child.ID = newID()
		child.Qty = added
		child.BuyBRL = buy
		child.ShippingBRL = ship
		child.OrderShippingBRL = orderShip
		child.Store = parent.Store
		child.BuyDate = parent.BuyDate
		child.Delivered = parent.Delivered
		child.PurchaseID = parent.PurchaseID
		child.Status = "holding"
		child.SellPrice = 0
		child.SellCurrency = ""
		child.SellDate = ""
		child.Buyer = ""
		child.Channel = ""
		child.WriteOff = ""
		child.SemLastro = false
		child.SplitFrom = ""
		child.Notes = "Aberto de " + parent.Name
		child.CreatedAt = now
		child.UpdatedAt = now
		if child.Slug == "" {
			child.Slug = SlugBase(child)
		}
		all = append(all, child)
	}

	if qty == parent.Qty {
		all = append(all[:at:at], all[at+1:]...)
	} else {
		p := &all[at]
		p.Qty = parent.Qty - qty
		p.ShippingBRL = round2(parent.ShippingBRL - ship)
		p.OrderShippingBRL = round2(parent.OrderShippingBRL - orderShip)
		p.UpdatedAt = now
	}
	if err := s.persist(all); err != nil {
		return Trade{}, err
	}
	return child, nil
}
