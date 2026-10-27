package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"opdeals/internal/orders"
)

type promo struct {
	PercentBps     int64
	FirstOrderOnly bool
}

var promos = map[string]promo{
	"BEMVINDO": {PercentBps: 500, FirstOrderOnly: true},
}

const (
	promoUnknown    = "Cupom não encontrado"
	promoNotFirst   = "Esse cupom vale só na primeira compra"
	promoNoStacking = "Esse cupom não acumula com cupons de indicação"
)

func normalizePromoCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

func lookupPromo(code string) (promo, bool) {
	p, ok := promos[normalizePromoCode(code)]
	return p, ok
}

func promoDiscountCents(items int64, p promo) int64 {
	return items * p.PercentBps / 10000
}

func countsAsPurchase(o orders.Order, now time.Time) bool {
	switch orders.EffectiveStatus(o, now) {
	case orders.StatusPending, orders.StatusPaid, orders.StatusShipped, orders.StatusDelivered, orders.StatusRefunded:
		return true
	}
	return false
}

func hasPurchase(existing []orders.Order, userID, cpf string, now time.Time) bool {
	for _, o := range existing {
		if !countsAsPurchase(o, now) {
			continue
		}
		if o.CollectaUserID == userID || (cpf != "" && o.Buyer.CPF == cpf) {
			return true
		}
	}
	return false
}

func promoRejection(p promo, existing []orders.Order, userID, cpf string, now time.Time) string {
	if p.FirstOrderOnly && hasPurchase(existing, userID, cpf, now) {
		return promoNotFirst
	}
	return ""
}

func (s *Server) handlePromoCheck(w http.ResponseWriter, r *http.Request) {
	if !s.requireStorefront(w, r) {
		return
	}
	var in struct {
		Code           string `json:"code"`
		CollectaUserID string `json:"collectaUserId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	userID := strings.TrimSpace(in.CollectaUserID)
	if userID == "" {
		writeError(w, http.StatusBadRequest, map[string]any{"error": "cupom exige login"})
		return
	}
	code := normalizePromoCode(in.Code)
	p, ok := promos[code]
	if !ok {
		writeError(w, http.StatusConflict, map[string]any{"error": "coupon_invalid", "message": promoUnknown})
		return
	}
	all, err := s.Checkout.Orders.List()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if msg := promoRejection(p, all, userID, "", time.Now()); msg != "" {
		writeError(w, http.StatusConflict, map[string]any{"error": "coupon_invalid", "message": msg})
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, map[string]any{"code": code, "percentBps": p.PercentBps})
}
