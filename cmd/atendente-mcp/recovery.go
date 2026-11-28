package main

// Recuperação de pedido do site que não foi pago (Pix expirado ou pagamento recusado).
// Pedido do Jairo de 26/11/2026, no chat do CEO Collecta: "faça precisamos de algo automatico aqui".
// Ele escolheu: só WhatsApp, pelo Atendente; o "Texto 1 (geral)"; sem cupom.
//
// O servidor monta a lista e a mensagem (PR #6 do collecta-deals) e fica desligado até o Jairo
// ligar RECUPERAR_PEDIDOS=1 e RECUPERAR_PEDIDOS_TEXTO no Railway. Desligado, ou com o servidor
// sem a rota, esta ferramenta devolve a lista vazia e o Atendente segue a rodada normal.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"opdeals/internal/whatsapp"
)

const rulesRecovery = "recuperacao"

type recoveryOrderOut struct {
	ID       string `json:"id"`
	Numero   int    `json:"numero"`
	Nome     string `json:"primeiroNome"`
	Telefone string `json:"telefone"`
	Mensagem string `json:"mensagem"`
}

type recoveryListOut struct {
	Ligado  bool               `json:"ligado"`
	Fora    bool               `json:"foraDoHorario,omitempty"`
	Pedidos []recoveryOrderOut `json:"pedidos"`
	Aviso   string             `json:"aviso,omitempty"`
	Regras  string             `json:"regras,omitempty"`
}

type recoveryIDInput struct {
	ID string `json:"id" jsonschema:"o id do pedido devolvido por recuperar_pedidos"`
}

type recoveryFailedInput struct {
	ID   string `json:"id" jsonschema:"o id do pedido devolvido por recuperar_pedidos"`
	Erro string `json:"erro" jsonschema:"o que a tela do WhatsApp mostrou (número inválido, sem WhatsApp)"`
}

func (a *app) addRecoveryTools(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "recuperar_pedidos",
		Description: "Lista os pedidos do site que não foram pagos e devem receber agora a mensagem de recuperação (uma por pedido, das 9h às 21h), com o telefone e a mensagem pronta. Lista vazia quando a recuperação está desligada ou fora do horário. Quando há pedido, o campo regras diz como enviar.",
	}, a.toolRecoveryList)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "recuperar_enviado",
		Description: "Marca que a mensagem de recuperação deste pedido foi enviada e aparece como Entregue no WhatsApp. Uma vez por pedido.",
	}, a.toolRecoverySent)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "recuperar_falhou",
		Description: "Avisa que o número deste pedido não tem WhatsApp ou é inválido. O pedido sai da lista para não tentar de novo.",
	}, a.toolRecoveryFailed)
}

func (a *app) recoverySkipPath() string {
	return filepath.Join(filepath.Dir(a.history.path), "recuperacao-falhou.json")
}

func (a *app) recoverySkipped() map[string]bool {
	out := map[string]bool{}
	data, err := os.ReadFile(a.recoverySkipPath())
	if err == nil {
		_ = json.Unmarshal(data, &out)
	}
	return out
}

func (a *app) toolRecoveryList(ctx context.Context, _ *mcp.CallToolRequest, _ noInput) (*mcp.CallToolResult, recoveryListOut, error) {
	out := recoveryListOut{Pedidos: []recoveryOrderOut{}}
	if a.cfg.Mode == modeDraft {
		out.Aviso = "modo rascunho: a recuperação não envia nada"
		return nil, out, nil
	}
	data, err := a.agentRequest(ctx, http.MethodGet, "recovery/orders", nil)
	if err != nil {
		// Servidor sem a rota (PR ainda não publicado) ou recuperação desligada: segue a rodada.
		out.Aviso = "recuperação de pedidos desligada ou indisponível no servidor; siga a rodada"
		return nil, out, nil
	}
	var list recoveryListOut
	if err := json.Unmarshal(data, &list); err != nil {
		// Servidor antigo devolve a página do site (HTML) para rota que não conhece.
		out.Aviso = "recuperação de pedidos desligada ou indisponível no servidor; siga a rodada"
		return nil, out, nil
	}
	out.Ligado, out.Fora = list.Ligado, list.Fora
	skipped := a.recoverySkipped()
	for _, p := range list.Pedidos {
		if skipped[p.ID] || strings.TrimSpace(p.Mensagem) == "" || strings.TrimSpace(p.Telefone) == "" {
			continue
		}
		if a.cfg.Mode == modeTest && !a.cfg.TestPhones[whatsapp.PhoneKey(p.Telefone)] {
			continue
		}
		out.Pedidos = append(out.Pedidos, p)
	}
	if len(out.Pedidos) > 0 {
		a.mu.Lock()
		for _, p := range out.Pedidos {
			a.pending["recuperacao:"+p.ID] = outboxItem{ID: p.ID, Phone: p.Telefone, Text: p.Mensagem}
		}
		a.mu.Unlock()
		out.Regras = rules[rulesRecovery]
	}
	return nil, out, nil
}

func (a *app) toolRecoverySent(ctx context.Context, _ *mcp.CallToolRequest, in recoveryIDInput) (*mcp.CallToolResult, okOutput, error) {
	if _, err := a.agentRequest(ctx, http.MethodPost, "recovery/orders/"+url.PathEscape(in.ID)+"/sent", []byte("{}")); err != nil {
		return nil, okOutput{}, err
	}
	a.mu.Lock()
	it, ok := a.pending["recuperacao:"+in.ID]
	delete(a.pending, "recuperacao:"+in.ID)
	a.mu.Unlock()
	if ok {
		if key, err := phoneKey(it.Phone); err == nil {
			if err := a.history.add(key, it.Text, "recuperacao", a.now()); err != nil {
				return nil, okOutput{}, fmt.Errorf("envio marcado, mas não consegui registrar no histórico: %v", err)
			}
		}
	}
	return nil, okOutput{OK: true}, nil
}

func (a *app) toolRecoveryFailed(_ context.Context, _ *mcp.CallToolRequest, in recoveryFailedInput) (*mcp.CallToolResult, okOutput, error) {
	skipped := a.recoverySkipped()
	skipped[in.ID] = true
	data, err := json.Marshal(skipped)
	if err != nil {
		return nil, okOutput{}, err
	}
	if err := os.MkdirAll(filepath.Dir(a.recoverySkipPath()), 0o755); err != nil {
		return nil, okOutput{}, err
	}
	if err := os.WriteFile(a.recoverySkipPath(), data, 0o644); err != nil {
		return nil, okOutput{}, err
	}
	return nil, okOutput{OK: true}, nil
}
