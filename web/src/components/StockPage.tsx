import { useCallback, useEffect, useMemo, useState } from "react";
import { Search, Store, Check, X, Wand2, Save, Combine, ImagePlus } from "lucide-react";
import {
  gameHasDeals,
  getGame,
  getPortfolio,
  mergeTrades,
  productIDFromTcgURL,
  setListings,
  type ListingInput,
  type PortfolioResponse,
  type TradeView,
} from "../api";
import { brl, brl0, usd } from "../format";
import CardArt from "./CardArt";
import { Button } from "./ui/button";
import { Card } from "./ui/card";
import { Checkbox } from "./ui/checkbox";
import { Input } from "./ui/input";
import { ToggleGroup } from "./ui/toggle-group";

// StockPage is the seller-facing control room for the public storefront: pick
// which held cards go on sale and set each one's asking price (in BRL). The
// public catalog only shows a card when it is both listed and priced.
const isBRGame = () => !gameHasDeals(getGame());

function cleanName(n: string): string {
  return n.replace(/\s*\([^)]*\)\s*$/, "");
}

const suggestOptions = [
  { value: "80", label: "80%" },
  { value: "85", label: "85%" },
  { value: "90", label: "90%" },
  { value: "100", label: "100%" },
];

interface RowState {
  askBRL: number;
  listed: boolean;
  imageURL: string;
}

// refBRL is a card's per-unit market reference in reais: for deals games the live
// TCGplayer price converted at the FX rate, for BR-only games the Liga floor
// (already in BRL, fx = 1), or the manual estimate for sealed products.
function refBRL(t: TradeView, fx: number): number {
  if (t.marketKnown && fx > 0) {
    return t.marketUSD / fx;
  }
  return t.manualBRL ?? 0;
}

function round2(n: number): number {
  return Math.round(n * 100) / 100;
}

export default function StockPage() {
  const [data, setData] = useState<PortfolioResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [query, setQuery] = useState("");
  const [rows, setRows] = useState<Record<string, RowState>>({});
  const [saving, setSaving] = useState(false);
  const [savedAt, setSavedAt] = useState(0);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [confirmMerge, setConfirmMerge] = useState(false);
  const [merging, setMerging] = useState(false);
  const [imgEditId, setImgEditId] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const r = await getPortfolio(100);
      setData(r);
      const seeded: Record<string, RowState> = {};
      for (const t of r.trades) {
        if (t.realized) {
          continue;
        }
        seeded[t.id] = {
          askBRL: t.askBRL ?? 0,
          listed: Boolean(t.listed),
          imageURL: t.imageURL ?? "",
        };
      }
      setRows(seeded);
      setSelected(new Set());
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "failed to load stock");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const fx = data?.fxRate ?? 0;
  const holdings = useMemo(() => (data?.trades ?? []).filter((t) => !t.realized), [data]);

  const q = query.trim().toLowerCase();
  const visible = useMemo(
    () =>
      holdings.filter((t) =>
        `${t.name} ${t.number} ${t.store ?? ""}`.toLowerCase().includes(q),
      ),
    [holdings, q],
  );

  const dirty = useMemo(() => {
    if (!data) {
      return false;
    }
    return holdings.some((t) => {
      const r = rows[t.id];
      return (
        r &&
        (round2(r.askBRL) !== round2(t.askBRL ?? 0) ||
          r.listed !== Boolean(t.listed) ||
          (r.imageURL ?? "") !== (t.imageURL ?? ""))
      );
    });
  }, [holdings, rows, data]);

  const stats = useMemo(() => {
    let onSale = 0;
    let value = 0;
    for (const t of holdings) {
      const r = rows[t.id];
      if (r?.listed && r.askBRL > 0) {
        onSale += 1;
        value += r.askBRL * Math.max(t.qty, 1);
      }
    }
    return { onSale, value };
  }, [holdings, rows]);

  const patch = (id: string, next: Partial<RowState>) =>
    setRows((prev) => ({ ...prev, [id]: { ...prev[id], ...next } }));

  const toggleSelect = (id: string) =>
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });

  const allVisibleSelected = visible.length > 0 && visible.every((t) => selected.has(t.id));
  const toggleSelectAllVisible = () =>
    setSelected((prev) => {
      const next = new Set(prev);
      if (allVisibleSelected) {
        for (const t of visible) {
          next.delete(t.id);
        }
      } else {
        for (const t of visible) {
          next.add(t.id);
        }
      }
      return next;
    });

  // mergePreview mirrors trades.Store.Merge: the oldest holding is the primary
  // (keeps its identity, price and listed state); quantity and shipping sum, and
  // the per-unit cost is the quantity-weighted average so the total cost is kept.
  const selectedTrades = useMemo(
    () => holdings.filter((t) => selected.has(t.id)),
    [holdings, selected],
  );
  const mergePreview = useMemo(() => {
    if (selectedTrades.length < 2) {
      return null;
    }
    const primary = selectedTrades.reduce((a, b) => (b.createdAt < a.createdAt ? b : a));
    const qty = selectedTrades.reduce((sum, t) => sum + Math.max(t.qty, 1), 0);
    const totalCost = selectedTrades.reduce((sum, t) => sum + t.costBRL, 0);
    const totalShip = selectedTrades.reduce((sum, t) => sum + t.shippingBRL, 0);
    const unit = qty > 0 ? round2((totalCost - totalShip) / qty) : 0;
    const differentCards = new Set(selectedTrades.map((t) => t.number)).size > 1;
    return { primary, qty, totalCost, unit, differentCards };
  }, [selectedTrades]);

  const doMerge = async () => {
    if (!mergePreview) {
      return;
    }
    setMerging(true);
    try {
      await mergeTrades(selectedTrades.map((t) => t.id));
      setConfirmMerge(false);
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "failed to merge");
    } finally {
      setMerging(false);
    }
  };

  // suggestAll seeds every visible card's price from its market reference at the
  // chosen %, without changing whether it's listed — a starting point to tweak.
  const suggestAll = (pct: number) => {
    setRows((prev) => {
      const next = { ...prev };
      for (const t of visible) {
        const base = refBRL(t, fx);
        if (base > 0) {
          next[t.id] = { ...next[t.id], askBRL: round2(base * (pct / 100)) };
        }
      }
      return next;
    });
  };

  const listAll = (listed: boolean) => {
    setRows((prev) => {
      const next = { ...prev };
      for (const t of visible) {
        next[t.id] = { ...next[t.id], listed };
      }
      return next;
    });
  };

  const save = async () => {
    setSaving(true);
    try {
      const items: ListingInput[] = holdings.map((t) => ({
        id: t.id,
        askBRL: rows[t.id]?.askBRL ?? 0,
        listed: Boolean(rows[t.id]?.listed),
        imageURL: rows[t.id]?.imageURL ?? "",
      }));
      await setListings(items);
      setSavedAt(Date.now());
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "failed to save");
    } finally {
      setSaving(false);
    }
  };

  const imgEditTrade = imgEditId
    ? holdings.find((h) => h.id === imgEditId) ?? null
    : null;

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="font-display text-lg font-extrabold text-white">Estoque</h1>
          <p className="mt-1 max-w-2xl text-sm text-slate-400">
            Defina o preço de venda (R$) e marque o que deve aparecer na vitrine pública.
            Uma carta só entra na vitrine quando está <strong className="text-slate-300">à venda</strong> e tem preço.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="accent" onClick={save} disabled={!dirty || saving}>
            <Save /> {saving ? "Salvando…" : dirty ? "Salvar alterações" : savedAt ? "Salvo" : "Salvar"}
          </Button>
        </div>
      </div>

      <div className="grid grid-cols-2 gap-3 sm:grid-cols-3">
        <Card className="flex items-center gap-3 p-4">
          <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-[8px] border-2 border-outline bg-brand text-white">
            <Store className="h-5 w-5" />
          </div>
          <div className="min-w-0">
            <div className="text-[10px] font-semibold uppercase tracking-wide text-brand-soft">Na vitrine</div>
            <div className="truncate text-xl font-bold tabular-nums text-white">{stats.onSale}</div>
            <div className="truncate text-[11px] text-slate-500">de {holdings.length} em estoque</div>
          </div>
        </Card>
        <Card className="flex items-center gap-3 p-4">
          <div className="min-w-0">
            <div className="text-[10px] font-semibold uppercase tracking-wide text-brand-soft">Valor pedido</div>
            <div className="truncate text-xl font-bold tabular-nums text-white">{brl0(stats.value)}</div>
            <div className="truncate text-[11px] text-slate-500">soma dos preços × qtd</div>
          </div>
        </Card>
        <Card className="hidden items-center gap-3 p-4 sm:flex">
          <div className="min-w-0">
            <div className="text-[10px] font-semibold uppercase tracking-wide text-brand-soft">Vitrine pública</div>
            <div className="truncate text-sm font-medium text-slate-300">em breve (Vercel)</div>
            <div className="truncate text-[11px] text-slate-500">link para enviar aos clientes</div>
          </div>
        </Card>
      </div>

      {error && (
        <div className="rounded-[14px] border-2 border-rose-900/50 bg-rose-950/30 px-4 py-3 text-sm text-rose-200">{error}</div>
      )}

      <div className="flex flex-wrap items-center gap-2">
        <div className="relative w-full min-w-[200px] max-w-xs sm:w-auto sm:flex-1">
          <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-500" />
          <Input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Buscar carta, número, loja…"
            className="w-full pl-9 pr-8"
          />
          {query && (
            <button
              onClick={() => setQuery("")}
              className="absolute right-2 top-1/2 -translate-y-1/2 rounded p-0.5 text-slate-500 hover:text-slate-300"
              title="Limpar"
            >
              <X className="h-4 w-4" />
            </button>
          )}
        </div>
        <div className="ml-auto flex flex-wrap items-center gap-2">
          <div className="flex items-center gap-2">
            <span className="flex items-center gap-1 text-[10px] font-medium uppercase tracking-wide text-slate-500">
              <Wand2 className="h-3 w-3" /> Sugerir
            </span>
            <ToggleGroup value="" onChange={(v) => suggestAll(Number(v))} options={suggestOptions} />
          </div>
          <Button variant="outline" onClick={() => listAll(true)} disabled={visible.length === 0}>
            Listar todas
          </Button>
          <Button variant="outline" onClick={() => listAll(false)} disabled={visible.length === 0}>
            Ocultar todas
          </Button>
        </div>
      </div>

      {selected.size > 0 && (
        <div className="sticker flex flex-wrap items-center gap-3 rounded-[12px] bg-surface px-4 py-3">
          <span className="font-pixel text-[9px] uppercase text-brand-soft">
            {selected.size} selecionada{selected.size > 1 ? "s" : ""}
          </span>
          {selected.size < 2 && (
            <span className="text-xs text-slate-500">selecione 2+ para combinar</span>
          )}
          <div className="ml-auto flex items-center gap-2">
            <Button variant="accent" onClick={() => setConfirmMerge(true)} disabled={selected.size < 2}>
              <Combine /> Combinar ({selected.size})
            </Button>
            <Button variant="ghost" onClick={() => setSelected(new Set())}>
              Limpar
            </Button>
          </div>
        </div>
      )}

      {loading ? (
        <Panel>Carregando estoque…</Panel>
      ) : holdings.length === 0 ? (
        <Panel>Nenhuma carta em estoque. Adicione compras no Portfolio para vê-las aqui.</Panel>
      ) : visible.length === 0 ? (
        <Panel>Nenhuma carta corresponde à busca.</Panel>
      ) : (
        <div className="overflow-x-auto rounded-[14px] border-[3px] border-outline bg-surface">
          <table className="w-full min-w-[720px] text-sm">
            <thead>
              <tr className="border-b-2 border-outline text-left text-[10px] uppercase tracking-wide text-muted">
                <th className="w-8 px-3 py-2">
                  <Checkbox
                    checked={allVisibleSelected}
                    onChange={toggleSelectAllVisible}
                    aria-label="Selecionar todas as visíveis"
                  />
                </th>
                <th className="px-3 py-2 font-bold">Carta</th>
                <th className="px-3 py-2 text-right font-bold">Qtd</th>
                <th className="px-3 py-2 text-right font-bold">{isBRGame() ? "Floor" : "TCG"} R$</th>
                <th className="px-3 py-2 text-right font-bold">Preço R$ (un.)</th>
                <th className="px-3 py-2 text-right font-bold">Total</th>
                <th className="px-3 py-2 text-center font-bold">Na vitrine</th>
              </tr>
            </thead>
            <tbody>
              {visible.map((t) => (
                <StockRow
                  key={t.id}
                  t={t}
                  fx={fx}
                  row={rows[t.id]}
                  onPatch={patch}
                  selected={selected.has(t.id)}
                  onToggle={toggleSelect}
                  onEditImage={setImgEditId}
                />
              ))}
            </tbody>
          </table>
        </div>
      )}

      {confirmMerge && mergePreview && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 p-4"
          onClick={() => !merging && setConfirmMerge(false)}
        >
          <div className="sticker sticker-5 w-full max-w-md rounded-[16px] bg-surface p-6" onClick={(e) => e.stopPropagation()}>
            <div className="font-pixel text-[10px] uppercase text-brand-soft">
              Combinar {selectedTrades.length} produtos
            </div>
            <h2 className="font-display mt-2 text-lg font-extrabold text-white">
              {cleanName(mergePreview.primary.name) || mergePreview.primary.number}
            </h2>
            <p className="mt-1 text-xs text-slate-400">
              Vira uma linha só. Mantém o preço, o "à venda" e a identidade da compra mais antiga; some as
              duplicadas também no Portfolio. <strong className="text-slate-300">Isto é permanente.</strong>
            </p>
            <div className="mt-4 grid grid-cols-3 gap-2 text-center">
              <div className="rounded-[10px] border-2 border-outline bg-ink px-2 py-3">
                <div className="font-pixel text-[7px] uppercase text-brand-soft">Qtd</div>
                <div className="mt-1 text-lg font-bold tabular-nums text-white">{mergePreview.qty}</div>
              </div>
              <div className="rounded-[10px] border-2 border-outline bg-ink px-2 py-3">
                <div className="font-pixel text-[7px] uppercase text-brand-soft">Custo un.</div>
                <div className="mt-1 text-lg font-bold tabular-nums text-white">{brl(mergePreview.unit)}</div>
              </div>
              <div className="rounded-[10px] border-2 border-outline bg-ink px-2 py-3">
                <div className="font-pixel text-[7px] uppercase text-brand-soft">Custo total</div>
                <div className="mt-1 text-lg font-bold tabular-nums text-white">{brl(mergePreview.totalCost)}</div>
              </div>
            </div>
            {mergePreview.differentCards && (
              <div className="mt-3 rounded-[10px] border-2 border-amber-500/40 bg-amber-500/10 px-3 py-2 text-xs text-amber-300">
                Atenção: você selecionou cartas com números diferentes — elas virarão uma linha só.
              </div>
            )}
            <div className="mt-5 flex justify-end gap-2">
              <Button variant="ghost" onClick={() => setConfirmMerge(false)} disabled={merging}>
                Cancelar
              </Button>
              <Button variant="accent" onClick={doMerge} disabled={merging}>
                <Combine /> {merging ? "Combinando…" : "Combinar"}
              </Button>
            </div>
          </div>
        </div>
      )}

      {imgEditId && imgEditTrade && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 p-4"
          onClick={() => setImgEditId(null)}
        >
          <div
            className="sticker sticker-5 w-full max-w-md rounded-[16px] bg-surface p-6"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="font-pixel text-[10px] uppercase text-brand-soft">
              Imagem do produto
            </div>
            <h2 className="font-display mt-2 text-lg font-extrabold text-white">
              {cleanName(imgEditTrade.name) || imgEditTrade.number}
            </h2>
            <p className="mt-1 text-xs text-slate-400">
              Cole a URL de uma imagem — ela aparece na vitrine no lugar do ícone padrão.
            </p>

            <div className="mt-4 flex items-start gap-4">
              <div className="h-[112px] w-[80px] shrink-0 overflow-hidden rounded-[8px] border-2 border-outline">
                <CardArt
                  set={imgEditTrade.set}
                  number={imgEditTrade.number}
                  name={imgEditTrade.name}
                  productID={productIDFromTcgURL(imgEditTrade.tcgUrl)}
                  imageURL={rows[imgEditId]?.imageURL}
                  className="h-[112px] w-[80px]"
                />
              </div>
              <div className="min-w-0 flex-1">
                <label className="font-pixel text-[8px] uppercase text-brand-soft">
                  URL da imagem
                </label>
                <Input
                  autoFocus
                  value={rows[imgEditId]?.imageURL ?? ""}
                  onChange={(e) => patch(imgEditId, { imageURL: e.target.value })}
                  placeholder="https://…/imagem.jpg"
                  className="mt-1 w-full"
                />
                <p className="mt-2 text-[11px] text-slate-500">
                  Deixe em branco para usar o ícone padrão.
                </p>
              </div>
            </div>

            <div className="mt-5 flex items-center justify-between gap-2">
              <Button
                variant="ghost"
                onClick={() => patch(imgEditId, { imageURL: "" })}
                disabled={!(rows[imgEditId]?.imageURL ?? "")}
              >
                <X /> Remover
              </Button>
              <div className="flex items-center gap-2">
                <Button variant="ghost" onClick={() => setImgEditId(null)}>
                  Fechar
                </Button>
                <Button
                  variant="accent"
                  onClick={() => {
                    setImgEditId(null);
                    void save();
                  }}
                  disabled={saving}
                >
                  <Save /> {saving ? "Salvando…" : "Salvar"}
                </Button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

function StockRow({
  t,
  fx,
  row,
  onPatch,
  selected,
  onToggle,
  onEditImage,
}: {
  t: TradeView;
  fx: number;
  row: RowState | undefined;
  onPatch: (id: string, next: Partial<RowState>) => void;
  selected: boolean;
  onToggle: (id: string) => void;
  onEditImage: (id: string) => void;
}) {
  const ask = row?.askBRL ?? 0;
  const listed = Boolean(row?.listed);
  const ref = refBRL(t, fx);
  const live = listed && ask > 0;
  const total = ask * Math.max(t.qty, 1);
  const askUSD = fx > 0 ? ask * fx : 0;

  return (
    <tr className={`border-b-2 border-outline/40 last:border-0 hover:bg-slate-800/40 ${selected ? "bg-brand/10" : ""}`}>
      <td className="px-3 py-2 text-center">
        <Checkbox checked={selected} onChange={() => onToggle(t.id)} aria-label={`Selecionar ${t.name}`} />
      </td>
      <td className="px-3 py-2">
        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={() => onEditImage(t.id)}
            title="Definir imagem do produto"
            className="group relative h-12 w-[34px] shrink-0 overflow-hidden rounded"
          >
            <CardArt
              set={t.set}
              number={t.number}
              name={t.name}
              productID={productIDFromTcgURL(t.tcgUrl)}
              imageURL={row?.imageURL}
              className="h-12 w-[34px] rounded"
            />
            <span className="absolute inset-0 flex items-center justify-center bg-black/50 opacity-0 transition-opacity group-hover:opacity-100">
              <ImagePlus className="h-3.5 w-3.5 text-white" />
            </span>
          </button>
          <div className="min-w-0">
            <div className="truncate font-medium text-slate-100" title={t.name}>
              {cleanName(t.name) || t.number}
            </div>
            <div className="font-mono text-[10px] text-slate-500">
              {t.number}
              {t.condition ? ` · ${t.condition}` : ""}
              {t.store ? ` · ${t.store}` : ""}
            </div>
          </div>
        </div>
      </td>
      <td className="px-3 py-2 text-right tabular-nums text-slate-300">{t.qty}</td>
      <td className="px-3 py-2 text-right tabular-nums text-slate-400">{ref > 0 ? brl0(ref) : "—"}</td>
      <td className="px-3 py-2 text-right">
        <div className="flex items-center justify-end gap-1">
          <span className="text-xs text-slate-500">R$</span>
          <Input
            type="number"
            value={ask ? String(ask) : ""}
            onChange={(e) => onPatch(t.id, { askBRL: Number(e.target.value) || 0 })}
            placeholder="0"
            className="w-24 text-right"
          />
        </div>
        {ask > 0 && fx > 0 && (
          <div className="mt-0.5 text-[10px] tabular-nums text-slate-500">≈ {usd(askUSD)}</div>
        )}
      </td>
      <td className="px-3 py-2 text-right tabular-nums text-slate-300">{ask > 0 ? brl0(total) : "—"}</td>
      <td className="px-3 py-2 text-center">
        <button
          onClick={() => onPatch(t.id, { listed: !listed })}
          title={live ? "Na vitrine — clique para ocultar" : listed ? "À venda, mas falta preço" : "Oculta — clique para colocar à venda"}
          className={`inline-flex items-center gap-1 rounded-[8px] border-2 border-outline px-2 py-1 text-xs font-bold ${
            live
              ? "bg-emerald-500/10 text-emerald-300 hover:bg-emerald-500/20"
              : listed
                ? "bg-amber-500/10 text-amber-300 hover:bg-amber-500/20"
                : "bg-surface text-slate-400 hover:bg-slate-800"
          }`}
        >
          {live ? <Check className="h-3 w-3" /> : <Store className="h-3 w-3" />}
          {live ? "À venda" : listed ? "Sem preço" : "Oculta"}
        </button>
      </td>
    </tr>
  );
}

function Panel({ children }: { children: React.ReactNode }) {
  return (
    <div className="rounded-[14px] border-[3px] border-outline bg-surface px-4 py-10 text-center text-slate-400">
      {children}
    </div>
  );
}
