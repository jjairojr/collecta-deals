package api

import (
	"fmt"
	"strings"
	"time"

	"opdeals/internal/orders"
	"opdeals/internal/trades"
)

func needsStock(from, to orders.Status) bool {
	if from != orders.StatusOversold && from != orders.StatusRefunded {
		return false
	}
	return to == orders.StatusPaid || to == orders.StatusShipped || to == orders.StatusDelivered
}

func releasesStock(to orders.Status) bool {
	return to == orders.StatusRefunded || to == orders.StatusCancelled
}

const keptAfterShipping = "estoque não devolvido: pedido já enviado, use Devolver ao estoque quando o produto voltar"

func (s *Server) refundStock(o *orders.Order, from orders.Status, now time.Time) {
	if from != orders.StatusShipped && from != orders.StatusDelivered {
		s.returnStock(o, now)
		return
	}
	if o.HoldsStock() && !strings.Contains(o.Notes, keptAfterShipping) {
		o.Notes = appendNote(o.Notes, keptAfterShipping)
	}
}

func (s *Server) planStock(o orders.Order) ([][]orders.Pick, string, error) {
	plan := make([][]orders.Pick, len(o.Lines))
	for i, l := range o.Lines {
		st := s.ledgerForGame(l.Game)
		if st == nil {
			return nil, "ledger desconhecido para " + l.Game, nil
		}
		all, err := st.List()
		if err != nil {
			return nil, "", err
		}
		covered, lost := soldFor(all, l.TradeIDs)
		if lost != "" {
			return nil, fmt.Sprintf("venda %s de %s sumiu do estoque: confira antes de mudar o status", lost, l.Name), nil
		}
		missing := l.Qty - covered
		if missing <= 0 {
			continue
		}
		picks, ok := orders.Allocate(bySKU(all, l.Game, l.SKU), missing)
		if !ok {
			return nil, fmt.Sprintf("sem estoque para %s x%d", l.Name, missing), nil
		}
		plan[i] = picks
	}
	return plan, "", nil
}

func soldFor(all []trades.Trade, ids []string) (int, string) {
	qty := make(map[string]int, len(all))
	for _, t := range all {
		qty[t.ID] = t.Qty
	}
	covered := 0
	for _, id := range ids {
		q, ok := qty[id]
		if !ok {
			return 0, id
		}
		covered += q
	}
	return covered, ""
}

func (s *Server) sellStock(o *orders.Order, plan [][]orders.Pick, now time.Time) string {
	buyer := fmt.Sprintf("%s — Site #%d", o.Buyer.Name, o.Number)
	date := now.In(saoPaulo).Format("2006-01-02")
	for i, picks := range plan {
		l := &o.Lines[i]
		st := s.ledgerForGame(l.Game)
		for _, p := range picks {
			sold, err := st.Sell(p.TradeID, trades.Sale{
				Qty:          p.Qty,
				SellPrice:    round2(float64(p.Qty) * l.UnitBRL),
				SellCurrency: "BRL",
				SellDate:     date,
				Buyer:        buyer,
				Channel:      trades.ChannelSite,
			})
			if err != nil {
				return fmt.Sprintf("baixa falhou em %s: %v", p.TradeID, err)
			}
			l.TradeIDs = append(l.TradeIDs, sold.ID)
			s.queueStockExit(l.Game, sold)
		}
	}
	return ""
}

func (s *Server) returnStock(o *orders.Order, now time.Time) {
	if !o.HoldsStock() {
		return
	}
	touched := make(map[string]bool)
	for i := range o.Lines {
		l := &o.Lines[i]
		st := s.ledgerForGame(l.Game)
		kept := make([]string, 0)
		for _, id := range l.TradeIDs {
			if st == nil {
				o.Notes = appendNote(o.Notes, fmt.Sprintf("estoque não devolvido em %s: ledger desconhecido para %s", id, l.Game))
				kept = append(kept, id)
				continue
			}
			res, err := st.UnsellSiteSale(id)
			if err != nil {
				o.Notes = appendNote(o.Notes, fmt.Sprintf("estoque não devolvido em %s: %v", id, err))
				kept = append(kept, id)
				continue
			}
			s.queueStockReturn(l.Game, res.Sale)
			touched[res.Restored.PurchaseID] = true
			l.ReturnedTradeIDs = append(l.ReturnedTradeIDs, id)
		}
		l.TradeIDs = kept
	}
	o.StockReturnedAt = now
	for purchaseID := range touched {
		if err := s.reallocateOrderShipping(purchaseID); err != nil {
			o.Notes = appendNote(o.Notes, "rateio do frete falhou: "+err.Error())
		}
	}
}
