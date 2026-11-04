package main

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"opdeals/internal/postsale"
	"opdeals/internal/whatsapp"
)

//go:embed instructions.md
var instructions string

const serverInstructions = "Você é o atendente da Collecta TCG no WhatsApp e na DM do Instagram. Antes de qualquer outra coisa, chame a ferramenta modo da collecta-loja. O campo regras do resultado traz as regras da rodada: leia inteiras e siga todas. As regras de cada caso chegam no campo regras da ferramenta do caso: sempre que vier, leia e siga. Sem as regras não abra conversa nem mande mensagem."

const (
	catalogTTL    = 60 * time.Second
	highlightMax  = 3
	searchLimit   = 8
	soldOutLimit  = 4
	historyWindow = 7 * 24 * time.Hour
)

var saoPaulo = mustLocation("America/Sao_Paulo")

var rulesVersion = func() string {
	sum := sha256.Sum256([]byte(instructions))
	return hex.EncodeToString(sum[:])[:8]
}()

func mustLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

type catalogItem struct {
	SKU       string  `json:"sku"`
	Game      string  `json:"game"`
	Featured  bool    `json:"featured"`
	AddedAt   string  `json:"addedAt"`
	Slug      string  `json:"slug"`
	GameLabel string  `json:"gameLabel"`
	Kind      string  `json:"kind"`
	Number    string  `json:"number"`
	Name      string  `json:"name"`
	Set       string  `json:"set"`
	Variant   string  `json:"variant"`
	Condition string  `json:"condition"`
	Qty       int     `json:"qty"`
	AskBRL    float64 `json:"askBRL"`
}

type catalog struct {
	Items   []catalogItem `json:"items"`
	SoldOut []catalogItem `json:"soldOut"`
}

type app struct {
	cfg      config
	client   *http.Client
	now      func() time.Time
	history  *history
	mu       sync.Mutex
	catalog  catalog
	loadedAt time.Time
	pending  map[string]outboxItem
}

func newApp(cfg config, client *http.Client, now func() time.Time) *app {
	return &app{cfg: cfg, client: client, now: now, history: newHistory(cfg.DataDir), pending: map[string]outboxItem{}}
}

func (a *app) server() *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{Name: "collecta-loja", Version: "1.0.0"}, &mcp.ServerOptions{Instructions: serverInstructions})
	mcp.AddTool(s, &mcp.Tool{
		Name:        "modo",
		Description: "Diz o modo do atendente (rascunho, teste ou ao_vivo), a data e hora em Goiânia, a versão das regras e as regras da rodada (siga todas). Chame no começo de cada rodada, antes de qualquer outra coisa.",
	}, a.toolMode)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "buscar_estoque",
		Description: "Busca cartas e produtos à venda no site da Collecta por nome, número (ex.: OP01-016) ou coleção. Devolve os itens exatos, os parecidos e os esgotados que batem com a busca, cada um com preço em reais, quantidade disponível e o link para comprar.",
	}, a.toolSearch)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "destaques",
		Description: "Até 3 produtos para recomendar de um jogo (onepiece, pokemon, riftbound, lorcana ou gundam): primeiro os que o dono marcou como destaque, depois os que chegaram por último. Cada um vem com o motivo, preço e link.",
	}, a.toolHighlights)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "novidades",
		Description: "As novidades que valem hoje, escritas pelo dono (situação e horário da loja física, pré-vendas, cupons, promoções, produtos que vão chegar), e as regras de conversa no campo regras. Chame uma vez na rodada antes de responder qualquer cliente.",
	}, a.toolNews)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "regras",
		Description: "Devolve as regras de um caso: instagram (quando há DM não lida), carrinhos (quando chega a mensagem [CARRINHOS]), conversa ou posvenda.",
	}, a.toolRules)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "registrar_origem",
		Description: "Anota como um cliente novo achou a loja (Instagram, Google, indicação...). Chame quando a pessoa responder. Se vier ja_registrado = true, essa pessoa já tinha respondido antes.",
	}, a.toolOrigin)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "autorizar_envio",
		Description: "Obrigatório antes de digitar e enviar QUALQUER mensagem para um cliente. Informe o telefone e o texto exato. Só envie se a resposta trouxer enviar = true; caso contrário siga o motivo.",
	}, a.toolAuthorize)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "historico_bot",
		Description: "Lista as mensagens que o atendente automático mandou para este telefone nos últimos 7 dias. Mensagem enviada pela loja que não está aqui foi escrita por uma pessoa da loja.",
	}, a.toolHistory)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "chamar_humano",
		Description: "Avisa o dono da loja no celular para assumir a conversa. Use nos casos listados nas instruções.",
	}, a.toolHandoff)
	if a.cfg.AgentToken != "" {
		a.addPostSaleTools(s)
	}
	return s
}

type noInput struct{}

type modeOutput struct {
	Versao string `json:"versao_regras"`
	Modo   string `json:"modo"`
	Agora  string `json:"agora"`
	Regras string `json:"regras"`
}

var weekdays = [...]string{"domingo", "segunda", "terça", "quarta", "quinta", "sexta", "sábado"}

func (a *app) toolMode(_ context.Context, _ *mcp.CallToolRequest, _ noInput) (*mcp.CallToolResult, modeOutput, error) {
	now := a.now().In(saoPaulo)
	return nil, modeOutput{
		Versao: rulesVersion,
		Modo:   a.cfg.Mode,
		Agora:  fmt.Sprintf("%s %s", weekdays[now.Weekday()], now.Format("02/01 15:04")),
		Regras: rules[rulesRound],
	}, nil
}

type searchInput struct {
	Consulta string `json:"consulta" jsonschema:"o que o cliente procura, sem o nome do jogo. Ex.: Shanks OP09, OP01-016, Booster Box Destined Rivals"`
}

type product struct {
	SKU        string `json:"sku"`
	Nome       string `json:"nome"`
	Jogo       string `json:"jogo"`
	Colecao    string `json:"colecao,omitempty"`
	Numero     string `json:"numero,omitempty"`
	Variante   string `json:"variante,omitempty"`
	Condicao   string `json:"condicao,omitempty"`
	Preco      string `json:"preco"`
	Disponivel int    `json:"disponivel"`
	Envelope   bool   `json:"cabe_no_envelope"`
	Link       string `json:"link"`
}

type searchOutput struct {
	Exatos    []product `json:"exatos"`
	Parecidos []product `json:"parecidos"`
	Esgotados []product `json:"esgotados"`
}

func (a *app) toolSearch(ctx context.Context, _ *mcp.CallToolRequest, in searchInput) (*mcp.CallToolResult, searchOutput, error) {
	if strings.TrimSpace(in.Consulta) == "" {
		return nil, searchOutput{}, errors.New("consulta vazia")
	}
	c, err := a.loadCatalog(ctx)
	if err != nil {
		return nil, searchOutput{}, fmt.Errorf("não consegui ler o estoque agora (%v): chame o humano se o cliente precisar de preço", err)
	}
	exact, similar := a.find(c.Items, in.Consulta, searchLimit)
	soldOut, _ := a.find(c.SoldOut, in.Consulta, soldOutLimit)
	return nil, searchOutput{Exatos: exact, Parecidos: similar, Esgotados: soldOut}, nil
}

func (a *app) loadCatalog(ctx context.Context) (catalog, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.loadedAt.IsZero() && a.now().Sub(a.loadedAt) < catalogTTL {
		return a.catalog, nil
	}
	fresh, err := a.fetchCatalog(ctx)
	if err != nil {
		if !a.loadedAt.IsZero() {
			return a.catalog, nil
		}
		return catalog{}, err
	}
	a.catalog = fresh
	a.loadedAt = a.now()
	return fresh, nil
}

func (a *app) fetchCatalog(ctx context.Context) (catalog, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.cfg.OpdealsURL+"/api/storefront", nil)
	if err != nil {
		return catalog{}, err
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return catalog{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return catalog{}, fmt.Errorf("vitrine respondeu %d", resp.StatusCode)
	}
	var c catalog
	if err := json.NewDecoder(resp.Body).Decode(&c); err != nil {
		return catalog{}, err
	}
	return c, nil
}

func (a *app) find(list []catalogItem, query string, limit int) ([]product, []product) {
	bySKU := make(map[string]catalogItem, len(list))
	items := make([]postsale.Item, 0, len(list))
	for _, it := range list {
		bySKU[it.SKU] = it
		items = append(items, postsale.Item{
			SKU:       it.SKU,
			Game:      it.GameLabel,
			Kind:      it.Kind,
			Name:      it.Name,
			Set:       it.Set,
			Number:    it.Number,
			Meta:      joinNonEmpty(it.Variant, it.Condition, it.GameLabel),
			UnitBRL:   it.AskBRL,
			Available: it.Qty,
		})
	}
	exact, similar := postsale.Search(items, query, limit)
	return a.products(exact, bySKU), a.products(similar, bySKU)
}

func (a *app) products(items []postsale.Item, bySKU map[string]catalogItem) []product {
	out := make([]product, 0, len(items))
	for _, it := range items {
		c := bySKU[it.SKU]
		out = append(out, product{
			SKU:        c.SKU,
			Nome:       c.Name,
			Jogo:       c.GameLabel,
			Colecao:    c.Set,
			Numero:     c.Number,
			Variante:   c.Variant,
			Condicao:   c.Condition,
			Preco:      "R$ " + postsale.FormatBRL(c.AskBRL),
			Disponivel: c.Qty,
			Envelope:   c.Kind == "",
			Link:       a.link(c),
		})
	}
	return out
}

func (a *app) link(it catalogItem) string {
	path := "carta"
	switch it.Kind {
	case "sealed":
		path = "selado"
	case "accessory":
		path = "acessorio"
	}
	return a.cfg.SiteURL + "/" + path + "/" + it.Slug
}

func joinNonEmpty(parts ...string) string {
	kept := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			kept = append(kept, p)
		}
	}
	return strings.Join(kept, " · ")
}

type highlightsInput struct {
	Jogo string `json:"jogo" jsonschema:"onepiece, pokemon, riftbound, lorcana ou gundam"`
}

type highlight struct {
	product
	Motivo string `json:"motivo"`
}

type highlightsOutput struct {
	Itens []highlight `json:"itens"`
}

func gameID(name string) string {
	return strings.NewReplacer(" ", "", "-", "", "_", "").Replace(postsale.Fold(strings.TrimSpace(name)))
}

func (a *app) toolHighlights(ctx context.Context, _ *mcp.CallToolRequest, in highlightsInput) (*mcp.CallToolResult, highlightsOutput, error) {
	game := gameID(in.Jogo)
	c, err := a.loadCatalog(ctx)
	if err != nil {
		return nil, highlightsOutput{}, fmt.Errorf("não consegui ler o estoque agora (%v)", err)
	}
	var featured, rest []catalogItem
	for _, it := range c.Items {
		if it.Game != game || it.Qty <= 0 {
			continue
		}
		if it.Featured {
			featured = append(featured, it)
		} else {
			rest = append(rest, it)
		}
	}
	sort.SliceStable(rest, func(i, j int) bool { return rest[i].AddedAt > rest[j].AddedAt })
	out := highlightsOutput{Itens: []highlight{}}
	bySKU := map[string]catalogItem{}
	for _, group := range []struct {
		items  []catalogItem
		motivo string
	}{{featured, "destaque da loja"}, {rest, "chegou agora"}} {
		for _, it := range group.items {
			if len(out.Itens) == highlightMax {
				return nil, out, nil
			}
			bySKU[it.SKU] = it
			views := a.products([]postsale.Item{{SKU: it.SKU}}, bySKU)
			out.Itens = append(out.Itens, highlight{product: views[0], Motivo: group.motivo})
		}
	}
	return nil, out, nil
}

type authorizeInput struct {
	Telefone string `json:"telefone" jsonschema:"telefone do cliente com DDD, como aparece no WhatsApp (ex.: +55 62 99999-9999), ou ig: seguido do usuário no Instagram (ex.: ig:collecta.fan)"`
	Texto    string `json:"texto" jsonschema:"o texto exato que você vai enviar"`
}

type authorizeOutput struct {
	Enviar bool   `json:"enviar"`
	Motivo string `json:"motivo"`
}

var instagramHandle = regexp.MustCompile(`^[a-z0-9._]{1,30}$`)

func phoneKey(phone string) (string, error) {
	raw := strings.TrimSpace(phone)
	if rest, ok := strings.CutPrefix(strings.ToLower(raw), "ig:"); ok {
		handle := strings.TrimPrefix(strings.TrimSpace(rest), "@")
		if !instagramHandle.MatchString(handle) {
			return "", fmt.Errorf("usuário do Instagram inválido: %q (use ig: seguido do @ que aparece no topo da conversa, ex.: ig:collecta.fan)", phone)
		}
		return "ig:" + handle, nil
	}
	key := whatsapp.PhoneKey(raw)
	if len(key) < 10 {
		return "", fmt.Errorf("telefone inválido: %q (abra os dados do contato e copie o número com DDD; no Instagram use ig:usuario)", phone)
	}
	return key, nil
}

func (a *app) toolAuthorize(ctx context.Context, _ *mcp.CallToolRequest, in authorizeInput) (*mcp.CallToolResult, authorizeOutput, error) {
	key, err := phoneKey(in.Telefone)
	if err != nil {
		return nil, authorizeOutput{}, err
	}
	text := strings.TrimSpace(in.Texto)
	if text == "" {
		return nil, authorizeOutput{}, errors.New("texto vazio")
	}
	if strings.ContainsAny(text, "—–") {
		return nil, authorizeOutput{Motivo: "troque o travessão por vírgula ou ponto e peça autorização de novo"}, nil
	}
	switch a.cfg.Mode {
	case modeDraft:
		if err := a.notify(ctx, "Rascunho do atendente", fmt.Sprintf("Para %s:\n%s", in.Telefone, text), "default"); err != nil {
			return nil, authorizeOutput{}, err
		}
		return nil, authorizeOutput{Motivo: "modo rascunho: NÃO envie. O texto foi para o dono. Marque a conversa como não lida e siga para a próxima."}, nil
	case modeTest:
		if !a.cfg.TestPhones[key] {
			return nil, authorizeOutput{Motivo: "modo teste: este número não é de teste. NÃO responda, marque a conversa como não lida e siga para a próxima."}, nil
		}
	}
	if err := a.history.add(key, text, a.now()); err != nil {
		return nil, authorizeOutput{}, fmt.Errorf("não consegui registrar o envio: %v", err)
	}
	return nil, authorizeOutput{Enviar: true, Motivo: "pode enviar"}, nil
}

type historyInput struct {
	Telefone string `json:"telefone" jsonschema:"telefone do cliente com DDD, ou ig:usuario no Instagram"`
}

type historyEntry struct {
	Texto string `json:"texto"`
	Em    string `json:"em"`
}

type historyOutput struct {
	Mensagens []historyEntry `json:"mensagens"`
}

func (a *app) toolHistory(_ context.Context, _ *mcp.CallToolRequest, in historyInput) (*mcp.CallToolResult, historyOutput, error) {
	key, err := phoneKey(in.Telefone)
	if err != nil {
		return nil, historyOutput{}, err
	}
	sent, err := a.history.since(key, a.now().Add(-historyWindow))
	if err != nil {
		return nil, historyOutput{}, err
	}
	out := historyOutput{Mensagens: make([]historyEntry, 0, len(sent))}
	for _, m := range sent {
		out.Mensagens = append(out.Mensagens, historyEntry{Texto: m.Text, Em: m.At.In(saoPaulo).Format("02/01 15:04")})
	}
	return nil, out, nil
}

type handoffInput struct {
	Telefone string `json:"telefone" jsonschema:"telefone do cliente com DDD, ou ig:usuario no Instagram"`
	Nome     string `json:"nome,omitempty" jsonschema:"nome do cliente, se aparecer"`
	Motivo   string `json:"motivo" jsonschema:"por que a conversa precisa de uma pessoa"`
	Resumo   string `json:"resumo" jsonschema:"resumo curto do que o cliente quer"`
}

type handoffOutput struct {
	OK      bool   `json:"ok"`
	Proximo string `json:"proximo"`
}

func (a *app) toolHandoff(ctx context.Context, _ *mcp.CallToolRequest, in handoffInput) (*mcp.CallToolResult, handoffOutput, error) {
	if strings.TrimSpace(in.Motivo) == "" {
		return nil, handoffOutput{}, errors.New("informe o motivo")
	}
	who := strings.TrimSpace(in.Nome + " " + in.Telefone)
	body := fmt.Sprintf("%s: %s\n%s", who, strings.TrimSpace(in.Motivo), strings.TrimSpace(in.Resumo))
	if err := a.notify(ctx, "Atendente: assumir conversa", body, "high"); err != nil {
		return nil, handoffOutput{}, err
	}
	return nil, handoffOutput{OK: true, Proximo: "ponha a conversa na lista Humano (Adicionar à lista, no cabeçalho) e marque como não lida"}, nil
}

func (a *app) notify(ctx context.Context, title, body, priority string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.cfg.NtfyURL, strings.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Title", title)
	req.Header.Set("Tags", "speech_balloon")
	req.Header.Set("Priority", priority)
	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("não consegui avisar o dono: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("não consegui avisar o dono: ntfy respondeu %d", resp.StatusCode)
	}
	return nil
}
