import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from "react";
import { brl0 } from "./format";

export interface PickedCard {
  set: string;
  number: string;
  name: string;
  priceBRL?: number;
}

// OFFER_FACTOR is the fraction of the current lowest price we propose to pay
// (~20% below), stated as negotiable in the copy-paste message.
const OFFER_FACTOR = 0.8;

interface SelectionValue {
  items: PickedCard[];
  count: number;
  has: (set: string, number: string) => boolean;
  toggle: (card: PickedCard) => void;
  backfill: (cards: PickedCard[]) => void;
  clear: () => void;
}

const SelectionContext = createContext<SelectionValue | null>(null);
const STORAGE_KEY = "opdeals.selection.v1";

// buildBuyMessage produces the BR-Portuguese copy-paste offer message: it lists
// each selected card by name + number with a proposed price ~20% below the
// current lowest (stated as negotiable), and a total offer.
export function buildBuyMessage(items: PickedCard[]): string {
  const lines = ["🏴‍☠️ COMPRO CARTAS ONE PIECE TCG 🏴‍☠️", "💸 Pago à vista no PIX, na hora!"];
  if (items.length === 0) {
    lines.push("Tenho interesse nas cartas da imagem (NM / Inglês).");
  } else {
    lines.push("", "Tenho interesse nas seguintes cartas (NM / Inglês):");
    let total = 0;
    for (const c of items) {
      const label = c.name.includes(c.number) ? c.name : `${c.name} (${c.number})`;
      if (c.priceBRL && c.priceBRL > 0) {
        const offer = Math.round(c.priceBRL * OFFER_FACTOR);
        total += offer;
        lines.push(`• ${label} — ${brl0(offer)}`);
      } else {
        lines.push(`• ${label}`);
      }
    }
    if (total > 0) {
      lines.push("", `💰 Minha oferta: ${brl0(total)} (valor negociável, podemos combinar!)`);
    }
  }
  lines.push(
    "Compro unidade, lote ou coleção inteira.",
    "📩 Chama no DM ou responde aqui — fecho rápido e pago bem! 🔥",
  );
  return lines.join("\n");
}

function isPicked(x: unknown): x is PickedCard {
  if (typeof x !== "object" || x === null) {
    return false;
  }
  const o = x as Record<string, unknown>;
  return typeof o.set === "string" && typeof o.number === "string" && typeof o.name === "string";
}

function load(): PickedCard[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) {
      return [];
    }
    const parsed: unknown = JSON.parse(raw);
    return Array.isArray(parsed) ? parsed.filter(isPicked) : [];
  } catch {
    return [];
  }
}

export function SelectionProvider({ children }: { children: ReactNode }) {
  const [items, setItems] = useState<PickedCard[]>(load);

  const persist = useCallback((next: PickedCard[]) => {
    setItems(next);
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(next));
    } catch {
      // ignore quota / disabled storage
    }
  }, []);

  const has = useCallback(
    (set: string, number: string) => items.some((i) => i.set === set && i.number === number),
    [items],
  );

  const toggle = useCallback(
    (card: PickedCard) => {
      const exists = items.some((i) => i.set === card.set && i.number === card.number);
      persist(
        exists
          ? items.filter((i) => !(i.set === card.set && i.number === card.number))
          : [...items, card],
      );
    },
    [items, persist],
  );

  // backfill fills in (or refreshes) the current price on already-selected
  // cards as the pages that know their prices render them. This heals entries
  // picked before a price was tracked, without disturbing the selection itself.
  const backfill = useCallback((cards: PickedCard[]) => {
    setItems((prev) => {
      const byKey = new Map<string, number>();
      for (const c of cards) {
        if (c.priceBRL && c.priceBRL > 0) {
          byKey.set(`${c.set}|${c.number}`, c.priceBRL);
        }
      }
      let changed = false;
      const next = prev.map((it) => {
        const price = byKey.get(`${it.set}|${it.number}`);
        if (price && it.priceBRL !== price) {
          changed = true;
          return { ...it, priceBRL: price };
        }
        return it;
      });
      if (!changed) {
        return prev;
      }
      try {
        localStorage.setItem(STORAGE_KEY, JSON.stringify(next));
      } catch {
        // ignore quota / disabled storage
      }
      return next;
    });
  }, []);

  const clear = useCallback(() => persist([]), [persist]);

  const value = useMemo<SelectionValue>(
    () => ({ items, count: items.length, has, toggle, backfill, clear }),
    [items, has, toggle, backfill, clear],
  );

  return <SelectionContext.Provider value={value}>{children}</SelectionContext.Provider>;
}

export function useSelection(): SelectionValue {
  const ctx = useContext(SelectionContext);
  if (!ctx) {
    throw new Error("useSelection must be used within a SelectionProvider");
  }
  return ctx;
}
