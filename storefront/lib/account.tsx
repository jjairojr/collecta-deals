"use client";

import { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";

export interface AccountUser {
  id: string;
  username: string;
  displayName: string;
  email: string | null;
  whatsapp: string | null;
}

type Status = "loading" | "anon" | "ready";

interface AccountContext {
  status: Status;
  user: AccountUser | null;
  refresh: () => Promise<void>;
}

const Ctx = createContext<AccountContext | null>(null);

export function AccountProvider({ children }: { children: React.ReactNode }) {
  const [status, setStatus] = useState<Status>("loading");
  const [user, setUser] = useState<AccountUser | null>(null);

  const refresh = useCallback(async () => {
    try {
      const res = await fetch("/api/conta/me", { cache: "no-store" });
      const data = (await res.json()) as { user: AccountUser | null };
      setUser(data.user);
      setStatus(data.user ? "ready" : "anon");
    } catch {
      setUser(null);
      setStatus("anon");
    }
  }, []);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const value = useMemo(() => ({ status, user, refresh }), [status, user, refresh]);
  return <Ctx.Provider value={value}>{children}</Ctx.Provider>;
}

export function useAccount(): AccountContext {
  const ctx = useContext(Ctx);
  if (!ctx) {
    throw new Error("useAccount outside AccountProvider");
  }
  return ctx;
}
