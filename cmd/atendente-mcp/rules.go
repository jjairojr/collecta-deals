package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	rulesRound             = "rodada"
	rulesConversation      = "conversa"
	rulesPostSaleCompose   = "posvenda_primeira"
	rulesPostSaleDelivery  = "posvenda_envios"
	rulesPostSaleResponses = "posvenda_respostas"
	rulesInstagram         = "instagram"
	rulesCarts             = "carrinhos"
)

var ruleGroupByHeading = map[string]string{
	"":                              rulesRound,
	"A cada rodada":                 rulesRound,
	"Navegador":                     rulesRound,
	"Marcar como não lida":          rulesRound,
	"Regras que nunca mudam":        rulesRound,
	"Antes de enviar, confira":      rulesConversation,
	"Ver foto que o cliente mandou": rulesConversation,
	"Como falar":                    rulesConversation,
	"Como conduzir a conversa":      rulesConversation,
	"Quando passar para o Jairo":    rulesConversation,
	"Dados da loja":                 rulesConversation,
	"Pós-venda: primeira mensagem":  rulesPostSaleCompose,
	"Pós-venda: envios":             rulesPostSaleDelivery,
	"Pós-venda: respostas":          rulesPostSaleResponses,
	"Instagram":                     rulesInstagram,
	"Carrinhos abandonados (mensagem do Pesquisador Collecta)": rulesCarts,
}

var rules = mustSplitRules(instructions)

func splitRules(src string) (map[string]string, error) {
	parts := map[string]*strings.Builder{}
	group := ruleGroupByHeading[""]
	for _, line := range strings.SplitAfter(src, "\n") {
		if heading, ok := strings.CutPrefix(line, "## "); ok {
			g, known := ruleGroupByHeading[strings.TrimSpace(heading)]
			if !known {
				return nil, fmt.Errorf("seção sem grupo nas regras: %q", strings.TrimSpace(heading))
			}
			group = g
		}
		if parts[group] == nil {
			parts[group] = &strings.Builder{}
		}
		parts[group].WriteString(line)
	}
	out := make(map[string]string, len(parts))
	for g, b := range parts {
		out[g] = strings.TrimSpace(b.String())
	}
	return out, nil
}

func mustSplitRules(src string) map[string]string {
	out, err := splitRules(src)
	if err != nil {
		panic(err)
	}
	return out
}

var rulesTopics = map[string][]string{
	"conversa":  {rulesConversation},
	"posvenda":  {rulesPostSaleCompose, rulesPostSaleDelivery, rulesPostSaleResponses},
	"instagram": {rulesInstagram},
	"carrinhos": {rulesCarts},
}

type rulesInput struct {
	Secao string `json:"secao" jsonschema:"instagram, carrinhos, conversa ou posvenda"`
}

type rulesOutput struct {
	Secao  string `json:"secao"`
	Regras string `json:"regras"`
}

func (a *app) toolRules(_ context.Context, _ *mcp.CallToolRequest, in rulesInput) (*mcp.CallToolResult, rulesOutput, error) {
	topic := strings.ToLower(strings.TrimSpace(in.Secao))
	groups, ok := rulesTopics[topic]
	if !ok {
		return nil, rulesOutput{}, fmt.Errorf("seção desconhecida: %q (use instagram, carrinhos, conversa ou posvenda)", in.Secao)
	}
	texts := make([]string, 0, len(groups))
	for _, g := range groups {
		texts = append(texts, rules[g])
	}
	return nil, rulesOutput{Secao: topic, Regras: strings.Join(texts, "\n\n")}, nil
}
