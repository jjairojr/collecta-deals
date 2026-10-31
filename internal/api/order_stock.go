package api

import (
	"fmt"
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
		picks, ok := orders.Allocate(bySKU(all, l.Game, l.SKU), l.Qty)
		if !ok {
			return nil, fmt.Sprintf("sem estoque para %s x%d", l.Name, l.Qty), nil
		}
		plan[i] = picks
	}
	return plan, "", nil
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
