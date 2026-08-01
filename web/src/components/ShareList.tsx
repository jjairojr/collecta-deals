import { useEffect, useMemo, useRef, useState } from "react";
import { Check, Copy, Download, ImageDown, Images, Send, X } from "lucide-react";
import { cardImageURL, getGame, productIDFromTcgURL, type TradeView } from "../api";
import { brl0, pct as pctFmt, usd } from "../format";
import { Button } from "./ui/button";
import { Card } from "./ui/card";
import { Input } from "./ui/input";
import { ToggleGroup } from "./ui/toggle-group";

// ShareList turns held products into something you can hand to a buyer: a
// copyable text block and a downloadable image grid.
//
// The "portfolio" variant seeds asking prices as a percentage of the live
// TCGplayer price (80/90/100%) and shows them in US$ or R$ — the currency also
// picks the language of the exported text. The "stock" variant is the seller's
// post for BR groups: prices come from what was already set on the Estoque page
// (R$), and only what is on sale starts marked. Either way, editing here never
// touches the ledger, and you choose whether to reveal what you paid.
type Currency = "USD" | "BRL";
type Variant = "portfolio" | "stock";

export default function ShareList({
  holdings,
  fxRate,
  onClose,
  variant = "portfolio",
}: {
  holdings: TradeView[];
  fxRate: number;
  onClose: () => void;
  variant?: Variant;
}) {
  const L = variant === "stock" ? PT_LABELS : EN_LABELS;
  const [currency, setCurrency] = useState<Currency>(
    variant === "stock" ? "BRL" : "USD",
  );
  const [includePaid, setIncludePaid] = useState(false);
  const [includeAsking, setIncludeAsking] = useState(true);
  const [showProfit, setShowProfit] = useState(true);
  const [askPct, setAskPct] = useState(90);
  const [rows, setRows] = useState<Record<string, RowState>>(() =>
    seedRows(holdings, {
      pct: 90,
      currency: variant === "stock" ? "BRL" : "USD",
      fxRate,
      variant,
      onlyListed: preferListed(holdings, variant),
    }),
  );
  const [previewMode, setPreviewMode] = useState<PreviewMode>("image");
  const [copied, setCopied] = useState(false);
  const [copiedImg, setCopiedImg] = useState(false);
  const [busy, setBusy] = useState(false);
  const [sharing, setSharing] = useState(false);

  // The stock panel is fed by the page's current search, so the list can grow
  // while the panel is open: anything not seeded yet gets its own row instead of
  // showing up unpriced and unselectable.
  useEffect(() => {
    setRows((prev) => {
      const missing = holdings.filter((t) => !prev[t.id]);
      if (missing.length === 0) {
        return prev;
      }
      const ctx: SeedCtx = {
        pct: askPct,
        currency,
        fxRate,
        variant,
        onlyListed: preferListed(holdings, variant),
      };
      const next = { ...prev };
      for (const t of missing) {
        next[t.id] = seedRow(t, ctx);
      }
      return next;
    });
  }, [holdings, askPct, currency, fxRate, variant]);

  const selected = useMemo(
    () => holdings.filter((t) => rows[t.id]?.include),
    [holdings, rows],
  );

  const opts: ShareOpts = { includePaid, includeAsking, variant };
  const totals = useMemo(() => {
    let asking = 0;
    let cost = 0;
    for (const t of selected) {
      asking += (rows[t.id]?.ask ?? 0) * Math.max(t.qty, 1);
      cost += costInCurrency(t, currency, fxRate);
    }
    const profit = asking - cost;
    return { asking, cost, profit, margin: cost > 0 ? (profit / cost) * 100 : 0 };
  }, [selected, rows, currency, fxRate]);
  const total = totals.asking;
  const showProfitCol = includeAsking && showProfit;

  const setRow = (id: string, patch: Partial<RowState>) =>
    setRows((r) => ({ ...r, [id]: { ...r[id], ...patch } }));

  const allSelected = holdings.length > 0 && holdings.every((t) => rows[t.id]?.include);
  const someSelected = holdings.some((t) => rows[t.id]?.include);

  const toggleAll = () => {
    const value = !allSelected;
    setRows((prev) => {
      const next: Record<string, RowState> = {};
      for (const t of holdings) {
        next[t.id] = { ...prev[t.id], include: value };
      }
      return next;
    });
  };

  // Editing a single card's % reprices only that card off its live TCG price;
  // typing a price directly keeps its % label in sync.
  const setRowPct = (t: TradeView, p: number) =>
    setRow(t.id, { pct: p, ask: askFromMarket(t, p, currency, fxRate) });

  const setRowAsk = (t: TradeView, ask: number) =>
    setRow(t.id, { ask, pct: pctFromMarket(t, ask, currency, fxRate) });

  // Re-pricing at a new % overwrites asking values (including manual edits) from
  // the live TCG price, while preserving which cards are selected.
  const applyPct = (p: number) => {
    setAskPct(p);
    setRows((prev) => {
      const next: Record<string, RowState> = {};
      for (const t of holdings) {
        next[t.id] = {
          include: prev[t.id]?.include ?? true,
          ask: askFromMarket(t, p, currency, fxRate),
          pct: p,
        };
      }
      return next;
    });
  };

  // Switching currency re-seeds every asking price from the live TCG price at
  // its current %, converted into the new currency (manual edits are re-based).
  const applyCurrency = (c: Currency) => {
    if (c === currency) {
      return;
    }
    setCurrency(c);
    setRows((prev) => {
      const next: Record<string, RowState> = {};
      for (const t of holdings) {
        const p = prev[t.id]?.pct ?? askPct;
        next[t.id] = {
          include: prev[t.id]?.include ?? true,
          ask: askFromMarket(t, p, c, fxRate),
          pct: p,
        };
      }
      return next;
    });
  };

  const copyText = async () => {
    const text = buildText(selected, rows, opts, currency);
    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1800);
    } catch {
      setCopied(false);
    }
  };

  const downloadImage = async () => {
    setBusy(true);
    try {
      const blobs = await buildImages(selected, rows, opts, currency);
      blobs.forEach((blob, i) => {
        const url = URL.createObjectURL(blob);
        const a = document.createElement("a");
        a.href = url;
        a.download = pageFilename(i, blobs.length);
        a.click();
        URL.revokeObjectURL(url);
      });
    } finally {
      setBusy(false);
    }
  };

  // copyImage puts the first page on the clipboard so it can be pasted straight
  // into a chat on desktop (where the native share sheet is usually
  // unavailable). Clipboard holds a single image, so multi-page lists fall back
  // to Download / Share for the remaining pages.
  const copyImage = async () => {
    setBusy(true);
    try {
      const blobs = await buildImages(selected, rows, opts, currency);
      if (blobs.length === 0 || typeof ClipboardItem === "undefined") {
        return;
      }
      await navigator.clipboard.write([new ClipboardItem({ "image/png": blobs[0] })]);
      setCopiedImg(true);
      window.setTimeout(() => setCopiedImg(false), 1800);
    } catch {
      setCopiedImg(false);
    } finally {
      setBusy(false);
    }
  };

  // shareList hands the list to the OS share sheet — on a phone this lands the
  // image + text straight into WhatsApp (or any app). It shares the PNG when the
  // platform allows files, else the text alone, and falls back to a wa.me link
  // (WhatsApp Web) on desktops without the Web Share API. The recipient and the
  // actual send stay in the user's hands inside WhatsApp.
  const shareList = async () => {
    const text = buildText(selected, rows, opts, currency);
    setSharing(true);
    try {
      const blobs = await buildImages(selected, rows, opts, currency);
      const files = blobs.map(
        (blob, i) => new File([blob], pageFilename(i, blobs.length), { type: "image/png" }),
      );
      if (files.length > 0 && navigator.canShare?.({ files })) {
        await navigator.share({ files, text });
        return;
      }
      if (navigator.share) {
        await navigator.share({ text });
        return;
      }
      window.open(`https://wa.me/?text=${encodeURIComponent(text)}`, "_blank", "noopener");
    } catch (err) {
      if (err instanceof Error && err.name === "AbortError") {
        return;
      }
      window.open(`https://wa.me/?text=${encodeURIComponent(text)}`, "_blank", "noopener");
    } finally {
      setSharing(false);
    }
  };

  const preview = buildText(selected, rows, opts, currency);

  return (
    <Card className="space-y-4 p-4">
      <div className="flex items-start justify-between gap-3">
        <div>
          <h3 className="font-display text-sm font-bold text-fg">{L.title}</h3>
          <p className="mt-0.5 text-xs text-slate-400">{L.hint(PAGE_SIZE)}</p>
        </div>
        <button
          onClick={onClose}
          className="rounded-[8px] border-2 border-outline bg-panel p-1.5 text-slate-400 hover:bg-raised hover:text-slate-100"
          title={L.close}
        >
          <X className="h-3.5 w-3.5" />
        </button>
      </div>

      <div className="flex flex-wrap items-center gap-2">
        {variant === "portfolio" && (
          <>
            <ToggleGroup
              value={currency}
              onChange={(v) => applyCurrency(v as Currency)}
              options={currencyOptions}
            />
            <span className="mx-1 h-5 w-px bg-slate-800" />
          </>
        )}
        <Toggle on={includeAsking} onClick={() => setIncludeAsking((v) => !v)}>
          {L.asking}
        </Toggle>
        <Toggle on={includePaid} onClick={() => setIncludePaid((v) => !v)}>
          {L.paid}
        </Toggle>
        {includeAsking && (
          <Toggle on={showProfit} onClick={() => setShowProfit((v) => !v)}>
            {L.profit}
          </Toggle>
        )}
        {includeAsking && (
          <div className="flex items-center gap-2">
            <span className="font-pixel text-[9px] uppercase text-brand-label">
              {L.pctOf}
            </span>
            <ToggleGroup
              value={String(askPct)}
              onChange={(v) => applyPct(Number(v))}
              options={askPctOptions}
            />
          </div>
        )}
        <span className="ml-auto text-xs text-slate-500">
          {L.selected(selected.length, holdings.length)}
        </span>
      </div>

      <div
        className={`sticker sticker-sm overflow-x-auto rounded-[12px] bg-panel ${
          holdings.length > LONG_LIST ? "max-h-[26rem] overflow-y-auto" : ""
        }`}
      >
        <table className="w-full min-w-[560px] text-sm">
          <thead>
            <tr className="font-pixel sticky top-0 z-10 border-b-2 border-outline bg-raised text-left text-[8px] uppercase text-brand-label [&>th]:bg-raised">
              <th className="px-3 py-2 font-bold">
                <input
                  type="checkbox"
                  aria-label={allSelected ? L.unmarkAll : L.markAll}
                  title={allSelected ? L.unmarkAll : L.markAll}
                  checked={allSelected}
                  ref={(el) => {
                    if (el) el.indeterminate = !allSelected && someSelected;
                  }}
                  onChange={toggleAll}
                  className="h-4 w-4 accent-brand"
                />
              </th>
              <th className="px-3 py-2 font-bold">{L.colCard}</th>
              <th className="px-3 py-2 text-right font-bold">{L.colRef}</th>
              {includePaid && <th className="px-3 py-2 text-right font-bold">{L.colPaid}</th>}
              {includeAsking && <th className="px-3 py-2 text-right font-bold">%</th>}
              {includeAsking && (
                <th className="px-3 py-2 text-right font-bold">{L.colAsk(currency)}</th>
              )}
              {showProfitCol && <th className="px-3 py-2 text-right font-bold">{L.profit}</th>}
            </tr>
          </thead>
          <tbody>
            {holdings.map((t) => {
              const row = rows[t.id];
              const paidUnit = t.costBRL / Math.max(t.qty, 1);
              const tcgInCurrency = marketInCurrency(t, currency, fxRate);
              return (
                <tr key={t.id} className="border-b-2 border-outline/15 last:border-0 hover:bg-raised/70">
                  <td className="px-3 py-2">
                    <input
                      type="checkbox"
                      checked={row?.include ?? false}
                      onChange={(e) => setRow(t.id, { include: e.target.checked })}
                      className="h-4 w-4 accent-brand"
                    />
                  </td>
                  <td className="px-3 py-2">
                    <div className="min-w-0">
                      <div className="truncate font-medium text-slate-100" title={t.name}>
                        {t.name}
                      </div>
                      <div className="font-mono text-[10px] text-slate-500">
                        {t.number}
                        {t.condition ? ` · ${t.condition}` : ""}
                        {t.qty > 1 ? ` · ×${t.qty}` : ""}
                      </div>
                    </div>
                  </td>
                  <td className="px-3 py-2 text-right tabular-nums text-slate-400">
                    {tcgInCurrency > 0 ? money(tcgInCurrency, currency) : "—"}
                  </td>
                  {includePaid && (
                    <td className="px-3 py-2 text-right tabular-nums text-slate-400">
                      {brl0(paidUnit)}
                    </td>
                  )}
                  {includeAsking && (
                    <td className="px-3 py-2 text-right">
                      <div className="flex items-center justify-end gap-1">
                        <Input
                          type="number"
                          value={String(row?.pct ?? 0)}
                          onChange={(e) => setRowPct(t, Number(e.target.value) || 0)}
                          disabled={tcgInCurrency <= 0}
                          className="w-16 text-right"
                        />
                        <span className="text-xs text-slate-500">%</span>
                      </div>
                    </td>
                  )}
                  {includeAsking && (
                    <td className="px-3 py-2 text-right">
                      <div className="flex items-center justify-end gap-1">
                        <span className="text-xs text-slate-500">
                          {currency === "USD" ? "$" : "R$"}
                        </span>
                        <Input
                          type="number"
                          value={String(row?.ask ?? 0)}
                          onChange={(e) => setRowAsk(t, Number(e.target.value) || 0)}
                          className="w-24 text-right"
                        />
                      </div>
                    </td>
                  )}
                  {showProfitCol && <ProfitCell t={t} row={row} currency={currency} fxRate={fxRate} />}
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>

      {includeAsking && selected.length > 0 && (
        <div className="flex flex-wrap items-center justify-end gap-x-6 gap-y-2 text-sm">
          <SummaryStat label={L.totalAsk} value={money(totals.asking, currency)} tone="accent" />
          {showProfit && (
            <SummaryStat label={L.yourCost} value={money(totals.cost, currency)} tone="muted" />
          )}
          {showProfit && (
            <SummaryStat
              label={L.yourProfit}
              value={`${money(totals.profit, currency)} · ${pctFmt(totals.margin)}`}
              tone={totals.profit >= 0 ? "gain" : "loss"}
            />
          )}
        </div>
      )}

      <div>
        <div className="mb-1.5 flex items-center justify-between">
          <span className="font-pixel text-[9px] uppercase text-brand-label">
            {L.preview}
          </span>
          <ToggleGroup
            value={previewMode}
            onChange={(v) => setPreviewMode(v as PreviewMode)}
            options={[
              { value: "image", label: L.tabImage },
              { value: "text", label: L.tabText },
            ]}
          />
        </div>
        {previewMode === "text" ? (
          <pre className="max-h-60 overflow-auto whitespace-pre-wrap rounded-[14px] border-[3px] border-outline bg-page p-3 text-xs text-slate-300">
            {preview || L.empty}
          </pre>
        ) : (
          <ImagePreview
            selected={selected}
            rows={rows}
            opts={opts}
            currency={currency}
            total={total}
            empty={L.empty}
            pageLabel={L.page}
          />
        )}
      </div>

      <div className="flex flex-wrap justify-end gap-2">
        <Button variant="outline" onClick={copyText} disabled={selected.length === 0}>
          {copied ? <Check /> : <Copy />} {copied ? L.copied : L.copyText}
        </Button>
        <Button variant="outline" onClick={copyImage} disabled={selected.length === 0 || busy}>
          {copiedImg ? <Check /> : <Images />} {copiedImg ? L.copied : L.copyImage}
        </Button>
        <Button variant="outline" onClick={downloadImage} disabled={selected.length === 0 || busy}>
          {busy ? <ImageDown /> : <Download />} {busy ? L.building : L.download}
        </Button>
        <Button variant="accent" onClick={shareList} disabled={selected.length === 0 || sharing}>
          <Send /> {sharing ? L.sharingNow : L.share}
        </Button>
      </div>
    </Card>
  );
}

const currencyOptions = [
  { value: "USD", label: "US$" },
  { value: "BRL", label: "R$" },
];

const askPctOptions = [
  { value: "80", label: "80%" },
  { value: "85", label: "85%" },
  { value: "90", label: "90%" },
  { value: "100", label: "100%" },
];

// Past LONG_LIST products the picker gets its own scroll area, so the preview
// and the share buttons stay reachable instead of sitting below a wall of rows.
const LONG_LIST = 25;

type PreviewMode = "image" | "text";

interface RowState {
  include: boolean;
  ask: number;
  pct: number;
}

interface ShareOpts {
  includePaid: boolean;
  includeAsking: boolean;
  variant: Variant;
}

// The portfolio panel talks to an international buyer in English; the stock
// panel is the seller's own post for BR groups, so it speaks Portuguese and
// never mentions TCGplayer — its reference is whatever the Estoque page shows.
interface Labels {
  title: string;
  hint: (pageSize: number) => string;
  asking: string;
  paid: string;
  profit: string;
  pctOf: string;
  selected: (n: number, total: number) => string;
  colCard: string;
  colRef: string;
  colPaid: string;
  colAsk: (currency: Currency) => string;
  totalAsk: string;
  yourCost: string;
  yourProfit: string;
  preview: string;
  tabImage: string;
  tabText: string;
  page: (index: number, count: number) => string;
  empty: string;
  copyText: string;
  copied: string;
  copyImage: string;
  download: string;
  building: string;
  share: string;
  sharingNow: string;
  close: string;
  markAll: string;
  unmarkAll: string;
}

const EN_LABELS: Labels = {
  title: "Share list",
  hint: (pageSize) =>
    `Pick cards, then Share to WhatsApp. Asking prices are a % of the live TCGplayer price, in US$ or R$ — edit any of them. Long lists split into pages of ${pageSize} so each image stays sharp.`,
  asking: "Asking price",
  paid: "Price paid",
  profit: "My profit",
  pctOf: "% of TCG",
  selected: (n, total) => `${n} of ${total} selected`,
  colCard: "Card",
  colRef: "TCG",
  colPaid: "Paid /ea",
  colAsk: (currency) => `Asking ${currency === "USD" ? "US$" : "R$"} /ea`,
  totalAsk: "Asking total",
  yourCost: "Your cost",
  yourProfit: "Your profit",
  preview: "Preview",
  tabImage: "Image",
  tabText: "Text",
  page: (index, count) => `Page ${index} / ${count}`,
  empty: "Select at least one card to build your list.",
  copyText: "Copy text",
  copied: "Copied!",
  copyImage: "Copy image",
  download: "Download",
  building: "Building…",
  share: "Share",
  sharingNow: "Sharing…",
  close: "Close",
  markAll: "Mark all",
  unmarkAll: "Unmark all",
};

const PT_LABELS: Labels = {
  title: "Lista para grupos",
  hint: (pageSize) =>
    `Marque o que entra no post e mande para o WhatsApp. Os preços já vêm do estoque — pode editar aqui que nada disso altera o que está salvo. Listas longas viram várias imagens de ${pageSize} produtos.`,
  asking: "Preço",
  paid: "Preço pago",
  profit: "Meu lucro",
  pctOf: "% da ref.",
  selected: (n, total) => `${n} de ${total} marcados`,
  colCard: "Produto",
  colRef: "Ref.",
  colPaid: "Pago /un",
  colAsk: () => "Preço R$ /un",
  totalAsk: "Total pedido",
  yourCost: "Meu custo",
  yourProfit: "Meu lucro",
  preview: "Prévia",
  tabImage: "Imagem",
  tabText: "Texto",
  page: (index, count) => `Imagem ${index} / ${count}`,
  empty: "Marque pelo menos um produto para montar a lista.",
  copyText: "Copiar texto",
  copied: "Copiado!",
  copyImage: "Copiar imagem",
  download: "Baixar",
  building: "Gerando…",
  share: "Compartilhar",
  sharingNow: "Enviando…",
  close: "Fechar",
  markAll: "Marcar todos",
  unmarkAll: "Desmarcar todos",
};

// money formats an amount in the chosen currency: USD with cents, BRL rounded
// (pt-BR grouping) — matching how the Quotes feature shows reais.
function money(value: number, currency: Currency): string {
  return currency === "USD" ? usd(value) : brl0(value);
}

// marketInCurrency converts a product's market reference into the display
// currency: the live TCG price when there is one, else the manual estimate that
// sealed products and accessories carry (already in BRL). fxRate is USD per BRL
// (e.g. 0.195), so BRL = USD / fxRate and USD = BRL * fxRate.
function marketInCurrency(t: TradeView, currency: Currency, fxRate: number): number {
  if (t.marketKnown && t.marketUSD > 0) {
    if (currency === "USD") {
      return t.marketUSD;
    }
    return fxRate > 0 ? t.marketUSD / fxRate : t.marketUSD;
  }
  const manual = t.manualBRL ?? 0;
  if (manual <= 0) {
    return 0;
  }
  return currency === "BRL" ? manual : manual * fxRate;
}

// costInCurrency converts a card's total cost basis (kept in BRL) into the
// display currency, so profit can be compared against the asking price.
// fxRate is USD per BRL, so USD = BRL * fxRate.
function costInCurrency(t: TradeView, currency: Currency, fxRate: number): number {
  return currency === "BRL" ? t.costBRL : t.costBRL * fxRate;
}

// askFromMarket returns the per-unit asking price at pct of the live TCG price,
// in the display currency. Cards with no known US price seed to 0 to fill in.
function askFromMarket(t: TradeView, pct: number, currency: Currency, fxRate: number): number {
  const base = marketInCurrency(t, currency, fxRate);
  if (base <= 0) {
    return 0;
  }
  return currency === "USD"
    ? Math.round(base * (pct / 100) * 100) / 100
    : Math.round(base * (pct / 100));
}

// pctFromMarket derives the % of the live TCG price a manual asking price
// represents, so the per-card % input stays in sync with price edits.
function pctFromMarket(t: TradeView, ask: number, currency: Currency, fxRate: number): number {
  const base = marketInCurrency(t, currency, fxRate);
  if (base <= 0) {
    return 0;
  }
  return Math.round((ask / base) * 100);
}

interface SeedCtx {
  pct: number;
  currency: Currency;
  fxRate: number;
  variant: Variant;
  onlyListed: boolean;
}

// preferListed decides how the stock panel opens: if anything is already on
// sale, only those start marked — that is the post you meant to write. With
// nothing on sale yet, marking everything beats opening an empty panel.
function preferListed(holdings: TradeView[], variant: Variant): boolean {
  return (
    variant === "stock" &&
    holdings.some((t) => t.listed && (t.askBRL ?? 0) > 0)
  );
}

// seedRow gives a product its starting state in the panel. The stock variant
// takes the price already set on the Estoque page; the portfolio variant prices
// off the live market at the chosen %.
function seedRow(t: TradeView, ctx: SeedCtx): RowState {
  if (ctx.variant === "stock") {
    const ask = t.askBRL ?? 0;
    return {
      include: ctx.onlyListed ? Boolean(t.listed) && ask > 0 : true,
      ask,
      pct: pctFromMarket(t, ask, ctx.currency, ctx.fxRate),
    };
  }
  return {
    include: true,
    ask: askFromMarket(t, ctx.pct, ctx.currency, ctx.fxRate),
    pct: ctx.pct,
  };
}

function seedRows(holdings: TradeView[], ctx: SeedCtx): Record<string, RowState> {
  const out: Record<string, RowState> = {};
  for (const t of holdings) {
    out[t.id] = seedRow(t, ctx);
  }
  return out;
}

// eaSuffix marks a price as per-unit, and only where it matters: a single copy
// has no "each" to spell out.
function eaSuffix(qty: number, currency: Currency): string {
  if (qty <= 1) {
    return "";
  }
  return currency === "BRL" ? " cada" : " ea";
}

// condLabel is the condition shown next to a product. Sealed boxes and
// accessories have none — printing "NM" on a booster box would be nonsense.
function condLabel(t: TradeView): string {
  if (t.kind) {
    return "";
  }
  return t.condition || "NM";
}

function buildText(
  selected: TradeView[],
  rows: Record<string, RowState>,
  opts: ShareOpts,
  currency: Currency,
): string {
  if (selected.length === 0) {
    return "";
  }
  const pt = currency === "BRL";
  const noun = itemNoun(selected.length, opts.variant, pt);
  const lines: string[] = [
    `${gameLabel()} — ${pt ? "à venda" : "for sale"} (${selected.length} ${noun})`,
    "",
  ];
  let total = 0;
  for (const t of selected) {
    const row = rows[t.id];
    lines.push(t.number ? `${t.name} (${t.number})` : t.name);
    const meta = [condLabel(t), `${t.qty}x`].filter(Boolean);
    if (opts.includeAsking) {
      const lineTotal = row.ask * Math.max(t.qty, 1);
      total += lineTotal;
      if (t.qty > 1) {
        meta.push(
          `${money(row.ask, currency)}${eaSuffix(t.qty, currency)} = ${money(lineTotal, currency)}`,
        );
      } else {
        meta.push(money(row.ask, currency));
      }
    }
    lines.push(meta.join(" · "));
    if (opts.includePaid) {
      const paid = brl0(t.costBRL / Math.max(t.qty, 1));
      lines.push(`${pt ? "pago" : "paid"}: ${paid}${eaSuffix(t.qty, currency)}`);
    }
    lines.push("");
  }
  if (opts.includeAsking) {
    lines.push(`💰 Total: ${money(total, currency)}`);
  }
  return lines.join("\n").trimEnd();
}

// itemNoun counts the list: the stock panel mixes singles, sealed and
// accessories, so it says "items" where the portfolio panel says "cards".
function itemNoun(count: number, variant: Variant, pt: boolean): string {
  if (variant === "stock") {
    if (pt) {
      return count === 1 ? "item" : "itens";
    }
    return count === 1 ? "item" : "items";
  }
  if (pt) {
    return count === 1 ? "carta" : "cartas";
  }
  return count === 1 ? "card" : "cards";
}

function gameLabel(): string {
  const g = getGame();
  if (g === "onepiece") return "One Piece";
  if (g === "pokemon") return "Pokémon";
  return g.charAt(0).toUpperCase() + g.slice(1);
}

// ---- Image grid (inline preview + PNG download share one paint/layout) ----

const IMG_W = 240;
const IMG_H = Math.round(IMG_W / 0.716);
const GAP = 18;
const PAD = 28;
const CAPTION_H = 66;
const TITLE_H = 54;
const FOOTER_H = 52;
const SCALE = 2;
// Cap each exported image at PAGE_SIZE cards (4 cols × 5 rows) so a long list
// splits into several short, sharp PNGs instead of one very tall image that
// WhatsApp compresses into a blur.
const PAGE_SIZE = 20;

const C_BG = "#141416";
const C_TITLE = "#ffffff";
const C_CELL = "#1f1f22";
const C_NAME = "#e6e6ec";
const C_META = "#9a9aa2";
const C_ASK = "#fdc4e5";
const C_PAID = "#6f6f77";
const C_MUTED = "#6f6f77";
const C_OUTLINE = "#0b0b0c";

interface Layout {
  cols: number;
  rowsN: number;
  width: number;
  height: number;
  gridTop: number;
  gridH: number;
}

// layout sizes one page. cols is fixed across pages (from the full list) so
// every page shares a width; showFooter reserves the total row only where it's
// drawn (the last page).
function layout(count: number, opts: ShareOpts, cols: number, showFooter: boolean): Layout {
  const rowsN = Math.max(1, Math.ceil(count / cols));
  const width = PAD * 2 + cols * IMG_W + (cols - 1) * GAP;
  const gridH = rowsN * (IMG_H + CAPTION_H) + Math.max(rowsN - 1, 0) * GAP;
  const footerH = opts.includeAsking && showFooter ? FOOTER_H : 0;
  const height = PAD * 2 + TITLE_H + gridH + footerH;
  return { cols, rowsN, width, height, gridTop: PAD + TITLE_H, gridH };
}

// gridCols is the shared column count for a list of `count` cards.
function gridCols(count: number): number {
  return Math.min(4, Math.max(1, count));
}

function paginate<T>(arr: T[], size: number): T[][] {
  const out: T[][] = [];
  for (let i = 0; i < arr.length; i += size) {
    out.push(arr.slice(i, i + size));
  }
  return out;
}

// PageInfo positions one page within the exported set: its index, the total
// page count, the full card count (for the header), and the shared column grid.
interface PageInfo {
  index: number;
  count: number;
  totalCards: number;
  cols: number;
}

// imgSrc resolves a product's art. A hand-picked image URL (the only art sealed
// products and accessories have) wins; otherwise the exact TCGplayer product
// image, routed through the same-origin proxy so the canvas isn't tainted,
// beats the number-keyed Liga lookup — the latter can't tell variant prints
// apart. A remote URL without CORS simply fails to load and the cell falls back
// to text, so it can never taint the canvas either.
function imgSrc(t: TradeView): string | null {
  if (t.imageURL) {
    return t.imageURL;
  }
  const pid = productIDFromTcgURL(t.tcgUrl);
  if (pid) {
    return cardImageURL(t.set, t.number, pid);
  }
  return t.set ? cardImageURL(t.set, t.number) : null;
}

function imgKey(t: TradeView): string {
  return `${t.imageURL ?? ""}|${t.set}|${t.number}|${productIDFromTcgURL(t.tcgUrl) ?? ""}`;
}

// paint draws one page: its cards, a header with a page indicator, and the
// grand total only on the last page. grandTotal is the sum across every page.
function paint(
  ctx: CanvasRenderingContext2D,
  cards: TradeView[],
  rows: Record<string, RowState>,
  opts: ShareOpts,
  currency: Currency,
  grandTotal: number,
  cache: Map<string, HTMLImageElement | null>,
  page: PageInfo,
) {
  const showFooter = opts.includeAsking && page.index === page.count - 1;
  const lay = layout(cards.length, opts, page.cols, showFooter);

  ctx.fillStyle = C_BG;
  ctx.fillRect(0, 0, lay.width, lay.height);

  ctx.fillStyle = C_TITLE;
  ctx.font = "600 24px ui-sans-serif, system-ui, sans-serif";
  ctx.textBaseline = "top";
  ctx.textAlign = "left";
  ctx.fillText(`${gameLabel()} — ${currency === "BRL" ? "à venda" : "for sale"}`, PAD, PAD);

  ctx.fillStyle = C_MUTED;
  ctx.font = "500 13px ui-sans-serif, system-ui, sans-serif";
  ctx.textAlign = "right";
  const noun = itemNoun(page.totalCards, opts.variant, currency === "BRL");
  const header =
    page.count > 1
      ? `${page.totalCards} ${noun} · ${page.index + 1}/${page.count}`
      : `${page.totalCards} ${noun}`;
  ctx.fillText(header, lay.width - PAD, PAD + 8);
  ctx.textAlign = "left";

  cards.forEach((t, i) => {
    const c = i % page.cols;
    const r = Math.floor(i / page.cols);
    const x = PAD + c * (IMG_W + GAP);
    const y = lay.gridTop + r * (IMG_H + CAPTION_H + GAP);
    drawCell(ctx, t, rows[t.id], opts, currency, cache.get(imgKey(t)) ?? null, x, y);
  });

  if (showFooter) {
    const fy = lay.gridTop + lay.gridH + 16;
    ctx.fillStyle = C_META;
    ctx.font = "500 14px ui-sans-serif, system-ui, sans-serif";
    ctx.textAlign = "left";
    ctx.fillText("Total", PAD, fy);
    ctx.fillStyle = C_ASK;
    ctx.font = "700 20px ui-sans-serif, system-ui, sans-serif";
    ctx.textAlign = "right";
    ctx.fillText(money(grandTotal, currency), lay.width - PAD, fy - 3);
    ctx.textAlign = "left";
  }
}

function drawCell(
  ctx: CanvasRenderingContext2D,
  t: TradeView,
  row: RowState,
  opts: ShareOpts,
  currency: Currency,
  img: HTMLImageElement | null,
  x: number,
  y: number,
) {
  ctx.fillStyle = C_CELL;
  ctx.fillRect(x, y, IMG_W, IMG_H);
  if (img) {
    ctx.drawImage(img, x, y, IMG_W, IMG_H);
  } else {
    ctx.fillStyle = C_MUTED;
    ctx.font = "12px ui-monospace, monospace";
    ctx.textAlign = "center";
    ctx.fillText(truncate(ctx, t.number || t.name, IMG_W - 16), x + IMG_W / 2, y + IMG_H / 2 - 6);
    ctx.textAlign = "left";
  }
  ctx.strokeStyle = C_OUTLINE;
  ctx.lineWidth = 2;
  ctx.strokeRect(x + 1, y + 1, IMG_W - 2, IMG_H - 2);

  const cy = y + IMG_H + 8;
  ctx.textAlign = "left";
  ctx.fillStyle = C_NAME;
  ctx.font = "600 14px ui-sans-serif, system-ui, sans-serif";
  ctx.fillText(truncate(ctx, t.name, IMG_W), x, cy);

  ctx.fillStyle = C_META;
  ctx.font = "12px ui-sans-serif, system-ui, sans-serif";
  const meta = [condLabel(t), `${t.qty}x`].filter(Boolean).join(" · ");
  ctx.fillText(meta, x, cy + 20);

  if (opts.includeAsking && row) {
    ctx.fillStyle = C_ASK;
    ctx.font = "700 16px ui-sans-serif, system-ui, sans-serif";
    ctx.fillText(`${money(row.ask, currency)}${eaSuffix(t.qty, currency)}`, x, cy + 38);
  }
  if (opts.includePaid) {
    ctx.fillStyle = C_PAID;
    ctx.font = "11px ui-sans-serif, system-ui, sans-serif";
    const px = opts.includeAsking ? x + 130 : x;
    ctx.fillText(`paid ${brl0(t.costBRL / Math.max(t.qty, 1))}`, px, cy + 40);
  }
}

function truncate(ctx: CanvasRenderingContext2D, text: string, maxW: number): string {
  if (ctx.measureText(text).width <= maxW) {
    return text;
  }
  let s = text;
  while (s.length > 1 && ctx.measureText(s + "…").width > maxW) {
    s = s.slice(0, -1);
  }
  return s + "…";
}

function loadImage(t: TradeView): Promise<HTMLImageElement | null> {
  return new Promise((resolve) => {
    const src = imgSrc(t);
    if (!src) {
      resolve(null);
      return;
    }
    const img = new Image();
    img.crossOrigin = "anonymous";
    img.onload = () => resolve(img);
    img.onerror = () => resolve(null);
    img.src = src;
  });
}

// ensureImages loads any card art not yet in the shared cache, so both the live
// preview and the PNG download draw from the same set of decoded images.
async function ensureImages(
  selected: TradeView[],
  cache: Map<string, HTMLImageElement | null>,
): Promise<void> {
  const missing = selected.filter((t) => !cache.has(imgKey(t)));
  if (missing.length === 0) {
    return;
  }
  await Promise.all(
    missing.map(async (t) => {
      cache.set(imgKey(t), await loadImage(t));
    }),
  );
}

function grandTotalOf(
  selected: TradeView[],
  rows: Record<string, RowState>,
): number {
  return selected.reduce((sum, t) => sum + (rows[t.id]?.ask ?? 0) * Math.max(t.qty, 1), 0);
}

// buildImages renders the list into one PNG per PAGE_SIZE-card page, so a long
// list ships as several short, sharp images. All card art is loaded once and
// shared across pages; the grand total prints on the final page only.
async function buildImages(
  selected: TradeView[],
  rows: Record<string, RowState>,
  opts: ShareOpts,
  currency: Currency,
): Promise<Blob[]> {
  if (selected.length === 0) {
    return [];
  }
  const cache = new Map<string, HTMLImageElement | null>();
  await ensureImages(selected, cache);
  const cols = gridCols(selected.length);
  const pages = paginate(selected, PAGE_SIZE);
  const grand = grandTotalOf(selected, rows);
  const blobs: Blob[] = [];
  for (let p = 0; p < pages.length; p++) {
    const cards = pages[p];
    const page: PageInfo = { index: p, count: pages.length, totalCards: selected.length, cols };
    const showFooter = opts.includeAsking && p === pages.length - 1;
    const lay = layout(cards.length, opts, cols, showFooter);
    const canvas = document.createElement("canvas");
    canvas.width = lay.width * SCALE;
    canvas.height = lay.height * SCALE;
    const ctx = canvas.getContext("2d");
    if (!ctx) {
      continue;
    }
    ctx.scale(SCALE, SCALE);
    paint(ctx, cards, rows, opts, currency, grand, cache, page);
    const blob = await new Promise<Blob | null>((resolve) =>
      canvas.toBlob(resolve, "image/png"),
    );
    if (blob) {
      blobs.push(blob);
    }
  }
  return blobs;
}

// pageFilename names page i of n: bare when there's a single page, numbered when
// the list spans several.
function pageFilename(index: number, count: number): string {
  const base = `${getGame()}-for-sale`;
  return count > 1 ? `${base}-${index + 1}.png` : `${base}.png`;
}

// ImagePreview renders the same grid inline so the list can be reviewed before
// download. Card art is cached across redraws, so price/currency edits repaint
// synchronously; a load bump forces one repaint once art arrives.
function ImagePreview({
  selected,
  rows,
  opts,
  currency,
  total,
  empty,
  pageLabel,
}: {
  selected: TradeView[];
  rows: Record<string, RowState>;
  opts: ShareOpts;
  currency: Currency;
  total: number;
  empty: string;
  pageLabel: (index: number, count: number) => string;
}) {
  const canvasRefs = useRef<(HTMLCanvasElement | null)[]>([]);
  const cacheRef = useRef<Map<string, HTMLImageElement | null>>(new Map());
  const [imgVersion, setImgVersion] = useState(0);

  const ids = selected.map((t) => t.id).join(",");
  const cols = gridCols(selected.length);
  const pages = paginate(selected, PAGE_SIZE);

  useEffect(() => {
    let cancelled = false;
    void ensureImages(selected, cacheRef.current).then(() => {
      if (!cancelled) {
        setImgVersion((v) => v + 1);
      }
    });
    return () => {
      cancelled = true;
    };
  }, [ids, selected]);

  useEffect(() => {
    pages.forEach((cards, p) => {
      const canvas = canvasRefs.current[p];
      if (!canvas) {
        return;
      }
      const page: PageInfo = { index: p, count: pages.length, totalCards: selected.length, cols };
      const showFooter = opts.includeAsking && p === pages.length - 1;
      const lay = layout(cards.length, opts, cols, showFooter);
      canvas.width = lay.width * SCALE;
      canvas.height = lay.height * SCALE;
      const ctx = canvas.getContext("2d");
      if (!ctx) {
        return;
      }
      ctx.scale(SCALE, SCALE);
      paint(ctx, cards, rows, opts, currency, total, cacheRef.current, page);
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [ids, rows, opts, currency, total, imgVersion, cols]);

  if (selected.length === 0) {
    return (
      <div className="rounded-[14px] border-[3px] border-outline bg-page p-6 text-center text-xs text-slate-500">
        {empty}
      </div>
    );
  }

  return (
    <div className="max-h-[36rem] space-y-4 overflow-auto rounded-[14px] border-[3px] border-outline bg-page p-3">
      {pages.map((_, p) => (
        <div key={p} className="space-y-1.5">
          {pages.length > 1 && (
            <div className="text-[10px] font-medium uppercase tracking-wide text-slate-500">
              {pageLabel(p + 1, pages.length)}
            </div>
          )}
          <canvas
            ref={(el) => {
              canvasRefs.current[p] = el;
            }}
            className="mx-auto h-auto max-w-full"
          />
        </div>
      ))}
    </div>
  );
}

// ProfitCell shows the per-line profit (asking proceeds minus cost basis) in the
// display currency, coloured by sign, with the margin underneath. It stays in
// the panel only — cost basis is never written into the buyer-facing export.
function ProfitCell({
  t,
  row,
  currency,
  fxRate,
}: {
  t: TradeView;
  row: RowState | undefined;
  currency: Currency;
  fxRate: number;
}) {
  const proceeds = (row?.ask ?? 0) * Math.max(t.qty, 1);
  const cost = costInCurrency(t, currency, fxRate);
  const profit = proceeds - cost;
  const margin = cost > 0 ? (profit / cost) * 100 : 0;
  const tone = profit >= 0 ? "text-emerald-400" : "text-rose-400";
  return (
    <td className="px-3 py-2 text-right">
      <div className={`font-medium tabular-nums ${tone}`}>{money(profit, currency)}</div>
      <div className="text-[10px] tabular-nums text-slate-500">{pctFmt(margin)}</div>
    </td>
  );
}

function SummaryStat({
  label,
  value,
  tone,
}: {
  label: string;
  value: string;
  tone: "accent" | "muted" | "gain" | "loss";
}) {
  const color =
    tone === "gain"
      ? "text-emerald-400"
      : tone === "loss"
        ? "text-rose-400"
        : tone === "muted"
          ? "text-slate-300"
          : "text-brand-label";
  return (
    <div className="flex items-baseline gap-2">
      <span className="font-pixel text-[9px] uppercase text-brand-label">{label}</span>
      <span className={`font-semibold tabular-nums ${color}`}>{value}</span>
    </div>
  );
}

function Toggle({
  on,
  onClick,
  children,
}: {
  on: boolean;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      onClick={onClick}
      className={`flex items-center gap-1.5 rounded-[8px] border-2 px-3 py-1.5 text-xs font-bold transition-colors ${
        on
          ? "border-outline bg-brand/20 text-brand-label"
          : "border-outline bg-panel text-slate-400 hover:bg-raised"
      }`}
    >
      <span
        className={`flex h-3.5 w-3.5 items-center justify-center rounded-[3px] ${
          on ? "bg-brand text-white" : "border-2 border-outline"
        }`}
      >
        {on && <Check className="h-3 w-3" />}
      </span>
      {children}
    </button>
  );
}
