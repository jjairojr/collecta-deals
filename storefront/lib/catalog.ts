import "server-only";
import {
  CATALOG,
  SEALED,
  buildSingleDetail,
  buildSealedDetail,
  curatedSingle,
  sealedBySlug,
} from "@/lib/mock";
import { gameSurface } from "@/lib/games";
import type { Game, Sealed, SealedDetail, Single, SingleDetail } from "@/lib/types";

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
}

interface RawResponse {
  fxRate: number;
  count: number;
  items: RawItem[];
}

export interface Catalog {
  singles: Single[];
  sealed: Sealed[];
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

// Absolute art URL: the backend's exact TCGplayer image when known, else a
// same-origin proxy to the Go card-image endpoint (keeps the backend host off
// the client).
function resolveImage(it: RawItem): string | undefined {
  if (it.imageURL) {
    return it.imageURL;
  }
  if (!it.number) {
    return undefined;
  }
  const p = new URLSearchParams({ game: it.game, set: it.set, number: it.number });
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
    condition,
    language: "", // backend has no language field yet
    price: Math.round(it.askBRL * 100),
    qty: it.qty > 0 ? it.qty : 1,
    grade: /^PSA/i.test(condition) ? condition : undefined,
    imageURL: resolveImage(it),
  };
}

function mapSealed(it: RawItem): Sealed {
  return {
    kind: "sealed",
    slug: slugFor(it),
    game: it.game,
    gameLabel: it.gameLabel,
    name: it.name,
    set: it.set,
    meta: it.set || it.gameLabel,
    price: Math.round(it.askBRL * 100),
    badge: "LACRADO",
    imageURL: resolveImage(it),
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

// Sealed has no quantity in the public view — collapse duplicate slugs, first wins.
function dedupSealed(items: Sealed[]): Sealed[] {
  const bySlug = new Map<string, Sealed>();
  for (const s of items) {
    if (!bySlug.has(s.slug)) {
      bySlug.set(s.slug, s);
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
    const singles = dedup(raw.filter((it) => it.kind !== "sealed").map(mapItem));
    const sealed = dedupSealed(raw.filter((it) => it.kind === "sealed").map(mapSealed));
    if (singles.length === 0 && sealed.length === 0) {
      return mockCatalog();
    }
    return {
      singles,
      sealed,
      games: deriveGames(singles),
      navGames: deriveGames([...singles, ...sealed]),
      live: true,
    };
  } catch {
    return mockCatalog();
  }
}

// Product detail: curated override (mock demo) → real single built up with mock
// sellers/history → null (404).
export async function loadSingleDetail(
  slug: string,
): Promise<SingleDetail | null> {
  const curated = curatedSingle(slug);
  if (curated) {
    return curated;
  }
  const { singles } = await loadCatalog();
  const s = singles.find((c) => c.slug === slug);
  return s ? buildSingleDetail(s) : null;
}

// Sealed detail: curated mock override → real sealed built up with mock
// specs/stock → null (404).
export async function loadSealedDetail(
  slug: string,
): Promise<SealedDetail | null> {
  const curated = sealedBySlug(slug);
  if (curated) {
    return curated;
  }
  const { sealed } = await loadCatalog();
  const s = sealed.find((c) => c.slug === slug);
  return s ? buildSealedDetail(s) : null;
}
