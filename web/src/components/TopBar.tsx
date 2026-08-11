import { RefreshCw } from "lucide-react";
import type { Status } from "../api";
import { brandFor } from "../brand";
import { timeAgo } from "../format";
import { Button } from "./ui/button";
import { SidebarMobileTrigger } from "./ui/sidebar";
import SnapshotIndicator from "./SnapshotIndicator";

function StatPill({ label, value, sub, live }: { label: string; value: string; sub?: string; live?: boolean }) {
  return (
    <div className="pill pill-sm flex items-center gap-2 bg-panel px-3 py-1.5">
      {live !== undefined && (
        <span className={`h-2 w-2 rounded-full border border-outline ${live ? "animate-pulse bg-amber-400" : "bg-emerald-400"}`} />
      )}
      <span className="font-pixel text-[8px] uppercase text-brand-label">{label}</span>
      <span className="text-sm font-bold tabular-nums text-fg">{value}</span>
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
    <header className="sticky top-0 z-30 flex h-16 items-center gap-3 border-b-4 border-outline bg-page px-4 sm:px-6">
      <SidebarMobileTrigger />
      <div className="flex items-center gap-2 lg:hidden">
        <div className="sticker sticker-sm flex h-8 w-8 items-center justify-center rounded-[8px] bg-royal">
          <BrandIcon className="h-4 w-4 text-white" />
        </div>
        <span className="font-display text-sm font-extrabold text-fg">{brand.title}</span>
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
                label={refreshing ? "Sincronizando" : "Atualizado"}
                value={refreshing ? "…" : timeAgo(status.updatedAt)}
              />
            </div>
          </>
        )}
        <SnapshotIndicator />
        {dealsEnabled && (
          <Button variant="accent" size="sm" onClick={onRefresh} disabled={refreshing}>
            <RefreshCw className={refreshing ? "animate-spin" : ""} />
            <span className="hidden sm:inline">{refreshing ? "Atualizando…" : "Atualizar"}</span>
          </Button>
        )}
      </div>
    </header>
  );
}
