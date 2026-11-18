package api

import (
	"encoding/json"
	"errors"
	"math"
	"net/http"

	"opdeals/internal/kits"
	"opdeals/internal/trades"
)

type kitView struct {
	kits.Kit
	SKU      string   `json:"sku"`
	TradeIDs []string `json:"tradeIDs"`
}

type gameLedger struct {
	game  string
	store *trades.Store
}

func (s *Server) kitLedgers(gs *GameStack) []gameLedger {
	out := make([]gameLedger, 0, 2)
	if gs.Trades != nil {
		out = append(out, gameLedger{game: gs.Game.ID, store: gs.Trades})
	}
	if s.accessories != nil {
		out = append(out, gameLedger{game: accessoryGameID, store: s.accessories})
	}
	return out
}

func (s *Server) kitStore(w http.ResponseWriter) (*kits.Store, bool) {
	if s.Kits == nil {
		http.Error(w, "kits indisponíveis", http.StatusServiceUnavailable)
		return nil, false
	}
	return s.Kits, true
}

func writeKitErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, kits.ErrNotFound):
		http.Error(w, err.Error(), http.StatusNotFound)
	case errors.Is(err, kits.ErrDuplicate):
		http.Error(w, err.Error(), http.StatusConflict)
	case errors.Is(err, kits.ErrInvalid):
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleKitsList(w http.ResponseWriter, r *http.Request) {
	st, ok := s.kitStore(w)
	if !ok {
		return
	}
	all, err := st.List()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	holders := make(map[string][]string)
	for _, l := range s.kitLedgers(s.stackFor(r.URL.Query())) {
		rows, err := l.store.List()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		for _, t := range rows {
			if t.Status != "holding" {
				continue
			}
			sku := trades.SKU(l.game, t)
			holders[sku] = append(holders[sku], t.ID)
		}
	}
	out := make([]kitView, 0, len(all))
	for _, k := range all {
		ids, ok := holders[k.BaseSKU]
		if !ok {
			continue
		}
		out = append(out, kitView{Kit: k, SKU: k.SKU(), TradeIDs: ids})
	}
	writeJSON(w, map[string]any{"kits": out})
}

func (s *Server) handleKitsCreate(w http.ResponseWriter, r *http.Request) {
	st, ok := s.kitStore(w)
	if !ok {
		return
	}
	var in struct {
		TradeID string  `json:"tradeID"`
		Units   int     `json:"units"`
		AskBRL  float64 `json:"askBRL"`
		CardBRL float64 `json:"cardBRL"`
		Listed  bool    `json:"listed"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	for _, l := range s.kitLedgers(s.stackFor(r.URL.Query())) {
		rows, err := l.store.List()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		for _, t := range rows {
			if t.ID != in.TradeID || t.Status != "holding" {
				continue
			}
			k, err := st.Add(kits.Kit{
				BaseSKU: trades.SKU(l.game, t),
				Units:   in.Units,
				AskBRL:  round2(in.AskBRL),
				CardBRL: round2(math.Max(in.CardBRL, 0)),
				Listed:  in.Listed,
			})
			if err != nil {
				writeKitErr(w, err)
				return
			}
			writeJSON(w, kitView{Kit: k, SKU: k.SKU(), TradeIDs: []string{t.ID}})
			return
		}
	}
	http.Error(w, "produto base não encontrado no estoque", http.StatusNotFound)
}

func (s *Server) handleKitsUpdate(w http.ResponseWriter, r *http.Request) {
	st, ok := s.kitStore(w)
	if !ok {
		return
	}
	var in struct {
		AskBRL  *float64 `json:"askBRL"`
		CardBRL *float64 `json:"cardBRL"`
		Listed  *bool    `json:"listed"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	k, err := st.Update(r.PathValue("id"), func(k *kits.Kit) {
		if in.AskBRL != nil {
			k.AskBRL = round2(*in.AskBRL)
		}
		if in.CardBRL != nil {
			k.CardBRL = round2(math.Max(*in.CardBRL, 0))
		}
		if in.Listed != nil {
			k.Listed = *in.Listed
		}
	})
	if err != nil {
		writeKitErr(w, err)
		return
	}
	writeJSON(w, kitView{Kit: k, SKU: k.SKU(), TradeIDs: []string{}})
}

func (s *Server) handleKitsDelete(w http.ResponseWriter, r *http.Request) {
	st, ok := s.kitStore(w)
	if !ok {
		return
	}
	if err := st.Delete(r.PathValue("id")); err != nil {
		writeKitErr(w, err)
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}
