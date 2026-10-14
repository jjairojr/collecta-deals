"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { useAccount } from "@/lib/account";
import { trackLogout } from "@/lib/analytics";

export default function LogoutButton() {
  const router = useRouter();
  const { refresh } = useAccount();
  const [busy, setBusy] = useState(false);

  const logout = async () => {
    setBusy(true);
    try {
      await fetch("/api/conta/sair", { method: "POST" });
      trackLogout();
    } finally {
      await refresh();
      router.push("/");
      router.refresh();
    }
  };

  return (
    <button
      type="button"
      onClick={() => void logout()}
      disabled={busy}
      className="arcade-press sticker rounded-[10px] bg-outline px-4 py-2.5 font-pixel text-[9px] text-white [--sh:4px] disabled:opacity-60"
    >
      {busy ? "SAINDO..." : "SAIR"}
    </button>
  );
}
