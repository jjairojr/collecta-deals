---
name: liga-singles-onepiece
description: Cadastra as singles de One Piece da Collecta na loja LigaMagic (tcg=11) via importação de CSV, resolvendo variantes pelo sufixo do número e casando cada carta por NOME antes de gravar; promos ficam fora por regra. Use quando o pedido for publicar, atualizar ou corrigir cartas avulsas de One Piece na Liga.
allowed-tools: Bash, Read, Write, Edit, mcp__claude-in-chrome__*
---

# Cadastrar singles de One Piece na LigaMagic

Loja `id=866280`, `tcg=11`. É o jogo **mais complexo dos três**: 86 edições,
numeração com sufixo de variante, promos numa edição separada e nomes de estoque
cheios de ruído.

Executado em **2026-07-30**: 167 de 181 cards, 365 unidades, R$ 21.498,10.

Pokémon → `liga-singles-pokemon` · Riftbound → `liga-singles-riftbound`
(o fluxo de importação é o mesmo daquela; aqui documento só o que difere).

---

## 🚫 REGRA DURA: não cadastre promos

**Decisão do dono, 2026-07-31: cartas promocionais ficam FORA da loja.** Não
tente resolver, não pergunte se dá pra encaixar — pule e reporte.

Está fora quem cair em **qualquer** um destes:

- edição **`2` (PC-01, One Piece Promotion Cards)** — 1203 cards, todos promo;
- número com sufixo de pacote promocional:
  `WP` `EP` `RP` `GC` `PP` `TC` `BS` `AP` `BP` `3A` `3W` `CP` `TTC`
  `OC` `OF` `OP` `RC` `SN` `DP`;
- nome do estoque com marcador de pacote: *Winner Pack, Event Pack, Regional
  Participation Pack, Offline/Online Regional, Championship, Gift Collection,
  Promotion Pack, Treasure Cup, Premium Card Collection, Anniversary
  Tournament/Treasure/Stamped, Battle Pack, Celebration Pack, Welcome Pack,
  Dash Pack, Double Pack Set, Serial Number*.

### Por que — não é preciosismo, custou dinheiro

Promo é onde tudo dá errado ao mesmo tempo:

1. **O nome é quase idêntico ao da carta comum.** "Shanks", "Monkey.D.Luffy",
   "Nico Robin" existem em dezenas de versões espalhadas por set, variante e
   pacote. A diferença entre elas some num nome truncado.
2. **Os preços são de ordens de grandeza diferentes.** Cadastrar a versão errada
   não gera um erro visível — gera um anúncio plausível com o preço errado. No
   estoque de hoje, `OP02-036` Nami **Parallel** está a R$ 430 e divide número e
   nome com a versão comum, que é uma carta de poucos reais.
3. **A referência de mercado quebra justamente nelas.** O catálogo casa o promo
   com a versão comum barata, então `marketUSD` vira centavos enquanto a carta
   vale dezenas de dólares — e qualquer preço sugerido a partir daí sai absurdo.

Em **2026-07-30** os dois únicos itens que foram para a loja com preço
destruído eram promos, e ambos pela razão 3:

```
P-106-WP    Monkey.D.Luffy (Winner Pack 2026 Vol. 2)
            custo R$ 329,99 · ref US$ 97,99 · mercado casado US$ 0,45
            -> anunciado a R$ 3,00 e VENDIDO (pedido #11542408)
EB02-055-WP Jinbe (Winner Pack 2025 Vol. 3)
            custo R$ 9,00 · ref US$ 7,95 · mercado casado US$ 0,08
            -> anunciado a R$ 0,29
```

Nenhuma carta não-promo apresentou esse defeito, nos três jogos.

### O que fazer com elas

Filtre **antes** de montar o CSV e liste no relatório final como
`PULADO: promo`, com nome e preço. O dono decide caso a caso se vale cadastrar
alguma na mão — aí ele confere o preço olhando, que é o único jeito seguro.

> **Pendência:** 32 promos (46 un., R$ 2.644,97) entraram na loja em 2026-07-30,
> antes desta regra existir — 26 da PC-01 e 6 com sufixo promocional em OP-14/OP-15.
> Precisam ser removidos (zerar quantidade ou importar em "Subtrair do Estoque").

---

## ⚠️ Cuidado redobrado com nomes parecidos

Vale para todo One Piece, promo ou não. O jogo tem o mesmo personagem em muitas
versões, e as baratas são **maioria** — então errar tende a cadastrar a cara como
barata, ou pior, vender a cara pelo preço da barata.

- **Nunca conclua por nome sozinho.** `Monkey.D.Luffy` casa com dezenas de linhas.
  O número é o identificador; o nome é a *confirmação*.
- **Nunca conclua por número sem sufixo.** `OP02-036` sozinho é a Nami comum;
  `OP02-036-PAR` é a Parallel de R$430. Um sufixo ignorado troca uma pela outra.
- **Nos sets antigos o nome da Liga não marca a variante:** `OP02-013-PAR` se
  chama só "Portgas.D.Ace", igual à base. Ali só o sufixo desempata — e é
  justamente onde o erro passa despercebido.
- **Desconfie de dois candidatos com o mesmo nome.** É o padrão base-vs-variante.
  Se o estoque não marca a variante, é a base; se marca, exija o sufixo
  correspondente. Sobrou dúvida, **não cadastre**.
- **Confira o preço contra o custo antes de publicar.** Preço abaixo do custo ou
  a uma fração da referência é sinal de carta trocada, não de promoção.

---

## ⚠️ Antes de importar: veja o que JÁ está cadastrado

**One Piece é o único jogo que já tinha cards na loja.** A operação
"Acrescentar ao Estoque" **soma** à quantidade existente — na primeira execução
35 cards vieram como `Atualização` e ficaram com o dobro (Perona 4 → 8,
Yamato 9 → 18).

Foi preciso um segundo import em **"Subtrair do Estoque"** com a quantidade
pré-existente (`Quant Total − Quant Somada`) para voltar ao valor do ledger.

Antes de importar, faça um dos dois:

- exporte a edição com `txt_edicao_11 = -1` (**"Todas (Somente do Estoque)"**) e
  veja o que já tem quantidade; ou
- importe e **leia a coluna Tipo Atualização** do resultado: toda linha
  `Atualização` já existia. Guarde `linha:somada:total` de cada uma — é o que
  permite montar o arquivo de subtração.

A tabela de resultado traz a **coluna Linha**, que é o número da linha no seu
arquivo (linha 3 = primeira linha de dados). Use isso para mapear de volta.

---

## Fase 0 — Estoque de produção

```bash
B=https://collecta-deals-production.up.railway.app
curl -s --max-time 30 "$B/api/trades?game=onepiece" > /tmp/op.json
```

`status == "holding"`, `qty > 0`, `askBRL > 0`, sem `kind`.

Campos: `number` (`OP12-034`), `set` (`OP12`), `variant`, `condition`, `name`.

⚠️ **O `set` do ledger é inconsistente**: convivem `OP12` e `OP-14`, `P` e `PR`.
Normalize removendo hífens antes de mapear. **O `set` também mente às vezes** —
reprints de PRB/PRB2 vêm com o número do set original.

⚠️ **Idioma não é guardado.** Confirmado com o dono em 2026-07-30: tudo inglês
(`EN`). Reconfirme para estoque de origem nova.

---

## Fase 1 — Edições

86 no `select[name=txt_edicao]` (`?view=ecom/admin/cartas/all&tcg=11`). Mapa em uso:

| ledger | ed | ledger | ed | ledger | ed |
| --- | --- | --- | --- | --- | --- |
| OP01 | 6 | OP07 | 34 | OP13 | 70 |
| OP02 | 3 | OP08 | 38 | OP14 | 73 |
| OP03 | 5 | OP09 | 47 | OP15 | 77 |
| OP04 | 1 | OP10 | 49 | OP16 | 81 |
| OP05 | 20 | OP11 | 53 | EB01 | 32 |
| OP06 | 24 | OP12 | 61 | EB02 | 52 |
| EB03 | 74 | EB04 | **78 (vazia)** | PRB | 46 |
| PRB2 | 65 | ST01 | 7 | ST12 | 31 |
| ST15 | 40 | ST21 | 51 | ST23 | 54 |
| ST29 | 76 | ST30 | 82 | P / PR | **2 (PC-01)** |

**`2` = One Piece Promotion Cards (PC-01), 1203 cards** — onde moram as versões
de pacote promocional de cards de *qualquer* set (uma carta com
`(Winner Pack 2026 Vol. 1)` no nome tem número `OP12-093` mas mora na PC-01 como
`OP12-093-WP`). **Exporte esta edição, mas só para reconhecer promos e pulá-los**
— nada dela entra no CSV de importação. Ver a regra dura no topo.

⚠️ **EB04 (`78`) exporta vazia** — a edição existe no dropdown mas não tem
catálogo na Liga. As 5 cartas de EB04 do estoque não têm como ser importadas.

---

## Fase 2 — Numeração e variantes

Formato: `<SET>-<NNN>[-<SUFIXO>]`, batendo com o `number` do ledger. **A variante
é o sufixo**, e cada variante é uma **linha própria** no catálogo.

| sufixo | significado |
| --- | --- |
| `AA` | Alternate Art |
| `PA` / `PAR` | Parallel (sets novos usam `PA`, antigos `PAR`) |
| `SP` | Special / SP |
| `MA` | Manga |
| `DP` | Dash Pack / Double Pack Set |
| `TR` | Treasure |
| `G` / `S` | Gold / Silver |
| `RE` | Reprint (PRB/PRB2) |
| `JR` / `TF` / `PF` | Jolly Roger / Textured / Pirate Foil (PRB/PRB2) |
| `WP` `EP` `RP` `GC` `PP` `TC` `BS` `3A` `3W` `CP` `TTC` `OC` `OF` `OP` `RC` | pacotes promo (Winner/Event/Regional/Gift/Promotion/Treasure Cup/Best Selection/3rd Anniversary…) |
| `SN` | Serial Number |

⚠️ **Nos sets antigos o nome da Liga NÃO marca a variante.** `OP10-001-PA` se
chama "Smoker (001) (Parallel)", mas `OP02-013-PAR` se chama só
"Portgas.D.Ace" — igual à carta base. Nesses casos **só o sufixo desempata**,
o nome não.

⚠️ **PRB/PRB2 mantêm o número do set original**: `Shanks (Reprint)` do PRB2 é
`OP06-007-RE`, não `PRB2-xxx`. Mas PRB2 também tem numeração própria
(`PRB02-004`).

---

## Fase 3 — O casamento (é aqui que mora o risco)

Para cada item, nesta ordem:

0. **Filtro promo.** O nome do estoque tem marcador de pacote promocional?
   → `PULADO: promo`, não entra no CSV. Isto vem **antes** de qualquer busca,
   porque o objetivo não é resolvê-los.
1. **Candidatos** = linhas cujo número é exatamente `num` **ou** começa com
   `num + '-'`, na edição do set. **Descarte candidatos promo** (edição `2` ou
   sufixo da lista) — eles só servem para reconhecer que o item é promo e
   registrar o `PULADO`, nunca para casar.
   Sem candidato → `NUMERO INEXISTENTE`, reporta.
2. **Nome exato**: canonicalize os dois lados (tire acento, tire os números de
   carta embutidos no nome tipo `(094)` ou `- OP07-015`, tire pontuação) e
   compare. **Exatamente um match → aceita.**
3. **Desempate por sufixo**, quando o nome não resolve:
   - estoque marca `Parallel` → sufixo `PA` ou `PAR`
   - marca `Alternate Art` → sufixo `AA`
   - **não marca variante nenhuma → sufixo vazio** (a carta base)
4. Sobrou mais de um, ou nenhum → **`AMBIGUO` / `NOME NAO BATE`, não cadastra.**
   Reporte com a lista de candidatos e deixe o dono decidir.

Nunca afrouxe o passo 4 para "fechar" o número. Cadastrar errado numa loja real
é pior que não cadastrar — ver o incidente das Celebrações em
`liga-singles-pokemon`.

**Antes de gerar o CSV, rode um sanity check de preço**: para cada linha, compare
`askBRL` com o custo e com a referência do item. Preço abaixo de metade do custo,
ou abaixo de um quarto da referência, é forte indício de carta trocada — segure a
linha e pergunte.

---

## Fase 4 — Exportar o catálogo

`?view=ecom/admin/export`, radio `txt_tipo_export=11`, select `txt_edicao_11`,
`form.requestSubmit(input[name=btExport])`. **Baixa sem recarregar**, então dá
para disparar várias em sequência:

```js
const eds = ['6','3','5','1','20'];
const f = document.forms['export'];
const s = f.querySelector('[name=txt_edicao_11]');
const btn = document.querySelector('input[name=btExport]');
eds.forEach((v,i) => setTimeout(() => {
  s.value = v; s.dispatchEvent(new Event('change',{bubbles:true}));
  f.requestSubmit(btn);
}, i * 5000));
```

⚠️ **O Chrome bloqueia downloads automáticos em rajada.** Com 3s de intervalo
metade se perde **em silêncio**; com **5s** passa quase tudo. Sempre confira
quais edições chegaram (leia `rows[2][1]` de cada arquivo) e **redispare as que
faltaram** — não presuma que baixou.

Baixar arquivo exige **permissão explícita do dono**. São ~30 arquivos.

---

## Fase 5 — Importar

⚠️ **Um arquivo só para todas as edições.** Cada linha carrega `Edição ID` e
`Carta ID`, então o import aceita as 28 edições juntas — 167 linhas, 22kb, bem
dentro dos limites (1000 linhas / 500kb). Isso evita 28 rodadas de upload.

Tela `?view=ecom/admin/import&tcg=11`. Fluxo e pegadinhas em
`liga-singles-riftbound` (Fase 4), mais estas duas:

- **Marque o rádio com `label.click()` no DOM**, não por coordenada. As
  coordenadas do screenshot e as do CSS têm **escala diferente** (viewport 1920
  → screenshot 1564 = fator 0,815), e cliques por coordenada erram o alvo em
  silêncio. `label.click()` dispara o handler do site e funciona sempre:

  ```js
  const r = document.forms['home'].querySelector('input[name="inpt-stock-action"][value="1"]');
  (r.closest('label') || r.parentElement).click();   // value 1 = acrescentar, 2 = subtrair
  ```
- Antes de submeter confira: `f.file.files.length`, `f.file_substract.files.length`
  (só um dos dois preenchido) e `txt_price_ignore_import.checked === false`.

### Desfazer uma soma indevida

Monte o CSV com as mesmas linhas e `QtdSomar = Total − Somada`, marque
**Subtrair do Estoque** (`value=2`) e suba em `file_substract`. O resultado deve
mostrar `Quant Total` igual à quantidade do ledger.

---

## Fase 6 — Conferir

`?view=ecom/itens&id=866280&tcg=11` — o contador `1-30 de N` tem que bater com o
número de cards casados. Confira preço e quantidade por amostragem (a grade fica
em iframe; use screenshot).

---

## Pendências conhecidas (2026-07-30) — 14 cards fora

| motivo | cards |
| --- | --- |
| EB04 sem catálogo na Liga | `EB04-012` Kikunojo · `EB04-022-AA` Issho · `EB04-048` Rob Lucci · `EB04-052` Sanji · `EB04-061` Monkey.D.Luffy (só existe `-SN` Serial Number) |
| sem número no ledger | `DON!! Card (Dracule Mihawk)` (OP14) · `DON!! Card (Luffy)(Gold)` (OP15) |
| variante ambígua | `OP02-018` Marco (Alternate Art) — candidatos base/`-E`/`-PP`/`-TC`, nenhum "Alternate Art" · `OP02-036` Nami (Parallel) — base/`-E`/`-BS` |
| marcador de produto sem par | `ST21-001` / `ST21-014` / `ST21-017` "(Luffy Deck)" — base vs `-PA` vs `-BS` |
| promo (agora regra) | `OP07-053` Ace 3rd Anniversary · `P-078` Adio Regional — sairiam pelo filtro de promo de qualquer forma |

Também ficaram **2 singles sem preço** no Estoque.

E os **32 promos cadastrados antes da regra** (46 un., R$ 2.644,97) seguem na
loja — remover é a primeira coisa a fazer numa próxima passada.

---

## Ao terminar

Relatório com: cards casados por edição, unidades, valor, quantas vieram como
`Atualização` (e se a subtração foi aplicada), o que não casou e por quê.

**Marque a flag no dashboard** para os cards que entraram — é como a aba Estoque
mostra o que já está na loja e evita a reimportação cega que dobra estoque:

```
POST $B/api/trades/liga?game=onepiece
{"items":[{"id":"<ledgerId>","ligaListed":true,"ligaQty":<qtd>,"ligaPriceBRL":<preco>}]}
```

Mande `ligaQty` e `ligaPriceBRL` **iguais ao que foi enviado à Liga**, não ao que
está no ledger agora: é a diferença entre os dois que faz a linha aparecer como
"Reimportar" depois. Conferência posterior: skill `liga-conferir`.
