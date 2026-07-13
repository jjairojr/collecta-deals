import { type ReactElement } from "react";
import { ResponsiveContainer } from "recharts";
import { cn } from "../../lib/utils";

export const chartColors = {
  sky: "#38bdf8",
  emerald: "#34d399",
  rose: "#fb7185",
  amber: "#fbbf24",
  slate: "#64748b",
  grid: "#1e293b",
  axis: "#64748b",
};

export const axisTick = { fill: "#94a3b8", fontSize: 11 };

export const tooltipCursor = { fill: "rgba(148, 163, 184, 0.08)" };

interface ChartTooltipRow {
  label: string;
  value: string;
  color?: string;
}

export function ChartTooltip({ title, rows }: { title: string; rows: ChartTooltipRow[] }) {
  return (
    <div className="rounded-lg border border-slate-700 bg-slate-900/95 px-3 py-2 text-xs shadow-lg backdrop-blur">
      <div className="mb-1 font-medium text-slate-200">{title}</div>
      {rows.map((r) => (
        <div key={r.label} className="flex items-center justify-between gap-4 tabular-nums">
          <span className="text-slate-400">{r.label}</span>
          <span className="font-semibold" style={{ color: r.color ?? "#e2e8f0" }}>
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
