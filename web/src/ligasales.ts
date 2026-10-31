import type { SaleChannel, SaleRow, WriteOffReason } from "./api";

export type Channel = SaleChannel;

export const channels: Channel[] = ["liga", "myp", "site", "balcao", "outros"];

export const channelLabels: Record<Channel, string> = {
  liga: "Liga",
  myp: "MyP",
  site: "Site",
  balcao: "Balcão",
  outros: "Outros",
};

export const writeOffs: WriteOffReason[] = ["aberto", "perda", "brinde", "uso"];

export const writeOffLabels: Record<WriteOffReason, string> = {
  aberto: "Aberto",
  perda: "Perda",
  brinde: "Brinde",
  uso: "Uso próprio",
};

const ORDER_SUFFIX = /\s*—?\s*(Liga|MyP|Site) #(\d+)\s*$/i;

export function orderNumber(buyer: string | undefined): string {
  return (buyer ?? "").match(ORDER_SUFFIX)?.[2] ?? "";
}

export function buyerName(buyer: string | undefined): string {
  return (buyer ?? "")
    .replace(ORDER_SUFFIX, "")
    .replace(/\s*—\s*Liga\s*$/i, "")
    .replace(/[—-]\s*$/, "")
    .trim();
}

export function isWriteOff(s: SaleRow): boolean {
  return Boolean(s.writeOff);
}

export function isBRL(s: SaleRow): boolean {
  return s.sellCurrency !== "USD";
}

export function channelOfSale(s: SaleRow): Channel {
  return s.channel ?? "outros";
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
    return `pedido:${channelOfSale(s)}:${order}`;
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
        channel: channelOfSale(s),
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
    p.units += isWriteOff(s) ? 0 : s.qty > 0 ? s.qty : 1;
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
