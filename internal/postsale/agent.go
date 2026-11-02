package postsale

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go/v2"
	"github.com/anthropics/anthropic-sdk-go/v2/option"

	"opdeals/internal/conversations"
	"opdeals/internal/orders"
)

const (
	DefaultModel  = "claude-opus-5-5"
	maxToolRounds = 6
)

type Turn struct {
	Role string
	Text string
}

type PitchInput struct {
	FirstName  string
	Bought     []orders.Line
	Candidates []Candidate
}

type ReplyInput struct {
	FirstName   string
	OrderNumber int
	Bought      []orders.Line
	Offer       *conversations.Offer
	Turns       []Turn
}

type Reply struct {
	Text          string
	Handoff       bool
	HandoffReason string
}

type Toolbox interface {
	SearchStock(query string) (string, error)
	OrderStatus() (string, error)
	ComplementLink(lines []conversations.LinkLine) (string, error)
}

type Brain interface {
	Pitch(ctx context.Context, in PitchInput) (string, string, error)
	Reply(ctx context.Context, in ReplyInput, tools Toolbox) (Reply, error)
}

type Claude struct {
	client anthropic.Client
	model  string
}

func NewClaude(apiKey, model string) *Claude {
	if model == "" {
		model = DefaultModel
	}
	return &Claude{client: anthropic.NewClient(option.WithAPIKey(apiKey)), model: model}
}

const pitchSystem = `Você escreve a frase de oferta de pós-venda da Collecta TCG, uma loja brasileira de cartas colecionáveis (One Piece, Pokémon, Riftbound, Lorcana, Gundam).
O cliente acabou de comprar no site e o pedido ainda não foi enviado, então qualquer carta que ele adicionar vai no mesmo pacote sem pagar frete de novo.
Escolha UM item da lista de candidatos (o que mais combina com o que ele comprou; os candidatos já vêm em ordem de prioridade) e escreva UMA frase curta, em português do Brasil, tom informal de loja gamer, que diga o nome da carta, o preço em reais e por que vale a pena (combina com o que ele levou, está saindo muito, está em destaque).
Regras: no máximo 220 caracteres, sem quebra de linha, sem emoji, sem inventar informação que não esteja nos dados, sem cumprimentar (a saudação e o agradecimento já estão na mensagem).`

func (c *Claude) Pitch(ctx context.Context, in PitchInput) (string, string, error) {
	if len(in.Candidates) == 0 {
		return "", "", errors.New("sem candidatos")
	}
	skus := make([]string, 0, len(in.Candidates))
	for _, cand := range in.Candidates {
		skus = append(skus, cand.SKU)
	}
	data, err := json.Marshal(pitchData{Cliente: in.FirstName, Comprou: BoughtView(in.Bought), Candidatos: in.Candidates})
	if err != nil {
		return "", "", err
	}
	resp, err := c.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     c.model,
		MaxTokens: 16000,
		System:    []anthropic.TextBlockParam{{Text: pitchSystem}},
		Messages:  []anthropic.MessageParam{anthropic.NewUserMessage(anthropic.NewTextBlock(string(data)))},
		OutputConfig: anthropic.OutputConfigParam{
			Effort: anthropic.OutputConfigEffortLow,
			Format: anthropic.JSONOutputFormatParam{Schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"sku":   map[string]any{"type": "string", "enum": skus},
					"pitch": map[string]any{"type": "string"},
				},
				"required":             []string{"sku", "pitch"},
				"additionalProperties": false,
			}},
		},
	})
	if err != nil {
		return "", "", err
	}
	if resp.StopReason == anthropic.StopReasonRefusal {
		return "", "", errors.New("modelo recusou")
	}
	var out struct {
		SKU   string `json:"sku"`
		Pitch string `json:"pitch"`
	}
	if err := json.Unmarshal([]byte(textOf(resp)), &out); err != nil {
		return "", "", err
	}
	for _, cand := range in.Candidates {
		if cand.SKU == out.SKU {
			return out.SKU, CleanPitch(out.Pitch), nil
		}
	}
	return "", "", fmt.Errorf("sku fora dos candidatos: %s", out.SKU)
}

const replySystem = `Você é o assistente de atendimento da Collecta TCG no WhatsApp, uma loja brasileira de cartas colecionáveis (One Piece, Pokémon, Riftbound, Lorcana, Gundam).
A conversa começou com a loja agradecendo uma compra do site e oferecendo adicionar mais um item no mesmo pacote, aproveitando o frete já pago, enquanto o pedido não é enviado.

Como falar:
- Português do Brasil, informal, simpático, direto, mensagens curtas de WhatsApp. Emoji só de vez em quando.
- Fale como "a Collecta" / "a gente". Nunca finja ser uma pessoa. Se perguntarem se é robô, diga que é o assistente automático da loja e ofereça chamar o responsável.
- Não force a venda. Se o cliente disser que não quer, agradeça e encerre com simpatia.

Fatos:
- Preço, estoque e status do pedido saem SOMENTE das ferramentas. Nunca invente carta, preço, prazo ou estoque.
- Se a carta pedida não estiver em estoque, diga isso e sugira as parecidas que a busca devolver.
- Só cartas avulsas (cabe_no_envelope = true) entram no complemento com frete grátis. Selados, acessórios e qualquer coisa que mude o pacote: chame o humano.
- Quando o cliente decidir o que quer, gere o link com gerar_link_complemento e mande o link. Explique que ele confirma o CPF e paga, e que vai tudo junto no mesmo envio.

Chame o humano (chamar_humano) quando: pedirem desconto ou negociação, troca, devolução, reclamação, problema com pagamento ou entrega, assunto fora de cartas e do pedido, cliente irritado, ou quando você não souber responder com as ferramentas. Ao chamar o humano, avise o cliente que o responsável da loja vai continuar a conversa.`

type toolSpec struct {
	name        string
	description string
	properties  map[string]any
	required    []string
}

var replyTools = []toolSpec{
	{
		name:        "buscar_estoque",
		description: "Busca cartas e produtos à venda na Collecta por nome, número ou coleção. Devolve os itens que batem com a busca e, quando não há item exato, os mais parecidos. Cada item traz sku, nome, detalhes, preço em reais, quantidade disponível e se cabe no envelope (só esses entram no complemento com frete grátis).",
		properties:  map[string]any{"consulta": map[string]any{"type": "string", "description": "O que o cliente procura, ex.: \"Shanks manga OP09\""}},
		required:    []string{"consulta"},
	},
	{
		name:        "status_pedido",
		description: "Mostra o pedido do cliente: itens, status (pago, enviado…), código de rastreio e até quando dá para adicionar itens no mesmo envio.",
		properties:  map[string]any{},
		required:    []string{},
	},
	{
		name:        "gerar_link_complemento",
		description: "Gera o link da loja com os itens já no carrinho, marcado para ir junto com o pedido do cliente e frete R$0. Use quando o cliente confirmar o que quer. Só aceita itens que cabem no envelope e estão em estoque, e só enquanto a janela do pedido estiver aberta.",
		properties: map[string]any{"itens": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"sku": map[string]any{"type": "string"},
					"qtd": map[string]any{"type": "integer", "minimum": 1},
				},
				"required":             []string{"sku", "qtd"},
				"additionalProperties": false,
			},
		}},
		required: []string{"itens"},
	},
	{
		name:        "chamar_humano",
		description: "Passa a conversa para o responsável da loja e pausa o assistente. Use nos casos listados nas instruções.",
		properties:  map[string]any{"motivo": map[string]any{"type": "string"}},
		required:    []string{"motivo"},
	},
}

func (c *Claude) Reply(ctx context.Context, in ReplyInput, tools Toolbox) (Reply, error) {
	ctxData, err := json.Marshal(replyData{Cliente: in.FirstName, Pedido: in.OrderNumber, Comprou: BoughtView(in.Bought), Oferta: in.Offer})
	if err != nil {
		return Reply{}, err
	}
	system := replySystem + "\n\nDados do pedido (internos, não são mensagem do cliente): " + string(ctxData)
	params := make([]anthropic.ToolUnionParam, 0, len(replyTools))
	for _, t := range replyTools {
		tp := anthropic.ToolParam{
			Name:        t.name,
			Description: anthropic.String(t.description),
			InputSchema: anthropic.ToolInputSchemaParam{Properties: t.properties, Required: t.required},
		}
		params = append(params, anthropic.ToolUnionParam{OfTool: &tp})
	}
	messages := historyMessages(in.Turns)
	var out Reply
	for range maxToolRounds {
		resp, err := c.client.Messages.New(ctx, anthropic.MessageNewParams{
			Model:        c.model,
			MaxTokens:    16000,
			System:       []anthropic.TextBlockParam{{Text: system}},
			Messages:     messages,
			Tools:        params,
			OutputConfig: anthropic.OutputConfigParam{Effort: anthropic.OutputConfigEffortLow},
		})
		if err != nil {
			return Reply{}, err
		}
		if resp.StopReason == anthropic.StopReasonRefusal {
			return Reply{Handoff: true, HandoffReason: "o modelo recusou responder"}, nil
		}
		messages = append(messages, resp.ToParam())
		results := []anthropic.ContentBlockParamUnion{}
		for _, block := range resp.Content {
			use, ok := block.AsAny().(anthropic.ToolUseBlock)
			if !ok {
				continue
			}
			result, isErr := runTool(use.Name, use.JSON.Input.Raw(), tools, &out)
			results = append(results, anthropic.NewToolResultBlock(use.ID, result, isErr))
		}
		if resp.StopReason != anthropic.StopReasonToolUse {
			out.Text = strings.TrimSpace(textOf(resp))
			return out, nil
		}
		messages = append(messages, anthropic.NewUserMessage(results...))
	}
	out.Handoff = true
	if out.HandoffReason == "" {
		out.HandoffReason = "o assistente não conseguiu fechar a resposta"
	}
	return out, nil
}

func runTool(name, raw string, tools Toolbox, out *Reply) (string, bool) {
	switch name {
	case "buscar_estoque":
		var in struct {
			Consulta string `json:"consulta"`
		}
		if err := json.Unmarshal([]byte(raw), &in); err != nil {
			return "entrada inválida: " + err.Error(), true
		}
		res, err := tools.SearchStock(in.Consulta)
		if err != nil {
			return err.Error(), true
		}
		return res, false
	case "status_pedido":
		res, err := tools.OrderStatus()
		if err != nil {
			return err.Error(), true
		}
		return res, false
	case "gerar_link_complemento":
		var in struct {
			Itens []struct {
				SKU string `json:"sku"`
				Qtd int    `json:"qtd"`
			} `json:"itens"`
		}
		if err := json.Unmarshal([]byte(raw), &in); err != nil {
			return "entrada inválida: " + err.Error(), true
		}
		lines := make([]conversations.LinkLine, 0, len(in.Itens))
		for _, it := range in.Itens {
			lines = append(lines, conversations.LinkLine{SKU: it.SKU, Qty: it.Qtd})
		}
		link, err := tools.ComplementLink(lines)
		if err != nil {
			return err.Error(), true
		}
		return link, false
	case "chamar_humano":
		var in struct {
			Motivo string `json:"motivo"`
		}
		if err := json.Unmarshal([]byte(raw), &in); err != nil {
			return "entrada inválida: " + err.Error(), true
		}
		out.Handoff = true
		out.HandoffReason = in.Motivo
		return "ok: o responsável foi avisado e vai assumir a conversa", false
	}
	return "ferramenta desconhecida: " + name, true
}

func historyMessages(turns []Turn) []anthropic.MessageParam {
	out := []anthropic.MessageParam{anthropic.NewUserMessage(anthropic.NewTextBlock("[Início da conversa. A primeira mensagem abaixo foi enviada pela loja.]"))}
	lastRole := "user"
	for _, t := range turns {
		role := "assistant"
		text := t.Text
		switch t.Role {
		case conversations.RoleCustomer:
			role = "user"
		case conversations.RoleOwner:
			text = "[mensagem do responsável da loja] " + text
		case conversations.RoleSystem:
			continue
		}
		if role == lastRole {
			if role == "user" {
				out = append(out, anthropic.NewAssistantMessage(anthropic.NewTextBlock("[sem resposta]")))
			} else {
				out = append(out, anthropic.NewUserMessage(anthropic.NewTextBlock("[sem resposta do cliente]")))
			}
		}
		if role == "user" {
			out = append(out, anthropic.NewUserMessage(anthropic.NewTextBlock(text)))
		} else {
			out = append(out, anthropic.NewAssistantMessage(anthropic.NewTextBlock(text)))
		}
		lastRole = role
	}
	return out
}

func textOf(resp *anthropic.Message) string {
	var b strings.Builder
	for _, block := range resp.Content {
		if t, ok := block.AsAny().(anthropic.TextBlock); ok {
			b.WriteString(t.Text)
		}
	}
	return b.String()
}

type pitchData struct {
	Cliente    string       `json:"cliente"`
	Comprou    []BoughtLine `json:"comprou"`
	Candidatos []Candidate  `json:"candidatos"`
}

type replyData struct {
	Cliente string               `json:"cliente"`
	Pedido  int                  `json:"pedido"`
	Comprou []BoughtLine         `json:"comprou"`
	Oferta  *conversations.Offer `json:"oferta"`
}

type BoughtLine struct {
	Nome     string  `json:"nome"`
	Detalhes string  `json:"detalhes,omitempty"`
	Qtd      int     `json:"qtd"`
	Preco    float64 `json:"preco"`
}

func BoughtView(lines []orders.Line) []BoughtLine {
	out := make([]BoughtLine, 0, len(lines))
	for _, l := range lines {
		out = append(out, BoughtLine{Nome: l.Name, Detalhes: l.Meta, Qtd: l.Qty, Preco: l.UnitBRL})
	}
	return out
}

func FallbackPitch(c Candidate) string {
	reason := "tá num preço top"
	if len(c.Reasons) > 0 {
		reason = c.Reasons[0]
	}
	return CleanPitch(fmt.Sprintf("A %s (%s) sai por R$ %s e %s.", c.Name, c.Meta, FormatBRL(c.UnitBRL), reason))
}

func FormatBRL(v float64) string {
	s := fmt.Sprintf("%.2f", v)
	whole, frac, _ := strings.Cut(s, ".")
	var b strings.Builder
	for i, r := range whole {
		if i > 0 && (len(whole)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(r)
	}
	return b.String() + "," + frac
}
