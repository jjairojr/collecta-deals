import type { SaleRow } from "./api";

// Sales are labelled by hand in the buyer field, so the channel has to be read
// back out of it. The liga-vendas skill writes "<nome> — Liga #<pedido>"; a baixa
// done by hand in the dashboard often leaves the field empty, and those are Liga
// sales too — they just lost their order number, hence a channel of their own
// instead of being lumped with a named buyer.
export type Channel = "liga" | "sem-marca" | "shopee" | "outros";

export const channels: Channel[] = ["liga", "sem-marca", "shopee", "outros"];

export const channelLabels: Record<Channel, string> = {
  liga: "Liga",
  "sem-marca": "Sem marca",
  shopee: "Shopee",
  outros: "Outros",
};

export function channelOf(buyer: string | undefined): Channel {
  const b = (buyer ?? "").trim();
  if (b === "") {
    return "sem-marca";
  }
  if (/liga/i.test(b)) {
    return "liga";
  }
  if (/shopee/i.test(b)) {
    return "shopee";
  }
  return "outros";
}

export function orderNumber(buyer: string | undefined): string {
  return (buyer ?? "").match(/Liga #(\d+)/i)?.[1] ?? "";
}

// buyerName drops the "— Liga #<pedido>" suffix so the order number is shown once.
export function buyerName(buyer: string | undefined): string {
  return (buyer ?? "")
    .replace(/—?\s*Liga #\d+/i, "")
    .replace(/[—-]\s*$/, "")
    .trim();
}

// A sale priced at a token R$ 1 per unit is bookkeeping, not revenue: it is how
// opening a sealed case is written off the ledger (four ME5 boxes "sold" for
// R$ 4). Left in, a single one of these shows up as a R$ 311 loss.
export function isSymbolic(s: SaleRow): boolean {
  const units = s.qty > 0 ? s.qty : 1;
  return s.valueBRL / units <= 1;
}

// Sales in USD are the TCGplayer side of the business, not the Liga store.
export function isBRL(s: SaleRow): boolean {
  return s.sellCurrency !== "USD";
}

export interface SalesPackage {
  key: string;
  order: string;
  buyer: string;
  date: string;
  channel: Channel;
  games: string[];
  units: number;
  revenueBRL: number;
  costBRL: number;
  profitBRL: number;
  marginPct: number;
  items: SaleRow[];
}

// packageKey identifies the shipped package. The order number is the real key
// when it survived the baixa; without it, one buyer's items sold on one day are
// the best available stand-in for one package.
function packageKey(s: SaleRow): string {
  const order = orderNumber(s.buyer);
  if (order) {
    return `pedido:${order}`;
  }
  return `${buyerName(s.buyer).toLowerCase()}|${s.sellDate ?? ""}`;
}

export function buildPackages(sales: SaleRow[]): SalesPackage[] {
  const byKey = new Map<string, SalesPackage>();
  for (const s of sales) {
    const key = packageKey(s);
    let p = byKey.get(key);
    if (!p) {
      p = {
        key,
        order: orderNumber(s.buyer),
        buyer: buyerName(s.buyer),
        date: s.sellDate ?? "",
        channel: channelOf(s.buyer),
        games: [],
        units: 0,
        revenueBRL: 0,
        costBRL: 0,
        profitBRL: 0,
        marginPct: 0,
        items: [],
      };
      byKey.set(key, p);
    }
    if (!p.buyer && buyerName(s.buyer)) {
      p.buyer = buyerName(s.buyer);
    }
    if (!p.games.includes(s.gameName)) {
      p.games.push(s.gameName);
    }
    p.units += s.qty > 0 ? s.qty : 1;
    p.revenueBRL += s.valueBRL;
    p.costBRL += s.costBRL;
    p.profitBRL += s.profitBRL;
    p.items.push(s);
  }
  const out = [...byKey.values()];
  for (const p of out) {
    p.marginPct = p.costBRL > 0 ? (p.profitBRL / p.costBRL) * 100 : 0;
    p.items.sort((a, b) => b.valueBRL - a.valueBRL);
  }
  out.sort((a, b) => b.date.localeCompare(a.date) || b.revenueBRL - a.revenueBRL);
  return out;
}

export interface SalesTotals {
  packages: number;
  items: number;
  units: number;
  revenueBRL: number;
  costBRL: number;
  profitBRL: number;
  marginPct: number;
}

export function totalsOf(packages: SalesPackage[]): SalesTotals {
  const t: SalesTotals = {
    packages: packages.length,
    items: 0,
    units: 0,
    revenueBRL: 0,
    costBRL: 0,
    profitBRL: 0,
    marginPct: 0,
  };
  for (const p of packages) {
    t.items += p.items.length;
    t.units += p.units;
    t.revenueBRL += p.revenueBRL;
    t.costBRL += p.costBRL;
    t.profitBRL += p.profitBRL;
  }
  t.marginPct = t.costBRL > 0 ? (t.profitBRL / t.costBRL) * 100 : 0;
  return t;
}

export interface DayPoint {
  day: string;
  revenueBRL: number;
  profitBRL: number;
}

export function dailySeries(packages: SalesPackage[]): DayPoint[] {
  const byDay = new Map<string, DayPoint>();
  for (const p of packages) {
    if (!p.date) {
      continue;
    }
    const d = byDay.get(p.date);
    if (d) {
      d.revenueBRL += p.revenueBRL;
      d.profitBRL += p.profitBRL;
    } else {
      byDay.set(p.date, { day: p.date, revenueBRL: p.revenueBRL, profitBRL: p.profitBRL });
    }
  }
  return [...byDay.values()].sort((a, b) => a.day.localeCompare(b.day));
}
