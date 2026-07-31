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
