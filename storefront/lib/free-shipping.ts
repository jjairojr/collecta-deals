import { brl } from "@/lib/format";

/** "R$ 600" for whole reais, "R$ 599,90" otherwise. */
export function freeShippingMinLabel(minBRL: number): string {
  return Number.isInteger(minBRL) ? `R$ ${minBRL}` : brl(Math.round(minBRL * 100));
}
