---
name: liga-orcamento-pokemon
description: Confere uma lista de venda de cartas Pokémon (colada de WhatsApp) contra o piso atual da LigaPokemon usando os snapshots locais do tracking, e aponta o que está caro, no piso ou abaixo. Use quando o pedido for conferir/orçar uma lista de cartas Pokémon, "essa lista bate com a liga?", avaliar proposta de vendedor ou lote de compra.
allowed-tools: Bash, Read, Write
---

# Orçamento Pokémon — conferir lista de vendedor contra a Liga

Responde "essa lista bate com o site da Liga?" sem abrir o site: os snapshots
do tracking (`data/tracking-pkm/<SET>/*.json`, capturas a cada ~6h) guardam
**todas as ofertas** de cada carta com loja, condição, idioma, quantidade e
preço. Daí sai o piso NM por idioma, a comparação carta a carta e os destaques
de negociação.

Esta skill é **só análise — não grava nada em lugar nenhum**. Precificar o
NOSSO estoque é `liga-precificar`; auditar a loja é `liga-conferir`; publicar
cartas é `liga-singles-pokemon`; criar Orçamento no dashboard é a aba
Orçamento do próprio dashboard.

---

## Fase 0 — Parsear a lista colada

Formato típico de WhatsApp, uma carta por linha:

```
1 Psyduck (226/217) 225,00 cada
1 Banette (234/217) 44,00 cada ingles
1 Trevenant do Lupo (237/217) 25,00
```

- `<qtd> <nome PT> (<num>/<den>) <preço> [cada] [ingles]` — vírgula decimal,
  "cada" opcional, flag de idioma solta no fim da linha (`ingles`, `ing`,
  `EN`, `japones`...). Sem flag = assumir **PT**.
- Linhas-cabeçalho com nome de set em PT ("Mega Evolucao",
  "Fogo Fantasmagorico", "Mais algumas que não estão as fotos") **trocam o
  contexto** das linhas seguintes — o denominador delas pode ser de outro set.
- Linha sem preço ou sem `(num/den)` → vai para a seção "não parseado" do
  relatório. **Nunca descarte em silêncio** — o dono precisa saber o que ficou
  de fora da soma.

---

## Fase 1 — Resolver o set pelo denominador

O denominador (`/217`) identifica o set, mas **não existe mapa estático no
repo** — os códigos são siglas próprias da Liga (`game.Pokemon()` em
`internal/game/game.go`). Construa o mapa em runtime escaneando o snapshot
mais recente de cada set:

```python
import json, glob, os, re
den2set = {}
for d in sorted(glob.glob('data/tracking-pkm/*/')):
    s = os.path.basename(d.rstrip('/'))
    if s == 'SEALED': continue
    files = sorted(glob.glob(d + '*.json'))
    if not files: continue
    for c in json.load(open(files[-1]))['cards']:
        m = re.search(r'/(\w+)\)', c['name'])
        if m: den2set.setdefault(m.group(1), set()).add(s); break
for den, sets in sorted(den2set.items()): print(den, sorted(sets))
```

Referência do mapa em 2026-08 (rode o snippet — sets novos aparecem sozinhos):

| den | set | | den | set |
| --- | --- | --- | --- | --- |
| 217 | ASC (Heróis Excelsos) | | 098 | SV10 |
| 193 | M2a | | 094 | PFL (Fogo Fantasmagórico) |
| 191 | SSP | | 088* | POR |
| 182 | DRI | | 084 | PBL |
| 165 | MEW | | 083 | M4 |
| 159 | JTG | | 081 | M5 |
| 132 | MEG (Mega Evolução) | | | |

- \* **POR usa `88` sem zero** no nome das cartas (`#012/88`) — o denominador
  da lista pode vir padded ou não; compare os dois jeitos.
- **Colisões conhecidas — desempate por NOME, nunca por chute:**
  `063 → {M1L, M1S}` · `080 → {M3, m2}` (m2 é minúsculo mesmo) ·
  `086 → {BLK, CRI, WHT}`. Procure o número nos candidatos e fique com o que
  tiver o nome correspondente; se mais de um casar, pare e pergunte.
- Cabeçalhos de set em PT na lista ajudam a desempatar:
  Heróis Excelsos→ASC, Mega Evolução→MEG, Fogo Fantasmagórico→PFL,
  Rivais Predestinados→DRI, Aventuras Juntos→JTG, Faíscas Impetuosas→SSP,
  **Coleção 151→MEW** (não confundir: MEW é o 151, den 165 — eu já errei
  isso chamando MEW de "Evoluções Prismáticas").

---

## Fase 2 — Casar cada carta no snapshot

Snapshot mais recente do set: `sorted(glob.glob('data/tracking-pkm/<SET>/*.json'))[-1]`
— os nomes de arquivo (`2026-08-01T06.json`) ordenam lexicograficamente, mesmo
critério do `Store.LatestDay` em `internal/tracking/storage.go`. Leia o
`capturedAt` e **reporte o frescor** — o dado tem até 6h de atraso.

- O campo `number` é **zero-padded a 3 dígitos** (`"003"`). A lista vem sem
  padding (`3/217`). Normalize dos dois lados.
- **Trava de nome (a lição CEL/CCC vale aqui também).** O número casou, mas o
  nome do snapshot (EN) tem que corresponder ao nome PT da lista, carta a
  carta, lido de verdade — número e denominador são estruturais e colidem;
  só o nome discrimina. Nome que não bater → linha `⚠️ verificar` no
  relatório e o preço **não entra** nas somas.
- Correspondências PT↔EN já validadas (donos de carta viram possessivo em EN):

  | Lista (PT) | Snapshot (EN) |
  | --- | --- |
  | X do Lupo | Hop's X |
  | X do Lauro | Larry's X |
  | X da Kissera | Iono's X |
  | X da Lílian | Lillie's X |
  | X da Marine | Marnie's X |
  | X da Equipe Rocket | Team Rocket's X |
  | X do N / PP Up do N | N's X |
  | Espírito de Luta da Iris | Iris's Fighting Spirit |
  | Amor e Paz | Anthea & Concordia |
  | Ordem da Chefia | Boss's Orders |
  | Ultra Bola | Ultra Ball |
  | Rotom Ventilador | Fan Rotom |
  | Trompete de Vidro | Glass Trumpet |
  | Treino de Faixa Preta | Black Belt's Training |

  Nome novo que você traduzir e confirmar: **acrescente à tabela**.
- Número ausente do snapshot → **"sem dados"**. A cobertura é parcial por
  design (o tracking segue uma lista curada — ASC tem 119 das 295 cartas).
  Nunca infira preço de carta parecida nem use outra printagem.

---

## Fase 3 — Calcular os pisos (o coração da skill)

> ⚠️ **`lowBRL` do snapshot é o piso cru de QUALQUER condição/idioma — nunca
> use como referência principal.** O caso que criou esta regra: Psyduck
> ASC 226 com `lowBRL: 199,90` — só que essa oferta é uma cópia **SP**; o piso
> NM real era R$ 200. Comparar lista NM contra piso de carta surrada infla o
> "desconto" do vendedor.

Recalcule o piso a partir de `stores[]`, filtrando
`priceKnown && known && quantity > 0`:

| Piso | Filtro | Uso |
| --- | --- | --- |
| NM-PT | `condition ∈ {1,2}` e `language == "8"` | referência default |
| NM-EN | `condition ∈ {1,2}` e `language == "2"` | referência quando a linha diz "ingles" |
| menor do site | nenhum (= `lowBRL`) | coluna informativa |

Códigos (mesmos das skills irmãs): condição `1` M · `2` NM · `3` SP · `4` MP ·
`5` HP · `6` D; idioma `2` EN · `6` JP · `8` PT · `11` PT/EN. Existem linhas
lixo com `condition:"0", language:"0"` — o filtro acima já as elimina.

Snippet canônico (LIST montada na Fase 0/2, uma tupla por linha):

```python
import json, glob

LIST = [
    # (set, num_3dig, nome_lista, preco, idioma_flag ou None)
    ("ASC", "226", "Psyduck", 225.00, None),
    ("ASC", "234", "Banette", 44.00, "EN"),
]

snaps, meta = {}, {}
for s in {r[0] for r in LIST}:
    f = sorted(glob.glob(f'data/tracking-pkm/{s}/*.json'))[-1]
    d = json.load(open(f))
    snaps[s] = {c['number']: c for c in d['cards']}
    meta[s] = d['capturedAt']

def floor(card, langs=None):
    ps = [o['priceBRL'] for o in card['stores']
          if o.get('priceKnown') and o.get('known') and o.get('quantity', 0) > 0
          and o['condition'] in ('1', '2')
          and (not langs or o['language'] in langs)]
    return min(ps) if ps else None

tot_l = tot_r = tot_low = 0
rows, semdados = [], []
for st, num, nome, preco, lang in LIST:
    card = snaps[st].get(num)
    if not card:
        semdados.append((st, num, nome, preco)); continue
    nmpt, nmen = floor(card, {'8'}), floor(card, {'2'})
    ref = nmen if lang == 'EN' else nmpt
    diff = None if not ref else (preco - ref) / ref * 100
    rows.append((st, num, card['name'], preco, card['lowBRL'], nmpt, nmen, diff, lang))
    tot_l += preco
    if ref: tot_r += ref
    tot_low += card['lowBRL']

for r in sorted(rows, key=lambda r: -(r[7] or 0)):
    fm = lambda v: f'{v:8.2f}' if v is not None else '      --'
    print(f'{r[0]} {r[1]} {r[2]:<38} lista {r[3]:8.2f} menor {fm(r[4])}'
          f' NM-PT {fm(r[5])} NM-EN {fm(r[6])} diff {f"{r[7]:+.1f}%" if r[7] is not None else "--"}')
print(f'\ntotais: lista {tot_l:,.2f} | piso NM ref {tot_r:,.2f}'
      f' ({(tot_l-tot_r)/tot_r*100:+.1f}%) | menor site {tot_low:,.2f}')
for s in semdados: print('SEM DADOS:', s)
for s, ts in meta.items(): print(f'snapshot {s}: {ts}')
```

O nome do snapshot impresso na tabela é a materialização da trava de nome da
Fase 2 — **leia a coluna inteira** antes de entregar, não confie que o número
bastou.

---

## Fase 4 — Relatório (o que sempre entregar)

- **Veredito primeiro**: a lista bate ou não com a Liga, e o % agregado
  (`total lista vs total piso NM`). Uma lista tirada da Liga tem cara de
  0 a +13% por carta com vários preços cravados no piso.
- Tabela carta a carta: nº, nome do snapshot, preço da lista, menor do site,
  piso NM-PT, piso NM-EN, diff % contra a referência do idioma da linha.
- **Destaques em duas listas**, ordenadas por diff:
  - abaixo do piso → oportunidade de compra;
  - bem acima do piso (>15%) → alavanca de negociação.
- Ressalvas obrigatórias, sempre:
  - data/hora de cada snapshot usado;
  - a comparação assume cópias **NM** — se o vendedor tiver SP/MP a referência
    cai (e a Liga tem mercado ativo de SP: ver o próprio Psyduck);
  - carta = piso EN exato mesmo sem flag pode ser cópia EN não declarada
    (aconteceu com Canari: lista 21,90 = piso EN, +22% sobre o PT);
  - lista de "sem dados" e "não parseado" com o valor que ficou de fora.

---

## Variante B — orçamento feito como Coleção no próprio site

Se o vendedor mandar um link `?view=colecao/colecao&id=NNNN` em vez de texto,
os preços já SÃO o menor preço da Liga por construção — o site precifica cada
carta ao vivo no "Menor P. Compra" (piso da variante+qualidade exatas). O
trabalho muda: extrair, somar e **verificar contra o snapshot**, não comparar
preço pedido vs mercado.

- Abrir no Chrome do usuário (CF challenge passa sozinho em ~8s; ritmo 7–10s
  entre navegações). Paginação: `&page=2` — 80 cartas por página.
- Cada linha: `1x NNN/165` + nome + `<Extra> - <Qual> <Rar> R$ <compra> R$ <venda>`.
  Extra vazio = carta normal. Linhas sem preço = sem oferta na Liga (ficam
  fora do total do site — reportar).
- Cabeçalho tem 3 totais (menor/médio/maior) e a linha `160x R$ X R$ Y` =
  soma Menor P. Compra / soma Menor P. Venda (buylist). Confira que seu parse
  fecha com o total do site ao centavo antes de confiar nele.
- Verificação: cruzar as cartas ≥R$50 (carregam ~90%+ do valor) com o piso
  NM-PT do snapshot. Numa coleção 151 de R$ 8.3k, 13/16 bateram com diff
  0,0% — divergência é sinal de coisa específica, não de erro geral:
  - oferta-piso vendida desde a captura (Mr. Mime: snapshot 220, site 259,90
    — o site está certo, é ao vivo);
  - **variante**: ver armadilha de Reverse Foil abaixo;
  - promo fora do tracking (Snorlax `051/∞` é da edição Escarlate e Violeta
    Promos, não do 151 → sem conferência independente).
- Leitura para o dono: o total do site é **custo de reposição no piso NM**,
  não preço justo de compra de lote — a soma "Menor P. Venda" (buylist) é o
  outro extremo. Idiomas alternativos derrubam o valor de uso: no 151, o
  Pikachu IR tinha JP a 350 contra PT a 611, e o Psyduck IR tinha KO a 300
  contra PT a 440.

---

## Armadilhas que já custaram análise errada

- **`lowBRL` ≠ piso NM** (Psyduck, Fase 3). É a maior fonte de veredito errado.
- **POR denomina `88` sem zero** enquanto todos os outros sets usam 3 dígitos.
- **`m2` é minúsculo** no nome do diretório — `glob` case-sensitive não acha
  `M2`.
- **Denominadores colidem** (063, 080, 086) e vão colidir mais com o tempo —
  o desempate é sempre pelo nome da carta.
- **Cobertura parcial não é ausência de mercado.** "Sem dados" significa que o
  tracking não segue a carta, não que ela não tem oferta na Liga. Não conclua
  "carta sem liquidez" a partir disso.
- **O tracking não distingue extras (Reverse Foil/Foil/Pre Release).** As
  ofertas em `stores[]` misturam as variantes da carta, então o piso do
  snapshot tende a ser o da versão normal (mais barata). No 151, o Gengar 094
  Reverse Foil valia R$ 85 no site e o snapshot dava piso de R$ 30 — a versão
  normal. Para carta comum/incomum com "Reverse Foil" na lista, o piso do
  snapshot **subestima**; confie no preço por variante do site (Variante B) ou
  marque a linha como não-conferível. Secretas (IR/IS/RD/RU, número acima do
  denominador) só existem foil, então nelas o problema não aparece.

---

**Aprendeu algo novo? Acrescente aqui — este arquivo é o repositório desse
conhecimento.**
