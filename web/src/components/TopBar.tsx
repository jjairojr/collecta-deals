import { RefreshCw } from "lucide-react";
import type { Status } from "../api";
import { brandFor } from "../brand";
import { timeAgo } from "../format";
import { Button } from "./ui/button";
import { SidebarMobileTrigger } from "./ui/sidebar";
import SnapshotIndicator from "./SnapshotIndicator";

function StatPill({ label, value, sub, live }: { label: string; value: string; sub?: string; live?: boolean }) {
  return (
    <div className="flex items-center gap-2 rounded-lg border border-slate-800 bg-slate-900/60 px-3 py-1.5">
      {live !== undefined && (
        <span className={`h-1.5 w-1.5 rounded-full ${live ? "animate-pulse bg-amber-400" : "bg-emerald-400"}`} />
      )}
      <span className="text-[10px] font-medium uppercase tracking-wide text-slate-500">{label}</span>
      <span className="text-sm font-semibold tabular-nums text-slate-200">{value}</span>
      {sub && <span className="text-[10px] text-slate-500">{sub}</span>}
    </div>
  );
}

export default function TopBar({
  game,
  status,
  onRefresh,
  dealsEnabled,
}: {
  game: string;
  status: Status | null;
  onRefresh: () => void;
  dealsEnabled: boolean;
}) {
  const refreshing = status?.refreshing ?? false;
  const brand = brandFor(game);
  const BrandIcon = brand.icon;

  return (
    <header className="sticky top-0 z-30 flex h-16 items-center gap-3 border-b border-slate-800/80 bg-slate-950/80 px-4 backdrop-blur-xl sm:px-6">
      <SidebarMobileTrigger />
      <div className="flex items-center gap-2 lg:hidden">
        <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-gradient-to-br from-accent-500/25 to-accent-500/5 ring-1 ring-inset ring-accent-500/25">
          <BrandIcon className="h-4 w-4 text-accent-300" />
        </div>
        <span className="text-sm font-bold tracking-tight text-slate-100">{brand.title}</span>
      </div>

      <div className="ml-auto flex items-center gap-2">
        {dealsEnabled && status && (
          <>
            <div className="hidden sm:block">
              <StatPill label="FX" value={status.fxRate ? status.fxRate.toFixed(4) : "—"} sub="BRL→USD" />
            </div>
            <div className="hidden md:block">
              <StatPill
                live={refreshing}
                label={refreshing ? "Syncing" : "Updated"}
                value={refreshing ? "…" : timeAgo(status.updatedAt)}
              />
            </div>
          </>
        )}
        <SnapshotIndicator />
        {dealsEnabled && (
          <Button variant="accent" size="sm" onClick={onRefresh} disabled={refreshing}>
            <RefreshCw className={refreshing ? "animate-spin" : ""} />
            <span className="hidden sm:inline">{refreshing ? "Refreshing…" : "Refresh"}</span>
          </Button>
        )}
      </div>
    </header>
  );
}
