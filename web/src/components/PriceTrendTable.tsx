import { useMemo, useState } from "react";
import { Bar, BarChart, CartesianGrid, Cell, ReferenceLine, Tooltip, XAxis, YAxis } from "recharts";
import type { CardTrend } from "../api";
import { brl } from "../format";
import { Card } from "./ui/card";
import { Select } from "./ui/select";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "./ui/table";
import { ChartContainer, ChartTooltip, axisTick, chartColors, tooltipCursor } from "./ui/chart";

function deltaColor(delta: number): string {
  if (delta > 0) return "text-emerald-400";
  if (delta < 0) return "text-rose-400";
  return "text-slate-400";
}

function deltaLabel(trend: CardTrend): string {
  if (trend.prevBRL <= 0) return "—";
  const sign = trend.deltaPct > 0 ? "+" : "";
  return `${sign}${trend.deltaPct.toFixed(1)}%`;
}

type SortKey = "move" | "priceDesc" | "priceAsc";

function isSortKey(v: string): v is SortKey {
  return v === "move" || v === "priceDesc" || v === "priceAsc";
}

interface MoverDatum {
  number: string;
  name: string;
  delta: number;
  lowBRL: number;
  prevBRL: number;
}

function MoverTooltip({
  active,
  payload,
}: {
  active?: boolean;
  payload?: { payload: MoverDatum }[];
}) {
  if (!active || !payload || payload.length === 0) {
    return null;
  }
  const d = payload[0].payload;
  const color = d.delta >= 0 ? chartColors.emerald : chartColors.rose;
  return (
    <ChartTooltip
      title={`${d.number} · ${d.name}`}
      rows={[
        { label: "Change", value: `${d.delta > 0 ? "+" : ""}${d.delta.toFixed(1)}%`, color },
        { label: "Prev", value: brl(d.prevBRL) },
        { label: "Today", value: brl(d.lowBRL) },
      ]}
    />
  );
}

export default function PriceTrendTable({ trends }: { trends: CardTrend[] }) {
  const [sort, setSort] = useState<SortKey>("move");

  const rows = useMemo(
    () =>
      [...trends].sort((a, b) => {
        if (sort === "priceDesc") return b.lowBRL - a.lowBRL;
        if (sort === "priceAsc") return a.lowBRL - b.lowBRL;
        return Math.abs(b.deltaPct) - Math.abs(a.deltaPct);
      }),
    [trends, sort],
  );

  const movers = useMemo<MoverDatum[]>(() => {
    const moved = trends
      .filter((t) => t.prevBRL > 0 && t.deltaPct !== 0)
      .sort((a, b) => b.deltaPct - a.deltaPct);
    if (moved.length === 0) {
      return [];
    }
    const gainers = moved.slice(0, 6);
    const losers = moved.slice(-6).filter((l) => !gainers.includes(l));
    return [...gainers, ...losers].map((t) => ({
      number: t.number,
      name: t.name,
      delta: Number(t.deltaPct.toFixed(1)),
      lowBRL: t.lowBRL,
      prevBRL: t.prevBRL,
    }));
  }, [trends]);

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-end gap-2">
        <label className="flex items-center gap-2 text-xs text-slate-400">
          Sort
          <Select
            value={sort}
            onChange={(e) => {
              if (isSortKey(e.target.value)) {
                setSort(e.target.value);
              }
            }}
            className="w-40"
          >
            <option value="move">Biggest move</option>
            <option value="priceDesc">Price high→low</option>
            <option value="priceAsc">Price low→high</option>
          </Select>
        </label>
      </div>

      {movers.length > 0 && (
        <Card className="p-4">
          <div className="mb-3 text-xs text-slate-400">
            Top movers ·{" "}
            <span className="text-emerald-300">gainers</span> and{" "}
            <span className="text-rose-300">losers</span> by day-over-day change
          </div>
          <ChartContainer height={Math.max(220, movers.length * 30)}>
            <BarChart
              data={movers}
              layout="vertical"
              margin={{ top: 4, right: 16, bottom: 4, left: 8 }}
            >
              <CartesianGrid horizontal={false} stroke={chartColors.grid} />
              <XAxis
                type="number"
                tick={axisTick}
                tickLine={false}
                axisLine={{ stroke: chartColors.grid }}
                tickFormatter={(v: number) => `${v > 0 ? "+" : ""}${v}%`}
              />
              <YAxis
                type="category"
                dataKey="number"
                width={90}
                tick={axisTick}
                tickLine={false}
                axisLine={false}
                interval={0}
              />
              <ReferenceLine x={0} stroke={chartColors.axis} />
              <Tooltip cursor={tooltipCursor} content={<MoverTooltip />} />
              <Bar dataKey="delta" radius={2} maxBarSize={20}>
                {movers.map((d) => (
                  <Cell
                    key={d.number}
                    fill={d.delta >= 0 ? chartColors.emerald : chartColors.rose}
                  />
                ))}
              </Bar>
            </BarChart>
          </ChartContainer>
        </Card>
      )}

      <div className="overflow-x-auto rounded-xl border border-slate-800 bg-slate-900/40">
        <Table className="min-w-[560px]">
          <TableHeader>
            <TableRow className="hover:bg-transparent">
              <TableHead>Card</TableHead>
              <TableHead>Name</TableHead>
              <TableHead className="text-right">Prev</TableHead>
              <TableHead className="text-right">Today</TableHead>
              <TableHead className="text-right">Change</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {rows.map((t) => (
              <TableRow key={t.number}>
                <TableCell className="whitespace-nowrap font-mono text-xs text-slate-300">
                  {t.url ? (
                    <a href={t.url} target="_blank" rel="noreferrer" className="hover:text-sky-300">
                      {t.number}
                    </a>
                  ) : (
                    t.number
                  )}
                </TableCell>
                <TableCell className="text-slate-100">{t.name}</TableCell>
                <TableCell className="whitespace-nowrap text-right tabular-nums text-slate-400">
                  {t.prevBRL > 0 ? brl(t.prevBRL) : "—"}
                </TableCell>
                <TableCell className="whitespace-nowrap text-right tabular-nums text-slate-100">
                  {brl(t.lowBRL)}
                </TableCell>
                <TableCell
                  className={`whitespace-nowrap text-right font-semibold tabular-nums ${deltaColor(
                    t.prevBRL > 0 ? t.deltaPct : 0,
                  )}`}
                >
                  {deltaLabel(t)}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
    </div>
  );
}
