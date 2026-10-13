package checkoutfake

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"opdeals/internal/collecta"
	"opdeals/internal/mercadopago"
	"opdeals/internal/shipping"
)

type Gateway struct {
	mu       sync.Mutex
	next     int64
	payments map[int64]mercadopago.Payment
}

func NewGateway() *Gateway {
	return &Gateway{next: 1000, payments: map[int64]mercadopago.Payment{}}
}

func (g *Gateway) CreatePreference(_ context.Context, in mercadopago.PreferenceInput) (mercadopago.Preference, error) {
	total := 0.0
	for _, it := range in.Items {
		total += float64(it.Qty) * it.UnitBRL
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.next++
	id := g.next
	g.payments[id] = mercadopago.Payment{
		ID:                id,
		Status:            "approved",
		StatusDetail:      "accredited",
		ExternalReference: in.OrderID,
		AmountBRL:         total,
		Method:            "pix",
	}
	sep := "?"
	if strings.Contains(in.ReturnURL, "?") {
		sep = "&"
	}
	return mercadopago.Preference{
		ID:        fmt.Sprintf("fake-%d", id),
		InitPoint: fmt.Sprintf("%s%sr=success&payment_id=%d&status=approved", in.ReturnURL, sep, id),
	}, nil
}

func (g *Gateway) GetPayment(_ context.Context, id int64) (mercadopago.Payment, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	p, ok := g.payments[id]
	if !ok {
		return mercadopago.Payment{}, fmt.Errorf("fake payment %d not found", id)
	}
	return p, nil
}

type Quoter struct{}

func (Quoter) Quote(_ context.Context, _, toCEP string, products []shipping.Product) ([]shipping.Service, error) {
	if strings.HasPrefix(toCEP, "99") {
		return nil, nil
	}
	weight := 0.0
	for _, p := range products {
		weight += p.Weight * float64(p.Quantity)
	}
	extra := weight * 10
	return []shipping.Service{
		{ID: 17, Name: "Mini Envios", Company: "Correios", PriceBRL: 12.5 + extra, DeliveryDays: 9},
		{ID: 1, Name: "PAC", Company: "Correios", PriceBRL: 21.9 + extra, DeliveryDays: 7},
		{ID: 2, Name: "SEDEX", Company: "Correios", PriceBRL: 38.4 + extra, DeliveryDays: 3},
	}, nil
}

type Coupons struct{}

func (Coupons) Reserve(_ context.Context, _ string, couponIDs []string, _ int64) ([]collecta.Coupon, error) {
	out := make([]collecta.Coupon, 0, len(couponIDs))
	for _, id := range couponIDs {
		out = append(out, collecta.Coupon{ID: id, Code: "IND-" + strings.ToUpper(id), ValueCents: 1000})
	}
	return out, nil
}

func (Coupons) Release(_ context.Context, _ []string) error {
	return nil
}
