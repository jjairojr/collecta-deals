import { useEffect } from "react";
import { BadgeCheck } from "lucide-react";
import { dealSelection, type Deal } from "../api";
import { brl, pct, usd } from "../format";
import { useSelection } from "../selection";
import { Badge, type BadgeProps } from "./ui/badge";
import { Checkbox } from "./ui/checkbox";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "./ui/table";
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

function RarityBadge({ rarity }: { rarity: string }) {
  return <Badge variant={rarityVariant[rarity] ?? "default"}>{rarity || "—"}</Badge>;
}

function marginColor(margin: number): string {
  if (margin >= 100) return "text-emerald-400";
  if (margin >= 30) return "text-emerald-300";
  if (margin >= 0) return "text-amber-300";
  return "text-rose-400";
}

function sourceLabel(source: string): string {
  if (source.startsWith("liga")) return "Liga";
  if (source === "mypcards") return "MYP";
  return "BR";
}

export default function DealsTable({ deals }: { deals: Deal[] }) {
  const { has, toggle, backfill } = useSelection();
  useEffect(() => {
    backfill(deals.map(dealSelection).filter((c): c is NonNullable<typeof c> => c !== null));
  }, [deals, backfill]);
  return (
    <div className="overflow-x-auto rounded-xl border border-slate-800 bg-slate-900/40">
      <Table className="min-w-[1040px]">
        <TableHeader>
          <TableRow className="hover:bg-transparent">
            <TableHead className="px-3"></TableHead>
            <TableHead>Card</TableHead>
            <TableHead>Name</TableHead>
            <TableHead>Rarity</TableHead>
            <TableHead className="text-right">Brazil</TableHead>
            <TableHead>BR supply</TableHead>
            <TableHead className="text-right">Buy (US$)</TableHead>
            <TableHead className="text-right">Sell (US$)</TableHead>
            <TableHead>US depth</TableHead>
            <TableHead className="text-right">Margin</TableHead>
            <TableHead className="text-right">Profit</TableHead>
            <TableHead className="text-right">Links</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {deals.map((d) => {
            const pick = dealSelection(d);
            return (
            <TableRow key={`${d.number}-${d.tcgUrl}`}>
              <TableCell className="px-3 py-2.5">
                {pick && (
                  <Checkbox
                    accent="emerald"
                    checked={has(pick.set, pick.number)}
                    onChange={() => toggle(pick)}
                  />
                )}
              </TableCell>
              <TableCell className="whitespace-nowrap font-mono text-xs text-slate-300">
                <span className="flex items-center gap-1">
                  {d.set && <span className="text-slate-500">{d.set} ·</span>}
                  {d.number}
                  {d.verified && (
                    <BadgeCheck
                      className="h-3.5 w-3.5 text-emerald-400"
                      aria-label="Stock verified"
                    />
                  )}
                </span>
              </TableCell>
              <TableCell className="text-slate-100">{d.name}</TableCell>
              <TableCell>
                <RarityBadge rarity={d.rarity} />
                {d.variant && d.variant !== "Normal" ? (
                  <span className="ml-1 text-xs text-slate-500">{d.variant}</span>
                ) : null}
              </TableCell>
              <TableCell className="whitespace-nowrap text-right tabular-nums text-slate-300">
                {brl(d.lowBRL)}
              </TableCell>
              <TableCell className="whitespace-nowrap">
                <BrSupply copies={d.brCopies} sellers={d.brSellers} />
              </TableCell>
              <TableCell className="whitespace-nowrap text-right tabular-nums text-slate-400">
                {usd(d.buyUSD)}
              </TableCell>
              <TableCell className="whitespace-nowrap text-right tabular-nums text-slate-100">
                {usd(d.sellUSD)}
              </TableCell>
              <TableCell className="whitespace-nowrap">
                <UsTrust listings={d.usListings} qty={d.usQty} />
              </TableCell>
              <TableCell
                className={`whitespace-nowrap text-right font-semibold tabular-nums ${marginColor(d.marginPct)}`}
              >
                {pct(d.marginPct)}
              </TableCell>
              <TableCell
                className={`whitespace-nowrap text-right tabular-nums ${d.profitUSD >= 0 ? "text-emerald-300" : "text-rose-400"}`}
              >
                {usd(d.profitUSD)}
              </TableCell>
              <TableCell className="whitespace-nowrap text-right">
                <div className="flex justify-end gap-1.5">
                  <a
                    href={d.tcgUrl}
                    target="_blank"
                    rel="noreferrer"
                    className="rounded bg-sky-500/10 px-2 py-1 text-xs font-medium text-sky-300 ring-1 ring-inset ring-sky-500/30 hover:bg-sky-500/20"
                  >
                    TCG
                  </a>
                  <a
                    href={d.buyUrl}
                    target="_blank"
                    rel="noreferrer"
                    className="rounded bg-emerald-500/10 px-2 py-1 text-xs font-medium text-emerald-300 ring-1 ring-inset ring-emerald-500/30 hover:bg-emerald-500/20"
                  >
                    {sourceLabel(d.source)}
                  </a>
                </div>
              </TableCell>
            </TableRow>
            );
          })}
        </TableBody>
      </Table>
    </div>
  );
}
