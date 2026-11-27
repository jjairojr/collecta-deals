"use client";

import Link from "next/link";
import { sealedItem, trackSelectItem } from "@/lib/analytics";
import { brl } from "@/lib/format";
import { CARD_PRICED_INSTALLMENTS, shipsFromLabel } from "@/lib/payment";
import ArtPlaceholder from "@/components/ui/ArtPlaceholder";
import { stockLabel } from "@/lib/stock";
import type { Accessory, Sealed } from "@/lib/types";

// Grid item for a boxy product (sealed or accessory). Light-pink sticker with a
// blue box-art block. Links to the product page for its kind.
export default function SealedCard({ item }: { item: Sealed | Accessory }) {
  const sealed = item.kind === "sealed";
  const kits = item.kits ?? [];
  const bestUnit = kits.reduce((min, k) => Math.min(min, Math.round(k.price / k.units)), item.price);
  const bestSaving = item.price > 0 ? Math.round(((item.price - bestUnit) / item.price) * 100) : 0;
  return (
    <Link
      href={`${sealed ? "/selado" : "/acessorio"}/${item.slug}`}
      onClick={() => trackSelectItem(sealedItem(item))}
      className="arcade-press sticker animate-rise flex flex-col overflow-hidden rounded-[14px] bg-brand-soft"
    >
      <div className="relative h-[196px] border-b-4 border-outline">
        <ArtPlaceholder
          hue="royal"
          angle="45"
          label={sealed ? "FOTO DA CAIXA" : "FOTO DO PRODUTO"}
          imageURL={item.imageURL}
          alt={item.name}
        />
        {item.badge && (
          <span className="absolute left-2 top-2 bg-brand px-1.5 py-1 font-pixel text-[8px] text-white">
            {item.badge}
          </span>
        )}
        {!item.shipsFrom && (
          <span className="absolute bottom-2 right-2 bg-outline px-1.5 py-1 font-pixel text-[8px] text-white">
            {stockLabel(item.qty)}
          </span>
        )}
      </div>

      <div className="flex flex-1 flex-col gap-1.5 p-4">
        <h3 className="font-display line-clamp-3 text-base font-bold leading-tight text-outline sm:text-xl">
          {item.name}
        </h3>
        <p className="text-xs text-on-soft">{item.meta}</p>
        {item.shipsFrom && (
          <p className="text-xs font-bold text-royal">Envio a partir de {shipsFromLabel(item.shipsFrom)}</p>
        )}
        <span className="mt-auto pt-1 font-pixel text-[10px] leading-relaxed text-royal sm:text-[12px]">
          {brl(item.price)}
          {item.cardPrice !== undefined && <span className="ml-1.5 text-[8px] sm:text-[9px]">NO PIX</span>}
        </span>
        {item.cardPrice !== undefined && (
          <span className="text-xs text-on-soft">
            ou {brl(item.cardPrice)} em até {CARD_PRICED_INSTALLMENTS}x sem juros no cartão
          </span>
        )}
        {bestSaving > 0 && (
          <span className="mt-2 flex items-center gap-2 rounded-[8px] border-[3px] border-outline bg-brand px-2 py-1.5 text-white">
            <span className="shrink-0 font-pixel text-[8px] sm:text-[9px]">KIT -{bestSaving}%</span>
            <span className="text-xs font-bold leading-tight">
              a partir de {brl(bestUnit)} cada
            </span>
          </span>
        )}
      </div>
    </Link>
  );
}
