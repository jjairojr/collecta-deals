import { Coins, ShoppingBag, Store, TrendingDown, TrendingUp } from "lucide-react";
import type { CardTrend } from "../api";
import { brl0 } from "../format";
import { Card } from "./ui/card";

function cleanName(n: string): string {
  return n.replace(/\s*\([^)]*\)\s*$/, "");
}

function Kpi({
  icon,
  label,
  value,
  sub,
  tone = "slate",
}: {
  icon: React.ReactNode;
  label: string;
  value: string;
  sub?: string;
  tone?: "slate" | "emerald" | "sky";
}) {
  const valueTone =
    tone === "emerald" ? "text-emerald-300" : tone === "sky" ? "text-sky-200" : "text-slate-100";
  return (
    <Card className="flex items-center gap-3 p-4">
      <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-slate-800/70 text-sky-300">
        {icon}
      </div>
      <div className="min-w-0">
        <div className="text-[10px] font-medium uppercase tracking-wide text-slate-500">{label}</div>
        <div className={`truncate text-lg font-semibold tabular-nums ${valueTone}`}>{value}</div>
        {sub && <div className="truncate text-[11px] text-slate-500">{sub}</div>}
      </div>
    </Card>
  );
}

export default function KpiStrip({
  units,
  revenueBRL,
  storesSelling,
  topMover,
}: {
  units: number;
  revenueBRL: number;
  storesSelling: number;
  topMover: CardTrend | null;
}) {
  const up = (topMover?.deltaPct ?? 0) >= 0;
  return (
    <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
      <Kpi
        icon={<Coins className="h-5 w-5" />}
        label="Revenue moved"
        value={brl0(revenueBRL)}
        sub="est. from stock drops"
        tone="emerald"
      />
      <Kpi
        icon={<ShoppingBag className="h-5 w-5" />}
        label="Units sold"
        value={units.toLocaleString("pt-BR")}
        tone="sky"
      />
      <Kpi
        icon={<Store className="h-5 w-5" />}
        label="Stores selling"
        value={storesSelling.toLocaleString("pt-BR")}
      />
      <Kpi
        icon={up ? <TrendingUp className="h-5 w-5" /> : <TrendingDown className="h-5 w-5" />}
        label="Top mover"
        value={topMover ? `${up ? "+" : ""}${Math.round(topMover.deltaPct)}%` : "—"}
        sub={topMover ? cleanName(topMover.name) : "no price moves"}
        tone={topMover ? (up ? "emerald" : "slate") : "slate"}
      />
    </div>
  );
}
