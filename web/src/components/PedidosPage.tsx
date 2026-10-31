import { useCallback, useEffect, useMemo, useState } from "react";
import {
  ChevronDown,
  ChevronRight,
  Clock,
  Coins,
  Copy,
  ExternalLink,
  ImageOff,
  KeyRound,
  PackageCheck,
  RefreshCw,
  Truck,
  X,
} from "lucide-react";
import {
  getAdminToken,
  listOrders,
  setAdminToken,
  updateOrder,
  type Order,
  type OrderPatch,
  type OrderStatus,
} from "../api";
import { brl2, fullStamp } from "../format";
import { isNewOrder } from "../orderAlerts";
import EmptyState from "./EmptyState";
import { Kpi } from "./PortfolioPage";
import { Badge, type BadgeProps } from "./ui/badge";
import { Button } from "./ui/button";
import { Card } from "./ui/card";
import { Input } from "./ui/input";
import { Select } from "./ui/select";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "./ui/table";
import { ToggleGroup } from "./ui/toggle-group";

type Filter = "all" | "paid" | "shipped" | "delivered" | "pending" | "problems" | "closed";

const filterStatuses: Record<Filter, OrderStatus[]> = {
  all: ["pending", "paid", "failed", "expired", "cancelled", "oversold", "shipped", "delivered", "refunded"],
  paid: ["paid"],
  shipped: ["shipped"],
  delivered: ["delivered"],
  pending: ["pending"],
  problems: ["oversold", "failed", "refunded"],
  closed: ["expired", "cancelled"],
};

const filterLabels: Record<Filter, string> = {
  all: "Todos",
  paid: "A enviar",
  shipped: "Enviados",
  delivered: "Entregues",
  pending: "Pendentes",
  problems: "Problemas",
  closed: "Expirados",
};

const FILTER_ORDER: Filter[] = ["all", "paid", "shipped", "delivered", "pending", "problems", "closed"];

const statusLabels: Record<OrderStatus, string> = {
  pending: "Aguardando pagamento",
  paid: "Pago",
  failed: "Pagamento falhou",
  expired: "Expirado",
  cancelled: "Cancelado",
  oversold: "Sem estoque",
  shipped: "Enviado",
  delivered: "Entregue",
  refunded: "Reembolsado",
};

const statusVariants: Record<OrderStatus, BadgeProps["variant"]> = {
  pending: "amber",
  paid: "emerald",
  failed: "rose",
  expired: "default",
  cancelled: "default",
  oversold: "rose",
  shipped: "sky",
  delivered: "default",
  refunded: "rose",
};

const manualTargets: Record<OrderStatus, OrderStatus[]> = {
  pending: ["cancelled"],
  failed: ["cancelled"],
  expired: ["cancelled"],
  oversold: ["paid", "shipped", "delivered", "refunded", "cancelled"],
  paid: ["shipped", "delivered", "refunded"],
  shipped: ["paid", "delivered", "refunded"],
  delivered: ["paid", "shipped", "refunded"],
  refunded: ["paid", "shipped", "delivered"],
  cancelled: [],
};

const fulfilled: OrderStatus[] = ["paid", "shipped", "delivered"];

function holdsStock(o: Order): boolean {
  return o.lines.some((l) => (l.tradeIDs ?? []).length > 0);
}

function isStatus(value: string): value is OrderStatus {
  return filterStatuses.all.some((s) => s === value);
}

function isFilter(value: string): value is Filter {
  return FILTER_ORDER.some((f) => f === value);
}

function hasTime(iso: string): boolean {
  return Boolean(iso) && !iso.startsWith("0001-");
}

function phoneLabel(digits: string): string {
  if (digits.length === 11) {
    return `(${digits.slice(0, 2)}) ${digits.slice(2, 7)}-${digits.slice(7)}`;
  }
  if (digits.length === 10) {
    return `(${digits.slice(0, 2)}) ${digits.slice(2, 6)}-${digits.slice(6)}`;
  }
  return digits;
}

function cepLabel(cep: string): string {
  return cep.length === 8 ? `${cep.slice(0, 5)}-${cep.slice(5)}` : cep;
}

function cpfLabel(cpf: string): string {
  return cpf.length === 11
    ? `${cpf.slice(0, 3)}.${cpf.slice(3, 6)}.${cpf.slice(6, 9)}-${cpf.slice(9)}`
    : cpf;
}

function addressLabel(o: Order): string {
  const a = o.address;
  if (!a.cep) {
    return [o.buyer.name, `${o.shipping.serviceName} · ${o.shipping.company}`, `Tel ${phoneLabel(o.buyer.phone)}`].join("\n");
  }
  const street = [a.street, a.number, a.complement].filter(Boolean).join(", ");
  return [
    o.buyer.name,
    street,
    a.district,
    `${a.city} - ${a.state}`,
    `CEP ${cepLabel(a.cep)}`,
    `Tel ${phoneLabel(o.buyer.phone)}`,
  ]
    .filter(Boolean)
    .join("\n");
}

function isUnauthorized(err: unknown): boolean {
  return err instanceof Error && err.message === "unauthorized";
}

export default function PedidosPage({ since, onChanged }: { since: number; onChanged: () => void }) {
  const [orders, setOrders] = useState<Order[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [unauthorized, setUnauthorized] = useState(false);
  const [showToken, setShowToken] = useState(false);
  const [picked, setPicked] = useState<Filter | null>(null);
  const [open, setOpen] = useState<string | null>(null);

  const load = useCallback(async () => {
    setRefreshing(true);
    try {
      setOrders(await listOrders());
      setError(null);
      setUnauthorized(false);
    } catch (err: unknown) {
      if (isUnauthorized(err)) {
        setUnauthorized(true);
        setError(null);
      } else {
        setError(err instanceof Error ? err.message : "falha ao carregar os pedidos");
      }
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const replace = useCallback(
    (updated: Order) => {
      setOrders((prev) => prev.map((o) => (o.id === updated.id ? updated : o)));
      onChanged();
    },
    [onChanged],
  );

  const counts = useMemo(() => {
    const byStatus: Record<OrderStatus, number> = {
      pending: 0,
      paid: 0,
      failed: 0,
      expired: 0,
      cancelled: 0,
      oversold: 0,
      shipped: 0,
      delivered: 0,
      refunded: 0,
    };
    let revenue = 0;
    for (const o of orders) {
      byStatus[o.status] += 1;
      if (fulfilled.includes(o.status)) {
        revenue += o.totals.totalBRL;
      }
    }
    return { byStatus, revenue };
  }, [orders]);

  const filter: Filter = picked ?? (counts.byStatus.paid > 0 ? "paid" : "all");

  const visible = useMemo(
    () => orders.filter((o) => filterStatuses[filter].includes(o.status)),
    [orders, filter],
  );

  const countOf = (f: Filter) =>
    filterStatuses[f].reduce((sum, s) => sum + counts.byStatus[s], 0);

  const tokenCard = (unauthorized || showToken) && (
    <TokenCard
      onSaved={() => {
        setShowToken(false);
        load();
      }}
      onCancel={unauthorized ? undefined : () => setShowToken(false)}
    />
  );

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <ToggleGroup
          value={filter}
          onChange={(v) => {
            if (isFilter(v)) {
              setPicked(v);
            }
          }}
          options={FILTER_ORDER.map((f) => ({
            value: f,
            label: `${filterLabels[f]} · ${countOf(f)}`,
          }))}
        />
        <div className="flex items-center gap-3">
          <button
            type="button"
            onClick={() => setShowToken((v) => !v)}
            className="inline-flex items-center gap-1 text-xs font-medium text-sky-300 hover:text-sky-200"
          >
            <KeyRound className="h-3 w-3" /> trocar token
          </button>
          <Button variant="outline" size="sm" onClick={load} disabled={refreshing}>
            <RefreshCw className={refreshing ? "animate-spin" : ""} /> Atualizar
          </Button>
        </div>
      </div>

      <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
        <Kpi
          icon={<PackageCheck className="h-5 w-5" />}
          label="Pagos para enviar"
          value={String(counts.byStatus.paid)}
          sub="aguardando postagem"
        />
        <Kpi
          icon={<Truck className="h-5 w-5" />}
          label="Enviados"
          value={String(counts.byStatus.shipped)}
          sub={`já postados · ${counts.byStatus.delivered} entregues`}
        />
        <Kpi
          icon={<Coins className="h-5 w-5" />}
          label="Receita paga"
          value={brl2(counts.revenue)}
          sub="pagos, enviados e entregues, com frete"
        />
        <Kpi
          icon={<Clock className="h-5 w-5" />}
          label="Pendentes"
          value={String(counts.byStatus.pending)}
          sub="aguardando pagamento"
        />
      </div>

      {tokenCard}

      {error && (
        <div className="rounded-[14px] border-2 border-rose-900/50 bg-rose-950/30 px-4 py-3 text-sm text-rose-200">
          {error}
        </div>
      )}

      {loading ? (
        <EmptyState>Carregando pedidos…</EmptyState>
      ) : unauthorized ? (
        <EmptyState hint="informe o token acima">Os pedidos pedem o token de administrador.</EmptyState>
      ) : orders.length === 0 ? (
        <EmptyState>Nenhum pedido pelo site ainda.</EmptyState>
      ) : visible.length === 0 ? (
        <EmptyState hint="troque o filtro">Nenhum pedido em “{filterLabels[filter]}”.</EmptyState>
      ) : (
        <div className="sticker sticker-sm overflow-x-auto rounded-[12px] bg-panel">
          <Table className="min-w-[900px]">
            <TableHeader>
              <TableRow className="hover:bg-transparent">
                <TableHead>Nº</TableHead>
                <TableHead>Data</TableHead>
                <TableHead>Comprador</TableHead>
                <TableHead>Status</TableHead>
                <TableHead className="text-right">Total</TableHead>
                <TableHead>Frete</TableHead>
                <TableHead>Rastreio</TableHead>
                <TableHead className="w-10" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {visible.map((o) => (
                <OrderRow
                  key={o.id}
                  order={o}
                  fresh={isNewOrder(o, since)}
                  open={open === o.id}
                  onToggle={() => setOpen((k) => (k === o.id ? null : o.id))}
                  onChanged={replace}
                  onUnauthorized={() => setUnauthorized(true)}
                />
              ))}
            </TableBody>
          </Table>
        </div>
      )}
    </div>
  );
}

export function TokenCard({ onSaved, onCancel }: { onSaved: () => void; onCancel?: () => void }) {
  const [token, setToken] = useState(getAdminToken);
  const submit = () => {
    setAdminToken(token.trim());
    onSaved();
  };
  return (
    <Card className="space-y-3 p-4">
      <div className="font-display text-base font-extrabold text-fg">Token de administrador</div>
      <p className="max-w-2xl text-sm text-slate-400">
        O servidor exige o token ADMIN para abrir o dashboard. Ele fica salvo só neste navegador.
      </p>
      <div className="flex flex-wrap items-end gap-2">
        <Field label="Token">
          <Input
            type="password"
            value={token}
            onChange={(e) => setToken(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter" && token.trim()) {
                submit();
              }
            }}
            className="w-72"
            autoFocus
          />
        </Field>
        <Button variant="primary" onClick={submit} disabled={!token.trim()}>
          Entrar
        </Button>
        {onCancel && (
          <Button variant="ghost" onClick={onCancel}>
            Fechar
          </Button>
        )}
      </div>
    </Card>
  );
}

function OrderRow({
  order,
  fresh,
  open,
  onToggle,
  onChanged,
  onUnauthorized,
}: {
  order: Order;
  fresh: boolean;
  open: boolean;
  onToggle: () => void;
  onChanged: (o: Order) => void;
  onUnauthorized: () => void;
}) {
  const [tracking, setTracking] = useState(order.trackingCode ?? "");
  const [notes, setNotes] = useState(order.notes ?? "");
  const [busy, setBusy] = useState(false);
  const [failed, setFailed] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);
  const [viewing, setViewing] = useState<string | null>(null);
  const [target, setTarget] = useState<OrderStatus>(order.status);

  useEffect(() => {
    setTracking(order.trackingCode ?? "");
    setNotes(order.notes ?? "");
  }, [order.trackingCode, order.notes]);

  useEffect(() => {
    setTarget(order.status);
  }, [order.status]);

  const patch = async (body: OrderPatch) => {
    setBusy(true);
    setFailed(null);
    try {
      onChanged(await updateOrder(order.id, body));
    } catch (err: unknown) {
      if (isUnauthorized(err)) {
        onUnauthorized();
      } else {
        setFailed(err instanceof Error ? err.message : "falha ao salvar");
      }
    } finally {
      setBusy(false);
    }
  };

  const copyAddress = async () => {
    await navigator.clipboard.writeText(addressLabel(order));
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1500);
  };

  const changeStatus = () => {
    if (
      (target === "cancelled" || target === "refunded") &&
      !window.confirm(
        holdsStock(order)
          ? `Mudar o pedido #${order.number} para "${statusLabels[target]}"? Os itens vendidos voltam para o estoque.`
          : `Mudar o pedido #${order.number} para "${statusLabels[target]}"?`,
      )
    ) {
      return;
    }
    const code = tracking.trim();
    patch(target === "shipped" && code !== (order.trackingCode ?? "") ? { status: target, trackingCode: code } : { status: target });
  };

  const targets = manualTargets[order.status];
  const shipped = order.status === "shipped" || order.status === "delivered";
  const canShip = order.status === "paid";
  const takesStock =
    (order.status === "oversold" || order.status === "refunded") && fulfilled.includes(target) && !holdsStock(order);
  const returnsStock = (target === "cancelled" || target === "refunded") && holdsStock(order);
  const whatsapp = `https://wa.me/55${order.buyer.phone}`;

  return (
    <>
      <TableRow className="cursor-pointer" onClick={onToggle}>
        <TableCell className="whitespace-nowrap font-semibold tabular-nums text-slate-100">
          #{order.number}
          {fresh && (
            <Badge variant="rose" className="ml-2">
              Novo
            </Badge>
          )}
        </TableCell>
        <TableCell className="whitespace-nowrap tabular-nums text-slate-400">
          {fullStamp(order.createdAt)}
        </TableCell>
        <TableCell>
          <div className="min-w-0">
            <div className="truncate font-medium text-slate-100">{order.buyer.name}</div>
            <div className="truncate text-[10px] text-slate-500">
              {order.address.city ? `${order.address.city} - ${order.address.state}` : order.shipping.serviceName}
            </div>
          </div>
        </TableCell>
        <TableCell>
          <Badge variant={statusVariants[order.status]}>{statusLabels[order.status]}</Badge>
        </TableCell>
        <TableCell className="text-right font-semibold tabular-nums text-slate-200">
          {brl2(order.totals.totalBRL)}
        </TableCell>
        <TableCell className="whitespace-nowrap text-[11px] text-slate-400">
          {[order.shipping.company, order.shipping.serviceName].filter(Boolean).join(" · ") || "—"}
        </TableCell>
        <TableCell className="font-mono text-[11px] text-slate-300">{order.trackingCode || "—"}</TableCell>
        <TableCell className="text-right">
          {open ? (
            <ChevronDown className="h-4 w-4 text-slate-500" />
          ) : (
            <ChevronRight className="h-4 w-4 text-slate-500" />
          )}
        </TableCell>
      </TableRow>
      {open && (
        <TableRow className="bg-raised/30 hover:bg-raised/30">
          <TableCell colSpan={8} className="p-4">
            <div className="grid gap-4 lg:grid-cols-3">
              <section className="space-y-2 lg:col-span-2">
                <SectionTitle>Itens</SectionTitle>
                <div className="sticker sticker-sm overflow-x-auto rounded-[10px] bg-panel">
                  <table className="w-full text-sm">
                    <tbody>
                      {order.lines.map((l) => (
                        <tr key={l.sku} className="border-b-2 border-outline/15 last:border-0">
                          <td className="w-32 px-3 py-3">
                            <LineArt src={l.imageURL} alt={l.name} onOpen={() => setViewing(l.imageURL ?? null)} />
                          </td>
                          <td className="px-3 py-2 tabular-nums text-slate-400">{l.qty}×</td>
                          <td className="px-3 py-2">
                            <div className="font-display text-base font-bold text-slate-100">{l.name}</div>
                            {l.meta && <div className="text-xs text-slate-500">{l.meta}</div>}
                          </td>
                          <td className="px-3 py-2 text-right tabular-nums text-slate-400">{brl2(l.unitBRL)}</td>
                          <td className="px-3 py-2 text-right font-semibold tabular-nums text-slate-200">
                            {brl2(l.qty * l.unitBRL)}
                          </td>
                        </tr>
                      ))}
                      <tr className="bg-raised/60">
                        <td colSpan={4} className="px-3 py-2 text-right text-[11px] text-slate-500">
                          itens {brl2(order.totals.itemsBRL)} + frete {brl2(order.totals.shippingBRL)}
                          {order.totals.discountBRL ? ` − cupom ${brl2(order.totals.discountBRL)}` : ""}
                        </td>
                        <td className="px-3 py-2 text-right font-bold tabular-nums text-slate-100">
                          {brl2(order.totals.totalBRL)}
                        </td>
                      </tr>
                    </tbody>
                  </table>
                </div>
                <dl className="grid grid-cols-2 gap-x-4 gap-y-1 text-[11px] text-slate-500 sm:grid-cols-4">
                  <Stamp label="Criado" value={order.createdAt} />
                  <Stamp label="Pago" value={order.paidAt} />
                  <Stamp label="Enviado" value={order.shippedAt} />
                  <Stamp label="Entregue" value={order.deliveredAt} />
                  <Stamp label="Estoque devolvido" value={order.stockReturnedAt ?? ""} />
                  {order.status === "pending" && <Stamp label="Expira" value={order.expiresAt} />}
                </dl>
              </section>

              <section className="space-y-3">
                <div className="space-y-2">
                  <div className="flex items-center justify-between gap-2">
                    <SectionTitle>Endereço</SectionTitle>
                    <Button variant="outline" size="sm" onClick={copyAddress}>
                      <Copy className="h-3 w-3" /> {copied ? "Copiado" : "Copiar endereço"}
                    </Button>
                  </div>
                  <p className="whitespace-pre-line text-sm text-slate-200">{addressLabel(order)}</p>
                </div>
                <div className="space-y-1">
                  <SectionTitle>Contato</SectionTitle>
                  <div className="text-sm text-slate-200">
                    <a href={`mailto:${order.buyer.email}`} className="hover:underline">
                      {order.buyer.email}
                    </a>
                  </div>
                  <div className="text-sm">
                    <a
                      href={whatsapp}
                      target="_blank"
                      rel="noreferrer"
                      className="inline-flex items-center gap-1 text-emerald-300 hover:underline"
                    >
                      {phoneLabel(order.buyer.phone)} <ExternalLink className="h-3 w-3" />
                    </a>
                  </div>
                  <div className="text-[11px] text-slate-500">CPF {cpfLabel(order.buyer.cpf)}</div>
                </div>
                <div className="space-y-1">
                  <SectionTitle>Mercado Pago</SectionTitle>
                  <div className="text-sm text-slate-200">
                    {order.mp.paymentID ? (
                      <a
                        href="https://www.mercadopago.com.br/activities"
                        target="_blank"
                        rel="noreferrer"
                        className="inline-flex items-center gap-1 font-mono hover:underline"
                      >
                        {order.mp.paymentID} <ExternalLink className="h-3 w-3" />
                      </a>
                    ) : (
                      <span className="text-slate-500">sem pagamento</span>
                    )}
                  </div>
                  <div className="text-[11px] text-slate-500">
                    {[order.mp.method, order.mp.status].filter(Boolean).join(" · ") || "—"}
                  </div>
                </div>
              </section>
            </div>

            <div className="mt-4 flex flex-wrap items-end gap-2 border-t-2 border-outline/20 pt-4">
              <Field label="Rastreio">
                <Input
                  value={tracking}
                  onChange={(e) => setTracking(e.target.value)}
                  placeholder="AA123456789BR"
                  className="w-48 font-mono"
                  disabled={busy || !(canShip || shipped)}
                />
              </Field>
              <Button
                variant="primary"
                size="sm"
                disabled={busy || !(canShip || shipped)}
                onClick={() =>
                  patch(shipped ? { trackingCode: tracking.trim() } : { status: "shipped", trackingCode: tracking.trim() })
                }
              >
                <Truck className="h-4 w-4" /> {shipped ? "Atualizar rastreio" : "Marcar enviado"}
              </Button>
              {targets.length > 0 && (
                <>
                  <Field label="Status">
                    <div className="w-52">
                      <Select
                        value={target}
                        disabled={busy}
                        onChange={(e) => {
                          if (isStatus(e.target.value)) {
                            setTarget(e.target.value);
                          }
                        }}
                      >
                        {[order.status, ...targets].map((s) => (
                          <option key={s} value={s}>
                            {statusLabels[s]}
                          </option>
                        ))}
                      </Select>
                    </div>
                  </Field>
                  <Button variant="outline" size="sm" disabled={busy || target === order.status} onClick={changeStatus}>
                    Mudar status
                  </Button>
                </>
              )}
              <Field label="Notas">
                <textarea
                  value={notes}
                  onChange={(e) => setNotes(e.target.value)}
                  rows={1}
                  disabled={busy}
                  className="flex min-h-9 w-72 rounded-[8px] border-2 border-outline bg-panel px-3 py-1.5 text-sm text-fg transition-colors placeholder:text-slate-500 disabled:cursor-not-allowed disabled:opacity-50"
                />
              </Field>
              <Button
                variant="outline"
                size="sm"
                disabled={busy || notes.trim() === (order.notes ?? "")}
                onClick={() => patch({ notes: notes.trim() })}
              >
                Salvar notas
              </Button>
            </div>
            {takesStock && (
              <p className="mt-2 text-[11px] text-amber-300">
                Mudar o status dá baixa no estoque dos itens deste pedido; sem estoque suficiente, o status não muda.
              </p>
            )}
            {returnsStock && (
              <p className="mt-2 text-[11px] text-amber-300">
                Mudar o status devolve ao estoque os itens já baixados deste pedido.
              </p>
            )}
            {failed && <p className="mt-2 text-[11px] text-rose-300">{failed}</p>}
            {viewing && <ArtViewer src={viewing} onClose={() => setViewing(null)} />}
          </TableCell>
        </TableRow>
      )}
    </>
  );
}

function LineArt({ src, alt, onOpen }: { src?: string; alt: string; onOpen: () => void }) {
  const [failed, setFailed] = useState(false);
  if (!src || failed) {
    return (
      <div
        className="relative flex h-40 w-28 flex-col items-center justify-center overflow-hidden rounded-[8px] border-2 border-outline/40 bg-panel"
        aria-hidden
      >
        <div className="art-stripes-45 pointer-events-none absolute inset-0 opacity-70" />
        <ImageOff className="relative h-8 w-8 text-brand-label/60" />
      </div>
    );
  }
  return (
    <button
      type="button"
      onClick={(e) => {
        e.stopPropagation();
        onOpen();
      }}
      className="sticker sticker-sm block overflow-hidden rounded-[8px] bg-panel transition-transform hover:-translate-y-0.5"
      aria-label={`Ver ${alt} em tamanho grande`}
    >
      <img
        src={src}
        alt={alt}
        width={112}
        height={160}
        loading="lazy"
        onError={() => setFailed(true)}
        className="h-40 w-28 object-cover"
      />
    </button>
  );
}

function ArtViewer({ src, onClose }: { src: string; onClose: () => void }) {
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        onClose();
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onClose]);
  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/85 p-6"
      onClick={(e) => {
        e.stopPropagation();
        onClose();
      }}
    >
      <button
        type="button"
        onClick={onClose}
        aria-label="Fechar"
        className="absolute right-5 top-5 rounded-full border-2 border-outline bg-panel p-2 text-slate-200"
      >
        <X className="h-5 w-5" />
      </button>
      <img
        src={src}
        alt=""
        onClick={(e) => e.stopPropagation()}
        className="sticker max-h-[90vh] max-w-[92vw] rounded-[12px] bg-panel object-contain"
      />
    </div>
  );
}

function Stamp({ label, value }: { label: string; value: string }) {
  if (!hasTime(value)) {
    return null;
  }
  return (
    <div>
      <dt className="font-pixel text-[8px] uppercase text-brand-label">{label}</dt>
      <dd className="tabular-nums text-slate-400">{fullStamp(value)}</dd>
    </div>
  );
}

function SectionTitle({ children }: { children: React.ReactNode }) {
  return <h3 className="font-pixel text-[8px] uppercase text-brand-label">{children}</h3>;
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <label className="flex flex-col gap-1">
      <span className="text-[10px] font-medium uppercase tracking-wide text-slate-500">{label}</span>
      {children}
    </label>
  );
}
