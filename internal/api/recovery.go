package api

import (
	"errors"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"opdeals/internal/orders"
	"opdeals/internal/postsale"
	"opdeals/internal/whatsapp"
)

// Recuperação de pedido do site que não foi pago (Pix expirado ou pagamento
// recusado): uma mensagem só, com o texto que o Jairo aprovou, mandada pelo
// Atendente no WhatsApp. Fica desligada até RECUPERAR_PEDIDOS=1 e
// RECUPERAR_PEDIDOS_TEXTO preenchido; desligada, a lista vem vazia e nada é
// marcado como enviado.
const (
	recoveryMinDelay   = 15 * time.Minute   // dá tempo de o cliente tentar de novo sozinho
	recoveryMaxAge     = 18 * time.Hour     // falha à noite ainda entra na manhã seguinte
	recoveryBuyerGap   = 7 * 24 * time.Hour // no máximo uma mensagem por cliente nesse prazo
	recoveryFirstHour  = 9                  // janela de envio, horário de Goiânia
	recoveryLastHour   = 21                 // até 20:59
	recoveryEnvFlag    = "RECUPERAR_PEDIDOS"
	recoveryEnvText    = "RECUPERAR_PEDIDOS_TEXTO"
	recoveryReasonPix  = "pix_expirado"
	recoveryReasonLate = "expirou_sem_pagamento"
	recoveryReasonFail = "pagamento_recusado"
)

type recoveryConfig struct {
	On   bool
	Text string
}

func recoveryFromEnv() recoveryConfig {
	text := strings.TrimSpace(os.Getenv(recoveryEnvText))
	return recoveryConfig{On: os.Getenv(recoveryEnvFlag) == "1" && text != "", Text: text}
}

type recoveryItem struct {
	Name string `json:"nome"`
	Qty  int    `json:"qtd"`
}

type recoveryOrder struct {
	ID        string         `json:"id"`
	Number    int            `json:"numero"`
	FirstName string         `json:"primeiroNome"`
	Phone     string         `json:"telefone"`
	Reason    string         `json:"motivo"`
	MPDetail  string         `json:"detalheMP,omitempty"`
	TotalBRL  float64        `json:"totalBRL"`
	Items     []recoveryItem `json:"itens"`
	Link      string         `json:"link"`
	FailedAt  time.Time      `json:"falhouEm"`
	Message   string         `json:"mensagem"`
}

type recoveryList struct {
	On         bool            `json:"ligado"`
	OutOfHours bool            `json:"foraDoHorario"`
	Orders     []recoveryOrder `json:"pedidos"`
}

// failedAt diz quando o pedido deixou de poder ser pago e por quê. ok=false
// quando o pedido não é candidato (pago, pendente, cancelado à mão, etc.).
func failedAt(o orders.Order, now time.Time) (time.Time, string, bool) {
	switch orders.EffectiveStatus(o, now) {
	case orders.StatusExpired:
		if o.PaymentMethod == orders.PaymentPix || o.MP.Method == "pix" || o.MP.Method == "bank_transfer" {
			return o.ExpiresAt, recoveryReasonPix, true
		}
		return o.ExpiresAt, recoveryReasonLate, true
	case orders.StatusFailed:
		if o.MP.Status == "approved" { // valor divergente: caso para o Jairo, não para mensagem
			return time.Time{}, "", false
		}
		return o.UpdatedAt, recoveryReasonFail, true
	}
	return time.Time{}, "", false
}

func buyerKeys(b orders.Buyer) []string {
	var keys []string
	if p := whatsapp.PhoneKey(b.Phone); p != "" {
		keys = append(keys, "tel:"+p)
	}
	if e := strings.ToLower(strings.TrimSpace(b.Email)); e != "" {
		keys = append(keys, "mail:"+e)
	}
	return keys
}

// recoveryCandidates aplica as regras: 15 min a 18 h depois da falha, nenhuma
// mensagem antes para o pedido, nenhuma para o mesmo cliente em 7 dias, e
// nenhum pedido mais novo do cliente (ele já tentou de novo ou comprou). Um
// pedido por cliente, o mais recente.
func recoveryCandidates(all []orders.Order, now time.Time) []orders.Order {
	lastNotified := map[string]time.Time{}
	newest := map[string]time.Time{}
	for _, o := range all {
		for _, k := range buyerKeys(o.Buyer) {
			if o.RecoveryNotifiedAt.After(lastNotified[k]) {
				lastNotified[k] = o.RecoveryNotifiedAt
			}
			if o.CreatedAt.After(newest[k]) {
				newest[k] = o.CreatedAt
			}
		}
	}
	var out []orders.Order
	taken := map[string]bool{}
	sorted := append([]orders.Order(nil), all...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].CreatedAt.After(sorted[j].CreatedAt) })
next:
	for _, o := range sorted {
		at, _, ok := failedAt(o, now)
		if !ok || !o.RecoveryNotifiedAt.IsZero() || len(o.Lines) == 0 || whatsapp.PhoneKey(o.Buyer.Phone) == "" {
			continue
		}
		if age := now.Sub(at); age < recoveryMinDelay || age > recoveryMaxAge {
			continue
		}
		for _, k := range buyerKeys(o.Buyer) {
			if taken[k] || newest[k].After(o.CreatedAt) || (!lastNotified[k].IsZero() && now.Sub(lastNotified[k]) < recoveryBuyerGap) {
				continue next
			}
		}
		for _, k := range buyerKeys(o.Buyer) {
			taken[k] = true
		}
		out = append(out, o)
	}
	return out
}

func inRecoveryHours(now time.Time) bool {
	h := now.In(saoPaulo).Hour()
	return h >= recoveryFirstHour && h < recoveryLastHour
}

func renderRecovery(text, firstName, link string, number int) string {
	r := strings.NewReplacer("{nome}", firstName, "{link}", link, "{numero}", strconv.Itoa(number))
	return r.Replace(text)
}

func (s *Server) handleRecoveryList(w http.ResponseWriter, r *http.Request) {
	cfg := recoveryFromEnv()
	now := s.psNow()
	out := recoveryList{On: cfg.On, Orders: []recoveryOrder{}}
	if !cfg.On {
		writeJSON(w, out)
		return
	}
	if !inRecoveryHours(now) {
		out.OutOfHours = true
		writeJSON(w, out)
		return
	}
	all, err := s.Checkout.Orders.List()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	site := strings.TrimRight(s.Checkout.SiteURL, "/")
	for _, o := range recoveryCandidates(all, now) {
		at, reason, _ := failedAt(o, now)
		first := postsale.FirstName(o.Buyer.Name)
		link := site + "/pedido/" + o.ID
		items := make([]recoveryItem, 0, len(o.Lines))
		for _, l := range o.Lines {
			items = append(items, recoveryItem{Name: l.Name, Qty: l.Qty})
		}
		out.Orders = append(out.Orders, recoveryOrder{
			ID: o.ID, Number: o.Number, FirstName: first, Phone: whatsapp.BrazilE164(o.Buyer.Phone),
			Reason: reason, MPDetail: o.MP.StatusDetail, TotalBRL: o.Totals.TotalBRL, Items: items,
			Link: link, FailedAt: at, Message: renderRecovery(cfg.Text, first, link, o.Number),
		})
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, out)
}

var errRecoveryAlreadySent = errors.New("mensagem de recuperação já enviada para este pedido")

func (s *Server) handleRecoverySent(w http.ResponseWriter, r *http.Request) {
	if !recoveryFromEnv().On {
		http.Error(w, "recuperação de pedidos desligada", http.StatusForbidden)
		return
	}
	now := s.psNow()
	o, err := s.Checkout.Orders.Transact(r.PathValue("id"), func(o *orders.Order) error {
		if !o.RecoveryNotifiedAt.IsZero() {
			return errRecoveryAlreadySent
		}
		o.RecoveryNotifiedAt = now
		return nil
	})
	switch {
	case errors.Is(err, orders.ErrNotFound):
		http.Error(w, "not found", http.StatusNotFound)
		return
	case errors.Is(err, errRecoveryAlreadySent):
		http.Error(w, err.Error(), http.StatusConflict)
		return
	case err != nil:
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.logCheckout("recuperação: mensagem do pedido #%d marcada como enviada", o.Number)
	writeJSON(w, map[string]any{"ok": true, "numero": o.Number, "enviadoEm": o.RecoveryNotifiedAt})
}
