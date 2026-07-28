import SealedCard from "@/components/SealedCard";
import TrackViewItemList from "@/components/analytics/TrackViewItemList";
import type { Accessory, Sealed } from "@/lib/types";

export default function SealedGrid({
  list,
  listId = "selado",
  listName = "Produto selado",
  empty = "NENHUM SELADO DISPONIVEL NO MOMENTO",
}: {
  list: (Sealed | Accessory)[];
  listId?: string;
  listName?: string;
  empty?: string;
}) {
  if (list.length === 0) {
    return (
      <p className="font-pixel text-[10px] leading-relaxed text-faint">
        {empty}
      </p>
    );
  }
  return (
    <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4 lg:gap-5">
      <TrackViewItemList items={list} listId={listId} listName={listName} />
      {list.map((s) => (
        <SealedCard key={s.slug} item={s} />
      ))}
    </div>
  );
}
