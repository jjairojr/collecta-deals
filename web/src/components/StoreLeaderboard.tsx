import { Fragment, useState } from "react";
import EmptyState from "./EmptyState";
import { ChevronDown, ChevronRight } from "lucide-react";
import type { StoreStat } from "../api";
import { brl } from "../format";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "./ui/table";
import { Window } from "./ui/card";
import { ToggleGroup } from "./ui/toggle-group";

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
    <Window
      className="sticker-sm"
      title="Top selling stores"
      actions={
        <ToggleGroup
          value={sort}
          onChange={(v) => onSortChange(v === "revenue" ? "revenue" : "units")}
          options={[
            { value: "units", label: "Units" },
            { value: "revenue", label: "Revenue" },
          ]}
        />
      }
    >
      {stores.length === 0 ? (
        <EmptyState bare>No sales inferred yet in this range.</EmptyState>
      ) : (
        <div className="overflow-x-auto">
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
    </Window>
  );
}
