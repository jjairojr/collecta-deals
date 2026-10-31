package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"opdeals/internal/orders"
	"opdeals/internal/promos"
)

const (
	promoUnknown    = "Cupom não encontrado"
	promoNotFirst   = "Esse cupom vale só na primeira compra"
	promoNoStacking = "Esse cupom não acumula com cupons de indicação"
	promoInactive   = "Esse cupom não está ativo"
	promoNotYet     = "Esse cupom ainda não começou a valer"
	promoExpired    = "Esse cupom expirou"
	promoExhausted  = "Esse cupom esgotou"
)

func normalizePromoCode(code string) string {
	return promos.NormalizeCode(code)
}

func (s *Server) lookupPromo(code string) (promos.Promo, bool) {
	if s.Checkout.Promos == nil {
		return promos.Promo{}, false
	}
	p, err := s.Checkout.Promos.Get(code)
	if err != nil {
		return promos.Promo{}, false
	}
	return p, true
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

func promoUses(existing []orders.Order, code string, now time.Time) int {
	n := 0
	for _, o := range existing {
		if o.Promo != nil && o.Promo.Code == code && countsAsPurchase(o, now) {
			n++
		}
	}
	return n
}

func promoRejection(p promos.Promo, existing []orders.Order, userID, cpf string, itemsCents int64, now time.Time) string {
	if !p.Active {
		return promoInactive
	}
	today := now.In(saoPaulo).Format("2006-01-02")
	if p.ValidFrom != "" && today < p.ValidFrom {
		return promoNotYet
	}
	if p.ValidUntil != "" && today > p.ValidUntil {
		return promoExpired
	}
	if p.MaxUses > 0 && promoUses(existing, p.Code, now) >= p.MaxUses {
		return promoExhausted
	}
	if itemsCents >= 0 && p.MinItemsCents > 0 && itemsCents < p.MinItemsCents {
		return fmt.Sprintf("Esse cupom vale para pedidos a partir de R$ %s", brlText(p.MinItemsCents))
	}
	if p.FirstOrderOnly && hasPurchase(existing, userID, cpf, now) {
		return promoNotFirst
	}
	return ""
}

func brlText(c int64) string {
	return strings.Replace(fmt.Sprintf("%.2f", brl(c)), ".", ",", 1)
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
	p, ok := s.lookupPromo(in.Code)
	if !ok {
		writeError(w, http.StatusConflict, map[string]any{"error": "coupon_invalid", "message": promoUnknown})
		return
	}
	all, err := s.Checkout.Orders.List()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if msg := promoRejection(p, all, userID, "", -1, time.Now()); msg != "" {
		writeError(w, http.StatusConflict, map[string]any{"error": "coupon_invalid", "message": msg})
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, map[string]any{
		"code":          p.Code,
		"kind":          p.Kind,
		"percentBps":    p.PercentBps,
		"valueCents":    p.ValueCents,
		"minItemsCents": p.MinItemsCents,
	})
}

type promoView struct {
	promos.Promo
	Uses        int     `json:"uses"`
	DiscountBRL float64 `json:"discountBRL"`
	RevenueBRL  float64 `json:"revenueBRL"`
}

func (s *Server) promoStore(w http.ResponseWriter, r *http.Request) (*promos.Store, bool) {
	if !s.requireAdmin(w, r) {
		return nil, false
	}
	if s.Checkout.Promos == nil {
		http.Error(w, "cupons indisponíveis", http.StatusServiceUnavailable)
		return nil, false
	}
	return s.Checkout.Promos, true
}

func (s *Server) promoOrders() ([]orders.Order, error) {
	if s.Checkout.Orders == nil {
		return nil, nil
	}
	return s.Checkout.Orders.List()
}

func (s *Server) handlePromosList(w http.ResponseWriter, r *http.Request) {
	st, ok := s.promoStore(w, r)
	if !ok {
		return
	}
	all, err := st.List()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ords, err := s.promoOrders()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	now := time.Now()
	out := make([]promoView, 0, len(all))
	for _, p := range all {
		v := promoView{Promo: p}
		for _, o := range ords {
			if o.Promo == nil || o.Promo.Code != p.Code || !countsAsPurchase(o, now) {
				continue
			}
			v.Uses++
			v.DiscountBRL = round2(v.DiscountBRL + o.Promo.ValueBRL)
			v.RevenueBRL = round2(v.RevenueBRL + o.Totals.TotalBRL)
		}
		out = append(out, v)
	}
	writeJSON(w, map[string]any{"promos": out})
}

func writePromoErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, promos.ErrNotFound):
		http.Error(w, err.Error(), http.StatusNotFound)
	case errors.Is(err, promos.ErrDuplicate):
		http.Error(w, err.Error(), http.StatusConflict)
	case errors.Is(err, promos.ErrInvalid):
		msg := err.Error()
		if i := strings.LastIndex(msg, "\n"); i >= 0 {
			msg = msg[i+1:]
		}
		http.Error(w, msg, http.StatusBadRequest)
	default:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handlePromosCreate(w http.ResponseWriter, r *http.Request) {
	st, ok := s.promoStore(w, r)
	if !ok {
		return
	}
	var in promos.Promo
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	p, err := st.Add(in)
	if err != nil {
		writePromoErr(w, err)
		return
	}
	writeJSON(w, p)
}

func (s *Server) handlePromosUpdate(w http.ResponseWriter, r *http.Request) {
	st, ok := s.promoStore(w, r)
	if !ok {
		return
	}
	var fields map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&fields); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	raw, err := json.Marshal(fields)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var probe promos.Promo
	if err := json.Unmarshal(raw, &probe); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var patchErr error
	p, err := st.Update(r.PathValue("code"), func(p *promos.Promo) {
		base, err := json.Marshal(p)
		if err != nil {
			patchErr = err
			return
		}
		var merged map[string]json.RawMessage
		if err := json.Unmarshal(base, &merged); err != nil {
			patchErr = err
			return
		}
		for k, v := range fields {
			merged[k] = v
		}
		body, err := json.Marshal(merged)
		if err != nil {
			patchErr = err
			return
		}
		var next promos.Promo
		if err := json.Unmarshal(body, &next); err != nil {
			patchErr = err
			return
		}
		*p = next
	})
	if patchErr != nil {
		http.Error(w, patchErr.Error(), http.StatusBadRequest)
		return
	}
	if err != nil {
		writePromoErr(w, err)
		return
	}
	writeJSON(w, p)
}

func (s *Server) handlePromosDelete(w http.ResponseWriter, r *http.Request) {
	st, ok := s.promoStore(w, r)
	if !ok {
		return
	}
	code := normalizePromoCode(r.PathValue("code"))
	ords, err := s.promoOrders()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for _, o := range ords {
		if o.Promo != nil && o.Promo.Code == code {
			http.Error(w, "esse cupom já foi usado em pedidos: desative em vez de excluir", http.StatusConflict)
			return
		}
	}
	if err := st.Delete(code); err != nil {
		writePromoErr(w, err)
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}
