# Login Collecta na vitrine + cupons (2026-10-13)

Plano: ~/.claude/plans/partitioned-purring-castle.md

## 1. API do app (collecta-app/server)
- [x] server/src/routes/store.ts (X-Store-Token, reserve, release)
- [x] registrar em index.ts, STORE_TOKEN no .env.example
- [x] yarn typecheck + curl

## 2. Go (opdeals)
- [x] internal/collecta client + testes
- [x] orders.Order: CollectaUserID, Coupons, CouponsReleasedAt, Totals.DiscountBRL
- [x] handleOrderCreate: couponIds, reserve, cancel/release, desconto na preferência
- [x] releaseCoupons + sweeper + applyPayment/patch
- [x] GET /api/orders?collectaUserId=
- [x] flags/env em cmd/server, checkoutfake
- [x] go build ./... && go test ./...

## 3. Vitrine (storefront)
- [x] lib/collecta.ts, lib/session.ts, lib/coupons.ts, lib/account.tsx, lib/referral.ts
- [x] app/api/conta/* (me, entrar, criar, sair, cupons, indicacao)
- [x] /conta, /conta/entrar, /conta/criar, /conta/pedidos
- [x] Header, layout AccountProvider
- [x] CheckoutView cupons + pré-preenchimento, api/pedido, OrderStatusView, whatsapp, analytics
- [x] tsc + npm run build

## Review

Tudo implementado nas três partes. Testado: `tsc --noEmit` (server do app), `go build ./... && go test ./...` (36 testes de api + 4 do cliente collecta, todos verdes), `tsc` + `npm run build` na vitrine (só /conta* viraram dinâmicas), E2E local com Chrome (API :3011, Go :8091 fake checkout, Next :3002): cadastro com ?ref=, /conta, checkout com cupom (R$120 − R$10 = R$110), pedido pago com desconto, /conta/pedidos, cancelamento admin libera o cupom no Postgres, cupom usado recusado com 409, sair/entrar.

Desvios do pedido original: `failed`/`expired` liberam só 30 min após `expiresAt` (pedido ainda pagável); retry do mesmo comprador também cancela pedido `failed`; desconto ≥ itens recusado (MP não aceita preferência zerada); ledger grava preço bruto.

Fora: `next lint` (não configurado no projeto), marcar cupom usado no caminho WhatsApp, envs no Railway/Vercel.
