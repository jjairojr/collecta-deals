"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import type { CartLine } from "@/lib/types";

const STORAGE_KEY = "collecta.cart.v1";

// A line is identified by product + seller + condition, so the same card from
// two sellers stays as two lines.
function lineKey(l: Pick<CartLine, "slug" | "seller" | "meta">): string {
  return `${l.slug}|${l.seller}|${l.meta}`;
}

interface CartContext {
  lines: CartLine[];
  count: number;
  total: number;
  add: (line: Omit<CartLine, "qty">, qty?: number) => void;
  setQty: (line: CartLine, qty: number) => void;
  remove: (line: CartLine) => void;
  clear: () => void;
}

const Ctx = createContext<CartContext | null>(null);

export function CartProvider({ children }: { children: React.ReactNode }) {
  const [lines, setLines] = useState<CartLine[]>([]);
  const hydrated = useRef(false);

  useEffect(() => {
    try {
      const raw = localStorage.getItem(STORAGE_KEY);
      if (raw) {
        setLines(JSON.parse(raw) as CartLine[]);
      }
    } catch {
      // ignore corrupt storage
    }
    hydrated.current = true;
  }, []);

  useEffect(() => {
    if (!hydrated.current) {
      return;
    }
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(lines));
    } catch {
      // ignore quota / private-mode failures
    }
  }, [lines]);

  const add = useCallback((line: Omit<CartLine, "qty">, qty = 1) => {
    setLines((prev) => {
      const key = lineKey(line);
      const cap = line.stock ?? Infinity;
      const found = prev.find((l) => lineKey(l) === key);
      if (found) {
        return prev.map((l) =>
          lineKey(l) === key
            ? { ...l, stock: line.stock, qty: Math.min(l.qty + qty, cap) }
            : l,
        );
      }
      return [...prev, { ...line, qty: Math.min(qty, cap) }];
    });
  }, []);

  const setQty = useCallback((line: CartLine, qty: number) => {
    const key = lineKey(line);
    const capped = Math.min(qty, line.stock ?? Infinity);
    setLines((prev) =>
      capped <= 0
        ? prev.filter((l) => lineKey(l) !== key)
        : prev.map((l) => (lineKey(l) === key ? { ...l, qty: capped } : l)),
    );
  }, []);

  const remove = useCallback((line: CartLine) => {
    const key = lineKey(line);
    setLines((prev) => prev.filter((l) => lineKey(l) !== key));
  }, []);

  const clear = useCallback(() => setLines([]), []);

  const value = useMemo<CartContext>(() => {
    const count = lines.reduce((n, l) => n + l.qty, 0);
    const total = lines.reduce((s, l) => s + l.price * l.qty, 0);
    return { lines, count, total, add, setQty, remove, clear };
  }, [lines, add, setQty, remove, clear]);

  return <Ctx.Provider value={value}>{children}</Ctx.Provider>;
}

export function useCart(): CartContext {
  const ctx = useContext(Ctx);
  if (!ctx) {
    throw new Error("useCart must be used inside <CartProvider>");
  }
  return ctx;
}
