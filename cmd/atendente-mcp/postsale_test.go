package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const testToken = "agent-token-for-tests-0123456789abcdef"

type agentRequest struct {
	method string
	path   string
	auth   string
	body   string
}

type fakeAgentAPI struct {
	mu        sync.Mutex
	requests  []agentRequest
	cancelled bool
}

func (a *fakeAgentAPI) handler(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	a.mu.Lock()
	a.requests = append(a.requests, agentRequest{method: r.Method, path: r.URL.EscapedPath(), auth: r.Header.Get("Authorization"), body: string(body)})
	a.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	switch {
	case r.URL.Path == "/api/postsale/agent/outbox/claim":
		_, _ = io.WriteString(w, `{"quiet":false,"items":[
			{"id":"t.o1.BR1","kind":"tracking","orderNumber":7,"phone":"5562988887777","firstName":"Ana","text":"Oi Ana! rastreio BR1"},
			{"id":"t.o2.BR2","kind":"tracking","orderNumber":8,"phone":"5511999998888","firstName":"Bia","text":"Oi Bia! rastreio BR2"}]}`)
	case strings.HasSuffix(r.URL.Path, "/delivered"), strings.HasSuffix(r.URL.Path, "/failed"), strings.HasSuffix(r.URL.Path, "/handoff"):
		_, _ = io.WriteString(w, `{"ok":true}`)
	case r.URL.Path == "/api/postsale/agent/watch":
		_, _ = io.WriteString(w, `{"conversations":[{"conversationId":"c1","phone":"5562988887777","firstName":"Ana","orderNumber":7,"state":"waiting","windowEndsAt":"2026-11-03T02:00:00Z","lastMessages":[{"role":"assistant","text":"Oi Ana!","at":"2026-11-02T14:00:00Z"}]}]}`)
	case strings.HasSuffix(r.URL.Path, "/inbound"):
		_, _ = io.WriteString(w, `{"action":"answer","conversationId":"c1","context":{"firstName":"Ana","orderNumber":7,"status":"paid","bought":[{"nome":"Teach","qtd":1,"preco":250}],"offer":{"sku":"s1","name":"Teach (Parallel)","meta":"OP16","unitBRL":300},"windowOpen":true,"canAddUntil":"03/11 02:00","autoSend":false,"history":[{"role":"assistant","text":"Oi Ana!"},{"role":"customer","text":"quero"}]}}`)
	case r.URL.Path == "/api/postsale/agent/compose" && a.cancelled:
		_, _ = io.WriteString(w, `{"items":[
			{"conversationId":"c10","channel":"liga","phone":"5562988887777","firstName":"Ana","orderNumber":12391287,"cancelled":true,"items":[{"name":"Budew (221/217)","qty":1}],"canAddHow":"liga, pedido cancelado sem pagamento"},
			{"conversationId":"c11","channel":"liga","phone":"5511977776666","firstName":"Bia","orderNumber":12389418,"cancelled":true,"items":[{"name":"Dreepy (247/217)","qty":1}],"canAddHow":"liga, pedido cancelado sem pagamento"}]}`)
	case strings.HasSuffix(r.URL.Path, "/skip"):
		_, _ = io.WriteString(w, `{"ok":true}`)
	case r.URL.Path == "/api/postsale/agent/compose":
		_, _ = io.WriteString(w, `{"items":[{"conversationId":"c9","channel":"liga","firstName":"Ana","orderNumber":12376472,"pickup":true,"shipped":true,"items":[{"name":"Shanks (OP09-004)","qty":1}],"canAddHow":"liga: Adicionar a um pedido já existente"}]}`)
	case strings.HasSuffix(r.URL.Path, "/compose"):
		_, _ = io.WriteString(w, `{"status":"draft"}`)
	case strings.HasSuffix(r.URL.Path, "/reply"):
		_, _ = io.WriteString(w, `{"status":"draft"}`)
	case strings.HasSuffix(r.URL.Path, "/link"):
		w.WriteHeader(http.StatusConflict)
		_, _ = io.WriteString(w, `{"error":"a janela para juntar com o pedido já fechou: chame o humano"}`)
	default:
		http.NotFound(w, r)
	}
}

func (a *fakeAgentAPI) all() []agentRequest {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]agentRequest(nil), a.requests...)
}

func callWith[In, Out any](t *testing.T, f *fixture, name string, in In) (Out, string) {
	t.Helper()
	var out Out
	res, err := f.session.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: in})
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	if res.IsError {
		if len(res.Content) > 0 {
			if tc, ok := res.Content[0].(*mcp.TextContent); ok {
				return out, tc.Text
			}
		}
		return out, "erro sem texto"
	}
	data, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	return out, ""
}

func TestPostSaleToolsOnlyWithToken(t *testing.T) {
	api := &fakeAgentAPI{}
	f := newFixtureWith(t, modeLive, testToken, api.handler)
	res, err := f.session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, tool := range res.Tools {
		if strings.HasPrefix(tool.Name, "posvenda_") {
			names = append(names, tool.Name)
		}
	}
	sort.Strings(names)
	want := "posvenda_confirmar,posvenda_conversas,posvenda_envios,posvenda_escrever,posvenda_falhou,posvenda_humano,posvenda_link,posvenda_mensagem,posvenda_registrar,posvenda_responder"
	if got := strings.Join(names, ","); got != want {
		t.Fatalf("tools = %s", got)
	}
}

func TestClaimSendsBearerAndRecordsHistoryOnConfirm(t *testing.T) {
	api := &fakeAgentAPI{}
	f := newFixtureWith(t, modeLive, testToken, api.handler)
	claim, errText := callWith[struct{}, claimOutput](t, f, "posvenda_envios", struct{}{})
	if errText != "" || len(claim.Items) != 2 || claim.Regras != rules[rulesPostSaleDelivery] {
		t.Fatalf("claim %+v %q", claim, errText)
	}
	if _, errText := callWith[outboxIDInput, okOutput](t, f, "posvenda_confirmar", outboxIDInput{ID: "t.o1.BR1"}); errText != "" {
		t.Fatal(errText)
	}
	reqs := api.all()
	if len(reqs) != 2 || reqs[0].auth != "Bearer "+testToken || reqs[1].path != "/api/postsale/agent/outbox/t.o1.BR1/delivered" {
		t.Fatalf("requests %+v", reqs)
	}
	hist, _ := callTool[historyOutput](t, f, "historico_bot", map[string]string{"telefone": "5562988887777"})
	if len(hist.Mensagens) != 1 || hist.Mensagens[0].Texto != "Oi Ana! rastreio BR1" {
		t.Fatalf("history %+v", hist)
	}
}

func TestClaimInTestModeKeepsOnlyTestPhones(t *testing.T) {
	api := &fakeAgentAPI{}
	f := newFixtureWith(t, modeTest, testToken, api.handler)
	claim, _ := callWith[struct{}, claimOutput](t, f, "posvenda_envios", struct{}{})
	if len(claim.Items) != 1 || claim.Items[0].FirstName != "Ana" {
		t.Fatalf("test-mode claim %+v", claim)
	}
}

func TestClaimInDraftModeNeverCallsServer(t *testing.T) {
	api := &fakeAgentAPI{}
	f := newFixtureWith(t, modeDraft, testToken, api.handler)
	claim, _ := callWith[struct{}, claimOutput](t, f, "posvenda_envios", struct{}{})
	if len(claim.Items) != 0 || claim.Aviso == "" || claim.Regras != "" || len(api.all()) != 0 {
		t.Fatalf("draft claim %+v requests %+v", claim, api.all())
	}
}

func TestInboundMapsFieldsAndReturnsContext(t *testing.T) {
	api := &fakeAgentAPI{}
	f := newFixtureWith(t, modeLive, testToken, api.handler)
	in := inboundInput{ConversaID: "c1", Mensagens: []inboundMessage{{Texto: "quero", Hora: "14:05"}, {Texto: "oi, é o Jairo", DaLoja: true}}}
	out, errText := callWith[inboundInput, inboundOutput](t, f, "posvenda_registrar", in)
	if errText != "" || out.Action != "answer" || out.Regras != rules[rulesConversation] || out.Context == nil || out.Context.Offer == nil || out.Context.Offer.SKU != "s1" || len(out.Context.History) != 2 {
		t.Fatalf("inbound %+v %q", out, errText)
	}
	reqs := api.all()
	last := reqs[len(reqs)-1]
	if last.path != "/api/postsale/agent/conversations/c1/inbound" || !strings.Contains(last.body, `"text":"quero","at":"14:05"`) || !strings.Contains(last.body, `"fromStore":true`) {
		t.Fatalf("request %+v", last)
	}
}

func TestReplyRejectsDashAndLinkSurfacesServerError(t *testing.T) {
	api := &fakeAgentAPI{}
	f := newFixtureWith(t, modeLive, testToken, api.handler)
	if _, errText := callWith[replyInput, replyOutput](t, f, "posvenda_responder", replyInput{ConversaID: "c1", Texto: "Fechado — te mando"}); !strings.Contains(errText, "travessão") {
		t.Fatalf("dash accepted: %q", errText)
	}
	out, errText := callWith[replyInput, replyOutput](t, f, "posvenda_responder", replyInput{ConversaID: "c1", Texto: "Fechado, te mando o link"})
	if errText != "" || out.Status != "draft" {
		t.Fatalf("reply %+v %q", out, errText)
	}
	_, errText = callWith[linkInput, linkOutput](t, f, "posvenda_link", linkInput{ConversaID: "c1", Itens: []linkItem{{SKU: "s1", Qtd: 1}}})
	if !strings.Contains(errText, "janela") {
		t.Fatalf("link error %q", errText)
	}
	_, errText = callWith[linkInput, linkOutput](t, f, "posvenda_link", linkInput{ConversaID: "../../outbox/claim", Itens: []linkItem{{SKU: "s1", Qtd: 1}}})
	reqs := api.all()
	if last := reqs[len(reqs)-1]; strings.Contains(last.path, "/outbox/claim") {
		t.Fatalf("conversation id escaped the path: %+v (%q)", last, errText)
	}
}

func TestSearchExposesSKUAndEnvelope(t *testing.T) {
	f := newFixture(t, modeLive)
	out, _ := callTool[searchOutput](t, f, "buscar_estoque", map[string]string{"consulta": "Shanks manga"})
	if len(out.Exatos) != 1 || out.Exatos[0].SKU != "op|shanks-op09" || !out.Exatos[0].Envelope {
		t.Fatalf("single %+v", out.Exatos)
	}
	out, _ = callTool[searchOutput](t, f, "buscar_estoque", map[string]string{"consulta": "booster box rivais"})
	if len(out.Exatos) != 1 || out.Exatos[0].Envelope {
		t.Fatalf("sealed flagged as envelope %+v", out.Exatos)
	}
}

func TestComposeListAndMessageMapping(t *testing.T) {
	api := &fakeAgentAPI{}
	f := newFixtureWith(t, modeLive, testToken, api.handler)
	list, errText := callWith[struct{}, composeOutput](t, f, "posvenda_escrever", struct{}{})
	if errText != "" || len(list.Items) != 1 || list.Items[0].Channel != "liga" || !list.Items[0].Pickup || !list.Items[0].Shipped || !strings.Contains(list.ComoEscrever, "a caminho") || list.Items[0].Items[0].Name != "Shanks (OP09-004)" || list.Regras != rules[rulesPostSaleCompose] {
		t.Fatalf("compose list %+v %q", list, errText)
	}
	reqs := api.all()
	if first := reqs[0]; first.method != http.MethodGet || first.path != "/api/postsale/agent/compose" || first.auth != "Bearer "+testToken {
		t.Fatalf("list request %+v", first)
	}
	if _, errText := callWith[composeInput, replyOutput](t, f, "posvenda_mensagem", composeInput{ConversaID: "c9", Texto: "Oi Ana — valeu"}); !strings.Contains(errText, "travessão") {
		t.Fatalf("dash accepted: %q", errText)
	}
	if n := len(api.all()); n != 1 {
		t.Fatalf("dash text reached the server: %d requests", n)
	}
	text := "Oii Ana, tudo bem? Muito obrigado pela compra! 💙\nEsse Shanks é lindo demais, já tá separado aqui\nSe quiser, separo mais alguma coisa pra você pegar junto 🩷"
	out, errText := callWith[composeInput, replyOutput](t, f, "posvenda_mensagem", composeInput{ConversaID: "c9", Texto: text})
	if errText != "" || out.Status != "draft" {
		t.Fatalf("compose %+v %q", out, errText)
	}
	reqs = api.all()
	last := reqs[len(reqs)-1]
	var body textBody
	if err := json.Unmarshal([]byte(last.body), &body); err != nil {
		t.Fatal(err)
	}
	if last.method != http.MethodPost || last.path != "/api/postsale/agent/conversations/c9/compose" || body.Text != text {
		t.Fatalf("compose request %+v", last)
	}
}

func TestWatchCarriesResponseRulesOnlyWithConversations(t *testing.T) {
	api := &fakeAgentAPI{}
	f := newFixtureWith(t, modeLive, testToken, api.handler)
	out, errText := callWith[struct{}, watchOutput](t, f, "posvenda_conversas", struct{}{})
	if errText != "" || len(out.Conversations) != 1 || out.Regras != rules[rulesPostSaleResponses] {
		t.Fatalf("watch %+v %q", out, errText)
	}
	empty := newFixtureWith(t, modeLive, testToken, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"conversations":[],"items":[]}`)
	})
	if out, _ := callWith[struct{}, watchOutput](t, empty, "posvenda_conversas", struct{}{}); out.Regras != "" {
		t.Fatal("conversas vazias com regras")
	}
	if out, _ := callWith[struct{}, composeOutput](t, empty, "posvenda_escrever", struct{}{}); out.Regras != "" {
		t.Fatal("lista vazia de primeira mensagem com regras")
	}
	if out, _ := callWith[struct{}, claimOutput](t, empty, "posvenda_envios", struct{}{}); out.Regras != "" {
		t.Fatal("envios vazios com regras")
	}
}

func TestComposeListSkipsCancelledAlreadyMessaged(t *testing.T) {
	api := &fakeAgentAPI{cancelled: true}
	f := newFixtureWith(t, modeLive, testToken, api.handler)
	key, err := phoneKey("(62) 98888-7777")
	if err != nil {
		t.Fatal(err)
	}
	if err := f.app.history.add(key, "Oii Ana, vi o Budew no seu carrinho", kindCart, f.now.Add(-3*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	list, errText := callWith[struct{}, composeOutput](t, f, "posvenda_escrever", struct{}{})
	if errText != "" || len(list.Items) != 1 || list.Items[0].ConversationID != "c11" || !list.Items[0].Cancelled || list.Items[0].CanAddHow != "liga, pedido cancelado sem pagamento" || !strings.Contains(list.ComoEscrever, "acabou cancelando") {
		t.Fatalf("compose list %+v %q", list, errText)
	}
	reqs := api.all()
	if len(reqs) != 2 || reqs[1].path != "/api/postsale/agent/conversations/c10/skip" || !strings.Contains(reqs[1].body, "7 dias") {
		t.Fatalf("requests %+v", reqs)
	}
	data, err := json.Marshal(list)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "5511977776666") {
		t.Fatalf("phone leaked to the agent: %s", data)
	}
}
