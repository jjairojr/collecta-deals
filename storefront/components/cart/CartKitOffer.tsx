"use client";

import { useEffect, useRef } from "react";
import { cartLineItem, trackAddToCart, trackViewItemList } from "@/lib/analytics";
import { useCart } from "@/lib/cart";
import { kitLabel, kitToCartLine } from "@/lib/cart-lines";
import { brl } from "@/lib/format";
import type { BoosterKitOffer } from "@/lib/offers";

const OFFER_LIST = "carrinho_kit_30anos";

// Cart suggestion of the 30th anniversary booster kit (picked from real stock by
// the page). Hidden once the kit is in the cart.
export default function CartKitOffer({ offer }: { offer?: BoosterKitOffer }) {
  const { lines, add } = useCart();
  const seen = useRef(false);
  const inCart = offer ? lines.some((l) => l.sku === offer.kit.sku) : true;
  const show = offer !== undefined && !inCart && lines.length > 0;

  useEffect(() => {
    if (show && offer && !seen.current) {
      seen.current = true;
      trackViewItemList([cartLineItem({ ...kitToCartLine(offer.item, offer.kit, offer.item.qty), qty: 1 })], OFFER_LIST, OFFER_LIST);
    }
  }, [show, offer]);

  if (!show || !offer) {
    return null;
  }
  const { item, kit } = offer;
  const each = Math.round(kit.price / kit.units);

  const onAdd = () => {
    const line = kitToCartLine(item, kit, item.qty);
    add(line);
    trackAddToCart(cartLineItem({ ...line, qty: 1 }), OFFER_LIST);
  };

  return (
    <div
      className="sticker flex items-center gap-4 rounded-[14px] bg-brand-soft p-4"
      style={{ ["--sh" as string]: "5px" }}
    >
      <div className="min-w-0 flex-1">
        <div className="font-pixel text-[8px] text-royal">LEVE MAIS, PAGUE MENOS</div>
        <div className="font-display mt-1 truncate text-lg font-bold leading-tight text-outline">
          {item.name} · {kitLabel(kit.units)}
        </div>
        <div className="text-sm text-on-soft">
          {brl(kit.price)} no Pix · {brl(each)} cada
          {item.price > each ? ` (avulso ${brl(item.price)})` : ""}
        </div>
      </div>
      <button
        type="button"
        onClick={onAdd}
        aria-label={`Adicionar ${item.name} ${kitLabel(kit.units)} ao carrinho`}
        className="arcade-press sticker shrink-0 rounded-[8px] bg-royal px-3 py-2 font-pixel text-[9px] text-white [--sh:3px]"
      >
        + ADD
      </button>
    </div>
  );
}
