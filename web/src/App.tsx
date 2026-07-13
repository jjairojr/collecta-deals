import { useCallback, useEffect, useState, type ReactNode } from "react";
import { LayoutGrid, List, Search, X } from "lucide-react";
import {
  getDeals,
  getGame,
  getGames,
  getStatus,
  searchDeals,
  setGame as setApiGame,
  triggerRefresh,
  type DealFilters,
  type DealsResponse,
  type GameInfo,
  type Status,
} from "./api";
import { fullStamp } from "./format";
import { isView, ligaLabels, navItems, searchHints, type View } from "./brand";
import DealsTable from "./components/DealsTable";
import DealsGrid from "./components/DealsGrid";
import TrackingPage from "./components/TrackingPage";
import PortfolioPage from "./components/PortfolioPage";
import AllPortfolioPage from "./components/AllPortfolioPage";
import BuyoutPage from "./components/BuyoutPage";
import BrowsePage from "./components/BrowsePage";
import QuotePage from "./components/QuotePage";
import SelectionTray from "./components/SelectionTray";
import AppSidebar from "./components/AppSidebar";
import TopBar from "./components/TopBar";
import PageHeader from "./components/PageHeader";
import { SelectionProvider } from "./selection";
import { SidebarInset, SidebarProvider } from "./components/ui/sidebar";
import { Card } from "./components/ui/card";
import { Input } from "./components/ui/input";
import { Select } from "./components/ui/select";
import { ToggleGroup } from "./components/ui/toggle-group";

const defaultFilters: DealFilters = {
  minMargin: 20,
  minPrice: 100,
  sort: "margin",
  set: "",
  limit: 100,
  verifiedOnly: true,
  spOnly: false,
  ignoreMinMargin: false,
};

const pageMeta: Record<View, { title: string; description: string }> = {
  deals: {
    title: "Deals",
    description: "Cross-border price gaps — cheapest Brazil listing vs live US floor.",
  },
  browse: {
    title: "Browse",
    description: "Explore the full catalog with live prices.",
  },
  tracking: {
    title: "Tracking",
    description: "Daily price trends and per-store sales across the Brazil market.",
  },
  sealed: {
    title: "Sealed",
    description: "Sealed products — price trends and inferred sales.",
  },
  portfolio: {
    title: "Portfolio",
    description: "Your holdings, valuation, and realized P&L.",
  },
  allportfolio: {
    title: "All Games",
    description: "Combined portfolio across every game.",
  },
  orcamento: {
    title: "Orçamento",
    description: "Build a purchase quote to send a customer.",
  },
  buyout: {
    title: "Buyout",
    description: "Value a bulk buyout lot against the market.",
  },
};

function iconFor(view: View) {
  return navItems.find((item) => item.key === view)?.icon;
}

export default function App() {
  const [view, setView] = useView();
  const [game, setGameState] = useState<string>(getGame());
  const [games, setGames] = useState<GameInfo[]>([]);
  const [status, setStatus] = useState<Status | null>(null);
  const [query, setQuery] = useState("");
  const [debounced, setDebounced] = useState("");
  const [filters, setFilters] = useState<DealFilters>(defaultFilters);
  const [resp, setResp] = useState<DealsResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const refreshStatus = useCallback(async () => {
    try {
      setStatus(await getStatus());
    } catch {
      setStatus(null);
    }
  }, []);

  useEffect(() => {
    refreshStatus();
    const id = window.setInterval(refreshStatus, 5000);
    return () => window.clearInterval(id);
  }, [refreshStatus, game]);

  useEffect(() => {
    getGames()
      .then((data) => setGames(data.games))
      .catch(() => setGames([]));
  }, []);

  const activeGame = games.find((g) => g.id === game);
  const dealsEnabled = activeGame ? activeGame.hasDeals : game !== "pokemon";

  const changeGame = useCallback(
    (next: string) => {
      if (next === game) {
        return;
      }
      setApiGame(next);
      setGameState(next);
      setStatus(null);
      setResp(null);
      setFilters((f) => ({ ...f, set: "" }));
      const params = new URLSearchParams(window.location.search);
      if (next === "onepiece") {
        params.delete("game");
      } else {
        params.set("game", next);
      }
      const q = params.toString();
      window.history.replaceState(null, "", q ? `?${q}` : window.location.pathname);
    },
    [game],
  );

  useEffect(() => {
    if (!dealsEnabled && view === "deals") {
      setView("tracking");
    }
  }, [dealsEnabled, view, setView]);

  useEffect(() => {
    const id = window.setTimeout(() => setDebounced(query.trim()), 300);
    return () => window.clearTimeout(id);
  }, [query]);

  const ready = status?.ready ?? false;
  const updatedAt = status?.updatedAt ?? "";

  useEffect(() => {
    if (!ready || view !== "deals" || !dealsEnabled) {
      return;
    }
    let current = true;
    setLoading(true);
    setError(null);
    const load = debounced ? searchDeals(debounced) : getDeals(filters);
    load
      .then((data) => {
        if (current) {
          setResp(data);
        }
      })
      .catch((err: unknown) => {
        if (current) {
          setError(err instanceof Error ? err.message : "request failed");
        }
      })
      .finally(() => {
        if (current) {
          setLoading(false);
        }
      });
    return () => {
      current = false;
    };
  }, [ready, debounced, filters, updatedAt, view, game, dealsEnabled]);

  const onRefresh = useCallback(async () => {
    await triggerRefresh();
    refreshStatus();
  }, [refreshStatus]);

  const searching = debounced.length > 0;
  const activeView: View = view === "deals" && !dealsEnabled ? "tracking" : view;
  const meta = pageMeta[activeView];
  const ViewIcon = iconFor(activeView);

  return (
    <SelectionProvider>
      <SidebarProvider>
        <AppSidebar
          game={game}
          games={games}
          view={activeView}
          onChangeView={setView}
          onChangeGame={changeGame}
          dealsEnabled={dealsEnabled}
          status={status}
        />
        <SidebarInset>
          <TopBar game={game} status={status} onRefresh={onRefresh} dealsEnabled={dealsEnabled} />
          <div key={game} className="flex flex-1 flex-col">
            <main key={activeView} className="animate-fade-in mx-auto w-full max-w-7xl flex-1 px-4 py-7 sm:px-6 lg:px-8">
              <PageHeader title={meta.title} description={meta.description} icon={ViewIcon} />

              {activeView === "buyout" ? (
                <div className="mt-6">
                  <BuyoutPage />
                </div>
              ) : activeView === "browse" ? (
                <div className="mt-6">
                  <BrowsePage />
                </div>
              ) : activeView === "tracking" ? (
                <div className="mt-6">
                  <TrackingPage key="singles" mode="singles" />
                </div>
              ) : activeView === "sealed" ? (
                <div className="mt-6">
                  <TrackingPage key="sealed" mode="sealed" />
                </div>
              ) : activeView === "portfolio" ? (
                <div className="mt-6">
                  <PortfolioPage />
                </div>
              ) : activeView === "allportfolio" ? (
                <div className="mt-6">
                  <AllPortfolioPage
                    onOpenGame={(id) => {
                      changeGame(id);
                      setView("portfolio");
                    }}
                  />
                </div>
              ) : activeView === "orcamento" ? (
                <div className="mt-6">
                  <QuotePage />
                </div>
              ) : (
                <>
                  <div className="relative mt-6">
                    <Search className="pointer-events-none absolute left-4 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-500" />
                    <Input
                      type="search"
                      value={query}
                      onChange={(e) => setQuery(e.target.value)}
                      placeholder={`Search any card by name or number — e.g. ${searchHints[game] ?? searchHints.onepiece}`}
                      className="h-12 rounded-xl pl-11 pr-10 text-base [&::-webkit-search-cancel-button]:hidden"
                    />
                    {query && (
                      <button
                        type="button"
                        onClick={() => setQuery("")}
                        aria-label="Clear search"
                        className="absolute right-3 top-1/2 -translate-y-1/2 rounded-md p-1 text-slate-500 transition-colors hover:bg-slate-800 hover:text-slate-200"
                      >
                        <X className="h-4 w-4" />
                      </button>
                    )}
                  </div>

                  {!searching && <Filters filters={filters} onChange={setFilters} sets={resp?.sets ?? []} />}

                  {ready && updatedAt && (
                    <p className="mt-4 text-xs text-slate-500">
                      Prices as of <span className="font-medium text-slate-400">{fullStamp(updatedAt)}</span>
                      {status?.refreshing && <span className="ml-1 text-amber-400">· refreshing…</span>}
                    </p>
                  )}

                  <div className="mt-6">
                    {!ready ? (
                      <FirstRun refreshing={status?.refreshing ?? false} />
                    ) : (
                      <Results searching={searching} query={debounced} loading={loading} error={error} resp={resp} />
                    )}
                  </div>
                </>
              )}
            </main>
            <Footer game={game} hasDeals={dealsEnabled} />
          </div>
        </SidebarInset>
      </SidebarProvider>
      <SelectionTray />
    </SelectionProvider>
  );
}

function readView(): View {
  const value = new URLSearchParams(window.location.search).get("tab") ?? "";
  return isView(value) ? value : "deals";
}

function useView(): [View, (v: View) => void] {
  const [view, setViewState] = useState<View>(readView);

  useEffect(() => {
    const onPop = () => setViewState(readView());
    window.addEventListener("popstate", onPop);
    return () => window.removeEventListener("popstate", onPop);
  }, []);

  const setView = useCallback((next: View) => {
    const params = new URLSearchParams(window.location.search);
    if (next === "deals") {
      params.delete("tab");
    } else {
      params.set("tab", next);
    }
    const query = params.toString();
    window.history.pushState(null, "", query ? `?${query}` : window.location.pathname);
    setViewState(next);
  }, []);

  return [view, setView];
}

const marginPresets = [20, 50, 100];

function FilterField({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="flex flex-col gap-1.5">
      <span className="text-[10px] font-medium uppercase tracking-wide text-slate-500">{label}</span>
      {children}
    </div>
  );
}

function Filters({
  filters,
  onChange,
  sets,
}: {
  filters: DealFilters;
  onChange: (f: DealFilters) => void;
  sets: string[];
}) {
  return (
    <Card className="mt-4 flex flex-wrap items-end gap-5 p-4">
      <FilterField label="Set">
        <Select
          value={filters.set}
          onChange={(e) => onChange({ ...filters, set: e.target.value })}
          className="w-36"
        >
          <option value="">All sets</option>
          {sets.map((s) => (
            <option key={s} value={s}>
              {s}
            </option>
          ))}
        </Select>
      </FilterField>

      <FilterField label="Min margin">
        <div className={`flex items-center gap-2 ${filters.ignoreMinMargin ? "pointer-events-none opacity-40" : ""}`}>
          <div className="relative">
            <Input
              type="number"
              value={filters.minMargin}
              disabled={filters.ignoreMinMargin}
              onChange={(e) => onChange({ ...filters, minMargin: Number(e.target.value) })}
              className="w-24 pr-7"
            />
            <span className="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 text-xs text-slate-500">
              %
            </span>
          </div>
          <div className="flex gap-1">
            {marginPresets.map((p) => (
              <button
                key={p}
                type="button"
                onClick={() => onChange({ ...filters, minMargin: p })}
                className={`rounded-md px-2 py-1.5 text-xs font-medium transition-colors ${
                  filters.minMargin === p
                    ? "bg-accent-500/20 text-accent-200"
                    : "bg-slate-800/60 text-slate-400 hover:text-slate-200"
                }`}
              >
                {p}%
              </button>
            ))}
          </div>
        </div>
      </FilterField>

      <FilterField label="Min sell price">
        <div className="relative">
          <span className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-xs text-slate-500">
            US$
          </span>
          <Input
            type="number"
            value={filters.minPrice}
            onChange={(e) => onChange({ ...filters, minPrice: Number(e.target.value) })}
            className="w-28 pl-10"
          />
        </div>
      </FilterField>

      <FilterField label="Sort by">
        <ToggleGroup
          value={filters.sort}
          onChange={(v) => onChange({ ...filters, sort: v })}
          options={[
            { value: "margin", label: "Margin %" },
            { value: "profit", label: "Profit US$" },
          ]}
        />
      </FilterField>

      <FilterField label="Show">
        <ToggleGroup
          value={String(filters.limit)}
          onChange={(v) => onChange({ ...filters, limit: Number(v) })}
          options={[
            { value: "50", label: "50" },
            { value: "100", label: "100" },
            { value: "250", label: "250" },
          ]}
        />
      </FilterField>

      <label className="flex cursor-pointer items-center gap-2 pb-1.5 text-xs text-slate-300">
        <input
          type="checkbox"
          checked={filters.verifiedOnly}
          onChange={(e) => onChange({ ...filters, verifiedOnly: e.target.checked })}
          className="h-4 w-4 cursor-pointer accent-emerald-500"
        />
        Verified stock only
        <span className="text-slate-500">(live-checked, high value)</span>
      </label>

      <label className="flex cursor-pointer items-center gap-2 pb-1.5 text-xs text-slate-300">
        <input
          type="checkbox"
          checked={filters.spOnly}
          onChange={(e) => onChange({ ...filters, spOnly: e.target.checked })}
          className="h-4 w-4 cursor-pointer accent-accent-500"
        />
        SP only
        <span className="text-slate-500">(special parallel art)</span>
      </label>

      <label className="flex cursor-pointer items-center gap-2 pb-1.5 text-xs text-slate-300">
        <input
          type="checkbox"
          checked={filters.ignoreMinMargin}
          onChange={(e) => onChange({ ...filters, ignoreMinMargin: e.target.checked })}
          className="h-4 w-4 cursor-pointer accent-rose-500"
        />
        Include losses
        <span className="text-slate-500">(ignore min margin)</span>
      </label>
    </Card>
  );
}

function Results({
  searching,
  query,
  loading,
  error,
  resp,
}: {
  searching: boolean;
  query: string;
  loading: boolean;
  error: string | null;
  resp: DealsResponse | null;
}) {
  const [layout, setLayout] = useState<"grid" | "table">("grid");
  if (error) {
    return <Panel>Could not load data: {error}</Panel>;
  }
  if (!resp && loading) {
    return <Panel>Loading…</Panel>;
  }
  if (!resp) {
    return <Panel>No data yet.</Panel>;
  }
  if (resp.deals.length === 0) {
    return (
      <Panel>
        {searching
          ? `No cards match "${query}".`
          : "No deals match these filters. Try lowering the minimum margin."}
      </Panel>
    );
  }
  return (
    <>
      <div className="mb-3 flex items-center justify-between text-sm text-slate-400">
        <span>
          {searching ? (
            <>
              <span className="font-medium text-slate-200">{resp.count}</span> result
              {resp.count === 1 ? "" : "s"} for "{query}"
            </>
          ) : (
            <>
              <span className="font-medium text-slate-200">{resp.count}</span> best deals
            </>
          )}
        </span>
        <div className="flex items-center gap-3">
          {loading && <span className="text-xs text-slate-500">updating…</span>}
          <div className="inline-flex rounded-lg border border-slate-700 bg-slate-900 p-0.5">
            <button
              onClick={() => setLayout("grid")}
              aria-label="Grid view"
              className={`rounded-md p-1.5 transition-colors ${
                layout === "grid" ? "bg-accent-500/20 text-accent-200" : "text-slate-400 hover:text-slate-200"
              }`}
            >
              <LayoutGrid className="h-4 w-4" />
            </button>
            <button
              onClick={() => setLayout("table")}
              aria-label="Table view"
              className={`rounded-md p-1.5 transition-colors ${
                layout === "table" ? "bg-accent-500/20 text-accent-200" : "text-slate-400 hover:text-slate-200"
              }`}
            >
              <List className="h-4 w-4" />
            </button>
          </div>
        </div>
      </div>
      {layout === "grid" ? <DealsGrid deals={resp.deals} /> : <DealsTable deals={resp.deals} />}
    </>
  );
}

function FirstRun({ refreshing }: { refreshing: boolean }) {
  return (
    <Card className="flex flex-col items-center gap-4 py-16 text-center">
      <div className="h-8 w-8 animate-spin rounded-full border-2 border-slate-700 border-t-accent-400" />
      <div>
        <p className="font-medium text-slate-200">
          {refreshing ? "Scanning marketplaces…" : "Waiting for first scan…"}
        </p>
        <p className="mt-1 text-sm text-slate-400">
          The first run fetches every set from Liga (rate-limited) — this can take 1–2 minutes.
        </p>
      </div>
    </Card>
  );
}

function Panel({ children }: { children: ReactNode }) {
  return <Card className="px-4 py-10 text-center text-slate-400">{children}</Card>;
}

function Footer({ game, hasDeals }: { game: string; hasDeals: boolean }) {
  const liga = ligaLabels[game] ?? "Liga";
  return (
    <footer className="mx-auto w-full max-w-7xl border-t border-slate-800/80 px-4 py-5 text-xs text-slate-500 sm:px-6 lg:px-8">
      {hasDeals ? (
        <>
          Margin is an FX-adjusted gross price gap (lowest current TCGPlayer listing vs cheapest {liga} price). It does
          not include TCGPlayer fees or shipping. High-value deals use live TCGPlayer listing prices and are verified to
          have current {liga} sellers. Catalog via TCGCSV; live prices via TCGPlayer; Brazil prices via {liga}.
        </>
      ) : (
        <>
          Brazil market data via {liga}. Prices are the current per-store floor; quantities and prices are decoded per
          snapshot. Sales are inferred from day-over-day per-store stock drops. Not affiliated with {liga}.
        </>
      )}
    </footer>
  );
}
