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

Toda baixa manda `channel` no `/sell` (`liga`, `myp`, `site`, `balcao`, `outros`); a venda
direta (balcão, live, WhatsApp, pagou por fora) é `balcao`. Venda antiga sem `channel` tem o
canal deduzido do buyer na leitura (`/api/sales` já devolve resolvido): filtre pelo `channel`
em vez de regex no buyer.

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
4. **Fila de sincronização → Liga + MyP**: toda saída de estoque fora da Liga
   (Site, balcão, MyP pago por fora, baixa sem venda) de item marcado "Na Liga"
   vira tarefa (pedido do MyP com `MyP #N` não vira: a `myp-vendas` já baixa a Liga) em `GET $OPDEALS_URL/api/syncqueue?status=pending` (header
   `X-Admin-Token`; resposta `{"tasks":[...]}` com `game`, `tradeID`, `name`,
   `number`, `variant`, `condition`, `action` `remove`|`restore`, `qty`, `reason`).
   Para cada tarefa: ache a oferta na Liga por número + nome + variante (mesma
   mecânica da Fase 5 da `myp-vendas`), tire (`remove`) ou devolva (`restore`)
   exatamente `qty` unidades, reduza/aumente o MyP pelas mesmas unidades e
   confira lendo a oferta de volta. Deu certo → `POST /api/syncqueue/<id>/done`.
   Não achou, achou mais de uma ou a Liga já está abaixo do esperado →
   `POST /api/syncqueue/<id>/error` com `{"error":"<motivo>"}` e pendência no
   relatório; nunca chute. A tela **Sincronizar** do dashboard mostra a fila e
   o dono pode marcar feita à mão. Vendas `Site #N`/balcão anteriores à fila
   (sem tarefa) seguem a regra de propagação abaixo, últimos 7 dias.
5. Releia os ledgers e confira que cada baixa feita está lá (`buyer` certo,
   qty certa). Baixa que não aparecer = pendência.
6. **Pós-venda da Liga**: rode a Fase 7 da `liga-vendas` (pedidos `Pagamento
   efetuado - Aguardando envio` pagos nas últimas 24h → `POST
   /api/postsale/liga-orders`). Reaproveite os detalhes já abertos na etapa 2;
   só primeiro nome, telefone, itens e retirada saem da Liga. No relatório, só
   as contagens `created`/`skipped`.

Releia o ledger entre as etapas 2 e 3: a etapa 2 muda ids e quantidades.

## Regra de propagação (idempotente)

**A Liga é a fonte da verdade** de quantidade e preço (regra do dono, 24/10/2026).
O dono cadastra estoque novo direto na Liga, então Liga acima do dashboard quer
dizer que o dashboard está atrasado, não que a Liga errou.

Depois de cada baixa, o alvo de cada canal é a quantidade **da Liga** para o
produto (a Liga já desconta sozinha as vendas dela; para venda do MyP/Site, a Liga
é reduzida só pelas unidades vendidas — decisão na Fase 5 da `myp-vendas`).

| situação | ação |
| --- | --- |
| MyP ≠ Liga | gravar no MyP a quantidade e o preço da Liga (zerou → excluir anúncio) |
| dashboard < Liga | lote extra no dashboard com a diferença (`buyBRL 0`, nota "Cadastrado a partir do estoque da LigaMagic <data> (lote extra, Liga tinha mais que o dashboard)") |
| dashboard > Liga | pendência no relatório (a redução é decisão do dono: reduzir via PUT do registro completo; qty 0 exige DELETE) |
| item sem lastro no dashboard | não mexa no canal; pendência |

Nunca reduza a Liga para bater com o dashboard. Pokémon JP anunciado duas
vezes na Liga (edição JP + edição EN) conta uma vez só.

Antes de gravar uma redução no MyP, confira se o mesmo item não vendeu também
lá desde a última rodada (pedido novo ainda não processado) — se vendeu, é
**venda dupla**: pendência marcada `URGENTE`.

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

2026-10-30 (rodada interativa — 3 pedidos Liga, 23 baixas, 61 ajustes no MyP):

- **Baixa manual do dono não desce para a Liga.** `sold` com buyer sem `Liga #`/`MyP #`/`Site #`
  ("MYP" = cliente que veio do MyP e pagou por fora, "Balcão", "site", "Venda por fora") é
  venda real: reduza a Liga e o MyP por essas unidades na etapa 4. Nunca crie lote extra
  para item que tenha uma baixa manual recente (Shiryu OP16-108 e Rob Lucci ST29-013-FA
  ficaram com a Liga acima do dashboard por isso).
- O MyP ficou dias sem receber as vendas da Liga. Varrer o MyP inteiro do jogo e cruzar
  com o estoque atual da Liga (`cartas/all&txt_filtro=1`) por número + preço resolve tudo
  de uma vez (mecânica na `myp-vendas`).

2026-11-01 (fila Sincronizar à mão — 8 tarefas: Site #24 com 6 cartas Pokémon JP + 2 GEM Pack de balcão):

- **Carta JP com anúncio gêmeo na Liga sai dos dois.** O dono anuncia a mesma AR JP na edição JP
  e na EN equivalente, com a mesma qty (Tauros 094/083 + 096/086, Ambipom 092/080 + 107/094,
  Snorunt 200/193 + 227/217, Ethan's Magcargo 197/193 + 222/217; o preço pode diferir). O ledger
  tem um registro só, então a tarefa de `qty` N tira N de cada gêmeo. Gêmeo = mesmo nome,
  idioma JP, mesma qty e soma das linhas maior que o ledger.
- Selado não aparece em `cartas/all`: o admin é `?view=ecom/admin/prod/all`, formulário `bcS`
  (`txt_produto` = nome), mesmos campos `txt_qty_typed[k]`/`txt_preco[k]`/`btSalvar`; a busca
  precisa ser submetida pelo próprio form para a gravação funcionar. Flash: "Produtos de sua
  lista alterados com sucesso." GEM Pack não está anunciado no MyP.
- Liga já zerada à mão (Goldeen 084, GEM Pack Vol.4): só marcar a tarefa como feita.
