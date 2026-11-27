export type PaymentMethod = "pix" | "card";

export const MAX_INSTALLMENTS = 12;
export const CARD_PRICED_INSTALLMENTS = 6;

export function installmentsFor(items: { cardPrice?: number }[]): number {
  return items.some((i) => i.cardPrice !== undefined) ? CARD_PRICED_INSTALLMENTS : MAX_INSTALLMENTS;
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
