import { useCallback, useEffect, useState, type ReactNode } from "react";
import { LayoutGrid, List, Search, X } from "lucide-react";
import {
  getDeals,
  searchDeals,
  type DealFilters,
  type DealSource,
  type DealsResponse,
  type Status,
} from "../api";
import { fullStamp } from "../format";
import { ligaLabels, searchHints } from "../brand";
import DealsTable from "./DealsTable";
import DealsGrid from "./DealsGrid";
import { Card } from "./ui/card";
import { Input } from "./ui/input";
import { Select } from "./ui/select";
import { Tabs, TabsList, TabsTrigger } from "./ui/tabs";
import { ToggleGroup } from "./ui/toggle-group";

const defaultFilters: DealFilters = {
  minMargin: 20,
  minPrice: 100,
  sort: "margin",
  set: "",
  source: "liga",
  limit: 100,
  verifiedOnly: true,
  spOnly: false,
  ignoreMinMargin: false,
};

const marginPresets = [20, 50, 100];

function readSource(): DealSource {
  const value = new URLSearchParams(window.location.search).get("src");
  return value === "mypcards" ? "mypcards" : "liga";
}

// MyP listings carry their stock inline at parse time and are never floor-checked
// per store, so copies/sellers are always zero and every deal is "verified" by
// construction. Both signals are hidden on the MyP tab rather than shown as noise.
function supportsStockDepth(source: DealSource): boolean {
  return source !== "mypcards";
}

export default function DealsPage({
  game,
  status,
  hasMyP,
}: {
  game: string;
  status: Status | null;
  hasMyP: boolean;
}) {
  const [source, setSourceState] = useState<DealSource>(() => (hasMyP ? readSource() : "liga"));
  const [query, setQuery] = useState("");
  const [debounced, setDebounced] = useState("");
  const [filters, setFilters] = useState<DealFilters>(() => ({
    ...defaultFilters,
    source: hasMyP ? readSource() : "liga",
  }));
  const [resp, setResp] = useState<DealsResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const setSource = useCallback((next: DealSource) => {
    setSourceState(next);
    setResp(null);
    setFilters((f) => ({
      ...f,
      source: next,
      verifiedOnly: supportsStockDepth(next) ? f.verifiedOnly : false,
    }));
    const params = new URLSearchParams(window.location.search);
    if (next === "liga") {
      params.delete("src");
    } else {
      params.set("src", next);
    }
    const q = params.toString();
    window.history.replaceState(null, "", q ? `?${q}` : window.location.pathname);
  }, []);

  useEffect(() => {
    const id = window.setTimeout(() => setDebounced(query.trim()), 300);
    return () => window.clearTimeout(id);
  }, [query]);

  const ready = status?.ready ?? false;
  const updatedAt = status?.updatedAt ?? "";

  useEffect(() => {
    if (!ready) {
      return;
    }
    let current = true;
    setLoading(true);
    setError(null);
    const load = debounced ? searchDeals(debounced, filters.source) : getDeals(filters);
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
  }, [ready, debounced, filters, updatedAt, game]);

  const searching = debounced.length > 0;
  const showDepth = supportsStockDepth(source);

  return (
    <>
      <div className="mt-6 flex flex-wrap items-center gap-3">
        {hasMyP && (
          <Tabs value={source} onValueChange={(v) => setSource(v as DealSource)}>
            <TabsList>
              <TabsTrigger value="liga">{ligaLabels[game] ?? "Liga"}</TabsTrigger>
              <TabsTrigger value="mypcards">MyP Cards</TabsTrigger>
            </TabsList>
          </Tabs>
        )}
        <span className="text-xs text-slate-500">
          {showDepth
            ? "Brazil floor from Liga, live-verified against current sellers."
            : "Brazil floor from mypcards.com, stock read inline from the listing."}
        </span>
      </div>

      <div className="relative mt-4">
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

      {!searching && (
        <Filters filters={filters} onChange={setFilters} sets={resp?.sets ?? []} showVerified={showDepth} />
      )}

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
          <Results
            searching={searching}
            query={debounced}
            loading={loading}
            error={error}
            resp={resp}
            source={source}
            showDepth={showDepth}
          />
        )}
      </div>
    </>
  );
}

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
  showVerified,
}: {
  filters: DealFilters;
  onChange: (f: DealFilters) => void;
  sets: string[];
  showVerified: boolean;
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

      {showVerified && (
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
      )}

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
  source,
  showDepth,
}: {
  searching: boolean;
  query: string;
  loading: boolean;
  error: string | null;
  resp: DealsResponse | null;
  source: DealSource;
  showDepth: boolean;
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
    if (searching) {
      return <Panel>No cards match "{query}".</Panel>;
    }
    return (
      <Panel>
        {source === "mypcards"
          ? "No MyP Cards deals. This source is opt-in — the server must run with -mypcards for it to be scanned."
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
      {layout === "grid" ? (
        <DealsGrid deals={resp.deals} showDepth={showDepth} />
      ) : (
        <DealsTable deals={resp.deals} showDepth={showDepth} />
      )}
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
