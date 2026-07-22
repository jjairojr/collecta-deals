import { useCallback, useEffect, useState } from "react";
import { ArrowLeft, Check, Copy, ExternalLink, FileText, Plus, Trash2 } from "lucide-react";
import {
  createQuote,
  deleteQuote,
  gameHasDeals,
  getGame,
  getQuote,
  listQuotes,
  tcgProductURL,
  updateQuote,
  type Quote,
  type QuoteItem,
  type QuoteMarket,
  type QuoteMatch,
} from "../api";
import { brl0, timeAgo, usd } from "../format";
import CardArt from "./CardArt";
import { Badge } from "./ui/badge";
import { Button } from "./ui/button";
import { Card } from "./ui/card";
import { Input } from "./ui/input";
import { ToggleGroup } from "./ui/toggle-group";

const isBRGame = () => !gameHasDeals(getGame());

const pctPresets = [
  { value: "50", label: "50%" },
  { value: "60", label: "60%" },
  { value: "70", label: "70%" },
  { value: "100", label: "100%" },
];

const marketOptions = [
  { value: "tcg", label: "TCGplayer" },
  { value: "liga", label: "Liga" },
];

function quoteTotal(items: QuoteItem[]): number {
  return items.reduce((sum, it) => sum + it.unitBRL * it.qty, 0);
}

function effPct(it: QuoteItem, pct: number): number {
  return it.pct && it.pct > 0 ? it.pct : pct;
}

function quoteOffer(items: QuoteItem[], pct: number): number {
  return items.reduce((sum, it) => sum + (it.unitBRL * it.qty * effPct(it, pct)) / 100, 0);
}

function blendedPct(items: QuoteItem[], pct: number): number {
  const total = quoteTotal(items);
  return total > 0 ? Math.round((quoteOffer(items, pct) / total) * 100) : Math.round(pct);
}

function seedUnitBRL(
  market: QuoteMarket,
  ligaLowBRL: number | undefined,
  marketUSD: number | undefined,
  fxRate: number,
): number {
  const tcgBRL = marketUSD && fxRate > 0 ? marketUSD / fxRate : 0;
  const ligaBRL = ligaLowBRL ?? 0;
  if (market === "liga") {
    return Math.round(ligaBRL > 0 ? ligaBRL : tcgBRL);
  }
  return Math.round(tcgBRL > 0 ? tcgBRL : ligaBRL);
}

export function buildQuoteText(q: Quote): string {
  const lines = [`🏴‍☠️ Orçamento — ${q.name}`];
  lines.push("", "Tenho interesse nas seguintes cartas (NM / Inglês):", "");
  for (const it of q.items) {
    const label = it.name.includes(it.number) ? it.name : `${it.name} (${it.number})`;
    const pct = effPct(it, q.pct);
    const offerUnit = Math.round((it.unitBRL * pct) / 100);
    if (it.qty > 1) {
      lines.push(`• ${label} ×${it.qty} — ${brl0(offerUnit)}/un = ${brl0(offerUnit * it.qty)} (${pct}%)`);
    } else {
      lines.push(`• ${label} — ${brl0(offerUnit)} (${pct}%)`);
    }
  }
  const total = quoteTotal(q.items);
  const offer = quoteOffer(q.items, q.pct);
  const blended = blendedPct(q.items, q.pct);
  lines.push("");
  if (blended < 100) {
    lines.push(`Valor de mercado: ${brl0(total)}`);
  }
  lines.push(`💰 Total oferta (${blended}%): ${brl0(offer)}`);
  lines.push("💸 Pago à vista no PIX, na hora!");
  return lines.join("\n");
}

export default function QuotePage() {
  const [quotes, setQuotes] = useState<Quote[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [editing, setEditing] = useState<Quote | null>(null);

  const load = useCallback(async () => {
    try {
      const r = await listQuotes();
      setQuotes(r.quotes);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "failed to load quotes");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  if (editing) {
    return (
      <QuoteEditor
        quote={editing}
        onBack={() => {
          setEditing(null);
          load();
        }}
        onSaved={(q) => setEditing(q)}
      />
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="text-lg font-semibold text-slate-100">Orçamentos</h1>
          <p className="mt-1 max-w-2xl text-sm text-slate-400">
            Monte orçamentos de compra de coleções: pesquise as cartas, ajuste quantidades e
            preços, aplique o deságio e copie a oferta pronta pro WhatsApp.
          </p>
        </div>
        <Button
          variant="primary"
          onClick={() =>
            setEditing({
              id: "",
              name: "",
              pct: 60,
              items: [],
              createdAt: "",
              updatedAt: "",
            })
          }
        >
          <Plus /> Novo orçamento
        </Button>
      </div>

      {error && (
        <div className="rounded-xl border border-rose-900/50 bg-rose-950/30 px-4 py-3 text-sm text-rose-200">
          {error}
        </div>
      )}

      {loading ? (
        <Panel>Carregando orçamentos…</Panel>
      ) : quotes.length === 0 ? (
        <Panel>Nenhum orçamento salvo. Clique em “Novo orçamento” pra montar o primeiro.</Panel>
      ) : (
        <div className="overflow-x-auto rounded-xl border border-slate-800 bg-slate-900/40">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-slate-800 text-left text-xs uppercase tracking-wide text-slate-500">
                <th className="w-full px-3 py-2 font-medium">Orçamento</th>
                <th className="whitespace-nowrap px-3 py-2 text-right font-medium">Cartas</th>
                <th className="whitespace-nowrap px-3 py-2 text-right font-medium">Mercado</th>
                <th className="whitespace-nowrap px-3 py-2 text-right font-medium">Oferta</th>
                <th className="whitespace-nowrap px-3 py-2 text-right font-medium">Atualizado</th>
                <th className="px-3 py-2" />
              </tr>
            </thead>
            <tbody>
              {quotes.map((q) => {
                const total = quoteTotal(q.items);
                const units = q.items.reduce((sum, it) => sum + it.qty, 0);
                return (
                  <tr key={q.id} className="border-b border-slate-800/60 last:border-0">
                    <td className="w-full max-w-0 px-3 py-2">
                      <button
                        onClick={() => setEditing(q)}
                        className="flex w-full min-w-0 items-center gap-2 text-left font-medium text-slate-100 hover:text-sky-300"
                      >
                        <FileText className="h-4 w-4 shrink-0 text-slate-500" />
                        <span className="truncate">{q.name}</span>
                      </button>
                    </td>
                    <td className="whitespace-nowrap px-3 py-2 text-right tabular-nums text-slate-300">
                      {units}
                      <span className="text-slate-500"> un</span>
                    </td>
                    <td className="whitespace-nowrap px-3 py-2 text-right tabular-nums text-slate-300">{brl0(total)}</td>
                    <td className="whitespace-nowrap px-3 py-2 text-right font-semibold tabular-nums text-emerald-300">
                      {brl0(quoteOffer(q.items, q.pct))}
                      <span className="ml-1 text-[10px] font-normal text-slate-500">
                        {blendedPct(q.items, q.pct)}%
                      </span>
                    </td>
                    <td className="whitespace-nowrap px-3 py-2 text-right text-xs text-slate-500">{timeAgo(q.updatedAt)}</td>
                    <td className="px-3 py-2">
                      <div className="flex justify-end">
                        <button
                          onClick={() => {
                            if (confirm(`Excluir o orçamento "${q.name}"?`)) {
                              deleteQuote(q.id).then(load);
                            }
                          }}
                          className="rounded-md border border-slate-700 bg-slate-800/60 p-1.5 text-slate-400 hover:bg-rose-950/40 hover:text-rose-300"
                          title="Excluir"
                        >
                          <Trash2 className="h-3.5 w-3.5" />
                        </button>
                      </div>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

function QuoteEditor({
  quote,
  onBack,
  onSaved,
}: {
  quote: Quote;
  onBack: () => void;
  onSaved: (q: Quote) => void;
}) {
  const brOnlyGame = isBRGame();
  const [name, setName] = useState(quote.name);
  const [pct, setPct] = useState(quote.pct);
  const [market, setMarket] = useState<QuoteMarket>(
    brOnlyGame ? "liga" : quote.market === "liga" ? "liga" : "tcg",
  );
  const [items, setItems] = useState<QuoteItem[]>(quote.items);
  const [fxRate, setFxRate] = useState(quote.fxRate ?? 0);
  const [saving, setSaving] = useState(false);
  const [copied, setCopied] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const total = quoteTotal(items);
  const offer = quoteOffer(items, pct);
  const brOnly = brOnlyGame;

  useEffect(() => {
    const missing = quote.items.filter((it) => !it.productID && !it.ligaUrl);
    if (missing.length === 0) {
      return;
    }
    let current = true;
    Promise.all(
      missing.map((it) =>
        getQuote(it.number || it.name, 50)
          .then((r) => r.matches.find((m) => m.number === it.number && m.name === it.name))
          .catch(() => undefined),
      ),
    ).then((found) => {
      if (!current) {
        return;
      }
      setItems((prev) =>
        prev.map((it) => {
          const m = found.find((f) => f && f.number === it.number && f.name === it.name);
          if (!m) {
            return it;
          }
          return {
            ...it,
            ligaUrl: it.ligaUrl ?? m.ligaUrl,
            productID: it.productID ?? m.productID,
          };
        }),
      );
    });
    return () => {
      current = false;
    };
  }, [quote]);

  const save = async () => {
    setSaving(true);
    try {
      const body = { name, pct, market, fxRate, items, notes: quote.notes };
      const saved = quote.id ? await updateQuote(quote.id, body) : await createQuote(body);
      onSaved(saved);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "failed to save quote");
    } finally {
      setSaving(false);
    }
  };

  const copy = async () => {
    const q: Quote = { ...quote, name: name || "lote", pct, items };
    await navigator.clipboard.writeText(buildQuoteText(q));
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1500);
  };

  const addItem = (m: QuoteMatch, fx: number) => {
    setFxRate(fx);
    setItems((prev) => {
      const idx = prev.findIndex((it) => it.number === m.number && it.name === m.name);
      if (idx >= 0) {
        return prev.map((it, i) => (i === idx ? { ...it, qty: it.qty + 1 } : it));
      }
      return [
        ...prev,
        {
          number: m.number,
          name: m.name,
          set: m.set,
          variant: m.variant,
          qty: 1,
          unitBRL: seedUnitBRL(market, m.ligaLowBRL, m.marketUSD, fx),
          marketUSD: m.marketUSD,
          ligaLowBRL: m.ligaLowBRL,
          ligaAvgBRL: m.ligaAvgBRL,
          ligaUrl: m.ligaUrl,
          productID: m.productID,
        },
      ];
    });
  };

  const changeMarket = (next: QuoteMarket) => {
    setMarket(next);
    setItems((prev) =>
      prev.map((it) => ({
        ...it,
        unitBRL: seedUnitBRL(next, it.ligaLowBRL, it.marketUSD, fxRate),
      })),
    );
  };

  const patchItem = (idx: number, patch: Partial<QuoteItem>) => {
    setItems((prev) => prev.map((it, i) => (i === idx ? { ...it, ...patch } : it)));
  };

  const removeItem = (idx: number) => {
    setItems((prev) => prev.filter((_, i) => i !== idx));
  };

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div className="flex flex-wrap items-end gap-3">
          <Button variant="ghost" onClick={onBack} title="Voltar pra lista">
            <ArrowLeft /> Voltar
          </Button>
          <Field label="Nome do orçamento">
            <Input
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="ex: lote do João 10/07"
              className="w-64"
            />
          </Field>
          <Field label="Deságio (% do mercado)">
            <div className="flex items-center gap-2">
              <div className="relative">
                <Input
                  type="number"
                  value={pct}
                  onChange={(e) => setPct(Number(e.target.value))}
                  className="w-20 pr-7"
                />
                <span className="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 text-xs text-slate-500">
                  %
                </span>
              </div>
              <ToggleGroup value={String(pct)} onChange={(v) => setPct(Number(v))} options={pctPresets} />
            </div>
          </Field>
          {!brOnly && (
            <Field label="Mercado base">
              <ToggleGroup
                value={market}
                onChange={(v) => changeMarket(v === "liga" ? "liga" : "tcg")}
                options={marketOptions}
              />
            </Field>
          )}
        </div>
        <Button variant="primary" onClick={save} disabled={saving}>
          {saving ? "Salvando…" : quote.id ? "Salvar alterações" : "Salvar orçamento"}
        </Button>
      </div>

      {error && (
        <div className="rounded-xl border border-rose-900/50 bg-rose-950/30 px-4 py-3 text-sm text-rose-200">
          {error}
        </div>
      )}

      <CardSearch onPick={addItem} brOnly={brOnly} />

      {items.length === 0 ? (
        <Panel>Pesquise uma carta acima pra começar o orçamento.</Panel>
      ) : (
        <div className="overflow-x-auto rounded-xl border border-slate-800 bg-slate-900/40">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-slate-800 text-left text-xs uppercase tracking-wide text-slate-500">
                <th className="w-full px-3 py-2 font-medium">Carta</th>
                <th className="whitespace-nowrap px-3 py-2 text-right font-medium">Qtd</th>
                <th className="whitespace-nowrap px-3 py-2 text-right font-medium">Liga R$</th>
                {!brOnly && <th className="whitespace-nowrap px-3 py-2 text-right font-medium">TCG</th>}
                <th className="whitespace-nowrap px-3 py-2 text-right font-medium">Unitário R$</th>
                <th className="whitespace-nowrap px-3 py-2 text-right font-medium">%</th>
                <th className="whitespace-nowrap px-3 py-2 text-right font-medium">Total</th>
                <th className="whitespace-nowrap px-3 py-2 text-right font-medium">Oferta</th>
                <th className="px-3 py-2" />
              </tr>
            </thead>
            <tbody>
              {items.map((it, idx) => (
                <tr key={`${it.number}-${it.name}`} className="border-b border-slate-800/60 last:border-0">
                  <td className="w-full max-w-0 px-3 py-2">
                    <div className="flex items-center gap-2">
                      <CardArt set={it.set} number={it.number} name={it.name} productID={it.productID} className="h-12 w-[34px] shrink-0 rounded" />
                      <div className="min-w-0">
                        <div className="truncate font-medium text-slate-100" title={it.name}>
                          {it.name}
                        </div>
                        <div className="flex items-center gap-1.5 font-mono text-[10px] text-slate-500">
                          {it.number}
                          {it.variant && <Badge variant="sky">{it.variant}</Badge>}
                        </div>
                      </div>
                    </div>
                  </td>
                  <td className="px-3 py-2 text-right">
                    <Input
                      type="number"
                      min={1}
                      value={it.qty}
                      onChange={(e) => patchItem(idx, { qty: Math.max(1, Number(e.target.value) || 1) })}
                      className="ml-auto w-16 text-right"
                    />
                  </td>
                  <td className="whitespace-nowrap px-3 py-2 text-right tabular-nums text-slate-300">
                    {it.ligaLowBRL ? brl0(it.ligaLowBRL) : "—"}
                    {it.ligaAvgBRL ? (
                      <div className="whitespace-nowrap text-[10px] text-slate-500">méd {brl0(it.ligaAvgBRL)}</div>
                    ) : null}
                  </td>
                  {!brOnly && (
                    <td className="whitespace-nowrap px-3 py-2 text-right tabular-nums text-slate-300">
                      {it.marketUSD ? usd(it.marketUSD) : "—"}
                      {it.marketUSD && fxRate > 0 ? (
                        <div className="whitespace-nowrap text-[10px] text-slate-500">≈ {brl0(it.marketUSD / fxRate)}</div>
                      ) : null}
                    </td>
                  )}
                  <td className="px-3 py-2 text-right">
                    <Input
                      type="number"
                      min={0}
                      value={it.unitBRL}
                      onChange={(e) => patchItem(idx, { unitBRL: Math.max(0, Number(e.target.value) || 0) })}
                      className="ml-auto w-24 text-right"
                    />
                  </td>
                  <td className="px-3 py-2 text-right">
                    <div className="relative ml-auto w-20">
                      <Input
                        type="number"
                        min={0}
                        value={effPct(it, pct)}
                        onChange={(e) => patchItem(idx, { pct: Math.max(0, Number(e.target.value) || 0) })}
                        className="w-20 pr-7 text-right"
                      />
                      <span className="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 text-xs text-slate-500">
                        %
                      </span>
                    </div>
                  </td>
                  <td className="whitespace-nowrap px-3 py-2 text-right font-medium tabular-nums text-slate-200">
                    {brl0(it.unitBRL * it.qty)}
                  </td>
                  <td className="whitespace-nowrap px-3 py-2 text-right font-semibold tabular-nums text-emerald-300">
                    {brl0((it.unitBRL * it.qty * effPct(it, pct)) / 100)}
                  </td>
                  <td className="whitespace-nowrap px-3 py-2">
                    <div className="flex items-center justify-end gap-1">
                      {it.ligaUrl && (
                        <a
                          href={it.ligaUrl}
                          target="_blank"
                          rel="noreferrer"
                          className="flex items-center gap-1 rounded-md bg-amber-500/10 px-2 py-1 text-xs font-medium text-amber-300 ring-1 ring-inset ring-amber-500/30 hover:bg-amber-500/20"
                          title="Ver na Liga"
                        >
                          Liga <ExternalLink className="h-3 w-3" />
                        </a>
                      )}
                      {!brOnly && it.productID ? (
                        <a
                          href={tcgProductURL(it.productID)}
                          target="_blank"
                          rel="noreferrer"
                          className="flex items-center gap-1 rounded-md bg-sky-500/10 px-2 py-1 text-xs font-medium text-sky-300 ring-1 ring-inset ring-sky-500/30 hover:bg-sky-500/20"
                          title="Ver no TCGplayer"
                        >
                          TCG <ExternalLink className="h-3 w-3" />
                        </a>
                      ) : null}
                      <button
                        onClick={() => removeItem(idx)}
                        className="rounded-md border border-slate-700 bg-slate-800/60 p-1.5 text-slate-400 hover:bg-rose-950/40 hover:text-rose-300"
                        title="Remover"
                      >
                        <Trash2 className="h-3.5 w-3.5" />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {items.length > 0 && (
        <Card className="flex flex-wrap items-center justify-between gap-3 p-4">
          <div className="flex flex-wrap items-center gap-6">
            <div>
              <div className="text-[10px] font-medium uppercase tracking-wide text-slate-500">
                Total mercado
              </div>
              <div className="text-lg font-semibold tabular-nums text-slate-200">{brl0(total)}</div>
            </div>
            <div>
              <div className="text-[10px] font-medium uppercase tracking-wide text-slate-500">
                Oferta ({blendedPct(items, pct)}%)
              </div>
              <div className="text-lg font-bold tabular-nums text-emerald-300">{brl0(offer)}</div>
            </div>
          </div>
          <Button variant="outline" onClick={copy}>
            {copied ? <Check className="text-emerald-300" /> : <Copy />}
            {copied ? "Copiado!" : "Copiar WhatsApp"}
          </Button>
        </Card>
      )}
    </div>
  );
}

function CardSearch({
  onPick,
  brOnly,
}: {
  onPick: (m: QuoteMatch, fxRate: number) => void;
  brOnly: boolean;
}) {
  const [query, setQuery] = useState("");
  const [matches, setMatches] = useState<QuoteMatch[]>([]);
  const [fx, setFx] = useState(0);

  useEffect(() => {
    if (query.trim().length < 2) {
      setMatches([]);
      return;
    }
    let current = true;
    const h = window.setTimeout(() => {
      getQuote(query, 50)
        .then((r) => {
          if (current) {
            setMatches(r.matches);
            setFx(r.fxRate);
          }
        })
        .catch(() => current && setMatches([]));
    }, 250);
    return () => {
      current = false;
      window.clearTimeout(h);
    };
  }, [query]);

  return (
    <div className="relative">
      <Field label="Adicionar carta (nome ou número)">
        <Input
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="ex: Zoro ou OP06-118"
          className="w-full"
        />
      </Field>
      {matches.length > 0 && (
        <ul className="absolute z-10 mt-1 max-h-72 w-full overflow-y-auto rounded-lg border border-slate-700 bg-slate-900 shadow-xl">
          {matches.map((m) => (
            <li key={`${m.number}-${m.name}`}>
              <button
                type="button"
                onClick={() => {
                  onPick(m, fx);
                  setQuery("");
                  setMatches([]);
                }}
                className="flex w-full items-center justify-between gap-3 px-3 py-2 text-left text-sm hover:bg-slate-800"
              >
                <span className="flex min-w-0 items-center gap-2">
                  <CardArt
                    set={m.set}
                    number={m.number}
                    name={m.name}
                    productID={m.productID}
                    className="h-14 w-[40px] shrink-0 rounded"
                  />
                  <span className="min-w-0">
                    <span className="block truncate text-slate-100">{m.name}</span>
                    <span className="font-mono text-[11px] text-slate-500">{m.number}</span>
                  </span>
                </span>
                <span className="flex shrink-0 items-center gap-3 tabular-nums">
                  <span className="text-slate-400">
                    Liga {m.ligaLowBRL ? brl0(m.ligaLowBRL) : "—"}
                  </span>
                  {!brOnly && (
                    <span className="text-emerald-300">
                      {usd(m.marketUSD)}
                      <span className="ml-1 text-[11px] text-slate-500">≈ {brl0(m.marketBRL)}</span>
                    </span>
                  )}
                </span>
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
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
