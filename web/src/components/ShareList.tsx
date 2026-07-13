import { useEffect, useMemo, useRef, useState } from "react";
import { Check, Copy, Download, ImageDown, Images, Send, X } from "lucide-react";
import { cardImageURL, getGame, productIDFromTcgURL, type TradeView } from "../api";
import { brl0, pct as pctFmt, usd } from "../format";
import { Button } from "./ui/button";
import { Card } from "./ui/card";
import { Input } from "./ui/input";
import { ToggleGroup } from "./ui/toggle-group";

// ShareList turns held cards into something you can hand to a buyer: a copyable
// text block and a downloadable image grid. Asking prices are seeded as a
// percentage of the live TCGplayer price (80/90/100%) and editable per card.
// Prices show in US$ (international buyers) or R$ (local sales); the currency
// also picks the language of the exported text. You choose whether to reveal
// what you paid (kept in BRL, the real cost basis).
type Currency = "USD" | "BRL";

export default function ShareList({
  holdings,
  fxRate,
  onClose,
}: {
  holdings: TradeView[];
  fxRate: number;
  onClose: () => void;
}) {
  const [currency, setCurrency] = useState<Currency>("USD");
  const [includePaid, setIncludePaid] = useState(false);
  const [includeAsking, setIncludeAsking] = useState(true);
  const [showProfit, setShowProfit] = useState(true);
  const [askPct, setAskPct] = useState(90);
  const [rows, setRows] = useState<Record<string, RowState>>(() =>
    seedRows(holdings, 90, "USD", fxRate),
  );
  const [previewMode, setPreviewMode] = useState<PreviewMode>("image");
  const [copied, setCopied] = useState(false);
  const [copiedImg, setCopiedImg] = useState(false);
  const [busy, setBusy] = useState(false);
  const [sharing, setSharing] = useState(false);

  const selected = useMemo(
    () => holdings.filter((t) => rows[t.id]?.include),
    [holdings, rows],
  );

  const opts: ShareOpts = { includePaid, includeAsking };
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
      const blob = await buildImage(selected, rows, opts, currency);
      if (!blob) {
        return;
      }
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `${getGame()}-for-sale.png`;
      a.click();
      URL.revokeObjectURL(url);
    } finally {
      setBusy(false);
    }
  };

  // copyImage puts the PNG on the clipboard so it can be pasted straight into a
  // chat on desktop (where the native share sheet is usually unavailable).
  const copyImage = async () => {
    setBusy(true);
    try {
      const blob = await buildImage(selected, rows, opts, currency);
      if (!blob || typeof ClipboardItem === "undefined") {
        return;
      }
      await navigator.clipboard.write([new ClipboardItem({ "image/png": blob })]);
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
      const blob = await buildImage(selected, rows, opts, currency);
      const file = blob ? new File([blob], `${getGame()}-for-sale.png`, { type: "image/png" }) : null;
      if (file && navigator.canShare?.({ files: [file] })) {
        await navigator.share({ files: [file], text });
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
          <h3 className="text-sm font-semibold text-slate-100">Share list</h3>
          <p className="mt-0.5 text-xs text-slate-400">
            Pick cards, then Share to WhatsApp. Asking prices are a % of the live
            TCGplayer price, in US$ or R$ — edit any of them.
          </p>
        </div>
        <button
          onClick={onClose}
          className="rounded-md border border-slate-700 bg-slate-800/60 p-1.5 text-slate-400 hover:bg-slate-800 hover:text-slate-100"
          title="Close"
        >
          <X className="h-3.5 w-3.5" />
        </button>
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <ToggleGroup
          value={currency}
          onChange={(v) => applyCurrency(v as Currency)}
          options={currencyOptions}
        />
        <span className="mx-1 h-5 w-px bg-slate-800" />
        <Toggle on={includeAsking} onClick={() => setIncludeAsking((v) => !v)}>
          Asking price
        </Toggle>
        <Toggle on={includePaid} onClick={() => setIncludePaid((v) => !v)}>
          Price paid
        </Toggle>
        {includeAsking && (
          <Toggle on={showProfit} onClick={() => setShowProfit((v) => !v)}>
            My profit
          </Toggle>
        )}
        {includeAsking && (
          <div className="flex items-center gap-2">
            <span className="text-[10px] font-medium uppercase tracking-wide text-slate-500">
              % of TCG
            </span>
            <ToggleGroup
              value={String(askPct)}
              onChange={(v) => applyPct(Number(v))}
              options={askPctOptions}
            />
          </div>
        )}
        <span className="ml-auto text-xs text-slate-500">
          {selected.length} of {holdings.length} selected
        </span>
      </div>

      <div className="overflow-x-auto rounded-xl border border-slate-800 bg-slate-900/40">
        <table className="w-full min-w-[560px] text-sm">
          <thead>
            <tr className="border-b border-slate-800 text-left text-xs uppercase tracking-wide text-slate-500">
              <th className="px-3 py-2 font-medium">
                <input
                  type="checkbox"
                  aria-label={allSelected ? "Unmark all" : "Mark all"}
                  title={allSelected ? "Unmark all" : "Mark all"}
                  checked={allSelected}
                  ref={(el) => {
                    if (el) el.indeterminate = !allSelected && someSelected;
                  }}
                  onChange={toggleAll}
                  className="h-4 w-4 accent-accent-500"
                />
              </th>
              <th className="px-3 py-2 font-medium">Card</th>
              <th className="px-3 py-2 text-right font-medium">TCG</th>
              {includePaid && <th className="px-3 py-2 text-right font-medium">Paid /ea</th>}
              {includeAsking && <th className="px-3 py-2 text-right font-medium">%</th>}
              {includeAsking && (
                <th className="px-3 py-2 text-right font-medium">
                  Asking {currency === "USD" ? "US$" : "R$"} /ea
                </th>
              )}
              {showProfitCol && <th className="px-3 py-2 text-right font-medium">My profit</th>}
            </tr>
          </thead>
          <tbody>
            {holdings.map((t) => {
              const row = rows[t.id];
              const paidUnit = t.costBRL / Math.max(t.qty, 1);
              const tcgInCurrency = marketInCurrency(t, currency, fxRate);
              return (
                <tr key={t.id} className="border-b border-slate-800/60 last:border-0">
                  <td className="px-3 py-2">
                    <input
                      type="checkbox"
                      checked={row?.include ?? false}
                      onChange={(e) => setRow(t.id, { include: e.target.checked })}
                      className="h-4 w-4 accent-accent-500"
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
                    {t.marketKnown ? money(tcgInCurrency, currency) : "—"}
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
                          disabled={!t.marketKnown || t.marketUSD <= 0}
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
          <SummaryStat label="Asking total" value={money(totals.asking, currency)} tone="accent" />
          {showProfit && (
            <SummaryStat label="Your cost" value={money(totals.cost, currency)} tone="muted" />
          )}
          {showProfit && (
            <SummaryStat
              label="Your profit"
              value={`${money(totals.profit, currency)} · ${pctFmt(totals.margin)}`}
              tone={totals.profit >= 0 ? "gain" : "loss"}
            />
          )}
        </div>
      )}

      <div>
        <div className="mb-1.5 flex items-center justify-between">
          <span className="text-[10px] font-medium uppercase tracking-wide text-slate-500">
            Preview
          </span>
          <ToggleGroup
            value={previewMode}
            onChange={(v) => setPreviewMode(v as PreviewMode)}
            options={previewOptions}
          />
        </div>
        {previewMode === "text" ? (
          <pre className="max-h-60 overflow-auto whitespace-pre-wrap rounded-xl border border-slate-800 bg-slate-950/60 p-3 text-xs text-slate-300">
            {preview || "Select at least one card to build your list."}
          </pre>
        ) : (
          <ImagePreview selected={selected} rows={rows} opts={opts} currency={currency} total={total} />
        )}
      </div>

      <div className="flex flex-wrap justify-end gap-2">
        <Button variant="outline" onClick={copyText} disabled={selected.length === 0}>
          {copied ? <Check /> : <Copy />} {copied ? "Copied!" : "Copy text"}
        </Button>
        <Button variant="outline" onClick={copyImage} disabled={selected.length === 0 || busy}>
          {copiedImg ? <Check /> : <Images />} {copiedImg ? "Copied!" : "Copy image"}
        </Button>
        <Button variant="outline" onClick={downloadImage} disabled={selected.length === 0 || busy}>
          {busy ? <ImageDown /> : <Download />} {busy ? "Building…" : "Download"}
        </Button>
        <Button variant="accent" onClick={shareList} disabled={selected.length === 0 || sharing}>
          <Send /> {sharing ? "Sharing…" : "Share"}
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

const previewOptions = [
  { value: "image", label: "Image" },
  { value: "text", label: "Text" },
];

type PreviewMode = "image" | "text";

interface RowState {
  include: boolean;
  ask: number;
  pct: number;
}

interface ShareOpts {
  includePaid: boolean;
  includeAsking: boolean;
}

// money formats an amount in the chosen currency: USD with cents, BRL rounded
// (pt-BR grouping) — matching how the Quotes feature shows reais.
function money(value: number, currency: Currency): string {
  return currency === "USD" ? usd(value) : brl0(value);
}

// marketInCurrency converts a card's live TCG price into the display currency.
// fxRate is USD per BRL (e.g. 0.195), so BRL = USD / fxRate.
function marketInCurrency(t: TradeView, currency: Currency, fxRate: number): number {
  if (!t.marketKnown || t.marketUSD <= 0) {
    return 0;
  }
  if (currency === "USD") {
    return t.marketUSD;
  }
  return fxRate > 0 ? t.marketUSD / fxRate : t.marketUSD;
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

function seedRows(
  holdings: TradeView[],
  pct: number,
  currency: Currency,
  fxRate: number,
): Record<string, RowState> {
  const out: Record<string, RowState> = {};
  for (const t of holdings) {
    out[t.id] = { include: true, ask: askFromMarket(t, pct, currency, fxRate), pct };
  }
  return out;
}

function eaSuffix(qty: number): string {
  return qty > 1 ? " ea" : "";
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
  const noun = selected.length === 1 ? (pt ? "carta" : "card") : pt ? "cartas" : "cards";
  const lines: string[] = [
    `${gameLabel()} — ${pt ? "à venda" : "for sale"} (${selected.length} ${noun})`,
    "",
  ];
  let total = 0;
  for (const t of selected) {
    const row = rows[t.id];
    lines.push(`${t.name} (${t.number})`);
    const meta = [t.condition || "NM", `${t.qty}x`];
    if (opts.includeAsking) {
      const lineTotal = row.ask * Math.max(t.qty, 1);
      total += lineTotal;
      if (t.qty > 1) {
        meta.push(`${money(row.ask, currency)}${eaSuffix(t.qty)} = ${money(lineTotal, currency)}`);
      } else {
        meta.push(money(row.ask, currency));
      }
    }
    lines.push(meta.join(" · "));
    if (opts.includePaid) {
      const paid = brl0(t.costBRL / Math.max(t.qty, 1));
      lines.push(`${pt ? "pago" : "paid"}: ${paid}${eaSuffix(t.qty)}`);
    }
    lines.push("");
  }
  if (opts.includeAsking) {
    lines.push(`💰 Total: ${money(total, currency)}`);
  }
  return lines.join("\n").trimEnd();
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

const C_BG = "#0f0f12";
const C_TITLE = "#f4f4f7";
const C_CELL = "#1c1c21";
const C_NAME = "#e6e6ec";
const C_META = "#9c9ca7";
const C_ASK = "#8c86ff";
const C_PAID = "#7a7a85";
const C_MUTED = "#7a7a85";

interface Layout {
  cols: number;
  rowsN: number;
  width: number;
  height: number;
  gridTop: number;
  gridH: number;
}

function layout(count: number, opts: ShareOpts): Layout {
  const cols = Math.min(4, Math.max(1, count));
  const rowsN = Math.ceil(count / cols);
  const width = PAD * 2 + cols * IMG_W + (cols - 1) * GAP;
  const gridH = rowsN * (IMG_H + CAPTION_H) + Math.max(rowsN - 1, 0) * GAP;
  const footerH = opts.includeAsking ? FOOTER_H : 0;
  const height = PAD * 2 + TITLE_H + gridH + footerH;
  return { cols, rowsN, width, height, gridTop: PAD + TITLE_H, gridH };
}

// imgSrc resolves a card's art, preferring its exact TCGplayer product image
// (routed through the same-origin proxy so the canvas isn't tainted) over the
// number-keyed Liga lookup — the latter can't tell variant prints apart.
function imgSrc(t: TradeView): string | null {
  const pid = productIDFromTcgURL(t.tcgUrl);
  if (pid) {
    return cardImageURL(t.set, t.number, pid);
  }
  return t.set ? cardImageURL(t.set, t.number) : null;
}

function imgKey(t: TradeView): string {
  return `${t.set}|${t.number}|${productIDFromTcgURL(t.tcgUrl) ?? ""}`;
}

function paint(
  ctx: CanvasRenderingContext2D,
  selected: TradeView[],
  rows: Record<string, RowState>,
  opts: ShareOpts,
  currency: Currency,
  total: number,
  cache: Map<string, HTMLImageElement | null>,
) {
  const lay = layout(selected.length, opts);

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
  ctx.fillText(`${selected.length} cards`, lay.width - PAD, PAD + 8);
  ctx.textAlign = "left";

  selected.forEach((t, i) => {
    const c = i % lay.cols;
    const r = Math.floor(i / lay.cols);
    const x = PAD + c * (IMG_W + GAP);
    const y = lay.gridTop + r * (IMG_H + CAPTION_H + GAP);
    drawCell(ctx, t, rows[t.id], opts, currency, cache.get(imgKey(t)) ?? null, x, y);
  });

  if (opts.includeAsking) {
    const fy = lay.gridTop + lay.gridH + 16;
    ctx.fillStyle = C_META;
    ctx.font = "500 14px ui-sans-serif, system-ui, sans-serif";
    ctx.textAlign = "left";
    ctx.fillText("Total", PAD, fy);
    ctx.fillStyle = C_ASK;
    ctx.font = "700 20px ui-sans-serif, system-ui, sans-serif";
    ctx.textAlign = "right";
    ctx.fillText(money(total, currency), lay.width - PAD, fy - 3);
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
    ctx.fillText(t.number, x + IMG_W / 2, y + IMG_H / 2 - 6);
    ctx.textAlign = "left";
  }

  const cy = y + IMG_H + 8;
  ctx.textAlign = "left";
  ctx.fillStyle = C_NAME;
  ctx.font = "600 14px ui-sans-serif, system-ui, sans-serif";
  ctx.fillText(truncate(ctx, t.name, IMG_W), x, cy);

  ctx.fillStyle = C_META;
  ctx.font = "12px ui-sans-serif, system-ui, sans-serif";
  const meta = [t.condition || "NM", `${t.qty}x`].join(" · ");
  ctx.fillText(meta, x, cy + 20);

  if (opts.includeAsking && row) {
    ctx.fillStyle = C_ASK;
    ctx.font = "700 16px ui-sans-serif, system-ui, sans-serif";
    ctx.fillText(`${money(row.ask, currency)}${eaSuffix(t.qty)}`, x, cy + 38);
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

async function buildImage(
  selected: TradeView[],
  rows: Record<string, RowState>,
  opts: ShareOpts,
  currency: Currency,
): Promise<Blob | null> {
  if (selected.length === 0) {
    return null;
  }
  const cache = new Map<string, HTMLImageElement | null>();
  await ensureImages(selected, cache);
  const lay = layout(selected.length, opts);
  const canvas = document.createElement("canvas");
  canvas.width = lay.width * SCALE;
  canvas.height = lay.height * SCALE;
  const ctx = canvas.getContext("2d");
  if (!ctx) {
    return null;
  }
  ctx.scale(SCALE, SCALE);
  const total = selected.reduce((sum, t) => sum + (rows[t.id]?.ask ?? 0) * Math.max(t.qty, 1), 0);
  paint(ctx, selected, rows, opts, currency, total, cache);
  return await new Promise((resolve) => canvas.toBlob(resolve, "image/png"));
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
}: {
  selected: TradeView[];
  rows: Record<string, RowState>;
  opts: ShareOpts;
  currency: Currency;
  total: number;
}) {
  const canvasRef = useRef<HTMLCanvasElement | null>(null);
  const cacheRef = useRef<Map<string, HTMLImageElement | null>>(new Map());
  const [imgVersion, setImgVersion] = useState(0);

  const ids = selected.map((t) => t.id).join(",");

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
    const canvas = canvasRef.current;
    if (!canvas || selected.length === 0) {
      return;
    }
    const lay = layout(selected.length, opts);
    canvas.width = lay.width * SCALE;
    canvas.height = lay.height * SCALE;
    const ctx = canvas.getContext("2d");
    if (!ctx) {
      return;
    }
    ctx.scale(SCALE, SCALE);
    paint(ctx, selected, rows, opts, currency, total, cacheRef.current);
  }, [ids, selected, rows, opts, currency, total, imgVersion]);

  if (selected.length === 0) {
    return (
      <div className="rounded-xl border border-slate-800 bg-slate-950/60 p-6 text-center text-xs text-slate-500">
        Select at least one card to build your list.
      </div>
    );
  }

  return (
    <div className="max-h-[36rem] overflow-auto rounded-xl border border-slate-800 bg-slate-950/60 p-3">
      <canvas ref={canvasRef} className="mx-auto h-auto max-w-full" />
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
          : "text-accent-300";
  return (
    <div className="flex items-baseline gap-2">
      <span className="text-[10px] uppercase tracking-wide text-slate-500">{label}</span>
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
      className={`flex items-center gap-1.5 rounded-lg border px-3 py-1.5 text-xs font-medium transition-colors ${
        on
          ? "border-accent-500/40 bg-accent-500/15 text-accent-200"
          : "border-slate-700 bg-slate-800/40 text-slate-400 hover:bg-slate-800"
      }`}
    >
      <span
        className={`flex h-3.5 w-3.5 items-center justify-center rounded ${
          on ? "bg-accent-500 text-slate-950" : "border border-slate-600"
        }`}
      >
        {on && <Check className="h-3 w-3" />}
      </span>
      {children}
    </button>
  );
}
