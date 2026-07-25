"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { useCart } from "@/lib/cart";

export default function CartToast() {
  const { lastAdded } = useCart();
  const [visible, setVisible] = useState(false);

  useEffect(() => {
    if (!lastAdded) {
      return;
    }
    setVisible(true);
    const t = setTimeout(() => setVisible(false), 2600);
    return () => clearTimeout(t);
  }, [lastAdded]);

  if (!lastAdded || !visible) {
    return null;
  }

  return (
    <div className="fixed inset-x-4 bottom-4 z-50 sm:inset-x-auto sm:right-6 sm:w-[360px]">
      <div
        role="status"
        className="sticker animate-rise flex items-center gap-3 rounded-[12px] bg-outline p-3.5 [--sh:5px]"
      >
        <span className="grid h-9 w-9 shrink-0 place-items-center rounded-[8px] border-2 border-outline bg-brand-soft font-pixel text-[8px] text-outline">
          OK
        </span>
        <div className="min-w-0 flex-1">
          <div className="font-pixel text-[8px] text-brand-soft">
            NO CARRINHO
          </div>
          <div className="truncate text-sm font-bold text-white">
            {lastAdded.name}
          </div>
        </div>
        <Link
          href="/carrinho"
          onClick={() => setVisible(false)}
          className="arcade-press sticker shrink-0 rounded-[8px] bg-brand px-3 py-2 font-pixel text-[9px] text-white [--sh:3px]"
        >
          VER
        </Link>
      </div>
    </div>
  );
}
