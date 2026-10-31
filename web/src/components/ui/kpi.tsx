import type { ReactNode } from "react";

function Kpi({ icon, label, value, sub }: { icon: ReactNode; label: string; value: string; sub?: string }) {
  return (
    <div className="sticker flex items-center gap-3 rounded-[14px] bg-panel p-4">
      <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-[8px] border-2 border-outline bg-brand text-white">
        {icon}
      </div>
      <div className="min-w-0">
        <div className="font-pixel text-[8px] uppercase leading-[1.5] text-brand-label">{label}</div>
        <div className="mt-1 truncate text-lg font-bold tabular-nums text-fg xl:text-xl">{value}</div>
        {sub && <div className="truncate text-[11px] text-slate-500">{sub}</div>}
      </div>
    </div>
  );
}

export { Kpi };
