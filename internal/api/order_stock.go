package api

import (
	"fmt"
	"strings"
	"time"

	"opdeals/internal/orders"
	"opdeals/internal/packs"
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

type packOpening struct {
	game   string
	parent string
	units  int
	need   int
}

func (s *Server) planStock(o orders.Order) ([][]orders.Pick, string, error) {
	plan, openings, short, err := s.planStockOnce(o)
	if err != nil || short != "" || len(openings) == 0 {
		return plan, short, err
	}
	for _, op := range openings {
		if failed := s.openForOrder(op); failed != "" {
			return nil, failed, nil
		}
	}
	plan, openings, short, err = s.planStockOnce(o)
	if err == nil && short == "" && len(openings) > 0 {
		short = "estoque mudou durante a abertura da embalagem"
	}
	return plan, short, err
}

func (s *Server) openForOrder(op packOpening) string {
	st := s.ledgerForGame(op.game)
	all, err := st.List()
	if err != nil {
		return "abertura falhou: " + err.Error()
	}
	parent, ok := holdingByID(all, op.parent)
	if !ok {
		return "embalagem " + op.parent + " sumiu do estoque"
	}
	links, err := s.packLinks()
	if err != nil {
		return "abertura falhou: " + err.Error()
	}
	link, ok := packs.ByParent(links, op.game, trades.SKU(op.game, parent))
	if !ok {
		return "embalagem " + parent.Name + " sem vínculo"
	}
	template, ok := packs.Template(op.game, all, link.ChildSKU)
	if !ok {
		return "conteúdo de " + parent.Name + " sem registro"
	}
	qty := (op.need + link.Units - 1) / link.Units
	if _, err := s.openPack(op.game, st, parent, qty, link, template); err != nil {
		return fmt.Sprintf("abertura de %s falhou: %v", parent.Name, err)
	}
	return ""
}

func (s *Server) planStockOnce(o orders.Order) ([][]orders.Pick, []packOpening, string, error) {
	plan := make([][]orders.Pick, len(o.Lines))
	taken := make(map[string]int)
	openings := make([]packOpening, 0)
	at := make(map[string]int)
	for i, l := range o.Lines {
		st := s.ledgerForGame(l.Game)
		if st == nil {
			return nil, nil, "ledger desconhecido para " + l.Game, nil
		}
		all, err := st.List()
		if err != nil {
			return nil, nil, "", err
		}
		covered, lost := soldFor(all, l.TradeIDs)
		if lost != "" {
			return nil, nil, fmt.Sprintf("venda %s de %s sumiu do estoque: confira antes de mudar o status", lost, l.Name), nil
		}
		missing := l.StockUnits() - covered
		if missing <= 0 {
			continue
		}
		stock, origins, err := s.expandStock(l.Game, all)
		if err != nil {
			return nil, nil, "", err
		}
		picks, ok := orders.Allocate(withoutTaken(bySKU(stock, l.Game, l.StockKey()), taken), missing)
		if !ok {
			return nil, nil, fmt.Sprintf("sem estoque para %s x%d", l.Name, missing), nil
		}
		for _, p := range picks {
			taken[p.TradeID] += p.Qty
			origin, virtual := origins[p.TradeID]
			if !virtual {
				continue
			}
			n, seen := at[origin.ParentID]
			if !seen {
				n = len(openings)
				at[origin.ParentID] = n
				openings = append(openings, packOpening{game: l.Game, parent: origin.ParentID, units: origin.Units})
			}
			openings[n].need += p.Qty
		}
		plan[i] = picks
	}
	return plan, openings, "", nil
}

func withoutTaken(ts []trades.Trade, taken map[string]int) []trades.Trade {
	out := make([]trades.Trade, 0, len(ts))
	for _, t := range ts {
		used := taken[t.ID]
		if used == 0 {
			out = append(out, t)
			continue
		}
		if t.Status != "holding" || t.Qty <= used {
			continue
		}
		t.Qty -= used
		out = append(out, t)
	}
	return out
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
				SellPrice:    round2(float64(p.Qty) * l.UnitBRL / float64(l.UnitsPerItem())),
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
			s.queueLinkedExit(l.Game, sold)
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
			s.queueLinkedReturn(l.Game, res.Sale)
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
