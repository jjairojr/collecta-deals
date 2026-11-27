// Turns the carousel slides the owner set on the Vitrine screen into what the
// home shows: for a product slide the photo, name and prices always come from
// today's catalog (never typed), and a slide whose product is out of stock drops
// out. With nothing left, the home keeps its usual hero.
import type { Catalog } from "@/lib/catalog";
import { findBySku } from "@/lib/catalog";
import { kitStock, kitToCartLine, sealedToCartLine, singleToCartLine } from "@/lib/cart-lines";
import { INTEREST_FREE, MAX_INSTALLMENTS } from "@/lib/payment";
import type { CartLine } from "@/lib/types";
import { findKit, productHref, type VitrineHeroSlide, type VitrineImage } from "@/lib/vitrine";

export interface HeroProductSlide {
  kind: "produto";
  key: string;
  name: string;
  game: string;
  gameLabel: string;
  imageURL?: string;
  /** Pix price in cents (the price the site leads with). */
  price: number;
  /** Card price in cents (the installments are on this one). */
  cardPrice: number;
  /** "Kit com 3" and the Pix price per unit, for a kit. */
  kitUnits?: number;
  perUnit?: number;
  /** Interest-free installment value in cents, only when the store pays the interest. */
  installment?: { count: number; each: number };
  installments: number;
  preorderFrom?: string;
  badge?: string;
  phrase?: string;
  href: string;
  line: Omit<CartLine, "qty">;
}

export interface HeroImageSlide {
  kind: "imagem";
  key: string;
  image: VitrineImage;
  badge?: string;
  phrase?: string;
  href: string;
}

export type HeroSlideView = HeroProductSlide | HeroImageSlide;

function installmentFor(cardCents: number): { count: number; each: number } | undefined {
  if (!INTEREST_FREE) {
    return undefined;
  }
  return { count: MAX_INSTALLMENTS, each: Math.round(cardCents / MAX_INSTALLMENTS) };
}

function productSlide(catalog: Catalog, sl: VitrineHeroSlide, index: number): HeroProductSlide | null {
  const sku = sl.sku ?? "";
  const base = { kind: "produto" as const, key: `${index}-${sku}`, badge: sl.badge, phrase: sl.phrase };
  const item = findBySku(catalog, sku);
  if (item) {
    if (item.qty <= 0) {
      return null;
    }
    const line = item.kind === "single" ? singleToCartLine(item) : sealedToCartLine(item, item.qty);
    const cardPrice = item.kind === "single" ? item.price : item.cardPrice ?? item.price;
    return {
      ...base,
      name: item.kind === "single" ? `${item.name} ${item.number}` : item.name,
      game: item.game,
      gameLabel: item.gameLabel,
      imageURL: item.imageURL,
      price: item.price,
      cardPrice,
      installment: installmentFor(cardPrice),
      installments: MAX_INSTALLMENTS,
      preorderFrom: item.kind === "single" ? undefined : item.shipsFrom,
      href: sl.href || productHref(item),
      line,
    };
  }
  const found = findKit(catalog, sku);
  if (!found || kitStock(found.product.qty, found.kit.units) <= 0) {
    return null;
  }
  const { product, kit } = found;
  const cardPrice = kit.cardPrice ?? kit.price;
  return {
    ...base,
    name: product.name,
    game: product.game,
    gameLabel: product.gameLabel,
    imageURL: product.imageURL,
    price: kit.price,
    cardPrice,
    kitUnits: kit.units,
    perUnit: Math.round(kit.price / kit.units),
    installment: installmentFor(cardPrice),
    installments: MAX_INSTALLMENTS,
    preorderFrom: product.shipsFrom,
    href: sl.href || productHref(product),
    line: kitToCartLine(product, kit, product.qty),
  };
}

/** The slides to show today, in the owner's order; empty means "keep the usual hero". */
export function heroSlides(catalog: Catalog, slides: VitrineHeroSlide[] | undefined): HeroSlideView[] {
  return (slides ?? []).flatMap((sl, i): HeroSlideView[] => {
    if (sl.kind === "imagem" && sl.image && sl.href) {
      return [{ kind: "imagem", key: `${i}-img`, image: sl.image, badge: sl.badge, phrase: sl.phrase, href: sl.href }];
    }
    const view = sl.kind === "produto" ? productSlide(catalog, sl, i) : null;
    return view ? [view] : [];
  });
}
