import { useCallback, useEffect, useId, useMemo, useState } from "react";
import EmptyState from "./EmptyState";
import {
  Plus,
  Trash2,
  Wallet,
  Coins,
  Banknote,
  ExternalLink,
  Share2,
  Package,
  Pencil,
  Truck,
  Check,
  Search,
  Download,
  ChevronDown,
  ChevronsUpDown,
  ArrowUp,
  ArrowDown,
  X,
} from "lucide-react";
import {
  createTrade,
  deleteTrade,
  gameHasDeals,
  getGame,
  getPortfolio,
  getQuote,
  productIDFromTcgURL,
  sellTrade,
  updateTrade,
  type PortfolioResponse,
  type QuoteMatch,
  type TradeView,
} from "../api";
import { brl0, usd, dayLabel, timeAgo, fullStamp } from "../format";
import { toCSV, downloadCSV } from "../csv";

// In BR-only games (no US deals pipeline) there is no US price: the portfolio
// values holdings in BRL and the API returns market figures already in reais
// (fxRate = 1). Deals-enabled games keep their USD (TCGplayer) valuation.
// isBRGame drives the labels/format.
const isBRGame = () => !gameHasDeals(getGame());

// marketMoney formats a market figure that is USD for deals games but BRL elsewhere.
function marketMoney(v: number): string {
  return isBRGame() ? brl0(v) : usd(v);
}

const VARIANT_SUGGESTIONS: Record<string, string[]> = {
  pokemon: ["Normal", "Reverse Holo", "Holo", "Promo"],
  onepiece: ["Normal", "Alternate Art", "Parallel", "SP", "Manga", "Promo"],
  riftbound: ["Normal", "Alternate Art", "Overnumbered", "Signature", "Promo"],
};

const DEFAULT_VARIANTS = [
  "Normal",
  "Foil",
  "Reverse Holo",
  "Holo",
  "Alternate Art",
  "Parallel",
  "SP",
  "Promo",
];
import CardArt from "./CardArt";
import ShareList from "./ShareList";
import { Button } from "./ui/button";
import { Card } from "./ui/card";
import { Input } from "./ui/input";
import { Badge } from "./ui/badge";
import { ToggleGroup } from "./ui/toggle-group";

function cleanName(n: string): string {
  return n.replace(/\s*\([^)]*\)\s*$/, "");
}

type Section = "singles" | "sealed" | "accessories";

const SECTION_KIND: Record<Section, "sealed" | "accessory" | undefined> = {
  singles: undefined,
  sealed: "sealed",
  accessories: "accessory",
};

interface SectionSummary {
  holdings: number;
  investedBRL: number;
  sold: number;
  costOfSoldBRL: number;
  proceedsBRL: number;
}

function sectionSummary(list: TradeView[]): SectionSummary {
  const s: SectionSummary = {
    holdings: 0,
    investedBRL: 0,
    sold: 0,
    costOfSoldBRL: 0,
    proceedsBRL: 0,
  };
  for (const t of list) {
    if (t.realized) {
      s.sold += 1;
      s.costOfSoldBRL += t.costBRL;
      s.proceedsBRL += t.valueBRL;
    } else {
      s.holdings += 1;
      s.investedBRL += t.costBRL;
    }
  }
  return s;
}

type SortKey = "name" | "cost" | "value" | "added";
interface SortState {
  key: SortKey;
  dir: "asc" | "desc";
}

function matchesQuery(t: TradeView, q: string): boolean {
  if (!q) {
    return true;
  }
  return `${t.name} ${t.number} ${t.store ?? ""}`.toLowerCase().includes(q);
}

function sortValue(t: TradeView, key: SortKey): number | string {
  switch (key) {
    case "name":
      return cleanName(t.name).toLowerCase();
    case "cost":
      return t.costBRL;
    case "value":
      return t.valueBRL;
    case "added":
      return Date.parse(t.createdAt) || 0;
  }
}

function sortTrades(list: TradeView[], sort: SortState): TradeView[] {
  const dir = sort.dir === "asc" ? 1 : -1;
  return [...list].sort((a, b) => {
    const av = sortValue(a, sort.key);
    const bv = sortValue(b, sort.key);
    if (typeof av === "string" && typeof bv === "string") {
      return av.localeCompare(bv) * dir;
    }
    if (typeof av === "number" && typeof bv === "number") {
      return (av - bv) * dir;
    }
    return 0;
  });
}

const CSV_HEADERS = [
  "status",
  "number",
  "name",
  "set",
  "condition",
  "qty",
  "store",
  "buyDate",
  "delivered",
  "costBRL",
  "soldBRL",
  "sellDate",
  "sellPrice",
  "sellCurrency",
  "buyer",
];

function round2(n: number): number {
  return Math.round(n * 100) / 100;
}

function csvRows(list: TradeView[]): (string | number)[][] {
  return list.map((t) => [
    t.realized ? "sold" : "held",
    t.number,
    t.name,
    t.set,
    t.condition ?? "",
    t.qty,
    t.store ?? "",
    t.buyDate ?? "",
    t.realized ? "" : t.delivered ? "yes" : "no",
    round2(t.costBRL),
    t.realized ? round2(t.valueBRL) : "",
    t.sellDate ?? "",
    t.sellPrice != null ? round2(t.sellPrice) : "",
    t.sellCurrency ?? "",
    t.buyer ?? "",
  ]);
}

type QuickFilter = "all" | "transit" | "delivered";
type GroupBy = "set" | "store" | "none";

// The page shows one list at a time: what is still in stock or what was sold.
// Stacking both meant scrolling past every holding to reach the sales.
type Tab = "holding" | "sold";

const QUICK: { v: QuickFilter; label: string }[] = [
  { v: "all", label: "Todas" },
  { v: "transit", label: "Em trânsito" },
  { v: "delivered", label: "Entregues" },
];

function passesQuick(t: TradeView, f: QuickFilter): boolean {
  switch (f) {
    case "transit":
      return !t.delivered;
    case "delivered":
      return Boolean(t.delivered);
    default:
      return true;
  }
}

interface Group {
  key: string;
  trades: TradeView[];
  count: number;
  invested: number;
  soldBRL: number;
}

function groupKeyOf(t: TradeView, by: GroupBy): string {
  if (by === "store") {
    return t.store?.trim() || "No store";
  }
  return t.set?.trim() || "—";
}

function groupTrades(list: TradeView[], by: GroupBy): Group[] {
  const map = new Map<string, TradeView[]>();
  for (const t of list) {
    const k = groupKeyOf(t, by);
    const arr = map.get(k);
    if (arr) {
      arr.push(t);
    } else {
      map.set(k, [t]);
    }
  }
  const groups: Group[] = [];
  for (const [key, trades] of map) {
    let count = 0;
    let invested = 0;
    let soldBRL = 0;
    for (const t of trades) {
      count += t.qty;
      invested += t.costBRL;
      if (t.realized) {
        soldBRL += t.valueBRL;
      }
    }
    groups.push({ key, trades, count, invested, soldBRL });
  }
  groups.sort((a, b) => b.invested - a.invested);
  return groups;
}

// lockedSection pins the page to one section and drops the section switcher — how
// the sidebar's Acessórios entry opens straight into the accessories ledger.
export default function PortfolioPage({
  lockedSection,
}: {
  lockedSection?: Section;
}) {
  const [section, setSection] = useState<Section>(lockedSection ?? "singles");
  const [tab, setTab] = useState<Tab>("holding");
  const [data, setData] = useState<PortfolioResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [adding, setAdding] = useState(false);
  const [sharing, setSharing] = useState(false);
  const [query, setQuery] = useState("");
  const [sort, setSort] = useState<SortState>({ key: "added", dir: "desc" });
  const [quick, setQuick] = useState<QuickFilter>("all");
  const [groupBy, setGroupBy] = useState<GroupBy>("none");
  const [expanded, setExpanded] = useState<Set<string>>(new Set());

  const load = useCallback(async () => {
    try {
      const r = await getPortfolio(90);
      setData(r);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "failed to load portfolio");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const refresh = load;

  const trades = data?.trades ?? [];
  const active = useMemo(
    () => trades.filter((t) => (t.kind || undefined) === SECTION_KIND[section]),
    [trades, section],
  );
  const summary = useMemo(() => sectionSummary(active), [active]);

  const q = query.trim().toLowerCase();
  const holdings = useMemo(
    () =>
      sortTrades(
        active.filter((t) => !t.realized && matchesQuery(t, q)),
        sort,
      ),
    [active, q, sort],
  );
  const sold = useMemo(
    () =>
      sortTrades(
        active.filter((t) => t.realized && matchesQuery(t, q)),
        sort,
      ),
    [active, q, sort],
  );
  const visibleHoldings = useMemo(
    () => holdings.filter((t) => passesQuick(t, quick)),
    [holdings, quick],
  );
  const holdingsInvested = useMemo(
    () => visibleHoldings.reduce((s, t) => s + t.costBRL, 0),
    [visibleHoldings],
  );
  const soldReceived = useMemo(
    () => sold.reduce((s, t) => s + t.valueBRL, 0),
    [sold],
  );
  const rows = tab === "sold" ? sold : visibleHoldings;
  const groups = useMemo(
    () => (groupBy === "none" ? [] : groupTrades(rows, groupBy)),
    [rows, groupBy],
  );
  const quickCounts = useMemo(
    () => ({
      all: holdings.length,
      transit: holdings.filter((t) => !t.delivered).length,
      delivered: holdings.filter((t) => Boolean(t.delivered)).length,
    }),
    [holdings],
  );
  const isSealed = section === "sealed";
  const isManual = section !== "singles";
  const anyExpanded = expanded.size > 0;
  const filtered = Boolean(q) || (tab === "holding" && quick !== "all");
  const emptyList =
    tab === "sold"
      ? filtered
        ? "Nenhuma venda bate com a busca."
        : "Nenhuma venda registrada ainda."
      : filtered
        ? `Nenhum${isManual ? " produto bate" : "a carta bate"} com os filtros.`
        : `Nada em estoque agora.`;

  const onSort = useCallback((key: SortKey) => {
    setSort((s) =>
      s.key === key
        ? { key, dir: s.dir === "asc" ? "desc" : "asc" }
        : { key, dir: "desc" },
    );
  }, []);

  const toggleGroup = useCallback((key: string) => {
    setExpanded((s) => {
      const next = new Set(s);
      if (next.has(key)) {
        next.delete(key);
      } else {
        next.add(key);
      }
      return next;
    });
  }, []);

  const changeGroupBy = useCallback((g: GroupBy) => {
    setGroupBy(g);
    setExpanded(new Set());
  }, []);

  const expandAll = useCallback(() => {
    setExpanded(new Set(groups.map((g) => g.key)));
  }, [groups]);

  const collapseAll = useCallback(() => setExpanded(new Set()), []);

  const exportCSV = useCallback(() => {
    const date = new Date().toISOString().slice(0, 10);
    downloadCSV(
      `portfolio-${getGame()}-${section}-${date}.csv`,
      toCSV(CSV_HEADERS, csvRows([...visibleHoldings, ...sold])),
    );
  }, [visibleHoldings, sold, section]);

  return (
    <div className="space-y-6">
      {!lockedSection && (
        <div className="flex flex-wrap items-center justify-end gap-4">
          <ToggleGroup
            value={section}
            onChange={(v) => {
              setSection(
                v === "sealed"
                  ? "sealed"
                  : v === "accessories"
                    ? "accessories"
                    : "singles",
              );
              setTab("holding");
              setAdding(false);
              setSharing(false);
            }}
            options={[
              { value: "singles", label: "Cartas" },
              { value: "sealed", label: "Selados" },
              { value: "accessories", label: "Acessórios" },
            ]}
          />
        </div>
      )}

      {data && (
        <div className="grid grid-cols-2 gap-3 lg:grid-cols-3">
          <Kpi
            icon={<Wallet className="h-5 w-5" />}
            label="Investido (em estoque)"
            value={brl0(summary.investedBRL)}
            sub={`${summary.holdings} ${isManual ? "produtos" : "cartas"}`}
          />
          <Kpi
            icon={<Banknote className="h-5 w-5" />}
            label="Investido total"
            value={brl0(summary.investedBRL + summary.costOfSoldBRL)}
            sub="em estoque + vendido, desde sempre"
          />
          <Kpi
            icon={<Coins className="h-5 w-5" />}
            label="Vendido"
            value={brl0(summary.proceedsBRL)}
            sub={`${summary.sold} ${summary.sold === 1 ? "venda" : "vendas"}`}
          />
        </div>
      )}

      {error && (
        <div className="rounded-[14px] border-2 border-rose-900/50 bg-rose-950/30 px-4 py-3 text-sm text-rose-200">
          {error}
        </div>
      )}

      {!loading && active.length > 0 && (
        <div className="flex flex-wrap items-center gap-3 border-b-[3px] border-outline pb-3">
          <ToggleGroup
            value={tab}
            onChange={(v) => setTab(v === "sold" ? "sold" : "holding")}
            options={[
              {
                value: "holding",
                label: (
                  <>
                    Em estoque <span className="opacity-60">· {holdings.length}</span>
                  </>
                ),
              },
              {
                value: "sold",
                label: (
                  <>
                    {isManual ? "Vendidos" : "Vendidas"}{" "}
                    <span className="opacity-60">· {sold.length}</span>
                  </>
                ),
              },
            ]}
          />
          <span
            className={`ml-auto tabular-nums text-xs font-medium ${tab === "sold" ? "text-emerald-300" : "text-slate-400"}`}
          >
            {tab === "sold"
              ? `${brl0(soldReceived)} recebidos`
              : `${brl0(holdingsInvested)} investidos`}
          </span>
        </div>
      )}

      <div className="flex flex-wrap items-center gap-2">
        <div className="relative w-full min-w-[200px] max-w-xs sm:w-auto sm:flex-1">
          <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-500" />
          <Input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder={
              isManual
                ? "Buscar produto, loja…"
                : "Buscar carta, número, loja…"
            }
            className="w-full pl-9 pr-8"
          />
          {query && (
            <button
              onClick={() => setQuery("")}
              className="absolute right-2 top-1/2 -translate-y-1/2 rounded p-0.5 text-slate-500 hover:text-slate-300"
              title="Limpar busca"
            >
              <X className="h-4 w-4" />
            </button>
          )}
        </div>
        <div className="ml-auto flex items-center gap-2">
          <Button
            variant="outline"
            onClick={exportCSV}
            disabled={active.length === 0}
            title="Baixar esta lista como planilha"
          >
            <Download /> Exportar CSV
          </Button>
          {!isManual && (
            <Button
              variant="outline"
              onClick={() => setSharing(true)}
              disabled={holdings.length === 0}
              title="Montar uma lista para mandar pro comprador"
            >
              <Share2 /> Lista p/ cliente
            </Button>
          )}
          <Button onClick={() => setAdding((a) => !a)}>
            <Plus />{" "}
            {adding
              ? "Fechar"
              : isSealed
                ? "Add selado"
                : section === "accessories"
                  ? "Add acessório"
                  : "Add compra"}
          </Button>
        </div>
      </div>

      {!loading && active.length > 0 && (
        <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
          {tab === "holding" && (
            <div className="flex flex-wrap items-center gap-1.5">
              {QUICK.map((qf) => (
                <Chip
                  key={qf.v}
                  active={quick === qf.v}
                  onClick={() => setQuick(qf.v)}
                >
                  {qf.label}{" "}
                  <span className="opacity-60">{quickCounts[qf.v]}</span>
                </Chip>
              ))}
            </div>
          )}
          <div className="ml-auto flex items-center gap-3">
            <div className="flex items-center gap-2">
              <span className="text-[10px] font-medium uppercase tracking-wide text-slate-500">
                Agrupar por
              </span>
              <ToggleGroup
                value={groupBy}
                onChange={(v) =>
                  changeGroupBy(
                    v === "store" ? "store" : v === "none" ? "none" : "set",
                  )
                }
                options={[
                  { value: "set", label: "Set" },
                  { value: "store", label: "Loja" },
                  { value: "none", label: "Nenhum" },
                ]}
              />
            </div>
            {groupBy !== "none" && groups.length > 0 && (
              <button
                onClick={anyExpanded ? collapseAll : expandAll}
                className="text-xs font-medium text-sky-300 hover:text-sky-200"
              >
                {anyExpanded ? "Recolher tudo" : "Abrir tudo"}
              </button>
            )}
          </div>
        </div>
      )}

      {adding &&
        (isManual ? (
          <AddManualForm
            kind={isSealed ? "sealed" : "accessory"}
            onAdded={() => {
              setAdding(false);
              refresh();
            }}
          />
        ) : (
          <AddTradeForm
            fxRate={data?.fxRate ?? 0}
            onAdded={() => {
              setAdding(false);
              refresh();
            }}
          />
        ))}

      {sharing && !isManual && holdings.length > 0 && (
        <ShareList
          holdings={holdings}
          fxRate={data?.fxRate ?? 0}
          onClose={() => setSharing(false)}
        />
      )}

      {loading ? (
        <Panel>Carregando portfólio…</Panel>
      ) : active.length === 0 ? (
        <Panel>
          {isSealed
            ? "Nenhum selado ainda. Clique em “Add selado” para lançar a primeira caixa."
            : section === "accessories"
              ? "Nenhum acessório ainda. Clique em “Add acessório” para lançar o primeiro produto."
              : "Nenhuma compra ainda. Clique em “Add compra” para lançar a primeira."}
        </Panel>
      ) : groupBy === "none" ? (
        isManual ? (
          <ManualTable
            trades={rows}
            onChanged={refresh}
            empty={emptyList}
            sort={sort}
            onSort={onSort}
          />
        ) : (
          <TradeTable
            trades={rows}
            onChanged={refresh}
            empty={emptyList}
            sort={sort}
            onSort={onSort}
          />
        )
      ) : groups.length === 0 ? (
        <Panel>{emptyList}</Panel>
      ) : (
        <div className="space-y-2">
          {groups.map((g) => (
            <GroupBlock
              key={g.key}
              g={g}
              manual={isManual}
              open={expanded.has(g.key)}
              onToggle={() => toggleGroup(g.key)}
              sort={sort}
              onSort={onSort}
              onChanged={refresh}
            />
          ))}
        </div>
      )}
    </div>
  );
}

export function Kpi({
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
    <div className="sticker flex items-center gap-3 rounded-[14px] bg-panel p-4">
      <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-[8px] border-2 border-outline bg-brand text-white">
        {icon}
      </div>
      <div className="min-w-0">
        <div className="font-pixel text-[8px] uppercase leading-[1.5] text-brand-label">
          {label}
        </div>
        <div className="mt-1 truncate text-lg font-bold tabular-nums text-fg xl:text-xl">
          {value}
        </div>
        {sub && (
          <div className="truncate text-[11px] text-slate-500">{sub}</div>
        )}
      </div>
    </div>
  );
}

export function Chip({
  active,
  onClick,
  children,
}: {
  active: boolean;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      onClick={onClick}
      className={`rounded-[8px] border-2 border-outline px-3 py-1 text-xs font-bold transition-colors ${
        active
          ? "bg-sky-500/20 text-sky-200"
          : "bg-panel text-slate-400 hover:text-slate-200"
      }`}
    >
      {children}
    </button>
  );
}

function GroupBlock({
  g,
  manual,
  open,
  onToggle,
  sort,
  onSort,
  onChanged,
}: SortableProps & {
  g: Group;
  manual: boolean;
  open: boolean;
  onToggle: () => void;
  onChanged: () => void;
}) {
  const noun = manual
    ? g.count === 1
      ? "item"
      : "itens"
    : g.count === 1
      ? "carta"
      : "cartas";
  return (
    <div className="overflow-hidden rounded-[14px] border-[3px] border-outline bg-panel">
      <button
        onClick={onToggle}
        className="flex w-full items-center gap-3 px-3 py-2.5 text-left hover:bg-raised/70"
      >
        <ChevronDown
          className={`h-4 w-4 shrink-0 text-slate-500 transition-transform ${open ? "" : "-rotate-90"}`}
        />
        <span className="truncate font-semibold text-slate-100">{g.key}</span>
        <span className="shrink-0 text-xs text-slate-500">
          {g.count} {noun}
        </span>
        <div className="ml-auto flex shrink-0 items-center gap-3 tabular-nums text-xs sm:gap-5">
          <span className="text-slate-500">
            inv <span className="text-slate-300">{brl0(g.invested)}</span>
          </span>
          {g.soldBRL > 0 && (
            <span className="text-slate-500">
              vendido{" "}
              <span className="text-emerald-300">{brl0(g.soldBRL)}</span>
            </span>
          )}
        </div>
      </button>
      {open && (
        <div className="border-t-2 border-outline">
          {manual ? (
            <ManualTable
              trades={g.trades}
              onChanged={onChanged}
              empty=""
              sort={sort}
              onSort={onSort}
              bare
            />
          ) : (
            <TradeTable
              trades={g.trades}
              onChanged={onChanged}
              empty=""
              sort={sort}
              onSort={onSort}
              bare
            />
          )}
        </div>
      )}
    </div>
  );
}

interface SortableProps {
  sort: SortState;
  onSort: (key: SortKey) => void;
}

function SortableTh({
  label,
  sortKey,
  sort,
  onSort,
  align = "right",
}: SortableProps & {
  label: string;
  sortKey: SortKey;
  align?: "left" | "right";
}) {
  const activeCol = sort.key === sortKey;
  return (
    <th
      className={`px-3 py-2 font-bold ${align === "right" ? "text-right" : "text-left"}`}
    >
      <button
        type="button"
        onClick={() => onSort(sortKey)}
        className={`inline-flex items-center gap-1 hover:text-slate-300 ${align === "right" ? "flex-row-reverse" : ""} ${activeCol ? "text-slate-300" : ""}`}
      >
        {label}
        {activeCol ? (
          sort.dir === "asc" ? (
            <ArrowUp className="h-3 w-3" />
          ) : (
            <ArrowDown className="h-3 w-3" />
          )
        ) : (
          <ChevronsUpDown className="h-3 w-3 opacity-40" />
        )}
      </button>
    </th>
  );
}

function TradeTable({
  trades,
  onChanged,
  empty,
  sort,
  onSort,
  bare,
}: SortableProps & {
  trades: TradeView[];
  onChanged: () => void;
  empty: string;
  bare?: boolean;
}) {
  if (trades.length === 0) {
    return empty ? <Panel>{empty}</Panel> : null;
  }
  return (
    <div
      className={
        bare
          ? "overflow-x-auto"
          : "sticker sticker-sm overflow-x-auto rounded-[12px] bg-panel"
      }
    >
      <table className="w-full min-w-[640px] text-sm">
        <thead>
          <tr className="font-pixel border-b-2 border-outline bg-raised text-left text-[8px] uppercase text-brand-label">
            <SortableTh
              label="Carta"
              sortKey="name"
              sort={sort}
              onSort={onSort}
              align="left"
            />
            <SortableTh
              label="Custo"
              sortKey="cost"
              sort={sort}
              onSort={onSort}
            />
            <SortableTh
              label="Vendido"
              sortKey="value"
              sort={sort}
              onSort={onSort}
            />
            <SortableTh
              label="Lançado"
              sortKey="added"
              sort={sort}
              onSort={onSort}
            />
            <th className="px-3 py-2" />
          </tr>
        </thead>
        <tbody>
          {trades.map((t) => (
            <TradeRow key={t.id} t={t} onChanged={onChanged} />
          ))}
        </tbody>
      </table>
    </div>
  );
}

function DeliveryToggle({
  t,
  onChanged,
}: {
  t: TradeView;
  onChanged: () => void;
}) {
  const [saving, setSaving] = useState(false);
  const toggle = async () => {
    setSaving(true);
    try {
      await updateTrade(t.id, { ...t, delivered: !t.delivered });
      onChanged();
    } finally {
      setSaving(false);
    }
  };
  return (
    <button
      onClick={toggle}
      disabled={saving}
      title={
        t.delivered
          ? "Entregue — clique para marcar em trânsito"
          : "Em trânsito — clique para marcar entregue"
      }
      className={`flex items-center gap-1 rounded-[8px] border-2 border-outline px-2 py-1 text-xs font-bold disabled:opacity-50 ${
        t.delivered
          ? "bg-emerald-500/10 text-emerald-300 hover:bg-emerald-500/20"
          : "bg-amber-500/10 text-amber-300 hover:bg-amber-500/20"
      }`}
    >
      {t.delivered ? (
        <Check className="h-3 w-3" />
      ) : (
        <Truck className="h-3 w-3" />
      )}
      {t.delivered ? "Entregue" : "Em trânsito"}
    </button>
  );
}

// SoldCell shows what a sale brought in; holdings have nothing to show here.
function SoldCell({ t }: { t: TradeView }) {
  if (!t.realized) {
    return <td className="px-3 py-2 text-right text-slate-600">—</td>;
  }
  return (
    <td className="px-3 py-2 text-right tabular-nums text-emerald-300">
      {brl0(t.valueBRL)}
      <div className="text-[10px] text-slate-500">
        {t.sellDate ? dayLabel(t.sellDate) : "vendido"}
      </div>
    </td>
  );
}

function TradeRow({
  t,
  onChanged,
}: {
  t: TradeView;
  onChanged: () => void;
}) {
  const [selling, setSelling] = useState(false);
  const [editing, setEditing] = useState(false);
  return (
    <>
      <tr
        className={`border-b-2 border-outline/15 last:border-0 hover:bg-raised/70 ${t.realized ? "opacity-60" : ""}`}
      >
        <td className="px-3 py-2">
          <div className="flex items-center gap-2">
            <CardArt
              set={t.set}
              number={t.number}
              name={t.name}
              productID={productIDFromTcgURL(t.tcgUrl)}
              imageURL={t.imageURL}
              className="h-12 w-[34px] shrink-0 rounded"
            />
            <div className="min-w-0">
              <div
                className="truncate font-medium text-slate-100"
                title={t.name}
              >
                {cleanName(t.name) || t.number}
              </div>
              <div className="font-mono text-[10px] text-slate-500">
                {t.number}
                {t.variant ? ` · ${t.variant}` : ""}
                {t.qty > 1 ? ` ·×${t.qty}` : ""}
                {t.store ? ` · ${t.store}` : ""}
              </div>
            </div>
          </div>
        </td>
        <td className="px-3 py-2 text-right tabular-nums text-slate-300">
          {brl0(t.costBRL)}
        </td>
        <SoldCell t={t} />
        <AddedCell t={t} />
        <td className="px-3 py-2">
          <div className="flex items-center justify-end gap-1">
            {t.tcgUrl && (
              <a
                href={t.tcgUrl}
                target="_blank"
                rel="noreferrer"
                className="flex items-center gap-1 rounded-[8px] border-2 border-outline bg-sky-500/10 px-2 py-1 text-xs font-bold text-sky-300 hover:bg-sky-500/20"
                title="See on TCGplayer"
              >
                TCG <ExternalLink className="h-3 w-3" />
              </a>
            )}
            {t.realized ? (
              <Badge variant="emerald">vendido</Badge>
            ) : (
              <>
                <DeliveryToggle t={t} onChanged={onChanged} />
                <button
                  onClick={() => {
                    setEditing((e) => !e);
                    setSelling(false);
                  }}
                  className="flex items-center gap-1 rounded-[8px] border-2 border-outline bg-panel px-2 py-1 text-xs font-bold text-slate-200 hover:bg-raised"
                  title="Editar o que você pagou e os detalhes da compra"
                >
                  <Pencil className="h-3 w-3" /> Editar
                </button>
                <button
                  onClick={() => {
                    setSelling((s) => !s);
                    setEditing(false);
                  }}
                  className="rounded-[8px] border-2 border-outline bg-panel px-2 py-1 text-xs font-bold text-slate-200 hover:bg-raised"
                >
                  Vender
                </button>
              </>
            )}
            <button
              onClick={() => {
                if (confirm("Excluir este lançamento?")) {
                  deleteTrade(t.id).then(onChanged);
                }
              }}
              className="rounded-[8px] border-2 border-outline bg-panel p-1.5 text-slate-400 hover:bg-rose-950/40 hover:text-rose-300"
              title="Excluir"
            >
              <Trash2 className="h-3.5 w-3.5" />
            </button>
          </div>
        </td>
      </tr>
      {editing && (
        <tr className="bg-slate-900/60">
          <td colSpan={5} className="px-3 py-3">
            <EditTradeForm
              t={t}
              onDone={() => {
                setEditing(false);
                onChanged();
              }}
            />
          </td>
        </tr>
      )}
      {selling && (
        <tr className="bg-slate-900/60">
          <td colSpan={5} className="px-3 py-3">
            <SellForm
              t={t}
              onDone={() => {
                setSelling(false);
                onChanged();
              }}
            />
          </td>
        </tr>
      )}
    </>
  );
}

// AddedCell shows when a trade was logged, compact with the exact stamp on hover.
function AddedCell({ t }: { t: TradeView }) {
  return (
    <td
      className="px-3 py-2 text-right text-xs tabular-nums text-slate-500"
      title={fullStamp(t.createdAt)}
    >
      {timeAgo(t.createdAt)}
    </td>
  );
}

function SellForm({ t, onDone }: { t: TradeView; onDone: () => void }) {
  const [qty, setQty] = useState(String(t.qty));
  const [priceEach, setPriceEach] = useState("");
  const [currency, setCurrency] = useState<"BRL" | "USD">("BRL");
  const [date, setDate] = useState("");
  const [buyer, setBuyer] = useState("");
  const [saving, setSaving] = useState(false);

  const sellQty = Math.min(Math.max(Number(qty) || 0, 1), t.qty);
  const each = Number(priceEach) || 0;
  const total = each * sellQty;
  const money = currency === "USD" ? usd : brl0;
  const partial = sellQty < t.qty;

  const submit = async () => {
    setSaving(true);
    try {
      await sellTrade(t.id, {
        qty: sellQty,
        sellPrice: total,
        sellCurrency: currency,
        sellDate: date,
        buyer,
      });
      onDone();
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="flex flex-wrap items-end gap-2">
      {t.qty > 1 && (
        <Field label={`Qty (of ${t.qty})`}>
          <Input
            type="number"
            min={1}
            max={t.qty}
            value={qty}
            onChange={(e) => setQty(e.target.value)}
            className="w-20"
          />
        </Field>
      )}
      <Field label={t.qty > 1 ? "Price each" : "Sell price"}>
        <Input
          type="number"
          value={priceEach}
          onChange={(e) => setPriceEach(e.target.value)}
          className="w-32"
          placeholder="0,00"
        />
      </Field>
      <Field label="Currency">
        <ToggleGroup
          value={currency}
          onChange={(v) => setCurrency(v === "USD" ? "USD" : "BRL")}
          options={[
            { value: "BRL", label: "R$" },
            { value: "USD", label: "US$" },
          ]}
        />
      </Field>
      <Field label="Date">
        <Input
          type="date"
          value={date}
          onChange={(e) => setDate(e.target.value)}
          className="w-40"
        />
      </Field>
      <Field label="Buyer (optional)">
        <Input
          value={buyer}
          onChange={(e) => setBuyer(e.target.value)}
          className="w-40"
          placeholder="P2P buyer"
        />
      </Field>
      {t.qty > 1 && each > 0 && (
        <div className="pb-2 text-xs text-slate-400">
          Total{" "}
          <span className="font-semibold text-slate-200">{money(total)}</span>
          {partial ? ` · ${t.qty - sellQty} stay holding` : ""}
        </div>
      )}
      <Button onClick={submit} disabled={saving || !each}>
        {saving ? "Saving…" : partial ? `Sell ${sellQty}` : "Mark sold"}
      </Button>
    </div>
  );
}

function AddTradeForm({
  fxRate,
  onAdded,
}: {
  fxRate: number;
  onAdded: () => void;
}) {
  const [query, setQuery] = useState("");
  const [matches, setMatches] = useState<QuoteMatch[]>([]);
  const [number, setNumber] = useState("");
  const [name, setName] = useState("");
  const [set, setSet] = useState("");
  const [refUSD, setRefUSD] = useState("");
  const [buyBRL, setBuyBRL] = useState("");
  const [shippingBRL, setShippingBRL] = useState("");
  const [qty, setQty] = useState("1");
  const [variant, setVariant] = useState("");
  const [condition, setCondition] = useState("NM");
  const [store, setStore] = useState("");
  const [buyDate, setBuyDate] = useState("");
  const [imageURL, setImageURL] = useState("");
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (query.trim().length < 2) {
      setMatches([]);
      return;
    }
    let current = true;
    const h = window.setTimeout(() => {
      getQuote(query)
        .then((r) => current && setMatches(r.matches))
        .catch(() => current && setMatches([]));
    }, 250);
    return () => {
      current = false;
      window.clearTimeout(h);
    };
  }, [query]);

  const pick = (m: QuoteMatch) => {
    setNumber(m.number);
    setName(m.name);
    setSet(m.set);
    setVariant(m.variant ?? "");
    setRefUSD(String(Math.round(m.marketUSD * 100) / 100));
    setQuery(`${m.number} · ${cleanName(m.name)}`);
    setMatches([]);
  };

  const submit = async () => {
    setSaving(true);
    try {
      await createTrade({
        number,
        name,
        set,
        variant,
        condition,
        qty: Number(qty) || 1,
        buyBRL: Number(buyBRL) || 0,
        shippingBRL: Number(shippingBRL) || 0,
        refUSD: Number(refUSD) || 0,
        store,
        buyDate,
        imageURL: imageURL.trim(),
        status: "holding",
      });
      onAdded();
    } finally {
      setSaving(false);
    }
  };

  const valid = (number || name) && Number(buyBRL) > 0;

  return (
    <Card className="space-y-3 p-4">
      <div className="relative">
        <Field label="Find card (name or number)">
          <Input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="e.g. Luffy or OP16-080"
            className="w-full"
          />
        </Field>
        {matches.length > 0 && (
          <ul className="absolute z-10 mt-1 max-h-72 w-full overflow-y-auto sticker sticker-sm rounded-[10px] bg-panel">
            {matches.map((m) => (
              <li key={`${m.number}-${m.name}`}>
                <button
                  type="button"
                  onClick={() => pick(m)}
                  className="flex w-full items-center justify-between gap-3 px-3 py-2 text-left text-sm hover:bg-raised"
                >
                  <span className="flex min-w-0 items-center gap-2">
                    <CardArt
                      set={m.set}
                      number={m.number}
                      name={m.name}
                      productID={m.productID}
                      className="h-14 w-[40px] shrink-0 rounded"
                    />
                    <span className="min-w-0">
                      <span className="block truncate text-slate-100">
                        {cleanName(m.name)}
                      </span>
                      <span className="font-mono text-[11px] text-slate-500">
                        {m.number}
                      </span>
                    </span>
                  </span>
                  <span className="shrink-0 tabular-nums text-emerald-300">
                    {marketMoney(m.marketUSD)}
                  </span>
                </button>
              </li>
            ))}
          </ul>
        )}
      </div>

      <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4">
        <Field label="Number">
          <Input
            value={number}
            onChange={(e) => setNumber(e.target.value)}
            placeholder="OP16-080"
          />
        </Field>
        <Field label="Name">
          <Input
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="Card name"
          />
        </Field>
        <Field label="Set">
          <Input
            value={set}
            onChange={(e) => setSet(e.target.value)}
            placeholder="OP16"
          />
        </Field>
        <Field
          label={
            isBRGame()
              ? "Market R$"
              : `TCG US$ ${fxRate > 0 && refUSD ? `(≈ ${brl0(Number(refUSD) / fxRate)})` : ""}`
          }
        >
          <Input
            type="number"
            value={refUSD}
            onChange={(e) => setRefUSD(e.target.value)}
            placeholder="0.00"
          />
        </Field>
        <Field label="Buy R$">
          <Input
            type="number"
            value={buyBRL}
            onChange={(e) => setBuyBRL(e.target.value)}
            placeholder="0,00"
          />
        </Field>
        <Field label="Frete R$">
          <Input
            type="number"
            value={shippingBRL}
            onChange={(e) => setShippingBRL(e.target.value)}
            placeholder="0,00"
          />
        </Field>
        <Field label="Qty">
          <Input
            type="number"
            value={qty}
            onChange={(e) => setQty(e.target.value)}
          />
        </Field>
        <Field label="Variant">
          <VariantInput value={variant} onChange={setVariant} />
        </Field>
        <Field label="Condition">
          <Input
            value={condition}
            onChange={(e) => setCondition(e.target.value)}
            placeholder="NM"
          />
        </Field>
        <Field label="Store">
          <Input
            value={store}
            onChange={(e) => setStore(e.target.value)}
            placeholder="Legends"
          />
        </Field>
        <Field label="Buy date">
          <Input
            type="date"
            value={buyDate}
            onChange={(e) => setBuyDate(e.target.value)}
          />
        </Field>
        <Field label="Image URL (optional)">
          <div className="flex items-center gap-2">
            <Input
              value={imageURL}
              onChange={(e) => setImageURL(e.target.value)}
              placeholder="https://…"
            />
            {imageURL.trim() && (
              <CardArt
                set=""
                number={number}
                name={name}
                imageURL={imageURL.trim()}
                className="h-10 w-[28px] shrink-0 rounded"
              />
            )}
          </div>
        </Field>
      </div>

      <div className="flex justify-end">
        <Button onClick={submit} disabled={!valid || saving}>
          {saving ? "Saving…" : "Save trade"}
        </Button>
      </div>
    </Card>
  );
}

function ManualTable({
  trades,
  onChanged,
  empty,
  sort,
  onSort,
  bare,
}: SortableProps & {
  trades: TradeView[];
  onChanged: () => void;
  empty: string;
  bare?: boolean;
}) {
  if (trades.length === 0) {
    return empty ? <Panel>{empty}</Panel> : null;
  }
  return (
    <div
      className={
        bare
          ? "overflow-x-auto"
          : "sticker sticker-sm overflow-x-auto rounded-[12px] bg-panel"
      }
    >
      <table className="w-full min-w-[640px] text-sm">
        <thead>
          <tr className="font-pixel border-b-2 border-outline bg-raised text-left text-[8px] uppercase text-brand-label">
            <SortableTh
              label="Produto"
              sortKey="name"
              sort={sort}
              onSort={onSort}
              align="left"
            />
            <SortableTh
              label="Custo"
              sortKey="cost"
              sort={sort}
              onSort={onSort}
            />
            <SortableTh
              label="Vendido"
              sortKey="value"
              sort={sort}
              onSort={onSort}
            />
            <SortableTh
              label="Lançado"
              sortKey="added"
              sort={sort}
              onSort={onSort}
            />
            <th className="px-3 py-2" />
          </tr>
        </thead>
        <tbody>
          {trades.map((t) => (
            <ManualRow key={t.id} t={t} onChanged={onChanged} />
          ))}
        </tbody>
      </table>
    </div>
  );
}

function ManualRow({
  t,
  onChanged,
}: {
  t: TradeView;
  onChanged: () => void;
}) {
  const [selling, setSelling] = useState(false);
  const [editing, setEditing] = useState(false);
  return (
    <>
      <tr
        className={`border-b-2 border-outline/15 last:border-0 hover:bg-raised/70 ${t.realized ? "opacity-60" : ""}`}
      >
        <td className="px-3 py-2">
          <div className="flex items-center gap-2">
            {t.imageURL ? (
              <CardArt
                set=""
                number={t.number}
                name={t.name}
                imageURL={t.imageURL}
                className="h-12 w-[34px] shrink-0 rounded"
              />
            ) : (
              <div className="art-stripes-90 flex h-12 w-[34px] shrink-0 items-center justify-center rounded bg-brand text-white">
                <Package className="h-5 w-5" />
              </div>
            )}
            <div className="min-w-0">
              <div
                className="truncate font-medium text-slate-100"
                title={t.name}
              >
                {t.name || t.number}
              </div>
              <div className="font-mono text-[10px] text-slate-500">
                {t.number}
                {t.qty > 1 ? ` ·×${t.qty}` : ""}
                {t.store ? ` · ${t.store}` : ""}
              </div>
            </div>
          </div>
        </td>
        <td className="px-3 py-2 text-right tabular-nums text-slate-300">
          {brl0(t.costBRL)}
        </td>
        <SoldCell t={t} />
        <AddedCell t={t} />
        <td className="px-3 py-2">
          <div className="flex items-center justify-end gap-1">
            {t.realized ? (
              <Badge variant="emerald">vendido</Badge>
            ) : (
              <>
                <DeliveryToggle t={t} onChanged={onChanged} />
                <button
                  onClick={() => {
                    setEditing((e) => !e);
                    setSelling(false);
                  }}
                  className="flex items-center gap-1 rounded-[8px] border-2 border-outline bg-panel px-2 py-1 text-xs font-bold text-slate-200 hover:bg-raised"
                  title="Editar o valor atual e o que você pagou"
                >
                  <Pencil className="h-3 w-3" /> Editar
                </button>
                <button
                  onClick={() => {
                    setSelling((s) => !s);
                    setEditing(false);
                  }}
                  className="rounded-[8px] border-2 border-outline bg-panel px-2 py-1 text-xs font-bold text-slate-200 hover:bg-raised"
                >
                  Vender
                </button>
              </>
            )}
            <button
              onClick={() => {
                if (confirm("Excluir este lançamento?")) {
                  deleteTrade(t.id).then(onChanged);
                }
              }}
              className="rounded-[8px] border-2 border-outline bg-panel p-1.5 text-slate-400 hover:bg-rose-950/40 hover:text-rose-300"
              title="Excluir"
            >
              <Trash2 className="h-3.5 w-3.5" />
            </button>
          </div>
        </td>
      </tr>
      {editing && (
        <tr className="bg-slate-900/60">
          <td colSpan={5} className="px-3 py-3">
            <EditTradeForm
              t={t}
              manual
              onDone={() => {
                setEditing(false);
                onChanged();
              }}
            />
          </td>
        </tr>
      )}
      {selling && (
        <tr className="bg-slate-900/60">
          <td colSpan={5} className="px-3 py-3">
            <SellForm
              t={t}
              onDone={() => {
                setSelling(false);
                onChanged();
              }}
            />
          </td>
        </tr>
      )}
    </>
  );
}

// EditTradeForm fixes an already-logged trade: what you paid (buy + frete + qty)
// plus the descriptive fields, so a mistyped cost no longer needs a delete/re-add.
// The manual variant (sealed/accessory) swaps Condition for the current-value
// estimate, which is what values those holdings (valuation.go uses ManualBRL * Qty).
function EditTradeForm({
  t,
  manual,
  onDone,
}: {
  t: TradeView;
  manual?: boolean;
  onDone: () => void;
}) {
  const [manualBRL, setManualBRL] = useState(
    t.manualBRL ? String(t.manualBRL) : "",
  );
  const [buyBRL, setBuyBRL] = useState(t.buyBRL ? String(t.buyBRL) : "");
  const [shippingBRL, setShippingBRL] = useState(
    t.shippingBRL ? String(t.shippingBRL) : "",
  );
  const [qty, setQty] = useState(String(t.qty));
  const [variant, setVariant] = useState(t.variant ?? "");
  const [condition, setCondition] = useState(t.condition ?? "");
  const [store, setStore] = useState(t.store ?? "");
  const [buyDate, setBuyDate] = useState(t.buyDate ?? "");
  const [imageURL, setImageURL] = useState(t.imageURL ?? "");
  const [saving, setSaving] = useState(false);

  const nextQty = Math.max(Number(qty) || 1, 1);
  const nextBuy = Number(buyBRL) || 0;
  const nextShip = Number(shippingBRL) || 0;
  const nextCost = nextQty * nextBuy + nextShip;

  const submit = async () => {
    setSaving(true);
    try {
      await updateTrade(t.id, {
        ...t,
        buyBRL: nextBuy,
        shippingBRL: nextShip,
        qty: nextQty,
        variant,
        condition,
        store,
        buyDate,
        imageURL: imageURL.trim(),
        ...(manual ? { manualBRL: Number(manualBRL) || 0 } : {}),
      });
      onDone();
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="flex flex-wrap items-end gap-2">
      {manual && (
        <Field label="Current value R$ (per unit)">
          <Input
            type="number"
            value={manualBRL}
            onChange={(e) => setManualBRL(e.target.value)}
            className="w-36"
            placeholder="0,00"
          />
        </Field>
      )}
      <Field label="Buy R$ (per unit)">
        <Input
          type="number"
          value={buyBRL}
          onChange={(e) => setBuyBRL(e.target.value)}
          className="w-32"
          placeholder="0,00"
        />
      </Field>
      <Field label="Frete R$">
        <Input
          type="number"
          value={shippingBRL}
          onChange={(e) => setShippingBRL(e.target.value)}
          className="w-28"
          placeholder="0,00"
        />
      </Field>
      <Field label="Qty">
        <Input
          type="number"
          min={1}
          value={qty}
          onChange={(e) => setQty(e.target.value)}
          className="w-20"
        />
      </Field>
      {!manual && (
        <Field label="Variant">
          <VariantInput value={variant} onChange={setVariant} className="w-32" />
        </Field>
      )}
      {!manual && (
        <Field label="Condition">
          <Input
            value={condition}
            onChange={(e) => setCondition(e.target.value)}
            className="w-24"
            placeholder="NM"
          />
        </Field>
      )}
      <Field label="Store">
        <Input
          value={store}
          onChange={(e) => setStore(e.target.value)}
          className="w-36"
          placeholder="Legends"
        />
      </Field>
      <Field label="Buy date">
        <Input
          type="date"
          value={buyDate}
          onChange={(e) => setBuyDate(e.target.value)}
          className="w-40"
        />
      </Field>
      <Field label="Image URL">
        <div className="flex items-center gap-2">
          <Input
            value={imageURL}
            onChange={(e) => setImageURL(e.target.value)}
            className="w-52"
            placeholder="https://…"
          />
          {imageURL.trim() && (
            <CardArt
              set=""
              number={t.number}
              name={t.name}
              imageURL={imageURL.trim()}
              className="h-10 w-[28px] shrink-0 rounded"
            />
          )}
        </div>
      </Field>
      <div className="pb-2 text-xs text-slate-400">
        Cost{" "}
        <span className="font-semibold text-slate-200">{brl0(nextCost)}</span>
        {nextCost !== t.costBRL ? (
          <span className="text-slate-500"> · was {brl0(t.costBRL)}</span>
        ) : (
          ""
        )}
      </div>
      <Button onClick={submit} disabled={saving}>
        {saving ? "Saving…" : "Save changes"}
      </Button>
    </div>
  );
}

function AddManualForm({
  kind,
  onAdded,
}: {
  kind: "sealed" | "accessory";
  onAdded: () => void;
}) {
  const sealed = kind === "sealed";
  const [query, setQuery] = useState("");
  const [matches, setMatches] = useState<QuoteMatch[]>([]);
  const [number, setNumber] = useState("");
  const [name, setName] = useState("");
  const [manualBRL, setManualBRL] = useState("");
  const [buyBRL, setBuyBRL] = useState("");
  const [shippingBRL, setShippingBRL] = useState("");
  const [qty, setQty] = useState("1");
  const [store, setStore] = useState("");
  const [buyDate, setBuyDate] = useState("");
  const [imageURL, setImageURL] = useState("");
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (!sealed || query.trim().length < 2) {
      setMatches([]);
      return;
    }
    let current = true;
    const h = window.setTimeout(() => {
      getQuote(query, 25, "sealed")
        .then((r) => current && setMatches(r.matches))
        .catch(() => current && setMatches([]));
    }, 250);
    return () => {
      current = false;
      window.clearTimeout(h);
    };
  }, [query, sealed]);

  const pick = (m: QuoteMatch) => {
    setNumber(m.number);
    setName(m.name);
    setQuery(m.name);
    setMatches([]);
  };

  const submit = async () => {
    setSaving(true);
    try {
      await createTrade({
        kind,
        number,
        name,
        set: sealed ? "SEALED" : "ACESSORIO",
        qty: Number(qty) || 1,
        buyBRL: Number(buyBRL) || 0,
        shippingBRL: Number(shippingBRL) || 0,
        manualBRL: Number(manualBRL) || 0,
        store,
        buyDate,
        imageURL: imageURL.trim(),
        status: "holding",
      });
      onAdded();
    } finally {
      setSaving(false);
    }
  };

  const valid = name && Number(buyBRL) > 0;

  return (
    <Card className="space-y-3 p-4">
      {sealed && (
        <div className="relative">
          <Field label="Find sealed product (Liga catalog)">
            <Input
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="e.g. Booster OP-16"
              className="w-full"
            />
          </Field>
          {matches.length > 0 && (
            <ul className="absolute z-10 mt-1 max-h-64 w-full overflow-y-auto sticker sticker-sm rounded-[10px] bg-panel">
              {matches.map((m) => (
                <li key={`${m.number}-${m.name}`}>
                  <button
                    type="button"
                    onClick={() => pick(m)}
                    className="flex w-full items-center justify-between gap-3 px-3 py-2 text-left text-sm hover:bg-raised"
                  >
                    <span className="truncate text-slate-100">{m.name}</span>
                    <span className="shrink-0 font-mono text-[11px] text-slate-500">
                      {m.number}
                    </span>
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>
      )}

      <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4">
        <Field label="Name">
          <Input
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder={
              sealed ? "Booster Box OP-16" : "Sleeves Ultra Pro (100)"
            }
          />
        </Field>
        <Field label="Current value R$ (per unit)">
          <Input
            type="number"
            value={manualBRL}
            onChange={(e) => setManualBRL(e.target.value)}
            placeholder="0,00"
          />
        </Field>
        <Field label="Buy R$ (per unit)">
          <Input
            type="number"
            value={buyBRL}
            onChange={(e) => setBuyBRL(e.target.value)}
            placeholder="0,00"
          />
        </Field>
        <Field label="Frete R$">
          <Input
            type="number"
            value={shippingBRL}
            onChange={(e) => setShippingBRL(e.target.value)}
            placeholder="0,00"
          />
        </Field>
        <Field label="Qty">
          <Input
            type="number"
            value={qty}
            onChange={(e) => setQty(e.target.value)}
          />
        </Field>
        <Field label="Store">
          <Input
            value={store}
            onChange={(e) => setStore(e.target.value)}
            placeholder="Legends"
          />
        </Field>
        <Field label="Buy date">
          <Input
            type="date"
            value={buyDate}
            onChange={(e) => setBuyDate(e.target.value)}
          />
        </Field>
        <Field label="Image URL (optional)">
          <div className="flex items-center gap-2">
            <Input
              value={imageURL}
              onChange={(e) => setImageURL(e.target.value)}
              placeholder="https://…"
            />
            {imageURL.trim() && (
              <CardArt
                set=""
                number={number}
                name={name}
                imageURL={imageURL.trim()}
                className="h-10 w-[28px] shrink-0 rounded"
              />
            )}
          </div>
        </Field>
      </div>

      <div className="flex justify-end">
        <Button onClick={submit} disabled={!valid || saving}>
          {saving ? "Saving…" : sealed ? "Save sealed" : "Save accessory"}
        </Button>
      </div>
    </Card>
  );
}

function VariantInput({
  value,
  onChange,
  className,
}: {
  value: string;
  onChange: (v: string) => void;
  className?: string;
}) {
  const id = useId();
  const options = VARIANT_SUGGESTIONS[getGame()] ?? DEFAULT_VARIANTS;
  return (
    <>
      <Input
        value={value}
        onChange={(e) => onChange(e.target.value)}
        list={id}
        placeholder="Foil"
        className={className}
      />
      <datalist id={id}>
        {options.map((v) => (
          <option key={v} value={v} />
        ))}
      </datalist>
    </>
  );
}

function Field({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <label className="flex flex-col gap-1">
      <span className="text-[10px] font-medium uppercase tracking-wide text-slate-500">
        {label}
      </span>
      {children}
    </label>
  );
}

function Panel({ children }: { children: React.ReactNode }) {
  return <EmptyState>{children}</EmptyState>;
}
