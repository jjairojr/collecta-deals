import { useCallback, useEffect, useMemo, useState } from "react";
import { Plus, Trash2 } from "lucide-react";
import {
  getVitrine,
  listVitrineProducts,
  saveVitrine,
  type VitrineBanner,
  type VitrineConfig,
  type VitrinePhrase,
  type VitrinePopup,
  type VitrineProduct,
  type VitrineTier,
} from "../api";
import { Button } from "./ui/button";
import { Card } from "./ui/card";
import { Checkbox } from "./ui/checkbox";
import { Input } from "./ui/input";

const MAX_PHRASES = 6;
const MAX_TIERS = 4;

function errorText(err: unknown): string {
  const raw = err instanceof Error ? err.message : "falha ao salvar";
  try {
    const parsed = JSON.parse(raw) as { error?: string };
    return (parsed.error ?? raw).replace(/^vitrine inválida\n?/, "");
  } catch {
    return raw;
  }
}

function emptyBanner(): VitrineBanner {
  return { title: "", text: "", cta: "", href: "", tiers: [] };
}

function emptyPopup(): VitrinePopup {
  return { enabled: false, sku: "", title: "", text: "" };
}

// Drops the empty optional fields so the server stores only what was filled.
function clean(c: VitrineConfig): VitrineConfig {
  const phrases = c.phrases.filter((p) => p.text.trim());
  const b = c.banner;
  const banner = b && b.title.trim() ? { ...b, tiers: (b.tiers ?? []).filter((t) => t.label.trim() || t.href.trim()) } : undefined;
  const p = c.popup;
  const popup = p && (p.enabled || p.sku.trim() || p.title.trim()) ? p : undefined;
  return { phrases, banner, popup };
}

/**
 * Tela "Vitrine": the owner edits the storefront's campaign content without a
 * deploy. The site picks it up within about a minute and shows each piece only
 * between its dates (Goiânia).
 */
export default function VitrinePage() {
  const [config, setConfig] = useState<VitrineConfig | null>(null);
  const [products, setProducts] = useState<VitrineProduct[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [saved, setSaved] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  const load = useCallback(async () => {
    try {
      const [c, list] = await Promise.all([getVitrine(), listVitrineProducts().catch(() => [] as VitrineProduct[])]);
      setConfig({ phrases: c.phrases ?? [], banner: c.banner ?? emptyBanner(), popup: c.popup ?? emptyPopup(), updatedAt: c.updatedAt });
      setProducts(list);
      setError(null);
    } catch (err: unknown) {
      setError(errorText(err));
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const save = async () => {
    if (!config) {
      return;
    }
    setSaving(true);
    setError(null);
    setSaved(null);
    try {
      const out = await saveVitrine(clean(config));
      setConfig({ phrases: out.phrases ?? [], banner: out.banner ?? emptyBanner(), popup: out.popup ?? emptyPopup(), updatedAt: out.updatedAt });
      setSaved("Salvo. O site mostra em até 1 minuto, dentro das datas.");
    } catch (err: unknown) {
      setError(errorText(err));
    } finally {
      setSaving(false);
    }
  };

  if (!config) {
    return <p className="text-sm text-slate-400">{error ?? "Carregando…"}</p>;
  }

  const setPhrases = (phrases: VitrinePhrase[]) => setConfig({ ...config, phrases });
  const banner = config.banner ?? emptyBanner();
  const setBanner = (b: Partial<VitrineBanner>) => setConfig({ ...config, banner: { ...banner, ...b } });
  const tiers = banner.tiers ?? [];
  const setTiers = (t: VitrineTier[]) => setBanner({ tiers: t });
  const popup = config.popup ?? emptyPopup();
  const setPopup = (p: Partial<VitrinePopup>) => setConfig({ ...config, popup: { ...popup, ...p } });

  return (
    <div className="space-y-4">
      <Card className="space-y-3 p-4">
        <SectionTitle title="Faixa do topo" hint="Frases extras que entram antes das fixas, cada uma com datas. Até 70 letras." />
        {config.phrases.map((p, i) => (
          <div key={i} className="grid grid-cols-1 gap-2 lg:grid-cols-[1fr_160px_160px_auto]">
            <Input
              value={p.text}
              maxLength={70}
              placeholder="ex.: Para chegar até 24/12, peça até 09/12"
              onChange={(e) => setPhrases(config.phrases.map((x, j) => (j === i ? { ...x, text: e.target.value } : x)))}
            />
            <DateField label="de" value={p.from} onChange={(v) => setPhrases(config.phrases.map((x, j) => (j === i ? { ...x, from: v } : x)))} />
            <DateField label="até" value={p.until} onChange={(v) => setPhrases(config.phrases.map((x, j) => (j === i ? { ...x, until: v } : x)))} />
            <Button variant="ghost" onClick={() => setPhrases(config.phrases.filter((_, j) => j !== i))} aria-label="Remover frase">
              <Trash2 className="h-4 w-4" />
            </Button>
          </div>
        ))}
        {config.phrases.length < MAX_PHRASES && (
          <Button variant="ghost" onClick={() => setPhrases([...config.phrases, { text: "" }])}>
            <Plus className="h-4 w-4" />
            Frase
          </Button>
        )}
      </Card>

      <Card className="space-y-3 p-4">
        <SectionTitle title="Banner da home" hint="Aparece no topo da home entre as datas. Sem título, não aparece." />
        <div className="grid grid-cols-1 gap-2 lg:grid-cols-2">
          <Field label="Título">
            <Input value={banner.title} maxLength={60} placeholder="ex.: Presentes de Natal" onChange={(e) => setBanner({ title: e.target.value })} />
          </Field>
          <Field label="Texto">
            <Input value={banner.text ?? ""} maxLength={160} placeholder="opcional" onChange={(e) => setBanner({ text: e.target.value })} />
          </Field>
          <Field label="Botão (texto)">
            <Input value={banner.cta ?? ""} maxLength={24} placeholder="opcional, ex.: Ver presentes" onChange={(e) => setBanner({ cta: e.target.value })} />
          </Field>
          <Field label="Botão (link do site)">
            <Input value={banner.href ?? ""} placeholder="ex.: /selado/pokemon" onChange={(e) => setBanner({ href: e.target.value })} />
          </Field>
          <DateField label="de" value={banner.from} onChange={(v) => setBanner({ from: v })} />
          <DateField label="até" value={banner.until} onChange={(v) => setBanner({ until: v })} />
        </div>
        <p className="text-[11px] text-slate-500">
          Blocos de presente por faixa de preço. Nas cartas avulsas, o link filtra por preço em reais: /singles?preco=0-100,
          /singles/pokemon?preco=100-300.
        </p>
        {tiers.map((t, i) => (
          <div key={i} className="grid grid-cols-1 gap-2 lg:grid-cols-[220px_1fr_auto]">
            <Input
              value={t.label}
              maxLength={30}
              placeholder="ex.: Até R$ 100"
              onChange={(e) => setTiers(tiers.map((x, j) => (j === i ? { ...x, label: e.target.value } : x)))}
            />
            <Input
              value={t.href}
              placeholder="ex.: /singles?preco=0-100"
              onChange={(e) => setTiers(tiers.map((x, j) => (j === i ? { ...x, href: e.target.value } : x)))}
            />
            <Button variant="ghost" onClick={() => setTiers(tiers.filter((_, j) => j !== i))} aria-label="Remover bloco">
              <Trash2 className="h-4 w-4" />
            </Button>
          </div>
        ))}
        {tiers.length < MAX_TIERS && (
          <Button variant="ghost" onClick={() => setTiers([...tiers, { label: "", href: "" }])}>
            <Plus className="h-4 w-4" />
            Bloco
          </Button>
        )}
      </Card>

      <Card className="space-y-3 p-4">
        <SectionTitle
          title="Pop-up de saída"
          hint="Só na página do produto e no carrinho, depois de 20 s, 1 vez por visita, quando a pessoa vai sair. Um produto, sem cupom."
        />
        <label className="flex items-center gap-2 text-sm text-slate-300">
          <Checkbox checked={popup.enabled} onChange={(e) => setPopup({ enabled: e.target.checked })} />
          Ligado
        </label>
        <div className="grid grid-cols-1 gap-2 lg:grid-cols-2">
          <Field label="Produto">
            <ProductPicker products={products} value={popup.sku} onChange={(sku) => setPopup({ sku })} />
          </Field>
          <Field label="Título">
            <Input value={popup.title} maxLength={60} placeholder="ex.: Leve o kit 3 do booster" onChange={(e) => setPopup({ title: e.target.value })} />
          </Field>
          <Field label="Texto">
            <Input value={popup.text ?? ""} maxLength={160} placeholder="opcional" onChange={(e) => setPopup({ text: e.target.value })} />
          </Field>
          <div className="grid grid-cols-2 gap-2">
            <DateField label="de" value={popup.from} onChange={(v) => setPopup({ from: v })} />
            <DateField label="até" value={popup.until} onChange={(v) => setPopup({ until: v })} />
          </div>
        </div>
      </Card>

      <div className="flex flex-wrap items-center justify-end gap-3">
        {config.updatedAt && <span className="text-[11px] text-slate-500">última mudança: {new Date(config.updatedAt).toLocaleString("pt-BR")}</span>}
        {saved && <span className="text-[11px] text-emerald-300">{saved}</span>}
        {error && <span className="text-[11px] text-rose-300">{error}</span>}
        <Button variant="primary" onClick={save} disabled={saving}>
          {saving ? "Salvando…" : "Salvar vitrine"}
        </Button>
      </div>
    </div>
  );
}

function ProductPicker({ products, value, onChange }: { products: VitrineProduct[]; value: string; onChange: (sku: string) => void }) {
  const [query, setQuery] = useState("");
  const current = products.find((p) => p.sku === value);
  const matches = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (q.length < 2) {
      return [];
    }
    return products.filter((p) => p.name.toLowerCase().includes(q)).slice(0, 8);
  }, [products, query]);
  return (
    <div className="space-y-1">
      <Input value={query} placeholder={current ? current.name : "buscar pelo nome"} onChange={(e) => setQuery(e.target.value)} />
      {matches.length > 0 && (
        <div className="max-h-56 overflow-auto rounded-md border border-slate-700">
          {matches.map((p) => (
            <button
              key={p.sku}
              type="button"
              onClick={() => {
                onChange(p.sku);
                setQuery("");
              }}
              className="block w-full px-3 py-1.5 text-left text-sm text-slate-200 hover:bg-slate-800"
            >
              {p.name} <span className="text-[11px] text-slate-500">{p.detail}</span>
            </button>
          ))}
        </div>
      )}
      <p className="text-[11px] text-slate-500">
        {current ? `Escolhido: ${current.name} (${current.detail})` : value ? `SKU ${value} (não está à venda agora)` : "Nenhum produto escolhido"}
      </p>
    </div>
  );
}

function SectionTitle({ title, hint }: { title: string; hint: string }) {
  return (
    <div>
      <h3 className="text-sm font-semibold text-slate-100">{title}</h3>
      <p className="text-[11px] text-slate-500">{hint}</p>
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

function DateField({ label, value, onChange }: { label: string; value?: string; onChange: (v: string) => void }) {
  return (
    <Field label={label}>
      <Input type="date" value={value ?? ""} onChange={(e) => onChange(e.target.value)} />
    </Field>
  );
}
