import { useEffect, useState } from "react";
import { Coins, PiggyBank, TrendingUp, Wallet } from "lucide-react";
import { getAllPortfolio, type AllPortfolioResponse } from "../api";
import { brl0 } from "../format";
import { Kpi, PnlKpi } from "./PortfolioPage";

export default function AllPortfolioPage({ onOpenGame }: { onOpenGame: (id: string) => void }) {
  const [data, setData] = useState<AllPortfolioResponse | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    getAllPortfolio(90)
      .then(setData)
      .catch((err: unknown) => setError(err instanceof Error ? err.message : "failed to load portfolios"));
  }, []);

  if (error) {
    return <Panel>Could not load portfolios: {error}</Panel>;
  }
  if (!data) {
    return <Panel>Loading portfolios…</Panel>;
  }

  const total = data.total;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-lg font-semibold text-slate-100">All Portfolios</h1>
        <p className="mt-1 max-w-2xl text-sm text-slate-400">
          Every game's holdings and sales combined in reais. Click a game to open its portfolio.
        </p>
      </div>

      <div className="grid grid-cols-2 gap-3 lg:grid-cols-5">
        <Kpi icon={<Wallet className="h-5 w-5" />} label="Invested (holding)" value={brl0(total.investedBRL)} sub={`${total.holdings} items`} />
        <Kpi icon={<Coins className="h-5 w-5" />} label="Current value" value={brl0(total.marketBRL)} sub="all games" />
        <PnlKpi icon={<TrendingUp className="h-5 w-5" />} label="Unrealized P&L" value={total.unrealizedBRL} />
        <PnlKpi icon={<PiggyBank className="h-5 w-5" />} label={`Realized (${total.sold} sold)`} value={total.realizedBRL} />
        <PnlKpi icon={<TrendingUp className="h-5 w-5" />} label="Total P&L" value={total.totalPnLBRL} strong />
      </div>

      <div className="overflow-x-auto rounded-xl border border-slate-800 bg-slate-900/40">
        <table className="w-full min-w-[760px] text-sm">
          <thead>
            <tr className="border-b border-slate-800 text-left text-xs uppercase tracking-wide text-slate-500">
              <th className="px-3 py-2 font-medium">Game</th>
              <th className="px-3 py-2 text-right font-medium">Holding</th>
              <th className="px-3 py-2 text-right font-medium">Invested</th>
              <th className="px-3 py-2 text-right font-medium">Current value</th>
              <th className="px-3 py-2 text-right font-medium">Unrealized</th>
              <th className="px-3 py-2 text-right font-medium">Realized</th>
              <th className="px-3 py-2 text-right font-medium">Total P&L</th>
            </tr>
          </thead>
          <tbody>
            {data.games.map((g) => (
              <tr
                key={g.game.id}
                onClick={() => onOpenGame(g.game.id)}
                className="cursor-pointer border-b border-slate-800/60 transition-colors last:border-0 hover:bg-slate-800/40"
              >
                <td className="px-3 py-2">
                  <div className="font-medium text-slate-100">{g.game.name}</div>
                  <div className="text-[10px] text-slate-500">
                    {g.summary.sold > 0 ? `${g.summary.sold} sold` : "no sales yet"}
                  </div>
                </td>
                <td className="px-3 py-2 text-right tabular-nums text-slate-300">{g.summary.holdings}</td>
                <td className="px-3 py-2 text-right tabular-nums text-slate-300">{brl0(g.summary.investedBRL)}</td>
                <td className="px-3 py-2 text-right tabular-nums text-slate-200">{brl0(g.summary.marketBRL)}</td>
                <PnlCell value={g.summary.unrealizedBRL} />
                <PnlCell value={g.summary.realizedBRL} />
                <PnlCell value={g.summary.totalPnLBRL} strong />
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

function PnlCell({ value, strong }: { value: number; strong?: boolean }) {
  const up = value >= 0;
  return (
    <td className={`px-3 py-2 text-right tabular-nums ${strong ? "font-semibold" : ""} ${up ? "text-emerald-300" : "text-rose-300"}`}>
      {up ? "+" : "−"}
      {brl0(Math.abs(value))}
    </td>
  );
}

function Panel({ children }: { children: React.ReactNode }) {
  return (
    <div className="rounded-xl border border-slate-800 bg-slate-900/40 px-4 py-10 text-center text-slate-400">
      {children}
    </div>
  );
}
