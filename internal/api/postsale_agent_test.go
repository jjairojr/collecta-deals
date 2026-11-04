package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"opdeals/internal/conversations"
	"opdeals/internal/orders"
)

const testAgentToken = "agent-token-0123456789abcdef0123456789"

func openBotServer(t *testing.T) *postSaleFixture {
	t.Helper()
	f := postSaleServer(t)
	f.s.PostSale.WhatsApp = nil
	f.s.PostSale.Outbox = true
	f.s.PostSale.AgentToken = testAgentToken
	return f
}

func agentAuth() map[string]string {
	return map[string]string{"Authorization": "Bearer " + testAgentToken}
}

func decodeBody[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var out T
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode %d %s: %v", rec.Code, rec.Body.String(), err)
	}
	return out
}

func (f *postSaleFixture) claim(t *testing.T) outboxClaim {
	t.Helper()
	rec := f.do(t, http.MethodPost, "/api/postsale/agent/outbox/claim", nil, agentAuth())
	if rec.Code != http.StatusOK {
		t.Fatalf("claim %d %s", rec.Code, rec.Body.String())
	}
	return decodeBody[outboxClaim](t, rec)
}

func (f *postSaleFixture) agentPost(t *testing.T, path string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	return f.do(t, http.MethodPost, path, body, agentAuth())
}

const composedText = "Oi Ana! Valeu demais pela compra 💙\nQuer que eu adicione mais alguma coisa antes de enviar? Vai sem frete extra, se não quiser, tranquilo!"

func (f *postSaleFixture) composed(t *testing.T) conversations.Conversation {
	t.Helper()
	f.s.postSaleTick(context.Background())
	c := f.only(t)
	if c.State != conversations.StateCompose {
		t.Fatalf("state %s", c.State)
	}
	if rec := f.agentPost(t, "/api/postsale/agent/conversations/"+c.ID+"/compose", map[string]string{"text": composedText}); rec.Code != http.StatusOK {
		t.Fatalf("compose %d %s", rec.Code, rec.Body.String())
	}
	return f.only(t)
}

func (f *postSaleFixture) approvedOffer(t *testing.T) conversations.Conversation {
	t.Helper()
	f.paidOrder(t, true)
	c := f.composed(t)
	if rec := f.do(t, http.MethodPost, "/api/conversations/"+c.ID+"/approve", nil, admin()); rec.Code != http.StatusOK {
		t.Fatalf("approve %d %s", rec.Code, rec.Body.String())
	}
	return f.only(t)
}

func (f *postSaleFixture) deliveredOffer(t *testing.T) conversations.Conversation {
	t.Helper()
	f.approvedOffer(t)
	items := f.claim(t).Items
	if len(items) != 1 || items[0].Kind != conversations.KindOffer {
		t.Fatalf("claim %+v", items)
	}
	if rec := f.agentPost(t, "/api/postsale/agent/outbox/"+items[0].ID+"/delivered", nil); rec.Code != http.StatusOK {
		t.Fatalf("delivered %d %s", rec.Code, rec.Body.String())
	}
	return f.only(t)
}

func (f *postSaleFixture) shippedOrder(t *testing.T, code string) orders.Order {
	t.Helper()
	o := f.paidOrder(t, false)
	o, err := f.s.Checkout.Orders.Update(o.ID, func(o *orders.Order) {
		o.Status = orders.StatusShipped
		o.ShippedAt = f.now.Add(-time.Hour)
		o.TrackingCode = code
		o.Shipping.Company = "Correios"
	})
	if err != nil {
		t.Fatal(err)
	}
	return o
}

func TestAgentTokenGate(t *testing.T) {
	f := openBotServer(t)
	path := "/api/postsale/agent/watch"
	for name, headers := range map[string]map[string]string{
		"none":   nil,
		"wrong":  {"Authorization": "Bearer nope"},
		"admin":  admin(),
		"scheme": {"Authorization": testAgentToken},
	} {
		if rec := f.do(t, http.MethodGet, path, nil, headers); rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s: %d", name, rec.Code)
		}
	}
	if rec := f.do(t, http.MethodGet, path, nil, agentAuth()); rec.Code != http.StatusOK {
		t.Fatalf("valid token %d", rec.Code)
	}
	if rec := f.do(t, http.MethodPost, "/api/whatsapp/webhook", nil, nil); rec.Code != http.StatusNotFound {
		t.Fatalf("webhook in openbot mode %d", rec.Code)
	}
	f.s.PostSale.AgentToken = ""
	if rec := f.do(t, http.MethodGet, path, nil, agentAuth()); rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("empty token %d", rec.Code)
	}
	cloud := postSaleServer(t)
	if rec := cloud.do(t, http.MethodGet, path, nil, agentAuth()); rec.Code != http.StatusNotFound {
		t.Fatalf("agent route in cloud mode %d", rec.Code)
	}
}

func TestOpenBotApproveQueuesOfferAndDeliveryOpensWindow(t *testing.T) {
	f := openBotServer(t)
	c := f.approvedOffer(t)
	if c.State != conversations.StateQueued {
		t.Fatalf("state %s", c.State)
	}
	if _, ok := c.QueuedOffer(); !ok {
		t.Fatalf("no queued offer: %+v", c.Messages)
	}
	claim := f.claim(t)
	if len(claim.Items) != 1 {
		t.Fatalf("claim %+v", claim)
	}
	item := claim.Items[0]
	if item.Phone != "5511999998888" || item.FirstName != "Ana" || item.Text != composedText {
		t.Fatalf("item %+v", item)
	}
	if again := f.claim(t); len(again.Items) != 0 {
		t.Fatalf("leased item claimed twice: %+v", again.Items)
	}
	for range 2 {
		if rec := f.agentPost(t, "/api/postsale/agent/outbox/"+item.ID+"/delivered", nil); rec.Code != http.StatusOK {
			t.Fatalf("delivered %d %s", rec.Code, rec.Body.String())
		}
	}
	c = f.only(t)
	if c.State != conversations.StateWait || !c.SentAt.Equal(f.now) || !c.WindowEndsAt.Equal(f.now.Add(12*time.Hour)) {
		t.Fatalf("after delivery %+v", c)
	}
	if c.Messages[0].Status != conversations.MsgSent {
		t.Fatalf("message %+v", c.Messages[0])
	}
}

func TestOpenBotOutboxRespectsQuietHours(t *testing.T) {
	f := openBotServer(t)
	f.now = time.Date(2026, 10, 31, 23, 0, 0, 0, saoPaulo)
	o := f.paidOrder(t, true)
	f.shippedOrder(t, "BR111")
	f.s.postSaleTick(context.Background())
	all, err := f.s.PostSale.Conversations.List()
	if err != nil || len(all) != 1 {
		t.Fatalf("conversations %v %v", all, err)
	}
	if rec := f.agentPost(t, "/api/postsale/agent/conversations/"+all[0].ID+"/compose", map[string]string{"text": composedText}); rec.Code != http.StatusOK {
		t.Fatalf("compose %d %s", rec.Code, rec.Body.String())
	}
	c := f.only(t)
	if rec := f.do(t, http.MethodPost, "/api/conversations/"+c.ID+"/approve", nil, admin()); rec.Code != http.StatusOK {
		t.Fatalf("approve %d %s", rec.Code, rec.Body.String())
	}
	if c = f.only(t); c.State != conversations.StateQueued || len(c.Messages) != 0 {
		t.Fatalf("night approval queued a message: %+v", c)
	}
	if claim := f.claim(t); !claim.Quiet || len(claim.Items) != 0 {
		t.Fatalf("night claim %+v", claim)
	}
	f.now = time.Date(2026, 11, 1, 8, 1, 0, 0, saoPaulo)
	if _, err := f.s.Checkout.Orders.Update(o.ID, func(o *orders.Order) { o.PaidAt = f.now.Add(-9 * time.Hour) }); err != nil {
		t.Fatal(err)
	}
	f.s.postSaleTick(context.Background())
	claim := f.claim(t)
	kinds := map[string]int{}
	for _, it := range claim.Items {
		kinds[it.Kind]++
	}
	if claim.Quiet || kinds[conversations.KindOffer] != 1 || kinds["tracking"] != 1 {
		t.Fatalf("morning claim %+v", claim)
	}
}

func TestOpenBotDiscardFailsQueuedOffer(t *testing.T) {
	f := openBotServer(t)
	c := f.approvedOffer(t)
	i, _ := c.QueuedOffer()
	msgID := c.Messages[i].ID
	if rec := f.do(t, http.MethodPost, "/api/conversations/"+c.ID+"/discard", nil, admin()); rec.Code != http.StatusOK {
		t.Fatalf("discard %d", rec.Code)
	}
	if claim := f.claim(t); len(claim.Items) != 0 {
		t.Fatalf("discarded offer claimable: %+v", claim.Items)
	}
	if rec := f.agentPost(t, "/api/postsale/agent/outbox/m."+c.ID+"."+msgID+"/delivered", nil); rec.Code != http.StatusOK {
		t.Fatalf("late delivered %d", rec.Code)
	}
	c = f.only(t)
	if c.State != conversations.StateClosed || c.Outcome != conversations.OutcomeSkipped || c.SentAt.IsZero() {
		t.Fatalf("late delivery reopened or lost: %+v", c)
	}
}

func TestOpenBotFailedOfferGoesBackToQueue(t *testing.T) {
	f := openBotServer(t)
	f.approvedOffer(t)
	item := f.claim(t).Items[0]
	if rec := f.agentPost(t, "/api/postsale/agent/outbox/"+item.ID+"/failed", map[string]interface{}{"error": "busca não achou o contato"}); rec.Code != http.StatusOK {
		t.Fatalf("failed %d %s", rec.Code, rec.Body.String())
	}
	c := f.only(t)
	if c.State != conversations.StateQueued || c.Outcome != "" || c.Messages[0].Status != conversations.MsgQueued || c.Messages[0].Error != "busca não achou o contato" {
		t.Fatalf("after transient failure %+v", c)
	}
	retry := f.claim(t)
	if len(retry.Items) != 1 || retry.Items[0].ID != item.ID {
		t.Fatalf("transient failure not retried: %+v", retry)
	}
}

func TestOpenBotPermanentFailedOfferClosesConversation(t *testing.T) {
	f := openBotServer(t)
	f.approvedOffer(t)
	item := f.claim(t).Items[0]
	if rec := f.agentPost(t, "/api/postsale/agent/outbox/"+item.ID+"/failed", map[string]interface{}{"error": "número sem WhatsApp", "permanent": true}); rec.Code != http.StatusOK {
		t.Fatalf("failed %d %s", rec.Code, rec.Body.String())
	}
	c := f.only(t)
	if c.State != conversations.StateClosed || c.Outcome != conversations.OutcomeFailed || c.Messages[0].Error != "número sem WhatsApp" {
		t.Fatalf("after failure %+v", c)
	}
	if again := f.claim(t); len(again.Items) != 0 {
		t.Fatalf("permanent failure retried: %+v", again)
	}
}

func TestOpenBotTrackingViaOutbox(t *testing.T) {
	f := openBotServer(t)
	o := f.shippedOrder(t, "BR123")
	claim := f.claim(t)
	if len(claim.Items) != 1 || claim.Items[0].Kind != "tracking" || !strings.Contains(claim.Items[0].Text, "BR123 (Correios)") || claim.Items[0].Phone != "5511999998888" {
		t.Fatalf("claim %+v", claim)
	}
	item := claim.Items[0]
	if again := f.claim(t); len(again.Items) != 0 {
		t.Fatalf("tracking claimed twice: %+v", again)
	}
	if rec := f.agentPost(t, "/api/postsale/agent/outbox/"+item.ID+"/failed", map[string]interface{}{"error": "aba travou"}); rec.Code != http.StatusOK {
		t.Fatalf("failed %d", rec.Code)
	}
	f.now = f.now.Add(31 * time.Minute)
	retry := f.claim(t)
	if len(retry.Items) != 1 {
		t.Fatalf("retry %+v", retry)
	}
	if rec := f.agentPost(t, "/api/postsale/agent/outbox/"+retry.Items[0].ID+"/delivered", nil); rec.Code != http.StatusOK {
		t.Fatalf("delivered %d", rec.Code)
	}
	got, err := f.s.Checkout.Orders.Get(o.ID)
	if err != nil || got.TrackingNotified != "BR123" {
		t.Fatalf("not marked %+v %v", got, err)
	}
	f.now = f.now.Add(31 * time.Minute)
	if after := f.claim(t); len(after.Items) != 0 {
		t.Fatalf("delivered tracking came back: %+v", after)
	}
}

func TestOpenBotPermanentTrackingFailureStops(t *testing.T) {
	f := openBotServer(t)
	f.shippedOrder(t, "BR777")
	item := f.claim(t).Items[0]
	f.agentPost(t, "/api/postsale/agent/outbox/"+item.ID+"/failed", map[string]interface{}{"error": "número sem WhatsApp", "permanent": true})
	f.now = f.now.Add(31 * time.Minute)
	if claim := f.claim(t); len(claim.Items) != 0 {
		t.Fatalf("permanent failure retried: %+v", claim)
	}
}

func TestOpenBotInboundAnswerDraftApproveDeliver(t *testing.T) {
	f := openBotServer(t)
	c := f.deliveredOffer(t)
	watch := decodeBody[struct {
		Conversations []watchConversation `json:"conversations"`
	}](t, f.do(t, http.MethodGet, "/api/postsale/agent/watch", nil, agentAuth()))
	if len(watch.Conversations) != 1 || watch.Conversations[0].Phone != "5511999998888" || len(watch.Conversations[0].LastMessages) != 1 {
		t.Fatalf("watch %+v", watch)
	}
	body := map[string]interface{}{"messages": []map[string]interface{}{{"text": "quero sim!", "at": "14:00"}}}
	resp := decodeBody[agentInboundResponse](t, f.agentPost(t, "/api/postsale/agent/conversations/"+c.ID+"/inbound", body))
	if resp.Action != "answer" || resp.Context == nil || resp.Context.OrderNumber != c.OrderNumber || !resp.Context.WindowOpen || len(resp.Context.Bought) != 1 {
		t.Fatalf("inbound %+v", resp)
	}
	if again := decodeBody[agentInboundResponse](t, f.agentPost(t, "/api/postsale/agent/conversations/"+c.ID+"/inbound", body)); again.Action != "none" {
		t.Fatalf("duplicate inbound %+v", again)
	}
	reply := decodeBody[map[string]string](t, f.agentPost(t, "/api/postsale/agent/conversations/"+c.ID+"/reply", map[string]string{"text": "Fechado! Te mando o link."}))
	if reply["status"] != "draft" {
		t.Fatalf("reply %+v", reply)
	}
	if rec := f.do(t, http.MethodPost, "/api/conversations/"+c.ID+"/approve", nil, admin()); rec.Code != http.StatusOK {
		t.Fatalf("approve reply %d %s", rec.Code, rec.Body.String())
	}
	claim := f.claim(t)
	if len(claim.Items) != 1 || claim.Items[0].Kind != conversations.KindReply || claim.Items[0].Text != "Fechado! Te mando o link." {
		t.Fatalf("reply claim %+v", claim)
	}
	f.agentPost(t, "/api/postsale/agent/outbox/"+claim.Items[0].ID+"/delivered", nil)
	c = f.only(t)
	customer := 0
	for _, m := range c.Messages {
		if m.Role == conversations.RoleCustomer {
			customer++
		}
	}
	last := c.Messages[len(c.Messages)-1]
	if customer != 1 || last.Status != conversations.MsgSent || last.Role != conversations.RoleAssistant {
		t.Fatalf("messages %+v", c.Messages)
	}
}

func TestOpenBotInboundAfterWindowHandsOff(t *testing.T) {
	f := openBotServer(t)
	c := f.deliveredOffer(t)
	f.now = f.now.Add(13 * time.Hour)
	f.s.postSaleTick(context.Background())
	resp := decodeBody[agentInboundResponse](t, f.agentPost(t, "/api/postsale/agent/conversations/"+c.ID+"/inbound", map[string]interface{}{"messages": []map[string]interface{}{{"text": "ainda dá?"}}}))
	if resp.Action != "handoff" || resp.Context != nil {
		t.Fatalf("late inbound %+v", resp)
	}
	if c = f.only(t); c.State != conversations.StateHuman {
		t.Fatalf("state %s", c.State)
	}
}

func TestOpenBotOwnerTypedOnPhonePausesAgent(t *testing.T) {
	f := openBotServer(t)
	c := f.deliveredOffer(t)
	echo := map[string]interface{}{"messages": []map[string]interface{}{{"text": "  " + strings.ReplaceAll(c.Messages[0].Text, " ", "  ") + " ", "fromStore": true}}}
	if resp := decodeBody[agentInboundResponse](t, f.agentPost(t, "/api/postsale/agent/conversations/"+c.ID+"/inbound", echo)); resp.Action != "none" || resp.Reason != "nada novo" {
		t.Fatalf("echo %+v", resp)
	}
	if c = f.only(t); c.State != conversations.StateWait {
		t.Fatalf("echo paused the agent: %s", c.State)
	}
	owner := map[string]interface{}{"messages": []map[string]interface{}{{"text": "Oi Ana, aqui é o Jairo da loja", "fromStore": true}}}
	resp := decodeBody[agentInboundResponse](t, f.agentPost(t, "/api/postsale/agent/conversations/"+c.ID+"/inbound", owner))
	if resp.Action != "none" || !strings.Contains(resp.Reason, "pelo celular") {
		t.Fatalf("owner %+v", resp)
	}
	c = f.only(t)
	found := false
	for _, m := range c.Messages {
		found = found || (m.Role == conversations.RoleOwner && m.Text == "Oi Ana, aqui é o Jairo da loja")
	}
	if c.State != conversations.StateHuman || !found {
		t.Fatalf("owner message %+v", c)
	}
}

func TestOpenBotLinkScopedToConversation(t *testing.T) {
	f := openBotServer(t)
	c := f.deliveredOffer(t)
	body := map[string]interface{}{"itens": []map[string]interface{}{{"sku": parallelSKU, "qtd": 1}}}
	link := decodeBody[map[string]string](t, f.agentPost(t, "/api/postsale/agent/conversations/"+c.ID+"/link", body))
	if !strings.HasPrefix(link["url"], "https://site.test/complemento/") {
		t.Fatalf("link %+v", link)
	}
	if rec := f.agentPost(t, "/api/postsale/agent/conversations/"+c.ID+"/handoff", map[string]string{"reason": "pediu desconto"}); rec.Code != http.StatusOK {
		t.Fatalf("handoff %d", rec.Code)
	}
	if rec := f.agentPost(t, "/api/postsale/agent/conversations/"+c.ID+"/link", body); rec.Code != http.StatusConflict {
		t.Fatalf("link after handoff %d", rec.Code)
	}
	if rec := f.agentPost(t, "/api/postsale/agent/conversations/nope/link", body); rec.Code != http.StatusNotFound {
		t.Fatalf("unknown conversation %d", rec.Code)
	}
}

func TestOpenBotOwnerReplyQueuesAndExpires(t *testing.T) {
	f := openBotServer(t)
	c := f.deliveredOffer(t)
	if rec := f.do(t, http.MethodPost, "/api/conversations/"+c.ID+"/reply", map[string]string{"text": "Separei aqui pra você"}, admin()); rec.Code != http.StatusOK {
		t.Fatalf("owner reply %d %s", rec.Code, rec.Body.String())
	}
	c = f.only(t)
	last := c.Messages[len(c.Messages)-1]
	if c.State != conversations.StateHuman || last.Status != conversations.MsgQueued || last.Role != conversations.RoleOwner {
		t.Fatalf("owner reply not queued: %+v", c)
	}
	f.now = f.now.Add(3 * time.Hour)
	f.s.postSaleTick(context.Background())
	c = f.only(t)
	if last = c.Messages[len(c.Messages)-1]; last.Status != conversations.MsgFailed {
		t.Fatalf("stale reply not expired: %+v", last)
	}
	if claim := f.claim(t); len(claim.Items) != 0 {
		t.Fatalf("expired reply claimable: %+v", claim)
	}
}

func TestOpenBotStaleAgentAlertDecision(t *testing.T) {
	f := openBotServer(t)
	start := f.now
	waiting := pendingWork{Outbox: 1, OldestAt: start}
	f.s.alertIfAgentStale(pendingWork{Open: 3}, start.Add(time.Hour))
	if !f.s.postSale.agentAlertAt.IsZero() {
		t.Fatal("alerted with only open conversations")
	}
	f.s.alertIfAgentStale(waiting, start)
	if !f.s.postSale.agentAlertAt.IsZero() {
		t.Fatal("alerted as soon as the item appeared")
	}
	f.s.alertIfAgentStale(waiting, start.Add(16*time.Minute))
	first := f.s.postSale.agentAlertAt
	if first.IsZero() {
		t.Fatal("no alert after 16 min with the item waiting and no agent")
	}
	f.s.alertIfAgentStale(waiting, start.Add(30*time.Minute))
	if !f.s.postSale.agentAlertAt.Equal(first) {
		t.Fatal("alerted twice within the hour")
	}
	f.now = start.Add(80 * time.Minute)
	f.claim(t)
	f.s.alertIfAgentStale(waiting, start.Add(85*time.Minute))
	if !f.s.postSale.agentAlertAt.Equal(first) {
		t.Fatal("alerted although the agent was just seen")
	}
}

func TestOpenBotStaleAlertIgnoresIdleAgentBeforeWork(t *testing.T) {
	f := openBotServer(t)
	f.claim(t)
	arrived := f.now.Add(25 * time.Minute)
	f.s.alertIfAgentStale(pendingWork{Compose: 1, OldestAt: arrived}, arrived.Add(time.Minute))
	if !f.s.postSale.agentAlertAt.IsZero() {
		t.Fatal("an agent idle between heartbeats raised the alert")
	}
	f.s.alertIfAgentStale(pendingWork{Compose: 1, OldestAt: arrived}, arrived.Add(16*time.Minute))
	if f.s.postSale.agentAlertAt.IsZero() {
		t.Fatal("no alert after the work waited 16 min")
	}
}

func (f *postSaleFixture) pending(t *testing.T) pendingWork {
	t.Helper()
	rec := f.do(t, http.MethodGet, "/api/postsale/agent/pending", nil, agentAuth())
	if rec.Code != http.StatusOK {
		t.Fatalf("pending %d %s", rec.Code, rec.Body.String())
	}
	return decodeBody[pendingWork](t, rec)
}

func TestOpenBotPendingCountsWithoutTouchingAgent(t *testing.T) {
	f := openBotServer(t)
	if rec := f.do(t, http.MethodGet, "/api/postsale/agent/pending", nil, admin()); rec.Code != http.StatusUnauthorized {
		t.Fatalf("admin token %d", rec.Code)
	}
	if p := f.pending(t); p != (pendingWork{}) {
		t.Fatalf("empty %+v", p)
	}
	f.paidOrder(t, true)
	f.s.postSaleTick(context.Background())
	p := f.pending(t)
	if p.Compose != 1 || p.Total() != 1 || !p.OldestAt.Equal(f.only(t).CreatedAt) {
		t.Fatalf("compose %+v", p)
	}
	if !f.s.agentSeenAt().IsZero() {
		t.Fatal("pending marked the agent as seen")
	}
	cloud := postSaleServer(t)
	if rec := cloud.do(t, http.MethodGet, "/api/postsale/agent/pending", nil, agentAuth()); rec.Code != http.StatusNotFound {
		t.Fatalf("pending in cloud mode %d", rec.Code)
	}
}

func TestOpenBotPendingFollowsOutboxLifecycle(t *testing.T) {
	f := openBotServer(t)
	f.approvedOffer(t)
	f.shippedOrder(t, "BR555")
	if p := f.pending(t); p.Outbox != 1 || p.Tracking != 1 || p.Compose != 0 || p.Open != 0 {
		t.Fatalf("queued %+v", p)
	}
	items := f.claim(t).Items
	if p := f.pending(t); p.Total() != 0 {
		t.Fatalf("leased items still pending %+v", p)
	}
	for _, it := range items {
		if it.Kind == conversations.KindOffer {
			f.agentPost(t, "/api/postsale/agent/outbox/"+it.ID+"/delivered", nil)
		}
	}
	f.now = f.now.Add(31 * time.Minute)
	p := f.pending(t)
	if p.Outbox != 0 || p.Tracking != 1 || p.Open != 1 {
		t.Fatalf("after delivery %+v", p)
	}
	if !p.OldestAt.Equal(f.now.Add(-time.Minute)) {
		t.Fatalf("tracking retry since %v, now %v", p.OldestAt, f.now)
	}
}

func TestOpenBotPendingQuietHours(t *testing.T) {
	f := openBotServer(t)
	f.now = time.Date(2026, 10, 31, 23, 0, 0, 0, saoPaulo)
	f.paidOrder(t, true)
	f.shippedOrder(t, "BR999")
	f.s.postSaleTick(context.Background())
	if p := f.pending(t); !p.Quiet || p.Total() != 0 {
		t.Fatalf("night %+v", p)
	}
	f.now = time.Date(2026, 11, 1, 8, 1, 0, 0, saoPaulo)
	p := f.pending(t)
	morning := time.Date(2026, 11, 1, 8, 0, 0, 0, saoPaulo)
	if p.Quiet || p.Compose != 1 || p.Tracking != 1 || !p.OldestAt.Equal(morning) {
		t.Fatalf("morning %+v", p)
	}
	f.s.alertIfAgentStale(p, f.now)
	if !f.s.postSale.agentAlertAt.IsZero() {
		t.Fatal("night backlog raised the alert right at 8h")
	}
}

func TestOpenBotConversationsListReportsTransport(t *testing.T) {
	f := openBotServer(t)
	f.claim(t)
	list := decodeBody[conversationList](t, f.do(t, http.MethodGet, "/api/conversations", nil, admin()))
	if list.Transport != "openbot" || !list.AgentSeenAt.Equal(f.now) {
		t.Fatalf("list %+v", list)
	}
	cloud := postSaleServer(t)
	if got := decodeBody[conversationList](t, cloud.do(t, http.MethodGet, "/api/conversations", nil, admin())); got.Transport != "cloud" || !got.AgentSeenAt.IsZero() {
		t.Fatalf("cloud list %+v", got)
	}
}
