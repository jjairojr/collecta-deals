package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"opdeals/internal/packs"
	"opdeals/internal/syncqueue"
	"opdeals/internal/trades"
)

type packView struct {
	packs.Link
	ParentName string   `json:"parentName"`
	ChildName  string   `json:"childName"`
	ParentIDs  []string `json:"parentIDs"`
	ChildIDs   []string `json:"childIDs"`
	ParentQty  int      `json:"parentQty"`
	ChildQty   int      `json:"childQty"`
}

func (s *Server) packLinks() ([]packs.Link, error) {
	if s.Packs == nil {
		return nil, nil
	}
	return s.Packs.List()
}

func (s *Server) expandStock(game string, all []trades.Trade) ([]trades.Trade, map[string]packs.Origin, error) {
	links, err := s.packLinks()
	if err != nil {
		return nil, nil, err
	}
	stock, origins := packs.Expand(game, all, links)
	return stock, origins, nil
}

func (s *Server) packClaims(reserved map[string]int) (map[string]int, error) {
	links, err := s.packLinks()
	if err != nil || len(links) == 0 {
		return reserved, err
	}
	out := reserved
	for _, id := range s.orderedGameIDs() {
		gs := s.games[id]
		if gs.Trades == nil {
			continue
		}
		all, err := gs.Trades.List()
		if err != nil {
			return nil, err
		}
		out = packs.Claims(gs.Game.ID, all, links, out)
	}
	return out, nil
}

func writePackErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, packs.ErrNotFound), errors.Is(err, trades.ErrNotFound):
		http.Error(w, err.Error(), http.StatusNotFound)
	case errors.Is(err, packs.ErrDuplicate), errors.Is(err, trades.ErrAlreadySold):
		http.Error(w, err.Error(), http.StatusConflict)
	case errors.Is(err, packs.ErrInvalid), errors.Is(err, trades.ErrInvalidOpen):
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) packContext(w http.ResponseWriter, r *http.Request) (*GameStack, []trades.Trade, bool) {
	if s.Packs == nil {
		http.Error(w, "vínculos indisponíveis", http.StatusServiceUnavailable)
		return nil, nil, false
	}
	gs := s.stackFor(r.URL.Query())
	if gs.Trades == nil {
		http.Error(w, "trades store unavailable", http.StatusServiceUnavailable)
		return nil, nil, false
	}
	all, err := gs.Trades.List()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil, nil, false
	}
	return gs, all, true
}

func viewOf(l packs.Link, game string, all []trades.Trade) packView {
	v := packView{Link: l, ParentIDs: []string{}, ChildIDs: []string{}}
	if t, ok := packs.Template(game, all, l.ParentSKU); ok {
		v.ParentName = t.Name
	}
	if t, ok := packs.Template(game, all, l.ChildSKU); ok {
		v.ChildName = t.Name
	}
	for _, t := range all {
		if t.Status != "holding" {
			continue
		}
		qty := t.Qty
		if qty <= 0 {
			qty = 1
		}
		switch trades.SKU(game, t) {
		case l.ParentSKU:
			v.ParentIDs = append(v.ParentIDs, t.ID)
			v.ParentQty += qty
		case l.ChildSKU:
			v.ChildIDs = append(v.ChildIDs, t.ID)
			v.ChildQty += qty
		}
	}
	return v
}

func (s *Server) handlePacksList(w http.ResponseWriter, r *http.Request) {
	gs, all, ok := s.packContext(w, r)
	if !ok {
		return
	}
	links, err := s.Packs.List()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	scoped := packs.ForGame(links, gs.Game.ID)
	out := make([]packView, 0, len(scoped))
	for _, l := range scoped {
		out = append(out, viewOf(l, gs.Game.ID, all))
	}
	writeJSON(w, map[string][]packView{"packs": out})
}

func holdingByID(all []trades.Trade, id string) (trades.Trade, bool) {
	for _, t := range all {
		if t.ID == id && t.Status == "holding" {
			return t, true
		}
	}
	return trades.Trade{}, false
}

func (s *Server) handlePacksCreate(w http.ResponseWriter, r *http.Request) {
	gs, all, ok := s.packContext(w, r)
	if !ok {
		return
	}
	var in struct {
		ParentTradeID string `json:"parentTradeID"`
		ChildTradeID  string `json:"childTradeID"`
		Units         int    `json:"units"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	parent, okParent := holdingByID(all, in.ParentTradeID)
	child, okChild := holdingByID(all, in.ChildTradeID)
	if !okParent || !okChild {
		http.Error(w, "embalagem ou conteúdo não encontrado no estoque", http.StatusNotFound)
		return
	}
	link, err := s.Packs.Add(packs.Link{
		Game:      gs.Game.ID,
		ParentSKU: trades.SKU(gs.Game.ID, parent),
		ChildSKU:  trades.SKU(gs.Game.ID, child),
		Units:     in.Units,
	})
	if err != nil {
		writePackErr(w, err)
		return
	}
	writeJSON(w, viewOf(link, gs.Game.ID, all))
}

func (s *Server) handlePacksDelete(w http.ResponseWriter, r *http.Request) {
	if s.Packs == nil {
		http.Error(w, "vínculos indisponíveis", http.StatusServiceUnavailable)
		return
	}
	if err := s.Packs.Delete(r.PathValue("id")); err != nil {
		writePackErr(w, err)
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

func contentTemplate(parent trades.Trade, name string) trades.Trade {
	t := parent
	t.Name = name
	t.Slug = ""
	t.GTIN = ""
	t.AskBRL = 0
	t.CardBRL = 0
	t.Listed = false
	t.Featured = false
	t.ImageURL = ""
	t.Photos = nil
	t.ShipsFrom = ""
	t.RefUSD = 0
	t.ManualBRL = 0
	t.LigaListed = false
	t.LigaQty = 0
	t.LigaPriceBRL = 0
	t.LigaAt = time.Time{}
	return t
}

func (s *Server) openPack(game string, st *trades.Store, parent trades.Trade, qty int, link packs.Link, template trades.Trade) (trades.Trade, error) {
	child, err := st.Open(parent.ID, qty, link.Units, template)
	if err != nil {
		return trades.Trade{}, err
	}
	if err := s.reallocateOrderShipping(parent.PurchaseID); err != nil {
		return trades.Trade{}, err
	}
	s.queueLigaTask(game, parent, parent.ID, -qty, "embalagem aberta")
	s.queueLinkedLiga(game, st, child, qty*link.Units, "embalagem aberta: "+parent.Name)
	return child, nil
}

func (s *Server) handleTradesOpen(w http.ResponseWriter, r *http.Request) {
	gs, all, ok := s.packContext(w, r)
	if !ok {
		return
	}
	var in struct {
		Qty       int    `json:"qty"`
		Units     int    `json:"units"`
		ChildName string `json:"childName"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if in.Qty <= 0 {
		in.Qty = 1
	}
	game := gs.Game.ID
	parent, found := holdingByID(all, r.PathValue("id"))
	if !found {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	links, err := s.Packs.List()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	link, linked := packs.ByParent(links, game, trades.SKU(game, parent))
	var template trades.Trade
	created := false
	if linked {
		template, found = packs.Template(game, all, link.ChildSKU)
		if !found {
			http.Error(w, "o conteúdo vinculado não tem registro: refaça o vínculo", http.StatusConflict)
			return
		}
	} else {
		name := strings.TrimSpace(in.ChildName)
		if name == "" {
			http.Error(w, "embalagem sem vínculo: informe childName e units", http.StatusBadRequest)
			return
		}
		template = contentTemplate(parent, name)
		link, err = s.Packs.Add(packs.Link{
			Game:      game,
			ParentSKU: trades.SKU(game, parent),
			ChildSKU:  trades.SKU(game, template),
			Units:     in.Units,
		})
		if err != nil {
			writePackErr(w, err)
			return
		}
		created = true
	}
	child, err := s.openPack(game, gs.Trades, parent, in.Qty, link, template)
	if err != nil {
		if created {
			if delErr := s.Packs.Delete(link.ID); delErr != nil {
				fmt.Printf("[PACKS] vínculo %s ficou sem conteúdo: %v\n", link.ID, delErr)
			}
		}
		writePackErr(w, err)
		return
	}
	writeJSON(w, map[string]any{"child": child, "link": link})
}

func (s *Server) queueLigaTask(game string, t trades.Trade, tradeID string, delta int, reason string) {
	if s.SyncQueue == nil || !t.LigaListed || delta == 0 {
		return
	}
	action := syncqueue.ActionRestore
	qty := delta
	if delta < 0 {
		action = syncqueue.ActionRemove
		qty = -delta
	}
	_, err := s.SyncQueue.Add(syncqueue.Task{
		Game:      game,
		TradeID:   tradeID,
		Name:      t.Name,
		Number:    t.Number,
		Variant:   t.Variant,
		Condition: t.Condition,
		Target:    syncqueue.TargetLiga,
		Action:    action,
		Qty:       qty,
		Reason:    reason,
	})
	if err != nil {
		fmt.Printf("[SYNC] falha ao enfileirar %s %s x%d (%s): %v\n", action, t.Name, qty, tradeID, err)
	}
}

func linkedTaskID(tradeID string) string {
	return packs.VirtualPrefix + tradeID
}

func (s *Server) queueLinkedLiga(game string, st *trades.Store, source trades.Trade, delta int, reason string) {
	if s.SyncQueue == nil || s.Packs == nil || st == nil || delta == 0 {
		return
	}
	links, err := s.Packs.List()
	if err != nil {
		fmt.Printf("[SYNC] vínculos ilegíveis para %s: %v\n", source.Name, err)
		return
	}
	link, ok := packs.ByParent(links, game, trades.SKU(game, source))
	if !ok {
		return
	}
	all, err := st.List()
	if err != nil {
		fmt.Printf("[SYNC] estoque ilegível para %s: %v\n", source.Name, err)
		return
	}
	for _, t := range all {
		if t.LigaListed && trades.SKU(game, t) == link.ChildSKU {
			s.queueLigaTask(game, t, linkedTaskID(source.ID), delta*link.Units, reason)
			return
		}
	}
}

func (s *Server) queueLinkedExit(game string, sold trades.Trade) {
	qty := sold.Qty
	if qty <= 0 {
		qty = 1
	}
	s.queueLinkedLiga(game, s.ledgerForGame(game), sold, -qty, "saiu "+sold.Name)
}

func (s *Server) queueLinkedReturn(game string, sale trades.Trade) {
	if s.SyncQueue == nil {
		return
	}
	cancelled, err := s.SyncQueue.CancelOpen(linkedTaskID(sale.ID), syncqueue.ActionRemove, "cancelada: venda desfeita")
	if err != nil {
		fmt.Printf("[SYNC] falha ao cancelar pendência vinculada de %s (%s): %v\n", sale.Name, sale.ID, err)
		return
	}
	if cancelled > 0 {
		return
	}
	qty := sale.Qty
	if qty <= 0 {
		qty = 1
	}
	s.queueLinkedLiga(game, s.ledgerForGame(game), sale, qty, "venda desfeita: "+sale.Name)
}
