# Página "Vendas da Liga" no dashboard

Objetivo: uma aba nova que junta todas as vendas já fechadas, agrupa por **pacote**
(= pedido enviado) e mostra receita, custo e **lucro de cada pacote**, com gráfico
diário. Cross-game, porque um pedido da Liga mistura jogos.

## Decisões do dono (06/08)

- "Pacote" = cada pedido da Liga (não produto selado).
- Escopo: toda venda em BRL, com filtro de canal (Liga / Sem marca / Shopee / Outros);
  vendas em US$ (TCGplayer) ficam fora.
- Todos os jogos juntos, não o jogo do seletor.

## Passos

- [x] 1. `GET /api/sales` (`internal/api/sales.go`): vendas realizadas de todos os
      jogos + acessórios **uma vez só**, cada linha com `game`/`gameName`.
- [x] 2. Testes (`internal/api/sales_test.go`): cross-game, acessório contado uma
      vez, proceeds em USD convertidos.
- [x] 3. `listSales()` + tipo `SaleRow` em `web/src/api.ts`.
- [x] 4. `web/src/ligasales.ts`: canal pelo `buyer`, nº do pedido, baixa simbólica,
      agrupamento em pacotes, série diária.
- [x] 5. `web/src/components/LigaSalesPage.tsx`: filtros, KPIs, gráfico, tabela de
      pacotes expansível.
- [x] 6. Navegação: `vendas` no `View`/`navGroups` (`brand.tsx`) e no `App.tsx`.
- [x] 7. `dayLabel` movido para `web/src/format.ts` (era local do `ExpensesPage`).

## Review

Feito e verificado localmente com os ledgers de produção copiados para um servidor
`-serve-only` na :8099 (o container do dono na :8080 não foi tocado).

Números conferidos contra a fonte: 120 vendas, 85 em BRL — Liga 22 / sem marca 48 /
outros 9 / Shopee 6. Com o filtro padrão (Liga + sem marca, escondendo as baixas
simbólicas): 14 pacotes, R$ 11.939,05 de receita, R$ 9.053,06 de custo,
R$ 2.885,99 de lucro (+32%). Os três pedidos com número bateram com o que a skill
`liga-vendas` registrou: #11541716 R$ 430,80 · #11542577 R$ 205,00 ·
#11576384 R$ 629,73 (+84%, 6 itens de Riftbound + One Piece no mesmo pacote).

Bug encontrado e corrigido no caminho: as barras do recharts renderizavam com uma
escala ~2,9× menor que o eixo (uma venda de R$ 5.098 desenhava até a marca de
R$ 1.900). A animação de crescimento das barras congela a escala do primeiro
layout e não recalcula nem em re-render; `isAnimationActive={false}` resolve.
Vale lembrar disso em qualquer gráfico de barras vertical novo — o
`InventoryPanel` não sofre porque as barras dele são horizontais.

Publicado em 06/08: imagem local rebuildada (`docker compose build/up -d opdeals`)
e deploy na Railway (`railway up`) — a página está no ar em
`?tab=vendas` com os 14 pacotes / R$ 11.939,05 / R$ 2.885,99 de lucro. O `:8080`
local mostra a aba vazia de propósito: os ledgers em `data/` estão atrasados
(One Piece 91 vs 213 trades; Pokémon 0 vendas vs 48) — os dados de venda vivem na
prod. `scripts/sync-down.sh` traria isso para cá se um dia for preciso.

⚠️ O `railway up` sobe a árvore de trabalho inteira: foi junto o WIP não commitado
de outra sessão em `internal/liga/` e `internal/tracking/` (extras/foil). Compila e
os testes passam; prod é `-serve-only`, então o código de scraping não roda lá.

Pendências para o dono:

- As baixas manuais sem `buyer` ("sem marca", 48 itens) viram um pacote por dia,
  o que pode juntar dois pedidos diferentes do mesmo dia numa linha só. Se quiser
  precisão por pedido, a baixa precisa carregar `Liga #<pedido>` — é o que a skill
  `liga-vendas` já faz quando é ela que dá a baixa.
