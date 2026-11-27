// The editable campaign content (faixa, banner da home e pop-up de saída) that the
// owner fills on the dashboard's "Vitrine" screen. The backend already filters it
// to what is live today in Goiânia, so nothing here knows about dates.
import type { Catalog } from "@/lib/catalog";
import { findBySku } from "@/lib/catalog";
import { kitToCartLine, sealedToCartLine, singleToCartLine } from "@/lib/cart-lines";
import type { Accessory, CartLine, KitOffer, Sealed } from "@/lib/types";

export interface VitrineTier {
  label: string;
  href: string;
}

export interface VitrineBanner {
  title: string;
  text?: string;
  cta?: string;
  href?: string;
  tiers?: VitrineTier[];
}

export interface VitrinePopup {
  sku: string;
  title: string;
  text?: string;
}

export interface VitrineLive {
  phrases: string[];
  banner?: VitrineBanner;
  popup?: VitrinePopup;
}

const EMPTY: VitrineLive = { phrases: [] };

/** Live campaign content; empty (nothing new on the site) when the backend has none or is down. */
export async function loadVitrine(): Promise<VitrineLive> {
  const base = process.env.STOREFRONT_API;
  if (!base) {
    return EMPTY;
  }
  try {
    const res = await fetch(`${base}/api/storefront/vitrine`, { next: { revalidate: 60 } });
    if (!res.ok) {
      return EMPTY;
    }
    const data = (await res.json()) as Partial<VitrineLive>;
    return {
      phrases: Array.isArray(data.phrases) ? data.phrases.filter((p) => typeof p === "string" && p.trim()) : [],
      banner: data.banner?.title ? data.banner : undefined,
      popup: data.popup?.sku ? data.popup : undefined,
    };
  } catch {
    return EMPTY;
  }
}

/** What the exit pop-up needs to show its product and put it in the cart. */
export interface PopupProduct {
  title: string;
  text?: string;
  href: string;
  line: Omit<CartLine, "qty">;
}

function productHref(item: { kind: string; slug: string }): string {
  if (item.kind === "single") {
    return `/carta/${item.slug}`;
  }
  return item.kind === "accessory" ? `/acessorio/${item.slug}` : `/selado/${item.slug}`;
}

function findKit(catalog: Catalog, sku: string): { product: Sealed | Accessory; kit: KitOffer } | null {
  for (const product of [...catalog.sealed, ...catalog.accessories]) {
    const kit = product.kits?.find((k) => k.sku === sku);
    if (kit) {
      return { product, kit };
    }
  }
  return null;
}

/**
 * Resolves the pop-up's SKU against today's catalog: a single, a sealed product,
 * an accessory or a kit. Out of stock (or gone) means no pop-up.
 */
export function popupProduct(catalog: Catalog, popup: VitrinePopup | undefined): PopupProduct | null {
  if (!popup) {
    return null;
  }
  const base = { title: popup.title, text: popup.text };
  const item = findBySku(catalog, popup.sku);
  if (item) {
    if (item.qty <= 0) {
      return null;
    }
    const line = item.kind === "single" ? singleToCartLine(item) : sealedToCartLine(item, item.qty);
    return { ...base, href: productHref(item), line };
  }
  const found = findKit(catalog, popup.sku);
  if (!found) {
    return null;
  }
  const line = kitToCartLine(found.product, found.kit, found.product.qty);
  if (!line.stock || line.stock <= 0) {
    return null;
  }
  return { ...base, href: productHref(found.product), line };
}
