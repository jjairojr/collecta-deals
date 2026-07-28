"use client";

import Link from "next/link";
import { sealedItem, trackSelectItem } from "@/lib/analytics";
import { brl } from "@/lib/format";
import ArtPlaceholder from "@/components/ui/ArtPlaceholder";
import type { Accessory, Sealed } from "@/lib/types";

// Grid item for a boxy product (sealed or accessory). Light-pink sticker with a
// blue box-art block. Links to the product page for its kind.
export default function SealedCard({ item }: { item: Sealed | Accessory }) {
  const sealed = item.kind === "sealed";
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
        <span className="absolute bottom-2 right-2 bg-outline px-1.5 py-1 font-pixel text-[8px] text-white">
          {item.qty} UN
        </span>
      </div>

      <div className="flex flex-1 flex-col gap-1.5 p-4">
        <h3 className="font-display line-clamp-3 text-base font-bold leading-tight text-outline sm:text-xl">
          {item.name}
        </h3>
        <p className="text-xs text-on-soft">{item.meta}</p>
        <span className="mt-auto pt-1 font-pixel text-[10px] leading-relaxed text-royal sm:text-[12px]">
          {brl(item.price)}
        </span>
      </div>
    </Link>
  );
}
