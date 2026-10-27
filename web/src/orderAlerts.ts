import { useCallback, useEffect, useMemo, useState } from "react";
import { listOrders, type Order } from "./api";

const SEEN_KEY = "opdeals.orders.seenAt";
const POLL_MS = 60_000;

function readSeenAt(): number {
  try {
    const value = Number(window.localStorage.getItem(SEEN_KEY));
    return Number.isFinite(value) ? value : 0;
  } catch {
    return 0;
  }
}

function writeSeenAt(value: number): void {
  try {
    window.localStorage.setItem(SEEN_KEY, String(value));
  } catch {
    void 0;
  }
}

export function isNewOrder(order: Order, since: number): boolean {
  if (order.status !== "paid") {
    return false;
  }
  const paid = Date.parse(order.paidAt);
  return Number.isFinite(paid) && paid > since;
}

export interface OrderAlerts {
  toShip: number;
  fresh: number;
  since: number;
  refresh: () => void;
}

export function useOrderAlerts(watching: boolean): OrderAlerts {
  const [orders, setOrders] = useState<Order[]>([]);
  const [seenAt, setSeenAt] = useState(readSeenAt);
  const [since, setSince] = useState(readSeenAt);

  const refresh = useCallback(async () => {
    try {
      setOrders(await listOrders());
    } catch {
      void 0;
    }
  }, []);

  useEffect(() => {
    refresh();
    const id = window.setInterval(refresh, POLL_MS);
    const onFocus = () => refresh();
    window.addEventListener("focus", onFocus);
    return () => {
      window.clearInterval(id);
      window.removeEventListener("focus", onFocus);
    };
  }, [refresh]);

  useEffect(() => {
    if (watching) {
      setSince(readSeenAt());
    }
  }, [watching]);

  useEffect(() => {
    if (!watching) {
      return;
    }
    const now = Date.now();
    writeSeenAt(now);
    setSeenAt(now);
  }, [watching, orders]);

  return useMemo(
    () => ({
      toShip: orders.filter((o) => o.status === "paid").length,
      fresh: orders.filter((o) => isNewOrder(o, seenAt)).length,
      since,
      refresh,
    }),
    [orders, seenAt, since, refresh],
  );
}
