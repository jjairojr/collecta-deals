---
name: baixa-vendas
description: Rodada única de baixa de vendas em todos os canais da Collecta — Liga, MyPCards e Site — dando baixa no dashboard de prod e tirando cada item vendido dos outros canais. Roda agendada (launchd, 3x/dia, argumento `--auto`) ou à mão. Use quando o pedido for "dar baixa em tudo", "baixa de vendas", "conciliar as vendas de todos os canais" ou ao ser chamada pelo scripts/baixa-vendas.sh.
allowed-tools: Bash, Read, Write, Edit, mcp__claude-in-chrome__*
---

# Baixa de vendas em todos os canais

Orquestra três fontes de venda sobre o mesmo estoque:

| canal | onde a venda nasce | dedup no `buyer` | skill com a mecânica |
| --- | --- | --- | --- |
| Liga | `?view=ecom/admin/pedidos` | `Liga #N` | `liga-vendas` |
| MyP | `/CollectaTCG/vendas` | `MyP #N` | `myp-vendas` |
| Site | já entra no dashboard pelo webhook do Mercado Pago | `Site #N` | — |

Leia **inteiras** `.claude/skills/liga-vendas/SKILL.md` e
`.claude/skills/myp-vendas/SKILL.md` antes de começar: regras duras, URLs,
formatos e aprendizados estão lá e valem aqui sem exceção.

## Modo `--auto`

Quando o argumento contém `--auto` (execução agendada, ninguém olhando):

- **Nunca pergunte, nunca chute.** Tudo que nas outras skills diz "pare e
  pergunte ao dono" vira pendência no relatório, e aquele item não é gravado em
  canal nenhum. O resto do pedido segue normalmente.
- **Bloqueio do Cloudflare** (Error 1007, desafio que não resolve em ~15 s,
  página de login no lugar do admin): pare aquele canal, registre no relatório
  e siga para o próximo canal. Não tente de novo na mesma execução.
- **Extensão do Chrome fora do ar**: escreva o relatório só com isso e encerre.
- O segundo argumento é o carimbo da execução (`AAAA-MM-DD-HHMM`); sem ele, use
  a hora atual.

## Ordem

1. **Fase 0 compartilhada**: baixe os ledgers de prod dos 3 jogos uma vez só e
   monte os conjuntos de pedidos já baixados (`Liga #`, `MyP #`, `Site #`).
2. **Liga → dashboard + MyP** (`liga-vendas` inteira, incluindo tirar do MyP).
3. **MyP → dashboard + Liga** (`myp-vendas` inteira).
4. **Site → Liga + MyP**: para cada `sold` com `Site #N` e `sellDate` nos
   últimos 7 dias, aplique a mesma regra de propagação abaixo nos dois canais.
5. Releia os ledgers e confira que cada baixa feita está lá (`buyer` certo,
   qty certa). Baixa que não aparecer = pendência.

Releia o ledger entre as etapas 2 e 3: a etapa 2 muda ids e quantidades.

## Regra de propagação (idempotente)

Para cada item vendido e cada canal que precisa refletir a venda, compare a
quantidade anunciada no canal (`canal`) com o que sobrou no dashboard para o
mesmo produto (número + variante + preço; `resto` = soma do `qty` dos holdings):

| situação | ação |
| --- | --- |
| `canal` ≤ `resto` | já refletido — não mexa |
| `canal` > `resto` | gravar `resto` (zerou → excluir/zerar anúncio) |
| item sem lastro no dashboard | não mexa no canal; pendência |

Nunca aumente quantidade em canal nenhum. Isso torna a rodada segura de repetir:
rodar duas vezes seguidas não muda nada na segunda.

Antes de gravar uma redução na Liga ou no MyP, confira se o mesmo item não
vendeu também naquele canal desde a última rodada (pedido novo ainda não
processado) — se vendeu, é **venda dupla**: pendência marcada `URGENTE`.

## Relatório

Escreva `data/baixa-vendas/<carimbo>.md` (crie a pasta se faltar). A **primeira
linha** é o resumo que vai para o push no celular, no máximo 120 caracteres:

```
3 baixas (Liga 2, MyP 1) · 1 pendência · 0 URGENTE
```

ou `nada novo` quando não houve venda. Depois:

- tabela das baixas: `Canal | Pedido | Data | Comprador | Item | Qty | Total R$ | Game | Ledger id | Propagado`;
- **Pendências** (uma por linha, com pedido, item e motivo) — URGENTE primeiro;
- pedidos pulados e por quê;
- canais que não rodaram (Cloudflare, sessão deslogada) e por quê.

Sem dados pessoais do comprador além do nome/nick (nada de CPF, endereço,
telefone).

## Aprendizados

Aprendizados de mecânica de cada canal vão nas skills `liga-vendas` e
`myp-vendas`. Aqui, só o que for da orquestração ou do modo agendado.

2026-10-24 (1ª rodada completa, interativa — 7 pedidos Liga, 18 baixas):

- Ordem que rendeu: listagem da Liga + fetch dos detalhes em segundo plano
  (`window.__lv`, 8 s entre pedidos) e, em paralelo, numa segunda aba, a
  listagem do MyP — hosts diferentes, não disputam o Cloudflare.
- Propagação para o MyP: uma busca por item em
  `/CollectaTCG/<pokemon|onepiece|riftbound>?PastaSearch[query]=<nº ou nome>`
  (`li.stream-item` com link `/estoque/update/`: `data-key`, `data-ga-item-id`,
  `.quantidade-num`, `.card-preco`). Reduzir por `fetch` POST do form de
  `/estoque/update/<id>` funcionou (flash "N x R$ X gravado com sucesso");
  excluir pelo `$.ajax` sem await e conferir 404.
- No MyP o `resto` é por set+número+variante (um anúncio agrupa lotes de
  preços diferentes, ex. Tyrunt M3 089: 4 a 14,90 + 1 a 14,99 = 5).
- Selados vendidos na Liga (Storm Emeralda, Kit Base Shop) não estão no MyP —
  só 8 selados foram anunciados lá; nada a propagar.
- Site → Liga pegou divergência no sentido contrário (Shanks OP17-022: Liga 1,
  dashboard 5). A regra não aumenta; vira alerta.
- Vendas de balcão/live (buyer `Balcão`, `Aberto em live`) também tiram estoque
  e não têm etapa aqui; listar no relatório os `sold` desses buyers no período.
