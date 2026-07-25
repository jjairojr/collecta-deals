import Link from "next/link";
import { brl } from "@/lib/format";
import ArtPlaceholder from "@/components/ui/ArtPlaceholder";
import type { Sealed } from "@/lib/types";

// Grid item for sealed product. Light-pink sticker with a blue box-art block.
// Links to the sealed product page.
export default function SealedCard({ item }: { item: Sealed }) {
  return (
    <Link
      href={`/selado/${item.slug}`}
      className="arcade-press sticker animate-rise flex flex-col overflow-hidden rounded-[14px] bg-brand-soft"
    >
      <div className="relative h-[196px] border-b-4 border-outline">
        <ArtPlaceholder
          hue="royal"
          angle="45"
          label="FOTO DA CAIXA"
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
        <span className="mt-auto pt-1 font-pixel text-[11px] text-royal sm:text-[12px]">
          {brl(item.price)}
        </span>
      </div>
    </Link>
  );
}
