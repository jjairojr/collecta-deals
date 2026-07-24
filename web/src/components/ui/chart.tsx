import { type ReactElement } from "react";
import { ResponsiveContainer } from "recharts";
import { cn } from "../../lib/utils";

export const chartColors = {
  sky: "#f6559b",
  emerald: "#34d399",
  rose: "#fb7185",
  amber: "#fbbf24",
  slate: "#6f6f77",
  grid: "#2a2a2e",
  axis: "#9a9aa2",
};

export const axisTick = { fill: "#9a9aa2", fontSize: 11 };

export const tooltipCursor = { fill: "rgba(246, 85, 155, 0.10)" };

interface ChartTooltipRow {
  label: string;
  value: string;
  color?: string;
}

export function ChartTooltip({ title, rows }: { title: string; rows: ChartTooltipRow[] }) {
  return (
    <div className="rounded-[8px] border-2 border-outline bg-surface px-3 py-2 text-xs shadow-[3px_3px_0_#0b0b0c]">
      <div className="font-display mb-1 font-bold text-white">{title}</div>
      {rows.map((r) => (
        <div key={r.label} className="flex items-center justify-between gap-4 tabular-nums">
          <span className="text-slate-400">{r.label}</span>
          <span className="font-semibold" style={{ color: r.color ?? "#ffffff" }}>
            {r.value}
          </span>
        </div>
      ))}
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
