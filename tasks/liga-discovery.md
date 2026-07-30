# Discovery — loja Collecta na LigaMagic

Levantamento **somente leitura** do admin da loja `id=866280` feito em 2026-07-29.
Nada foi editado (ver "O que foi tocado" no fim). Objetivo: mapear o que dá pra
ligar, o que dá pra desligar e o que falta, considerando que o escopo hoje é
**One Piece, Riftbound, Pokémon e acessórios**.

---

## STATUS DA EXECUÇÃO — 2026-07-30

| item | estado |
| --- | --- |
| P0.1 Pokémon público | ✅ feito e verificado (`Exibição de Cartas = Todas Edições`) |
| P0.2 Pokémon "Somente com Estoque" | ✅ feito e verificado |
| P0.3 Categoria Riftbound no menu | ✅ criada (id `256180`), aparece em 3ª posição |
| P0.4 Cadastrar estoque | ⏳ é o próximo passo — ver `/liga-cadastrar` |
| P1.5 Esconder 7 categorias fora de escopo | ⚠️ flag `Exibir somente para funcionários` marcado nas 7, **visão do cliente não verificada** |
| P1.6 Podar subcategorias de selados | ⛔ não feito de propósito (ver nota) |
| P1.7 Podar acessórios cauda-longa | ⛔ não feito (mesma nota) |
| P1.8 Blocos de Destaque | ✅ agora Riftbound / Pokémon / One Piece / One Piece(destaque) — zero Magic e Yugioh |

**P1.5 — o que falta confirmar:** o campo se chama `txt_prod_exclusivo` ("prod"
sugere que talvez restrinja só os *produtos* da categoria, não a entrada do
menu). Estou logado como dono, então continuo vendo as 7 no menu — o que é o
comportamento esperado para staff, mas não prova nada. `fetch` sem credenciais
toma 403 do Cloudflare. **Abrir a loja numa janela privada resolve em 10s.** Se as
7 ainda aparecerem, o único caminho definitivo é apagar as categorias (é
irreversível, então precisa da sua decisão).

**P1.6/P1.7 — por que não fiz:** seriam ~66 requisições no admin. O ban de IP do
Cloudflare (Error 1007) já disparou uma vez durante o discovery e derruba o seu
acesso também. Além disso o ganho é baixo: subcategoria escondida continua
visível para staff, então não simplifica o cadastro — só afeta o menu do cliente
quando houver produto. Fazer aos poucos, junto do cadastro de estoque.

**Pendência cosmética:** a ordem dos blocos ficou Riftbound → Pokémon → One
Piece. O ideal é One Piece primeiro (é o jogo com estoque). O campo `txt_ordem`
não aparece no modo edição do bloco; provavelmente é arrastar-e-soltar.

---

## 1. Estado atual em uma tela

| | |
| --- | --- |
| Plano | **Basic (Loja Virtual)** — R$ 136,99/mês, status **Pública** |
| Vencimento / próxima cobrança | 29/08/2026 · cartão X-5436 |
| Pedidos | **0** (nenhum pedido em nenhum status) |
| Vendas | **R$ 0,00** nos últimos 14 dias |
| Produtos cadastrados | **0** (nenhum selado, nenhum acessório) |
| Cards cadastrados | **1** — One Piece, Shanks `#OP01-120` |
| Recesso no Marketplace | não configurado (loja ativa) |
| Layout / identidade visual | ✅ feito (ver `storefront/design/liga/README.md`) |

A loja está **operacionalmente pronta e comercialmente vazia**: pagamento, envio,
layout e marketplace estão configurados; falta catálogo.

---

## 2. O modelo mental que importa

Isto não é óbvio no admin e explica quase todos os problemas abaixo:

> **O menu da loja vem de `Cards e Produtos → Categorias`, não das preferências
> de card game.** As duas coisas são independentes, e é exatamente aí que estão os
> furos: existe jogo público sem entrada no menu, e entrada no menu de jogo que
> está privado.

Três chaves diferentes controlam se um jogo aparece:

1. **Exibição de Cartas** (`join&p=5` → Preferências por Card Game) — `Nenhuma`
   deixa o jogo **privado**; `Todas Edições` deixa público.
2. **Categoria principal** (`categorias`, Regra Específica = *Card Game*) — é o
   que cria o item de menu.
3. **Plugin de Marketplace** (`assinatura`) — é o que indexa seu estoque no
   comparador de preços da Liga (LigaPokemon, LigaOnePiece, …).

---

## 3. Custo mensal atual

| item | R$/mês |
| --- | --- |
| Basic (Loja Virtual) | 136,99 |
| Marketplace **LigaPokemon** | 129,99 |
| Marketplace **LigaOnePiece** | 39,99 |
| Marketplace **LigaRiftbound** | 0,00 |
| Marketplace **LigaLorcana** | 0,00 |
| Geração automática de Nota Fiscal | 70,99 |
| **Total** | **R$ 377,96** |

Plugins **não** assinados: Marketplace LigaMagic (409,99), LigaYugioh (73,99),
LigaFAB (21,99), LigaDigimon (9,99), Vanguard/DBMasters/DBFusion/StarWars/Gundam/
Sorcery/Funko (R$ 0,00 cada), Buylist (120,99), Análise de Preços e Estoque
(138,99), Cards mais Vendidos (83,99), Cloudflare Argo (100,00), Regras de
limitação de estoque (140,49), ComparaJogos (99,99).

---

## 4. Escopo vs. o que está configurado

| jogo | tcg id | Exibição de Cartas | Itens exibidos | público? | no menu? | Marketplace pago |
| --- | --- | --- | --- | --- | --- | --- |
| **One Piece** | 11 | Todas Edições | Somente c/ estoque | ✅ | ✅ | R$ 39,99 |
| **Riftbound** | 19 | Todas Edições | Somente c/ estoque | ✅ | ❌ **não existe** | R$ 0,00 |
| **Pokémon** | 2 | ❌ **Nenhuma** | Todos | ❌ **privado** | ✅ | **R$ 129,99** |
| Magic | 1 | Nenhuma | — | ❌ | ✅ (abre vazio) | — |
| Yugioh | 3 | Nenhuma | — | ❌ | ✅ (abre vazio) | — |
| outros 11 jogos | — | Nenhuma | — | ❌ | ❌ | — |

Verificado no site público:

- `?view=ecom/itens&id=866280&tcg=11` → cai direto no card do Shanks (1 resultado)
- `?view=ecom/itens&id=866280&tcg=19` (Riftbound) → busca vazia
- `?view=ecom/itens&id=866280&tcg=2` (Pokémon) → busca vazia

### Os dois furos mais caros

1. **Pagamos R$ 129,99/mês pelo Marketplace LigaPokemon com o catálogo Pokémon
   privado.** No admin, `Cards e Produtos → Cadastrar Cards` com Pokémon
   selecionado exibe o aviso *"Pokemon se encontra privado em sua loja — clique
   aqui para deixá-lo público"*. É um clique.
2. **Riftbound está público e com marketplace ativo, mas não tem categoria de
   menu** — o cliente não tem como chegar nele navegando.

---

## 5. Plano de ação

### P0 — destravar o escopo (barato, alto impacto)

1. **Tornar Pokémon público.** `cartas/all&tcg=2` → link do aviso, ou `join&p=5`
   → Pokémon → Exibição de Cartas = *Todas Edições*.
2. **Alinhar Pokémon aos outros dois:** Modo de Exibição Padrão dos Itens =
   *Somente com Estoque* (hoje está *Todos*; One Piece e Riftbound já estão em
   Somente com Estoque).
3. **Criar a categoria principal "Riftbound"** em `categorias` (Tipo = Categoria
   Principal, Regra Específica = *Card Game*), espelhando One Piece/Pokémon.
4. **Cadastrar estoque.** Com "Somente com Estoque" ligado, sem estoque as
   páginas ficam vazias — é o gargalo real, não a configuração.

### P1 — esconder o que não vamos usar

5. **Categorias de menu fora de escopo.** Hoje o menu tem 11 entradas e 7 estão
   fora do escopo: `Magic: The Gathering`, `Yugioh`, `Funko`, `Liga Bolts`,
   `Vestuário`, `Colecionáveis`, `Sets / Playsets`. Magic e Yugioh são as piores:
   estão no menu **e** com cartas em "Nenhuma", então abrem vazias — pior do que
   não existir.
6. **Podar as subcategorias de Produtos Selados.** São ~40, e a maioria é de
   Magic/Yugioh.
   - **Manter** (servem a OP/RFT/PKM): Boosters Avulsos, Caixas de Boosters,
     Coleção Treinador Avançado (ETB), Ferramentas de Treinador, Latas, Box
     Colecionável, Kit Colecionável, Decks Selados, Blisters, Gift Set,
     Collectors Edition, Pacotes / Fat Packs, Pacote de Pré-lançamento,
     Start/Starter Deck, Special Anniversary Box, Caixa / Lata Vazia, Outros.
   - **Remover** (Magic/Yugioh/outros): Secret Lair, Planeswalker Decks, Event
     Decks, Brawl Decks, Duel Decks, From the Vault, Global Series, Clash Pack,
     Intro Pack, Signature Spellbook, Decks Temáticos, Challenger Decks,
     Commander / Produtos Multiplayer, Desafio Estratégico, Speed Duel, Deck
     Estrutural, Trial Deck, Blitz Deck, Expert / Advanced Deck, Deck Campeonato
     Mundial, Deck Edição Especial, Outros Multiplayer.
7. **Acessórios** tem 15 subcategorias e todas são plausíveis. Se não vamos
   vender, dá pra podar as de cauda longa: Livros, HQ / Comic Book, Broche / Pin,
   Moeda Colecionável, Field Center Card, Ficha / Token.
8. **Blocos de Destaque da home** (`home_config`) ainda são Magic / Pokemon /
   Yugioh / One Piece. Trocar para **One Piece / Pokémon / Riftbound**.

### P2 — conteúdo que já está linkado mas não existe

9. **FAQ está vazio.** É linkado no rodapé *e* no Banner Superior que configurei
   apontando para `ecom/faq`. Hoje a página mostra "Nenhuma Pergunta Frequente
   cadastrada". Cadastrar 4–6 perguntas (conferência das cartas, prazo de envio,
   formas de pagamento, trocas/devoluções) **ou** repontar o banner.
10. **Termos e Condições vazio** (`termos`).
11. **Horário de funcionamento vazio** — a coluna existe no rodapé da loja e
    aparece sem conteúdo. Configurável em `horarios`.

### P3 — manter desligado (confirmado que não usamos agora)

| item | estado | observação |
| --- | --- | --- |
| Torneios / Eventos | config vazia | Calendário já está em *Não Exibir* ✅ |
| Buylist | não criada | exige plugin R$ 120,99 pra ficar pública |
| Ingressos | não usado | — |
| Pre Order | não usado | — |
| Marketplaces externos (`mpExterno`) | bloqueado | exige plugin próprio |
| Marketplace LigaMagic / Yugioh / FAB / Digimon | não assinados | fora do escopo |

### P4 — decisões de custo (suas, não minhas)

- **Geração automática de NF — R$ 70,99/mês** com 0 pedidos. Faz sentido pausar
  até ter volume, mas é decisão fiscal.
- **Marketplace LigaLorcana ativo a R$ 0,00** — fora do escopo, mas não custa
  nada. Higiene, não economia.
- Se Pokémon não for prioridade nos próximos meses, os R$ 129,99/mês do
  LigaPokemon são o maior item cortável do plano.

---

## 6. Economia do Marketplace da Liga

- Taxa: **3,99% + R$ 0,40** por venda de valor ≥ R$ 20,00 (à vista e parcelado).
- Recebimento: **30 dias + confirmação de entrega** — opção padrão pra lojas
  novas; após 6 meses dá pra pedir outras opções via suporte.
- **Vendas feitas dentro da própria loja virtual não pagam taxa de marketplace**
  — só a taxa do meio de pagamento usado. Isso favorece direcionar tráfego
  próprio pra loja em vez do comparador.

---

## 7. Selo de Loja Verificada — roadmap

| requisito | estado |
| --- | --- |
| ≥ 3 meses em algum Marketplace | ❌ menos de 3 meses |
| Aprovação de pedidos automática | ✅ já está |
| Prazo de envio de 1 dia | ✅ já está (corrobora o "ENVIO EM 24H" dos banners) |
| ≥ 10 avaliações em 12 meses, média ≥ 4,79 | ❌ 0 avaliações |
| Sem disputas | ✅ 0 disputas |

Ou seja: o selo depende só de **tempo + volume de vendas avaliadas**. Nada a
configurar.

---

## 8. Já pronto — não mexer

- **Pagamento:** PIX / Nubank com **5% de desconto** + LigaSegura Integrada.
- **Envio:** Correios via **Melhor Envio** + Retirar na Loja (balcão).
- **Layout/identidade:** cabeçalho, logo, cores, favicons, banners e hero — ver
  `storefront/design/liga/README.md`.
- **Calendário de torneios:** já oculto.
- **Home do Site:** "Única para Todos TCGs" (faz sentido com 3 jogos).

---

## 9. Referência rápida

### IDs de card game (`tcg=`)

`1` Magic · `2` **Pokémon** · `3` Yugioh · `4` Battle Scenes · `5` Vanguard ·
`6` SW Destiny · `7` DB Masters · `8` Flesh and Blood · `9` Lorcana ·
`10` Digimon · `11` **One Piece** · `12` SW Unlimited · `13` DB Fusion World ·
`17` Gundam · `18` Sorcery · `19` **Riftbound**

### Categorias úteis (`cat=`)

`254685` Boosters Avulsos · `254686` Caixas de Boosters · `254674` Sleeves /
Shields · `254724` Funko

### Telas do admin

| tela | URL (`?view=`) |
| --- | --- |
| Preferências por card game | `ecom/admin/join&p=5` |
| Categorias (monta o menu) | `ecom/admin/categorias` |
| Cadastrar cards / público-privado | `ecom/admin/cartas/all&tcg=N` |
| Cadastrar produtos | `ecom/admin/prod/all` |
| Plano e plugins | `ecom/admin/assinatura` |
| Taxas do marketplace | `ecom/admin/marketplace_join` |
| Selo de loja verificada | `ecom/admin/selo` |
| Pagamento / Envio | `ecom/admin/join&p=2` / `&p=3` |
| Home (blocos + notícias) | `ecom/admin/home_config` |
| Banners | `ecom/admin/banners` |
| Layout responsivo | `ecom/admin/home_responsiva_config` |
| Config. da loja + favicons | `ecom/admin/join&p=4` |
| FAQ / Termos / Horários | `ecom/admin/faq` · `termos` · `horarios` |

---

## 10. O que foi tocado neste discovery

Nada gravado. Duas interações de leitura, por transparência:

- cliquei **Buscar** em `prod/all` (busca GET, retornou vazio — foi assim que
  confirmei 0 produtos);
- mudei o seletor de "Exibir 10/25/50/100" da tabela de categorias para ver as 70
  linhas — é paginação client-side (DataTables), não persiste nada.

Nenhum botão Salvar / Cadastrar / Atualizar foi acionado.
