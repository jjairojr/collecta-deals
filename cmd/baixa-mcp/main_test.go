package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"opdeals/internal/syncqueue"
	"opdeals/internal/trades"
)

type request struct {
	method string
	path   string
	game   string
	body   string
}

type ntfyCall struct {
	title    string
	priority string
	body     string
}

type fixture struct {
	app      *app
	session  *mcp.ClientSession
	dir      string
	mu       sync.Mutex
	ledgers  map[string][]trades.Trade
	tasks    []syncqueue.Task
	requests []request
	pushes   []ntfyCall
}

func baseLedgers(now time.Time) map[string][]trades.Trade {
	recent := now.AddDate(0, 0, -3).Format(time.DateOnly)
	old := now.AddDate(0, 0, -90).Format(time.DateOnly)
	return map[string][]trades.Trade{
		"onepiece": {
			{ID: "h1", Name: "Monkey.D.Luffy", Number: "OP15-119", Qty: 3, Status: "holding", AskBRL: 155, LigaListed: true, LigaQty: 3, LigaPriceBRL: 155},
			{ID: "h2", Name: "Shanks", Number: "OP09-004", Variant: "Manga", Qty: 1, Status: "holding"},
			{ID: "s1", Name: "Monkey.D.Luffy", Number: "OP15-119", Qty: 1, Status: "sold", SellPrice: 155, SellDate: recent, Buyer: "Ana — Liga #12000001", Channel: "liga"},
			{ID: "s2", Name: "Monkey.D.Luffy", Number: "OP15-119", Qty: 1, Status: "sold", SellPrice: 150, SellDate: old, Buyer: "Bia — Liga #11000001", Channel: "liga"},
		},
		"pokemon": {
			{ID: "p1", Name: "Tauros", Number: "094/083", Qty: 2, Status: "holding"},
			{ID: "s3", Name: "Tyrunt", Number: "089/088", Qty: 1, Status: "sold", SellDate: recent, Buyer: "nick — MyP #660256", Channel: "myp"},
		},
		"riftbound": {},
	}
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	now := time.Date(2026, 11, 3, 9, 0, 0, 0, saoPaulo)
	f := &fixture{dir: t.TempDir(), ledgers: baseLedgers(now)}
	f.tasks = []syncqueue.Task{{ID: "q1", Game: "pokemon", TradeID: "p1", Name: "Tauros", Number: "094/083", Action: "remove", Qty: 1, Reason: "Site #24", Status: "pending", CreatedAt: now}}
	store := httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(store.Close)
	ntfy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		f.mu.Lock()
		f.pushes = append(f.pushes, ntfyCall{title: r.Header.Get("Title"), priority: r.Header.Get("Priority"), body: string(body)})
		f.mu.Unlock()
	}))
	t.Cleanup(ntfy.Close)
	cfg := config{OpdealsURL: store.URL, AdminToken: "segredo", NtfyURL: ntfy.URL + "/topico", DataDir: f.dir}
	f.app = newApp(cfg, http.DefaultClient, func() time.Time { return now })
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

func (f *fixture) serve(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-Admin-Token") != "segredo" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	body, _ := io.ReadAll(r.Body)
	game := r.URL.Query().Get("game")
	f.mu.Lock()
	defer f.mu.Unlock()
	if r.Method != http.MethodGet {
		f.requests = append(f.requests, request{method: r.Method, path: r.URL.Path, game: game, body: string(body)})
	}
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/api/trades":
		writeJSON(w, map[string][]trades.Trade{"trades": f.ledgers[game]})
	case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/sell"):
		var in sale
		_ = json.Unmarshal(body, &in)
		id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/trades/"), "/sell")
		for i, t := range f.ledgers[game] {
			if t.ID != id {
				continue
			}
			sold := t
			sold.ID = "novo-" + id
			sold.Qty = in.Qty
			sold.Status = "sold"
			sold.SellPrice = in.SellPrice
			sold.SellDate = in.SellDate
			sold.Buyer = in.Buyer
			sold.Channel = in.Channel
			f.ledgers[game][i].Qty -= in.Qty
			f.ledgers[game] = append(f.ledgers[game], sold)
			writeJSON(w, sold)
			return
		}
		http.NotFound(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/api/trades":
		var in trades.Trade
		_ = json.Unmarshal(body, &in)
		in.ID = "criado"
		f.ledgers[game] = append(f.ledgers[game], in)
		writeJSON(w, in)
	case r.URL.Path == "/api/trades/liga":
		writeJSON(w, map[string]int{"updated": strings.Count(string(body), `"id"`)})
	case r.Method == http.MethodGet && r.URL.Path == "/api/syncqueue":
		writeJSON(w, map[string][]syncqueue.Task{"tasks": f.tasks})
	case strings.HasPrefix(r.URL.Path, "/api/syncqueue/"):
		writeJSON(w, f.tasks[0])
	case r.URL.Path == "/api/postsale/liga-orders":
		writeJSON(w, map[string]int{"created": 1, "skipped": 0})
	default:
		http.NotFound(w, r)
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func (f *fixture) writes() []request {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]request(nil), f.requests...)
}

func (f *fixture) pushed() []ntfyCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]ntfyCall(nil), f.pushes...)
}

func callTool[T any](t *testing.T, f *fixture, name string, args map[string]any) (T, string) {
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

func sellArgs(id, pedido string, qty int) map[string]any {
	return map[string]any{
		"jogo": "onepiece", "id": id, "qty": qty, "total_brl": 155.0 * float64(qty), "data": "2026-11-02",
		"comprador": "Fulano  de Tal", "canal": "liga", "pedido": pedido,
	}
}

func TestLoadConfig(t *testing.T) {
	env := func(m map[string]string) func(string) string {
		return func(k string) string { return m[k] }
	}
	if _, err := loadConfig(env(map[string]string{"NTFY_TOPIC": "x"})); err == nil {
		t.Fatal("sem token devia falhar")
	}
	if _, err := loadConfig(env(map[string]string{"OPDEALS_ADMIN_TOKEN": "t"})); err == nil {
		t.Fatal("sem ntfy devia falhar")
	}
	cfg, err := loadConfig(env(map[string]string{"OPDEALS_ADMIN_TOKEN": "t", "NTFY_TOPIC": "abc", "BAIXA_DADOS": "/tmp/x", "OPDEALS_URL": "http://x/"}))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.NtfyURL != "https://ntfy.sh/abc" || cfg.OpdealsURL != "http://x" || cfg.DataDir != "/tmp/x" {
		t.Fatalf("config errada: %+v", cfg)
	}
}

func TestStartListsWrittenOffOrders(t *testing.T) {
	f := newFixture(t)
	out, errText := callTool[startOutput](t, f, "inicio", nil)
	if errText != "" {
		t.Fatal(errText)
	}
	if strings.Join(out.PedidosLiga, ",") != "11000001,12000001" {
		t.Fatalf("pedidos errados: %+v", out)
	}
	if out.Carimbo != "2026-11-03-0900" || out.Versao != rulesVersion {
		t.Fatalf("carimbo/versão errados: %+v", out)
	}
	if out.Regras != instructions {
		t.Fatal("inicio devia devolver as regras inteiras")
	}
}

func TestServerInstructionsFitClientLimit(t *testing.T) {
	f := newFixture(t)
	got := f.session.InitializeResult().Instructions
	if got != serverInstructions || len(got) > 2000 {
		t.Fatalf("instruções do servidor com %d bytes", len(got))
	}
	if !strings.Contains(got, "inicio") {
		t.Fatal("instruções do servidor deviam mandar chamar inicio")
	}
}

func TestSearchSplitsStockAndRecentSales(t *testing.T) {
	f := newFixture(t)
	out, errText := callTool[searchOutput](t, f, "estoque_buscar", map[string]any{"jogo": "onepiece", "consulta": "#op15-119"})
	if errText != "" {
		t.Fatal(errText)
	}
	if len(out.EmEstoque) != 1 || out.EmEstoque[0].ID != "h1" || out.EmEstoque[0].LigaQty != 3 {
		t.Fatalf("em estoque errado: %+v", out.EmEstoque)
	}
	if len(out.VendidosRecentes) != 1 || out.VendidosRecentes[0].ID != "s1" {
		t.Fatalf("vendidos recentes devia ter só s1: %+v", out.VendidosRecentes)
	}
	byName, _ := callTool[searchOutput](t, f, "estoque_buscar", map[string]any{"jogo": "onepiece", "consulta": "shanks"})
	if len(byName.EmEstoque) != 1 || byName.EmEstoque[0].ID != "h2" {
		t.Fatalf("busca por nome falhou: %+v", byName)
	}
	if _, errText := callTool[searchOutput](t, f, "estoque_buscar", map[string]any{"jogo": "magic", "consulta": "x"}); errText == "" {
		t.Fatal("jogo inválido devia falhar")
	}
}

func TestSellBuildsBuyerAndChannel(t *testing.T) {
	f := newFixture(t)
	out, errText := callTool[sellOutput](t, f, "dar_baixa", sellArgs("h1", "#12345678", 2))
	if errText != "" {
		t.Fatal(errText)
	}
	if out.RestanteQty != 1 || out.RestanteID != "h1" || !out.NaLiga || out.LigaQty != 3 {
		t.Fatalf("saída errada: %+v", out)
	}
	w := f.writes()
	if len(w) != 1 || w[0].path != "/api/trades/h1/sell" || w[0].game != "onepiece" {
		t.Fatalf("chamada errada: %+v", w)
	}
	var sent sale
	if err := json.Unmarshal([]byte(w[0].body), &sent); err != nil {
		t.Fatal(err)
	}
	want := sale{Qty: 2, SellPrice: 310, SellCurrency: "BRL", SellDate: "2026-11-02", Buyer: "Fulano de Tal — Liga #12345678", Channel: "liga"}
	if sent != want {
		t.Fatalf("payload errado:\n got %+v\nwant %+v", sent, want)
	}
}

func TestSellRefuses(t *testing.T) {
	cases := []struct {
		name string
		args map[string]any
		want string
	}{
		{"pedido já baixado", sellArgs("h1", "12000001", 1), "já tem baixa"},
		{"qty acima do estoque", sellArgs("h1", "12345678", 4), "maior que o estoque"},
		{"registro vendido", sellArgs("s1", "12345678", 1), "já está vendido"},
		{"registro inexistente", sellArgs("zz", "12345678", 1), "não existe"},
		{"canal site", func() map[string]any { a := sellArgs("h1", "1", 1); a["canal"] = "site"; return a }(), "canal inválido"},
		{"canal myp", func() map[string]any { a := sellArgs("h1", "1", 1); a["canal"] = "myp"; return a }(), "MyPCards foi desativado"},
		{"pedido com letras", sellArgs("h1", "12a", 1), "pedido inválido"},
		{"data inválida", func() map[string]any { a := sellArgs("h1", "1", 1); a["data"] = "02/11/2026"; return a }(), "data inválida"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newFixture(t)
			_, errText := callTool[sellOutput](t, f, "dar_baixa", c.args)
			if !strings.Contains(errText, c.want) {
				t.Fatalf("erro %q não contém %q", errText, c.want)
			}
			if w := f.writes(); len(w) != 0 {
				t.Fatalf("não devia gravar: %+v", w)
			}
		})
	}
}

func TestSellSameOrderOtherCardIsAllowed(t *testing.T) {
	f := newFixture(t)
	if _, errText := callTool[sellOutput](t, f, "dar_baixa", sellArgs("h2", "12000001", 1)); errText != "" {
		t.Fatalf("outro item do mesmo pedido devia passar: %s", errText)
	}
}

func TestSellUnbacked(t *testing.T) {
	f := newFixture(t)
	args := map[string]any{
		"jogo": "pokemon", "nome": "Combo de Pacotes ME5", "qty": 1, "total_brl": 120.0, "data": "2026-11-02",
		"comprador": "Ana", "canal": "liga", "pedido": "12999999",
	}
	out, errText := callTool[unbackedOutput](t, f, "venda_sem_lastro", args)
	if errText != "" {
		t.Fatal(errText)
	}
	if out.Comprador != "Ana — Liga #12999999" {
		t.Fatalf("comprador errado: %+v", out)
	}
	var created trades.Trade
	if err := json.Unmarshal([]byte(f.writes()[0].body), &created); err != nil {
		t.Fatal(err)
	}
	if created.Status != "sold" || !created.SemLastro || created.BuyBRL != 0 || created.Channel != "liga" || created.SellPrice != 120 {
		t.Fatalf("registro errado: %+v", created)
	}
	if _, errText := callTool[unbackedOutput](t, f, "venda_sem_lastro", args); !strings.Contains(errText, "já tem baixa") {
		t.Fatalf("segunda vez devia recusar: %q", errText)
	}
}

func TestLigaFlag(t *testing.T) {
	f := newFixture(t)
	out, errText := callTool[ligaFlagOutput](t, f, "flag_liga", map[string]any{
		"jogo": "onepiece", "itens": []map[string]any{{"id": "h1", "na_liga": true, "liga_qty": 1, "liga_preco_brl": 155.0}},
	})
	if errText != "" || out.Atualizados != 1 {
		t.Fatalf("flag falhou: %+v %s", out, errText)
	}
	if !strings.Contains(f.writes()[0].body, `"ligaQty":1`) {
		t.Fatalf("payload errado: %s", f.writes()[0].body)
	}
	_, errText = callTool[ligaFlagOutput](t, f, "flag_liga", map[string]any{
		"jogo": "onepiece", "itens": []map[string]any{{"id": "s1", "na_liga": false, "liga_qty": 0, "liga_preco_brl": 0.0}},
	})
	if !strings.Contains(errText, "não está em estoque") || len(f.writes()) != 1 {
		t.Fatalf("registro vendido devia ser recusado: %q", errText)
	}
}

func TestQueue(t *testing.T) {
	f := newFixture(t)
	out, _ := callTool[queueOutput](t, f, "fila_sincronizar", nil)
	if len(out.Tarefas) != 1 || out.Tarefas[0].Acao != "remove" || out.Tarefas[0].TradeID != "p1" {
		t.Fatalf("fila errada: %+v", out)
	}
	if _, errText := callTool[okOutput](t, f, "fila_feita", map[string]any{"id": "q1"}); errText != "" {
		t.Fatal(errText)
	}
	if _, errText := callTool[okOutput](t, f, "fila_erro", map[string]any{"id": "q1", "erro": " "}); errText == "" {
		t.Fatal("erro sem motivo devia falhar")
	}
	if _, errText := callTool[okOutput](t, f, "fila_erro", map[string]any{"id": "q1", "erro": "duas ofertas"}); errText != "" {
		t.Fatal(errText)
	}
	w := f.writes()
	if len(w) != 2 || w[0].path != "/api/syncqueue/q1/done" || w[1].path != "/api/syncqueue/q1/error" || !strings.Contains(w[1].body, "duas ofertas") {
		t.Fatalf("chamadas erradas: %+v", w)
	}
}

func TestPostSaleSendsOnlyAllowedFields(t *testing.T) {
	f := newFixture(t)
	out, errText := callTool[postSaleOutput](t, f, "posvenda_liga", map[string]any{"pedidos": []map[string]any{{
		"codigo": "12376472", "primeiro_nome": "Ana Maria", "telefone": "(62) 99827-4848", "pago_em": "2026-11-02T14:05:00-03:00",
		"retirada": false, "enviado": true, "itens": []map[string]any{{"nome": "Shanks (OP09-004)", "qty": 1}},
	}}})
	if errText != "" || out.Criados != 1 {
		t.Fatalf("pós-venda falhou: %+v %s", out, errText)
	}
	want := `{"orders":[{"code":"12376472","firstName":"Ana","phone":"(62) 99827-4848","paidAt":"2026-11-02T14:05:00-03:00","pickup":false,"shipped":true,"items":[{"name":"Shanks (OP09-004)","qty":1}]}]}`
	if got := f.writes()[0].body; got != want {
		t.Fatalf("payload errado:\n got %s\nwant %s", got, want)
	}
	_, errText = callTool[postSaleOutput](t, f, "posvenda_liga", map[string]any{"pedidos": []map[string]any{{
		"codigo": "1", "primeiro_nome": "Ana", "telefone": "(62) 99827-4848", "pago_em": "02/11 14:05",
		"retirada": false, "enviado": false, "itens": []map[string]any{},
	}}})
	if !strings.Contains(errText, "pago_em inválido") {
		t.Fatalf("data errada devia falhar: %q", errText)
	}
}

func TestReport(t *testing.T) {
	f := newFixture(t)
	out, errText := callTool[reportOutput](t, f, "relatorio", map[string]any{
		"resumo": "2 baixas · 1 pendência · 1 URGENTE",
		"corpo":  "| Liga | 12345678 | Fulano | Luffy OP15-119 | 2 |",
	})
	if errText != "" || !out.Enviado {
		t.Fatalf("relatório falhou: %+v %s", out, errText)
	}
	if out.Arquivo != filepath.Join(f.dir, "2026-11-03-0900.md") {
		t.Fatalf("arquivo errado: %s", out.Arquivo)
	}
	data, err := os.ReadFile(out.Arquivo)
	if err != nil || !strings.HasPrefix(string(data), "2 baixas · 1 pendência · 1 URGENTE\n") {
		t.Fatalf("conteúdo errado: %q %v", data, err)
	}
	p := f.pushed()
	if len(p) != 1 || p[0].priority != "high" || p[0].title != "Baixa de vendas" {
		t.Fatalf("push errado: %+v", p)
	}
	if _, errText := callTool[reportOutput](t, f, "relatorio", map[string]any{"resumo": "nada novo", "corpo": ""}); errText != "" {
		t.Fatal(errText)
	}
	if p := f.pushed(); p[1].priority != "default" {
		t.Fatalf("nada novo devia ter prioridade normal: %+v", p[1])
	}
}

func TestReportRefusesPersonalData(t *testing.T) {
	cases := map[string]string{
		"CPF":      "comprador 123.456.789-00",
		"e-mail":   "contato ana@gmail.com",
		"telefone": "ligar (62) 99827-4848",
	}
	for kind, body := range cases {
		t.Run(kind, func(t *testing.T) {
			f := newFixture(t)
			_, errText := callTool[reportOutput](t, f, "relatorio", map[string]any{"resumo": "1 baixa", "corpo": body})
			if !strings.Contains(errText, kind) {
				t.Fatalf("devia recusar %s: %q", kind, errText)
			}
			if len(f.pushed()) != 0 {
				t.Fatal("não devia mandar push")
			}
		})
	}
	f := newFixture(t)
	if _, errText := callTool[reportOutput](t, f, "relatorio", map[string]any{"resumo": strings.Repeat("x", 121), "corpo": ""}); errText == "" {
		t.Fatal("resumo longo devia falhar")
	}
}

func pendingArgs(pedido, numero, nome string) map[string]any {
	return map[string]any{
		"jogo": "onepiece", "pedido": pedido, "numero": numero, "nome": nome, "qty": 1, "total_brl": 28.98,
		"data": "2026-11-03", "comprador": "Nicolas", "canal": "liga", "motivo": "dois lotes",
	}
}

func TestPendingCarriesOverUntilWrittenOff(t *testing.T) {
	f := newFixture(t)
	if _, errText := callTool[okOutput](t, f, "registrar_pendencia", pendingArgs("12385475", "OP09-004", "Shanks")); errText != "" {
		t.Fatal(errText)
	}
	if _, errText := callTool[okOutput](t, f, "registrar_pendencia", pendingArgs("12385475", "OP09-004", "Shanks")); errText != "" {
		t.Fatal(errText)
	}
	out, _ := callTool[startOutput](t, f, "inicio", nil)
	if len(out.Pendencias) != 1 || out.Pendencias[0].Pedido != "12385475" || out.Pendencias[0].TotalBRL != 28.98 || out.Pendencias[0].Comprador != "Nicolas" {
		t.Fatalf("pendência devia voltar uma vez: %+v", out.Pendencias)
	}
	if _, errText := callTool[sellOutput](t, f, "dar_baixa", map[string]any{
		"jogo": "onepiece", "id": "h2", "qty": 1, "total_brl": 28.98, "data": "2026-11-03", "comprador": "Nicolas", "canal": "liga", "pedido": "12385475",
	}); errText != "" {
		t.Fatal(errText)
	}
	out, _ = callTool[startOutput](t, f, "inicio", nil)
	if len(out.Pendencias) != 0 {
		t.Fatalf("pendência baixada devia sumir: %+v", out.Pendencias)
	}
}

func TestPendingExpires(t *testing.T) {
	f := newFixture(t)
	old := pendingItem{Pedido: "1", Jogo: "onepiece", Nome: "Brook", Numero: "OP15-032", Criada: f.app.now().Add(-15 * 24 * time.Hour)}
	if err := f.app.pending.add(old); err != nil {
		t.Fatal(err)
	}
	out, _ := callTool[startOutput](t, f, "inicio", nil)
	if len(out.Pendencias) != 0 {
		t.Fatalf("pendência de 15 dias devia expirar: %+v", out.Pendencias)
	}
	if _, errText := callTool[okOutput](t, f, "registrar_pendencia", map[string]any{
		"jogo": "onepiece", "pedido": "1", "nome": "Brook", "qty": 1, "total_brl": 1.0, "data": "2026-11-03", "comprador": "N", "canal": "liga", "motivo": " ",
	}); !strings.Contains(errText, "motivo") {
		t.Fatalf("motivo vazio devia falhar: %q", errText)
	}
}
