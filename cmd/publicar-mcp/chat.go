package main

// Aprovação pelo chat do OpenBot, no lugar do Aprovar no celular.
// Decisão do Jairo de 23/11/2026: "Ta aprovado, não quero esse fluxo pelo celular não, quero por aqui mesmo".
//
// Liga quando existe <DataDir>/aprovacao-pelo-chat. O arquivo lista, uma por linha, os agentes do
// OpenBot em cujo chat a frase do Jairo vale (vazio: só o CEO Collecta). Apagar o arquivo volta ao celular.
//
// A trava: pedir_aprovacao e propor_teste guardam o código de aprovação do pedido em <DataDir>/chat
// e não mandam nada ao celular. aprovar_pelo_chat só usa esse código depois de achar, no banco do
// OpenBot (só leitura), uma mensagem escrita pelo Jairo (autor humano, não rotina nem agente) que
// responde a uma mensagem do agente com o código do pedido e aprova sem negar nada. Anúncio ainda
// exige o valor da verba na frase dele.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"opdeals/internal/adtestq"
	"opdeals/internal/publishq"
)

const (
	chatModeFile     = "aprovacao-pelo-chat"
	chatTicketDir    = "chat"
	defaultOpenBotDB = "/Users/jjairo/Library/Application Support/OpenBot/openbot.db"
	ceoCollecta      = "agent-4259324b-fdc1-481a-9b4e-35866aac267a"
	chatWindow       = 72 * time.Hour
	kindPost         = "publicacao"
	kindAd           = "anuncio"
)

var (
	agentIDPattern = regexp.MustCompile(`^agent-[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	codePattern    = regexp.MustCompile(`\b[0-9A-Fa-f]{6}\b`)
	numberPattern  = regexp.MustCompile(`\d[\d.]*`)
	positiveWords  = regexp.MustCompile(`(?i)\baprov|\bconfirm|\bpode (postar|publicar|subir|soltar|mandar|liberar|seguir)|\blibera`)
	negativeWords  = regexp.MustCompile(`(?i)\bn[ãa]o (aprov|confirm|pode|post|publi|sob|sub|mand|lib)|\brejeit|\brecus|\breprov|\bcancel|\bsegura\b|\bespera\b`)
	allWords       = regexp.MustCompile(`(?i)\b(todas|todos|as duas|os dois|ambas|ambos|as 2|os 2|as tr[êe]s|os tr[êe]s|tudo)\b`)
)

type chatTicket struct {
	Kind        string    `json:"tipo"`
	ID          string    `json:"id"`
	Code        string    `json:"codigo"`
	Token       string    `json:"token"`
	CreatedAt   time.Time `json:"criado_em"`
	BudgetReais int64     `json:"verba_reais,omitempty"`
	Summary     string    `json:"resumo"`
	Files       []string  `json:"arquivos,omitempty"`
	Decided     string    `json:"decidido,omitempty"`
	TikTok      string    `json:"tiktok,omitempty"`
	MessageID   string    `json:"mensagem,omitempty"`
	Phrase      string    `json:"frase,omitempty"`
	DecidedAt   time.Time `json:"decidido_em,omitzero"`
}

func codeOf(id string) string {
	if len(id) < 6 {
		return strings.ToUpper(id)
	}
	return strings.ToUpper(id[:6])
}

// chatApprovers diz se a aprovação pelo chat está ligada e em quais chats a frase do Jairo vale.
func (a *app) chatApprovers() ([]string, bool) {
	data, err := os.ReadFile(filepath.Join(a.cfg.DataDir, chatModeFile))
	if err != nil {
		return nil, false
	}
	var ids []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(strings.SplitN(line, "#", 2)[0])
		if agentIDPattern.MatchString(line) {
			ids = append(ids, line)
		}
	}
	if len(ids) == 0 {
		ids = []string{ceoCollecta}
	}
	return ids, true
}

func (a *app) ticketPath(id string) string {
	return filepath.Join(a.cfg.DataDir, chatTicketDir, id+".json")
}

func (a *app) saveTicket(t chatTicket) error {
	if err := os.MkdirAll(filepath.Join(a.cfg.DataDir, chatTicketDir), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(a.ticketPath(t.ID), data, 0o600)
}

func (a *app) loadTickets() ([]chatTicket, error) {
	paths, err := filepath.Glob(filepath.Join(a.cfg.DataDir, chatTicketDir, "*.json"))
	if err != nil {
		return nil, err
	}
	out := make([]chatTicket, 0, len(paths))
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		var t chatTicket
		if err := json.Unmarshal(data, &t); err != nil {
			return nil, fmt.Errorf("%s: %v", filepath.Base(p), err)
		}
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

func (a *app) ticketForPost(it publishq.Item) chatTicket {
	files := make([]string, 0, len(it.Files))
	for _, f := range it.Files {
		files = append(files, previewOf(f.Path))
	}
	return chatTicket{Kind: kindPost, ID: it.ID, Code: codeOf(it.ID), Token: it.Token, CreatedAt: a.now(), Summary: ownerBody(it), Files: files}
}

func (a *app) ticketForAd(t adtestq.Test, left int64) chatTicket {
	files := make([]string, 0, len(t.Creatives))
	summary := fmt.Sprintf("Teste de anúncio %s, público %s, %s em %d dias (sobra %s na semana).\nPor quê: %s", t.Objective, t.Audience, reais(t.BudgetCents), t.Days, reais(left), t.Note)
	if t.Link != "" {
		summary += "\nLink: " + t.Link
	}
	for i, c := range t.Creatives {
		files = append(files, previewOf(c.Path))
		summary += fmt.Sprintf("\nCriativo %d (%s): %s", i+1, c.Hook, c.Text)
	}
	return chatTicket{Kind: kindAd, ID: t.ID, Code: codeOf(t.ID), Token: t.Token, CreatedAt: a.now(), BudgetReais: t.BudgetCents / 100, Summary: summary, Files: files}
}

type chatMsg struct {
	Thread   string `json:"thread"`
	ID       string `json:"id"`
	Author   string `json:"author"`
	ItemType string `json:"item_type"`
	Source   string `json:"source"`
	Text     string `json:"text"`
	At       string `json:"at"`
}

func (m chatMsg) human() bool {
	return m.Author == "user" && m.ItemType == "" && m.Source == "user"
}

func (m chatMsg) fromAgent() bool {
	return m.Author == "assistant" && (m.ItemType == "agentMessage" || m.ItemType == "final_answer" || m.ItemType == "")
}

// readChat lê, só leitura, as mensagens dos chats dos agentes indicados desde um instante.
func (a *app) readChat(ctx context.Context, agents []string, since time.Time) ([]chatMsg, error) {
	quoted := make([]string, 0, len(agents))
	for _, id := range agents {
		if !agentIDPattern.MatchString(id) {
			return nil, fmt.Errorf("agente %q inválido", id)
		}
		quoted = append(quoted, "'"+id+"'")
	}
	query := fmt.Sprintf(`SELECT m.thread_id AS thread, m.message_id AS id, m.author AS author, COALESCE(m.item_type,'') AS item_type,
  COALESCE(json_extract(m.message_json,'$.source'),'') AS source, COALESCE(json_extract(m.message_json,'$.text'),'') AS text, m.created_at AS at
FROM projection_thread_messages m JOIN projection_threads t ON t.thread_id=m.thread_id
WHERE t.agent_id IN (%s) AND m.created_at >= '%s' AND m.author IN ('user','assistant')
ORDER BY m.thread_id, m.created_at, m.ordinal`, strings.Join(quoted, ","), since.UTC().Format("2006-01-02T15:04:05.000Z"))
	db := a.cfg.OpenBotDB
	if db == "" {
		db = defaultOpenBotDB
	}
	out, err := exec.CommandContext(ctx, "/usr/bin/sqlite3", "-readonly", "-json", db, query).Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return nil, fmt.Errorf("não consegui ler o chat do OpenBot: %s", strings.TrimSpace(string(ee.Stderr)))
		}
		return nil, fmt.Errorf("não consegui ler o chat do OpenBot: %v", err)
	}
	var msgs []chatMsg
	if len(strings.TrimSpace(string(out))) == 0 {
		return msgs, nil
	}
	if err := json.Unmarshal(out, &msgs); err != nil {
		return nil, fmt.Errorf("chat do OpenBot em formato inesperado: %v", err)
	}
	return msgs, nil
}

type chatVerdict struct {
	Action string
	Msg    chatMsg
	Reason string
}

func codesIn(text string, known map[string]bool) []string {
	var out []string
	for _, c := range codePattern.FindAllString(text, -1) {
		c = strings.ToUpper(c)
		if known[c] {
			out = append(out, c)
		}
	}
	return out
}

func hasBudget(text string, reaisValue int64) bool {
	for _, n := range numberPattern.FindAllString(text, -1) {
		n = strings.ReplaceAll(strings.TrimRight(n, "."), ".", "")
		if v, err := strconv.ParseInt(n, 10, 64); err == nil && v == reaisValue {
			return true
		}
	}
	return false
}

// judge procura a última mensagem do Jairo sobre o pedido e diz o que ela decide.
// Uma mensagem do Jairo é sobre o pedido quando cita o código dele ou quando vem logo depois de
// mensagens do agente que citam o código. known são os códigos dos pedidos guardados.
func judge(msgs []chatMsg, target chatTicket, known map[string]bool) (chatVerdict, bool) {
	var best chatVerdict
	found := false
	shown := map[string]bool{}
	thread := ""
	for _, m := range msgs {
		if m.Thread != thread {
			thread = m.Thread
			shown = map[string]bool{}
		}
		if m.fromAgent() {
			for _, c := range codesIn(m.Text, known) {
				shown[c] = true
			}
			continue
		}
		if !m.human() {
			continue
		}
		ctxCodes := shown
		shown = map[string]bool{}
		named := codesIn(m.Text, known)
		about := ctxCodes[target.Code]
		for _, c := range named {
			if c == target.Code {
				about = true
			}
		}
		if !about || (len(named) > 0 && !hasCode(named, target.Code)) {
			continue
		}
		v := chatVerdict{Msg: m}
		pos, neg := positiveWords.MatchString(m.Text), negativeWords.MatchString(m.Text)
		if len(named) > 0 {
			// Frase com códigos: cada código vale pelo verbo do trecho dele (desde 26/11/2026),
			// ex.: "aprovo E10EB4, rejeito CDB388 e rejeito 790BC5".
			pos, neg = codeVerbs(m.Text, known, target.Code)
		}
		switch {
		case len(named) == 0 && len(ctxCodes) > 1 && !allWords.MatchString(m.Text):
			v.Reason = "a mensagem respondeu a mais de um pedido sem dizer qual (peça o código ou \"todas\")"
		case pos && neg:
			v.Reason = "a mensagem aprova e nega ao mesmo tempo"
		case neg:
			v.Action = "rejeitar"
		case pos && target.Kind == kindAd && !hasBudget(m.Text, target.BudgetReais):
			v.Reason = fmt.Sprintf("anúncio só com o valor na frase dele (R$ %d)", target.BudgetReais)
		case pos:
			v.Action = "aprovar"
		default:
			v.Reason = "a mensagem não diz aprovo nem rejeito"
		}
		if !found || m.At >= best.Msg.At {
			best, found = v, true
		}
	}
	return best, found
}

// codeVerbs diz se o trecho que decide um código aprova ou nega. O trecho de cada código vai do
// código anterior até ele (o verbo vem antes); o último código leva também o resto da frase, para
// pegar "aprovo X mas não pode postar". Código sem verbo no trecho herda a decisão do anterior
// ("aprovo A e B"). Negação ("não aprovo") não conta como aprovação.
func codeVerbs(text string, known map[string]bool, target string) (pos, neg bool) {
	type occ struct {
		code       string
		start, end int
	}
	var occs []occ
	for _, l := range codePattern.FindAllStringIndex(text, -1) {
		if c := strings.ToUpper(text[l[0]:l[1]]); known[c] {
			occs = append(occs, occ{c, l[0], l[1]})
		}
	}
	prevPos, prevNeg := false, false
	for i, o := range occs {
		from := 0
		if i > 0 {
			from = occs[i-1].end
		}
		seg := text[from:o.start]
		if i == len(occs)-1 {
			seg += " " + text[o.end:]
		}
		p, n := segmentVerbs(seg)
		if !p && !n {
			p, n = prevPos, prevNeg
		}
		if o.code == target {
			return p, n
		}
		prevPos, prevNeg = p, n
	}
	return false, false
}

func segmentVerbs(seg string) (pos, neg bool) {
	negSpans := negativeWords.FindAllStringIndex(seg, -1)
	neg = len(negSpans) > 0
	for _, p := range positiveWords.FindAllStringIndex(seg, -1) {
		inside := false
		for _, n := range negSpans {
			if p[0] >= n[0] && p[0] < n[1] {
				inside = true
			}
		}
		if !inside {
			pos = true
		}
	}
	return pos, neg
}

func hasCode(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

type chatDecideInput struct {
	Codigo     string `json:"codigo" jsonschema:"o código de 6 letras do pedido (vem de pedir_aprovacao, propor_teste ou esperando_jairo)"`
	Acao       string `json:"acao" jsonschema:"aprovar ou rejeitar, como o Jairo escreveu"`
	SoConferir bool   `json:"so_conferir,omitempty" jsonschema:"true: só confere a frase, sem aprovar nem rejeitar"`
}

type chatDecideOutput struct {
	Codigo   string `json:"codigo"`
	ID       string `json:"id"`
	Tipo     string `json:"tipo"`
	Acao     string `json:"acao"`
	Feito    bool   `json:"feito"`
	Frase    string `json:"frase_do_jairo"`
	Mensagem string `json:"mensagem_id"`
	Quando   string `json:"quando"`
	Proximo  string `json:"proximo"`
}

type waitingItem struct {
	Codigo   string   `json:"codigo"`
	ID       string   `json:"id"`
	Tipo     string   `json:"tipo"`
	Resumo   string   `json:"resumo"`
	Arquivos []string `json:"arquivos"`
	Pedido   string   `json:"pedido_em"`
}

type waitingOutput struct {
	Ligado  bool          `json:"aprovacao_pelo_chat"`
	Chats   []string      `json:"chats_que_valem"`
	Itens   []waitingItem `json:"itens"`
	Proximo string        `json:"proximo"`
}

func (a *app) addChatTools(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "esperando_jairo",
		Description: "Para o CEO Collecta. Só leitura: posts e testes de anúncio que esperam a decisão do Jairo pelo chat, cada um com o código de 6 letras, o resumo (legenda, horário, verba) e as prévias para mostrar a ele.",
	}, a.toolWaiting)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "aprovar_pelo_chat",
		Description: "Para o CEO Collecta. Aprova ou rejeita um pedido pelo código, só se o Jairo escreveu a decisão no chat logo depois da mensagem em que você mostrou o pedido com o código (ou citando o código). A ferramenta confere a frase dele no OpenBot; mensagem de agente, de rotina ou repassada não vale. Anúncio exige o valor da verba na frase dele.",
	}, a.toolChatDecide)
}

func (a *app) pendingTickets(ctx context.Context) ([]chatTicket, error) {
	all, err := a.loadTickets()
	if err != nil {
		return nil, err
	}
	var posts struct {
		Items []publishq.Item `json:"items"`
	}
	if err := a.call(ctx, http.MethodGet, "/api/publish?status="+publishq.StatusPending, nil, &posts); err != nil {
		return nil, err
	}
	var ads struct {
		Items []adtestq.Test `json:"items"`
	}
	if err := a.call(ctx, http.MethodGet, "/api/adtests?status="+adtestq.StatusPending, nil, &ads); err != nil {
		return nil, err
	}
	open := map[string]bool{}
	for _, it := range posts.Items {
		open[kindPost+it.ID] = true
	}
	for _, t := range ads.Items {
		open[kindAd+t.ID] = true
	}
	var out []chatTicket
	for _, t := range all {
		if t.Decided == "" && open[t.Kind+t.ID] {
			out = append(out, t)
		}
	}
	return out, nil
}

func (a *app) toolWaiting(ctx context.Context, _ *mcp.CallToolRequest, _ noInput) (*mcp.CallToolResult, waitingOutput, error) {
	chats, on := a.chatApprovers()
	out := waitingOutput{Ligado: on, Chats: chats, Itens: []waitingItem{}}
	if !on {
		out.Proximo = "a aprovação pelo chat está desligada: o Jairo decide pelo celular"
		return nil, out, nil
	}
	pending, err := a.pendingTickets(ctx)
	if err != nil {
		return nil, waitingOutput{}, err
	}
	for _, t := range pending {
		out.Itens = append(out.Itens, waitingItem{Codigo: t.Code, ID: t.ID, Tipo: t.Kind, Resumo: t.Summary, Arquivos: t.Files, Pedido: t.CreatedAt.In(saoPaulo).Format("02/01 15:04")})
	}
	out.Proximo = "mostre cada item ao Jairo com o código escrito na mesma mensagem e as prévias; quando ele responder, chame aprovar_pelo_chat com o código. Anúncio: peça que ele diga o valor."
	return nil, out, nil
}

func (a *app) toolChatDecide(ctx context.Context, _ *mcp.CallToolRequest, in chatDecideInput) (*mcp.CallToolResult, chatDecideOutput, error) {
	chats, on := a.chatApprovers()
	if !on {
		return nil, chatDecideOutput{}, errors.New("a aprovação pelo chat está desligada: o Jairo decide pelo celular")
	}
	action := strings.ToLower(strings.TrimSpace(in.Acao))
	if action != "aprovar" && action != "rejeitar" {
		return nil, chatDecideOutput{}, errors.New("acao precisa ser aprovar ou rejeitar")
	}
	code := strings.ToUpper(strings.TrimSpace(in.Codigo))
	pending, err := a.pendingTickets(ctx)
	if err != nil {
		return nil, chatDecideOutput{}, err
	}
	// known leva todos os códigos já pedidos, decididos ou não: assim uma resposta antiga a dois
	// pedidos continua ambígua depois que um deles foi decidido.
	all, err := a.loadTickets()
	if err != nil {
		return nil, chatDecideOutput{}, err
	}
	known := map[string]bool{}
	for _, t := range all {
		known[t.Code] = true
	}
	var target *chatTicket
	for i := range pending {
		if pending[i].Code == code {
			target = &pending[i]
		}
	}
	if target == nil {
		return nil, chatDecideOutput{}, fmt.Errorf("nenhum pedido esperando o Jairo com o código %q: veja esperando_jairo", in.Codigo)
	}
	since := target.CreatedAt
	if a.now().Sub(since) > chatWindow {
		since = a.now().Add(-chatWindow)
	}
	msgs, err := a.readChat(ctx, chats, since)
	if err != nil {
		return nil, chatDecideOutput{}, err
	}
	v, ok := judge(msgs, *target, known)
	if !ok {
		return nil, chatDecideOutput{}, fmt.Errorf("não achei mensagem do Jairo sobre %s: mostre o pedido a ele com o código na mesma mensagem e espere a resposta", code)
	}
	if v.Action == "" {
		return nil, chatDecideOutput{}, fmt.Errorf("a última mensagem do Jairo sobre %s não decide: %s. Frase: %q", code, v.Reason, v.Msg.Text)
	}
	if v.Action != action {
		return nil, chatDecideOutput{}, fmt.Errorf("o Jairo escreveu %s, não %s, para %s. Frase: %q", v.Action, action, code, v.Msg.Text)
	}
	out := chatDecideOutput{Codigo: code, ID: target.ID, Tipo: target.Kind, Acao: action, Frase: v.Msg.Text, Mensagem: v.Msg.ID, Quando: v.Msg.At}
	if in.SoConferir {
		out.Proximo = "frase conferida; nada foi aprovado nem rejeitado"
		return nil, out, nil
	}
	path := "/api/aprovar/"
	if target.Kind == kindAd {
		path = "/api/aprovar-anuncio/"
	}
	q := url.Values{"acao": {action}, "t": {target.Token}}
	if err := a.call(ctx, http.MethodPost, path+url.PathEscape(target.ID)+"?"+q.Encode(), nil, nil); err != nil {
		return nil, chatDecideOutput{}, err
	}
	target.Decided, target.MessageID, target.Phrase, target.DecidedAt = action, v.Msg.ID, v.Msg.Text, a.now()
	if err := a.saveTicket(*target); err != nil {
		return nil, chatDecideOutput{}, fmt.Errorf("%s %s no painel, mas não gravei o registro: %v", code, action, err)
	}
	line := fmt.Sprintf("%s pelo chat %s %s mensagem %s: %q", action, target.Kind, target.ID, v.Msg.ID, v.Msg.Text)
	if target.Kind == kindAd {
		a.adLog(line)
	} else {
		a.logLine(line)
	}
	out.Feito = true
	out.Proximo = "registrado. Post aprovado sai pelo Publicador no horário; teste aprovado sobe na próxima rodada dos anúncios."
	return nil, out, nil
}
