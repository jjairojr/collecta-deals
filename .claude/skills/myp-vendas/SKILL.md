---
name: myp-vendas
description: Confere os pedidos da loja CollectaTCG no MyPCards (/CollectaTCG/vendas), dá baixa de cada venda no dashboard de prod via POST /api/trades/{id}/sell com dedup por nº de pedido (`MyP #N`) no buyer, e tira o item vendido da Liga. Use quando o pedido for verificar vendas do MyP, "vendeu no myp?", dar baixa do MyP, ou conciliar pedidos do MyP com o dashboard e a Liga.
allowed-tools: Bash, Read, Write, Edit, mcp__claude-in-chrome__*
---

# Conferir vendas do MyP e dar baixa no dashboard + Liga

Loja **CollectaTCG** no MyPCards (aberta em 19/10/2026). O mesmo estoque está
anunciado na Liga (866280) e no site, então toda venda do MyP precisa de:

1. baixa no dashboard de **prod** (`POST /api/trades/{id}/sell`);
2. redução/zeragem do anúncio na **Liga**;
3. ajuste do `ligaQty` do holding que sobrou.

O próprio MyP já decrementa o anúncio dele na venda — não mexa no MyP.

Argumento `--auto` (execução agendada, sem ninguém olhando): nunca pergunte;
tudo que exigiria pergunta vira linha de **pendência** no relatório e nada é
gravado para aquele item.

---

## 🚫 REGRAS DURAS

As regras 1–7 da skill `liga-vendas` valem aqui, com `MyP #<pedido>` no lugar
de `Liga #<pedido>`. Em resumo:

1. **Dedup por pedido**: buyer = `"<nick> — MyP #<pedido>"`. Pedido que já tem
   `MyP #<pedido>` em algum `sold` de qualquer jogo nunca é reprocessado.
2. `sellPrice` = **TOTAL bruto** do item (unitário × qty); frete fora; taxa do
   MyP fora.
3. Casamento por **ga-id + nome + preço**; dois candidatos de preço diferente ou
   nenhum candidato = pendência, nunca chute.
4. Nunca `PUT /api/trades/{id}`. Baixa é sempre `/sell`.
5. Prod: `https://collecta-deals-production.up.railway.app`.
6. Ritmo: 6–8 s entre fetches na Liga (Cloudflare 1007); MyP aguenta 1 fetch/s,
   mas é serial (concorrência devolve página vazia).
7. Conta só pedido pago.

---

## Fase 0 — Ledger de prod e dedup

```bash
B=https://collecta-deals-production.up.railway.app
T=$(sed -n 's/^ADMIN_TOKEN=//p' /Users/jjairo/Documents/www/opdeals/scripts/deploy.env)
for g in onepiece pokemon riftbound; do curl -s -H "X-Admin-Token: $T" "$B/api/trades?game=$g" > "$S/t-$g.json"; done
```

`$S` = scratchpad da sessão. Envelope `{"trades":[...]}`. Pedidos já baixados:
regex `MyP #(\d+)` no `buyer` dos `status:"sold"`.

⚠️ O dono também baixa à mão com buyer `"MYP Cards"` sem número (Ninetales 199
OBF em 19/10 = pedido #655970). Antes de baixar um item, procure `sold` do mesmo
nome+número com `sellDate` ≥ data do pedido e buyer contendo `MYP` (sem `#`):
bateu, trate como já baixado e reporte. Pedidos anteriores a 23/10/2026 que não
têm `MyP #` foram resolvidos à mão: #655970 (19/10, MohanCards) já baixado,
#655927 cancelado.

---

## Fase 1 — Listagem de vendas

Sessão do dono logada no Chrome. Em uma aba qualquer do `mypcards.com`:

```js
const d=new DOMParser().parseFromString(await (await fetch('/CollectaTCG/vendas')).text(),'text/html')
```

Três tabelas: **Aguard. envio**, **Enviado**, **Outros Status**. Colunas:
`Código · Data · Envio · Frete · Total · Comprador · Status · Próximo passo`.
Link do pedido: `table a[href^="/pedido/"]` (texto = nº). O nick do comprador é
o link `/<nick>` da mesma linha.

Contam como venda: `Aguard. envio` (status `Pago em` no detalhe), `Enviado`,
`Concluído`/`Entregue`. Não contam: `Cancelado pelo comprador`, `Cancelado`,
aguardando pagamento. Status novo e desconhecido = pendência.

A listagem mostra o mês corrente ("Resumo do mês"); confira se há paginação
(`.pagination`) e percorra tudo.

---

## Fase 2 — Detalhe do pedido

```js
const d=new DOMParser().parseFromString(await (await fetch('/pedido/<id>')).text(),'text/html')
const t=(d.querySelector('main')||d.body).innerText.replace(/\s*\n\s*/g,'\n')
```

⚠️ O detalhe traz **CPF e endereço do comprador** — nunca devolva esse trecho na
saída do `javascript_tool`. Corte a partir do último `Cor\n` (fim do cabeçalho
de ordenação da lista de itens) até `Subtotal`.

Formato de cada item (mapeado em 23/10/2026, pedido #660256):

```
Roronoa Zoro (EB04-007)                         <- nome PT
Roronoa Zoro (EB04-007)                         <- nome EN
Roronoa Zoro (EB04-007) - Adventure on Kami's Island   <- nome - edição
one_op15_eb04-007                               <- ga-id (jogo_set_número+sufixo)
NM - Quase nova                                 <- condição (pode vir "Promo · NM - ...")
2 x
R$ 90,90                                        <- unitário
Total:
R$ 181,80
```

Fechamento: `Subtotal (N itens): R$ x`, `Frete (<tipo>): R$ y`, `Total: R$ z`.
Confira que a soma dos totais dos itens = Subtotal. Data do pedido = cabeçalho
`Pedido #<id> - DD/MM/AAAA`.

Jogo pelo prefixo do ga-id: `one_` → onepiece, `pokemon_`/`pokemomn_` →
pokemon, `riftbound_` → riftbound, `mp_` → selado (casar por nome em qualquer
jogo). Mapa completo de ga-ids e sufixos (p1/p2/wp/dp, sets TG etc.) na memória
`opdeals-myp-store`.

Devolva só dados compactos: `.replace(/https?:\S+/g,'[url]').replace(/[?&=]/g,'~')`.

---

## Fase 3 — Casar item ↔ ledger

Número da carta = parte do ga-id depois do set, em maiúsculas (`eb04-007` →
`EB04-007`). Casar com `number` do ledger do jogo:

- Sufixo do ga-id `wp` → ledger `-WP`; `p1`/`p2` no MyP **não** é confiável
  (OP17-005 base veio como `op17-005p1`): o desempate é o **preço unitário ==
  `ligaPriceBRL`/`askBRL`** do holding.
- Só `status:"holding"` conta. Vários holdings do mesmo número com o MESMO preço
  (lotes duplicados, ex. S-Snake OP08-112 1 un + 4 un) = mesmo produto: baixe do
  lote mais antigo (`createdAt`), espalhando se precisar.
- Holdings do mesmo número com preços diferentes e nenhum igual ao unitário,
  ou qty do pedido maior que a soma dos holdings = pendência.

---

## Fase 4 — Baixa no dashboard

Gere as chamadas em Python (`urllib`), não em `curl -d '<json>'` (apóstrofo em
nome quebra o shell). Toda chamada ao prod leva o header `X-Admin-Token` (valor de `ADMIN_TOKEN` em `scripts/deploy.env`).

```python
body={"qty":q,"sellPrice":round(unit*q,2),"sellCurrency":"BRL","sellDate":"<AAAA-MM-DD do pedido>","buyer":f"{nick} — MyP #{pedido}","channel":"myp"}
POST f"{B}/api/trades/{id}/sell?game={game}"
```

Venda parcial faz split (nasce registro `sold` com id novo; o original fica com
o resto e com o `ligaQty` antigo).

---

## Fase 5 — Tirar da Liga

Para cada item baixado, ache a linha no admin da Liga e reduza a quantidade.

Admin em `www.ligamagic.com.br` (não no host do jogo). Busca por nome que
**grava** (precisa ser a URL do formulário, com `btBuscar=Buscar`):

```
https://www.ligamagic.com.br/?view=ecom%2Fadmin%2Fcartas%2Fall&tcg=<11|2|19>&txt_estoque=&page=1&search_type=1&txt_carta=<nome>&txt_edicao=&txt_raridade=&txt_cor=&txt_qualid=&txt_grading=&txt_grading_company=&txt_extras_opcao=0&txt_filtro=&txt_estoque_de=&txt_estoque_ate=&txt_idioma=&btBuscar=Buscar&txt_order=
```

- `txt_filtro=` (vazio) mostra também cadastros zerados; `txt_filtro=1` só com
  estoque. Lista corta em 50 linhas por página.
- Campos por linha `k`: `h_numero_carta[k]` (`EB04-007`, `OP12-062-WP`,
  variantes `-AA`, `-JP`, `-SP`), `h_id[k]` (id da oferta),
  `txt_qty_typed[k]` (estoque visível; vazio = 0), `txt_preco[k]` (`"90,90"`),
  `txt_idioma[k]` (2 EN, 6 JP, 8 PT).
- Linha certa = `h_numero_carta` exato + `txt_preco` == unitário vendido.

Decisão por item (`antes` = `ligaQty` do holding no ledger ANTES da baixa,
`liga` = `txt_qty_typed` atual, `v` = vendidos):

| situação | ação |
| --- | --- |
| `liga` ≤ `antes − v` | já tirado à mão — não mexa (foi o caso de todo o #660256) |
| `liga` == `antes` | gravar `liga − v` |
| outro valor | pendência (venda na Liga em paralelo ou flag defasada) |

Gravar: com a aba **navegada** para a URL da busca (fetch não grava), setar
**só** `txt_qty_typed[k]` (focus, value, eventos `input`/`change`, blur) e
`q.form.requestSubmit(q.form.querySelector('[name=btSalvar]'))`. ⚠️ Nunca
escreva no `txt_qty[k]` oculto: isso fez o estoque **somar** (30→60→90) na
`liga-precificar`. Espere ~5 s; o reload deve trazer "Cards de sua lista
alterados com sucesso." e a linha com a quantidade nova (zerada some da busca
com `txt_filtro=1`). Validado em 24/10/2026 (Gloriosa 9→8, Sanji AA 1→0). Antes de gravar, confira nos pedidos recentes da Liga
(`?view=ecom/admin/pedidos`) se o mesmo item não vendeu lá também — se vendeu,
é venda dupla: pendência urgente para o dono.

---

## Fase 6 — Flag Liga no dashboard

Para cada holding que sobrou após split:

```bash
curl -s -X POST -H "X-Admin-Token: $T" "$B/api/trades/liga?game=<game>" -H 'Content-Type: application/json' \
  -d '{"items":[{"id":"<id>","ligaListed":true,"ligaQty":<qtd na Liga deste lote>,"ligaPriceBRL":<inalterado>}]}'
```

Lote zerado na Liga → `"ligaListed":false`. Só envie os itens afetados.

---

## Fase 7 — Relatório

Tabela: `Pedido | Data | Comprador | Item | Qty | Total R$ | Game | Ledger id | Dashboard | Liga`.
Abaixo: pedidos pulados (já baixados, cancelados), pendências com o motivo, e
alertas de preço (unitário do pedido ≠ ask do ledger).

---

## Aprendizados de execuções

2026-10-23 (1ª execução — pedido #660256, zapiti, 8 itens OP, R$ 353,64):

- 6 registros casaram por número + preço exato; S-Snake OP08-112 tinha 2 lotes a
  2,99 (1 un de 22/10 + 4 un de 07/10) — baixadas 2 un do lote de 07/10.
- A Liga já estava zerada à mão para os 5 itens esgotados e o S-Snake estava em 3
  (5 − 2): nenhuma gravação na Liga foi necessária; só o `ligaQty` do lote que
  sobrou (4 → 2).
- Monkey.D.Luffy EB04-061 tinha `ligaQty` 2 com qty 1 (defasado desde o Site #9).
