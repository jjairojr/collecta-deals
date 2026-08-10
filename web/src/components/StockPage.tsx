import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import EmptyState from "./EmptyState";
import {
  AlertTriangle,
  ArrowDown,
  ArrowUp,
  Check,
  ChevronsUpDown,
  Combine,
  Copy,
  ImagePlus,
  Save,
  Search,
  Share2,
  Star,
  Store,
  Tags,
  Wand2,
  X,
} from "lucide-react";
import {
  gameHasDeals,
  getGame,
  getPortfolio,
  mergeTrades,
  productIDFromTcgURL,
  setLigaState,
  setListings,
  type LigaInput,
  type ListingInput,
  type PortfolioResponse,
  type TradeView,
} from "../api";
import { brl, brl0, usd } from "../format";
import CardArt from "./CardArt";
import ShareList from "./ShareList";
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

// rowMeta is the small line under a product's name. Accessories say so, since
// they are shared by every game and show up in all of them.
function rowMeta(t: TradeView): string {
  return [
    t.kind === "accessory" ? "Acessório · todos os jogos" : t.number,
    t.variant,
    t.condition,
    t.store,
  ]
    .filter(Boolean)
    .join(" · ");
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
  featured: boolean;
  ligaListed: boolean;
}

// LigaSync says where a holding stands on the LigaMagic store. "stale" means it
// is registered there but the ledger moved since — the quantity or the price no
// longer match what was sent, so it needs a targeted reimport. Reimporting
// everything is not an option: Liga's import sums quantities instead of setting
// them, so a blind rerun doubles the stock.
type LigaSync = "off" | "sync" | "stale";

function ligaSync(t: TradeView): LigaSync {
  if (!t.ligaListed) {
    return "off";
  }
  const qtyDrift = t.qty !== (t.ligaQty ?? 0);
  const priceDrift = round2(t.askBRL ?? 0) !== round2(t.ligaPriceBRL ?? 0);
  return qtyDrift || priceDrift ? "stale" : "sync";
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

const PREVIEW_W = 240;
const PREVIEW_H = 336;

type SortKey = "name" | "qty" | "ref" | "ask" | "total" | "listed" | "liga";
interface SortState {
  key: SortKey;
  dir: "asc" | "desc";
}

// listedRank orders the vitrine column by how close a card is to being sold:
// on sale > listed but missing a price > hidden.
function listedRank(row: RowState | undefined): number {
  if (!row?.listed) {
    return 0;
  }
  return row.askBRL > 0 ? 2 : 1;
}

// ligaRank sorts the Liga column by how much attention a row needs: out of the
// store first, then needs reimport, then done.
function ligaRank(t: TradeView): number {
  switch (ligaSync(t)) {
    case "off":
      return 0;
    case "stale":
      return 1;
    case "sync":
      return 2;
  }
}

function sortValue(
  t: TradeView,
  key: SortKey,
  fx: number,
  row: RowState | undefined,
): number | string {
  switch (key) {
    case "name":
      return (cleanName(t.name) || t.number).toLowerCase();
    case "qty":
      return t.qty;
    case "ref":
      return refBRL(t, fx);
    case "ask":
      return row?.askBRL ?? 0;
    case "total":
      return (row?.askBRL ?? 0) * Math.max(t.qty, 1);
    case "listed":
      return listedRank(row);
    case "liga":
      return ligaRank(t);
  }
}

function sortStock(
  list: TradeView[],
  sort: SortState,
  fx: number,
  rows: Record<string, RowState>,
): TradeView[] {
  const dir = sort.dir === "asc" ? 1 : -1;
  return [...list].sort((a, b) => {
    const av = sortValue(a, sort.key, fx, rows[a.id]);
    const bv = sortValue(b, sort.key, fx, rows[b.id]);
    if (typeof av === "string" && typeof bv === "string") {
      return av.localeCompare(bv) * dir;
    }
    if (typeof av === "number" && typeof bv === "number") {
      return (av - bv) * dir;
    }
    return 0;
  });
}

function SortableTh({
  label,
  sortKey,
  sort,
  onSort,
  align = "right",
}: {
  label: string;
  sortKey: SortKey;
  sort: SortState;
  onSort: (key: SortKey) => void;
  align?: "left" | "right" | "center";
}) {
  const activeCol = sort.key === sortKey;
  const alignClass =
    align === "right" ? "text-right" : align === "center" ? "text-center" : "text-left";
  return (
    <th className={`px-3 py-2 font-bold ${alignClass}`}>
      <button
        type="button"
        onClick={() => onSort(sortKey)}
        className={`inline-flex items-center gap-1 hover:text-slate-300 ${align === "right" ? "flex-row-reverse" : ""} ${activeCol ? "text-slate-300" : ""}`}
      >
        {label}
        {activeCol ? (
          sort.dir === "asc" ? <ArrowUp className="h-3 w-3" /> : <ArrowDown className="h-3 w-3" />
        ) : (
          <ChevronsUpDown className="h-3 w-3 opacity-40" />
        )}
      </button>
    </th>
  );
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
  const [sort, setSort] = useState<SortState>({ key: "total", dir: "desc" });
  const [onlyMissingOnLiga, setOnlyMissingOnLiga] = useState(false);
  const [sharing, setSharing] = useState(false);

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
          featured: Boolean(t.featured),
          ligaListed: Boolean(t.ligaListed),
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

  // Sorting reads the draft prices/listings through a ref so a row never jumps
  // out from under the cursor while it is being edited: the order is only
  // recomputed when the sort, the search or the loaded data changes.
  const rowsRef = useRef(rows);
  rowsRef.current = rows;

  const q = query.trim().toLowerCase();
  const visible = useMemo(
    () =>
      sortStock(
        holdings.filter(
          (t) =>
            `${t.name} ${t.number} ${t.store ?? ""}`.toLowerCase().includes(q) &&
            (!onlyMissingOnLiga || ligaSync(t) !== "sync"),
        ),
        sort,
        fx,
        rowsRef.current,
      ),
    [holdings, q, sort, fx, onlyMissingOnLiga],
  );

  // The share panel posts exactly what is on screen, priced with the draft
  // edits: you can tweak a price for the post and send it before deciding
  // whether to save it to the ledger.
  const shareItems = useMemo(
    () =>
      visible.map((t) => ({
        ...t,
        name: cleanName(t.name) || t.number,
        askBRL: rows[t.id]?.askBRL ?? t.askBRL ?? 0,
        listed: rows[t.id]?.listed ?? Boolean(t.listed),
        imageURL: rows[t.id]?.imageURL || t.imageURL,
      })),
    [visible, rows],
  );

  const onSort = useCallback((key: SortKey) => {
    setSort((s) => (s.key === key ? { key, dir: s.dir === "asc" ? "desc" : "asc" } : { key, dir: "desc" }));
  }, []);

  // Only holdings whose Liga flag was actually toggled go in the Liga write. A
  // row that was already marked must not be resent, or saving an unrelated price
  // edit would refresh the recorded Liga quantity and silently clear a "needs
  // reimport" warning without anything having been sent to the store.
  const ligaChanged = useMemo(
    () => holdings.filter((t) => Boolean(rows[t.id]?.ligaListed) !== Boolean(t.ligaListed)),
    [holdings, rows],
  );

  const dirty = useMemo(() => {
    if (!data) {
      return false;
    }
    if (ligaChanged.length > 0) {
      return true;
    }
    return holdings.some((t) => {
      const r = rows[t.id];
      return (
        r &&
        (round2(r.askBRL) !== round2(t.askBRL ?? 0) ||
          r.listed !== Boolean(t.listed) ||
          (r.imageURL ?? "") !== (t.imageURL ?? "") ||
          r.featured !== Boolean(t.featured))
      );
    });
  }, [holdings, rows, data, ligaChanged]);

  const stats = useMemo(() => {
    let onSale = 0;
    let value = 0;
    let onLiga = 0;
    let staleOnLiga = 0;
    for (const t of holdings) {
      const r = rows[t.id];
      if (r?.listed && r.askBRL > 0) {
        onSale += 1;
        value += r.askBRL * Math.max(t.qty, 1);
      }
      const sync = ligaSync(t);
      if (sync !== "off") {
        onLiga += 1;
      }
      if (sync === "stale") {
        staleOnLiga += 1;
      }
    }
    return { onSale, value, onLiga, staleOnLiga };
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
        featured: Boolean(rows[t.id]?.featured),
      }));
      await setListings(items);
      if (ligaChanged.length > 0) {
        const ligaItems: LigaInput[] = ligaChanged.map((t) => ({
          id: t.id,
          ligaListed: Boolean(rows[t.id]?.ligaListed),
          ligaQty: t.qty,
          ligaPriceBRL: rows[t.id]?.askBRL ?? 0,
        }));
        await setLigaState(ligaItems);
      }
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
      <div>
        <p className="mt-1 max-w-2xl text-sm text-slate-400">
          Defina o preço de venda (R$) e marque o que deve aparecer na vitrine pública.
          Uma carta só entra na vitrine quando está <strong className="text-slate-300">à venda</strong> e tem preço.
          Os acessórios não pertencem a nenhum jogo: aparecem nesta lista em todos eles.
        </p>
      </div>

      <div className="grid grid-cols-2 gap-3 sm:grid-cols-3">
        <Card className="flex items-center gap-3 p-4">
          <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-[8px] border-2 border-outline bg-brand text-white">
            <Store className="h-5 w-5" />
          </div>
          <div className="min-w-0">
            <div className="text-[10px] font-semibold uppercase tracking-wide text-brand-label">Na vitrine</div>
            <div className="truncate text-xl font-bold tabular-nums text-fg">{stats.onSale}</div>
            <div className="truncate text-[11px] text-slate-500">de {holdings.length} em estoque</div>
          </div>
        </Card>
        <Card className="flex items-center gap-3 p-4">
          <div className="min-w-0">
            <div className="text-[10px] font-semibold uppercase tracking-wide text-brand-label">Valor pedido</div>
            <div className="truncate text-xl font-bold tabular-nums text-fg">{brl0(stats.value)}</div>
            <div className="truncate text-[11px] text-slate-500">soma dos preços × qtd</div>
          </div>
        </Card>
        <Card className="flex items-center gap-3 p-4">
          <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-[8px] border-2 border-outline bg-raised text-slate-300">
            <Tags className="h-5 w-5" />
          </div>
          <div className="min-w-0">
            <div className="text-[10px] font-semibold uppercase tracking-wide text-brand-label">Na LigaMagic</div>
            <div className="truncate text-xl font-bold tabular-nums text-fg">{stats.onLiga}</div>
            <div className="truncate text-[11px] text-slate-500">
              {stats.staleOnLiga > 0
                ? `${stats.staleOnLiga} precisa${stats.staleOnLiga > 1 ? "m" : ""} reimportar`
                : `de ${holdings.length} em estoque`}
            </div>
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
          <Button
            variant={onlyMissingOnLiga ? "accent" : "outline"}
            onClick={() => setOnlyMissingOnLiga((v) => !v)}
            title="Mostrar só o que ainda não está na LigaMagic ou precisa ser reimportado"
          >
            <Tags /> Faltam na Liga
          </Button>
          <Button variant="outline" onClick={() => listAll(true)} disabled={visible.length === 0}>
            Listar todas
          </Button>
          <Button variant="outline" onClick={() => listAll(false)} disabled={visible.length === 0}>
            Ocultar todas
          </Button>
          <Button
            variant={sharing ? "accent" : "outline"}
            onClick={() => setSharing((v) => !v)}
            disabled={visible.length === 0}
            title="Montar a lista do que está à venda para postar nos grupos"
          >
            <Share2 /> Compartilhar
          </Button>
        </div>
      </div>

      {sharing && shareItems.length > 0 && (
        <ShareList
          variant="stock"
          holdings={shareItems}
          fxRate={fx}
          onClose={() => setSharing(false)}
        />
      )}

      {loading ? (
        <Panel>Carregando estoque…</Panel>
      ) : holdings.length === 0 ? (
        <Panel>Nenhuma carta em estoque. Adicione compras no Portfolio para vê-las aqui.</Panel>
      ) : visible.length === 0 ? (
        <Panel>Nenhuma carta corresponde à busca.</Panel>
      ) : (
        <div className="sticker sticker-sm overflow-x-auto rounded-[12px] bg-panel">
          <table className="w-full min-w-[720px] text-sm">
            <thead>
              <tr className="font-pixel border-b-2 border-outline bg-raised text-left text-[8px] uppercase text-brand-label">
                <th className="w-8 px-3 py-2">
                  <Checkbox
                    checked={allVisibleSelected}
                    onChange={toggleSelectAllVisible}
                    aria-label="Selecionar todas as visíveis"
                  />
                </th>
                <SortableTh label="Carta" sortKey="name" sort={sort} onSort={onSort} align="left" />
                <SortableTh label="Qtd" sortKey="qty" sort={sort} onSort={onSort} />
                <SortableTh
                  label={`${isBRGame() ? "Floor" : "TCG"} R$`}
                  sortKey="ref"
                  sort={sort}
                  onSort={onSort}
                />
                <SortableTh label="Preço R$ (un.)" sortKey="ask" sort={sort} onSort={onSort} />
                <SortableTh label="Total" sortKey="total" sort={sort} onSort={onSort} />
                <SortableTh label="Na vitrine" sortKey="listed" sort={sort} onSort={onSort} align="center" />
                <SortableTh label="Liga" sortKey="liga" sort={sort} onSort={onSort} align="center" />
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

      <div className="sticky bottom-0 z-30 -mx-4 border-t-[3px] border-outline bg-panel/95 px-4 py-3 backdrop-blur sm:-mx-6 sm:px-6 lg:-mx-8 lg:px-8">
        <div className="flex flex-wrap items-center gap-3">
          {selected.size > 0 ? (
            <>
              <span className="font-pixel text-[9px] uppercase text-brand-label">
                {selected.size} selecionada{selected.size > 1 ? "s" : ""}
              </span>
              {selected.size < 2 && (
                <span className="text-xs text-slate-500">selecione 2+ para combinar</span>
              )}
            </>
          ) : (
            <span className="text-xs text-slate-500">
              {stats.onSale} de {holdings.length} na vitrine · {brl0(stats.value)}
            </span>
          )}
          <div className="ml-auto flex items-center gap-2">
            {selected.size > 0 && (
              <>
                <Button
                  variant="outline"
                  onClick={() => setConfirmMerge(true)}
                  disabled={selected.size < 2}
                >
                  <Combine /> Combinar ({selected.size})
                </Button>
                <Button variant="ghost" onClick={() => setSelected(new Set())}>
                  Limpar
                </Button>
              </>
            )}
            <Button variant="accent" onClick={save} disabled={!dirty || saving}>
              <Save /> {saving ? "Salvando…" : dirty ? "Salvar alterações" : savedAt ? "Salvo" : "Salvar"}
            </Button>
          </div>
        </div>
      </div>

      {confirmMerge && mergePreview && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-outline/60 p-4"
          onClick={() => !merging && setConfirmMerge(false)}
        >
          <div className="win sticker-5 w-full max-w-md" onClick={(e) => e.stopPropagation()}>
            <div className="win-bar">
              <span className="win-title font-pixel !text-[9px] uppercase text-brand-label">
                Combinar {selectedTrades.length} produtos
              </span>
            </div>
            <div className="p-6">
            <h2 className="font-display text-lg font-extrabold text-fg">
              {cleanName(mergePreview.primary.name) || mergePreview.primary.number}
            </h2>
            <p className="mt-1 text-xs text-slate-400">
              Vira uma linha só. Mantém o preço, o "à venda" e a identidade da compra mais antiga; some as
              duplicadas também no Portfolio. <strong className="text-slate-300">Isto é permanente.</strong>
            </p>
            <div className="mt-4 grid grid-cols-3 gap-2 text-center">
              <div className="rounded-[10px] border-2 border-outline bg-page px-2 py-3">
                <div className="font-pixel text-[7px] uppercase text-brand-label">Qtd</div>
                <div className="mt-1 text-lg font-bold tabular-nums text-fg">{mergePreview.qty}</div>
              </div>
              <div className="rounded-[10px] border-2 border-outline bg-page px-2 py-3">
                <div className="font-pixel text-[7px] uppercase text-brand-label">Custo un.</div>
                <div className="mt-1 text-lg font-bold tabular-nums text-fg">{brl(mergePreview.unit)}</div>
              </div>
              <div className="rounded-[10px] border-2 border-outline bg-page px-2 py-3">
                <div className="font-pixel text-[7px] uppercase text-brand-label">Custo total</div>
                <div className="mt-1 text-lg font-bold tabular-nums text-fg">{brl(mergePreview.totalCost)}</div>
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
        </div>
      )}

      {imgEditId && imgEditTrade && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-outline/60 p-4"
          onClick={() => setImgEditId(null)}
        >
          <div
            className="win sticker-5 w-full max-w-md"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="win-bar">
              <span className="win-title font-pixel !text-[9px] uppercase text-brand-label">
                Imagem do produto
              </span>
            </div>
            <div className="p-6">
            <h2 className="font-display text-lg font-extrabold text-fg">
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
                <label className="font-pixel text-[8px] uppercase text-brand-label">
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
  const featured = Boolean(row?.featured);
  const ref = refBRL(t, fx);
  const live = listed && ask > 0;
  // The Liga chip follows the draft flag so a click reacts immediately, but the
  // "needs reimport" state can only come from saved data — it compares the ledger
  // against what was actually sent to the store.
  const ligaFlag = Boolean(row?.ligaListed);
  const sync: LigaSync = !ligaFlag ? "off" : t.ligaListed ? ligaSync(t) : "sync";
  const total = ask * Math.max(t.qty, 1);
  const askUSD = fx > 0 ? ask * fx : 0;
  const [preview, setPreview] = useState<{ x: number; y: number } | null>(null);

  const openPreview = (e: React.SyntheticEvent<HTMLElement>) => {
    const r = e.currentTarget.getBoundingClientRect();
    const x = Math.min(r.right + 12, window.innerWidth - PREVIEW_W - 12);
    const y = Math.min(
      Math.max(r.top + r.height / 2 - PREVIEW_H / 2, 12),
      window.innerHeight - PREVIEW_H - 12,
    );
    setPreview({ x, y });
  };

  return (
    <tr className={`border-b-2 border-outline/15 last:border-0 hover:bg-raised/70 ${selected ? "bg-brand/10" : ""}`}>
      <td className="px-3 py-2 text-center">
        <Checkbox checked={selected} onChange={() => onToggle(t.id)} aria-label={`Selecionar ${t.name}`} />
      </td>
      <td className="px-3 py-2">
        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={() => onEditImage(t.id)}
            onMouseEnter={openPreview}
            onMouseLeave={() => setPreview(null)}
            onFocus={openPreview}
            onBlur={() => setPreview(null)}
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
            <span className="absolute inset-0 flex items-center justify-center bg-outline/55 opacity-0 transition-opacity group-hover:opacity-100">
              <ImagePlus className="h-3.5 w-3.5 text-white" />
            </span>
          </button>
          <div className="min-w-0">
            <div className="flex min-w-0 items-center gap-1">
              <div className="truncate font-medium text-slate-100" title={t.name}>
                {cleanName(t.name) || t.number}
              </div>
              <CopyButton text={cleanName(t.name) || t.number} />
            </div>
            <div className="font-mono text-[10px] text-slate-500">{rowMeta(t)}</div>
          </div>
        </div>
        {preview && (
          <div
            className="pointer-events-none fixed z-40 overflow-hidden rounded-[12px] border-[3px] border-outline bg-panel shadow-2xl shadow-black/60"
            style={{ left: preview.x, top: preview.y, width: PREVIEW_W, height: PREVIEW_H }}
          >
            <CardArt
              set={t.set}
              number={t.number}
              name={t.name}
              productID={productIDFromTcgURL(t.tcgUrl)}
              imageURL={row?.imageURL}
              className="h-full w-full"
            />
          </div>
        )}
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
        <div className="inline-flex items-center gap-1">
          <button
            onClick={() => onPatch(t.id, { listed: !listed })}
            title={live ? "Na vitrine — clique para ocultar" : listed ? "À venda, mas falta preço" : "Oculta — clique para colocar à venda"}
            className={`inline-flex items-center gap-1 rounded-[8px] border-2 border-outline px-2 py-1 text-xs font-bold ${
              live
                ? "bg-emerald-500/10 text-emerald-300 hover:bg-emerald-500/20"
                : listed
                  ? "bg-amber-500/10 text-amber-300 hover:bg-amber-500/20"
                  : "bg-panel text-slate-400 hover:bg-raised"
            }`}
          >
            {live ? <Check className="h-3 w-3" /> : <Store className="h-3 w-3" />}
            {live ? "À venda" : listed ? "Sem preço" : "Oculta"}
          </button>
          {(t.kind === "sealed" || t.kind === "accessory") && (
            <button
              onClick={() => onPatch(t.id, { featured: !featured })}
              title={featured ? "Destaque na home — clique para remover" : "Marcar como destaque na home"}
              aria-label={featured ? `Remover destaque de ${t.name}` : `Destacar ${t.name}`}
              className={`inline-flex items-center rounded-[8px] border-2 border-outline p-1 ${
                featured
                  ? "bg-brand/20 text-brand-label hover:bg-brand/30"
                  : "bg-panel text-slate-500 hover:bg-raised hover:text-slate-300"
              }`}
            >
              <Star className={`h-3 w-3 ${featured ? "fill-current" : ""}`} />
            </button>
          )}
        </div>
      </td>
      <td className="px-3 py-2 text-center">
        <button
          onClick={() => onPatch(t.id, { ligaListed: !ligaFlag })}
          title={
            sync === "stale"
              ? `Cadastrada na Liga com ${t.ligaQty} un. a ${brl(t.ligaPriceBRL ?? 0)} — o estoque mudou desde então, precisa reimportar`
              : sync === "sync"
                ? "Cadastrada na LigaMagic — clique para desmarcar"
                : "Fora da LigaMagic — clique para marcar como cadastrada"
          }
          className={`inline-flex items-center gap-1 rounded-[8px] border-2 border-outline px-2 py-1 text-xs font-bold ${
            sync === "sync"
              ? "bg-emerald-500/10 text-emerald-300 hover:bg-emerald-500/20"
              : sync === "stale"
                ? "bg-amber-500/10 text-amber-300 hover:bg-amber-500/20"
                : "bg-panel text-slate-400 hover:bg-raised"
          }`}
        >
          {sync === "sync" ? (
            <Check className="h-3 w-3" />
          ) : sync === "stale" ? (
            <AlertTriangle className="h-3 w-3" />
          ) : (
            <Tags className="h-3 w-3" />
          )}
          {sync === "sync" ? "Na Liga" : sync === "stale" ? "Reimportar" : "Fora"}
        </button>
      </td>
    </tr>
  );
}

function CopyButton({ text }: { text: string }) {
  const [copied, setCopied] = useState(false);

  const onCopy = async () => {
    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1400);
    } catch {
      setCopied(false);
    }
  };

  return (
    <button
      type="button"
      onClick={onCopy}
      title={copied ? "Copiado" : "Copiar nome"}
      aria-label={`Copiar nome ${text}`}
      className="shrink-0 rounded p-1 text-slate-500 hover:bg-raised hover:text-slate-200"
    >
      {copied ? (
        <Check className="h-3.5 w-3.5 text-emerald-400" />
      ) : (
        <Copy className="h-3.5 w-3.5" />
      )}
    </button>
  );
}

function Panel({ children }: { children: React.ReactNode }) {
  return (
    <EmptyState>
      {children}
    </EmptyState>
  );
}
