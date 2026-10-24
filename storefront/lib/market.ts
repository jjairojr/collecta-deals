import "server-only";
import type { Single } from "@/lib/types";

export interface MarketRow {
  language: string;
  lowBRL: number;
  medianBRL: number;
  date: string;
}

const LANGS: { code: string; label: string; marker: RegExp }[] = [
  { code: "8", label: "Português", marker: /\bPT\b|portugu/i },
  { code: "2", label: "Inglês", marker: /\bEN\b|ingl/i },
  { code: "6", label: "Japonês", marker: /\bJP\b|japon/i },
];

function languagesFor(item: Single): { code: string; label: string }[] {
  const text = `${item.variant ?? ""} ${item.name}`;
  const marked = LANGS.filter((l) => l.marker.test(text));
  if (marked.length === 1) {
    return marked;
  }
  if (marked.length > 1) {
    return [];
  }
  return item.game === "pokemon" ? LANGS.slice(0, 2) : LANGS.slice(1, 2);
}

interface RawMarket {
  lowBRL?: number;
  medianBRL?: number;
  date?: string;
}

export async function loadMarket(item: Single): Promise<MarketRow[]> {
  const base = process.env.STOREFRONT_API;
  if (!base || !item.number || !item.set) {
    return [];
  }
  const rows = await Promise.all(
    languagesFor(item).map(async (lang) => {
      const p = new URLSearchParams({ game: item.game, set: item.set, number: item.number, lang: lang.code });
      try {
        const res = await fetch(`${base}/api/storefront/market?${p.toString()}`, { next: { revalidate: 3600 } });
        if (!res.ok) {
          return null;
        }
        const m = (await res.json()) as RawMarket;
        if (!m.lowBRL || !m.medianBRL || !m.date) {
          return null;
        }
        return { language: lang.label, lowBRL: m.lowBRL, medianBRL: m.medianBRL, date: m.date };
      } catch {
        return null;
      }
    }),
  );
  return rows.filter((r): r is MarketRow => r !== null);
}
