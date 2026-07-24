# Handoff: Collecta — Arcade TCG Marketplace

## Overview
Collecta is a Brazilian marketplace for trading-card products: **singles** (individual cards, graded or raw) and **sealed product** (booster boxes, ETBs, bundles, starters) across three games — **Pokémon**, **One Piece** and **Riftbound**. Currency is BRL (R$), all copy is Portuguese (pt-BR).

The design is a single-page HTML prototype with 5 screens switched by a top "arcade" tab bar: Home, Browse (singles + filters), Single card product page, Sealed product page, Cart.

## About the Design Files
The files in this bundle are **design references created in HTML** — prototypes showing intended look and behavior, **not production code to copy directly**. The task is to **recreate these designs in the target codebase's existing environment** (React/Next, Vue, Rails views, native, whatever is in place) using its established components, routing, and data layer. If no codebase exists yet, choose the most appropriate framework and implement the designs there.

The prototype uses a small in-house streaming-component runtime (`support.js`, `<x-dc>`, `{{ }}` holes, `<sc-for>`, `<sc-if>`). **Do not port that runtime.** Read it as: template = markup, `renderVals()` = the view model. All styling is inline on purpose (prototype constraint) — in production move it to the codebase's normal styling approach (CSS modules, Tailwind, styled-components…), keeping the exact values listed below.

## Fidelity
**High fidelity.** Colors, typography, spacing, borders, shadows and copy are final and should be matched closely. Product imagery is intentionally **placeholder** (striped blocks with monospace labels like `FOTO DA CARTA`) — replace with real photography; keep the aspect ratios (5:7 for cards, 4:3 for sealed boxes).

## Design Tokens

### Colors (from the Collecta brand sheet)
| Token | Hex | Use |
|---|---|---|
| Pink (primary) | `#F6559B` | Header, primary buttons, prices, active tab, accents |
| Light pink | `#FDC4E5` | Secondary buttons, badges, sealed cards, pixel-label text on dark |
| Blue | `#1355B3` | Hero background, secondary CTA, seller bar, announcement bar |
| Ink / outline | `#0b0b0c` | Every border and hard shadow; darkest surface |
| Page background | `#141416` | App background |
| Surface | `#1f1f22` | Cards, panels, inputs on dark |
| Text primary | `#ffffff` | |
| Text secondary | `#9a9aa2` | Meta lines (set, number, condition) |
| Text tertiary | `#6f6f77` | Fine print, disabled, strikethrough prices |
| Text on light pink | `#8a3f68` | Meta text inside light-pink cards |

### Typography
- **Display** — `Baloo 2`, weight 800 (700 for card titles). Used for the logo, H1s, section headings, product names. Letter-spacing `-1px` to `-2px` on large sizes, line-height `.92–1.1`.
- **Pixel / UI labels** — `Press Start 2P`, 8–12px for chips, badges, tabs, buttons, prices-as-score, table rows; 18–34px only for HIGH SCORES heading and the big price.
- **Body** — `DM Sans`, 400/500/700, 12–19px, line-height 1.4–1.6.
- ⚠️ **Press Start 2P has no accented glyphs.** Every string rendered in it must be accent-free (`HOME`, `SALDO`, `PRE-VENDA`, `POKEMON`, `ESTADO`, `HISTORICO DE PRECO`). Accents are fine in Baloo 2 / DM Sans. Enforce this in copy review or with a lint rule.

Type sizes used: 76px (hero H1), 54/52/50px (product H1), 44px (section H2), 38px (browse H2), 36px (footer logo), 34px (header logo, big price), 30px (game card), 19–21px (card titles, hero lead), 15–16px (body), 12–13px (meta), 8–12px (pixel labels).

### The "arcade sticker" style (core visual rule)
Almost every surface is: **solid brand color + `border: 4–5px solid #0b0b0c` + `border-radius: 10–18px` + hard offset shadow `box-shadow: 4–8px 4–8px 0 #0b0b0c`** (no blur, no spread). Small chips use 3px borders, no shadow. Buttons: 4px border + 5–6px hard shadow.
- Radii: 6px (thumb art), 8–10px (chips, inputs, buttons), 14px (cards/panels), 16–18px (large panels, game cards), 50% (mascot circle).
- Spacing scale in use: 6, 8, 10, 12, 14, 16, 18, 20, 22, 24, 26, 28, 32, 40, 48, 64, 80px. Page gutter 28px, max width **1360px** (cart screen 1100px).
- Text shadows on the logo/hero H1 are stacked hard offsets: `3px 3px 0 #1355B3, 5px 5px 0 #0b0b0c` (logo) and `5px 5px 0 #F6559B, 9px 9px 0 #0b0b0c` (hero H1).

### Effects
- **Scanlines**: fixed full-screen overlay, `pointer-events:none`, `z-index:90`, `repeating-linear-gradient(to bottom, rgba(0,0,0,.20) 0 1px, transparent 1px 3px)`, `mix-blend-mode:multiply`. Toggleable.
- **blink** keyframes (1.4–1.6s, `steps(1)`): opacity 1 → .15, used on the announcement dot and "INSERT COIN".
- **bob** keyframes (4s ease-in-out infinite): `translateY(0) rotate(-3deg)` ↔ `translateY(-12px) rotate(3deg)` on the hero mascot.
- Placeholder art fill: `repeating-linear-gradient(90deg | 45deg | 135deg, rgba(11,11,12,.12–.14) 0 10–14px, transparent …)` over a brand color.
- Respect `prefers-reduced-motion`: disable bob/blink.

## Global chrome (all screens)

1. **Announcement bar** — bg `#1355B3`, bottom border 4px `#0b0b0c`, 8px/28px padding, Press Start 2P 9px, color `#FDC4E5`: blinking dot, "FRETE GRATIS ACIMA DE R$ 250", "ENVIO EM 24H", "+18.400 CARTAS NO ESTOQUE"; right-aligned white "P1 · CONVIDADO".
2. **Header** — bg `#F6559B`, bottom border 6px `#0b0b0c`, 16px/28px padding, flex gap 24px:
   - Logo: 52px rounded-14px mascot tile (blue bg, 4px ink border, 4px hard shadow) + "COLLECTA" in Baloo 800 34px white with the stacked shadow. Click → Home.
   - Search: white pill, 4px ink border, radius 10px, 4px hard shadow, max-width 520px; input 12/14px DM Sans; `BUSCAR` button light-pink with a 4px ink left border.
   - Right: "SALDO / R$ 0,00" (Press Start 2P 8–10px, label `#FDC4E5`) and a blue **CARRINHO** button with a light-pink pill count badge.
3. **Screen/tab bar** — bg `#0b0b0c`, bottom border 4px `#1355B3`. Tabs: Press Start 2P 10px, 16/22px padding, 2px `#141416` divider; active = pink bg + white text, inactive = `#8a8a92` on black. Right side: blinking pink "INSERT COIN". In production these tabs are **routes**, not local state (`/`, `/singles`, `/carta/:slug`, `/selado/:slug`, `/carrinho`).
4. **Footer** — bg `#F6559B`, top border 6px ink, 48/28px padding, 4-col grid `1.2fr .6fr .6fr .6fr`: brand block (Baloo 36px logo + 14px white paragraph) and 3 link columns with Press Start 2P 9px ink headings (`COMPRAR`, `AJUDA`, `COLLECTA`) and 14px white links. Bottom strip: 3px ink top border, centered Press Start 2P 8px "© 2026 COLLECTA · GAME OVER? NUNCA. CONTINUE (9)".

## Screens

### 1. Home (`/`)
- **Hero** — bg `#1355B3`, bottom border 6px ink, overflow hidden, padding 64/28/72px, grid `1.15fr .85fr`, gap 40px, items centered. Three stacked background layers (deliberately **non-tiling** — an earlier repeated-pattern version was rejected):
  1. `radial-gradient(120% 150% at 78% 45%, rgba(246,85,155,.55) 0%, rgba(246,85,155,0) 55%)`
  2. vertical grid lines: `linear-gradient(to right, rgba(255,255,255,.06) 1px, transparent 1px)`, `background-size: 96px 100%`
  3. bottom fade: 120px `linear-gradient(to top, rgba(11,11,12,.35), transparent)`
- Left column: light-pink sticker badge "PLAYER 1 · READY"; H1 "SUA COLEÇÃO / COMEÇA AQUI" (Baloo 800, 76px, white, stacked pink+ink shadow); 19px `#FDC4E5` lead with the three game names bold white; two CTAs — pink **EXPLORAR SINGLES** → browse, light-pink **VER SELADOS** → sealed page.
- Right column: 300px mascot circle (6px ink border) with a pink 300px circle offset behind it, `bob` animation, plus a white rotated(-4deg) sticker "PEGA ESSA!" at bottom-right.
- **Game select** — H2 "SELECIONE SEU JOGO" + pink pixel "3 UNIVERSOS"; 3-col grid gap 22px. Each cabinet card: brand bg (pink / blue / light pink), 5px ink border, radius 18px, 8px hard shadow; 186px art area with 135° stripes and a monospace label chip, 5px ink bottom border; body 20/22/22px with Baloo 800 30px name and pixel 9px count (`9.812 CARTAS`, `5.140 CARTAS`, `3.448 CARTAS`). Click → browse filtered by game.
- **Singles em destaque** — H2 + "VER TODAS ›" link (pixel 10px, 3px pink bottom border); 4-col grid gap 20px of **Single Card** components (below), 8 items.
- **Produto selado** — H2 + 4-col grid of **Sealed Card** components, 4 items.
- **HIGH SCORES** panel (toggleable) — bg `#0b0b0c`, 5px pink border, radius 18px, padding 30/32px; Press Start 2P 18px `#FDC4E5` heading + 13px `#6f6f77` subtitle "Colecionadores com mais trocas neste mês"; rows grid `50px 1fr 130px 110px`, gap 16px, padding 14/12px, pixel 11px, alternating row bg `#1f1f22`/`#141416`, position in pink, trades in light pink, score right-aligned. Data: 1ST BRUNO_TCG 312 TROCAS 98.420 · 2ND MARI.PULLS 287 91.105 · 3RD DOJO_CARDS 241 84.660 · 4TH RIFT_KID 198 72.310 · 5TH SEALED_SP 176 68.940.
- **Removed by the client:** the "VENDIDO AGORA" live-sold ticker. Do not reintroduce it.

### 2. Browse — singles (`/singles`)
Grid `270px 1fr`, gap 28px, padding 36/28px, items start.
- **Filter sidebar** — `#1f1f22`, 4px ink border, radius 14px, 6px hard shadow, padding 22px. Pink pixel 12px "FILTROS". Groups (light-pink pixel 9px heading, 22px apart) with wrap chips gap 8px: **JOGO** (Pokémon•, One Piece, Riftbound) · **TIPO** (Singles•, Selados) · **ESTADO** (PSA 10•, PSA 9, NM•, LP) · **IDIOMA** (EN•, JP, PT) · **RARIDADE** (SIR, Alt Art•, Holo). Chip = 3px ink border, radius 8px, 7/10px padding, 12px/700; selected = pink bg + white, else `#0b0b0c` bg + `#c9c9d1`. Then **PRECO**: 14px track `#0b0b0c` with 3px pink border radius 10px, filled pink from 8% to 66%, labels "R$ 12" / "R$ 900". (Chips are multi-select; price is a dual-thumb range.)
- **Toolbar** — H2 "Pokémon · Singles" (Baloo 800 38px) + pixel 9px `1.284 RESULTADOS`; right: sort chips `RELEVANCIA` (active pink) / `MENOR PRECO` / `NOVIDADES`.
- **Grid** — 4 cols, gap 20px, 12 Single Cards; each card's bottom row shows price + a blue `+ ADD` pixel chip (adds to cart without leaving the page).
- **Pagination** — centered, gap 10px, chips `‹ 1 2 3 … 32 ›`, current = pink.

### 3. Single card page (`/carta/:slug`)
Padding 36/28px. Breadcrumb pixel 9px `#6f6f77`: "POKEMON > OBSIDIAN FLAMES > CHARIZARD EX". Grid `.9fr 1.1fr`, gap 40px.
- **Left** — pink frame (5px ink border, radius 18px, 8px shadow, padding 24px) containing a 5:7 blue art area with 4px ink border, radius 8px, placeholder label "FOTO REAL DA CARTA / SLAB PSA · FRENTE". Below: 4 square thumbs, gap 12px, `#1f1f22`, 3px border (active = pink, others ink), radius 10px, pixel 7px labels FRENTE / VERSO / CANTOS / SELO PSA.
- **Right**
  - Badge row: `PSA 10 GEM MT` (light pink), `EN` (blue/white), `SO 1 UNIDADE` (black bg, pink text + pink border).
  - H1 "Charizard ex" (Baloo 800 54px) + 16px `#9a9aa2` "Obsidian Flames · 199/165 · Special Illustration Rare".
  - **Buy box** (`#1f1f22`, 4px ink border, radius 14px, 6px shadow, padding 24px): price `R$ 420,00` in Press Start 2P 34px pink + struck `R$ 489,00` 14px; 13px line "ou 12x de R$ 38,90 · média de mercado R$ 455"; then quantity stepper (white, 4px ink border, radius 10px, light-pink −/+ pads, pixel 14px value), pink **ADICIONAR AO CARRINHO** (flex 1, min-width 220px), and a wishlist ♥ button (transparent, 4px light-pink border).
  - **Seller bar** — blue, 4px ink border, radius 14px, 6px shadow, padding 18/20px: 48px mascot avatar, "Collecta Oficial" (Baloo 700 20px) + pixel 9px "99,4% POSITIVO · 4.821 VENDAS", right chip "ENVIA EM 24H".
  - **Price history** — `#1f1f22` panel, pixel 10px light-pink title "HISTORICO DE PRECO - 6 MESES"; 6 bars in a 110px-tall flex row, gap 8px, each bar 3px ink border, radius 4px 4px 0 0, heights 38/52/46/68/84/92% for FEV…JUL, latest bar pink, others light pink, pixel 7px labels.
  - **Outros vendedores** — pink pixel 10px heading; rows grid `1fr auto auto`, gap 16px, `#1f1f22`, 3px ink border, radius 10px, padding 14/16px: seller (15px/700 white) + condition (12px `#9a9aa2`), light-pink pixel 12px price, blue `+ ADD` chip. Data: DojoCards SP · PSA 10 · envia em 48h · R$ 439,00 / Bruno TCG · PSA 9 · 24h · R$ 352,00 / Mari Pulls · NM sem grade · 72h · R$ 298,00.

### 4. Sealed product page (`/selado/:slug`)
Breadcrumb "ONE PIECE > SELADOS > BOOSTER BOX OP-09". Grid `1fr 1fr`, gap 40px.
- **Left** — light-pink frame (5px ink border, radius 18px, 8px shadow, padding 26px) with a 4:3 blue art area (45° stripes) labelled "FOTO DA CAIXA / LACRE VISÍVEL"; below, 3 white square thumbs (3px ink border, radius 10px) LACRE / LATERAL / CODIGO.
- **Right** — badges `PRE-VENDA` (pink/white) + `LACRADO` (light pink); H1 "Booster Box OP-09" (Baloo 800 50px); 16px meta "Emperors in the New World · 24 packs · Japonês".
  - Buy box: pixel 32px pink `R$ 899,00`; 13px "12x de R$ 83,20 · lançamento 12/09"; light-pink pixel 9px "RESTAM 7 DE 40 CAIXAS"; stock bar 18px tall, `#0b0b0c` bg, 3px pink border, radius 10px, fill 82% `repeating-linear-gradient(90deg,#F6559B 0 10px,#FDC4E5 10px 20px)`; stepper + blue **GARANTIR PRE-VENDA**.
  - Spec grid 2×2, gap 14px: `#0b0b0c` cards, 3px blue border, radius 10px, padding 16px, pixel 8px light-pink key + 15px/700 white value — CONTEUDO "24 packs de 12 cartas" · IDIOMA "Japonês" · LANCAMENTO "12 set 2026" · GARANTIA "Lacre original de fábrica".

### 5. Cart (`/carrinho`)
Max-width 1100px, padding 36/28px. H1 "Carrinho" (Baloo 800 52px) + pink pixel 9px "3 ITENS · 1 FICHA PARA CONTINUAR". Grid `1.4fr .6fr`, gap 28px.
- **Line items** — `#1f1f22`, 4px ink border, radius 14px, 5px shadow, padding 16px, grid `78px 1fr auto auto`, gap 18px: 5:7 art thumb (3px ink border, radius 6px, brand-color stripes), name (Baloo 700 21px) + 12px meta + pixel 8px light-pink seller, compact stepper, right column with pixel 13px pink line total and a 12px `#6f6f77` "remover". Items: Charizard ex (Obsidian Flames · 199/165 · PSA 10, COLLECTA OFICIAL, qty 1, R$ 420,00) · Booster Box OP-09 (pré-venda 12/09, qty 1, R$ 899,00) · Jinx, Estopim (Riftbound Origins · ORI-101 · NM, RIFT_KID, qty 2, R$ 264,00).
- **Free-shipping nudge** — blue bar, 4px ink border, radius 14px, padding 16/20px, 44px mascot avatar + 15px white text "Faltam **R$ 22,00** para o frete grátis. Bora completar o set?" (R$ value in `#FDC4E5`). Threshold R$ 250.
- **Summary panel** — `#0b0b0c`, 5px pink border, radius 16px, padding 26px: light-pink pixel 11px "RESUMO"; rows 15px `#c9c9d1` with white values — Subtotal R$ 1.228,00 · Frete (Sedex) R$ 28,00 · Cupom ARCADE10 −R$ 122,80 (pink); 3px pink divider; TOTAL pixel 10px vs pixel 22px pink R$ 1.133,20; 12px "em até 12x de R$ 104,90"; full-width pink **INSERIR FICHA - PAGAR** button with a 4px **white** border and `box-shadow: 0 0 0 4px #0b0b0c` (double-ring arcade button); fine print pixel 8px "PIX - CARTAO - BOLETO / COMPRA PROTEGIDA COLLECTA".

## Reusable components

### Single Card (grid item)
`#1f1f22`, 4px ink border, radius 14px, 6px hard shadow, flex column, overflow hidden. Art: `aspect-ratio:5/7`, brand-color bg with 90° stripes, 4px ink bottom border, centered monospace label "FOTO DA CARTA"; top-left grade badge (`#0b0b0c` bg, `#FDC4E5` pixel 8px), top-right language badge (white bg, ink text) — language badge only in the featured grid. Body padding 14px, gap 7–8px: name (Baloo 700 18–19px white), `set · number` (12px `#9a9aa2`), footer row pushed to bottom with pixel 11–12px pink price and either stock text (11px `#6f6f77`) or a blue `+ ADD` chip. Whole card is a link to the product page; `+ ADD` must `stopPropagation`.

### Sealed Card (grid item)
Light-pink card, 4px ink border, radius 14px, 6px shadow. Art: 196px tall, blue bg, 45° stripes, 4px ink bottom border, centered label; top-left pink pixel badge (`PRE-VENDA`, `SO RESTAM 4`, `NOVO`, `HOT`). Body padding 16px: name (Baloo 700 20px ink), 12px `#8a3f68` meta, pixel 12px blue price.

### Quantity stepper
White, 4px ink border, radius 10px, overflow hidden; −/+ pads light pink, full-height, pixel 14px ink; value pixel 14px, min-width 24px centered. Min 1; cap at available stock (prototype has no cap).

### Chip / badge
3px ink border, radius 8px; filter chips 7/10px padding 12px/700; pixel badges 5–6/6–9px padding, Press Start 2P 8–10px, no radius on product badges.

## Interactions & Behavior
- Tab bar switches screens (→ routes in production); every switch scrolls to top.
- Home game cards and all product cards → their product/browse pages. Header logo → home. "VER TODAS ›" → browse.
- Stepper +/− changes quantity (floor 1). **ADICIONAR AO CARRINHO / GARANTIR PRE-VENDA** increments the header cart badge by the chosen quantity; production should also show a toast/mini-cart and persist the cart server-side or in localStorage.
- Filters, sort, pagination, search and "remover" are **visual only** in the prototype — implement as real query-param-driven state (`?jogo=pokemon&estado=psa10&min=12&max=900&sort=relevancia&page=2`), with server-side filtering and debounced search + typeahead.
- Hover states are not specified in the prototype; recommended pattern in keeping with the style: translate the element 2px toward its shadow and shrink the shadow by 2px (`transform: translate(2px,2px); box-shadow: 4px 4px…` → `2px 2px`), and on `:active` flatten the shadow to 0. Focus: 3px `#FDC4E5` outline with 2px offset — never remove focus rings.
- Loading: skeleton cards with the same stripe placeholder, no spinners. Empty states: mascot illustration + Baloo headline + pixel-font hint (e.g. "NENHUMA CARTA ENCONTRADA / TENTE OUTRO FILTRO") — the client asked for the mascot in empty states.
- Errors: pink 4px-border panel, ink bg, pixel-font title, DM Sans body + retry button.
- Responsive: prototype is desktop-first at 1360px. Breakpoints to add — ≥1200px as designed; 900–1199px singles grid 3 cols, product pages stay 2-col; <900px single column, filters become a full-screen sheet behind a `FILTROS` button, hero H1 → 44–52px, mascot 200px, sticky bottom buy bar on product pages, cart summary below the items. Reduce hard shadows to 3–4px on mobile.
- Accessibility: the pink/white and light-pink/ink pairings pass; **avoid `#F6559B` text on `#1355B3`**. Press Start 2P below 10px is decorative — never use it for essential long-form text, and keep real `aria-label`s on icon-only buttons (♥). Badges must not be the only carrier of information (add text alternatives).

## State Management
Prototype state: `{ screen, qty, cartCount }`.
Production needs: current route + product slug; catalog query (game, type, condition, language, rarity, price range, sort, page, search term); product detail (variants/offers by seller); cart (line items with product id, seller id, condition, qty, unit price, plus coupon and shipping quote); user/session (balance "SALDO", wishlist); async states per fetch.
Data fetching: catalog list (paginated + facet counts), product detail + other-sellers list, price history series (6 monthly points), sealed stock/preorder counts, cart mutations, coupon validation, shipping quote by CEP. Prices are integers in cents, formatted `pt-BR` (`R$ 1.133,20`).

## Assets
- `assets/mascot.png` — the Collecta mascot (curly-haired character holding cards), cropped from the client's brand sheet. Used in the header logo tile, hero circle, seller bar, cart nudge. Ask the client for the original vector/transparent-PNG mascot; the crop carries a patterned background.
- `assets/pattern.png` — tiled COLLECTA wordmark pattern from the brand sheet. **Currently unused** — the client rejected the repeated pattern in the hero. Keep out of large areas; acceptable at small scale (app icons, packing-slip graphics).
- Fonts: Google Fonts — `Baloo 2` (600/700/800), `Press Start 2P`, `DM Sans` (400/500/700). Self-host in production.
- Product imagery: all placeholders. Real photos needed for cards (5:7) and sealed product (4:3), plus card back / corners / PSA-label detail shots.
- Brand colors come from the client's brand sheet (`uploads/pasted-1784917840244-0.png`, included).

## Copy notes
All UI copy is pt-BR and part of the design — reuse the exact strings in this README. Keep the arcade voice: SALDO, P1 · CONVIDADO, PLAYER 1 · READY, INSERT COIN, HIGH SCORES, INSERIR FICHA, "GAME OVER? NUNCA. CONTINUE (9)", "PEGA ESSA!". Remember the accent restriction on Press Start 2P strings.

## Screenshots
`screenshots/` contains one capture per screen (desktop, viewport ~910px wide so the layout is compressed vs. the 1360px design width):
- `01-home.png` · `02-browse-singles.png` · `03-single-card.png` · `04-sealed-product.png` · `05-cart.png`

Open the HTML prototype in a browser at ≥1400px for the intended proportions.

## Files
- `Collecta Marketplace.dc.html` — the full prototype (all 5 screens). Template markup first, then the `Component` class holding the view model / mock data at the bottom of the file.
- `support.js` — prototype runtime only. **Not** to be ported.
- `assets/mascot.png`, `assets/pattern.png` — artwork.
- `brand/collecta-brand-sheet.png` — the client's original brand reference.
