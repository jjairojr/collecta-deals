---
name: cardrush-capturar
description: Atualiza os preços da Card Rush (cardrush-pokemon.jp) que alimentam a página "Importar Japão" do dashboard — captura AR e SAR japonesas pelo Chrome e grava data/cardrush-pkm.json. Use quando o pedido for atualizar a Card Rush, atualizar preços japoneses, "roda a card rush", ou quando a página Importar Japão estiver com captura velha.
allowed-tools: Bash, Read, Write, mcp__claude-in-chrome__*
---

# Capturar preços da Card Rush

A página **Importar Japão** (`/api/jp-import`) cruza o preço da Card Rush com os
pisos JP/PT/EN da Liga. A Liga se atualiza sozinha pelo scheduler (o capturador
`pokemon jp import capture` grava só as cartas secretas em `data/jpimport-pkm`,
e as coleções já rastreadas vêm de `data/tracking-pkm`). A Card Rush **não**:
ela tem Cloudflare e só responde em IPv4, então o servidor não consegue buscar
sozinho. Esse passo é sempre pelo Chrome do dono.

## Passo a passo

1. Abra uma aba nova em `https://www.cardrush-pokemon.jp/product-group/447`
   (qualquer página do site serve; precisa ser a origem da Card Rush).
2. Rode o conteúdo de `scripts/cardrush-capture.js` com `javascript_tool`. Ele
   busca `AR <coleção>` em `/product-list` para cada coleção da lista, com 2,5 s
   entre páginas (~3 min), descarta cópias com estado (`〔状態…〕`, PSA) e fica
   com o NM mais barato em estoque de cada carta.
3. Acompanhe `window.__cardrushLog` / `window.__cardrushDone` até terminar.
4. O resultado vira um `<pre>` compacto na página
   (`set,num,rar,yen,stock,productId;…`). Leia com `get_page_text`, grave em
   um arquivo do scratchpad e converta:
   `python3 -I scripts/cardrush-import.py <arquivo.txt> data/cardrush-pkm.json`
5. Não precisa reiniciar nada: a API recarrega pelo mtime do arquivo.

## Ao adicionar uma coleção nova

- Inclua o código da Card Rush (o `[M6]` do nome do produto) na lista `sets`
  de `scripts/cardrush-capture.js`.
- Inclua a coleção em `internal/jpimport/sets.go` com a sigla JP da Liga e as
  siglas ocidentais onde as cartas dela saíram em português (consulte
  `ligapokemon.com.br/?view=cards/edicoes`). Coleção sem versão PT ainda fica
  com `West` vazio e aparece como "PT não lançado".
- Se o bloco SIR da coleção ocidental não começar no primeiro nome repetido
  (ex.: 30C não tem bloco de full art), ponha o número em `sirStart`.

## Gotchas

- As páginas por raridade (`product-group/447`, `22`, `23`) **não** listam todas
  as cartas. A busca por coleção é a única fonte completa.
- A palavra-chave `AR` também casa `SAR`, `CHR` e coleções com nome parecido
  (`M2` casa `M2a`). O script filtra pelo código exato do produto.
- O `get_page_text` devolve tudo, então o formato compacto importa: o JSON
  completo com nome japonês passa de 75 mil caracteres.
