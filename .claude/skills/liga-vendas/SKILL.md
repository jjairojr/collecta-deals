---
name: liga-vendas
description: Confere os pedidos da loja Collecta na LigaMagic (?view=ecom/admin/pedidos) e dá baixa de cada venda no dashboard de prod via POST /api/trades/{id}/sell, com dedup por nº de pedido no campo buyer. Use quando o pedido for verificar vendas da Liga, dar baixa no estoque, "vendeu na liga?", "vendemos algo?", ou conciliar pedidos da Liga com o dashboard.
allowed-tools: Bash, Read, Write, Edit, mcp__claude-in-chrome__*
---

# Conferir vendas da Liga e dar baixa no dashboard

Loja id=866280 na LigaMagic. Esta skill lê os pedidos da loja no admin da Liga,
identifica vendas ainda não registradas no dashboard, e dá baixa de cada uma em
**produção** (Railway) via `POST /api/trades/{id}/sell`. A fonte da verdade dos
pedidos é o admin da Liga; a fonte da verdade do estoque é a prod — nunca
`data/*.json` local.

Uma venda na Liga NÃO volta sozinha para o dashboard (`liga-conferir` documenta
esse gap): o item vendido continua como holding no Estoque até alguém dar baixa.
Esta skill fecha esse gap. Cadastrar/republicar é assunto das `liga-singles-*`;
auditoria completa da flag "Na Liga" é a skill `liga-conferir`.

---

## 🚫 REGRAS DURAS

1. **Dedup por pedido.** Todo sell gravado por esta skill carrega
   `Liga #<pedido>` no campo `buyer`. Antes de baixar qualquer coisa, monte o
   conjunto de pedidos já processados (Fase 0) e NUNCA reprocesse um pedido que
   já está lá. Baixa dupla = estoque fantasma negativo e P&L dobrado.
2. **`sellPrice` é o TOTAL da venda, não o unitário** (unitário × qty). O
   handler grava o que receber sem multiplicar. Valor **bruto**: o que o
   comprador pagou pelo item, sem descontar taxa da Liga (decisão do dono,
   2026-08-01). Frete fica fora.
3. **Case por NOME + número, nunca por coincidência estrutural.** Em 2026-07-30
   casar por "quantidade de cartas do set" publicou Kyogre no lugar de
   Venusaur. Ambiguidade (dois candidatos sobrevivem ao filtro) → pare e
   pergunte ao dono, nunca chute.
4. **NUNCA use `PUT /api/trades/{id}` para "só baixar quantidade".** O PUT
   substitui o registro inteiro (`*t = in` no handler): mandar `{"qty":1}`
   apaga nome, set, askBRL e flags da Liga. Baixa é sempre `/sell`.
5. **Estoque real vive na Railway.** Toda leitura e escrita vai em
   `https://collecta-deals-production.up.railway.app`.
6. **Ritmo — já causou incidente.** Rajada de navegações gera **Cloudflare
   Error 1007 (ban de IP)** e o ban derruba o acesso do dono também. 7–10s
   entre navegações, ~2 páginas por batch, e **pare e avise** ao primeiro
   sinal de bloqueio.
7. **Pedido só conta com pagamento confirmado.** Cancelado, aguardando
   pagamento ou em disputa não gera baixa. Na dúvida sobre um status, pergunte.

---

## Fase 0 — Ler o estoque de prod e o histórico de baixas

Puxe os trades de todos os jogos com estoque na Liga e extraia os pedidos já
baixados (regex `Liga #(\d+)` sobre o `buyer` dos `status:"sold"`):

```bash
B=https://collecta-deals-production.up.railway.app
for g in onepiece pokemon riftbound; do
  curl -s "$B/api/trades?game=$g" > /tmp/trades-$g.json
done
python3 - <<'EOF'
import json, re
done = set()
for g in ["onepiece", "pokemon", "riftbound"]:
    for t in json.load(open(f"/tmp/trades-{g}.json"))["trades"]:
        if t.get("status") == "sold":
            m = re.search(r"Liga #(\d+)", t.get("buyer", ""))
            if m:
                done.add(m.group(1))
print("pedidos ja baixados:", sorted(done) or "nenhum")
EOF
```

⚠️ A resposta é um envelope `{"targetPct","fxRate","summary","trades":[...]}` —
os registros estão em `["trades"]`, não na raiz.

⚠️ **O dono também dá baixa manualmente**, com `buyer` vazio ou só `"Liga"`,
sem número de pedido — o dedup da regra 1 não enxerga essas. Na primeira
execução (2026-08-01) TODOS os 3 pedidos pendentes já tinham sido baixados
assim no mesmo dia (às vezes com sellPrice líquido ~96% do bruto, ex.: Luffy
R$ 310 → sold R$ 297,60). Por isso, antes de baixar qualquer item, procure um
`sold` do MESMO nome+número com `sellDate` próximo da data do pedido — bateu,
trate como já baixado, liste no relatório e siga para a Fase 5 (o ligaQty do
holding restante costuma ficar defasado nessas baixas manuais). Não baixe de
novo e não tente "consertar" o buyer via PUT (regra 4).

Guarde também os holdings em memória: você vai precisar de `id`, `name`,
`number`, `qty`, `ligaListed`, `ligaQty`, `ligaPriceBRL` para casar e para a
Fase 5. `game=` inválido não dá erro — cai em `onepiece` silenciosamente, então
confira que está lendo o jogo certo. Conferência rápida de escala (2026-08-01):
onepiece 188 holdings / pokemon 63 / riftbound 161.

---

## Fase 1 — Abrir os pedidos no admin da Liga

Sessão do dono já está logada no Chrome (claude-in-chrome) — não há automação
de login. Página de pedidos (mapeada em 2026-08-01):

```
https://www.ligamagic.com.br/?view=ecom/admin/pedidos
```

⚠️ `?view=ecom/admin/compras` (a referência antiga herdada de liga-conferir)
dá **404** — o nome certo é `pedidos`. O menu do admin: Pedidos →
Gerenciamento de Pedidos.

Mecânica da listagem (2026-08-01):

- A primeira navegação pode cair no desafio passivo do Cloudflare
  ("Performing security verification") — espere ~9s e siga; resolve sozinho.
  Isso NÃO é o ban 1007; só pare se aparecer erro explícito.
- A listagem não usa `<table>` — `get_page_text` funciona bem e traz tudo:
  nº (`#11560671`), cliente, data/hora, data pag., status, forma de
  pagamento/envio, valor total. Paginação no formato `1-9 de 9`.
- Bloco "Pedidos por Status" no topo resume o que precisa de ação.
- Status possíveis: `Aguardando Pagamento`, `Cancelado`, `Enviado`,
  `Pagamento efetuado - Aguardando envio`, `Retirado no Balcão`.
  Contam como venda: **Enviado**, **Pagamento efetuado - Aguardando envio**
  e **Retirado no Balcão** (pago). Cancelado tem às vezes Data Pag.
  preenchida — cancelou depois de pago; status atual manda, não conta.
- URL de detalhe do pedido:
  `?view=ecom/admin/compra&id=866280&cod=<pedido>`

Compare os números com o conjunto da Fase 0 — só pedidos novos e com pagamento
confirmado seguem para a Fase 2. Se não houver pedido novo, reporte e encerre.

Referência de taxas (não entram no sellPrice): 4,99% + R$ 0,40 com repasse em
14 dias, 3,99% + R$ 0,40 em 30 dias.

---

## Fase 2 — Extrair os itens de cada pedido novo

Abra o detalhe de cada pedido novo (respeitando o ritmo da regra 6) e extraia
por item: **nome, número da carta, edição, condição/idioma, qty, preço
unitário** — e o jogo:

| tcg | game (API) |
| --- | --- |
| 11  | onepiece   |
| 2   | pokemon    |
| 19  | riftbound  |

Mecânica do detalhe (2026-08-01):

- ⚠️ `get_page_text` no detalhe pega SÓ o bloco de comentários (o extractor
  prioriza o `<article>`). Use `javascript_tool` com `document.body.innerText`
  fatiado: os itens ficam entre `Itens do Pedido` e
  `Alteração de Informações de Pagamento`; a data/valor pagos ficam no bloco
  `Confirmação de Pagamento` (campo `Data do Pagamento`).
- Formato de cada item no texto:
  `2x Monkey.D.Luffy (119) (#OP15-119) (Código: OP-15OP15-119)` seguido de
  `R$ 155,00 (unid.)`, `R$ 310,00 (subtotal)`, edição por extenso
  (`Adventure on Kami's Island` / `Fogo Branco`), idioma (`EN`/`PT`) e
  condição (`NM`). No fim: `Valor dos Itens`, `Frete`, `Valor Total` —
  use `Valor dos Itens`/subtotais; frete fica fora (regra 2).
- Pokémon usa número tipo `#098/086`; One Piece usa `#OP15-119`.

⚠️ A saída do `javascript_tool` é bloqueada quando ecoa URL com query string —
devolva só dados (números, nomes, valores), nunca hrefs. Ao fatiar
`innerText`, aplique `.replace(/https?:\S+/g,'[url]')` por garantia.

---

## Fase 3 — Casar item ↔ registro do ledger

Para cada item do pedido, ache o registro no `/tmp/trades-<game>.json` da
Fase 0 casando por **NOME + número** (e condição/idioma quando o ledger
distinguir variantes). Regra 3 vale integralmente: se dois registros
sobreviverem ao filtro, ou se nenhum casar, **pare e mostre ao dono** o item do
pedido e os candidatos — não registre nada no chute.

Sanidade antes de baixar (lição do card de R$ 330 vendido a R$ 3): compare o
preço unitário do pedido com o `askBRL`/`ligaPriceBRL` do registro. Divergência
grande não bloqueia a baixa (a venda já aconteceu), mas entra no relatório
final como alerta de precificação.

---

## Fase 4 — Dar baixa

Uma chamada por item vendido:

```bash
B=https://collecta-deals-production.up.railway.app
curl -s -X POST "$B/api/trades/<id>/sell?game=<game>" \
  -H 'Content-Type: application/json' \
  -d '{"qty":2,"sellPrice":260.0,"sellCurrency":"BRL","sellDate":"2026-08-01","buyer":"Fulano — Liga #11542408"}'
```

- `qty` = unidades vendidas no pedido.
- `sellPrice` = **TOTAL bruto** (unitário × qty) — regra 2.
- `sellDate` = data do pedido (YYYY-MM-DD), não a de hoje.
- `buyer` = `"<Nome do comprador> — Liga #<pedido>"` — o `Liga #N` é a chave
  de dedup da regra 1; sem ele a próxima execução baixa de novo.

Comportamento do servidor: venda parcial faz split (nasce um registro novo
`status:"sold"` com id novo; o original perde `qty`); venda total marca o
próprio registro como `sold`. `qty` maior que o estoque não dá erro — marca
tudo como vendido. Resposta 200 traz o registro vendido; 404 = id errado.

---

## Fase 5 — Sincronizar a flag Liga

A Liga já decrementou o estoque dela na venda, mas o `ligaQty` do holding
restante NÃO acompanha (o split copia, não decrementa) — a linha ficaria
"Reimportar" âmbar no Estoque para sempre. Para cada holding que sobrou após
baixa parcial:

```bash
curl -s -X POST "$B/api/trades/liga?game=<game>" \
  -H 'Content-Type: application/json' \
  -d '{"items":[{"id":"<id>","ligaListed":true,"ligaQty":<ligaQty - vendidos>,"ligaPriceBRL":<inalterado>}]}'
```

- `ligaQty` novo = `ligaQty` anterior − unidades vendidas; `ligaPriceBRL` não
  muda (a venda não altera o preço anunciado).
- Se o item zerou na Liga, mande `"ligaListed":false` (o servidor limpa qty,
  preço e data sozinho).
- Venda total no dashboard: o registro virou `sold` e sumiu do Estoque —
  nada a fazer aqui.
- ⚠️ Só envie os itens afetados pela baixa. Reenviar item em dia apaga em
  silêncio um aviso de "desatualizado" legítimo (`liga-conferir`).

---

## Fase 6 — Relatório final

Tabela com uma linha por item baixado:

| Pedido | Comprador | Item | Qty | Total R$ | Game | Ledger id | Baixa | Flag Liga |

E abaixo dela:

- Pedidos pulados e por quê (já baixado, não pago, cancelado).
- Alertas de preço da Fase 3 (unitário do pedido muito abaixo/acima do ask).
- Confirmação no dashboard: `qty` caiu no Estoque (`$B/?tab=estoque`) e a
  venda aparece no Realized do Portfólio.

Precisou recadastrar algo que a flag dizia estar na Liga? Isso é auditoria —
skill `liga-conferir`. Repor o item vendido na Liga é `liga-singles-<jogo>`.

Aprendeu algo novo (estrutura da página de pedidos, um status inesperado, um
caso de casamento difícil)? Acrescente aqui — este arquivo é o repositório
desse conhecimento.

---

## Aprendizados de execuções

2026-08-04 (2ª execução — 10 pedidos novos, 9 já baixados manualmente):

- As baixas manuais do dono continuam a regra, não a exceção: buyer vazio ou
  `"Liga"`, sellPrice ora bruto ora líquido (~96%). A checagem de sold por
  nome+número+sellDate da Fase 0 pegou todas.
- **Selados vendem na Liga como produto próprio, mas no ledger podem ser outra
  coisa**: "Kit com 12 Blister Quádruplo ME4" (1× R$ 399) foi baixado
  manualmente como 12 unidades do Blister Quádruplo avulso. Ao casar selado,
  procure também o componente avulso antes de concluir que não existe.
- "Combo de Pacotes - Megaevolução 5" vendido na Liga não existia no ledger
  pokemon sob nenhum nome (nem "combo") — item vendido sem cadastro; reportar
  ao dono, não inventar registro.
- Pedidos de selado usam nome por extenso sem número de carta; o campo número
  do ledger pode ter um id interno (ex.: Caixa ME5 = `136634`) — casar por
  nome mesmo.
- Item One Piece com tag `Foil` na Liga casou com registro de variant vazia no
  ledger (Vinsmoke Reiju OP12-063, preço bateu exato) — a tag Foil da Liga não
  implica variant "Foil" no ledger; desempate é pelo nome exato + preço.
