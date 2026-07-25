import SealedCard from "@/components/SealedCard";
import type { Sealed } from "@/lib/types";

export default function SealedGrid({ list }: { list: Sealed[] }) {
  if (list.length === 0) {
    return (
      <p className="font-pixel text-[10px] leading-relaxed text-faint">
        NENHUM SELADO DISPONIVEL NO MOMENTO
      </p>
    );
  }
  return (
    <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4 lg:gap-5">
      {list.map((s) => (
        <SealedCard key={s.slug} item={s} />
      ))}
    </div>
  );
}
