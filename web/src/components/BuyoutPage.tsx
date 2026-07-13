import { useEffect, useState } from "react";
import { ExternalLink, LayoutGrid, List } from "lucide-react";
import { getBuyout, getGame, type BuyoutCandidate, type BuyoutMode, type BuyoutSort } from "../api";
import { brl, usd } from "../format";
import { useSets } from "../useSets";
import { useSelection, type PickedCard } from "../selection";
import BuyoutGrid from "./BuyoutGrid";
import SetSelect, { ALL_SETS } from "./SetSelect";
import { Badge } from "./ui/badge";
import { Card } from "./ui/card";
import { Checkbox } from "./ui/checkbox";
import { Input } from "./ui/input";
import { Select } from "./ui/select";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "./ui/table";
import { ToggleGroup } from "./ui/toggle-group";

function isBuyoutSort(v: string): v is BuyoutSort {
  return v === "best" || v === "score" || v === "lift" || v === "profit" || v === "copies";
}

const mainCharHints: Record<string, string> = {
  onepiece: "(Straw Hats + Hancock + Law)",
  pokemon: "(Pikachu, Charizard, Eeveelutions…)",
  riftbound: "(Jinx, Ahri, Yasuo, Viktor…)",
  lorcana: "(Mickey, Elsa, Stitch, Maleficent…)",
  gundam: "(RX-78, Char, Unicorn, Barbatos…)",
};

export default function BuyoutPage() {
  const sets = useSets();
  const [set, setSet] = useState(ALL_SETS);
  const [mode, setMode] = useState<BuyoutMode>("buyout");
  const [budget, setBudget] = useState(500);
  const [minFloor, setMinFloor] = useState(20);
  const [minGap, setMinGap] = useState(50);
  const [shipping, setShipping] = useState(15);
  const [sort, setSort] = useState<BuyoutSort>("best");
  const [mainChars, setMainChars] = useState(false);
  const [spOnly, setSpOnly] = useState(false);
  const snipe = mode === "snipe";
  const isOnePiece = getGame() === "onepiece";
  const [layout, setLayout] = useState<"grid" | "table">("grid");
  const [rows, setRows] = useState<BuyoutCandidate[]>([]);
  const [date, setDate] = useState("");
  const [fxRate, setFxRate] = useState(0);
  const [ready, setReady] = useState(true);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const { has, toggle, backfill } = useSelection();

  useEffect(() => {
    if (sets.length > 0 && set !== ALL_SETS && !sets.includes(set)) {
      setSet(sets[0]);
    }
  }, [sets, set]);

  useEffect(() => {
    backfill(
      rows.map((c) => ({
        set: c.set ?? set,
        number: c.number,
        name: c.name,
        priceBRL: c.floor,
      })),
    );
  }, [rows, set, backfill]);

  useEffect(() => {
    const id = window.setTimeout(() => {
      let current = true;
      setLoading(true);
      setError(null);
      getBuyout(set, budget, minFloor, shipping, sort, mainChars, mode, minGap, spOnly)
        .then((r) => {
          if (current) {
            setRows(r.candidates);
            setDate(r.date);
            setFxRate(r.fxRate);
            setReady(r.ready);
          }
        })
        .catch((err: unknown) =>
          setError(err instanceof Error ? err.message : "failed to load buyout"),
        )
        .finally(() => current && setLoading(false));
      return () => {
        current = false;
      };
    }, 250);
    return () => window.clearTimeout(id);
  }, [set, budget, minFloor, shipping, sort, mainChars, mode, minGap, spOnly]);

  function switchMode(next: BuyoutMode) {
    setMode(next);
    setSort(next === "snipe" ? "lift" : "best");
  }

  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="text-lg font-semibold text-slate-100">
            {snipe ? "Snipe underpriced cards" : "Best Cards to Buyout"}
          </h1>
          <p className="mt-1 max-w-2xl text-sm text-slate-400">
            {snipe ? (
              <>
                Cards where the cheapest NM listing is at least your{" "}
                <span className="text-slate-200">min gap</span> below the next price — a seller
                mispriced a copy. Buy every copy at that floor and flip near the next listing. Ranked
                by biggest gap; shipping is subtracted from profit.
              </>
            ) : (
              <>
                Cards where clearing the cheapest NM listings pushes the floor up the most for
                your budget. A thick cluster of cheap copies scores low — it eats the budget for little
                lift. Each distinct store you buy from adds one shipping fee, subtracted from profit.
              </>
            )}
          </p>
        </div>
        <ToggleGroup
          value={mode}
          onChange={(v) => switchMode(v === "snipe" ? "snipe" : "buyout")}
          options={[
            { value: "buyout", label: "Buyout" },
            { value: "snipe", label: "Sniper" },
          ]}
        />
      </div>

      <Card className="flex flex-wrap items-end gap-4 p-4">
        <SetSelect sets={sets} value={set} onChange={setSet} allowAll />
        {snipe ? (
          <ControlField label="Min gap (%)">
            <Input
              type="number"
              value={minGap}
              min={0}
              step={10}
              onChange={(e) => setMinGap(Math.max(0, Number(e.target.value)))}
              className="w-28"
            />
          </ControlField>
        ) : (
          <ControlField label="Buyout budget (R$)">
            <Input
              type="number"
              value={budget}
              min={0}
              step={50}
              onChange={(e) => setBudget(Math.max(0, Number(e.target.value)))}
              className="w-32"
            />
          </ControlField>
        )}
        <ControlField label="Min floor (R$)">
          <Input
            type="number"
            value={minFloor}
            min={0}
            step={5}
            onChange={(e) => setMinFloor(Math.max(0, Number(e.target.value)))}
            className="w-28"
          />
        </ControlField>
        <ControlField label="Shipping / store (R$)">
          <Input
            type="number"
            value={shipping}
            min={0}
            step={1}
            onChange={(e) => setShipping(Math.max(0, Number(e.target.value)))}
            className="w-32"
          />
        </ControlField>
        <ControlField label="Order by">
          <Select
            value={sort}
            onChange={(e) => {
              if (isBuyoutSort(e.target.value)) {
                setSort(e.target.value);
              }
            }}
            className="w-60"
          >
            <option value="best">Best (few copies · high lift/profit)</option>
            <option value="score">Corner score</option>
            <option value="lift">Highest lift %</option>
            <option value="profit">Highest profit</option>
            <option value="copies">Fewest copies</option>
          </Select>
        </ControlField>
        <label className="flex cursor-pointer items-center gap-2 pb-1.5 text-xs text-slate-300">
          <Checkbox
            accent="emerald"
            checked={mainChars}
            onChange={(e) => setMainChars(e.target.checked)}
          />
          Main characters only
          <span className="text-slate-500">{mainCharHints[getGame()] ?? mainCharHints.onepiece}</span>
        </label>
        {isOnePiece && (
          <label className="flex cursor-pointer items-center gap-2 pb-1.5 text-xs text-slate-300">
            <Checkbox accent="emerald" checked={spOnly} onChange={(e) => setSpOnly(e.target.checked)} />
            SP only
            <span className="text-slate-500">(special parallel art)</span>
          </label>
        )}
      </Card>

      {error ? (
        <Panel>Could not load: {error}</Panel>
      ) : !ready ? (
        <Panel>No priced snapshot yet — this needs a capture that includes the price ladder.</Panel>
      ) : rows.length === 0 ? (
        <Panel>
          {snipe
            ? "No snipes at this gap and floor. Try lowering the min gap %."
            : "No buyout candidates for this budget and floor. Try raising the budget."}
        </Panel>
      ) : (
        <>
          <div className="flex items-center justify-between text-sm text-slate-400">
            <span>
              <span className="font-medium text-slate-200">{rows.length}</span> candidate
              {rows.length === 1 ? "" : "s"}
              {date && <span className="ml-2 text-xs text-slate-500">snapshot {date}</span>}
              {loading && <span className="ml-2 text-xs text-slate-500">updating…</span>}
            </span>
            <div className="inline-flex rounded-lg border border-slate-700 bg-slate-900 p-0.5">
              <button
                onClick={() => setLayout("grid")}
                aria-label="Grid view"
                className={`rounded-md p-1.5 transition-colors ${
                  layout === "grid"
                    ? "bg-emerald-500/20 text-emerald-200"
                    : "text-slate-400 hover:text-slate-200"
                }`}
              >
                <LayoutGrid className="h-4 w-4" />
              </button>
              <button
                onClick={() => setLayout("table")}
                aria-label="Table view"
                className={`rounded-md p-1.5 transition-colors ${
                  layout === "table"
                    ? "bg-emerald-500/20 text-emerald-200"
                    : "text-slate-400 hover:text-slate-200"
                }`}
              >
                <List className="h-4 w-4" />
              </button>
            </div>
          </div>

          {layout === "grid" ? (
            <BuyoutGrid rows={rows} set={set} mode={mode} fxRate={fxRate} has={has} toggle={toggle} />
          ) : (
            <BuyoutTable rows={rows} set={set} fxRate={fxRate} has={has} toggle={toggle} />
          )}
        </>
      )}
      <p className="text-xs text-slate-500">
        {snipe ? (
          <>
            "Next" is the second-cheapest NM price — the market ceiling proven by another
            listing. Others can relist below you, so treat it as a realistic target, not a guaranteed
            price. Profit assumes you buy every copy at the floor and resell near the next price, net
            of one shipping fee per store. TCG sell is the current lowest US listing (Near Mint).
          </>
        ) : (
          <>
            "New floor" is the next NM listing after clearing the cheap ones — others can still relist
            below you, so treat it as the realistic ceiling, not a guaranteed price. Profit assumes you
            resell every cleared copy at the new floor. TCG sell is the current lowest US listing (Near
            Mint) — headroom compares it against the cornered floor in dollars. Demand isn't fully
            modeled yet (sellers used as a liquidity proxy); day-over-day sales velocity will sharpen
            the score once there are 2+ days.
          </>
        )}
      </p>
    </div>
  );
}

function ControlField({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <label className="flex flex-col gap-1.5 text-[10px] font-medium uppercase tracking-wide text-slate-500">
      {label}
      {children}
    </label>
  );
}

function BuyoutTable({
  rows,
  set,
  fxRate,
  has,
  toggle,
}: {
  rows: BuyoutCandidate[];
  set: string;
  fxRate: number;
  has: (set: string, number: string) => boolean;
  toggle: (item: PickedCard) => void;
}) {
  return (
    <div className="overflow-x-auto rounded-xl border border-slate-800 bg-slate-900/40">
      <Table className="min-w-[1040px]">
        <TableHeader>
          <TableRow className="hover:bg-transparent">
            <TableHead className="px-3"></TableHead>
            {set === ALL_SETS && <TableHead className="px-3">Set</TableHead>}
            <TableHead className="px-3">Card</TableHead>
            <TableHead className="px-3">Name</TableHead>
            <TableHead className="px-3 text-right">Floor → next</TableHead>
            <TableHead className="px-3 text-right">Lift</TableHead>
            {fxRate > 0 && <TableHead className="px-3 text-right">TCG sell</TableHead>}
            <TableHead className="px-3 text-right">Cards</TableHead>
            <TableHead className="px-3 text-right">Ship</TableHead>
            <TableHead className="px-3 text-right">Copies</TableHead>
            <TableHead className="px-3 text-right">Stores</TableHead>
            <TableHead className="px-3 text-right">Net profit</TableHead>
            <TableHead className="px-3 text-right">NM supply</TableHead>
            <TableHead className="px-3 text-right">Sellers</TableHead>
            <TableHead className="px-3 text-right">Link</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((c) => (
            <TableRow key={`${c.set ?? ""}${c.number}`}>
              <TableCell className="px-3 py-2.5">
                <Checkbox
                  accent="emerald"
                  checked={has(c.set ?? set, c.number)}
                  onChange={() =>
                    toggle({ set: c.set ?? set, number: c.number, name: c.name, priceBRL: c.floor })
                  }
                />
              </TableCell>
              {set === ALL_SETS && (
                <TableCell className="whitespace-nowrap px-3 py-2.5 font-mono text-xs text-slate-400">
                  {c.set}
                </TableCell>
              )}
              <TableCell className="whitespace-nowrap px-3 py-2.5 font-mono text-xs text-slate-300">
                {c.number}
              </TableCell>
              <TableCell className="px-3 py-2.5 text-slate-100">{c.name}</TableCell>
              <TableCell className="whitespace-nowrap px-3 py-2.5 text-right tabular-nums text-slate-300">
                {brl(c.floor)} <span className="text-slate-500">→</span> {brl(c.nextFloor)}
              </TableCell>
              <TableCell className="whitespace-nowrap px-3 py-2.5 text-right">
                <Badge variant="emerald">+{Math.round(c.liftPct)}%</Badge>
              </TableCell>
              {fxRate > 0 && (
                <TableCell className="whitespace-nowrap px-3 py-2.5 text-right tabular-nums text-sky-200">
                  {c.tcgUrl && c.sellUSD ? (
                    <a
                      href={c.tcgUrl}
                      target="_blank"
                      rel="noreferrer"
                      className="inline-flex items-center gap-1 hover:underline"
                    >
                      {usd(c.sellUSD)} <ExternalLink className="h-3 w-3" />
                    </a>
                  ) : (
                    <span className="text-slate-600">—</span>
                  )}
                </TableCell>
              )}
              <TableCell className="whitespace-nowrap px-3 py-2.5 text-right tabular-nums text-slate-300">
                {brl(c.buyoutCost)}
              </TableCell>
              <TableCell className="whitespace-nowrap px-3 py-2.5 text-right tabular-nums text-slate-400">
                {brl(c.shippingCost)}
              </TableCell>
              <TableCell className="whitespace-nowrap px-3 py-2.5 text-right tabular-nums text-slate-400">
                {c.copiesToClear}
              </TableCell>
              <TableCell className="whitespace-nowrap px-3 py-2.5 text-right tabular-nums text-slate-400">
                {c.storeCount}
              </TableCell>
              <TableCell
                className={`whitespace-nowrap px-3 py-2.5 text-right tabular-nums ${c.profitBRL >= 0 ? "text-emerald-200" : "text-rose-300"}`}
              >
                {brl(c.profitBRL)}
              </TableCell>
              <TableCell className="whitespace-nowrap px-3 py-2.5 text-right tabular-nums text-slate-400">
                {c.nmSupply}
              </TableCell>
              <TableCell className="whitespace-nowrap px-3 py-2.5 text-right tabular-nums text-slate-400">
                {c.sellers}
              </TableCell>
              <TableCell className="whitespace-nowrap px-3 py-2.5 text-right">
                <a
                  href={c.url}
                  target="_blank"
                  rel="noreferrer"
                  className="rounded bg-emerald-500/10 px-2 py-1 text-xs font-medium text-emerald-300 ring-1 ring-inset ring-emerald-500/30 hover:bg-emerald-500/20"
                >
                  Liga
                </a>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

function Panel({ children }: { children: React.ReactNode }) {
  return (
    <div className="rounded-xl border border-slate-800 bg-slate-900/40 px-4 py-10 text-center text-slate-400">
      {children}
    </div>
  );
}
