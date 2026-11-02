package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"opdeals/internal/conversations"
	"opdeals/internal/orders"
	"opdeals/internal/postsale"
	"opdeals/internal/shipping"
	"opdeals/internal/tracking"
	"opdeals/internal/trades"
	"opdeals/internal/whatsapp"
)

type PostSale struct {
	Conversations    *conversations.Store
	WhatsApp         whatsapp.Sender
	Brain            postsale.Brain
	AppSecret        string
	VerifyToken      string
	OfferTemplate    string
	TrackingTemplate string
	TemplateLang     string
	AutoSend         bool
	NtfyTopic        string
	Outbox           bool
	AgentToken       string
}

type postSaleState struct {
	locks        sync.Map
	mu           sync.Mutex
	skipped      map[string]bool
	market       map[string]float64
	marketAt     time.Time
	now          func() time.Time
	agentSeen    time.Time
	agentAlertAt time.Time
	outboxSince  time.Time
}

func (s *Server) psNow() time.Time {
	if s.postSale.now != nil {
		return s.postSale.now()
	}
	return time.Now()
}

const (
	salesWindowDays   = 14
	marketCacheTTL    = time.Hour
	trackingFreshness = 72 * time.Hour
	agentTimeout      = 2 * time.Minute
	historyMatchFor   = 14 * 24 * time.Hour
	humanIdleFor      = 7 * 24 * time.Hour
	trackingRetry     = 30 * time.Minute
	outboxLease       = 10 * time.Minute
	queuedReplyTTL    = 2 * time.Hour
	agentStaleAfter   = 15 * time.Minute
	agentAlertEvery   = time.Hour
)

func (s *Server) PostSaleEnabled() bool {
	return s.PostSale.Conversations != nil && s.Checkout.Orders != nil
}

func (s *Server) canDeliver() bool {
	return s.PostSale.WhatsApp != nil || s.PostSale.Outbox
}

func (s *Server) logPostSale(format string, args ...any) {
	fmt.Printf("[POSVENDA] "+format+"\n", args...)
}

func (s *Server) notify(title, msg string) {
	topic := s.PostSale.NtfyTopic
	if topic == "" {
		return
	}
	go func() {
		req, err := http.NewRequest(http.MethodPost, "https://ntfy.sh/"+topic, strings.NewReader(msg))
		if err != nil {
			return
		}
		req.Header.Set("Title", title)
		req.Header.Set("Tags", "speech_balloon")
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			s.logPostSale("ntfy: %v", err)
			return
		}
		resp.Body.Close()
	}()
}

func renderOffer(firstName string, number int, pitch string) string {
	return fmt.Sprintf("Oi %s! Obrigado por comprar na Collecta 💗 Seu pedido #%d tá sendo separado. Quer aproveitar o mesmo frete? %s Responde aqui que a gente adiciona antes de enviar!", firstName, number, pitch)
}

func offerText(c conversations.Conversation) string {
	if c.Opening != "" {
		return c.Opening
	}
	return renderOffer(postsale.FirstName(c.Name), c.OrderNumber, c.Pitch)
}

func (s *Server) RunPostSale(ctx context.Context, every time.Duration) {
	s.postSaleTick(ctx)
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.postSaleTick(ctx)
		}
	}
}

func (s *Server) postSaleTick(ctx context.Context) {
	if !s.PostSaleEnabled() {
		return
	}
	now := s.psNow()
	all, err := s.Checkout.Orders.List()
	if err != nil {
		s.logPostSale("pedidos: %v", err)
		return
	}
	s.sweepConversations(ctx, all, now)
	s.notifyTracking(ctx, all, now)
	s.draftOffers(ctx, all, now)
}

func (s *Server) draftOffers(ctx context.Context, all []orders.Order, now time.Time) {
	existing, err := s.PostSale.Conversations.List()
	if err != nil {
		s.logPostSale("conversas: %v", err)
		return
	}
	if s.PostSale.Outbox {
		s.composeOffers(all, existing, now)
		return
	}
	var items []postsale.Item
	var velocity map[string]float64
	for _, o := range all {
		if s.wasSkipped(o.ID) || !cloudEligible(o, existing, now) {
			continue
		}
		if items == nil {
			items, err = s.stockItems()
			if err != nil {
				s.logPostSale("estoque: %v", err)
				return
			}
			velocity = s.salesVelocity(items)
		}
		cands := postsale.Candidates(o.Lines, o.Totals.ItemsBRL, items, velocity)
		if len(cands) == 0 {
			s.markSkipped(o.ID)
			s.logPostSale("pedido #%d sem candidato de oferta", o.Number)
			continue
		}
		chosen, pitch := s.choosePitch(ctx, o, cands)
		c, created, err := s.PostSale.Conversations.Create(func(existing []conversations.Conversation) (conversations.Conversation, bool) {
			if !cloudEligible(o, existing, s.psNow()) {
				return conversations.Conversation{}, false
			}
			state := conversations.StateDraft
			if s.PostSale.AutoSend {
				state = conversations.StateQueued
			}
			return conversations.Conversation{
				Channel:     conversations.ChannelSite,
				Phone:       whatsapp.BrazilE164(o.Buyer.Phone),
				Name:        o.Buyer.Name,
				OrderID:     o.ID,
				OrderNumber: o.Number,
				State:       state,
				Offer:       &conversations.Offer{SKU: chosen.SKU, Name: chosen.Name, Meta: chosen.Meta, UnitBRL: chosen.UnitBRL, ImageURL: chosen.ImageURL},
				Pitch:       pitch,
				LinkToken:   conversations.NewID(),
				LinkLines:   []conversations.LinkLine{{SKU: chosen.SKU, Qty: 1}},
				Messages:    []conversations.Message{},
			}, true
		})
		if err != nil {
			s.logPostSale("criar conversa #%d: %v", o.Number, err)
			continue
		}
		if !created {
			continue
		}
		existing = append(existing, c)
		if c.State == conversations.StateDraft {
			s.notify("Pós-venda: aprovar oferta", fmt.Sprintf("Pedido #%d (%s): %s", o.Number, postsale.FirstName(o.Buyer.Name), pitch))
		}
	}
}

func cloudEligible(o orders.Order, existing []conversations.Conversation, now time.Time) bool {
	return o.MarketingOptIn && o.Shipping.ServiceID >= 0 && postsale.EligibleForOffer(o, existing, now)
}

func (s *Server) composeOffers(all []orders.Order, existing []conversations.Conversation, now time.Time) {
	for _, o := range all {
		if !postsale.EligibleForOffer(o, existing, now) {
			continue
		}
		c, created, err := s.PostSale.Conversations.Create(func(existing []conversations.Conversation) (conversations.Conversation, bool) {
			if !postsale.EligibleForOffer(o, existing, s.psNow()) {
				return conversations.Conversation{}, false
			}
			return conversations.Conversation{
				Channel:     conversations.ChannelSite,
				Phone:       whatsapp.BrazilE164(o.Buyer.Phone),
				Name:        o.Buyer.Name,
				OrderID:     o.ID,
				OrderNumber: o.Number,
				Pickup:      o.Shipping.ServiceID == shipping.PickupServiceID,
				Items:       orderItems(o.Lines),
				State:       conversations.StateCompose,
				Messages:    []conversations.Message{},
			}, true
		})
		if err != nil {
			s.logPostSale("criar conversa #%d: %v", o.Number, err)
			continue
		}
		if created {
			existing = append(existing, c)
		}
	}
}

func orderItems(lines []orders.Line) []conversations.Item {
	out := make([]conversations.Item, 0, len(lines))
	for _, l := range lines {
		out = append(out, conversations.Item{Name: l.Name, Qty: l.Qty})
	}
	return out
}

func (s *Server) wasSkipped(orderID string) bool {
	s.postSale.mu.Lock()
	defer s.postSale.mu.Unlock()
	return s.postSale.skipped[orderID]
}

func (s *Server) markSkipped(orderID string) {
	s.postSale.mu.Lock()
	defer s.postSale.mu.Unlock()
	if s.postSale.skipped == nil {
		s.postSale.skipped = make(map[string]bool)
	}
	s.postSale.skipped[orderID] = true
}

func (s *Server) choosePitch(ctx context.Context, o orders.Order, cands []postsale.Candidate) (postsale.Candidate, string) {
	if s.PostSale.Brain != nil {
		cctx, cancel := context.WithTimeout(ctx, agentTimeout)
		sku, pitch, err := s.PostSale.Brain.Pitch(cctx, postsale.PitchInput{FirstName: postsale.FirstName(o.Buyer.Name), Bought: o.Lines, Candidates: cands})
		cancel()
		if err == nil && pitch != "" {
			for _, c := range cands {
				if c.SKU == sku {
					return c, pitch
				}
			}
		}
		if err != nil {
			s.logPostSale("pitch #%d: %v", o.Number, err)
		}
	}
	return cands[0], postsale.FallbackPitch(cands[0])
}

func (s *Server) stockItems() ([]postsale.Item, error) {
	idx, err := s.buildSKUIndex()
	if err != nil {
		return nil, err
	}
	reserved, err := s.Checkout.Orders.Reserved(s.psNow())
	if err != nil {
		return nil, err
	}
	items := make([]postsale.Item, 0, len(idx))
	for sku, e := range idx {
		eligible := orders.Eligible(e.trades)
		if len(eligible) == 0 {
			continue
		}
		unit, _ := orders.UnitPrice(e.trades)
		first := eligible[0]
		featured := false
		for _, t := range eligible {
			featured = featured || t.Featured
		}
		items = append(items, postsale.Item{
			SKU:       sku,
			Game:      e.game,
			Kind:      first.Kind,
			Name:      first.Name,
			Set:       first.Set,
			Number:    first.Number,
			Meta:      lineMeta(first),
			ImageURL:  lineImage(eligible, e.game, e.lookup),
			UnitBRL:   round2(unit),
			Available: orders.Available(e.trades) - reserved[sku],
			Featured:  featured,
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].SKU < items[j].SKU })
	return items, nil
}

func (s *Server) salesVelocity(items []postsale.Item) map[string]float64 {
	since := s.psNow().AddDate(0, 0, -salesWindowDays).Format("2006-01-02")
	out := make(map[string]float64)
	count := func(gameID string, all []trades.Trade) {
		for _, t := range all {
			if t.Status != "sold" || t.SellDate < since {
				continue
			}
			out[trades.SKU(gameID, t)] += float64(max(t.Qty, 1)) * 2
		}
	}
	for _, id := range s.orderedGameIDs() {
		gs := s.games[id]
		if gs.Trades == nil {
			continue
		}
		if all, err := gs.Trades.List(); err == nil {
			count(gs.Game.ID, all)
		}
	}
	market := s.marketVelocity()
	for _, it := range items {
		out[it.SKU] += market[it.Game+"|"+strings.ToUpper(it.Number)]
	}
	return out
}

func (s *Server) marketVelocity() map[string]float64 {
	s.postSale.mu.Lock()
	if s.postSale.market != nil && s.psNow().Sub(s.postSale.marketAt) < marketCacheTTL {
		m := s.postSale.market
		s.postSale.mu.Unlock()
		return m
	}
	s.postSale.mu.Unlock()
	from := s.psNow().AddDate(0, 0, -salesWindowDays).Format("2006-01-02")
	out := make(map[string]float64)
	for _, id := range s.orderedGameIDs() {
		gs := s.games[id]
		if gs.Track == nil || !gs.Game.UniqueCardNumbers {
			continue
		}
		for _, set := range s.trackSets(gs) {
			days, err := gs.Track.LoadRange(set, from, "")
			if err != nil || len(days) < 2 {
				continue
			}
			for _, cs := range tracking.TopSoldCards(days) {
				out[gs.Game.ID+"|"+strings.ToUpper(cs.Number)] += float64(cs.Units)
			}
		}
	}
	s.postSale.mu.Lock()
	s.postSale.market = out
	s.postSale.marketAt = s.psNow()
	s.postSale.mu.Unlock()
	return out
}

func (s *Server) sweepConversations(ctx context.Context, all []orders.Order, now time.Time) {
	convs, err := s.PostSale.Conversations.List()
	if err != nil {
		s.logPostSale("conversas: %v", err)
		return
	}
	byID := make(map[string]orders.Order, len(all))
	for _, o := range all {
		byID[o.ID] = o
	}
	for _, c := range convs {
		switch c.State {
		case conversations.StateCompose, conversations.StateDraft, conversations.StateQueued:
			if note := unsentStale(c, byID, now); note != "" {
				s.closeConversation(c.ID, conversations.OutcomeSkipped, note)
				continue
			}
			if c.State == conversations.StateQueued && !postsale.Quiet(now, saoPaulo) {
				if _, err := s.sendOffer(ctx, c.ID); err != nil {
					s.logPostSale("enviar oferta #%d: %v", c.OrderNumber, err)
				}
			}
		case conversations.StateWait, conversations.StateActive:
			if !c.WindowEndsAt.IsZero() && now.After(c.WindowEndsAt) {
				s.closeConversation(c.ID, conversations.OutcomeExpired, "janela de 12h encerrada")
			}
		case conversations.StateHuman:
			if now.Sub(c.UpdatedAt) > humanIdleFor && now.Sub(c.LastInboundAt) > humanIdleFor {
				s.closeConversation(c.ID, "", "encerrada por inatividade")
			}
		}
		if !c.IsLiga() && c.ComplementOrder == 0 && c.OrderNumber > 0 && !c.SentAt.IsZero() && c.Outcome != conversations.OutcomeFailed {
			s.detectComplement(c, all, now)
		}
	}
	if s.PostSale.Outbox {
		s.sweepOutbox(all, now)
	}
}

func unsentStale(c conversations.Conversation, byID map[string]orders.Order, now time.Time) string {
	if now.Sub(c.CreatedAt) > postsale.DraftTTL {
		if c.State == conversations.StateCompose {
			return "mensagem não enviada: a IA não escreveu a tempo"
		}
		return "oferta não enviada: ficou sem aprovação"
	}
	if c.IsLiga() {
		return ""
	}
	if o, ok := byID[c.OrderID]; !ok || orders.EffectiveStatus(o, now) != orders.StatusPaid {
		return "oferta não enviada: pedido já saiu ou mudou de status"
	}
	return ""
}

func orderStillPaid(c conversations.Conversation, byID map[string]orders.Order, now time.Time) bool {
	if c.IsLiga() {
		return true
	}
	o, ok := byID[c.OrderID]
	return ok && orders.EffectiveStatus(o, now) == orders.StatusPaid
}

func (s *Server) detectComplement(c conversations.Conversation, all []orders.Order, now time.Time) {
	for _, o := range all {
		if o.Shipping.JoinOrder != c.OrderNumber || o.CreatedAt.Before(c.CreatedAt) {
			continue
		}
		switch orders.EffectiveStatus(o, now) {
		case orders.StatusPaid, orders.StatusShipped, orders.StatusDelivered:
		default:
			continue
		}
		_, err := s.PostSale.Conversations.Transact(c.ID, func(cv *conversations.Conversation) error {
			cv.ComplementOrder = o.Number
			cv.ComplementBRL = o.Totals.ItemsBRL
			cv.Outcome = conversations.OutcomeComplemented
			cv.Messages = append(cv.Messages, conversations.NewMessage(conversations.RoleSystem, fmt.Sprintf("complemento pago: pedido #%d (R$ %s) vai junto com o #%d", o.Number, postsale.FormatBRL(o.Totals.ItemsBRL), c.OrderNumber), conversations.MsgReceived, now))
			if cv.State != conversations.StateHuman {
				cv.State = conversations.StateClosed
			}
			return nil
		})
		if err != nil {
			s.logPostSale("complemento #%d: %v", o.Number, err)
			return
		}
		s.notify("Pós-venda: complemento pago", fmt.Sprintf("Pedido #%d vai junto com o #%d (R$ %s)", o.Number, c.OrderNumber, postsale.FormatBRL(o.Totals.ItemsBRL)))
		return
	}
}

func (s *Server) closeConversation(id, outcome, note string) {
	_, err := s.PostSale.Conversations.Transact(id, func(c *conversations.Conversation) error {
		if note != "" {
			c.Messages = append(c.Messages, conversations.NewMessage(conversations.RoleSystem, note, conversations.MsgReceived, s.psNow()))
		}
		c.Close(outcome)
		return nil
	})
	if err != nil {
		s.logPostSale("encerrar %s: %v", id, err)
	}
}

func (s *Server) conversationLock(id string) *sync.Mutex {
	v, _ := s.postSale.locks.LoadOrStore(id, &sync.Mutex{})
	mu, _ := v.(*sync.Mutex)
	return mu
}

var errWhatsAppOff = errors.New("WhatsApp não configurado")

func (s *Server) sendOffer(ctx context.Context, id string) (conversations.Conversation, error) {
	if !s.canDeliver() {
		return conversations.Conversation{}, errWhatsAppOff
	}
	mu := s.conversationLock(id)
	mu.Lock()
	defer mu.Unlock()
	c, err := s.PostSale.Conversations.Get(id)
	if err != nil {
		return c, err
	}
	if c.State != conversations.StateQueued {
		return c, nil
	}
	first := postsale.FirstName(c.Name)
	now := s.psNow()
	if s.PostSale.Outbox {
		if _, queued := c.QueuedOffer(); queued {
			return c, nil
		}
		return s.PostSale.Conversations.Transact(id, func(cv *conversations.Conversation) error {
			msg := conversations.NewMessage(conversations.RoleAssistant, offerText(*cv), conversations.MsgQueued, now)
			msg.Kind = conversations.KindOffer
			cv.Messages = append(cv.Messages, msg)
			return nil
		})
	}
	waID, sendErr := s.PostSale.WhatsApp.SendTemplate(ctx, c.Phone, s.PostSale.OfferTemplate, s.PostSale.TemplateLang, []string{first, strconv.Itoa(c.OrderNumber), c.Pitch})
	return s.PostSale.Conversations.Transact(id, func(cv *conversations.Conversation) error {
		msg := conversations.NewMessage(conversations.RoleAssistant, renderOffer(first, cv.OrderNumber, cv.Pitch), conversations.MsgSent, now)
		if sendErr != nil {
			msg.Status = conversations.MsgFailed
			msg.Error = sendErr.Error()
			cv.Messages = append(cv.Messages, msg)
			cv.Close(conversations.OutcomeFailed)
			return nil
		}
		msg.WAID = waID
		cv.Messages = append(cv.Messages, msg)
		markOfferSent(cv, now)
		return nil
	})
}

func markOfferSent(cv *conversations.Conversation, at time.Time) {
	cv.State = conversations.StateWait
	cv.SentAt = at
	cv.WindowEndsAt = at.Add(postsale.Window)
}

func trackingDue(all []orders.Order, now time.Time) []orders.Order {
	var due []orders.Order
	for _, o := range all {
		if o.TrackingCode == "" || o.TrackingNotified == o.TrackingCode || o.Shipping.ServiceID == shipping.JoinServiceID {
			continue
		}
		status := orders.EffectiveStatus(o, now)
		if status != orders.StatusShipped && status != orders.StatusDelivered {
			continue
		}
		if o.ShippedAt.IsZero() || now.Sub(o.ShippedAt) > trackingFreshness {
			continue
		}
		if !o.TrackingNotifiedAt.IsZero() && now.Sub(o.TrackingNotifiedAt) < trackingRetry {
			continue
		}
		if len(whatsapp.BrazilE164(o.Buyer.Phone)) < 12 {
			continue
		}
		due = append(due, o)
	}
	return due
}

func renderTracking(o orders.Order) string {
	carrier := strings.TrimSpace(o.Shipping.Company)
	if carrier != "" {
		carrier = " (" + carrier + ")"
	}
	return fmt.Sprintf("Oi %s! Seu pedido #%d da Collecta já foi enviado 📦 Código de rastreio: %s%s. Qualquer dúvida é só chamar aqui!", postsale.FirstName(o.Buyer.Name), o.Number, o.TrackingCode, carrier)
}

func (s *Server) notifyTracking(ctx context.Context, all []orders.Order, now time.Time) {
	if s.PostSale.Outbox || s.PostSale.WhatsApp == nil || s.PostSale.TrackingTemplate == "" || postsale.Quiet(now, saoPaulo) {
		return
	}
	for _, o := range trackingDue(all, now) {
		to := whatsapp.BrazilE164(o.Buyer.Phone)
		code := o.TrackingCode
		_, err := s.PostSale.WhatsApp.SendTemplate(ctx, to, s.PostSale.TrackingTemplate, s.PostSale.TemplateLang, []string{postsale.FirstName(o.Buyer.Name), strconv.Itoa(o.Number), code})
		if _, uerr := s.Checkout.Orders.Update(o.ID, func(ord *orders.Order) {
			ord.TrackingNotifiedAt = now
			if err == nil {
				ord.TrackingNotified = code
			}
		}); uerr != nil {
			s.logPostSale("marcar rastreio #%d: %v", o.Number, uerr)
		}
		if err != nil {
			s.logPostSale("rastreio #%d: %v", o.Number, err)
		}
	}
}

func (s *Server) handleWhatsAppVerify(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if s.PostSale.VerifyToken == "" || q.Get("hub.mode") != "subscribe" || q.Get("hub.verify_token") != s.PostSale.VerifyToken {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	_, _ = io.WriteString(w, q.Get("hub.challenge"))
}

func (s *Server) handleWhatsAppWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := whatsapp.ValidateSignature(body, r.Header.Get("X-Hub-Signature-256"), s.PostSale.AppSecret); err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	inbound, err := whatsapp.ParseWebhook(body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	for _, in := range inbound {
		s.receive(in)
	}
	writeJSON(w, map[string]bool{"ok": true})
}

func matchConversation(all []conversations.Conversation, from string, now time.Time) (conversations.Conversation, bool) {
	key := whatsapp.PhoneKey(from)
	for _, c := range all {
		if whatsapp.PhoneKey(c.Phone) != key || c.SentAt.IsZero() || now.Sub(c.UpdatedAt) > historyMatchFor {
			continue
		}
		if c.Unsent() {
			continue
		}
		if c.State == conversations.StateClosed && c.Outcome == conversations.OutcomeFailed {
			continue
		}
		return c, true
	}
	return conversations.Conversation{}, false
}

func (s *Server) receive(in whatsapp.Inbound) {
	all, err := s.PostSale.Conversations.List()
	if err != nil {
		s.logPostSale("conversas: %v", err)
		return
	}
	c, ok := matchConversation(all, in.From, s.psNow())
	if !ok {
		s.logPostSale("mensagem de %s fora do pós-venda, ignorada", in.From)
		return
	}
	updated, answer, _, err := s.recordInbound(c.ID, in)
	if errors.Is(err, errDuplicate) {
		return
	}
	if err != nil {
		s.logPostSale("registrar mensagem: %v", err)
		return
	}
	if answer {
		go s.runAgent(updated.ID)
	}
}

func (s *Server) recordInbound(id string, in whatsapp.Inbound) (conversations.Conversation, bool, string, error) {
	now := s.psNow()
	answer := false
	handoff := ""
	updated, err := s.PostSale.Conversations.Transact(id, func(cv *conversations.Conversation) error {
		for _, m := range cv.Messages {
			if in.ID != "" && m.WAID == in.ID {
				return errDuplicate
			}
		}
		at := in.At
		if at.IsZero() {
			at = now
		}
		msg := conversations.NewMessage(conversations.RoleCustomer, in.Text, conversations.MsgReceived, at)
		msg.WAID = in.ID
		cv.DropDrafts()
		cv.Messages = append(cv.Messages, msg)
		cv.LastInboundAt = now
		switch {
		case cv.State == conversations.StateHuman:
			handoff = "cliente respondeu"
		case cv.State == conversations.StateClosed || now.After(cv.WindowEndsAt):
			cv.State = conversations.StateHuman
			handoff = "cliente respondeu depois da janela de 12h"
		default:
			cv.State = conversations.StateActive
			answer = true
		}
		return nil
	})
	if err != nil {
		return updated, false, "", err
	}
	if handoff != "" {
		s.notify("Pós-venda: "+handoff, fmt.Sprintf("%s (pedido #%d): %s", postsale.FirstName(updated.Name), updated.OrderNumber, in.Text))
	}
	return updated, answer, handoff, nil
}

var errDuplicate = errors.New("mensagem duplicada")

func (s *Server) runAgent(id string) {
	if s.PostSale.Brain == nil {
		s.handOff(id, "assistente de IA desligado (ANTHROPIC_API_KEY ausente)")
		return
	}
	mu := s.conversationLock(id)
	mu.Lock()
	defer mu.Unlock()
	c, err := s.PostSale.Conversations.Get(id)
	if err != nil || c.State != conversations.StateActive {
		return
	}
	base, err := s.Checkout.Orders.Get(c.OrderID)
	if err != nil {
		s.handOff(id, "pedido do pós-venda não encontrado")
		return
	}
	turns := make([]postsale.Turn, 0, len(c.Messages))
	for _, m := range c.Messages {
		if m.Status == conversations.MsgFailed || m.Status == conversations.MsgDraft {
			continue
		}
		turns = append(turns, postsale.Turn{Role: m.Role, Text: m.Text})
	}
	ctx, cancel := context.WithTimeout(context.Background(), agentTimeout)
	defer cancel()
	reply, err := s.PostSale.Brain.Reply(ctx, postsale.ReplyInput{
		FirstName:   postsale.FirstName(c.Name),
		OrderNumber: c.OrderNumber,
		Bought:      base.Lines,
		Offer:       c.Offer,
		Turns:       turns,
	}, &convTools{s: s, conv: c, order: base})
	if err != nil {
		s.logPostSale("agente #%d: %v", c.OrderNumber, err)
		s.handOff(id, "erro no assistente: "+err.Error())
		return
	}
	if reply.Text != "" {
		if err := s.deliverAgentText(ctx, id, reply.Text); err != nil {
			s.logPostSale("resposta #%d: %v", c.OrderNumber, err)
		}
	}
	if reply.Handoff {
		if current, err := s.PostSale.Conversations.Get(id); err == nil && current.State == conversations.StateActive {
			s.handOff(id, reply.HandoffReason)
		}
	}
}

var errNotActive = errors.New("a conversa saiu do assistente enquanto ele respondia")

func (s *Server) deliverAgentText(ctx context.Context, id, text string) error {
	if !s.PostSale.AutoSend {
		c, err := s.PostSale.Conversations.Transact(id, func(cv *conversations.Conversation) error {
			if cv.State != conversations.StateActive {
				return errNotActive
			}
			cv.Messages = append(cv.Messages, conversations.NewMessage(conversations.RoleAssistant, text, conversations.MsgDraft, s.psNow()))
			return nil
		})
		if err == nil {
			s.notify("Pós-venda: aprovar resposta", fmt.Sprintf("%s (pedido #%d): %s", postsale.FirstName(c.Name), c.OrderNumber, text))
		}
		return err
	}
	c, err := s.PostSale.Conversations.Get(id)
	if err != nil {
		return err
	}
	if c.State != conversations.StateActive {
		return errNotActive
	}
	_, err = s.sendText(ctx, id, conversations.RoleAssistant, text)
	return err
}

func (s *Server) handOff(id, reason string) {
	c, err := s.PostSale.Conversations.Transact(id, func(cv *conversations.Conversation) error {
		cv.State = conversations.StateHuman
		cv.Messages = append(cv.Messages, conversations.NewMessage(conversations.RoleSystem, "IA passou para você: "+reason, conversations.MsgReceived, s.psNow()))
		return nil
	})
	if err != nil {
		s.logPostSale("handoff %s: %v", id, err)
		return
	}
	s.notify("Pós-venda: assumir conversa", fmt.Sprintf("%s (pedido #%d): %s", postsale.FirstName(c.Name), c.OrderNumber, reason))
}

func (s *Server) sendText(ctx context.Context, id, role, text string) (conversations.Conversation, error) {
	if !s.canDeliver() {
		return conversations.Conversation{}, errWhatsAppOff
	}
	if s.PostSale.Outbox {
		return s.PostSale.Conversations.Transact(id, func(cv *conversations.Conversation) error {
			msg := conversations.NewMessage(role, text, conversations.MsgQueued, s.psNow())
			msg.Kind = conversations.KindReply
			cv.Messages = append(cv.Messages, msg)
			return nil
		})
	}
	c, err := s.PostSale.Conversations.Get(id)
	if err != nil {
		return c, err
	}
	if c.LastInboundAt.IsZero() || s.psNow().Sub(c.LastInboundAt) > postsale.ServiceWindow {
		return c, errServiceWindow
	}
	waID, sendErr := s.PostSale.WhatsApp.SendText(ctx, c.Phone, text)
	updated, err := s.PostSale.Conversations.Transact(id, func(cv *conversations.Conversation) error {
		msg := conversations.NewMessage(role, text, conversations.MsgSent, s.psNow())
		msg.WAID = waID
		if sendErr != nil {
			msg.Status = conversations.MsgFailed
			msg.Error = sendErr.Error()
		}
		cv.Messages = append(cv.Messages, msg)
		return nil
	})
	if err != nil {
		return updated, err
	}
	return updated, sendErr
}

var errServiceWindow = errors.New("o cliente não escreve há mais de 24h: o WhatsApp só deixa mandar template agora")

type convTools struct {
	s     *Server
	conv  conversations.Conversation
	order orders.Order
}

type stockView struct {
	SKU        string  `json:"sku"`
	Nome       string  `json:"nome"`
	Detalhes   string  `json:"detalhes"`
	Jogo       string  `json:"jogo"`
	Preco      float64 `json:"preco"`
	Disponivel int     `json:"disponivel"`
	Envelope   bool    `json:"cabe_no_envelope"`
}

func (t *convTools) SearchStock(query string) (string, error) {
	items, err := t.s.stockItems()
	if err != nil {
		return "", err
	}
	exact, similar := searchItems(items, query)
	body, err := json.Marshal(stockSearch{Exatos: exact, Parecidos: similar})
	return string(body), err
}

const searchLimit = 8

type stockSearch struct {
	Exatos    []stockView `json:"exatos"`
	Parecidos []stockView `json:"parecidos"`
}

type orderStatusView struct {
	Numero        int                   `json:"numero"`
	Status        orders.Status         `json:"status"`
	Itens         []postsale.BoughtLine `json:"itens"`
	Frete         string                `json:"frete"`
	Rastreio      string                `json:"rastreio,omitempty"`
	PodeAdicionar string                `json:"pode_adicionar_ate,omitempty"`
	JanelaAberta  bool                  `json:"janela_aberta"`
}

type postSaleLink struct {
	OrderNumber int                      `json:"orderNumber"`
	FirstName   string                   `json:"firstName"`
	ExpiresAt   time.Time                `json:"expiresAt"`
	Lines       []conversations.LinkLine `json:"lines"`
}

type conversationList struct {
	Conversations []conversations.Conversation `json:"conversations"`
	Metrics       conversationMetrics          `json:"metrics"`
	Transport     string                       `json:"transport"`
	AgentSeenAt   time.Time                    `json:"agentSeenAt,omitzero"`
}

func searchItems(items []postsale.Item, query string) ([]stockView, []stockView) {
	inStock := make([]postsale.Item, 0, len(items))
	for _, it := range items {
		if it.Available > 0 {
			inStock = append(inStock, it)
		}
	}
	exact, similar := postsale.Search(inStock, query, searchLimit)
	return stockViews(exact), stockViews(similar)
}

func stockViews(items []postsale.Item) []stockView {
	out := make([]stockView, 0, len(items))
	for _, it := range items {
		out = append(out, stockView{SKU: it.SKU, Nome: it.Name, Detalhes: it.Meta, Jogo: it.Game, Preco: it.UnitBRL, Disponivel: it.Available, Envelope: it.FitsEnvelope()})
	}
	return out
}

func (t *convTools) OrderStatus() (string, error) {
	o, err := t.s.Checkout.Orders.Get(t.order.ID)
	if err != nil {
		return "", err
	}
	c, err := t.s.PostSale.Conversations.Get(t.conv.ID)
	if err != nil {
		return "", err
	}
	view := orderStatusView{
		Numero:   o.Number,
		Status:   orders.EffectiveStatus(o, t.s.psNow()),
		Itens:    postsale.BoughtView(o.Lines),
		Frete:    strings.TrimSpace(o.Shipping.Company + " " + o.Shipping.ServiceName),
		Rastreio: o.TrackingCode,
	}
	if !c.WindowEndsAt.IsZero() {
		view.PodeAdicionar = c.WindowEndsAt.In(saoPaulo).Format("02/01 15:04")
		view.JanelaAberta = t.s.psNow().Before(c.WindowEndsAt)
	}
	body, err := json.Marshal(view)
	return string(body), err
}

func (t *convTools) ComplementLink(lines []conversations.LinkLine) (string, error) {
	if len(lines) == 0 {
		return "", errors.New("nenhum item")
	}
	c, err := t.s.PostSale.Conversations.Get(t.conv.ID)
	if err != nil {
		return "", err
	}
	if c.WindowEndsAt.IsZero() || t.s.psNow().After(c.WindowEndsAt) {
		return "", errors.New("a janela para juntar com o pedido já fechou: chame o humano")
	}
	o, err := t.s.Checkout.Orders.Get(c.OrderID)
	if err != nil {
		return "", err
	}
	if orders.EffectiveStatus(o, t.s.psNow()) != orders.StatusPaid {
		return "", errors.New("o pedido já foi enviado ou mudou de status: chame o humano")
	}
	items, err := t.s.stockItems()
	if err != nil {
		return "", err
	}
	bySKU := make(map[string]postsale.Item, len(items))
	for _, it := range items {
		bySKU[it.SKU] = it
	}
	merged := make([]conversations.LinkLine, 0, len(lines))
	pos := make(map[string]int, len(lines))
	for _, l := range lines {
		if l.Qty <= 0 {
			return "", fmt.Errorf("quantidade inválida para %s", l.SKU)
		}
		if i, ok := pos[l.SKU]; ok {
			merged[i].Qty += l.Qty
			continue
		}
		pos[l.SKU] = len(merged)
		merged = append(merged, l)
	}
	for _, l := range merged {
		it, ok := bySKU[l.SKU]
		if !ok {
			return "", fmt.Errorf("sku %s não está à venda", l.SKU)
		}
		if !it.FitsEnvelope() {
			return "", fmt.Errorf("%s não cabe no envelope: chame o humano para combinar o frete", it.Name)
		}
		if l.Qty > it.Available {
			return "", fmt.Errorf("%s: só %d disponível", it.Name, it.Available)
		}
	}
	lines = merged
	updated, err := t.s.PostSale.Conversations.Transact(c.ID, func(cv *conversations.Conversation) error {
		if cv.LinkToken == "" {
			cv.LinkToken = conversations.NewID()
		}
		cv.LinkLines = lines
		return nil
	})
	if err != nil {
		return "", err
	}
	return strings.TrimRight(t.s.Checkout.SiteURL, "/") + "/complemento/" + updated.LinkToken, nil
}

func (s *Server) handlePostSaleLink(w http.ResponseWriter, r *http.Request) {
	if !s.requireStorefront(w, r) {
		return
	}
	token := r.PathValue("token")
	all, err := s.PostSale.Conversations.List()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	now := s.psNow()
	for _, c := range all {
		if c.LinkToken == "" || c.LinkToken != token {
			continue
		}
		w.Header().Set("Cache-Control", "no-store")
		o, err := s.Checkout.Orders.Get(c.OrderID)
		if err != nil || c.WindowEndsAt.IsZero() || now.After(c.WindowEndsAt) || orders.EffectiveStatus(o, now) != orders.StatusPaid {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusGone)
			_ = json.NewEncoder(w).Encode(struct {
				Error       string `json:"error"`
				OrderNumber int    `json:"orderNumber"`
			}{"expired", c.OrderNumber})
			return
		}
		writeJSON(w, postSaleLink{OrderNumber: c.OrderNumber, FirstName: postsale.FirstName(c.Name), ExpiresAt: c.WindowEndsAt, Lines: c.LinkLines})
		return
	}
	http.Error(w, "not found", http.StatusNotFound)
}

type conversationMetrics struct {
	OffersSent    int     `json:"offersSent"`
	Replies       int     `json:"replies"`
	Complements   int     `json:"complements"`
	RevenueBRL    float64 `json:"revenueBRL"`
	PendingDrafts int     `json:"pendingDrafts"`
}

func metricsOf(all []conversations.Conversation) conversationMetrics {
	var m conversationMetrics
	for _, c := range all {
		if !c.SentAt.IsZero() {
			m.OffersSent++
		}
		for _, msg := range c.Messages {
			if msg.Role == conversations.RoleCustomer {
				m.Replies++
				break
			}
		}
		if c.ComplementOrder > 0 {
			m.Complements++
			m.RevenueBRL += c.ComplementBRL
		}
		if _, ok := c.PendingDraft(); ok || c.State == conversations.StateDraft {
			m.PendingDrafts++
		}
	}
	m.RevenueBRL = round2(m.RevenueBRL)
	return m
}

func (s *Server) handleConversationsList(w http.ResponseWriter, r *http.Request) {
	all, err := s.PostSale.Conversations.List()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	list := conversationList{Conversations: all, Metrics: metricsOf(all), Transport: "cloud"}
	if s.PostSale.Outbox {
		list.Transport = "openbot"
		list.AgentSeenAt = s.agentSeenAt()
	}
	writeJSON(w, list)
}

func decodeText(r *http.Request) (string, error) {
	var in struct {
		Text string `json:"text"`
	}
	if r.ContentLength == 0 {
		return "", nil
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	return strings.TrimSpace(in.Text), nil
}

func (s *Server) writeConversation(w http.ResponseWriter, c conversations.Conversation, err error) {
	if errors.Is(err, conversations.ErrNotFound) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	writeJSON(w, c)
}

func (s *Server) handleConversationApprove(w http.ResponseWriter, r *http.Request) {
	text, err := decodeText(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	id := r.PathValue("id")
	mu := s.conversationLock(id)
	mu.Lock()
	c, err := s.PostSale.Conversations.Get(id)
	mu.Unlock()
	if err != nil {
		s.writeConversation(w, c, err)
		return
	}
	if c.State == conversations.StateDraft {
		if text != "" && c.Opening != "" {
			text = postsale.CleanOpening(text)
		} else if text != "" {
			text = postsale.CleanPitch(text)
		}
		c, err = s.PostSale.Conversations.Transact(id, func(cv *conversations.Conversation) error {
			if cv.State != conversations.StateDraft {
				return errors.New("a oferta já foi aprovada")
			}
			switch {
			case text == "":
			case cv.Opening != "":
				cv.Opening = text
			default:
				cv.Pitch = text
			}
			cv.State = conversations.StateQueued
			return nil
		})
		if err == nil && !postsale.Quiet(s.psNow(), saoPaulo) {
			c, err = s.sendOffer(r.Context(), id)
		}
		s.writeConversation(w, c, err)
		return
	}
	i, ok := c.PendingDraft()
	if !ok {
		http.Error(w, "nada para aprovar", http.StatusConflict)
		return
	}
	if text == "" {
		text = c.Messages[i].Text
	}
	if !s.PostSale.Outbox && (c.LastInboundAt.IsZero() || s.psNow().Sub(c.LastInboundAt) > postsale.ServiceWindow) {
		http.Error(w, errServiceWindow.Error(), http.StatusConflict)
		return
	}
	if _, err := s.PostSale.Conversations.Transact(id, func(cv *conversations.Conversation) error {
		if j, ok := cv.PendingDraft(); ok {
			cv.Messages = append(cv.Messages[:j], cv.Messages[j+1:]...)
		}
		return nil
	}); err != nil {
		s.writeConversation(w, c, err)
		return
	}
	c, err = s.sendText(r.Context(), id, conversations.RoleAssistant, text)
	s.writeConversation(w, c, err)
}

func (s *Server) handleConversationDiscard(w http.ResponseWriter, r *http.Request) {
	c, err := s.PostSale.Conversations.Transact(r.PathValue("id"), func(cv *conversations.Conversation) error {
		if cv.Unsent() {
			cv.Messages = append(cv.Messages, conversations.NewMessage(conversations.RoleSystem, "oferta descartada", conversations.MsgReceived, s.psNow()))
			cv.Close(conversations.OutcomeSkipped)
			return nil
		}
		i, ok := cv.PendingDraft()
		if !ok {
			return errors.New("nada para descartar")
		}
		cv.Messages = append(cv.Messages[:i], cv.Messages[i+1:]...)
		return nil
	})
	s.writeConversation(w, c, err)
}

func (s *Server) handleConversationReply(w http.ResponseWriter, r *http.Request) {
	text, err := decodeText(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if text == "" {
		http.Error(w, "mensagem vazia", http.StatusBadRequest)
		return
	}
	id := r.PathValue("id")
	mu := s.conversationLock(id)
	mu.Lock()
	defer mu.Unlock()
	c, err := s.PostSale.Conversations.Get(id)
	if err != nil {
		s.writeConversation(w, c, err)
		return
	}
	if c.Unsent() {
		http.Error(w, "a oferta ainda não foi enviada: aprove ou descarte primeiro", http.StatusConflict)
		return
	}
	c, err = s.sendText(r.Context(), id, conversations.RoleOwner, text)
	if err != nil {
		s.writeConversation(w, c, err)
		return
	}
	c, err = s.PostSale.Conversations.Transact(id, func(cv *conversations.Conversation) error {
		cv.DropDrafts()
		cv.State = conversations.StateHuman
		return nil
	})
	s.writeConversation(w, c, err)
}

func (s *Server) handleConversationRelease(w http.ResponseWriter, r *http.Request) {
	c, err := s.PostSale.Conversations.Transact(r.PathValue("id"), func(cv *conversations.Conversation) error {
		if cv.State != conversations.StateHuman {
			return errors.New("a conversa não está com você")
		}
		if cv.WindowEndsAt.IsZero() || s.psNow().After(cv.WindowEndsAt) {
			return errors.New("a janela de 12h já fechou: siga pelo atendimento manual")
		}
		cv.State = conversations.StateActive
		cv.Messages = append(cv.Messages, conversations.NewMessage(conversations.RoleSystem, "conversa devolvida para a IA", conversations.MsgReceived, s.psNow()))
		return nil
	})
	s.writeConversation(w, c, err)
}

func (s *Server) handleConversationClose(w http.ResponseWriter, r *http.Request) {
	c, err := s.PostSale.Conversations.Transact(r.PathValue("id"), func(cv *conversations.Conversation) error {
		cv.DropDrafts()
		cv.Messages = append(cv.Messages, conversations.NewMessage(conversations.RoleSystem, "conversa encerrada", conversations.MsgReceived, s.psNow()))
		cv.Close("")
		return nil
	})
	s.writeConversation(w, c, err)
}

func (s *Server) postSaleRoutes(mux *http.ServeMux) {
	if s.PostSale.Outbox {
		s.agentRoutes(mux)
	} else {
		mux.HandleFunc("GET /api/whatsapp/webhook", s.handleWhatsAppVerify)
		mux.HandleFunc("POST /api/whatsapp/webhook", s.handleWhatsAppWebhook)
	}
	mux.HandleFunc("GET /api/postsale/link/{token}", s.handlePostSaleLink)
	mux.HandleFunc("POST /api/postsale/liga-orders", s.adminOnly(s.handleLigaOrders))
	mux.HandleFunc("GET /api/conversations", s.handleConversationsList)
	mux.HandleFunc("POST /api/conversations/{id}/approve", s.handleConversationApprove)
	mux.HandleFunc("POST /api/conversations/{id}/discard", s.handleConversationDiscard)
	mux.HandleFunc("POST /api/conversations/{id}/reply", s.handleConversationReply)
	mux.HandleFunc("POST /api/conversations/{id}/release", s.handleConversationRelease)
	mux.HandleFunc("POST /api/conversations/{id}/close", s.handleConversationClose)
}
