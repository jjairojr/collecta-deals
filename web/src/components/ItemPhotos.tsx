import { useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { Camera, ChevronLeft, ChevronRight, Trash2, X } from "lucide-react";
import { deleteTradePhoto, uploadTradePhotos, type TradeView } from "../api";
import { shrinkPhoto } from "../photo";
import { Button } from "./ui/button";

export function photosOf(item: TradeView): string[] {
  const list = item.photos ?? [];
  if (item.imageURL && !list.includes(item.imageURL)) {
    return [item.imageURL, ...list];
  }
  return list;
}

export function PhotoButton({
  item,
  game,
  label,
  variant = "ghost",
  onChanged,
  onFailed,
}: {
  item: TradeView;
  game?: string;
  label?: string;
  variant?: "ghost" | "outline" | "primary";
  onChanged: () => Promise<void> | void;
  onFailed?: (message: string) => void;
}) {
  const input = useRef<HTMLInputElement>(null);
  const [busy, setBusy] = useState(false);

  const upload = async (files: FileList | null) => {
    if (!files || files.length === 0) {
      return;
    }
    setBusy(true);
    try {
      const blobs = await Promise.all(Array.from(files).map(shrinkPhoto));
      await uploadTradePhotos(item.id, blobs, game);
      await onChanged();
    } catch (err: unknown) {
      onFailed?.(err instanceof Error ? err.message : "falha ao enviar as fotos");
    } finally {
      setBusy(false);
      if (input.current) {
        input.current.value = "";
      }
    }
  };

  return (
    <>
      <input
        ref={input}
        type="file"
        accept="image/*"
        multiple
        className="hidden"
        onChange={(e) => upload(e.target.files)}
      />
      <Button
        variant={variant}
        size="sm"
        disabled={busy}
        title="Enviar fotos do item (pode escolher várias)"
        onClick={() => input.current?.click()}
      >
        <Camera className="h-4 w-4" />
        {label ? (busy ? "Enviando…" : label) : null}
      </Button>
    </>
  );
}

export function PhotoViewer({
  item,
  game,
  start = 0,
  onClose,
  onChanged,
}: {
  item: TradeView;
  game?: string;
  start?: number;
  onClose: () => void;
  onChanged: () => Promise<void> | void;
}) {
  const photos = photosOf(item);
  const [index, setIndex] = useState(Math.min(start, Math.max(photos.length - 1, 0)));
  const [busy, setBusy] = useState(false);
  const [failed, setFailed] = useState<string | null>(null);
  const current = photos[Math.min(index, photos.length - 1)];

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        onClose();
      } else if (e.key === "ArrowRight") {
        setIndex((i) => (i + 1) % Math.max(photos.length, 1));
      } else if (e.key === "ArrowLeft") {
        setIndex((i) => (i - 1 + photos.length) % Math.max(photos.length, 1));
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [photos.length, onClose]);

  useEffect(() => {
    if (photos.length === 0) {
      onClose();
    }
  }, [photos.length, onClose]);

  const remove = async () => {
    if (!current) {
      return;
    }
    setBusy(true);
    setFailed(null);
    try {
      await deleteTradePhoto(item.id, current, game);
      setIndex((i) => Math.max(0, Math.min(i, photos.length - 2)));
      await onChanged();
    } catch (err: unknown) {
      setFailed(err instanceof Error ? err.message : "falha ao apagar a foto");
    } finally {
      setBusy(false);
    }
  };

  const ownPhoto = current?.startsWith("/api/item-photos/");

  return createPortal(
    <div
      className="fixed inset-0 z-50 flex flex-col bg-black/90 p-4"
      onClick={onClose}
      role="dialog"
      aria-label={`Fotos de ${item.name}`}
    >
      <div className="flex items-center gap-2 text-slate-200" onClick={(e) => e.stopPropagation()}>
        <div className="min-w-0 flex-1 truncate font-semibold">{item.name}</div>
        <span className="text-xs text-slate-400">
          {photos.length > 0 ? `${index + 1} / ${photos.length}` : "sem fotos"}
        </span>
        <PhotoButton item={item} game={game} label="Adicionar fotos" variant="outline" onChanged={onChanged} onFailed={setFailed} />
        {ownPhoto && (
          <Button variant="ghost" size="sm" disabled={busy} title="Apagar esta foto" onClick={remove}>
            <Trash2 className="h-4 w-4" />
          </Button>
        )}
        <Button variant="ghost" size="sm" onClick={onClose} title="Fechar (Esc)">
          <X className="h-4 w-4" />
        </Button>
      </div>
      {failed && <p className="mt-1 text-[11px] text-rose-300">{failed}</p>}
      <div className="relative flex min-h-0 flex-1 items-center justify-center" onClick={(e) => e.stopPropagation()}>
        {photos.length > 1 && (
          <button
            type="button"
            className="absolute left-0 top-1/2 -translate-y-1/2 rounded-full bg-black/60 p-2 text-white hover:bg-black/80"
            onClick={() => setIndex((i) => (i - 1 + photos.length) % photos.length)}
            title="Anterior"
          >
            <ChevronLeft className="h-6 w-6" />
          </button>
        )}
        {current && (
          <img src={current} alt={item.name} className="max-h-full max-w-full rounded object-contain" />
        )}
        {photos.length > 1 && (
          <button
            type="button"
            className="absolute right-0 top-1/2 -translate-y-1/2 rounded-full bg-black/60 p-2 text-white hover:bg-black/80"
            onClick={() => setIndex((i) => (i + 1) % photos.length)}
            title="Próxima"
          >
            <ChevronRight className="h-6 w-6" />
          </button>
        )}
      </div>
      {photos.length > 1 && (
        <div className="mt-3 flex justify-center gap-2 overflow-x-auto" onClick={(e) => e.stopPropagation()}>
          {photos.map((p, i) => (
            <button
              key={p}
              type="button"
              onClick={() => setIndex(i)}
              className={`h-16 w-16 shrink-0 overflow-hidden rounded border-2 ${
                i === index ? "border-brand" : "border-transparent opacity-60 hover:opacity-100"
              }`}
            >
              <img src={p} alt="" className="h-full w-full object-cover" />
            </button>
          ))}
        </div>
      )}
    </div>,
    document.body,
  );
}

export function PhotoStrip({
  item,
  game,
  onChanged,
  onFailed,
}: {
  item: TradeView;
  game?: string;
  onChanged: () => Promise<void> | void;
  onFailed?: (message: string) => void;
}) {
  const photos = photosOf(item);
  const [viewing, setViewing] = useState<number | null>(null);

  const remove = async (url: string) => {
    try {
      await deleteTradePhoto(item.id, url, game);
      await onChanged();
    } catch (err: unknown) {
      onFailed?.(err instanceof Error ? err.message : "falha ao apagar a foto");
    }
  };

  return (
    <div className="flex flex-wrap items-center gap-2">
      {photos.map((p, i) => (
        <div key={p} className="group relative h-20 w-20 overflow-hidden rounded border-2 border-outline">
          <button type="button" className="h-full w-full" onClick={() => setViewing(i)} title="Ver em tamanho grande">
            <img src={p} alt="" className="h-full w-full object-cover" />
          </button>
          {p.startsWith("/api/item-photos/") && (
            <button
              type="button"
              onClick={() => remove(p)}
              title="Apagar esta foto"
              className="absolute right-1 top-1 rounded-full bg-black/70 p-1 text-white opacity-0 transition-opacity group-hover:opacity-100"
            >
              <Trash2 className="h-3 w-3" />
            </button>
          )}
        </div>
      ))}
      <PhotoButton
        item={item}
        game={game}
        label={photos.length > 0 ? "Adicionar fotos" : "Enviar fotos"}
        variant="outline"
        onChanged={onChanged}
        onFailed={onFailed}
      />
      <span className="text-[10px] text-slate-500">
        A primeira foto vira a imagem do item no estoque, na vitrine e na lista de venda.
      </span>
      {viewing !== null && (
        <PhotoViewer item={item} game={game} start={viewing} onClose={() => setViewing(null)} onChanged={onChanged} />
      )}
    </div>
  );
}
