package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"opdeals/internal/whatsapp"
)

type outboxItem struct {
	ID             string `json:"id"`
	Kind           string `json:"kind"`
	ConversationID string `json:"conversationId,omitempty"`
	OrderNumber    int    `json:"orderNumber"`
	Phone          string `json:"phone"`
	FirstName      string `json:"firstName"`
	Text           string `json:"text"`
}

type claimOutput struct {
	Quiet  bool         `json:"quiet"`
	Items  []outboxItem `json:"items"`
	Aviso  string       `json:"aviso,omitempty"`
	Regras string       `json:"regras,omitempty"`
}

type okOutput struct {
	OK bool `json:"ok"`
}

type watchMessage struct {
	Role string `json:"role"`
	Text string `json:"text"`
	At   string `json:"at"`
}

type watchConversation struct {
	ConversationID string         `json:"conversationId"`
	Phone          string         `json:"phone"`
	FirstName      string         `json:"firstName"`
	OrderNumber    int            `json:"orderNumber"`
	State          string         `json:"state"`
	WindowEndsAt   string         `json:"windowEndsAt"`
	LastMessages   []watchMessage `json:"lastMessages"`
}

type watchOutput struct {
	Conversations []watchConversation `json:"conversations"`
	Regras        string              `json:"regras,omitempty"`
}

type boughtLine struct {
	Nome     string  `json:"nome"`
	Detalhes string  `json:"detalhes,omitempty"`
	Qtd      int     `json:"qtd"`
	Preco    float64 `json:"preco"`
}

type offer struct {
	SKU     string  `json:"sku"`
	Name    string  `json:"name"`
	Meta    string  `json:"meta"`
	UnitBRL float64 `json:"unitBRL"`
}

type turn struct {
	Role string `json:"role"`
	Text string `json:"text"`
}

type inboundContext struct {
	Channel     string       `json:"channel"`
	FirstName   string       `json:"firstName"`
	OrderNumber int          `json:"orderNumber"`
	Pickup      bool         `json:"pickup,omitempty"`
	Cancelled   bool         `json:"cancelled,omitempty"`
	CanAddHow   string       `json:"canAddHow"`
	Status      string       `json:"status,omitempty"`
	Rastreio    string       `json:"rastreio,omitempty"`
	Bought      []boughtLine `json:"bought"`
	Offer       *offer       `json:"offer,omitempty"`
	WindowOpen  bool         `json:"windowOpen"`
	CanAddUntil string       `json:"canAddUntil,omitempty"`
	AutoSend    bool         `json:"autoSend"`
	History     []turn       `json:"history"`
}

type inboundOutput struct {
	Action         string          `json:"action"`
	Reason         string          `json:"reason,omitempty"`
	ConversationID string          `json:"conversationId"`
	Context        *inboundContext `json:"context,omitempty"`
	Regras         string          `json:"regras,omitempty"`
}

type replyOutput struct {
	Status string `json:"status"`
}

type linkOutput struct {
	URL string `json:"url"`
}

type composeOrderItem struct {
	Name string `json:"name"`
	Qty  int    `json:"qty"`
}

type composeItem struct {
	ConversationID string             `json:"conversationId"`
	Channel        string             `json:"channel"`
	FirstName      string             `json:"firstName"`
	OrderNumber    int                `json:"orderNumber"`
	Pickup         bool               `json:"pickup"`
	Shipped        bool               `json:"shipped"`
	Cancelled      bool               `json:"cancelled"`
	Items          []composeOrderItem `json:"items"`
	CanAddHow      string             `json:"canAddHow"`
}

type composeOutput struct {
	ComoEscrever string        `json:"como_escrever"`
	Items        []composeItem `json:"items"`
	Regras       string        `json:"regras,omitempty"`
}

const composeGuide = `Escreva como o Jairo, dono da loja, escreve do celular: 2 balões curtos separados por uma quebra de linha.
Balão 1: agradecimento + carinho com o que comprou, numa frase só. Um produto: "Oii [primeiro nome], tudo bem? Muito obrigado pela compra! 💙 Esse Jolteon é lindo demais, já tá separado aqui". Vários: "Oii [primeiro nome], tudo bem? Muito obrigado pela compra! 💙 Suas cartas já tão separadas aqui", sem nomes.
Balão 1 (shipped = true, o pedido já saiu): troque "já tá separado aqui" por "já tá a caminho" ("Suas cartas já tão a caminho"). Nunca diga que está separado.
Balão 2 (Liga, com envio): "E se quiser ver mais coisa, nosso site é esse aqui: https://www.collectatcg.com.br 🩷"
Balão 2 (site, com envio): "Se quiser aproveitar o mesmo frete e levar mais alguma coisa, me fala que eu adiciono no seu pedido 🩷"
Balão 2 (retirada): "Se quiser, separo mais alguma coisa pra você pegar junto 🩷"
Nunca: "aqui é da Collecta", "recebemos seu pedido", número do pedido, lista de cartas, quantidades, preço, passo a passo da Liga, travessão.
Pedido cancelado (cancelled = true, a Liga cancelou porque não foi pago): nada de agradecer pela compra. Balão 1: "Oii [primeiro nome], tudo bem? Vi que seu pedido lá na nossa loja da Liga acabou cancelando antes do pagamento, aconteceu alguma coisa? 💙" (um produto: "o pedido do Jolteon"). Balão 2: "Se ainda quiser, me fala que eu te ajudo a fechar 🩷". Nunca: "por que cancelou", cupom, desconto, valor, pressa, prometer que ainda tem.`

func (a *app) addPostSaleTools(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "posvenda_escrever",
		Description: "Lista os pedidos pagos (site ou Liga) que ainda precisam da primeira mensagem de pós-venda, com o primeiro nome do cliente, os itens, se é retirada e como ele pode adicionar mais coisas. Escreva a mensagem de cada um e entregue com posvenda_mensagem.",
	}, a.toolComposeList)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "posvenda_mensagem",
		Description: "Entrega a primeira mensagem de pós-venda que você escreveu para uma conversa de posvenda_escrever. Ela vira rascunho para o dono aprovar ou entra na fila de envio. NÃO digite no WhatsApp: ela volta depois por posvenda_envios.",
	}, a.toolCompose)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "posvenda_envios",
		Description: "Pega as mensagens de pós-venda que a loja precisa enviar agora (oferta aprovada, resposta aprovada, código de rastreio). Cada item fica reservado para você por 10 minutos: envie exatamente o texto e confirme com posvenda_confirmar, ou avise com posvenda_falhou.",
	}, a.toolClaim)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "posvenda_confirmar",
		Description: "Confirma que a mensagem de pós-venda com este id foi enviada no WhatsApp.",
	}, a.toolDelivered)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "posvenda_falhou",
		Description: "Avisa que a mensagem de pós-venda com este id não foi enviada. Use permanente = true só quando o número não tem WhatsApp.",
	}, a.toolFailed)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "posvenda_conversas",
		Description: "Lista as conversas de pós-venda abertas (oferta enviada e janela de 12h aberta), com o telefone e as últimas mensagens que a loja já conhece. Só olhe mensagens mais novas que essas.",
	}, a.toolWatch)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "posvenda_registrar",
		Description: "Registra as mensagens novas de uma conversa de pós-venda. Devolve a ação: answer (responda com posvenda_responder usando o contexto), handoff (o dono assume: marque como não lida) ou none (não faça nada).",
	}, a.toolInbound)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "posvenda_responder",
		Description: "Entrega sua resposta para uma conversa de pós-venda. Ela vira rascunho para o dono aprovar ou entra na fila de envio. NÃO digite a resposta no WhatsApp: ela volta depois por posvenda_envios.",
	}, a.toolReply)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "posvenda_humano",
		Description: "Passa uma conversa de pós-venda para o dono e pausa o atendente nela.",
	}, a.toolPostSaleHandoff)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "posvenda_link",
		Description: "Gera o link do complemento: os itens vão no mesmo envio do pedido, com frete zero. Só cartas avulsas em estoque (cabe_no_envelope = true no buscar_estoque) e só com a janela aberta.",
	}, a.toolLink)
}

func (a *app) agentRequest(ctx context.Context, method, path string, payload []byte) ([]byte, error) {
	var body io.Reader
	if payload != nil {
		body = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, a.cfg.OpdealsURL+"/api/postsale/agent/"+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+a.cfg.AgentToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("servidor da loja fora do ar: %v", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		var e struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(data, &e) == nil && e.Error != "" {
			return nil, errors.New(e.Error)
		}
		return nil, fmt.Errorf("servidor da loja respondeu %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	return data, nil
}

func agentPost[In, Out any](a *app, ctx context.Context, path string, in In) (Out, error) {
	var out Out
	payload, err := json.Marshal(in)
	if err != nil {
		return out, err
	}
	data, err := a.agentRequest(ctx, http.MethodPost, path, payload)
	if err != nil {
		return out, err
	}
	return out, json.Unmarshal(data, &out)
}

func conversationPath(id, action string) string {
	return "conversations/" + url.PathEscape(id) + "/" + action
}

func (a *app) toolClaim(ctx context.Context, _ *mcp.CallToolRequest, _ noInput) (*mcp.CallToolResult, claimOutput, error) {
	if a.cfg.Mode == modeDraft {
		return nil, claimOutput{Items: []outboxItem{}, Aviso: "modo rascunho: o pós-venda não envia nada"}, nil
	}
	claim, err := agentPost[struct{}, claimOutput](a, ctx, "outbox/claim", struct{}{})
	if err != nil {
		return nil, claimOutput{}, err
	}
	kept := make([]outboxItem, 0, len(claim.Items))
	for _, it := range claim.Items {
		if a.cfg.Mode == modeTest && !a.cfg.TestPhones[whatsapp.PhoneKey(it.Phone)] {
			continue
		}
		kept = append(kept, it)
	}
	a.mu.Lock()
	for _, it := range kept {
		a.pending[it.ID] = it
	}
	a.mu.Unlock()
	claim.Items = kept
	if len(kept) > 0 {
		claim.Regras = rules[rulesPostSaleDelivery]
	}
	return nil, claim, nil
}

type outboxIDInput struct {
	ID string `json:"id" jsonschema:"o id do item devolvido por posvenda_envios"`
}

func (a *app) toolDelivered(ctx context.Context, _ *mcp.CallToolRequest, in outboxIDInput) (*mcp.CallToolResult, okOutput, error) {
	if _, err := agentPost[struct{}, okOutput](a, ctx, "outbox/"+url.PathEscape(in.ID)+"/delivered", struct{}{}); err != nil {
		return nil, okOutput{}, err
	}
	a.mu.Lock()
	it, ok := a.pending[in.ID]
	delete(a.pending, in.ID)
	a.mu.Unlock()
	if ok {
		if key, err := phoneKey(it.Phone); err == nil {
			if err := a.history.add(key, it.Text, a.now()); err != nil {
				return nil, okOutput{}, fmt.Errorf("envio confirmado, mas não consegui registrar no histórico: %v", err)
			}
		}
	}
	return nil, okOutput{OK: true}, nil
}

type failedInput struct {
	ID         string `json:"id" jsonschema:"o id do item devolvido por posvenda_envios"`
	Erro       string `json:"erro" jsonschema:"o que deu errado"`
	Permanente bool   `json:"permanente,omitempty" jsonschema:"true só se o número não tem WhatsApp"`
}

type failedBody struct {
	Error     string `json:"error"`
	Permanent bool   `json:"permanent"`
}

func (a *app) toolFailed(ctx context.Context, _ *mcp.CallToolRequest, in failedInput) (*mcp.CallToolResult, okOutput, error) {
	if _, err := agentPost[failedBody, okOutput](a, ctx, "outbox/"+url.PathEscape(in.ID)+"/failed", failedBody{Error: in.Erro, Permanent: in.Permanente}); err != nil {
		return nil, okOutput{}, err
	}
	a.mu.Lock()
	delete(a.pending, in.ID)
	a.mu.Unlock()
	return nil, okOutput{OK: true}, nil
}

func (a *app) toolWatch(ctx context.Context, _ *mcp.CallToolRequest, _ noInput) (*mcp.CallToolResult, watchOutput, error) {
	var out watchOutput
	data, err := a.agentRequest(ctx, http.MethodGet, "watch", nil)
	if err != nil {
		return nil, out, err
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, out, err
	}
	if len(out.Conversations) > 0 {
		out.Regras = rules[rulesPostSaleResponses]
	}
	return nil, out, nil
}

type inboundMessage struct {
	Texto  string `json:"texto" jsonschema:"o texto da mensagem"`
	Hora   string `json:"hora,omitempty" jsonschema:"a hora que aparece na mensagem, ex.: 14:05"`
	DaLoja bool   `json:"da_loja,omitempty" jsonschema:"true se a mensagem foi enviada pela loja (lado direito, com os tiques)"`
}

type inboundInput struct {
	ConversaID string           `json:"conversa_id" jsonschema:"conversationId devolvido por posvenda_conversas"`
	Mensagens  []inboundMessage `json:"mensagens" jsonschema:"as mensagens novas, da mais antiga para a mais nova"`
}

type serverMessage struct {
	Text      string `json:"text"`
	At        string `json:"at,omitempty"`
	FromStore bool   `json:"fromStore,omitempty"`
}

type inboundBody struct {
	Messages []serverMessage `json:"messages"`
}

func (a *app) toolInbound(ctx context.Context, _ *mcp.CallToolRequest, in inboundInput) (*mcp.CallToolResult, inboundOutput, error) {
	body := inboundBody{Messages: make([]serverMessage, 0, len(in.Mensagens))}
	for _, m := range in.Mensagens {
		body.Messages = append(body.Messages, serverMessage{Text: m.Texto, At: m.Hora, FromStore: m.DaLoja})
	}
	out, err := agentPost[inboundBody, inboundOutput](a, ctx, conversationPath(in.ConversaID, "inbound"), body)
	if err == nil && out.Action == "answer" {
		out.Regras = rules[rulesConversation]
	}
	return nil, out, err
}

type replyInput struct {
	ConversaID string `json:"conversa_id" jsonschema:"conversationId da conversa"`
	Texto      string `json:"texto" jsonschema:"a resposta para o cliente"`
}

type textBody struct {
	Text string `json:"text"`
}

func (a *app) toolReply(ctx context.Context, _ *mcp.CallToolRequest, in replyInput) (*mcp.CallToolResult, replyOutput, error) {
	if strings.ContainsAny(in.Texto, "—–") {
		return nil, replyOutput{}, errors.New("troque o travessão por vírgula ou ponto")
	}
	out, err := agentPost[textBody, replyOutput](a, ctx, conversationPath(in.ConversaID, "reply"), textBody{Text: in.Texto})
	return nil, out, err
}

const ligaComposeHint = "liga: mande o site da Collecta (https://www.collectatcg.com.br) para ele ver mais coisa. Não explique a opção da Liga e não prometa frete junto"

type serverComposeItem struct {
	composeItem
	Phone string `json:"phone"`
}

const cancelSkipReason = "o atendente já mandou mensagem para esse número nos últimos 7 dias"

func (a *app) toolComposeList(ctx context.Context, _ *mcp.CallToolRequest, _ noInput) (*mcp.CallToolResult, composeOutput, error) {
	out := composeOutput{Items: []composeItem{}}
	data, err := a.agentRequest(ctx, http.MethodGet, "compose", nil)
	if err != nil {
		return nil, out, err
	}
	var listed struct {
		Items []serverComposeItem `json:"items"`
	}
	if err := json.Unmarshal(data, &listed); err != nil {
		return nil, out, err
	}
	for _, it := range listed.Items {
		item := it.composeItem
		if item.Cancelled {
			recent, err := a.messagedRecently(it.Phone)
			if err != nil {
				return nil, out, err
			}
			if recent {
				if _, err := agentPost[reasonBody, okOutput](a, ctx, conversationPath(item.ConversationID, "skip"), reasonBody{Reason: cancelSkipReason}); err != nil {
					return nil, out, err
				}
				continue
			}
		} else if item.Channel == "liga" {
			item.CanAddHow = ligaComposeHint
		}
		out.Items = append(out.Items, item)
	}
	out.ComoEscrever = composeGuide
	if len(out.Items) > 0 {
		out.Regras = rules[rulesPostSaleCompose]
	}
	return nil, out, nil
}

func (a *app) messagedRecently(phone string) (bool, error) {
	key, err := phoneKey(phone)
	if err != nil {
		return false, nil
	}
	sent, err := a.history.since(key, a.now().Add(-historyWindow))
	return len(sent) > 0, err
}

type composeInput struct {
	ConversaID string `json:"conversa_id" jsonschema:"conversationId devolvido por posvenda_escrever"`
	Texto      string `json:"texto" jsonschema:"a primeira mensagem para o cliente, com quebra de linha entre os balões"`
}

func (a *app) toolCompose(ctx context.Context, _ *mcp.CallToolRequest, in composeInput) (*mcp.CallToolResult, replyOutput, error) {
	if strings.ContainsAny(in.Texto, "—–") {
		return nil, replyOutput{}, errors.New("troque o travessão por vírgula ou ponto")
	}
	out, err := agentPost[textBody, replyOutput](a, ctx, conversationPath(in.ConversaID, "compose"), textBody{Text: in.Texto})
	return nil, out, err
}

type postSaleHandoffInput struct {
	ConversaID string `json:"conversa_id" jsonschema:"conversationId da conversa"`
	Motivo     string `json:"motivo" jsonschema:"por que o dono precisa assumir"`
}

type reasonBody struct {
	Reason string `json:"reason"`
}

func (a *app) toolPostSaleHandoff(ctx context.Context, _ *mcp.CallToolRequest, in postSaleHandoffInput) (*mcp.CallToolResult, handoffOutput, error) {
	if _, err := agentPost[reasonBody, okOutput](a, ctx, conversationPath(in.ConversaID, "handoff"), reasonBody{Reason: in.Motivo}); err != nil {
		return nil, handoffOutput{}, err
	}
	return nil, handoffOutput{OK: true, Proximo: "ponha a conversa na lista Humano (Adicionar à lista, no cabeçalho) e marque como não lida"}, nil
}

type linkItem struct {
	SKU string `json:"sku" jsonschema:"o sku que veio no buscar_estoque"`
	Qtd int    `json:"qtd" jsonschema:"quantidade"`
}

type linkInput struct {
	ConversaID string     `json:"conversa_id" jsonschema:"conversationId da conversa"`
	Itens      []linkItem `json:"itens" jsonschema:"itens que o cliente quer adicionar"`
}

type linkBody struct {
	Itens []linkItem `json:"itens"`
}

func (a *app) toolLink(ctx context.Context, _ *mcp.CallToolRequest, in linkInput) (*mcp.CallToolResult, linkOutput, error) {
	out, err := agentPost[linkBody, linkOutput](a, ctx, conversationPath(in.ConversaID, "link"), linkBody{Itens: in.Itens})
	return nil, out, err
}
