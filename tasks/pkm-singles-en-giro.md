# Singles de Pokémon em inglês — o que fazer para girar

Diagnóstico feito em 2026-08-08 sobre as 73 singles EN compradas no TCGplayer
(custo R$ 13.015,79, 84 unidades, todas cadastradas na Liga em 2026-08-03) e os
snapshots de `data/tracking-pkm/`.

## Achado principal: as 73 estão anunciadas como "Normal", e o mercado é Foil

O campo `extras` do estoque da Liga vale `2` = Foil. Nas 30 cartas nossas que o
tracking cobre e têm mercado, **29 estão com `extras=0` (Normal / Sem Extras)**
num comparador que é 86–100% Foil. Confirmado ao vivo em
`Team Rocket's Mimikyu (#238/217)`:

- `cards_stock`: 53 ofertas — 52 com `extras=2`, e exatamente 1 com `extras=0`,
  da loja `866280` (Collecta).
- Filtro da página: `Normal / Sem Extras 1` × `Foil 15`.
- "Preço Médio de Venda / Normal" mostra R$ 154,90 nas três colunas (= o nosso
  próprio preço, a Liga não tem venda nenhuma registrada nesse balde). O Foil
  real é R$ 54,00 / R$ 106,49 / R$ 249,99.

É o mesmo defeito das 129 singles de Riftbound (coluna 15 do CSV em branco).
Todas as 73 são secret rare / illustration rare / ex / V / VSTAR / GX / full art
— **não existe versão Normal de nenhuma delas**.

Consequência: quem filtra "Foil" (o filtro natural para uma Ilustração Rara) e a
Lista de Compras da Liga não enxergam a loja. Quem navega a lista inteira ainda
vê — a perda de visibilidade é parcial, não total.

O mesmo problema atinge **26 ofertas em português** (`extras=0`) nos snapshots.

## Achados de preço/posição (medidos no tracking-pkm)

**1. Idioma não é o problema.** Giro por anúncio/dia: EN 0,0240 × PT 0,0274 no
agregado, e EN é *melhor* que PT nas faixas R$ 15–150 (0,0333 × 0,0237 em
R$ 15–40; 0,0126 × 0,0085 em R$ 80–150).

**2. Posição quase não move agulha em carta cara.** Giro por posição no
comparador, ticket R$ 150+: 1º = 0,0061 un/dia, 51º+ = 0,0046 — razão 1,3×. Na
faixa R$ 50–150 a razão é 2,2×. Não vale queimar margem por centavo em carta de
R$ 300.

**3. O que manda é o ticket.** Dias para vender 1 unidade por anúncio (EN):

| faixa | dias/un |
| --- | --- |
| R$ 15–40 | 30 |
| R$ 40–80 | 42 |
| R$ 80–150 | 79 |
| R$ 150–300 | 106 |
| R$ 300–600 | 144 |
| R$ 600+ | 691 |

58 dos 73 anúncios estão em R$ 150+. Isso é o prazo, não um erro de preço.

**4. Seis cartas estão abaixo do piso Foil do mercado** — dinheiro na mesa:
Pikachu ex SSP (ask 500,00 / piso 599,99), Servine BLK (279,90 / 369,90),
Misty's Psyduck DRI (662,15 / 697,00).

**5. Duas foram compradas caro demais** — casar o piso dá prejuízo:
Tynamo BLK (custo 108,29 / piso 74,99) e Fraxure BLK (custo 151,65 / piso
149,90).

**6. 38 das 73 estão em sets fora do tracking** (CRZ:GG, HIF:SV, MCAP, MEP, OBF,
PAL, PAR, PGO, PR, SFA, SM01, SVI, SWSH04/09/10/12, SWSH12:TG, TEF, TWM) — não dá
para medir posição nelas hoje. `DefaultTrackSets` está em
`internal/game/game.go:352`.

## Plano

- [x] 1. Corrigir Foil nas 73 EN em `?view=ecom/admin/cartas/all&tcg=2`,
      filtrando `txt_filtro=1` + `txt_extras_opcao=1` (Sem) + `txt_extras[]=2` e
      marcando `txt_extras_<n>[]=2` linha a linha. Atualiza o anúncio existente,
      estoque intacto. **Reimportar não conserta — o import soma.**
      Feito em 2026-08-08 — ver "Execução do passo 1" abaixo.
- [ ] 2. Auditar as PT com o mesmo filtro e corrigir as que forem foil
      (26 identificadas nos snapshots; o total é provavelmente maior).
- [ ] 3. Subir preço das que estão abaixo do piso Foil (item 4 acima).
- [ ] 4. Decidir Tynamo e Fraxure: aceitar prejuízo ou segurar.
- [ ] 5. Adicionar os 19 sets faltantes ao `DefaultTrackSets` para medir posição
      no resto do estoque.
- [ ] 6. Reprecificar tudo **depois** do passo 1 — com os anúncios no balde
      certo, a posição atual medida contra Normal não vale nada.

## Execução do passo 1 (2026-08-08)

Admin correto é `ligacollectatcg.com.br` (não `ligapokemon.com.br`, que devolve
"não está logado" para essa tela).

Busca `Cards com Estoque` + `Inglês` + extras `Sem` `Foil` devolveu **73 linhas**
(50 + 23). Casei cada linha por **número + preço + quantidade** contra o ledger
antes de marcar; 71 bateram com as 73 EN do ledger, mais 2 avulsas de PBL já
cadastradas antes (`Lurantis ex #004/084` R$ 5,00 e `Mega Darkrai ex #048/084`
R$ 12,00 — cards `ex`, só existem em foil). Marquei as 73.

Depois: a mesma busca devolve **0 linhas**, e `Com Foil` + `Inglês` devolve
**75** (as 73 + `AZ's Tranquility` R$ 196,99 e uma de R$ 49,90 que já estavam
corretas). Preço e quantidade conferidos linha a linha, todos intactos.

Na página pública do `Team Rocket's Mimikyu (#238/217)`: filtro passou de
`Normal 1 / Foil 15` para **`Normal 0 / Foil 16`**, e as 53 ofertas do
`cards_stock` estão em `extras=2`, a nossa inclusive.

**Pendência achada:** `Chansey` (TWM, ask R$ 349,90, 1 un) está no ledger como
`ligaListed` mas **não aparece na Liga** com estoque em inglês. Verificar se o
cadastro falhou.

**Armadilha nova:** o primeiro `requestSubmit(btSalvar)` gravou os 50; o segundo,
disparado na página que veio *do próprio reload do save*, **falhou em silêncio**
— a mensagem "Alterada" continuava na tela e as 23 linhas seguiam sem Foil. Só
depois de refazer a busca (`btBuscar`) numa página nova o save pegou. Confirmar
sempre com uma busca nova, nunca com o reload do save.

## Ressalvas do método

- Venda é inferida por queda de quantidade entre snapshots; transições com
  intervalo > 24h foram descartadas.
- Pre Order não é gravado no snapshot: o "piso" pode não existir em pronta
  entrega.
- Prazo em dias é ordem de grandeza; as razões entre faixas são robustas.
