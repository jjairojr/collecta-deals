import { useEffect, useMemo, useState } from "react";
import { Bar, BarChart, CartesianGrid, Tooltip, XAxis, YAxis } from "recharts";
import { Coins, Receipt, Store, TrendingUp } from "lucide-react";
import { listExpenses, listSales, type Expense, type SaleRow } from "../api";
import { brl0, brl2 } from "../format";
import {
  currentMonth,
  eligibleSales,
  expenseMonths,
  expensesByCategory,
  expensesInRange,
  filterSales,
  groupTotals,
  isUSD,
  kindLabels,
  kindOf,
  kinds,
  monthIndex,
  monthLabel,
  monthRange,
  monthlySeries,
  saleMonths,
  shiftMonth,
  sideLabels,
  sideOf,
  sides,
  totalsOf,
  undatedSales,
  type FinanceFilters,
  type Kind,
  type MonthRow,
  type Side,
} from "../finance";
import EmptyState from "./EmptyState";
import { Chip, Kpi } from "./PortfolioPage";
import { Card } from "./ui/card";
import { Checkbox } from "./ui/checkbox";
import { Input } from "./ui/input";
import { ChartContainer, ChartTooltip, axisTick, chartColors, tooltipCursor } from "./ui/chart";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "./ui/table";
import { ToggleGroup } from "./ui/toggle-group";

const PRESETS = [
  { value: "3", label: "3 meses" },
  { value: "6", label: "6 meses" },
  { value: "12", label: "12 meses" },
  { value: "all", label: "Tudo" },
];

const violet = "#a78bfa";

function pctLabel(value: number): string {
  return `${value >= 0 ? "+" : ""}${Math.round(value)}%`;
}

function moneyClass(value: number): string {
  return value >= 0 ? "text-emerald-300" : "text-rose-300";
}

function FilterLabel({ children }: { children: string }) {
  return (
    <span className="ml-1 font-pixel text-[8px] uppercase text-brand-label first:ml-0">{children}</span>
  );
}

function LegendDot({ color, children }: { color: string; children: string }) {
  return (
    <span className="flex items-center gap-1.5">
      <span className="h-2.5 w-2.5 rounded-[3px] border-2 border-outline" style={{ backgroundColor: color }} />
      {children}
    </span>
  );
}

function MonthTooltip({
  active,
  payload,
  rows,
}: {
  active?: boolean;
  payload?: { payload: MonthRow }[];
  rows: { key: keyof MonthRow; label: string; color: string }[];
}) {
  if (!active || !payload || payload.length === 0) {
    return null;
  }
  const d = payload[0].payload;
  return (
    <ChartTooltip
      title={monthLabel(d.month)}
      rows={rows.map((r) => ({ label: r.label, value: brl2(Number(d[r.key])), color: r.color }))}
    />
  );
}

export default function FinanceiroPage() {
  const [sales, setSales] = useState<SaleRow[]>([]);
  const [expenses, setExpenses] = useState<Expense[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [preset, setPreset] = useState("6");
  const [from, setFrom] = useState(() => shiftMonth(currentMonth(), -5));
  const [to, setTo] = useState(currentMonth);
  const [games, setGames] = useState<string[]>([]);
  const [selectedKinds, setSelectedKinds] = useState<Kind[]>([]);
  const [selectedSides, setSelectedSides] = useState<Side[]>([]);
  const [unmarkedIsLiga, setUnmarkedIsLiga] = useState(true);
  const [hideSymbolic, setHideSymbolic] = useState(true);
  const [includeUSD, setIncludeUSD] = useState(true);

  useEffect(() => {
    Promise.all([listSales(), listExpenses()])
      .then(([s, e]) => {
        setSales(s.sales);
        setExpenses(e.expenses);
        setError(null);
      })
      .catch((err: unknown) =>
        setError(err instanceof Error ? err.message : "falha ao carregar o financeiro"),
      )
      .finally(() => setLoading(false));
  }, []);

  const known = useMemo(() => {
    const months = [...saleMonths(eligibleSales(sales, false)), ...expenseMonths(expenses)].sort();
    return { first: months[0] ?? currentMonth(), last: months[months.length - 1] ?? currentMonth() };
  }, [sales, expenses]);

  const applyPreset = (value: string) => {
    setPreset(value);
    if (value === "all") {
      setFrom(known.first);
      setTo(monthIndex(known.last) > monthIndex(currentMonth()) ? known.last : currentMonth());
      return;
    }
    const end = currentMonth();
    setFrom(shiftMonth(end, -(Number(value) - 1)));
    setTo(end);
  };

  const filters: FinanceFilters = useMemo(
    () => ({
      from,
      to,
      games,
      kinds: selectedKinds,
      sides: selectedSides,
      unmarkedIsLiga,
      hideSymbolic,
      includeUSD,
    }),
    [from, to, games, selectedKinds, selectedSides, unmarkedIsLiga, hideSymbolic, includeUSD],
  );

  const months = useMemo(() => monthRange(from, to), [from, to]);
  const visible = useMemo(() => filterSales(sales, filters), [sales, filters]);
  const totals = useMemo(() => totalsOf(visible), [visible]);
  const bySide = useMemo(() => groupTotals(visible, (s) => sideOf(s, unmarkedIsLiga)), [visible, unmarkedIsLiga]);
  const byKind = useMemo(() => groupTotals(visible, kindOf), [visible]);
  const byGame = useMemo(() => groupTotals(visible, (s) => s.gameName), [visible]);
  const series = useMemo(
    () => monthlySeries(visible, expenses, months, unmarkedIsLiga),
    [visible, expenses, months, unmarkedIsLiga],
  );
  const expenseTotal = useMemo(() => expensesInRange(expenses, months), [expenses, months]);
  const categories = useMemo(() => expensesByCategory(expenses, months), [expenses, months]);

  const undated = useMemo(() => undatedSales(sales, filters), [sales, filters]);
  const usdTotal = useMemo(
    () => visible.filter(isUSD).reduce((sum, s) => sum + s.valueBRL, 0),
    [visible],
  );

  const gameOptions = useMemo(() => {
    const found = new Map<string, string>();
    for (const s of eligibleSales(sales, hideSymbolic, includeUSD)) {
      if (kindOf(s) !== "accessory") {
        found.set(s.game, s.gameName);
      }
    }
    return [...found.entries()].sort((a, b) => a[1].localeCompare(b[1]));
  }, [sales, hideSymbolic, includeUSD]);

  const net = totals.grossBRL - expenseTotal;
  const liga = bySide.get("liga");
  const fora = bySide.get("fora");

  const toggle = <T,>(list: T[], value: T, set: (next: T[]) => void) =>
    set(list.includes(value) ? list.filter((v) => v !== value) : [...list, value]);

  if (loading) {
    return <EmptyState>Carregando o financeiro…</EmptyState>;
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
      <Card className="space-y-3 p-4">
        <div className="flex flex-wrap items-center gap-3">
          <ToggleGroup value={preset} options={PRESETS} onChange={applyPreset} />
          <div className="flex items-center gap-2 text-xs text-slate-400">
            <Input
              type="month"
              value={from}
              max={to}
              onChange={(e) => {
                setFrom(e.target.value);
                setPreset("");
              }}
              className="w-[9.5rem]"
            />
            <span>até</span>
            <Input
              type="month"
              value={to}
              min={from}
              onChange={(e) => {
                setTo(e.target.value);
                setPreset("");
              }}
              className="w-[9.5rem]"
            />
          </div>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          <FilterLabel>Canal</FilterLabel>
          {sides.map((s) => (
            <Chip
              key={s}
              active={selectedSides.length === 0 || selectedSides.includes(s)}
              onClick={() => toggle(selectedSides, s, setSelectedSides)}
            >
              {sideLabels[s]}
            </Chip>
          ))}
          <FilterLabel>Tipo</FilterLabel>
          {kinds.map((k) => (
            <Chip
              key={k}
              active={selectedKinds.length === 0 || selectedKinds.includes(k)}
              onClick={() => toggle(selectedKinds, k, setSelectedKinds)}
            >
              {kindLabels[k]}
            </Chip>
          ))}
          <FilterLabel>Jogo</FilterLabel>
          {gameOptions.map(([id, name]) => (
            <Chip
              key={id}
              active={games.length === 0 || games.includes(id)}
              onClick={() => toggle(games, id, setGames)}
            >
              {name}
            </Chip>
          ))}
        </div>

        <div className="flex flex-wrap items-center gap-4 text-xs text-slate-400">
          <label className="flex cursor-pointer items-center gap-2">
            <Checkbox checked={unmarkedIsLiga} onChange={(e) => setUnmarkedIsLiga(e.target.checked)} />
            Contar baixas sem comprador como Liga
          </label>
          <label className="flex cursor-pointer items-center gap-2">
            <Checkbox checked={hideSymbolic} onChange={(e) => setHideSymbolic(e.target.checked)} />
            Ocultar baixas simbólicas (R$ 1/un.)
          </label>
          <label className="flex cursor-pointer items-center gap-2">
            <Checkbox checked={includeUSD} onChange={(e) => setIncludeUSD(e.target.checked)} />
            Incluir vendas em US$ (TCGplayer), convertidas
          </label>
          <span className="text-[11px] text-slate-500">
            Venda em US$ entra sempre como fora da Liga. Despesa recorrente conta uma vez por mês do período.
          </span>
        </div>

        {undated.length > 0 && (
          <p className="text-[11px] text-amber-300">
            {undated.length} venda{undated.length > 1 ? "s" : ""} sem data (
            {brl2(undated.reduce((sum, s) => sum + s.valueBRL, 0))}) fica
            {undated.length > 1 ? "m" : ""} fora de qualquer período — preencha a data na aba Vendas para
            entrar na conta.
          </p>
        )}
      </Card>

      <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
        <Kpi
          icon={<Coins className="h-5 w-5" />}
          label="Venda total"
          value={brl2(totals.revenueBRL)}
          sub={`${totals.units} un. em ${totals.items} itens`}
        />
        <Kpi
          icon={<Store className="h-5 w-5" />}
          label="Venda Liga"
          value={brl2(liga?.revenueBRL ?? 0)}
          sub={
            totals.revenueBRL > 0
              ? `${Math.round(((liga?.revenueBRL ?? 0) / totals.revenueBRL) * 100)}% do total · lucro ${brl2(liga?.grossBRL ?? 0)}`
              : "sem vendas no período"
          }
        />
        <Kpi
          icon={<Store className="h-5 w-5" />}
          label="Venda fora da Liga"
          value={brl2(fora?.revenueBRL ?? 0)}
          sub={
            totals.revenueBRL > 0
              ? `${Math.round(((fora?.revenueBRL ?? 0) / totals.revenueBRL) * 100)}% do total · lucro ${brl2(fora?.grossBRL ?? 0)}${usdTotal > 0 ? ` · US$ ${brl2(usdTotal)}` : ""}`
              : "sem vendas no período"
          }
        />
        <Kpi
          icon={<TrendingUp className="h-5 w-5" />}
          label="Lucro bruto"
          value={brl2(totals.grossBRL)}
          sub={`margem ${pctLabel(totals.marginPct)} sobre a venda`}
        />
      </div>

      <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
        <Kpi
          icon={<Receipt className="h-5 w-5" />}
          label="Custo das vendas"
          value={brl2(totals.costBRL)}
          sub="o que essas mercadorias custaram"
        />
        <Kpi
          icon={<Receipt className="h-5 w-5" />}
          label="Despesas"
          value={brl2(expenseTotal)}
          sub={`${months.length} ${months.length === 1 ? "mês" : "meses"} no período`}
        />
        <Kpi
          icon={<TrendingUp className="h-5 w-5" />}
          label="Lucro líquido"
          value={brl2(net)}
          sub="lucro bruto menos despesas"
        />
        <Kpi
          icon={<TrendingUp className="h-5 w-5" />}
          label="Margem líquida"
          value={totals.revenueBRL > 0 ? pctLabel((net / totals.revenueBRL) * 100) : "—"}
          sub={
            expenseTotal > 0 && months.length > 0
              ? `despesa média ${brl2(expenseTotal / months.length)}/mês`
              : "sem despesa lançada"
          }
        />
      </div>

      {series.length > 0 && (
        <div className="grid gap-4 xl:grid-cols-2">
          <Card className="p-4">
            <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
              <h2 className="font-display text-base font-extrabold text-fg">Receita, lucro e despesa por mês</h2>
              <div className="flex items-center gap-3 text-[11px] text-slate-400">
                <LegendDot color={chartColors.brand}>Receita</LegendDot>
                <LegendDot color={chartColors.emerald}>Lucro bruto</LegendDot>
                <LegendDot color={chartColors.rose}>Despesas</LegendDot>
              </div>
            </div>
            <ChartContainer height={280}>
              <BarChart data={series} margin={{ top: 4, right: 8, bottom: 4, left: 8 }}>
                <CartesianGrid vertical={false} stroke={chartColors.grid} />
                <XAxis
                  dataKey="month"
                  tick={axisTick}
                  tickLine={false}
                  axisLine={{ stroke: chartColors.grid }}
                  tickFormatter={monthLabel}
                />
                <YAxis tick={axisTick} tickLine={false} axisLine={false} width={70} tickFormatter={brl0} />
                <Tooltip
                  cursor={tooltipCursor}
                  content={
                    <MonthTooltip
                      rows={[
                        { key: "revenueBRL", label: "Receita", color: chartColors.brand },
                        { key: "grossBRL", label: "Lucro bruto", color: chartColors.emerald },
                        { key: "expensesBRL", label: "Despesas", color: chartColors.rose },
                        { key: "netBRL", label: "Líquido", color: violet },
                      ]}
                    />
                  }
                />
                <Bar dataKey="revenueBRL" fill={chartColors.brand} radius={[4, 4, 0, 0]} maxBarSize={30} isAnimationActive={false} />
                <Bar dataKey="grossBRL" fill={chartColors.emerald} radius={[4, 4, 0, 0]} maxBarSize={30} isAnimationActive={false} />
                <Bar dataKey="expensesBRL" fill={chartColors.rose} radius={[4, 4, 0, 0]} maxBarSize={30} isAnimationActive={false} />
              </BarChart>
            </ChartContainer>
          </Card>

          <Card className="p-4">
            <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
              <h2 className="font-display text-base font-extrabold text-fg">Liga × fora da Liga</h2>
              <div className="flex items-center gap-3 text-[11px] text-slate-400">
                <LegendDot color={chartColors.brand}>Liga</LegendDot>
                <LegendDot color={violet}>Fora</LegendDot>
              </div>
            </div>
            <ChartContainer height={280}>
              <BarChart data={series} margin={{ top: 4, right: 8, bottom: 4, left: 8 }}>
                <CartesianGrid vertical={false} stroke={chartColors.grid} />
                <XAxis
                  dataKey="month"
                  tick={axisTick}
                  tickLine={false}
                  axisLine={{ stroke: chartColors.grid }}
                  tickFormatter={monthLabel}
                />
                <YAxis tick={axisTick} tickLine={false} axisLine={false} width={70} tickFormatter={brl0} />
                <Tooltip
                  cursor={tooltipCursor}
                  content={
                    <MonthTooltip
                      rows={[
                        { key: "ligaBRL", label: "Liga", color: chartColors.brand },
                        { key: "foraBRL", label: "Fora da Liga", color: violet },
                        { key: "revenueBRL", label: "Total", color: chartColors.emerald },
                      ]}
                    />
                  }
                />
                <Bar dataKey="ligaBRL" stackId="rev" fill={chartColors.brand} maxBarSize={30} isAnimationActive={false} />
                <Bar dataKey="foraBRL" stackId="rev" fill={violet} radius={[4, 4, 0, 0]} maxBarSize={30} isAnimationActive={false} />
              </BarChart>
            </ChartContainer>
          </Card>
        </div>
      )}

      {visible.length === 0 ? (
        <EmptyState hint="troque o período ou os filtros">Nenhuma venda no recorte atual.</EmptyState>
      ) : (
        <div className="sticker sticker-sm overflow-x-auto rounded-[12px] bg-panel">
          <Table className="min-w-[860px]">
            <TableHeader>
              <TableRow className="hover:bg-transparent">
                <TableHead>Mês</TableHead>
                <TableHead className="text-right">Receita</TableHead>
                <TableHead className="text-right">Liga</TableHead>
                <TableHead className="text-right">Fora</TableHead>
                <TableHead className="text-right">Custo</TableHead>
                <TableHead className="text-right">Lucro bruto</TableHead>
                <TableHead className="text-right">Despesas</TableHead>
                <TableHead className="text-right">Líquido</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {series.map((row) => (
                <TableRow key={row.month}>
                  <TableCell className="whitespace-nowrap font-medium text-slate-200">
                    {monthLabel(row.month)}
                  </TableCell>
                  <TableCell className="text-right font-semibold tabular-nums text-slate-200">
                    {brl2(row.revenueBRL)}
                  </TableCell>
                  <TableCell className="text-right tabular-nums text-slate-400">{brl2(row.ligaBRL)}</TableCell>
                  <TableCell className="text-right tabular-nums text-slate-400">{brl2(row.foraBRL)}</TableCell>
                  <TableCell className="text-right tabular-nums text-slate-400">{brl2(row.costBRL)}</TableCell>
                  <TableCell className={`text-right tabular-nums ${moneyClass(row.grossBRL)}`}>
                    {brl2(row.grossBRL)}
                  </TableCell>
                  <TableCell className="text-right tabular-nums text-rose-300">
                    {row.expensesBRL > 0 ? `-${brl2(row.expensesBRL)}` : brl2(0)}
                  </TableCell>
                  <TableCell className={`text-right font-bold tabular-nums ${moneyClass(row.netBRL)}`}>
                    {brl2(row.netBRL)}
                  </TableCell>
                </TableRow>
              ))}
              <TableRow className="bg-raised/40 hover:bg-raised/40">
                <TableCell className="font-bold text-slate-100">Total</TableCell>
                <TableCell className="text-right font-bold tabular-nums text-slate-100">
                  {brl2(totals.revenueBRL)}
                </TableCell>
                <TableCell className="text-right tabular-nums text-slate-300">
                  {brl2(liga?.revenueBRL ?? 0)}
                </TableCell>
                <TableCell className="text-right tabular-nums text-slate-300">
                  {brl2(fora?.revenueBRL ?? 0)}
                </TableCell>
                <TableCell className="text-right tabular-nums text-slate-300">{brl2(totals.costBRL)}</TableCell>
                <TableCell className={`text-right font-bold tabular-nums ${moneyClass(totals.grossBRL)}`}>
                  {brl2(totals.grossBRL)}
                </TableCell>
                <TableCell className="text-right font-bold tabular-nums text-rose-300">
                  {expenseTotal > 0 ? `-${brl2(expenseTotal)}` : brl2(0)}
                </TableCell>
                <TableCell className={`text-right font-bold tabular-nums ${moneyClass(net)}`}>
                  {brl2(net)}
                </TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </div>
      )}

      <div className="grid gap-4 xl:grid-cols-3">
        <BreakdownCard
          title="Por tipo de produto"
          rows={kinds
            .map((k) => ({ label: kindLabels[k], t: byKind.get(k) }))
            .filter((r) => r.t !== undefined)}
        />
        <BreakdownCard
          title="Por jogo"
          rows={[...byGame.entries()]
            .sort((a, b) => b[1].revenueBRL - a[1].revenueBRL)
            .map(([name, t]) => ({ label: name, t }))}
        />
        <Card className="p-4">
          <h2 className="mb-3 font-display text-base font-extrabold text-fg">Despesas por categoria</h2>
          {categories.length === 0 ? (
            <p className="text-xs text-slate-500">Nenhuma despesa lançada no período.</p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow className="hover:bg-transparent">
                  <TableHead className="px-2">Categoria</TableHead>
                  <TableHead className="px-2 text-right">Total</TableHead>
                  <TableHead className="px-2 text-right">Fixa</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {categories.map((c) => (
                  <TableRow key={c.category}>
                    <TableCell className="px-2 text-slate-200">{c.category}</TableCell>
                    <TableCell className="px-2 text-right tabular-nums text-rose-300">
                      {brl2(c.amountBRL)}
                    </TableCell>
                    <TableCell className="px-2 text-right tabular-nums text-slate-500">
                      {c.recurringBRL > 0 ? brl2(c.recurringBRL) : "—"}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </Card>
      </div>
    </div>
  );
}

function BreakdownCard({
  title,
  rows,
}: {
  title: string;
  rows: { label: string; t?: { units: number; revenueBRL: number; costBRL: number; grossBRL: number; marginPct: number } }[];
}) {
  const filled = rows.filter((r) => r.t);
  return (
    <Card className="p-4">
      <h2 className="mb-3 font-display text-base font-extrabold text-fg">{title}</h2>
      {filled.length === 0 ? (
        <p className="text-xs text-slate-500">Nada no recorte atual.</p>
      ) : (
        <Table>
          <TableHeader>
            <TableRow className="hover:bg-transparent">
              <TableHead className="px-2">Item</TableHead>
              <TableHead className="px-2 text-right">Un.</TableHead>
              <TableHead className="px-2 text-right">Receita</TableHead>
              <TableHead className="px-2 text-right">Lucro</TableHead>
              <TableHead className="px-2 text-right">Mg</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {filled.map((r) => (
              <TableRow key={r.label}>
                <TableCell className="px-2 text-slate-200">{r.label}</TableCell>
                <TableCell className="px-2 text-right tabular-nums text-slate-400">{r.t?.units}</TableCell>
                <TableCell className="px-2 text-right tabular-nums text-slate-200">
                  {brl2(r.t?.revenueBRL ?? 0)}
                </TableCell>
                <TableCell className={`px-2 text-right tabular-nums ${moneyClass(r.t?.grossBRL ?? 0)}`}>
                  {brl2(r.t?.grossBRL ?? 0)}
                </TableCell>
                <TableCell className={`px-2 text-right tabular-nums ${moneyClass(r.t?.grossBRL ?? 0)}`}>
                  {pctLabel(r.t?.marginPct ?? 0)}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}
    </Card>
  );
}
