import { useCallback, useEffect, useMemo, useState } from "react";
import {
  Bot,
  Check,
  Coins,
  ExternalLink,
  FilePen,
  ImageOff,
  MessageCircle,
  Package,
  Pencil,
  RefreshCw,
  Reply,
  Send,
  Trash2,
  UserRound,
  X,
} from "lucide-react";
import {
  approveConversation,
  closeConversation,
  discardConversationDraft,
  listConversations,
  releaseConversation,
  replyConversation,
  type Conversation,
  type ConversationItem,
  type ConversationMessage,
  type ConversationMetrics,
  type ConversationOffer,
  type ConversationOutcome,
  type ConversationState,
  type ConversationTransport,
} from "../api";
import { brl2, fullStamp, hasTime, spClock, timeAgo, timeLeft } from "../format";
import { cn } from "../lib/utils";
import EmptyState from "./EmptyState";
import { Badge, type BadgeProps } from "./ui/badge";
import { Button } from "./ui/button";
import { Input } from "./ui/input";
import { Kpi } from "./ui/kpi";
import { ToggleGroup } from "./ui/toggle-group";

type Tab = "pending" | "active" | "closed";

const TABS: Tab[] = ["pending", "active", "closed"];

const tabLabels: Record<Tab, string> = {
  pending: "Pendentes",
  active: "Ativas",
  closed: "Encerradas",
};

const POLL_MS = 20_000;

const PITCH_MAX = 300;

const OPENING_MAX = 600;

const AGENT_STALE_MS = 15 * 60_000;

const CLAIM_LEASE_MS = 10 * 60_000;

const stateLabels: Record<ConversationState, string> = {
  compose: "IA escrevendo",
  draft: "Rascunho",
  queued: "Na fila",
  waiting: "Aguardando cliente",
  active: "IA conversando",
  human: "Com você",
  closed: "Encerrada",
};

const stateVariants: Record<ConversationState, BadgeProps["variant"]> = {
  compose: "violet",
  draft: "amber",
  queued: "sky",
  waiting: "default",
  active: "emerald",
  human: "rose",
  closed: "default",
};

const outcomeLabels: Record<Exclude<ConversationOutcome, "">, string> = {
  declined: "cliente recusou",
  expired: "janela expirou",
  complemented: "complementou",
  skipped: "oferta descartada",
  failed: "falhou",
};

const emptyMetrics: ConversationMetrics = {
  offersSent: 0,
  replies: 0,
  complements: 0,
  revenueBRL: 0,
  pendingDrafts: 0,
};

function isTab(value: string): value is Tab {
  return TABS.some((t) => t === value);
}

function pendingDraft(c: Conversation): ConversationMessage | null {
  for (let i = c.messages.length - 1; i >= 0; i -= 1) {
    if (c.messages[i].status === "draft") {
      return c.messages[i];
    }
  }
  return null;
}

function isOpeningDraft(c: Conversation): boolean {
  return c.state === "draft" && pendingDraft(c) === null;
}

function hasDraft(c: Conversation): boolean {
  return c.state === "draft" || pendingDraft(c) !== null;
}

function tabOf(c: Conversation): Tab {
  if (hasDraft(c)) {
    return "pending";
  }
  return c.state === "closed" ? "closed" : "active";
}

function oneLine(text: string): string {
  return text.replace(/[\r\n]+/g, " ").slice(0, PITCH_MAX);
}

function phoneLabel(digits: string): string {
  const local = digits.length === 13 && digits.startsWith("55") ? digits.slice(2) : digits;
  if (local.length === 11) {
    return `(${local.slice(0, 2)}) ${local.slice(2, 7)}-${local.slice(7)}`;
  }
  if (local.length === 10) {
    return `(${local.slice(0, 2)}) ${local.slice(2, 6)}-${local.slice(6)}`;
  }
  return digits;
}

function whatsappURL(digits: string): string {
  return `https://wa.me/${digits.startsWith("55") && digits.length > 11 ? digits : `55${digits}`}`;
}

function isLiga(c: Conversation): boolean {
  return c.channel === "liga";
}

function orderLabel(c: Conversation): string {
  return isLiga(c) ? `Liga #${c.orderNumber}` : `pedido #${c.orderNumber}`;
}

function lastText(c: Conversation): string {
  const last = c.messages[c.messages.length - 1];
  if (last) {
    return last.text;
  }
  if (c.state === "compose") {
    return "a IA vai escrever a primeira mensagem";
  }
  return c.opening || c.pitch;
}

function ChannelBadges({ conversation: c }: { conversation: Conversation }) {
  return (
    <>
      <Badge variant={isLiga(c) ? "amber" : "sky"}>{isLiga(c) ? "Liga" : "Site"}</Badge>
      {c.cancelled && <Badge variant="rose">Cancelado</Badge>}
      {c.pickup && (
        <Badge variant="emerald">
          <Package className="h-3 w-3" /> Retirada
        </Badge>
      )}
    </>
  );
}

function errorText(err: unknown, fallback: string): string {
  return err instanceof Error ? err.message : fallback;
}

function agentStale(seenAt: string, now: number): boolean {
  return !hasTime(seenAt) || now - new Date(seenAt).getTime() > AGENT_STALE_MS;
}

function hasQueuedOffer(c: Conversation): boolean {
  return c.messages.some((m) => m.status === "queued" && m.kind === "offer");
}

function deliveryNote(m: ConversationMessage, openBot: boolean, now: number): string {
  if (m.status === "queued") {
    if (!openBot) {
      return "na fila";
    }
    const claimed = m.claimedAt && hasTime(m.claimedAt) && now - new Date(m.claimedAt).getTime() < CLAIM_LEASE_MS;
    return claimed ? "OpenBot enviando" : "aguardando OpenBot";
  }
  if (openBot && m.status === "sent" && m.role !== "customer") {
    return "entregue";
  }
  return "";
}

export default function ConversasPage({ onOpenOrder }: { onOpenOrder: () => void }) {
  const [conversations, setConversations] = useState<Conversation[]>([]);
  const [metrics, setMetrics] = useState<ConversationMetrics>(emptyMetrics);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [picked, setPicked] = useState<Tab | null>(null);
  const [selected, setSelected] = useState<string | null>(null);
  const [now, setNow] = useState(() => Date.now());
  const [transport, setTransport] = useState<ConversationTransport>("cloud");
  const [agentSeenAt, setAgentSeenAt] = useState("");

  const load = useCallback(async () => {
    setRefreshing(true);
    try {
      const r = await listConversations();
      setConversations(r.conversations ?? []);
      setMetrics(r.metrics ?? emptyMetrics);
      setTransport(r.transport ?? "cloud");
      setAgentSeenAt(r.agentSeenAt ?? "");
      setError(null);
    } catch (err: unknown) {
      setError(errorText(err, "falha ao carregar as conversas"));
    } finally {
      setNow(Date.now());
      setLoading(false);
      setRefreshing(false);
    }
  }, []);

  useEffect(() => {
    load();
    const id = window.setInterval(load, POLL_MS);
    return () => window.clearInterval(id);
  }, [load]);

  const counts = useMemo(() => {
    const byTab: Record<Tab, number> = { pending: 0, active: 0, closed: 0 };
    for (const c of conversations) {
      byTab[tabOf(c)] += 1;
    }
    return byTab;
  }, [conversations]);

  const tab: Tab = picked ?? (counts.pending > 0 ? "pending" : "active");

  const visible = useMemo(() => conversations.filter((c) => tabOf(c) === tab), [conversations, tab]);

  const current = conversations.find((c) => c.id === selected) ?? visible[0] ?? null;

  const replace = useCallback(
    (updated: Conversation) => {
      setConversations((prev) => prev.map((c) => (c.id === updated.id ? updated : c)));
      setSelected(updated.id);
      load();
    },
    [load],
  );

  return (
    <div className="space-y-6">
      <div className="grid grid-cols-2 gap-3 lg:grid-cols-5">
        <Kpi icon={<Send className="h-5 w-5" />} label="Ofertas enviadas" value={String(metrics.offersSent)} sub="pós-venda no WhatsApp" />
        <Kpi icon={<Reply className="h-5 w-5" />} label="Respostas" value={String(metrics.replies)} sub="clientes que responderam" />
        <Kpi icon={<Check className="h-5 w-5" />} label="Complementos" value={String(metrics.complements)} sub="pedidos extras fechados" />
        <Kpi icon={<Coins className="h-5 w-5" />} label="Receita extra" value={brl2(metrics.revenueBRL)} sub="dos complementos" />
        <Kpi icon={<FilePen className="h-5 w-5" />} label="Rascunhos pendentes" value={String(metrics.pendingDrafts)} sub="esperando sua aprovação" />
      </div>

      <div className="flex flex-wrap items-center justify-between gap-3">
        <ToggleGroup
          value={tab}
          onChange={(v) => {
            if (isTab(v)) {
              setPicked(v);
              setSelected(null);
            }
          }}
          options={TABS.map((t) => ({ value: t, label: `${tabLabels[t]} · ${counts[t]}` }))}
        />
        <div className="flex flex-wrap items-center gap-2">
          {transport === "openbot" && (
            <Badge variant={agentStale(agentSeenAt, now) ? "rose" : "emerald"}>
              <Bot className="h-3 w-3" /> OpenBot visto {hasTime(agentSeenAt) ? timeAgo(agentSeenAt) : "nunca"}
            </Badge>
          )}
          <Button variant="outline" size="sm" onClick={load} disabled={refreshing}>
            <RefreshCw className={refreshing ? "animate-spin" : ""} /> Atualizar
          </Button>
        </div>
      </div>

      {error && (
        <div className="rounded-[14px] border-2 border-rose-900/50 bg-rose-950/30 px-4 py-3 text-sm text-rose-200">
          {error}
        </div>
      )}

      {loading ? (
        <EmptyState>Carregando conversas…</EmptyState>
      ) : conversations.length === 0 ? (
        <EmptyState hint="aparecem quando um pedido é pago">Nenhuma conversa de pós-venda ainda.</EmptyState>
      ) : (
        <div className="grid gap-4 lg:grid-cols-[360px_minmax(0,1fr)]">
          <div className="sticker sticker-sm max-h-[75vh] overflow-y-auto rounded-[12px] bg-panel">
            {visible.length === 0 ? (
              <p className="px-4 py-8 text-center text-sm text-slate-500">Nenhuma conversa em “{tabLabels[tab]}”.</p>
            ) : (
              visible.map((c) => (
                <ConversationItem
                  key={c.id}
                  conversation={c}
                  active={current?.id === c.id}
                  onSelect={() => setSelected(c.id)}
                />
              ))
            )}
          </div>
          {current ? (
            <ConversationDetail
              key={current.id}
              conversation={current}
              now={now}
              openBot={transport === "openbot"}
              onChanged={replace}
              onOpenOrder={onOpenOrder}
            />
          ) : (
            <EmptyState>Escolha uma conversa.</EmptyState>
          )}
        </div>
      )}
    </div>
  );
}

function ConversationItem({
  conversation: c,
  active,
  onSelect,
}: {
  conversation: Conversation;
  active: boolean;
  onSelect: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onSelect}
      className={cn(
        "block w-full border-b-2 border-outline/20 px-4 py-3 text-left transition-colors last:border-0",
        active ? "bg-raised" : "hover:bg-raised/50",
      )}
    >
      <div className="flex items-center justify-between gap-2">
        <span className="truncate font-semibold text-slate-100">{c.name || phoneLabel(c.phone)}</span>
        <span className="shrink-0 text-[10px] tabular-nums text-slate-500">{timeAgo(c.updatedAt)}</span>
      </div>
      <div className="mt-1 flex flex-wrap items-center gap-1.5">
        <Badge variant={stateVariants[c.state]}>{stateLabels[c.state]}</Badge>
        <ChannelBadges conversation={c} />
        {hasDraft(c) && c.state !== "draft" && <Badge variant="amber">Rascunho</Badge>}
        {c.orderNumber > 0 && <span className="text-[10px] tabular-nums text-slate-500">{orderLabel(c)}</span>}
      </div>
      <p className="mt-1 truncate text-xs text-slate-400">{lastText(c) || "—"}</p>
    </button>
  );
}

function ConversationDetail({
  conversation: c,
  now,
  openBot,
  onChanged,
  onOpenOrder,
}: {
  conversation: Conversation;
  now: number;
  openBot: boolean;
  onChanged: (c: Conversation) => void;
  onOpenOrder: () => void;
}) {
  const [busy, setBusy] = useState(false);
  const [failed, setFailed] = useState<string | null>(null);
  const [reply, setReply] = useState("");

  const run = async (call: () => Promise<Conversation>): Promise<boolean> => {
    setBusy(true);
    setFailed(null);
    try {
      onChanged(await call());
      return true;
    } catch (err: unknown) {
      setFailed(errorText(err, "falha ao salvar"));
      return false;
    } finally {
      setBusy(false);
    }
  };

  const draft = pendingDraft(c);
  const opening = isOpeningDraft(c);
  const closed = c.state === "closed";
  const windowOpen = hasTime(c.windowEndsAt) && new Date(c.windowEndsAt).getTime() > now;

  const send = async () => {
    const text = reply.trim();
    if (!text) {
      return;
    }
    if (await run(() => replyConversation(c.id, text))) {
      setReply("");
    }
  };

  const close = () => {
    if (window.confirm(`Encerrar a conversa com ${c.name || phoneLabel(c.phone)}? A IA para de responder.`)) {
      run(() => closeConversation(c.id));
    }
  };

  const discardOpening = () => {
    if (window.confirm("Descartar a mensagem? A conversa é encerrada e nada é enviado ao cliente.")) {
      run(() => discardConversationDraft(c.id));
    }
  };

  return (
    <div className="sticker sticker-sm flex min-w-0 flex-col rounded-[12px] bg-panel">
      <header className="space-y-2 border-b-2 border-outline/30 p-4">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <div className="min-w-0">
            <div className="font-display truncate text-lg font-extrabold text-fg">{c.name || phoneLabel(c.phone)}</div>
            <a
              href={whatsappURL(c.phone)}
              target="_blank"
              rel="noreferrer"
              className="inline-flex items-center gap-1 text-xs text-emerald-300 hover:underline"
            >
              {phoneLabel(c.phone)} <ExternalLink className="h-3 w-3" />
            </a>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <ChannelBadges conversation={c} />
            <Badge variant={stateVariants[c.state]}>{stateLabels[c.state]}</Badge>
            {c.outcome && <Badge variant={c.outcome === "complemented" ? "emerald" : c.outcome === "failed" ? "rose" : "default"}>{outcomeLabels[c.outcome]}</Badge>}
          </div>
        </div>
        <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-[11px] text-slate-400">
          {c.orderNumber > 0 &&
            (isLiga(c) ? (
              <span className="font-semibold text-slate-200">Pedido Liga #{c.orderNumber}</span>
            ) : (
              <button type="button" onClick={onOpenOrder} className="font-semibold text-slate-200 hover:underline">
                Pedido #{c.orderNumber}
              </button>
            ))}
          {windowOpen && (
            <span className="text-amber-300">
              janela até {spClock(c.windowEndsAt, now)} · faltam {timeLeft(c.windowEndsAt, now)}
            </span>
          )}
          {c.state === "queued" &&
            (openBot && hasQueuedOffer(c) ? (
              <span className="text-sky-300">aguardando o OpenBot enviar (até 3 min)</span>
            ) : (
              <span>sai às 8h (fora do horário de silêncio)</span>
            ))}
          {hasTime(c.sentAt) && <span>oferta enviada {fullStamp(c.sentAt)}</span>}
          {hasTime(c.lastInboundAt) && <span>última resposta {fullStamp(c.lastInboundAt)}</span>}
          {c.complementOrder > 0 && (
            <span className="text-emerald-300">
              complemento: pedido #{c.complementOrder} · {brl2(c.complementBRL)}
            </span>
          )}
        </div>
      </header>

      <div className="space-y-4 p-4">
        {c.offer && !opening && <OfferCard offer={c.offer} />}

        {c.items && c.items.length > 0 && <OrderItems items={c.items} />}

        {c.state === "compose" && (
          <div className="space-y-2 rounded-[10px] border-2 border-dashed border-outline/50 bg-raised/30 p-3">
            <p className="text-sm text-slate-300">
              A IA do OpenBot vai escrever a primeira mensagem na próxima rodada. Depois ela aparece aqui para você aprovar.
            </p>
            <Button variant="ghost" size="sm" disabled={busy} onClick={discardOpening}>
              <Trash2 className="h-4 w-4" /> Descartar
            </Button>
          </div>
        )}

        {opening && (
          <OpeningDraft
            conversation={c}
            busy={busy}
            onApprove={(text) => run(() => approveConversation(c.id, text))}
            onDiscard={discardOpening}
          />
        )}

        {c.messages.length > 0 && (
          <div className="space-y-2">
            {c.messages.map((m) => (
              <MessageBubble
                key={m.id}
                message={m}
                note={deliveryNote(m, openBot, now)}
                pending={draft?.id === m.id}
                busy={busy}
                onApprove={(text) => run(() => approveConversation(c.id, text))}
                onDiscard={() => run(() => discardConversationDraft(c.id))}
              />
            ))}
          </div>
        )}

        {failed && <p className="text-[11px] text-rose-300">{failed}</p>}
      </div>

      {!closed && (
        <footer className="mt-auto space-y-2 border-t-2 border-outline/30 p-4">
          <textarea
            value={reply}
            onChange={(e) => setReply(e.target.value)}
            rows={2}
            disabled={busy}
            placeholder={
              openBot
                ? "Escreva para o cliente: o OpenBot envia pelo WhatsApp em até 3 min e a IA pausa"
                : "Escreva para o cliente — a IA pausa e a conversa passa para você"
            }
            className="flex w-full rounded-[8px] border-2 border-outline bg-panel px-3 py-2 text-sm text-fg transition-colors placeholder:text-slate-500 disabled:cursor-not-allowed disabled:opacity-50"
          />
          <div className="flex flex-wrap items-center gap-2">
            <Button variant="primary" size="sm" disabled={busy || !reply.trim()} onClick={send}>
              <Send className="h-4 w-4" /> Responder
            </Button>
            {c.state === "human" && (
              <Button variant="outline" size="sm" disabled={busy} onClick={() => run(() => releaseConversation(c.id))}>
                <Bot className="h-4 w-4" /> Devolver à IA
              </Button>
            )}
            <Button variant="ghost" size="sm" disabled={busy} onClick={close} className="ml-auto">
              <X className="h-4 w-4" /> Encerrar
            </Button>
          </div>
        </footer>
      )}
    </div>
  );
}

function OfferCard({ offer }: { offer: ConversationOffer }) {
  return (
    <div className="flex items-center gap-3 rounded-[10px] border-2 border-outline/40 bg-raised/40 p-3">
      <OfferArt src={offer.imageURL} alt={offer.name} />
      <div className="min-w-0">
        <div className="font-pixel text-[8px] uppercase text-brand-label">Oferta</div>
        <div className="font-display truncate text-base font-bold text-slate-100">{offer.name}</div>
        {offer.meta && <div className="truncate text-xs text-slate-500">{offer.meta}</div>}
        <div className="mt-1 text-sm font-semibold tabular-nums text-slate-200">{brl2(offer.unitBRL)}</div>
      </div>
    </div>
  );
}

function OrderItems({ items }: { items: ConversationItem[] }) {
  return (
    <div className="rounded-[10px] border-2 border-outline/40 bg-raised/40 p-3">
      <div className="font-pixel text-[8px] uppercase text-brand-label">Itens do pedido</div>
      <ul className="mt-1 space-y-0.5 text-sm text-slate-200">
        {items.map((it, i) => (
          <li key={`${it.name}-${i}`} className="truncate">
            {it.qty}x {it.name}
          </li>
        ))}
      </ul>
    </div>
  );
}

function OfferArt({ src, alt }: { src: string; alt: string }) {
  const [failed, setFailed] = useState(false);
  if (!src || failed) {
    return (
      <div className="flex h-20 w-14 shrink-0 items-center justify-center rounded-[6px] border-2 border-outline/40 bg-panel" aria-hidden>
        <ImageOff className="h-5 w-5 text-slate-500" />
      </div>
    );
  }
  return (
    <img
      src={src}
      alt={alt}
      width={56}
      height={80}
      loading="lazy"
      onError={() => setFailed(true)}
      className="h-20 w-14 shrink-0 rounded-[6px] border-2 border-outline object-cover"
    />
  );
}

function OpeningDraft({
  conversation: c,
  busy,
  onApprove,
  onDiscard,
}: {
  conversation: Conversation;
  busy: boolean;
  onApprove: (text?: string) => Promise<boolean>;
  onDiscard: () => void;
}) {
  const composed = Boolean(c.opening);
  const original = c.opening || c.pitch;
  const [editing, setEditing] = useState(false);
  const [pitch, setPitch] = useState(original);

  const save = async () => {
    const text = composed ? pitch.trim() : oneLine(pitch).trim();
    if (await onApprove(text === original ? undefined : text)) {
      setEditing(false);
    }
  };

  return (
    <div className="space-y-3 rounded-[10px] border-2 border-dashed border-amber-400/70 bg-amber-500/5 p-3">
      <div className="font-pixel text-[8px] uppercase text-amber-300">
        {composed ? "Mensagem da IA" : "Oferta de abertura"} · aguardando aprovação
      </div>
      {c.offer && <OfferCard offer={c.offer} />}
      {editing && composed ? (
        <div className="space-y-1">
          <textarea
            value={pitch}
            onChange={(e) => setPitch(e.target.value)}
            rows={4}
            maxLength={OPENING_MAX}
            disabled={busy}
            autoFocus
            className="flex w-full rounded-[8px] border-2 border-outline bg-panel px-3 py-2 text-sm text-fg disabled:opacity-50"
          />
          <div className="text-right text-[10px] tabular-nums text-slate-500">
            {pitch.length}/{OPENING_MAX}
          </div>
        </div>
      ) : editing ? (
        <div className="space-y-1">
          <Input
            value={pitch}
            onChange={(e) => setPitch(oneLine(e.target.value))}
            onKeyDown={(e) => {
              if (e.key === "Enter" && pitch.trim()) {
                save();
              }
            }}
            maxLength={PITCH_MAX}
            disabled={busy}
            autoFocus
          />
          <div className="text-right text-[10px] tabular-nums text-slate-500">
            {pitch.length}/{PITCH_MAX}
          </div>
        </div>
      ) : (
        <p className="whitespace-pre-line text-sm text-slate-200">{original || "—"}</p>
      )}
      <DraftActions
        editing={editing}
        busy={busy}
        canSave={Boolean(pitch.trim())}
        onApprove={() => onApprove()}
        onEdit={() => {
          setPitch(original);
          setEditing(true);
        }}
        onSave={save}
        onCancel={() => setEditing(false)}
        onDiscard={onDiscard}
      />
    </div>
  );
}

function MessageBubble({
  message: m,
  note,
  pending,
  busy,
  onApprove,
  onDiscard,
}: {
  message: ConversationMessage;
  note: string;
  pending: boolean;
  busy: boolean;
  onApprove: (text?: string) => Promise<boolean>;
  onDiscard: () => void;
}) {
  const [editing, setEditing] = useState(false);
  const [text, setText] = useState(m.text);

  if (m.role === "system") {
    return <p className="py-1 text-center text-[11px] text-slate-500">{m.text}</p>;
  }

  const customer = m.role === "customer";
  const draft = m.status === "draft";

  const save = async () => {
    const value = text.trim();
    if (await onApprove(value === m.text ? undefined : value)) {
      setEditing(false);
    }
  };

  return (
    <div className={cn("flex", customer ? "justify-start" : "justify-end")}>
      <div
        className={cn(
          "max-w-[80%] space-y-1 rounded-[12px] border-2 px-3 py-2 text-sm",
          customer && "border-outline bg-raised text-fg",
          m.role === "assistant" && !draft && "border-outline bg-brand-soft text-on-soft",
          m.role === "owner" && !draft && "border-outline bg-royal text-white",
          draft && "border-dashed border-amber-400/70 bg-amber-500/5 text-slate-200",
        )}
      >
        <div className="flex items-center gap-1 text-[10px] font-semibold opacity-80">
          {customer ? <UserRound className="h-3 w-3" /> : m.role === "assistant" ? <Bot className="h-3 w-3" /> : <MessageCircle className="h-3 w-3" />}
          {customer ? "Cliente" : m.role === "assistant" ? (draft ? "Rascunho da IA" : "IA") : "Você"}
          {hasTime(m.at) && <span className="font-normal tabular-nums">· {fullStamp(m.at)}</span>}
          {note && <span className="font-normal">· {note}</span>}
        </div>
        {editing ? (
          <textarea
            value={text}
            onChange={(e) => setText(e.target.value)}
            rows={3}
            disabled={busy}
            autoFocus
            className="flex w-full min-w-72 rounded-[8px] border-2 border-outline bg-panel px-3 py-2 text-sm text-fg disabled:opacity-50"
          />
        ) : (
          <p className="whitespace-pre-line">{m.text}</p>
        )}
        {m.status === "failed" && (
          <p className="text-[11px] text-rose-300">não enviada{m.error ? `: ${m.error}` : ""}</p>
        )}
        {pending && (
          <DraftActions
            editing={editing}
            busy={busy}
            canSave={Boolean(text.trim())}
            onApprove={() => onApprove()}
            onEdit={() => {
              setText(m.text);
              setEditing(true);
            }}
            onSave={save}
            onCancel={() => setEditing(false)}
            onDiscard={onDiscard}
          />
        )}
      </div>
    </div>
  );
}

function DraftActions({
  editing,
  busy,
  canSave,
  onApprove,
  onEdit,
  onSave,
  onCancel,
  onDiscard,
}: {
  editing: boolean;
  busy: boolean;
  canSave: boolean;
  onApprove: () => void;
  onEdit: () => void;
  onSave: () => void;
  onCancel: () => void;
  onDiscard: () => void;
}) {
  if (editing) {
    return (
      <div className="flex flex-wrap items-center gap-2 pt-1">
        <Button variant="primary" size="sm" disabled={busy || !canSave} onClick={onSave}>
          <Check className="h-4 w-4" /> Salvar e aprovar
        </Button>
        <Button variant="ghost" size="sm" disabled={busy} onClick={onCancel}>
          Cancelar
        </Button>
      </div>
    );
  }
  return (
    <div className="flex flex-wrap items-center gap-2 pt-1">
      <Button variant="primary" size="sm" disabled={busy} onClick={onApprove}>
        <Check className="h-4 w-4" /> Aprovar
      </Button>
      <Button variant="outline" size="sm" disabled={busy} onClick={onEdit}>
        <Pencil className="h-4 w-4" /> Editar
      </Button>
      <Button variant="ghost" size="sm" disabled={busy} onClick={onDiscard}>
        <Trash2 className="h-4 w-4" /> Descartar
      </Button>
    </div>
  );
}

export function useOrderFollowUps(enabled: boolean): Record<string, string> {
  const [labels, setLabels] = useState<Record<string, string>>({});

  useEffect(() => {
    if (!enabled) {
      return;
    }
    listConversations()
      .then((r) => setLabels(followUpLabels(r.conversations ?? [], Date.now())))
      .catch(() => setLabels({}));
  }, [enabled]);

  return labels;
}

const holdStates: ConversationState[] = ["queued", "waiting", "active", "human"];

function followUpLabels(conversations: Conversation[], now: number): Record<string, string> {
  const labels: Record<string, string> = {};
  for (const c of conversations) {
    if (!c.orderID || isLiga(c)) {
      continue;
    }
    if (c.state === "draft") {
      labels[c.orderID] = "pós-venda: aguardando aprovação";
      continue;
    }
    if (c.state === "compose") {
      labels[c.orderID] = "pós-venda: IA escrevendo";
      continue;
    }
    if (labels[c.orderID]) {
      continue;
    }
    const ends = new Date(c.windowEndsAt).getTime();
    if (holdStates.includes(c.state) && hasTime(c.windowEndsAt) && ends > now) {
      labels[c.orderID] = `segurar envio até ${spClock(c.windowEndsAt, now)}`;
    }
  }
  return labels;
}
