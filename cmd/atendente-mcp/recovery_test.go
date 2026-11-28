package main

import (
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"opdeals/internal/whatsapp"
)

type fakeRecovery struct {
	mu   sync.Mutex
	off  bool
	html bool
	sent []string
	auth []string
}

const recoveryMsg = "Oi, Ana! Aqui é da Collecta. Vi que o pagamento do pedido #77 não foi concluído. Se ainda quiser, os itens estão aqui: https://www.collectatcg.com.br/pedido/o77. Qualquer dúvida, me chama por aqui."

func (f *fakeRecovery) handler(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	f.auth = append(f.auth, r.Header.Get("Authorization"))
	f.mu.Unlock()
	if f.off {
		http.NotFound(w, r)
		return
	}
	if f.html {
		_, _ = io.WriteString(w, "<!doctype html><html><body>Collecta</body></html>")
		return
	}
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/api/postsale/agent/recovery/orders":
		_, _ = io.WriteString(w, `{"ligado":true,"pedidos":[
			{"id":"o77","numero":77,"primeiroNome":"Ana","telefone":"5562988887777","motivo":"pix_expirado","mensagem":"`+recoveryMsg+`"},
			{"id":"o78","numero":78,"primeiroNome":"Bia","telefone":"5511999998888","motivo":"pagamento_recusado","mensagem":"Oi, Bia!"}]}`)
	case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/sent"):
		f.mu.Lock()
		f.sent = append(f.sent, r.URL.Path)
		f.mu.Unlock()
		_, _ = io.WriteString(w, `{"ok":true}`)
	default:
		http.NotFound(w, r)
	}
}

func TestRecoveryOffWhenServerHasNoRoute(t *testing.T) {
	api := &fakeRecovery{off: true}
	f := newFixtureWith(t, modeLive, testToken, api.handler)
	out, errText := callWith[struct{}, recoveryListOut](t, f, "recuperar_pedidos", struct{}{})
	if errText != "" || len(out.Pedidos) != 0 || out.Regras != "" || !strings.Contains(out.Aviso, "desligada") {
		t.Fatalf("sem a rota, segue a rodada sem pedido: %+v %q", out, errText)
	}
}

func TestRecoveryOffWhenServerAnswersWithSitePage(t *testing.T) {
	api := &fakeRecovery{html: true}
	f := newFixtureWith(t, modeLive, testToken, api.handler)
	out, errText := callWith[struct{}, recoveryListOut](t, f, "recuperar_pedidos", struct{}{})
	if errText != "" || len(out.Pedidos) != 0 || !strings.Contains(out.Aviso, "desligada") {
		t.Fatalf("página do site no lugar da lista também é desligada: %+v %q", out, errText)
	}
}

func TestRecoveryListSendAndHistory(t *testing.T) {
	api := &fakeRecovery{}
	f := newFixtureWith(t, modeLive, testToken, api.handler)
	out, errText := callWith[struct{}, recoveryListOut](t, f, "recuperar_pedidos", struct{}{})
	if errText != "" || len(out.Pedidos) != 2 || out.Regras != rules[rulesRecovery] || !strings.Contains(out.Regras, "NÃO chame `autorizar_envio`") {
		t.Fatalf("lista errada: %+v %q", out, errText)
	}
	if out.Pedidos[0].Mensagem != recoveryMsg {
		t.Fatalf("a mensagem tem de vir exatamente como o servidor mandou: %q", out.Pedidos[0].Mensagem)
	}
	if _, errText := callWith[recoveryIDInput, okOutput](t, f, "recuperar_enviado", recoveryIDInput{ID: "o77"}); errText != "" {
		t.Fatal(errText)
	}
	if len(api.sent) != 1 || api.sent[0] != "/api/postsale/agent/recovery/orders/o77/sent" || api.auth[0] != "Bearer "+testToken {
		t.Fatalf("marcação errada: %+v %+v", api.sent, api.auth)
	}
	hist, _ := callTool[historyOutput](t, f, "historico_bot", map[string]string{"telefone": "5562988887777"})
	if len(hist.Mensagens) != 1 || hist.Mensagens[0].Texto != recoveryMsg {
		t.Fatalf("a mensagem enviada devia ir para o histórico: %+v", hist)
	}
	if _, errText := callWith[recoveryFailedInput, okOutput](t, f, "recuperar_falhou", recoveryFailedInput{ID: "o78", Erro: "número sem WhatsApp"}); errText != "" {
		t.Fatal(errText)
	}
	again, _ := callWith[struct{}, recoveryListOut](t, f, "recuperar_pedidos", struct{}{})
	for _, p := range again.Pedidos {
		if p.ID == "o78" {
			t.Fatal("número sem WhatsApp não volta para a lista")
		}
	}
}

func TestRecoveryDraftAndTestModes(t *testing.T) {
	api := &fakeRecovery{}
	draft := newFixtureWith(t, modeDraft, testToken, api.handler)
	out, _ := callWith[struct{}, recoveryListOut](t, draft, "recuperar_pedidos", struct{}{})
	if len(out.Pedidos) != 0 || len(api.auth) != 0 {
		t.Fatalf("modo rascunho não chama o servidor nem envia: %+v", out)
	}
	test := newFixtureWith(t, modeTest, testToken, api.handler)
	out, _ = callWith[struct{}, recoveryListOut](t, test, "recuperar_pedidos", struct{}{})
	if len(out.Pedidos) != 1 || !test.app.cfg.TestPhones[whatsapp.PhoneKey(out.Pedidos[0].Telefone)] {
		t.Fatalf("modo teste só manda para telefone de teste: %+v", out.Pedidos)
	}
}
