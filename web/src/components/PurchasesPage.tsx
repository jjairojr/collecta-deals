import { useCallback, useEffect, useMemo, useState } from "react";
import {
  Boxes,
  ChevronDown,
  ChevronRight,
  Coins,
  FolderInput,
  Link2,
  Package,
  Pencil,
  Plus,
  ShoppingCart,
  Trash2,
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
  productIDFromTcgURL,
  updatePurchase,
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

function kindOf(item: PurchaseItem): ItemKind {
  return item.kind === "sealed" ? "sealed" : item.kind === "accessory" ? "accessory" : "single";
}

function today(): string {
  return new Date().toISOString().slice(0, 10);
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
  const byGame = useCallback(
    (item: PurchaseItem) => gameFilter === "all" || item.game === gameFilter,
    [gameFilter],
  );

  const visibleOrders = useMemo(
    () =>
      orders
        .filter((o) => (gameFilter === "all" ? true : o.items.some(byGame)))
        .filter((o) => matchesOrder(o, q)),
    [orders, q, gameFilter, byGame],
  );
  const visibleLoose = useMemo(
    () => loose.filter(byGame).filter((it) => matchesItem(it, q)),
    [loose, q, byGame],
  );

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
        Cada compra é um pedido: você abre com a data, o vendedor e o frete, e vai lançando os itens
        dentro dela — de qualquer jogo. O item entra no portfólio do jogo que você escolher, e o
        total aqui já soma o frete do pedido.
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
      const body = { date, store: store.trim(), shippingBRL: num(shipping), notes: notes.trim() };
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
          <Input value={store} onChange={(e) => setStore(e.target.value)} placeholder="ex.: Legends, Fulano (WhatsApp)" />
        </Field>
        <Field label="Frete do pedido R$">
          <Input value={shipping} onChange={(e) => setShipping(e.target.value)} inputMode="decimal" placeholder="0,00" />
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
        <button
          type="button"
          onClick={onToggle}
          className="flex min-w-0 flex-1 items-center gap-2 text-left"
        >
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
            {order.shippingBRL ? `itens ${brl2(order.itemsCostBRL)} + frete ${brl2(order.shippingBRL)}` : "itens + frete"}
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
            <div className="overflow-x-auto">
              <Table className="min-w-[720px]">
                <TableHeader>
                  <TableRow className="hover:bg-transparent">
                    <TableHead>Item</TableHead>
                    <TableHead>Jogo</TableHead>
                    <TableHead className="text-right">Un.</TableHead>
                    <TableHead className="text-right">Preço un.</TableHead>
                    <TableHead className="text-right">Custo</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead />
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {order.items.map((it) => (
                    <ItemRow key={it.id} item={it} orders={orders} onChanged={onChanged} />
                  ))}
                </TableBody>
              </Table>
            </div>
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
  onChanged,
}: {
  item: PurchaseItem;
  orders: PurchaseView[];
  onChanged: () => Promise<void> | void;
}) {
  const [busy, setBusy] = useState(false);
  const [moving, setMoving] = useState(false);
  const unit = item.qty > 0 ? item.buyBRL : 0;
  const others = orders.filter((o) => o.id !== item.purchaseID);

  const run = async (fn: () => Promise<unknown>) => {
    setBusy(true);
    try {
      await fn();
      await onChanged();
    } finally {
      setBusy(false);
    }
  };

  if (moving) {
    return (
      <TableRow className="hover:bg-transparent">
        <TableCell colSpan={7}>
          <MoveItem
            item={item}
            orders={others}
            busy={busy}
            onCancel={() => setMoving(false)}
            onMove={async (target) => {
              await run(() => attachPurchaseItems(target, [{ id: item.id, game: item.game }]));
              setMoving(false);
            }}
          />
        </TableCell>
      </TableRow>
    );
  }

  return (
    <TableRow>
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
              {[item.number, item.set, item.variant, kindLabels[kindOf(item)]].filter(Boolean).join(" · ")}
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
          <span className="text-emerald-300">vendido {brl2(item.valueBRL)}</span>
        ) : (
          <span className="text-slate-500">em estoque</span>
        )}
      </TableCell>
      <TableCell className="text-right">
        <div className="flex items-center justify-end gap-1">
          <Button
            variant="ghost"
            size="sm"
            disabled={busy || others.length === 0}
            title={
              others.length === 0
                ? "Não há outra compra para mover este item"
                : "Mover para outra compra"
            }
            onClick={() => setMoving(true)}
          >
            <FolderInput className="h-4 w-4" />
          </Button>
          <Button
            variant="ghost"
            size="sm"
            disabled={busy}
            title="Tirar desta compra (o item continua no portfólio)"
            onClick={() => run(() => detachPurchaseItems([{ id: item.id, game: item.game }]))}
          >
            <Unlink className="h-4 w-4" />
          </Button>
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
          <Input value={buyBRL} onChange={(e) => setBuyBRL(e.target.value)} inputMode="decimal" placeholder="0,00" />
        </Field>
        <Field label="Frete do item R$">
          <Input value={shipping} onChange={(e) => setShipping(e.target.value)} inputMode="decimal" placeholder="0,00" />
        </Field>
        <Field label={kind === "single" ? "Referência de mercado" : "Valor atual R$ (un.)"}>
          <Input value={refValue} onChange={(e) => setRefValue(e.target.value)} inputMode="decimal" placeholder="0,00" />
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
        escolhida acima, e herda a data e o vendedor dela. Depois de salvar, o formulário fica aberto
        para o próximo item.
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

  // Anos de lançamentos avulsos são centenas de linhas; a lista mostra as
  // primeiras e diz quantas ficaram de fora, em vez de travar a tela.
  const shown = items.slice(0, LOOSE_LIMIT);

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
              <TableHead>Item</TableHead>
              <TableHead>Jogo</TableHead>
              <TableHead>Comprado em</TableHead>
              <TableHead>Vendedor</TableHead>
              <TableHead className="text-right">Un.</TableHead>
              <TableHead className="text-right">Custo</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {shown.map((it) => (
              <TableRow key={it.id}>
                <TableCell>
                  <Checkbox checked={picked.has(it.id)} onChange={() => toggle(it.id)} />
                </TableCell>
                <TableCell>
                  <div className="flex items-center gap-2">
                    <CardArt
                      set={it.set}
                      number={it.number}
                      name={it.name}
                      game={it.game}
                      imageURL={it.imageURL}
                      productID={productIDFromTcgURL(it.tcgUrl)}
                      className="h-12 w-[34px] shrink-0 rounded"
                    />
                    <div className="min-w-0">
                      <div className="truncate text-slate-100">{it.name}</div>
                      <div className="truncate text-[10px] text-slate-500">
                        {[it.number, it.set, kindLabels[kindOf(it)]].filter(Boolean).join(" · ")}
                      </div>
                    </div>
                  </div>
                </TableCell>
                <TableCell className="whitespace-nowrap text-[11px] text-slate-400">{it.gameName}</TableCell>
                <TableCell className="whitespace-nowrap text-[11px] text-slate-400">
                  {it.buyDate ? dayLabel(it.buyDate) : "—"}
                </TableCell>
                <TableCell className="whitespace-nowrap text-[11px] text-slate-400">{it.store || "—"}</TableCell>
                <TableCell className="text-right tabular-nums text-slate-400">{it.qty}</TableCell>
                <TableCell className="text-right font-semibold tabular-nums text-slate-200">
                  {brl2(it.costBRL)}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
      {items.length > shown.length && (
        <p className="text-[11px] text-slate-500">
          Mostrando {shown.length} de {items.length} itens avulsos — use a busca ou o filtro de jogo
          para chegar no resto.
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
