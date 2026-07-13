import { Fragment, useState } from "react";
import { ChevronDown, ChevronRight } from "lucide-react";
import type { StoreStat } from "../api";
import { brl } from "../format";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "./ui/table";

export default function StoreLeaderboard({
  stores,
  sort,
  onSortChange,
}: {
  stores: StoreStat[];
  sort: "units" | "revenue";
  onSortChange: (sort: "units" | "revenue") => void;
}) {
  const [expanded, setExpanded] = useState<number | null>(null);

  return (
    <div>
      <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
        <h2 className="text-sm font-semibold text-slate-200">Top selling stores</h2>
        <div className="inline-flex rounded-lg border border-slate-700 bg-slate-900 p-0.5 text-xs">
          <button
            onClick={() => onSortChange("units")}
            className={`rounded-md px-3 py-1 font-medium transition-colors ${
              sort === "units" ? "bg-sky-500/20 text-sky-200" : "text-slate-400 hover:text-slate-200"
            }`}
          >
            Units
          </button>
          <button
            onClick={() => onSortChange("revenue")}
            className={`rounded-md px-3 py-1 font-medium transition-colors ${
              sort === "revenue"
                ? "bg-sky-500/20 text-sky-200"
                : "text-slate-400 hover:text-slate-200"
            }`}
          >
            Revenue
          </button>
        </div>
      </div>
      {stores.length === 0 ? (
        <div className="rounded-xl border border-slate-800 bg-slate-900/40 px-4 py-10 text-center text-slate-400">
          No sales inferred yet in this range.
        </div>
      ) : (
        <div className="overflow-x-auto rounded-xl border border-slate-800 bg-slate-900/40">
          <Table className="min-w-[520px]">
            <TableHeader>
              <TableRow className="hover:bg-transparent">
                <TableHead>#</TableHead>
                <TableHead>Store</TableHead>
                <TableHead className="text-right">Units sold</TableHead>
                <TableHead className="text-right">Revenue</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {stores.map((s, i) => {
                const open = expanded === s.storeId;
                return (
                  <Fragment key={s.storeId}>
                    <TableRow
                      onClick={() => setExpanded(open ? null : s.storeId)}
                      className="cursor-pointer"
                    >
                      <TableCell className="tabular-nums text-slate-500">{i + 1}</TableCell>
                      <TableCell className="text-slate-100">
                        <span className="mr-1.5 inline-flex w-3 align-middle text-slate-500">
                          {open ? (
                            <ChevronDown className="h-3.5 w-3.5" />
                          ) : (
                            <ChevronRight className="h-3.5 w-3.5" />
                          )}
                        </span>
                        {s.storeName || `Store ${s.storeId}`}
                        <span className="ml-2 text-xs text-slate-500">
                          {s.cards.length} card{s.cards.length === 1 ? "" : "s"}
                        </span>
                      </TableCell>
                      <TableCell
                        className={`whitespace-nowrap text-right font-semibold tabular-nums ${
                          sort === "units" ? "text-sky-200" : "text-slate-200"
                        }`}
                      >
                        {s.unitsSold}
                      </TableCell>
                      <TableCell
                        className={`whitespace-nowrap text-right tabular-nums ${
                          sort === "revenue" ? "text-emerald-300" : "text-slate-300"
                        }`}
                      >
                        {brl(s.revenueBRL)}
                      </TableCell>
                    </TableRow>
                    {open && (
                      <TableRow className="bg-slate-900/60 hover:bg-slate-900/60">
                        <TableCell />
                        <TableCell colSpan={3}>
                          <div className="mb-1 text-xs uppercase tracking-wide text-slate-500">
                            Cards sold
                          </div>
                          <table className="w-full text-xs">
                            <tbody>
                              {s.cards.map((c) => (
                                <tr key={c.number} className="text-slate-300">
                                  <td className="py-0.5 pr-3 font-mono text-slate-400">{c.number}</td>
                                  <td className="py-0.5 pr-3">{c.name}</td>
                                  <td className="py-0.5 pr-3 text-right tabular-nums text-sky-200">
                                    {c.units}×
                                  </td>
                                  <td className="py-0.5 pr-3 text-right tabular-nums text-slate-400">
                                    {c.units > 0 ? `@ ${brl(c.revenueBRL / c.units)}` : ""}
                                  </td>
                                  <td className="py-0.5 text-right tabular-nums text-emerald-300">
                                    {brl(c.revenueBRL)}
                                  </td>
                                </tr>
                              ))}
                            </tbody>
                          </table>
                        </TableCell>
                      </TableRow>
                    )}
                  </Fragment>
                );
              })}
            </TableBody>
          </Table>
        </div>
      )}
    </div>
  );
}
