import type { CardSale, CardSeller, LangSale, SnapshotSales } from "./api";

function cardKey(c: CardSale): string {
  return `${c.set ?? ""}|${c.number}`;
}

export function mergeSnapshotCards(snapshots: SnapshotSales[]): CardSale[] {
  const byCard = new Map<string, CardSale>();
  const sellersByCard = new Map<string, Map<number, CardSeller>>();
  const langsByCard = new Map<string, Map<string, LangSale>>();
  for (const snap of snapshots) {
    for (const c of snap.cards ?? []) {
      const key = cardKey(c);
      const existing = byCard.get(key);
      if (existing) {
        existing.units += c.units;
        existing.revenueBRL += c.revenueBRL;
      } else {
        byCard.set(key, { ...c, sellers: undefined, languages: undefined });
        sellersByCard.set(key, new Map());
        langsByCard.set(key, new Map());
      }
      const sellers = sellersByCard.get(key);
      if (sellers) {
        for (const s of c.sellers ?? []) {
          const se = sellers.get(s.storeId);
          if (se) {
            se.units += s.units;
            se.revenueBRL += s.revenueBRL;
          } else {
            sellers.set(s.storeId, { ...s });
          }
        }
      }
      const langs = langsByCard.get(key);
      if (langs) {
        for (const l of c.languages ?? []) {
          const acc = langs.get(l.code);
          if (acc) {
            acc.units += l.units;
            acc.revenueBRL += l.revenueBRL;
          } else {
            langs.set(l.code, { ...l });
          }
        }
      }
    }
  }
  const out: CardSale[] = [];
  for (const [key, card] of byCard) {
    const sellers = [...(sellersByCard.get(key)?.values() ?? [])].sort(
      (a, b) => b.units - a.units || a.storeName.localeCompare(b.storeName),
    );
    const languages = [...(langsByCard.get(key)?.values() ?? [])].sort(
      (a, b) => b.units - a.units || a.code.localeCompare(b.code),
    );
    out.push({ ...card, sellers, languages });
  }
  out.sort(
    (a, b) => b.revenueBRL - a.revenueBRL || b.units - a.units || a.number.localeCompare(b.number),
  );
  return out;
}

export interface SalesTotals {
  units: number;
  revenueBRL: number;
  storesSelling: number;
  cards: number;
}

export function salesTotals(snapshots: SnapshotSales[]): SalesTotals {
  const stores = new Set<number>();
  const cards = new Set<string>();
  let units = 0;
  let revenueBRL = 0;
  for (const snap of snapshots) {
    units += snap.units;
    revenueBRL += snap.revenueBRL;
    for (const c of snap.cards ?? []) {
      cards.add(cardKey(c));
      for (const s of c.sellers ?? []) {
        stores.add(s.storeId);
      }
    }
  }
  return { units, revenueBRL, storesSelling: stores.size, cards: cards.size };
}
