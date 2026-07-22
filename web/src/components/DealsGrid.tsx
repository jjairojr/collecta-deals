import { useEffect, useState } from "react";
import { BadgeCheck, ExternalLink, TrendingUp } from "lucide-react";
import { dealImageURL, dealSelection, type Deal } from "../api";
import { brl, usd } from "../format";
import { useSelection } from "../selection";
import { Badge, type BadgeProps } from "./ui/badge";
import { Card } from "./ui/card";
import { Checkbox } from "./ui/checkbox";
import { BrSupply, UsTrust } from "./DealDepth";

const rarityVariant: Record<string, BadgeProps["variant"]> = {
  L: "amber",
  SEC: "fuchsia",
  SR: "violet",
  R: "sky",
  UC: "emerald",
  C: "default",
  P: "rose",
};

function sourceLabel(source: string): string {
  if (source.startsWith("liga")) return "Liga";
  if (source === "mypcards") return "MyP";
  return "BR";
}

function marginTier(margin: number): { text: string; bar: string; ring: string } {
  if (margin >= 100) {
    return { text: "text-emerald-300", bar: "bg-emerald-400", ring: "ring-emerald-400/40" };
  }
  if (margin >= 40) {
    return { text: "text-emerald-300", bar: "bg-emerald-500", ring: "ring-emerald-500/30" };
  }
  if (margin >= 0) {
    return { text: "text-amber-300", bar: "bg-amber-400", ring: "ring-amber-400/30" };
  }
  return { text: "text-rose-300", bar: "bg-rose-500", ring: "ring-rose-500/30" };
}

function DealArt({ deal }: { deal: Deal }) {
  const [failed, setFailed] = useState(false);
  const src = dealImageURL(deal);
  if (!src || failed) {
    return (
      <div className="flex aspect-[350/489] w-full items-center justify-center bg-slate-800 p-3 text-center">
        <span className="font-mono text-xs text-slate-500">{deal.number}</span>
      </div>
    );
  }
  return (
    <img
      src={src}
      alt={deal.name}
      loading="lazy"
      onError={() => setFailed(true)}
      className="aspect-[350/489] w-full bg-slate-800 object-cover"
    />
  );
}

function DealCard({ deal, showDepth }: { deal: Deal; showDepth: boolean }) {
  const tier = marginTier(deal.marginPct);
  const barWidth = Math.max(4, Math.min(100, deal.marginPct));
  const rounded = Math.round(deal.marginPct);
  const { has, toggle } = useSelection();
  const pick = dealSelection(deal);
  return (
    <Card className="group flex flex-col overflow-hidden p-0 transition hover:border-slate-600">
      <div className="relative">
        <DealArt deal={deal} />
        {pick && (
          <label className="absolute left-2 top-2 flex h-7 w-7 cursor-pointer items-center justify-center rounded-lg bg-slate-950/80 ring-1 ring-inset ring-slate-700 backdrop-blur">
            <Checkbox
              accent="emerald"
              checked={has(pick.set, pick.number)}
              onChange={() => toggle(pick)}
            />
          </label>
        )}
        <div
          className={`absolute right-2 top-2 flex items-center gap-1 rounded-lg bg-slate-950/85 px-2 py-1 text-sm font-bold tabular-nums ring-1 ring-inset backdrop-blur ${tier.text} ${tier.ring}`}
        >
          <TrendingUp className="h-3.5 w-3.5" />
          {rounded > 0 ? "+" : ""}
          {rounded}%
        </div>
        <div className="absolute inset-x-0 bottom-0 bg-gradient-to-t from-slate-950/90 to-transparent px-3 pb-2 pt-8">
          <div className="flex items-center gap-1.5">
            <Badge variant={rarityVariant[deal.rarity] ?? "default"}>{deal.rarity || "—"}</Badge>
            {deal.variant && deal.variant !== "Normal" && (
              <span className="text-[11px] text-slate-300">{deal.variant}</span>
            )}
            {deal.verified && (
              <span
                title={`Stock verified live on ${sourceLabel(deal.source)}`}
                className="ml-auto flex items-center gap-1 rounded bg-emerald-500/15 px-1.5 py-0.5 text-[10px] font-medium text-emerald-300 ring-1 ring-inset ring-emerald-500/30"
              >
                <BadgeCheck className="h-3 w-3" />
                Stock
              </span>
            )}
          </div>
        </div>
      </div>

      <div className="flex flex-1 flex-col gap-3 p-3">
        <div>
          <div className="truncate text-sm font-medium text-slate-100" title={deal.name}>
            {deal.name.replace(/\s*\([^)]*\)\s*$/, "")}
          </div>
          <div className="font-mono text-[11px] text-slate-500">
            {deal.set ? `${deal.set} · ${deal.number}` : deal.number}
          </div>
        </div>

        <div className="flex items-end justify-between gap-2 text-sm">
          <div>
            <div className="text-[10px] uppercase tracking-wide text-slate-500">Buy</div>
            <div className="font-semibold tabular-nums text-slate-100">{brl(deal.lowBRL)}</div>
            <div className="text-[11px] tabular-nums text-slate-500">{usd(deal.buyUSD)}</div>
          </div>
          <div className="text-slate-600">→</div>
          <div className="text-right">
            <div className="text-[10px] uppercase tracking-wide text-slate-500">Sell</div>
            <div className="font-semibold tabular-nums text-slate-100">{usd(deal.sellUSD)}</div>
            <div className={`text-[11px] font-medium tabular-nums ${tier.text}`}>
              +{usd(deal.profitUSD)}
            </div>
          </div>
        </div>

        <div className="h-1.5 overflow-hidden rounded-full bg-slate-800">
          <div className={`h-full rounded-full ${tier.bar}`} style={{ width: `${barWidth}%` }} />
        </div>

        <div className="flex flex-wrap items-center justify-between gap-x-2 gap-y-1">
          {showDepth && <BrSupply copies={deal.brCopies} sellers={deal.brSellers} />}
          <UsTrust listings={deal.usListings} qty={deal.usQty} />
        </div>

        <div className="mt-auto flex gap-2">
          <a
            href={deal.buyUrl}
            target="_blank"
            rel="noreferrer"
            className="flex flex-1 items-center justify-center gap-1 rounded-lg bg-emerald-500/10 px-2 py-1.5 text-xs font-medium text-emerald-300 ring-1 ring-inset ring-emerald-500/30 hover:bg-emerald-500/20"
          >
            Buy · {sourceLabel(deal.source)}
          </a>
          <a
            href={deal.tcgUrl}
            target="_blank"
            rel="noreferrer"
            className="flex flex-1 items-center justify-center gap-1 rounded-lg bg-sky-500/10 px-2 py-1.5 text-xs font-medium text-sky-300 ring-1 ring-inset ring-sky-500/30 hover:bg-sky-500/20"
          >
            Sell · TCG <ExternalLink className="h-3 w-3" />
          </a>
        </div>
      </div>
    </Card>
  );
}

export default function DealsGrid({ deals, showDepth = true }: { deals: Deal[]; showDepth?: boolean }) {
  const { backfill } = useSelection();
  useEffect(() => {
    backfill(deals.map(dealSelection).filter((c): c is NonNullable<typeof c> => c !== null));
  }, [deals, backfill]);
  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
      {deals.map((d) => (
        <DealCard key={`${d.number}-${d.tcgUrl}`} deal={d} showDepth={showDepth} />
      ))}
    </div>
  );
}
