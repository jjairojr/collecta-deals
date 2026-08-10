# Aba Financeiro no dashboard

Página que responde "quanto entrou, de onde veio e o que sobrou depois das despesas",
juntando as vendas de todos os jogos (`/api/sales`) com o ledger de despesas
(`/api/expenses`).

## Escopo

- [x] `web/src/finance.ts` — helpers puros: recorte por período, lado (Liga × fora),
      tipo (single/selado/acessório), série mensal, despesas por mês e por categoria.
- [x] `web/src/components/FinanceiroPage.tsx` — filtros, KPIs, gráficos e tabelas.
- [x] `web/src/brand.tsx` — view `financeiro` + item de menu.
- [x] `web/src/App.tsx` — título/descrição e rota.
- [x] `tsc` limpo.

## Decisões

- **Período por mês** (`YYYY-MM`, de/até + presets). Despesa recorrente é cobrada por
  mês; um recorte por dia arbitrário faria rateio que ninguém pediu.
- **"Sem marca" conta como Liga por padrão**, com checkbox para desligar. É o que a
  `ligasales.ts` já documenta: baixa manual sem nº de pedido quase sempre é Liga.
  Desligando, ela vira "fora da Liga" — os dois extremos da mesma dúvida.
- **Vendas em US$ ficam de fora** (TCGplayer, não é a loja) e baixas simbólicas de
  R$ 1/un. (abertura de case) são ocultáveis, como na aba Vendas.
- **Despesa não é rateada por canal.** Ela entra uma vez, no resultado do mês.
  Lucro por canal é sempre bruto; líquido só existe no total.

## Editar venda (aba Vendas)

- [x] `updateTrade` aceita o jogo da linha — a aba Vendas cruza jogos, então não dá
      para usar o seletor global.
- [x] Linha do item vira formulário: unidades, venda total, data e comprador/canal.
      O `PUT /api/trades/{id}` substitui o registro inteiro, então o formulário envia
      a venda completa e só troca esses quatro campos.

## Resultado

Entregue: KPIs (venda total, Liga, fora da Liga, lucro bruto, despesas, lucro
líquido), gráfico de receita/lucro/despesa por mês, gráfico Liga × fora, tabela
DRE mensal, quebra por tipo de produto e por jogo, e despesas por categoria.
