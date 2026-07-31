# Peças impressas da Collecta

Três peças na mesma identidade arcade da loja (rosa `#F6559B`, azul royal
`#1355B3`, ink `#0b0b0c`, Baloo 2 + Press Start 2P, mascote):

1. **Flyer A6 "CONTINUE?"** — vai dentro da encomenda, oferece 5% OFF na próxima
   compra feita pelo site
2. **Cartão de visita 90×50mm** — frente e verso
3. **Cartela de adesivos A6** — 12 adesivos

## Gerar

```
npm install
npm run build
```

## O que sai em `dist/`

### Pra mandar pra gráfica (PDF com sangria de 3mm)

| arquivo | tamanho final | páginas |
| --- | --- | --- |
| `flyer-a6-sangria.pdf` | 105×148mm (A6) | 1 |
| `cartao-frente-verso-sangria.pdf` | 90×50mm | 2 — pág. 1 frente, pág. 2 verso |
| `adesivos-a6-sangria.pdf` | 105×148mm (A6) | 1 |

### Pra imprimir em casa (sem sangria, já no tamanho final)

| arquivo | o que é |
| --- | --- |
| `flyer-a4-4up.pdf` | 4 flyers numa A4 |
| `cartao-a4-10up.pdf` | 10 cartões por página, 2 páginas (frentes / versos) pra duplex |
| `adesivos-a4-4up.pdf` | 4 cartelas numa A4 — imprima em papel adesivo |

Imprima sempre em **escala 100% / tamanho real**, nunca "ajustar à página".
Esses arquivos ocupam quase toda a folha, e impressora doméstica não imprime até
a borda — vai sobrar uma margem branca. Pra resultado de verdade, use os PDFs
com sangria numa gráfica.

### Previews

`flyer-a6-sangria.png`, `cartao-preview.png` (frente e verso empilhados) e
`adesivos-a6-sangria.png`. Servem pra aprovar e mandar no zap. Os previews
mostram a área de sangria, então a borda visível é maior que o corte final.

Os `.html` são os fontes renderizados — fontes e imagens vão embutidos em
base64, abrem em qualquer navegador, sem internet.

## Mandando pra gráfica

- **Formato: PDF**, os `*-sangria.pdf`. Texto, QR Code e todas as formas são
  vetor; só o mascote é bitmap (346×360px, usado em no máximo 30mm = ~293 dpi).
  As três fontes vão embutidas no PDF, então não tem risco de substituição.
- **Sangria já incluída**: os arquivos são 3mm maiores que o corte de cada lado
  (flyer e cartela 111×154mm, cartão 96×56mm). A gráfica corta e sobra o tamanho
  final. Margem de segurança do conteúdo: ~4mm do corte no cartão, ~7mm no flyer.
- **Sem marcas de corte** — a maioria das gráficas online não pede. Se a sua
  exigir, dá pra acrescentar.
- **Os PDFs estão em RGB.** O rosa `#F6559B` é bem saturado e perde um pouco de
  brilho na conversão pra CMYK. Se a gráfica aceitar RGB, deixe que ela converta;
  se exigir CMYK, avise que precisamos gerar uma versão convertida.
- **Papel sugerido**: flyer em couché fosco 250g; cartão em couché 300g (com
  laminação fosca a arte escura fica bem melhor e não marca digital); adesivos em
  vinil branco com corte kiss-cut nos contornos brancos de cada adesivo.

## Mudar o conteúdo

Constantes no topo de `lib.mjs`: `COUPON`, `SITE`, `INSTAGRAM`, `WHATSAPP`.
Cada peça tem seu QR com UTM próprio (`print`, `cartao`, `adesivo`), então o GA
separa de onde veio cada visita. Depois de mexer em qualquer URL, vale reconferir
se o QR decodifica lendo o PNG gerado.

Arquivos: `lib.mjs` (tokens, fontes, QR, render), `art/flyer.mjs`,
`art/card.mjs`, `art/stickers.mjs`, `build.mjs` (monta e exporta tudo).

## Cupom

O checkout é por WhatsApp, então o desconto é aplicado à mão. O flyer instrui:
monte o carrinho no site → finalize no zap → mande o cupom `CONTINUE5`.
