package main

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"opdeals/internal/postsale"
	"opdeals/internal/trades"
)

//go:embed instructions.md
var instructions string

const serverInstructions = "Você é a rodada de baixa de vendas da Collecta TCG. Antes de qualquer outra coisa, chame a ferramenta inicio da collecta-baixa. O campo regras do resultado traz as regras completas da rodada: leia inteiras e siga todas, do começo ao fim. Sem as regras não navegue nem grave nada. Termine sempre chamando relatorio, mesmo sem venda nova."

const (
	recentWindow = 45 * 24 * time.Hour
	searchLimit  = 30
	summaryMax   = 120
	pushMax      = 3000
)

var saoPaulo = mustLocation("America/Sao_Paulo")

var rulesVersion = func() string {
	sum := sha256.Sum256([]byte(instructions))
	return hex.EncodeToString(sum[:])[:8]
}()

var (
	ligaOrderRe  = regexp.MustCompile(`Liga #(\d+)`)
	channelLabel = map[string]string{trades.ChannelLiga: "Liga"}
	cpfRe        = regexp.MustCompile(`\b\d{3}\.\d{3}\.\d{3}-\d{2}\b|\b\d{11}\b`)
	emailRe      = regexp.MustCompile(`[\w.+-]+@[\w-]+\.[\w.]+`)
	phoneRe      = regexp.MustCompile(`\(\d{2}\)\s?\d{4,5}-?\d{4}|\+55\s?\d{2}|\b9\d{4}-\d{4}\b`)
	urgentRe     = regexp.MustCompile(`\b[1-9]\d* URGENTE`)
)

func mustLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

type app struct {
	cfg         config
	client      *http.Client
	now         func() time.Time
	pending     *pendingStore
	urgent      *urgentStore
	mu          sync.Mutex
	stamp       string
	withSale    map[string]bool
	urgentCalls int
	urgentNew   int
}

func newApp(cfg config, client *http.Client, now func() time.Time) *app {
	return &app{cfg: cfg, client: client, now: now, pending: newPendingStore(cfg.DataDir), urgent: newUrgentStore(cfg.DataDir), withSale: map[string]bool{}}
}

func (a *app) server() *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{Name: "collecta-baixa", Version: "1.0.0"}, &mcp.ServerOptions{Instructions: serverInstructions})
	mcp.AddTool(s, &mcp.Tool{
		Name:        "inicio",
		Description: "Chame primeiro em toda rodada. Devolve o carimbo da rodada, a hora em Goiânia, a versão e o texto completo das regras (siga todas), os números de pedido da Liga que já têm baixa no dashboard e os itens de pedidos que ficaram sem baixa em rodadas anteriores (pendencias_abertas).",
	}, a.toolStart)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "registrar_pendencia",
		Description: "Guarda um item de pedido pago da Liga que ficou sem baixa nesta rodada, com os dados da venda, para a próxima rodada tentar de novo. Chame para todo item não baixado de pedido pago. Some sozinho quando o item recebe a baixa com Liga #pedido ou depois de 14 dias.",
	}, a.toolAddPending)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "estoque_buscar",
		Description: "Procura no estoque do dashboard por número da carta (OP15-119, 098/086) ou nome. Devolve os registros em estoque e os vendidos nos últimos 45 dias (para achar baixas manuais do dono).",
	}, a.toolSearch)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "dar_baixa",
		Description: "Dá baixa de um item vendido na Liga num registro em estoque. Recusa pedido que já baixou esse item e quantidade acima da disponível.",
	}, a.toolSell)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "venda_sem_lastro",
		Description: "Registra uma venda confirmada da Liga que não tem nenhum registro no estoque do dashboard. Use só depois de estoque_buscar não achar o item por número nem por nome.",
	}, a.toolSellUnbacked)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "flag_liga",
		Description: "Atualiza a flag Na Liga (quantidade e preço anunciados) dos registros que sobraram depois de uma baixa. Mande só os itens afetados.",
	}, a.toolLigaFlag)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "fila_sincronizar",
		Description: "Lista as tarefas pendentes da fila Sincronizar: saídas e voltas de estoque que precisam ser repetidas na Liga.",
	}, a.toolQueue)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "fila_feita",
		Description: "Marca uma tarefa da fila Sincronizar como feita, depois de conferir a Liga.",
	}, a.toolQueueDone)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "fila_erro",
		Description: "Marca uma tarefa da fila Sincronizar como erro, com o motivo. Use quando não achou a oferta, achou mais de uma ou a Liga já estava abaixo do esperado.",
	}, a.toolQueueError)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "posvenda_liga",
		Description: "Manda para o pós-venda os pedidos da Liga pagos nas últimas 24h e ainda não retirados, e os cancelados pelo Sistema sem pagamento nas últimas 24h. Só primeiro nome, telefone, itens, data do pagamento ou do cancelamento, retirada e enviado.",
	}, a.toolPostSale)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "registrar_urgente",
		Description: "Registra um URGENTE da rodada (pedido cancelado depois da baixa, baixa recusada por falta de estoque num item que a Liga ainda anuncia). Chame para cada URGENTE antes do relatorio. Se vier ja_avisado = true, o dono já recebeu esse aviso: ele vai na linha de já avisados, não como URGENTE novo.",
	}, a.toolUrgent)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "relatorio",
		Description: "Fecha a rodada: grava o relatório e manda o resumo para o celular do dono. Chame uma vez no fim de toda rodada, mesmo sem venda nova ou com canal bloqueado.",
	}, a.toolReport)
	return s
}

func (a *app) currentStamp() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.stamp == "" {
		a.stamp = a.now().In(saoPaulo).Format("2006-01-02-1504")
	}
	return a.stamp
}

type noInput struct{}

type startOutput struct {
	Carimbo     string        `json:"carimbo"`
	Agora       string        `json:"agora"`
	Versao      string        `json:"versao_regras"`
	Regras      string        `json:"regras"`
	PedidosLiga []string      `json:"pedidos_liga"`
	Pendencias  []pendingItem `json:"pendencias_abertas"`
	Urgentes    []urgentItem  `json:"urgentes_ja_avisados"`
}

func (a *app) toolStart(ctx context.Context, _ *mcp.CallToolRequest, _ noInput) (*mcp.CallToolResult, startOutput, error) {
	a.mu.Lock()
	a.stamp = ""
	a.urgentCalls = 0
	a.urgentNew = 0
	a.mu.Unlock()
	done := map[string]bool{}
	ledgers := map[string][]trades.Trade{}
	for _, g := range games {
		all, err := a.ledger(ctx, g)
		if err != nil {
			return nil, startOutput{}, fmt.Errorf("não consegui ler o estoque de %s: %v", g, err)
		}
		ledgers[g] = all
		for _, t := range all {
			if t.Status != "sold" {
				continue
			}
			for _, m := range ligaOrderRe.FindAllStringSubmatch(t.Buyer, -1) {
				done[m[1]] = true
			}
		}
	}
	open, err := a.pending.open(a.now(), ledgers)
	if err != nil {
		return nil, startOutput{}, fmt.Errorf("não consegui ler as pendências: %v", err)
	}
	urgents, err := a.urgent.prune(a.now(), done)
	if err != nil {
		return nil, startOutput{}, fmt.Errorf("não consegui ler os urgentes: %v", err)
	}
	a.mu.Lock()
	a.withSale = done
	a.mu.Unlock()
	return nil, startOutput{
		Pendencias:  open,
		Urgentes:    urgents,
		Carimbo:     a.currentStamp(),
		Agora:       a.now().In(saoPaulo).Format("02/01/2006 15:04"),
		Versao:      rulesVersion,
		Regras:      instructions,
		PedidosLiga: sortedKeys(done),
	}, nil
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

type pendingInput struct {
	saleFields
	Nome   string `json:"nome" jsonschema:"nome da carta ou do produto como no pedido"`
	Numero string `json:"numero,omitempty" jsonschema:"número da carta, se tiver"`
	Motivo string `json:"motivo" jsonschema:"por que não deu baixa"`
}

func (a *app) toolAddPending(_ context.Context, _ *mcp.CallToolRequest, in pendingInput) (*mcp.CallToolResult, okOutput, error) {
	if _, err := in.check(); err != nil {
		return nil, okOutput{}, err
	}
	if cleanText(in.Nome) == "" || cleanText(in.Motivo) == "" {
		return nil, okOutput{}, errors.New("informe o nome do item e o motivo")
	}
	item := pendingItem{
		Pedido: strings.TrimPrefix(strings.TrimSpace(in.Pedido), "#"), Jogo: in.Jogo, Numero: cleanText(in.Numero), Nome: cleanText(in.Nome),
		Qty: in.Qty, TotalBRL: in.TotalBRL, Data: in.Data, Comprador: cleanText(in.Comprador), Motivo: cleanText(in.Motivo), Criada: a.now(),
	}
	if err := a.pending.add(item); err != nil {
		return nil, okOutput{}, fmt.Errorf("não consegui guardar a pendência: %v", err)
	}
	return nil, okOutput{OK: true}, nil
}

type searchInput struct {
	Jogo     string `json:"jogo" jsonschema:"onepiece, pokemon ou riftbound"`
	Consulta string `json:"consulta" jsonschema:"número da carta (OP15-119, 098/086) ou parte do nome"`
}

type holdingView struct {
	ID           string  `json:"id"`
	Nome         string  `json:"nome"`
	Numero       string  `json:"numero,omitempty"`
	Colecao      string  `json:"colecao,omitempty"`
	Variante     string  `json:"variante,omitempty"`
	Condicao     string  `json:"condicao,omitempty"`
	Tipo         string  `json:"tipo,omitempty"`
	Qty          int     `json:"qty"`
	PrecoBRL     float64 `json:"preco_brl"`
	NaLiga       bool    `json:"na_liga"`
	LigaQty      int     `json:"liga_qty"`
	LigaPrecoBRL float64 `json:"liga_preco_brl"`
}

type soldView struct {
	ID        string  `json:"id"`
	Nome      string  `json:"nome"`
	Numero    string  `json:"numero,omitempty"`
	Variante  string  `json:"variante,omitempty"`
	Qty       int     `json:"qty"`
	TotalBRL  float64 `json:"total_brl"`
	Data      string  `json:"data"`
	Comprador string  `json:"comprador"`
	Canal     string  `json:"canal,omitempty"`
}

type searchOutput struct {
	EmEstoque        []holdingView `json:"em_estoque"`
	VendidosRecentes []soldView    `json:"vendidos_recentes"`
}

func compact(s string) string {
	return strings.Join(strings.Fields(postsale.Fold(s)), "")
}

func matches(t trades.Trade, q string) bool {
	if q == "" {
		return false
	}
	return strings.Contains(compact(t.Number), q) || strings.Contains(compact(t.Name), q)
}

func (a *app) soldRecently(t trades.Trade) bool {
	at := t.UpdatedAt
	if d, err := time.ParseInLocation(time.DateOnly, t.SellDate, saoPaulo); err == nil {
		at = d
	}
	return a.now().Sub(at) <= recentWindow
}

func holdingOf(t trades.Trade) holdingView {
	return holdingView{
		ID: t.ID, Nome: t.Name, Numero: t.Number, Colecao: t.Set, Variante: t.Variant, Condicao: t.Condition,
		Tipo: t.Kind, Qty: max(t.Qty, 1), PrecoBRL: t.AskBRL, NaLiga: t.LigaListed, LigaQty: t.LigaQty, LigaPrecoBRL: t.LigaPriceBRL,
	}
}

func (a *app) toolSearch(ctx context.Context, _ *mcp.CallToolRequest, in searchInput) (*mcp.CallToolResult, searchOutput, error) {
	if !validGame(in.Jogo) {
		return nil, searchOutput{}, fmt.Errorf("jogo inválido: %q (use onepiece, pokemon ou riftbound)", in.Jogo)
	}
	q := compact(strings.TrimPrefix(strings.TrimSpace(in.Consulta), "#"))
	if q == "" {
		return nil, searchOutput{}, errors.New("consulta vazia")
	}
	all, err := a.ledger(ctx, in.Jogo)
	if err != nil {
		return nil, searchOutput{}, err
	}
	out := searchOutput{EmEstoque: []holdingView{}, VendidosRecentes: []soldView{}}
	for _, t := range all {
		if !matches(t, q) {
			continue
		}
		switch {
		case t.Status == "sold" && a.soldRecently(t) && len(out.VendidosRecentes) < searchLimit:
			out.VendidosRecentes = append(out.VendidosRecentes, soldView{
				ID: t.ID, Nome: t.Name, Numero: t.Number, Variante: t.Variant, Qty: max(t.Qty, 1),
				TotalBRL: t.SellPrice, Data: t.SellDate, Comprador: t.Buyer, Canal: t.Channel,
			})
		case t.Status != "sold" && len(out.EmEstoque) < searchLimit:
			out.EmEstoque = append(out.EmEstoque, holdingOf(t))
		}
	}
	return nil, out, nil
}

type saleFields struct {
	Jogo      string  `json:"jogo" jsonschema:"onepiece, pokemon ou riftbound"`
	Qty       int     `json:"qty" jsonschema:"unidades vendidas no pedido"`
	TotalBRL  float64 `json:"total_brl" jsonschema:"total bruto do item: preço unitário × qty, sem frete e sem taxa"`
	Data      string  `json:"data" jsonschema:"data do pedido, AAAA-MM-DD"`
	Comprador string  `json:"comprador" jsonschema:"primeiro nome do comprador"`
	Canal     string  `json:"canal" jsonschema:"liga"`
	Pedido    string  `json:"pedido" jsonschema:"número do pedido, só dígitos"`
}

func (f saleFields) check() (string, error) {
	if !validGame(f.Jogo) {
		return "", fmt.Errorf("jogo inválido: %q (use onepiece, pokemon ou riftbound)", f.Jogo)
	}
	label, ok := channelLabel[f.Canal]
	if !ok {
		return "", fmt.Errorf("canal inválido: %q (só liga: o MyPCards foi desativado)", f.Canal)
	}
	pedido := strings.TrimPrefix(strings.TrimSpace(f.Pedido), "#")
	if pedido == "" || strings.Trim(pedido, "0123456789") != "" {
		return "", fmt.Errorf("pedido inválido: %q (só os dígitos do número do pedido)", f.Pedido)
	}
	if f.Qty < 1 {
		return "", errors.New("qty precisa ser pelo menos 1")
	}
	if f.TotalBRL < 0 {
		return "", errors.New("total_brl não pode ser negativo")
	}
	if _, err := time.Parse(time.DateOnly, f.Data); err != nil {
		return "", fmt.Errorf("data inválida: %q (use AAAA-MM-DD)", f.Data)
	}
	comprador := strings.Join(strings.Fields(f.Comprador), " ")
	if comprador == "" {
		return "", errors.New("informe o comprador")
	}
	return comprador + " — " + label + " #" + pedido, nil
}

func (f saleFields) tag() string {
	return channelLabel[f.Canal] + " #" + strings.TrimPrefix(strings.TrimSpace(f.Pedido), "#")
}

func sameCard(a, b trades.Trade) bool {
	return compact(a.Name) == compact(b.Name) && compact(a.Number) == compact(b.Number) && compact(a.Variant) == compact(b.Variant)
}

func alreadyWrittenOff(all []trades.Trade, card trades.Trade, tag string) bool {
	re := regexp.MustCompile(regexp.QuoteMeta(tag) + `\b`)
	for _, t := range all {
		if t.Status == "sold" && re.MatchString(t.Buyer) && sameCard(t, card) {
			return true
		}
	}
	return false
}

type sellInput struct {
	saleFields
	ID string `json:"id" jsonschema:"id do registro em estoque, vindo de estoque_buscar"`
}

type sellOutput struct {
	VendidoID    string  `json:"vendido_id"`
	Comprador    string  `json:"comprador"`
	Qty          int     `json:"qty"`
	TotalBRL     float64 `json:"total_brl"`
	RestanteID   string  `json:"restante_id,omitempty"`
	RestanteQty  int     `json:"restante_qty"`
	NaLiga       bool    `json:"na_liga"`
	LigaQty      int     `json:"liga_qty"`
	LigaPrecoBRL float64 `json:"liga_preco_brl"`
}

func (a *app) toolSell(ctx context.Context, _ *mcp.CallToolRequest, in sellInput) (*mcp.CallToolResult, sellOutput, error) {
	buyer, err := in.check()
	if err != nil {
		return nil, sellOutput{}, err
	}
	all, err := a.ledger(ctx, in.Jogo)
	if err != nil {
		return nil, sellOutput{}, err
	}
	var holding *trades.Trade
	for i := range all {
		if all[i].ID == in.ID {
			holding = &all[i]
		}
	}
	switch {
	case holding == nil:
		return nil, sellOutput{}, fmt.Errorf("registro %s não existe em %s: busque de novo com estoque_buscar", in.ID, in.Jogo)
	case holding.Status == "sold":
		return nil, sellOutput{}, fmt.Errorf("registro %s já está vendido: use um registro de em_estoque", in.ID)
	case alreadyWrittenOff(all, *holding, in.tag()):
		return nil, sellOutput{}, fmt.Errorf("o pedido %s já tem baixa de %s %s: não baixe de novo", in.tag(), holding.Name, holding.Number)
	case in.Qty > max(holding.Qty, 1):
		return nil, sellOutput{}, fmt.Errorf("qty %d maior que o estoque do registro (%d): procure outro registro do mesmo item ou registre pendência", in.Qty, max(holding.Qty, 1))
	}
	sold, err := a.sell(ctx, in.Jogo, in.ID, sale{
		Qty: in.Qty, SellPrice: in.TotalBRL, SellCurrency: "BRL", SellDate: in.Data, Buyer: buyer, Channel: in.Canal,
	})
	if err != nil {
		return nil, sellOutput{}, err
	}
	out := sellOutput{
		VendidoID: sold.ID, Comprador: sold.Buyer, Qty: max(sold.Qty, 1), TotalBRL: sold.SellPrice,
		RestanteQty: max(holding.Qty, 1) - in.Qty,
		NaLiga:      holding.LigaListed, LigaQty: holding.LigaQty, LigaPrecoBRL: holding.LigaPriceBRL,
	}
	if out.RestanteQty > 0 {
		out.RestanteID = holding.ID
	}
	return nil, out, nil
}

type unbackedInput struct {
	saleFields
	Nome     string `json:"nome" jsonschema:"nome da carta ou do produto como no pedido"`
	Numero   string `json:"numero,omitempty" jsonschema:"número da carta, se tiver"`
	Colecao  string `json:"colecao,omitempty" jsonschema:"coleção ou edição"`
	Variante string `json:"variante,omitempty"`
	Condicao string `json:"condicao,omitempty"`
}

type unbackedOutput struct {
	VendidoID string `json:"vendido_id"`
	Comprador string `json:"comprador"`
}

func (a *app) toolSellUnbacked(ctx context.Context, _ *mcp.CallToolRequest, in unbackedInput) (*mcp.CallToolResult, unbackedOutput, error) {
	buyer, err := in.check()
	if err != nil {
		return nil, unbackedOutput{}, err
	}
	if strings.TrimSpace(in.Nome) == "" {
		return nil, unbackedOutput{}, errors.New("informe o nome do item")
	}
	card := trades.Trade{Name: strings.TrimSpace(in.Nome), Number: strings.TrimSpace(in.Numero), Variant: strings.TrimSpace(in.Variante)}
	all, err := a.ledger(ctx, in.Jogo)
	if err != nil {
		return nil, unbackedOutput{}, err
	}
	if alreadyWrittenOff(all, card, in.tag()) {
		return nil, unbackedOutput{}, fmt.Errorf("o pedido %s já tem baixa de %s %s: não registre de novo", in.tag(), card.Name, card.Number)
	}
	card.Set = strings.TrimSpace(in.Colecao)
	card.Condition = strings.TrimSpace(in.Condicao)
	card.Qty = in.Qty
	card.Status = "sold"
	card.SemLastro = true
	card.SellPrice = in.TotalBRL
	card.SellCurrency = "BRL"
	card.SellDate = in.Data
	card.Buyer = buyer
	card.Channel = in.Canal
	card.Notes = fmt.Sprintf("Venda do %s sem registro no estoque do dashboard", channelLabel[in.Canal])
	created, err := a.createTrade(ctx, in.Jogo, card)
	if err != nil {
		return nil, unbackedOutput{}, err
	}
	return nil, unbackedOutput{VendidoID: created.ID, Comprador: created.Buyer}, nil
}

type ligaFlagItem struct {
	ID           string  `json:"id" jsonschema:"id do registro em estoque"`
	NaLiga       bool    `json:"na_liga" jsonschema:"false quando o item zerou na Liga"`
	LigaQty      int     `json:"liga_qty" jsonschema:"quantidade anunciada na Liga agora"`
	LigaPrecoBRL float64 `json:"liga_preco_brl" jsonschema:"preço unitário anunciado na Liga (não muda com a venda)"`
}

type ligaFlagInput struct {
	Jogo  string         `json:"jogo" jsonschema:"onepiece, pokemon ou riftbound"`
	Itens []ligaFlagItem `json:"itens"`
}

type ligaFlagOutput struct {
	Atualizados int `json:"atualizados"`
}

func (a *app) toolLigaFlag(ctx context.Context, _ *mcp.CallToolRequest, in ligaFlagInput) (*mcp.CallToolResult, ligaFlagOutput, error) {
	if !validGame(in.Jogo) {
		return nil, ligaFlagOutput{}, fmt.Errorf("jogo inválido: %q", in.Jogo)
	}
	if len(in.Itens) == 0 {
		return nil, ligaFlagOutput{}, errors.New("nenhum item")
	}
	all, err := a.ledger(ctx, in.Jogo)
	if err != nil {
		return nil, ligaFlagOutput{}, err
	}
	holdings := map[string]bool{}
	for _, t := range all {
		if t.Status != "sold" {
			holdings[t.ID] = true
		}
	}
	items := make([]ligaState, 0, len(in.Itens))
	for _, it := range in.Itens {
		if !holdings[it.ID] {
			return nil, ligaFlagOutput{}, fmt.Errorf("registro %s não está em estoque em %s: nada foi gravado", it.ID, in.Jogo)
		}
		if it.LigaQty < 0 || it.LigaPrecoBRL < 0 {
			return nil, ligaFlagOutput{}, fmt.Errorf("registro %s: quantidade e preço não podem ser negativos", it.ID)
		}
		items = append(items, ligaState{ID: it.ID, Listed: it.NaLiga, Qty: it.LigaQty, PriceBRL: it.LigaPrecoBRL})
	}
	n, err := a.setLiga(ctx, in.Jogo, items)
	if err != nil {
		return nil, ligaFlagOutput{}, err
	}
	return nil, ligaFlagOutput{Atualizados: n}, nil
}

type taskView struct {
	ID       string `json:"id"`
	Jogo     string `json:"jogo"`
	TradeID  string `json:"trade_id"`
	Nome     string `json:"nome"`
	Numero   string `json:"numero,omitempty"`
	Variante string `json:"variante,omitempty"`
	Condicao string `json:"condicao,omitempty"`
	Alvo     string `json:"alvo,omitempty"`
	Acao     string `json:"acao"`
	Qty      int    `json:"qty"`
	Motivo   string `json:"motivo,omitempty"`
	Criada   string `json:"criada"`
}

type queueOutput struct {
	Tarefas []taskView `json:"tarefas"`
}

func (a *app) toolQueue(ctx context.Context, _ *mcp.CallToolRequest, _ noInput) (*mcp.CallToolResult, queueOutput, error) {
	tasks, err := a.pendingTasks(ctx)
	if err != nil {
		return nil, queueOutput{}, err
	}
	out := queueOutput{Tarefas: make([]taskView, 0, len(tasks))}
	for _, t := range tasks {
		out.Tarefas = append(out.Tarefas, taskView{
			ID: t.ID, Jogo: t.Game, TradeID: t.TradeID, Nome: t.Name, Numero: t.Number, Variante: t.Variant,
			Condicao: t.Condition, Alvo: t.Target, Acao: t.Action, Qty: t.Qty, Motivo: t.Reason,
			Criada: t.CreatedAt.In(saoPaulo).Format("02/01/2006 15:04"),
		})
	}
	return nil, out, nil
}

type taskInput struct {
	ID string `json:"id" jsonschema:"id da tarefa, vindo de fila_sincronizar"`
}

type taskErrorInput struct {
	ID   string `json:"id" jsonschema:"id da tarefa, vindo de fila_sincronizar"`
	Erro string `json:"erro" jsonschema:"por que a tarefa não foi feita"`
}

type okOutput struct {
	OK bool `json:"ok"`
}

func (a *app) toolQueueDone(ctx context.Context, _ *mcp.CallToolRequest, in taskInput) (*mcp.CallToolResult, okOutput, error) {
	if strings.TrimSpace(in.ID) == "" {
		return nil, okOutput{}, errors.New("informe o id da tarefa")
	}
	if err := a.finishTask(ctx, in.ID, ""); err != nil {
		return nil, okOutput{}, err
	}
	return nil, okOutput{OK: true}, nil
}

func (a *app) toolQueueError(ctx context.Context, _ *mcp.CallToolRequest, in taskErrorInput) (*mcp.CallToolResult, okOutput, error) {
	msg := strings.TrimSpace(in.Erro)
	if strings.TrimSpace(in.ID) == "" || msg == "" {
		return nil, okOutput{}, errors.New("informe o id da tarefa e o erro")
	}
	if err := a.finishTask(ctx, in.ID, msg); err != nil {
		return nil, okOutput{}, err
	}
	return nil, okOutput{OK: true}, nil
}

type postSaleItem struct {
	Nome string `json:"nome"`
	Qty  int    `json:"qty"`
}

type postSaleOrder struct {
	Codigo       string         `json:"codigo" jsonschema:"número do pedido da Liga"`
	PrimeiroNome string         `json:"primeiro_nome" jsonschema:"só o primeiro nome"`
	Telefone     string         `json:"telefone" jsonschema:"(DD) NNNNNNNNN, do bloco Endereço do Cliente"`
	PagoEm       string         `json:"pago_em,omitempty" jsonschema:"pedido pago: Data do Pagamento em RFC 3339 com -03:00, ex. 2026-11-02T14:05:00-03:00"`
	CanceladoEm  string         `json:"cancelado_em,omitempty" jsonschema:"pedido cancelado pelo Sistema sem pagamento: hora do 'Status alterado para: Cancelado' no Histórico de Ações, em RFC 3339 com -03:00. Deixe pago_em vazio"`
	Retirada     bool           `json:"retirada"`
	Enviado      bool           `json:"enviado" jsonschema:"true quando o status é Enviado"`
	Itens        []postSaleItem `json:"itens"`
}

type postSaleInput struct {
	Pedidos []postSaleOrder `json:"pedidos"`
}

type postSaleOutput struct {
	Criados   int `json:"criados"`
	Ignorados int `json:"ignorados"`
}

func (a *app) toolPostSale(ctx context.Context, _ *mcp.CallToolRequest, in postSaleInput) (*mcp.CallToolResult, postSaleOutput, error) {
	if len(in.Pedidos) == 0 {
		return nil, postSaleOutput{}, nil
	}
	orders := make([]ligaOrder, 0, len(in.Pedidos))
	for _, p := range in.Pedidos {
		field, value := "pago_em", p.PagoEm
		if p.CanceladoEm != "" {
			if p.PagoEm != "" {
				return nil, postSaleOutput{}, fmt.Errorf("pedido %s: cancelado depois de pago não vai para o pós-venda (mande só pago_em ou só cancelado_em)", p.Codigo)
			}
			field, value = "cancelado_em", p.CanceladoEm
		}
		if _, err := time.Parse(time.RFC3339, value); err != nil {
			return nil, postSaleOutput{}, fmt.Errorf("pedido %s: %s inválido %q (use 2026-11-02T14:05:00-03:00)", p.Codigo, field, value)
		}
		first := strings.Fields(p.PrimeiroNome)
		if len(first) == 0 {
			return nil, postSaleOutput{}, fmt.Errorf("pedido %s: falta o primeiro nome", p.Codigo)
		}
		items := make([]ligaOrderItem, 0, len(p.Itens))
		for _, it := range p.Itens {
			items = append(items, ligaOrderItem{Name: it.Nome, Qty: it.Qty})
		}
		orders = append(orders, ligaOrder{
			Code: p.Codigo, FirstName: first[0], Phone: p.Telefone, PaidAt: p.PagoEm, CancelledAt: p.CanceladoEm,
			Pickup: p.Retirada, Shipped: p.Enviado, Items: items,
		})
	}
	res, err := a.importLigaOrders(ctx, orders)
	if err != nil {
		return nil, postSaleOutput{}, err
	}
	return nil, postSaleOutput{Criados: res.Created, Ignorados: res.Skipped}, nil
}

type urgentInput struct {
	Pedido string `json:"pedido" jsonschema:"número do pedido da Liga, só dígitos"`
	Item   string `json:"item" jsonschema:"o item, como no relatório (ex.: Monkey.D.Luffy OP15-119)"`
	Motivo string `json:"motivo" jsonschema:"por que é urgente (ex.: cancelado depois da baixa)"`
}

type urgentOutput struct {
	JaAvisado bool `json:"ja_avisado"`
}

func (a *app) toolUrgent(_ context.Context, _ *mcp.CallToolRequest, in urgentInput) (*mcp.CallToolResult, urgentOutput, error) {
	pedido := strings.TrimPrefix(strings.TrimSpace(in.Pedido), "#")
	if pedido == "" || strings.Trim(pedido, "0123456789") != "" {
		return nil, urgentOutput{}, fmt.Errorf("pedido inválido: %q (só os dígitos do número do pedido)", in.Pedido)
	}
	if cleanText(in.Item) == "" || cleanText(in.Motivo) == "" {
		return nil, urgentOutput{}, errors.New("informe o item e o motivo")
	}
	a.mu.Lock()
	withSale := a.withSale[pedido]
	a.mu.Unlock()
	known, err := a.urgent.register(urgentItem{Pedido: pedido, Item: cleanText(in.Item), Motivo: cleanText(in.Motivo), ComBaixa: withSale, Criado: a.now()})
	if err != nil {
		return nil, urgentOutput{}, fmt.Errorf("não consegui guardar o urgente: %v", err)
	}
	a.mu.Lock()
	a.urgentCalls++
	if !known {
		a.urgentNew++
	}
	a.mu.Unlock()
	return nil, urgentOutput{JaAvisado: known}, nil
}

func (a *app) priority(summary string) (string, error) {
	a.mu.Lock()
	calls, fresh := a.urgentCalls, a.urgentNew
	a.mu.Unlock()
	remind, err := a.urgent.takeReminder(a.now())
	if err != nil {
		return "", fmt.Errorf("não consegui ler os urgentes: %v", err)
	}
	if remind || fresh > 0 || (calls == 0 && urgentRe.MatchString(summary)) {
		return "high", nil
	}
	return "default", nil
}

type reportInput struct {
	Resumo string `json:"resumo" jsonschema:"uma linha, até 120 caracteres. Ex.: 3 baixas (Liga 2, MyP 1) · 1 pendência · 0 URGENTE, ou nada novo"`
	Corpo  string `json:"corpo" jsonschema:"o relatório em markdown: baixas, pendências (URGENTE primeiro), pedidos pulados, canais que não rodaram"`
}

type reportOutput struct {
	Arquivo string `json:"arquivo"`
	Enviado bool   `json:"enviado"`
}

func personalData(s string) string {
	switch {
	case cpfRe.MatchString(s):
		return "CPF"
	case emailRe.MatchString(s):
		return "e-mail"
	case phoneRe.MatchString(s):
		return "telefone"
	}
	return ""
}

func (a *app) toolReport(ctx context.Context, _ *mcp.CallToolRequest, in reportInput) (*mcp.CallToolResult, reportOutput, error) {
	summary := strings.TrimSpace(in.Resumo)
	switch {
	case summary == "":
		return nil, reportOutput{}, errors.New("resumo vazio")
	case strings.Contains(summary, "\n"):
		return nil, reportOutput{}, errors.New("resumo precisa ser uma linha só")
	case len([]rune(summary)) > summaryMax:
		return nil, reportOutput{}, fmt.Errorf("resumo com %d caracteres: o máximo é %d", len([]rune(summary)), summaryMax)
	}
	body := strings.TrimSpace(in.Corpo)
	if kind := personalData(summary + "\n" + body); kind != "" {
		return nil, reportOutput{}, fmt.Errorf("o relatório tem %s de cliente: tire e mande de novo (só nome ou nick)", kind)
	}
	stamp := a.currentStamp()
	path := filepath.Join(a.cfg.DataDir, stamp+".md")
	if err := os.WriteFile(path, []byte(summary+"\n\n"+body+"\n"), 0o600); err != nil {
		return nil, reportOutput{}, fmt.Errorf("não consegui gravar o relatório: %v", err)
	}
	if err := a.appendRun(stamp, summary); err != nil {
		return nil, reportOutput{}, err
	}
	priority, err := a.priority(summary)
	if err != nil {
		return nil, reportOutput{Arquivo: path}, err
	}
	push := summary
	if body != "" {
		push += "\n\n" + body
	}
	if r := []rune(push); len(r) > pushMax {
		push = string(r[:pushMax])
	}
	if err := a.notify(ctx, "Baixa de vendas", push, priority); err != nil {
		return nil, reportOutput{Arquivo: path}, err
	}
	a.mu.Lock()
	a.stamp = ""
	a.mu.Unlock()
	return nil, reportOutput{Arquivo: path, Enviado: true}, nil
}

func (a *app) appendRun(stamp, summary string) error {
	f, err := os.OpenFile(filepath.Join(a.cfg.DataDir, "runs.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(f, "%s %s\n", stamp, summary); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
