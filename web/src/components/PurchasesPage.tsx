import { Fragment, useCallback, useEffect, useMemo, useState } from "react";
import {
  ArrowUpDown,
  Boxes,
  Check,
  ChevronDown,
  ChevronRight,
  Coins,
  FolderInput,
  HandCoins,
  Link2,
  Merge,
  Package,
  Pencil,
  Plus,
  ShoppingCart,
  Trash2,
  Truck,
  Unlink,
  X,
} from "lucide-react";
import {
  addPurchaseItem,
  attachPurchaseItems,
  createPurchase,
  deletePurchase,
  deleteTrade,
  detachPurchaseItems,
  getGames,
  getQuote,
  listPurchases,
  mergeTrades,
  productIDFromTcgURL,
  sellTrade,
  updatePurchase,
  updateTrade,
  type GameInfo,
  type ItemRef,
  type PurchaseItem,
  type PurchaseView,
  type QuoteMatch,
} from "../api";
import { brl2, dayLabel } from "../format";
import CardArt from "./CardArt";
import EmptyState from "./EmptyState";
import { Chip, Kpi } from "./PortfolioPage";
import { Button } from "./ui/button";
import { Card } from "./ui/card";
import { Checkbox } from "./ui/checkbox";
import { Input } from "./ui/input";
import { Select } from "./ui/select";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "./ui/table";

// Accessories are stocked once for every game, so the picker offers them beside
// the games: the server files them in the shared ledger by their kind.
const ACCESSORIES = "acessorios";

// Depois de anos de lançamentos os avulsos são centenas: a tabela corta aqui e
// diz quantos ficaram de fora, para a busca continuar respondendo.
const LOOSE_LIMIT = 200;

type ItemKind = "single" | "sealed" | "accessory";

const kindLabels: Record<ItemKind, string> = {
  single: "Carta",
  sealed: "Selado",
  accessory: "Acessório",
};

// tradeGame says which ledger a write should hit: accessories live in the shared
// one, which the endpoints reach when no game is sent.
function tradeGame(item: PurchaseItem): string | undefined {
  return item.game === ACCESSORIES ? undefined : item.game;
}

function kindOf(item: PurchaseItem): ItemKind {
  return item.kind === "sealed" ? "sealed" : item.kind === "accessory" ? "accessory" : "single";
}

// Categorias na ordem em que uma compra é conferida: as cartas, que são o grosso
// do pedido, depois os selados e por último os acessórios.
const GROUP_ORDER: ItemKind[] = ["single", "sealed", "accessory"];

const groupLabels: Record<ItemKind, string> = {
  single: "Cartas",
  sealed: "Selados",
  accessory: "Acessórios",
};

// A sale never leaves the purchase it came from, so status is a filter instead of
// a separate screen: the default hides what is already sold.
type StatusFilter = "holding" | "sold" | "all";

const statusLabels: Record<StatusFilter, string> = {
  holding: "Em estoque",
  sold: "Vendidos",
  all: "Todos",
};

type SortKey = "name" | "game" | "qty" | "unit" | "cost" | "date";

interface SortState {
  key: SortKey;
  dir: "asc" | "desc";
}

function sortValue(item: PurchaseItem, key: SortKey): string | number {
  switch (key) {
    case "game":
      return item.gameName;
    case "qty":
      return item.qty;
    case "unit":
      return item.buyBRL;
    case "cost":
      return item.costBRL;
    case "date":
      return item.buyDate ?? "";
    default:
      return item.name.toLowerCase();
  }
}

function sortItems(list: PurchaseItem[], sort: SortState): PurchaseItem[] {
  const dir = sort.dir === "asc" ? 1 : -1;
  return [...list].sort((a, b) => {
    const va = sortValue(a, sort.key);
    const vb = sortValue(b, sort.key);
    if (va === vb) {
      return a.name.localeCompare(b.name);
    }
    return (
      (typeof va === "number" && typeof vb === "number" ? va - vb : String(va).localeCompare(String(vb))) *
      dir
    );
  });
}

function useSort(initial: SortState): [SortState, (key: SortKey) => void] {
  const [sort, setSort] = useState<SortState>(initial);
  const toggle = (key: SortKey) =>
    setSort((s) => (s.key === key ? { key, dir: s.dir === "asc" ? "desc" : "asc" } : { key, dir: "desc" }));
  return [sort, toggle];
}

// SortHead is a column header that sorts by its own key, showing which way the
// list is ordered right now.
function SortHead({
  label,
  sortKey,
  sort,
  onSort,
  className,
}: {
  label: string;
  sortKey: SortKey;
  sort: SortState;
  onSort: (key: SortKey) => void;
  className?: string;
}) {
  const active = sort.key === sortKey;
  return (
    <TableHead className={className}>
      <button
        type="button"
        onClick={() => onSort(sortKey)}
        className={`inline-flex items-center gap-1 uppercase transition-colors hover:text-fg ${active ? "text-fg" : ""}`}
      >
        {label}
        <ArrowUpDown className={`h-3 w-3 ${active ? "opacity-100" : "opacity-30"}`} />
      </button>
    </TableHead>
  );
}

// today is the local date: toISOString would roll over to tomorrow every evening
// in São Paulo, dating a purchase or a sale a day ahead.
function today(): string {
  const d = new Date();
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
}

function num(v: string): number {
  const n = Number(v.replace(",", "."));
  return Number.isFinite(n) ? n : 0;
}

function matchesItem(item: PurchaseItem, q: string): boolean {
  if (!q) {
    return true;
  }
  return `${item.name} ${item.number} ${item.set} ${item.store ?? ""} ${item.gameName}`
    .toLowerCase()
    .includes(q);
}

function matchesOrder(order: PurchaseView, q: string): boolean {
  if (!q) {
    return true;
  }
  if (`${order.store ?? ""} ${order.notes ?? ""} ${order.date}`.toLowerCase().includes(q)) {
    return true;
  }
  return order.items.some((it) => matchesItem(it, q));
}

export default function PurchasesPage() {
  const [orders, setOrders] = useState<PurchaseView[]>([]);
  const [loose, setLoose] = useState<PurchaseItem[]>([]);
  const [games, setGames] = useState<GameInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [tab, setTab] = useState<"orders" | "loose">("orders");
  const [query, setQuery] = useState("");
  const [gameFilter, setGameFilter] = useState<string>("all");
  const [creating, setCreating] = useState(false);
  const [open, setOpen] = useState<string | null>(null);
  const [status, setStatus] = useState<StatusFilter>("holding");
  const [kinds, setKinds] = useState<Set<ItemKind>>(new Set());
  const [transitOnly, setTransitOnly] = useState(false);

  const toggleKind = (kind: ItemKind) =>
    setKinds((prev) => {
      const next = new Set(prev);
      if (next.has(kind)) {
        next.delete(kind);
      } else {
        next.add(kind);
      }
      return next;
    });

  const load = useCallback(async () => {
    try {
      const r = await listPurchases();
      setOrders(r.purchases);
      setLoose(r.unassigned);
      setError(null);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "falha ao carregar as compras");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  useEffect(() => {
    getGames()
      .then((d) => setGames(d.games))
      .catch(() => setGames([]));
  }, []);

  const q = query.trim().toLowerCase();
  // One filter set drives both tabs: an item is shown when its game, category,
  // status and the search all match. Orders keep their own totals, which stay the
  // full order — a filter narrows what is listed, never what was paid.
  const keep = useCallback(
    (item: PurchaseItem) => {
      if (gameFilter !== "all" && item.game !== gameFilter) {
        return false;
      }
      if (kinds.size > 0 && !kinds.has(kindOf(item))) {
        return false;
      }
      if (status === "holding" && item.realized) {
        return false;
      }
      if (status === "sold" && !item.realized) {
        return false;
      }
      if (transitOnly && (item.realized || item.delivered)) {
        return false;
      }
      return matchesItem(item, q);
    },
    [gameFilter, kinds, status, transitOnly, q],
  );

  const visibleOrders = useMemo(
    () =>
      orders
        .map((o) => ({ ...o, items: o.items.filter(keep) }))
        .filter((o) => o.items.length > 0 || (!q && gameFilter === "all" && kinds.size === 0 && !transitOnly))
        .filter((o) => o.items.length > 0 || matchesOrder(o, q)),
    [orders, keep, q, gameFilter, kinds, transitOnly],
  );
  const visibleLoose = useMemo(() => loose.filter(keep), [loose, keep]);

  const totals = useMemo(() => {
    let spent = 0;
    let units = 0;
    let items = 0;
    let sold = 0;
    let proceeds = 0;
    for (const o of visibleOrders) {
      spent += o.totalBRL;
      units += o.units;
      items += o.items.length;
      sold += o.sold;
      proceeds += o.proceedsBRL;
    }
    return { spent, units, items, sold, proceeds };
  }, [visibleOrders]);

  const gameOptions = useMemo(
    () => [...games.map((g) => ({ id: g.id, name: g.name })), { id: ACCESSORIES, name: "Acessórios" }],
    [games],
  );

  if (loading) {
    return <EmptyState>Carregando compras…</EmptyState>;
  }
  if (error) {
    return (
      <div className="rounded-[14px] border-2 border-rose-900/50 bg-rose-950/30 px-4 py-3 text-sm text-rose-200">
        {error}
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <p className="max-w-2xl text-sm text-slate-400">
        Cada compra é um pedido: você abre com a data, o vendedor e o frete, e vai lançando os itens dentro
        dela — de qualquer jogo. O item entra no portfólio do jogo que você escolher, e o total aqui já soma o
        frete do pedido.
      </p>

      <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
        <Kpi
          icon={<Coins className="h-5 w-5" />}
          label="Total em compras"
          value={brl2(totals.spent)}
          sub={`${visibleOrders.length} ${visibleOrders.length === 1 ? "compra" : "compras"} no filtro`}
        />
        <Kpi
          icon={<Package className="h-5 w-5" />}
          label="Itens comprados"
          value={String(totals.items)}
          sub={`${totals.units} unidades`}
        />
        <Kpi
          icon={<ShoppingCart className="h-5 w-5" />}
          label="Já vendido"
          value={brl2(totals.proceeds)}
          sub={`${totals.sold} itens desses pedidos`}
        />
        <Kpi
          icon={<Boxes className="h-5 w-5" />}
          label="Itens avulsos"
          value={String(loose.length)}
          sub="lançados fora de uma compra"
        />
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <Chip active={tab === "orders"} onClick={() => setTab("orders")}>
          Compras · {orders.length}
        </Chip>
        <Chip active={tab === "loose"} onClick={() => setTab("loose")}>
          Avulsos · {loose.length}
        </Chip>
        <div className="ml-auto flex flex-wrap items-center gap-2">
          <Input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Buscar carta, vendedor…"
            className="w-56"
          />
          <div className="w-44">
            <Select value={gameFilter} onChange={(e) => setGameFilter(e.target.value)}>
              <option value="all">Todos os jogos</option>
              {gameOptions.map((g) => (
                <option key={g.id} value={g.id}>
                  {g.name}
                </option>
              ))}
            </Select>
          </div>
          {tab === "orders" && (
            <Button variant="primary" onClick={() => setCreating((v) => !v)}>
              {creating ? <X className="h-4 w-4" /> : <Plus className="h-4 w-4" />}
              {creating ? "Cancelar" : "Nova compra"}
            </Button>
          )}
        </div>
      </div>

      <div className="flex flex-wrap items-center gap-x-4 gap-y-2 border-b-[3px] border-outline pb-3">
        <div className="flex flex-wrap items-center gap-1.5">
          {(Object.keys(statusLabels) as StatusFilter[]).map((s) => (
            <Chip key={s} active={status === s} onClick={() => setStatus(s)}>
              {statusLabels[s]}
            </Chip>
          ))}
        </div>
        <div className="flex flex-wrap items-center gap-1.5">
          {GROUP_ORDER.map((kind) => (
            <Chip key={kind} active={kinds.has(kind)} onClick={() => toggleKind(kind)}>
              {groupLabels[kind]}
            </Chip>
          ))}
        </div>
        <Chip active={transitOnly} onClick={() => setTransitOnly((v) => !v)}>
          <span className="inline-flex items-center gap-1">
            <Truck className="h-3 w-3" /> Em trânsito
          </span>
        </Chip>
        <span className="ml-auto text-[11px] text-slate-500">
          {kinds.size === 0 ? "todas as categorias" : `${kinds.size} de 3 categorias`}
        </span>
      </div>

      {tab === "orders" && creating && (
        <PurchaseForm
          onCancel={() => setCreating(false)}
          onSaved={async (id) => {
            setCreating(false);
            await load();
            setOpen(id);
          }}
        />
      )}

      {tab === "orders" ? (
        visibleOrders.length === 0 ? (
          <EmptyState hint={orders.length > 0 ? "ajuste a busca ou o jogo" : "clique em Nova compra"}>
            {orders.length > 0 ? "Nenhuma compra bate com o filtro." : "Nenhuma compra registrada ainda."}
          </EmptyState>
        ) : (
          <div className="space-y-3">
            {visibleOrders.map((o) => (
              <OrderCard
                key={o.id}
                order={o}
                orders={orders}
                games={gameOptions}
                open={open === o.id}
                onToggle={() => setOpen((k) => (k === o.id ? null : o.id))}
                onChanged={load}
              />
            ))}
          </div>
        )
      ) : (
        <LooseItems items={visibleLoose} orders={orders} total={loose.length} onChanged={load} />
      )}
    </div>
  );
}

function PurchaseForm({
  initial,
  onCancel,
  onSaved,
}: {
  initial?: PurchaseView;
  onCancel: () => void;
  onSaved: (id: string) => void;
}) {
  const [date, setDate] = useState(initial?.date ?? today());
  const [store, setStore] = useState(initial?.store ?? "");
  const [shipping, setShipping] = useState(initial ? String(initial.shippingBRL ?? "") : "");
  const [notes, setNotes] = useState(initial?.notes ?? "");
  const [saving, setSaving] = useState(false);
  const [failed, setFailed] = useState<string | null>(null);

  const submit = async () => {
    setSaving(true);
    setFailed(null);
    try {
      const body = {
        date,
        store: store.trim(),
        shippingBRL: num(shipping),
        notes: notes.trim(),
      };
      const saved = initial ? await updatePurchase(initial.id, body) : await createPurchase(body);
      onSaved(saved.id);
    } catch (err: unknown) {
      setFailed(err instanceof Error ? err.message : "falha ao salvar");
    } finally {
      setSaving(false);
    }
  };

  return (
    <Card className="space-y-3 p-4">
      <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
        <Field label="Data">
          <Input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
        </Field>
        <Field label="Vendedor / loja">
          <Input
            value={store}
            onChange={(e) => setStore(e.target.value)}
            placeholder="ex.: Legends, Fulano (WhatsApp)"
          />
        </Field>
        <Field label="Frete do pedido R$">
          <Input
            value={shipping}
            onChange={(e) => setShipping(e.target.value)}
            inputMode="decimal"
            placeholder="0,00"
          />
        </Field>
        <Field label="Observação">
          <Input value={notes} onChange={(e) => setNotes(e.target.value)} placeholder="opcional" />
        </Field>
      </div>
      {failed && <p className="text-[11px] text-rose-300">{failed}</p>}
      <div className="flex justify-end gap-2">
        <Button variant="ghost" onClick={onCancel} disabled={saving}>
          Cancelar
        </Button>
        <Button variant="primary" onClick={submit} disabled={saving}>
          {saving ? "Salvando…" : initial ? "Salvar compra" : "Criar compra"}
        </Button>
      </div>
    </Card>
  );
}

function OrderCard({
  order,
  orders,
  games,
  open,
  onToggle,
  onChanged,
}: {
  order: PurchaseView;
  // Every order, so an item can be moved from this one into another.
  orders: PurchaseView[];
  games: { id: string; name: string }[];
  open: boolean;
  onToggle: () => void;
  onChanged: () => Promise<void> | void;
}) {
  const [editing, setEditing] = useState(false);
  const [adding, setAdding] = useState(false);
  const [failed, setFailed] = useState<string | null>(null);
  const [sort, onSort] = useSort({ key: "cost", dir: "desc" });
  const [collapsed, setCollapsed] = useState<Set<ItemKind>>(new Set());
  const [picked, setPicked] = useState<Set<string>>(new Set());

  const togglePick = (id: string) =>
    setPicked((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });

  const pickMany = (rows: PurchaseItem[]) =>
    setPicked((prev) => {
      const next = new Set(prev);
      const allPicked = rows.every((it) => next.has(it.id));
      for (const it of rows) {
        if (allPicked) {
          next.delete(it.id);
        } else {
          next.add(it.id);
        }
      }
      return next;
    });

  const toggleGroup = (kind: ItemKind) =>
    setCollapsed((prev) => {
      const next = new Set(prev);
      if (next.has(kind)) {
        next.delete(kind);
      } else {
        next.add(kind);
      }
      return next;
    });

  const groups = GROUP_ORDER.map((kind) => {
    const items = order.items.filter((it) => kindOf(it) === kind);
    return {
      kind,
      items,
      open: !collapsed.has(kind),
      units: items.reduce((s, it) => s + it.qty, 0),
      cost: items.reduce((s, it) => s + it.costBRL, 0),
    };
  }).filter((g) => g.items.length > 0);

  const remove = async () => {
    setFailed(null);
    try {
      await deletePurchase(order.id);
      await onChanged();
    } catch (err: unknown) {
      setFailed(err instanceof Error ? err.message : "falha ao excluir");
    }
  };

  const gameNames = [...new Set(order.items.map((it) => it.gameName))];

  return (
    <Card className="overflow-hidden">
      <div className="flex flex-wrap items-center gap-3 px-4 py-3">
        <button type="button" onClick={onToggle} className="flex min-w-0 flex-1 items-center gap-2 text-left">
          {open ? (
            <ChevronDown className="h-4 w-4 shrink-0 text-slate-500" />
          ) : (
            <ChevronRight className="h-4 w-4 shrink-0 text-slate-500" />
          )}
          <div className="min-w-0">
            <div className="truncate font-semibold text-slate-100">
              {order.store || "Compra sem vendedor"}
            </div>
            <div className="truncate text-[11px] text-slate-500">
              {[
                order.date ? dayLabel(order.date) : "sem data",
                `${order.items.length} itens · ${order.units} un.`,
                gameNames.join(" · "),
                order.notes,
              ]
                .filter(Boolean)
                .join(" — ")}
            </div>
          </div>
        </button>
        <div className="text-right">
          <div className="font-bold tabular-nums text-slate-100">{brl2(order.totalBRL)}</div>
          <div className="text-[10px] text-slate-500">
            {order.shippingBRL
              ? `itens ${brl2(order.itemsCostBRL)} + frete ${brl2(order.shippingBRL)}`
              : "itens + frete"}
          </div>
        </div>
        <div className="flex items-center gap-1">
          <Button variant="ghost" size="sm" onClick={() => setEditing((v) => !v)} title="Editar compra">
            <Pencil className="h-4 w-4" />
          </Button>
          <Button variant="ghost" size="sm" onClick={remove} title="Excluir compra">
            <Trash2 className="h-4 w-4" />
          </Button>
        </div>
      </div>

      {failed && <p className="px-4 pb-2 text-[11px] text-rose-300">{failed}</p>}

      {editing && (
        <div className="border-t-2 border-outline/20 p-3">
          <PurchaseForm
            initial={order}
            onCancel={() => setEditing(false)}
            onSaved={async () => {
              setEditing(false);
              await onChanged();
            }}
          />
        </div>
      )}

      {open && (
        <div className="space-y-3 border-t-2 border-outline/20 bg-raised/30 p-3">
          {order.items.length === 0 ? (
            <p className="px-1 text-sm text-slate-500">Nenhum item nesta compra ainda.</p>
          ) : (
            <>
              <MergeBar
                items={order.items.filter((it) => picked.has(it.id))}
                onDone={async () => {
                  setPicked(new Set());
                  await onChanged();
                }}
              />
              <div className="overflow-x-auto">
                <Table className="min-w-[720px]">
                  <TableHeader>
                    <TableRow className="hover:bg-transparent">
                      <TableHead className="w-10" />
                      <SortHead label="Item" sortKey="name" sort={sort} onSort={onSort} />
                      <SortHead label="Jogo" sortKey="game" sort={sort} onSort={onSort} />
                      <SortHead
                        label="Un."
                        sortKey="qty"
                        sort={sort}
                        onSort={onSort}
                        className="text-right"
                      />
                      <SortHead
                        label="Preço un."
                        sortKey="unit"
                        sort={sort}
                        onSort={onSort}
                        className="text-right"
                      />
                      <SortHead
                        label="Custo"
                        sortKey="cost"
                        sort={sort}
                        onSort={onSort}
                        className="text-right"
                      />
                      <TableHead>Status</TableHead>
                      <TableHead />
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {groups.map((g) => (
                      <Fragment key={g.kind}>
                        <TableRow className="bg-raised/60">
                          <TableCell className="py-2">
                            <Checkbox
                              checked={g.items.length > 0 && g.items.every((it) => picked.has(it.id))}
                              onChange={() => pickMany(g.items)}
                              title="Selecionar os itens desta categoria"
                            />
                          </TableCell>
                          <TableCell
                            colSpan={2}
                            className="cursor-pointer py-2"
                            onClick={() => toggleGroup(g.kind)}
                          >
                            <div className="flex items-center gap-2">
                              {g.open ? (
                                <ChevronDown className="h-4 w-4 text-slate-500" />
                              ) : (
                                <ChevronRight className="h-4 w-4 text-slate-500" />
                              )}
                              <span className="font-semibold text-slate-200">{groupLabels[g.kind]}</span>
                              <span className="text-[11px] text-slate-500">
                                {g.items.length} {g.items.length === 1 ? "item" : "itens"}
                              </span>
                            </div>
                          </TableCell>
                          <TableCell className="py-2 text-right tabular-nums text-slate-400">
                            {g.units}
                          </TableCell>
                          <TableCell className="py-2" />
                          <TableCell className="py-2 text-right font-semibold tabular-nums text-slate-200">
                            {brl2(g.cost)}
                          </TableCell>
                          <TableCell className="py-2" colSpan={2} />
                        </TableRow>
                        {g.open &&
                          sortItems(g.items, sort).map((it) => (
                            <ItemRow
                              key={it.id}
                              item={it}
                              orders={orders}
                              selected={picked.has(it.id)}
                              onSelect={() => togglePick(it.id)}
                              onChanged={onChanged}
                            />
                          ))}
                      </Fragment>
                    ))}
                  </TableBody>
                </Table>
              </div>
            </>
          )}

          {adding ? (
            <AddItemForm
              order={order}
              orders={orders}
              games={games}
              onCancel={() => setAdding(false)}
              onSaved={onChanged}
            />
          ) : (
            <Button variant="primary" size="sm" onClick={() => setAdding(true)}>
              <Plus className="h-4 w-4" />
              Adicionar item
            </Button>
          )}
        </div>
      )}
    </Card>
  );
}

function ItemRow({
  item,
  orders,
  selected,
  onSelect,
  showOrigin,
  onChanged,
}: {
  item: PurchaseItem;
  orders: PurchaseView[];
  selected?: boolean;
  onSelect?: () => void;
  // Outside a purchase card there is no header saying when and from whom the item
  // came, so the row carries it.
  showOrigin?: boolean;
  onChanged: () => Promise<void> | void;
}) {
  const [busy, setBusy] = useState(false);
  const [panel, setPanel] = useState<"none" | "move" | "edit" | "sell">("none");
  const unit = item.qty > 0 ? item.buyBRL : 0;
  const others = orders.filter((o) => o.id !== item.purchaseID);
  const columns = onSelect ? 8 : 7;

  const run = async (fn: () => Promise<unknown>) => {
    setBusy(true);
    try {
      await fn();
      await onChanged();
    } finally {
      setBusy(false);
    }
  };

  if (panel !== "none") {
    return (
      <TableRow className="hover:bg-transparent">
        <TableCell colSpan={columns} className="bg-raised/40">
          {panel === "move" && (
            <MoveItem
              item={item}
              orders={others}
              busy={busy}
              onCancel={() => setPanel("none")}
              onMove={async (target) => {
                await run(() => attachPurchaseItems(target, [{ id: item.id, game: item.game }]));
                setPanel("none");
              }}
            />
          )}
          {panel === "edit" && (
            <EditItemForm
              item={item}
              onCancel={() => setPanel("none")}
              onSaved={async () => {
                setPanel("none");
                await onChanged();
              }}
            />
          )}
          {panel === "sell" && (
            <SellItemForm
              item={item}
              onCancel={() => setPanel("none")}
              onSold={async () => {
                setPanel("none");
                await onChanged();
              }}
            />
          )}
        </TableCell>
      </TableRow>
    );
  }

  return (
    <TableRow className={item.realized ? "opacity-70" : undefined}>
      {onSelect && (
        <TableCell>
          <Checkbox checked={Boolean(selected)} onChange={onSelect} />
        </TableCell>
      )}
      <TableCell>
        <div className="flex items-center gap-2">
          <CardArt
            set={item.set}
            number={item.number}
            name={item.name}
            game={item.game}
            imageURL={item.imageURL}
            productID={productIDFromTcgURL(item.tcgUrl)}
            className="h-12 w-[34px] shrink-0 rounded"
          />
          <div className="min-w-0">
            <div className="truncate text-slate-100">{item.name}</div>
            <div className="truncate text-[10px] text-slate-500">
              {[
                item.number,
                item.set,
                item.variant,
                kindLabels[kindOf(item)],
                showOrigin && item.buyDate ? dayLabel(item.buyDate) : "",
                showOrigin ? item.store : "",
              ]
                .filter(Boolean)
                .join(" · ")}
            </div>
          </div>
        </div>
      </TableCell>
      <TableCell className="whitespace-nowrap text-[11px] text-slate-400">{item.gameName}</TableCell>
      <TableCell className="text-right tabular-nums text-slate-400">{item.qty}</TableCell>
      <TableCell className="text-right tabular-nums text-slate-400">{brl2(unit)}</TableCell>
      <TableCell className="text-right font-semibold tabular-nums text-slate-200">
        {brl2(item.costBRL)}
      </TableCell>
      <TableCell className="whitespace-nowrap text-[11px]">
        {item.realized ? (
          <span className="text-emerald-300">
            vendido {brl2(item.valueBRL)}
            {item.sellDate ? (
              <span className="block text-[10px] text-slate-500">{dayLabel(item.sellDate)}</span>
            ) : null}
          </span>
        ) : (
          <DeliveryToggle item={item} onChanged={onChanged} />
        )}
      </TableCell>
      <TableCell className="text-right">
        <div className="flex items-center justify-end gap-1">
          {!item.realized && (
            <>
              <Button
                variant="ghost"
                size="sm"
                disabled={busy}
                title="Dar baixa: vender este item"
                onClick={() => setPanel("sell")}
              >
                <HandCoins className="h-4 w-4" />
              </Button>
              <Button
                variant="ghost"
                size="sm"
                disabled={busy}
                title="Editar o item (preço, quantidade, variante, loja…)"
                onClick={() => setPanel("edit")}
              >
                <Pencil className="h-4 w-4" />
              </Button>
            </>
          )}
          <Button
            variant="ghost"
            size="sm"
            disabled={busy || others.length === 0}
            title={
              others.length === 0 ? "Não há outra compra para mover este item" : "Mover para outra compra"
            }
            onClick={() => setPanel("move")}
          >
            <FolderInput className="h-4 w-4" />
          </Button>
          {item.purchaseID && (
            <Button
              variant="ghost"
              size="sm"
              disabled={busy}
              title="Tirar desta compra (o item continua no estoque, como avulso)"
              onClick={() => run(() => detachPurchaseItems([{ id: item.id, game: item.game }]))}
            >
              <Unlink className="h-4 w-4" />
            </Button>
          )}
          <Button
            variant="ghost"
            size="sm"
            disabled={busy}
            title="Excluir o item do portfólio"
            onClick={() => run(() => deleteTrade(item.id, item.game === ACCESSORIES ? undefined : item.game))}
          >
            <Trash2 className="h-4 w-4" />
          </Button>
        </div>
      </TableCell>
    </TableRow>
  );
}

// MergeBar collapses several lots of the same card into one, summing quantity and
// freight and averaging what was paid. Merging only works inside one ledger, so
// the selection has to be of a single game, and a sold lot can no longer change.
function MergeBar({ items, onDone }: { items: PurchaseItem[]; onDone: () => Promise<void> | void }) {
  const [busy, setBusy] = useState(false);
  const [failed, setFailed] = useState<string | null>(null);

  if (items.length === 0) {
    return null;
  }
  const games = new Set(items.map((it) => it.game));
  const anySold = items.some((it) => it.realized);
  const canMerge = items.length >= 2 && games.size === 1 && !anySold;
  const why = anySold
    ? "um lote vendido não pode ser juntado"
    : games.size > 1
      ? "só dá para juntar itens do mesmo jogo"
      : "selecione dois ou mais itens";

  const merge = async () => {
    setBusy(true);
    setFailed(null);
    try {
      await mergeTrades(
        items.map((it) => it.id),
        tradeGame(items[0]),
      );
      await onDone();
    } catch (err: unknown) {
      setFailed(err instanceof Error ? err.message : "falha ao juntar");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="mb-2 flex flex-wrap items-center gap-2 rounded-[10px] border-2 border-outline bg-panel px-3 py-2">
      <span className="text-xs text-slate-400">{items.length} selecionados</span>
      <div className="ml-auto flex items-center gap-2">
        {!canMerge && <span className="text-[11px] text-slate-500">{why}</span>}
        <Button variant="primary" size="sm" onClick={merge} disabled={busy || !canMerge}>
          <Merge className="h-4 w-4" />
          Juntar em um lote
        </Button>
      </div>
      {failed && <p className="w-full text-[11px] text-rose-300">{failed}</p>}
    </div>
  );
}

// DeliveryToggle flips whether the item already arrived. An import takes weeks,
// so "em trânsito" is the state most of a purchase sits in.
function DeliveryToggle({ item, onChanged }: { item: PurchaseItem; onChanged: () => Promise<void> | void }) {
  const [saving, setSaving] = useState(false);
  const toggle = async () => {
    setSaving(true);
    try {
      await updateTrade(item.id, { ...item, delivered: !item.delivered }, tradeGame(item));
      await onChanged();
    } finally {
      setSaving(false);
    }
  };
  return (
    <button
      type="button"
      onClick={toggle}
      disabled={saving}
      title={
        item.delivered
          ? "Entregue — clique para marcar em trânsito"
          : "Em trânsito — clique para marcar entregue"
      }
      className={`flex items-center gap-1 rounded-[8px] border-2 border-outline px-2 py-1 text-xs font-bold disabled:opacity-50 ${
        item.delivered
          ? "bg-emerald-500/10 text-emerald-300 hover:bg-emerald-500/20"
          : "bg-amber-500/10 text-amber-300 hover:bg-amber-500/20"
      }`}
    >
      {item.delivered ? <Check className="h-3 w-3" /> : <Truck className="h-3 w-3" />}
      {item.delivered ? "Entregue" : "Em trânsito"}
    </button>
  );
}

// EditItemForm changes what was actually bought. Accessories live in the shared
// ledger, which the write endpoints reach without a game.
function EditItemForm({
  item,
  onCancel,
  onSaved,
}: {
  item: PurchaseItem;
  onCancel: () => void;
  onSaved: () => Promise<void> | void;
}) {
  const single = kindOf(item) === "single";
  const [name, setName] = useState(item.name);
  const [number, setNumber] = useState(item.number);
  const [set, setSet] = useState(item.set);
  const [variant, setVariant] = useState(item.variant ?? "");
  const [condition, setCondition] = useState(item.condition ?? "");
  const [qty, setQty] = useState(String(item.qty));
  const [buyBRL, setBuyBRL] = useState(String(item.buyBRL));
  const [shipping, setShipping] = useState(String(item.shippingBRL || ""));
  const [store, setStore] = useState(item.store ?? "");
  const [buyDate, setBuyDate] = useState(item.buyDate ?? "");
  const [imageURL, setImageURL] = useState(item.imageURL ?? "");
  const [notes, setNotes] = useState(item.notes ?? "");
  const [saving, setSaving] = useState(false);
  const [failed, setFailed] = useState<string | null>(null);

  const submit = async () => {
    setSaving(true);
    setFailed(null);
    try {
      await updateTrade(
        item.id,
        {
          ...item,
          name,
          number,
          set,
          variant,
          condition,
          qty: Number(qty) || 1,
          buyBRL: num(buyBRL),
          shippingBRL: num(shipping),
          store,
          buyDate,
          imageURL: imageURL.trim(),
          notes,
        },
        tradeGame(item),
      );
      await onSaved();
    } catch (err: unknown) {
      setFailed(err instanceof Error ? err.message : "falha ao salvar");
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="space-y-3">
      <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4">
        <Field label="Nome">
          <Input value={name} onChange={(e) => setName(e.target.value)} />
        </Field>
        {single && (
          <>
            <Field label="Número">
              <Input value={number} onChange={(e) => setNumber(e.target.value)} />
            </Field>
            <Field label="Coleção">
              <Input value={set} onChange={(e) => setSet(e.target.value)} />
            </Field>
            <Field label="Variante">
              <Input value={variant} onChange={(e) => setVariant(e.target.value)} placeholder="Foil" />
            </Field>
            <Field label="Condição">
              <Input value={condition} onChange={(e) => setCondition(e.target.value)} placeholder="NM" />
            </Field>
          </>
        )}
        <Field label="Un.">
          <Input value={qty} onChange={(e) => setQty(e.target.value)} inputMode="numeric" />
        </Field>
        <Field label="Preço pago R$ (un.)">
          <Input value={buyBRL} onChange={(e) => setBuyBRL(e.target.value)} inputMode="decimal" />
        </Field>
        <Field label="Frete do item R$">
          <Input value={shipping} onChange={(e) => setShipping(e.target.value)} inputMode="decimal" />
        </Field>
        <Field label="Vendedor / loja">
          <Input value={store} onChange={(e) => setStore(e.target.value)} />
        </Field>
        <Field label="Comprado em">
          <Input type="date" value={buyDate} onChange={(e) => setBuyDate(e.target.value)} />
        </Field>
        <Field label="Imagem (URL)">
          <Input value={imageURL} onChange={(e) => setImageURL(e.target.value)} placeholder="https://…" />
        </Field>
        <Field label="Observação">
          <Input value={notes} onChange={(e) => setNotes(e.target.value)} />
        </Field>
      </div>
      {failed && <p className="text-[11px] text-rose-300">{failed}</p>}
      <div className="flex justify-end gap-2">
        <Button variant="ghost" size="sm" onClick={onCancel} disabled={saving}>
          Cancelar
        </Button>
        <Button variant="primary" size="sm" onClick={submit} disabled={saving}>
          {saving ? "Salvando…" : "Salvar item"}
        </Button>
      </div>
    </div>
  );
}

// SellItemForm gives an item its exit. Selling fewer units than the lot has splits
// it, so the rest stays in stock with its share of the freight.
function SellItemForm({
  item,
  onCancel,
  onSold,
}: {
  item: PurchaseItem;
  onCancel: () => void;
  onSold: () => Promise<void> | void;
}) {
  const [qty, setQty] = useState(String(item.qty));
  const [priceEach, setPriceEach] = useState("");
  const [currency, setCurrency] = useState<"BRL" | "USD">("BRL");
  const [date, setDate] = useState(today());
  const [buyer, setBuyer] = useState("");
  const [saving, setSaving] = useState(false);
  const [failed, setFailed] = useState<string | null>(null);

  const units = Math.min(Math.max(Number(qty) || 0, 1), item.qty);
  const each = num(priceEach);
  const total = each * units;
  const partial = units < item.qty;

  const submit = async () => {
    setSaving(true);
    setFailed(null);
    try {
      await sellTrade(
        item.id,
        { qty: units, sellPrice: total, sellCurrency: currency, sellDate: date, buyer: buyer.trim() },
        tradeGame(item),
      );
      await onSold();
    } catch (err: unknown) {
      setFailed(err instanceof Error ? err.message : "falha ao dar baixa");
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="space-y-2">
      <div className="flex flex-wrap items-end gap-2">
        <span className="min-w-0 flex-1 truncate text-sm text-slate-300">
          Vender <span className="font-semibold text-slate-100">{item.name}</span>
        </span>
        {item.qty > 1 && (
          <Field label={`Un. (de ${item.qty})`}>
            <Input
              value={qty}
              onChange={(e) => setQty(e.target.value)}
              inputMode="numeric"
              className="w-20"
            />
          </Field>
        )}
        <Field label={item.qty > 1 ? "Preço por un." : "Preço de venda"}>
          <Input
            value={priceEach}
            onChange={(e) => setPriceEach(e.target.value)}
            inputMode="decimal"
            placeholder="0,00"
            className="w-32"
          />
        </Field>
        <Field label="Moeda">
          <Select
            value={currency}
            onChange={(e) => setCurrency(e.target.value === "USD" ? "USD" : "BRL")}
            className="w-24"
          >
            <option value="BRL">R$</option>
            <option value="USD">US$</option>
          </Select>
        </Field>
        <Field label="Data">
          <Input type="date" value={date} onChange={(e) => setDate(e.target.value)} className="w-40" />
        </Field>
        <Field label="Comprador / canal">
          <Input
            value={buyer}
            onChange={(e) => setBuyer(e.target.value)}
            placeholder="ex.: Liga #11641862, balcão"
            className="w-52"
          />
        </Field>
        <Button variant="primary" size="sm" onClick={submit} disabled={saving || each <= 0}>
          {saving ? "Salvando…" : partial ? `Vender ${units}` : "Dar baixa"}
        </Button>
        <Button variant="ghost" size="sm" onClick={onCancel} disabled={saving}>
          <X className="h-4 w-4" />
        </Button>
      </div>
      <p className="text-[10px] text-slate-500">
        {currency === "USD" ? "US$ " : "R$ "}
        {(total || 0).toFixed(2)} no total{partial ? ` · ${item.qty - units} ficam em estoque` : ""}. O
        comprador define o canal: “Liga #nº” marca pedido da Liga.
      </p>
      {failed && <p className="text-[11px] text-rose-300">{failed}</p>}
    </div>
  );
}

// MoveItem picks which order an item should belong to. Attaching to another order
// is the move: an item points at exactly one purchase.
function MoveItem({
  item,
  orders,
  busy,
  onCancel,
  onMove,
}: {
  item: PurchaseItem;
  orders: PurchaseView[];
  busy: boolean;
  onCancel: () => void;
  onMove: (target: string) => Promise<void> | void;
}) {
  const [target, setTarget] = useState("");
  return (
    <div className="flex flex-wrap items-center gap-2">
      <span className="min-w-0 flex-1 truncate text-sm text-slate-300">
        Mover <span className="font-semibold text-slate-100">{item.name}</span> para:
      </span>
      <div className="w-64">
        <Select value={target} onChange={(e) => setTarget(e.target.value)}>
          <option value="">Escolha a compra…</option>
          {orders.map((o) => (
            <option key={o.id} value={o.id}>
              {[o.date, o.store || "sem vendedor"].filter(Boolean).join(" · ")}
            </option>
          ))}
        </Select>
      </div>
      <Button variant="primary" size="sm" disabled={busy || !target} onClick={() => onMove(target)}>
        <Link2 className="h-4 w-4" />
        Mover
      </Button>
      <Button variant="ghost" size="sm" onClick={onCancel} disabled={busy}>
        <X className="h-4 w-4" />
      </Button>
    </div>
  );
}

function AddItemForm({
  order,
  orders,
  games,
  onCancel,
  onSaved,
}: {
  order: PurchaseView;
  orders: PurchaseView[];
  games: { id: string; name: string }[];
  onCancel: () => void;
  onSaved: () => Promise<void> | void;
}) {
  // The item lands in the order it was opened from, but a lot bought in one go
  // often gets typed in from another order's card — so the target is a field.
  const [target, setTarget] = useState(order.id);
  const [game, setGame] = useState(games[0]?.id ?? "onepiece");
  const [kind, setKind] = useState<ItemKind>("single");
  const [query, setQuery] = useState("");
  const [matches, setMatches] = useState<QuoteMatch[]>([]);
  const [number, setNumber] = useState("");
  const [name, setName] = useState("");
  const [set, setSet] = useState("");
  const [variant, setVariant] = useState("");
  const [condition, setCondition] = useState("NM");
  const [qty, setQty] = useState("1");
  const [buyBRL, setBuyBRL] = useState("");
  const [shipping, setShipping] = useState("");
  const [refValue, setRefValue] = useState("");
  const [saving, setSaving] = useState(false);
  const [failed, setFailed] = useState<string | null>(null);

  // Accessories have no catalog to search and belong to no game, so picking them
  // both locks the game picker and drops the search box.
  const searchable = kind !== "accessory" && game !== ACCESSORIES;

  useEffect(() => {
    if (game === ACCESSORIES) {
      setKind("accessory");
    }
  }, [game]);

  useEffect(() => {
    if (!searchable || query.trim().length < 2) {
      setMatches([]);
      return;
    }
    let current = true;
    const h = window.setTimeout(() => {
      getQuote(query, 25, kind === "sealed" ? "sealed" : undefined, game)
        .then((r) => current && setMatches(r.matches))
        .catch(() => current && setMatches([]));
    }, 250);
    return () => {
      current = false;
      window.clearTimeout(h);
    };
  }, [query, kind, game, searchable]);

  const pick = (m: QuoteMatch) => {
    setNumber(m.number);
    setName(m.name);
    setSet(m.set);
    setVariant(m.variant ?? "");
    setRefValue(m.marketUSD ? String(Math.round(m.marketUSD * 100) / 100) : "");
    setQuery(m.number ? `${m.number} · ${m.name}` : m.name);
    setMatches([]);
  };

  const reset = () => {
    setQuery("");
    setNumber("");
    setName("");
    setSet(kind === "sealed" ? "SEALED" : "");
    setVariant("");
    setQty("1");
    setBuyBRL("");
    setShipping("");
    setRefValue("");
  };

  const submit = async () => {
    setSaving(true);
    setFailed(null);
    try {
      const manual = kind !== "single";
      await addPurchaseItem(target, game, {
        kind: kind === "single" ? undefined : kind === "sealed" ? "sealed" : "accessory",
        number,
        name,
        set: kind === "sealed" ? "SEALED" : kind === "accessory" ? "ACESSORIO" : set,
        variant,
        condition: kind === "single" ? condition : undefined,
        qty: Number(qty) || 1,
        buyBRL: num(buyBRL),
        shippingBRL: num(shipping),
        refUSD: manual ? 0 : num(refValue),
        manualBRL: manual ? num(refValue) : 0,
        status: "holding",
      });
      reset();
      await onSaved();
    } catch (err: unknown) {
      setFailed(err instanceof Error ? err.message : "falha ao salvar o item");
    } finally {
      setSaving(false);
    }
  };

  const valid = (number || name) && num(buyBRL) > 0;

  return (
    <Card className="space-y-3 p-4">
      <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
        <Field label="Compra">
          <Select value={target} onChange={(e) => setTarget(e.target.value)}>
            {orders.map((o) => (
              <option key={o.id} value={o.id}>
                {[o.date, o.store || "sem vendedor"].filter(Boolean).join(" · ")}
              </option>
            ))}
          </Select>
        </Field>
        <Field label="Jogo">
          <Select value={game} onChange={(e) => setGame(e.target.value)}>
            {games.map((g) => (
              <option key={g.id} value={g.id}>
                {g.name}
              </option>
            ))}
          </Select>
        </Field>
        <Field label="Tipo">
          <Select
            value={kind}
            onChange={(e) => setKind(e.target.value as ItemKind)}
            disabled={game === ACCESSORIES}
          >
            <option value="single">Carta</option>
            <option value="sealed">Selado</option>
            <option value="accessory">Acessório</option>
          </Select>
        </Field>
        {searchable && (
          <div className="relative col-span-2 lg:col-span-4">
            <Field label={kind === "sealed" ? "Buscar produto selado" : "Buscar carta (nome ou número)"}>
              <Input
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                placeholder={kind === "sealed" ? "ex.: Booster OP-16" : "ex.: Luffy ou OP16-080"}
                className="w-full"
              />
            </Field>
            {matches.length > 0 && (
              <ul className="sticker sticker-sm absolute z-10 mt-1 max-h-72 w-full overflow-y-auto rounded-[10px] bg-panel">
                {matches.map((m) => (
                  <li key={`${m.number}-${m.name}`}>
                    <button
                      type="button"
                      onClick={() => pick(m)}
                      className="flex w-full items-center justify-between gap-3 px-3 py-2 text-left text-sm hover:bg-raised"
                    >
                      <span className="flex min-w-0 items-center gap-2">
                        <CardArt
                          set={m.set}
                          number={m.number}
                          name={m.name}
                          game={game}
                          productID={m.productID}
                          className="h-12 w-[34px] shrink-0 rounded"
                        />
                        <span className="min-w-0">
                          <span className="block truncate text-slate-100">{m.name}</span>
                          <span className="font-mono text-[11px] text-slate-500">{m.number}</span>
                        </span>
                      </span>
                      {m.ligaLowBRL ? (
                        <span className="shrink-0 tabular-nums text-emerald-300">{brl2(m.ligaLowBRL)}</span>
                      ) : null}
                    </button>
                  </li>
                ))}
              </ul>
            )}
          </div>
        )}
      </div>

      <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4">
        <Field label="Nome">
          <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="Nome do item" />
        </Field>
        {kind === "single" && (
          <>
            <Field label="Número">
              <Input value={number} onChange={(e) => setNumber(e.target.value)} placeholder="OP16-080" />
            </Field>
            <Field label="Coleção">
              <Input value={set} onChange={(e) => setSet(e.target.value)} placeholder="OP16" />
            </Field>
            <Field label="Variante">
              <Input value={variant} onChange={(e) => setVariant(e.target.value)} placeholder="Foil" />
            </Field>
            <Field label="Condição">
              <Input value={condition} onChange={(e) => setCondition(e.target.value)} placeholder="NM" />
            </Field>
          </>
        )}
        <Field label="Un.">
          <Input value={qty} onChange={(e) => setQty(e.target.value)} inputMode="numeric" />
        </Field>
        <Field label="Preço pago R$ (un.)">
          <Input
            value={buyBRL}
            onChange={(e) => setBuyBRL(e.target.value)}
            inputMode="decimal"
            placeholder="0,00"
          />
        </Field>
        <Field label="Frete do item R$">
          <Input
            value={shipping}
            onChange={(e) => setShipping(e.target.value)}
            inputMode="decimal"
            placeholder="0,00"
          />
        </Field>
        <Field label={kind === "single" ? "Referência de mercado" : "Valor atual R$ (un.)"}>
          <Input
            value={refValue}
            onChange={(e) => setRefValue(e.target.value)}
            inputMode="decimal"
            placeholder="0,00"
          />
        </Field>
      </div>

      {failed && <p className="text-[11px] text-rose-300">{failed}</p>}
      <div className="flex justify-end gap-2">
        <Button variant="ghost" onClick={onCancel} disabled={saving}>
          Fechar
        </Button>
        <Button variant="primary" onClick={submit} disabled={!valid || saving}>
          {saving ? "Salvando…" : "Adicionar item"}
        </Button>
      </div>
      <p className="text-[10px] text-slate-500">
        O item entra no portfólio de {games.find((g) => g.id === game)?.name ?? game}, dentro da compra
        escolhida acima, e herda a data e o vendedor dela. Depois de salvar, o formulário fica aberto para o
        próximo item.
      </p>
    </Card>
  );
}

function LooseItems({
  items,
  orders,
  total,
  onChanged,
}: {
  items: PurchaseItem[];
  orders: PurchaseView[];
  total: number;
  onChanged: () => Promise<void> | void;
}) {
  const [picked, setPicked] = useState<Set<string>>(new Set());
  const [target, setTarget] = useState("");
  const [busy, setBusy] = useState(false);
  const [failed, setFailed] = useState<string | null>(null);
  const [sort, onSort] = useSort({ key: "date", dir: "desc" });
  const [collapsed, setCollapsed] = useState<Set<ItemKind>>(new Set());

  const toggle = (id: string) =>
    setPicked((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });

  const toggleGroup = (kind: ItemKind) =>
    setCollapsed((prev) => {
      const next = new Set(prev);
      if (next.has(kind)) {
        next.delete(kind);
      } else {
        next.add(kind);
      }
      return next;
    });

  // Picking a whole category at once is the point of the grouping here: bulk was
  // bought as one lot, so it gets attached to an order as one lot.
  const toggleAll = (rows: PurchaseItem[]) =>
    setPicked((prev) => {
      const next = new Set(prev);
      const allPicked = rows.every((it) => next.has(it.id));
      for (const it of rows) {
        if (allPicked) {
          next.delete(it.id);
        } else {
          next.add(it.id);
        }
      }
      return next;
    });

  const attach = async () => {
    if (!target || picked.size === 0) {
      return;
    }
    setBusy(true);
    setFailed(null);
    try {
      const refs: ItemRef[] = items
        .filter((it) => picked.has(it.id))
        .map((it) => ({ id: it.id, game: it.game }));
      await attachPurchaseItems(target, refs);
      setPicked(new Set());
      await onChanged();
    } catch (err: unknown) {
      setFailed(err instanceof Error ? err.message : "falha ao vincular");
    } finally {
      setBusy(false);
    }
  };

  if (total === 0) {
    return <EmptyState>Todo item lançado está dentro de uma compra.</EmptyState>;
  }
  if (items.length === 0) {
    return <EmptyState hint="ajuste a busca ou o jogo">Nenhum item avulso bate com o filtro.</EmptyState>;
  }

  // Anos de lançamentos avulsos são centenas de linhas, então cada categoria
  // mostra as primeiras e diz quantas ficaram de fora, em vez de travar a tela.
  const groups = GROUP_ORDER.map((kind) => {
    const all = sortItems(
      items.filter((it) => kindOf(it) === kind),
      sort,
    );
    return {
      kind,
      total: all.length,
      rows: all.slice(0, LOOSE_LIMIT),
      units: all.reduce((s, it) => s + it.qty, 0),
      cost: all.reduce((s, it) => s + it.costBRL, 0),
      open: !collapsed.has(kind),
    };
  }).filter((g) => g.total > 0);

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center gap-2">
        <span className="text-xs text-slate-500">
          {picked.size > 0 ? `${picked.size} selecionados` : "Selecione itens para vincular a uma compra"}
        </span>
        <div className="ml-auto flex items-center gap-2">
          <div className="w-64">
            <Select value={target} onChange={(e) => setTarget(e.target.value)}>
              <option value="">Escolha a compra…</option>
              {orders.map((o) => (
                <option key={o.id} value={o.id}>
                  {[o.date, o.store || "sem vendedor"].filter(Boolean).join(" · ")}
                </option>
              ))}
            </Select>
          </div>
          <Button variant="primary" onClick={attach} disabled={busy || !target || picked.size === 0}>
            <Link2 className="h-4 w-4" />
            Vincular
          </Button>
        </div>
      </div>
      {failed && <p className="text-[11px] text-rose-300">{failed}</p>}

      <div className="sticker sticker-sm overflow-x-auto rounded-[12px] bg-panel">
        <Table className="min-w-[760px]">
          <TableHeader>
            <TableRow className="hover:bg-transparent">
              <TableHead className="w-10" />
              <SortHead label="Item" sortKey="name" sort={sort} onSort={onSort} />
              <SortHead label="Jogo" sortKey="game" sort={sort} onSort={onSort} />
              <SortHead label="Un." sortKey="qty" sort={sort} onSort={onSort} className="text-right" />
              <SortHead label="Preço un." sortKey="unit" sort={sort} onSort={onSort} className="text-right" />
              <SortHead label="Custo" sortKey="cost" sort={sort} onSort={onSort} className="text-right" />
              <TableHead>Status</TableHead>
              <TableHead />
            </TableRow>
          </TableHeader>
          <TableBody>
            {groups.map((g) => (
              <Fragment key={g.kind}>
                <TableRow className="bg-raised/60">
                  <TableCell className="py-2">
                    <Checkbox
                      checked={g.rows.length > 0 && g.rows.every((it) => picked.has(it.id))}
                      onChange={() => toggleAll(g.rows)}
                      title="Selecionar os itens listados desta categoria"
                    />
                  </TableCell>
                  <TableCell colSpan={3} className="cursor-pointer py-2" onClick={() => toggleGroup(g.kind)}>
                    <div className="flex items-center gap-2">
                      {g.open ? (
                        <ChevronDown className="h-4 w-4 text-slate-500" />
                      ) : (
                        <ChevronRight className="h-4 w-4 text-slate-500" />
                      )}
                      <span className="font-semibold text-slate-200">{groupLabels[g.kind]}</span>
                      <span className="text-[11px] text-slate-500">
                        {g.total} {g.total === 1 ? "item" : "itens"}
                        {g.total > g.rows.length && ` · mostrando ${g.rows.length}`}
                      </span>
                    </div>
                  </TableCell>
                  <TableCell className="py-2 text-right tabular-nums text-slate-400">{g.units}</TableCell>
                  <TableCell className="py-2" />
                  <TableCell className="py-2 text-right font-semibold tabular-nums text-slate-200">
                    {brl2(g.cost)}
                  </TableCell>
                  <TableCell className="py-2" colSpan={2} />
                </TableRow>
                {g.open &&
                  g.rows.map((it) => (
                    <ItemRow
                      key={it.id}
                      item={it}
                      orders={orders}
                      showOrigin
                      selected={picked.has(it.id)}
                      onSelect={() => toggle(it.id)}
                      onChanged={onChanged}
                    />
                  ))}
              </Fragment>
            ))}
          </TableBody>
        </Table>
      </div>
      {groups.some((g) => g.total > g.rows.length) && (
        <p className="text-[11px] text-slate-500">
          Cada categoria mostra até {LOOSE_LIMIT} itens — use a busca ou o filtro de jogo para chegar no
          resto.
        </p>
      )}
    </div>
  );
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <label className="flex flex-col gap-1">
      <span className="text-[10px] font-medium uppercase tracking-wide text-slate-500">{label}</span>
      {children}
    </label>
  );
}
