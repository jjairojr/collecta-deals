package api

import (
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"opdeals/internal/metaevents"
	"opdeals/internal/orders"
	"opdeals/internal/trades"
)

// Compras para o Meta (Conversions API e lista de clientes). O painel monta os
// eventos com o dado do cliente já com hash, então o e-mail e o telefone nunca
// saem daqui em texto: quem envia ao Meta (o publicar-mcp no Mac) só vê hashes.

const (
	metaSiteURL        = "https://www.collectatcg.com.br"
	metaEventsMaxDays  = 7   // o Meta recusa evento com mais de 7 dias
	metaCustomersDays  = 180 // histórico da lista de clientes
	metaCustomersLimit = 365
)

var (
	ligaBuyerCode = regexp.MustCompile(`(?i)\bliga\s*#?\s*(\d+)`)
	buyerEmail    = regexp.MustCompile(`[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}`)
	buyerPhone    = regexp.MustCompile(`\+?\d[\d\s().-]{8,}\d`)
)

func (s *Server) registerMetaRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/meta/events", s.adminOnly(s.handleMetaEvents))
	mux.HandleFunc("GET /api/meta/customers", s.adminOnly(s.handleMetaCustomers))
}

// metaSale é uma venda pronta para virar evento ou linha da lista de clientes.
type metaSale struct {
	id     string // event_id, estável: o Meta descarta o repetido
	source string
	at     time.Time
	value  float64
	items  int
	person metaevents.Person
	order  string
	url    string
	ads    *orders.AdsContext
}

type metaSkipped struct {
	SiteSemNavegador int `json:"site_sem_navegador"`
	LigaSemTelefone  int `json:"liga_sem_telefone"`
	BalcaoSemContato int `json:"balcao_sem_contato"`
}

func paidSale(o orders.Order) bool {
	switch o.Status {
	case orders.StatusPaid, orders.StatusShipped, orders.StatusDelivered:
		return !o.PaidAt.IsZero()
	}
	return false
}

// metaSales junta as vendas do site, da Liga e do balcão pagas desde since.
func (s *Server) metaSales(since, now time.Time) ([]metaSale, metaSkipped, error) {
	var out []metaSale
	var skipped metaSkipped
	if s.Checkout.Orders != nil {
		all, err := s.Checkout.Orders.List()
		if err != nil {
			return nil, skipped, err
		}
		for _, o := range all {
			if !paidSale(o) || o.PaidAt.Before(since) {
				continue
			}
			items := 0
			for _, l := range o.Lines {
				items += l.Qty
			}
			out = append(out, metaSale{
				id: "pedido-" + o.ID, source: metaevents.SourceWebsite, at: o.PaidAt, value: o.Totals.TotalBRL, items: items,
				person: metaevents.Person{Name: o.Buyer.Name, Email: o.Buyer.Email, Phone: o.Buyer.Phone, City: o.Address.City, State: o.Address.State, Zip: o.Address.CEP},
				order:  "site-" + strconv.Itoa(o.Number), url: metaSiteURL + "/pedido/" + o.ID, ads: o.Ads,
			})
		}
	}

	// Liga e balcão vêm do registro de vendas; o telefone da Liga vem da conversa
	// de pós-venda do pedido (liga:<número>).
	phones := map[string]metaevents.Person{}
	if s.PostSale.Conversations != nil {
		convs, err := s.PostSale.Conversations.List()
		if err != nil {
			return nil, skipped, err
		}
		for _, c := range convs {
			if strings.HasPrefix(c.OrderID, ligaOrderIDPfx) && c.Phone != "" {
				phones[strings.TrimPrefix(c.OrderID, ligaOrderIDPfx)] = metaevents.Person{Name: c.Name, Phone: c.Phone}
			}
		}
	}
	type group struct {
		sale  metaSale
		day   string
		buyer string
	}
	groups := map[string]*group{}
	var keys []string
	err := s.forEachLedger(90, func(_, _ string, p trades.PortfolioResponse) {
		for _, v := range p.Trades {
			if !v.Realized || v.WriteOff != "" || v.ValueBRL <= 0 {
				continue
			}
			day, ok := saleDay(v.SellDate, now)
			if !ok {
				continue
			}
			var key string
			sale := metaSale{at: day, value: v.ValueBRL, items: max(v.Qty, 1)}
			switch v.Channel {
			case trades.ChannelLiga:
				m := ligaBuyerCode.FindStringSubmatch(v.Buyer)
				if m == nil {
					continue
				}
				key = "liga-" + m[1]
				sale.id, sale.source, sale.order = key, metaevents.SourceOther, key
			case trades.ChannelBalcao:
				buyer := strings.ToLower(strings.Join(strings.Fields(v.Buyer), " "))
				key = "balcao-" + v.SellDate + "-" + metaevents.Hash(buyer)[:10]
				sale.id, sale.source = key, metaevents.SourceStore
				sale.person = metaevents.Person{Email: buyerEmail.FindString(v.Buyer), Phone: buyerPhone.FindString(v.Buyer)}
				if sale.person.Email != "" || sale.person.Phone != "" {
					name := buyerEmail.ReplaceAllString(buyerPhone.ReplaceAllString(v.Buyer, " "), " ")
					sale.person.Name = strings.Trim(name, " -–—,;:/|")
				}
			default:
				continue
			}
			if g, ok := groups[key]; ok {
				g.sale.value += sale.value
				g.sale.items += sale.items
				if sale.at.After(g.sale.at) {
					g.sale.at = sale.at
				}
				continue
			}
			groups[key] = &group{sale: sale}
			keys = append(keys, key)
		}
	})
	if err != nil {
		return nil, skipped, err
	}
	sort.Strings(keys)
	for _, k := range keys {
		sale := groups[k].sale
		if sale.at.Before(since) {
			continue
		}
		if sale.source == metaevents.SourceOther {
			p, ok := phones[strings.TrimPrefix(k, "liga-")]
			if !ok {
				skipped.LigaSemTelefone++
				continue
			}
			sale.person = p
		}
		if sale.source == metaevents.SourceStore && sale.person.Email == "" && sale.person.Phone == "" {
			skipped.BalcaoSemContato++
			continue
		}
		out = append(out, sale)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].at.Before(out[j].at) })
	return out, skipped, nil
}

// saleDay lê a data da venda (AAAA-MM-DD) como meio-dia em São Paulo; uma venda
// de hoje que ainda não chegou ao meio-dia fica com a hora de agora.
func saleDay(date string, now time.Time) (time.Time, bool) {
	if len(date) < 10 {
		return time.Time{}, false
	}
	day, err := time.ParseInLocation("2006-01-02", date[:10], saoPaulo)
	if err != nil {
		return time.Time{}, false
	}
	at := day.Add(12 * time.Hour)
	if at.After(now) {
		at = now
	}
	return at, true
}

type metaEventsResponse struct {
	Events  []metaevents.Event `json:"events"`
	Skipped metaSkipped        `json:"skipped"`
}

// handleMetaEvents devolve as compras dos últimos dias (no máximo 7) como
// eventos Purchase da Conversions API, com o dado do cliente já com hash.
func (s *Server) handleMetaEvents(w http.ResponseWriter, r *http.Request) {
	days := min(max(intParam(r.URL.Query(), "days", metaEventsMaxDays), 1), metaEventsMaxDays)
	now := time.Now()
	// Uma hora de folga, para o evento não passar de 7 dias no caminho até o Meta.
	since := now.Add(-time.Duration(days)*24*time.Hour + time.Hour)
	sales, skipped, err := s.metaSales(since, now)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	resp := metaEventsResponse{Events: []metaevents.Event{}, Skipped: skipped}
	for _, sale := range sales {
		ev := metaevents.Event{
			EventName: "Purchase", EventTime: sale.at.Unix(), EventID: sale.id, ActionSource: sale.source,
			UserData:   sale.person.User(),
			CustomData: metaevents.CustomData{Value: round2(sale.value), Currency: metaevents.Currency, OrderID: sale.order, NumItems: sale.items},
		}
		if sale.source == metaevents.SourceWebsite {
			// Compra do site: o Meta pede o navegador do cliente. Pedido feito antes
			// de o site guardar o navegador fica só com o Pixel do navegador.
			if sale.ads == nil || sale.ads.UserAgent == "" {
				resp.Skipped.SiteSemNavegador++
				continue
			}
			ev.EventSourceURL = sale.url
			ev.UserData.UserAgent, ev.UserData.FBP, ev.UserData.FBC = sale.ads.UserAgent, sale.ads.FBP, sale.ads.FBC
		}
		if !ev.UserData.Matchable() {
			continue
		}
		resp.Events = append(resp.Events, ev)
	}
	writeJSON(w, resp)
}

type metaCustomersResponse struct {
	Schema []string   `json:"schema"`
	Data   [][]string `json:"data"`
	Counts struct {
		Site   int `json:"site"`
		Liga   int `json:"liga"`
		Balcao int `json:"balcao"`
	} `json:"counts"`
}

// handleMetaCustomers devolve quem comprou nos últimos dias (180 por padrão) como
// linhas com hash da lista de clientes do Meta, sem repetir a mesma pessoa.
func (s *Server) handleMetaCustomers(w http.ResponseWriter, r *http.Request) {
	days := min(max(intParam(r.URL.Query(), "days", metaCustomersDays), 1), metaCustomersLimit)
	now := time.Now()
	sales, _, err := s.metaSales(now.Add(-time.Duration(days)*24*time.Hour), now)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	resp := metaCustomersResponse{Schema: metaevents.CustomerSchema, Data: [][]string{}}
	seen := map[string]bool{}
	for _, sale := range sales {
		row, ok := sale.person.Row()
		if !ok {
			continue
		}
		key := row[0] + "|" + row[1]
		if seen[key] {
			continue
		}
		seen[key] = true
		resp.Data = append(resp.Data, row)
		switch sale.source {
		case metaevents.SourceWebsite:
			resp.Counts.Site++
		case metaevents.SourceOther:
			resp.Counts.Liga++
		default:
			resp.Counts.Balcao++
		}
	}
	writeJSON(w, resp)
}
