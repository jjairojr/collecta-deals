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
import { setsOf } from "@/lib/sets";
import LEGACY_SLUGS from "@/lib/legacy-slugs.json";
import LEGACY_SOLD from "@/lib/legacy-sold.json";
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
  sku?: string;
  slug?: string;
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
  addedAt?: string;
  noArt?: boolean;
}

const legacySold: RawItem[] = LEGACY_SOLD;

interface RawResponse {
  fxRate: number;
  count: number;
  artVersion?: string;
  items: RawItem[];
  soldOut?: RawItem[];
}

export interface SoldOut {
  singles: Single[];
  sealed: Sealed[];
  accessories: Accessory[];
}

export interface Catalog {
  singles: Single[];
  sealed: Sealed[];
  accessories: Accessory[];
  soldOut: SoldOut;
  redirects: Map<string, string>;
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
  if (it.slug) {
    return it.slug;
  }
  return slugify(
    [it.game, it.name, it.number, it.variant, it.condition]
      .filter(Boolean)
      .join("-"),
  );
}

// The stock key the backend sells by. A backend from before checkout sends
// none; grouping by the URL slug then keeps the catalog whole (checkout itself
// is off until that backend ships).
function skuFor(it: RawItem): string {
  return it.sku ?? `legacy|${slugFor(it)}`;
}

// Proxied art is cached for a day at every layer (Next data cache, Vercel CDN,
// browser), and the Next data cache outlives deploys. Stamping the URL with the
// backend's art version (its image cache's write time) and the deployment makes
// a corrected print show up as soon as either side changes, instead of whenever
// the caches happen to expire.
const DEPLOY_VERSION = (
  process.env.VERCEL_DEPLOYMENT_ID ??
  process.env.VERCEL_GIT_COMMIT_SHA ??
  "dev"
).slice(0, 12);

function artVersion(backend: string | undefined): string {
  return backend ? `${backend}-${DEPLOY_VERSION}` : DEPLOY_VERSION;
}

// Art URL: a seller-supplied photo when there is one, else a same-origin proxy
// to the Go card-image endpoint (keeps the backend host off the client). The
// productID goes along so the backend serves the exact print's TCGplayer art,
// falling back to the Liga page image for prints the TCG CDN has no photo for.
function resolveImage(it: RawItem, version: string): string | undefined {
  if (it.imageURL) {
    return it.imageURL;
  }
  if (!it.number || it.noArt) {
    return undefined;
  }
  const p = new URLSearchParams({ game: it.game, set: it.set, number: it.number });
  if (it.productID) {
    p.set("productID", String(it.productID));
  }
  p.set("v", version);
  return `/img?${p.toString()}`;
}

function mapItem(it: RawItem, version: string): Single {
  const condition = it.condition?.trim() || "NM";
  return {
    kind: "single",
    slug: slugFor(it),
    sku: skuFor(it),
    game: it.game,
    gameLabel: it.gameLabel,
    name: it.name,
    set: it.set,
    number: it.number,
    variant: it.variant?.trim() || undefined,
    condition,
    language: "", // backend has no language field yet
    price: Math.round(it.askBRL * 100),
    qty: Math.max(it.qty, 0),
    grade: /^PSA/i.test(condition) ? condition : undefined,
    imageURL: resolveImage(it, version),
    addedAt: it.addedAt,
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

function mapSealed(it: RawItem, version: string): Sealed {
  const set = cleanSet(it.set);
  const language = sealedLanguage(it.name);
  const format = sealedFormat(it.name);
  // The slug still hashes the raw name: it is the public URL and the Merchant
  // Center product id, so cleaning the label must not move it.
  const name = stripLangTag(it.name);
  return {
    kind: "sealed",
    slug: slugFor(it),
    sku: skuFor(it),
    game: it.game,
    gameLabel: it.gameLabel,
    name,
    format,
    headline: sealedHeadline(name, format),
    set,
    meta: [set || it.gameLabel, language].filter(Boolean).join(" · "),
    language,
    price: Math.round(it.askBRL * 100),
    qty: Math.max(it.qty, 0),
    badge: "LACRADO",
    imageURL: resolveImage(it, version),
    featured: it.featured,
    addedAt: it.addedAt,
  };
}

// Accessories belong to no game — the backend publishes them under a neutral
// "acessorios" id — so their meta line carries the product's own set/brand when
// there is one instead of a game name.
function mapAccessory(it: RawItem, version: string): Accessory {
  const set = cleanSet(it.set);
  return {
    kind: "accessory",
    slug: slugFor(it),
    sku: skuFor(it),
    game: it.game,
    gameLabel: it.gameLabel,
    name: it.name,
    set,
    meta: set || "Acessório",
    price: Math.round(it.askBRL * 100),
    qty: Math.max(it.qty, 0),
    badge: "ACESSORIO",
    imageURL: resolveImage(it, version),
    featured: it.featured,
    addedAt: it.addedAt,
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

// Collapse the ledger rows behind one stock key (sku), summing quantity. The
// backend sells the cheapest row first, so the group shows the lowest price —
// the same number checkout will charge.
interface Product {
  slug: string;
  sku: string;
  qty: number;
  price: number;
  featured?: boolean;
  addedAt?: string;
}

function dedupProducts<T extends Product>(items: T[], aliases: Map<string, string>): T[] {
  const bySKU = new Map<string, { item: T; firstAdded?: string }>();
  for (const s of items) {
    const group = bySKU.get(s.sku);
    if (!group) {
      bySKU.set(s.sku, { item: { ...s }, firstAdded: s.addedAt });
      continue;
    }
    const cur = group.item;
    cur.qty += s.qty;
    cur.price = Math.min(cur.price, s.price);
    cur.featured = cur.featured || s.featured;
    if (s.slug !== cur.slug) {
      const older = s.addedAt && group.firstAdded && s.addedAt < group.firstAdded;
      aliases.set(older ? cur.slug : s.slug, older ? s.slug : cur.slug);
      if (older) {
        cur.slug = s.slug;
      }
    }
    if (s.addedAt && (!group.firstAdded || s.addedAt < group.firstAdded)) {
      group.firstAdded = s.addedAt;
    }
    if (s.addedAt && (!cur.addedAt || s.addedAt > cur.addedAt)) {
      cur.addedAt = s.addedAt;
    }
  }
  return [...bySKU.values()].map((g) => g.item);
}

function resolveAliases(aliases: Map<string, string>): Map<string, string> {
  const out = new Map<string, string>();
  for (const from of aliases.keys()) {
    let to = aliases.get(from) ?? from;
    for (let hops = 0; aliases.has(to) && hops < 8; hops++) {
      to = aliases.get(to) ?? to;
    }
    out.set(from, to);
  }
  return out;
}

// Mock catalog for design work with no backend configured. It never stands in
// for a configured backend: a store page showing demo cards with demo prices
// is worse than an empty shelf, and a failed revalidation throws so Next keeps
// serving the last good page instead.
function mockCatalog(): Catalog {
  const singles = CATALOG.map((s) => ({ ...s }));
  const sealed = SEALED.map((s) => ({ ...s }));
  return {
    singles,
    sealed,
    accessories: [],
    soldOut: { singles: [], sealed: [], accessories: [] },
    redirects: new Map(),
    games: deriveGames(singles),
    navGames: deriveGames([...singles, ...sealed]),
    live: false,
  };
}

const RETRY_STATUSES = new Set([502, 503, 504]);
const RETRY_DELAYS_MS = [2000, 5000, 10000, 20000];

async function fetchStorefront(base: string): Promise<RawResponse> {
  for (let attempt = 0; ; attempt++) {
    const res = await fetch(`${base}/api/storefront`, {
      next: { revalidate: 60 },
    });
    if (res.ok) {
      return (await res.json()) as RawResponse;
    }
    const delay = RETRY_DELAYS_MS[attempt];
    if (!RETRY_STATUSES.has(res.status) || delay === undefined) {
      throw new Error(`storefront api ${res.status}`);
    }
    await new Promise((r) => setTimeout(r, delay));
  }
}

export async function loadCatalog(): Promise<Catalog> {
  const base = process.env.STOREFRONT_API;
  if (!base) {
    return mockCatalog();
  }
  const data = await fetchStorefront(base);
  const version = artVersion(data.artVersion);
  const aliases = new Map<string, string>();
  const shelf = (raw: RawItem[]) => ({
    singles: dedupProducts(
      raw.filter((it) => !it.kind).map((it) => mapItem(it, version)),
      aliases,
    ),
    sealed: dedupProducts(
      raw.filter((it) => it.kind === "sealed").map((it) => mapSealed(it, version)),
      aliases,
    ),
    accessories: dedupProducts(
      raw.filter((it) => it.kind === "accessory").map((it) => mapAccessory(it, version)),
      aliases,
    ),
  });
  const { singles, sealed, accessories } = shelf(data.items ?? []);
  const gone = shelf([...(data.soldOut ?? []), ...legacySold]);
  const taken = new Set([...singles, ...sealed, ...accessories].map((p) => p.slug));
  const soldOut: SoldOut = {
    singles: gone.singles.filter((p) => !taken.has(p.slug)).map((p) => ({ ...p, soldOut: true })),
    sealed: gone.sealed.filter((p) => !taken.has(p.slug)).map((p) => ({ ...p, soldOut: true })),
    accessories: gone.accessories.filter((p) => !taken.has(p.slug)).map((p) => ({ ...p, soldOut: true })),
  };
  return {
    singles,
    sealed,
    accessories,
    soldOut,
    redirects: productRedirects(resolveAliases(aliases), [
      ...[...singles, ...soldOut.singles].map((p) => ({ slug: p.slug, path: `/carta/${p.slug}` })),
      ...[...sealed, ...soldOut.sealed].map((p) => ({ slug: p.slug, path: `/selado/${p.slug}` })),
      ...[...accessories, ...soldOut.accessories].map((p) => ({ slug: p.slug, path: `/acessorio/${p.slug}` })),
    ]),
    games: deriveGames(singles),
    navGames: deriveGames([...singles, ...sealed]),
    live: true,
  };
}

// Product detail. Curated mock overrides only apply when the catalog itself is
// the mock fallback — a live catalog must never serve demo products. `live`
// travels with the item so pages can noindex mock-derived content.
export function findBySku(
  catalog: Catalog,
  sku: string,
): Single | Sealed | Accessory | null {
  return (
    catalog.singles.find((c) => c.sku === sku) ??
    catalog.sealed.find((c) => c.sku === sku) ??
    catalog.accessories.find((c) => c.sku === sku) ??
    null
  );
}

function productRedirects(
  aliases: Map<string, string>,
  products: { slug: string; path: string }[],
): Map<string, string> {
  const pathOf = new Map(products.map((p) => [p.slug, p.path]));
  const out = new Map<string, string>();
  const add = (from: string, to: string) => {
    const path = pathOf.get(to);
    if (path && !pathOf.has(from)) {
      out.set(from, path);
    }
  };
  for (const [from, to] of aliases) {
    add(from, to);
  }
  const legacy: Record<string, string> = LEGACY_SLUGS;
  for (const [from, to] of Object.entries(legacy)) {
    add(from, to);
  }
  return out;
}

export type DetailResult<T> = { item: T; live: boolean } | { redirect: string };

function findProduct<T extends { slug: string }>(live: T[], gone: T[], slug: string): T | undefined {
  return live.find((c) => c.slug === slug) ?? gone.find((c) => c.slug === slug);
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
  const s = findProduct(catalog.singles, catalog.soldOut.singles, slug);
  if (s) {
    return { item: buildSingleDetail(s), live: catalog.live };
  }
  const redirect = catalog.redirects.get(slug);
  return redirect ? { redirect } : null;
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
  const s = findProduct(catalog.sealed, catalog.soldOut.sealed, slug);
  if (s) {
    return { item: buildSealedDetail(s), live: catalog.live };
  }
  const redirect = catalog.redirects.get(slug);
  return redirect ? { redirect } : null;
}

export async function loadAccessoryDetail(
  slug: string,
): Promise<DetailResult<AccessoryDetail> | null> {
  const catalog = await loadCatalog();
  const a = findProduct(catalog.accessories, catalog.soldOut.accessories, slug);
  if (a) {
    return { item: buildAccessoryDetail(a), live: catalog.live };
  }
  const redirect = catalog.redirects.get(slug);
  return redirect ? { redirect } : null;
}

export function scopeToGame(catalog: Catalog, game: string) {
  const singles = catalog.singles.filter((s) => s.game === game);
  const opts = catalog.games
    .filter((g) => g.id === game)
    .map((g) => ({ id: g.id, label: g.name }));
  const sets = setsOf(singles, game);
  return {
    singles,
    games: opts,
    navGames: opts,
    sets,
    setLabels: Object.fromEntries(sets.map((s) => [s.code, s.label])),
  };
}
