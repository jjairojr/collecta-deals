package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"opdeals/internal/syncqueue"
	"opdeals/internal/trades"
)

func (s *Server) queueStockExit(game string, sold trades.Trade) {
	if s.SyncQueue == nil || !sold.LigaListed {
		return
	}
	channel := trades.ChannelOf(sold)
	if channel == trades.ChannelLiga {
		return
	}
	reason := "venda " + channel
	if w := trades.WriteOffOf(sold); w != "" {
		reason = "baixa sem venda: " + w
	}
	qty := sold.Qty
	if qty <= 0 {
		qty = 1
	}
	_, err := s.SyncQueue.Add(syncqueue.Task{
		Game:      game,
		TradeID:   sold.ID,
		Name:      sold.Name,
		Number:    sold.Number,
		Variant:   sold.Variant,
		Condition: sold.Condition,
		Target:    syncqueue.TargetLiga,
		Action:    syncqueue.ActionRemove,
		Qty:       qty,
		Reason:    reason,
	})
	if err != nil {
		fmt.Printf("[SYNC] falha ao enfileirar %s x%d (%s): %v\n", sold.Name, qty, sold.ID, err)
	}
}

func (s *Server) queueStockReturn(game string, sale trades.Trade) {
	if s.SyncQueue == nil || !sale.LigaListed || trades.ChannelOf(sale) == trades.ChannelLiga {
		return
	}
	cancelled, err := s.SyncQueue.CancelOpen(sale.ID, syncqueue.ActionRemove, "cancelada: venda desfeita")
	if err != nil {
		fmt.Printf("[SYNC] falha ao cancelar pendência de %s (%s): %v\n", sale.Name, sale.ID, err)
		return
	}
	if cancelled > 0 {
		return
	}
	qty := sale.Qty
	if qty <= 0 {
		qty = 1
	}
	_, err = s.SyncQueue.Add(syncqueue.Task{
		Game:      game,
		TradeID:   sale.ID,
		Name:      sale.Name,
		Number:    sale.Number,
		Variant:   sale.Variant,
		Condition: sale.Condition,
		Target:    syncqueue.TargetLiga,
		Action:    syncqueue.ActionRestore,
		Qty:       qty,
		Reason:    "venda desfeita",
	})
	if err != nil {
		fmt.Printf("[SYNC] falha ao enfileirar devolução %s x%d (%s): %v\n", sale.Name, qty, sale.ID, err)
	}
}

func (s *Server) handleSyncQueueList(w http.ResponseWriter, r *http.Request) {
	if s.SyncQueue == nil {
		http.Error(w, "sync queue unavailable", http.StatusServiceUnavailable)
		return
	}
	all, err := s.SyncQueue.List(r.URL.Query().Get("status"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string][]syncqueue.Task{"tasks": all})
}

func (s *Server) setSyncStatus(w http.ResponseWriter, r *http.Request, status, errMsg string) {
	if s.SyncQueue == nil {
		http.Error(w, "sync queue unavailable", http.StatusServiceUnavailable)
		return
	}
	task, err := s.SyncQueue.SetStatus(r.PathValue("id"), status, errMsg)
	if errors.Is(err, syncqueue.ErrNotFound) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, task)
}

func (s *Server) handleSyncQueueDone(w http.ResponseWriter, r *http.Request) {
	s.setSyncStatus(w, r, syncqueue.StatusDone, "")
}

func (s *Server) handleSyncQueueRetry(w http.ResponseWriter, r *http.Request) {
	s.setSyncStatus(w, r, syncqueue.StatusPending, "")
}

func (s *Server) handleSyncQueueError(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Error string `json:"error"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.setSyncStatus(w, r, syncqueue.StatusError, in.Error)
}
