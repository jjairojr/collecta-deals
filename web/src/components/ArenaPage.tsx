import { useCallback, useEffect, useRef, useState, type ReactNode } from "react";
import {
  Baby,
  Ban,
  CalendarCheck,
  Check,
  ChevronDown,
  ChevronRight,
  Coins,
  Copy,
  Download,
  Flag,
  Flame,
  Gavel,
  Maximize2,
  Plus,
  RefreshCw,
  Send,
  ShieldAlert,
  Swords,
  Trophy,
  Undo2,
  UserPlus,
  Users,
  X,
} from "lucide-react";
import {
  annulArenaDuel,
  approveArenaMember,
  createArenaEvent,
  creditArena,
  endArenaEvent,
  getArenaEvent,
  getArenaOverview,
  getArenaStatement,
  listArenaDuels,
  listArenaEvents,
  listArenaGames,
  listArenaMembers,
  listArenaPending,
  rejectArenaMember,
  removeArenaMember,
  resolveArenaDuel,
  reverseArenaEntry,
  setArenaCheckin,
  setArenaMinor,
  type ArenaCreditEntry,
  type ArenaDuel,
  type ArenaDuelFilter,
  type ArenaDuelStatus,
  type ArenaEntry,
  type ArenaEvent,
  type ArenaEventDetail,
  type ArenaMember,
  type ArenaOutcome,
  type ArenaOverview,
  type ArenaPending,
  type ArenaPlayer,
  type ArenaPodiumEntry,
  type ArenaPointKind,
  type ArenaReport,
} from "../api";
import { brl2, fullStamp } from "../format";
import EmptyState from "./EmptyState";
import { Badge, type BadgeProps } from "./ui/badge";
import { Button } from "./ui/button";
import { Card } from "./ui/card";
import { Checkbox } from "./ui/checkbox";
import { Input } from "./ui/input";
import { Kpi } from "./ui/kpi";
import { Select } from "./ui/select";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "./ui/table";
import { Tabs, TabsList, TabsTrigger } from "./ui/tabs";
import { ToggleGroup } from "./ui/toggle-group";

type Section = "geral" | "pedidos" | "membros" | "torneios" | "lancar" | "x1" | "artes";

const SECTIONS: Section[] = ["geral", "pedidos", "membros", "torneios", "lancar", "x1", "artes"];

const sectionLabels: Record<Section, string> = {
  geral: "Visão geral",
  pedidos: "Pedidos",
  membros: "Membros",
  torneios: "Torneios",
  lancar: "Lançar Collectas",
  x1: "X1",
  artes: "Artes",
};

const kindLabels: Record<ArenaPointKind, string> = {
  WELCOME: "Boas-vindas",
  ATTENDANCE: "Presença no torneio",
  PURCHASE: "Compra",
  ADJUST: "Ajuste",
  ENTRY_REDEEM: "Inscrição paga com Collectas",
  DUEL_STAKE: "Valor do X1",
  DUEL_PAYOUT: "Vitória no X1",
  DUEL_REFUND: "X1 devolvido",
  REVERSAL: "Estorno",
};

const duelStatusLabels: Record<ArenaDuelStatus, string> = {
  INVITED: "Convite enviado",
  DECLINED: "Recusado",
  EXPIRED: "Convite expirou",
  NEGOTIATING: "Definindo valores",
  LOCKED: "Valendo",
  DISPUTED: "Em disputa",
  SETTLED: "Encerrado",
  DRAWN: "Empate",
  CANCELLED: "Cancelado",
  ANNULLED: "Anulado",
};

const duelStatusVariants: Record<ArenaDuelStatus, BadgeProps["variant"]> = {
  INVITED: "sky",
  DECLINED: "default",
  EXPIRED: "default",
  NEGOTIATING: "sky",
  LOCKED: "amber",
  DISPUTED: "rose",
  SETTLED: "emerald",
  DRAWN: "emerald",
  CANCELLED: "default",
  ANNULLED: "default",
};

const reportLabels: Record<ArenaReport, string> = {
  WIN: "ganhei",
  LOSS: "perdi",
  DRAW: "empate",
};

const DUEL_FILTERS: ArenaDuelFilter[] = ["OPEN", "DISPUTED", "ALL"];

const duelFilterLabels: Record<ArenaDuelFilter, string> = {
  OPEN: "Abertos",
  DISPUTED: "Em disputa",
  ALL: "Todos",
};

const OPEN_DUEL: ArenaDuelStatus[] = ["INVITED", "NEGOTIATING", "LOCKED", "DISPUTED"];

const DECIDABLE_DUEL: ArenaDuelStatus[] = ["LOCKED", "DISPUTED"];

const REVERSIBLE: ArenaPointKind[] = ["PURCHASE", "ADJUST"];

const OUTCOMES: ArenaOutcome[] = ["CHALLENGER", "OPPONENT", "DRAW"];

type CreditKind = "PURCHASE" | "ADJUST";

const POINTS_PER_REAL = 30;
const MIN_PURCHASE_CENTS = 100;
const MAX_PURCHASE_CENTS = 10_000_000;
const MAX_ADJUST = 1_000_000;
const SEARCH_DELAY_MS = 300;

function isSection(value: string): value is Section {
  return SECTIONS.some((s) => s === value);
}

function isDuelFilter(value: string): value is ArenaDuelFilter {
  return DUEL_FILTERS.some((f) => f === value);
}

function isOutcome(value: string): value is ArenaOutcome {
  return OUTCOMES.some((o) => o === value);
}

function isCreditKind(value: string): value is CreditKind {
  return value === "PURCHASE" || value === "ADJUST";
}

function points(n: number): string {
  return n.toLocaleString("pt-BR");
}

function collectas(n: number): string {
  return `${points(n)} Collectas`;
}

function signedCollectas(n: number): string {
  return `${n > 0 ? "+" : ""}${collectas(n)}`;
}

function errorText(err: unknown, fallback: string): string {
  return err instanceof Error ? err.message : fallback;
}

function playerLabel(p: ArenaPlayer): string {
  return p.displayName || p.username;
}

function dateLabel(iso: string | null): string {
  if (!iso) {
    return "—";
  }
  const d = new Date(iso);
  return Number.isNaN(d.getTime()) ? "—" : d.toLocaleDateString("pt-BR");
}

function todayISO(): string {
  const d = new Date();
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
}

function validReason(reason: string): boolean {
  const t = reason.trim();
  return t.length >= 3 && t.length <= 200;
}

function num(v: string): number {
  const n = Number(v.replace(",", "."));
  return Number.isFinite(n) ? n : 0;
}

function purchasePoints(cents: number): number {
  return Math.floor((cents * POINTS_PER_REAL) / 100);
}

function adjustAmount(text: string): number {
  const t = text.trim();
  return /^[-+]?\d+$/.test(t) ? Number(t) : 0;
}

function outcomeLabel(duel: ArenaDuel, outcome: ArenaOutcome): string {
  if (outcome === "DRAW") {
    return "Empate";
  }
  return `Venceu ${playerLabel(outcome === "CHALLENGER" ? duel.challenger : duel.opponent)}`;
}

function useDebounced(value: string, ms: number): string {
  const [debounced, setDebounced] = useState(value);
  useEffect(() => {
    const id = window.setTimeout(() => setDebounced(value), ms);
    return () => window.clearTimeout(id);
  }, [value, ms]);
  return debounced;
}

function useLoad<T>(fetcher: () => Promise<T>, fallback: string) {
  const [data, setData] = useState<T | null>(null);
  const [error, setError] = useState<string | null>(null);
  const seq = useRef(0);
  const load = useCallback(async () => {
    seq.current += 1;
    const id = seq.current;
    try {
      const next = await fetcher();
      if (id === seq.current) {
        setData(next);
        setError(null);
      }
    } catch (err: unknown) {
      if (id === seq.current) {
        setError(errorText(err, fallback));
      }
    }
  }, [fetcher, fallback]);
  useEffect(() => {
    load();
  }, [load]);
  return { data, error, load };
}

function useRunner() {
  const [busy, setBusy] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const run = useCallback(async (key: string, work: () => Promise<unknown>): Promise<boolean> => {
    setBusy(key);
    setError(null);
    try {
      await work();
      return true;
    } catch (err: unknown) {
      setError(errorText(err, "falha ao salvar"));
      return false;
    } finally {
      setBusy(null);
    }
  }, []);
  return { busy, error, run };
}

export default function ArenaPage() {
  const [section, setSection] = useState<Section>("geral");
  const [version, setVersion] = useState(0);
  const [overview, setOverview] = useState<ArenaOverview | null>(null);
  const [overviewError, setOverviewError] = useState<string | null>(null);

  const loadOverview = useCallback(async () => {
    try {
      setOverview(await getArenaOverview());
      setOverviewError(null);
    } catch (err: unknown) {
      setOverviewError(errorText(err, "falha ao carregar a Arena"));
    }
  }, []);

  useEffect(() => {
    loadOverview();
  }, [loadOverview]);

  const refresh = () => {
    setVersion((v) => v + 1);
    loadOverview();
  };

  const tabLabel = (s: Section): string => {
    const count = s === "pedidos" ? overview?.pending : s === "x1" ? overview?.disputes : undefined;
    return count ? `${sectionLabels[s]} · ${count}` : sectionLabels[s];
  };

  const key = `${section}-${version}`;

  return (
    <div className="space-y-6">
      <p className="max-w-3xl text-sm text-slate-400">
        Arena Collecta: o programa de pontos da loja. Aqui você aprova quem pediu para entrar, cria os torneios com
        check-in, lança Collectas por compra ou ajuste, estorna lançamentos errados e decide os X1 em disputa. Os dados
        vêm do servidor do app Collecta.
      </p>

      <div className="flex flex-wrap items-center justify-between gap-3">
        <Tabs
          value={section}
          onValueChange={(v) => {
            if (isSection(v)) {
              setSection(v);
            }
          }}
        >
          <TabsList className="flex-wrap">
            {SECTIONS.map((s) => (
              <TabsTrigger key={s} value={s}>
                {tabLabel(s)}
              </TabsTrigger>
            ))}
          </TabsList>
        </Tabs>
        <Button variant="outline" size="sm" onClick={refresh}>
          <RefreshCw /> Atualizar
        </Button>
      </div>

      {overviewError && <ErrorBox>{overviewError}</ErrorBox>}

      {section === "geral" ? (
        <OverviewSection overview={overview} failed={overviewError !== null} />
      ) : section === "pedidos" ? (
        <PendingSection key={key} onChanged={loadOverview} />
      ) : section === "membros" ? (
        <MembersSection key={key} onChanged={loadOverview} />
      ) : section === "torneios" ? (
        <EventsSection key={key} onChanged={loadOverview} />
      ) : section === "lancar" ? (
        <CreditSection key={key} onChanged={loadOverview} />
      ) : section === "artes" ? (
        <ArtsSection />
      ) : (
        <DuelsSection key={key} onChanged={loadOverview} />
      )}
    </div>
  );
}

function OverviewSection({ overview, failed }: { overview: ArenaOverview | null; failed: boolean }) {
  if (!overview) {
    return <EmptyState>{failed ? "Não deu para carregar a Arena agora." : "Carregando a Arena…"}</EmptyState>;
  }
  return (
    <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
      <Kpi icon={<Users className="h-5 w-5" />} label="Membros" value={points(overview.members)} sub="aprovados na Arena" />
      <Kpi
        icon={<UserPlus className="h-5 w-5" />}
        label="Pedidos pendentes"
        value={points(overview.pending)}
        sub="esperando aprovação"
      />
      <Kpi icon={<Baby className="h-5 w-5" />} label="Menores" value={points(overview.minors)} sub="membros menores de idade" />
      <Kpi
        icon={<Coins className="h-5 w-5" />}
        label="Collectas em circulação"
        value={points(overview.circulating)}
        sub="somando todos os saldos"
      />
      <Kpi
        icon={<Flame className="h-5 w-5" />}
        label="Collectas queimados"
        value={points(overview.burned)}
        sub="taxa retida nos X1"
      />
      <Kpi
        icon={<Swords className="h-5 w-5" />}
        label="X1 na semana"
        value={points(overview.duelsWeek)}
        sub="encerrados nos últimos 7 dias"
      />
      <Kpi
        icon={<ShieldAlert className="h-5 w-5" />}
        label="Disputas abertas"
        value={points(overview.disputes)}
        sub="X1 esperando sua decisão"
      />
      <Kpi
        icon={<CalendarCheck className="h-5 w-5" />}
        label="Check-in aberto"
        value={points(overview.openEvents)}
        sub="torneios recebendo check-in"
      />
    </div>
  );
}

function PendingSection({ onChanged }: { onChanged: () => void }) {
  const { data: pending, error: loadError, load } = useLoad(listArenaPending, "falha ao carregar os pedidos");
  const { busy, error, run } = useRunner();

  const act = async (id: string, work: () => Promise<unknown>) => {
    if (await run(id, work)) {
      await load();
      onChanged();
    }
  };

  const reject = (p: ArenaPending) => {
    if (window.confirm(`Recusar o pedido de ${playerLabel(p)} para entrar na Arena?`)) {
      act(p.id, () => rejectArenaMember(p.id));
    }
  };

  if (!pending) {
    return loadError ? <ErrorBox>{loadError}</ErrorBox> : <EmptyState>Carregando pedidos…</EmptyState>;
  }

  return (
    <div className="space-y-4">
      {(error || loadError) && <ErrorBox>{error ?? loadError}</ErrorBox>}
      {pending.length === 0 ? (
        <EmptyState>Nenhum pedido para entrar na Arena agora.</EmptyState>
      ) : (
        <div className="sticker sticker-sm overflow-x-auto rounded-[12px] bg-panel">
          <Table className="min-w-[820px]">
            <TableHeader>
              <TableRow className="hover:bg-transparent">
                <TableHead>Jogador</TableHead>
                <TableHead>Pediu em</TableHead>
                <TableHead>Conta criada</TableHead>
                <TableHead className="text-right">Saldo</TableHead>
                <TableHead>Idade</TableHead>
                <TableHead />
              </TableRow>
            </TableHeader>
            <TableBody>
              {pending.map((p) => (
                <PendingRow
                  key={p.id}
                  player={p}
                  busy={busy === p.id}
                  onApprove={(minor) => act(p.id, () => approveArenaMember(p.id, minor))}
                  onReject={() => reject(p)}
                />
              ))}
            </TableBody>
          </Table>
        </div>
      )}
    </div>
  );
}

function PendingRow({
  player,
  busy,
  onApprove,
  onReject,
}: {
  player: ArenaPending;
  busy: boolean;
  onApprove: (minor: boolean) => void;
  onReject: () => void;
}) {
  const [minor, setMinor] = useState(player.arenaMinor);
  return (
    <TableRow>
      <TableCell>
        <PlayerName player={player} />
      </TableCell>
      <TableCell className="whitespace-nowrap text-[11px] text-slate-400">
        {player.arenaRequestedAt ? fullStamp(player.arenaRequestedAt) : "—"}
      </TableCell>
      <TableCell className="whitespace-nowrap text-[11px] text-slate-400">{dateLabel(player.createdAt)}</TableCell>
      <TableCell className="whitespace-nowrap text-right tabular-nums text-slate-200">{collectas(player.pointsBalance)}</TableCell>
      <TableCell>
        <label className="flex items-center gap-2 whitespace-nowrap text-xs text-slate-300">
          <Checkbox checked={minor} disabled={busy} onChange={(e) => setMinor(e.target.checked)} />
          menor de idade
        </label>
      </TableCell>
      <TableCell className="text-right">
        <div className="flex items-center justify-end gap-1">
          <Button variant="primary" size="sm" disabled={busy} onClick={() => onApprove(minor)}>
            <Check /> Aprovar
          </Button>
          <Button variant="ghost" size="sm" disabled={busy} onClick={onReject}>
            <Ban /> Recusar
          </Button>
        </div>
      </TableCell>
    </TableRow>
  );
}

function MembersSection({ onChanged }: { onChanged: () => void }) {
  const [q, setQ] = useState("");
  const query = useDebounced(q, SEARCH_DELAY_MS);
  const fetcher = useCallback(() => listArenaMembers(query), [query]);
  const { data: members, error: loadError, load } = useLoad(fetcher, "falha ao carregar os membros");
  const { busy, error, run } = useRunner();
  const [open, setOpen] = useState<string | null>(null);
  const [removing, setRemoving] = useState<string | null>(null);

  const changed = useCallback(async () => {
    await load();
    onChanged();
  }, [load, onChanged]);

  const act = async (id: string, work: () => Promise<unknown>) => {
    if (await run(id, work)) {
      setRemoving(null);
      await changed();
    }
  };

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-end gap-2">
        <Field label="Buscar jogador">
          <Input value={q} onChange={(e) => setQ(e.target.value)} placeholder="nome ou @usuário" className="w-72" />
        </Field>
      </div>
      {(error || loadError) && <ErrorBox>{error ?? loadError}</ErrorBox>}
      {!members ? (
        !loadError && <EmptyState>Carregando membros…</EmptyState>
      ) : members.length === 0 ? (
        <EmptyState>{query.trim() ? `Ninguém encontrado para “${query.trim()}”.` : "Nenhum membro na Arena ainda."}</EmptyState>
      ) : (
        <div className="sticker sticker-sm overflow-x-auto rounded-[12px] bg-panel">
          <Table className="min-w-[900px]">
            <TableHeader>
              <TableRow className="hover:bg-transparent">
                <TableHead>Jogador</TableHead>
                <TableHead>Status</TableHead>
                <TableHead className="text-right">Saldo</TableHead>
                <TableHead className="text-right">Check-ins</TableHead>
                <TableHead>Membro desde</TableHead>
                <TableHead>Idade</TableHead>
                <TableHead />
              </TableRow>
            </TableHeader>
            <TableBody>
              {members.map((m) => (
                <MemberRow
                  key={m.id}
                  member={m}
                  busy={busy === m.id}
                  open={open === m.id}
                  removing={removing === m.id}
                  onToggle={() => setOpen((k) => (k === m.id ? null : m.id))}
                  onMinor={(minor) => act(m.id, () => setArenaMinor(m.id, minor))}
                  onAskRemove={() => setRemoving((k) => (k === m.id ? null : m.id))}
                  onRemove={(reason) => act(m.id, () => removeArenaMember(m.id, reason))}
                  onCancelRemove={() => setRemoving(null)}
                  onChanged={changed}
                />
              ))}
            </TableBody>
          </Table>
        </div>
      )}
    </div>
  );
}

function MemberRow({
  member,
  busy,
  open,
  removing,
  onToggle,
  onMinor,
  onAskRemove,
  onRemove,
  onCancelRemove,
  onChanged,
}: {
  member: ArenaMember;
  busy: boolean;
  open: boolean;
  removing: boolean;
  onToggle: () => void;
  onMinor: (minor: boolean) => void;
  onAskRemove: () => void;
  onRemove: (reason: string) => void;
  onCancelRemove: () => void;
  onChanged: () => Promise<void>;
}) {
  const removed = member.arenaStatus === "REMOVED";
  return (
    <>
      <TableRow className={removed ? "opacity-60" : undefined}>
        <TableCell>
          <PlayerName player={member} />
        </TableCell>
        <TableCell>
          <Badge variant={removed ? "default" : "emerald"}>{removed ? "Removido" : "Membro"}</Badge>
        </TableCell>
        <TableCell className="whitespace-nowrap text-right font-semibold tabular-nums text-slate-100">
          {collectas(member.pointsBalance)}
        </TableCell>
        <TableCell className="text-right tabular-nums text-slate-200">{points(member.checkins)}</TableCell>
        <TableCell className="whitespace-nowrap text-[11px] text-slate-400">{dateLabel(member.arenaApprovedAt)}</TableCell>
        <TableCell>
          <label className="flex items-center gap-2 whitespace-nowrap text-xs text-slate-300">
            <Checkbox checked={member.arenaMinor} disabled={busy} onChange={(e) => onMinor(e.target.checked)} />
            menor de idade
          </label>
        </TableCell>
        <TableCell className="text-right">
          <div className="flex items-center justify-end gap-1">
            <Button variant="outline" size="sm" onClick={onToggle}>
              {open ? <ChevronDown /> : <ChevronRight />} Extrato
            </Button>
            {!removed && (
              <Button variant="ghost" size="sm" disabled={busy} onClick={onAskRemove}>
                <Ban /> Remover
              </Button>
            )}
          </div>
        </TableCell>
      </TableRow>
      {removing && (
        <TableRow className="bg-raised/30 hover:bg-raised/30">
          <TableCell colSpan={8} className="p-4">
            <p className="mb-2 text-[11px] text-amber-300">
              Remover tira {playerLabel(member)} da Arena e anula os X1 abertos dele, devolvendo os Collectas apostados.
            </p>
            <ReasonForm label="Motivo da remoção" confirm="Remover da Arena" busy={busy} onSubmit={onRemove} onCancel={onCancelRemove} />
          </TableCell>
        </TableRow>
      )}
      {open && (
        <TableRow className="bg-raised/30 hover:bg-raised/30">
          <TableCell colSpan={8} className="p-4">
            <StatementPanel member={member} onChanged={onChanged} />
          </TableCell>
        </TableRow>
      )}
    </>
  );
}

function StatementPanel({ member, onChanged }: { member: ArenaMember; onChanged: () => Promise<void> }) {
  const [entries, setEntries] = useState<ArenaEntry[]>([]);
  const [cursor, setCursor] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [reversing, setReversing] = useState<string | null>(null);
  const { busy, error, run } = useRunner();

  const loadPage = useCallback(
    async (after: string | null) => {
      setLoading(true);
      try {
        const page = await getArenaStatement(member.id, after ?? undefined);
        setEntries((prev) => (after ? [...prev, ...page.entries] : page.entries));
        setCursor(page.nextCursor);
        setLoadError(null);
      } catch (err: unknown) {
        setLoadError(errorText(err, "falha ao carregar o extrato"));
      } finally {
        setLoading(false);
      }
    },
    [member.id],
  );

  useEffect(() => {
    loadPage(null);
  }, [loadPage]);

  const reverse = async (entry: ArenaEntry, reason: string) => {
    if (await run(entry.id, () => reverseArenaEntry(entry.id, reason))) {
      setReversing(null);
      await loadPage(null);
      await onChanged();
    }
  };

  const reversed = new Set(entries.flatMap((e) => (e.reversedEntryId ? [e.reversedEntryId] : [])));

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <h3 className="font-pixel text-[8px] uppercase text-brand-label">Extrato de {playerLabel(member)}</h3>
        <span className="text-xs text-slate-400">
          Saldo atual <span className="font-semibold tabular-nums text-slate-100">{collectas(member.pointsBalance)}</span>
        </span>
      </div>
      {(error || loadError) && <ErrorBox>{error ?? loadError}</ErrorBox>}
      {entries.length === 0 ? (
        <p className="text-sm text-slate-500">{loading ? "Carregando extrato…" : "Nenhum lançamento ainda."}</p>
      ) : (
        <div className="sticker sticker-sm overflow-x-auto rounded-[10px] bg-panel">
          <Table className="min-w-[820px]">
            <TableHeader>
              <TableRow className="hover:bg-transparent">
                <TableHead>Quando</TableHead>
                <TableHead>Tipo</TableHead>
                <TableHead className="text-right">Valor</TableHead>
                <TableHead>Motivo</TableHead>
                <TableHead>Torneio</TableHead>
                <TableHead>Por</TableHead>
                <TableHead />
              </TableRow>
            </TableHeader>
            <TableBody>
              {entries.map((e) => {
                const canReverse = REVERSIBLE.includes(e.kind) && !reversed.has(e.id);
                return (
                  <StatementRow
                    key={e.id}
                    entry={e}
                    reversed={reversed.has(e.id)}
                    canReverse={canReverse}
                    reversing={reversing === e.id}
                    busy={busy === e.id}
                    onAskReverse={() => setReversing((k) => (k === e.id ? null : e.id))}
                    onReverse={(reason) => reverse(e, reason)}
                    onCancel={() => setReversing(null)}
                  />
                );
              })}
            </TableBody>
          </Table>
        </div>
      )}
      {cursor && (
        <Button variant="outline" size="sm" disabled={loading} onClick={() => loadPage(cursor)}>
          {loading ? "Carregando…" : "Carregar mais"}
        </Button>
      )}
    </div>
  );
}

function StatementRow({
  entry,
  reversed,
  canReverse,
  reversing,
  busy,
  onAskReverse,
  onReverse,
  onCancel,
}: {
  entry: ArenaEntry;
  reversed: boolean;
  canReverse: boolean;
  reversing: boolean;
  busy: boolean;
  onAskReverse: () => void;
  onReverse: (reason: string) => void;
  onCancel: () => void;
}) {
  const where = [entry.event?.name, entry.game?.name].filter(Boolean).join(" · ");
  return (
    <>
      <TableRow className={reversed ? "opacity-60" : undefined}>
        <TableCell className="whitespace-nowrap text-[11px] text-slate-400">{fullStamp(entry.createdAt)}</TableCell>
        <TableCell className="whitespace-nowrap text-xs text-slate-200">
          {kindLabels[entry.kind]}
          {reversed && (
            <Badge variant="default" className="ml-2">
              Estornado
            </Badge>
          )}
        </TableCell>
        <TableCell
          className={`whitespace-nowrap text-right font-semibold tabular-nums ${
            entry.amount >= 0 ? "text-emerald-300" : "text-rose-300"
          }`}
        >
          {signedCollectas(entry.amount)}
        </TableCell>
        <TableCell className="text-[11px] text-slate-400">{entry.reason || "—"}</TableCell>
        <TableCell className="text-[11px] text-slate-400">{where || "—"}</TableCell>
        <TableCell className="whitespace-nowrap text-[11px] text-slate-500">{entry.actor || "—"}</TableCell>
        <TableCell className="text-right">
          {canReverse && (
            <Button variant="ghost" size="sm" disabled={busy} onClick={onAskReverse}>
              <Undo2 /> Estornar
            </Button>
          )}
        </TableCell>
      </TableRow>
      {reversing && (
        <TableRow className="hover:bg-transparent">
          <TableCell colSpan={7} className="p-3">
            <ReasonForm
              label={`Motivo do estorno de ${signedCollectas(entry.amount)}`}
              confirm="Estornar"
              busy={busy}
              onSubmit={onReverse}
              onCancel={onCancel}
            />
          </TableCell>
        </TableRow>
      )}
    </>
  );
}

function EventsSection({ onChanged }: { onChanged: () => void }) {
  const { data: events, error: loadError, load } = useLoad(listArenaEvents, "falha ao carregar os torneios");
  const { busy, error, run } = useRunner();
  const [open, setOpen] = useState<string | null>(null);
  const [creating, setCreating] = useState(false);

  const changed = async () => {
    await load();
    onChanged();
  };

  const toggleCheckin = async (id: string, next: boolean) => {
    if (await run(id, () => setArenaCheckin(id, next))) {
      await changed();
    }
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2">
        {(error || loadError) && <p className="text-[11px] text-rose-300">{error ?? loadError}</p>}
        <div className="ml-auto">
          {!creating && (
            <Button variant="primary" onClick={() => setCreating(true)}>
              <Plus className="h-4 w-4" />
              Novo torneio
            </Button>
          )}
        </div>
      </div>

      {creating && (
        <EventForm
          onCancel={() => setCreating(false)}
          onCreated={async () => {
            setCreating(false);
            await changed();
          }}
        />
      )}

      {!events ? (
        !loadError && <EmptyState>Carregando torneios…</EmptyState>
      ) : events.length === 0 ? (
        <EmptyState hint="clique em Novo torneio">Nenhum torneio criado ainda.</EmptyState>
      ) : (
        <div className="sticker sticker-sm overflow-x-auto rounded-[12px] bg-panel">
          <Table className="min-w-[860px]">
            <TableHeader>
              <TableRow className="hover:bg-transparent">
                <TableHead>Torneio</TableHead>
                <TableHead>Jogo</TableHead>
                <TableHead>Data</TableHead>
                <TableHead>Código</TableHead>
                <TableHead className="text-right">Check-ins</TableHead>
                <TableHead className="text-right">X1</TableHead>
                <TableHead>Check-in</TableHead>
                <TableHead className="w-10" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {events.map((ev) => (
                <EventRow
                  key={ev.id}
                  event={ev}
                  open={open === ev.id}
                  busy={busy === ev.id}
                  onToggle={() => setOpen((k) => (k === ev.id ? null : ev.id))}
                  onCheckin={() => toggleCheckin(ev.id, !ev.checkinOpen)}
                  onChanged={changed}
                />
              ))}
            </TableBody>
          </Table>
        </div>
      )}
    </div>
  );
}

function EventRow({
  event,
  open,
  busy,
  onToggle,
  onCheckin,
  onChanged,
}: {
  event: ArenaEvent;
  open: boolean;
  busy: boolean;
  onToggle: () => void;
  onCheckin: () => void;
  onChanged: () => Promise<void>;
}) {
  return (
    <>
      <TableRow className="cursor-pointer" onClick={onToggle}>
        <TableCell>
          <div className="font-medium text-slate-100">{event.name}</div>
          <div className="truncate text-[10px] text-slate-500">
            {event.podium.length > 0 ? podiumLine(event.podium) : `criado por ${event.createdBy}`}
          </div>
        </TableCell>
        <TableCell className="whitespace-nowrap text-xs text-slate-300">{event.game.name}</TableCell>
        <TableCell className="whitespace-nowrap text-[11px] text-slate-400">{dateLabel(event.playedAt)}</TableCell>
        <TableCell className="font-pixel text-xs tracking-[0.2em] text-brand-label">{event.checkinCode}</TableCell>
        <TableCell className="text-right tabular-nums text-slate-200">{points(event.checkins)}</TableCell>
        <TableCell className="text-right tabular-nums text-slate-200">{points(event.duels)}</TableCell>
        <TableCell>
          {event.endedAt ? <Badge>Encerrado</Badge> : <OpenToggle open={event.checkinOpen} disabled={busy} onToggle={onCheckin} />}
        </TableCell>
        <TableCell className="text-right">
          {open ? <ChevronDown className="h-4 w-4 text-slate-500" /> : <ChevronRight className="h-4 w-4 text-slate-500" />}
        </TableCell>
      </TableRow>
      {open && (
        <TableRow className="bg-raised/30 hover:bg-raised/30">
          <TableCell colSpan={8} className="p-4">
            <EventDetailPanel eventId={event.id} onChanged={onChanged} />
          </TableCell>
        </TableRow>
      )}
    </>
  );
}

function placeLabel(place: number): string {
  return `${place}º`;
}

function podiumLine(podium: ArenaPodiumEntry[]): string {
  return podium.map((entry) => `${placeLabel(entry.place)} ${playerLabel(entry.user)}`).join(" · ");
}

function OpenToggle({ open, disabled, onToggle }: { open: boolean; disabled: boolean; onToggle: () => void }) {
  return (
    <button
      type="button"
      disabled={disabled}
      onClick={(e) => {
        e.stopPropagation();
        onToggle();
      }}
      title={open ? "Aberto — clique para fechar o check-in" : "Fechado — clique para abrir o check-in"}
      className={`rounded-[8px] border-2 border-outline px-2 py-1 text-xs font-bold disabled:opacity-50 ${
        open ? "bg-emerald-500/10 text-emerald-300 hover:bg-emerald-500/20" : "bg-panel text-slate-400 hover:text-slate-200"
      }`}
    >
      {open ? "Aberto" : "Fechado"}
    </button>
  );
}

function EventForm({ onCancel, onCreated }: { onCancel: () => void; onCreated: () => Promise<void> }) {
  const { data: games, error: gamesError } = useLoad(listArenaGames, "falha ao carregar os jogos");
  const [gameId, setGameId] = useState("");
  const [name, setName] = useState("");
  const [date, setDate] = useState(todayISO);
  const [checkinOpen, setCheckinOpen] = useState(true);
  const { busy, error, run } = useRunner();

  const fallback = games?.find((g) => g.slug === "onepiece") ?? games?.[0];
  const selected = gameId || String(fallback?.id ?? "");
  const ready = selected !== "" && name.trim().length > 0 && name.trim().length <= 60 && date !== "";

  const submit = async () => {
    const ok = await run("create", () =>
      createArenaEvent({
        gameId: Number(selected),
        name: name.trim(),
        playedAt: new Date(`${date}T12:00:00`).toISOString(),
        checkinOpen,
      }),
    );
    if (ok) {
      await onCreated();
    }
  };

  return (
    <Card className="space-y-3 p-4">
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <Field label="Jogo">
          <Select value={selected} onChange={(e) => setGameId(e.target.value)} disabled={!games || busy !== null}>
            {(games ?? []).map((g) => (
              <option key={g.id} value={String(g.id)}>
                {g.name}
              </option>
            ))}
          </Select>
        </Field>
        <Field label="Nome do torneio">
          <Input value={name} onChange={(e) => setName(e.target.value)} maxLength={60} placeholder="ex.: Liga de sábado" />
        </Field>
        <Field label="Data">
          <Input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
        </Field>
        <label className="flex items-center gap-2 self-end pb-2 text-sm text-slate-300">
          <Checkbox checked={checkinOpen} onChange={(e) => setCheckinOpen(e.target.checked)} />
          Check-in aberto
        </label>
      </div>
      {(error || gamesError) && <p className="text-[11px] text-rose-300">{error ?? gamesError}</p>}
      <div className="flex justify-end gap-2">
        <Button variant="ghost" onClick={onCancel} disabled={busy !== null}>
          <X className="h-4 w-4" />
          Cancelar
        </Button>
        <Button variant="primary" onClick={submit} disabled={busy !== null || !ready}>
          {busy ? "Criando…" : "Criar torneio"}
        </Button>
      </div>
    </Card>
  );
}

function CheckinCodeBox({ name, code, open }: { name: string; code: string; open: boolean }) {
  const [big, setBig] = useState(false);
  return (
    <div className="flex flex-wrap items-center justify-between gap-3 rounded-[12px] border-2 border-outline bg-panel px-4 py-3">
      <div>
        <div className="font-pixel text-[8px] uppercase text-brand-label">Código do check-in</div>
        <div className="mt-1 font-display text-3xl font-bold tracking-[0.3em] text-fg">{code}</div>
        <div className="mt-1 text-[11px] text-slate-400">
          {open
            ? "Mostre ou fale no balcão quando o torneio começar. Só quem digitar o código no app ganha a presença."
            : "Check-in fechado: o código não funciona até você abrir o check-in."}
        </div>
      </div>
      <Button variant="primary" size="sm" onClick={() => setBig(true)}>
        <Maximize2 /> Mostrar em tela cheia
      </Button>
      {big && (
        <button
          type="button"
          onClick={() => setBig(false)}
          className="fixed inset-0 z-50 flex flex-col items-center justify-center gap-6 bg-royal text-white"
        >
          <span className="font-pixel text-sm uppercase">Check-in da Arena Collecta</span>
          <span className="font-display text-3xl font-bold">{name}</span>
          <span className="font-display text-[22vw] font-bold leading-none tracking-[0.15em]">{code}</span>
          <span className="text-base">Abra o app Collecta, toque em Fazer check-in e digite o código.</span>
          <span className="text-xs opacity-70">clique para fechar</span>
        </button>
      )}
    </div>
  );
}

function EventDetailPanel({ eventId, onChanged }: { eventId: string; onChanged: () => Promise<void> }) {
  const fetcher = useCallback(() => getArenaEvent(eventId), [eventId]);
  const { data: event, error, load } = useLoad(fetcher, "falha ao carregar o torneio");
  const [notice, setNotice] = useState<string | null>(null);

  if (!event) {
    return error ? <ErrorBox>{error}</ErrorBox> : <p className="text-sm text-slate-500">Carregando check-ins…</p>;
  }
  const paid = event.checkins.filter((c) => c.paidWithPoints).length;
  return (
    <div className="space-y-3">
      {event.endedAt ? (
        <div className="rounded-[12px] border-2 border-outline bg-panel px-4 py-3 text-sm text-slate-300">
          Torneio encerrado em {fullStamp(event.endedAt)}. Check-in e X1 fechados.
        </div>
      ) : (
        <CheckinCodeBox name={event.name} code={event.checkinCode} open={event.checkinOpen} />
      )}
      <EndEventBox
        key={event.checkins.map((c) => `${c.user.id}:${c.place ?? ""}`).join(",")}
        event={event}
        notice={notice}
        onEdit={() => setNotice(null)}
        onDone={async (message) => {
          setNotice(message);
          await load();
          await onChanged();
        }}
      />
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <h3 className="font-pixel text-[8px] uppercase text-brand-label">Check-ins de {event.name}</h3>
        <span className="text-xs text-slate-400">
          {points(event.checkins.length)} check-ins · {points(paid)} pagos com Collectas
        </span>
      </div>
      {event.checkins.length === 0 ? (
        <p className="text-sm text-slate-500">Ninguém fez check-in ainda.</p>
      ) : (
        <div className="sticker sticker-sm overflow-x-auto rounded-[10px] bg-panel">
          <Table className="min-w-[560px]">
            <TableHeader>
              <TableRow className="hover:bg-transparent">
                <TableHead>Jogador</TableHead>
                <TableHead>Check-in</TableHead>
                <TableHead>Inscrição</TableHead>
                <TableHead>Pódio</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {event.checkins.map((c) => (
                <TableRow key={c.user.id}>
                  <TableCell>
                    <PlayerName player={c.user} minor={c.user.arenaMinor} />
                  </TableCell>
                  <TableCell className="whitespace-nowrap text-[11px] text-slate-400">{fullStamp(c.createdAt)}</TableCell>
                  <TableCell>{c.paidWithPoints ? <Badge variant="amber">Pago com Collectas</Badge> : "—"}</TableCell>
                  <TableCell>{c.place ? <Badge variant="emerald">{placeLabel(c.place)} lugar</Badge> : "—"}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}
    </div>
  );
}

const PODIUM_PLACES = [1, 2, 3];

function placeOf(event: ArenaEventDetail, place: number): string {
  return event.checkins.find((c) => c.place === place)?.user.id ?? "";
}

function EndEventBox({
  event,
  notice,
  onEdit,
  onDone,
}: {
  event: ArenaEventDetail;
  notice: string | null;
  onEdit: () => void;
  onDone: (message: string) => Promise<void>;
}) {
  const [picks, setPicks] = useState<string[]>(() => PODIUM_PLACES.map((place) => placeOf(event, place)));
  const [confirming, setConfirming] = useState(false);
  const { busy, error, run } = useRunner();
  const ended = event.endedAt !== null;
  const chosen = picks.filter((id) => id !== "");
  const repeated = new Set(chosen).size !== chosen.length;
  const changed = PODIUM_PLACES.some((place, index) => placeOf(event, place) !== picks[index]);

  const setPick = (index: number, id: string) => {
    onEdit();
    setPicks((current) => current.map((value, i) => (i === index ? id : value)));
  };

  const submit = async () => {
    let annulled = 0;
    const ok = await run("end", async () => {
      const result = await endArenaEvent(event.id, {
        first: picks[0] || null,
        second: picks[1] || null,
        third: picks[2] || null,
      });
      annulled = result.annulled;
    });
    if (ok) {
      setConfirming(false);
      await onDone(ended ? "Pódio salvo." : annulled > 0 ? `Torneio encerrado. ${points(annulled)} X1 ${annulled === 1 ? "anulado" : "anulados"}, Collectas devolvidos.` : "Torneio encerrado.");
    }
  };

  return (
    <Card className="space-y-3 p-4">
      <div className="flex items-center gap-2">
        <Trophy className="h-4 w-4 text-brand-label" />
        <h3 className="font-pixel text-[8px] uppercase text-brand-label">{ended ? "Pódio" : "Encerrar torneio"}</h3>
      </div>
      <p className="text-[11px] text-slate-400">
        {ended
          ? "Corrija o pódio se precisar. Quem entrar no pódio recebe uma notificação."
          : "Escolha o pódio (opcional) e encerre. O check-in fecha, ninguém mais chama X1 neste torneio e o pódio entra no ranking de Títulos do app. Pódio não dá Collectas."}
      </p>
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
        {PODIUM_PLACES.map((place, index) => (
          <Field key={place} label={`${placeLabel(place)} lugar`}>
            <Select value={picks[index]} onChange={(e) => setPick(index, e.target.value)} disabled={busy !== null || event.checkins.length === 0}>
              <option value="">Ninguém</option>
              {event.checkins.map((c) => (
                <option key={c.user.id} value={c.user.id}>
                  {playerLabel(c.user)} (@{c.user.username})
                </option>
              ))}
            </Select>
          </Field>
        ))}
      </div>
      {repeated && <p className="text-[11px] text-rose-300">Cada jogador só pode ter uma colocação.</p>}
      {!ended && event.openDuels > 0 && (
        <p className="text-[11px] text-amber-300">
          {event.openDuels === 1
            ? "1 X1 em andamento neste torneio vai ser anulado e os Collectas voltam para os jogadores."
            : `${points(event.openDuels)} X1 em andamento neste torneio vão ser anulados e os Collectas voltam para os jogadores.`}
          {event.disputedDuels > 0
            ? ` ${points(event.disputedDuels)} ${event.disputedDuels === 1 ? "está" : "estão"} em disputa: decida na aba X1 antes, se quiser que conte.`
            : ""}
        </p>
      )}
      {error && <p className="text-[11px] text-rose-300">{error}</p>}
      {notice && <p className="text-[11px] text-emerald-300">{notice}</p>}
      <div className="flex justify-end gap-2">
        {ended ? (
          <Button variant="primary" onClick={submit} disabled={busy !== null || repeated || !changed}>
            <Trophy /> {busy ? "Salvando…" : "Salvar pódio"}
          </Button>
        ) : confirming ? (
          <>
            <Button variant="ghost" onClick={() => setConfirming(false)} disabled={busy !== null}>
              <X /> Voltar
            </Button>
            <Button variant="primary" onClick={submit} disabled={busy !== null || repeated}>
              <Flag /> {busy ? "Encerrando…" : "Confirmar encerramento"}
            </Button>
          </>
        ) : (
          <Button variant="primary" onClick={() => setConfirming(true)} disabled={repeated}>
            <Flag /> Encerrar torneio
          </Button>
        )}
      </div>
    </Card>
  );
}

function CreditSection({ onChanged }: { onChanged: () => void }) {
  const [q, setQ] = useState("");
  const query = useDebounced(q, SEARCH_DELAY_MS);
  const fetcher = useCallback(
    () => (query.trim() ? listArenaMembers(query) : Promise.resolve<ArenaMember[]>([])),
    [query],
  );
  const { data: found, error: searchError } = useLoad(fetcher, "falha ao buscar jogadores");
  const [member, setMember] = useState<ArenaMember | null>(null);
  const [kind, setKind] = useState<CreditKind>("PURCHASE");
  const [value, setValue] = useState("");
  const [adjust, setAdjust] = useState("");
  const [reason, setReason] = useState("");
  const [result, setResult] = useState<{ entry: ArenaCreditEntry; player: ArenaMember } | null>(null);
  const { busy, error, run } = useRunner();

  const cents = Math.round(num(value) * 100);
  const preview = purchasePoints(cents);
  const amount = adjustAmount(adjust);
  const amountReady =
    kind === "PURCHASE"
      ? cents >= MIN_PURCHASE_CENTS && cents <= MAX_PURCHASE_CENTS
      : amount !== 0 && Math.abs(amount) <= MAX_ADJUST;
  const ready = member !== null && amountReady && validReason(reason);
  const candidates = (found ?? []).filter((m) => m.arenaStatus === "MEMBER");

  const submit = async () => {
    if (!member) {
      return;
    }
    const player = member;
    const input =
      kind === "PURCHASE"
        ? { kind, userId: player.id, cents, reason: reason.trim() }
        : { kind, userId: player.id, amount, reason: reason.trim() };
    const ok = await run("credit", async () => {
      const entry = await creditArena(input);
      setResult({ entry, player });
      setMember({ ...player, pointsBalance: player.pointsBalance + entry.amount });
    });
    if (ok) {
      setValue("");
      setAdjust("");
      setReason("");
      onChanged();
    }
  };

  return (
    <div className="space-y-4">
      {result && (
        <div className="rounded-[14px] border-2 border-emerald-900/50 bg-emerald-950/30 px-4 py-3 text-sm text-emerald-200">
          <div className="font-semibold">
            Lançado {signedCollectas(result.entry.amount)} para {playerLabel(result.player)}.
          </div>
          <div className="text-[11px] text-emerald-300/80">
            {kindLabels[result.entry.kind]} · {result.entry.reason || "sem motivo"} · {fullStamp(result.entry.createdAt)}
          </div>
        </div>
      )}

      <Card className="space-y-4 p-4">
        <div className="space-y-2">
          <SectionTitle>Jogador</SectionTitle>
          {member ? (
            <div className="flex flex-wrap items-center gap-3">
              <PlayerName player={member} minor={member.arenaMinor} />
              <span className="text-xs text-slate-400">
                saldo <span className="font-semibold tabular-nums text-slate-100">{collectas(member.pointsBalance)}</span>
              </span>
              <Button variant="ghost" size="sm" disabled={busy !== null} onClick={() => setMember(null)}>
                Trocar jogador
              </Button>
            </div>
          ) : (
            <div className="space-y-2">
              <Input
                value={q}
                onChange={(e) => setQ(e.target.value)}
                placeholder="buscar por nome ou @usuário"
                className="w-80"
                autoFocus
              />
              {searchError && <p className="text-[11px] text-rose-300">{searchError}</p>}
              {query.trim() && found && candidates.length === 0 && (
                <p className="text-[11px] text-slate-500">Nenhum membro ativo encontrado.</p>
              )}
              {candidates.length > 0 && (
                <div className="flex flex-wrap gap-2">
                  {candidates.slice(0, 12).map((m) => (
                    <button
                      key={m.id}
                      type="button"
                      onClick={() => setMember(m)}
                      className="rounded-[8px] border-2 border-outline bg-panel px-3 py-1.5 text-left hover:bg-raised"
                    >
                      <div className="text-sm font-medium text-slate-100">{playerLabel(m)}</div>
                      <div className="text-[10px] text-slate-500">
                        @{m.username} · {collectas(m.pointsBalance)}
                      </div>
                    </button>
                  ))}
                </div>
              )}
            </div>
          )}
        </div>

        <div className="space-y-2">
          <SectionTitle>Lançamento</SectionTitle>
          <ToggleGroup
            value={kind}
            onChange={(v) => {
              if (isCreditKind(v)) {
                setKind(v);
              }
            }}
            options={[
              { value: "PURCHASE", label: "Compra" },
              { value: "ADJUST", label: "Ajuste" },
            ]}
          />
          <div className="flex flex-wrap items-end gap-3">
            {kind === "PURCHASE" ? (
              <>
                <Field label="Valor da compra R$">
                  <Input
                    value={value}
                    onChange={(e) => setValue(e.target.value)}
                    inputMode="decimal"
                    placeholder="100,00"
                    className="w-40"
                  />
                </Field>
                <div className="pb-2 text-sm text-slate-300">
                  {cents > 0 ? (
                    <>
                      {brl2(cents / 100)} ={" "}
                      <span className="font-semibold tabular-nums text-emerald-300">{collectas(preview)}</span>
                      <span className="text-slate-500"> ({POINTS_PER_REAL} Collectas por R$ 1)</span>
                    </>
                  ) : (
                    <span className="text-slate-500">{POINTS_PER_REAL} Collectas por R$ 1 · mínimo R$ 1,00</span>
                  )}
                </div>
              </>
            ) : (
              <>
                <Field label="Ajuste (Collectas)">
                  <Input
                    value={adjust}
                    onChange={(e) => setAdjust(e.target.value)}
                    inputMode="numeric"
                    placeholder="ex.: 500 ou -200"
                    className="w-40"
                  />
                </Field>
                <div className="pb-2 text-sm text-slate-300">
                  {amount !== 0 ? (
                    <span className={`font-semibold tabular-nums ${amount > 0 ? "text-emerald-300" : "text-rose-300"}`}>
                      {signedCollectas(amount)}
                    </span>
                  ) : (
                    <span className="text-slate-500">número inteiro, negativo para debitar</span>
                  )}
                </div>
              </>
            )}
          </div>
          <Field label="Motivo">
            <Input
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              maxLength={200}
              placeholder={kind === "PURCHASE" ? "ex.: compra no balcão, pedido #123" : "ex.: correção de lançamento"}
              className="max-w-xl"
            />
          </Field>
        </div>

        {error && <p className="text-[11px] text-rose-300">{error}</p>}
        <div className="flex justify-end">
          <Button variant="primary" onClick={submit} disabled={busy !== null || !ready}>
            <Coins className="h-4 w-4" />
            {busy ? "Lançando…" : "Lançar Collectas"}
          </Button>
        </div>
      </Card>
    </div>
  );
}

function DuelsSection({ onChanged }: { onChanged: () => void }) {
  const [filter, setFilter] = useState<ArenaDuelFilter>("OPEN");
  const fetcher = useCallback(() => listArenaDuels(filter), [filter]);
  const { data: duels, error: loadError, load } = useLoad(fetcher, "falha ao carregar os X1");
  const { busy, error, run } = useRunner();
  const [action, setAction] = useState<{ id: string; kind: "resolve" | "annul" } | null>(null);

  const act = async (id: string, work: () => Promise<unknown>) => {
    if (await run(id, work)) {
      setAction(null);
      await load();
      onChanged();
    }
  };

  const ask = (id: string, kind: "resolve" | "annul") =>
    setAction((cur) => (cur && cur.id === id && cur.kind === kind ? null : { id, kind }));

  return (
    <div className="space-y-4">
      <ToggleGroup
        value={filter}
        onChange={(v) => {
          if (isDuelFilter(v)) {
            setFilter(v);
          }
        }}
        options={DUEL_FILTERS.map((f) => ({ value: f, label: duelFilterLabels[f] }))}
      />
      {(error || loadError) && <ErrorBox>{error ?? loadError}</ErrorBox>}
      {!duels ? (
        !loadError && <EmptyState>Carregando X1…</EmptyState>
      ) : duels.length === 0 ? (
        <EmptyState>Nenhum X1 em “{duelFilterLabels[filter]}”.</EmptyState>
      ) : (
        <div className="sticker sticker-sm overflow-x-auto rounded-[12px] bg-panel">
          <Table className="min-w-[1000px]">
            <TableHeader>
              <TableRow className="hover:bg-transparent">
                <TableHead>Jogadores</TableHead>
                <TableHead>Torneio</TableHead>
                <TableHead className="text-right">Valores</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Relatos</TableHead>
                <TableHead>Criado</TableHead>
                <TableHead />
              </TableRow>
            </TableHeader>
            <TableBody>
              {duels.map((d) => (
                <DuelRow
                  key={d.id}
                  duel={d}
                  busy={busy === d.id}
                  action={action && action.id === d.id ? action.kind : null}
                  onAsk={(kind) => ask(d.id, kind)}
                  onCancel={() => setAction(null)}
                  onResolve={(outcome, reason) => act(d.id, () => resolveArenaDuel(d.id, outcome, reason))}
                  onAnnul={(reason) => act(d.id, () => annulArenaDuel(d.id, reason))}
                />
              ))}
            </TableBody>
          </Table>
        </div>
      )}
    </div>
  );
}

function DuelRow({
  duel,
  busy,
  action,
  onAsk,
  onCancel,
  onResolve,
  onAnnul,
}: {
  duel: ArenaDuel;
  busy: boolean;
  action: "resolve" | "annul" | null;
  onAsk: (kind: "resolve" | "annul") => void;
  onCancel: () => void;
  onResolve: (outcome: ArenaOutcome, reason: string) => void;
  onAnnul: (reason: string) => void;
}) {
  const [outcome, setOutcome] = useState("");
  const decidable = DECIDABLE_DUEL.includes(duel.status);
  const open = OPEN_DUEL.includes(duel.status);
  return (
    <>
      <TableRow>
        <TableCell>
          <div className="whitespace-nowrap font-medium text-slate-100">
            {playerLabel(duel.challenger)} <span className="text-slate-500">x</span> {playerLabel(duel.opponent)}
          </div>
          <div className="text-[10px] text-slate-500">
            @{duel.challenger.username} desafiou @{duel.opponent.username}
          </div>
        </TableCell>
        <TableCell>
          <div className="text-xs text-slate-200">{duel.event.name}</div>
          <div className="text-[10px] text-slate-500">{duel.event.game.name}</div>
        </TableCell>
        <TableCell className="whitespace-nowrap text-right tabular-nums text-slate-200">
          {points(duel.challengerStake)} x {points(duel.opponentStake)}
          {duel.feeBurned > 0 && <div className="text-[10px] text-slate-500">{collectas(duel.feeBurned)} queimados</div>}
        </TableCell>
        <TableCell>
          <Badge variant={duelStatusVariants[duel.status]}>{duelStatusLabels[duel.status]}</Badge>
          {duel.outcome && <div className="mt-1 text-[11px] text-slate-300">{outcomeLabel(duel, duel.outcome)}</div>}
          {duel.resolvedBy && (
            <div className="mt-1 max-w-56 text-[10px] text-slate-500">
              decidido por {duel.resolvedBy}
              {duel.resolveReason ? `: ${duel.resolveReason}` : ""}
            </div>
          )}
        </TableCell>
        <TableCell className="text-[11px] text-slate-400">
          <div className="whitespace-nowrap">
            {playerLabel(duel.challenger)}: {duel.challengerReport ? reportLabels[duel.challengerReport] : "—"}
          </div>
          <div className="whitespace-nowrap">
            {playerLabel(duel.opponent)}: {duel.opponentReport ? reportLabels[duel.opponentReport] : "—"}
          </div>
        </TableCell>
        <TableCell className="whitespace-nowrap text-[11px] text-slate-400">{fullStamp(duel.createdAt)}</TableCell>
        <TableCell className="text-right">
          <div className="flex items-center justify-end gap-1">
            {decidable && (
              <Button variant="primary" size="sm" disabled={busy} onClick={() => onAsk("resolve")}>
                <Gavel /> Decidir
              </Button>
            )}
            {open && (
              <Button variant="ghost" size="sm" disabled={busy} onClick={() => onAsk("annul")}>
                <Ban /> Anular
              </Button>
            )}
          </div>
        </TableCell>
      </TableRow>
      {action && (
        <TableRow className="bg-raised/30 hover:bg-raised/30">
          <TableCell colSpan={8} className="p-4">
            {action === "resolve" ? (
              <ReasonForm
                label="Motivo da decisão"
                confirm="Decidir X1"
                busy={busy}
                disabled={!isOutcome(outcome)}
                onSubmit={(reason) => {
                  if (isOutcome(outcome)) {
                    onResolve(outcome, reason);
                  }
                }}
                onCancel={onCancel}
              >
                <Field label="Resultado">
                  <div className="w-64">
                    <Select value={outcome} onChange={(e) => setOutcome(e.target.value)} disabled={busy}>
                      <option value="">Escolha o resultado</option>
                      {OUTCOMES.map((o) => (
                        <option key={o} value={o}>
                          {outcomeLabel(duel, o)}
                        </option>
                      ))}
                    </Select>
                  </div>
                </Field>
              </ReasonForm>
            ) : (
              <>
                <p className="mb-2 text-[11px] text-amber-300">
                  Anular cancela o X1 e devolve os Collectas apostados para os dois jogadores.
                </p>
                <ReasonForm label="Motivo da anulação" confirm="Anular X1" busy={busy} onSubmit={onAnnul} onCancel={onCancel} />
              </>
            )}
          </TableCell>
        </TableRow>
      )}
    </>
  );
}

function ReasonForm({
  label,
  confirm,
  busy,
  disabled,
  onSubmit,
  onCancel,
  children,
}: {
  label: string;
  confirm: string;
  busy: boolean;
  disabled?: boolean;
  onSubmit: (reason: string) => void;
  onCancel: () => void;
  children?: ReactNode;
}) {
  const [reason, setReason] = useState("");
  const ready = validReason(reason) && !disabled && !busy;
  return (
    <div className="flex flex-wrap items-end gap-2">
      {children}
      <Field label={label}>
        <Input
          value={reason}
          onChange={(e) => setReason(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter" && ready) {
              onSubmit(reason.trim());
            }
          }}
          maxLength={200}
          placeholder="de 3 a 200 letras"
          className="w-80"
          autoFocus
        />
      </Field>
      <Button variant="primary" size="sm" disabled={!ready} onClick={() => onSubmit(reason.trim())}>
        {busy ? "Salvando…" : confirm}
      </Button>
      <Button variant="ghost" size="sm" disabled={busy} onClick={onCancel}>
        <X /> Cancelar
      </Button>
    </div>
  );
}

function PlayerName({ player, minor }: { player: ArenaPlayer; minor?: boolean }) {
  return (
    <div className="min-w-0">
      <div className="flex items-center gap-2 truncate font-medium text-slate-100">
        {playerLabel(player)}
        {minor && <Badge variant="sky">Menor</Badge>}
      </div>
      <div className="truncate text-[10px] text-slate-500">@{player.username}</div>
    </div>
  );
}

function ErrorBox({ children }: { children: ReactNode }) {
  return (
    <div className="rounded-[14px] border-2 border-rose-900/50 bg-rose-950/30 px-4 py-3 text-sm text-rose-200">{children}</div>
  );
}

const COLLECTA_PUBLIC = "https://api-production-93074.up.railway.app";

const GROUP_TEXT = `Chegou a Arena Collecta: o programa de pontos dos torneios da loja.

Como entrar:
1. Baixe o Collecta e crie sua conta: ${COLLECTA_PUBLIC}/baixar
2. Na tela inicial, toque em Arena Collecta e depois em Entrar na Arena. A loja aprova e você recebe um aviso.
3. No torneio, escaneie o QR do balcão e digite o código que a loja mostra: 1.000 Collectas por presença e mais 5.000 no seu primeiro torneio.
4. Chame alguém pro X1 e coloque seus Collectas em jogo.

Com 35.000 Collectas você paga a inscrição de um torneio.`;

interface ArenaArt {
  src: string;
  filename: string;
  title: string;
  hint: string;
  text: string;
}

const ARTS: ArenaArt[] = [
  {
    src: "/arena/como-entrar.png",
    filename: "arena-collecta-como-entrar.png",
    title: "Passo a passo para os grupos",
    hint: "Formato story (1080×1920). Mande nos grupos junto com o texto.",
    text: GROUP_TEXT,
  },
  {
    src: "/arena/qr-balcao.png",
    filename: "arena-collecta-qr-balcao.png",
    title: "QR do balcão",
    hint: "Folha A4 para imprimir. O QR abre o check-in no app e, para quem não tem o app, leva pra App Store e Google Play.",
    text: `Check-in da Arena Collecta: ${COLLECTA_PUBLIC}/a/checkin`,
  },
];

function ArtsSection() {
  return (
    <div className="grid gap-4 lg:grid-cols-2">
      {ARTS.map((art) => (
        <ArtCard key={art.src} art={art} />
      ))}
    </div>
  );
}

function ArtCard({ art }: { art: ArenaArt }) {
  const [busy, setBusy] = useState(false);
  const [note, setNote] = useState<string | null>(null);

  const flash = (message: string) => {
    setNote(message);
    window.setTimeout(() => setNote(null), 2500);
  };

  const copyText = async () => {
    try {
      await navigator.clipboard.writeText(art.text);
      flash("Texto copiado.");
    } catch {
      flash("Não deu para copiar o texto.");
    }
  };

  const share = async () => {
    setBusy(true);
    try {
      const blob = await (await fetch(art.src)).blob();
      const file = new File([blob], art.filename, { type: "image/png" });
      if (navigator.canShare?.({ files: [file] })) {
        await navigator.share({ files: [file], text: art.text });
        return;
      }
      if (typeof ClipboardItem !== "undefined") {
        await navigator.clipboard.write([new ClipboardItem({ "image/png": blob })]);
        flash("Imagem copiada. Cole no WhatsApp e depois copie o texto.");
        return;
      }
      flash("Este navegador não compartilha imagens. Use Baixar.");
    } catch (err) {
      if (!(err instanceof Error && err.name === "AbortError")) {
        flash("Não deu para compartilhar. Use Baixar.");
      }
    } finally {
      setBusy(false);
    }
  };

  return (
    <Card className="flex flex-col gap-4 p-4">
      <div>
        <h3 className="font-display text-sm font-bold text-fg">{art.title}</h3>
        <p className="mt-0.5 text-xs text-slate-400">{art.hint}</p>
      </div>
      <a href={art.src} target="_blank" rel="noopener" className="block self-center">
        <img src={art.src} alt={art.title} className="max-h-[520px] rounded-[10px] border-2 border-outline" />
      </a>
      <pre className="max-h-48 overflow-auto whitespace-pre-wrap rounded-[10px] border-2 border-outline bg-panel p-3 font-sans text-xs text-slate-300">
        {art.text}
      </pre>
      <div className="flex flex-wrap items-center gap-2">
        <Button variant="primary" size="sm" onClick={share} disabled={busy}>
          <Send /> {busy ? "Preparando…" : "Compartilhar"}
        </Button>
        <Button variant="outline" size="sm" onClick={() => saveArt(art)}>
          <Download /> Baixar
        </Button>
        <Button variant="outline" size="sm" onClick={copyText}>
          <Copy /> Copiar texto
        </Button>
        {note && <span className="text-xs text-emerald-300">{note}</span>}
      </div>
    </Card>
  );
}

function saveArt(art: ArenaArt) {
  const link = document.createElement("a");
  link.href = art.src;
  link.download = art.filename;
  link.click();
}

function SectionTitle({ children }: { children: ReactNode }) {
  return <h3 className="font-pixel text-[8px] uppercase text-brand-label">{children}</h3>;
}

function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <label className="flex flex-col gap-1">
      <span className="text-[10px] font-medium uppercase tracking-wide text-slate-500">{label}</span>
      {children}
    </label>
  );
}
