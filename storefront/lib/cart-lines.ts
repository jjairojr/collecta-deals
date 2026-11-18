import type { Accessory, CartLine, KitOffer, Sealed, Single } from "@/lib/types";

export const STORE_SELLER = "Collecta Oficial";

export function singleToCartLine(item: Single): Omit<CartLine, "qty"> {
  return {
    slug: item.slug,
    sku: item.sku,
    kind: "single",
    name: item.name,
    meta: [item.set, item.number, item.variant, item.condition].filter(Boolean).join(" · "),
    seller: STORE_SELLER,
    price: item.price,
    stock: item.qty,
    imageURL: item.imageURL,
  };
}

export function sealedToCartLine(item: Sealed | Accessory, stock: number): Omit<CartLine, "qty"> {
  return {
    slug: item.slug,
    sku: item.sku,
    kind: item.kind,
    name: item.name,
    meta: [...new Set([item.set, item.meta].filter(Boolean))].join(" · "),
    seller: STORE_SELLER,
    price: item.price,
    cardPrice: item.cardPrice,
    shipsFrom: item.shipsFrom,
    stock,
    imageURL: item.imageURL,
  };
}

export function kitLabel(units: number): string {
  return `Kit ${units} un`;
}

export function kitStock(stock: number, units: number): number {
  return Math.floor(stock / units);
}

export function kitToCartLine(
  item: Sealed | Accessory,
  kit: KitOffer,
  stock: number,
): Omit<CartLine, "qty"> {
  const base = sealedToCartLine(item, kitStock(stock, kit.units));
  return {
    ...base,
    sku: kit.sku,
    name: `${item.name} · ${kitLabel(kit.units)}`,
    meta: [base.meta, kitLabel(kit.units)].filter(Boolean).join(" · "),
    price: kit.price,
    cardPrice: kit.cardPrice,
  };
}
