"use client";

import Link from "next/link";
import { useEffect, useRef, useState } from "react";
import { sealedItem, trackAddToCart, trackViewItemList } from "@/lib/analytics";
import { useCart } from "@/lib/cart";
import { sealedToCartLine } from "@/lib/cart-lines";
import { brl } from "@/lib/format";
import type { Accessory } from "@/lib/types";

const OFFER_LIST = "oferta_toast_acessorio";

// "Added to cart" toast. After a single or a graded slab it also offers one card
// accessory (sleeve or toploader, picked from real stock by the layout) with a
// one-tap add. Without an offer it closes after 2.6 s; with one it stays longer
// and pauses while the pointer is over it.
export default function CartToast({ accessory }: { accessory?: Accessory }) {
  const { lastAdded, lines, add } = useCart();
  const [visible, setVisible] = useState(false);
  const [hover, setHover] = useState(false);
  const seen = useRef<number | null>(null);

  const offer =
    accessory &&
    lastAdded?.kind === "single" &&
    lastAdded.sku !== accessory.sku &&
    !lines.some((l) => l.sku === accessory.sku)
      ? accessory
      : undefined;

  useEffect(() => {
    if (!lastAdded) {
      return;
    }
    setVisible(true);
  }, [lastAdded]);

  useEffect(() => {
    if (!visible || hover) {
      return;
    }
    const t = setTimeout(() => setVisible(false), offer ? 7000 : 2600);
    return () => clearTimeout(t);
  }, [visible, hover, offer, lastAdded]);

  useEffect(() => {
    if (visible && offer && lastAdded && seen.current !== lastAdded.nonce) {
      seen.current = lastAdded.nonce;
      trackViewItemList([sealedItem(offer)], OFFER_LIST, OFFER_LIST);
    }
  }, [visible, offer, lastAdded]);

  if (!lastAdded || !visible) {
    return null;
  }

  const addOffer = () => {
    if (!offer) {
      return;
    }
    add(sealedToCartLine(offer, offer.qty));
    trackAddToCart(sealedItem(offer), OFFER_LIST);
  };

  return (
    <div
      className="fixed inset-x-4 bottom-4 z-50 sm:inset-x-auto sm:right-6 sm:w-[360px]"
      onMouseEnter={() => setHover(true)}
      onMouseLeave={() => setHover(false)}
    >
      <div role="status" className="sticker animate-rise rounded-[12px] bg-outline p-3.5 [--sh:5px]">
        <div className="flex items-center gap-3">
          <span className="grid h-9 w-9 shrink-0 place-items-center rounded-[8px] border-2 border-outline bg-brand-soft font-pixel text-[8px] text-outline">
            OK
          </span>
          <div className="min-w-0 flex-1">
            <div className="font-pixel text-[8px] text-brand-soft">NO CARRINHO</div>
            <div className="truncate text-sm font-bold text-white">{lastAdded.name}</div>
          </div>
          <Link
            href="/carrinho"
            onClick={() => setVisible(false)}
            className="arcade-press sticker shrink-0 rounded-[8px] bg-brand px-3 py-2 font-pixel text-[9px] text-white [--sh:3px]"
          >
            VER
          </Link>
        </div>
        {offer && (
          <div className="mt-3 flex items-center gap-3 border-t-2 border-surface pt-3">
            <div className="min-w-0 flex-1">
              <div className="font-pixel text-[8px] text-brand-soft">PROTEJA A CARTA</div>
              <div className="truncate text-sm text-white">{offer.name}</div>
              <div className="font-pixel text-[9px] text-brand">{brl(offer.price)}</div>
            </div>
            <button
              type="button"
              onClick={addOffer}
              aria-label={`Adicionar ${offer.name} ao carrinho`}
              className="arcade-press sticker shrink-0 rounded-[8px] bg-royal px-3 py-2 font-pixel text-[9px] text-white [--sh:3px]"
            >
              + ADD
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
