"use client";

// Measurement for the campaign pieces (banner da home, pop-up de saída): GA4
// view_promotion / select_promotion, and add_to_cart tagged with where it came
// from (item_list_name in GA4, content_category in the Meta Pixel). Kept apart
// from lib/analytics.ts so the existing calls stay as they are.
import { sendGAEvent } from "@next/third-parties/google";
import { GA_ID, type GAItem } from "@/lib/analytics";
import { pixelTrack } from "@/lib/meta-pixel";

export interface Promotion {
  /** Stable id, e.g. "banner_natal" or "popup_saida". */
  id: string;
  name: string;
  /** Where on the page: "home_topo", "popup". */
  slot: string;
}

type Params = Record<string, string | number | boolean | GAItem[] | undefined>;

function ga(name: string, params: Params): void {
  if (!GA_ID || typeof window === "undefined") {
    return;
  }
  window.dataLayer = window.dataLayer ?? [];
  sendGAEvent("event", name, process.env.NODE_ENV === "development" ? { ...params, debug_mode: true } : params);
}

function promoParams(p: Promotion, items: GAItem[]): Params {
  return {
    promotion_id: p.id,
    promotion_name: p.name,
    creative_name: p.name,
    creative_slot: p.slot,
    items: items.length > 0 ? items : undefined,
  };
}

/** The piece was seen (call once per showing). */
export function trackViewPromotion(p: Promotion, items: GAItem[] = []): void {
  ga("view_promotion", promoParams(p, items));
}

/** The buyer clicked the piece (a link or its add button). */
export function trackSelectPromotion(p: Promotion, items: GAItem[] = []): void {
  ga("select_promotion", promoParams(p, items));
}

/** add_to_cart that came from a campaign piece; listName says which one. */
export function trackAddToCartFrom(item: GAItem, listName: string): void {
  const tagged = { ...item, item_list_name: listName };
  const value = Math.round(item.price * 100 * item.quantity) / 100;
  pixelTrack("AddToCart", {
    value,
    currency: "BRL",
    content_type: "product",
    content_ids: [item.item_id],
    contents: [{ id: item.item_id, quantity: item.quantity }],
    num_items: item.quantity,
    content_name: item.item_name,
    content_category: listName,
  });
  ga("add_to_cart", { currency: "BRL", value, item_list_name: listName, items: [tagged as GAItem] });
}
