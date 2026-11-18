import { useEffect, useState } from "react";
import { Check, Layers, Plus, Save, Store, Trash2 } from "lucide-react";
import { createKit, deleteKit, updateKit, type Kit, type TradeView } from "../api";
import { brl } from "../format";
import { Button } from "./ui/button";
import { Input } from "./ui/input";

function round2(n: number): number {
  return Math.round(n * 100) / 100;
}

function savingPct(unitBRL: number, units: number, kitBRL: number): number {
  const full = unitBRL * units;
  if (full <= 0 || kitBRL <= 0) {
    return 0;
  }
  return Math.round(((full - kitBRL) / full) * 100);
}

function perUnitNote(unitBRL: number, units: number, kitBRL: number): string {
  if (kitBRL <= 0 || units <= 0) {
    return "";
  }
  const each = `${brl(kitBRL / units)} por un.`;
  const pct = savingPct(unitBRL, units, kitBRL);
  if (pct > 0) {
    return `${each} · ${pct}% abaixo do avulso`;
  }
  if (pct < 0) {
    return `${each} · ${-pct}% acima do avulso`;
  }
  return each;
}

export default function KitsEditor({
  trade,
  unitBRL,
  kits,
  onChanged,
}: {
  trade: TradeView;
  unitBRL: number;
  kits: Kit[];
  onChanged: () => Promise<void>;
}) {
  const [units, setUnits] = useState("");
  const [ask, setAsk] = useState("");
  const [card, setCard] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const run = async (action: () => Promise<unknown>) => {
    setBusy(true);
    setError(null);
    try {
      await action();
      await onChanged();
    } catch (err) {
      setError(err instanceof Error ? err.message : "não deu para salvar o kit");
    } finally {
      setBusy(false);
    }
  };

  const newUnits = Math.floor(Number(units)) || 0;
  const newAsk = Number(ask) || 0;
  const canCreate = newUnits >= 2 && newAsk > 0 && !busy;

  const add = () =>
    run(async () => {
      await createKit({
        tradeID: trade.id,
        units: newUnits,
        askBRL: round2(newAsk),
        cardBRL: round2(Number(card) || 0),
        listed: true,
      });
      setUnits("");
      setAsk("");
      setCard("");
    });

  return (
    <div className="mt-5 border-t-2 border-outline/30 pt-4">
      <div className="flex items-center gap-2">
        <Layers className="h-4 w-4 text-brand-label" />
        <span className="font-pixel text-[8px] uppercase text-brand-label">Kits na vitrine</span>
      </div>
      <p className="mt-1 text-[11px] text-slate-500">
        Venda várias unidades de uma vez com preço próprio. O kit usa o mesmo estoque do produto
        ({Math.max(trade.qty, 1)} un. neste lote) e a baixa sai em unidades quando o pedido é pago.
      </p>

      {kits.length > 0 && (
        <div className="mt-3 space-y-2">
          {kits.map((k) => (
            <KitRow key={k.id} kit={k} unitBRL={unitBRL} busy={busy} run={run} />
          ))}
        </div>
      )}

      <div className="mt-3 grid grid-cols-[72px_1fr_1fr_auto] items-end gap-2">
        <div>
          <label className="font-pixel text-[7px] uppercase text-brand-label">Unidades</label>
          <Input
            type="number"
            min={2}
            value={units}
            onChange={(e) => setUnits(e.target.value)}
            placeholder="2"
            className="mt-1 w-full"
          />
        </div>
        <div>
          <label className="font-pixel text-[7px] uppercase text-brand-label">Kit R$ (Pix)</label>
          <Input
            type="number"
            value={ask}
            onChange={(e) => setAsk(e.target.value)}
            placeholder={unitBRL > 0 && newUnits >= 2 ? String(round2(unitBRL * newUnits)) : "0"}
            className="mt-1 w-full"
          />
        </div>
        <div>
          <label className="font-pixel text-[7px] uppercase text-brand-label">Cartão R$</label>
          <Input
            type="number"
            value={card}
            onChange={(e) => setCard(e.target.value)}
            placeholder="igual ao Pix"
            className="mt-1 w-full"
          />
        </div>
        <Button variant="outline" onClick={add} disabled={!canCreate}>
          <Plus /> Criar
        </Button>
      </div>
      {newUnits >= 2 && newAsk > 0 && (
        <p className="mt-1 text-[11px] tabular-nums text-slate-400">{perUnitNote(unitBRL, newUnits, newAsk)}</p>
      )}
      {error && <p className="mt-2 text-xs text-rose-300">{error}</p>}
    </div>
  );
}

function KitRow({
  kit,
  unitBRL,
  busy,
  run,
}: {
  kit: Kit;
  unitBRL: number;
  busy: boolean;
  run: (action: () => Promise<unknown>) => Promise<void>;
}) {
  const [ask, setAsk] = useState(String(kit.askBRL));
  const [card, setCard] = useState(kit.cardBRL ? String(kit.cardBRL) : "");

  useEffect(() => {
    setAsk(String(kit.askBRL));
    setCard(kit.cardBRL ? String(kit.cardBRL) : "");
  }, [kit.askBRL, kit.cardBRL]);

  const askValue = round2(Number(ask) || 0);
  const cardValue = round2(Number(card) || 0);
  const dirty = askValue !== round2(kit.askBRL) || cardValue !== round2(kit.cardBRL ?? 0);

  return (
    <div className="rounded-[10px] border-2 border-outline bg-page px-3 py-2">
      <div className="grid grid-cols-[72px_1fr_1fr_auto] items-center gap-2">
        <div className="text-sm font-bold tabular-nums text-fg">{kit.units} un.</div>
        <Input
          type="number"
          value={ask}
          onChange={(e) => setAsk(e.target.value)}
          aria-label={`Preço Pix do kit de ${kit.units}`}
          className="w-full"
        />
        <Input
          type="number"
          value={card}
          onChange={(e) => setCard(e.target.value)}
          placeholder="igual ao Pix"
          aria-label={`Preço no cartão do kit de ${kit.units}`}
          className="w-full"
        />
        <div className="flex items-center gap-1">
          {dirty && (
            <button
              type="button"
              onClick={() => run(() => updateKit(kit.id, { askBRL: askValue, cardBRL: cardValue }))}
              disabled={busy || askValue <= 0}
              title="Salvar preço do kit"
              className="inline-flex items-center rounded-[8px] border-2 border-outline bg-brand p-1 text-white disabled:opacity-50"
            >
              <Save className="h-3.5 w-3.5" />
            </button>
          )}
          <button
            type="button"
            onClick={() => run(() => updateKit(kit.id, { listed: !kit.listed }))}
            disabled={busy}
            title={kit.listed ? "Na vitrine — clique para ocultar" : "Oculto — clique para colocar à venda"}
            className={`inline-flex items-center gap-1 rounded-[8px] border-2 border-outline px-2 py-1 text-xs font-bold ${
              kit.listed
                ? "bg-emerald-500/10 text-emerald-300 hover:bg-emerald-500/20"
                : "bg-panel text-slate-400 hover:bg-raised"
            }`}
          >
            {kit.listed ? <Check className="h-3 w-3" /> : <Store className="h-3 w-3" />}
            {kit.listed ? "À venda" : "Oculto"}
          </button>
          <button
            type="button"
            onClick={() => run(() => deleteKit(kit.id))}
            disabled={busy}
            title="Apagar kit"
            aria-label={`Apagar kit de ${kit.units}`}
            className="inline-flex items-center rounded-[8px] border-2 border-outline bg-panel p-1 text-slate-500 hover:bg-raised hover:text-rose-300"
          >
            <Trash2 className="h-3.5 w-3.5" />
          </button>
        </div>
      </div>
      <p className="mt-1 text-[11px] tabular-nums text-slate-400">{perUnitNote(unitBRL, kit.units, askValue)}</p>
    </div>
  );
}
