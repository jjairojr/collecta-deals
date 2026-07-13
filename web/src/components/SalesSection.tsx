import { useMemo, useState } from "react";
import { ShoppingBag } from "lucide-react";
import type { SnapshotSales } from "../api";
import { mergeSnapshotCards } from "../salesutil";
import SoldCardTile from "./SoldCardTile";
import SalesBySnapshot from "./SalesBySnapshot";
import { ToggleGroup } from "./ui/toggle-group";

type View = "totals" | "snapshot";

function isView(v: string): v is View {
  return v === "totals" || v === "snapshot";
}

export default function SalesSection({
  snapshots,
  set,
  loaded,
}: {
  snapshots: SnapshotSales[];
  set: string;
  loaded: boolean;
}) {
  const [view, setView] = useState<View>("totals");
  const totals = useMemo(() => mergeSnapshotCards(snapshots), [snapshots]);

  return (
    <section className="space-y-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <span className="text-sky-300">
            <ShoppingBag className="h-4 w-4" />
          </span>
          <h2 className="text-sm font-semibold text-slate-200">What's selling</h2>
          <span className="text-xs text-slate-500">
            · {view === "totals" ? "top sellers in range, by revenue" : "cards sold per snapshot"}
          </span>
        </div>
        <ToggleGroup
          value={view}
          onChange={(v) => isView(v) && setView(v)}
          options={[
            { value: "totals", label: "Totals" },
            { value: "snapshot", label: "By snapshot" },
          ]}
        />
      </div>

      {!loaded ? (
        <div className="rounded-xl border border-slate-800 bg-slate-900/30 px-4 py-10 text-center text-sm text-slate-500">
          Loading sales…
        </div>
      ) : view === "totals" ? (
        totals.length === 0 ? (
          <Empty />
        ) : (
          <div className="grid grid-cols-3 gap-3 sm:grid-cols-4 md:grid-cols-6">
            {totals.map((c) => (
              <SoldCardTile key={`${c.set ?? ""}${c.number}`} card={c} set={set} />
            ))}
          </div>
        )
      ) : (
        <SalesBySnapshot snapshots={snapshots} set={set} />
      )}
    </section>
  );
}

function Empty() {
  return (
    <div className="flex flex-col items-center gap-2 rounded-xl border border-dashed border-slate-800 bg-slate-900/30 py-12 text-center">
      <p className="font-medium text-slate-300">No sales in this range yet</p>
      <p className="max-w-md text-sm text-slate-500">
        Sales are inferred from day-over-day per-store stock drops — they appear once there are at
        least two snapshots in the selected window.
      </p>
    </div>
  );
}
