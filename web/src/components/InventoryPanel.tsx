import { useEffect, useMemo, useState } from "react";
import { Bar, BarChart, CartesianGrid, Cell, Tooltip, XAxis, YAxis } from "recharts";
import { Boxes, Coins, Crown, Store } from "lucide-react";
import { cardImageURL, getInventory, type ExpensiveCard, type InventorySummary, type StoreInventoryStat } from "../api";
import { brl0 } from "../format";
import { Card } from "./ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "./ui/table";
import { ToggleGroup } from "./ui/toggle-group";
import { ChartContainer, ChartTooltip, axisTick, chartColors, tooltipCursor } from "./ui/chart";

type SortKey = "value" | "units";

function isSortKey(v: string): v is SortKey {
  return v === "value" || v === "units";
}

interface StoreDatum {
  name: string;
  value: number;
  units: number;
}

function StoreTooltip({
  active,
  payload,
  sort,
}: {
  active?: boolean;
  payload?: { payload: StoreDatum }[];
  sort: SortKey;
}) {
  if (!active || !payload || payload.length === 0) {
    return null;
  }
  const d = payload[0].payload;
  return (
    <ChartTooltip
      title={d.name}
      rows={[
        {
          label: "Value if sold",
          value: brl0(d.value),
          color: sort === "value" ? chartColors.emerald : undefined,
        },
        {
          label: "Units",
          value: d.units.toLocaleString("pt-BR"),
          color: sort === "units" ? chartColors.sky : undefined,
        },
      ]}
    />
  );
}

function Kpi({
  icon,
  label,
  value,
  sub,
}: {
  icon: React.ReactNode;
  label: string;
  value: string;
  sub?: string;
}) {
  return (
    <Card className="flex items-center gap-3 p-4">
      <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-slate-800/70 text-sky-300">
        {icon}
      </div>
      <div className="min-w-0">
        <div className="text-[10px] font-medium uppercase tracking-wide text-slate-500">{label}</div>
        <div className="truncate text-lg font-semibold tabular-nums text-slate-100">{value}</div>
        {sub && <div className="truncate text-[11px] text-slate-500">{sub}</div>}
      </div>
    </Card>
  );
}

function ChaseCard({ set, card }: { set: string; card: ExpensiveCard }) {
  const [failed, setFailed] = useState(false);
  return (
    <Card className="overflow-hidden p-0">
      <div className="relative">
        {failed ? (
          <div className="flex aspect-[350/489] w-full items-center justify-center bg-slate-800 text-center">
            <span className="px-2 font-mono text-[11px] text-slate-500">{card.number}</span>
          </div>
        ) : (
          <img
            src={cardImageURL(set, card.number)}
            alt={card.name}
            loading="lazy"
            onError={() => setFailed(true)}
            className="aspect-[350/489] w-full bg-slate-800 object-cover"
          />
        )}
        <div className="absolute right-2 top-2 rounded-lg bg-slate-950/85 px-2 py-1 text-sm font-bold tabular-nums text-emerald-300 ring-1 ring-inset ring-emerald-500/30 backdrop-blur">
          {brl0(card.lowBRL)}
        </div>
      </div>
      <div className="flex flex-col gap-1.5 p-2.5">
        <div className="truncate text-xs font-medium text-slate-100" title={card.name}>
          {card.name.replace(/\s*\([^)]*\)\s*$/, "")}
        </div>
        <div className="font-mono text-[10px] text-slate-500">
          {card.number} · {card.totalQty} in market · {card.stores} stores
        </div>
        <div className="flex flex-wrap gap-1">
          {card.holders.slice(0, 3).map((h) => (
            <span
              key={h.storeId}
              className="rounded bg-slate-800/70 px-1.5 py-0.5 text-[10px] text-slate-300"
            >
              {h.storeName} <span className="text-slate-500">×{h.quantity}</span>
            </span>
          ))}
        </div>
      </div>
    </Card>
  );
}

export default function InventoryPanel({ set = "OP-16" }: { set?: string }) {
  const [summary, setSummary] = useState<InventorySummary | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [sort, setSort] = useState<SortKey>("value");

  useEffect(() => {
    let current = true;
    getInventory(set)
      .then((r) => {
        if (current) {
          setSummary(r.ready ? r.summary : null);
        }
      })
      .catch((err: unknown) =>
        setError(err instanceof Error ? err.message : "failed to load inventory"),
      );
    return () => {
      current = false;
    };
  }, [set]);

  const stores = useMemo<StoreInventoryStat[]>(() => {
    if (!summary) {
      return [];
    }
    return [...summary.stores].sort((a, b) =>
      sort === "units" ? b.units - a.units : b.valueBRL - a.valueBRL,
    );
  }, [summary, sort]);

  const chartData = useMemo<StoreDatum[]>(
    () =>
      stores.slice(0, 12).map((s) => ({
        name: s.storeName || `Store ${s.storeId}`,
        value: Math.round(s.valueBRL),
        units: s.units,
      })),
    [stores],
  );

  if (error) {
    return <Panel>Could not load inventory: {error}</Panel>;
  }
  if (!summary) {
    return null;
  }

  const barColor = sort === "value" ? chartColors.emerald : chartColors.sky;
  const chase = summary.expensive[0];

  return (
    <div className="space-y-6">
      <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
        <Kpi
          icon={<Coins className="h-5 w-5" />}
          label="Market value"
          value={brl0(summary.totalValue)}
          sub="if every copy sold at floor"
        />
        <Kpi
          icon={<Boxes className="h-5 w-5" />}
          label="Units tracked"
          value={summary.totalUnits.toLocaleString("pt-BR")}
          sub={`${summary.date}`}
        />
        <Kpi
          icon={<Store className="h-5 w-5" />}
          label="Active stores"
          value={summary.activeStores.toLocaleString("pt-BR")}
        />
        <Kpi
          icon={<Crown className="h-5 w-5" />}
          label="Chase card"
          value={chase ? brl0(chase.lowBRL) : "—"}
          sub={chase ? chase.number : undefined}
        />
      </div>

      <div className="flex items-center justify-between gap-3">
        <h3 className="text-sm font-semibold text-slate-200">Stores by holdings</h3>
        <ToggleGroup
          value={sort}
          onChange={(v) => {
            if (isSortKey(v)) {
              setSort(v);
            }
          }}
          options={[
            { value: "value", label: "Value if sold" },
            { value: "units", label: "Supply" },
          ]}
        />
      </div>

      {chartData.length > 0 && (
        <Card className="p-4">
          <ChartContainer height={Math.max(220, chartData.length * 30)}>
            <BarChart data={chartData} layout="vertical" margin={{ top: 4, right: 16, bottom: 4, left: 8 }}>
              <CartesianGrid horizontal={false} stroke={chartColors.grid} />
              <XAxis
                type="number"
                tick={axisTick}
                tickLine={false}
                axisLine={{ stroke: chartColors.grid }}
                tickFormatter={(v: number) =>
                  sort === "value" ? brl0(v) : v.toLocaleString("pt-BR")
                }
              />
              <YAxis
                type="category"
                dataKey="name"
                width={140}
                tick={axisTick}
                tickLine={false}
                axisLine={false}
                interval={0}
              />
              <Tooltip cursor={tooltipCursor} content={<StoreTooltip sort={sort} />} />
              <Bar dataKey={sort === "value" ? "value" : "units"} radius={[0, 4, 4, 0]} maxBarSize={22}>
                {chartData.map((d) => (
                  <Cell key={d.name} fill={barColor} />
                ))}
              </Bar>
            </BarChart>
          </ChartContainer>
        </Card>
      )}

      <div className="overflow-x-auto rounded-xl border border-slate-800 bg-slate-900/40">
        <Table className="min-w-[620px]">
          <TableHeader>
            <TableRow className="hover:bg-transparent">
              <TableHead>#</TableHead>
              <TableHead>Store</TableHead>
              <TableHead className="text-right">Units</TableHead>
              <TableHead className="text-right">Value if sold</TableHead>
              <TableHead>Priciest card</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {stores.map((s, i) => (
              <TableRow key={s.storeId}>
                <TableCell className="tabular-nums text-slate-500">{i + 1}</TableCell>
                <TableCell className="text-slate-100">{s.storeName}</TableCell>
                <TableCell className="whitespace-nowrap text-right tabular-nums text-slate-300">
                  {s.units.toLocaleString("pt-BR")}
                </TableCell>
                <TableCell
                  className={`whitespace-nowrap text-right font-semibold tabular-nums ${
                    sort === "value" ? "text-emerald-300" : "text-slate-200"
                  }`}
                >
                  {brl0(s.valueBRL)}
                </TableCell>
                <TableCell className="whitespace-nowrap text-xs text-slate-400">
                  <span className="font-mono text-slate-300">{s.topCardNumber}</span>{" "}
                  <span className="text-slate-500">{brl0(s.topCardBRL)}</span>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>

      <div>
        <h3 className="mb-3 text-sm font-semibold text-slate-200">
          Most expensive cards <span className="font-normal text-slate-500">& who holds them</span>
        </h3>
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5">
          {summary.expensive.map((c) => (
            <ChaseCard key={c.number} set={set} card={c} />
          ))}
        </div>
      </div>
    </div>
  );
}

function Panel({ children }: { children: React.ReactNode }) {
  return (
    <div className="rounded-xl border border-slate-800 bg-slate-900/40 px-4 py-10 text-center text-slate-400">
      {children}
    </div>
  );
}
