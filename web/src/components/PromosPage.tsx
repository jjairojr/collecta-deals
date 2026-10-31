import { useCallback, useEffect, useState } from "react";
import { Coins, Pencil, Plus, Ticket, Trash2, X } from "lucide-react";
import { createPromo, deletePromo, listPromos, updatePromo, type Promo, type PromoInput, type PromoKind } from "../api";
import { brl2, dayLabel } from "../format";
import EmptyState from "./EmptyState";
import { Button } from "./ui/button";
import { Card } from "./ui/card";
import { Checkbox } from "./ui/checkbox";
import { Input } from "./ui/input";
import { Kpi } from "./ui/kpi";
import { Select } from "./ui/select";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "./ui/table";

function num(v: string): number {
  const n = Number(v.replace(",", "."));
  return Number.isFinite(n) ? n : 0;
}

function centsText(c: number | undefined): string {
  return c ? String(c / 100).replace(".", ",") : "";
}

function discountLabel(p: PromoInput): string {
  return p.kind === "fixed" ? `${brl2((p.valueCents ?? 0) / 100)} nos itens` : `${(p.percentBps ?? 0) / 100}% nos itens`;
}

function rulesLabel(p: PromoInput): string {
  return [
    p.firstOrderOnly ? "só 1ª compra" : "",
    p.minItemsCents ? `pedido a partir de ${brl2(p.minItemsCents / 100)}` : "",
  ]
    .filter(Boolean)
    .join(" · ");
}

function validityLabel(p: PromoInput): string {
  if (p.validFrom && p.validUntil) {
    return `${dayLabel(p.validFrom)} a ${dayLabel(p.validUntil)}`;
  }
  if (p.validFrom) {
    return `a partir de ${dayLabel(p.validFrom)}`;
  }
  if (p.validUntil) {
    return `até ${dayLabel(p.validUntil)}`;
  }
  return "sem prazo";
}

export default function PromosPage() {
  const [promos, setPromos] = useState<Promo[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [editing, setEditing] = useState<Promo | "new" | null>(null);
  const [busy, setBusy] = useState<string | null>(null);

  const load = useCallback(async () => {
    try {
      const r = await listPromos();
      setPromos(r.promos);
      setError(null);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "falha ao carregar os cupons");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const run = async (code: string, fn: () => Promise<unknown>) => {
    setBusy(code);
    setError(null);
    try {
      await fn();
      await load();
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "falha ao salvar");
    } finally {
      setBusy(null);
    }
  };

  const remove = (p: Promo) => {
    if (!window.confirm(`Excluir o cupom ${p.code}? Isso não pode ser desfeito.`)) {
      return;
    }
    run(p.code, () => deletePromo(p.code));
  };

  if (loading) {
    return <EmptyState>Carregando cupons…</EmptyState>;
  }

  const active = promos.filter((p) => p.active).length;
  const uses = promos.reduce((s, p) => s + p.uses, 0);
  const discount = promos.reduce((s, p) => s + p.discountBRL, 0);
  const revenue = promos.reduce((s, p) => s + p.revenueBRL, 0);

  return (
    <div className="space-y-6">
      <p className="max-w-2xl text-sm text-slate-400">
        Códigos de desconto que o cliente digita no checkout da vitrine. Valem só nos itens, nunca no frete, exigem
        login e não acumulam com os cupons de indicação do app. Um cupom que já foi usado não pode ser excluído:
        desative.
      </p>

      <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
        <Kpi icon={<Ticket className="h-5 w-5" />} label="Cupons ativos" value={String(active)} sub={`${promos.length} no total`} />
        <Kpi icon={<Ticket className="h-5 w-5" />} label="Usos" value={String(uses)} sub="pedidos com cupom" />
        <Kpi icon={<Coins className="h-5 w-5" />} label="Desconto dado" value={brl2(discount)} sub="somando todos os cupons" />
        <Kpi icon={<Coins className="h-5 w-5" />} label="Vendido com cupom" value={brl2(revenue)} sub="total pago nesses pedidos" />
      </div>

      <div className="flex items-center gap-2">
        {error && <p className="text-[11px] text-rose-300">{error}</p>}
        <div className="ml-auto">
          {editing === null && (
            <Button variant="primary" onClick={() => setEditing("new")}>
              <Plus className="h-4 w-4" />
              Novo cupom
            </Button>
          )}
        </div>
      </div>

      {editing !== null && (
        <PromoForm
          initial={editing === "new" ? undefined : editing}
          onCancel={() => setEditing(null)}
          onSaved={async () => {
            setEditing(null);
            await load();
          }}
        />
      )}

      {promos.length === 0 ? (
        <EmptyState hint="clique em Novo cupom">Nenhum cupom criado ainda.</EmptyState>
      ) : (
        <div className="sticker sticker-sm overflow-x-auto rounded-[12px] bg-panel">
          <Table className="min-w-[860px]">
            <TableHeader>
              <TableRow className="hover:bg-transparent">
                <TableHead>Código</TableHead>
                <TableHead>Desconto</TableHead>
                <TableHead>Regras</TableHead>
                <TableHead>Validade</TableHead>
                <TableHead className="text-right">Usos</TableHead>
                <TableHead className="text-right">Desconto dado</TableHead>
                <TableHead>Ativo</TableHead>
                <TableHead />
              </TableRow>
            </TableHeader>
            <TableBody>
              {promos.map((p) => (
                <TableRow key={p.code} className={p.active ? undefined : "opacity-60"}>
                  <TableCell>
                    <div className="font-mono font-semibold text-slate-100">{p.code}</div>
                    {p.notes && <div className="truncate text-[10px] text-slate-500">{p.notes}</div>}
                  </TableCell>
                  <TableCell className="whitespace-nowrap text-xs text-slate-200">{discountLabel(p)}</TableCell>
                  <TableCell className="text-[11px] text-slate-400">{rulesLabel(p) || "—"}</TableCell>
                  <TableCell className="whitespace-nowrap text-[11px] text-slate-400">{validityLabel(p)}</TableCell>
                  <TableCell className="text-right tabular-nums text-slate-200">
                    {p.uses}
                    {p.maxUses ? <span className="text-slate-500"> / {p.maxUses}</span> : null}
                  </TableCell>
                  <TableCell className="text-right tabular-nums text-slate-200">{brl2(p.discountBRL)}</TableCell>
                  <TableCell>
                    <button
                      type="button"
                      disabled={busy === p.code}
                      onClick={() => run(p.code, () => updatePromo(p.code, { active: !p.active }))}
                      title={p.active ? "Ativo — clique para desativar" : "Inativo — clique para ativar"}
                      className={`rounded-[8px] border-2 border-outline px-2 py-1 text-xs font-bold disabled:opacity-50 ${
                        p.active
                          ? "bg-emerald-500/10 text-emerald-300 hover:bg-emerald-500/20"
                          : "bg-panel text-slate-400 hover:text-slate-200"
                      }`}
                    >
                      {p.active ? "Ativo" : "Inativo"}
                    </button>
                  </TableCell>
                  <TableCell className="text-right">
                    <div className="flex items-center justify-end gap-1">
                      <Button variant="ghost" size="sm" disabled={busy === p.code} title="Editar cupom" onClick={() => setEditing(p)}>
                        <Pencil className="h-4 w-4" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        disabled={busy === p.code || p.uses > 0}
                        title={p.uses > 0 ? "Já foi usado: desative em vez de excluir" : "Excluir cupom"}
                        onClick={() => remove(p)}
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
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

function PromoForm({
  initial,
  onCancel,
  onSaved,
}: {
  initial?: Promo;
  onCancel: () => void;
  onSaved: () => Promise<void> | void;
}) {
  const [code, setCode] = useState(initial?.code ?? "");
  const [kind, setKind] = useState<PromoKind>(initial?.kind ?? "percent");
  const [percent, setPercent] = useState(initial?.percentBps ? String(initial.percentBps / 100).replace(".", ",") : "");
  const [value, setValue] = useState(centsText(initial?.valueCents));
  const [firstOrderOnly, setFirstOrderOnly] = useState(initial?.firstOrderOnly ?? false);
  const [minItems, setMinItems] = useState(centsText(initial?.minItemsCents));
  const [validFrom, setValidFrom] = useState(initial?.validFrom ?? "");
  const [validUntil, setValidUntil] = useState(initial?.validUntil ?? "");
  const [maxUses, setMaxUses] = useState(initial?.maxUses ? String(initial.maxUses) : "");
  const [active, setActive] = useState(initial?.active ?? true);
  const [notes, setNotes] = useState(initial?.notes ?? "");
  const [saving, setSaving] = useState(false);
  const [failed, setFailed] = useState<string | null>(null);

  const submit = async () => {
    setSaving(true);
    setFailed(null);
    const body: PromoInput = {
      code: code.trim().toUpperCase(),
      kind,
      percentBps: kind === "percent" ? Math.round(num(percent) * 100) : 0,
      valueCents: kind === "fixed" ? Math.round(num(value) * 100) : 0,
      firstOrderOnly,
      minItemsCents: Math.round(num(minItems) * 100),
      validFrom,
      validUntil,
      maxUses: Math.max(0, Math.floor(num(maxUses))),
      active,
      notes: notes.trim(),
    };
    try {
      if (initial) {
        await updatePromo(initial.code, body);
      } else {
        await createPromo(body);
      }
      await onSaved();
    } catch (err: unknown) {
      setFailed(err instanceof Error ? err.message : "falha ao salvar");
    } finally {
      setSaving(false);
    }
  };

  return (
    <Card className="space-y-3 p-4">
      <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
        <Field label="Código">
          <Input
            value={code}
            onChange={(e) => setCode(e.target.value.toUpperCase())}
            placeholder="ex.: NATAL10"
            disabled={Boolean(initial)}
          />
        </Field>
        <Field label="Tipo de desconto">
          <Select value={kind} onChange={(e) => setKind(e.target.value === "fixed" ? "fixed" : "percent")}>
            <option value="percent">Percentual</option>
            <option value="fixed">Valor fixo R$</option>
          </Select>
        </Field>
        {kind === "percent" ? (
          <Field label="Desconto %">
            <Input value={percent} onChange={(e) => setPercent(e.target.value)} inputMode="decimal" placeholder="10" />
          </Field>
        ) : (
          <Field label="Desconto R$">
            <Input value={value} onChange={(e) => setValue(e.target.value)} inputMode="decimal" placeholder="25,00" />
          </Field>
        )}
        <Field label="Pedido mínimo R$ (itens)">
          <Input value={minItems} onChange={(e) => setMinItems(e.target.value)} inputMode="decimal" placeholder="opcional" />
        </Field>
        <Field label="Vale a partir de">
          <Input type="date" value={validFrom} onChange={(e) => setValidFrom(e.target.value)} />
        </Field>
        <Field label="Vale até">
          <Input type="date" value={validUntil} onChange={(e) => setValidUntil(e.target.value)} />
        </Field>
        <Field label="Limite de usos">
          <Input value={maxUses} onChange={(e) => setMaxUses(e.target.value)} inputMode="numeric" placeholder="sem limite" />
        </Field>
        <Field label="Observação">
          <Input value={notes} onChange={(e) => setNotes(e.target.value)} placeholder="opcional" />
        </Field>
      </div>
      <div className="flex flex-wrap items-center gap-4 text-sm text-slate-300">
        <label className="flex items-center gap-2">
          <Checkbox checked={firstOrderOnly} onChange={(e) => setFirstOrderOnly(e.target.checked)} />
          Só na primeira compra
        </label>
        <label className="flex items-center gap-2">
          <Checkbox checked={active} onChange={(e) => setActive(e.target.checked)} />
          Ativo
        </label>
      </div>
      {failed && <p className="text-[11px] text-rose-300">{failed}</p>}
      <div className="flex justify-end gap-2">
        <Button variant="ghost" onClick={onCancel} disabled={saving}>
          <X className="h-4 w-4" />
          Cancelar
        </Button>
        <Button variant="primary" onClick={submit} disabled={saving || !code.trim()}>
          {saving ? "Salvando…" : initial ? "Salvar cupom" : "Criar cupom"}
        </Button>
      </div>
    </Card>
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
