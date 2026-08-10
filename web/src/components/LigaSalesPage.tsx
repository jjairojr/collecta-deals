import { useCallback, useEffect, useMemo, useState } from "react";
import { Bar, BarChart, CartesianGrid, Tooltip, XAxis, YAxis } from "recharts";
import { Check, ChevronDown, ChevronRight, Coins, Package, Pencil, Receipt, TrendingUp, X } from "lucide-react";
import { listSales, updateTrade, type SaleRow } from "../api";
import { brl0, brl2, dayLabel } from "../format";
import {
  buildPackages,
  channelLabels,
  channelOf,
  channels,
  dailySeries,
  isBRL,
  isSymbolic,
  totalsOf,
  type Channel,
  type SalesPackage,
} from "../ligasales";
import EmptyState from "./EmptyState";
import { Chip, Kpi } from "./PortfolioPage";
import { Button } from "./ui/button";
import { Card } from "./ui/card";
import { Checkbox } from "./ui/checkbox";
import { Input } from "./ui/input";
import { ChartContainer, ChartTooltip, axisTick, chartColors, tooltipCursor } from "./ui/chart";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "./ui/table";

const DEFAULT_CHANNELS: Channel[] = ["liga", "sem-marca"];

function pctLabel(value: number): string {
  return `${value >= 0 ? "+" : ""}${Math.round(value)}%`;
}

function moneyClass(value: number): string {
  return value >= 0 ? "text-emerald-300" : "text-rose-300";
}

function LegendDot({ color, children }: { color: string; children: string }) {
  return (
    <span className="flex items-center gap-1.5">
      <span
        className="h-2.5 w-2.5 rounded-[3px] border-2 border-outline"
        style={{ backgroundColor: color }}
      />
      {children}
    </span>
  );
}

function DayTooltip({
  active,
  payload,
}: {
  active?: boolean;
  payload?: { payload: { day: string; revenueBRL: number; profitBRL: number } }[];
}) {
  if (!active || !payload || payload.length === 0) {
    return null;
  }
  const d = payload[0].payload;
  return (
    <ChartTooltip
      title={dayLabel(d.day)}
      rows={[
        { label: "Receita", value: brl2(d.revenueBRL), color: chartColors.brand },
        { label: "Lucro", value: brl2(d.profitBRL), color: chartColors.emerald },
      ]}
    />
  );
}

export default function LigaSalesPage() {
  const [sales, setSales] = useState<SaleRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selected, setSelected] = useState<Channel[]>(DEFAULT_CHANNELS);
  const [hideSymbolic, setHideSymbolic] = useState(true);
  const [open, setOpen] = useState<string | null>(null);

  const load = useCallback(async () => {
    try {
      const r = await listSales();
      setSales(r.sales);
      setError(null);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "falha ao carregar as vendas");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const brlSales = useMemo(() => sales.filter(isBRL), [sales]);
  const usdCount = sales.length - brlSales.length;
  const visible = useMemo(
    () => (hideSymbolic ? brlSales.filter((s) => !isSymbolic(s)) : brlSales),
    [brlSales, hideSymbolic],
  );
  const counts = useMemo(() => {
    const c = new Map<Channel, number>();
    for (const s of visible) {
      const ch = channelOf(s.buyer);
      c.set(ch, (c.get(ch) ?? 0) + 1);
    }
    return c;
  }, [visible]);

  const packages = useMemo(
    () => buildPackages(visible.filter((s) => selected.includes(channelOf(s.buyer)))),
    [visible, selected],
  );
  const totals = useMemo(() => totalsOf(packages), [packages]);
  const series = useMemo(() => dailySeries(packages), [packages]);

  const toggle = (ch: Channel) =>
    setSelected((prev) => (prev.includes(ch) ? prev.filter((c) => c !== ch) : [...prev, ch]));

  if (loading) {
    return <EmptyState>Carregando vendas…</EmptyState>;
  }
  if (error) {
    return (
      <div className="rounded-[14px] border-2 border-rose-900/50 bg-rose-950/30 px-4 py-3 text-sm text-rose-200">
        {error}
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="space-y-2">
        <div className="flex flex-wrap items-center gap-2">
          {channels.map((ch) => (
            <Chip key={ch} active={selected.includes(ch)} onClick={() => toggle(ch)}>
              {channelLabels[ch]} · {counts.get(ch) ?? 0}
            </Chip>
          ))}
          <label className="ml-2 flex cursor-pointer items-center gap-2 text-xs text-slate-400">
            <Checkbox
              checked={hideSymbolic}
              onChange={(e) => setHideSymbolic(e.target.checked)}
            />
            Ocultar baixas simbólicas (R$ 1/un.)
          </label>
        </div>
        <p className="text-[11px] text-slate-500">
          “Sem marca” são baixas feitas à mão, sem nº de pedido — quase sempre vendas da Liga.
          {usdCount > 0 && ` ${usdCount} vendas em US$ (TCGplayer) ficam fora desta página.`}
        </p>
      </div>

      <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
        <Kpi
          icon={<Coins className="h-5 w-5" />}
          label="Receita"
          value={brl2(totals.revenueBRL)}
          sub={`${totals.units} unidades em ${totals.items} itens`}
        />
        <Kpi
          icon={<Receipt className="h-5 w-5" />}
          label="Custo"
          value={brl2(totals.costBRL)}
          sub="o que essas cartas custaram"
        />
        <Kpi
          icon={<TrendingUp className="h-5 w-5" />}
          label="Lucro"
          value={brl2(totals.profitBRL)}
          sub={`margem ${pctLabel(totals.marginPct)}`}
        />
        <Kpi
          icon={<Package className="h-5 w-5" />}
          label="Pacotes"
          value={String(totals.packages)}
          sub={
            totals.packages > 0
              ? `lucro médio ${brl2(totals.profitBRL / totals.packages)}`
              : "nenhum pedido no filtro"
          }
        />
      </div>

      {series.length > 0 && (
        <Card className="p-4">
          <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
            <h2 className="font-display text-base font-extrabold text-fg">Receita e lucro por dia</h2>
            <div className="flex items-center gap-3 text-[11px] text-slate-400">
              <LegendDot color={chartColors.brand}>Receita</LegendDot>
              <LegendDot color={chartColors.emerald}>Lucro</LegendDot>
            </div>
          </div>
          <ChartContainer height={280}>
            <BarChart data={series} margin={{ top: 4, right: 8, bottom: 4, left: 8 }}>
              <CartesianGrid vertical={false} stroke={chartColors.grid} />
              <XAxis
                dataKey="day"
                tick={axisTick}
                tickLine={false}
                axisLine={{ stroke: chartColors.grid }}
                tickFormatter={dayLabel}
              />
              <YAxis
                tick={axisTick}
                tickLine={false}
                axisLine={false}
                width={70}
                tickFormatter={(v: number) => brl0(v)}
              />
              <Tooltip cursor={tooltipCursor} content={<DayTooltip />} />
              <Bar dataKey="revenueBRL" fill={chartColors.brand} radius={[4, 4, 0, 0]} maxBarSize={34} isAnimationActive={false} />
              <Bar dataKey="profitBRL" fill={chartColors.emerald} radius={[4, 4, 0, 0]} maxBarSize={34} isAnimationActive={false} />
            </BarChart>
          </ChartContainer>
        </Card>
      )}

      {packages.length === 0 ? (
        <EmptyState hint="ajuste os filtros acima">
          Nenhuma venda nos canais selecionados.
        </EmptyState>
      ) : (
        <div className="sticker sticker-sm overflow-x-auto rounded-[12px] bg-panel">
          <Table className="min-w-[900px]">
            <TableHeader>
              <TableRow className="hover:bg-transparent">
                <TableHead>Data</TableHead>
                <TableHead>Pacote</TableHead>
                <TableHead className="text-right">Itens</TableHead>
                <TableHead className="text-right">Receita</TableHead>
                <TableHead className="text-right">Custo</TableHead>
                <TableHead className="text-right">Lucro</TableHead>
                <TableHead className="text-right">Margem</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {packages.map((p) => (
                <PackageRows
                  key={p.key}
                  p={p}
                  open={open === p.key}
                  onToggle={() => setOpen((k) => (k === p.key ? null : p.key))}
                  onSaved={load}
                />
              ))}
            </TableBody>
          </Table>
        </div>
      )}
    </div>
  );
}

function PackageRows({
  p,
  open,
  onToggle,
  onSaved,
}: {
  p: SalesPackage;
  open: boolean;
  onToggle: () => void;
  onSaved: () => void;
}) {
  return (
    <>
      <TableRow className="cursor-pointer" onClick={onToggle}>
        <TableCell className="whitespace-nowrap tabular-nums text-slate-400">
          {p.date ? dayLabel(p.date) : "—"}
        </TableCell>
        <TableCell>
          <div className="flex items-center gap-2">
            {open ? (
              <ChevronDown className="h-4 w-4 shrink-0 text-slate-500" />
            ) : (
              <ChevronRight className="h-4 w-4 shrink-0 text-slate-500" />
            )}
            <div className="min-w-0">
              <div className="truncate font-medium text-slate-100">
                {p.order ? `Pedido #${p.order}` : p.buyer || "Sem nº de pedido"}
              </div>
              <div className="truncate text-[10px] text-slate-500">
                {[p.order ? p.buyer : "", p.games.join(" · ")].filter(Boolean).join(" — ")}
              </div>
            </div>
          </div>
        </TableCell>
        <TableCell className="text-right tabular-nums text-slate-400">
          {p.items.length} <span className="text-slate-600">/ {p.units} un.</span>
        </TableCell>
        <TableCell className="text-right font-semibold tabular-nums text-slate-200">
          {brl2(p.revenueBRL)}
        </TableCell>
        <TableCell className="text-right tabular-nums text-slate-400">{brl2(p.costBRL)}</TableCell>
        <TableCell className={`text-right font-bold tabular-nums ${moneyClass(p.profitBRL)}`}>
          {brl2(p.profitBRL)}
        </TableCell>
        <TableCell className={`text-right tabular-nums ${moneyClass(p.profitBRL)}`}>
          {pctLabel(p.marginPct)}
        </TableCell>
      </TableRow>
      {open && (
        <TableRow className="hover:bg-transparent">
          <TableCell colSpan={7} className="bg-raised/40 px-4 py-3">
            <Table className="min-w-[640px]">
              <TableHeader>
                <TableRow className="hover:bg-transparent">
                  <TableHead className="border-b-0 bg-transparent px-2 py-1">Carta</TableHead>
                  <TableHead className="border-b-0 bg-transparent px-2 py-1 text-right">Un.</TableHead>
                  <TableHead className="border-b-0 bg-transparent px-2 py-1 text-right">Receita</TableHead>
                  <TableHead className="border-b-0 bg-transparent px-2 py-1 text-right">Custo</TableHead>
                  <TableHead className="border-b-0 bg-transparent px-2 py-1 text-right">Lucro</TableHead>
                  <TableHead className="border-b-0 bg-transparent px-2 py-1" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {p.items.map((s) => (
                  <SaleItemRow key={s.id} s={s} onSaved={onSaved} />
                ))}
              </TableBody>
            </Table>
          </TableCell>
        </TableRow>
      )}
    </>
  );
}

function SaleItemRow({ s, onSaved }: { s: SaleRow; onSaved: () => void }) {
  const [editing, setEditing] = useState(false);
  const [qty, setQty] = useState(String(s.qty));
  const [price, setPrice] = useState(String(s.sellPrice ?? s.valueBRL));
  const [date, setDate] = useState(s.sellDate ?? "");
  const [buyer, setBuyer] = useState(s.buyer ?? "");
  const [saving, setSaving] = useState(false);
  const [failed, setFailed] = useState<string | null>(null);

  const start = () => {
    setQty(String(s.qty));
    setPrice(String(s.sellPrice ?? s.valueBRL));
    setDate(s.sellDate ?? "");
    setBuyer(s.buyer ?? "");
    setFailed(null);
    setEditing(true);
  };

  const save = async () => {
    const units = Number(qty.replace(",", "."));
    const total = Number(price.replace(",", "."));
    if (!Number.isFinite(units) || units <= 0 || !Number.isFinite(total) || total < 0) {
      setFailed("quantidade e valor precisam ser números");
      return;
    }
    setSaving(true);
    try {
      await updateTrade(
        s.id,
        { ...s, qty: units, sellPrice: total, sellDate: date, buyer: buyer.trim() },
        s.game === "acessorios" ? undefined : s.game,
      );
      setEditing(false);
      onSaved();
    } catch (err: unknown) {
      setFailed(err instanceof Error ? err.message : "falha ao salvar");
    } finally {
      setSaving(false);
    }
  };

  if (!editing) {
    return (
      <TableRow className="hover:bg-transparent">
        <TableCell className="px-2 py-1">
          <div className="truncate text-slate-200">{s.name}</div>
          <div className="truncate text-[10px] text-slate-500">
            {[s.number, s.set, s.gameName, s.kind === "sealed" ? "selado" : s.kind === "accessory" ? "acessório" : "single"]
              .filter(Boolean)
              .join(" · ")}
          </div>
        </TableCell>
        <TableCell className="px-2 py-1 text-right tabular-nums text-slate-400">{s.qty}</TableCell>
        <TableCell className="px-2 py-1 text-right tabular-nums text-slate-200">
          {brl2(s.valueBRL)}
        </TableCell>
        <TableCell className="px-2 py-1 text-right tabular-nums text-slate-400">
          {brl2(s.costBRL)}
        </TableCell>
        <TableCell className={`px-2 py-1 text-right font-semibold tabular-nums ${moneyClass(s.profitBRL)}`}>
          {brl2(s.profitBRL)}
        </TableCell>
        <TableCell className="px-2 py-1 text-right">
          <button
            type="button"
            onClick={start}
            title="Editar esta venda"
            className="rounded-[6px] border-2 border-outline bg-panel p-1 text-slate-400 hover:text-fg"
          >
            <Pencil className="h-3.5 w-3.5" />
          </button>
        </TableCell>
      </TableRow>
    );
  }

  return (
    <TableRow className="hover:bg-transparent">
      <TableCell colSpan={6} className="px-2 py-2">
        <div className="flex flex-wrap items-end gap-2">
          <div className="min-w-[9rem] flex-1">
            <FieldLabel>{s.name}</FieldLabel>
            <div className="truncate text-[10px] text-slate-500">
              {[s.number, s.set, s.gameName].filter(Boolean).join(" · ")}
            </div>
          </div>
          <label className="w-[4.5rem]">
            <FieldLabel>Un.</FieldLabel>
            <Input value={qty} onChange={(e) => setQty(e.target.value)} inputMode="numeric" />
          </label>
          <label className="w-[7.5rem]">
            <FieldLabel>Venda total</FieldLabel>
            <Input value={price} onChange={(e) => setPrice(e.target.value)} inputMode="decimal" />
          </label>
          <label className="w-[9.5rem]">
            <FieldLabel>Data</FieldLabel>
            <Input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
          </label>
          <label className="min-w-[12rem] flex-1">
            <FieldLabel>Comprador / canal</FieldLabel>
            <Input
              value={buyer}
              onChange={(e) => setBuyer(e.target.value)}
              placeholder="ex.: Fulano — Liga #11641862, shopee, balcão"
            />
          </label>
          <div className="flex items-center gap-1">
            <Button variant="primary" size="sm" onClick={save} disabled={saving}>
              <Check className="h-4 w-4" />
            </Button>
            <Button variant="ghost" size="sm" onClick={() => setEditing(false)} disabled={saving}>
              <X className="h-4 w-4" />
            </Button>
          </div>
        </div>
        <p className="mt-1 text-[10px] text-slate-500">
          Venda total é o valor do item inteiro (unitário × unidades), sem frete. O comprador define o canal:
          “Liga #nº” marca pedido da Liga, vazio conta como venda sem marca.
        </p>
        {failed && <p className="mt-1 text-[11px] text-rose-300">{failed}</p>}
      </TableCell>
    </TableRow>
  );
}

function FieldLabel({ children }: { children: React.ReactNode }) {
  return <div className="font-pixel text-[8px] uppercase text-brand-label">{children}</div>;
}
