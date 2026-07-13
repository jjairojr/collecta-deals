import { useCallback, useEffect, useState } from "react";
import { ChevronDown, Clock } from "lucide-react";
import { getLatestSnapshot, getRecentSnapshots, type LatestSnapshot } from "../api";
import { fullStamp, stampLabel, timeAgo } from "../format";

export default function SnapshotIndicator() {
  const [latest, setLatest] = useState<LatestSnapshot | null>(null);
  const [dates, setDates] = useState<string[]>([]);
  const [open, setOpen] = useState(false);

  const load = useCallback(() => {
    getLatestSnapshot()
      .then((l) => setLatest(l.capturedAt ? l : null))
      .catch(() => setLatest(null));
    getRecentSnapshots(12)
      .then((r) => setDates(r.dates))
      .catch(() => setDates([]));
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const toggle = () => {
    if (!open) {
      load();
    }
    setOpen((o) => !o);
  };

  return (
    <div className="relative">
      <button
        type="button"
        onClick={toggle}
        className="flex items-center gap-2 rounded-lg border border-slate-800 bg-slate-900/60 px-3 py-1.5 transition-colors hover:border-slate-700 hover:bg-slate-900"
      >
        <Clock className="h-3.5 w-3.5 text-sky-300" />
        <span className="text-[10px] font-medium uppercase tracking-wide text-slate-500">
          Snapshot
        </span>
        <span className="text-sm font-semibold tabular-nums text-slate-200">
          {latest ? fullStamp(latest.capturedAt) : "never"}
        </span>
        {latest && <span className="text-[10px] text-slate-500">{timeAgo(latest.capturedAt)}</span>}
        <ChevronDown
          className={`h-3 w-3 text-slate-500 transition-transform ${open ? "rotate-180" : ""}`}
        />
      </button>
      {open && (
        <>
          <button
            type="button"
            aria-hidden
            tabIndex={-1}
            className="fixed inset-0 z-10 cursor-default"
            onClick={() => setOpen(false)}
          />
          <div className="absolute right-0 z-20 mt-1 w-56 rounded-lg border border-slate-700 bg-slate-900 shadow-xl">
            <div className="border-b border-slate-800 px-3 py-2 text-[10px] font-medium uppercase tracking-wide text-slate-500">
              Recent snapshots
            </div>
            {dates.length === 0 ? (
              <div className="px-3 py-3 text-sm text-slate-500">No snapshots yet</div>
            ) : (
              <ul className="max-h-72 overflow-y-auto py-1">
                {dates.map((d, i) => (
                  <li
                    key={d}
                    className="flex items-center justify-between gap-2 px-3 py-1.5 text-sm hover:bg-slate-800/50"
                  >
                    <span className="tabular-nums text-slate-200">{stampLabel(d)}</span>
                    {i === 0 && (
                      <span className="rounded bg-emerald-500/15 px-1.5 py-0.5 text-[10px] font-semibold text-emerald-300 ring-1 ring-inset ring-emerald-500/30">
                        latest
                      </span>
                    )}
                  </li>
                ))}
              </ul>
            )}
          </div>
        </>
      )}
    </div>
  );
}
