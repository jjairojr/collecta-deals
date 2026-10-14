"use client";

import { sendGAEvent } from "@next/third-parties/google";
import type { Accessory, CartLine, ProductKind, Sealed, Single } from "@/lib/types";
import { pixelTrack, type PixelParams } from "@/lib/meta-pixel";

export const GA_ID = process.env.NEXT_PUBLIC_GA_ID;

const CURRENCY = "BRL";

export interface GAItem {
  item_id: string;
  item_name: string;
  item_category?: string;
  item_category2: ProductKind;
  item_category3?: string;
  item_variant?: string;
  item_brand?: string;
  price: number;
  quantity: number;
}

type GAValue = string | number | boolean | GAItem[];
type GAParams = Record<string, GAValue | undefined>;

const PIXEL_EVENTS: Record<string, string> = {
  view_item: "ViewContent",
  add_to_cart: "AddToCart",
  add_to_wishlist: "AddToWishlist",
  search: "Search",
  begin_checkout: "InitiateCheckout",
  generate_lead: "Lead",
  purchase: "Purchase",
  whatsapp_contact: "Contact",
  sign_up: "CompleteRegistration",
};

function pixelParams(params: GAParams): PixelParams {
  const items = Array.isArray(params.items) ? params.items : [];
  const out: PixelParams = {};
  if (typeof params.value === "number") {
    out.value = params.value;
    out.currency = CURRENCY;
  }
  if (typeof params.search_term === "string") {
    out.search_string = params.search_term;
  }
  if (items.length > 0) {
    out.content_type = "product";
    out.content_ids = items.map((i) => i.item_id);
    out.contents = items.map((i) => ({ id: i.item_id, quantity: i.quantity }));
    out.num_items = items.reduce((sum, i) => sum + i.quantity, 0);
    if (items.length === 1) {
      out.content_name = items[0].item_name;
    }
  }
  return out;
}

function send(name: string, params: GAParams): void {
  const pixelEvent = PIXEL_EVENTS[name];
  if (pixelEvent) {
    pixelTrack(pixelEvent, pixelParams(params));
  }
  if (!GA_ID) {
    return;
  }
  window.dataLayer = window.dataLayer ?? [];
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
    item_variant: [item.variant, item.condition, item.language]
      .filter(Boolean)
      .join(" · "),
    price: reais(item.price),
    quantity,
  };
}

export function sealedItem(item: Sealed | Accessory, quantity = 1): GAItem {
  return {
    item_id: item.slug,
    item_name: item.name,
    item_category: item.game,
    item_category2: item.kind,
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

export function catalogItem(item: Single | Sealed | Accessory): GAItem {
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

export function trackApplyCoupon(coupon: string, discountCents: number): void {
  send("apply_coupon", {
    currency: CURRENCY,
    coupon,
    discount: reais(discountCents),
  });
}

export function trackGenerateLead(
  totalCents: number,
  source: "whatsapp_checkout" | "mercadopago" = "whatsapp_checkout",
): void {
  send("generate_lead", {
    currency: CURRENCY,
    value: reais(totalCents),
    lead_source: source,
  });
}

export interface PurchaseLine {
  name: string;
  kind: ProductKind;
  meta?: string;
  qty: number;
  unitBRL: number;
}

export function trackPurchase(
  orderId: string,
  lines: PurchaseLine[],
  totalBRL: number,
  shippingBRL: number,
): void {
  send("purchase", {
    transaction_id: orderId,
    currency: CURRENCY,
    value: totalBRL,
    shipping: shippingBRL,
    items: lines.map((l) => ({
      item_id: l.name,
      item_name: l.name,
      item_category2: l.kind,
      item_variant: l.meta,
      price: l.unitBRL,
      quantity: l.qty,
    })),
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

export function trackLogin(): void {
  send("login", { method: "collecta" });
}

export function trackSignUp(withReferral: boolean): void {
  send("sign_up", { method: "collecta", referral: withReferral });
}

export function trackLogout(): void {
  send("logout", {});
}

export function trackLoginPrompt(origin: string): void {
  send("login_prompt", { origin });
}

export function trackViewAccount(unlocked: number, locked: number, used: number, referrals: number): void {
  send("view_account", {
    coupons_unlocked: unlocked,
    coupons_locked: locked,
    coupons_used: used,
    referrals,
  });
}

export function trackViewOrders(count: number): void {
  send("view_orders", { orders: count });
}

export function trackViewOrder(status: string, totalBRL: number, withCoupon: boolean): void {
  send("view_order", { order_status: status, value: totalBRL, currency: CURRENCY, with_coupon: withCoupon });
}

export function trackShareReferral(method: "whatsapp" | "copy_code" | "copy_link"): void {
  send("share", { method, content_type: "referral" });
}

export function trackRemoveCoupon(coupon: string): void {
  send("remove_coupon", { coupon });
}

export function trackCouponError(reason: string): void {
  send("coupon_error", { reason });
}

export function trackGetDirections(origin: string): void {
  send("get_directions", { origin });
}

export function trackSocialClick(network: string, origin: string): void {
  send("social_click", { network, origin });
}

export function trackViewStore(): void {
  send("view_store", {});
}

export function trackViewOrderArt(name: string): void {
  send("view_order_art", { item_name: name });
}
