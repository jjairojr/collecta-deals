export interface Deal {
  number: string;
  name: string;
  set?: string;
  rarity: string;
  variant: string;
  source: string;
  buyUrl: string;
  tcgUrl: string;
  lowBRL: number;
  buyUSD: number;
  sellUSD: number;
  marginPct: number;
  profitUSD: number;
  verified: boolean;
  usListings: number;
  usQty: number;
  brCopies: number;
  brSellers: number;
}

export interface DealsResponse {
  ready: boolean;
  refreshing: boolean;
  updatedAt: string;
  fxRate: number;
  count: number;
  deals: Deal[];
  sets: string[];
}

export interface Status {
  ready: boolean;
  refreshing: boolean;
  updatedAt: string;
  fxRate: number;
  listings: number;
  prices: number;
}

// DealSource scopes a deals query to one Brazilian marketplace. "liga" matches
// every per-game Liga source; "" keeps the historical cheapest-across-sources
// behaviour.
export type DealSource = "" | "liga" | "mypcards";

export interface DealFilters {
  minMargin: number;
  minPrice: number;
  sort: string;
  set: string;
  source: DealSource;
  limit: number;
  verifiedOnly: boolean;
  spOnly: boolean;
  ignoreMinMargin: boolean;
}

const base = "/api";

// The active game scopes every BR tracking call (?game=). One Piece is the
// default so existing links keep working; Pokémon adds the second game.
let currentGame = readGame();

function readGame(): string {
  try {
    const u = new URLSearchParams(window.location.search).get("game");
    if (u) {
      return u;
    }
    return localStorage.getItem("game") ?? "onepiece";
  } catch {
    return "onepiece";
  }
}

export function getGame(): string {
  return currentGame;
}

export function setGame(g: string): void {
  currentGame = g;
  try {
    localStorage.setItem("game", g);
  } catch {
    // ignore storage failures (private mode)
  }
}

// gp appends the active game to a params object so BR tracking endpoints resolve
// to the right per-game store.
function gp(params: URLSearchParams): URLSearchParams {
  params.set("game", currentGame);
  return params;
}

export interface GameInfo {
  id: string;
  name: string;
  hasDeals: boolean;
  hasMyP: boolean;
  multiLang: boolean;
}

let gamesCache: GameInfo[] = [];

export async function getGames(): Promise<{ default: string; games: GameInfo[] }> {
  const data = await getJSON<{ default: string; games: GameInfo[] }>(`${base}/games`);
  gamesCache = data.games;
  return data;
}

// gameHasDeals reports whether a game runs the US deals pipeline, from the last
// /api/games response. Before that loads, only Pokémon is assumed BR-only.
export function gameHasDeals(id: string): boolean {
  const info = gamesCache.find((g) => g.id === id);
  return info ? info.hasDeals : id !== "pokemon";
}

// gameIsMultiLang reports whether a game's Brazil market prices every language as
// the same product, making a listing's language worth showing. Before /api/games
// loads, only Pokémon is assumed multi-language.
export function gameIsMultiLang(id: string): boolean {
  const info = gamesCache.find((g) => g.id === id);
  return info ? info.multiLang : id === "pokemon";
}

async function getJSON<T>(url: string): Promise<T> {
  const res = await fetch(url);
  if (!res.ok) {
    throw new Error(`request failed: ${res.status}`);
  }
  return (await res.json()) as T;
}

export function getStatus(): Promise<Status> {
  return getJSON<Status>(`${base}/status?${gp(new URLSearchParams()).toString()}`);
}

export function getDeals(filters: DealFilters): Promise<DealsResponse> {
  const params = gp(new URLSearchParams({
    minMargin: String(filters.ignoreMinMargin ? -1e9 : filters.minMargin),
    minPrice: String(filters.minPrice),
    sort: filters.sort,
    limit: String(filters.limit),
    requireInStock: String(filters.verifiedOnly),
    spOnly: String(filters.spOnly),
  }));
  if (filters.set) {
    params.set("set", filters.set);
  }
  if (filters.source) {
    params.set("source", filters.source);
  }
  return getJSON<DealsResponse>(`${base}/deals?${params.toString()}`);
}

export function searchDeals(query: string, source: DealSource = "", limit = 200): Promise<DealsResponse> {
  const params = gp(new URLSearchParams({ q: query, limit: String(limit) }));
  if (source) {
    params.set("source", source);
  }
  return getJSON<DealsResponse>(`${base}/search?${params.toString()}`);
}

export async function triggerRefresh(): Promise<void> {
  await fetch(`${base}/refresh?${gp(new URLSearchParams()).toString()}`, { method: "POST" });
}

export interface CardTrend {
  set?: string;
  number: string;
  name: string;
  lowBRL: number;
  prevBRL: number;
  deltaPct: number;
  url: string;
}

export type TrendRange = "daily" | "weekly" | "monthly";

export interface TrendsResponse {
  set: string;
  range: TrendRange;
  date: string;
  prevDate: string;
  count: number;
  trends: CardTrend[];
}

export interface CardSeller {
  storeId: number;
  storeName: string;
  units: number;
  revenueBRL: number;
  priceBRL: number;
  language?: string;
}

export interface LangSale {
  code: string;
  units: number;
  revenueBRL: number;
}

export interface CardSale {
  set?: string;
  number: string;
  name: string;
  url?: string;
  units: number;
  revenueBRL: number;
  sellers?: CardSeller[];
  languages?: LangSale[];
}

export interface SnapshotSales {
  date: string;
  prevDate: string;
  capturedAt?: string;
  prevCapturedAt?: string;
  units: number;
  revenueBRL: number;
  cards: CardSale[];
}

export function getSalesBySnapshot(
  set: string,
  from = "",
  to = "",
): Promise<{ set: string; from: string; to: string; snapshots: SnapshotSales[] }> {
  const params = gp(new URLSearchParams({ set }));
  if (from) {
    params.set("from", from);
  }
  if (to) {
    params.set("to", to);
  }
  return getJSON<{ set: string; from: string; to: string; snapshots: SnapshotSales[] }>(
    `${base}/tracking/sold-by-snapshot?${params.toString()}`,
  );
}

export interface StoreStat {
  storeId: number;
  storeName: string;
  unitsSold: number;
  revenueBRL: number;
  cards: CardSale[];
}

export interface LeaderboardResponse {
  set: string;
  from: string;
  to: string;
  sort: string;
  stores: StoreStat[];
}

export interface PricePoint {
  date: string;
  lowBRL: number;
}

export interface CardHistoryResponse {
  set: string;
  number: string;
  points: PricePoint[];
}

export interface DatesResponse {
  set: string;
  dates: string[];
}

export function getTrends(set = "OP-16", range: TrendRange = "daily"): Promise<TrendsResponse> {
  const params = gp(new URLSearchParams({ set, range }));
  return getJSON<TrendsResponse>(`${base}/tracking/trends?${params.toString()}`);
}

export function getLeaderboard(
  set: string,
  sort: "units" | "revenue",
  from = "",
  to = "",
): Promise<LeaderboardResponse> {
  const params = gp(new URLSearchParams({ set, sort }));
  if (from) {
    params.set("from", from);
  }
  if (to) {
    params.set("to", to);
  }
  return getJSON<LeaderboardResponse>(`${base}/tracking/leaderboard?${params.toString()}`);
}

export interface LatestSnapshot {
  capturedAt: string;
  date: string;
  set: string;
}

export function getLatestSnapshot(): Promise<LatestSnapshot> {
  return getJSON<LatestSnapshot>(`${base}/tracking/latest?${gp(new URLSearchParams()).toString()}`);
}

export function getTrackingDates(set = "OP-16"): Promise<DatesResponse> {
  const params = gp(new URLSearchParams({ set }));
  return getJSON<DatesResponse>(`${base}/tracking/dates?${params.toString()}`);
}

export function getRecentSnapshots(limit = 12): Promise<{ dates: string[] }> {
  const params = gp(new URLSearchParams({ limit: String(limit) }));
  return getJSON<{ dates: string[] }>(`${base}/tracking/snapshots?${params.toString()}`);
}

export interface StoreInventoryStat {
  storeId: number;
  storeName: string;
  units: number;
  cards: number;
  valueBRL: number;
  topCardNumber: string;
  topCardName: string;
  topCardBRL: number;
}

export interface CardHolder {
  storeId: number;
  storeName: string;
  quantity: number;
}

export interface ExpensiveCard {
  number: string;
  name: string;
  lowBRL: number;
  totalQty: number;
  stores: number;
  holders: CardHolder[];
}

export interface InventorySummary {
  date: string;
  activeStores: number;
  totalUnits: number;
  totalValue: number;
  stores: StoreInventoryStat[];
  expensive: ExpensiveCard[];
}

export interface InventoryResponse {
  set: string;
  ready: boolean;
  summary: InventorySummary;
}

export function getInventory(set = "OP-16"): Promise<InventoryResponse> {
  const params = gp(new URLSearchParams({ set }));
  return getJSON<InventoryResponse>(`${base}/tracking/inventory?${params.toString()}`);
}

export interface BuyoutCandidate {
  set?: string;
  number: string;
  name: string;
  url: string;
  floor: number;
  nextFloor: number;
  liftPct: number;
  buyoutCost: number;
  shippingCost: number;
  storeCount: number;
  copiesToClear: number;
  profitBRL: number;
  nmSupply: number;
  sellers: number;
  score: number;
  sellUSD?: number;
  tcgUrl?: string;
}

export type BuyoutSort = "best" | "score" | "lift" | "profit" | "copies";

export type BuyoutMode = "buyout" | "snipe";

export interface BuyoutResponse {
  set: string;
  date: string;
  ready: boolean;
  budget: number;
  minFloor: number;
  shipping: number;
  fxRate: number;
  sort: string;
  mode?: BuyoutMode;
  minGap?: number;
  candidates: BuyoutCandidate[];
}

export function getTrackingSets(): Promise<{ sets: string[] }> {
  return getJSON<{ sets: string[] }>(`${base}/tracking/sets?${gp(new URLSearchParams()).toString()}`);
}

export function getBuyout(
  set: string,
  budget: number,
  minFloor: number,
  shipping: number,
  sort: BuyoutSort = "best",
  mainChars = false,
  mode: BuyoutMode = "buyout",
  minGap = 50,
  spOnly = false,
): Promise<BuyoutResponse> {
  const params = gp(new URLSearchParams({
    set,
    budget: String(budget),
    minFloor: String(minFloor),
    shipping: String(shipping),
    sort,
  }));
  if (mainChars) {
    params.set("chars", "main");
  }
  if (spOnly) {
    params.set("sp", "true");
  }
  if (mode === "snipe") {
    params.set("mode", "snipe");
    params.set("minGap", String(minGap));
  }
  return getJSON<BuyoutResponse>(`${base}/tracking/buyout?${params.toString()}`);
}

export async function triggerCapture(sealed = false): Promise<void> {
  const path = sealed ? "tracking/capture-sealed" : "tracking/capture";
  await fetch(`${base}/${path}?${gp(new URLSearchParams()).toString()}`, { method: "POST" });
}

export interface TrackCard {
  number: string;
  name: string;
  lowBRL: number;
}

export function getCards(set: string): Promise<{ set: string; cards: TrackCard[] }> {
  const params = gp(new URLSearchParams({ set }));
  return getJSON<{ set: string; cards: TrackCard[] }>(`${base}/tracking/cards?${params.toString()}`);
}

// imgVersion busts browser-cached card images (max-age=86400) when the server
// side has re-resolved them — bumped after the poisoned set-insensitive cache
// served wrong art for per-set-numbered games.
const imgVersion = "2";

export function cardImageURL(set: string, number: string, productID?: number): string {
  const params = gp(new URLSearchParams({ set, number, v: imgVersion }));
  if (productID) {
    params.set("productID", String(productID));
  }
  return `${base}/card-image?${params.toString()}`;
}

export function tcgProductImageURL(productID: number): string {
  return `https://product-images.tcgplayer.com/fit-in/200x279/${productID}.jpg`;
}

export function tcgProductURL(productID: number): string {
  return `https://www.tcgplayer.com/product/${productID}`;
}

// productIDFromTcgURL pulls the TCGplayer product id out of a tcgUrl
// (".../product/696052"), so card art can be resolved by the per-variant
// TCGplayer image instead of the number-keyed Liga fallback (which serves base
// art for special prints like "Kuzan (Manga)").
export function productIDFromTcgURL(url?: string): number | undefined {
  if (!url) {
    return undefined;
  }
  const m = url.match(/\/product\/(\d+)/);
  return m ? Number(m[1]) : undefined;
}

// dealCardKey resolves a deal's set/number for image and selection lookups.
// Liga buyUrls carry ?ed=/&num= query params; other sources (mypcards) use a
// path-based product URL, so fall back to the deal's own fields, which every
// source populates.
function dealCardKey(deal: Deal): { set: string; number: string } | null {
  let set = deal.set ?? "";
  let number = deal.number;
  try {
    const u = new URL(deal.buyUrl);
    set = u.searchParams.get("ed") ?? set;
    number = u.searchParams.get("num") ?? number;
  } catch {
    // Non-URL buyUrl — fall through to the deal fields.
  }
  if (!set) {
    return null;
  }
  return { set, number };
}

export function dealSelection(
  deal: Deal,
): { set: string; number: string; name: string; priceBRL: number } | null {
  const key = dealCardKey(deal);
  if (!key) {
    return null;
  }
  return { set: key.set, number: key.number, name: deal.name, priceBRL: deal.lowBRL };
}

export function dealImageURL(deal: Deal): string | null {
  const key = dealCardKey(deal);
  if (!key) {
    return null;
  }
  const params = gp(new URLSearchParams({ set: key.set, number: key.number, v: imgVersion }));
  const ligaUrl = deal.buyUrl.includes("ed=");
  if (ligaUrl) {
    // Liga: let the backend resolve the number-keyed Liga page image, unchanged.
    params.set("url", deal.buyUrl);
  } else {
    // Other sources carry no Liga page; use the exact TCGplayer product art.
    const productID = productIDFromTcgURL(deal.tcgUrl);
    if (productID) {
      params.set("productID", String(productID));
    }
  }
  return `${base}/card-image?${params.toString()}`;
}

export async function exportImage(cards: { set: string; number: string }[]): Promise<Blob> {
  const res = await fetch(`${base}/export/image?${gp(new URLSearchParams()).toString()}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ cards }),
  });
  if (!res.ok) {
    throw new Error(`export failed: ${res.status}`);
  }
  return await res.blob();
}

export interface Trade {
  id: string;
  kind?: "sealed";
  number: string;
  name: string;
  set: string;
  variant?: string;
  condition?: string;
  qty: number;
  buyBRL: number;
  shippingBRL: number;
  store?: string;
  buyDate?: string;
  delivered?: boolean;
  refUSD?: number;
  manualBRL?: number;
  status: "holding" | "sold";
  sellPrice?: number;
  sellCurrency?: "BRL" | "USD";
  sellDate?: string;
  buyer?: string;
  notes?: string;
  createdAt: string;
  updatedAt: string;
}

export interface TradeView extends Trade {
  costBRL: number;
  marketUSD: number;
  marketKnown: boolean;
  tcgUrl?: string;
  valueBRL: number;
  profitBRL: number;
  marginPct: number;
  realized: boolean;
}

export interface PortfolioSummary {
  targetPct: number;
  fxRate: number;
  holdings: number;
  investedBRL: number;
  marketBRL: number;
  unrealizedBRL: number;
  sold: number;
  costOfSoldBRL: number;
  proceedsBRL: number;
  realizedBRL: number;
  totalPnLBRL: number;
}

export interface PortfolioResponse {
  targetPct: number;
  fxRate: number;
  summary: PortfolioSummary;
  trades: TradeView[];
}

export interface QuoteMatch {
  number: string;
  name: string;
  set: string;
  variant?: string;
  marketUSD: number;
  marketBRL: number;
  ligaLowBRL?: number;
  ligaAvgBRL?: number;
  ligaUrl?: string;
  productID?: number;
}

export interface QuoteItem {
  number: string;
  name: string;
  set: string;
  variant?: string;
  qty: number;
  unitBRL: number;
  pct?: number;
  marketUSD?: number;
  ligaLowBRL?: number;
  ligaAvgBRL?: number;
  ligaUrl?: string;
  productID?: number;
}

export type QuoteMarket = "tcg" | "liga";

export interface Quote {
  id: string;
  name: string;
  pct: number;
  market?: QuoteMarket;
  fxRate?: number;
  items: QuoteItem[];
  notes?: string;
  createdAt: string;
  updatedAt: string;
}

export function getPortfolio(pct: number): Promise<PortfolioResponse> {
  const params = gp(new URLSearchParams({ pct: String(pct) }));
  return getJSON<PortfolioResponse>(`${base}/trades?${params.toString()}`);
}

export function getQuote(
  q: string,
  limit = 25,
  kind?: "sealed",
): Promise<{ fxRate: number; matches: QuoteMatch[] }> {
  const params = gp(new URLSearchParams({ q, limit: String(limit) }));
  if (kind) {
    params.set("kind", kind);
  }
  return getJSON<{ fxRate: number; matches: QuoteMatch[] }>(
    `${base}/trades/quote?${params.toString()}`,
  );
}

export interface GamePortfolio {
  game: GameInfo;
  summary: PortfolioSummary;
}

export interface AllPortfolioResponse {
  targetPct: number;
  total: PortfolioSummary;
  games: GamePortfolio[];
}

export function getAllPortfolio(pct: number): Promise<AllPortfolioResponse> {
  return getJSON<AllPortfolioResponse>(`${base}/portfolio/all?pct=${pct}`);
}

async function sendJSON<T>(url: string, method: string, body: unknown): Promise<T> {
  const res = await fetch(url, {
    method,
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (!res.ok) {
    throw new Error(`request failed: ${res.status}`);
  }
  return (await res.json()) as T;
}

export function listQuotes(): Promise<{ quotes: Quote[] }> {
  return getJSON<{ quotes: Quote[] }>(`${base}/quotes?${gp(new URLSearchParams()).toString()}`);
}

export function createQuote(q: Omit<Quote, "id" | "createdAt" | "updatedAt">): Promise<Quote> {
  return sendJSON<Quote>(`${base}/quotes?${gp(new URLSearchParams()).toString()}`, "POST", q);
}

export function updateQuote(id: string, q: Omit<Quote, "id" | "createdAt" | "updatedAt">): Promise<Quote> {
  return sendJSON<Quote>(`${base}/quotes/${id}?${gp(new URLSearchParams()).toString()}`, "PUT", q);
}

export async function deleteQuote(id: string): Promise<void> {
  const res = await fetch(`${base}/quotes/${id}?${gp(new URLSearchParams()).toString()}`, { method: "DELETE" });
  if (!res.ok) {
    throw new Error(`delete failed: ${res.status}`);
  }
}

export function createTrade(t: Partial<Trade>): Promise<Trade> {
  return sendJSON<Trade>(`${base}/trades?${gp(new URLSearchParams()).toString()}`, "POST", t);
}

export function updateTrade(id: string, t: Partial<Trade>): Promise<Trade> {
  return sendJSON<Trade>(`${base}/trades/${id}?${gp(new URLSearchParams()).toString()}`, "PUT", t);
}

export interface SellTradeInput {
  qty: number;
  sellPrice: number;
  sellCurrency: "BRL" | "USD";
  sellDate?: string;
  buyer?: string;
}

export function sellTrade(id: string, sale: SellTradeInput): Promise<Trade> {
  return sendJSON<Trade>(`${base}/trades/${id}/sell?${gp(new URLSearchParams()).toString()}`, "POST", sale);
}

export async function deleteTrade(id: string): Promise<void> {
  const res = await fetch(`${base}/trades/${id}?${gp(new URLSearchParams()).toString()}`, { method: "DELETE" });
  if (!res.ok) {
    throw new Error(`delete failed: ${res.status}`);
  }
}
