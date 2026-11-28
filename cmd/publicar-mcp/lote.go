package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"opdeals/internal/adtestq"
)

// Aprovação por lote semanal (Jairo, 28/11/2026: "eu aprovo a verba da semana e os
// anúncios dentro dela sobem sem me pedir"). O Jairo escreve no chat do CEO a verba
// da semana; aprovar_lote_semanal confere a frase dele no OpenBot, como o
// aprovar_pelo_chat, e guarda o lote. Daí em diante, um teste de vendas ou de
// seguidores proposto dentro da semana sobe sem código se couber no que sobra do
// lote e nos tetos de sempre. O que não cabe continua pedindo código. Posts do
// Instagram nunca passam por aqui: seguem um a um.

const (
	loteFile      = "lotes-anuncio.json"
	loteMaxActive = 3 // testes aprovados ou no ar ao mesmo tempo
)

var (
	loteWeek    = regexp.MustCompile(`(?i)aprovo a verba da semana de (\d{1,2})/(\d{1,2}) a (\d{1,2})/(\d{1,2})\s*:?\s*R\$\s*([\d.]+)(?:,(\d{2}))?`)
	loteProduct = regexp.MustCompile(`(?i)venda s[óo] de ([^.\n;]+)`)
	loteCPA     = regexp.MustCompile(`(?i)CPA m[áa]ximo\s*:?\s*R\$\s*([\d.]+)(?:,(\d{2}))?\s*por compra`)
	loteChat    = regexp.MustCompile(`(?i)R\$\s*([\d.]+)(?:,(\d{2}))?\s*por conversa`)
)

// lote é a verba de uma semana (segunda a domingo) que o Jairo aprovou de uma vez.
type lote struct {
	ID         string    `json:"id"` // a segunda-feira, AAAA-MM-DD
	From       string    `json:"de"`
	To         string    `json:"ate"`
	BudgetReai int64     `json:"verba_reais"`
	Product    string    `json:"produto"`
	CPAReais   float64   `json:"cpa_compra_reais"`
	ChatReais  float64   `json:"cpa_conversa_reais,omitempty"`
	MessageID  string    `json:"mensagem"`
	Phrase     string    `json:"frase"`
	SavedAt    time.Time `json:"gravado_em"`
}

func (l lote) window() (time.Time, time.Time) {
	from, _ := time.ParseInLocation("2006-01-02", l.From, saoPaulo)
	to, _ := time.ParseInLocation("2006-01-02", l.To, saoPaulo)
	return from, to.Add(24*time.Hour - time.Nanosecond)
}

func (l lote) covers(t time.Time) bool {
	from, to := l.window()
	return !t.Before(from) && !t.After(to)
}

func (l lote) label() string {
	from, to := l.window()
	return fmt.Sprintf("%s a %s", from.Format("02/01"), to.Format("02/01"))
}

func reaisOf(intPart, cents string) (float64, bool) {
	n, err := strconv.ParseInt(strings.ReplaceAll(strings.TrimRight(intPart, "."), ".", ""), 10, 64)
	if err != nil {
		return 0, false
	}
	v := float64(n)
	if cents != "" {
		c, _ := strconv.Atoi(cents)
		v += float64(c) / 100
	}
	return v, true
}

// parseLote lê a frase do Jairo. O ano é o da semana mais perto de agora (em
// dezembro, "05/01" é janeiro do ano seguinte).
func parseLote(text string, now time.Time) (lote, error) {
	weeks := loteWeek.FindAllStringSubmatch(text, -1)
	switch {
	case len(weeks) == 0:
		return lote{}, errors.New(`a frase não segue o formato "Aprovo a verba da semana de DD/MM a DD/MM: R$ X. Venda só de <produto>. CPA máximo: R$ Y por compra"`)
	case len(weeks) > 1:
		return lote{}, errors.New("a frase tem mais de uma semana: um lote por frase")
	case negativeWords.MatchString(text):
		return lote{}, errors.New("a frase tem negação (não, rejeito, cancela...): não vale como aprovação")
	}
	m := weeks[0]
	day := func(d, mo string) (time.Time, bool) {
		dd, _ := strconv.Atoi(d)
		mm, _ := strconv.Atoi(mo)
		best := time.Time{}
		for _, y := range []int{now.Year() - 1, now.Year(), now.Year() + 1} {
			t := time.Date(y, time.Month(mm), dd, 0, 0, 0, 0, saoPaulo)
			if t.Day() != dd || int(t.Month()) != mm {
				return time.Time{}, false
			}
			if best.IsZero() || absDur(t.Sub(now)) < absDur(best.Sub(now)) {
				best = t
			}
		}
		return best, true
	}
	from, ok1 := day(m[1], m[2])
	to, ok2 := day(m[3], m[4])
	if !ok1 || !ok2 {
		return lote{}, errors.New("data inválida na frase")
	}
	if from.Weekday() != time.Monday || to.Sub(from) != 6*24*time.Hour {
		return lote{}, fmt.Errorf("a semana do lote vai de segunda a domingo: %s a %s não é", from.Format("02/01"), to.Format("02/01"))
	}
	if to.Add(24 * time.Hour).Before(now) {
		return lote{}, fmt.Errorf("a semana %s a %s já passou", from.Format("02/01"), to.Format("02/01"))
	}
	budget, ok := reaisOf(m[5], m[6])
	switch {
	case !ok || budget <= 0:
		return lote{}, errors.New("valor da verba inválido")
	case budget != float64(int64(budget)):
		return lote{}, errors.New("a verba do lote vai em reais inteiros")
	case int64(budget)*100 > adtestq.WeeklyCapCents:
		return lote{}, fmt.Errorf("R$ %.0f passa do teto da semana (%s)", budget, reais(adtestq.WeeklyCapCents))
	}
	l := lote{ID: from.Format("2006-01-02"), From: from.Format("2006-01-02"), To: to.Format("2006-01-02"), BudgetReai: int64(budget)}
	if p := loteProduct.FindStringSubmatch(text); p != nil {
		l.Product = strings.TrimSpace(p[1])
	}
	if l.Product == "" {
		return lote{}, errors.New(`falta "Venda só de <produto>" na frase`)
	}
	c := loteCPA.FindStringSubmatch(text)
	if c == nil {
		return lote{}, errors.New(`falta "CPA máximo: R$ Y por compra" na frase`)
	}
	l.CPAReais, _ = reaisOf(c[1], c[2])
	if ch := loteChat.FindStringSubmatch(text); ch != nil {
		l.ChatReais, _ = reaisOf(ch[1], ch[2])
	}
	return l, nil
}

func absDur(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}

func (a *app) loadLotes() ([]lote, error) {
	data, err := os.ReadFile(filepath.Join(a.cfg.DataDir, loteFile))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var all []lote
	if err := json.Unmarshal(data, &all); err != nil {
		return nil, fmt.Errorf("%s: %v", loteFile, err)
	}
	return all, nil
}

func (a *app) saveLote(l lote) error {
	all, err := a.loadLotes()
	if err != nil {
		return err
	}
	out := []lote{l}
	for _, x := range all {
		if x.ID != l.ID { // a frase nova da mesma semana troca a anterior
			out = append(out, x)
		}
	}
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	tmp := filepath.Join(a.cfg.DataDir, loteFile+".tmp")
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(a.cfg.DataDir, loteFile))
}

// currentLote é o lote da semana de agora, se houver.
func (a *app) currentLote() (*lote, error) {
	all, err := a.loadLotes()
	if err != nil {
		return nil, err
	}
	now := a.now()
	for i := range all {
		if all[i].covers(now) {
			return &all[i], nil
		}
	}
	return nil, nil
}

// loteUsed soma a verba dos testes já aprovados por este lote.
func (a *app) loteUsed(l lote) (int64, error) {
	tickets, err := a.loadTickets()
	if err != nil {
		return 0, err
	}
	var used int64
	for _, t := range tickets {
		if t.Lote == l.ID && t.Decided == "aprovar" {
			used += t.BudgetReais
		}
	}
	return used, nil
}

// approveByLote aprova o teste recém-proposto pelo lote da semana, se couber.
// Devolve o motivo quando não cabe (o teste segue pedindo código).
func (a *app) approveByLote(ctx context.Context, created adtestq.Test, ticket *chatTicket, weekLeft int64) (bool, string, error) {
	l, err := a.currentLote()
	if err != nil || l == nil {
		return false, "", err
	}
	if created.Objective == "mensagens" {
		return false, "compra de cartas (mensagens) tem verba própria e não entra no lote", nil
	}
	used, err := a.loteUsed(*l)
	if err != nil {
		return false, "", err
	}
	budget := created.BudgetCents / 100
	if used+budget > l.BudgetReai {
		return false, fmt.Sprintf("passa do lote %s: sobram R$ %d e o teste pede R$ %d", l.label(), l.BudgetReai-used, budget), nil
	}
	var list struct {
		Items []adtestq.Test `json:"items"`
	}
	if err := a.call(ctx, http.MethodGet, "/api/adtests", nil, &list); err != nil {
		return false, "", err
	}
	active := 0
	for _, x := range list.Items {
		if x.ID != created.ID && (x.Status == adtestq.StatusApproved || x.Status == adtestq.StatusRunning) {
			active++
		}
	}
	if active >= loteMaxActive {
		return false, fmt.Sprintf("já há %d testes aprovados ou no ar (o lote sobe até %d ao mesmo tempo)", active, loteMaxActive), nil
	}
	q := url.Values{"acao": {"aprovar"}, "t": {created.Token}}
	if err := a.call(ctx, http.MethodPost, "/api/aprovar-anuncio/"+url.PathEscape(created.ID)+"?"+q.Encode(), nil, nil); err != nil {
		return false, "", err
	}
	ticket.Decided, ticket.Lote, ticket.MessageID, ticket.DecidedAt = "aprovar", l.ID, l.MessageID, a.now()
	ticket.Phrase = fmt.Sprintf("lote %s (R$ %d): %s", l.label(), l.BudgetReai, l.Phrase)
	if err := a.saveTicket(*ticket); err != nil {
		return true, "", fmt.Errorf("teste %s aprovado pelo lote, mas não gravei o registro: %v", created.ID, err)
	}
	left := l.BudgetReai - used - budget
	a.adLog(fmt.Sprintf("aprovar pelo lote %s (mensagem %s) anuncio %s R$ %d; sobram R$ %d no lote", l.label(), l.MessageID, created.ID, budget, left))
	body := fmt.Sprintf("Aprovado pelo lote %s, sem código.\n%s\nSobram R$ %d no lote e %s na semana.", l.label(), ticket.Summary, left, reais(weekLeft))
	if err := a.push(ctx, map[string]string{"Title": "Teste aprovado pelo lote", "Tags": "white_check_mark"}, strings.NewReader(body)); err != nil {
		a.adLog("aviso do lote não chegou: " + err.Error())
	}
	return true, "", nil
}

type loteInput struct {
	MensagemID string `json:"mensagem_id,omitempty" jsonschema:"opcional: o id da mensagem do Jairo; sem ele, vale a última frase de lote dele no chat nas últimas 72 horas"`
	SoConferir bool   `json:"so_conferir,omitempty" jsonschema:"true: só confere a frase, sem gravar o lote"`
}

type loteOutput struct {
	De       string  `json:"de"`
	Ate      string  `json:"ate"`
	Verba    string  `json:"verba"`
	Usado    string  `json:"usado,omitempty"`
	Sobra    string  `json:"sobra,omitempty"`
	Produto  string  `json:"produto"`
	CPA      float64 `json:"cpa_compra_reais"`
	Conversa float64 `json:"cpa_conversa_reais,omitempty"`
	Frase    string  `json:"frase_do_jairo"`
	Mensagem string  `json:"mensagem_id"`
	Gravado  bool    `json:"gravado"`
	Proximo  string  `json:"proximo"`
}

func (a *app) addLoteTool(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name: "aprovar_lote_semanal",
		Description: "Para o CEO Collecta. Grava a verba da semana que o Jairo aprovou de uma vez, conferindo a frase dele no chat do OpenBot (mensagem de agente, rotina ou repassada não vale). Formato: \"Aprovo a verba da semana de DD/MM a DD/MM: R$ X. Venda só de <produto>. CPA máximo: R$ Y por compra (e R$ Z por conversa na compra de cartas)\". " +
			"A semana vai de segunda a domingo e a verba não passa do teto da semana. Com o lote gravado, teste de vendas ou seguidores proposto na semana sobe sem código se couber no que sobra do lote, nos tetos de sempre e em até 3 testes ao mesmo tempo; o resto continua pedindo código. Só para testes de anúncio: post do Instagram segue um a um.",
	}, a.toolLote)
}

func (a *app) toolLote(ctx context.Context, _ *mcp.CallToolRequest, in loteInput) (*mcp.CallToolResult, loteOutput, error) {
	chats, on := a.chatApprovers()
	if !on {
		return nil, loteOutput{}, errors.New("a aprovação pelo chat está desligada: o lote só vale com ela ligada")
	}
	msgs, err := a.readChat(ctx, chats, a.now().Add(-chatWindow))
	if err != nil {
		return nil, loteOutput{}, err
	}
	var pick *chatMsg
	for i := range msgs {
		m := msgs[i]
		if !m.human() || !loteWeek.MatchString(m.Text) {
			continue
		}
		if in.MensagemID != "" && m.ID != in.MensagemID {
			continue
		}
		if pick == nil || m.At >= pick.At {
			pick = &msgs[i]
		}
	}
	if pick == nil {
		if in.MensagemID != "" {
			return nil, loteOutput{}, fmt.Errorf("a mensagem %s não é do Jairo no chat do CEO, ou não é uma frase de lote, ou tem mais de 72 horas", in.MensagemID)
		}
		return nil, loteOutput{}, errors.New(`não achei frase de lote do Jairo no chat do CEO nas últimas 72 horas. Peça a ele: "Aprovo a verba da semana de DD/MM a DD/MM: R$ X. Venda só de <produto>. CPA máximo: R$ Y por compra"`)
	}
	l, err := parseLote(pick.Text, a.now())
	if err != nil {
		return nil, loteOutput{}, fmt.Errorf("%v. Frase: %q", err, pick.Text)
	}
	l.MessageID, l.Phrase, l.SavedAt = pick.ID, pick.Text, a.now()
	from, to := l.window()
	out := loteOutput{De: from.Format("02/01/2006"), Ate: to.Format("02/01/2006"), Verba: reais(l.BudgetReai * 100), Produto: l.Product,
		CPA: l.CPAReais, Conversa: l.ChatReais, Frase: pick.Text, Mensagem: pick.ID}
	if in.SoConferir {
		out.Proximo = "frase conferida; o lote não foi gravado"
		return nil, out, nil
	}
	if err := a.saveLote(l); err != nil {
		return nil, loteOutput{}, err
	}
	used, err := a.loteUsed(l)
	if err != nil {
		return nil, loteOutput{}, err
	}
	out.Gravado, out.Usado, out.Sobra = true, reais(used*100), reais((l.BudgetReai-used)*100)
	a.adLog(fmt.Sprintf("lote %s gravado: R$ %d, produto %q, CPA R$ %.2f por compra (mensagem %s)", l.label(), l.BudgetReai, l.Product, l.CPAReais, l.MessageID))
	out.Proximo = "lote gravado. Testes de vendas ou seguidores propostos nesta semana sobem sem código enquanto couberem; produto e CPA ficam com a Revisão e a Coordenação. esperando_jairo mostra o lote e quanto sobra."
	return nil, out, nil
}

type loteStatus struct {
	De      string `json:"de"`
	Ate     string `json:"ate"`
	Verba   string `json:"verba"`
	Usado   string `json:"usado"`
	Sobra   string `json:"sobra"`
	Produto string `json:"produto"`
	Frase   string `json:"frase_do_jairo"`
}

// loteNow é o resumo do lote vigente para o esperando_jairo.
func (a *app) loteNow() (*loteStatus, error) {
	l, err := a.currentLote()
	if err != nil || l == nil {
		return nil, err
	}
	used, err := a.loteUsed(*l)
	if err != nil {
		return nil, err
	}
	from, to := l.window()
	return &loteStatus{De: from.Format("02/01"), Ate: to.Format("02/01"), Verba: reais(l.BudgetReai * 100), Usado: reais(used * 100),
		Sobra: reais((l.BudgetReai - used) * 100), Produto: l.Product, Frase: l.Phrase}, nil
}
