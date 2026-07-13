import { useCallback, useState } from "react";
import { Check, Copy, ImageDown, Trash2 } from "lucide-react";
import { exportImage } from "../api";
import { buildBuyMessage, useSelection } from "../selection";
import { Button } from "./ui/button";

export default function SelectionTray() {
  const { items, count, clear } = useSelection();
  const [busy, setBusy] = useState(false);
  const [copied, setCopied] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const onExport = useCallback(async () => {
    setBusy(true);
    setError(null);
    try {
      const blob = await exportImage(items.map((i) => ({ set: i.set, number: i.number })));
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = "compro-cartas.png";
      document.body.appendChild(a);
      a.click();
      a.remove();
      URL.revokeObjectURL(url);
    } catch (err) {
      setError(err instanceof Error ? err.message : "export failed");
    } finally {
      setBusy(false);
    }
  }, [items]);

  const onCopy = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(buildBuyMessage(items));
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1800);
    } catch {
      setError("could not copy — clipboard blocked");
    }
  }, [items]);

  if (count === 0) {
    return null;
  }

  return (
    <div className="fixed inset-x-0 bottom-0 z-40 border-t border-slate-700 bg-slate-900/95 backdrop-blur">
      <div className="mx-auto flex max-w-6xl flex-wrap items-center gap-3 px-4 py-3">
        <span className="text-sm text-slate-200">
          <span className="font-semibold text-sky-300">{count}</span> card{count === 1 ? "" : "s"}{" "}
          selected
        </span>
        {error && <span className="text-xs text-rose-300">{error}</span>}
        <div className="ml-auto flex items-center gap-2">
          <Button variant="outline" onClick={onCopy}>
            {copied ? <Check className="text-emerald-400" /> : <Copy />}
            {copied ? "Copied" : "Copy message"}
          </Button>
          <Button variant="ghost" onClick={clear}>
            <Trash2 />
            Clear
          </Button>
          <Button variant="primary" onClick={onExport} disabled={busy}>
            <ImageDown />
            {busy ? "Building…" : "Export image"}
          </Button>
        </div>
      </div>
    </div>
  );
}
