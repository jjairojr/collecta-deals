import { useState } from "react";
import { Link2, PackageOpen, Trash2 } from "lucide-react";
import { createPack, deletePack, openPack, type PackLink, type TradeView } from "../api";
import { Button } from "./ui/button";
import { Input } from "./ui/input";

export function packStockNote(link: PackLink): string {
  const inside = link.parentQty * link.units;
  return `${link.childQty} solta${link.childQty === 1 ? "" : "s"} + ${inside} em ${link.parentQty} fechada${link.parentQty === 1 ? "" : "s"}`;
}

export default function PacksEditor({
  trade,
  asParent,
  asChild,
  candidates,
  onChanged,
}: {
  trade: TradeView;
  asParent: PackLink | undefined;
  asChild: PackLink | undefined;
  candidates: TradeView[];
  onChanged: () => Promise<void>;
}) {
  const [parentID, setParentID] = useState("");
  const [units, setUnits] = useState("");
  const [openQty, setOpenQty] = useState("1");
  const [childName, setChildName] = useState("");
  const [childUnits, setChildUnits] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const run = async (action: () => Promise<unknown>) => {
    setBusy(true);
    setError(null);
    try {
      await action();
      await onChanged();
    } catch (err) {
      setError(err instanceof Error ? err.message : "não deu para salvar o vínculo");
    } finally {
      setBusy(false);
    }
  };

  const linkUnits = Math.floor(Number(units)) || 0;
  const qty = Math.max(Math.floor(Number(openQty)) || 0, 0);
  const newUnits = Math.floor(Number(childUnits)) || 0;
  const held = Math.max(trade.qty, 1);

  return (
    <div className="mt-5 border-t-2 border-outline/30 pt-4">
      <div className="flex items-center gap-2">
        <PackageOpen className="h-4 w-4 text-brand-label" />
        <span className="font-pixel text-[8px] uppercase text-brand-label">Embalagem</span>
      </div>
      <p className="mt-1 text-[11px] text-slate-500">
        Vincule o case à box (ou a box ao booster). A vitrine passa a vender o conteúdo das embalagens
        fechadas, e a venda abre uma embalagem sozinha quando acaba o solto.
      </p>

      {asChild && (
        <div className="mt-3 flex items-center justify-between gap-2 rounded-[10px] border-2 border-outline bg-page px-3 py-2">
          <div className="min-w-0 text-xs text-slate-300">
            <div className="truncate font-bold text-fg">
              {asChild.units} por {asChild.parentName}
            </div>
            <div className="tabular-nums text-slate-400">{packStockNote(asChild)}</div>
          </div>
          <button
            type="button"
            onClick={() => run(() => deletePack(asChild.id))}
            disabled={busy}
            title="Desfazer vínculo"
            aria-label={`Desfazer vínculo com ${asChild.parentName}`}
            className="inline-flex items-center rounded-[8px] border-2 border-outline bg-panel p-1 text-slate-500 hover:bg-raised hover:text-rose-300"
          >
            <Trash2 className="h-3.5 w-3.5" />
          </button>
        </div>
      )}

      {!asChild && (
        <div className="mt-3 grid grid-cols-[1fr_72px_auto] items-end gap-2">
          <div className="min-w-0">
            <label className="font-pixel text-[7px] uppercase text-brand-label">Vem dentro de</label>
            <select
              value={parentID}
              onChange={(e) => setParentID(e.target.value)}
              className="mt-1 h-9 w-full rounded-[8px] border-2 border-outline bg-page px-2 text-sm text-fg"
            >
              <option value="">Escolha a embalagem…</option>
              {candidates.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.name} ({Math.max(c.qty, 1)})
                </option>
              ))}
            </select>
          </div>
          <div>
            <label className="font-pixel text-[7px] uppercase text-brand-label">Unidades</label>
            <Input
              type="number"
              min={2}
              value={units}
              onChange={(e) => setUnits(e.target.value)}
              placeholder="12"
              className="mt-1 w-full"
            />
          </div>
          <Button
            variant="outline"
            onClick={() => run(() => createPack({ parentTradeID: parentID, childTradeID: trade.id, units: linkUnits }))}
            disabled={busy || !parentID || linkUnits < 2}
          >
            <Link2 /> Vincular
          </Button>
        </div>
      )}

      {asParent && (
        <div className="mt-3 flex items-end justify-between gap-2 rounded-[10px] border-2 border-outline bg-page px-3 py-2">
          <div className="min-w-0 text-xs text-slate-300">
            <div className="truncate font-bold text-fg">
              Contém {asParent.units} × {asParent.childName}
            </div>
            <div className="tabular-nums text-slate-400">{packStockNote(asParent)}</div>
          </div>
          <div className="flex items-end gap-2">
            <Input
              type="number"
              min={1}
              max={held}
              value={openQty}
              onChange={(e) => setOpenQty(e.target.value)}
              aria-label="Quantas embalagens abrir"
              className="w-16"
            />
            <Button
              variant="outline"
              onClick={() => run(() => openPack(trade.id, { qty }))}
              disabled={busy || qty < 1 || qty > held}
            >
              <PackageOpen /> Abrir
            </Button>
          </div>
        </div>
      )}

      {!asParent && (
        <div className="mt-3 grid grid-cols-[1fr_72px_auto] items-end gap-2">
          <div className="min-w-0">
            <label className="font-pixel text-[7px] uppercase text-brand-label">Abrir em produto novo</label>
            <Input
              value={childName}
              onChange={(e) => setChildName(e.target.value)}
              placeholder="Nome do conteúdo (ex.: Booster Box OP18)"
              className="mt-1 w-full"
            />
          </div>
          <div>
            <label className="font-pixel text-[7px] uppercase text-brand-label">Unidades</label>
            <Input
              type="number"
              min={2}
              value={childUnits}
              onChange={(e) => setChildUnits(e.target.value)}
              placeholder="12"
              className="mt-1 w-full"
            />
          </div>
          <Button
            variant="outline"
            onClick={() => run(() => openPack(trade.id, { qty: 1, units: newUnits, childName: childName.trim() }))}
            disabled={busy || !childName.trim() || newUnits < 2}
          >
            <PackageOpen /> Abrir 1
          </Button>
        </div>
      )}
      {error && <p className="mt-2 text-xs text-rose-300">{error}</p>}
    </div>
  );
}
