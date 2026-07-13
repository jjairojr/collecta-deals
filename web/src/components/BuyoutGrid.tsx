import { ArrowRight, ExternalLink, Package, Store, TrendingUp } from "lucide-react";
import { type BuyoutCandidate, type BuyoutMode } from "../api";
import { brl, usd } from "../format";
import { type PickedCard } from "../selection";
import CardArt from "./CardArt";
import { Card } from "./ui/card";
import { Checkbox } from "./ui/checkbox";

function liftTier(lift: number): { text: string; bar: string; ring: string } {
  if (lift >= 60) {
    return { text: "text-emerald-300", bar: "bg-emerald-400", ring: "ring-emerald-400/40" };
  }
  if (lift >= 25) {
    return { text: "text-emerald-300", bar: "bg-emerald-500", ring: "ring-emerald-500/30" };
  }
  return { text: "text-amber-300", bar: "bg-amber-400", ring: "ring-amber-400/30" };
}

function BuyoutCard({
  cand,
  set,
  mode,
  fxRate,
  picked,
  onToggle,
}: {
  cand: BuyoutCandidate;
  set: string;
  mode: BuyoutMode;
  fxRate: number;
  picked: boolean;
  onToggle: () => void;
}) {
  const cardSet = cand.set ?? set;
  const tier = liftTier(cand.liftPct);
  const barWidth = Math.max(6, Math.min(100, cand.liftPct));
  const totalCost = cand.buyoutCost + cand.shippingCost;
  const floorUSD = fxRate > 0 ? cand.floor * fxRate : 0;
  const headroom = cand.sellUSD && floorUSD ? cand.sellUSD - floorUSD : 0;

  return (
    <Card className="group flex flex-col overflow-hidden p-0 transition hover:border-slate-600">
      <div className="relative">
        <CardArt
          set={cardSet}
          number={cand.number}
          name={cand.name}
          className="aspect-[350/489] w-full"
        />
        <label className="absolute left-2 top-2 flex h-7 w-7 cursor-pointer items-center justify-center rounded-lg bg-slate-950/80 ring-1 ring-inset ring-slate-700 backdrop-blur">
          <Checkbox accent="emerald" checked={picked} onChange={onToggle} />
        </label>
        <div
          className={`absolute right-2 top-2 flex items-center gap-1 rounded-lg bg-slate-950/85 px-2 py-1 text-sm font-bold tabular-nums ring-1 ring-inset backdrop-blur ${tier.text} ${tier.ring}`}
        >
          <TrendingUp className="h-3.5 w-3.5" />+{Math.round(cand.liftPct)}%
        </div>
        <div className="absolute inset-x-0 bottom-0 flex items-center gap-1.5 bg-gradient-to-t from-slate-950/90 to-transparent px-3 pb-2 pt-8">
          {cardSet && (
            <span className="font-mono text-[11px] font-medium text-slate-300">{cardSet}</span>
          )}
          <span className="font-mono text-[11px] text-slate-500">{cand.number}</span>
        </div>
      </div>

      <div className="flex flex-1 flex-col gap-3 p-3">
        <div className="truncate text-sm font-medium text-slate-100" title={cand.name}>
          {cand.name.replace(/\s*\([^)]*\)\s*$/, "")}
        </div>

        <div>
          <div className="flex items-center justify-between text-sm">
            <div>
              <div className="text-[10px] uppercase tracking-wide text-slate-500">
                {mode === "snipe" ? "Cheapest" : "Floor now"}
              </div>
              <div className="font-semibold tabular-nums text-slate-100">{brl(cand.floor)}</div>
            </div>
            <ArrowRight className={`h-4 w-4 ${tier.text}`} />
            <div className="text-right">
              <div className="text-[10px] uppercase tracking-wide text-slate-500">
                {mode === "snipe" ? "Next price" : "New floor"}
              </div>
              <div className={`font-semibold tabular-nums ${tier.text}`}>{brl(cand.nextFloor)}</div>
            </div>
          </div>
          <div className="mt-2 h-1.5 overflow-hidden rounded-full bg-slate-800">
            <div className={`h-full rounded-full ${tier.bar}`} style={{ width: `${barWidth}%` }} />
          </div>
        </div>

        <div className="rounded-lg bg-slate-800/40 px-2.5 py-2 text-xs text-slate-400">
          <div className="flex items-center gap-3">
            <span className="flex items-center gap-1">
              <Package className="h-3.5 w-3.5" />
              {cand.copiesToClear} {cand.copiesToClear === 1 ? "copy" : "copies"}
            </span>
            <span className="flex items-center gap-1">
              <Store className="h-3.5 w-3.5" />
              {cand.storeCount} {cand.storeCount === 1 ? "store" : "stores"}
            </span>
          </div>
          <div className="mt-1.5 flex items-baseline justify-between">
            <span>
              {brl(totalCost)}{" "}
              <span className="text-[11px] text-slate-500">
                ({brl(cand.buyoutCost)} + {brl(cand.shippingCost)} ship)
              </span>
            </span>
            <span
              className={`font-semibold tabular-nums ${cand.profitBRL >= 0 ? "text-emerald-300" : "text-rose-300"}`}
            >
              {cand.profitBRL >= 0 ? "+" : ""}
              {brl(cand.profitBRL)}
            </span>
          </div>
        </div>

        {fxRate > 0 && (
          <div className="flex items-center justify-between rounded-lg bg-sky-500/5 px-2.5 py-2 text-xs ring-1 ring-inset ring-sky-500/15">
            <div>
              <div className="text-[10px] uppercase tracking-wide text-slate-500">TCG sell</div>
              {cand.sellUSD ? (
                <div className="font-semibold tabular-nums text-sky-200">{usd(cand.sellUSD)}</div>
              ) : (
                <div className="text-slate-500">no US match</div>
              )}
            </div>
            {floorUSD > 0 && (
              <div className="text-right">
                <div className="text-[10px] uppercase tracking-wide text-slate-500">Floor ≈</div>
                <div className="tabular-nums text-slate-300">{usd(floorUSD)}</div>
              </div>
            )}
            {headroom > 0 && (
              <div className="text-right">
                <div className="text-[10px] uppercase tracking-wide text-slate-500">Headroom</div>
                <div className="font-medium tabular-nums text-emerald-300">+{usd(headroom)}</div>
              </div>
            )}
          </div>
        )}

        <div className="flex items-center justify-between text-[11px] text-slate-500">
          <span>
            {cand.nmSupply} NM in stock · {cand.sellers} sellers
          </span>
        </div>

        <div className="mt-auto flex gap-2">
          <a
            href={cand.url}
            target="_blank"
            rel="noreferrer"
            className="flex flex-1 items-center justify-center gap-1 rounded-lg bg-emerald-500/10 px-2 py-1.5 text-xs font-medium text-emerald-300 ring-1 ring-inset ring-emerald-500/30 hover:bg-emerald-500/20"
          >
            Buy · Liga
          </a>
          {cand.tcgUrl && (
            <a
              href={cand.tcgUrl}
              target="_blank"
              rel="noreferrer"
              className="flex flex-1 items-center justify-center gap-1 rounded-lg bg-sky-500/10 px-2 py-1.5 text-xs font-medium text-sky-300 ring-1 ring-inset ring-sky-500/30 hover:bg-sky-500/20"
            >
              TCG <ExternalLink className="h-3 w-3" />
            </a>
          )}
        </div>
      </div>
    </Card>
  );
}

export default function BuyoutGrid({
  rows,
  set,
  mode,
  fxRate,
  has,
  toggle,
}: {
  rows: BuyoutCandidate[];
  set: string;
  mode: BuyoutMode;
  fxRate: number;
  has: (set: string, number: string) => boolean;
  toggle: (item: PickedCard) => void;
}) {
  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
      {rows.map((c) => {
        const cardSet = c.set ?? set;
        return (
          <BuyoutCard
            key={`${c.set ?? ""}${c.number}`}
            cand={c}
            set={set}
            mode={mode}
            fxRate={fxRate}
            picked={has(cardSet, c.number)}
            onToggle={() => toggle({ set: cardSet, number: c.number, name: c.name, priceBRL: c.floor })}
          />
        );
      })}
    </div>
  );
}
