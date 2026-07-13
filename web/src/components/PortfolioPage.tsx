import { useCallback, useEffect, useMemo, useState } from "react";
import { Plus, Trash2, Wallet, Coins, TrendingUp, PiggyBank, ExternalLink, Share2, Package, Pencil, Truck, Check } from "lucide-react";
import {
  createTrade,
  deleteTrade,
  gameHasDeals,
  getGame,
  getPortfolio,
  getQuote,
  productIDFromTcgURL,
  updateTrade,
  type PortfolioResponse,
  type QuoteMatch,
  type TradeView,
} from "../api";
import { brl0, usd } from "../format";

// In BR-only games (no US deals pipeline) there is no US price: the portfolio
// values holdings in BRL and the API returns market figures already in reais
// (fxRate = 1). Deals-enabled games keep their USD (TCGplayer) valuation.
// isBRGame drives the labels/format.
const isBRGame = () => !gameHasDeals(getGame());

// marketMoney formats a market figure that is USD for deals games but BRL elsewhere.
function marketMoney(v: number): string {
  return isBRGame() ? brl0(v) : usd(v);
}
import CardArt from "./CardArt";
import ShareList from "./ShareList";
import { Button } from "./ui/button";
import { Card } from "./ui/card";
import { Input } from "./ui/input";
import { Badge } from "./ui/badge";
import { ToggleGroup } from "./ui/toggle-group";

const pctOptions = [
  { value: "85", label: "85%" },
  { value: "90", label: "90%" },
  { value: "95", label: "95%" },
  { value: "100", label: "100%" },
];

function cleanName(n: string): string {
  return n.replace(/\s*\([^)]*\)\s*$/, "");
}

type Section = "singles" | "sealed";

interface SectionSummary {
  holdings: number;
  investedBRL: number;
  marketBRL: number;
  unrealizedBRL: number;
  sold: number;
  realizedBRL: number;
  totalPnLBRL: number;
}

function sectionSummary(list: TradeView[]): SectionSummary {
  const s: SectionSummary = {
    holdings: 0,
    investedBRL: 0,
    marketBRL: 0,
    unrealizedBRL: 0,
    sold: 0,
    realizedBRL: 0,
    totalPnLBRL: 0,
  };
  let costOfSold = 0;
  let proceeds = 0;
  for (const t of list) {
    if (t.realized) {
      s.sold += 1;
      costOfSold += t.costBRL;
      proceeds += t.valueBRL;
    } else {
      s.holdings += 1;
      s.investedBRL += t.costBRL;
      s.marketBRL += t.valueBRL;
    }
  }
  s.unrealizedBRL = s.marketBRL - s.investedBRL;
  s.realizedBRL = proceeds - costOfSold;
  s.totalPnLBRL = s.unrealizedBRL + s.realizedBRL;
  return s;
}

export default function PortfolioPage() {
  const [pct, setPct] = useState(90);
  const [section, setSection] = useState<Section>("singles");
  const [data, setData] = useState<PortfolioResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [adding, setAdding] = useState(false);
  const [sharing, setSharing] = useState(false);

  const load = useCallback(async (p: number) => {
    try {
      const r = await getPortfolio(p);
      setData(r);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "failed to load portfolio");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load(pct);
  }, [load, pct]);

  const refresh = useCallback(() => load(pct), [load, pct]);

  const trades = data?.trades ?? [];
  const active = useMemo(
    () => trades.filter((t) => (t.kind === "sealed") === (section === "sealed")),
    [trades, section],
  );
  const summary = useMemo(() => sectionSummary(active), [active]);
  const holdings = useMemo(
    () => active.filter((t) => !t.realized).sort((a, b) => b.valueBRL - a.valueBRL),
    [active],
  );
  const sold = useMemo(
    () => active.filter((t) => t.realized).sort((a, b) => b.valueBRL - a.valueBRL),
    [active],
  );
  const isSealed = section === "sealed";

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="text-lg font-semibold text-slate-100">Portfolio</h1>
          <p className="mt-1 max-w-2xl text-sm text-slate-400">
            {isSealed
              ? "Your sealed buys and sales, valued at your own current-value estimate — no TCGplayer or Liga comparison."
              : isBRGame()
                ? "Your buys and sales, valued against the current Liga Brazil floor. Track what you're up — and down — across the whole collection."
                : "Your buys and sales, valued against live TCGplayer prices. Track what you're up — and down — across the whole collection."}
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-4">
          <ToggleGroup
            value={section}
            onChange={(v) => {
              setSection(v === "sealed" ? "sealed" : "singles");
              setAdding(false);
              setSharing(false);
            }}
            options={[
              { value: "singles", label: "Singles" },
              { value: "sealed", label: "Sealed" },
            ]}
          />
          {!isSealed && (
            <div className="flex items-center gap-2">
              <span className="text-[10px] font-medium uppercase tracking-wide text-slate-500">
                Value holdings at
              </span>
              <ToggleGroup
                value={String(pct)}
                onChange={(v) => setPct(Number(v))}
                options={pctOptions}
              />
            </div>
          )}
        </div>
      </div>

      {data && (
        <div className="grid grid-cols-2 gap-3 lg:grid-cols-5">
          <Kpi icon={<Wallet className="h-5 w-5" />} label="Invested (holding)" value={brl0(summary.investedBRL)} sub={`${summary.holdings} ${isSealed ? "products" : "cards"}`} />
          <Kpi icon={<Coins className="h-5 w-5" />} label="Current value" value={brl0(summary.marketBRL)} sub={isSealed ? "your estimate" : `@ ${pct}% of ${isBRGame() ? "floor" : "TCG"}`} />
          <PnlKpi icon={<TrendingUp className="h-5 w-5" />} label="Unrealized P&L" value={summary.unrealizedBRL} />
          <PnlKpi icon={<PiggyBank className="h-5 w-5" />} label={`Realized (${summary.sold} sold)`} value={summary.realizedBRL} />
          <PnlKpi icon={<TrendingUp className="h-5 w-5" />} label="Total P&L" value={summary.totalPnLBRL} strong />
        </div>
      )}

      {error && (
        <div className="rounded-xl border border-rose-900/50 bg-rose-950/30 px-4 py-3 text-sm text-rose-200">
          {error}
        </div>
      )}

      <div className="flex items-center justify-between">
        <h2 className="text-sm font-semibold text-slate-200">
          Holdings <span className="font-normal text-slate-500">· {holdings.length}</span>
        </h2>
        <div className="flex items-center gap-2">
          {!isSealed && (
            <Button
              variant="outline"
              onClick={() => setSharing(true)}
              disabled={holdings.length === 0}
              title="Build a list to send to buyers"
            >
              <Share2 /> Share list
            </Button>
          )}
          <Button onClick={() => setAdding((a) => !a)}>
            <Plus /> {adding ? "Close" : isSealed ? "Add sealed" : "Add trade"}
          </Button>
        </div>
      </div>

      {adding &&
        (isSealed ? (
          <AddSealedForm
            onAdded={() => {
              setAdding(false);
              refresh();
            }}
          />
        ) : (
          <AddTradeForm
            fxRate={data?.fxRate ?? 0}
            onAdded={() => {
              setAdding(false);
              refresh();
            }}
          />
        ))}

      {sharing && !isSealed && holdings.length > 0 && (
        <ShareList holdings={holdings} fxRate={data?.fxRate ?? 0} onClose={() => setSharing(false)} />
      )}

      {loading ? (
        <Panel>Loading portfolio…</Panel>
      ) : active.length === 0 ? (
        <Panel>
          {isSealed
            ? "No sealed products yet. Click “Add sealed” to log your first box."
            : "No trades yet. Click “Add trade” to log your first buy."}
        </Panel>
      ) : (
        <div className="space-y-6">
          {isSealed ? (
            <SealedTable trades={holdings} onChanged={refresh} empty="No sealed products held right now." />
          ) : (
            <TradeTable trades={holdings} onChanged={refresh} empty="No cards held right now." />
          )}
          {sold.length > 0 && (
            <div className="space-y-3">
              <h2 className="text-sm font-semibold text-slate-200">
                Sold <span className="font-normal text-slate-500">· {sold.length}</span>
              </h2>
              {isSealed ? (
                <SealedTable trades={sold} onChanged={refresh} empty="" />
              ) : (
                <TradeTable trades={sold} onChanged={refresh} empty="" />
              )}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

export function Kpi({ icon, label, value, sub }: { icon: React.ReactNode; label: string; value: string; sub?: string }) {
  return (
    <Card className="flex items-center gap-3 p-4">
      <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-slate-800/70 text-sky-300">{icon}</div>
      <div className="min-w-0">
        <div className="text-[10px] font-medium uppercase tracking-wide text-slate-500">{label}</div>
        <div className="truncate text-lg font-semibold tabular-nums text-slate-100">{value}</div>
        {sub && <div className="truncate text-[11px] text-slate-500">{sub}</div>}
      </div>
    </Card>
  );
}

export function PnlKpi({ icon, label, value, strong }: { icon: React.ReactNode; label: string; value: number; strong?: boolean }) {
  const up = value >= 0;
  const tone = up ? "text-emerald-300" : "text-rose-300";
  return (
    <Card className={`flex items-center gap-3 p-4 ${strong ? "ring-1 ring-inset ring-sky-500/20" : ""}`}>
      <div className={`flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-slate-800/70 ${tone}`}>{icon}</div>
      <div className="min-w-0">
        <div className="text-[10px] font-medium uppercase tracking-wide text-slate-500">{label}</div>
        <div className={`truncate text-lg font-semibold tabular-nums ${tone}`}>
          {up ? "+" : "−"}
          {brl0(Math.abs(value))}
        </div>
      </div>
    </Card>
  );
}

function TradeTable({ trades, onChanged, empty }: { trades: TradeView[]; onChanged: () => void; empty: string }) {
  if (trades.length === 0) {
    return empty ? <Panel>{empty}</Panel> : null;
  }
  return (
    <div className="overflow-x-auto rounded-xl border border-slate-800 bg-slate-900/40">
      <table className="w-full min-w-[760px] text-sm">
        <thead>
          <tr className="border-b border-slate-800 text-left text-xs uppercase tracking-wide text-slate-500">
            <th className="px-3 py-2 font-medium">Card</th>
            <th className="px-3 py-2 text-right font-medium">Cost</th>
            <th className="px-3 py-2 text-right font-medium">{isBRGame() ? "Floor" : "TCG"}</th>
            <th className="px-3 py-2 text-right font-medium">Value / Sold</th>
            <th className="px-3 py-2 text-right font-medium">P&L</th>
            <th className="px-3 py-2 text-right font-medium">Margin</th>
            <th className="px-3 py-2" />
          </tr>
        </thead>
        <tbody>
          {trades.map((t) => (
            <TradeRow key={t.id} t={t} onChanged={onChanged} />
          ))}
        </tbody>
      </table>
    </div>
  );
}

function DeliveryToggle({ t, onChanged }: { t: TradeView; onChanged: () => void }) {
  const [saving, setSaving] = useState(false);
  const toggle = async () => {
    setSaving(true);
    try {
      await updateTrade(t.id, { ...t, delivered: !t.delivered });
      onChanged();
    } finally {
      setSaving(false);
    }
  };
  return (
    <button
      onClick={toggle}
      disabled={saving}
      title={t.delivered ? "Delivered — click to mark in transit" : "In transit — click to mark delivered"}
      className={`flex items-center gap-1 rounded-md px-2 py-1 text-xs font-medium ring-1 ring-inset disabled:opacity-50 ${
        t.delivered
          ? "bg-emerald-500/10 text-emerald-300 ring-emerald-500/30 hover:bg-emerald-500/20"
          : "bg-amber-500/10 text-amber-300 ring-amber-500/30 hover:bg-amber-500/20"
      }`}
    >
      {t.delivered ? <Check className="h-3 w-3" /> : <Truck className="h-3 w-3" />}
      {t.delivered ? "Delivered" : "In transit"}
    </button>
  );
}

function TradeRow({ t, onChanged }: { t: TradeView; onChanged: () => void }) {
  const [selling, setSelling] = useState(false);
  const up = t.profitBRL >= 0;
  return (
    <>
      <tr className="border-b border-slate-800/60 last:border-0">
        <td className="px-3 py-2">
          <div className="flex items-center gap-2">
            <CardArt set={t.set} number={t.number} name={t.name} productID={productIDFromTcgURL(t.tcgUrl)} className="h-12 w-[34px] shrink-0 rounded" />
            <div className="min-w-0">
              <div className="truncate font-medium text-slate-100" title={t.name}>
                {cleanName(t.name) || t.number}
              </div>
              <div className="font-mono text-[10px] text-slate-500">
                {t.number}
                {t.qty > 1 ? ` ·×${t.qty}` : ""}
                {t.store ? ` · ${t.store}` : ""}
              </div>
            </div>
          </div>
        </td>
        <td className="px-3 py-2 text-right tabular-nums text-slate-300">{brl0(t.costBRL)}</td>
        <td className="px-3 py-2 text-right tabular-nums text-slate-400">
          {t.marketKnown ? marketMoney(t.marketUSD) : "—"}
        </td>
        <td className="px-3 py-2 text-right tabular-nums text-slate-200">
          {brl0(t.valueBRL)}
          {t.realized && <div className="text-[10px] text-slate-500">{t.sellDate || "sold"}</div>}
        </td>
        <td className={`px-3 py-2 text-right font-semibold tabular-nums ${up ? "text-emerald-300" : "text-rose-300"}`}>
          {up ? "+" : "−"}
          {brl0(Math.abs(t.profitBRL))}
        </td>
        <td className={`px-3 py-2 text-right tabular-nums ${up ? "text-emerald-300" : "text-rose-300"}`}>
          {up ? "+" : ""}
          {Math.round(t.marginPct)}%
        </td>
        <td className="px-3 py-2">
          <div className="flex items-center justify-end gap-1">
            {t.tcgUrl && (
              <a
                href={t.tcgUrl}
                target="_blank"
                rel="noreferrer"
                className="flex items-center gap-1 rounded-md bg-sky-500/10 px-2 py-1 text-xs font-medium text-sky-300 ring-1 ring-inset ring-sky-500/30 hover:bg-sky-500/20"
                title="See on TCGplayer"
              >
                TCG <ExternalLink className="h-3 w-3" />
              </a>
            )}
            {t.realized ? (
              <Badge variant="emerald">sold</Badge>
            ) : (
              <>
                <DeliveryToggle t={t} onChanged={onChanged} />
                <button
                  onClick={() => setSelling((s) => !s)}
                  className="rounded-md border border-slate-700 bg-slate-800/60 px-2 py-1 text-xs text-slate-200 hover:bg-slate-800"
                >
                  Sell
                </button>
              </>
            )}
            <button
              onClick={() => {
                if (confirm("Delete this trade?")) {
                  deleteTrade(t.id).then(onChanged);
                }
              }}
              className="rounded-md border border-slate-700 bg-slate-800/60 p-1.5 text-slate-400 hover:bg-rose-950/40 hover:text-rose-300"
              title="Delete"
            >
              <Trash2 className="h-3.5 w-3.5" />
            </button>
          </div>
        </td>
      </tr>
      {selling && (
        <tr className="bg-slate-900/60">
          <td colSpan={7} className="px-3 py-3">
            <SellForm
              t={t}
              onDone={() => {
                setSelling(false);
                onChanged();
              }}
            />
          </td>
        </tr>
      )}
    </>
  );
}

function SellForm({ t, onDone }: { t: TradeView; onDone: () => void }) {
  const [price, setPrice] = useState("");
  const [currency, setCurrency] = useState<"BRL" | "USD">("BRL");
  const [date, setDate] = useState("");
  const [buyer, setBuyer] = useState("");
  const [saving, setSaving] = useState(false);

  const submit = async () => {
    setSaving(true);
    try {
      await updateTrade(t.id, {
        ...t,
        status: "sold",
        sellPrice: Number(price) || 0,
        sellCurrency: currency,
        sellDate: date,
        buyer,
      });
      onDone();
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="flex flex-wrap items-end gap-2">
      <Field label="Sell price">
        <Input type="number" value={price} onChange={(e) => setPrice(e.target.value)} className="w-32" placeholder="0,00" />
      </Field>
      <Field label="Currency">
        <ToggleGroup
          value={currency}
          onChange={(v) => setCurrency(v === "USD" ? "USD" : "BRL")}
          options={[
            { value: "BRL", label: "R$" },
            { value: "USD", label: "US$" },
          ]}
        />
      </Field>
      <Field label="Date">
        <Input type="date" value={date} onChange={(e) => setDate(e.target.value)} className="w-40" />
      </Field>
      <Field label="Buyer (optional)">
        <Input value={buyer} onChange={(e) => setBuyer(e.target.value)} className="w-40" placeholder="P2P buyer" />
      </Field>
      <Button onClick={submit} disabled={saving || !price}>
        {saving ? "Saving…" : "Mark sold"}
      </Button>
    </div>
  );
}

function AddTradeForm({ fxRate, onAdded }: { fxRate: number; onAdded: () => void }) {
  const [query, setQuery] = useState("");
  const [matches, setMatches] = useState<QuoteMatch[]>([]);
  const [number, setNumber] = useState("");
  const [name, setName] = useState("");
  const [set, setSet] = useState("");
  const [refUSD, setRefUSD] = useState("");
  const [buyBRL, setBuyBRL] = useState("");
  const [shippingBRL, setShippingBRL] = useState("");
  const [qty, setQty] = useState("1");
  const [condition, setCondition] = useState("NM");
  const [store, setStore] = useState("");
  const [buyDate, setBuyDate] = useState("");
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (query.trim().length < 2) {
      setMatches([]);
      return;
    }
    let current = true;
    const h = window.setTimeout(() => {
      getQuote(query)
        .then((r) => current && setMatches(r.matches))
        .catch(() => current && setMatches([]));
    }, 250);
    return () => {
      current = false;
      window.clearTimeout(h);
    };
  }, [query]);

  const pick = (m: QuoteMatch) => {
    setNumber(m.number);
    setName(m.name);
    setSet(m.set);
    setRefUSD(String(Math.round(m.marketUSD * 100) / 100));
    setQuery(`${m.number} · ${cleanName(m.name)}`);
    setMatches([]);
  };

  const submit = async () => {
    setSaving(true);
    try {
      await createTrade({
        number,
        name,
        set,
        condition,
        qty: Number(qty) || 1,
        buyBRL: Number(buyBRL) || 0,
        shippingBRL: Number(shippingBRL) || 0,
        refUSD: Number(refUSD) || 0,
        store,
        buyDate,
        status: "holding",
      });
      onAdded();
    } finally {
      setSaving(false);
    }
  };

  const valid = (number || name) && Number(buyBRL) > 0;

  return (
    <Card className="space-y-3 p-4">
      <div className="relative">
        <Field label="Find card (name or number)">
          <Input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="e.g. Luffy or OP16-080"
            className="w-full"
          />
        </Field>
        {matches.length > 0 && (
          <ul className="absolute z-10 mt-1 max-h-64 w-full overflow-y-auto rounded-lg border border-slate-700 bg-slate-900 shadow-xl">
            {matches.map((m) => (
              <li key={`${m.number}-${m.name}`}>
                <button
                  type="button"
                  onClick={() => pick(m)}
                  className="flex w-full items-center justify-between gap-3 px-3 py-2 text-left text-sm hover:bg-slate-800"
                >
                  <span className="min-w-0">
                    <span className="truncate text-slate-100">{cleanName(m.name)}</span>{" "}
                    <span className="font-mono text-[11px] text-slate-500">{m.number}</span>
                  </span>
                  <span className="shrink-0 tabular-nums text-emerald-300">{marketMoney(m.marketUSD)}</span>
                </button>
              </li>
            ))}
          </ul>
        )}
      </div>

      <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4">
        <Field label="Number">
          <Input value={number} onChange={(e) => setNumber(e.target.value)} placeholder="OP16-080" />
        </Field>
        <Field label="Name">
          <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="Card name" />
        </Field>
        <Field label="Set">
          <Input value={set} onChange={(e) => setSet(e.target.value)} placeholder="OP16" />
        </Field>
        <Field
          label={
            isBRGame()
              ? "Market R$"
              : `TCG US$ ${fxRate > 0 && refUSD ? `(≈ ${brl0(Number(refUSD) / fxRate)})` : ""}`
          }
        >
          <Input type="number" value={refUSD} onChange={(e) => setRefUSD(e.target.value)} placeholder="0.00" />
        </Field>
        <Field label="Buy R$">
          <Input type="number" value={buyBRL} onChange={(e) => setBuyBRL(e.target.value)} placeholder="0,00" />
        </Field>
        <Field label="Frete R$">
          <Input type="number" value={shippingBRL} onChange={(e) => setShippingBRL(e.target.value)} placeholder="0,00" />
        </Field>
        <Field label="Qty">
          <Input type="number" value={qty} onChange={(e) => setQty(e.target.value)} />
        </Field>
        <Field label="Condition">
          <Input value={condition} onChange={(e) => setCondition(e.target.value)} placeholder="NM" />
        </Field>
        <Field label="Store">
          <Input value={store} onChange={(e) => setStore(e.target.value)} placeholder="Legends" />
        </Field>
        <Field label="Buy date">
          <Input type="date" value={buyDate} onChange={(e) => setBuyDate(e.target.value)} />
        </Field>
      </div>

      <div className="flex justify-end">
        <Button onClick={submit} disabled={!valid || saving}>
          {saving ? "Saving…" : "Save trade"}
        </Button>
      </div>
    </Card>
  );
}

function SealedTable({ trades, onChanged, empty }: { trades: TradeView[]; onChanged: () => void; empty: string }) {
  if (trades.length === 0) {
    return empty ? <Panel>{empty}</Panel> : null;
  }
  return (
    <div className="overflow-x-auto rounded-xl border border-slate-800 bg-slate-900/40">
      <table className="w-full min-w-[760px] text-sm">
        <thead>
          <tr className="border-b border-slate-800 text-left text-xs uppercase tracking-wide text-slate-500">
            <th className="px-3 py-2 font-medium">Product</th>
            <th className="px-3 py-2 text-right font-medium">Cost</th>
            <th className="px-3 py-2 text-right font-medium">Current R$</th>
            <th className="px-3 py-2 text-right font-medium">Value / Sold</th>
            <th className="px-3 py-2 text-right font-medium">P&L</th>
            <th className="px-3 py-2 text-right font-medium">Margin</th>
            <th className="px-3 py-2" />
          </tr>
        </thead>
        <tbody>
          {trades.map((t) => (
            <SealedRow key={t.id} t={t} onChanged={onChanged} />
          ))}
        </tbody>
      </table>
    </div>
  );
}

function SealedRow({ t, onChanged }: { t: TradeView; onChanged: () => void }) {
  const [selling, setSelling] = useState(false);
  const [editing, setEditing] = useState(false);
  const up = t.profitBRL >= 0;
  return (
    <>
      <tr className="border-b border-slate-800/60 last:border-0">
        <td className="px-3 py-2">
          <div className="flex items-center gap-2">
            <div className="flex h-12 w-[34px] shrink-0 items-center justify-center rounded bg-slate-800/70 text-slate-400">
              <Package className="h-5 w-5" />
            </div>
            <div className="min-w-0">
              <div className="truncate font-medium text-slate-100" title={t.name}>
                {t.name || t.number}
              </div>
              <div className="font-mono text-[10px] text-slate-500">
                {t.number}
                {t.qty > 1 ? ` ·×${t.qty}` : ""}
                {t.store ? ` · ${t.store}` : ""}
              </div>
            </div>
          </div>
        </td>
        <td className="px-3 py-2 text-right tabular-nums text-slate-300">{brl0(t.costBRL)}</td>
        <td className="px-3 py-2 text-right tabular-nums text-slate-400">
          {t.manualBRL ? brl0(t.manualBRL) : "—"}
        </td>
        <td className="px-3 py-2 text-right tabular-nums text-slate-200">
          {brl0(t.valueBRL)}
          {t.realized && <div className="text-[10px] text-slate-500">{t.sellDate || "sold"}</div>}
        </td>
        <td className={`px-3 py-2 text-right font-semibold tabular-nums ${up ? "text-emerald-300" : "text-rose-300"}`}>
          {up ? "+" : "−"}
          {brl0(Math.abs(t.profitBRL))}
        </td>
        <td className={`px-3 py-2 text-right tabular-nums ${up ? "text-emerald-300" : "text-rose-300"}`}>
          {up ? "+" : ""}
          {Math.round(t.marginPct)}%
        </td>
        <td className="px-3 py-2">
          <div className="flex items-center justify-end gap-1">
            {t.realized ? (
              <Badge variant="emerald">sold</Badge>
            ) : (
              <>
                <DeliveryToggle t={t} onChanged={onChanged} />
                <button
                  onClick={() => {
                    setEditing((e) => !e);
                    setSelling(false);
                  }}
                  className="flex items-center gap-1 rounded-md border border-slate-700 bg-slate-800/60 px-2 py-1 text-xs text-slate-200 hover:bg-slate-800"
                  title="Update current value"
                >
                  <Pencil className="h-3 w-3" /> Value
                </button>
                <button
                  onClick={() => {
                    setSelling((s) => !s);
                    setEditing(false);
                  }}
                  className="rounded-md border border-slate-700 bg-slate-800/60 px-2 py-1 text-xs text-slate-200 hover:bg-slate-800"
                >
                  Sell
                </button>
              </>
            )}
            <button
              onClick={() => {
                if (confirm("Delete this trade?")) {
                  deleteTrade(t.id).then(onChanged);
                }
              }}
              className="rounded-md border border-slate-700 bg-slate-800/60 p-1.5 text-slate-400 hover:bg-rose-950/40 hover:text-rose-300"
              title="Delete"
            >
              <Trash2 className="h-3.5 w-3.5" />
            </button>
          </div>
        </td>
      </tr>
      {editing && (
        <tr className="bg-slate-900/60">
          <td colSpan={7} className="px-3 py-3">
            <EditValueForm
              t={t}
              onDone={() => {
                setEditing(false);
                onChanged();
              }}
            />
          </td>
        </tr>
      )}
      {selling && (
        <tr className="bg-slate-900/60">
          <td colSpan={7} className="px-3 py-3">
            <SellForm
              t={t}
              onDone={() => {
                setSelling(false);
                onChanged();
              }}
            />
          </td>
        </tr>
      )}
    </>
  );
}

function EditValueForm({ t, onDone }: { t: TradeView; onDone: () => void }) {
  const [value, setValue] = useState(t.manualBRL ? String(t.manualBRL) : "");
  const [saving, setSaving] = useState(false);

  const submit = async () => {
    setSaving(true);
    try {
      await updateTrade(t.id, { ...t, manualBRL: Number(value) || 0 });
      onDone();
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="flex flex-wrap items-end gap-2">
      <Field label="Current value R$ (per unit)">
        <Input type="number" value={value} onChange={(e) => setValue(e.target.value)} className="w-40" placeholder="0,00" />
      </Field>
      <Button onClick={submit} disabled={saving || !value}>
        {saving ? "Saving…" : "Update value"}
      </Button>
    </div>
  );
}

function AddSealedForm({ onAdded }: { onAdded: () => void }) {
  const [query, setQuery] = useState("");
  const [matches, setMatches] = useState<QuoteMatch[]>([]);
  const [number, setNumber] = useState("");
  const [name, setName] = useState("");
  const [manualBRL, setManualBRL] = useState("");
  const [buyBRL, setBuyBRL] = useState("");
  const [shippingBRL, setShippingBRL] = useState("");
  const [qty, setQty] = useState("1");
  const [store, setStore] = useState("");
  const [buyDate, setBuyDate] = useState("");
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (query.trim().length < 2) {
      setMatches([]);
      return;
    }
    let current = true;
    const h = window.setTimeout(() => {
      getQuote(query, 25, "sealed")
        .then((r) => current && setMatches(r.matches))
        .catch(() => current && setMatches([]));
    }, 250);
    return () => {
      current = false;
      window.clearTimeout(h);
    };
  }, [query]);

  const pick = (m: QuoteMatch) => {
    setNumber(m.number);
    setName(m.name);
    setQuery(m.name);
    setMatches([]);
  };

  const submit = async () => {
    setSaving(true);
    try {
      await createTrade({
        kind: "sealed",
        number,
        name,
        set: "SEALED",
        qty: Number(qty) || 1,
        buyBRL: Number(buyBRL) || 0,
        shippingBRL: Number(shippingBRL) || 0,
        manualBRL: Number(manualBRL) || 0,
        store,
        buyDate,
        status: "holding",
      });
      onAdded();
    } finally {
      setSaving(false);
    }
  };

  const valid = name && Number(buyBRL) > 0;

  return (
    <Card className="space-y-3 p-4">
      <div className="relative">
        <Field label="Find sealed product (Liga catalog)">
          <Input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="e.g. Booster OP-16"
            className="w-full"
          />
        </Field>
        {matches.length > 0 && (
          <ul className="absolute z-10 mt-1 max-h-64 w-full overflow-y-auto rounded-lg border border-slate-700 bg-slate-900 shadow-xl">
            {matches.map((m) => (
              <li key={`${m.number}-${m.name}`}>
                <button
                  type="button"
                  onClick={() => pick(m)}
                  className="flex w-full items-center justify-between gap-3 px-3 py-2 text-left text-sm hover:bg-slate-800"
                >
                  <span className="truncate text-slate-100">{m.name}</span>
                  <span className="shrink-0 font-mono text-[11px] text-slate-500">{m.number}</span>
                </button>
              </li>
            ))}
          </ul>
        )}
      </div>

      <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4">
        <Field label="Name">
          <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="Booster Box OP-16" />
        </Field>
        <Field label="Current value R$ (per unit)">
          <Input type="number" value={manualBRL} onChange={(e) => setManualBRL(e.target.value)} placeholder="0,00" />
        </Field>
        <Field label="Buy R$ (per unit)">
          <Input type="number" value={buyBRL} onChange={(e) => setBuyBRL(e.target.value)} placeholder="0,00" />
        </Field>
        <Field label="Frete R$">
          <Input type="number" value={shippingBRL} onChange={(e) => setShippingBRL(e.target.value)} placeholder="0,00" />
        </Field>
        <Field label="Qty">
          <Input type="number" value={qty} onChange={(e) => setQty(e.target.value)} />
        </Field>
        <Field label="Store">
          <Input value={store} onChange={(e) => setStore(e.target.value)} placeholder="Legends" />
        </Field>
        <Field label="Buy date">
          <Input type="date" value={buyDate} onChange={(e) => setBuyDate(e.target.value)} />
        </Field>
      </div>

      <div className="flex justify-end">
        <Button onClick={submit} disabled={!valid || saving}>
          {saving ? "Saving…" : "Save sealed"}
        </Button>
      </div>
    </Card>
  );
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <label className="flex flex-col gap-1">
      <span className="text-[10px] font-medium uppercase tracking-wide text-slate-500">{label}</span>
      {children}
    </label>
  );
}

function Panel({ children }: { children: React.ReactNode }) {
  return (
    <div className="rounded-xl border border-slate-800 bg-slate-900/40 px-4 py-10 text-center text-slate-400">
      {children}
    </div>
  );
}
