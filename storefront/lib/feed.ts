import "server-only";
import { loadCatalog } from "@/lib/catalog";
import { absoluteURL } from "@/lib/site";
import type { Accessory, Sealed, Single } from "@/lib/types";

// One normalized catalog row, serialized by the Google Merchant (RSS) and Meta
// (CSV) routes. Both platforms want the same facts under different spellings.
export interface FeedItem {
  id: string;
  title: string;
  description: string;
  link: string;
  imageLink: string;
  price: string;
  inStock: boolean;
  quantity: number;
  condition: "new" | "used";
  brand: string;
  productType: string;
  googleCategory: string;
  labels: string[];
}

// Feed `brand` is the manufacturer, not the store. Falls back to the game label
// for any game the backend starts carrying before this map catches up.
const MAKERS: Record<string, string> = {
  pokemon: "The Pokémon Company",
  onepiece: "Bandai",
  riftbound: "Riot Games",
  lorcana: "Ravensburger",
  gundam: "Bandai",
};

// Google product taxonomy paths. Worth re-checking against the current taxonomy
// revision if Merchant Center ever flags a category.
const CAT_CARDS =
  "Arts & Entertainment > Hobbies & Creative Arts > Collectibles > Collectible Trading Cards";
const CAT_GAMES = "Toys & Games > Games > Card Games";

function money(cents: number): string {
  return `${(cents / 100).toFixed(2)} BRL`;
}

function clamp(s: string, max: number): string {
  const t = s.trim().replace(/\s+/g, " ");
  return t.length <= max ? t : `${t.slice(0, max - 1).trimEnd()}…`;
}

// Bidding segment for Google Ads / Meta — lets cheap commons and chase cards sit
// in different campaigns without restructuring the catalog.
function priceBand(cents: number): string {
  if (cents < 2500) {
    return "ate-25";
  }
  if (cents < 10000) {
    return "25-100";
  }
  if (cents < 30000) {
    return "100-300";
  }
  return "300-mais";
}

function maker(game: string, fallback: string): string {
  return MAKERS[game] ?? fallback;
}

// Sealed boxes and accessories carry a product id rather than a collector
// number, so the /img proxy resolves art for some and 404s for others. A broken
// image_link disapproves that item, so the few proxy-backed products get checked
// once per feed build. Singles skip the check — art keyed by set + collector
// number is exactly what the endpoint is for, and there are 366 of them.
async function usableImage(url: string | undefined): Promise<string | null> {
  if (!url) {
    return null;
  }
  const absolute = absoluteURL(url);
  if (!url.startsWith("/img")) {
    return absolute;
  }
  try {
    const res = await fetch(absolute, { method: "HEAD" });
    return res.ok ? absolute : null;
  } catch {
    return null;
  }
}

function singleRow(s: Single): FeedItem | null {
  if (!s.imageURL) {
    return null;
  }
  const number = s.number ? ` ${s.number}` : "";
  const numberNote = s.number ? ` (${s.number})` : "";
  const setNote = s.set ? `, set ${s.set}` : "";
  return {
    id: s.slug,
    title: clamp(
      `${s.name}${number} — Carta ${s.gameLabel}${s.set ? ` ${s.set}` : ""} ${s.condition}`,
      150,
    ),
    description: clamp(
      `Carta ${s.name}${numberNote} de ${s.gameLabel}${setNote}, condição ${s.condition}. Conferida uma a uma na Collecta, com pedido pelo WhatsApp e envio para todo o Brasil.`,
      5000,
    ),
    link: absoluteURL(`/carta/${s.slug}`),
    imageLink: absoluteURL(s.imageURL),
    price: money(s.price),
    inStock: s.qty > 0,
    quantity: s.qty,
    // Singles are pulled from packs, so they ship as `used` — which is also what
    // the product page's JSON-LD declares. Merchant Center cross-checks the feed
    // against the landing page and disapproves on a mismatch.
    condition: "used",
    brand: maker(s.game, s.gameLabel),
    productType: ["Singles", s.gameLabel, s.set].filter(Boolean).join(" > "),
    googleCategory: CAT_CARDS,
    labels: [s.gameLabel, s.set, s.condition, priceBand(s.price)].filter(Boolean),
  };
}

async function sealedRow(s: Sealed): Promise<FeedItem | null> {
  const image = await usableImage(s.imageURL);
  if (!image) {
    return null;
  }
  const lang = s.language ? ` em ${s.language}` : "";
  return {
    id: s.slug,
    title: clamp(`${s.name} — ${s.gameLabel} lacrado`, 150),
    description: clamp(
      `${s.name} — produto selado de ${s.gameLabel}${lang}, com lacre original. Pedido pelo WhatsApp e envio para todo o Brasil na Collecta.`,
      5000,
    ),
    link: absoluteURL(`/selado/${s.slug}`),
    imageLink: image,
    price: money(s.price),
    inStock: s.qty > 0,
    quantity: s.qty,
    condition: "new",
    brand: maker(s.game, s.gameLabel),
    productType: ["Selados", s.gameLabel, s.set].filter(Boolean).join(" > "),
    googleCategory: CAT_CARDS,
    labels: [s.gameLabel, s.set, "lacrado", priceBand(s.price)].filter(Boolean),
  };
}

async function accessoryRow(a: Accessory): Promise<FeedItem | null> {
  const image = await usableImage(a.imageURL);
  if (!image) {
    return null;
  }
  return {
    id: a.slug,
    title: clamp(`${a.name} — Acessório TCG`, 150),
    description: clamp(
      `${a.name} — acessório para card game, produto novo, serve para qualquer jogo. Pedido pelo WhatsApp e envio para todo o Brasil na Collecta.`,
      5000,
    ),
    link: absoluteURL(`/acessorio/${a.slug}`),
    imageLink: image,
    price: money(a.price),
    inStock: a.qty > 0,
    quantity: a.qty,
    condition: "new",
    brand: a.set || "Collecta",
    productType: ["Acessórios", a.set].filter(Boolean).join(" > "),
    googleCategory: CAT_GAMES,
    labels: ["Acessórios", a.set, "novo", priceBand(a.price)].filter(Boolean),
  };
}

// Returns null when the backend is unreachable and loadCatalog fell back to the
// mock catalog. The routes turn that into a 503 so Merchant Center and Meta keep
// the last good fetch instead of ingesting demo products or wiping the catalog.
export async function buildFeed(): Promise<FeedItem[] | null> {
  const { singles, sealed, accessories, live } = await loadCatalog();
  if (!live) {
    return null;
  }
  const boxed = await Promise.all([
    ...sealed.map(sealedRow),
    ...accessories.map(accessoryRow),
  ]);
  return [...singles.map(singleRow), ...boxed].flatMap((it) =>
    it ? [it] : [],
  );
}
