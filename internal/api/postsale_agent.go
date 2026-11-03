package api

import (
	"cmp"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"opdeals/internal/conversations"
	"opdeals/internal/orders"
	"opdeals/internal/postsale"
	"opdeals/internal/whatsapp"
)

const agentBodyLimit = 64 << 10

var errClaimed = errors.New("item já reservado ou fora da fila")

func (s *Server) agentRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/postsale/agent/outbox/claim", s.agentOnly(s.handleAgentClaim))
	mux.HandleFunc("POST /api/postsale/agent/outbox/{id}/delivered", s.agentOnly(s.handleAgentDelivered))
	mux.HandleFunc("POST /api/postsale/agent/outbox/{id}/failed", s.agentOnly(s.handleAgentFailed))
	mux.HandleFunc("GET /api/postsale/agent/watch", s.agentOnly(s.handleAgentWatch))
	mux.HandleFunc("GET /api/postsale/agent/compose", s.agentOnly(s.handleAgentComposeList))
	mux.HandleFunc("POST /api/postsale/agent/conversations/{id}/compose", s.agentOnly(s.handleAgentCompose))
	mux.HandleFunc("POST /api/postsale/agent/conversations/{id}/inbound", s.agentOnly(s.handleAgentInbound))
	mux.HandleFunc("POST /api/postsale/agent/conversations/{id}/reply", s.agentOnly(s.handleAgentReply))
	mux.HandleFunc("POST /api/postsale/agent/conversations/{id}/handoff", s.agentOnly(s.handleAgentHandoff))
	mux.HandleFunc("POST /api/postsale/agent/conversations/{id}/link", s.agentOnly(s.handleAgentLink))
}

func (s *Server) agentOnly(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := s.PostSale.AgentToken
		if token == "" {
			http.Error(w, "agent token not configured", http.StatusServiceUnavailable)
			return
		}
		given, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok || subtle.ConstantTimeCompare([]byte(given), []byte(token)) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		s.postSale.mu.Lock()
		s.postSale.agentSeen = s.psNow()
		s.postSale.mu.Unlock()
		r.Body = http.MaxBytesReader(w, r.Body, agentBodyLimit)
		h(w, r)
	}
}

func (s *Server) agentSeenAt() time.Time {
	s.postSale.mu.Lock()
	defer s.postSale.mu.Unlock()
	return s.postSale.agentSeen
}

func agentError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func agentConversationError(w http.ResponseWriter, err error) {
	if errors.Is(err, conversations.ErrNotFound) {
		agentError(w, http.StatusNotFound, "conversa não encontrada")
		return
	}
	agentError(w, http.StatusConflict, err.Error())
}

type outboxItem struct {
	ID             string `json:"id"`
	Kind           string `json:"kind"`
	ConversationID string `json:"conversationId,omitempty"`
	OrderNumber    int    `json:"orderNumber"`
	Phone          string `json:"phone"`
	FirstName      string `json:"firstName"`
	Text           string `json:"text"`
}

type outboxClaim struct {
	Quiet bool         `json:"quiet"`
	Items []outboxItem `json:"items"`
}

func (s *Server) handleAgentClaim(w http.ResponseWriter, r *http.Request) {
	now := s.psNow()
	quiet := postsale.Quiet(now, saoPaulo)
	all, err := s.Checkout.Orders.List()
	if err != nil {
		agentError(w, http.StatusInternalServerError, err.Error())
		return
	}
	byID := make(map[string]orders.Order, len(all))
	for _, o := range all {
		byID[o.ID] = o
	}
	convs, err := s.PostSale.Conversations.List()
	if err != nil {
		agentError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := outboxClaim{Quiet: quiet, Items: []outboxItem{}}
	for _, c := range convs {
		for _, m := range c.Messages {
			if m.Status != conversations.MsgQueued || (!m.ClaimedAt.IsZero() && now.Sub(m.ClaimedAt) < outboxLease) {
				continue
			}
			if m.Kind == conversations.KindOffer && (quiet || c.State != conversations.StateQueued || !orderStillPaid(c, byID, now)) {
				continue
			}
			claimed, err := s.claimMessage(c.ID, m.ID, now)
			if err != nil {
				continue
			}
			out.Items = append(out.Items, outboxItem{
				ID:             "m." + c.ID + "." + m.ID,
				Kind:           claimed.Kind,
				ConversationID: c.ID,
				OrderNumber:    c.OrderNumber,
				Phone:          c.Phone,
				FirstName:      postsale.FirstName(c.Name),
				Text:           claimed.Text,
			})
		}
	}
	if !quiet {
		for _, o := range trackingDue(all, now) {
			code := o.TrackingCode
			updated, err := s.Checkout.Orders.Transact(o.ID, func(ord *orders.Order) error {
				if ord.TrackingCode != code || ord.TrackingNotified == code || (!ord.TrackingNotifiedAt.IsZero() && now.Sub(ord.TrackingNotifiedAt) < trackingRetry) {
					return errClaimed
				}
				ord.TrackingNotifiedAt = now
				return nil
			})
			if err != nil {
				continue
			}
			out.Items = append(out.Items, outboxItem{
				ID:          "t." + o.ID + "." + code,
				Kind:        "tracking",
				OrderNumber: updated.Number,
				Phone:       whatsapp.BrazilE164(updated.Buyer.Phone),
				FirstName:   postsale.FirstName(updated.Buyer.Name),
				Text:        renderTracking(updated),
			})
		}
	}
	writeJSON(w, out)
}

func (s *Server) claimMessage(convID, msgID string, now time.Time) (conversations.Message, error) {
	var claimed conversations.Message
	_, err := s.PostSale.Conversations.Transact(convID, func(cv *conversations.Conversation) error {
		i := messageIndex(*cv, msgID)
		if i < 0 {
			return errClaimed
		}
		m := &cv.Messages[i]
		if m.Status != conversations.MsgQueued || (!m.ClaimedAt.IsZero() && now.Sub(m.ClaimedAt) < outboxLease) {
			return errClaimed
		}
		if m.Kind == conversations.KindOffer && cv.State != conversations.StateQueued {
			return errClaimed
		}
		m.ClaimedAt = now
		claimed = *m
		return nil
	})
	return claimed, err
}

func messageIndex(c conversations.Conversation, msgID string) int {
	for i, m := range c.Messages {
		if m.ID == msgID {
			return i
		}
	}
	return -1
}

func parseOutboxID(id string) (string, string, string, bool) {
	parts := strings.SplitN(id, ".", 3)
	if len(parts) != 3 || parts[1] == "" || parts[2] == "" || (parts[0] != "m" && parts[0] != "t") {
		return "", "", "", false
	}
	return parts[0], parts[1], parts[2], true
}

func (s *Server) handleAgentDelivered(w http.ResponseWriter, r *http.Request) {
	kind, ref, sub, ok := parseOutboxID(r.PathValue("id"))
	if !ok {
		agentError(w, http.StatusBadRequest, "id inválido")
		return
	}
	now := s.psNow()
	if kind == "t" {
		if _, err := s.Checkout.Orders.Update(ref, func(o *orders.Order) {
			if o.TrackingCode == sub {
				o.TrackingNotified = sub
			}
		}); err != nil {
			agentError(w, http.StatusNotFound, "pedido não encontrado")
			return
		}
		writeJSON(w, map[string]bool{"ok": true})
		return
	}
	_, err := s.PostSale.Conversations.Transact(ref, func(cv *conversations.Conversation) error {
		i := messageIndex(*cv, sub)
		if i < 0 {
			return conversations.ErrNotFound
		}
		m := &cv.Messages[i]
		if m.Status == conversations.MsgSent {
			return nil
		}
		if m.Status != conversations.MsgQueued && m.Status != conversations.MsgFailed {
			return errClaimed
		}
		m.Status = conversations.MsgSent
		m.Error = ""
		m.At = now
		if m.Kind == conversations.KindOffer {
			if cv.State == conversations.StateQueued {
				markOfferSent(cv, now)
			} else if cv.SentAt.IsZero() {
				cv.SentAt = now
			}
		}
		return nil
	})
	if err != nil {
		agentConversationError(w, err)
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

func (s *Server) handleAgentFailed(w http.ResponseWriter, r *http.Request) {
	kind, ref, sub, ok := parseOutboxID(r.PathValue("id"))
	if !ok {
		agentError(w, http.StatusBadRequest, "id inválido")
		return
	}
	var in struct {
		Error     string `json:"error"`
		Permanent bool   `json:"permanent"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		agentError(w, http.StatusBadRequest, err.Error())
		return
	}
	reason := strings.TrimSpace(in.Error)
	if reason == "" {
		reason = "falha no WhatsApp Web"
	}
	if kind == "t" {
		if in.Permanent {
			if _, err := s.Checkout.Orders.Update(ref, func(o *orders.Order) {
				if o.TrackingCode == sub {
					o.TrackingNotified = sub
				}
			}); err != nil {
				agentError(w, http.StatusNotFound, "pedido não encontrado")
				return
			}
		}
		s.logPostSale("rastreio %s não entregue: %s", ref, reason)
		writeJSON(w, map[string]bool{"ok": true})
		return
	}
	c, err := s.PostSale.Conversations.Transact(ref, func(cv *conversations.Conversation) error {
		i := messageIndex(*cv, sub)
		if i < 0 {
			return conversations.ErrNotFound
		}
		m := &cv.Messages[i]
		if m.Status != conversations.MsgQueued {
			return errClaimed
		}
		m.Status = conversations.MsgFailed
		m.Error = reason
		if m.Kind == conversations.KindOffer {
			cv.Close(conversations.OutcomeFailed)
		}
		return nil
	})
	if err != nil {
		agentConversationError(w, err)
		return
	}
	s.notify("Pós-venda: mensagem não entregue", fmt.Sprintf("%s (pedido #%d): %s", postsale.FirstName(c.Name), c.OrderNumber, reason))
	writeJSON(w, map[string]bool{"ok": true})
}

type watchMessage struct {
	Role string    `json:"role"`
	Text string    `json:"text"`
	At   time.Time `json:"at"`
}

type watchConversation struct {
	ConversationID string              `json:"conversationId"`
	Phone          string              `json:"phone"`
	FirstName      string              `json:"firstName"`
	OrderNumber    int                 `json:"orderNumber"`
	State          conversations.State `json:"state"`
	WindowEndsAt   time.Time           `json:"windowEndsAt"`
	LastMessages   []watchMessage      `json:"lastMessages"`
}

func (s *Server) handleAgentWatch(w http.ResponseWriter, _ *http.Request) {
	convs, err := s.PostSale.Conversations.List()
	if err != nil {
		agentError(w, http.StatusInternalServerError, err.Error())
		return
	}
	now := s.psNow()
	out := struct {
		Conversations []watchConversation `json:"conversations"`
	}{Conversations: []watchConversation{}}
	for _, c := range convs {
		if (c.State != conversations.StateWait && c.State != conversations.StateActive) || c.SentAt.IsZero() || !now.Before(c.WindowEndsAt) {
			continue
		}
		var last []watchMessage
		for _, m := range c.Messages {
			if m.Role == conversations.RoleSystem || (m.Status != conversations.MsgSent && m.Status != conversations.MsgReceived) {
				continue
			}
			last = append(last, watchMessage{Role: m.Role, Text: m.Text, At: m.At})
		}
		if len(last) > 3 {
			last = last[len(last)-3:]
		}
		out.Conversations = append(out.Conversations, watchConversation{
			ConversationID: c.ID,
			Phone:          c.Phone,
			FirstName:      postsale.FirstName(c.Name),
			OrderNumber:    c.OrderNumber,
			State:          c.State,
			WindowEndsAt:   c.WindowEndsAt,
			LastMessages:   last,
		})
	}
	writeJSON(w, out)
}

type agentTurn struct {
	Role string `json:"role"`
	Text string `json:"text"`
}

type agentContext struct {
	Channel     string                `json:"channel"`
	FirstName   string                `json:"firstName"`
	OrderNumber int                   `json:"orderNumber"`
	Pickup      bool                  `json:"pickup,omitempty"`
	CanAddHow   string                `json:"canAddHow"`
	Status      orders.Status         `json:"status,omitempty"`
	Rastreio    string                `json:"rastreio,omitempty"`
	Bought      []postsale.BoughtLine `json:"bought"`
	Offer       *conversations.Offer  `json:"offer"`
	WindowOpen  bool                  `json:"windowOpen"`
	CanAddUntil string                `json:"canAddUntil,omitempty"`
	AutoSend    bool                  `json:"autoSend"`
	History     []agentTurn           `json:"history"`
}

type agentInboundResponse struct {
	Action         string        `json:"action"`
	Reason         string        `json:"reason,omitempty"`
	ConversationID string        `json:"conversationId"`
	Context        *agentContext `json:"context,omitempty"`
}

func parseAgentTime(raw string, now time.Time) time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return now
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return notAfter(t, now)
	}
	if hm, err := time.ParseInLocation("15:04", raw, saoPaulo); err == nil {
		local := now.In(saoPaulo)
		t := time.Date(local.Year(), local.Month(), local.Day(), hm.Hour(), hm.Minute(), 0, 0, saoPaulo)
		if t.After(now.Add(time.Minute)) {
			t = t.AddDate(0, 0, -1)
		}
		return notAfter(t, now)
	}
	return now
}

func notAfter(t, limit time.Time) time.Time {
	if t.After(limit) {
		return limit
	}
	return t
}

func normalizedText(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func webMessageID(phone string, at time.Time, text string) string {
	sum := sha256.Sum256([]byte(whatsapp.PhoneKey(phone) + "|" + at.Truncate(time.Minute).UTC().Format(time.RFC3339) + "|" + normalizedText(text)))
	return "web:" + hex.EncodeToString(sum[:])[:20]
}

func (s *Server) recordStoreMessage(id, text string, at time.Time) (bool, error) {
	norm := normalizedText(text)
	_, err := s.PostSale.Conversations.Transact(id, func(cv *conversations.Conversation) error {
		for _, m := range cv.Messages {
			if m.Role != conversations.RoleAssistant && m.Role != conversations.RoleOwner {
				continue
			}
			if (m.Status == conversations.MsgSent || m.Status == conversations.MsgQueued) && normalizedText(m.Text) == norm {
				return errDuplicate
			}
		}
		msg := conversations.NewMessage(conversations.RoleOwner, text, conversations.MsgSent, at)
		msg.WAID = webMessageID(cv.Phone, at, text)
		cv.DropDrafts()
		cv.Messages = append(cv.Messages, msg, conversations.NewMessage(conversations.RoleSystem, "você respondeu direto no WhatsApp: IA pausada", conversations.MsgReceived, s.psNow()))
		if cv.State != conversations.StateClosed {
			cv.State = conversations.StateHuman
		}
		return nil
	})
	if errors.Is(err, errDuplicate) {
		return false, nil
	}
	return err == nil, err
}

func (s *Server) handleAgentInbound(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	c, err := s.PostSale.Conversations.Get(id)
	if err != nil {
		agentConversationError(w, err)
		return
	}
	if c.Unsent() {
		agentError(w, http.StatusConflict, "a oferta desta conversa ainda não foi entregue")
		return
	}
	var in struct {
		Messages []struct {
			Text      string `json:"text"`
			At        string `json:"at"`
			FromStore bool   `json:"fromStore"`
		} `json:"messages"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		agentError(w, http.StatusBadRequest, err.Error())
		return
	}
	now := s.psNow()
	ownerTyped := false
	customerNew := false
	handoff := ""
	for _, m := range in.Messages {
		text := strings.TrimSpace(m.Text)
		if text == "" {
			continue
		}
		at := parseAgentTime(m.At, now)
		if m.FromStore {
			recorded, err := s.recordStoreMessage(id, text, at)
			if err != nil {
				agentConversationError(w, err)
				return
			}
			ownerTyped = ownerTyped || recorded
			continue
		}
		_, _, reason, err := s.recordInbound(id, whatsapp.Inbound{ID: webMessageID(c.Phone, at, text), From: c.Phone, Text: text, At: at})
		if errors.Is(err, errDuplicate) {
			continue
		}
		if err != nil {
			agentConversationError(w, err)
			return
		}
		customerNew = true
		if reason != "" {
			handoff = reason
		}
	}
	c, err = s.PostSale.Conversations.Get(id)
	if err != nil {
		agentConversationError(w, err)
		return
	}
	resp := agentInboundResponse{Action: "none", Reason: "nada novo", ConversationID: id}
	switch {
	case ownerTyped:
		resp.Reason = "a loja respondeu pelo celular: a IA pausou nesta conversa"
	case customerNew && c.State == conversations.StateActive:
		resp.Action = "answer"
		resp.Reason = ""
		resp.Context, err = s.agentContextFor(c)
		if err != nil {
			agentConversationError(w, err)
			return
		}
	case customerNew:
		resp.Action = "handoff"
		resp.Reason = cmp.Or(handoff, "a conversa está com o dono")
	}
	writeJSON(w, resp)
}

func (s *Server) agentContextFor(c conversations.Conversation) (*agentContext, error) {
	now := s.psNow()
	ctx := &agentContext{
		Channel:     channelOf(c),
		FirstName:   postsale.FirstName(c.Name),
		OrderNumber: c.OrderNumber,
		Pickup:      c.Pickup,
		CanAddHow:   canAddHow(c),
		Offer:       c.Offer,
		WindowOpen:  !c.WindowEndsAt.IsZero() && now.Before(c.WindowEndsAt),
		AutoSend:    s.PostSale.AutoSend,
		History:     []agentTurn{},
	}
	if c.IsLiga() {
		ctx.Bought = itemsView(c.Items)
	} else {
		base, err := s.Checkout.Orders.Get(c.OrderID)
		if err != nil {
			return nil, err
		}
		ctx.Status = orders.EffectiveStatus(base, now)
		ctx.Rastreio = base.TrackingCode
		ctx.Bought = postsale.BoughtView(base.Lines)
	}
	if !c.WindowEndsAt.IsZero() {
		ctx.CanAddUntil = c.WindowEndsAt.In(saoPaulo).Format("02/01 15:04")
	}
	for _, m := range c.Messages {
		if m.Role == conversations.RoleSystem || m.Status == conversations.MsgDraft || m.Status == conversations.MsgFailed {
			continue
		}
		ctx.History = append(ctx.History, agentTurn{Role: m.Role, Text: m.Text})
	}
	return ctx, nil
}

func (s *Server) handleAgentReply(w http.ResponseWriter, r *http.Request) {
	text, err := decodeText(r)
	if err != nil {
		agentError(w, http.StatusBadRequest, err.Error())
		return
	}
	if text == "" {
		agentError(w, http.StatusBadRequest, "texto vazio")
		return
	}
	if err := s.deliverAgentText(r.Context(), r.PathValue("id"), text); err != nil {
		agentConversationError(w, err)
		return
	}
	status := "draft"
	if s.PostSale.AutoSend {
		status = "queued"
	}
	writeJSON(w, map[string]string{"status": status})
}

func (s *Server) handleAgentHandoff(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		agentError(w, http.StatusBadRequest, err.Error())
		return
	}
	id := r.PathValue("id")
	c, err := s.PostSale.Conversations.Get(id)
	if err != nil {
		agentConversationError(w, err)
		return
	}
	if c.State == conversations.StateClosed {
		agentError(w, http.StatusConflict, "conversa encerrada")
		return
	}
	s.handOff(id, cmp.Or(strings.TrimSpace(in.Reason), "o atendente pediu ajuda"))
	writeJSON(w, map[string]bool{"ok": true})
}

func (s *Server) handleAgentLink(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Itens []struct {
			SKU string `json:"sku"`
			Qtd int    `json:"qtd"`
		} `json:"itens"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		agentError(w, http.StatusBadRequest, err.Error())
		return
	}
	c, err := s.PostSale.Conversations.Get(r.PathValue("id"))
	if err != nil {
		agentConversationError(w, err)
		return
	}
	if c.IsLiga() {
		agentError(w, http.StatusConflict, ligaAddHint)
		return
	}
	if c.State != conversations.StateWait && c.State != conversations.StateActive {
		agentError(w, http.StatusConflict, "a conversa não está com o atendente")
		return
	}
	base, err := s.Checkout.Orders.Get(c.OrderID)
	if err != nil {
		agentError(w, http.StatusConflict, "pedido do pós-venda não encontrado")
		return
	}
	lines := make([]conversations.LinkLine, 0, len(in.Itens))
	for _, it := range in.Itens {
		lines = append(lines, conversations.LinkLine{SKU: it.SKU, Qty: it.Qtd})
	}
	url, err := (&convTools{s: s, conv: c, order: base}).ComplementLink(lines)
	if err != nil {
		agentError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, map[string]string{"url": url})
}

const (
	ligaAddHint = "pedido da Liga: oriente a usar 'Adicionar a um pedido já existente' na Liga"
	siteAddHow  = "site: dá pra adicionar no mesmo envio sem frete extra. Se o cliente quiser, gere o link com posvenda_link depois que ele responder"
	ligaAddHow  = "liga: o cliente faz um novo pedido na Liga e escolhe 'Adicionar a um pedido já existente'. Nunca venda fora da Liga"
)

func channelOf(c conversations.Conversation) string {
	if c.IsLiga() {
		return conversations.ChannelLiga
	}
	return conversations.ChannelSite
}

func canAddHow(c conversations.Conversation) string {
	if c.IsLiga() {
		return ligaAddHow
	}
	return siteAddHow
}

func itemsView(items []conversations.Item) []postsale.BoughtLine {
	out := make([]postsale.BoughtLine, 0, len(items))
	for _, it := range items {
		out = append(out, postsale.BoughtLine{Nome: it.Name, Qtd: it.Qty})
	}
	return out
}

type composeItem struct {
	ConversationID string               `json:"conversationId"`
	Channel        string               `json:"channel"`
	FirstName      string               `json:"firstName"`
	OrderNumber    int                  `json:"orderNumber"`
	Pickup         bool                 `json:"pickup"`
	Shipped        bool                 `json:"shipped"`
	Items          []conversations.Item `json:"items"`
	CanAddHow      string               `json:"canAddHow"`
}

func (s *Server) handleAgentComposeList(w http.ResponseWriter, _ *http.Request) {
	all, err := s.Checkout.Orders.List()
	if err != nil {
		agentError(w, http.StatusInternalServerError, err.Error())
		return
	}
	byID := make(map[string]orders.Order, len(all))
	for _, o := range all {
		byID[o.ID] = o
	}
	convs, err := s.PostSale.Conversations.List()
	if err != nil {
		agentError(w, http.StatusInternalServerError, err.Error())
		return
	}
	now := s.psNow()
	out := struct {
		Items []composeItem `json:"items"`
	}{Items: []composeItem{}}
	for _, c := range convs {
		if c.State != conversations.StateCompose || unsentStale(c, byID, now) != "" {
			continue
		}
		items := c.Items
		if items == nil {
			items = []conversations.Item{}
		}
		out.Items = append(out.Items, composeItem{
			ConversationID: c.ID,
			Channel:        channelOf(c),
			FirstName:      postsale.FirstName(c.Name),
			OrderNumber:    c.OrderNumber,
			Pickup:         c.Pickup,
			Shipped:        c.Shipped,
			Items:          items,
			CanAddHow:      canAddHow(c),
		})
	}
	writeJSON(w, out)
}

var errNotCompose = errors.New("a mensagem desta conversa já foi escrita ou a conversa foi encerrada")

func (s *Server) handleAgentCompose(w http.ResponseWriter, r *http.Request) {
	text, err := decodeText(r)
	if err != nil {
		agentError(w, http.StatusBadRequest, err.Error())
		return
	}
	switch {
	case text == "":
		agentError(w, http.StatusBadRequest, "texto vazio")
		return
	case strings.ContainsAny(text, "—–"):
		agentError(w, http.StatusBadRequest, "troque o travessão por vírgula ou ponto")
		return
	case len([]rune(text)) > postsale.MaxOpeningRunes:
		agentError(w, http.StatusBadRequest, fmt.Sprintf("mensagem longa demais: no máximo %d caracteres", postsale.MaxOpeningRunes))
		return
	}
	text = postsale.CleanOpening(text)
	id := r.PathValue("id")
	mu := s.conversationLock(id)
	mu.Lock()
	c, err := s.PostSale.Conversations.Transact(id, func(cv *conversations.Conversation) error {
		if cv.State != conversations.StateCompose {
			return errNotCompose
		}
		cv.Opening = text
		cv.State = conversations.StateDraft
		if s.PostSale.AutoSend {
			cv.State = conversations.StateQueued
		}
		return nil
	})
	mu.Unlock()
	if err != nil {
		agentConversationError(w, err)
		return
	}
	if c.State == conversations.StateDraft {
		s.notify("Pós-venda: aprovar mensagem", fmt.Sprintf("%s (%s #%d): %s", postsale.FirstName(c.Name), channelLabel(c), c.OrderNumber, text))
	} else if !postsale.Quiet(s.psNow(), saoPaulo) {
		if _, err := s.sendOffer(r.Context(), id); err != nil {
			s.logPostSale("enfileirar mensagem #%d: %v", c.OrderNumber, err)
		}
	}
	writeJSON(w, map[string]string{"status": string(c.State)})
}

func channelLabel(c conversations.Conversation) string {
	if c.IsLiga() {
		return "Liga"
	}
	return "pedido"
}

func (s *Server) sweepOutbox(all []orders.Order, now time.Time) {
	convs, err := s.PostSale.Conversations.List()
	if err != nil {
		s.logPostSale("conversas: %v", err)
		return
	}
	quiet := postsale.Quiet(now, saoPaulo)
	pending := 0
	for _, c := range convs {
		if c.State == conversations.StateCompose {
			pending++
		}
		for _, m := range c.Messages {
			if m.Status != conversations.MsgQueued {
				continue
			}
			if m.Kind == conversations.KindReply && now.Sub(m.At) > queuedReplyTTL {
				s.expireQueued(c, m)
				continue
			}
			if m.Kind == conversations.KindOffer && quiet {
				continue
			}
			pending++
		}
	}
	if !quiet {
		pending += len(trackingDue(all, now))
	}
	s.alertIfAgentStale(pending, now)
}

func (s *Server) expireQueued(c conversations.Conversation, m conversations.Message) {
	_, err := s.PostSale.Conversations.Transact(c.ID, func(cv *conversations.Conversation) error {
		i := messageIndex(*cv, m.ID)
		if i < 0 || cv.Messages[i].Status != conversations.MsgQueued {
			return errClaimed
		}
		cv.Messages[i].Status = conversations.MsgFailed
		cv.Messages[i].Error = "o OpenBot não entregou a tempo"
		return nil
	})
	if err != nil {
		return
	}
	s.notify("Pós-venda: mensagem não entregue", fmt.Sprintf("%s (pedido #%d): o OpenBot não entregou em 2h: %s", postsale.FirstName(c.Name), c.OrderNumber, m.Text))
}

func (s *Server) alertIfAgentStale(pending int, now time.Time) {
	s.postSale.mu.Lock()
	if s.postSale.outboxSince.IsZero() {
		s.postSale.outboxSince = now
	}
	seen := s.postSale.agentSeen
	reference := seen
	if reference.Before(s.postSale.outboxSince) {
		reference = s.postSale.outboxSince
	}
	alerted := s.postSale.agentAlertAt
	if pending == 0 || now.Sub(reference) <= agentStaleAfter || (!alerted.IsZero() && now.Sub(alerted) < agentAlertEvery) {
		s.postSale.mu.Unlock()
		return
	}
	s.postSale.agentAlertAt = now
	s.postSale.mu.Unlock()
	last := "nunca apareceu desde o último deploy"
	if !seen.IsZero() {
		last = "visto às " + seen.In(saoPaulo).Format("15:04")
	}
	s.notify("OpenBot parado", fmt.Sprintf("%d mensagem(ns) do pós-venda esperando o OpenBot (%s). Confira se o Mac está acordado e o OpenBot aberto.", pending, last))
}
