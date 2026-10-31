import { useCallback, useEffect, useState } from "react";
import { Check, RotateCcw } from "lucide-react";
import { listSyncQueue, markSyncDone, retrySync, type SyncStatus, type SyncTask } from "../api";
import { dayLabel } from "../format";
import EmptyState from "./EmptyState";
import { Chip } from "./PortfolioPage";
import { Button } from "./ui/button";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "./ui/table";

const POLL_MS = 60_000;

const statusLabels: Record<SyncStatus, string> = {
  pending: "Pendentes",
  error: "Com erro",
  done: "Feitas",
};

const actionLabels: Record<SyncTask["action"], string> = {
  remove: "Tirar",
  restore: "Devolver",
};

const targetLabels: Record<SyncTask["target"], string> = {
  liga: "Liga",
  myp: "MyP",
};

export function useSyncPending(): number {
  const [count, setCount] = useState(0);
  const refresh = useCallback(async () => {
    try {
      const r = await listSyncQueue("pending");
      setCount(r.tasks.length);
    } catch {
      setCount(0);
    }
  }, []);
  useEffect(() => {
    refresh();
    const id = window.setInterval(refresh, POLL_MS);
    window.addEventListener("focus", refresh);
    return () => {
      window.clearInterval(id);
      window.removeEventListener("focus", refresh);
    };
  }, [refresh]);
  return count;
}

export default function SyncQueuePage() {
  const [tasks, setTasks] = useState<SyncTask[]>([]);
  const [status, setStatus] = useState<SyncStatus>("pending");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState<string | null>(null);

  const load = useCallback(async () => {
    try {
      const r = await listSyncQueue();
      setTasks(r.tasks);
      setError(null);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "falha ao carregar a fila");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const act = async (id: string, fn: (id: string) => Promise<SyncTask>) => {
    setBusy(id);
    try {
      await fn(id);
      await load();
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "falha ao atualizar");
    } finally {
      setBusy(null);
    }
  };

  if (loading) {
    return <EmptyState>Carregando a fila…</EmptyState>;
  }

  const counts: Record<SyncStatus, number> = { pending: 0, error: 0, done: 0 };
  for (const t of tasks) {
    counts[t.status] += 1;
  }
  const visible = tasks.filter((t) => t.status === status);

  return (
    <div className="space-y-4">
      <p className="max-w-2xl text-sm text-slate-400">
        Toda saída de estoque fora da Liga (balcão, site, MyP, baixa sem venda) de um item que está na Liga vira uma
        pendência aqui. A rotina de baixa de vendas tira da Liga e marca como feita; se você já acertou na mão, marque
        como feita.
      </p>
      <div className="flex flex-wrap items-center gap-1.5">
        {(Object.keys(statusLabels) as SyncStatus[]).map((s) => (
          <Chip key={s} active={status === s} onClick={() => setStatus(s)}>
            {statusLabels[s]} · {counts[s]}
          </Chip>
        ))}
      </div>
      {error && <p className="text-[11px] text-rose-300">{error}</p>}
      {visible.length === 0 ? (
        <EmptyState>
          {status === "pending" ? "Nada para sincronizar com a Liga." : `Nenhuma tarefa ${statusLabels[status].toLowerCase()}.`}
        </EmptyState>
      ) : (
        <div className="sticker sticker-sm overflow-x-auto rounded-[12px] bg-panel">
          <Table className="min-w-[760px]">
            <TableHeader>
              <TableRow className="hover:bg-transparent">
                <TableHead>Item</TableHead>
                <TableHead>Jogo</TableHead>
                <TableHead>Ação</TableHead>
                <TableHead className="text-right">Un.</TableHead>
                <TableHead>Motivo</TableHead>
                <TableHead>Quando</TableHead>
                <TableHead />
              </TableRow>
            </TableHeader>
            <TableBody>
              {visible.map((t) => (
                <TableRow key={t.id}>
                  <TableCell>
                    <div className="truncate text-slate-100">{t.name}</div>
                    <div className="truncate text-[10px] text-slate-500">
                      {[t.number, t.variant, t.condition].filter(Boolean).join(" · ")}
                    </div>
                    {t.error && <div className="text-[10px] text-rose-300">{t.error}</div>}
                  </TableCell>
                  <TableCell className="whitespace-nowrap text-[11px] text-slate-400">{t.game}</TableCell>
                  <TableCell className="whitespace-nowrap text-xs text-slate-200">
                    {actionLabels[t.action]} da {targetLabels[t.target]}
                  </TableCell>
                  <TableCell className="text-right tabular-nums text-slate-200">{t.qty}</TableCell>
                  <TableCell className="text-[11px] text-slate-400">{t.reason}</TableCell>
                  <TableCell className="whitespace-nowrap text-[11px] text-slate-500">
                    {dayLabel(t.createdAt.slice(0, 10))}
                    {t.doneAt && t.status === "done" ? ` · feita ${dayLabel(t.doneAt.slice(0, 10))}` : ""}
                  </TableCell>
                  <TableCell className="text-right">
                    <div className="flex items-center justify-end gap-1">
                      {t.status !== "done" && (
                        <Button
                          variant="ghost"
                          size="sm"
                          disabled={busy === t.id}
                          title="Marcar como feita (acertei na Liga na mão)"
                          onClick={() => act(t.id, markSyncDone)}
                        >
                          <Check className="h-4 w-4" />
                        </Button>
                      )}
                      {t.status !== "pending" && (
                        <Button
                          variant="ghost"
                          size="sm"
                          disabled={busy === t.id}
                          title="Voltar para pendente"
                          onClick={() => act(t.id, retrySync)}
                        >
                          <RotateCcw className="h-4 w-4" />
                        </Button>
                      )}
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}
    </div>
  );
}
