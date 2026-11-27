// Stock shown on product cards: the real count, as "RESTAM X" when it is low.
// Never "últimas unidades" or any urgency wording.
export const LOW_STOCK = 3;

export function stockLabel(qty: number): string {
  return qty > 0 && qty <= LOW_STOCK ? `RESTAM ${qty}` : `${qty} UN`;
}
