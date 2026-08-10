# Vendetta (Riftbound) — Liga → dashboard

Fonte: export do estoque da loja 866280 (tcg=19, edição VEN) + pedidos do admin da Liga.
Destino: prod `https://collecta-deals-production.up.railway.app` (`?game=riftbound`).

## Decisões do dono

- 12 caixas de Booster Vendetta abertas (05/08).
- Custo rateado proporcional ao mercado TCGplayer.
- As 22 linhas com 6 un. e preço vazio são placeholder na Liga → não entram.

## Passos

- [ ] 1. Cadastrar 191 linhas / 872 unidades (821 em estoque + 51 já vendidas)
      `POST /api/trades?game=riftbound` — set VEN, condition NM, buyDate 2026-08-05,
      store "Abertura Caixa Vendetta", delivered true, askBRL = preço da Liga,
      tcgUrl = produto TCGplayer casado (carrega a arte), buyBRL = rateio.
- [ ] 2. Dar baixa do pedido #11600287 (18 linhas, 51 un., pago 06/08)
      `POST /api/trades/{id}/sell` — sellPrice com o desconto de 5% do pedido,
      buyer "Lucas Simoes Gomes — Liga #11600287", sellDate 2026-08-06.
- [ ] 3. Marcar a flag Liga dos holdings que sobraram
      `POST /api/trades/liga` — ligaQty/ligaPriceBRL = o que está na loja hoje.
- [ ] 4. Baixar o selado: "Caixa de Booster - Vendetta" qty 30 → 18
      `PUT /api/trades/062ac95651805e23` (registro inteiro, não só qty).
- [ ] 5. Conferir: contagem, custo total, artes, Estoque e Realized do portfólio.

## Pendências para o dono

- 22 linhas de chase na Liga com 6 un. e sem preço (Signature/Overnumbered) —
  precisam da quantidade real; hoje ficam fora do dashboard e invisíveis na loja.
- Os hits reais dos 12 boxes não estão em lugar nenhum: o lote cadastrado vale
  R$ 3.198 de mercado contra R$ 6.558 de custo.

## Review

(preencher ao terminar)
