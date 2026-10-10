---
name: importar-eua
description: Acha as cartas que valem trazer dos EUA para vender no Brasil, cruzando o piso e o giro reais da Liga (snapshots do tracking) com o preço do TCGplayer, filtrando por meta/rotação e terminando numa lista de compra por vendedor. Use quando o pedido for garimpar produto para importar, montar mala/pedido dos EUA, "o que vale a pena trazer", ou avaliar se uma carta específica compensa importar.
allowed-tools: Bash, Read, Write, Edit, WebSearch, WebFetch, mcp__claude-in-chrome__*
---

# Importar dos EUA para vender no Brasil

Responde "o que trazer, quanto trazer e de quem comprar" com dado medido. O
Brasil paga prêmio em carta **em inglês** porque não existe canal oficial — a
Copag imprime só PT, Lorcana/Riftbound não têm distribuição nenhuma. Esse prêmio
é estrutural, mas está longe de ser uniforme: em One Piece o fluxo é invertido.

Isto é o **contrário** do `internal/compare` do repositório. O motor de deals
acha carta barata no BR para vender nos EUA. Aqui o sentido é EUA → BR, então
não reaproveite o ranking de `Deal` — só o casamento de carta.

Precificar o que já está no estoque é `liga-precificar`; conferir lista de
vendedor é `liga-orcamento-*`.

---

## As quatro regras que mandam

**1. Piso do BR sem profundidade é ficção.** Uma carta com 2 lojas e 5 cópias
tem "piso" que é o preço pedido de um vendedor só, não mercado. Exija
**≥ 6 lojas EN** antes de acreditar em qualquer gap. Foi o que derrubou a
recomendação nº 1 de uma rodada inteira: Maleficent - Monstrous Dragon (LOR9
108) aparecia a 8× (US$ 2,79 contra piso R$ 120) com "54 vendas" — tinha 2 lojas
e 5 cópias, e o giro era ruído de relistagem. É reprint de um 9-custo fora do
meta; não vale nada.

**2. Giro vale mais que multiplicador.** Multiplicador alto com zero venda é
capital parado. Ordene por `lucro/unidade × unidades vendidas`, não por razão de
preço. E **nunca compre mais que ~1,5× o que o BR inteiro vendeu na janela
medida** — se o país absorveu 12 cópias em 5 semanas, trazer 40 é virar o
estoque parado do mercado.

**3. Meta e rotação vetam antes da conta.** Carta rotacionada é item de
colecionador, e colecionador não gira. Todo o topo de uma lista de Pokémon caiu
por isso (ver Fase 3).

**4. Custo é a variante MAIS CARA que casa.** Quando o mesmo número tem duas
versões nos EUA e você não sabe qual é a do BR, use `max(preços)`. Se o gap
sobrevive ao pior caso, é real. Se só sobrevive ao melhor caso, é erro de match.

---

## Fase 0 — Escolher o jogo (a direção do fluxo muda)

Medido em agosto/2026, mediana `piso BR / custo US` entre cartas com US$ ≥ 5:

| Jogo | Mediana | Veredito |
| --- | --- | --- |
| Lorcana | 1,51× | prêmio no mercado inteiro, menos importador brigando |
| Riftbound | 1,07× | paridade — só Epic foil e Alternate Art valem, carta a carta |
| One Piece | **0,37×** | **fluxo invertido**: compre aqui, venda lá |
| Pokémon | — | melhor volume e liquidez; ver ressalva de match abaixo |

Recalcule essa mediana antes de qualquer garimpo. Se der abaixo de 1, pare e diga
ao dono que a rota é a inversa.

Cobertura por jogo: One Piece e Lorcana têm `data/snapshot*.json` com listings +
prices já casados; Pokémon não tem snapshot de deals e precisa do join manual
(Fase 1). Sets japoneses do tracking (`M1L`, `M2a`, `M3`…) ficam de fora —
importar JP dos EUA não faz sentido.

---

## Fase 1 — Montar os dois lados

**Lado BR** — `data/tracking-<jogo>/<SET>/*.json`, arquivo mais recente. Só
`language == '2'` (inglês; PT=`8`, JP=`6`), `condition in ('1','2')`.
Guarde piso, nº de lojas distintas e quantidade total.

**Lado US** — TCGCSV por categoria: Pokémon `3`, One Piece `68`, Lorcana `71`,
Gundam `86`, Riftbound `89`.

```bash
curl -s "https://tcgcsv.com/tcgplayer/3/groups" -o groups.json
curl -s "https://tcgcsv.com/tcgplayer/3/${GROUP}/products" -o p.json
curl -s "https://tcgcsv.com/tcgplayer/3/${GROUP}/prices"   -o pr.json
```

`urllib` leva **401** nesse host — use `curl`. As abreviações dos grupos casam
com os códigos de set da Liga em Pokémon (PBL, CRI, POR…). Em Lorcana o grupo
vem como `"12"` e a Liga usa `LOR12`. Em One Piece e Gundam o número já é global
(`OP01-001`), não escope por set; em Lorcana e Riftbound **escope**, e normalize
número tirando denominador e zeros à esquerda (`078/084` → `78`). Riftbound tem
sufixo por variante: `A`→alt, `S`/`*`/`P`→sig.

Quando existir `data/snapshot-<jogo>.json`, prefira o join por
`(SetCode, Number, Variant)` dele — o `Variant` já vem normalizado dos dois
lados, o que resolve alt art/manga/parallel que o número sozinho não separa.

### Números ambíguos são a armadilha nº 1

Dois produtos com o mesmo número no mesmo set existem e destroem a análise. Em
White Flare, `084/086` é tanto "Hilda" (Uncommon, US$ 1,06) quanto "Hilda (Poke
Ball Pattern)" (US$ 3,00) — o dicionário sobrescreve e sai um falso 16×.
**Marque `ambiguo = len(candidatos) > 1` e trate pela regra 4** (usar o mais
caro), ou descarte. Em One Piece isso atinge ~85% das cartas: a Liga cadastra
alt art/manga sob o mesmo número sem campo de variante, e por isso o topo de
qualquer lista de OP é pouco confiável.

---

## Fase 2 — Giro real do BR

Mesma inferência do `internal/tracking/analysis.go`: entre dois snapshots
consecutivos, oferta cuja quantidade caiu vendeu a diferença. Só conta se a loja
existe nos dois lados e `known` nos dois.

```python
def key(c, s):
    return (c['number'], s['storeId'], s.get('condition'),
            s.get('language'), s.get('extras', 0), s.get('sellType', 1))
```

> **Lorcana e capturas antigas não gravam `extras`/`sellType`.** Sem esses
> campos a mesma loja aparece em várias linhas (foil e não-foil na mesma página)
> e a chave colapsa: o `dict` fica com a última linha, a quantidade oscila entre
> snapshots e vira venda fantasma. Em `tracking-lor/LOR13` são **286 linhas
> duplicadas em 1106** — 26%. Inflou o giro de "With a Few Good Friends" para
> R$ 146 de preço médio numa carta cujo piso é R$ 13,90.
>
> **Correção:** agregue por `(número, loja, condição, idioma)` somando
> quantidade e pegando o menor preço; meça a queda no agregado; e para receita
> use o piso do card no snapshot atual, não o preço da linha. Confira antes:
> Pokémon tem 1% de duplicata (ignorável), Riftbound ~0%, Lorcana 26%.

Reporte `soldEN` (unidades) e o **preço médio realizado**. Preço realizado
manda mais que o piso: é o que o mercado pagou de fato.

---

## Fase 3 — O portão de meta e rotação

Não é opcional e não pode sair da memória — os dois jogos rotacionam e a
informação envelhece. Pesquise antes de cada rodada.

**Pokémon** — legalidade é por *regulation mark*, não por set. A rotação de
abril/2026 tirou o `G`; valem `H`, `I`, `J`. Isso mata o 151 (MEW) inteiro:
Charizard ex SIR, Charmander, Poliwhirl, Mew ex — todos apareciam com gap de
2–5× e **zero venda EN em 5 semanas**. Preço de colecionador parado.

O que vale em Pokémon são os **full art de staple**: joga em todo deck e o BR
não tem oferta em inglês. Uso medido (pokemoncard.io/top): Lillie's Determination
54%, Ultra Ball 37%, Night Stretcher 36%, Poké Pad 35%, Boss's Orders 34%.
ACE SPEC também (Precious Trolley 6,6%). Cruze sempre com o gap — Lillie's é a
nº 1 do formato e mesmo assim rende pouco por unidade (custa US$ 19,50).

**Riftbound** — **não rotaciona até 2028**; todos os sets são legais em Standard
Constructed. É o único dos quatro sem esse risco, o que permite comprar set
antigo (OGN) sem prazo de validade. Meta em agosto/2026: Kennen, Heart of the
Tempest 9%, Master Yi e Irelia atrás. Fontes: `riftbound.gg`, `riftdecks.com`,
`riftmana.com`. Cuidado com o set novo: **Vendetta (VEN) saiu em 31/07/2026 e
não tem diretório em `data/tracking-rft`** — tem preço US mas nenhum giro BR
medido, então fica fora do ranking por falta de dado, não por falta de
oportunidade.

**Lorcana** — Core Constructed são os sets **9 a 13** desde a rotação de
24/07/2026 (saíram os sets 5–8). Próxima rotação só em julho/2027, então quem
comprar LOR9 tem ~11 meses de vida competitiva. Arquétipo que dá nome a deck é
compra segura: "Amber/Emerald **Elinor**" fez de Elinor - Renowned Diplomat
staple garantido; Woody - Jungle Guide sustenta Steelsong; Dale - Ready for His
Shot tem deck próprio.

Fontes: `inkdecks.com` (uso por carta), `melee.gg` (resultado cru), `lorcana.gg`
(rotação), `pokemoncard.io/top` (uso em Standard), `limitlesstcg.com`.
**`inkdecks.com` e `lorcana.gg` devolvem 403 no WebFetch** — use WebSearch, que
traz o resumo, ou o navegador.

**Enchanted/SIR não são as campeãs.** 179 Enchanted de Lorcana acima de US$ 20
têm mediana 1,42× — praticamente igual às não-Enchanted (1,32×) — mas travam
R$ 1.000–4.000 num item que vende 1–2 por mês. Eeyore Enchanted: custo R$ 3.621,
piso R$ 4.000, **1,6% de margem**. A campeã de verdade é a legendary de
US$ 3–15 com giro alto.

---

## Fase 4 — Ranquear e dimensionar

```
custo    = usMkt / fxRate          # fxRate do snapshot é BRL→USD (0.19663)
venda    = piso_BR × 0,92          # ou o preço médio realizado, se houver
lucro    = venda − custo
score    = lucro × min(soldEN, 40)
```

Filtros mínimos: `lojas_BR ≥ 6`, `soldEN ≥ 3`, `usMkt ≥ 3`, `lucro ≥ 10`.
Apresente duas tabelas: **lucro total no período** (as campeãs) e **lucro por
unidade** (as gordas de giro baixo). Nunca apresente multiplicador sozinho — o
dono já pediu explicitamente para não decidir por ele.

Corte sempre e diga por quê: piso BR abaixo do custo US (existe, e mais do que
parece), lucro/unidade abaixo de ~R$ 10, e item caro com margem percentual
pequena (US$ 25 para ganhar R$ 15 é capital mal usado).

---

## Fase 5 — Achar os lotes no TCGplayer

Preço de tabela não é o que você paga; o que importa é quem tem **quantidade**.
A API pública de listings dá vendedor, quantidade, condição, printing e frete:

```bash
curl -s -X POST "https://mp-search-api.tcgplayer.com/v1/product/${PID}/listings" \
  -H "Content-Type: application/json" \
  -H "Origin: https://www.tcgplayer.com" -H "Referer: https://www.tcgplayer.com/" \
  -H "User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0 Safari/537.36" \
  -d '{"filters":{"term":{"sellerStatus":"Live","channelId":0},"range":{"quantity":{"gte":4}}},"from":0,"size":50,"sort":{"field":"price+shipping","order":"asc"},"context":{"shippingCountry":"US","cart":{}}}'
```

- **Sem `User-Agent` de browser dá 403.** O header é obrigatório.
- `size` acima de 50 dá **400**. Para achar lote fundo, filtre
  `quantity.gte: 4` em vez de aumentar o `size` — sem isso os 50 mais baratos
  são todos anúncios de 1 cópia e o lote grande fica invisível. Foi assim que
  apareceram 45× Maleficent, 43× Lilo & Stitch, 29× Night Stretcher.
- `productId` sai de `data/snapshot-*.json` (`prices[].ProductID`) ou dos
  `products` do TCGCSV.
- Filtre `condition == 'Near Mint'` e cheque `printing`: Lorcana é
  `Normal`/`Cold Foil`, Pokémon quase sempre `Holofoil` nas UR/SIR/ACE SPEC.

**Consolidar por vendedor tem preço — meça os dois cenários antes de escolher.**
Cada vendedor cobra frete próprio (~US$ 1,49), então o instinto é agrupar. Só que
o vendedor que tem tudo raramente é o mais barato em tudo, e o prêmio que ele
cobra pode passar de longe o frete economizado. Rode sempre as duas contas:

| Cenário | Riftbound (medido) | Lorcana (medido) |
| --- | --- | --- |
| Lote mais barato por carta | ~15 pedidos, **66%** de retorno | 16 pedidos |
| Guloso por cobertura | 8 pedidos, **46%** de retorno | 16 pedidos, custo ~igual |

Em Riftbound consolidar custou 20 pontos de retorno: Arise! passou de US$ 15,98
para US$ 23,00, Seal of Insight de US$ 15,00 para US$ 19,00, Defiant Dance de
US$ 18,99 para US$ 25,00 — cada um desses saltos vale muitos fretes de US$ 1,49.
Em Lorcana o spread entre o mais barato e o vendedor-que-tem-tudo era pequeno e
consolidar saiu de graça. **Não presuma: compare `Σ preço + Σ frete` nos dois.**

Descarte na hora qualquer pedido com frete acima de ~US$ 7 (apareceu um de
US$ 14,99) — a menos que ele sozinho feche uma carta que ninguém mais tem.

Pokémon consolida pior que os dois: o mercado é dominado por anúncio de 1 cópia,
então aceite mais pedidos e não force teto apertado, senão a busca não fecha a
quantidade.

---

## Fase 6 — Entregar o pedido para quem vai comprar

**O TCGplayer bloqueia o Brasil.** Não é problema de cartão: eles não processam
comprador brasileiro, com Wise, PayPal ou qualquer coisa. A rota é a pessoa nos
EUA comprar com o cartão dela e ser reembolsada por Wise.

Carrinho não se compartilha, mas o **Mass Entry sim**. A URL aceita a lista
pré-preenchida, e a própria página tem um botão **"Create a Shareable Link"**.

### Formato da linha

Documentação oficial (`Getting Started With Mass Entry`):

```
Quantidade → Nome da Carta → [Código do Set] → Número na coleção
```

```
1 Lightning Bolt [SLD] 84
```

Os três últimos campos são opcionais e vão do específico ao genérico: sem o
número, qualquer arte daquele set serve; sem o set, qualquer impressão serve.

**Em Pokémon o set é obrigatório na prática.** "Meowth", "Psyduck", "Ultra
Ball", "Night Stretcher", "Boss's Orders", "Hilda", "Latias ex", "Genesect ex"
existem em dezenas de sets — sem `[SET] número` a pessoa escolhe à mão carta por
carta e traz a errada. Escreva sempre completo:

```
20 Enriching Energy [SSP] 191
18 Boss's Orders [ASC] 256
14 Poke Pad [POR] 113
```

**O código do TCGCSV nem sempre é o código do Mass Entry.** Em Pokémon
`groups[].abbreviation` bate (SSP, ASC, POR, PBL, CRI, MEG, PFL, DRI, JTG, MEW,
BLK, WHT). Em **Riftbound não bate**: `[UNL]`, `[OGN]` e `[SFD]` fazem as 15
linhas falharem com *"The following card(s) could not be found"*, mesmo com o
Product Line certo. Em Lorcana a abreviação do TCGCSV é numérica (`"12"`).

Regra prática: **só use `[SET]` em Pokémon.** Nos outros jogos mande só
`qtd Nome da Carta` — os nomes são únicos o bastante — e, se precisar do set,
pegue o código real no botão **"Show Set/Series Codes"** da própria página
(ele lista `Nome do Set [CÓDIGO]` e copia ao clicar).

> Aviso para automação: o seletor de Product Line **não aceita seleção por
> teclado nem por clique programático** — reverte para o primeiro item da lista
> (Akora TCG) e os códigos exibidos passam a ser do jogo errado. Não insista;
> peça o print ao dono ou trabalhe sem set code.

### Formato da URL

```
https://www.tcgplayer.com/massentry?productline=Pokemon&c=20%20Enriching%20Energy%20%5BSSP%5D%20191||18%20Boss%27s%20Orders%20%5BASC%5D%20256
```

- Separador entre itens: `||` (literal, não precisa escapar).
- `productline` tem que ser **o rótulo exato do dropdown**, URL-encoded.
  Hífen no lugar de espaço falha **em silêncio**: `productline=Disney-Lorcana`
  carrega a lista mas deixa o seletor vazio. Valores certos:
  `Pokemon` · `Disney%20Lorcana` · `One%20Piece%20Card%20Game` ·
  `Riftbound%3A%20League%20of%20Legends%20Trading%20Card%20Game` ·
  `Gundam%20Card%20Game` · `Magic%3A%20The%20Gathering`.
- Espaço `%20`, apóstrofo `%27`, `&` `%26`, colchetes `%5B` `%5D`.

### Item Options (o gear ao lado do Add to Cart)

Por padrão vem **todas as impressões e tudo de Moderately Played para cima** —
ou seja, o carrinho pode encher de Reverse Holo e carta jogada. Mande a pessoa
deixar só **Near Mint** e a impressão certa (Lorcana `Normal`, Pokémon
`Holofoil` nas UR/SIR/ACE SPEC) antes de adicionar.

- **O link leva só carta e quantidade — a escolha de vendedor se perde.** Mande
  junto a tabela de lotes (vendedor + quantidade + preço), senão a pessoa clica
  Add to Cart e cai no espalhamento.
- **O "Optimize" espalha.** Ele minimiza preço por item e subestima o frete por
  vendedor: 28 Woody saíram de 15 lojas numa tentativa real. Para as cartas de
  volume, compre no anúncio grande na mão (abre o produto, filtra NM/English,
  escolhe a quantidade no seletor ao lado do preço do vendedor certo).
- `tcgplayer.com/cart-optimizer` **dá 404**, inclusive pelo link do rodapé — o
  Cart Optimizer é um botão dentro do carrinho.
- Para vendedor que tem várias cartas da lista, rodapé → **"Find a Seller"** →
  busca pelo nome → monta um carrinho só. Um único vendedor cobriu 10 das 11
  cartas de um plano de Lorcana com US$ 0,99 de frete.

**eBay não substitui.** Preço de carta empata com o TCGplayer, mas o frete
internacional é **por vendedor** (R$ 137–147) e tudo é unidade avulsa. Só presta
em lote grande: um "lot of 12" a R$ 357 + R$ 185 de frete dá R$ 45/carta contra
R$ 12,70 no TCGplayer. Busque com `lot`, `playset`, `x4` — sem isso a busca só
devolve avulso. Vale como plano B porque **aceita comprador brasileiro**.

---

## Fase 7 — O entregável

O produto final **não é um ranking, é um plano de compra**. Toda rodada termina
com as quatro peças abaixo, nesta ordem. Entregar só parte disso é entregar pela
metade — o dono não consegue decidir sem ver quanto cada linha rende.

### 1. A tabela do plano (obrigatória, é o coração da entrega)

Uma linha por carta, com **quantidade decidida** e o lucro daquela quantidade:

| Qtd | Carta | Set | US$/un | US$ total | Piso BR | Vende a | Lucro total | Giro |

- **`US$/un` é custo ponderado**, não o preço do anúncio mais barato: some os
  lotes do mais barato para cima até fechar a quantidade e divida. Comprar 13
  quando o lote barato tem 3 significa pagar mais caro nas outras 10, e o
  ponderado é o único número honesto.
- `Vende a` = piso BR × 0,92 (ou o preço médio realizado, se houver).
- `Giro` = unidades EN vendidas no BR na janela, e **diga a janela em dias**.
  É o teto de quanto faz sentido trazer, então tem que estar visível na linha.
- Ordene por lucro total decrescente.
- Rodapé com: nº de cartas, custo total em US$ e R$, venda total, **lucro**,
  **retorno %** e **peso** (carta ≈ 1,7 g).

### 2. Onde está o dinheiro e o que cortar

Duas leituras que saem da mesma tabela e mudam a decisão de capital:

- **Concentração:** quantas linhas do topo fazem metade do lucro, e com quanto
  de capital. Em Riftbound foram 5 linhas: R$ 3.046 de R$ 5.638 com US$ 447 —
  retorno de 134% contra 69% do plano inteiro. Se o dono quiser começar pequeno,
  é exatamente essa a lista.
- **Piores por real investido:** as linhas de capital alto e lucro baixo, com o
  plano recalculado sem elas. Cortar Seal of Rage, Defiant Dance e Seal of
  Insight tirou US$ 536 (um terço do capital) por 15% do lucro: o plano foi de
  144 cartas/69% para 111 cartas/**88%**.

Sempre ofereça as duas versões — completa e enxuta — com o retorno de cada uma.

### 3. O texto pronto para colar no Mass Entry

Em bloco de código, no formato da Fase 6, já com a regra de set code do jogo
certo. Nunca só o link: **link não dá para conferir antes de colar**, e o dono
costuma estar num terminal onde markdown não é clicável.

### 4. Os pedidos por vendedor

Vendedor, quantidade, preço unitário e frete — é o que identifica o anúncio sem
ambiguidade quando outra pessoa vai comprar. Marque quais agrupamentos valem e
quais não (Fase 5).

### E mais, em qualquer ordem

- **O que foi cortado e por quê** — rotação, liquidez fina, margem percentual
  pequena, match ambíguo, preço que subiu entre o snapshot e o anúncio. Isso
  vale tanto quanto a lista: é o que impede o dono de "consertar" o plano
  recolocando o que você tirou.
- **A alfândega.** Isenção de bagagem aérea é US$ 1.000 por pessoa; o excedente
  paga 50%. Some os planos de todos os jogos antes de dizer que fecha. E avise:
  o plano só funciona porque vem na mala — **redirecionador para o Brasil paga
  60% de Imposto de Importação sem faixa de isenção mais 17–20% de ICMS** desde
  maio/2026, o que inverte a conta inteira.
- A data dos snapshots dos dois lados. Se o lado BR for de 8 dias atrás e o US
  de agora, diga — e confira à mão os 3 itens que concentram o capital.

---

## Ressalvas do método

- **Refaça a conta com o preço do lote — isso reprova candidato.** O
  `marketPrice` do snapshot é média histórica e atrasa; o lote real cobra prêmio
  ou já subiu. Em Riftbound, Invert Timelines (OGN 201) entrou no ranking com
  US$ 14,27 do snapshot e o lote mais barato ao vivo estava a **US$ 47,97** —
  R$ 593 de lucro projetado viraram **R$ 134 de prejuízo**. Na mesma rodada Pyke
  caiu de R$ 46 para R$ 11/un e Thrill of the Hunt para R$ 1. **A Fase 5 não é
  logística, é a segunda validação do ranking**: rode-a em toda a lista antes de
  fechar quantidade, e recalcule o lucro com o preço do anúncio.
- **Janela de tracking desigual entre jogos.** Não presuma — conte os arquivos.
  Pokémon ~5 semanas, Riftbound 34 dias em 70 capturas (parece curto pelo número
  de diretórios de set, não é). Giro baixo em jogo com histórico curto é
  subestimativa, não ausência de demanda; normalize por dias antes de comparar.
- **Lote de 45 cópias some rápido.** Entre a análise e a compra costumam passar
  horas; refaça a Fase 5 na hora de fechar, e tenha o segundo da fila anotado.
