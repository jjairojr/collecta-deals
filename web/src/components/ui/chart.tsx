import { type ReactElement } from "react";
import { ResponsiveContainer } from "recharts";
import { cn } from "../../lib/utils";

export const chartColors = {
  brand: "#f6559b",
  emerald: "#34d399",
  rose: "#fb7185",
  grid: "#2a2a2e",
  axis: "#9a9aa2",
};

export const axisTick = { fill: "#9a9aa2", fontSize: 11 };

export const tooltipCursor = { fill: "rgba(246, 85, 155, 0.14)" };

interface ChartTooltipRow {
  label: string;
  value: string;
  color?: string;
}

export function ChartTooltip({ title, rows }: { title: string; rows: ChartTooltipRow[] }) {
  return (
    <div className="win sticker-sm min-w-[160px] text-xs">
      <div className="win-bar !gap-2 !px-2.5 !py-1.5">
        <span className="win-title !text-[11px]">{title}</span>
      </div>
      <div className="flex flex-col gap-0.5 px-3 py-2">
        {rows.map((r) => (
          <div key={r.label} className="flex items-center justify-between gap-4 tabular-nums">
            <span className="text-slate-400">{r.label}</span>
            <span className="font-semibold" style={{ color: r.color ?? "#ffffff" }}>
              {r.value}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}

export function ChartContainer({
  children,
  height = 280,
  className,
}: {
  children: ReactElement;
  height?: number;
  className?: string;
}) {
  return (
    <div className={cn("w-full", className)} style={{ height }}>
      <ResponsiveContainer width="100%" height="100%">
        {children}
      </ResponsiveContainer>
    </div>
  );
}
