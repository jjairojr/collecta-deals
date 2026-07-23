import { useCallback, useEffect, useMemo, useState } from "react";
import { Plus, Trash2, Wallet, Coins, TrendingUp, PiggyBank, ExternalLink, Share2, Package, Pencil, Truck, Check, Search, Download, ChevronDown, ChevronsUpDown, ArrowUp, ArrowDown, X } from "lucide-react";
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
import { brl0, usd, timeAgo, fullStamp } from "../format";
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
import CardArt from "./CardArt";
import ShareList from "./ShareList";
import { Button } from "./ui/button";
import { Card } from "./ui/card";
import { Input } from "./ui/input";
import { Badge } from "./ui/badge";
import { ToggleGroup } from "./ui/toggle-group";

const pctOptions = [
  { value: "85", label: "85%" },
  { value: "90", label: "90%" },
  { value: "95", label: "95%" },
  { value: "100", label: "100%" },
];

function cleanName(n: string): string {
  return n.replace(/\s*\([^)]*\)\s*$/, "");
}

type Section = "singles" | "sealed";

interface SectionSummary {
  holdings: number;
  investedBRL: number;
  marketBRL: number;
  unrealizedBRL: number;
  sold: number;
  realizedBRL: number;
  totalPnLBRL: number;
}

function sectionSummary(list: TradeView[]): SectionSummary {
  const s: SectionSummary = {
    holdings: 0,
    investedBRL: 0,
    marketBRL: 0,
    unrealizedBRL: 0,
    sold: 0,
    realizedBRL: 0,
    totalPnLBRL: 0,
  };
  let costOfSold = 0;
  let proceeds = 0;
  for (const t of list) {
    if (t.realized) {
      s.sold += 1;
      costOfSold += t.costBRL;
      proceeds += t.valueBRL;
    } else {
      s.holdings += 1;
      s.investedBRL += t.costBRL;
      s.marketBRL += t.valueBRL;
    }
  }
  s.unrealizedBRL = s.marketBRL - s.investedBRL;
  s.realizedBRL = proceeds - costOfSold;
  s.totalPnLBRL = s.unrealizedBRL + s.realizedBRL;
  return s;
}

type SortKey = "name" | "cost" | "market" | "value" | "pnl" | "margin" | "added";
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
    case "market":
      return t.marketKnown ? t.marketUSD : (t.manualBRL ?? 0);
    case "value":
      return t.valueBRL;
    case "pnl":
      return t.profitBRL;
    case "margin":
      return t.marginPct;
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
  "status", "number", "name", "set", "condition", "qty", "store", "buyDate",
  "delivered", "costBRL", "marketUnit", "valueBRL", "pnlBRL", "marginPct",
  "sellDate", "sellPrice", "sellCurrency", "buyer",
];

function round2(n: number): number {
  return Math.round(n * 100) / 100;
}

function csvRows(list: TradeView[]): (string | number)[][] {
  return list.map((t) => {
    const marketUnit =
      t.kind === "sealed"
        ? (t.manualBRL ? round2(t.manualBRL) : "")
        : (t.marketKnown ? round2(t.marketUSD) : "");
    return [
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
      marketUnit,
      round2(t.valueBRL),
      round2(t.profitBRL),
      Math.round(t.marginPct),
      t.sellDate ?? "",
      t.sellPrice != null ? round2(t.sellPrice) : "",
      t.sellCurrency ?? "",
      t.buyer ?? "",
    ];
  });
}

type QuickFilter = "all" | "gainers" | "losers" | "transit" | "delivered";
type GroupBy = "set" | "store" | "none";

const QUICK: { v: QuickFilter; label: string }[] = [
  { v: "all", label: "All" },
  { v: "gainers", label: "Gainers" },
  { v: "losers", label: "Losers" },
  { v: "transit", label: "In transit" },
  { v: "delivered", label: "Delivered" },
];

function passesQuick(t: TradeView, f: QuickFilter): boolean {
  switch (f) {
    case "gainers":
      return t.profitBRL > 0;
    case "losers":
      return t.profitBRL < 0;
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
  value: number;
  pnl: number;
  marginPct: number;
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
    let value = 0;
    let pnl = 0;
    for (const t of trades) {
      count += t.qty;
      invested += t.costBRL;
      value += t.valueBRL;
      pnl += t.profitBRL;
    }
    groups.push({ key, trades, count, invested, value, pnl, marginPct: invested > 0 ? (pnl / invested) * 100 : 0 });
  }
  groups.sort((a, b) => b.value - a.value);
  return groups;
}

interface Insights {
  topGainer?: TradeView;
  topLoser?: TradeView;
  biggest?: TradeView;
  transitCount: number;
  transitValue: number;
}

function computeInsights(list: TradeView[]): Insights {
  const r: Insights = { transitCount: 0, transitValue: 0 };
  for (const t of list) {
    if (t.profitBRL > 0 && (!r.topGainer || t.profitBRL > r.topGainer.profitBRL)) {
      r.topGainer = t;
    }
    if (t.profitBRL < 0 && (!r.topLoser || t.profitBRL < r.topLoser.profitBRL)) {
      r.topLoser = t;
    }
    if (!r.biggest || t.valueBRL > r.biggest.valueBRL) {
      r.biggest = t;
    }
    if (!t.delivered) {
      r.transitCount += t.qty;
      r.transitValue += t.valueBRL;
    }
  }
  return r;
}

export default function PortfolioPage() {
  const [pct, setPct] = useState(90);
  const [section, setSection] = useState<Section>("singles");
  const [data, setData] = useState<PortfolioResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [adding, setAdding] = useState(false);
  const [sharing, setSharing] = useState(false);
  const [query, setQuery] = useState("");
  const [sort, setSort] = useState<SortState>({ key: "value", dir: "desc" });
  const [soldOpen, setSoldOpen] = useState(true);
  const [quick, setQuick] = useState<QuickFilter>("all");
  const [groupBy, setGroupBy] = useState<GroupBy>("set");
  const [expanded, setExpanded] = useState<Set<string>>(new Set());

  const load = useCallback(async (p: number) => {
    try {
      const r = await getPortfolio(p);
      setData(r);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "failed to load portfolio");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load(pct);
  }, [load, pct]);

  const refresh = useCallback(() => load(pct), [load, pct]);

  const trades = data?.trades ?? [];
  const active = useMemo(
    () => trades.filter((t) => (t.kind === "sealed") === (section === "sealed")),
    [trades, section],
  );
  const summary = useMemo(() => sectionSummary(active), [active]);

  const q = query.trim().toLowerCase();
  const holdings = useMemo(
    () => sortTrades(active.filter((t) => !t.realized && matchesQuery(t, q)), sort),
    [active, q, sort],
  );
  const sold = useMemo(
    () => sortTrades(active.filter((t) => t.realized && matchesQuery(t, q)), sort),
    [active, q, sort],
  );
  const visibleHoldings = useMemo(() => holdings.filter((t) => passesQuick(t, quick)), [holdings, quick]);
  const soldVisible = useMemo(
    () => (quick === "transit" || quick === "delivered" ? [] : sold.filter((t) => passesQuick(t, quick))),
    [sold, quick],
  );
  const maxValue = useMemo(() => visibleHoldings.reduce((m, t) => Math.max(m, t.valueBRL), 0), [visibleHoldings]);
  const holdingsValue = useMemo(() => visibleHoldings.reduce((s, t) => s + t.valueBRL, 0), [visibleHoldings]);
  const soldRealized = useMemo(() => soldVisible.reduce((s, t) => s + t.profitBRL, 0), [soldVisible]);
  const insights = useMemo(() => computeInsights(visibleHoldings), [visibleHoldings]);
  const holdingsGroups = useMemo(
    () => (groupBy === "none" ? [] : groupTrades(visibleHoldings, groupBy)),
    [visibleHoldings, groupBy],
  );
  const soldGroups = useMemo(
    () => (groupBy === "none" ? [] : groupTrades(soldVisible, groupBy)),
    [soldVisible, groupBy],
  );
  const quickCounts = useMemo(
    () => ({
      all: holdings.length,
      gainers: holdings.filter((t) => t.profitBRL > 0).length,
      losers: holdings.filter((t) => t.profitBRL < 0).length,
      transit: holdings.filter((t) => !t.delivered).length,
      delivered: holdings.filter((t) => Boolean(t.delivered)).length,
    }),
    [holdings],
  );
  const isSealed = section === "sealed";
  const anyExpanded = expanded.size > 0;

  const onSort = useCallback((key: SortKey) => {
    setSort((s) => (s.key === key ? { key, dir: s.dir === "asc" ? "desc" : "asc" } : { key, dir: "desc" }));
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
    setExpanded(new Set([...holdingsGroups, ...soldGroups].map((g) => g.key)));
  }, [holdingsGroups, soldGroups]);

  const collapseAll = useCallback(() => setExpanded(new Set()), []);

  const exportCSV = useCallback(() => {
    const rows = csvRows([...visibleHoldings, ...soldVisible]);
    const date = new Date().toISOString().slice(0, 10);
    downloadCSV(`portfolio-${getGame()}-${section}-${date}.csv`, toCSV(CSV_HEADERS, rows));
  }, [visibleHoldings, soldVisible, section]);

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="text-lg font-semibold text-slate-100">Portfolio</h1>
          <p className="mt-1 max-w-2xl text-sm text-slate-400">
            {isSealed
              ? "Your sealed buys and sales, valued at your own current-value estimate — no TCGplayer or Liga comparison."
              : isBRGame()
                ? "Your buys and sales, valued against the current Liga Brazil floor. Track what you're up — and down — across the whole collection."
                : "Your buys and sales, valued against live TCGplayer prices. Track what you're up — and down — across the whole collection."}
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-4">
          <ToggleGroup
            value={section}
            onChange={(v) => {
              setSection(v === "sealed" ? "sealed" : "singles");
              setAdding(false);
              setSharing(false);
            }}
            options={[
              { value: "singles", label: "Singles" },
              { value: "sealed", label: "Sealed" },
            ]}
          />
          {!isSealed && (
            <div className="flex items-center gap-2">
              <span className="text-[10px] font-medium uppercase tracking-wide text-slate-500">
                Value holdings at
              </span>
              <ToggleGroup
                value={String(pct)}
                onChange={(v) => setPct(Number(v))}
                options={pctOptions}
              />
            </div>
          )}
        </div>
      </div>

      {data && (
        <div className="grid grid-cols-2 gap-3 lg:grid-cols-5">
          <Kpi icon={<Wallet className="h-5 w-5" />} label="Invested (holding)" value={brl0(summary.investedBRL)} sub={`${summary.holdings} ${isSealed ? "products" : "cards"}`} />
          <Kpi icon={<Coins className="h-5 w-5" />} label="Current value" value={brl0(summary.marketBRL)} sub={isSealed ? "your estimate" : `@ ${pct}% of ${isBRGame() ? "floor" : "TCG"}`} />
          <PnlKpi icon={<TrendingUp className="h-5 w-5" />} label="Unrealized P&L" value={summary.unrealizedBRL} />
          <PnlKpi icon={<PiggyBank className="h-5 w-5" />} label={`Realized (${summary.sold} sold)`} value={summary.realizedBRL} />
          <PnlKpi icon={<TrendingUp className="h-5 w-5" />} label="Total P&L" value={summary.totalPnLBRL} strong />
        </div>
      )}

      {!loading && visibleHoldings.length > 0 && (
        <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
          <Insight
            label="Top gainer"
            primary={insights.topGainer ? cleanName(insights.topGainer.name) : "—"}
            secondary={insights.topGainer ? `+${brl0(insights.topGainer.profitBRL)}` : undefined}
            tone="up"
          />
          <Insight
            label="Top loser"
            primary={insights.topLoser ? cleanName(insights.topLoser.name) : "—"}
            secondary={insights.topLoser ? `−${brl0(Math.abs(insights.topLoser.profitBRL))}` : undefined}
            tone="down"
          />
          <Insight
            label="Biggest position"
            primary={insights.biggest ? cleanName(insights.biggest.name) : "—"}
            secondary={insights.biggest ? brl0(insights.biggest.valueBRL) : undefined}
          />
          <Insight
            label="In transit"
            primary={insights.transitCount ? `${insights.transitCount} ${isSealed ? "items" : "cards"}` : "None"}
            secondary={insights.transitCount ? brl0(insights.transitValue) : undefined}
          />
        </div>
      )}

      {error && (
        <div className="rounded-xl border border-rose-900/50 bg-rose-950/30 px-4 py-3 text-sm text-rose-200">
          {error}
        </div>
      )}

      <div className="flex flex-wrap items-center gap-2">
        <div className="relative w-full min-w-[200px] max-w-xs sm:w-auto sm:flex-1">
          <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-500" />
          <Input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder={isSealed ? "Search products, store…" : "Search cards, number, store…"}
            className="w-full pl-9 pr-8"
          />
          {query && (
            <button
              onClick={() => setQuery("")}
              className="absolute right-2 top-1/2 -translate-y-1/2 rounded p-0.5 text-slate-500 hover:text-slate-300"
              title="Clear search"
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
            title="Download this view as a spreadsheet"
          >
            <Download /> Export CSV
          </Button>
          {!isSealed && (
            <Button
              variant="outline"
              onClick={() => setSharing(true)}
              disabled={holdings.length === 0}
              title="Build a list to send to buyers"
            >
              <Share2 /> Share list
            </Button>
          )}
          <Button onClick={() => setAdding((a) => !a)}>
            <Plus /> {adding ? "Close" : isSealed ? "Add sealed" : "Add trade"}
          </Button>
        </div>
      </div>

      {!loading && active.length > 0 && (
        <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
          <div className="flex flex-wrap items-center gap-1.5">
            {QUICK.map((qf) => (
              <Chip key={qf.v} active={quick === qf.v} onClick={() => setQuick(qf.v)}>
                {qf.label} <span className="opacity-60">{quickCounts[qf.v]}</span>
              </Chip>
            ))}
          </div>
          <div className="ml-auto flex items-center gap-3">
            <div className="flex items-center gap-2">
              <span className="text-[10px] font-medium uppercase tracking-wide text-slate-500">Group by</span>
              <ToggleGroup
                value={groupBy}
                onChange={(v) => changeGroupBy(v === "store" ? "store" : v === "none" ? "none" : "set")}
                options={[
                  { value: "set", label: "Set" },
                  { value: "store", label: "Store" },
                  { value: "none", label: "None" },
                ]}
              />
            </div>
            {groupBy !== "none" && holdingsGroups.length + soldGroups.length > 0 && (
              <button
                onClick={anyExpanded ? collapseAll : expandAll}
                className="text-xs font-medium text-sky-300 hover:text-sky-200"
              >
                {anyExpanded ? "Collapse all" : "Expand all"}
              </button>
            )}
          </div>
        </div>
      )}

      {adding &&
        (isSealed ? (
          <AddSealedForm
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

      {sharing && !isSealed && holdings.length > 0 && (
        <ShareList holdings={holdings} fxRate={data?.fxRate ?? 0} onClose={() => setSharing(false)} />
      )}

      {loading ? (
        <Panel>Loading portfolio…</Panel>
      ) : active.length === 0 ? (
        <Panel>
          {isSealed
            ? "No sealed products yet. Click “Add sealed” to log your first box."
            : "No trades yet. Click “Add trade” to log your first buy."}
        </Panel>
      ) : (
        <div className="space-y-6">
          <div className="space-y-3">
            <div className="flex items-center gap-2">
              <h2 className="text-sm font-semibold text-slate-200">
                Holdings <span className="font-normal text-slate-500">· {visibleHoldings.length}</span>
              </h2>
              <span className="ml-auto tabular-nums text-xs font-medium text-slate-400">
                {brl0(holdingsValue)} value
              </span>
            </div>
            {groupBy === "none" ? (
              isSealed ? (
                <SealedTable
                  trades={visibleHoldings}
                  onChanged={refresh}
                  empty={q || quick !== "all" ? "No products match your filters." : "No sealed products held right now."}
                  sort={sort}
                  onSort={onSort}
                  maxValue={maxValue}
                />
              ) : (
                <TradeTable
                  trades={visibleHoldings}
                  onChanged={refresh}
                  empty={q || quick !== "all" ? "No cards match your filters." : "No cards held right now."}
                  sort={sort}
                  onSort={onSort}
                  maxValue={maxValue}
                />
              )
            ) : holdingsGroups.length === 0 ? (
              <Panel>{q || quick !== "all" ? "No cards match your filters." : "Nothing held right now."}</Panel>
            ) : (
              <div className="space-y-2">
                {holdingsGroups.map((g) => (
                  <GroupBlock
                    key={g.key}
                    g={g}
                    isSealed={isSealed}
                    open={expanded.has(g.key)}
                    onToggle={() => toggleGroup(g.key)}
                    sort={sort}
                    onSort={onSort}
                    maxValue={maxValue}
                    onChanged={refresh}
                  />
                ))}
              </div>
            )}
          </div>
          {soldVisible.length > 0 && (
            <div className="space-y-3">
              <button
                onClick={() => setSoldOpen((o) => !o)}
                className="flex w-full items-center gap-2 text-left"
              >
                <ChevronDown
                  className={`h-4 w-4 text-slate-500 transition-transform ${soldOpen ? "" : "-rotate-90"}`}
                />
                <h2 className="text-sm font-semibold text-slate-200">
                  Sold <span className="font-normal text-slate-500">· {soldVisible.length}</span>
                </h2>
                <span
                  className={`ml-auto tabular-nums text-xs font-medium ${soldRealized >= 0 ? "text-emerald-300" : "text-rose-300"}`}
                >
                  {soldRealized >= 0 ? "+" : "−"}
                  {brl0(Math.abs(soldRealized))} realized
                </span>
              </button>
              {soldOpen &&
                (groupBy === "none" ? (
                  isSealed ? (
                    <SealedTable trades={soldVisible} onChanged={refresh} empty="" sort={sort} onSort={onSort} maxValue={0} />
                  ) : (
                    <TradeTable trades={soldVisible} onChanged={refresh} empty="" sort={sort} onSort={onSort} maxValue={0} />
                  )
                ) : (
                  <div className="space-y-2">
                    {soldGroups.map((g) => (
                      <GroupBlock
                        key={g.key}
                        g={g}
                        isSealed={isSealed}
                        open={expanded.has(g.key)}
                        onToggle={() => toggleGroup(g.key)}
                        sort={sort}
                        onSort={onSort}
                        maxValue={0}
                        onChanged={refresh}
                      />
                    ))}
                  </div>
                ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

export function Kpi({ icon, label, value, sub }: { icon: React.ReactNode; label: string; value: string; sub?: string }) {
  return (
    <Card className="flex items-center gap-3 p-4">
      <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-slate-800/70 text-sky-300">{icon}</div>
      <div className="min-w-0">
        <div className="text-[10px] font-medium uppercase tracking-wide text-slate-500">{label}</div>
        <div className="truncate text-lg font-semibold tabular-nums text-slate-100">{value}</div>
        {sub && <div className="truncate text-[11px] text-slate-500">{sub}</div>}
      </div>
    </Card>
  );
}

export function PnlKpi({ icon, label, value, strong }: { icon: React.ReactNode; label: string; value: number; strong?: boolean }) {
  const up = value >= 0;
  const tone = up ? "text-emerald-300" : "text-rose-300";
  return (
    <Card className={`flex items-center gap-3 p-4 ${strong ? "ring-1 ring-inset ring-sky-500/20" : ""}`}>
      <div className={`flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-slate-800/70 ${tone}`}>{icon}</div>
      <div className="min-w-0">
        <div className="text-[10px] font-medium uppercase tracking-wide text-slate-500">{label}</div>
        <div className={`truncate text-lg font-semibold tabular-nums ${tone}`}>
          {up ? "+" : "−"}
          {brl0(Math.abs(value))}
        </div>
      </div>
    </Card>
  );
}

function Insight({
  label,
  primary,
  secondary,
  tone,
}: {
  label: string;
  primary: string;
  secondary?: string;
  tone?: "up" | "down";
}) {
  const toneCls = tone === "up" ? "text-emerald-300" : tone === "down" ? "text-rose-300" : "text-slate-100";
  return (
    <Card className="p-3">
      <div className="text-[10px] font-medium uppercase tracking-wide text-slate-500">{label}</div>
      <div className={`mt-0.5 truncate text-sm font-semibold ${toneCls}`} title={primary}>
        {primary}
      </div>
      {secondary && <div className="truncate text-[11px] tabular-nums text-slate-500">{secondary}</div>}
    </Card>
  );
}

function Chip({ active, onClick, children }: { active: boolean; onClick: () => void; children: React.ReactNode }) {
  return (
    <button
      onClick={onClick}
      className={`rounded-full px-3 py-1 text-xs font-medium ring-1 ring-inset transition-colors ${
        active
          ? "bg-sky-500/20 text-sky-200 ring-sky-500/30"
          : "bg-slate-900 text-slate-400 ring-slate-700 hover:text-slate-200"
      }`}
    >
      {children}
    </button>
  );
}

function GroupBlock({
  g,
  isSealed,
  open,
  onToggle,
  sort,
  onSort,
  maxValue,
  onChanged,
}: SortableProps & {
  g: Group;
  isSealed: boolean;
  open: boolean;
  onToggle: () => void;
  maxValue: number;
  onChanged: () => void;
}) {
  const up = g.pnl >= 0;
  const noun = isSealed ? (g.count === 1 ? "item" : "items") : g.count === 1 ? "card" : "cards";
  return (
    <div className="overflow-hidden rounded-xl border border-slate-800 bg-slate-900/40">
      <button
        onClick={onToggle}
        className="flex w-full items-center gap-3 px-3 py-2.5 text-left hover:bg-slate-800/40"
      >
        <ChevronDown
          className={`h-4 w-4 shrink-0 text-slate-500 transition-transform ${open ? "" : "-rotate-90"}`}
        />
        <span className="truncate font-semibold text-slate-100">{g.key}</span>
        <span className="shrink-0 text-xs text-slate-500">
          {g.count} {noun}
        </span>
        <div className="ml-auto flex shrink-0 items-center gap-3 tabular-nums text-xs sm:gap-5">
          <span className="hidden text-slate-500 sm:inline">
            inv <span className="text-slate-300">{brl0(g.invested)}</span>
          </span>
          <span className="text-slate-500">
            val <span className="text-slate-200">{brl0(g.value)}</span>
          </span>
          <span className={`font-semibold ${up ? "text-emerald-300" : "text-rose-300"}`}>
            {up ? "+" : "−"}
            {brl0(Math.abs(g.pnl))}
          </span>
          <MarginPill pct={g.marginPct} up={up} />
        </div>
      </button>
      {open && (
        <div className="border-t border-slate-800">
          {isSealed ? (
            <SealedTable trades={g.trades} onChanged={onChanged} empty="" sort={sort} onSort={onSort} maxValue={maxValue} bare />
          ) : (
            <TradeTable trades={g.trades} onChanged={onChanged} empty="" sort={sort} onSort={onSort} maxValue={maxValue} bare />
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
}: SortableProps & { label: string; sortKey: SortKey; align?: "left" | "right" }) {
  const activeCol = sort.key === sortKey;
  return (
    <th
      className={`px-3 py-2 font-medium ${align === "right" ? "text-right" : "text-left"}`}
    >
      <button
        type="button"
        onClick={() => onSort(sortKey)}
        className={`inline-flex items-center gap-1 hover:text-slate-300 ${align === "right" ? "flex-row-reverse" : ""} ${activeCol ? "text-slate-300" : ""}`}
      >
        {label}
        {activeCol ? (
          sort.dir === "asc" ? <ArrowUp className="h-3 w-3" /> : <ArrowDown className="h-3 w-3" />
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
  maxValue,
  bare,
}: SortableProps & { trades: TradeView[]; onChanged: () => void; empty: string; maxValue: number; bare?: boolean }) {
  if (trades.length === 0) {
    return empty ? <Panel>{empty}</Panel> : null;
  }
  return (
    <div className={bare ? "overflow-x-auto" : "overflow-x-auto rounded-xl border border-slate-800 bg-slate-900/40"}>
      <table className="w-full min-w-[760px] text-sm">
        <thead>
          <tr className="border-b border-slate-800 text-left text-xs uppercase tracking-wide text-slate-500">
            <SortableTh label="Card" sortKey="name" sort={sort} onSort={onSort} align="left" />
            <SortableTh label="Cost" sortKey="cost" sort={sort} onSort={onSort} />
            <SortableTh label={isBRGame() ? "Floor" : "TCG"} sortKey="market" sort={sort} onSort={onSort} />
            <SortableTh label="Value / Sold" sortKey="value" sort={sort} onSort={onSort} />
            <SortableTh label="P&L" sortKey="pnl" sort={sort} onSort={onSort} />
            <SortableTh label="Margin" sortKey="margin" sort={sort} onSort={onSort} />
            <SortableTh label="Added" sortKey="added" sort={sort} onSort={onSort} />
            <th className="px-3 py-2" />
          </tr>
        </thead>
        <tbody>
          {trades.map((t) => (
            <TradeRow key={t.id} t={t} onChanged={onChanged} maxValue={maxValue} />
          ))}
        </tbody>
      </table>
    </div>
  );
}

function DeliveryToggle({ t, onChanged }: { t: TradeView; onChanged: () => void }) {
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
      title={t.delivered ? "Delivered — click to mark in transit" : "In transit — click to mark delivered"}
      className={`flex items-center gap-1 rounded-md px-2 py-1 text-xs font-medium ring-1 ring-inset disabled:opacity-50 ${
        t.delivered
          ? "bg-emerald-500/10 text-emerald-300 ring-emerald-500/30 hover:bg-emerald-500/20"
          : "bg-amber-500/10 text-amber-300 ring-amber-500/30 hover:bg-amber-500/20"
      }`}
    >
      {t.delivered ? <Check className="h-3 w-3" /> : <Truck className="h-3 w-3" />}
      {t.delivered ? "Delivered" : "In transit"}
    </button>
  );
}

function ValueCell({ t, maxValue }: { t: TradeView; maxValue: number }) {
  const showBar = !t.realized && maxValue > 0 && t.valueBRL > 0;
  const width = showBar ? Math.max(3, (t.valueBRL / maxValue) * 100) : 0;
  return (
    <td className="relative px-3 py-2 text-right tabular-nums text-slate-200">
      {showBar && (
        <div
          className="pointer-events-none absolute inset-y-1 right-0 rounded-l bg-sky-500/10"
          style={{ width: `${width}%` }}
        />
      )}
      <span className="relative z-10">{brl0(t.valueBRL)}</span>
      {t.realized && <div className="relative z-10 text-[10px] text-slate-500">{t.sellDate || "sold"}</div>}
    </td>
  );
}

function MarginPill({ pct, up }: { pct: number; up: boolean }) {
  return (
    <span
      className={`inline-flex rounded-md px-1.5 py-0.5 text-xs font-medium tabular-nums ring-1 ring-inset ${
        up
          ? "bg-emerald-500/10 text-emerald-300 ring-emerald-500/20"
          : "bg-rose-500/10 text-rose-300 ring-rose-500/20"
      }`}
    >
      {up ? "+" : ""}
      {Math.round(pct)}%
    </span>
  );
}

function TradeRow({ t, onChanged, maxValue }: { t: TradeView; onChanged: () => void; maxValue: number }) {
  const [selling, setSelling] = useState(false);
  const [editing, setEditing] = useState(false);
  const up = t.profitBRL >= 0;
  return (
    <>
      <tr className={`border-b border-slate-800/60 last:border-0 ${t.realized ? "opacity-60" : ""}`}>
        <td className="px-3 py-2">
          <div className="flex items-center gap-2">
            <CardArt set={t.set} number={t.number} name={t.name} productID={productIDFromTcgURL(t.tcgUrl)} className="h-12 w-[34px] shrink-0 rounded" />
            <div className="min-w-0">
              <div className="truncate font-medium text-slate-100" title={t.name}>
                {cleanName(t.name) || t.number}
              </div>
              <div className="font-mono text-[10px] text-slate-500">
                {t.number}
                {t.qty > 1 ? ` ·×${t.qty}` : ""}
                {t.store ? ` · ${t.store}` : ""}
              </div>
            </div>
          </div>
        </td>
        <td className="px-3 py-2 text-right tabular-nums text-slate-300">{brl0(t.costBRL)}</td>
        <td className="px-3 py-2 text-right tabular-nums text-slate-400">
          {t.marketKnown ? marketMoney(t.marketUSD) : "—"}
        </td>
        <ValueCell t={t} maxValue={maxValue} />
        <td className={`px-3 py-2 text-right font-semibold tabular-nums ${up ? "text-emerald-300" : "text-rose-300"}`}>
          {up ? "+" : "−"}
          {brl0(Math.abs(t.profitBRL))}
        </td>
        <td className="px-3 py-2 text-right">
          <MarginPill pct={t.marginPct} up={up} />
        </td>
        <AddedCell t={t} />
        <td className="px-3 py-2">
          <div className="flex items-center justify-end gap-1">
            {t.tcgUrl && (
              <a
                href={t.tcgUrl}
                target="_blank"
                rel="noreferrer"
                className="flex items-center gap-1 rounded-md bg-sky-500/10 px-2 py-1 text-xs font-medium text-sky-300 ring-1 ring-inset ring-sky-500/30 hover:bg-sky-500/20"
                title="See on TCGplayer"
              >
                TCG <ExternalLink className="h-3 w-3" />
              </a>
            )}
            {t.realized ? (
              <Badge variant="emerald">sold</Badge>
            ) : (
              <>
                <DeliveryToggle t={t} onChanged={onChanged} />
                <button
                  onClick={() => {
                    setEditing((e) => !e);
                    setSelling(false);
                  }}
                  className="flex items-center gap-1 rounded-md border border-slate-700 bg-slate-800/60 px-2 py-1 text-xs text-slate-200 hover:bg-slate-800"
                  title="Edit what you paid and the trade details"
                >
                  <Pencil className="h-3 w-3" /> Edit
                </button>
                <button
                  onClick={() => {
                    setSelling((s) => !s);
                    setEditing(false);
                  }}
                  className="rounded-md border border-slate-700 bg-slate-800/60 px-2 py-1 text-xs text-slate-200 hover:bg-slate-800"
                >
                  Sell
                </button>
              </>
            )}
            <button
              onClick={() => {
                if (confirm("Delete this trade?")) {
                  deleteTrade(t.id).then(onChanged);
                }
              }}
              className="rounded-md border border-slate-700 bg-slate-800/60 p-1.5 text-slate-400 hover:bg-rose-950/40 hover:text-rose-300"
              title="Delete"
            >
              <Trash2 className="h-3.5 w-3.5" />
            </button>
          </div>
        </td>
      </tr>
      {editing && (
        <tr className="bg-slate-900/60">
          <td colSpan={8} className="px-3 py-3">
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
          <td colSpan={8} className="px-3 py-3">
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
    <td className="px-3 py-2 text-right text-xs tabular-nums text-slate-500" title={fullStamp(t.createdAt)}>
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
          <Input type="number" min={1} max={t.qty} value={qty} onChange={(e) => setQty(e.target.value)} className="w-20" />
        </Field>
      )}
      <Field label={t.qty > 1 ? "Price each" : "Sell price"}>
        <Input type="number" value={priceEach} onChange={(e) => setPriceEach(e.target.value)} className="w-32" placeholder="0,00" />
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
        <Input type="date" value={date} onChange={(e) => setDate(e.target.value)} className="w-40" />
      </Field>
      <Field label="Buyer (optional)">
        <Input value={buyer} onChange={(e) => setBuyer(e.target.value)} className="w-40" placeholder="P2P buyer" />
      </Field>
      {t.qty > 1 && each > 0 && (
        <div className="pb-2 text-xs text-slate-400">
          Total <span className="font-semibold text-slate-200">{money(total)}</span>
          {partial ? ` · ${t.qty - sellQty} stay holding` : ""}
        </div>
      )}
      <Button onClick={submit} disabled={saving || !each}>
        {saving ? "Saving…" : partial ? `Sell ${sellQty}` : "Mark sold"}
      </Button>
    </div>
  );
}

function AddTradeForm({ fxRate, onAdded }: { fxRate: number; onAdded: () => void }) {
  const [query, setQuery] = useState("");
  const [matches, setMatches] = useState<QuoteMatch[]>([]);
  const [number, setNumber] = useState("");
  const [name, setName] = useState("");
  const [set, setSet] = useState("");
  const [refUSD, setRefUSD] = useState("");
  const [buyBRL, setBuyBRL] = useState("");
  const [shippingBRL, setShippingBRL] = useState("");
  const [qty, setQty] = useState("1");
  const [condition, setCondition] = useState("NM");
  const [store, setStore] = useState("");
  const [buyDate, setBuyDate] = useState("");
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
        condition,
        qty: Number(qty) || 1,
        buyBRL: Number(buyBRL) || 0,
        shippingBRL: Number(shippingBRL) || 0,
        refUSD: Number(refUSD) || 0,
        store,
        buyDate,
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
          <ul className="absolute z-10 mt-1 max-h-64 w-full overflow-y-auto rounded-lg border border-slate-700 bg-slate-900 shadow-xl">
            {matches.map((m) => (
              <li key={`${m.number}-${m.name}`}>
                <button
                  type="button"
                  onClick={() => pick(m)}
                  className="flex w-full items-center justify-between gap-3 px-3 py-2 text-left text-sm hover:bg-slate-800"
                >
                  <span className="min-w-0">
                    <span className="truncate text-slate-100">{cleanName(m.name)}</span>{" "}
                    <span className="font-mono text-[11px] text-slate-500">{m.number}</span>
                  </span>
                  <span className="shrink-0 tabular-nums text-emerald-300">{marketMoney(m.marketUSD)}</span>
                </button>
              </li>
            ))}
          </ul>
        )}
      </div>

      <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4">
        <Field label="Number">
          <Input value={number} onChange={(e) => setNumber(e.target.value)} placeholder="OP16-080" />
        </Field>
        <Field label="Name">
          <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="Card name" />
        </Field>
        <Field label="Set">
          <Input value={set} onChange={(e) => setSet(e.target.value)} placeholder="OP16" />
        </Field>
        <Field
          label={
            isBRGame()
              ? "Market R$"
              : `TCG US$ ${fxRate > 0 && refUSD ? `(≈ ${brl0(Number(refUSD) / fxRate)})` : ""}`
          }
        >
          <Input type="number" value={refUSD} onChange={(e) => setRefUSD(e.target.value)} placeholder="0.00" />
        </Field>
        <Field label="Buy R$">
          <Input type="number" value={buyBRL} onChange={(e) => setBuyBRL(e.target.value)} placeholder="0,00" />
        </Field>
        <Field label="Frete R$">
          <Input type="number" value={shippingBRL} onChange={(e) => setShippingBRL(e.target.value)} placeholder="0,00" />
        </Field>
        <Field label="Qty">
          <Input type="number" value={qty} onChange={(e) => setQty(e.target.value)} />
        </Field>
        <Field label="Condition">
          <Input value={condition} onChange={(e) => setCondition(e.target.value)} placeholder="NM" />
        </Field>
        <Field label="Store">
          <Input value={store} onChange={(e) => setStore(e.target.value)} placeholder="Legends" />
        </Field>
        <Field label="Buy date">
          <Input type="date" value={buyDate} onChange={(e) => setBuyDate(e.target.value)} />
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

function SealedTable({
  trades,
  onChanged,
  empty,
  sort,
  onSort,
  maxValue,
  bare,
}: SortableProps & { trades: TradeView[]; onChanged: () => void; empty: string; maxValue: number; bare?: boolean }) {
  if (trades.length === 0) {
    return empty ? <Panel>{empty}</Panel> : null;
  }
  return (
    <div className={bare ? "overflow-x-auto" : "overflow-x-auto rounded-xl border border-slate-800 bg-slate-900/40"}>
      <table className="w-full min-w-[760px] text-sm">
        <thead>
          <tr className="border-b border-slate-800 text-left text-xs uppercase tracking-wide text-slate-500">
            <SortableTh label="Product" sortKey="name" sort={sort} onSort={onSort} align="left" />
            <SortableTh label="Cost" sortKey="cost" sort={sort} onSort={onSort} />
            <SortableTh label="Current R$" sortKey="market" sort={sort} onSort={onSort} />
            <SortableTh label="Value / Sold" sortKey="value" sort={sort} onSort={onSort} />
            <SortableTh label="P&L" sortKey="pnl" sort={sort} onSort={onSort} />
            <SortableTh label="Margin" sortKey="margin" sort={sort} onSort={onSort} />
            <SortableTh label="Added" sortKey="added" sort={sort} onSort={onSort} />
            <th className="px-3 py-2" />
          </tr>
        </thead>
        <tbody>
          {trades.map((t) => (
            <SealedRow key={t.id} t={t} onChanged={onChanged} maxValue={maxValue} />
          ))}
        </tbody>
      </table>
    </div>
  );
}

function SealedRow({ t, onChanged, maxValue }: { t: TradeView; onChanged: () => void; maxValue: number }) {
  const [selling, setSelling] = useState(false);
  const [editing, setEditing] = useState(false);
  const up = t.profitBRL >= 0;
  return (
    <>
      <tr className={`border-b border-slate-800/60 last:border-0 ${t.realized ? "opacity-60" : ""}`}>
        <td className="px-3 py-2">
          <div className="flex items-center gap-2">
            <div className="flex h-12 w-[34px] shrink-0 items-center justify-center rounded bg-slate-800/70 text-slate-400">
              <Package className="h-5 w-5" />
            </div>
            <div className="min-w-0">
              <div className="truncate font-medium text-slate-100" title={t.name}>
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
        <td className="px-3 py-2 text-right tabular-nums text-slate-300">{brl0(t.costBRL)}</td>
        <td className="px-3 py-2 text-right tabular-nums text-slate-400">
          {t.manualBRL ? brl0(t.manualBRL) : "—"}
        </td>
        <ValueCell t={t} maxValue={maxValue} />
        <td className={`px-3 py-2 text-right font-semibold tabular-nums ${up ? "text-emerald-300" : "text-rose-300"}`}>
          {up ? "+" : "−"}
          {brl0(Math.abs(t.profitBRL))}
        </td>
        <td className="px-3 py-2 text-right">
          <MarginPill pct={t.marginPct} up={up} />
        </td>
        <AddedCell t={t} />
        <td className="px-3 py-2">
          <div className="flex items-center justify-end gap-1">
            {t.realized ? (
              <Badge variant="emerald">sold</Badge>
            ) : (
              <>
                <DeliveryToggle t={t} onChanged={onChanged} />
                <button
                  onClick={() => {
                    setEditing((e) => !e);
                    setSelling(false);
                  }}
                  className="flex items-center gap-1 rounded-md border border-slate-700 bg-slate-800/60 px-2 py-1 text-xs text-slate-200 hover:bg-slate-800"
                  title="Edit current value and what you paid"
                >
                  <Pencil className="h-3 w-3" /> Edit
                </button>
                <button
                  onClick={() => {
                    setSelling((s) => !s);
                    setEditing(false);
                  }}
                  className="rounded-md border border-slate-700 bg-slate-800/60 px-2 py-1 text-xs text-slate-200 hover:bg-slate-800"
                >
                  Sell
                </button>
              </>
            )}
            <button
              onClick={() => {
                if (confirm("Delete this trade?")) {
                  deleteTrade(t.id).then(onChanged);
                }
              }}
              className="rounded-md border border-slate-700 bg-slate-800/60 p-1.5 text-slate-400 hover:bg-rose-950/40 hover:text-rose-300"
              title="Delete"
            >
              <Trash2 className="h-3.5 w-3.5" />
            </button>
          </div>
        </td>
      </tr>
      {editing && (
        <tr className="bg-slate-900/60">
          <td colSpan={8} className="px-3 py-3">
            <EditTradeForm
              t={t}
              sealed
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
          <td colSpan={8} className="px-3 py-3">
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
// The sealed variant swaps Condition for the manual current-value estimate, which
// is what values a sealed holding (valuation.go uses ManualBRL * Qty).
function EditTradeForm({ t, sealed, onDone }: { t: TradeView; sealed?: boolean; onDone: () => void }) {
  const [manualBRL, setManualBRL] = useState(t.manualBRL ? String(t.manualBRL) : "");
  const [buyBRL, setBuyBRL] = useState(t.buyBRL ? String(t.buyBRL) : "");
  const [shippingBRL, setShippingBRL] = useState(t.shippingBRL ? String(t.shippingBRL) : "");
  const [qty, setQty] = useState(String(t.qty));
  const [condition, setCondition] = useState(t.condition ?? "");
  const [store, setStore] = useState(t.store ?? "");
  const [buyDate, setBuyDate] = useState(t.buyDate ?? "");
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
        condition,
        store,
        buyDate,
        ...(sealed ? { manualBRL: Number(manualBRL) || 0 } : {}),
      });
      onDone();
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="flex flex-wrap items-end gap-2">
      {sealed && (
        <Field label="Current value R$ (per unit)">
          <Input type="number" value={manualBRL} onChange={(e) => setManualBRL(e.target.value)} className="w-36" placeholder="0,00" />
        </Field>
      )}
      <Field label="Buy R$ (per unit)">
        <Input type="number" value={buyBRL} onChange={(e) => setBuyBRL(e.target.value)} className="w-32" placeholder="0,00" />
      </Field>
      <Field label="Frete R$">
        <Input type="number" value={shippingBRL} onChange={(e) => setShippingBRL(e.target.value)} className="w-28" placeholder="0,00" />
      </Field>
      <Field label="Qty">
        <Input type="number" min={1} value={qty} onChange={(e) => setQty(e.target.value)} className="w-20" />
      </Field>
      {!sealed && (
        <Field label="Condition">
          <Input value={condition} onChange={(e) => setCondition(e.target.value)} className="w-24" placeholder="NM" />
        </Field>
      )}
      <Field label="Store">
        <Input value={store} onChange={(e) => setStore(e.target.value)} className="w-36" placeholder="Legends" />
      </Field>
      <Field label="Buy date">
        <Input type="date" value={buyDate} onChange={(e) => setBuyDate(e.target.value)} className="w-40" />
      </Field>
      <div className="pb-2 text-xs text-slate-400">
        Cost <span className="font-semibold text-slate-200">{brl0(nextCost)}</span>
        {nextCost !== t.costBRL ? <span className="text-slate-500"> · was {brl0(t.costBRL)}</span> : ""}
      </div>
      <Button onClick={submit} disabled={saving}>
        {saving ? "Saving…" : "Save changes"}
      </Button>
    </div>
  );
}

function AddSealedForm({ onAdded }: { onAdded: () => void }) {
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
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (query.trim().length < 2) {
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
  }, [query]);

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
        kind: "sealed",
        number,
        name,
        set: "SEALED",
        qty: Number(qty) || 1,
        buyBRL: Number(buyBRL) || 0,
        shippingBRL: Number(shippingBRL) || 0,
        manualBRL: Number(manualBRL) || 0,
        store,
        buyDate,
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
          <ul className="absolute z-10 mt-1 max-h-64 w-full overflow-y-auto rounded-lg border border-slate-700 bg-slate-900 shadow-xl">
            {matches.map((m) => (
              <li key={`${m.number}-${m.name}`}>
                <button
                  type="button"
                  onClick={() => pick(m)}
                  className="flex w-full items-center justify-between gap-3 px-3 py-2 text-left text-sm hover:bg-slate-800"
                >
                  <span className="truncate text-slate-100">{m.name}</span>
                  <span className="shrink-0 font-mono text-[11px] text-slate-500">{m.number}</span>
                </button>
              </li>
            ))}
          </ul>
        )}
      </div>

      <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4">
        <Field label="Name">
          <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="Booster Box OP-16" />
        </Field>
        <Field label="Current value R$ (per unit)">
          <Input type="number" value={manualBRL} onChange={(e) => setManualBRL(e.target.value)} placeholder="0,00" />
        </Field>
        <Field label="Buy R$ (per unit)">
          <Input type="number" value={buyBRL} onChange={(e) => setBuyBRL(e.target.value)} placeholder="0,00" />
        </Field>
        <Field label="Frete R$">
          <Input type="number" value={shippingBRL} onChange={(e) => setShippingBRL(e.target.value)} placeholder="0,00" />
        </Field>
        <Field label="Qty">
          <Input type="number" value={qty} onChange={(e) => setQty(e.target.value)} />
        </Field>
        <Field label="Store">
          <Input value={store} onChange={(e) => setStore(e.target.value)} placeholder="Legends" />
        </Field>
        <Field label="Buy date">
          <Input type="date" value={buyDate} onChange={(e) => setBuyDate(e.target.value)} />
        </Field>
      </div>

      <div className="flex justify-end">
        <Button onClick={submit} disabled={!valid || saving}>
          {saving ? "Saving…" : "Save sealed"}
        </Button>
      </div>
    </Card>
  );
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <label className="flex flex-col gap-1">
      <span className="text-[10px] font-medium uppercase tracking-wide text-slate-500">{label}</span>
      {children}
    </label>
  );
}

function Panel({ children }: { children: React.ReactNode }) {
  return (
    <div className="rounded-xl border border-slate-800 bg-slate-900/40 px-4 py-10 text-center text-slate-400">
      {children}
    </div>
  );
}
