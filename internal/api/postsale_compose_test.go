package api

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"opdeals/internal/conversations"
	"opdeals/internal/orders"
	"opdeals/internal/postsale"
)

func (f *postSaleFixture) paidOrderWith(t *testing.T, edit func(*orders.Order)) orders.Order {
	t.Helper()
	o := f.paidOrder(t, false)
	o, err := f.s.Checkout.Orders.Update(o.ID, edit)
	if err != nil {
		t.Fatal(err)
	}
	return o
}

func (f *postSaleFixture) conversations(t *testing.T) []conversations.Conversation {
	t.Helper()
	all, err := f.s.PostSale.Conversations.List()
	if err != nil {
		t.Fatal(err)
	}
	return all
}

type composeList struct {
	Items []composeItem `json:"items"`
}

func (f *postSaleFixture) composeList(t *testing.T) composeList {
	t.Helper()
	rec := f.do(t, http.MethodGet, "/api/postsale/agent/compose", nil, agentAuth())
	if rec.Code != http.StatusOK {
		t.Fatalf("compose list %d %s", rec.Code, rec.Body.String())
	}
	return decodeBody[composeList](t, rec)
}

type ligaImport struct {
	Orders []ligaOrderIn `json:"orders"`
}

type ligaOrderWithCPF struct {
	ligaOrderIn
	CPF string `json:"cpf"`
}

func ligaOrder(code string, paidAt time.Time) ligaOrderIn {
	return ligaOrderIn{
		Code:      code,
		FirstName: "Ana Paula",
		Phone:     "(62) 99827-4848",
		PaidAt:    paidAt,
		Items:     []conversations.Item{{Name: "Shanks (OP09-004)", Qty: 1}},
	}
}

func (f *postSaleFixture) importLiga(t *testing.T, orders ...ligaOrderIn) ligaImportResult {
	t.Helper()
	rec := f.do(t, http.MethodPost, "/api/postsale/liga-orders", ligaImport{Orders: orders}, admin())
	if rec.Code != http.StatusOK {
		t.Fatalf("liga import %d %s", rec.Code, rec.Body.String())
	}
	return decodeBody[ligaImportResult](t, rec)
}

func TestOpenBotSiteOrderWithoutOptInGetsOneComposeAfterDelay(t *testing.T) {
	f := openBotServer(t)
	o := f.paidOrderWith(t, func(o *orders.Order) { o.PaidAt = f.now.Add(-10 * time.Minute) })
	f.s.postSaleTick(context.Background())
	if all := f.conversations(t); len(all) != 0 {
		t.Fatalf("conversation before 15 min: %+v", all)
	}
	f.now = f.now.Add(6 * time.Minute)
	f.s.postSaleTick(context.Background())
	f.s.postSaleTick(context.Background())
	c := f.only(t)
	if c.OrderID != o.ID || c.State != conversations.StateCompose || c.Channel != conversations.ChannelSite || c.Offer != nil || c.Pitch != "" || c.Pickup {
		t.Fatalf("conversation %+v", c)
	}
	if len(c.Items) != 1 || c.Items[0].Name != "Teach" || c.Items[0].Qty != 2 {
		t.Fatalf("items %+v", c.Items)
	}
	f.paidOrder(t, true)
	f.s.postSaleTick(context.Background())
	if all := f.conversations(t); len(all) != 1 {
		t.Fatalf("second order within cooldown got a conversation: %d", len(all))
	}
}

func TestOpenBotPickupIncludedJoinExcluded(t *testing.T) {
	f := openBotServer(t)
	f.paidOrderWith(t, func(o *orders.Order) { o.Shipping.ServiceID = -2 })
	f.s.postSaleTick(context.Background())
	if all := f.conversations(t); len(all) != 0 {
		t.Fatalf("join order got a conversation: %+v", all)
	}
	pickup := f.paidOrderWith(t, func(o *orders.Order) { o.Shipping.ServiceID = -1 })
	f.s.postSaleTick(context.Background())
	if c := f.only(t); c.OrderID != pickup.ID || !c.Pickup || c.State != conversations.StateCompose {
		t.Fatalf("pickup conversation %+v", c)
	}
	list := f.composeList(t)
	if len(list.Items) != 1 || !list.Items[0].Pickup {
		t.Fatalf("compose list %+v", list)
	}
}

func TestCloudModeKeepsOptInAndShippingRule(t *testing.T) {
	f := postSaleServer(t)
	f.paidOrderWith(t, func(o *orders.Order) { o.Shipping.ServiceID = -1; o.MarketingOptIn = true })
	f.paidOrder(t, false)
	f.s.postSaleTick(context.Background())
	if all := f.conversations(t); len(all) != 0 {
		t.Fatalf("cloud mode drafted without opt-in or for pickup: %+v", all)
	}
	if rec := f.do(t, http.MethodPost, "/api/postsale/liga-orders", ligaImport{Orders: []ligaOrderIn{ligaOrder("1", f.now)}}, admin()); rec.Code != http.StatusConflict {
		t.Fatalf("liga import in cloud mode %d", rec.Code)
	}
}

func TestLigaImportCreatesComposeAndDedups(t *testing.T) {
	f := openBotServer(t)
	body := ligaImport{Orders: []ligaOrderIn{ligaOrder("12376472", f.now.Add(-2*time.Hour))}}
	if rec := f.do(t, http.MethodPost, "/api/postsale/liga-orders", body, nil); rec.Code != http.StatusUnauthorized {
		t.Fatalf("no admin token %d", rec.Code)
	}
	if rec := f.do(t, http.MethodPost, "/api/postsale/liga-orders", body, agentAuth()); rec.Code != http.StatusUnauthorized {
		t.Fatalf("agent token accepted %d", rec.Code)
	}
	withCPF := struct {
		Orders []ligaOrderWithCPF `json:"orders"`
	}{Orders: []ligaOrderWithCPF{{ligaOrderIn: ligaOrder("999", f.now), CPF: "52998224725"}}}
	if rec := f.do(t, http.MethodPost, "/api/postsale/liga-orders", withCPF, admin()); rec.Code != http.StatusBadRequest {
		t.Fatalf("cpf field accepted %d", rec.Code)
	}
	old := ligaOrder("12370000", f.now.Add(-25*time.Hour))
	old.Phone = "(11) 98888-7777"
	if got := f.importLiga(t, ligaOrder("12376472", f.now.Add(-2*time.Hour)), old); got.Created != 1 || got.Skipped != 1 {
		t.Fatalf("first import %+v", got)
	}
	if got := f.importLiga(t, ligaOrder("12376472", f.now.Add(-2*time.Hour))); got.Created != 0 || got.Skipped != 1 {
		t.Fatalf("dedup %+v", got)
	}
	c := f.only(t)
	if c.Channel != conversations.ChannelLiga || c.OrderID != "liga:12376472" || c.OrderNumber != 12376472 || c.Phone != "5562998274848" || c.Name != "Ana" || c.State != conversations.StateCompose {
		t.Fatalf("liga conversation %+v", c)
	}
	if len(c.Items) != 1 || c.Items[0].Name != "Shanks (OP09-004)" || c.Items[0].Qty != 1 {
		t.Fatalf("items %+v", c.Items)
	}
	list := f.composeList(t)
	if len(list.Items) != 1 || list.Items[0].Channel != "liga" || !strings.Contains(list.Items[0].CanAddHow, "Adicionar a um pedido já existente") {
		t.Fatalf("compose list %+v", list)
	}
	f.s.postSaleTick(context.Background())
	if c = f.only(t); c.State != conversations.StateCompose {
		t.Fatalf("sweep closed a liga conversation without order feed: %+v", c)
	}
}

func TestLigaImportCarriesShippedToCompose(t *testing.T) {
	f := openBotServer(t)
	shipped := ligaOrder("12380950", f.now.Add(-time.Hour))
	shipped.Shipped = true
	waiting := ligaOrder("12380826", f.now.Add(-time.Hour))
	waiting.Phone = "(11) 98888-7777"
	if got := f.importLiga(t, shipped, waiting); got.Created != 2 {
		t.Fatalf("import %+v", got)
	}
	byOrder := map[int]bool{}
	for _, it := range f.composeList(t).Items {
		byOrder[it.OrderNumber] = it.Shipped
	}
	if len(byOrder) != 2 || !byOrder[12380950] || byOrder[12380826] {
		t.Fatalf("shipped flags %+v", byOrder)
	}
}

func TestComposeFlowDraftApproveClaimDeliver(t *testing.T) {
	f := openBotServer(t)
	f.paidOrder(t, false)
	f.s.postSaleTick(context.Background())
	list := f.composeList(t)
	if len(list.Items) != 1 || list.Items[0].Channel != "site" || list.Items[0].FirstName != "Ana" || len(list.Items[0].Items) != 1 || list.Items[0].Items[0].Name != "Teach" {
		t.Fatalf("compose list %+v", list)
	}
	id := list.Items[0].ConversationID
	path := "/api/postsale/agent/conversations/" + id + "/compose"
	for _, bad := range []string{"Oi Ana — valeu!", "Oi Ana – valeu!", "", strings.Repeat("a", postsale.MaxOpeningRunes+1)} {
		if rec := f.agentPost(t, path, map[string]string{"text": bad}); rec.Code != http.StatusBadRequest {
			t.Fatalf("bad text %q accepted: %d", bad, rec.Code)
		}
	}
	reply := decodeBody[map[string]string](t, f.agentPost(t, path, map[string]string{"text": composedText}))
	if reply["status"] != "draft" {
		t.Fatalf("compose %+v", reply)
	}
	if rec := f.agentPost(t, path, map[string]string{"text": composedText}); rec.Code != http.StatusConflict {
		t.Fatalf("second compose %d", rec.Code)
	}
	c := f.only(t)
	if c.State != conversations.StateDraft || c.Opening != composedText || len(c.Messages) != 0 {
		t.Fatalf("draft %+v", c)
	}
	if got := f.composeList(t); len(got.Items) != 0 {
		t.Fatalf("written conversation still listed: %+v", got)
	}
	if claim := f.claim(t); len(claim.Items) != 0 {
		t.Fatalf("draft claimable before approval: %+v", claim)
	}
	if rec := f.do(t, http.MethodPost, "/api/conversations/"+c.ID+"/approve", nil, admin()); rec.Code != http.StatusOK {
		t.Fatalf("approve %d %s", rec.Code, rec.Body.String())
	}
	claim := f.claim(t)
	if len(claim.Items) != 1 || claim.Items[0].Kind != conversations.KindOffer || claim.Items[0].Text != composedText {
		t.Fatalf("claim %+v", claim)
	}
	if rec := f.agentPost(t, "/api/postsale/agent/outbox/"+claim.Items[0].ID+"/delivered", nil); rec.Code != http.StatusOK {
		t.Fatalf("delivered %d", rec.Code)
	}
	c = f.only(t)
	if c.State != conversations.StateWait || !c.WindowEndsAt.Equal(f.now.Add(12*time.Hour)) || c.Messages[0].Text != composedText {
		t.Fatalf("after delivery %+v", c)
	}
}

func TestComposeOwnerEditKeepsLineBreaks(t *testing.T) {
	f := openBotServer(t)
	f.paidOrder(t, false)
	c := f.composed(t)
	edited := "Oi Ana! Obrigado 🩷\nQuer mais alguma coisa no pacote?"
	if rec := f.do(t, http.MethodPost, "/api/conversations/"+c.ID+"/approve", map[string]string{"text": edited}, admin()); rec.Code != http.StatusOK {
		t.Fatalf("approve %d %s", rec.Code, rec.Body.String())
	}
	if claim := f.claim(t); len(claim.Items) != 1 || claim.Items[0].Text != edited {
		t.Fatalf("claim %+v", claim)
	}
}

func TestComposeAutoSendQueuesStraightAway(t *testing.T) {
	f := openBotServer(t)
	f.s.PostSale.AutoSend = true
	f.paidOrder(t, false)
	c := f.composed(t)
	if c.State != conversations.StateQueued {
		t.Fatalf("state %s", c.State)
	}
	if i, ok := c.QueuedOffer(); !ok || c.Messages[i].Text != composedText {
		t.Fatalf("queued offer %+v", c.Messages)
	}
}

func TestLigaConversationDeliversAndRefusesLink(t *testing.T) {
	f := openBotServer(t)
	f.importLiga(t, ligaOrder("12376472", f.now.Add(-time.Hour)))
	c := f.only(t)
	if rec := f.agentPost(t, "/api/postsale/agent/conversations/"+c.ID+"/compose", map[string]string{"text": composedText}); rec.Code != http.StatusOK {
		t.Fatalf("compose %d %s", rec.Code, rec.Body.String())
	}
	if rec := f.do(t, http.MethodPost, "/api/conversations/"+c.ID+"/approve", nil, admin()); rec.Code != http.StatusOK {
		t.Fatalf("approve %d %s", rec.Code, rec.Body.String())
	}
	claim := f.claim(t)
	if len(claim.Items) != 1 || claim.Items[0].Text != composedText || claim.Items[0].Phone != "5562998274848" {
		t.Fatalf("liga claim %+v", claim)
	}
	f.agentPost(t, "/api/postsale/agent/outbox/"+claim.Items[0].ID+"/delivered", nil)
	inbound := struct {
		Messages []agentTurn `json:"messages"`
	}{Messages: []agentTurn{{Text: "quero sim, tem mais Shanks?"}}}
	resp := decodeBody[agentInboundResponse](t, f.agentPost(t, "/api/postsale/agent/conversations/"+c.ID+"/inbound", inbound))
	if resp.Action != "answer" || resp.Context == nil || resp.Context.Channel != "liga" || len(resp.Context.Bought) != 1 || resp.Context.Bought[0].Nome != "Shanks (OP09-004)" || resp.Context.Status != "" {
		t.Fatalf("liga inbound %+v", resp)
	}
	rec := f.agentPost(t, "/api/postsale/agent/conversations/"+c.ID+"/link", struct {
		Itens []conversations.LinkLine `json:"itens"`
	}{Itens: []conversations.LinkLine{{SKU: parallelSKU, Qty: 1}}})
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "Adicionar a um pedido já existente") {
		t.Fatalf("liga link %d %s", rec.Code, rec.Body.String())
	}
}

func TestComposeExpiresAfterDraftTTL(t *testing.T) {
	f := openBotServer(t)
	f.paidOrder(t, false)
	f.s.postSaleTick(context.Background())
	other := ligaOrder("555", f.now)
	other.Phone = "(21) 98888-1111"
	f.importLiga(t, other)
	all := f.conversations(t)
	if len(all) != 2 {
		t.Fatalf("conversations %d", len(all))
	}
	for _, c := range all {
		if _, err := f.s.PostSale.Conversations.Transact(c.ID, func(cv *conversations.Conversation) error {
			cv.CreatedAt = f.now
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	f.s.postSaleTick(context.Background())
	if list := f.composeList(t); len(list.Items) != 2 {
		t.Fatalf("compose list before TTL %+v", list)
	}
	f.now = f.now.Add(postsale.DraftTTL + time.Minute)
	f.s.postSaleTick(context.Background())
	for _, c := range f.conversations(t) {
		if c.State != conversations.StateClosed || c.Outcome != conversations.OutcomeSkipped {
			t.Fatalf("not expired %+v", c)
		}
	}
	if list := f.composeList(t); len(list.Items) != 0 {
		t.Fatalf("expired compose listed %+v", list)
	}
}
