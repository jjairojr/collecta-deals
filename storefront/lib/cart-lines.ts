import type { Accessory, CartLine, Sealed, Single } from "@/lib/types";

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
    stock,
    imageURL: item.imageURL,
  };
}
