import type { Expense, SaleRow } from "./api";
import { channelOf, isBRL, isSymbolic } from "./ligasales";

export type Side = "liga" | "fora";
export type Kind = "single" | "sealed" | "accessory";

export const sides: Side[] = ["liga", "fora"];
export const kinds: Kind[] = ["single", "sealed", "accessory"];

export const sideLabels: Record<Side, string> = {
  liga: "Liga",
  fora: "Fora da Liga",
};

export const kindLabels: Record<Kind, string> = {
  single: "Singles",
  sealed: "Selados",
  accessory: "Acessórios",
};

export function kindOf(sale: SaleRow): Kind {
  return sale.kind ?? "single";
}

export function isUSD(sale: SaleRow): boolean {
  return !isBRL(sale);
}

export function sideOf(sale: SaleRow, unmarkedIsLiga: boolean): Side {
  if (isUSD(sale)) {
    return "fora";
  }
  const channel = channelOf(sale.buyer);
  if (channel === "liga") {
    return "liga";
  }
  if (channel === "sem-marca") {
    return unmarkedIsLiga ? "liga" : "fora";
  }
  return "fora";
}

export function monthOf(date: string): string {
  return (date ?? "").slice(0, 7);
}

export function monthIndex(month: string): number {
  return Number(month.slice(0, 4)) * 12 + (Number(month.slice(5, 7)) - 1);
}

export function shiftMonth(month: string, by: number): string {
  const idx = monthIndex(month) + by;
  const year = Math.floor(idx / 12);
  const m = idx - year * 12 + 1;
  return `${year}-${String(m).padStart(2, "0")}`;
}

export function currentMonth(): string {
  return new Date().toISOString().slice(0, 7);
}

export function monthLabel(month: string): string {
  const label = new Date(`${month}-01T12:00:00`).toLocaleDateString("pt-BR", {
    month: "short",
    year: "2-digit",
  });
  return label.replace(".", "").replace(" de ", "/");
}

export function monthRange(from: string, to: string): string[] {
  const out: string[] = [];
  if (!from || !to || monthIndex(from) > monthIndex(to)) {
    return out;
  }
  for (let m = from; monthIndex(m) <= monthIndex(to); m = shiftMonth(m, 1)) {
    out.push(m);
  }
  return out;
}

export interface FinanceFilters {
  from: string;
  to: string;
  games: string[];
  kinds: Kind[];
  sides: Side[];
  unmarkedIsLiga: boolean;
  hideSymbolic: boolean;
  includeUSD: boolean;
}

export function saleMonths(sales: SaleRow[]): string[] {
  const found = new Set<string>();
  for (const s of sales) {
    const m = monthOf(s.sellDate ?? "");
    if (m) {
      found.add(m);
    }
  }
  return [...found].sort();
}

export function expenseMonths(expenses: Expense[]): string[] {
  const found = new Set<string>();
  for (const e of expenses) {
    const m = monthOf(e.date);
    if (m) {
      found.add(m);
    }
  }
  return [...found].sort();
}

export function eligibleSales(sales: SaleRow[], hideSymbolic: boolean, includeUSD = true): SaleRow[] {
  const kept = includeUSD ? sales : sales.filter(isBRL);
  return hideSymbolic ? kept.filter((s) => !isSymbolic(s)) : kept;
}

export function matchesScope(s: SaleRow, f: FinanceFilters): boolean {
  if (f.games.length > 0 && !f.games.includes(s.game)) {
    return false;
  }
  if (f.kinds.length > 0 && !f.kinds.includes(kindOf(s))) {
    return false;
  }
  return f.sides.length === 0 || f.sides.includes(sideOf(s, f.unmarkedIsLiga));
}

export function filterSales(sales: SaleRow[], f: FinanceFilters): SaleRow[] {
  return eligibleSales(sales, f.hideSymbolic, f.includeUSD).filter((s) => {
    const month = monthOf(s.sellDate ?? "");
    if (!month || monthIndex(month) < monthIndex(f.from) || monthIndex(month) > monthIndex(f.to)) {
      return false;
    }
    return matchesScope(s, f);
  });
}

export function undatedSales(sales: SaleRow[], f: FinanceFilters): SaleRow[] {
  return eligibleSales(sales, f.hideSymbolic, f.includeUSD).filter(
    (s) => !monthOf(s.sellDate ?? "") && matchesScope(s, f),
  );
}

export function expenseInMonth(e: Expense, month: string): boolean {
  if (!e.recurring) {
    return monthOf(e.date) === month;
  }
  if (monthIndex(monthOf(e.date)) > monthIndex(month)) {
    return false;
  }
  return !e.endDate || monthIndex(month) <= monthIndex(monthOf(e.endDate));
}

export function expensesInMonth(expenses: Expense[], month: string): Expense[] {
  return expenses.filter((e) => expenseInMonth(e, month));
}

export function expensesInRange(expenses: Expense[], months: string[]): number {
  let total = 0;
  for (const month of months) {
    for (const e of expensesInMonth(expenses, month)) {
      total += e.amountBRL;
    }
  }
  return total;
}

export interface CategoryTotal {
  category: string;
  amountBRL: number;
  recurringBRL: number;
}

export function expensesByCategory(expenses: Expense[], months: string[]): CategoryTotal[] {
  const byCategory = new Map<string, CategoryTotal>();
  for (const month of months) {
    for (const e of expensesInMonth(expenses, month)) {
      const key = e.category?.trim() || "Sem categoria";
      const row = byCategory.get(key) ?? { category: key, amountBRL: 0, recurringBRL: 0 };
      row.amountBRL += e.amountBRL;
      if (e.recurring) {
        row.recurringBRL += e.amountBRL;
      }
      byCategory.set(key, row);
    }
  }
  return [...byCategory.values()].sort((a, b) => b.amountBRL - a.amountBRL);
}

export interface Totals {
  units: number;
  items: number;
  revenueBRL: number;
  costBRL: number;
  grossBRL: number;
  marginPct: number;
}

export function emptyTotals(): Totals {
  return { units: 0, items: 0, revenueBRL: 0, costBRL: 0, grossBRL: 0, marginPct: 0 };
}

function addSale(t: Totals, s: SaleRow): void {
  t.items += 1;
  t.units += s.qty > 0 ? s.qty : 1;
  t.revenueBRL += s.valueBRL;
  t.costBRL += s.costBRL;
  t.grossBRL += s.profitBRL;
  t.marginPct = t.revenueBRL > 0 ? (t.grossBRL / t.revenueBRL) * 100 : 0;
}

export function totalsOf(sales: SaleRow[]): Totals {
  const t = emptyTotals();
  for (const s of sales) {
    addSale(t, s);
  }
  return t;
}

export function groupTotals<K extends string>(
  sales: SaleRow[],
  keyOf: (s: SaleRow) => K,
): Map<K, Totals> {
  const out = new Map<K, Totals>();
  for (const s of sales) {
    const key = keyOf(s);
    const t = out.get(key) ?? emptyTotals();
    addSale(t, s);
    out.set(key, t);
  }
  return out;
}

export interface MonthRow {
  month: string;
  revenueBRL: number;
  ligaBRL: number;
  foraBRL: number;
  costBRL: number;
  grossBRL: number;
  expensesBRL: number;
  netBRL: number;
}

export function monthlySeries(
  sales: SaleRow[],
  expenses: Expense[],
  months: string[],
  unmarkedIsLiga: boolean,
): MonthRow[] {
  const rows = new Map<string, MonthRow>();
  for (const month of months) {
    rows.set(month, {
      month,
      revenueBRL: 0,
      ligaBRL: 0,
      foraBRL: 0,
      costBRL: 0,
      grossBRL: 0,
      expensesBRL: expensesInRange(expenses, [month]),
      netBRL: 0,
    });
  }
  for (const s of sales) {
    const row = rows.get(monthOf(s.sellDate ?? ""));
    if (!row) {
      continue;
    }
    row.revenueBRL += s.valueBRL;
    row.costBRL += s.costBRL;
    row.grossBRL += s.profitBRL;
    if (sideOf(s, unmarkedIsLiga) === "liga") {
      row.ligaBRL += s.valueBRL;
    } else {
      row.foraBRL += s.valueBRL;
    }
  }
  const out = [...rows.values()];
  for (const row of out) {
    row.netBRL = row.grossBRL - row.expensesBRL;
  }
  return out.sort((a, b) => a.month.localeCompare(b.month));
}
