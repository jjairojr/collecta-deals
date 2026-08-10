import { useCallback, useEffect, useId, useMemo, useState } from "react";
import {
  CalendarClock,
  ChevronLeft,
  ChevronRight,
  Coins,
  Pencil,
  Plus,
  Receipt,
  RefreshCw,
  Trash2,
  Wallet,
} from "lucide-react";
import {
  createExpense,
  deleteExpense,
  listExpenses,
  updateExpense,
  type Expense,
} from "../api";
import { brl2, dayLabel } from "../format";
import EmptyState from "./EmptyState";
import { Kpi } from "./PortfolioPage";
import { Button } from "./ui/button";
import { Card } from "./ui/card";
import { Input } from "./ui/input";
import { ToggleGroup } from "./ui/toggle-group";

const CATEGORY_SUGGESTIONS = [
  "Frete",
  "Embalagem",
  "Taxas",
  "Anúncios",
  "Material",
  "Aluguel",
  "Assinatura",
  "Outros",
];

// Months are compared as "YYYY-MM" strings; monthIndex turns one into a
// linear count so recurring spans (start → end) can be measured in months.
function monthOf(date: string): string {
  return date.slice(0, 7);
}

function monthIndex(month: string): number {
  const y = Number(month.slice(0, 4));
  const m = Number(month.slice(5, 7));
  return y * 12 + (m - 1);
}

function currentMonth(): string {
  return new Date().toISOString().slice(0, 7);
}

function shiftMonth(month: string, by: number): string {
  const idx = monthIndex(month) + by;
  const y = Math.floor(idx / 12);
  const m = idx - y * 12 + 1;
  return `${y}-${String(m).padStart(2, "0")}`;
}

function lastDayOfMonth(month: string): string {
  const y = Number(month.slice(0, 4));
  const m = Number(month.slice(5, 7));
  const last = new Date(y, m, 0).getDate();
  return `${month}-${String(last).padStart(2, "0")}`;
}

function monthLabel(month: string): string {
  const label = new Date(`${month}-01T12:00:00`).toLocaleDateString("pt-BR", {
    month: "long",
    year: "numeric",
  });
  return label.charAt(0).toUpperCase() + label.slice(1);
}

function activeInMonth(e: Expense, month: string): boolean {
  if (!e.recurring) {
    return monthOf(e.date) === month;
  }
  if (monthOf(e.date) > month) {
    return false;
  }
  return !e.endDate || month <= monthOf(e.endDate);
}

// monthsActive counts how many months a recurring expense has charged, from its
// start month through min(end month, the month being totalled).
function monthsActive(e: Expense, upTo: string): number {
  let end = upTo;
  if (e.endDate && monthOf(e.endDate) < end) {
    end = monthOf(e.endDate);
  }
  const span = monthIndex(end) - monthIndex(monthOf(e.date)) + 1;
  return Math.max(span, 0);
}

function allTimeTotal(expenses: Expense[]): number {
  const now = currentMonth();
  let total = 0;
  for (const e of expenses) {
    if (e.recurring) {
      total += e.amountBRL * monthsActive(e, now);
    } else {
      total += e.amountBRL;
    }
  }
  return total;
}

export default function ExpensesPage() {
  const [expenses, setExpenses] = useState<Expense[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [month, setMonth] = useState(currentMonth);
  const [adding, setAdding] = useState(false);

  const load = useCallback(async () => {
    try {
      const r = await listExpenses();
      setExpenses(r.expenses);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "failed to load expenses");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const fixed = useMemo(
    () => expenses.filter((e) => e.recurring && activeInMonth(e, month)),
    [expenses, month],
  );
  const oneOffs = useMemo(
    () => expenses.filter((e) => !e.recurring && activeInMonth(e, month)),
    [expenses, month],
  );
  const fixedTotal = useMemo(
    () => fixed.reduce((s, e) => s + e.amountBRL, 0),
    [fixed],
  );
  const oneOffTotal = useMemo(
    () => oneOffs.reduce((s, e) => s + e.amountBRL, 0),
    [oneOffs],
  );
  const allTime = useMemo(() => allTimeTotal(expenses), [expenses]);

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center gap-1">
          <button
            onClick={() => setMonth((m) => shiftMonth(m, -1))}
            className="rounded-[8px] border-2 border-outline bg-panel p-1.5 text-slate-400 hover:bg-raised hover:text-slate-200"
            title="Mês anterior"
          >
            <ChevronLeft className="h-4 w-4" />
          </button>
          <span className="min-w-[170px] text-center font-display text-base font-extrabold text-fg">
            {monthLabel(month)}
          </span>
          <button
            onClick={() => setMonth((m) => shiftMonth(m, 1))}
            className="rounded-[8px] border-2 border-outline bg-panel p-1.5 text-slate-400 hover:bg-raised hover:text-slate-200"
            title="Próximo mês"
          >
            <ChevronRight className="h-4 w-4" />
          </button>
          {month !== currentMonth() && (
            <button
              onClick={() => setMonth(currentMonth())}
              className="ml-2 text-xs font-medium text-sky-300 hover:text-sky-200"
            >
              Hoje
            </button>
          )}
        </div>
        <Button onClick={() => setAdding((a) => !a)}>
          <Plus /> {adding ? "Fechar" : "Nova despesa"}
        </Button>
      </div>

      <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
        <Kpi
          icon={<Wallet className="h-5 w-5" />}
          label="Total do mês"
          value={brl2(fixedTotal + oneOffTotal)}
          sub={monthLabel(month)}
        />
        <Kpi
          icon={<RefreshCw className="h-5 w-5" />}
          label="Fixas"
          value={brl2(fixedTotal)}
          sub={`${fixed.length} por mês`}
        />
        <Kpi
          icon={<Receipt className="h-5 w-5" />}
          label="Variáveis"
          value={brl2(oneOffTotal)}
          sub={`${oneOffs.length} lançamentos`}
        />
        <Kpi
          icon={<Coins className="h-5 w-5" />}
          label="Total all time"
          value={brl2(allTime)}
          sub={`${expenses.length} despesas cadastradas`}
        />
      </div>

      {error && (
        <div className="rounded-[14px] border-2 border-rose-900/50 bg-rose-950/30 px-4 py-3 text-sm text-rose-200">
          {error}
        </div>
      )}

      {adding && (
        <AddExpenseForm
          month={month}
          onAdded={() => {
            setAdding(false);
            load();
          }}
        />
      )}

      {loading ? (
        <EmptyState>Carregando despesas…</EmptyState>
      ) : (
        <div className="space-y-6">
          <section className="space-y-3">
            <h2 className="font-display text-base font-extrabold text-fg">
              Fixas do mês{" "}
              <span className="font-normal text-slate-500">
                · {fixed.length}
              </span>
            </h2>
            {fixed.length === 0 ? (
              <EmptyState>
                Nenhuma despesa fixa ativa em {monthLabel(month)}.
              </EmptyState>
            ) : (
              <ExpenseTable
                expenses={fixed}
                month={month}
                recurring
                onChanged={load}
              />
            )}
          </section>
          <section className="space-y-3">
            <h2 className="font-display text-base font-extrabold text-fg">
              Lançamentos do mês{" "}
              <span className="font-normal text-slate-500">
                · {oneOffs.length}
              </span>
            </h2>
            {oneOffs.length === 0 ? (
              <EmptyState>
                Nenhum lançamento em {monthLabel(month)}. Clique em “Nova
                despesa” para registrar.
              </EmptyState>
            ) : (
              <ExpenseTable expenses={oneOffs} month={month} onChanged={load} />
            )}
          </section>
        </div>
      )}
    </div>
  );
}

function ExpenseTable({
  expenses,
  month,
  recurring,
  onChanged,
}: {
  expenses: Expense[];
  month: string;
  recurring?: boolean;
  onChanged: () => void;
}) {
  return (
    <div className="sticker sticker-sm overflow-x-auto rounded-[12px] bg-panel">
      <table className="w-full min-w-[640px] text-sm">
        <thead>
          <tr className="font-pixel border-b-2 border-outline bg-raised text-left text-[8px] uppercase text-brand-label">
            <th className="px-3 py-2 font-bold">Despesa</th>
            <th className="px-3 py-2 font-bold">{recurring ? "Desde" : "Data"}</th>
            <th className="px-3 py-2 text-right font-bold">
              {recurring ? "Valor / mês" : "Valor"}
            </th>
            <th className="px-3 py-2" />
          </tr>
        </thead>
        <tbody>
          {expenses.map((e) => (
            <ExpenseRow
              key={e.id}
              e={e}
              month={month}
              onChanged={onChanged}
            />
          ))}
        </tbody>
      </table>
    </div>
  );
}

function ExpenseRow({
  e,
  month,
  onChanged,
}: {
  e: Expense;
  month: string;
  onChanged: () => void;
}) {
  const [editing, setEditing] = useState(false);

  const endRecurring = async () => {
    if (!confirm(`Encerrar "${e.description}" em ${monthLabel(month)}?`)) {
      return;
    }
    await updateExpense(e.id, { ...e, endDate: lastDayOfMonth(month) });
    onChanged();
  };

  return (
    <>
      <tr className="border-b-2 border-outline/15 last:border-0 hover:bg-raised/70">
        <td className="px-3 py-2">
          <div className="flex items-center gap-2">
            <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-[8px] border-2 border-outline bg-brand text-white">
              {e.recurring ? (
                <CalendarClock className="h-4 w-4" />
              ) : (
                <Receipt className="h-4 w-4" />
              )}
            </div>
            <div className="min-w-0">
              <div className="truncate font-medium text-slate-100" title={e.description}>
                {e.description}
              </div>
              <div className="truncate text-[10px] text-slate-500">
                {[e.category, e.store, e.notes].filter(Boolean).join(" · ")}
                {e.recurring && e.endDate
                  ? `${e.category || e.store || e.notes ? " · " : ""}até ${monthLabel(monthOf(e.endDate))}`
                  : ""}
              </div>
            </div>
          </div>
        </td>
        <td className="px-3 py-2 tabular-nums text-slate-400">
          {e.recurring ? monthLabel(monthOf(e.date)) : dayLabel(e.date)}
        </td>
        <td className="px-3 py-2 text-right font-semibold tabular-nums text-slate-200">
          {brl2(e.amountBRL)}
        </td>
        <td className="px-3 py-2">
          <div className="flex items-center justify-end gap-1">
            {e.recurring && !e.endDate && (
              <button
                onClick={endRecurring}
                className="rounded-[8px] border-2 border-outline bg-amber-500/10 px-2 py-1 text-xs font-bold text-amber-300 hover:bg-amber-500/20"
                title="Parar de contar esta despesa a partir do mês seguinte"
              >
                Encerrar
              </button>
            )}
            <button
              onClick={() => setEditing((v) => !v)}
              className="flex items-center gap-1 rounded-[8px] border-2 border-outline bg-panel px-2 py-1 text-xs font-bold text-slate-200 hover:bg-raised"
            >
              <Pencil className="h-3 w-3" /> Editar
            </button>
            <button
              onClick={() => {
                if (confirm("Apagar esta despesa?")) {
                  deleteExpense(e.id).then(onChanged);
                }
              }}
              className="rounded-[8px] border-2 border-outline bg-panel p-1.5 text-slate-400 hover:bg-rose-950/40 hover:text-rose-300"
              title="Apagar"
            >
              <Trash2 className="h-3.5 w-3.5" />
            </button>
          </div>
        </td>
      </tr>
      {editing && (
        <tr className="bg-slate-900/60">
          <td colSpan={4} className="px-3 py-3">
            <ExpenseForm
              initial={e}
              onDone={() => {
                setEditing(false);
                onChanged();
              }}
            />
          </td>
        </tr>
      )}
    </>
  );
}

function AddExpenseForm({
  month,
  onAdded,
}: {
  month: string;
  onAdded: () => void;
}) {
  const defaultDate =
    month === currentMonth()
      ? new Date().toISOString().slice(0, 10)
      : `${month}-01`;
  return (
    <Card className="p-4">
      <ExpenseForm
        initial={{ date: defaultDate }}
        onDone={onAdded}
      />
    </Card>
  );
}

// ExpenseForm handles both create (initial without id) and edit (initial is a
// full Expense) so row editing and the add card share one set of fields.
function ExpenseForm({
  initial,
  onDone,
}: {
  initial: Partial<Expense>;
  onDone: () => void;
}) {
  const listId = useId();
  const [recurring, setRecurring] = useState(Boolean(initial.recurring));
  const [date, setDate] = useState(initial.date ?? "");
  const [endDate, setEndDate] = useState(initial.endDate ?? "");
  const [description, setDescription] = useState(initial.description ?? "");
  const [category, setCategory] = useState(initial.category ?? "");
  const [amount, setAmount] = useState(
    initial.amountBRL ? String(initial.amountBRL) : "",
  );
  const [store, setStore] = useState(initial.store ?? "");
  const [notes, setNotes] = useState(initial.notes ?? "");
  const [saving, setSaving] = useState(false);

  const valid = description.trim() !== "" && Number(amount) > 0 && date !== "";

  const submit = async () => {
    setSaving(true);
    try {
      const body = {
        date,
        recurring,
        endDate: recurring ? endDate : "",
        description: description.trim(),
        category: category.trim(),
        amountBRL: Number(amount),
        store: store.trim(),
        notes: notes.trim(),
      };
      if (initial.id) {
        await updateExpense(initial.id, body);
      } else {
        await createExpense(body);
      }
      onDone();
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="flex flex-wrap items-end gap-2">
      <Field label="Tipo">
        <ToggleGroup
          value={recurring ? "fixed" : "single"}
          onChange={(v) => setRecurring(v === "fixed")}
          options={[
            { value: "single", label: "Única" },
            { value: "fixed", label: "Fixa mensal" },
          ]}
        />
      </Field>
      <Field label={recurring ? "Desde" : "Data"}>
        <Input
          type="date"
          value={date}
          onChange={(e) => setDate(e.target.value)}
          className="w-40"
        />
      </Field>
      {recurring && (
        <Field label="Até (opcional)">
          <Input
            type="date"
            value={endDate}
            onChange={(e) => setEndDate(e.target.value)}
            className="w-40"
          />
        </Field>
      )}
      <Field label="Descrição">
        <Input
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          placeholder={recurring ? "Aluguel da loja" : "Envelopes e bubble wrap"}
          className="w-56"
        />
      </Field>
      <Field label="Categoria">
        <Input
          value={category}
          onChange={(e) => setCategory(e.target.value)}
          list={listId}
          placeholder="Frete"
          className="w-36"
        />
        <datalist id={listId}>
          {CATEGORY_SUGGESTIONS.map((c) => (
            <option key={c} value={c} />
          ))}
        </datalist>
      </Field>
      <Field label={recurring ? "Valor R$ / mês" : "Valor R$"}>
        <Input
          type="number"
          value={amount}
          onChange={(e) => setAmount(e.target.value)}
          placeholder="0,00"
          className="w-32"
        />
      </Field>
      <Field label="Loja / fornecedor">
        <Input
          value={store}
          onChange={(e) => setStore(e.target.value)}
          placeholder="Correios"
          className="w-40"
        />
      </Field>
      <Field label="Notas">
        <Input
          value={notes}
          onChange={(e) => setNotes(e.target.value)}
          className="w-48"
        />
      </Field>
      <Button onClick={submit} disabled={!valid || saving}>
        {saving ? "Salvando…" : initial.id ? "Salvar" : "Adicionar"}
      </Button>
    </div>
  );
}

function Field({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <label className="flex flex-col gap-1">
      <span className="text-[10px] font-medium uppercase tracking-wide text-slate-500">
        {label}
      </span>
      {children}
    </label>
  );
}
