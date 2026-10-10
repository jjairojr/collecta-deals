import { useCallback, useEffect, useMemo, useState } from "react";
import { Bar, BarChart, CartesianGrid, Tooltip, XAxis, YAxis } from "recharts";
import {
  ArrowUpDown,
  Check,
  ChevronDown,
  ChevronRight,
  Coins,
  Package,
  Pencil,
  Receipt,
  TrendingUp,
  X,
} from "lucide-react";
import { listSales, productIDFromTcgURL, updateTrade, type SaleRow } from "../api";
import { brl0, brl2, dayLabel } from "../format";
import {
  buildPackages,
  buyerName,
  channelLabels,
  channelOfSale,
  channels,
  dailySeries,
  isSymbolic,
  orderNumber,
  totalsOf,
  type Channel,
  type SalesPackage,
} from "../ligasales";
import CardArt from "./CardArt";
import EmptyState from "./EmptyState";
import { Chip, Kpi } from "./PortfolioPage";
import { Button } from "./ui/button";
import { Card } from "./ui/card";
import { Checkbox } from "./ui/checkbox";
import { Input } from "./ui/input";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "./ui/table";
import { ChartContainer, ChartTooltip, axisTick, chartColors, tooltipCursor } from "./ui/chart";

// The page opens on what sold, newest first — the question asked most often is
// "o que vendeu?", and grouping by package answered a different one.
type SalesView = "items" | "packages";

type Period = 7 | 30 | 90 | 0;

const periodLabels: Record<Period, string> = {
  7: "7 dias",
  30: "30 dias",
  90: "90 dias",
  0: "Tudo",
};

const DEFAULT_CHANNELS: Channel[] = ["liga", "sem-marca", "shopee", "outros"];

type SortKey = "date" | "revenue" | "profit" | "margin";

interface SortState {
  key: SortKey;
  dir: "asc" | "desc";
}

function pctLabel(value: number): string {
  return `${value >= 0 ? "+" : ""}${Math.round(value)}%`;
}

function moneyClass(value: number): string {
  return value >= 0 ? "text-emerald-300" : "text-rose-300";
}

function marginOf(s: SaleRow): number {
  return s.costBRL > 0 ? (s.profitBRL / s.costBRL) * 100 : 0;
}

function sortValue(s: SaleRow, key: SortKey): string | number {
  switch (key) {
    case "revenue":
      return s.valueBRL;
    case "profit":
      return s.profitBRL;
    case "margin":
      return marginOf(s);
    default:
      return s.sellDate ?? "";
  }
}

function sortSales(list: SaleRow[], sort: SortState): SaleRow[] {
  const dir = sort.dir === "asc" ? 1 : -1;
  return [...list].sort((a, b) => {
    const va = sortValue(a, sort.key);
    const vb = sortValue(b, sort.key);
    if (va === vb) {
      return b.valueBRL - a.valueBRL;
    }
    return (typeof va === "number" && typeof vb === "number" ? va - vb : String(va).localeCompare(String(vb))) * dir;
  });
}

function matches(s: SaleRow, q: string): boolean {
  if (!q) {
    return true;
  }
  return `${s.name} ${s.number} ${s.set} ${s.gameName} ${s.buyer ?? ""}`.toLowerCase().includes(q);
}

// cutoff is the oldest sell date a period still shows, as YYYY-MM-DD in local
// time — sell dates are days, not instants.
function cutoff(days: Period): string {
  if (days === 0) {
    return "";
  }
  const d = new Date();
  d.setDate(d.getDate() - days);
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
}

function LegendDot({ color, children }: { color: string; children: string }) {
  return (
    <span className="flex items-center gap-1.5">
      <span className="h-2.5 w-2.5 rounded-[3px] border-2 border-outline" style={{ backgroundColor: color }} />
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
  const [view, setView] = useState<SalesView>("items");
  const [period, setPeriod] = useState<Period>(30);
  const [selected, setSelected] = useState<Channel[]>(DEFAULT_CHANNELS);
  const [hideSymbolic, setHideSymbolic] = useState(true);
  const [query, setQuery] = useState("");
  const [sort, setSort] = useState<SortState>({ key: "date", dir: "desc" });
  const [open, setOpen] = useState<string | null>(null);
  const [showChart, setShowChart] = useState(false);

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

  // Uma baixa feita fora do navegador (skill liga-vendas, outra aba) não avisa
  // esta tela: sem isso a lista fica congelada no que foi carregado na abertura.
  useEffect(() => {
    const onFocus = () => {
      if (document.visibilityState === "visible") {
        load();
      }
    };
    window.addEventListener("focus", onFocus);
    document.addEventListener("visibilitychange", onFocus);
    return () => {
      window.removeEventListener("focus", onFocus);
      document.removeEventListener("visibilitychange", onFocus);
    };
  }, [load]);

  const onSort = (key: SortKey) =>
    setSort((s) => (s.key === key ? { key, dir: s.dir === "asc" ? "desc" : "asc" } : { key, dir: "desc" }));

  const q = query.trim().toLowerCase();
  const from = cutoff(period);

  // Counts on the channel chips are of everything else already filtered, so a
  // chip says how many sales turning it on would actually add.
  const beforeChannel = useMemo(
    () =>
      sales
        .filter((s) => (hideSymbolic ? !isSymbolic(s) : true))
        .filter((s) => (from ? (s.sellDate ?? "") >= from : true))
        .filter((s) => matches(s, q)),
    [sales, hideSymbolic, from, q],
  );
  const counts = useMemo(() => {
    const c = new Map<Channel, number>();
    for (const s of beforeChannel) {
      const ch = channelOfSale(s);
      c.set(ch, (c.get(ch) ?? 0) + 1);
    }
    return c;
  }, [beforeChannel]);

  const visible = useMemo(
    () => beforeChannel.filter((s) => selected.includes(channelOfSale(s))),
    [beforeChannel, selected],
  );
  const rows = useMemo(() => sortSales(visible, sort), [visible, sort]);
  const packages = useMemo(() => buildPackages(visible), [visible]);
  const totals = useMemo(() => totalsOf(packages), [packages]);
  const series = useMemo(() => dailySeries(packages), [packages]);
  const lastSale = useMemo(
    () => sales.reduce((max, s) => ((s.sellDate ?? "") > max ? (s.sellDate ?? "") : max), ""),
    [sales],
  );

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
    <div className="space-y-5">
      <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
        <Kpi
          icon={<Coins className="h-5 w-5" />}
          label="Receita"
          value={brl2(totals.revenueBRL)}
          sub={`${totals.units} un. em ${totals.items} vendas`}
        />
        <Kpi
          icon={<Receipt className="h-5 w-5" />}
          label="Custo"
          value={brl2(totals.costBRL)}
          sub="o que esses itens custaram"
        />
        <Kpi
          icon={<TrendingUp className="h-5 w-5" />}
          label="Lucro"
          value={brl2(totals.profitBRL)}
          sub={`margem ${pctLabel(totals.marginPct)}`}
        />
        <Kpi
          icon={<Package className="h-5 w-5" />}
          label="Última venda"
          value={lastSale ? dayLabel(lastSale) : "—"}
          sub={`${totals.packages} pedidos no filtro`}
        />
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <Chip active={view === "items"} onClick={() => setView("items")}>
          Vendas
        </Chip>
        <Chip active={view === "packages"} onClick={() => setView("packages")}>
          Por pedido
        </Chip>
        <div className="ml-4 flex flex-wrap items-center gap-1.5">
          {([7, 30, 90, 0] as Period[]).map((p) => (
            <Chip key={p} active={period === p} onClick={() => setPeriod(p)}>
              {periodLabels[p]}
            </Chip>
          ))}
        </div>
        <div className="ml-auto flex flex-wrap items-center gap-2">
          <Input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Buscar carta, comprador, pedido…"
            className="w-64"
          />
          <Button variant="outline" onClick={() => setShowChart((v) => !v)}>
            {showChart ? "Esconder gráfico" : "Ver gráfico"}
          </Button>
        </div>
      </div>

      <div className="flex flex-wrap items-center gap-x-4 gap-y-2 border-b-[3px] border-outline pb-3">
        <div className="flex flex-wrap items-center gap-1.5">
          {channels.map((ch) => (
            <Chip key={ch} active={selected.includes(ch)} onClick={() => toggle(ch)}>
              {channelLabels[ch]} · {counts.get(ch) ?? 0}
            </Chip>
          ))}
        </div>
        <label className="flex cursor-pointer items-center gap-2 text-xs text-slate-400">
          <Checkbox checked={hideSymbolic} onChange={(e) => setHideSymbolic(e.target.checked)} />
          Ocultar baixas de R$ 1/un. (abertura de caixa)
        </label>
        <span className="ml-auto text-[11px] text-slate-500">
          “Sem marca” são baixas feitas à mão, sem nº de pedido — quase sempre vendas da Liga.
        </span>
      </div>

      {showChart && series.length > 0 && (
        <Card className="p-4">
          <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
            <h2 className="font-display text-base font-extrabold text-fg">Receita e lucro por dia</h2>
            <div className="flex items-center gap-3 text-[11px] text-slate-400">
              <LegendDot color={chartColors.brand}>Receita</LegendDot>
              <LegendDot color={chartColors.emerald}>Lucro</LegendDot>
            </div>
          </div>
          <ChartContainer height={220}>
            <BarChart data={series} margin={{ top: 4, right: 8, bottom: 4, left: 8 }}>
              <CartesianGrid vertical={false} stroke={chartColors.grid} />
              <XAxis
                dataKey="day"
                tick={axisTick}
                tickLine={false}
                axisLine={{ stroke: chartColors.grid }}
                tickFormatter={dayLabel}
              />
              <YAxis tick={axisTick} tickLine={false} axisLine={false} width={70} tickFormatter={(v: number) => brl0(v)} />
              <Tooltip cursor={tooltipCursor} content={<DayTooltip />} />
              <Bar dataKey="revenueBRL" fill={chartColors.brand} radius={[4, 4, 0, 0]} maxBarSize={34} isAnimationActive={false} />
              <Bar dataKey="profitBRL" fill={chartColors.emerald} radius={[4, 4, 0, 0]} maxBarSize={34} isAnimationActive={false} />
            </BarChart>
          </ChartContainer>
        </Card>
      )}

      {rows.length === 0 ? (
        <EmptyState hint={period === 0 ? "ajuste os canais ou a busca" : "tente um período maior"}>
          Nenhuma venda no filtro.
        </EmptyState>
      ) : view === "items" ? (
        <div className="sticker sticker-sm overflow-x-auto rounded-[12px] bg-panel">
          <Table className="min-w-[900px]">
            <TableHeader>
              <TableRow className="hover:bg-transparent">
                <SortHead label="Data" sortKey="date" sort={sort} onSort={onSort} />
                <TableHead>Item</TableHead>
                <TableHead>Comprador / canal</TableHead>
                <TableHead className="text-right">Un.</TableHead>
                <SortHead label="Receita" sortKey="revenue" sort={sort} onSort={onSort} className="text-right" />
                <TableHead className="text-right">Custo</TableHead>
                <SortHead label="Lucro" sortKey="profit" sort={sort} onSort={onSort} className="text-right" />
                <SortHead label="Margem" sortKey="margin" sort={sort} onSort={onSort} className="text-right" />
                <TableHead />
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map((s) => (
                <SaleRowLine key={s.id} s={s} onSaved={load} />
              ))}
            </TableBody>
          </Table>
        </div>
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

function SortHead({
  label,
  sortKey,
  sort,
  onSort,
  className,
}: {
  label: string;
  sortKey: SortKey;
  sort: SortState;
  onSort: (key: SortKey) => void;
  className?: string;
}) {
  const active = sort.key === sortKey;
  return (
    <TableHead className={className}>
      <button
        type="button"
        onClick={() => onSort(sortKey)}
        className={`inline-flex items-center gap-1 uppercase transition-colors hover:text-fg ${active ? "text-fg" : ""}`}
      >
        {label}
        <ArrowUpDown className={`h-3 w-3 ${active ? "opacity-100" : "opacity-30"}`} />
      </button>
    </TableHead>
  );
}

// SaleRowLine is one sale, the way it is asked about: which card, when, to whom,
// for how much.
function SaleRowLine({ s, onSaved }: { s: SaleRow; onSaved: () => void }) {
  const [editing, setEditing] = useState(false);
  const order = orderNumber(s.buyer);
  const who = buyerName(s.buyer);

  if (editing) {
    return (
      <TableRow className="hover:bg-transparent">
        <TableCell colSpan={9} className="bg-raised/40">
          <SaleEditor s={s} onCancel={() => setEditing(false)} onSaved={onSaved} />
        </TableCell>
      </TableRow>
    );
  }

  return (
    <TableRow>
      <TableCell className="whitespace-nowrap tabular-nums text-slate-400">
        {s.sellDate ? dayLabel(s.sellDate) : "sem data"}
      </TableCell>
      <TableCell>
        <div className="flex items-center gap-2">
          <CardArt
            set={s.set}
            number={s.number}
            name={s.name}
            game={s.game}
            imageURL={s.imageURL}
            productID={productIDFromTcgURL(s.tcgUrl)}
            className="h-12 w-[34px] shrink-0 rounded"
          />
          <div className="min-w-0">
            <div className="truncate text-slate-100">{s.name}</div>
            <div className="truncate text-[10px] text-slate-500">
              {[s.number, s.set, s.gameName].filter(Boolean).join(" · ")}
            </div>
          </div>
        </div>
      </TableCell>
      <TableCell className="max-w-[16rem]">
        <div className="truncate text-[12px] text-slate-300">
          {who || channelLabels[channelOfSale(s)]}
        </div>
        {order && <div className="text-[10px] text-slate-500">Liga #{order}</div>}
      </TableCell>
      <TableCell className="text-right tabular-nums text-slate-400">{s.qty}</TableCell>
      <TableCell className="text-right font-semibold tabular-nums text-slate-100">{brl2(s.valueBRL)}</TableCell>
      <TableCell className="text-right tabular-nums text-slate-400">{brl2(s.costBRL)}</TableCell>
      <TableCell className={`text-right font-bold tabular-nums ${moneyClass(s.profitBRL)}`}>
        {brl2(s.profitBRL)}
      </TableCell>
      <TableCell className={`text-right tabular-nums ${moneyClass(s.profitBRL)}`}>
        {pctLabel(marginOf(s))}
      </TableCell>
      <TableCell className="text-right">
        <Button variant="ghost" size="sm" onClick={() => setEditing(true)} title="Editar esta venda">
          <Pencil className="h-4 w-4" />
        </Button>
      </TableCell>
    </TableRow>
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
        <TableCell className="text-right font-semibold tabular-nums text-slate-200">{brl2(p.revenueBRL)}</TableCell>
        <TableCell className="text-right tabular-nums text-slate-400">{brl2(p.costBRL)}</TableCell>
        <TableCell className={`text-right font-bold tabular-nums ${moneyClass(p.profitBRL)}`}>
          {brl2(p.profitBRL)}
        </TableCell>
        <TableCell className={`text-right tabular-nums ${moneyClass(p.profitBRL)}`}>{pctLabel(p.marginPct)}</TableCell>
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
                  <PackageItemRow key={s.id} s={s} onSaved={onSaved} />
                ))}
              </TableBody>
            </Table>
          </TableCell>
        </TableRow>
      )}
    </>
  );
}

function PackageItemRow({ s, onSaved }: { s: SaleRow; onSaved: () => void }) {
  const [editing, setEditing] = useState(false);

  if (editing) {
    return (
      <TableRow className="hover:bg-transparent">
        <TableCell colSpan={6} className="px-2 py-2">
          <SaleEditor s={s} onCancel={() => setEditing(false)} onSaved={onSaved} />
        </TableCell>
      </TableRow>
    );
  }

  return (
    <TableRow className="hover:bg-transparent">
      <TableCell className="px-2 py-1">
        <div className="flex items-center gap-2">
          <CardArt
            set={s.set}
            number={s.number}
            name={s.name}
            game={s.game}
            imageURL={s.imageURL}
            productID={productIDFromTcgURL(s.tcgUrl)}
            className="h-12 w-[34px] shrink-0 rounded"
          />
          <div className="min-w-0">
            <div className="truncate text-slate-200">{s.name}</div>
            <div className="truncate text-[10px] text-slate-500">
              {[s.number, s.set, s.gameName, s.kind === "sealed" ? "selado" : s.kind === "accessory" ? "acessório" : "single"]
                .filter(Boolean)
                .join(" · ")}
            </div>
          </div>
        </div>
      </TableCell>
      <TableCell className="px-2 py-1 text-right tabular-nums text-slate-400">{s.qty}</TableCell>
      <TableCell className="px-2 py-1 text-right tabular-nums text-slate-200">{brl2(s.valueBRL)}</TableCell>
      <TableCell className="px-2 py-1 text-right tabular-nums text-slate-400">{brl2(s.costBRL)}</TableCell>
      <TableCell className={`px-2 py-1 text-right font-semibold tabular-nums ${moneyClass(s.profitBRL)}`}>
        {brl2(s.profitBRL)}
      </TableCell>
      <TableCell className="px-2 py-1 text-right">
        <Button variant="ghost" size="sm" onClick={() => setEditing(true)} title="Editar esta venda">
          <Pencil className="h-3.5 w-3.5" />
        </Button>
      </TableCell>
    </TableRow>
  );
}

function SaleEditor({ s, onCancel, onSaved }: { s: SaleRow; onCancel: () => void; onSaved: () => void }) {
  const [qty, setQty] = useState(String(s.qty));
  const [price, setPrice] = useState(String(s.sellPrice ?? s.valueBRL));
  const [date, setDate] = useState(s.sellDate ?? "");
  const [buyer, setBuyer] = useState(s.buyer ?? "");
  const [saving, setSaving] = useState(false);
  const [failed, setFailed] = useState<string | null>(null);

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
      onSaved();
      onCancel();
    } catch (err: unknown) {
      setFailed(err instanceof Error ? err.message : "falha ao salvar");
    } finally {
      setSaving(false);
    }
  };

  return (
    <div>
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
          <Button variant="ghost" size="sm" onClick={onCancel} disabled={saving}>
            <X className="h-4 w-4" />
          </Button>
        </div>
      </div>
      <p className="mt-1 text-[10px] text-slate-500">
        Venda total é o valor do item inteiro (unitário × unidades), sem frete. O comprador define o canal:
        “Liga #nº” marca pedido da Liga, vazio conta como venda sem marca.
      </p>
      {failed && <p className="mt-1 text-[11px] text-rose-300">{failed}</p>}
    </div>
  );
}

function FieldLabel({ children }: { children: React.ReactNode }) {
  return <div className="font-pixel text-[8px] uppercase text-brand-label">{children}</div>;
}
