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

/** Art uploaded on the Vitrine screen; URLs here are already absolute. */
export interface VitrineImage {
  desktop: string;
  mobile: string;
  alt: string;
}

/** Image that replaces the text bar at the top while it is live. */
export interface VitrineBarImage extends VitrineImage {
  href: string;
}

export interface VitrineBanner {
  title: string;
  text?: string;
  cta?: string;
  href?: string;
  tiers?: VitrineTier[];
  image?: VitrineImage;
}

export interface VitrinePopup {
  sku: string;
  title: string;
  text?: string;
}

/** One slide of the home carousel, as the backend sends it (already live today). */
export interface VitrineHeroSlide {
  kind: "produto" | "imagem";
  sku?: string;
  image?: VitrineImage;
  badge?: string;
  phrase?: string;
  href?: string;
}

export type HeroStyle = "vitrine" | "destaque";

export interface VitrineLive {
  phrases: string[];
  barImage?: VitrineBarImage;
  banner?: VitrineBanner;
  popup?: VitrinePopup;
  heroStyle?: HeroStyle;
  heroSlides?: VitrineHeroSlide[];
}

const EMPTY: VitrineLive = { phrases: [] };

// The backend stores the art and answers with paths under its own
// /api/storefront/vitrine/img/; the browser loads them from the backend's domain.
// Only those paths are accepted: a piece with any other image is dropped.
const IMAGE_PATH = /^\/api\/storefront\/vitrine\/img\/[0-9a-f]{16}\.(webp|jpg|png)$/;

function withImage<T extends VitrineImage>(base: string, im: T): T | undefined {
  if (!IMAGE_PATH.test(im.desktop) || !IMAGE_PATH.test(im.mobile) || !im.alt?.trim()) {
    return undefined;
  }
  const root = base.replace(/\/$/, "");
  return { ...im, desktop: `${root}${im.desktop}`, mobile: `${root}${im.mobile}` };
}

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
    const banner = data.banner?.title ? data.banner : undefined;
    const heroSlides = (Array.isArray(data.heroSlides) ? data.heroSlides : []).flatMap((sl): VitrineHeroSlide[] => {
      if (sl.kind === "produto" && sl.sku) {
        return [sl];
      }
      if (sl.kind === "imagem" && sl.image && sl.href) {
        const image = withImage(base, sl.image);
        return image ? [{ ...sl, image }] : [];
      }
      return [];
    });
    return {
      phrases: Array.isArray(data.phrases) ? data.phrases.filter((p) => typeof p === "string" && p.trim()) : [],
      barImage: data.barImage?.href ? withImage(base, data.barImage) : undefined,
      banner: banner ? { ...banner, image: banner.image && banner.href ? withImage(base, banner.image) : undefined } : undefined,
      popup: data.popup?.sku ? data.popup : undefined,
      heroStyle: data.heroStyle === "destaque" ? "destaque" : "vitrine",
      heroSlides,
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

export function productHref(item: { kind: string; slug: string }): string {
  if (item.kind === "single") {
    return `/carta/${item.slug}`;
  }
  return item.kind === "accessory" ? `/acessorio/${item.slug}` : `/selado/${item.slug}`;
}

export function findKit(catalog: Catalog, sku: string): { product: Sealed | Accessory; kit: KitOffer } | null {
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
