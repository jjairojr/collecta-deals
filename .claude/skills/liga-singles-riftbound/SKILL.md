---
name: liga-singles-riftbound
description: Cadastra as singles de Riftbound da Collecta na loja LigaMagic (tcg=19) via importação de CSV, casando cada carta por NOME + número antes de gravar. Use quando o pedido for publicar, atualizar ou corrigir cartas avulsas de Riftbound na Liga.
allowed-tools: Bash, Read, Write, Edit, mcp__claude-in-chrome__*
---

# Cadastrar singles de Riftbound na LigaMagic

Loja `id=866280`, `tcg=19`. Publica as cartas avulsas que a Collecta tem em
estoque, lendo **a mesma fonte que alimenta a vitrine**.

Escopo: **só singles de Riftbound**. Pokémon → `liga-singles-pokemon`
(fluxo diferente: lá o casamento é na tela do admin, aqui é por CSV).
Selados → `/liga-cadastrar selados`.

Executado com sucesso em **2026-07-30**: 143 cards, 401 unidades, R$ 26.653,44.

---

## A regra que não se quebra

> **Nunca grave numa linha sem conferir o NOME da carta.**

Número e contagem de set são *estruturais* — colidem entre edições. Só o nome
desempata. Em Pokémon isso já publicou três cartas erradas na loja real (ver
`liga-singles-pokemon`). Aqui o casamento roda offline, então a asserção de nome
é barata: **compare `nomeEN` do catálogo com `name` do estoque e derrube a linha
quando divergir**, antes de gerar o arquivo.

A própria Liga reforça: a tela de importação avisa que *"o Nome e o ID do item
serão validados por segurança"*.

---

## Fase 0 — Ler o estoque de produção

Nunca use `data/trades*.json` local — fica defasado em dias.

```bash
B=https://collecta-deals-production.up.railway.app
curl -s --max-time 30 "$B/api/trades?game=riftbound" > /tmp/rft.json
```

Entra quem tem `status == "holding"` **e** `qty > 0` **e** `askBRL > 0` **e**
nenhum `kind` (`sealed` é selado, `accessory` é acessório).

Ao contrário de Pokémon, o item de Riftbound traz identificadores próprios:

| campo | exemplo | uso |
| --- | --- | --- |
| `name` | `Sett - Brawler (Alternate Art)` | **a chave de validação** |
| `number` | `164A/298` ou `105` | número impresso; pode vir sem denominador |
| `set` | `OGN`, `SFD`, `UNL` | edição — mas **confira**, veja runas Promo |
| `variant` | `Alternate Art`, `Promo`, `Overnumbered` | contexto |
| `condition` | `NM` (ou vazio) | default `NM` |

⚠️ **O ledger não guarda idioma.** Confirmado com o dono em 2026-07-30: as
singles de Riftbound são **todas inglês** (`EN`). Reconfirme se aparecer estoque
de origem nova — chutar idioma publica anúncio errado.

---

## Fase 1 — Edições

Aqui o código do set está **dentro do nome da edição**, então não há a
ambiguidade que existe em Pokémon:

| set do ledger | edição | id |
| --- | --- | --- |
| `OGN` | Origins (OGN) | `1` |
| — | Origins: Proving Grounds (OGS) | `2` |
| — | Riftbound Promotional Cards (OGN-PR) | `3` |
| — | Riftbound Organized Play Promotional Cards (ROPP) | `4` |
| `SFD` | Spiritforged (SFD) | `5` |
| `UNL` | Unleashed (UNL) | `6` |
| `VEN` | Vendetta (VEN) | `7` |

Os mesmos ids valem na busca do admin e no export.

### Numeração

- **Sem zero à esquerda** no catálogo: `3`, `43`, `179`. O ledger manda `007A/298`
  → normalize para `7A` (corte no `/`, tire zeros, **preserve a letra**).
- **Alternate Art é linha separada**, com sufixo `A` no número e `(Alternate Art)`
  no nome. Não é flag: a coluna `Altered Art` do CSV vem vazia nessas linhas.
- **Runas aparecem em várias edições com o mesmo `Carta ID`** (Chaos Rune é
  cid=166 em SFD e em UNL). Cadastrar numa ou noutra dá no mesmo card.
- **Runas `(Promo)` moram em ROPP**, não no set que o ledger diz. Os números
  `R01B`–`R06B` só existem lá (nome na Liga: `Mind Rune (R03b)`). Confirmado com
  o dono antes de cadastrar, porque o nome não bate exatamente.

---

## Fase 2 — Exportar o catálogo

O CSV de importação precisa do **`Carta ID` interno da Liga**, que só vem do
export. Baixar arquivo exige **permissão explícita do dono** — peça antes.

Tela `?view=ecom/admin/export`, form `export`:

1. `input[name=txt_tipo_export][value="19"]` → `.checked = true` + `change`
2. `select[name=txt_edicao_19]` → id da edição
3. `form.requestSubmit(input[name=btExport])`

O submit **baixa sem recarregar a página** — dá para exportar várias edições em
sequência sem gastar navegação. Os arquivos caem em `~/Downloads` como
`19_n_<hash>.csv` (~40–50kb por edição). Exporte todas as edições que o estoque
toca **mais as promocionais**, senão as runas Promo aparecem como "sem linha".

Colunas (0-indexed, separador vírgula, UTF-8, **duas linhas de cabeçalho**):

```
0 Tipo(19)  1 EdiçãoID  2 Sigla  3 CartaID  4 Número  5 Edição  6 Raridade
7 Cor  8 NomePT  9 NomeEN  10 Idioma  11 Qualidade  12 QtdExistente
13 QtdSomar  14 Preço  15 Foil  16 Promo  17 AlteredArt  18 Assinada
19 Misprint  20 Estoque
```

⚠️ Ao montar o catálogo em memória, carregue **todos** os `19_n_*.csv`, não os
N mais recentes — cada export novo muda o "mais recente" e some com edição.

---

## Fase 3 — Montar o arquivo de importação

Chave: `(EdiçãoID, número normalizado)`. Para cada item do estoque:

1. resolve a edição pelo `set` (com override explícito para casos conhecidos);
2. acha a linha no catálogo;
3. **assere o nome** — `norm(nomeEN) == norm(name)`, ignorando acento e caixa;
4. divergiu? **não entra** — vai para o relatório e o dono decide.

⚠️ **Sanity check de preço antes de gerar o arquivo.** Compare `askBRL` com o
custo e com a referência: preço abaixo de metade do custo, ou abaixo de um quarto
da referência, costuma ser carta trocada ou referência de mercado quebrada — não
"promoção". Em One Piece isso publicou uma carta de R$330 a R$3,00, e ela vendeu.
Segure a linha e pergunte.

Preenche só quatro colunas na linha do catálogo:

| coluna | valor |
| --- | --- |
| `10` Idioma | `EN` |
| `11` Qualidade | `condition` do ledger, default `NM` |
| `13` QtdSomar | `qty` |
| `14` Preço | `askBRL` com **vírgula** decimal (`12,10`) |

Mantenha as duas linhas de cabeçalho e **um arquivo por edição**. Limites: 500kb
e 1000 linhas; só linhas com quantidade **e** preço são processadas.

---

## Fase 4 — Importar

Tela `?view=ecom/admin/import&tcg=19`. Sem `&tcg=` dá "TCG inválido".

> ### ⚠️ O rádio precisa de clique REAL no label
>
> `inpt-stock-action` está estilizado. Marcar `.checked = true` por JS **não
> revela o formulário** e o submit é descartado **em silêncio**: a página
> recarrega limpa, sem mensagem de erro, e nada é gravado. Custou duas
> tentativas fantasma antes de eu conferir o estoque e ver que não tinha entrado.
>
> Clique no **texto** "Acrescentar ao Estoque" (o dot do rádio não pega). Aí
> aparecem Arquivo, "Ignorar preço", "Cálculo de Estoque" e o botão Importar.

Sequência por arquivo:

1. clique no label `Acrescentar ao Estoque` → confira que o form abriu
2. `file_upload` no input `name=file` da seção (**não** `file_substract`)
3. confira: `ignorarPreco = false`, `descontarQtd = false`
4. clique no botão azul **Importar** (clique real, não `requestSubmit`)
5. leia a tabela de resultado

O resultado traz uma linha por card com `Novo` (não existia) ou `Atualização`
(já existia, quantidade/preço somados). **Numa primeira carga tudo deve ser
`Novo`** — qualquer `Atualização` significa que aquele card **já estava
cadastrado** e a quantidade acabou de **dobrar**.

Aconteceu de verdade em One Piece (35 cards, 2026-07-30). Para desfazer: monte um
CSV com as mesmas linhas e `QtdSomar = Quant Total − Quant Somada`, marque
**Subtrair do Estoque** e suba em `file_substract`. A coluna **Linha** do
resultado é o número da linha no seu arquivo (linha 3 = 1ª de dados) — é como se
mapeia de volta. Ver `liga-singles-onepiece`.

⚠️ **Marque o rádio com `label.click()` no DOM**, não por coordenada: as
coordenadas do screenshot e as do CSS têm escala diferente e o clique erra em
silêncio.

```js
const r = document.forms['home'].querySelector('input[name="inpt-stock-action"][value="1"]');
(r.closest('label') || r.parentElement).click();
```

Depois de importar, o botão **Importar novamente** volta ao formulário — mas o
rádio volta a zero, então repita o passo 1.

---

## Fase 5 — Conferir

Loja pública: `?view=ecom/itens&id=866280&tcg=19` (a categoria "Cartas de
Riftbound" usa **Link fixo** apontando para `tcg=19`; `&cat=` de categoria
card-game vem sempre vazia).

O contador `1-30 de N` tem que bater com o número de cards importados. Confira
alguns preços contra o estoque por amostragem.

⚠️ A grade da loja renderiza dentro de iframe — `document.body.innerText` da
página externa **não** contém os cards. Use screenshot para conferir.

---

## Mecânica do admin — vale para toda a Liga

1. `form.requestSubmit(botao)` é o submit confiável **para formulários comuns**;
   o de importação é a exceção (precisa do clique real, ver Fase 4).
2. Nunca `MouseEvent('click')` em checkbox (alterna e você grava o oposto).
   Rádio pode clicar — clique real só liga.
3. Verifique **recarregando / relendo o estoque**, nunca o form em memória.
4. A extensão do Chrome cai sozinha de vez em quando, e **um clique perdido é
   indistinguível de um clique que não fez efeito**. Depois de qualquer submit,
   confirme pelo estado real antes de seguir.
5. A saída do `javascript_tool` é bloqueada se ecoar URL com query string.
6. **Ritmo:** rajada de navegações gera **Cloudflare Error 1007 (ban de IP)** e
   derruba o acesso do dono. 7–10s entre navegações, `wait` de no máximo 10s por
   ação, ~2 páginas por batch. Foi por isso que este fluxo é por CSV: cadastrar
   pelo admin custaria ~40 navegações (50 linhas por página, sem controle de
   paginação), contra ~8 interações por CSV.

---

## Pendências conhecidas (2026-07-30)

- **UNL `R05A` "Chaos Rune" (6 un., R$47,92) e UNL `R06A` "Order Rune"
  (7 un., R$44)** ficaram **de fora** a pedido do dono: o número termina em `A`
  (= Alternate Art) mas nome e `variant` dizem carta normal. Registro
  contraditório no ledger — resolver lá antes de cadastrar.
- 15 singles seguem **sem preço** no Estoque e por isso fora da loja.

---

## Ao terminar

Relatório com: quantos cards entraram por edição, quantas unidades, valor total,
quais linhas não casaram e por quê, e o que ficou pendente. Aprendeu algo novo?
Acrescente aqui.

**Marque a flag no dashboard** para o que entrou, com os valores efetivamente
enviados (não os do ledger no momento) — é o que faz a aba Estoque avisar
"Reimportar" quando o estoque mudar depois:

```
POST $B/api/trades/liga?game=riftbound
{"items":[{"id":"<ledgerId>","ligaListed":true,"ligaQty":<qtd>,"ligaPriceBRL":<preco>}]}
```

Conferência posterior: skill `liga-conferir`.
