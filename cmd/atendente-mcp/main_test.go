package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const storefrontJSON = `{
  "items": [
    {"sku":"op|shanks-op09","slug":"onepiece-shanks-op09-004-manga-nm","game":"onepiece","featured":true,"addedAt":"2026-10-01T10:00:00Z","gameLabel":"One Piece","number":"OP09-004","name":"Shanks","set":"OP09","variant":"Manga","condition":"NM","qty":1,"askBRL":1234.5},
    {"sku":"op|shanks-op01","slug":"onepiece-shanks-op01-120-nm","game":"onepiece","addedAt":"2026-10-20T10:00:00Z","gameLabel":"One Piece","number":"OP01-120","name":"Shanks","set":"OP01","condition":"NM","qty":3,"askBRL":45},
    {"sku":"pkm|bb","slug":"pokemon-booster-box-rivais-predestinados","game":"pokemon","addedAt":"2026-10-25T10:00:00Z","gameLabel":"Pokémon","kind":"sealed","name":"Booster Box Rivais Predestinados","qty":2,"askBRL":899.9},
    {"sku":"acc|sleeve","slug":"acessorios-sleeve-dragon-shield","gameLabel":"Acessórios","kind":"accessory","name":"Sleeve Dragon Shield","qty":10,"askBRL":79}
  ],
  "soldOut": [
    {"sku":"op|nami","slug":"onepiece-nami-op01-016-nm","gameLabel":"One Piece","number":"OP01-016","name":"Nami","set":"OP01","condition":"NM","qty":0,"askBRL":20}
  ]
}`

type ntfyCall struct {
	title    string
	priority string
	body     string
}

type fixture struct {
	app        *app
	session    *mcp.ClientSession
	now        time.Time
	storeHits  atomic.Int32
	storeFails atomic.Bool
	mu         sync.Mutex
	pushes     []ntfyCall
}

func newFixture(t *testing.T, mode string) *fixture {
	t.Helper()
	return newFixtureWith(t, mode, "", nil)
}

func newFixtureWith(t *testing.T, mode, agentToken string, agent http.HandlerFunc) *fixture {
	t.Helper()
	f := &fixture{now: time.Date(2026, 11, 2, 15, 0, 0, 0, saoPaulo)}
	store := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if agent != nil && strings.HasPrefix(r.URL.Path, "/api/postsale/agent/") {
			agent(w, r)
			return
		}
		if r.URL.Path != "/api/storefront" {
			http.NotFound(w, r)
			return
		}
		f.storeHits.Add(1)
		if f.storeFails.Load() {
			http.Error(w, "down", http.StatusBadGateway)
			return
		}
		_, _ = io.WriteString(w, storefrontJSON)
	}))
	t.Cleanup(store.Close)
	ntfy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		f.mu.Lock()
		f.pushes = append(f.pushes, ntfyCall{title: r.Header.Get("Title"), priority: r.Header.Get("Priority"), body: string(body)})
		f.mu.Unlock()
	}))
	t.Cleanup(ntfy.Close)
	cfg := config{
		OpdealsURL: store.URL,
		SiteURL:    "https://www.collectatcg.com.br",
		NtfyURL:    ntfy.URL + "/topico",
		Mode:       mode,
		TestPhones: map[string]bool{"6288887777": true},
		DataDir:    t.TempDir(),
		AgentToken: agentToken,
		NewsPath:   filepath.Join(t.TempDir(), "novidades.md"),
	}
	f.app = newApp(cfg, http.DefaultClient, func() time.Time { return f.now })
	ctx := context.Background()
	serverSide, clientSide := mcp.NewInMemoryTransports()
	if _, err := f.app.server().Connect(ctx, serverSide, nil); err != nil {
		t.Fatal(err)
	}
	session, err := mcp.NewClient(&mcp.Implementation{Name: "teste"}, nil).Connect(ctx, clientSide, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	f.session = session
	return f
}

func (f *fixture) pushed() []ntfyCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]ntfyCall(nil), f.pushes...)
}

func callTool[T any](t *testing.T, f *fixture, name string, args map[string]string) (T, string) {
	t.Helper()
	var out T
	res, err := f.session.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	if res.IsError {
		text := ""
		if len(res.Content) > 0 {
			if tc, ok := res.Content[0].(*mcp.TextContent); ok {
				text = tc.Text
			}
		}
		return out, text
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

func TestLoadConfig(t *testing.T) {
	env := func(m map[string]string) func(string) string {
		return func(k string) string { return m[k] }
	}
	cfg, err := loadConfig(env(map[string]string{"NTFY_TOPIC": "abc", "ATENDENTE_DADOS": "/tmp/x"}))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Mode != modeDraft || cfg.NtfyURL != "https://ntfy.sh/abc" || cfg.OpdealsURL != "https://collecta-deals-production.up.railway.app" {
		t.Fatalf("defaults = %+v", cfg)
	}
	if _, err := loadConfig(env(map[string]string{})); err == nil {
		t.Fatal("missing NTFY_TOPIC accepted")
	}
	if _, err := loadConfig(env(map[string]string{"NTFY_TOPIC": "abc", "ATENDENTE_MODO": "ligado"})); err == nil {
		t.Fatal("invalid mode accepted")
	}
	if _, err := loadConfig(env(map[string]string{"NTFY_TOPIC": "abc", "ATENDENTE_MODO": "teste"})); err == nil {
		t.Fatal("test mode without phones accepted")
	}
	cfg, err = loadConfig(env(map[string]string{"NTFY_TOPIC": "abc", "ATENDENTE_DADOS": "/tmp/x", "ATENDENTE_MODO": "teste", "ATENDENTE_TESTE_TELEFONES": "+55 (62) 98888-7777, 62 3333-4444"}))
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.TestPhones["6288887777"] || !cfg.TestPhones["6233334444"] || len(cfg.TestPhones) != 2 {
		t.Fatalf("test phones = %v", cfg.TestPhones)
	}
}

func TestToolsAndInstructions(t *testing.T) {
	f := newFixture(t, modeLive)
	res, err := f.session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, tool := range res.Tools {
		names = append(names, tool.Name)
	}
	sort.Strings(names)
	want := "autorizar_envio,buscar_estoque,chamar_humano,destaques,historico_bot,modo,novidades,registrar_origem,regras"
	if got := strings.Join(names, ","); got != want {
		t.Fatalf("tools = %s, want %s", got, want)
	}
	got := f.session.InitializeResult().Instructions
	if got != serverInstructions || len(got) > 2000 || !strings.Contains(got, "modo") {
		t.Fatalf("instruções do servidor com %d bytes", len(got))
	}
}

func TestModeReportsVersionAndTime(t *testing.T) {
	f := newFixture(t, modeTest)
	out, errText := callTool[modeOutput](t, f, "modo", nil)
	if errText != "" {
		t.Fatal(errText)
	}
	if out.Modo != modeTest || out.Agora != "segunda 02/11 15:00" || len(out.Versao) != 8 {
		t.Fatalf("modo = %+v", out)
	}
	if out.Regras != rules[rulesRound] || !strings.Contains(out.Regras, "## A cada rodada") || !strings.Contains(out.Regras, "## Regras que nunca mudam") {
		t.Fatal("modo devia devolver as regras da rodada")
	}
	if strings.Contains(out.Regras, "## Como falar") || strings.Contains(out.Regras, "## Pós-venda") || len(out.Regras) > 8000 {
		t.Fatalf("modo devolveu regras demais: %d bytes", len(out.Regras))
	}
}

func TestRulesSplitKeepsEverySection(t *testing.T) {
	total := 0
	for _, text := range rules {
		total += len(text)
	}
	if want := len(strings.TrimSpace(instructions)); total < want-100 || total > want {
		t.Fatalf("grupos somam %d bytes, regras têm %d", total, want)
	}
	for _, line := range strings.Split(instructions, "\n") {
		heading, ok := strings.CutPrefix(line, "## ")
		if !ok {
			continue
		}
		found := 0
		for _, text := range rules {
			if strings.Contains(text, "## "+heading+"\n") {
				found++
			}
		}
		if found != 1 {
			t.Fatalf("seção %q aparece em %d grupos", heading, found)
		}
	}
	if _, err := splitRules("## Seção nova\ntexto\n"); err == nil {
		t.Fatal("seção sem grupo aceita")
	}
}

func TestRulesTool(t *testing.T) {
	f := newFixture(t, modeLive)
	out, errText := callTool[rulesOutput](t, f, "regras", map[string]string{"secao": " Instagram "})
	if errText != "" || out.Secao != "instagram" || out.Regras != rules[rulesInstagram] {
		t.Fatalf("instagram = %+v %q", out, errText)
	}
	out, _ = callTool[rulesOutput](t, f, "regras", map[string]string{"secao": "posvenda"})
	for _, g := range []string{rulesPostSaleCompose, rulesPostSaleDelivery, rulesPostSaleResponses} {
		if !strings.Contains(out.Regras, rules[g]) {
			t.Fatalf("posvenda sem %s", g)
		}
	}
	_, errText = callTool[rulesOutput](t, f, "regras", map[string]string{"secao": "tudo"})
	if !strings.Contains(errText, "seção desconhecida") {
		t.Fatalf("error = %q", errText)
	}
}

func TestSearchLinksByKindAndPrice(t *testing.T) {
	f := newFixture(t, modeLive)
	out, errText := callTool[searchOutput](t, f, "buscar_estoque", map[string]string{"consulta": "Shanks manga"})
	if errText != "" {
		t.Fatal(errText)
	}
	if len(out.Exatos) != 1 || out.Exatos[0].Link != "https://www.collectatcg.com.br/carta/onepiece-shanks-op09-004-manga-nm" || out.Exatos[0].Preco != "R$ 1.234,50" {
		t.Fatalf("exatos = %+v", out.Exatos)
	}
	if len(out.Parecidos) != 1 || out.Parecidos[0].Numero != "OP01-120" {
		t.Fatalf("parecidos = %+v", out.Parecidos)
	}
	out, _ = callTool[searchOutput](t, f, "buscar_estoque", map[string]string{"consulta": "booster box rivais"})
	if len(out.Exatos) != 1 || out.Exatos[0].Link != "https://www.collectatcg.com.br/selado/pokemon-booster-box-rivais-predestinados" {
		t.Fatalf("sealed = %+v", out.Exatos)
	}
	out, _ = callTool[searchOutput](t, f, "buscar_estoque", map[string]string{"consulta": "sleeve dragon"})
	if len(out.Exatos) != 1 || out.Exatos[0].Link != "https://www.collectatcg.com.br/acessorio/acessorios-sleeve-dragon-shield" {
		t.Fatalf("accessory = %+v", out.Exatos)
	}
}

func TestSearchFindsSoldOutByNumber(t *testing.T) {
	f := newFixture(t, modeLive)
	out, _ := callTool[searchOutput](t, f, "buscar_estoque", map[string]string{"consulta": "OP01-016"})
	if len(out.Exatos) != 0 {
		t.Fatalf("sold-out card listed as in stock: %+v", out.Exatos)
	}
	if len(out.Esgotados) != 1 || out.Esgotados[0].Nome != "Nami" || out.Esgotados[0].Disponivel != 0 {
		t.Fatalf("esgotados = %+v", out.Esgotados)
	}
}

func TestCatalogCacheAndStaleFallback(t *testing.T) {
	f := newFixture(t, modeLive)
	callTool[searchOutput](t, f, "buscar_estoque", map[string]string{"consulta": "shanks"})
	callTool[searchOutput](t, f, "buscar_estoque", map[string]string{"consulta": "nami"})
	if hits := f.storeHits.Load(); hits != 1 {
		t.Fatalf("storefront hits = %d, want 1 within TTL", hits)
	}
	f.storeFails.Store(true)
	f.now = f.now.Add(2 * time.Minute)
	out, errText := callTool[searchOutput](t, f, "buscar_estoque", map[string]string{"consulta": "shanks"})
	if errText != "" || len(out.Exatos) == 0 {
		t.Fatalf("stale fallback failed: %q %+v", errText, out)
	}
	if hits := f.storeHits.Load(); hits != 2 {
		t.Fatalf("storefront hits = %d, want a refresh attempt", hits)
	}
}

func TestSearchErrorsWhenStorefrontDown(t *testing.T) {
	f := newFixture(t, modeLive)
	f.storeFails.Store(true)
	_, errText := callTool[searchOutput](t, f, "buscar_estoque", map[string]string{"consulta": "shanks"})
	if !strings.Contains(errText, "não consegui ler o estoque") {
		t.Fatalf("error = %q", errText)
	}
}

func TestAuthorizeDraftModeNotifiesAndBlocks(t *testing.T) {
	f := newFixture(t, modeDraft)
	out, errText := callTool[authorizeOutput](t, f, "autorizar_envio", map[string]string{"telefone": "+55 11 99999-8888", "texto": "Oi! Temos sim."})
	if errText != "" {
		t.Fatal(errText)
	}
	if out.Enviar || !strings.Contains(out.Motivo, "NÃO envie") {
		t.Fatalf("draft authorize = %+v", out)
	}
	pushes := f.pushed()
	if len(pushes) != 1 || !strings.Contains(pushes[0].body, "Oi! Temos sim.") || pushes[0].title != "Rascunho do atendente" {
		t.Fatalf("pushes = %+v", pushes)
	}
	hist, _ := callTool[historyOutput](t, f, "historico_bot", map[string]string{"telefone": "11999998888"})
	if len(hist.Mensagens) != 0 {
		t.Fatalf("draft recorded as sent: %+v", hist)
	}
}

func TestAuthorizeTestModeOnlyTestPhones(t *testing.T) {
	f := newFixture(t, modeTest)
	out, _ := callTool[authorizeOutput](t, f, "autorizar_envio", map[string]string{"telefone": "+55 11 99999-8888", "texto": "Oi"})
	if out.Enviar {
		t.Fatal("non-test phone authorized in test mode")
	}
	out, _ = callTool[authorizeOutput](t, f, "autorizar_envio", map[string]string{"telefone": "(62) 98888-7777", "texto": "Oi, teste"})
	if !out.Enviar {
		t.Fatalf("test phone refused: %+v", out)
	}
	hist, _ := callTool[historyOutput](t, f, "historico_bot", map[string]string{"telefone": "556288887777"})
	if len(hist.Mensagens) != 1 || hist.Mensagens[0].Texto != "Oi, teste" || hist.Mensagens[0].Em != "02/11 15:00" {
		t.Fatalf("history = %+v", hist)
	}
}

func TestAuthorizeLiveKeepsSevenDaysOfHistory(t *testing.T) {
	f := newFixture(t, modeLive)
	callTool[authorizeOutput](t, f, "autorizar_envio", map[string]string{"telefone": "62 99999-1111", "texto": "antiga"})
	f.now = f.now.Add(8 * 24 * time.Hour)
	out, _ := callTool[authorizeOutput](t, f, "autorizar_envio", map[string]string{"telefone": "62 99999-1111", "texto": "nova"})
	if !out.Enviar {
		t.Fatalf("live refused: %+v", out)
	}
	hist, _ := callTool[historyOutput](t, f, "historico_bot", map[string]string{"telefone": "6299991111"})
	if len(hist.Mensagens) != 1 || hist.Mensagens[0].Texto != "nova" {
		t.Fatalf("history = %+v", hist)
	}
}

func TestHistoryIgnoresRepeatWithinTenMinutes(t *testing.T) {
	h := newHistory(t.TempDir())
	at := time.Date(2026, 11, 3, 19, 0, 0, 0, saoPaulo)
	for _, d := range []time.Duration{0, 2 * time.Minute, 20 * time.Minute} {
		if err := h.add("556299991111", "Oii, tudo bem?", "", at.Add(d)); err != nil {
			t.Fatal(err)
		}
	}
	got, err := h.since("556299991111", at.Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("history = %+v", got)
	}
}

func TestAuthorizeRejectsDashAndBadPhone(t *testing.T) {
	f := newFixture(t, modeLive)
	out, _ := callTool[authorizeOutput](t, f, "autorizar_envio", map[string]string{"telefone": "62 99999-1111", "texto": "Temos sim — R$ 10"})
	if out.Enviar || !strings.Contains(out.Motivo, "travessão") {
		t.Fatalf("dash authorized: %+v", out)
	}
	_, errText := callTool[authorizeOutput](t, f, "autorizar_envio", map[string]string{"telefone": "Maria", "texto": "Oi"})
	if !strings.Contains(errText, "telefone inválido") {
		t.Fatalf("error = %q", errText)
	}
}

func TestAuthorizeInstagramHandle(t *testing.T) {
	f := newFixture(t, modeLive)
	out, errText := callTool[authorizeOutput](t, f, "autorizar_envio", map[string]string{"telefone": "ig:@Collecta.Fan", "texto": "Oii, tudo bem?"})
	if !out.Enviar || errText != "" {
		t.Fatalf("instagram refused: %+v %q", out, errText)
	}
	hist, _ := callTool[historyOutput](t, f, "historico_bot", map[string]string{"telefone": "ig:collecta.fan"})
	if len(hist.Mensagens) != 1 {
		t.Fatalf("history = %+v", hist)
	}
	other, _ := callTool[historyOutput](t, f, "historico_bot", map[string]string{"telefone": "62 99999-1111"})
	if len(other.Mensagens) != 0 {
		t.Fatalf("instagram leaked into phone history: %+v", other)
	}
	_, errText = callTool[authorizeOutput](t, f, "autorizar_envio", map[string]string{"telefone": "ig:nome com espaço", "texto": "Oi"})
	if !strings.Contains(errText, "Instagram inválido") {
		t.Fatalf("error = %q", errText)
	}
}

func TestAuthorizeTestModeBlocksInstagram(t *testing.T) {
	f := newFixtureWith(t, modeTest, "", nil)
	out, _ := callTool[authorizeOutput](t, f, "autorizar_envio", map[string]string{"telefone": "ig:collecta.fan", "texto": "Oi"})
	if out.Enviar {
		t.Fatalf("test mode sent to instagram: %+v", out)
	}
}

func TestHandoffPushesHighPriority(t *testing.T) {
	f := newFixture(t, modeLive)
	out, errText := callTool[handoffOutput](t, f, "chamar_humano", map[string]string{"telefone": "62 99999-1111", "nome": "Ana", "motivo": "quer vender coleção", "resumo": "200 cartas de Pokémon"})
	if errText != "" || !out.OK {
		t.Fatalf("handoff = %+v %q", out, errText)
	}
	pushes := f.pushed()
	if len(pushes) != 1 || pushes[0].priority != "high" || !strings.Contains(pushes[0].body, "Ana 62 99999-1111: quer vender coleção") {
		t.Fatalf("pushes = %+v", pushes)
	}
}

func TestHighlightsFeaturedFirstThenNewest(t *testing.T) {
	f := newFixture(t, modeLive)
	out, errText := callTool[highlightsOutput](t, f, "destaques", map[string]string{"jogo": "One Piece"})
	if errText != "" || len(out.Itens) != 2 {
		t.Fatalf("destaques %+v %q", out, errText)
	}
	if out.Itens[0].Numero != "OP09-004" || out.Itens[0].Motivo != "destaque da loja" || out.Itens[0].Link == "" {
		t.Fatalf("first %+v", out.Itens[0])
	}
	if out.Itens[1].Numero != "OP01-120" || out.Itens[1].Motivo != "chegou agora" {
		t.Fatalf("second %+v", out.Itens[1])
	}
	pkm, _ := callTool[highlightsOutput](t, f, "destaques", map[string]string{"jogo": "pokémon"})
	if len(pkm.Itens) != 1 || pkm.Itens[0].Preco != "R$ 899,90" {
		t.Fatalf("pokemon %+v", pkm)
	}
}

const newsFile = `# Novidades da Collecta
Texto antes do primeiro título é ignorado.

## Loja física em reforma
até: 14/11/2026
A loja física tá em reforma e abre dia 15/11.

## Loja física aberta
desde: 15/11/2026
Aberta todo dia das 12h às 22h.

## Cupom BEMVINDO
5% na primeira compra no site.

## Pré-venda antiga
ate: 01/11/2026
Já acabou.

## Data errada
até: 31/02/2026
Não deve aparecer.

## Sem texto
desde: 01/11/2026
`

func newsTitles(t *testing.T, f *fixture) []string {
	t.Helper()
	out, errText := callTool[newsOutput](t, f, "novidades", nil)
	if errText != "" {
		t.Fatal(errText)
	}
	titles := make([]string, 0, len(out.Novidades))
	for _, n := range out.Novidades {
		titles = append(titles, n.Titulo)
	}
	return titles
}

func TestNewsCarriesConversationRules(t *testing.T) {
	f := newFixture(t, modeLive)
	out, errText := callTool[newsOutput](t, f, "novidades", nil)
	if errText != "" || out.Regras != rules[rulesConversation] || !strings.Contains(out.Regras, "## Como falar") || !strings.Contains(out.Regras, "## Dados da loja") {
		t.Fatalf("novidades sem regras de conversa: %q", errText)
	}
}

func TestNewsFilteredByDate(t *testing.T) {
	f := newFixture(t, modeLive)
	if got := newsTitles(t, f); len(got) != 0 {
		t.Fatalf("missing file should give no news, got %v", got)
	}
	if err := os.WriteFile(f.app.cfg.NewsPath, []byte(newsFile), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(newsTitles(t, f), "|"); got != "Loja física em reforma|Cupom BEMVINDO" {
		t.Fatalf("02/11 news = %s", got)
	}
	f.now = time.Date(2026, 11, 14, 23, 59, 0, 0, saoPaulo)
	if got := strings.Join(newsTitles(t, f), "|"); got != "Loja física em reforma|Cupom BEMVINDO" {
		t.Fatalf("14/11 news = %s", got)
	}
	f.now = time.Date(2026, 11, 15, 0, 1, 0, 0, saoPaulo)
	if got := strings.Join(newsTitles(t, f), "|"); got != "Loja física aberta|Cupom BEMVINDO" {
		t.Fatalf("15/11 news = %s", got)
	}
}

func TestRegisterOriginOncePerPhone(t *testing.T) {
	f := newFixture(t, modeLive)
	out, errText := callTool[originOutput](t, f, "registrar_origem", map[string]string{"telefone": "+55 62 99999-1111", "origem": "  Instagram   da loja "})
	if errText != "" || !out.OK || out.JaRegistrado {
		t.Fatalf("first = %+v %q", out, errText)
	}
	out, _ = callTool[originOutput](t, f, "registrar_origem", map[string]string{"telefone": "62 9999-1111", "origem": "Google"})
	if !out.JaRegistrado {
		t.Fatalf("duplicate not detected: %+v", out)
	}
	data, err := os.ReadFile(filepath.Join(f.app.cfg.DataDir, "origem.csv"))
	if err != nil {
		t.Fatal(err)
	}
	if got := string(data); got != "data,telefone,origem\n02/11/2026 15:00,6299991111,Instagram da loja\n" {
		t.Fatalf("csv = %q", got)
	}
}

func cartArgs(phone, text string) map[string]string {
	return map[string]string{"telefone": phone, "texto": text, "tipo": "carrinho"}
}

func TestAuthorizeCartAllowsTwoBalloonsOncePerWeek(t *testing.T) {
	f := newFixture(t, modeLive)
	for _, text := range []string{"Oii Ana, vi a case de Radiance no seu carrinho", "Deixei o cupom CONTINUE5 pra você 🩷"} {
		if out, errText := callTool[authorizeOutput](t, f, "autorizar_envio", cartArgs("62 99999-1111", text)); !out.Enviar || errText != "" {
			t.Fatalf("balão de carrinho recusado: %+v %q", out, errText)
		}
		f.now = f.now.Add(time.Minute)
	}
	out, _ := callTool[authorizeOutput](t, f, "autorizar_envio", cartArgs("62 99999-1111", "Terceiro balão"))
	if out.Enviar || !strings.Contains(out.Motivo, "2 balões") {
		t.Fatalf("terceiro balão devia ser recusado: %+v", out)
	}
	f.now = f.now.Add(3 * 24 * time.Hour)
	out, _ = callTool[authorizeOutput](t, f, "autorizar_envio", cartArgs("62 99999-1111", "Oii Ana, de novo"))
	if out.Enviar || !strings.Contains(out.Motivo, "7 dias") {
		t.Fatalf("segunda abordagem na semana devia ser recusada: %+v", out)
	}
	f.now = f.now.Add(5 * 24 * time.Hour)
	if out, _ = callTool[authorizeOutput](t, f, "autorizar_envio", cartArgs("62 99999-1111", "Oii Ana, chegou novidade")); !out.Enviar {
		t.Fatalf("depois de 7 dias devia liberar: %+v", out)
	}
}

func TestAuthorizeCartSkipsNumberWithRecentChat(t *testing.T) {
	f := newFixture(t, modeLive)
	callTool[authorizeOutput](t, f, "autorizar_envio", map[string]string{"telefone": "62 99999-1111", "texto": "Oii, tudo bem?"})
	f.now = f.now.Add(5 * time.Minute)
	out, _ := callTool[authorizeOutput](t, f, "autorizar_envio", cartArgs("62 99999-1111", "Vi o Jolteon no seu carrinho"))
	if out.Enviar || !strings.Contains(out.Motivo, "7 dias") {
		t.Fatalf("número com conversa recente devia ser pulado: %+v", out)
	}
}

func TestAuthorizeCartDailyLimit(t *testing.T) {
	f := newFixture(t, modeLive)
	for i := range cartPerDay {
		phone := fmt.Sprintf("62 99999-%04d", i)
		if out, _ := callTool[authorizeOutput](t, f, "autorizar_envio", cartArgs(phone, "Oii, vi seu carrinho "+phone)); !out.Enviar {
			t.Fatalf("carrinho %d recusado: %+v", i, out)
		}
	}
	out, _ := callTool[authorizeOutput](t, f, "autorizar_envio", cartArgs("62 98888-0000", "Oii, vi seu carrinho"))
	if out.Enviar || !strings.Contains(out.Motivo, "limite de 10") {
		t.Fatalf("11º carrinho do dia devia ser recusado: %+v", out)
	}
	if out, _ = callTool[authorizeOutput](t, f, "autorizar_envio", cartArgs("62 99999-0000", "Deixei o cupom CONTINUE5")); !out.Enviar {
		t.Fatalf("2º balão de quem já foi abordado hoje devia passar: %+v", out)
	}
	if out, _ = callTool[authorizeOutput](t, f, "autorizar_envio", map[string]string{"telefone": "62 98888-0000", "texto": "Oii, tudo bem?"}); !out.Enviar {
		t.Fatalf("conversa normal não entra no limite de carrinho: %+v", out)
	}
	f.now = f.now.Add(24 * time.Hour)
	if out, _ = callTool[authorizeOutput](t, f, "autorizar_envio", cartArgs("62 97777-0000", "Oii, vi seu carrinho")); !out.Enviar {
		t.Fatalf("no dia seguinte o limite zera: %+v", out)
	}
	if _, errText := callTool[authorizeOutput](t, f, "autorizar_envio", map[string]string{"telefone": "62 97777-1111", "texto": "Oi", "tipo": "promo"}); !strings.Contains(errText, "tipo inválido") {
		t.Fatalf("tipo desconhecido devia falhar: %q", errText)
	}
}

func TestAuthorizeCartDraftModeDoesNotCount(t *testing.T) {
	f := newFixture(t, modeDraft)
	callTool[authorizeOutput](t, f, "autorizar_envio", cartArgs("62 99999-1111", "Oii, vi seu carrinho"))
	today, err := f.app.history.keysOfKindOn(kindCart, f.now)
	if err != nil || len(today) != 0 {
		t.Fatalf("rascunho não devia contar no limite: %v %v", today, err)
	}
}
