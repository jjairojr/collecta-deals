# Google Analytics 4 — Collecta storefront

O código já dispara todos os eventos abaixo. Tudo fica **desligado** até `NEXT_PUBLIC_GA_ID` existir no ambiente — sem a env, nenhum script do Google é carregado.

## Setup no dashboard (fazer uma vez)

1. **Criar a propriedade GA4** em [analytics.google.com](https://analytics.google.com):
   - Admin → Criar propriedade → nome "Collecta", fuso **America/Sao_Paulo**, moeda **BRL**.
   - Criar um **Web data stream** para `https://collectatcg.com.br` → copiar o **Measurement ID** (`G-XXXXXXXXXX`).
2. **Configurar a env na Vercel**: Settings → Environment Variables → `NEXT_PUBLIC_GA_ID = G-XXXXXXXXXX` (Production e Preview) → redeploy.
3. **Enhanced Measurement** (no data stream → engrenagem): deixar tudo ligado. Em *Site search*, garantir que o parâmetro `q` está na lista (é o que a busca do site usa na URL `/singles?q=...`).
4. **Key events (conversões)**: Admin → Events → marcar como key event:
   - `generate_lead` — clique em "Fechar pedido no WhatsApp" (conversão principal, leva o valor do carrinho).
   - `begin_checkout` — mesmo clique, no formato e-commerce (com itens).
   - Opcional: `add_to_cart`.
5. **Custom dimensions** (Admin → Custom definitions → Create, escopo **Event**):
   | Dimension name | Event parameter |
   |---|---|
   | Filter type | `filter_type` |
   | Filter value | `filter_value` |
   | Sort | `sort_id` |
   | Contact origin | `origin` |
   | Lead source | `lead_source` |
6. **Retenção de dados**: Admin → Data settings → Data retention → **14 months**.
7. **Google Signals** (opcional): Admin → Data settings → Data collection → ativar, para demografia/interesses e cross-device.
8. **Search Console**: Admin → Product links → Search Console link, para ver queries orgânicas dentro do GA.

## Eventos disparados pelo código

### E-commerce (taxonomia recomendada do GA4 — alimenta os relatórios de Monetização)

| Evento | Quando | Onde no código |
|---|---|---|
| `page_view` | toda navegação (SPA incluída) | automático via `<GoogleAnalytics>` em `app/layout.tsx` |
| `view_item_list` | grade de singles/selados renderizada (browse, home, selados por jogo) | `SinglesBrowse`, `SealedGrid`, `TrackViewItemList` |
| `select_item` | clique num card de produto | `SingleCard`, `SealedCard` |
| `view_item` | abertura de página de produto | `SingleDetailView`, `SealedDetailView` |
| `add_to_cart` | + ADD no card ou botão da página de produto (com quantidade) | `SingleCard`, `SingleDetailView`, `SealedDetailView` |
| `remove_from_cart` | "remover" no carrinho | `CartView` |
| `view_cart` | abertura do carrinho com itens | `CartView` |
| `begin_checkout` | clique em "Fechar pedido no WhatsApp" (itens + cupom + total) | `CartView` |
| `generate_lead` | mesmo clique, com `value` = total e `lead_source=whatsapp_checkout` | `CartView` |
| `add_to_wishlist` | coração na página da carta | `SingleDetailView` |
| `search` | busca no header | `Header` |

Itens carregam: `item_id` (slug), `item_name`, `item_category` (jogo), `item_category2` (`single`/`sealed`), `item_category3` (set/coleção), `item_variant` (condição · idioma), `price` em reais, `quantity`. Moeda sempre `BRL`.

### Customizados

| Evento | Parâmetros | Quando |
|---|---|---|
| `filter_change` | `filter_type` (jogo/colecao/estado/idioma), `filter_value`, `filter_active` | toggle de filtro no browse |
| `sort_change` | `sort_id` | troca de ordenação |
| `apply_coupon` | `coupon`, `discount` | cupom válido aplicado no carrinho |
| `whatsapp_contact` | `origin` (footer) | clique em "Fale no WhatsApp" |

## Validação

- Local: `NEXT_PUBLIC_GA_ID=G-XXXXXXXXXX npm run dev` — em dev todos os eventos vão com `debug_mode: true`, então aparecem em **Admin → DebugView** sem sujar os relatórios.
- Produção: aba Network filtrando `collect?v=2` mostra cada hit com o nome do evento (`en=`).

## Notas

- Sem banner de consentimento por decisão de produto (2026-07); se um dia entrar, implementar Google Consent Mode v2 antes da tag.
- Checkout é handoff para WhatsApp — não existe `purchase` no site. A venda fechada vive no WhatsApp; `generate_lead`/`begin_checkout` são o fim do funil mensurável. Se quiser medir conversão real, registrar as vendas manualmente ou via Measurement Protocol depois.
