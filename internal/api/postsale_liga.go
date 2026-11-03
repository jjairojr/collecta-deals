package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"opdeals/internal/conversations"
	"opdeals/internal/postsale"
	"opdeals/internal/whatsapp"
)

const (
	ligaBodyLimit  = 256 << 10
	ligaMaxItems   = 50
	ligaOrderIDPfx = "liga:"
)

type ligaOrderIn struct {
	Code      string               `json:"code"`
	FirstName string               `json:"firstName"`
	Phone     string               `json:"phone"`
	PaidAt    time.Time            `json:"paidAt"`
	Pickup    bool                 `json:"pickup"`
	Shipped   bool                 `json:"shipped"`
	Items     []conversations.Item `json:"items"`
}

type ligaImportResult struct {
	Created int `json:"created"`
	Skipped int `json:"skipped"`
}

func (s *Server) adminOnly(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.requireAdmin(w, r) {
			h(w, r)
		}
	}
}

func ligaCode(raw string) (string, int, bool) {
	code := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(raw), "#"))
	n, err := strconv.Atoi(code)
	if err != nil || n <= 0 {
		return "", 0, false
	}
	return code, n, true
}

func ligaItems(in []conversations.Item) []conversations.Item {
	out := make([]conversations.Item, 0, len(in))
	for _, it := range in {
		name := strings.Join(strings.Fields(it.Name), " ")
		if name == "" {
			continue
		}
		out = append(out, conversations.Item{Name: name, Qty: max(it.Qty, 1)})
		if len(out) == ligaMaxItems {
			break
		}
	}
	return out
}

func (s *Server) handleLigaOrders(w http.ResponseWriter, r *http.Request) {
	if !s.PostSale.Outbox {
		http.Error(w, "pós-venda da Liga só funciona com o OpenBot (POSTSALE_TRANSPORT=openbot)", http.StatusConflict)
		return
	}
	var in struct {
		Orders []ligaOrderIn `json:"orders"`
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, ligaBodyLimit))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		http.Error(w, "corpo inválido: "+err.Error(), http.StatusBadRequest)
		return
	}
	var out ligaImportResult
	for _, o := range in.Orders {
		if s.importLigaOrder(o) {
			out.Created++
		} else {
			out.Skipped++
		}
	}
	if out.Created > 0 {
		s.logPostSale("liga: %d pedido(s) importado(s), %d ignorado(s)", out.Created, out.Skipped)
	}
	writeJSON(w, out)
}

func (s *Server) importLigaOrder(o ligaOrderIn) bool {
	code, number, ok := ligaCode(o.Code)
	items := ligaItems(o.Items)
	first := postsale.FirstName(o.FirstName)
	if !ok || first == "" || len(items) == 0 {
		return false
	}
	orderID := ligaOrderIDPfx + code
	_, created, err := s.PostSale.Conversations.Create(func(existing []conversations.Conversation) (conversations.Conversation, bool) {
		if !postsale.EligibleLigaOrder(orderID, o.Phone, o.PaidAt, existing, s.psNow()) {
			return conversations.Conversation{}, false
		}
		return conversations.Conversation{
			Channel:     conversations.ChannelLiga,
			Phone:       whatsapp.BrazilE164(o.Phone),
			Name:        first,
			OrderID:     orderID,
			OrderNumber: number,
			Pickup:      o.Pickup,
			Shipped:     o.Shipped,
			Items:       items,
			State:       conversations.StateCompose,
			Messages:    []conversations.Message{},
		}, true
	})
	if err != nil {
		s.logPostSale("liga #%s: %v", code, err)
		return false
	}
	return created
}
