import SealedCard from "@/components/SealedCard";
import TrackViewItemList from "@/components/analytics/TrackViewItemList";
import type { Sealed } from "@/lib/types";

export default function SealedGrid({
  list,
  listId = "selado",
  listName = "Produto selado",
}: {
  list: Sealed[];
  listId?: string;
  listName?: string;
}) {
  if (list.length === 0) {
    return (
      <p className="font-pixel text-[10px] leading-relaxed text-faint">
        NENHUM SELADO DISPONIVEL NO MOMENTO
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
