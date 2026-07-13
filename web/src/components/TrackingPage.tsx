import { useCallback, useEffect, useMemo, useState } from "react";
import {
  getLatestSnapshot,
  getLeaderboard,
  getSalesBySnapshot,
  getTrends,
  triggerCapture,
  type CardTrend,
  type LatestSnapshot,
  type SnapshotSales,
  type StoreStat,
  type TrendRange,
} from "../api";
import { Boxes, Camera, Clock } from "lucide-react";
import KpiStrip from "./KpiStrip";
import SalesSection from "./SalesSection";
import PriceMovers from "./PriceMovers";
import StoreLeaderboard from "./StoreLeaderboard";
import InventoryPanel from "./InventoryPanel";
import SetSelect, { ALL_SETS } from "./SetSelect";
import { useSets } from "../useSets";
import { salesTotals } from "../salesutil";
import { fullStamp, timeAgo } from "../format";
import { Button } from "./ui/button";
import { Card } from "./ui/card";
import { ToggleGroup } from "./ui/toggle-group";

type Window = "24h" | "7d" | "30d" | "all";

const windowOptions = [
  { value: "24h", label: "24h" },
  { value: "7d", label: "7d" },
  { value: "30d", label: "30d" },
  { value: "all", label: "All" },
];

function isWindow(v: string): v is Window {
  return v === "24h" || v === "7d" || v === "30d" || v === "all";
}

function windowFrom(win: Window, latestKey: string): string {
  if (win === "all" || !latestKey) {
    return "";
  }
  const m = latestKey.match(/^(\d{4})-(\d{2})-(\d{2})/);
  if (!m) {
    return "";
  }
  const days = win === "24h" ? 1 : win === "7d" ? 7 : 30;
  const dt = new Date(Number(m[1]), Number(m[2]) - 1, Number(m[3]));
  dt.setDate(dt.getDate() - days);
  const mm = String(dt.getMonth() + 1).padStart(2, "0");
  const dd = String(dt.getDate()).padStart(2, "0");
  return `${dt.getFullYear()}-${mm}-${dd}`;
}

function moverRange(win: Window): TrendRange {
  return win === "24h" ? "daily" : win === "7d" ? "weekly" : "monthly";
}

const SEALED_SET = "SEALED";

export default function TrackingPage({ mode = "singles" }: { mode?: "singles" | "sealed" }) {
  const isSealed = mode === "sealed";
  const allSets = useSets();
  const sets = useMemo(
    () => (isSealed ? allSets.filter((s) => s === SEALED_SET) : allSets.filter((s) => s !== SEALED_SET)),
    [allSets, isSealed],
  );
  const [set, setSet] = useState(isSealed ? SEALED_SET : ALL_SETS);
  const [win, setWin] = useState<Window>("all");
  const [snapshot, setSnapshot] = useState<LatestSnapshot | null>(null);
  const [snapshots, setSnapshots] = useState<SnapshotSales[]>([]);
  const [snapLoaded, setSnapLoaded] = useState(false);
  const [trends, setTrends] = useState<CardTrend[]>([]);
  const [prevDate, setPrevDate] = useState("");
  const [trendsLoaded, setTrendsLoaded] = useState(false);
  const [stores, setStores] = useState<StoreStat[]>([]);
  const [sort, setSort] = useState<"units" | "revenue">("revenue");
  const [error, setError] = useState<string | null>(null);
  const [capturing, setCapturing] = useState(false);

  const isAll = !isSealed && set === ALL_SETS;
  const latestDate = snapshot?.date ?? "";
  const from = windowFrom(win, latestDate);

  const loadLatest = useCallback(async () => {
    try {
      const l = await getLatestSnapshot();
      setSnapshot(l.capturedAt ? l : null);
    } catch {
      setSnapshot(null);
    }
  }, []);

  useEffect(() => {
    loadLatest();
  }, [loadLatest]);

  useEffect(() => {
    if (sets.length > 0 && set !== ALL_SETS && !sets.includes(set)) {
      setSet(sets[0]);
    }
  }, [sets, set]);

  useEffect(() => {
    let current = true;
    setSnapLoaded(false);
    getSalesBySnapshot(set, from, "")
      .then((r) => current && setSnapshots(r.snapshots))
      .catch((err: unknown) => {
        if (current) {
          setError(err instanceof Error ? err.message : "failed to load sales");
        }
      })
      .finally(() => current && setSnapLoaded(true));
    return () => {
      current = false;
    };
  }, [set, from]);

  useEffect(() => {
    let current = true;
    setTrendsLoaded(false);
    getTrends(set, moverRange(win))
      .then((r) => {
        if (current) {
          setTrends(r.trends);
          setPrevDate(r.prevDate);
        }
      })
      .finally(() => current && setTrendsLoaded(true));
    return () => {
      current = false;
    };
  }, [set, win]);

  useEffect(() => {
    if (isAll) {
      return;
    }
    let current = true;
    getLeaderboard(set, sort, from, "")
      .then((r) => current && setStores(r.stores))
      .catch((err: unknown) =>
        setError(err instanceof Error ? err.message : "failed to load leaderboard"),
      );
    return () => {
      current = false;
    };
  }, [set, isAll, sort, from]);

  const onCapture = useCallback(async () => {
    setCapturing(true);
    try {
      await triggerCapture(isSealed);
    } finally {
      window.setTimeout(() => {
        setCapturing(false);
        loadLatest();
      }, 2000);
    }
  }, [loadLatest, isSealed]);

  const totals = useMemo(() => salesTotals(snapshots), [snapshots]);
  const topMover = useMemo(() => {
    let best: CardTrend | null = null;
    for (const t of trends) {
      if (t.prevBRL > 0 && t.deltaPct !== 0 && (!best || Math.abs(t.deltaPct) > Math.abs(best.deltaPct))) {
        best = t;
      }
    }
    return best;
  }, [trends]);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-lg font-semibold text-slate-100">
          {isSealed ? "Sealed products" : "Market pulse"}
        </h1>
        <p className="mt-1 max-w-2xl text-sm text-slate-400">
          {isSealed
            ? "Booster boxes, packs & decks — units sold and price swings, inferred from per-store snapshots."
            : "What's moving — units sold and price swings, inferred from daily per-store snapshots."}
        </p>
      </div>

      <Card className="flex flex-wrap items-center justify-between gap-3 p-3">
        <div className="flex flex-wrap items-center gap-3">
          {!isSealed && <SetSelect sets={sets} value={set} onChange={setSet} allowAll />}
          <div className="flex items-center gap-2">
            <span className="text-[10px] font-medium uppercase tracking-wide text-slate-500">
              Range
            </span>
            <ToggleGroup value={win} onChange={(v) => isWindow(v) && setWin(v)} options={windowOptions} />
          </div>
          {isAll && (
            <span className="rounded-lg border border-slate-800 bg-slate-900/60 px-3 py-1.5 text-xs text-slate-400">
              <span className="font-semibold text-slate-200">{sets.length}</span> collections pooled
            </span>
          )}
          {isSealed && (
            <span className="rounded-lg border border-slate-800 bg-slate-900/60 px-3 py-1.5 text-xs text-slate-400">
              Boxes · packs · decks
            </span>
          )}
        </div>
        <div className="flex items-center gap-3">
          <div className="flex items-center gap-2 rounded-lg border border-slate-800 bg-slate-900/60 px-3 py-1.5">
            <Clock className="h-3.5 w-3.5 text-sky-300" />
            <span className="text-[10px] font-medium uppercase tracking-wide text-slate-500">
              Last snapshot
            </span>
            {snapshot ? (
              <>
                <span className="text-sm font-semibold tabular-nums text-slate-200">
                  {fullStamp(snapshot.capturedAt)}
                </span>
                <span className="text-[10px] text-slate-500">{timeAgo(snapshot.capturedAt)}</span>
              </>
            ) : (
              <span className="text-sm font-semibold text-slate-400">never</span>
            )}
          </div>
          <Button onClick={onCapture} disabled={capturing}>
            <Camera className={capturing ? "animate-pulse" : ""} />
            {capturing ? "Capturing…" : "Capture"}
          </Button>
        </div>
      </Card>

      {error && (
        <div className="rounded-xl border border-rose-900/50 bg-rose-950/30 px-4 py-3 text-sm text-rose-200">
          Could not load tracking data: {error}
        </div>
      )}

      <KpiStrip
        units={totals.units}
        revenueBRL={totals.revenueBRL}
        storesSelling={totals.storesSelling}
        topMover={topMover}
      />

      <SalesSection snapshots={snapshots} set={set} loaded={snapLoaded} />

      <PriceMovers trends={trends} prevDate={prevDate} set={set} loaded={trendsLoaded} />

      {!isAll && (
        <section className="space-y-3">
          <StoreLeaderboard stores={stores} sort={sort} onSortChange={setSort} />
        </section>
      )}

      <section className="space-y-3">
        <div className="flex items-center gap-2">
          <span className="text-sky-300">
            <Boxes className="h-4 w-4" />
          </span>
          <h2 className="text-sm font-semibold text-slate-200">Collection inventory</h2>
          <span className="text-xs text-slate-500">· who holds what, and the chase cards</span>
        </div>
        <InventoryPanel key={set} set={set} />
      </section>
    </div>
  );
}
