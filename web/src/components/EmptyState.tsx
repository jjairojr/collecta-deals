import type { ReactNode } from "react";
import mascot from "../assets/mascot.png";
import { cn } from "../lib/utils";

export default function EmptyState({
  children,
  hint,
  bare,
  className,
}: {
  children: ReactNode;
  hint?: string;
  bare?: boolean;
  className?: string;
}) {
  return (
    <div
      className={cn(
        "flex flex-col items-center gap-3 px-4 py-10 text-center",
        bare ? "bg-panel" : "sticker sticker-sm rounded-[12px] bg-panel",
        className,
      )}
    >
      <img
        src={mascot}
        alt=""
        className="h-16 w-16 shrink-0 rounded-full border-[3px] border-outline object-cover"
      />
      <p className="max-w-md text-sm text-slate-400">{children}</p>
      {hint && <p className="font-pixel text-[8px] uppercase text-brand-label">{hint}</p>}
    </div>
  );
}
