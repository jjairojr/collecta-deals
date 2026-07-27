// Marketplace domain types. All prices are integers in cents (see lib/format).
// This is the design-first shape: it covers what the 5 screens render, ahead of
// the Go backend actually supplying sealed/sellers/history data.

// Internal game id from the Go backend (pokemon, onepiece, riftbound, lorcana,
// gundam, …). Kept as a string so the catalog isn't limited to the design's 3.
export type GameId = string;

export interface Game {
  id: GameId;
  name: string;
  count: number;
  hue: "pink" | "royal" | "soft";
}

export type ProductKind = "single" | "sealed";

export interface Seller {
  id: string;
  name: string;
  rating: number;
  sales: number;
  ships: string;
}

export interface Offer {
  seller: Seller;
  condition: string;
  price: number;
  ships: string;
}

// A single card listing (grid item + product page).
export interface Single {
  kind: "single";
  slug: string;
  game: GameId;
  gameLabel: string;
  name: string;
  set: string;
  number: string;
  condition: string;
  language: string;
  price: number;
  wasPrice?: number;
  qty: number;
  grade?: string;
  imageURL?: string;
}

export interface SingleDetail extends Single {
  rarity?: string;
  marketAvg?: number;
  seller: Seller;
  offers: Offer[];
  priceHistory: PricePoint[];
}

export interface PricePoint {
  label: string;
  value: number;
}

// A sealed product (booster box, ETB, etc.).
export interface Sealed {
  kind: "sealed";
  slug: string;
  game: GameId;
  gameLabel: string;
  name: string;
  set: string;
  meta: string;
  language?: string;
  price: number;
  qty: number;
  badge?: string;
  imageURL?: string;
  featured?: boolean;
}

export interface SealedDetail extends Sealed {
  stockLeft: number;
  stockTotal: number;
  specs: { key: string; value: string }[];
}

export interface HighScore {
  position: string;
  player: string;
  trades: number;
  score: number;
}

// A line in the buyer's cart. Coupon/shipping are visual-only; checkout hands
// off to WhatsApp.
export interface CartLine {
  slug: string;
  kind: ProductKind;
  name: string;
  meta: string;
  seller: string;
  price: number;
  qty: number;
  // Available stock — caps how many the buyer can put in the cart.
  stock?: number;
  imageURL?: string;
}
