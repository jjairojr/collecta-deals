---
name: liga-precificar
description: Define o preço de venda de itens da Collecta na LigaMagic medindo o giro real por posição no comparador, a partir dos snapshots do tracking. Use quando o pedido for precificar, reprecificar, liquidar estoque, decidir "por quanto vender", ou avaliar se um item está caro/barato demais para girar.
allowed-tools: Bash, Read, Write, Edit, mcp__claude-in-chrome__*
---

# Precificar para girar na LigaMagic

Responde "por quanto vender para girar com margem" usando dado medido, não
palpite: os snapshots do tracking guardam preço e quantidade de **cada loja**
a cada 6h, então dá para inferir quantas unidades saíram em cada faixa de
posição do comparador — e daí qual preço maximiza lucro/dia ou zera o estoque
num prazo.

Cadastrar item novo é outra coisa (`/liga-cadastrar`, `liga-singles-*`);
conferir o que está publicado é `liga-conferir`.

---

## As três regras que mandam

**1. Giro é por ANÚNCIO, não por unidade.** Um anúncio na 110ª posição vende
~0,27 un/dia tendo você 5 ou 39 unidades. Comprar mais de um SKU lento não
acelera nada — só trava capital. Daí sai a régua de compra:

> estoque ideal = giro no preço-alvo × dias de cobertura desejados

**2. A elasticidade escala com o ticket.** Medido em ME5 (razão de giro entre
top-5 e 100º+):

| Ticket | Razão | Consequência |
| --- | --- | --- |
| R$ 26–50 (blisters) | 2,5–4,5× | inelástico: **segure margem** |
| R$ 150–400 (caixa, combo, ETB) | 10–18× | elástico: **brigue por posição** |

Item barato entra no carrinho como "completa frete", sem pesquisa de preço.
Preço único para o catálogo inteiro é o erro clássico.

**3. Não basta o rank — importa a parede à sua frente.** Ficar 1 centavo acima
de uma loja com 141 unidades é ficar invisível: o comprador esvazia ela
primeiro. Sempre olhe a **quantidade** dos vizinhos, não só o preço.

---

## Fase 0 — Custo e taxa (sem isso não há análise)

Custo vem de **produção**, nunca de `data/*.json` local:

```bash
B=https://collecta-deals-production.up.railway.app
curl -s --max-time 25 "$B/api/trades?game=pokemon" | python3 -c "
import json,sys
d=json.load(sys.stdin)
items=d if isinstance(d,list) else d.get('trades',d.get('items',[]))
for t in items:
    if t.get('status')!='holding' or not t.get('qty'): continue
    print('{:<9} qty={:<4} custo={:<9} ask={:<9} {}'.format(
        (t.get('id') or '')[:8], t.get('qty'), t.get('buyBRL') or '-',
        t.get('askBRL') or '-', (t.get('name') or '')[:52]))
"
```

`buyBRL` já inclui rateio de frete. Confira contra a nota do fornecedor: a
COPAG trabalha **43% off do preço sugerido, flat** em toda a linha — se a razão
custo/sugerido não der ~0,57 em todos os itens, você entendeu errado a unidade
de compra (caixa master × unidade de venda).

**Taxa do Marketplace da Liga** (só em venda pelo marketplace, não pela loja
própria): `4,99% + R$ 0,40` no repasse em 14 dias, `3,99% + R$ 0,40` em 30 dias.
Líquido = `preço × (1 − taxa) − 0,40`.

> ⚠️ **Loja nova fica em repasse de 60 dias + validação de entrega.** Vender em
> 14 dias não é receber em 14 dias. Antes de recomendar liquidação a 1–2% de
> margem, **confirme o prazo real em LigaSegura** — se for 60 dias, abrir mão da
> margem não compra velocidade de caixa e a liquidação perde o sentido.

---

## Fase 1 — Medir o giro por posição

Fonte: `data/tracking-<jogo>/<SET>/*.json` (`SEALED` para selados). Cada arquivo
traz `cards[].stores[]` com `priceBRL`, `quantity`, `known`, `priceKnown`.

Inferência: entre dois snapshots consecutivos, uma oferta cuja quantidade caiu
**e continua > 0** vendeu a diferença. Bucketize pela posição que ela ocupava
*antes* da queda, e divida por quantas observações de anúncio houve naquele
bucket — sem normalizar, a faixa "101+" parece a melhor só porque tem 180 lojas.

```python
import json, glob, collections, bisect, os
from datetime import datetime

EDGES = [3,5,10,20,35,50,75,100,150,200,10**9]
def bucket(r): return bisect.bisect_left(EDGES, r)
def stamp(f):
    b = os.path.basename(f)[:-5]
    return datetime.strptime(b, '%Y-%m-%dT%H' if 'T' in b else '%Y-%m-%d')

files = sorted(glob.glob('data/tracking-pkm/SEALED/*.json'))
sold, hours, prev, prevT, last = collections.Counter(), collections.Counter(), None, None, None
for f in files:
    card = next((c for c in json.load(open(f))['cards'] if c['name'] == NOME_EXATO), None)
    if not card: continue
    t = stamp(f)
    offers = sorted([s for s in card['stores'] if s.get('priceKnown') and s.get('known')],
                    key=lambda s: s['priceBRL'])
    last = offers
    cur = {(s['storeId'], s['condition'], s['language']): (i+1, s['priceBRL'], s['quantity'])
           for i, s in enumerate(offers)}
    if prev:
        gap = (t - prevT).total_seconds() / 3600
        if gap <= 24:
            for k, (rank, price, q) in cur.items():
                p = prev.get(k)
                if not p: continue
                hours[bucket(p[0])] += gap
                if q < p[2] and q > 0: sold[bucket(p[0])] += p[2] - q
    prev, prevT = cur, t
vel = [sold[i]/hours[i]*24 if hours[i] else 0 for i in range(len(EDGES))]
```

**Pondere pelo tempo decorrido de cada transição, não conte "quantos diffs".**
As capturas não são regulares: em `tracking-pkm/SEALED` a mediana é 6h mas a
média é 16h, com buracos de até 144h. Tratar um buraco de 144h como se fosse
12h infla o giro de forma absurda — e o erro não é uniforme entre buckets, então
ele distorce a comparação, não só a escala.

Descarte transições com `gap > 24h`: numa janela longa a queda de quantidade
fica limitada pelo estoque da loja e a taxa vira mentira nos dois sentidos.

Isso importa de verdade: recalcular ME5 assim mudou "39 caixas em 14 dias" para
**27 dias**, e o blister quádruplo de 15 dias para **7**.

Depois **suavize** com regressão isotônica decrescente (PAVA). Sem isso o
otimizador escolhe picos de ruído — em ME5 o blister triplo aparecia com giro
maior na 98ª que na 67ª posição.

### Armadilhas que já custaram análise errada

- **Case pelo nome EXATO, nunca por substring.** `"Coleção Treinador Avançado -
  Megaevolução 5"` casa com 39 produtos, incluindo `- Pokémon Center` e as
  variantes `(ING)`/`(PT-BR)`. Descobri isso depois de recomendar um preço
  baseado na curva do produto errado. Ache primeiro o nome exato pelo card que
  tem oferta da Collecta, e use igualdade daí em diante.
- **Filtre queda-para-zero.** Some anúncio removido, não venda. Só conta
  `q < anterior and q > 0`.
- **Não extrapole a cauda.** Aplicar o giro do bucket "101-150" a um preço no
  topo da faixa gera recomendações absurdas (a conta "cuspiu" vender ETB a
  R$ 999). **Trave o teto de preço candidato na mediana do mercado** — acima dela
  você depende de comprador que não compara, o que não é estratégia de giro.
- **Confira `extras` antes de confiar em qualquer posição.** Foil e Normal são
  produtos diferentes e convivem na mesma página. Snapshots capturados antes de
  2026-08-05 **não gravam o campo** — neles o ranking mistura os dois e a posição
  é ficção. Em Riftbound isso escondeu algo maior: 128 das nossas singles estavam
  *cadastradas* como Normal sendo Foil, sozinhas num balde sem compradores. Dois
  sinais de alerta: nossa oferta é a única de um tipo entre 20+ do outro, e o
  "Preço Médio de Venda" daquele tipo mostra o nosso próprio preço nas três
  colunas (= a Liga não tem venda nenhuma registrada ali).
- **Baixa concorrência pode ser baixa demanda.** Sets com 3–5 lojas por carta
  pareciam ouro; eram japoneses (91–97% das ofertas em idioma `6`). Cheque o mix
  de idioma antes de chamar de oportunidade: PT=`8`, EN=`2`, JP=`6`, PT/EN=`11`.
- **Separe o que é robusto do que não é.** As *razões* entre buckets (a
  elasticidade, qual posição gira mais) aguentam bem — sobrevivem a mudança de
  bucketização, suavização e ponderação. Já o *prazo em dias* é frágil: depende
  da ponderação por tempo e some se as capturas falharem. Recomende preço com
  confiança; dê prazo como ordem de grandeza.

---

## Fase 2 — A fronteira preço × prazo × lucro

Para cada preço candidato (use os preços reais dos concorrentes, menos R$ 0,01):

```
rank      = nº de concorrentes mais baratos + 1
liquido   = preço × (1 − taxa) − 0,40
margem    = liquido − custo
un/dia    = vel[bucket(rank)]
dias      = qty / (un/dia)
lucro     = margem × qty
ROI/dia   = (margem / custo) × un/dia
```

Apresente **três cenários**, nunca um número só:

| Cenário | Critério |
| --- | --- |
| Liquidar | preço mais rápido que ainda não dá prejuízo |
| Equilíbrio | maior lucro/dia com teto na mediana |
| Segurar | maior lucro total, aceitando o prazo |

E calcule o **custo da pressa** por SKU — quanto de retorno mensal você abre mão
liquidando: `(Δlucro / capital) / (Δdias / 30)`. Foi o número que decidiu tudo em
ME5: liquidar a caixa custava 6,4%/mês (valia), liquidar o blister triplo custava
72%/mês (não valia). **Liquidação é decisão por SKU, não por lote.**

Quando o piso do mercado estiver no seu custo de atacado, diga com todas as
letras que não dá para brigar por preço e ofereça as saídas de fora da Liga
(lote B2B, vitrine própria, WhatsApp) — sem taxa, sem comparador, dinheiro na
hora.

---

## Fase 3 — Conferir ao vivo antes de gravar

O snapshot tem até 6h de atraso e a análise inteira depende de posição. Abra a
página pública do produto (`?view=prod/view&pcode=<pcode>`, o `pcode` está em
`cards[].url` do snapshot) e leia as primeiras ~10 ofertas.

Confira três coisas:

1. **A posição-alvo ainda vale** naquele preço.
2. **Onde estão as paredes de estoque.** Preferir ficar 2 centavos *abaixo* de
   quem tem muita unidade a ficar 1 centavo acima. Em ME5 isso rendeu três
   ajustes que não custaram praticamente nada: ETB R$ 339,89 → R$ 339,87 (passar
   à frente de 141 un da CollectHub), blister triplo R$ 26,60 → R$ 26,58 (34 un
   da Cop Cards), combo para o 5º em vez do 3º (mesma velocidade, +38% de lucro).
3. **Quantas das ofertas à sua frente são `Pre Order`.** O snapshot **não grava
   esse campo** — para o tracking um pré-venda é uma oferta normal, então o
   "piso do mercado" que você calculou pode não existir em pronta entrega.

> **Pre Order é o furo conhecido deste método.** Em Riftbound (Caixa Vendetta) as
> 4 ofertas mais baratas eram pré-venda: o piso real de pronta entrega era
> R$ 699,90, não R$ 689,99. Em One Piece (ST-36) 8 das 10 primeiras eram
> pré-venda. Consequências:
> - O preço-alvo certo é **1 centavo abaixo da primeira pronta entrega**, não do
>   menor preço absoluto.
> - A curva de giro **mistura os dois** e portanto subestima quem é o primeiro em
>   pronta entrega. Trate o giro do seu bucket como piso, não como estimativa.
> - Muita pré-venda barata é aviso de que o preço **vai cair** quando o estoque
>   chegar. Segurar item nessa situação é apostar contra o fluxo.

Ritmo: `ligapokemon.com.br` é Cloudflare-challenged. **7–10s entre navegações,
no máximo 2 páginas por `browser_batch`**; ao primeiro sinal de bloqueio, pare e
avise — o ban de IP derruba o acesso do dono também.

---

## Fase 4 — Gravar: dashboard primeiro, Liga depois

**Dashboard** (`?tab=estoque`, seletor de jogo na sidebar) é a fonte da verdade e
alimenta a vitrine pública. Campo `Preço R$ (un.)`, decimal com **vírgula**.
Confirme pelo total da linha (`preço × qtd`) e salve em `Salvar alterações` — o
botão vira `Salvo` e a coluna Liga passa a `Reimportar`.

Itens do ledger nem sempre têm o nome da Liga: o ETB é
`Mega Evolution - Pitch Black - Elite Trainer Box` e o combo é
`Booster Bundle`. Busque por termo curto e confira o `id` contra
`data/liga-skus.json`.

**Liga** — `?view=ecom/admin/prod/all`, `Meu Cadastro`, busca por termo no campo
Produto. Case as linhas por **`h_sid`**, nunca por índice.

```js
const alvo = {'360225':'308,49', '360030':'339,87'};
document.querySelectorAll('[name^="h_sid"]').forEach(el => {
  const i = el.name.match(/\[(\d+)\]/)?.[1];
  const novo = alvo[el.value];
  if (!i || !novo) return;
  const pr = document.querySelector(`[name="txt_preco[${i}]"]`);
  pr.focus(); pr.value = novo;
  pr.dispatchEvent(new Event('input', {bubbles:true}));
  pr.dispatchEvent(new Event('change', {bubbles:true}));
  pr.blur();
});
```

Regras não-negociáveis desta tela:

- **Só `txt_preco`.** Não encoste em `txt_qty` / `txt_qty_typed` — escrever no
  hidden fez o estoque **somar** (30→60→90) em vez de setar.
- Salve com `form.requestSubmit(btSalvar)`. Clique por coordenada falha em
  silêncio neste admin.
- **Verifique recarregando**, nunca lendo o form em memória. Espere
  *"Produtos de sua lista alterados com sucesso"* **e** confira preço e
  quantidade nas linhas depois do reload.
- Preço tem máscara: vírgula para centavos, nunca ponto.

---

## Fase 5 — O que sempre reportar

- Tabela antes/depois com posição e margem, e o lucro esperado total.
- **Todo ajuste que você fez em cima do plano** e por quê (parede de estoque,
  mercado que andou desde o snapshot).
- Divergências entre Liga e dashboard. Em ME5 apareceram 3 anúncios de "Kit"
  (`Kit com 4 Caixa de Booster`, `Kit com 12 Blister…`) que **não existiam no
  ledger** — prometiam 40 caixas além das 39 avulsas e, depois da reprecificação,
  ficaram mais caros por unidade que o próprio avulso da loja. Kit e avulso do
  mesmo produto precisam ser reprecificados juntos, e é preciso perguntar ao dono
  se dividem estoque físico.
- As ressalvas do método (proxy de venda, prazo de repasse) — sem elas o dono
  toma decisão de capital com confiança que o dado não sustenta.

---

## Reprecificar é rotina, não evento

Sua posição cai sozinha conforme concorrentes entram. Preço definido hoje
degrada em duas semanas. Rode esta skill de novo a cada captura relevante do
tracking, ou pelo menos semanalmente nos SKUs elásticos (ticket alto), e
compare o rank atual com o alvo.
