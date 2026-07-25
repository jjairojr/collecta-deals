"use client";

import { sendGAEvent } from "@next/third-parties/google";
import type { CartLine, Sealed, Single } from "@/lib/types";

export const GA_ID = process.env.NEXT_PUBLIC_GA_ID;

const CURRENCY = "BRL";

export interface GAItem {
  item_id: string;
  item_name: string;
  item_category?: string;
  item_category2: "single" | "sealed";
  item_category3?: string;
  item_variant?: string;
  item_brand?: string;
  price: number;
  quantity: number;
}

type GAValue = string | number | boolean | GAItem[];
type GAParams = Record<string, GAValue | undefined>;

function send(name: string, params: GAParams): void {
  if (!GA_ID) {
    return;
  }
  if (process.env.NODE_ENV === "development") {
    sendGAEvent("event", name, { ...params, debug_mode: true });
    return;
  }
  sendGAEvent("event", name, params);
}

function reais(cents: number): number {
  return Math.round(cents) / 100;
}

function itemsValue(items: GAItem[]): number {
  const cents = items.reduce(
    (sum, i) => sum + Math.round(i.price * 100) * i.quantity,
    0,
  );
  return reais(cents);
}

export function singleItem(item: Single, quantity = 1): GAItem {
  return {
    item_id: item.slug,
    item_name: item.name,
    item_category: item.game,
    item_category2: "single",
    item_category3: item.set,
    item_variant: [item.condition, item.language].filter(Boolean).join(" · "),
    price: reais(item.price),
    quantity,
  };
}

export function sealedItem(item: Sealed, quantity = 1): GAItem {
  return {
    item_id: item.slug,
    item_name: item.name,
    item_category: item.game,
    item_category2: "sealed",
    item_category3: item.set,
    item_variant: item.language,
    price: reais(item.price),
    quantity,
  };
}

export function cartLineItem(l: CartLine): GAItem {
  return {
    item_id: l.slug,
    item_name: l.name,
    item_category2: l.kind,
    item_variant: l.meta,
    item_brand: l.seller,
    price: reais(l.price),
    quantity: l.qty,
  };
}

export function catalogItem(item: Single | Sealed): GAItem {
  return item.kind === "single" ? singleItem(item) : sealedItem(item);
}

export function trackViewItem(item: GAItem): void {
  send("view_item", {
    currency: CURRENCY,
    value: itemsValue([item]),
    items: [item],
  });
}

export function trackViewItemList(
  items: GAItem[],
  listId: string,
  listName: string,
): void {
  send("view_item_list", {
    item_list_id: listId,
    item_list_name: listName,
    items,
  });
}

export function trackSelectItem(item: GAItem, listName?: string): void {
  send("select_item", { item_list_name: listName, items: [item] });
}

export function trackAddToCart(item: GAItem): void {
  send("add_to_cart", {
    currency: CURRENCY,
    value: itemsValue([item]),
    items: [item],
  });
}

export function trackRemoveFromCart(line: CartLine): void {
  const item = cartLineItem(line);
  send("remove_from_cart", {
    currency: CURRENCY,
    value: itemsValue([item]),
    items: [item],
  });
}

export function trackViewCart(lines: CartLine[]): void {
  const items = lines.map(cartLineItem);
  send("view_cart", {
    currency: CURRENCY,
    value: itemsValue(items),
    items,
  });
}

export function trackBeginCheckout(
  lines: CartLine[],
  totalCents: number,
  coupon?: string,
): void {
  send("begin_checkout", {
    currency: CURRENCY,
    value: reais(totalCents),
    coupon,
    items: lines.map(cartLineItem),
  });
}

export function trackGenerateLead(totalCents: number): void {
  send("generate_lead", {
    currency: CURRENCY,
    value: reais(totalCents),
    lead_source: "whatsapp_checkout",
  });
}

export function trackSearch(term: string): void {
  send("search", { search_term: term });
}

export function trackAddToWishlist(item: GAItem): void {
  send("add_to_wishlist", {
    currency: CURRENCY,
    value: itemsValue([item]),
    items: [item],
  });
}

export function trackApplyCoupon(code: string, discountCents: number): void {
  send("apply_coupon", {
    coupon: code,
    currency: CURRENCY,
    discount: reais(discountCents),
  });
}

export function trackFilterChange(
  filterType: string,
  filterValue: string,
  active: boolean,
): void {
  send("filter_change", {
    filter_type: filterType,
    filter_value: filterValue,
    filter_active: active,
  });
}

export function trackSortChange(sortId: string): void {
  send("sort_change", { sort_id: sortId });
}

export function trackWhatsAppContact(origin: string): void {
  send("whatsapp_contact", { origin });
}
