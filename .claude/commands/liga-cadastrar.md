---
description: Cadastra o estoque da Collecta na loja LigaMagic (selados ou singles), a partir dos ledgers locais
argument-hint: selados | singles [--jogo op|pkm|rft] [--dry-run]
allowed-tools: Bash, Read, Edit, Write, mcp__claude-in-chrome__*
---

# Cadastrar estoque na loja LigaMagic

Publica na loja Liga (`id=866280`) o que a Collecta já tem em estoque, usando
**os mesmos ledgers que alimentam a vitrine** — para os dois storefronts nunca
divergirem em preço ou quantidade.

Argumentos: `$ARGUMENTS`
(`selados` ou `singles`; opcional `--jogo op|pkm|rft`; `--dry-run` só planeja)

---

## Fonte da verdade — é PRODUÇÃO, não os arquivos locais

O estoque real vive na **Railway**, não em `data/*.json`. Os arquivos locais
ficam defasados em dias e já me fizeram reportar número errado — **nunca** use
`data/trades*.json` como fonte para este comando.

```
BASE=https://collecta-deals-production.up.railway.app
GET $BASE/api/trades?game=<id>
```

| jogo | `game=` | tcg id na Liga |
| --- | --- | --- |
| One Piece | `onepiece` | 11 |
| Pokémon | `pokemon` | 2 |
| Riftbound | `riftbound` | 19 |
| (fora do escopo hoje) | `lorcana`, `gundam` | — |

⚠️ `game=` inválido **não dá erro**: cai silenciosamente no jogo padrão
(`onepiece`). Se os três jogos vierem com os mesmos dados, o id está errado —
confira em `GET $BASE/api/games`. Os ids são as palavras inteiras
(`pokemon`, não `pkm`).

Um item entra na Liga quando: `status == "holding"` **e** `qty > 0` **e**
`askBRL > 0`. Selados são `kind == "sealed"`; singles são os que não têm `kind`.
Acessórios são `kind == "accessory"` e aparecem na lista de todos os jogos.

Interface humana equivalente: **aba Estoque** em
`$BASE/?tab=estoque` — coluna `Preço R$ (un.)` e o toggle `À venda / Oculta`,
com o seletor de jogo no topo da sidebar. É onde o usuário define preço; este
comando só lê o resultado.

O mapeamento item↔produto-da-Liga fica em **`data/liga-skus.json`**
(`{ ledgerId: { sid, eid, lastQty, lastPrice, updatedAt } }`). É o que torna o
comando idempotente: item já mapeado é **atualizado**, não duplicado. Crie o
arquivo como `{}` se não existir.

---

## Fase 0 — Gate de preço (não pule)

```bash
B=https://collecta-deals-production.up.railway.app
for g in onepiece pokemon riftbound; do
  echo "=== $g"
  curl -s --max-time 25 "$B/api/trades?game=$g" | python3 -c "
import json,sys
d=json.load(sys.stdin)
items=d if isinstance(d,list) else d.get('trades',d.get('items',[]))
s=[t for t in items if t.get('status')=='holding' and t.get('kind')=='sealed' and t.get('qty',0)>0]
for t in s:
    ask=t.get('askBRL') or 0
    print('  {} qty={:<3} custo={:<8} ask={:<8} {}'.format('OK       ' if ask>0 else 'SEM PRECO',
        t.get('qty'), t.get('buyBRL') or '-', ask or '-', (t.get('name') or '')[:48]))
"
done
```

Se algum item aparecer como `SEM PRECO`, **pare e pergunte ao usuário**. Não
invente preço de venda: é decisão comercial dele.

O caminho normal é ele mesmo preencher em `$BASE/?tab=estoque` (coluna
`Preço R$ (un.)`, depois `À venda` e `Salvar`) — aí você relê a API. Se ele
preferir passar uma regra de margem, calcule e **mostre a tabela para aprovação
antes** de qualquer escrita.

Item sem preço fica de fora desta rodada; não é motivo para abortar o resto.
Liste no relatório final o que ficou pendente.

---

## Fase 1 — Casar com a base da Liga (selados)

A Liga **já tem o catálogo de selados pronto**, com nome, foto e categoria. Você
não cria produto: acha o existente e informa estoque/preço.

Tela: `?view=ecom/admin/prod/all`

1. `txt_cad_type` = **`2`** (Base interna do Sistema)
2. `txt_prod_type` = **`1`** (Produto Selado (Card Game))
3. `txt_produto` = termo de busca — use o miolo do nome, não a string inteira
   do ledger. Ex.: `Caixa de Booster - Vendetta` → busque **`Vendetta`**;
   `Mega Evolution - Pitch Black - Elite Trainer Box` → busque **`Pitch Black`**.
4. Submeta com `form.requestSubmit(btBuscar)`.

Cada linha de resultado `[n]` traz:

| campo | uso |
| --- | --- |
| `h_sid[n]` | id do produto na Liga — **grave em `liga-skus.json`** |
| `h_eid[n]` | id do seu cadastro; vazio = ainda não cadastrado |
| `txt_qty_typed[n]` | quantidade |
| `txt_preco[n]` | preço unitário (R$) |
| `txt_idioma[n]` | idioma |
| `txt_qualidade[n]` | condição |
| `label_categ_selected[n]` | categoria já atribuída pela Liga |

**Idioma:** `2` Inglês · `8` Português · `11` Português/Inglês · `6` Japonês ·
`3` Espanhol · `1` Alemão · `4` Francês · `5` Italiano · `7` Coreano.
O prefixo `(ING)` nos nomes do ledger One Piece significa idioma `2`.

**Qualidade:** para selado use sempre **`2` (Lacrado)**.
Outras: `1` Aberto · `3` Novo · `4` Novo com embalagem aberta ·
`5` Novo sem embalagem · `6` Usado · `7` Com defeito/avaria.

Escolha a linha pela **categoria**, não pela posição — uma busca por "Vendetta"
devolve Booster Avulso, Caixa de Booster, Deck Selado e Pacote de Pré-lançamento.
"Caixa de Booster" do ledger ⇒ categoria `Caixas de Boosters`.

### Armadilhas de casamento (todas custaram tempo na 1ª execução)

- **O nome do produto está na célula 6 da linha**, não num `<a>`. Formato:
  `<nome> Categoria: <categoria>` — corte no `"Categoria:"`. Procurar `<a>`
  devolve string vazia e parece que a busca falhou.
- **O filtro `txt_categoria` é ignorado** nesta busca (manda no querystring e
  volta tudo). Filtre por `txt_prod_tcg` + termo em `txt_produto`. Buscar
  Card Game sozinho traz centenas (One Piece selado = 212).
- **⚠️ Idioma: (ING) e (PT-BR) convivem com nome quase idêntico.** A busca
  "Megaevolução 5" devolveu 25 itens, 12 deles PT-BR, com pares tipo
  `(ING) Blister Triplo …` e `(PT-BR) Blister Triplo …`. Casar pelo errado
  cadastra o produto inglês com estoque nacional. **Sempre confira o prefixo
  contra o do ledger** e escolha o `txt_idioma` correspondente.
- **Variantes escondidas depois do truncamento.** Havia dois
  `(ING) Coleção Treinador Avançado - Megaevolução 5 - Escuridão Absoluta`;
  o segundo terminava em `- Pokémon Center`. **Leia o nome inteiro** antes de
  desempatar.
- **A Liga traduz os termos de produto.** Equivalências confirmadas:
  `Starter Deck` → **Deck Inicial** · `Elite Trainer Box` → **Coleção Treinador
  Avançado** · `Booster Bundle` → **Combo de Pacotes** (Pacotes / Fat Packs) ·
  `Booster Box` → **Caixa de Booster**. Buscar o termo em inglês não acha nada.
- Itens do ledger vindos de fonte não-Liga (sem prefixo `(ING)`/`(PT-BR)`, ex.
  `Mega Evolution - Pitch Black - Elite Trainer Box`) **não casam por nome**;
  resolva por set + tipo de produto e **confirme com o usuário**.
- Casar **por `h_sid`**, nunca por índice de linha: os índices não são
  contíguos nem ordenados (a busca do Pokémon usou 10, 14, 15, 18, 20).

Se o produto não existir na base: `Novo Produto` (`?view=ecom/admin/prod/new`).
Aí sim é cadastro manual com foto — e vale checar antes se não é só variação do
termo de busca.

Confirme o casamento com o usuário na primeira execução (mostre
`ledger → sid → categoria`) antes de gravar.

## Fase 1b — Singles ⚠️ fluxo ainda não validado

Tela: `?view=ecom/admin/cartas/all&tcg=<id>` (busca por card, exige ao menos um
campo preenchido). Os três jogos do escopo já estão públicos e em "Somente com
Estoque". **Antes de cadastrar em lote, valide o fluxo com 1 card** e documente
os nomes de campo aqui, do mesmo jeito que a Fase 1 documenta os selados.
Não presuma que os campos são iguais aos de produto.

---

## Fase 2 — Gravar

Preencha por `h_sid` (não por índice) e salve com
`form.requestSubmit(document.querySelector('[name=btSalvar]'))`.

**Salvar em lote funciona**: várias linhas no mesmo submit gravam todas de uma
vez (`Produtos de sua lista alterados com sucesso`). A regra de "um por submit"
vale só para **upload de arquivo**, não para campos. Uma busca → uma gravação de
todas as linhas daquela busca é o caminho eficiente e reduz requisições.

Campos por linha `[n]`: `txt_qty_typed` **e** `txt_qty` (escreva nos dois),
`txt_preco`, `txt_idioma`, `txt_qualidade`.

**O campo de preço tem máscara**: digitar `130` vira `130,00` sozinho. Para
centavos use vírgula (`39,90`), nunca ponto.

Depois do submit a página recarrega mantendo a busca. **Confirme que `h_eid[n]`
deixou de ser vazio** — é o id do seu cadastro. Só então grave em
`liga-skus.json`.

Com `--dry-run`: só monte e mostre a tabela `item → sid → qty → preço → idioma
→ qualidade`, sem submeter nada.

---

## Mecânica do admin da Liga — aprendido na prática, siga à risca

1. **`form.requestSubmit(botao)` é o único submit confiável.** Clique por
   coordenada falha em silêncio com frequência neste admin: a página não
   recarrega, os valores continuam no DOM e *parece* que salvou.
2. **Nunca dispare `MouseEvent('click')` em checkbox.** Clique real alterna o
   estado, então `checked = true` + click volta para `false` e você grava o
   oposto. Faça `el.checked = true` e dispare só `input` e `change`.
3. **Verifique sempre recarregando a página**, nunca lendo o formulário em
   memória. Submit que falhou é indistinguível de sucesso sem recarregar.
4. **Campos espelhados:** alguns selects têm um hidden com o *texto*
   (`txtCategoria`) além do select com o *valor* (`txtCategoriaValue`). O
   handler de change **não** sincroniza os dois — escreva nos dois.
5. **Um arquivo por submit.** Enviar vários uploads juntos aborta o save inteiro
   (o erro sai como falha de favicon, mas descarta os textos também).
6. **A saída do `javascript_tool` é bloqueada se ecoar URL com query string.**
   Retorne só os dados (nomes, ids, contagens), nunca hrefs completos.
7. **Ritmo — isto já causou incidente.** Rajada de navegações no admin gera
   **Cloudflare Error 1007: ban do IP**, e o ban derruba o acesso do usuário
   também, não só o seu. Regras: 7–10s entre navegações, no máximo ~2 páginas
   por `browser_batch`, e **pare e avise** ao primeiro sinal de bloqueio em vez
   de insistir.
8. **`tipo_abertura` é invertido entre telas:** em Banners `1` = mesma janela;
   em Notícias `0` = mesma janela. Confira os `option.value` antes de setar.

---

## Fase 2b — Mover para a categoria do jogo (OBRIGATÓRIO)

O menu de Produtos Selados é **por jogo**, não por tipo. A Liga atribui todo
selado novo a uma categoria de **tipo** (Caixas de Boosters, Blisters…), e essas
39 subs estão **ocultas**. Então todo produto recém-cadastrado cai numa categoria
invisível — **é obrigatório movê-lo**, senão ele não aparece na loja.

| categoria destino | `cat=` |
| --- | --- |
| Produtos Selados → One Piece | `256333` |
| Produtos Selados → Pokémon | `256334` |
| Produtos Selados → Riftbound | `256335` |

Como mover, **um produto por vez**:

1. `?view=ecom/admin/prod/edit&eid=<eid>`
2. `document.querySelector('[name=txt_categoria]').value = '<cat>'` + `change`
3. `form.requestSubmit(botão "Salvar Produto")` — redireciona para `prod/all&ed=1`
4. Confira reabrindo o `prod/edit` (o `select` deve mostrar `.. One Piece` etc.)

**Essa tela é a SUA listagem, não o catálogo compartilhado da Liga.** Confirmado:
ela traz `txt_preco`, `txt_idioma`, `txt_condicao`, `txt_cod_sku` e os campos
fiscais (NCM, ICMS, PIS/COFINS/IPI/CBS). Nome/descrição/fabricante vêm herdados
do catálogo mas são da sua listagem. Mudar só a categoria não altera mais nada —
verificado campo a campo.

⚠️ **O "Aplicar" em lote NÃO move categoria.** Ele só aplica estoque/preço/
idioma/condição. E cuidado: desmarcar "Somente campos sem preenchimento" com os
campos do cabeçalho vazios faz um Salvar **zerar as quantidades das linhas**.
Deixe essa proteção marcada.

Trade-off aceito pelo dono: um produto tem **uma** categoria, então indo pro jogo
ele perde o agrupamento por tipo. Dentro de "One Piece" convivem deck inicial e
booster box sem filtro de tipo.

### Mostrar/ocultar categoria (links GET, não são toggle)

- ocultar: `?view=ecom/admin/categorias&ocultar=<id>`
- exibir: `?view=ecom/admin/categorias&exibir=<id>`

Bater `ocultar=` numa categoria já oculta **não** a reexibe. Na listagem, o ícone
`visibility_hide` = oculta; e quando está oculta o link do ícone vira `exibir=`,
então um regex procurando só `ocultar=` devolve `undefined`.

Só categorias **não-card-game** têm esse ícone. Categorias com Regra = Card Game
(Cartas de One Piece etc.) só têm Editar/Deletar — não há como ocultá-las.

### Regra da categoria decide o que ela lista

`Regra = Card Game` lista **cartas**; `Sem Regra` lista **produtos**. Uma sub de
selado com Card Game renderiza vazia e nem aparece no seletor de categoria de
produto. As três subs de jogo são **Sem Regra** de propósito.

Criar sub com Card Game exige o campo "Modo de exibição"
(`txt_avulsas_exibicao`) — a validação só aparece via `alert()`, então intercepte
`window.alert` para vê-la.

## Fase 3 — Conferir na loja pública

Vá direto na categoria do jogo: `?view=ecom/itens&id=866280&cat=<256333|256334|256335>`.
A página mostra foto, bandeira de idioma, `N un.` e `R$ x,xx` — confira contra a API.

Depois passe o mouse em **Produtos Selados** na loja: deve listar exatamente
One Piece · Pokémon · Riftbound.

As 39 subs de tipo estão ocultas (`254683`–`254721`). Acessórios segue por tipo
(`254674` Sleeves / Shields etc.), esse não foi reestruturado.

⚠️ O `cat=` da URL pública é só o número; no `<select>` da busca do admin o
mesmo item aparece como `254686_10` (com sufixo). Não confunda.

Como os três jogos estão em **"Somente com Estoque"**, item sem estoque na Liga
simplesmente não aparece — página vazia é sintoma de cadastro não gravado, não de
layout quebrado.

Você está logado como dono: categorias marcadas "Exibir somente para
funcionários" continuam visíveis para você. Para conferir a visão do cliente,
peça ao usuário abrir a loja numa janela privada — `fetch` sem credenciais toma
403 do Cloudflare.

---

## Ao terminar

- Atualize `data/liga-skus.json`.
- Relatório: quantos itens, quais deram erro e por quê, e o que ficou pendente.
- Aprendeu algo novo da mecânica do admin? Acrescente na seção acima —
  este arquivo é o repositório desse conhecimento.
