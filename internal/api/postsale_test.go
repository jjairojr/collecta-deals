package api

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"opdeals/internal/conversations"
	"opdeals/internal/orders"
	"opdeals/internal/postsale"
	"opdeals/internal/trades"
)

const testAppSecret = "app-secret"

type sentMessage struct {
	to       string
	template string
	params   []string
	text     string
}

type fakeSender struct {
	mu   sync.Mutex
	sent []sentMessage
}

func (f *fakeSender) SendTemplate(_ context.Context, to, name, _ string, params []string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, sentMessage{to: to, template: name, params: params})
	return fmt.Sprintf("wamid.%d", len(f.sent)), nil
}

func (f *fakeSender) SendText(_ context.Context, to, body string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, sentMessage{to: to, text: body})
	return fmt.Sprintf("wamid.%d", len(f.sent)), nil
}

func (f *fakeSender) all() []sentMessage {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]sentMessage(nil), f.sent...)
}

type fakeBrain struct {
	link string
	err  error
}

func (b *fakeBrain) Pitch(_ context.Context, in postsale.PitchInput) (string, string, error) {
	return in.Candidates[0].SKU, "A Teach Parallel combina com a sua", nil
}

func (b *fakeBrain) Reply(_ context.Context, in postsale.ReplyInput, tools postsale.Toolbox) (postsale.Reply, error) {
	if _, err := tools.SearchStock("teach parallel"); err != nil {
		return postsale.Reply{}, err
	}
	b.link, b.err = tools.ComplementLink([]conversations.LinkLine{{SKU: parallelSKU, Qty: 1}})
	if b.err != nil {
		return postsale.Reply{Handoff: true, HandoffReason: b.err.Error()}, nil
	}
	return postsale.Reply{Text: "Fechado! Link: " + b.link}, nil
}

const parallelSKU = "single|onepiece|teach (parallel)|op16-080|parallel|"

type postSaleFixture struct {
	*checkoutFixture
	sender *fakeSender
	now    time.Time
}

func postSaleServer(t *testing.T) *postSaleFixture {
	t.Helper()
	f := checkoutServer(t)
	extra := []trades.Trade{
		{Number: "OP16-080", Name: "Teach (Parallel)", Set: "OP16", Variant: "Parallel", Qty: 1, BuyBRL: 150, BuyDate: "2026-09-01", Status: "holding"},
		{Number: "OP01-001", Name: "Zoro", Set: "OP01", Qty: 1, BuyBRL: 10, BuyDate: "2026-09-01", Status: "holding"},
		{Number: "", Kind: "sealed", Name: "Booster Box OP16", Set: "SEALED", Qty: 1, BuyBRL: 100, BuyDate: "2026-09-01", Status: "holding"},
	}
	listings := map[string]trades.Listing{}
	asks := []float64{300, 40, 200}
	for i, tr := range extra {
		added, err := f.store.Add(tr)
		if err != nil {
			t.Fatal(err)
		}
		listings[added.ID] = trades.Listing{AskBRL: asks[i], Listed: true}
	}
	if _, err := f.store.SetListings(listings); err != nil {
		t.Fatal(err)
	}
	sender := &fakeSender{}
	f.s.PostSale = PostSale{
		Conversations:    conversations.NewStore(filepath.Join(t.TempDir(), "conversations.json")),
		WhatsApp:         sender,
		AppSecret:        testAppSecret,
		VerifyToken:      "verify",
		OfferTemplate:    "collecta_pos_venda",
		TrackingTemplate: "collecta_rastreio",
		TemplateLang:     "pt_BR",
	}
	pf := &postSaleFixture{checkoutFixture: f, sender: sender, now: time.Date(2026, 10, 31, 14, 0, 0, 0, saoPaulo)}
	f.s.postSale.now = func() time.Time { return pf.now }
	return pf
}

func (f *postSaleFixture) paidOrder(t *testing.T, optIn bool) orders.Order {
	t.Helper()
	o, err := f.s.Checkout.Orders.Create(func(map[string]int, []orders.Order) (orders.Order, error) {
		return orders.Order{
			Lines:          []orders.Line{{SKU: teachSKU, Game: "onepiece", Kind: "single", Name: "Teach", Meta: "OP16 · OP16-080", Qty: 2, UnitBRL: 250}},
			Buyer:          orders.Buyer{Name: "Ana Souza", Email: "ana@example.com", Phone: "11999998888", CPF: "52998224725"},
			Shipping:       orders.Shipping{ServiceID: 17, PriceBRL: 12},
			Totals:         orders.Totals{ItemsBRL: 500, ShippingBRL: 12, TotalBRL: 512},
			MarketingOptIn: optIn,
		}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	o, err = f.s.Checkout.Orders.Update(o.ID, func(o *orders.Order) {
		o.Status = orders.StatusPaid
		o.PaidAt = f.now.Add(-time.Hour)
	})
	if err != nil {
		t.Fatal(err)
	}
	return o
}

func (f *postSaleFixture) only(t *testing.T) conversations.Conversation {
	t.Helper()
	all, err := f.s.PostSale.Conversations.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 {
		t.Fatalf("conversations = %d", len(all))
	}
	return all[0]
}

func (f *postSaleFixture) inbound(t *testing.T, waID, text string) *httptest.ResponseRecorder {
	t.Helper()
	body := fmt.Sprintf(`{"entry":[{"changes":[{"field":"messages","value":{"contacts":[{"wa_id":"551199998888","profile":{"name":"Ana"}}],"messages":[{"id":%q,"from":"551199998888","timestamp":"%d","type":"text","text":{"body":%q}}]}}]}]}`, waID, f.now.Unix(), text)
	mac := hmac.New(sha256.New, []byte(testAppSecret))
	mac.Write([]byte(body))
	req := httptest.NewRequest(http.MethodPost, "/api/whatsapp/webhook", strings.NewReader(body))
	req.Header.Set("X-Hub-Signature-256", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	rec := httptest.NewRecorder()
	f.s.Handler().ServeHTTP(rec, req)
	return rec
}

func (f *postSaleFixture) waitFor(t *testing.T, cond func(conversations.Conversation) bool) conversations.Conversation {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if c := f.only(t); cond(c) {
			return c
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("condition never met: %+v", f.only(t))
	return conversations.Conversation{}
}

func admin() map[string]string { return map[string]string{"X-Admin-Token": "admin"} }

func TestPostSaleDraftsOneOfferWithFallbackPitch(t *testing.T) {
	f := postSaleServer(t)
	f.paidOrder(t, false)
	o := f.paidOrder(t, true)
	f.s.postSaleTick(context.Background())
	f.s.postSaleTick(context.Background())
	c := f.only(t)
	if c.OrderID != o.ID || c.State != conversations.StateDraft || c.Phone != "5511999998888" {
		t.Fatalf("conversation %+v", c)
	}
	if c.Offer == nil || c.Offer.SKU != parallelSKU {
		t.Fatalf("offer %+v", c.Offer)
	}
	if !strings.Contains(c.Pitch, "Teach (Parallel)") || !strings.Contains(c.Pitch, "R$ 300,00") || !strings.Contains(c.Pitch, "combina com Teach") {
		t.Fatalf("pitch %q", c.Pitch)
	}
	if len(f.sender.all()) != 0 {
		t.Fatal("nothing should be sent before approval")
	}
}

func TestPostSaleApproveSendsTemplateAndOpensWindow(t *testing.T) {
	f := postSaleServer(t)
	f.paidOrder(t, true)
	f.s.postSaleTick(context.Background())
	c := f.only(t)
	rec := f.do(t, http.MethodPost, "/api/conversations/"+c.ID+"/approve", map[string]string{"text": "Pitch\neditado"}, admin())
	if rec.Code != http.StatusOK {
		t.Fatalf("approve %d %s", rec.Code, rec.Body.String())
	}
	sent := f.sender.all()
	if len(sent) != 1 || sent[0].template != "collecta_pos_venda" || sent[0].to != "5511999998888" {
		t.Fatalf("sent %+v", sent)
	}
	if got := sent[0].params; len(got) != 3 || got[0] != "Ana" || got[1] != "1" || got[2] != "Pitch editado" {
		t.Fatalf("params %v", got)
	}
	c = f.only(t)
	if c.State != conversations.StateWait || !c.WindowEndsAt.Equal(f.now.Add(12*time.Hour)) || len(c.Messages) != 1 {
		t.Fatalf("after approve %+v", c)
	}
}

func TestPostSaleApprovalAtNightWaitsForMorning(t *testing.T) {
	f := postSaleServer(t)
	f.now = time.Date(2026, 10, 31, 23, 0, 0, 0, saoPaulo)
	o := f.paidOrder(t, true)
	f.s.postSaleTick(context.Background())
	c := f.only(t)
	if rec := f.do(t, http.MethodPost, "/api/conversations/"+c.ID+"/approve", nil, admin()); rec.Code != http.StatusOK {
		t.Fatalf("approve %d %s", rec.Code, rec.Body.String())
	}
	if c = f.only(t); c.State != conversations.StateQueued || len(f.sender.all()) != 0 {
		t.Fatalf("night approve sent: %+v", c)
	}
	f.now = time.Date(2026, 11, 1, 8, 1, 0, 0, saoPaulo)
	if _, err := f.s.Checkout.Orders.Update(o.ID, func(o *orders.Order) { o.PaidAt = f.now.Add(-9 * time.Hour) }); err != nil {
		t.Fatal(err)
	}
	f.s.postSaleTick(context.Background())
	if c = f.only(t); c.State != conversations.StateWait || len(f.sender.all()) != 1 {
		t.Fatalf("morning send: %+v", c)
	}
}

func TestPostSaleReplyDraftsAgentAnswerAndLinkWorks(t *testing.T) {
	f := postSaleServer(t)
	brain := &fakeBrain{}
	f.s.PostSale.Brain = brain
	o := f.paidOrder(t, true)
	f.s.postSaleTick(context.Background())
	c := f.only(t)
	if rec := f.do(t, http.MethodPost, "/api/conversations/"+c.ID+"/approve", nil, admin()); rec.Code != http.StatusOK {
		t.Fatalf("approve %d", rec.Code)
	}
	if rec := f.inbound(t, "in-1", "quero sim!"); rec.Code != http.StatusOK {
		t.Fatalf("webhook %d %s", rec.Code, rec.Body.String())
	}
	f.inbound(t, "in-1", "quero sim!")
	c = f.waitFor(t, func(c conversations.Conversation) bool { _, ok := c.PendingDraft(); return ok })
	if c.State != conversations.StateActive {
		t.Fatalf("state %s", c.State)
	}
	customer := 0
	for _, m := range c.Messages {
		if m.Role == conversations.RoleCustomer {
			customer++
		}
	}
	if customer != 1 {
		t.Fatalf("duplicate inbound stored: %d", customer)
	}
	if brain.err != nil || brain.link != "https://site.test/complemento/"+c.LinkToken {
		t.Fatalf("link %q err %v", brain.link, brain.err)
	}
	rec := f.do(t, http.MethodGet, "/api/postsale/link/"+c.LinkToken, nil, storefrontHeaders())
	var link struct {
		OrderNumber int                      `json:"orderNumber"`
		Lines       []conversations.LinkLine `json:"lines"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &link); err != nil || rec.Code != http.StatusOK {
		t.Fatalf("link %d %s", rec.Code, rec.Body.String())
	}
	if link.OrderNumber != o.Number || len(link.Lines) != 1 || link.Lines[0].SKU != parallelSKU {
		t.Fatalf("link body %+v", link)
	}
	if rec := f.do(t, http.MethodPost, "/api/conversations/"+c.ID+"/approve", nil, admin()); rec.Code != http.StatusOK {
		t.Fatalf("approve reply %d %s", rec.Code, rec.Body.String())
	}
	sent := f.sender.all()
	if last := sent[len(sent)-1]; !strings.HasPrefix(last.text, "Fechado! Link: https://site.test/complemento/") {
		t.Fatalf("last sent %+v", last)
	}
	f.now = f.now.Add(13 * time.Hour)
	if rec := f.do(t, http.MethodGet, "/api/postsale/link/"+c.LinkToken, nil, storefrontHeaders()); rec.Code != http.StatusGone {
		t.Fatalf("expired link %d", rec.Code)
	}
}

func TestPostSaleComplementDetectedAndCounted(t *testing.T) {
	f := postSaleServer(t)
	base := f.paidOrder(t, true)
	f.s.postSaleTick(context.Background())
	c := f.only(t)
	f.do(t, http.MethodPost, "/api/conversations/"+c.ID+"/approve", nil, admin())
	join, err := f.s.Checkout.Orders.Create(func(map[string]int, []orders.Order) (orders.Order, error) {
		return orders.Order{Shipping: orders.Shipping{ServiceID: -2, JoinOrder: base.Number}, Totals: orders.Totals{ItemsBRL: 300, TotalBRL: 300}}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.Checkout.Orders.Update(join.ID, func(o *orders.Order) { o.Status = orders.StatusPaid; o.PaidAt = f.now }); err != nil {
		t.Fatal(err)
	}
	f.s.postSaleTick(context.Background())
	c = f.only(t)
	if c.ComplementOrder != join.Number || c.Outcome != conversations.OutcomeComplemented || c.State != conversations.StateClosed {
		t.Fatalf("complement %+v", c)
	}
	rec := f.do(t, http.MethodGet, "/api/conversations", nil, admin())
	var list struct {
		Metrics conversationMetrics `json:"metrics"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if list.Metrics.OffersSent != 1 || list.Metrics.Complements != 1 || list.Metrics.RevenueBRL != 300 {
		t.Fatalf("metrics %+v", list.Metrics)
	}
}

func TestPostSaleLateReplyGoesToHuman(t *testing.T) {
	f := postSaleServer(t)
	f.s.PostSale.Brain = &fakeBrain{}
	f.paidOrder(t, true)
	f.s.postSaleTick(context.Background())
	c := f.only(t)
	f.do(t, http.MethodPost, "/api/conversations/"+c.ID+"/approve", nil, admin())
	f.now = f.now.Add(13 * time.Hour)
	f.s.postSaleTick(context.Background())
	if c = f.only(t); c.State != conversations.StateClosed || c.Outcome != conversations.OutcomeExpired {
		t.Fatalf("expired %+v", c)
	}
	f.inbound(t, "late-1", "ainda dá pra adicionar?")
	if c = f.only(t); c.State != conversations.StateHuman {
		t.Fatalf("late reply %+v", c)
	}
	if rec := f.do(t, http.MethodPost, "/api/conversations/"+c.ID+"/reply", map[string]string{"text": "Dá sim, vou ver aqui"}, admin()); rec.Code != http.StatusOK {
		t.Fatalf("owner reply %d %s", rec.Code, rec.Body.String())
	}
	sent := f.sender.all()
	if sent[len(sent)-1].text != "Dá sim, vou ver aqui" {
		t.Fatalf("sent %+v", sent)
	}
}

func TestPostSaleIgnoresStrangersAndBadSignatures(t *testing.T) {
	f := postSaleServer(t)
	if rec := f.inbound(t, "x", "oi, quanto custa?"); rec.Code != http.StatusOK {
		t.Fatalf("stranger %d", rec.Code)
	}
	all, err := f.s.PostSale.Conversations.List()
	if err != nil || len(all) != 0 {
		t.Fatalf("stranger created conversation: %v %v", all, err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/whatsapp/webhook", strings.NewReader(`{}`))
	req.Header.Set("X-Hub-Signature-256", "sha256=00")
	rec := httptest.NewRecorder()
	f.s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("bad signature %d", rec.Code)
	}
	verify := httptest.NewRecorder()
	f.s.Handler().ServeHTTP(verify, httptest.NewRequest(http.MethodGet, "/api/whatsapp/webhook?hub.mode=subscribe&hub.verify_token=verify&hub.challenge=42", nil))
	if verify.Code != http.StatusOK || verify.Body.String() != "42" {
		t.Fatalf("verify %d %q", verify.Code, verify.Body.String())
	}
}

func TestPostSaleTrackingSentOnce(t *testing.T) {
	f := postSaleServer(t)
	o := f.paidOrder(t, false)
	if _, err := f.s.Checkout.Orders.Update(o.ID, func(o *orders.Order) {
		o.Status = orders.StatusShipped
		o.ShippedAt = f.now.Add(-time.Hour)
		o.TrackingCode = "BR123"
	}); err != nil {
		t.Fatal(err)
	}
	f.s.postSaleTick(context.Background())
	f.s.postSaleTick(context.Background())
	sent := f.sender.all()
	if len(sent) != 1 || sent[0].template != "collecta_rastreio" || sent[0].params[2] != "BR123" {
		t.Fatalf("tracking %+v", sent)
	}
	got, err := f.s.Checkout.Orders.Get(o.ID)
	if err != nil || got.TrackingNotified != "BR123" {
		t.Fatalf("not marked: %+v %v", got, err)
	}
}

func TestPostSaleOldShipmentsAreNotNotified(t *testing.T) {
	f := postSaleServer(t)
	o := f.paidOrder(t, false)
	if _, err := f.s.Checkout.Orders.Update(o.ID, func(o *orders.Order) {
		o.Status = orders.StatusShipped
		o.ShippedAt = f.now.Add(-10 * 24 * time.Hour)
		o.TrackingCode = "BR999"
	}); err != nil {
		t.Fatal(err)
	}
	f.s.postSaleTick(context.Background())
	if len(f.sender.all()) != 0 {
		t.Fatal("old shipment notified")
	}
}

func TestPostSaleNewCustomerMessageDropsStaleDraft(t *testing.T) {
	f := postSaleServer(t)
	f.s.PostSale.Brain = &fakeBrain{}
	f.paidOrder(t, true)
	f.s.postSaleTick(context.Background())
	c := f.only(t)
	f.do(t, http.MethodPost, "/api/conversations/"+c.ID+"/approve", nil, admin())
	f.inbound(t, "a", "quero")
	f.waitFor(t, func(c conversations.Conversation) bool { _, ok := c.PendingDraft(); return ok })
	f.inbound(t, "b", "e tem mais?")
	c = f.waitFor(t, func(c conversations.Conversation) bool {
		drafts := 0
		for _, m := range c.Messages {
			if m.Status == conversations.MsgDraft {
				drafts++
			}
		}
		_, last := c.PendingDraft()
		return drafts == 1 && last
	})
	if c.Messages[len(c.Messages)-2].Text != "e tem mais?" {
		t.Fatalf("messages %+v", c.Messages)
	}
}

func TestPostSaleDiscardedOfferGetsNoComplementCredit(t *testing.T) {
	f := postSaleServer(t)
	base := f.paidOrder(t, true)
	f.s.postSaleTick(context.Background())
	c := f.only(t)
	if rec := f.do(t, http.MethodPost, "/api/conversations/"+c.ID+"/discard", nil, admin()); rec.Code != http.StatusOK {
		t.Fatalf("discard %d", rec.Code)
	}
	join, err := f.s.Checkout.Orders.Create(func(map[string]int, []orders.Order) (orders.Order, error) {
		return orders.Order{Shipping: orders.Shipping{ServiceID: -2, JoinOrder: base.Number}, Totals: orders.Totals{ItemsBRL: 50, TotalBRL: 50}}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.Checkout.Orders.Update(join.ID, func(o *orders.Order) { o.Status = orders.StatusPaid; o.PaidAt = f.now }); err != nil {
		t.Fatal(err)
	}
	f.s.postSaleTick(context.Background())
	if c = f.only(t); c.ComplementOrder != 0 || c.Outcome != conversations.OutcomeSkipped {
		t.Fatalf("credited %+v", c)
	}
}

func TestPostSaleReplyOutsideServiceWindowKeepsState(t *testing.T) {
	f := postSaleServer(t)
	f.s.PostSale.Brain = &fakeBrain{}
	f.paidOrder(t, true)
	f.s.postSaleTick(context.Background())
	c := f.only(t)
	f.do(t, http.MethodPost, "/api/conversations/"+c.ID+"/approve", nil, admin())
	f.inbound(t, "a", "quero")
	f.waitFor(t, func(c conversations.Conversation) bool { _, ok := c.PendingDraft(); return ok })
	f.now = f.now.Add(25 * time.Hour)
	rec := f.do(t, http.MethodPost, "/api/conversations/"+c.ID+"/reply", map[string]string{"text": "oi"}, admin())
	if rec.Code != http.StatusConflict {
		t.Fatalf("reply %d", rec.Code)
	}
	if c = f.only(t); c.State != conversations.StateActive {
		t.Fatalf("state changed %+v", c.State)
	}
	if _, ok := c.PendingDraft(); !ok {
		t.Fatal("draft lost on failed reply")
	}
}
