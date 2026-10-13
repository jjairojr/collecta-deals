package api

import (
	"context"
	"errors"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"

	"opdeals/internal/collecta"
	"opdeals/internal/mercadopago"
	"opdeals/internal/orders"
)

const (
	maxCoupons         = 20
	couponReleaseGrace = 30 * time.Minute
	couponCallTimeout  = 5 * time.Second
)

func normalizeCouponIDs(ids []string) ([]string, error) {
	out := make([]string, 0, len(ids))
	seen := make(map[string]bool, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			return nil, errors.New("cupom inválido")
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	if len(out) > maxCoupons {
		return nil, errors.New("cupons demais")
	}
	return out, nil
}

func cents(v float64) int64 {
	return int64(math.Round(v * 100))
}

func brl(c int64) float64 {
	return float64(c) / 100
}

func lineCents(l orders.Line) int64 {
	return int64(l.Qty) * cents(l.UnitBRL)
}

func itemsCents(lines []orders.Line) int64 {
	var sum int64
	for _, l := range lines {
		sum += lineCents(l)
	}
	return sum
}

func couponsReleasable(o orders.Order, now time.Time) bool {
	if len(o.Coupons) == 0 || !o.CouponsReleasedAt.IsZero() {
		return false
	}
	switch orders.EffectiveStatus(o, now) {
	case orders.StatusCancelled, orders.StatusOversold:
		return true
	case orders.StatusExpired, orders.StatusFailed:
		return now.After(o.ExpiresAt.Add(couponReleaseGrace))
	}
	return false
}

func (s *Server) reserveCoupons(ctx context.Context, order orders.Order, couponIDs []string) (orders.Order, int, map[string]any) {
	cancel := func(note string) {
		s.Checkout.Orders.Update(order.ID, func(o *orders.Order) {
			o.Status = orders.StatusCancelled
			o.Notes = appendNote(o.Notes, note)
		})
	}
	reserved, err := s.Checkout.Coupons.Reserve(ctx, order.CollectaUserID, couponIDs, itemsCents(order.Lines))
	if err != nil {
		cancel("cupom recusado: " + err.Error())
		var apiErr *collecta.APIError
		if errors.As(err, &apiErr) && apiErr.Rejected() {
			return order, http.StatusConflict, map[string]any{"error": "coupon_invalid", "message": apiErr.Message}
		}
		s.logCheckout("reserva de cupons pedido #%d: %v", order.Number, err)
		return order, http.StatusBadGateway, map[string]any{"error": "cupons indisponíveis no momento"}
	}
	coupons := make([]orders.Coupon, 0, len(reserved))
	var discount int64
	for _, c := range reserved {
		coupons = append(coupons, orders.Coupon{ID: c.ID, Code: c.Code, ValueBRL: brl(c.ValueCents)})
		discount += c.ValueCents
	}
	items := itemsCents(order.Lines)
	updated, err := s.Checkout.Orders.Update(order.ID, func(o *orders.Order) {
		o.Coupons = coupons
		o.Totals.DiscountBRL = brl(discount)
		o.Totals.TotalBRL = brl(max(items+cents(o.Totals.ShippingBRL)-discount, 0))
	})
	if err != nil {
		if rerr := s.Checkout.Coupons.Release(ctx, codesOf(coupons)); rerr != nil {
			s.logCheckout("release após falha de gravação pedido #%d: %v", order.Number, rerr)
		}
		cancel("gravação dos cupons falhou: " + err.Error())
		return order, http.StatusInternalServerError, map[string]any{"error": err.Error()}
	}
	if discount >= items {
		cancel("desconto cobre o pedido inteiro")
		s.releaseLater(ctx, updated)
		return updated, http.StatusConflict, map[string]any{"error": "coupon_invalid", "message": "o desconto não pode cobrir o pedido inteiro"}
	}
	return updated, 0, nil
}

func codesOf(coupons []orders.Coupon) []string {
	codes := make([]string, 0, len(coupons))
	for _, c := range coupons {
		codes = append(codes, c.Code)
	}
	return codes
}

func (s *Server) releaseLater(ctx context.Context, o orders.Order) {
	cctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), couponCallTimeout)
	defer cancel()
	if err := s.releaseCoupons(cctx, o); err != nil {
		s.logCheckout("release cupons pedido #%d: %v", o.Number, err)
	}
}

func (s *Server) releaseCoupons(ctx context.Context, o orders.Order) error {
	if s.Checkout.Coupons == nil || len(o.Coupons) == 0 {
		return nil
	}
	if _, busy := s.releasing.LoadOrStore(o.ID, struct{}{}); busy {
		return nil
	}
	defer s.releasing.Delete(o.ID)
	now := time.Now()
	current, err := s.Checkout.Orders.Get(o.ID)
	if err != nil {
		return err
	}
	if !couponsReleasable(current, now) {
		return nil
	}
	all, err := s.Checkout.Orders.List()
	if err != nil {
		return err
	}
	held := make(map[string]bool)
	for _, other := range all {
		if other.ID == current.ID {
			continue
		}
		switch orders.EffectiveStatus(other, now) {
		case orders.StatusPending, orders.StatusPaid, orders.StatusShipped:
			for _, c := range other.Coupons {
				held[c.Code] = true
			}
		}
	}
	codes := make([]string, 0, len(current.Coupons))
	for _, c := range current.Coupons {
		if !held[c.Code] {
			codes = append(codes, c.Code)
		}
	}
	if len(codes) > 0 {
		if err := s.Checkout.Coupons.Release(ctx, codes); err != nil {
			return err
		}
	}
	_, err = s.Checkout.Orders.Transact(current.ID, func(cur *orders.Order) error {
		if cur.CouponsReleasedAt.IsZero() {
			cur.CouponsReleasedAt = now
		}
		return nil
	})
	return err
}

func (s *Server) SweepCoupons(ctx context.Context, every time.Duration) {
	s.sweepCoupons(ctx)
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.sweepCoupons(ctx)
		}
	}
}

func (s *Server) sweepCoupons(ctx context.Context) {
	if s.Checkout.Coupons == nil || s.Checkout.Orders == nil {
		return
	}
	all, err := s.Checkout.Orders.List()
	if err != nil {
		s.logCheckout("sweep cupons: %v", err)
		return
	}
	now := time.Now()
	for _, o := range all {
		if !couponsReleasable(o, now) {
			continue
		}
		cctx, cancel := context.WithTimeout(ctx, couponCallTimeout)
		err := s.releaseCoupons(cctx, o)
		cancel()
		if err == nil {
			continue
		}
		s.logCheckout("sweep release pedido #%d: %v", o.Number, err)
		var apiErr *collecta.APIError
		if !errors.As(err, &apiErr) {
			return
		}
	}
}

func discountShares(lines []orders.Line, discount int64) []int64 {
	shares := make([]int64, len(lines))
	total := itemsCents(lines)
	if discount <= 0 || total <= 0 {
		return shares
	}
	type frac struct {
		idx int
		rem int64
	}
	fracs := make([]frac, 0, len(lines))
	var given int64
	for i, l := range lines {
		raw := discount * lineCents(l)
		shares[i] = raw / total
		given += shares[i]
		fracs = append(fracs, frac{idx: i, rem: raw % total})
	}
	sort.SliceStable(fracs, func(i, j int) bool { return fracs[i].rem > fracs[j].rem })
	for i := int64(0); i < discount-given && int(i) < len(fracs); i++ {
		shares[fracs[i].idx]++
	}
	return shares
}

func lineTitle(l orders.Line) string {
	title := l.Name
	if l.Meta != "" {
		title += " (" + l.Meta + ")"
	}
	if len(title) > 250 {
		title = title[:250]
	}
	return title
}

func discountedItems(lines []orders.Line, discount int64) []mercadopago.Item {
	items := make([]mercadopago.Item, 0, len(lines)+1)
	shares := discountShares(lines, discount)
	for i, l := range lines {
		title := lineTitle(l)
		if shares[i] == 0 {
			items = append(items, mercadopago.Item{ID: l.SKU, Title: title, Qty: l.Qty, UnitBRL: l.UnitBRL})
			continue
		}
		qty := int64(l.Qty)
		remaining := lineCents(l) - shares[i]
		unit := remaining / qty
		rem := remaining % qty
		if unit == 0 {
			items = append(items, mercadopago.Item{ID: l.SKU, Title: title, Qty: 1, UnitBRL: brl(remaining)})
			continue
		}
		if qty-rem > 0 {
			items = append(items, mercadopago.Item{ID: l.SKU, Title: title, Qty: int(qty - rem), UnitBRL: brl(unit)})
		}
		if rem > 0 {
			items = append(items, mercadopago.Item{ID: l.SKU, Title: title, Qty: int(rem), UnitBRL: brl(unit + 1)})
		}
	}
	return items
}
