// Design-first mock catalog. Prices are in cents. This stands in for the Go
// backend until it supplies sealed/sellers/history data; the screens read only
// from here so swapping in a real data layer later is a localized change.

import type {
  HighScore,
  Sealed,
  SealedDetail,
  Seller,
  Single,
  SingleDetail,
} from "@/lib/types";

// Featured singles + MORE_SINGLES seed the mock catalog used as a fallback when
// the real backend is unconfigured/down (see lib/catalog). Game visual mappings
// live in lib/games; game lists are derived from the catalog, not hardcoded here.
export const FEATURED_SINGLES: Single[] = [
  {
    kind: "single",
    slug: "charizard-ex-obsidian-flames",
    game: "pokemon",
    gameLabel: "Pokémon",
    name: "Charizard ex",
    set: "Obsidian Flames",
    number: "199/165",
    condition: "PSA 10",
    language: "EN",
    price: 42000,
    wasPrice: 48900,
    qty: 1,
    grade: "PSA 10",
  },
  {
    kind: "single",
    slug: "pikachu-vmax-rainbow",
    game: "pokemon",
    gameLabel: "Pokémon",
    name: "Pikachu VMAX",
    set: "Vivid Voltage",
    number: "188/185",
    condition: "PSA 9",
    language: "EN",
    price: 31500,
    qty: 2,
    grade: "PSA 9",
  },
  {
    kind: "single",
    slug: "monkey-d-luffy-leader",
    game: "onepiece",
    gameLabel: "One Piece",
    name: "Monkey D. Luffy",
    set: "Romance Dawn",
    number: "OP01-003",
    condition: "NM",
    language: "JP",
    price: 18900,
    qty: 4,
  },
  {
    kind: "single",
    slug: "jinx-estopim-origins",
    game: "riftbound",
    gameLabel: "Riftbound",
    name: "Jinx, Estopim",
    set: "Origins",
    number: "ORI-101",
    condition: "NM",
    language: "EN",
    price: 13200,
    qty: 6,
  },
  {
    kind: "single",
    slug: "mewtwo-ex-alt-art",
    game: "pokemon",
    gameLabel: "Pokémon",
    name: "Mewtwo ex",
    set: "151",
    number: "150/165",
    condition: "NM",
    language: "EN",
    price: 27800,
    qty: 3,
  },
  {
    kind: "single",
    slug: "roronoa-zoro-alt",
    game: "onepiece",
    gameLabel: "One Piece",
    name: "Roronoa Zoro",
    set: "Paramount War",
    number: "OP02-046",
    condition: "NM",
    language: "JP",
    price: 22400,
    wasPrice: 26000,
    qty: 2,
  },
  {
    kind: "single",
    slug: "ahri-spirit-blossom",
    game: "riftbound",
    gameLabel: "Riftbound",
    name: "Ahri, Flor de Alma",
    set: "Origins",
    number: "ORI-058",
    condition: "PSA 10",
    language: "EN",
    price: 36900,
    qty: 1,
    grade: "PSA 10",
  },
  {
    kind: "single",
    slug: "gengar-vmax-fusion",
    game: "pokemon",
    gameLabel: "Pokémon",
    name: "Gengar VMAX",
    set: "Fusion Strike",
    number: "271/264",
    condition: "LP",
    language: "EN",
    price: 15900,
    qty: 5,
  },
];

export const SEALED: Sealed[] = [
  {
    kind: "sealed",
    slug: "booster-box-op-09",
    game: "onepiece",
    gameLabel: "One Piece",
    name: "Booster Box OP-09",
    set: "Emperors in the New World",
    meta: "24 packs · Japonês",
    price: 89900,
    qty: 7,
    badge: "PRE-VENDA",
  },
  {
    kind: "sealed",
    slug: "etb-obsidian-flames",
    game: "pokemon",
    gameLabel: "Pokémon",
    name: "Elite Trainer Box",
    set: "Obsidian Flames",
    meta: "9 packs · Inglês",
    price: 42900,
    qty: 6,
    badge: "HOT",
  },
  {
    kind: "sealed",
    slug: "booster-box-origins",
    game: "riftbound",
    gameLabel: "Riftbound",
    name: "Booster Box Origins",
    set: "Origins",
    meta: "24 packs · Inglês",
    price: 67900,
    qty: 3,
    badge: "NOVO",
  },
  {
    kind: "sealed",
    slug: "booster-box-151",
    game: "pokemon",
    gameLabel: "Pokémon",
    name: "Booster Box 151",
    set: "Scarlet & Violet 151",
    meta: "36 packs · Japonês",
    price: 129900,
    qty: 4,
    badge: "SO RESTAM 4",
  },
];

export const HIGH_SCORES: HighScore[] = [
  { position: "1ST", player: "BRUNO_TCG", trades: 312, score: 98420 },
  { position: "2ND", player: "MARI.PULLS", trades: 287, score: 91105 },
  { position: "3RD", player: "DOJO_CARDS", trades: 241, score: 84660 },
  { position: "4TH", player: "RIFT_KID", trades: 198, score: 72310 },
  { position: "5TH", player: "SEALED_SP", trades: 176, score: 68940 },
];

const OFFICIAL_SELLER = {
  id: "collecta-oficial",
  name: "Collecta Oficial",
  rating: 99.4,
  sales: 4821,
  ships: "ENVIA EM 24H",
};

// Product detail lookups. Only the slugs the product pages link to need a full
// detail record; the grids use the summaries above.
const SINGLE_DETAILS: Record<string, SingleDetail> = {
  "charizard-ex-obsidian-flames": {
    ...FEATURED_SINGLES[0],
    rarity: "Special Illustration Rare",
    marketAvg: 45500,
    seller: OFFICIAL_SELLER,
    priceHistory: [
      { label: "FEV", value: 38 },
      { label: "MAR", value: 52 },
      { label: "ABR", value: 46 },
      { label: "MAI", value: 68 },
      { label: "JUN", value: 84 },
      { label: "JUL", value: 92 },
    ],
    offers: [
      {
        seller: { id: "dojo-sp", name: "DojoCards SP", rating: 98.7, sales: 1203, ships: "48h" },
        condition: "PSA 10",
        price: 43900,
        ships: "envia em 48h",
      },
      {
        seller: { id: "bruno-tcg", name: "Bruno TCG", rating: 99.1, sales: 3410, ships: "24h" },
        condition: "PSA 9",
        price: 35200,
        ships: "envia em 24h",
      },
      {
        seller: { id: "mari-pulls", name: "Mari Pulls", rating: 98.2, sales: 980, ships: "72h" },
        condition: "NM sem grade",
        price: 29800,
        ships: "envia em 72h",
      },
    ],
  },
};

const SEALED_DETAILS: Record<string, SealedDetail> = {
  "booster-box-op-09": {
    ...SEALED[0],
    installmentsNote: "12x de R$ 83,20 · lançamento 12/09",
    stockLeft: 7,
    stockTotal: 40,
    specs: [
      { key: "CONTEUDO", value: "24 packs de 12 cartas" },
      { key: "IDIOMA", value: "Japonês" },
      { key: "LANCAMENTO", value: "12 set 2026" },
      { key: "GARANTIA", value: "Lacre original de fábrica" },
    ],
  },
};

// Extra singles so the Browse grid + pagination look populated. Together with
// FEATURED_SINGLES these form the full catalog the /singles screen filters over.
const MORE_SINGLES: Single[] = [
  {
    kind: "single",
    slug: "umbreon-vmax-alt-art",
    game: "pokemon",
    gameLabel: "Pokémon",
    name: "Umbreon VMAX",
    set: "Evolving Skies",
    number: "215/203",
    condition: "PSA 10",
    language: "EN",
    price: 189900,
    qty: 1,
    grade: "PSA 10",
  },
  {
    kind: "single",
    slug: "nami-parallel",
    game: "onepiece",
    gameLabel: "One Piece",
    name: "Nami",
    set: "Romance Dawn",
    number: "OP01-016",
    condition: "NM",
    language: "JP",
    price: 9800,
    qty: 8,
  },
  {
    kind: "single",
    slug: "charizard-vmax-shining",
    game: "pokemon",
    gameLabel: "Pokémon",
    name: "Charizard VMAX",
    set: "Shining Fates",
    number: "SV107/SV122",
    condition: "PSA 9",
    language: "EN",
    price: 98900,
    wasPrice: 112000,
    qty: 2,
    grade: "PSA 9",
  },
  {
    kind: "single",
    slug: "trafalgar-law-leader",
    game: "onepiece",
    gameLabel: "One Piece",
    name: "Trafalgar Law",
    set: "Paramount War",
    number: "OP02-069",
    condition: "NM",
    language: "PT",
    price: 14500,
    qty: 5,
  },
  {
    kind: "single",
    slug: "yasuo-origins",
    game: "riftbound",
    gameLabel: "Riftbound",
    name: "Yasuo, o Imperdoável",
    set: "Origins",
    number: "ORI-072",
    condition: "NM",
    language: "EN",
    price: 8900,
    qty: 7,
  },
  {
    kind: "single",
    slug: "rayquaza-vmax-alt",
    game: "pokemon",
    gameLabel: "Pokémon",
    name: "Rayquaza VMAX",
    set: "Evolving Skies",
    number: "218/203",
    condition: "NM",
    language: "EN",
    price: 46900,
    qty: 3,
  },
  {
    kind: "single",
    slug: "boa-hancock-alt",
    game: "onepiece",
    gameLabel: "One Piece",
    name: "Boa Hancock",
    set: "Kingdoms of Intrigue",
    number: "OP04-089",
    condition: "LP",
    language: "JP",
    price: 21900,
    qty: 2,
  },
  {
    kind: "single",
    slug: "viktor-origins",
    game: "riftbound",
    gameLabel: "Riftbound",
    name: "Viktor, o Arauto",
    set: "Origins",
    number: "ORI-140",
    condition: "PSA 9",
    language: "EN",
    price: 25900,
    qty: 1,
    grade: "PSA 9",
  },
];

// Full single catalog (featured first). Browse reads from here.
export const CATALOG: Single[] = [...FEATURED_SINGLES, ...MORE_SINGLES];

export const RARITY: Record<string, string> = {
  "charizard-ex-obsidian-flames": "Special Illustration Rare",
  "umbreon-vmax-alt-art": "Alt Art Secret",
  "charizard-vmax-shining": "Shiny Vault",
  "ahri-spirit-blossom": "Signature Illustration",
};

// Cart economics (visual-only; checkout still hands off to WhatsApp).
export const FREE_SHIPPING = 25000; // R$ 250,00
export const SHIPPING = 2800; // R$ 28,00
export const COUPON = { code: "ARCADE10", pct: 10 };

const OFFER_SELLERS: Seller[] = [
  { id: "dojo-sp", name: "DojoCards SP", rating: 98.7, sales: 1203, ships: "48h" },
  { id: "bruno-tcg", name: "Bruno TCG", rating: 99.1, sales: 3410, ships: "24h" },
  { id: "mari-pulls", name: "Mari Pulls", rating: 98.2, sales: 980, ships: "72h" },
];

// A deterministic 6-month history shape so re-renders are stable (no random).
function historyFor(slug: string): SingleDetail["priceHistory"] {
  const seed = [...slug].reduce((n, c) => n + c.charCodeAt(0), 0);
  const base = [40, 34, 52, 60, 74, 92];
  return ["FEV", "MAR", "ABR", "MAI", "JUN", "JUL"].map((label, i) => ({
    label,
    value: Math.min(96, base[i] + ((seed + i * 7) % 14)),
  }));
}

function offersFor(s: Single): SingleDetail["offers"] {
  const conditions = [s.condition, "NM sem grade", "LP"];
  return OFFER_SELLERS.map((seller, i) => ({
    seller,
    condition: conditions[i] ?? "NM",
    price: Math.round(s.price * [1.04, 0.84, 0.71][i]),
    ships: `envia em ${seller.ships}`,
  }));
}

// Builds a full detail for any single (real or mock) — synthesizes the mock
// sellers/history/market data the backend doesn't provide, around the real price.
export function buildSingleDetail(s: Single): SingleDetail {
  return {
    ...s,
    rarity: RARITY[s.slug],
    marketAvg: Math.round(s.price * 1.08),
    seller: OFFICIAL_SELLER,
    priceHistory: historyFor(s.slug),
    offers: offersFor(s),
  };
}

export function buildSealedDetail(s: Sealed): SealedDetail {
  const [content, metaLanguage] = s.meta.split(" · ");
  const specs = s.language
    ? [
        { key: "JOGO", value: s.gameLabel },
        { key: "IDIOMA", value: s.language },
        { key: "GARANTIA", value: "Lacre original de fábrica" },
      ]
    : [
        { key: "CONTEUDO", value: content ?? "24 packs" },
        { key: "IDIOMA", value: metaLanguage ?? "Inglês" },
        { key: "GARANTIA", value: "Lacre original de fábrica" },
      ];
  return {
    ...s,
    installmentsNote: `12x de ${(s.price / 12 / 100).toLocaleString("pt-BR", { style: "currency", currency: "BRL" })}`,
    stockLeft: s.qty,
    stockTotal: s.qty,
    specs,
  };
}

// Curated detail override for a specific slug (mock demo card), else null so the
// caller falls back to buildSingleDetail over the real catalog.
export function curatedSingle(slug: string): SingleDetail | null {
  return SINGLE_DETAILS[slug] ?? null;
}

export function sealedBySlug(slug: string): SealedDetail | null {
  if (SEALED_DETAILS[slug]) {
    return SEALED_DETAILS[slug];
  }
  const s = SEALED.find((c) => c.slug === slug);
  return s ? buildSealedDetail(s) : null;
}
