import "server-only";
import {
  CATALOG,
  SEALED,
  buildSingleDetail,
  buildSealedDetail,
  buildAccessoryDetail,
  curatedSingle,
  sealedBySlug,
} from "@/lib/mock";
import { gameSurface } from "@/lib/games";
import type {
  Accessory,
  AccessoryDetail,
  Game,
  Sealed,
  SealedDetail,
  Single,
  SingleDetail,
} from "@/lib/types";

// Shape of one item from the Go backend's GET /api/storefront (see
// internal/api/storefront.go). Prices are in reais; there is no slug/language.
interface RawItem {
  game: string;
  gameLabel: string;
  kind?: string;
  number: string;
  name: string;
  set: string;
  variant?: string;
  condition?: string;
  qty: number;
  productID?: number;
  imageURL?: string;
  askBRL: number;
  askUSD?: number;
  featured?: boolean;
}

interface RawResponse {
  fxRate: number;
  count: number;
  items: RawItem[];
}

export interface Catalog {
  singles: Single[];
  sealed: Sealed[];
  accessories: Accessory[];
  games: Game[]; // games that have singles — home cabinets + /singles JOGO filter
  navGames: Game[]; // games with any stock (singles or sealed) — header nav
  live: boolean; // true when served by the real backend, false when mock fallback
}

function slugify(s: string): string {
  return s
    .normalize("NFD")
    .replace(/\p{Diacritic}/gu, "")
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/(^-|-$)/g, "");
}

function slugFor(it: RawItem): string {
  return slugify(
    [it.game, it.name, it.number, it.variant, it.condition]
      .filter(Boolean)
      .join("-"),
  );
}

// Art URL: a seller-supplied photo when there is one, else a same-origin proxy
// to the Go card-image endpoint (keeps the backend host off the client). The
// productID goes along so the backend serves the exact print's TCGplayer art,
// falling back to the Liga page image for prints the TCG CDN has no photo for.
function resolveImage(it: RawItem): string | undefined {
  if (it.imageURL) {
    return it.imageURL;
  }
  if (!it.number) {
    return undefined;
  }
  const p = new URLSearchParams({ game: it.game, set: it.set, number: it.number });
  if (it.productID) {
    p.set("productID", String(it.productID));
  }
  return `/img?${p.toString()}`;
}

function mapItem(it: RawItem): Single {
  const condition = it.condition?.trim() || "NM";
  return {
    kind: "single",
    slug: slugFor(it),
    game: it.game,
    gameLabel: it.gameLabel,
    name: it.name,
    set: it.set,
    number: it.number,
    variant: it.variant?.trim() || undefined,
    condition,
    language: "", // backend has no language field yet
    price: Math.round(it.askBRL * 100),
    qty: it.qty > 0 ? it.qty : 1,
    grade: /^PSA/i.test(condition) ? condition : undefined,
    imageURL: resolveImage(it),
  };
}

// Language rides along inside the product name, at either end: "(PT-BR) Caixa
// de Booster …" and "… Booster Bundle (PT-BR)".
const SEALED_LANGS: [RegExp, string][] = [
  [/^\((ING|EN)\)|\((ING|EN)\)$/i, "Inglês"],
  [/^\((JAP|JPN)\)|\((JAP|JPN)\)$/i, "Japonês"],
  [/^\((PORT?|PT(-BR)?|BR)\)|\((PORT?|PT(-BR)?|BR)\)$/i, "Português"],
];

const LANG_ALT = "ING|EN|JAP|JPN|PORT|POR|PT-BR|PT|BR";
const LANG_TAG = new RegExp(
  `^\\s*\\((?:${LANG_ALT})\\)\\s*|\\s*\\((?:${LANG_ALT})\\)\\s*$`,
  "gi",
);

// Format label first, because a title that opens with "(PT-BR)" spends the most
// valuable characters of a search result on the Liga's internal notation — and
// the language already travels in its own field.
const SEALED_FORMATS: [RegExp, string][] = [
  [/elite trainer box|\betb\b/i, "Elite Trainer Box"],
  [/booster bundle/i, "Booster Bundle"],
  [/caixa de booster|booster box|box display/i, "Booster Box"],
  [/blister/i, "Blister"],
  [/\bdeck\b/i, "Deck"],
];

function sealedLanguage(name: string): string | undefined {
  const n = name.trim();
  for (const [re, lang] of SEALED_LANGS) {
    if (re.test(n)) {
      return lang;
    }
  }
  return undefined;
}

function stripLangTag(name: string): string {
  return name.replace(LANG_TAG, "").trim();
}

// "Caixa de Booster - Megaevolução 5 - Escuridão Absoluta" becomes
// "Booster Box Megaevolução 5 - Escuridão Absoluta": the searched term leads and
// the redundant Portuguese phrasing goes, without inventing anything.
function sealedHeadline(name: string, format: string | undefined): string {
  if (!format) {
    return name;
  }
  const matched = SEALED_FORMATS.find(([, label]) => label === format);
  if (!matched) {
    return name;
  }
  const rest = name
    .replace(matched[0], "")
    .replace(/^[\s\-–—·|]+|[\s\-–—·|]+$/g, "")
    .trim();
  return rest ? `${format} ${rest}` : format;
}

function sealedFormat(name: string): string | undefined {
  for (const [re, label] of SEALED_FORMATS) {
    if (re.test(name)) {
      return label;
    }
  }
  return undefined;
}

function cleanSet(set: string): string {
  const s = set.trim();
  return /^(sealed|acessorio)$/i.test(s) ? "" : s;
}

function mapSealed(it: RawItem): Sealed {
  const set = cleanSet(it.set);
  const language = sealedLanguage(it.name);
  const format = sealedFormat(it.name);
  // The slug still hashes the raw name: it is the public URL and the Merchant
  // Center product id, so cleaning the label must not move it.
  const name = stripLangTag(it.name);
  return {
    kind: "sealed",
    slug: slugFor(it),
    game: it.game,
    gameLabel: it.gameLabel,
    name,
    format,
    headline: sealedHeadline(name, format),
    set,
    meta: [set || it.gameLabel, language].filter(Boolean).join(" · "),
    language,
    price: Math.round(it.askBRL * 100),
    qty: it.qty > 0 ? it.qty : 1,
    badge: "LACRADO",
    imageURL: resolveImage(it),
    featured: it.featured,
  };
}

// Accessories belong to no game — the backend publishes them under a neutral
// "acessorios" id — so their meta line carries the product's own set/brand when
// there is one instead of a game name.
function mapAccessory(it: RawItem): Accessory {
  const set = cleanSet(it.set);
  return {
    kind: "accessory",
    slug: slugFor(it),
    game: it.game,
    gameLabel: it.gameLabel,
    name: it.name,
    set,
    meta: set || "Acessório",
    price: Math.round(it.askBRL * 100),
    qty: it.qty > 0 ? it.qty : 1,
    badge: "ACESSORIO",
    imageURL: resolveImage(it),
    featured: it.featured,
  };
}

// Games for the home cabinets and the /singles JOGO filter. Counts singles only,
// since a cabinet links to the singles browse — the "N CARTAS" badge must match
// what that screen shows.
function deriveGames(items: { game: string; gameLabel: string }[]): Game[] {
  const seen = new Map<string, { label: string; count: number }>();
  for (const s of items) {
    const cur = seen.get(s.game);
    if (cur) {
      cur.count += 1;
    } else {
      seen.set(s.game, { label: s.gameLabel, count: 1 });
    }
  }
  return [...seen.entries()]
    .map(([id, { label, count }]) => ({
      id,
      name: label,
      count,
      hue: gameSurface(id),
    }))
    .sort((a, b) => b.count - a.count);
}

// Dedup identical listings (same slug), summing quantity.
function dedup(items: Single[]): Single[] {
  const bySlug = new Map<string, Single>();
  for (const s of items) {
    const cur = bySlug.get(s.slug);
    if (cur) {
      cur.qty += s.qty;
    } else {
      bySlug.set(s.slug, { ...s });
    }
  }
  return [...bySlug.values()];
}

// Collapse duplicate slugs, summing quantity.
function dedupProducts<T extends { slug: string; qty: number; featured?: boolean }>(
  items: T[],
): T[] {
  const bySlug = new Map<string, T>();
  for (const s of items) {
    const cur = bySlug.get(s.slug);
    if (cur) {
      cur.qty += s.qty;
      cur.featured = cur.featured || s.featured;
    } else {
      bySlug.set(s.slug, { ...s });
    }
  }
  return [...bySlug.values()];
}

// Mock fallback so the design still renders when the backend is unconfigured or
// down. Uses the mock CATALOG + SEALED (which already carry slugs).
function mockCatalog(): Catalog {
  const singles = CATALOG.map((s) => ({ ...s }));
  const sealed = SEALED.map((s) => ({ ...s }));
  return {
    singles,
    sealed,
    accessories: [],
    games: deriveGames(singles),
    navGames: deriveGames([...singles, ...sealed]),
    live: false,
  };
}

export async function loadCatalog(): Promise<Catalog> {
  const base = process.env.STOREFRONT_API;
  if (!base) {
    return mockCatalog();
  }
  try {
    const res = await fetch(`${base}/api/storefront`, {
      next: { revalidate: 60 },
    });
    if (!res.ok) {
      return mockCatalog();
    }
    const data = (await res.json()) as RawResponse;
    const raw = data.items ?? [];
    const singles = dedup(raw.filter((it) => !it.kind).map(mapItem));
    const sealed = dedupProducts(raw.filter((it) => it.kind === "sealed").map(mapSealed));
    const accessories = dedupProducts(
      raw.filter((it) => it.kind === "accessory").map(mapAccessory),
    );
    if (singles.length === 0 && sealed.length === 0 && accessories.length === 0) {
      return mockCatalog();
    }
    return {
      singles,
      sealed,
      accessories,
      games: deriveGames(singles),
      navGames: deriveGames([...singles, ...sealed]),
      live: true,
    };
  } catch {
    return mockCatalog();
  }
}

// Product detail. Curated mock overrides only apply when the catalog itself is
// the mock fallback — a live catalog must never serve demo products. `live`
// travels with the item so pages can noindex mock-derived content.
export interface DetailResult<T> {
  item: T;
  live: boolean;
}

export async function loadSingleDetail(
  slug: string,
): Promise<DetailResult<SingleDetail> | null> {
  const catalog = await loadCatalog();
  if (!catalog.live) {
    const curated = curatedSingle(slug);
    if (curated) {
      return { item: curated, live: false };
    }
  }
  const s = catalog.singles.find((c) => c.slug === slug);
  return s ? { item: buildSingleDetail(s), live: catalog.live } : null;
}

export async function loadSealedDetail(
  slug: string,
): Promise<DetailResult<SealedDetail> | null> {
  const catalog = await loadCatalog();
  if (!catalog.live) {
    const curated = sealedBySlug(slug);
    if (curated) {
      return { item: curated, live: false };
    }
  }
  const s = catalog.sealed.find((c) => c.slug === slug);
  return s ? { item: buildSealedDetail(s), live: catalog.live } : null;
}

export async function loadAccessoryDetail(
  slug: string,
): Promise<DetailResult<AccessoryDetail> | null> {
  const catalog = await loadCatalog();
  const a = catalog.accessories.find((c) => c.slug === slug);
  return a ? { item: buildAccessoryDetail(a), live: catalog.live } : null;
}
