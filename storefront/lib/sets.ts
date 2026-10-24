import type { Single } from "@/lib/types";

const SET_NAMES: Record<string, Record<string, string>> = {
  onepiece: {
    OP01: "Romance Dawn",
    OP02: "Paramount War",
    OP03: "Pillars of Strength",
    OP04: "Kingdoms of Intrigue",
    OP05: "Awakening of the New Era",
    OP06: "Wings of the Captain",
    OP07: "500 Years in the Future",
    OP08: "Two Legends",
    OP09: "Emperors in the New World",
    OP10: "Royal Blood",
    OP11: "A Fist of Divine Speed",
    OP12: "Legacy of the Master",
    P: "Promo",
  },
  pokemon: {
    SV1: "Scarlet & Violet",
    PAL: "Paldea Evolved",
    OBF: "Obsidian Flames",
    MEW: "151",
    PAR: "Paradox Rift",
    PAF: "Paldean Fates",
    TEF: "Temporal Forces",
    TWM: "Twilight Masquerade",
    SFA: "Shrouded Fable",
    SSP: "Surging Sparks",
    JTG: "Journey Together",
    DRI: "Destined Rivals",
    BLK: "Black Bolt",
    WHT: "White Flare",
    MEG: "Mega Evolution",
    PFL: "Phantasmal Flames",
    PBL: "Escuridão Absoluta",
    BRS: "Brilliant Stars",
    ASR: "Astral Radiance",
    PGO: "Pokémon GO",
    SIT: "Silver Tempest",
    CRZ: "Crown Zenith",
    FST: "Fusion Strike",
    HIF: "Hidden Fates",
  },
  riftbound: {
    OGN: "Origins",
    SFD: "Spiritforged",
  },
};

export function setSlug(code: string): string {
  return code
    .normalize("NFD")
    .replace(/\p{Diacritic}/gu, "")
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/(^-|-$)/g, "");
}

export function setName(game: string, code: string): string | undefined {
  return SET_NAMES[game]?.[code.replace(/-/g, "").toUpperCase()];
}

export function baseCardName(name: string): string {
  return name.replace(/\s*\([^)]*\)/g, "").trim();
}

export function setLabel(game: string, code: string): string {
  const name = setName(game, code);
  return name ? `${code} ${name}` : code;
}

export interface SetSummary {
  code: string;
  slug: string;
  label: string;
  count: number;
  minPrice: number;
}

export function setsOf(singles: Single[], game: string): SetSummary[] {
  const bySet = new Map<string, Single[]>();
  for (const s of singles) {
    if (s.game !== game || !s.set) {
      continue;
    }
    const list = bySet.get(s.set) ?? [];
    list.push(s);
    bySet.set(s.set, list);
  }
  return [...bySet.entries()]
    .map(([code, list]) => ({
      code,
      slug: setSlug(code),
      label: setLabel(game, code),
      count: list.length,
      minPrice: Math.min(...list.map((s) => s.price)),
    }))
    .sort((a, b) => b.count - a.count || a.code.localeCompare(b.code));
}

export function findSet(singles: Single[], game: string, slug: string): SetSummary | null {
  return setsOf(singles, game).find((s) => s.slug === slug) ?? null;
}

export function setPath(game: string, code: string): string {
  return `/singles/${game}/colecao/${setSlug(code)}`;
}
