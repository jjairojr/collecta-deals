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
T=$(sed -n 's/^ADMIN_TOKEN=//p' /Users/jjairo/Documents/www/opdeals/scripts/deploy.env)
for g in onepiece pokemon riftbound; do
  curl -s -H "X-Admin-Token: $T" "$B/api/trades?game=$g" > /tmp/trades-$g.json
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
https://www.ligacollectatcg.com.br/?view=ecom/admin/pedidos
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
T=$(sed -n 's/^ADMIN_TOKEN=//p' /Users/jjairo/Documents/www/opdeals/scripts/deploy.env)
curl -s -X POST -H "X-Admin-Token: $T" "$B/api/trades/<id>/sell?game=<game>" \
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
curl -s -X POST -H "X-Admin-Token: $T" "$B/api/trades/liga?game=<game>" \
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

2026-08-14 (3ª execução — 90 pedidos na Liga, 8 itens baixados):

- **Atalho de triagem que funcionou**: a maior parte dos pedidos antigos já foi
  baixada manualmente (buyer vazio). Em vez de abrir os 40+ pedidos pagos da
  listagem, pegue a maior `sellDate` entre os `sold` de todos os jogos — só
  pedidos com data posterior a ela são candidatos garantidos. Depois confirme
  a fronteira abrindo 1-2 pedidos do último dia já baixado (aqui 12/08:
  Perona 4× e Viola/Baby 5/Boa Hancock bateram com os `sold` de 2026-08-12).
- Somar valor dos pedidos por dia e comparar com a soma dos `sold` do dia NÃO
  valida nada: o dono também vende no balcão e em outros canais, então os
  totais divergem nos dois sentidos.
- **Desconto vs taxa**: o detalhe do pedido pode ter linha `Desconto:` (5% do
  valor dos itens). Esse desconto entra no `sellPrice` (o comprador pagou
  menos); a taxa da Liga continua fora (regra 2). Nas baixas manuais do dono
  aparecem tanto ×0,95 (desconto real) quanto ×0,96 (pedido sem desconto — ele
  descontou taxa por conta própria).
- Baixa manual pode agregar mais unidades do que o pedido tem (Articuno 161:
  `sold` de 5 un em 12/08 para um pedido de 3 un em 13/08, mesmo unitário
  ×0,95, e zero holding restante). Bateu nome+número+unitário e não sobrou
  estoque → já baixado, não refaça.
- Liga vendeu itens que não existem no ledger (Zekrom do N `#031/∞` MEP,
  Eevee ex `#174/∞` SVP) e um cujo único registro já estava `sold`
  (Bulbasaur `#133/132` MEG). Reportar ao dono, não inventar registro.
- `Booster Avulso - <set>` na Liga (Riftbound Vendetta, 20 un a R$ 20,90) não
  tem contrapartida no ledger, que só guarda `Caixa de Booster - Vendetta`.
  Decidir com o dono como abater caixa aberta antes de baixar.
- `ligaPriceBRL` do ledger fica defasado: os selados ME5 saíram a 308,49 /
  34,87 / 26,58 (o `askBRL`) enquanto o ledger guardava 405 / 49,90 / 39,90.
  Não bloqueia a baixa, mas vale alerta no relatório.

2026-08-20 (4ª execução — 128 pedidos na Liga, 6 itens baixados):

- Extrair o detalhe sem navegar: `javascript_tool` na própria página de pedidos
  com `fetch('/?view=ecom/admin/compra&id=866280&cod=<pedido>')` +
  `DOMParser` + `innerText` é muito mais rápido e leve que navegar. Mantenha o
  ritmo mesmo assim (8s entre fetches). Parseie linha a linha (`\d+x nome`,
  `R$ x (unid.)`, `R$ x (subtotal)`); regex de bloco único engole o item
  seguinte e some com metade do pedido.
- A saída do `javascript_tool` é bloqueada ("Cookie/query string data") também
  quando o texto contém `Cod: 32.240.245-2` ou qualquer `?`/`&`/`=`. Filtre
  esses campos antes de devolver.
- Triagem por fronteira funcionou de novo: maior `sellDate` com `Liga #` marca
  o corte; os pedidos posteriores já baixados manualmente aparecem como `sold`
  de buyer vazio com sellPrice = bruto × 0,96.
- Selado idêntico em vários pedidos no mesmo dia é ambíguo por natureza: 5
  pedidos de Caixa ME5 a R$ 308,49 pagos em 18/08 contra 4 baixas manuais de
  R$ 296,15 — dá para saber que falta uma, não a qual pedido pertence. Pergunte
  ao dono antes de escolher o número do pedido.
- Quantidade do pedido maior que o estoque do ledger (Jayce 088/166: 3 un
  vendidas, 1 un cadastrada) não é caso de baixar assim mesmo — pare e
  pergunte, é sinal de cadastro faltando.
- Duas cartas iguais vendidas em pedidos diferentes esgotam o ledger: Dawn
  118/094 tinha 1 registro, baixado no pedido 11709019 (17/08); o pedido
  11716961 (18/08) vendeu outra sem lastro no estoque.

2026-09-08 (5ª execução — 253 pedidos na Liga, 24 novos, 83 itens baixados):

- Fronteira por `Liga #` funcionou de novo: maior `sellDate` com `Liga #` era
  03/09 (#11865431); nenhuma baixa manual sem número desde 01/09 (só um
  "shopee"), então tudo depois da fronteira era candidato garantido.
- `javascript_tool`: a resposta é cortada em ~1000 caracteres e o CDP dá
  timeout em 45s. Guarde os pedidos em `window.__lv` (fetch + DOMParser, 8s
  entre pedidos, **no máximo 4 pedidos por chamada**) e leia depois em fatias
  compactas por pedido (`qty;nome #num;unit;set;idioma;cond;Foil`). Se a
  chamada estourar o timeout, a promise continua rodando na página — confira
  `Object.keys(window.__lv)` em vez de refazer o fetch.
- O bloqueio "Cookie/query string data" dispara com qualquer `=`, `?` ou `&`
  no texto devolvido, inclusive num rótulo tipo `desc=null`. Passe a saída
  final por `.replace(/[?&=]/g,'~')`.
- Ledger Pokémon tem duplicatas JP do mesmo número com variant `AR (JP)`
  (ask vazio, import em massa) e `Foil JP` (precificado, `ligaListed`). Para
  venda JP, o `Foil JP` com `ligaPriceBRL` é o anunciado; `AR (JP)` sem ask e
  a Liga dizendo EN/PT (Ethan's Adventure 221/182, Dedenne 093/088, Forest of
  Vitality 109/088) é ambíguo — perguntar, não chutar.
- Número da Liga vem na numeração do set anunciado (JP `073JP/064`, `115/108`)
  enquanto o ledger guarda a numeração EN equivalente (Cufant #076 SFA,
  Houndour #204 OBF, Spewpa #089 POR). Casar por nome + preço + `ligaListed`
  quando o número diverge assim.
- Pedido com itens de 2 jogos (#11903114: Pokémon + Riftbound) — o `tcg` é
  por item, não por pedido.
- Vendidos sem lastro no ledger desta vez: Caixa de Booster Attack of the Vine
  (rft, R$ 1.190), Pikachu ex 179/131 PRE, Decidueye ex 100/088 e Yveltal ex
  105/088 POR, Judge 228/091 PAF, Manectric 086/081 JP, Gangplank Naval #86 VEN,
  e itens já esgotados (Sprite Fountain #78, Decree of Focus #40).

2026-09-09 (6ª execução — 263 pedidos na Liga, 10 novos pagos, 10 itens baixados):

- Fronteira por `Liga #` de novo (#11905148, 08/09). Dois cancelados no meio
  (#11908179, #11905614) ignorados pelo status.
- **Caixa ME5 vendida na Liga já estava baixada à mão**: 4 caixas em 3 pedidos
  de 08/09 (#11912013, #11909086, #11907625 ×2) contra um `sold` manual de
  5 un em 08/09 a R$ 1.521,60 (= 5 × 317 × 0,96), buyer vazio. Não rebaixar;
  reportar. O holding da caixa ficou com qty 1 e ligaQty 27 — flag defasada.
- Numeração JP da Liga ≠ ledger, de novo: Spheal `#111/106` (Super Electric
  Breaker) = ledger `#199 SSP Foil JP`; Lunatone `#GG11/GG70` = `#184 s12a`;
  Bibarel `#200/172` = `#200 s12a`. Confirmar pelo `ligaPriceBRL`/ask e pela
  oferta COLLECTA ao vivo (`cmd/floors`) quando a captura for anterior à venda.
- Grimsley's Move `#120/094` vendida como EN a 12,80: o único holding é o
  `AR (JP)` sem ask, mas a captura ao vivo mostrava a oferta COLLECTA nele em
  EN a 12,80 — é ele. Baixado.
- Brock's Scouting `#179/159` PT a 22,99: dois holdings (`AR (JP)` ask 22,89 e
  `Foil` ask 23,90 `ligaListed`), um só anúncio na Liga (PT, 22,89 → 22,99 na
  reprecificação de 05/09). Ambíguo → perguntado ao dono, não baixado.
- Judge `#228/091` PAF EN sem lastro (só existe Judge #222 DRI) — mesmo caso
  da 5ª execução; reportado.

2026-09-12 (7ª execução — 272 pedidos na Liga, 7 novos pagos, 15 itens baixados):

- Fronteira por `Liga #` (#11915822, 09/09). Dois cancelados no meio ignorados.
- Kit vendido na Liga (`Kit com 12 Blister Triplo - Megaevolução 4/5`, R$ 315,90)
  não existe no ledger: baixado como 12 blisters avulsos por kit no holding do
  blister (precedente da 2ª execução), `sellPrice` = valor do kit × qtd de kits.
- Acessório vendido num pedido de cartas (`Shield Meeple BR - Padrão Premium`)
  casa no ledger de acessórios (`Sleeves Meeple BR - PADRÃO`), que aparece na
  lista de qualquer `game=`; o `/sell` resolve pelo id com `game=pokemon`.
  O valor dele não vem como item parseado (sem `Código:`) — é a diferença entre
  `Valor dos Itens` e a soma das cartas.
- Card JP anunciado sob a edição EN (Liga: `Dhelmise (#091/084) PBL JP`)
  enquanto a captura ao vivo ainda mostra a oferta COLLECTA no `M5 088`: são
  dois anúncios distintos; o vendido é o registro legado `Dhelmise (091/084)`
  (set PBL), não o `M5 AR (JP)` que continua à venda.
- Pedido com qty maior que o ledger (2× Luffy ST21-014, 1 holding): baixada a
  unidade existente, a outra reportada sem lastro.
- Sem lastro desta vez: Joltik SCR 150 JP, Inkay M6 085 JP, Galarian Zapdos
  SWSH283 SSPR JP, 1× Luffy ST21-014.

2026-09-22 (8ª execução — 315 pedidos na Liga, 31 novos pagos, 72 itens baixados):

- A fronteira `Liga #` (#11932665, 11/09) apareceu como **Cancelado** na listagem:
  o pedido foi cancelado depois da baixa. O item (Galarian Zapdos SWSH283 JP)
  voltou ao estoque da Liga e vendeu de novo no #11966335 (15/09) a R$ 15,00.
  O `sold` continua preso ao pedido cancelado (R$ 14,79) e não há holding para
  rebaixar — reportar ao dono para ele ajustar o registro à mão.
- Desconto de 5% apareceu em 8 dos 31 pedidos (`Desconto:` no detalhe). Aplicado
  como fator 0,95 sobre unitário × qty; a soma dos itens líquidos bate com o
  `Valor Total` − frete da listagem.
- `Pagamento efetuado - Aguardando envio` sem `Data Pag.` na listagem
  (#12026327, retirada no balcão) e `Retirado no Balcão`/`Enviado` sem
  `Data Pag.` (#11975012, #11950281, #11966504) foram baixados pelo status.
- Selado: "Coleção Treinador Avançado - Celebração de 30 Anos (ING)" e
  "(JAP) Caixa de Booster - Mega Dream ex" vêm com `tcg=90` (Produtos), sem
  número. A caixa casou com `Box Mega Dream EX` (ledger sem ask, sem flag Liga
  — não mandar `ligaListed:false` para holding que nunca teve flag). A ETB de
  30 anos não existe no ledger.
- Kits de 12 blisters ME4 (Quádruplo R$ 415,90 / Triplo R$ 315,90) baixados
  como 12 un do blister avulso cada, mesmo precedente da 7ª execução.
- Liga numera JP pelo set PT/EN: Grafaiai `080/071` = ledger PAL #223 Foil JP;
  Togedemaru `090/080` = PFL #104; Ethan's Magcargo `222/217` = M2a #197;
  Dreepy `247/217` = M2a #211. Casar por nome + `ligaPriceBRL` exato.
- Armarouge `086/084` JP a 19,99 ×2 pedidos: o registro PBL #086 (19,99) já
  estava `sold` (05/09) e o único holding é M5 #083 AR (JP) ask 23,79 com
  `ligaQty` 3 — os 3 anúncios da Liga = 1 PBL + 2 M5; baixada 1 un no M5, a
  outra ficou sem lastro.
- `askBRL` do ledger defasado frente ao anúncio: Banette TG07/TG30 vendida a
  34,90 (ledger 19,97; o `pkm-reprecificado-2026-09-12.csv` já mostrava 34,90),
  Exeggutor MEG 135 a 15,49 (ledger 24,50). O CSV de reprecificação é a
  segunda fonte para confirmar o preço anunciado.
- Sem lastro desta vez: ETB 30 anos EN, Brassius 135/131 EN, Mega Gengar ex
  056/094 EN, Stoutland 248/236 JP, Ditto GG22 e Paras GG32 EN, Heracross
  194/193 EN (só há Foil JP a 64,90), N's Reshiram Stamped 167b (dois holdings
  JTG sem ask, nenhum Stamped — ambíguo), Hisuian Voltorb GG01 JP, Banette
  197/184 JP, Exeggutor 066/063 JP (M1L já vendido 09/09), Goldeen 087/084 JP
  (dois `sold` antigos), Meowth ex 062/088 EN (POR vendido 10/08), Rosa's
  Encouragement 123/088 PT, Crumbling Sands #39 / Decree of Focus #40 /
  Decree of Unity #131 (VEN esgotados), 1× Dreepy 247/217 (2 pedidos, 1
  holding), 1× Solosis 118/086 PT (2 vendidos, holding listado qty 1 + um
  `AR (JP)` sem ask), 1× Armarouge.

2026-10-12 (9ª execução — 370 pedidos na Liga, 13 novos pagos, 40 itens baixados):

- **A Fase 0 pode vir vazia de propósito.** Em 07/10/2026 12:30 o dono zerou os três
  ledgers de prod e recriou tudo a partir do export da Liga (`tasks/liga-sync-2026-10-07/`,
  export das 13:11, nota "Cadastrado a partir do estoque da LigaMagic"). O histórico de
  `sold` com `Liga #` ficou só no backup `data/.sync-bak/prewipe-20261007-123024/`
  (último pedido baixado antes do wipe: #12026327, 21/09). Quando não houver nenhum
  `sold`, confira `createdAt` dos holdings: se todos nasceram no mesmo dia, a fronteira é
  o horário do export que gerou o ledger — a Liga já tinha descontado as vendas
  anteriores do estoque exportado, então só pedidos criados DEPOIS do export precisam
  de baixa. Não use o backup pré-wipe para dedup: ele descreve outro ledger.
- Desconto novo: cupom `Desconto (#CONTINUE5)` de 10% (Carlos Pratti, 2 pedidos); o
  `Desconto:` sem cupom segue 5%. Fator aplicado por item sobre unitário × qty.
- O cliente pode fechar um segundo pedido como "Adicionar a um pedido já existente"
  (#12203534 sobre #12203353): são dois números na listagem, cada um com seus itens;
  baixar os dois.
- Selado One Piece japonês (`(JAP) Kit Colecionável - Base Shop Limited Card Collection
  Vol.1`, categoria Produtos) está no ledger **pokemon** como `BASE Shop` kind sealed,
  qty 9 sem ask — casado por nome, baixado com `game=pokemon`.
- Sem lastro: Toxtricity `#134/128` PT (Celebração de 30 Anos, não existe no export nem
  no ledger) e Singularity `#105` OGN (export de 07/10 com qty 0, holding ausente).
- Preços do pedido abaixo do `askBRL` em 8 cartas do pedido grande (Dragonite-V 130 vs
  149,99; Munkidori 210 vs 263,09; Cofagrigus 160 vs 205,69; Eelektross 135 vs 162,10;
  Landorus 165 vs 179,99; Deoxys-VSTAR 470 vs 499,89; Salamence ex 290 vs 319,79;
  Bewear 140 vs 156) e Ezreal - Prodigy #149 SFD 95 vs 113,99: o anúncio da Liga já
  estava reprecificado para baixo (rodada de 10/10) e o ledger não acompanhou.

2026-10-13 (10ª execução — 376 pedidos na Liga, 2 novos pagos, 111 itens baixados):

- Fronteira `Liga #` = #12204163 (12/10). Quatro cancelados depois dela ignorados
  (#12214893, #12211420, #12211319, #12209168).
- Pedido de "playset de comuns" do Riftbound (#12208690: 111 cartas VEN × 3 un, R$ 0,09
  a R$ 3,99, total R$ 84,61 + frete 30,75): o `innerText` do detalhe tem 140 k
  caracteres — parsear em `window.__it` e ler em 4 fatias de 28 itens no formato
  `q;nome;nº;unit;F`. Casou 111/111 por nome + nº + Foil (variant do ledger) contra
  `ligaPriceBRL`; 74 holdings parciais tiveram `ligaQty` decrementado.
- `(CHN) Caixa de Booster - GEM Pack Vol.5` (#12208277, R$ 289,90) não existe no ledger
  pokemon: só há Vol.2, Vol.3 (sem ask), Vol.4 e Vol. 6 (ask 319,90, qty 19). Não baixado
  — perguntado ao dono qual registro representa a Vol.5.

2026-10-18 (11ª execução — 389 pedidos na Liga, 9 novos pagos, 41 itens baixados):

- Fronteira `Liga #` = #12208690 (12/10). #12208277 (GEM Pack Vol.5) segue pendente de
  resposta do dono desde a 10ª execução; não baixado. Cancelados ignorados (#12247038,
  #12230193, #12220983); #12256803 aguardando pagamento.
- `Enviado` via PIX sem `Data Pag.` (#12219480) baixado pelo status, como antes.
- Descontos: 5% (#12237998, #12217566) e 80/806,90 ≈ 10% (#12219480, cupom). Fator
  aplicado por item.
- Riftbound: a Liga marca `Foil` por item no detalhe (linha solta depois da condição);
  o número sem sufixo (`#178`, `#19`) casa com o registro sem `A`, o `#R05A`/`#143A` com
  o AA. O preço do pedido = `ligaPriceBRL`/ask desempata.
- Sem lastro: pedido #12241188 inteiro (Pikachu 023, Eevee 118, Snorlax 119 da
  Celebração de 30 Anos PT), Chaos Rune #166A OGN ×3, Garen - Commander #13b OGS,
  Astral Heron #44 VEN (R$ 249,90), Cataclysmic Duel #90 ×2 (único registro já vendido
  no #12208690), Boa Hancock OP07-051, 1× Dracule Mihawk OP01-070 (pedido 3, ledger 2),
  1× Swain #65 (pedido 2, ledger 1).

2026-10-20 (12ª execução — 393 pedidos na Liga, 4 novos pagos, 7 itens baixados):

- Fronteira `Liga #` = #12261686 (18/10). #12256803 (Kit Base Shop Vol.1), que estava
  "Aguardando Pagamento" na 11ª, virou Enviado/pago — baixado agora; revisitar pedidos
  pendentes de pagamento a cada execução.
- **Canal novo: MyPCards (loja CollectaTCG, desde 19/10).** O mesmo estoque está anunciado
  lá, então todo item baixado aqui precisa sair (ou ter a qty reduzida) no MyP: achar o
  anúncio em `/CollectaTCG/<jogo>?PastaSearch[query]=<número>` (link
  `/estoque/update/<id>`) e excluir com `$.ajax({url:'/estoque/delete?id=<id>',type:'POST'})`
  disparado sem await (vários de uma vez num `forEach`); a página navega logo depois e
  mata o `javascript_tool`, então confira depois com `fetch('/estoque/update/<id>',
  {redirect:'manual'})` → 404 = removido. `fetch` POST manual dá 403 (21/10). Reduzir qtd:
  `new FormData(form)` da página de update, trocar `Estoque[quantidadeestoque]`, POST no
  action → flash "N x R$ X gravado com sucesso". Mapa ga→id de anúncio: varrer
  `/CollectaTCG/<jogo>?page=N` (link `/estoque/update/`, `data-ga-item-id`). Vendas do MyP aparecem no ledger
  com buyer "MYP Cards" (baixa manual do dono) e devem sair da Liga.
- Kit Base Shop Vol.1 agora tem registro de nome exato no ledger **onepiece** (ask
  398,90); o `BASE Shop` do pokemon (q8, sem ask) é provável duplicata do mesmo estoque.
- Sem lastro: Rowlet `#090/088` JP Foil (Equilíbrio Perfeito) R$ 19,65 (#12268010).

2026-10-21 (13ª execução — 4 pedidos novos pagos, 21 itens, 13 baixados):

- Vendas do site (`— Site #N`, checkout Mercado Pago) também precisam sair da Liga e do
  MyP. Neste caso a Liga já estava zerada/decrementada para os 5 itens do Site #7 —
  conferir o admin antes de mexer; só o `ligaQty` do dashboard precisou de ajuste.
- Pedido Pokémon grande de JP Foil: 8 de 12 itens sem lastro no ledger (Beautifly 219/217,
  Togekiss 203/193, Chespin 087/086, Frogadier 089/086, Aurorus 084/080, Xerneas 089/083,
  Dhelmise 091/084 JP; Fomantis 085/084 PT só casaria com M5 #082 AR (JP) de outro preço).
- Script de baixa em shell quebra com apóstrofo no nome ("Lisia's Appeal"): gerar as
  chamadas em Python (urllib) em vez de `curl -d '<json>'`. Toda chamada ao prod leva o header `X-Admin-Token` (valor de `ADMIN_TOKEN` em `scripts/deploy.env`).

2026-10-22 (14ª execução — 1 pedido novo pago, 16 itens, 3 baixados):

- #12289145: 13 de 16 cartas JP Foil sem lastro de novo (o dono cadastra JP direto na Liga
  sem passar pelo dashboard). Banette TG07 LORTG Foil JP casou por nome+nº+idioma com
  ask 23,99 vs vendido 16,99 (único candidato).
- Na mesma rodada, sync Liga → dashboard do OP: varrer `cartas/all&tcg=11` (busca com
  `btBuscar`), comparar soma de qty por `h_numero_carta` com os holdings, e criar o que
  falta via `POST /api/trades` (buyBRL 0, nota "Cadastrado a partir do estoque da
  LigaMagic <data>") + `/api/trades/listings` + `/api/trades/liga`. Qty maior na Liga vira
  lote extra (registro novo com a diferença), não PUT no registro antigo.
- Sync Liga → dashboard do **Pokémon** (22/10): o dono anuncia cada JP AR DUAS vezes na Liga
  — na edição JP (M5, M2A, M3, M4, M2, M1L/M1S) e na EN equivalente (PBL, ASC, POR, CRI, PFL,
  MEG; às vezes MEP/CRZ/CEC/OBF/CZGG com preço igual). Contar uma vez só: canônico = edição JP,
  qty = max das duas linhas; parear só por nome EN + par de edição conhecido ou preço igual
  (parear só por nome juntou Spiritomb LORTG com M1L). Mapa edid→sigla: `select[name=txt_edicao]`
  do próprio admin (texto termina em "(SIGLA)"). Saída grande do admin: clicar na página e
  `navigator.clipboard.writeText` → `pbpaste` (48 KB passou).

2026-10-24 (15ª execução, via `baixa-vendas` — 7 pedidos pagos, 19 linhas, 18 baixas):

- **A listagem mudou de layout**: `innerText` agora vem em colunas, uma por
  linha — `#N | cliente | data | hora | data pag. (opcional) | status | pagamento | envio | R$ total`.
  Separe por `\|(?=#\d{8}\|)` depois de trocar as quebras por `|`; a data pag.
  some quando o pedido não foi pago (cancelado).
- Detalhe do item continua no formato antigo; parser linha a linha
  (`^(\d+)x nome`, depois unitário, edição, idioma, condição, `Cod:`, `Foil`).
- Tyrunt `#070/∞` MEP JP = ledger M3 089 Foil JP; Flygon `#101/094` PFL JP = M2
  088 Foil JP — os dois por nome + `ligaPriceBRL` 14,90 exato.
- `(JAP) Caixa de Booster - Storm Emeralda` = ledger pokemon `Storm Emeralda
  (With Shrink)` (sem ask). Kit Base Shop Vol.1 de novo no ledger **onepiece**.

2026-10-24 (16ª execução — 2 pedidos novos pagos, 6 linhas One Piece, 6 baixadas):

- Fronteira `Liga #` = #12310823. Novos: #12314875 e #12314771, ambos sem desconto.
- O mesmo número pode ter dois holdings (lote original + "lote extra" criado no sync
  Liga → dashboard). Não é ambiguidade: é a mesma carta. Baixar do lote mais antigo
  (`createdAt`) e decrementar o `ligaQty` só dele; a soma dos `ligaQty` dos lotes é o
  estoque da Liga.
- MyP: o anúncio tem uma linha só por carta com a qty somada dos lotes. Reduzir para
  qty atual − vendidos via FormData da página de update (fetch POST com o FormData do
  próprio form funcionou, flash "N x R$ X gravado com sucesso").

2026-10-27 (auditoria Liga × aba Vendas — 42 pedidos pagos desde 13/10):

- **Admin da loja agora em `www.ligacollectatcg.com.br`** (mesmos paths `?view=ecom/admin/...`).
  Em `www.ligamagic.com.br` o admin responde "Você não está logado" mesmo com a sessão ativa.
- Item vendido "sem lastro" nunca entrava na aba Vendas, então o faturamento ficava menor.
  Correção: `POST /api/trades` com `status:"sold"`, `buyBRL 0`, `sellPrice`/`sellDate`/`buyer`
  do pedido e nota "Venda da Liga sem registro no estoque do dashboard". Não mexe em holding:
  a Liga já tinha descontado essas unidades. Lançados 38 registros (10 pedidos).
- Auditoria completa: detalhe de todos os pedidos pagos em `window.__det` (8 s entre fetches),
  `JSON.stringify` → clipboard → `pbpaste`, parser linha a linha e casamento por nome contra
  os `sold` com `Liga #N`. A soma dos itens bateu com "Valor dos Itens" nos 42.
- Gravar várias linhas da busca `cartas/all&txt_filtro=1` num submit só funciona; ir da última
  página para a primeira, porque a linha zerada some e desloca as páginas seguintes. Conferir
  qty e preço atuais de cada linha antes de escrever.

2026-10-29 (17ª execução — 4 pedidos novos pagos, 50 linhas, 57 baixas):

- **A listagem mudou de layout de novo**: cada pedido é um bloco rotulado, uma
  linha por campo — `#N / Cliente`, `Data`, `<data>`, `<hora>`, `Data Pag.`,
  `<data>`, `Status`, `<status>`, `Pagamento / Envio`, …, `Valor Total`,
  `R$ x`. Ache `^#(\d{8}) / (.*)$` e leia o valor seguinte a cada rótulo nas ~16
  linhas abaixo; `Data Pag.` some no cancelado sem pagamento.
- Pedido já baixado e depois cancelado: #12327804 (Parasect TG01, sellPrice
  0,001) virou `Cancelado`. Não há holding para devolver; reportar ao dono.
- Venda em vários canais esgota o ledger antes da Liga: Gloriosa OP17-046 teve
  4 na Liga sem lastro porque MyP e Site já tinham levado o resto e o anúncio
  da Liga não tinha sido reduzido. Baixar o que existe e reportar como possível
  venda sem estoque.
- Lotes múltiplos do mesmo número: alocar a venda do lote mais antigo
  (`createdAt`) para o mais novo, filtrando por número exato e ask = unitário
  do pedido (separa base de Parallel, como Franky OP09-072 vs OP09-072-P).
