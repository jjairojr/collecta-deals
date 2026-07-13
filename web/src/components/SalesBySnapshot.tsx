import { useState } from "react";
import { ChevronDown, ChevronRight } from "lucide-react";
import type { SnapshotSales } from "../api";
import { brl0, fullStamp, stampLabel } from "../format";
import SoldCardTile from "./SoldCardTile";

// snapStamp shows the real capture time (matching the header) when present,
// falling back to the storage-slot key label for older snapshots that predate it.
function snapStamp(iso: string | undefined, dateKey: string): string {
  if (iso && !iso.startsWith("0001")) {
    const s = fullStamp(iso);
    if (s) {
      return s;
    }
  }
  return stampLabel(dateKey);
}

function SnapshotRow({ snap, set }: { snap: SnapshotSales; set: string }) {
  const [open, setOpen] = useState(false);
  const hasSales = snap.units > 0;
  const cards = [...(snap.cards ?? [])].sort(
    (a, b) => b.revenueBRL - a.revenueBRL || b.units - a.units || a.number.localeCompare(b.number),
  );
  return (
    <div className="rounded-xl border border-slate-800 bg-slate-900/40">
      <button
        type="button"
        disabled={!hasSales}
        onClick={() => setOpen((o) => !o)}
        className={`flex w-full items-center gap-3 px-4 py-3 text-left ${
          hasSales ? "transition-colors hover:bg-slate-900/70" : "cursor-default"
        }`}
      >
        <span className="text-slate-500">
          {hasSales ? (
            open ? (
              <ChevronDown className="h-4 w-4" />
            ) : (
              <ChevronRight className="h-4 w-4" />
            )
          ) : (
            <ChevronRight className="h-4 w-4 opacity-30" />
          )}
        </span>
        <div className="flex-1">
          <div className="text-sm font-semibold text-slate-200">
            {snapStamp(snap.capturedAt, snap.date)}
          </div>
          <div className="text-[11px] text-slate-500">
            since {snapStamp(snap.prevCapturedAt, snap.prevDate)}
          </div>
        </div>
        {hasSales ? (
          <div className="flex items-center gap-4 text-right">
            <div>
              <div className="text-sm font-semibold tabular-nums text-emerald-300">
                {brl0(snap.revenueBRL)}
              </div>
              <div className="text-[10px] uppercase tracking-wide text-slate-500">moved</div>
            </div>
            <div>
              <div className="text-sm font-semibold tabular-nums text-sky-300">{snap.units}</div>
              <div className="text-[10px] uppercase tracking-wide text-slate-500">units</div>
            </div>
            <div className="hidden text-xs text-slate-500 sm:block">
              {snap.cards.length} card{snap.cards.length === 1 ? "" : "s"}
            </div>
          </div>
        ) : (
          <span className="text-xs text-slate-600">no sales</span>
        )}
      </button>
      {open && hasSales && (
        <div className="border-t border-slate-800 p-3">
          <div className="grid grid-cols-3 gap-3 sm:grid-cols-4 md:grid-cols-6">
            {cards.map((c) => (
              <SoldCardTile key={`${c.set ?? ""}${c.number}`} card={c} set={set} />
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

export default function SalesBySnapshot({
  snapshots,
  set,
}: {
  snapshots: SnapshotSales[];
  set: string;
}) {
  if (snapshots.length === 0) {
    return (
      <div className="rounded-xl border border-slate-800 bg-slate-900/30 px-4 py-8 text-center text-sm text-slate-500">
        No snapshot intervals in this range yet.
      </div>
    );
  }
  return (
    <div className="space-y-2">
      {snapshots.map((snap) => (
        <SnapshotRow key={snap.date} snap={snap} set={set} />
      ))}
    </div>
  );
}
