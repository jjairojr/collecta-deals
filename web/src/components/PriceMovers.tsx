import { useMemo } from "react";
import { TrendingDown, TrendingUp } from "lucide-react";
import type { CardTrend } from "../api";
import { brl0, stampLabel } from "../format";
import CardArt from "./CardArt";
import { Card } from "./ui/card";

function cleanName(n: string): string {
  return n.replace(/\s*\([^)]*\)\s*$/, "");
}

function MoverTile({ trend, set }: { trend: CardTrend; set: string }) {
  const up = trend.deltaPct >= 0;
  const rounded = Math.round(trend.deltaPct);
  return (
    <Card className="overflow-hidden p-0">
      <div className="relative">
        <CardArt
          set={trend.set ?? set}
          number={trend.number}
          name={trend.name}
          className="aspect-[350/489] w-full"
        />
        <div
          className={`absolute left-2 top-2 flex items-center gap-1 rounded-lg px-2 py-0.5 text-xs font-bold tabular-nums shadow ${
            up ? "bg-emerald-400 text-slate-950" : "bg-rose-400 text-slate-950"
          }`}
        >
          {up ? <TrendingUp className="h-3 w-3" /> : <TrendingDown className="h-3 w-3" />}
          {up ? "+" : ""}
          {rounded}%
        </div>
      </div>
      <div className="p-2">
        <div className="truncate text-xs font-medium text-slate-100" title={trend.name}>
          {cleanName(trend.name)}
        </div>
        <div className="font-mono text-[10px] text-slate-500">
          {trend.number}
          {trend.set ? ` · ${trend.set}` : ""}
        </div>
        <div className="mt-0.5 text-[11px] tabular-nums text-slate-400">
          {brl0(trend.prevBRL)} <span className="text-slate-600">→</span>{" "}
          <span className={up ? "text-emerald-300" : "text-rose-300"}>{brl0(trend.lowBRL)}</span>
        </div>
      </div>
    </Card>
  );
}

export default function PriceMovers({
  trends,
  prevDate,
  set,
  loaded,
}: {
  trends: CardTrend[];
  prevDate: string;
  set: string;
  loaded: boolean;
}) {
  const movers = useMemo(
    () =>
      trends
        .filter((t) => t.prevBRL > 0 && t.deltaPct !== 0)
        .sort((a, b) => Math.abs(b.deltaPct) - Math.abs(a.deltaPct)),
    [trends],
  );
  const gainers = useMemo(() => movers.filter((m) => m.deltaPct > 0).slice(0, 12), [movers]);
  const losers = useMemo(() => movers.filter((m) => m.deltaPct < 0).slice(0, 6), [movers]);

  return (
    <section className="space-y-3">
      <div className="flex items-center gap-2">
        <span className="text-sky-300">
          <TrendingUp className="h-4 w-4" />
        </span>
        <h2 className="text-sm font-semibold text-slate-200">Biggest price movers</h2>
        <span className="text-xs text-slate-500">
          · {prevDate ? `floor change vs ${stampLabel(prevDate)}` : "floor-price changes"}
        </span>
      </div>
      {!loaded ? (
        <div className="rounded-xl border border-slate-800 bg-slate-900/30 px-4 py-10 text-center text-sm text-slate-500">
          Loading price moves…
        </div>
      ) : movers.length === 0 ? (
        <div className="rounded-xl border border-slate-800 bg-slate-900/30 px-4 py-8 text-center text-sm text-slate-500">
          No price moves in this range — floors were flat.
        </div>
      ) : (
        <div className="space-y-4">
          {gainers.length > 0 && (
            <div className="grid grid-cols-3 gap-3 sm:grid-cols-4 md:grid-cols-6">
              {gainers.map((t) => (
                <MoverTile key={`g${t.set ?? ""}${t.number}`} trend={t} set={set} />
              ))}
            </div>
          )}
          {losers.length > 0 && (
            <div className="grid grid-cols-3 gap-3 sm:grid-cols-4 md:grid-cols-6">
              {losers.map((t) => (
                <MoverTile key={`l${t.set ?? ""}${t.number}`} trend={t} set={set} />
              ))}
            </div>
          )}
        </div>
      )}
    </section>
  );
}
