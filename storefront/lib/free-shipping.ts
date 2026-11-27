import { brl } from "@/lib/format";

/** "R$ 600" for whole reais, "R$ 599,90" otherwise. */
export function freeShippingMinLabel(minBRL: number): string {
  return Number.isInteger(minBRL) ? `R$ ${minBRL}` : brl(Math.round(minBRL * 100));
}

// The Correios stop offering PAC above a declared value: quoting through Melhor
// Envio (27/11/2026) PAC came back for R$ 4.200 in products and not for R$ 4.500
// (insurance = product value). Above this, the free PAC rule cannot apply, so a
// product page above it does not promise it.
export const PAC_MAX_ITEMS_BRL = 4200;

/** "frete grátis no PAC para todo o Brasil em compras a partir de R$ 600" */
export function freeShippingSentence(minBRL: number): string {
  return `frete grátis no PAC para todo o Brasil em compras a partir de ${freeShippingMinLabel(minBRL)}`;
}

/** Whether a product page priced at `priceCents` (Pix) can show the free PAC rule. */
export function pageShowsFreeShipping(minBRL: number, priceCents: number): boolean {
  return minBRL > 0 && priceCents / 100 <= PAC_MAX_ITEMS_BRL;
}

