"use client";

import Link from "next/link";
import { useCart } from "@/lib/cart";
import { brl } from "@/lib/format";
import { gameArtHue } from "@/lib/games";
import ArtPlaceholder from "@/components/ui/ArtPlaceholder";
import type { Single } from "@/lib/types";

export function singleToCartLine(item: Single) {
  return {
    slug: item.slug,
    kind: "single" as const,
    name: item.name,
    meta: `${item.set} · ${item.number} · ${item.condition}`,
    seller: "Collecta Oficial",
    price: item.price,
    stock: item.qty,
    imageURL: item.imageURL,
  };
}

// Grid item used across Home (featured) and Browse. The whole card links to the
// product page; the + ADD chip adds to the cart without navigating.
export default function SingleCard({
  item,
  showLanguage = false,
}: {
  item: Single;
  showLanguage?: boolean;
}) {
  const { add } = useCart();

  const onAdd = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    add(singleToCartLine(item));
  };

  return (
    <Link
      href={`/carta/${item.slug}`}
      className="arcade-press sticker animate-rise flex flex-col overflow-hidden rounded-[14px] bg-surface"
    >
      <div className="relative aspect-[5/7] border-b-4 border-outline">
        <ArtPlaceholder
          hue={gameArtHue(item.game)}
          angle="90"
          label="FOTO DA CARTA"
          imageURL={item.imageURL}
          alt={item.name}
        />
        {item.grade && (
          <span className="absolute left-2 top-2 bg-outline px-1.5 py-1 font-pixel text-[8px] text-brand-soft">
            {item.grade}
          </span>
        )}
        {showLanguage && item.language && (
          <span className="absolute right-2 top-2 bg-white px-1.5 py-1 font-pixel text-[8px] text-outline">
            {item.language}
          </span>
        )}
        <span className="absolute bottom-2 right-2 bg-outline px-1.5 py-1 font-pixel text-[8px] text-white">
          {item.qty} UN
        </span>
      </div>

      <div className="flex flex-1 flex-col gap-2 p-3.5">
        <div className="min-w-0">
          <h3 className="font-display truncate text-[19px] font-bold leading-tight text-white">
            {item.name}
          </h3>
          <p className="truncate text-xs text-muted">
            {item.set} · {item.number}
          </p>
        </div>

        <div className="mt-auto flex items-center justify-between gap-2 pt-1">
          <span className="font-pixel text-[12px] text-brand">
            {brl(item.price)}
          </span>
          <button
            type="button"
            onClick={onAdd}
            aria-label={`Adicionar ${item.name} ao carrinho`}
            className="arcade-press sticker flex items-center gap-1 rounded-[8px] bg-royal px-2.5 py-1.5 font-pixel text-[9px] text-white"
            style={{ ["--sh" as string]: "3px" }}
          >
            + ADD
          </button>
        </div>
      </div>
    </Link>
  );
}
