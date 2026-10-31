package api

import (
	"encoding/json"
	"net/http"
	"slices"
	"time"

	"opdeals/internal/orders"
	"opdeals/internal/syncqueue"
	"opdeals/internal/trades"
)

func findTrade(st *trades.Store, id string) (trades.Trade, bool, error) {
	all, err := st.List()
	if err != nil {
		return trades.Trade{}, false, err
	}
	for _, t := range all {
		if t.ID == id {
			return t, true, nil
		}
	}
	return trades.Trade{}, false, nil
}

func (s *Server) moveBlocker(from string, t trades.Trade) (string, error) {
	if t.Status != "sold" && t.Listed {
		return "o item está na vitrine: tire da vitrine no Estoque antes de trocar o jogo", nil
	}
	if t.Status != "sold" && t.LigaListed {
		return "o item está marcado Na Liga: tire da Liga e desmarque no Estoque antes de trocar o jogo", nil
	}
	if s.Checkout.Orders != nil {
		all, err := s.Checkout.Orders.List()
		if err != nil {
			return "", err
		}
		sku := trades.SKU(from, t)
		now := time.Now()
		for _, o := range all {
			status := orders.EffectiveStatus(o, now)
			open := status == orders.StatusPending || status == orders.StatusPaid || status == orders.StatusOversold
			for _, l := range o.Lines {
				if slices.Contains(l.TradeIDs, t.ID) {
					return "o item saiu num pedido do site: o pedido guarda o jogo dele, então não dá para trocar o jogo", nil
				}
				if open && l.Game == from && l.SKU == sku {
					return "o item está num pedido do site em aberto: conclua ou cancele o pedido em Pedidos antes de trocar o jogo", nil
				}
			}
		}
	}
	if s.SyncQueue != nil {
		all, err := s.SyncQueue.List("")
		if err != nil {
			return "", err
		}
		for _, task := range all {
			if task.TradeID == t.ID && (task.Status == syncqueue.StatusPending || task.Status == syncqueue.StatusError) {
				return "o item tem pendência na fila de sincronização: resolva em Sincronizar antes de trocar o jogo", nil
			}
		}
	}
	return "", nil
}

func (s *Server) handleTradesMove(w http.ResponseWriter, r *http.Request) {
	var in struct {
		From string `json:"from"`
		To   string `json:"to"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	src, okFrom := s.games[in.From]
	dst, okTo := s.games[in.To]
	if !okFrom || !okTo || src.Trades == nil || dst.Trades == nil {
		http.Error(w, "jogo desconhecido", http.StatusBadRequest)
		return
	}
	if in.From == in.To {
		http.Error(w, "o item já está nesse jogo", http.StatusBadRequest)
		return
	}
	id := r.PathValue("id")
	t, found, err := findTrade(src.Trades, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !found {
		if s.accessories != nil {
			if _, isAccessory, err := findTrade(s.accessories, id); err == nil && isAccessory {
				http.Error(w, "acessório vale para todos os jogos e não troca de jogo", http.StatusConflict)
				return
			}
		}
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if t.Kind == trades.KindAccessory {
		http.Error(w, "acessório vale para todos os jogos e não troca de jogo", http.StatusConflict)
		return
	}
	reason, err := s.moveBlocker(in.From, t)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if reason != "" {
		http.Error(w, reason, http.StatusConflict)
		return
	}
	taken, err := src.Trades.Take(func(x trades.Trade) bool { return x.ID == id })
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(taken) == 0 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err := dst.Trades.Append(taken); err != nil {
		if back := src.Trades.Append(taken); back != nil {
			http.Error(w, "falha ao mover e ao devolver o item: "+back.Error(), http.StatusInternalServerError)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, taken[0])
}
