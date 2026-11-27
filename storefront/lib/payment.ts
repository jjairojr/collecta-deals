export type PaymentMethod = "pix" | "card";

// Até 6x no cartão, para todos os produtos, sempre sobre o preço no
// cartão (decisão do Jairo, 27/11). O checkout manda o mesmo limite ao Mercado Pago.
export const MAX_INSTALLMENTS = 6;

// "Sem juros" só vale se a conta do Mercado Pago estiver no parcelamento sem
// acréscimos (o vendedor paga o custo). Sem isso, o Checkout Pro cobra os juros
// do cliente em 2x a 6x, e o site não pode prometer. Liga com
// NEXT_PUBLIC_PARCELAS_SEM_JUROS=1 na Vercel (inlined no build).
export const INTEREST_FREE = process.env.NEXT_PUBLIC_PARCELAS_SEM_JUROS === "1";
/** " sem juros" quando a conta absorve os juros; vazio enquanto não. */
export const SEM_JUROS = INTEREST_FREE ? " sem juros" : "";
/** Mesmo limite de MAX_INSTALLMENTS: o preço no cartão não muda mais o parcelamento. */
export const CARD_PRICED_INSTALLMENTS = MAX_INSTALLMENTS;

export function installmentsFor(_items: { cardPrice?: number }[]): number {
  return MAX_INSTALLMENTS;
}

export function unitPriceFor(item: { price: number; cardPrice?: number }, method: PaymentMethod): number {
  return method === "card" ? (item.cardPrice ?? item.price) : item.price;
}

// The checkout lets a card pay in up to `max` installments; each one has to be at
// least MIN_INSTALLMENT_CENTS, so a cheap card shows fewer. The value is the
// price split evenly: no interest is added on the site.
const MIN_INSTALLMENT_CENTS = 500;

export function installmentPlan(cents: number, max = MAX_INSTALLMENTS): { count: number; each: number } | null {
  const count = Math.min(max, Math.floor(cents / MIN_INSTALLMENT_CENTS));
  if (count < 2) {
    return null;
  }
  return { count, each: Math.round(cents / count) };
}

export function shipsFromLabel(date: string): string {
  const [, month, day] = date.split("-");
  return day && month ? `${day}/${month}` : date;
}
