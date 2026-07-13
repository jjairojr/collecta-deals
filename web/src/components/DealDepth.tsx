import { AlertTriangle, Layers, Store } from "lucide-react";

// UsTrust turns the US comparison price from a bare number into a trust signal:
// how many gold-star NM listings back the live floor. One listing is an outlier,
// not a market; many listings mean the gap you see is real. Zero = the card was
// not live-priced (e.g. an arbitrary search result), shown as unknown, never fake.
export function UsTrust({ listings, qty }: { listings: number; qty: number }) {
  if (listings <= 0) {
    return <span className="text-[11px] text-slate-600">US depth —</span>;
  }
  const thin = listings === 1;
  const deep = listings >= 5;
  const tone = thin
    ? "text-amber-300 ring-amber-500/30 bg-amber-500/10"
    : deep
      ? "text-emerald-300 ring-emerald-500/30 bg-emerald-500/10"
      : "text-sky-300 ring-sky-500/30 bg-sky-500/10";
  return (
    <span
      title={`US floor backed by ${listings} gold-star NM listing${listings === 1 ? "" : "s"}${
        qty > 0 ? `, ${qty} copies available` : ""
      }${thin ? " — one seller, treat the gap with caution" : ""}`}
      className={`inline-flex items-center gap-1 rounded px-1.5 py-0.5 text-[10px] font-medium tabular-nums ring-1 ring-inset ${tone}`}
    >
      {thin ? <AlertTriangle className="h-3 w-3" /> : <Layers className="h-3 w-3" />}
      {thin ? "thin · 1 US listing" : deep ? `deep · ${listings} US` : `${listings} US listings`}
    </span>
  );
}

// BrSupply shows how much stock actually sits at the Brazil floor: copies at the
// floor price and distinct sellers. It answers "one lucky flip or real supply I can
// load up on". Zero sellers = stock not verified (unknown).
export function BrSupply({ copies, sellers }: { copies: number; sellers: number }) {
  if (sellers <= 0) {
    return <span className="text-[11px] text-slate-600">BR supply —</span>;
  }
  const single = copies <= 1 && sellers <= 1;
  return (
    <span
      title={`${copies} cop${copies === 1 ? "y" : "ies"} at the BR floor across ${sellers} seller${
        sellers === 1 ? "" : "s"
      }${single ? " — a single flip, not a stack" : ""}`}
      className={`inline-flex items-center gap-1 text-[11px] tabular-nums ${
        single ? "text-amber-300" : "text-slate-400"
      }`}
    >
      <Store className="h-3 w-3" />
      {copies > 0 ? `${copies} @ floor` : "in stock"} · {sellers} seller{sellers === 1 ? "" : "s"}
    </span>
  );
}
