---
name: liga-conferir
description: Confere o que está cadastrado na loja LigaMagic contra o estoque do dashboard da Collecta, item a item, e atualiza a flag "Na Liga". Use quando o pedido for auditar, conferir ou sincronizar o que está na Liga versus o dashboard.
allowed-tools: Bash, Read, Write, Edit, mcp__claude-in-chrome__*
---

# Conferir dashboard × LigaMagic

Compara o estoque da Collecta com o que está de fato publicado na loja
`id=866280` e diz, por item: **está na Liga**, **falta cadastrar**, ou
**cadastrado mas desatualizado** (quantidade ou preço divergem).

O resultado alimenta a flag da aba **Estoque** (`POST /api/trades/liga`), que é
onde o dono enxerga isso no dia a dia.

Cadastrar é outra coisa — use `liga-singles-onepiece`, `liga-singles-riftbound`,
`liga-singles-pokemon` ou `/liga-cadastrar selados`.

---

## Por que esta conferência existe

O import da Liga **soma** quantidade em vez de definir. Reimportar "tudo por
garantia" dobra o estoque do que já estava lá — aconteceu com 35 cards de One
Piece em 2026-07-30. Então antes de qualquer reimportação é obrigatório saber
**exatamente** o que já está publicado.

---

## Fase 0 — Os dois lados

**Dashboard (verdade do estoque)** — sempre produção, nunca `data/*.json` local:

```bash
B=https://collecta-deals-production.up.railway.app
curl -s --max-time 30 "$B/api/trades?game=onepiece" > /tmp/dash-op.json
```

Conta como "deveria estar na Liga" quem tem `status == "holding"`, `qty > 0` e
`askBRL > 0`. Os campos da flag já vêm na resposta:
`ligaListed`, `ligaQty`, `ligaPriceBRL`, `ligaAt`.

**Liga (verdade da loja)** — exporte com `txt_edicao_<tcg> = -1`
(**"Todas (Somente do Estoque)"**) em `?view=ecom/admin/export`. Isso devolve
**só o que tem estoque cadastrado**, que é exatamente a lista que interessa e
evita baixar o catálogo inteiro.

| jogo | `game=` | tcg | radio `txt_tipo_export` | select |
| --- | --- | --- | --- | --- |
| One Piece | `onepiece` | 11 | `11` | `txt_edicao_11` |
| Pokémon | `pokemon` | 2 | `2` | `txt_edicao_2` |
| Riftbound | `riftbound` | 19 | `19` | `txt_edicao_19` |

```js
const f = document.forms['export'];
const r = f.querySelector('input[name=txt_tipo_export][value="11"]');
r.checked = true; r.dispatchEvent(new Event('change',{bubbles:true}));
const s = f.querySelector('[name=txt_edicao_11]');
s.value = '-1'; s.dispatchEvent(new Event('change',{bubbles:true}));
f.requestSubmit(document.querySelector('input[name=btExport]'));
```

O arquivo cai em `~/Downloads` como `<tcg>_n_<hash>.csv`. Colunas relevantes:
`1` EdiçãoID · `2` Sigla · `3` CartaID · `4` Número · `8/9` Nome PT/EN ·
`10` Idioma · `11` Qualidade · **`12` Quantidade Existente** · `14` Preço.

⚠️ Sempre confira qual edição chegou (`rows[2][1]`) — o Chrome bloqueia
downloads em rajada e some com arquivos **em silêncio**.

---

## Fase 1 — Casar os dois lados

O casamento é o mesmo das skills de cadastro, e a regra não muda:
**nunca conclua por número sozinho, confira o nome.** Número e contagem colidem
entre edições — foi assim que Kyogre foi publicado no lugar de Venusaur.

Atalho que evita re-casar tudo: se a flag já estiver preenchida, `ligaQty` e
`ligaPriceBRL` dizem o que foi enviado. A conferência então vira uma comparação
de três colunas:

| dashboard | Liga | veredito |
| --- | --- | --- |
| `ligaListed` falso, tem preço | ausente do export | **falta cadastrar** |
| `ligaListed` verdadeiro | presente, mesma qtd e preço | **ok** |
| `ligaListed` verdadeiro | presente, qtd/preço diferentes | **desatualizado** |
| `ligaListed` verdadeiro | **ausente do export** | **flag mentindo** — vendeu na Liga ou o cadastro sumiu |
| `ligaListed` falso | presente no export | **flag mentindo** — cadastrado sem marcar |

As duas últimas linhas são o motivo de a conferência existir: a flag é uma
anotação, não uma leitura ao vivo. **Uma venda na Liga não volta para o
dashboard**, então item vendido lá aparece aqui.

---

## Fase 2 — Relatório

Sempre por jogo, com totais e a lista dos divergentes:

```
=== ONE PIECE
  no dashboard com preço : 181
  na Liga (export)       : 167
  ok                     : 165
  falta cadastrar        :  14   <- lista
  desatualizado          :   2   <- lista com de/para de qtd e preço
  flag mentindo          :   0
```

Para "desatualizado", mostre sempre **de → para** de quantidade e preço: é o que
o dono precisa para decidir entre acrescentar a diferença ou subtrair.

---

## Fase 3 — Atualizar a flag

```
POST $B/api/trades/liga?game=<id>
{"items":[{"id":"<ledgerId>","ligaListed":true,"ligaQty":4,"ligaPriceBRL":340.0}]}
```

Escreve só as colunas da Liga — nunca toca em preço de venda nem em vitrine
(quem faz isso é `/api/trades/listings`). Mandar `ligaListed:false` limpa
`ligaQty`, `ligaPriceBRL` e `ligaAt`.

Só mande os itens cujo veredito **mudou**. Reenviar um item já em dia atualiza
`ligaQty`/`ligaPriceBRL` para os valores atuais e **apaga silenciosamente** um
aviso de "desatualizado" sem que nada tenha sido enviado à loja.

Confira depois em `$B/?tab=estoque` — coluna **Liga**: verde `Na Liga`, âmbar
`Reimportar`, cinza `Fora`. O botão **Faltam na Liga** filtra os dois últimos.

---

## Conferir preço contra o mercado da Liga

O marketplace por jogo (`ligaonepiece.com.br`, `ligapokemon.com.br`…) tem a
página da carta:

```
?view=cards/card&card=<Nome> (<NUM>)&ed=<SIGLA>&num=<NUM>
```

⚠️ **Duas fontes de preço na mesma página, e elas não são a mesma coisa:**

| bloco | o que é | serve para |
| --- | --- | --- |
| **Preço Médio de Venda no Marketplace** | histórico de **vendas** (mín / méd / máx), separado por Normal e Foil | tendência |
| **Lojas Vendendo** | o que os concorrentes pedem **agora** | **decidir preço** |

Use o **Lojas Vendendo**. O bloco de média já me fez errar feio em 2026-07-31:
para o Morley `OP12-093-WP` ele mostrava `Normal R$ 22,65`, e eu reportei isso
como piso — só que **ninguém vende esse card como Normal** (as 30 ofertas são
Foil, a partir de R$5,00), e aquele 22,65 vinha de uma venda antiga e isolada.
O mesmo erro inverteu o veredito de Franky, Zoro 094, Rayleigh, Arlong e Kaku.

Ao comparar:

- **Case a condição.** Nosso estoque é NM; a oferta mais barata da lista costuma
  ser D/MP/SP e não é concorrente. Compare com o **NM mais barato**.
- **Case a variante.** Muitos promos só existem em Foil; comparar com a linha
  "Normal" inventa um preço que não existe.
- **Lista vazia = sem mercado**, não "estamos caros" (Kaku `OP07-080-CP` não tem
  nenhum vendedor).

⚠️ Os preços do Lojas Vendendo são **ofuscados por sprite** — no DOM vem `R$`
sem dígito. Só dá para ler por **screenshot**. Não tente extrair por texto: você
vai achar que não há preço, ou vai pegar o do bloco errado.

## Corrigir uma divergência

- **Falta cadastrar** → skill de cadastro do jogo.
- **Desatualizado, quantidade subiu** → importe em "Acrescentar ao Estoque"
  **só a diferença** (o import soma).
- **Desatualizado, quantidade caiu** → "Subtrair do Estoque" com a diferença.
- **Só o preço mudou** → o import atualiza o preço; mande `QtdSomar = 0`… mas
  ⚠️ a Liga **só processa linhas com quantidade E preço**, então preço sozinho
  não passa por CSV: ajuste na tela (`?view=ecom/admin/cartas/all&tcg=<id>`,
  busque a carta, edite `txt_preco`, Salvar).
- **Flag mentindo, sumiu da Liga** → confira os pedidos
  (`?view=ecom/admin/compras`) antes de recadastrar: pode ter vendido, e aí o
  certo é dar baixa no estoque do dashboard, não republicar.

---

## Mecânica do admin

Herda tudo de `liga-singles-riftbound`; o que mais importa aqui:

1. **Ritmo:** 7–10s entre navegações, `wait` de no máximo 10s por ação. Rajada
   gera **Cloudflare Error 1007** e derruba o acesso do dono.
2. **5s entre exports**, senão o Chrome bloqueia downloads em silêncio.
3. Rádio e botões da tela de importação: use `label.click()` no DOM — clique por
   coordenada erra porque screenshot e CSS têm escalas diferentes.
4. A saída do `javascript_tool` é bloqueada se ecoar URL com query string.
5. A loja pública renderiza dentro de iframe: `innerText` não traz os cards, use
   screenshot.

---

## Aprendizados 2026-08-03 (Pokémon, 342 linhas × 147 cadastros)

- **Export sem download**: o mesmo POST via `fetch` in-page das skills de cadastro
  funciona com `txt_edicao_2=-1` (Somente do Estoque) + `btExport` no FormData —
  a resposta é o CSV direto, sem depender do Chrome salvar arquivo. Se vier
  "Just a moment..." (challenge da CF no fetch), navegar de verdade até a tela
  de export resolve o cookie e o fetch seguinte passa.
- **O cabeçalho desse export tem quebra de linha DENTRO de célula** ("Idioma
  (DE, TW…)"). Não use "pular 2 linhas": filtre as linhas de dados por
  `/^"2",/` (tipo do TCG na col 0).
- **Linhas com Quantidade Existente 0 vêm no export** (cadastro zerado =
  vendido/removido). Para a conferência, qty 0 = ausente da loja.
- Casamentos que quebram se ignorados: linhas legadas do dashboard guardam o
  **número no nome** (`(#NN/DD)`) com campo number vazio — extrair por regex;
  siglas divergentes dash→Liga: `SVI→SV1`, `SM01→SUM`, `MCAP→SVP` (e o número
  vira `167b`), `CRZ:GG→CZGG`, `SWSH04→VIV`, `SWSH09→BRS`, `SWSH10→ASR`,
  `SWSH12→SIT`, TGs→`ARTG`/`SITTG`, "(2021)"→CCC, "(2011)"→BLW, "Coleção
  151"→MEW. Nomes PT×EN do mesmo card (Clarita=Gwynn, Defensor Férreo=Iron
  Defender) não são divergência.
- **Import de CSV grande pode pular linha sem avisar** (1 de 75 ficou de fora —
  Solgaleo SUM 155). A conferência pós-import não é opcional.

## Aprendizados 2026-09-13 (3 jogos, 376 flags reescritas)

- Script reutilizável: `conferir.py <game> <tcg>` (ao lado deste SKILL.md) lê
  `exp-<game>.csv` (export "Somente do Estoque", 3 exports em sequência com 6s
  entre `requestSubmit(btExport)` caem em `~/Downloads/<tcg>_n_<hash>.csv`) e
  `trades-<game>.json` de prod, imprime o relatório por veredito e grava
  `flag-updates-<game>.json` pronto para `POST /api/trades/liga`.
- Casamento que funcionou: One Piece pelo número-base (`OP06-081`) ignorando a
  sigla (promos ficam em `PC-01`, EB04 dentro de `OP-14/OP-15`, reprints em
  `PRB/PRB2`), desempate por nome normalizado (o sufixo `-AA/-PA/-RE/-WP…` da
  Liga corresponde ao parêntese do nome no ledger), depois número completo,
  depois preço. Pokémon/Riftbound por (sigla, número, idioma) com fallback
  por número+preço e por nome+idioma+preço (`ask` ou `ligaPriceBRL`); um
  fallback só por nome sem âncora de preço casou Bronzong PBL com ARTG TG11 —
  usar apenas para holding sem preço e nome único.
- Ordenar os holdings com `ligaListed` e preço primeiro: o ledger tem pares
  duplicados (`AR (JP)` sem preço + `Foil JP` precificado) apontando para um
  único anúncio, e sem isso o registro errado leva a flag e o certo vira
  "sumiu". Uma linha da Liga só pode casar com um registro; o segundo vira
  `dash_duplicado` (18 no Pokémon, 4 no Riftbound, 3 no One Piece).
- Pokémon na Liga tem a MESMA carta JP cadastrada duas vezes, na edição JP e
  na EN (Dhelmise `M5 088` + `PBL 091`, Silvally `M5 093` + `PBL 095`, Thievul
  em 4 edições…) — unidades anunciadas que não existem, mesmo padrão do
  OP01-070/-RE. E 131 linhas da Liga sem nenhum registro no ledger, várias
  caras (Charmeleon MEW 169 R$989, Eevee TWM 188 R$849, Lillie's Clefairy ex
  R$649, Deoxys-VSTAR GG46 ×2 R$599, Pikachu GG30 ×3 R$589).
- "Reimportar" depois da conferência é divergência real dash×Liga: no
  Riftbound 99 itens têm preço na Liga menor que o `askBRL` do dashboard
  (reprecificações antigas feitas só na Liga).
