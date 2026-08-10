---
name: dashboard-cadastrar
description: Cadastra compras de cartas no dashboard de produção da Collecta (POST /api/trades), precificando pelo TCGplayer via TCGCSV quando a compra foi em USD, com rateio de frete/impostos e seed de imagens. Use quando o pedido for cadastrar/registrar/dar entrada de cartas compradas no dashboard/estoque de prod.
---

# Cadastrar compras no dashboard de prod

Registra uma lista de cartas compradas (ex.: colada do WhatsApp ou do carrinho TCGplayer) como holdings no dashboard de produção.

```
B=https://collecta-deals-production.up.railway.app
```

## Entrada esperada

- Lista no formato `qtd Nome - num/denom [SET] num` (variações toleradas; parsear manualmente).
- Jogo (`?game=`: `onepiece` | `pokemon` | `riftbound` | `lorcana` | `gundam`). **Valor inválido cai SILENCIOSAMENTE em One Piece** — conferir a grafia.
- Custo: ou preço unitário já em BRL, ou "preço TCGplayer −X%" + frete/impostos totais em USD.
- Perguntar se não estiver claro: desconto %, rateio do frete (proporcional ao valor é o padrão), câmbio.

## Passo 1 — preços em USD (TCGCSV)

Endpoints (exigem **User-Agent de browser** — sem ele retorna 401):

```
https://tcgcsv.com/tcgplayer/3/groups                  # Pokémon EN (categoria 85 = Pokemon Japan)
https://tcgcsv.com/tcgplayer/3/{groupId}/products      # extendedData.Number = "167/159"
https://tcgcsv.com/tcgplayer/3/{groupId}/prices        # linhas por (productId, subTypeName)
```

- Resolver set code → groupId pela `abbreviation` do `/groups`, normalizada (uppercase, sem espaços). Os códigos da Liga batem quase 1:1 (`JTG`, `SSP`, `CRZ:GG`, `SWSH10:TG`, `HIF:SV`, `MEP`…).
- **Ambiguidade `PR`**: dezenas de grupos usam a sigla `PR`; XY Promos = groupId **1451**. Promos avulsas com carimbo (ex. "Journey Together Stamped") moram em **2374 "Miscellaneous Cards & Products"** (código Liga `MCAP`).
- Casar carta por **numerador** do Number (cortar no `/`; já vem zero-padded: `037`, `GG54`, `TG01`, `SV93`, `XY186`). Se um número tiver 2 produtos (ex. MEW 205, versões Stamped), desempatar pelo nome.
- Preço base = **`directLowPrice` (TCGplayer Direct = lojas verificadas) e, se null, `marketPrice`**. Nunca `lowPrice` cru (tem listing de $0.01 de loja aleatória).
- Subtipo: preferir `Holofoil` → `Normal` → `Reverse Holofoil` (hits/secretas só existem em Holofoil).

## Passo 2 — custo em BRL

- `buyUSD = base × (1 − desconto)`.
- Frete/impostos: ratear proporcional ao valor — `frete_linha = total × (buyUSD×qty) / Σ(buyUSD×qty)`; dividir por `qty` para obter o `shippingBRL` unitário.
- Câmbio: `GET $B/api/trades?game=onepiece` → `fxRate` (sentido **BRL→USD**; inverter: `USD→BRL = 1/fxRate`). O envelope do Pokémon retorna `fxRate=1` (jogo BR-only) — não usar.

## Passo 3 — cadastrar (um POST por linha)

Sempre rodar um **dry-run** primeiro: imprimir tabela (carta, base USD, fonte direct/market, buyBRL, shipBRL) e a lista de não-casadas. Não-casadas NÃO são postadas — resolver manualmente. Depois postar com ~0.4s entre chamadas e guardar os `id`s retornados.

```bash
curl -s -X POST "$B/api/trades?game=pokemon" -H 'Content-Type: application/json' -d '{
  "number":"167","name":"N'\''s Reshiram","set":"JTG","variant":"EN","condition":"NM",
  "qty":1,"buyBRL":95.20,"shippingBRL":9.65,"store":"TCGplayer",
  "buyDate":"2026-08-03","delivered":false,"status":"holding"}'
```

Convenções (padrão do import PBL, já em prod):
- `number`: numerador zero-padded 3 dígitos, sem denominador; alfanuméricos como estão (`GG54`, `XY186`).
- `set`: código curto (`JTG`, `CRZ:GG`, `MCAP`).
- Idioma vai na `variant` (não existe campo language): `"EN"`, ou `"Foil EN"` / `"Reverse Foil EN"`; vazio = PT normal. PT e EN da mesma carta são linhas separadas.
- `condition`: `"NM"` salvo indicação.
- `buyBRL`/`shippingBRL` são **unitários**.
- Sem auth; só `/api/admin/reload` exige `X-Admin-Token`.
- **NUNCA** sincronizar `trades*.json` por arquivo — prod é dono do ledger; tudo via API. Erros corrigem-se com `PUT /api/trades/{id}` (substitui o registro INTEIRO) ou `DELETE /api/trades/{id}`.

## Passo 4 — imagens

1. Comparar as chaves `"SET/NUM"` cadastradas com `data/tracking-pkm/images.json` (ou `tracking`/`tracking-rft`/... conforme o jogo).
2. Para as faltantes, semear `"SET/NUM": "https://product-images.tcgplayer.com/fit-in/400x559/{productId}.jpg"` (productId veio do casamento no Passo 1). Fazer backup do arquivo antes.
3. Upload direcionado (não precisa do sync-up pesado) + reload:

```bash
bash -c 'source scripts/_common.sh && vf_upload "data/tracking-pkm/images.json" "/tracking-pkm/images.json" && reload_prod'
```

## Verificação

- `GET $B/api/trades?game=...` → filtrar por `store`+`buyDate` do lote: contagem de linhas, soma `(buyBRL+shippingBRL)×qty` ≈ `(Σ buyUSD + frete) × câmbio`.
- Amostrar 2–3 cartas (nome/variant/custo) e 2–3 artes via `GET $B/api/card-image?game=...&set=...&number=...` (esperar 200).

## Precificar (askBRL) com piso da Liga

Para dar preço às cartas cadastradas: piso **NM do idioma da carta** na Liga; sem oferta no idioma → regra combinada com o dono (em 2026-08-03 foi **2× o preço TCGplayer em BRL**). Sets fora do tracking exigem leitura no site:

- URL direta de carta: `?view=cards/card&card=<Nome>+(%23<num>%2F<den>)` — numerador zero-padded. Nomes com sufixo GX/V/VSTAR são hifenizados pela Liga (`Lumineon-V`), mas a busca tolera sem hífen. Promos: den `∞` (`Bulbasaur (037/∞)` ed=MEP); carimbadas viram sufixo `b` no número em SVP (`167b`). Se cair em "Busca:", usar a grade `?view=cards/search&card=<nome>` e clicar.
- **Preços das ofertas são sprite** (não estão no DOM; `innerText` vem sem os dígitos) → leitura VISUAL por screenshot. O resumo "Preço Médio de Venda" é texto, mas são estatísticas de VENDA (phantom), não listings.
- Filtrar idioma: checkbox "Inglês" na sidebar (clique de mouse real; `label.click()` via JS marca mas não dispara o AJAX). O filtro persiste entre navegações na sessão — **conferir a cada página** se só o idioma certo está marcado (já entrou "Português" junto por clique às cegas).
- Cloudflare: o ban 1007 pode ser **intermitente em janelas de ~10 min** — se der "Access denied", esperar 60–90s e tentar de novo; não é permanente.

## Referência de execução

Script usado no lote de 2026-08-03 (75 linhas, 88 unidades, R$ 13.336): padrão reutilizável de parse → TCGCSV → rateio → dry-run → post. Reescrever no scratchpad a cada uso; a lógica está toda descrita acima.
