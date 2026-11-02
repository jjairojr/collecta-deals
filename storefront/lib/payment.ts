export type PaymentMethod = "pix" | "card";

export const MAX_INSTALLMENTS = 12;
export const CARD_PRICED_INSTALLMENTS = 6;

export function installmentsFor(items: { cardPrice?: number }[]): number {
  return items.some((i) => i.cardPrice !== undefined) ? CARD_PRICED_INSTALLMENTS : MAX_INSTALLMENTS;
}

export function unitPriceFor(item: { price: number; cardPrice?: number }, method: PaymentMethod): number {
  return method === "card" ? (item.cardPrice ?? item.price) : item.price;
}

export function shipsFromLabel(date: string): string {
  const [, month, day] = date.split("-");
  return day && month ? `${day}/${month}` : date;
}
