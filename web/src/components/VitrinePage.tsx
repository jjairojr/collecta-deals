import { useCallback, useEffect, useMemo, useState } from "react";
import { Plus, Trash2 } from "lucide-react";
import {
  getVitrine,
  listVitrineProducts,
  saveVitrine,
  uploadVitrineImage,
  type VitrineBanner,
  type VitrineBarImage,
  type VitrineImage,
  type VitrineConfig,
  type VitrineHeroSlide,
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
const MAX_BAR_IMAGES = 6;
const MAX_POPUPS = 8;
const MAX_SLIDES = 8;

// Sizes sent to the art team (27/11): the site shows the art at these
// proportions, so a file with another one gets cut. Weight is a target, not a
// hard limit (the server refuses only above 1 MB).
const ART = {
  faixa: {
    desktop: { w: 2880, h: 120, kb: 150 },
    mobile: { w: 1170, h: 180, kb: 100 },
  },
  banner: {
    desktop: { w: 2720, h: 800, kb: 350 },
    mobile: { w: 1080, h: 1080, kb: 250 },
  },
} as const;

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

function emptySlide(): VitrineHeroSlide {
  return { enabled: true, kind: "produto", sku: "", badge: "", phrase: "", href: "" };
}

function emptyImage(): VitrineImage {
  return { desktop: "", mobile: "", alt: "" };
}

function emptyBarImage(): VitrineBarImage {
  return { desktop: "", mobile: "", alt: "", href: "" };
}

function hasImage(im?: VitrineImage): boolean {
  return Boolean(im && (im.desktop || im.mobile || im.alt.trim()));
}

// Drops the empty optional fields so the server stores only what was filled.
function clean(c: VitrineConfig): VitrineConfig {
  const phrases = c.phrases.filter((p) => p.text.trim());
  const barImages = (c.barImages ?? []).filter((b) => hasImage(b) || b.href.trim());
  const b = c.banner;
  const banner =
    b && b.title.trim()
      ? {
          ...b,
          tiers: (b.tiers ?? []).filter((t) => t.label.trim() || t.href.trim()),
          image: hasImage(b.image) ? b.image : undefined,
        }
      : undefined;
  const popups = (c.popups ?? []).filter((p) => p.enabled || p.sku.trim() || p.title.trim());
  const heroSlides = (c.heroSlides ?? [])
    .filter((s) => (s.kind === "produto" ? Boolean(s.sku?.trim()) : hasImage(s.image) || Boolean(s.href?.trim())))
    .map((s) => (s.kind === "produto" ? { ...s, image: undefined } : { ...s, sku: undefined }));
  return { phrases, barImages, banner, popups, heroStyle: c.heroStyle ?? "rosa", heroSlides };
}

// Puts what the server sent into the editing shape: an old single pop-up becomes
// the first of the list.
function editable(c: VitrineConfig): VitrineConfig {
  return {
    phrases: c.phrases ?? [],
    barImages: c.barImages ?? [],
    banner: c.banner ?? emptyBanner(),
    popups: c.popups ?? (c.popup ? [c.popup] : []),
    heroStyle: c.heroStyle ?? "rosa",
    heroSlides: c.heroSlides ?? [],
    updatedAt: c.updatedAt,
  };
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
      setConfig(editable(c));
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
      setConfig(editable(out));
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
  const popups = config.popups ?? [];
  const slides = config.heroSlides ?? [];
  const setSlide = (i: number, p: Partial<VitrineHeroSlide>) =>
    setConfig({ ...config, heroSlides: slides.map((x, j) => (j === i ? { ...x, ...p } : x)) });
  const moveSlide = (i: number, delta: number) => {
    const j = i + delta;
    if (j < 0 || j >= slides.length) {
      return;
    }
    const next = [...slides];
    [next[i], next[j]] = [next[j], next[i]];
    setConfig({ ...config, heroSlides: next });
  };
  const setPopup = (i: number, p: Partial<VitrinePopup>) =>
    setConfig({ ...config, popups: popups.map((x, j) => (j === i ? { ...x, ...p } : x)) });
  const barImages = config.barImages ?? [];
  const setBarImage = (i: number, b: Partial<VitrineBarImage>) =>
    setConfig({ ...config, barImages: barImages.map((x, j) => (j === i ? { ...x, ...b } : x)) });
  const bannerImage: VitrineImage = banner.image ?? { desktop: "", mobile: "", alt: "" };
  const setBannerImage = (im: Partial<VitrineImage>) => setBanner({ image: { ...bannerImage, ...im } });

  return (
    <div className="space-y-4">
      <Card className="space-y-3 p-4">
        <SectionTitle
          title="Carrossel do topo da home"
          hint="Substitui o topo da home enquanto houver um slide valendo. Produto: foto, nome, preço no Pix e parcela vêm do catálogo; some sozinho se sair de estoque ou da data. Imagem: a arte enviada (computador 2720×800, celular 1080×1080), com link. A ordem da lista é a ordem do carrossel. Sem slide valendo, volta o topo de sempre."
        />
        <div className="flex flex-wrap gap-4 text-sm text-slate-300">
          {([
            ["rosa", "Rosa: fundo rosa da marca, cartão azul"],
            ["azul", "Azul: fundo azul royal, cartão rosa"],
          ] as const).map(([value, label]) => (
            <label key={value} className="flex items-center gap-2">
              <input type="radio" name="heroStyle" checked={(config.heroStyle ?? "rosa") === value} onChange={() => setConfig({ ...config, heroStyle: value })} />
              {label}
            </label>
          ))}
        </div>
        {slides.map((slide, i) => (
          <div key={i} className="space-y-2 rounded-md border border-slate-800 p-3">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <div className="flex items-center gap-4 text-sm text-slate-300">
                <span className="text-slate-500">#{i + 1}</span>
                <label className="flex items-center gap-2">
                  <Checkbox checked={slide.enabled} onChange={(e) => setSlide(i, { enabled: e.target.checked })} />
                  Ligado
                </label>
                <select
                  className="rounded-md border border-slate-700 bg-slate-900 px-2 py-1 text-sm"
                  value={slide.kind}
                  onChange={(e) => setSlide(i, e.target.value === "imagem" ? { kind: "imagem", image: slide.image ?? emptyImage() } : { kind: "produto" })}
                >
                  <option value="produto">Produto</option>
                  <option value="imagem">Imagem</option>
                </select>
              </div>
              <div className="flex items-center gap-1">
                <Button variant="ghost" onClick={() => moveSlide(i, -1)} disabled={i === 0} aria-label="Subir slide">↑</Button>
                <Button variant="ghost" onClick={() => moveSlide(i, 1)} disabled={i === slides.length - 1} aria-label="Descer slide">↓</Button>
                <Button variant="ghost" onClick={() => setConfig({ ...config, heroSlides: slides.filter((_, j) => j !== i) })} aria-label="Remover slide">
                  <Trash2 className="h-4 w-4" />
                </Button>
              </div>
            </div>
            {slide.kind === "produto" ? (
              <Field label="Produto (kits aparecem como “Kit N”)">
                <ProductPicker products={products} value={slide.sku ?? ""} onChange={(sku) => setSlide(i, { sku })} />
              </Field>
            ) : (
              <div className="space-y-2">
                <div className="grid grid-cols-1 gap-2 lg:grid-cols-2">
                  <ImageUpload label="Computador" spec={ART.banner.desktop} value={slide.image?.desktop ?? ""} onChange={(desktop) => setSlide(i, { image: { ...(slide.image ?? emptyImage()), desktop } })} />
                  <ImageUpload label="Celular" spec={ART.banner.mobile} value={slide.image?.mobile ?? ""} onChange={(mobile) => setSlide(i, { image: { ...(slide.image ?? emptyImage()), mobile } })} />
                </div>
                <Field label="Descrição (para quem não vê a imagem)">
                  <Input value={slide.image?.alt ?? ""} maxLength={120} onChange={(e) => setSlide(i, { image: { ...(slide.image ?? emptyImage()), alt: e.target.value } })} />
                </Field>
              </div>
            )}
            <div className="grid grid-cols-1 gap-2 lg:grid-cols-[140px_1fr_1fr_160px_160px]">
              <Field label="Selo">
                <Input value={slide.badge ?? ""} maxLength={20} placeholder="Kit, Natal…" onChange={(e) => setSlide(i, { badge: e.target.value })} />
              </Field>
              <Field label="Frase (até 90 letras)">
                <Input value={slide.phrase ?? ""} maxLength={90} placeholder="sem preço, frete ou pressa" onChange={(e) => setSlide(i, { phrase: e.target.value })} />
              </Field>
              <Field label={slide.kind === "imagem" ? "Link do site" : "Link (opcional; padrão: página do produto)"}>
                <Input value={slide.href ?? ""} placeholder="ex.: /selado/pokemon" onChange={(e) => setSlide(i, { href: e.target.value })} />
              </Field>
              <DateField label="de" value={slide.from} onChange={(v) => setSlide(i, { from: v })} />
              <DateField label="até" value={slide.until} onChange={(v) => setSlide(i, { until: v })} />
            </div>
          </div>
        ))}
        {slides.length < MAX_SLIDES && (
          <Button variant="ghost" onClick={() => setConfig({ ...config, heroSlides: [...slides, emptySlide()] })}>
            <Plus className="h-4 w-4" />
            Slide
          </Button>
        )}
      </Card>

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
        <SectionTitle
          title="Faixa do topo com imagem"
          hint="Enquanto vale, a imagem toma o lugar da faixa de texto em todas as páginas, e a imagem inteira é um link. Vale a primeira da lista dentro das datas."
        />
        {barImages.map((b, i) => (
          <div key={i} className="space-y-2 rounded-md border border-slate-800 p-3">
            <div className="grid grid-cols-1 gap-2 lg:grid-cols-2">
              <ImageUpload label="Computador" spec={ART.faixa.desktop} value={b.desktop} onChange={(desktop) => setBarImage(i, { desktop })} />
              <ImageUpload label="Celular" spec={ART.faixa.mobile} value={b.mobile} onChange={(mobile) => setBarImage(i, { mobile })} />
            </div>
            <div className="grid grid-cols-1 gap-2 lg:grid-cols-[1fr_1fr_160px_160px_auto]">
              <Field label="Descrição (para quem não vê a imagem)">
                <Input value={b.alt} maxLength={120} placeholder="ex.: Pikachu e Luffy: presentes de Natal" onChange={(e) => setBarImage(i, { alt: e.target.value })} />
              </Field>
              <Field label="Link do site">
                <Input value={b.href} placeholder="ex.: /selado" onChange={(e) => setBarImage(i, { href: e.target.value })} />
              </Field>
              <DateField label="de" value={b.from} onChange={(v) => setBarImage(i, { from: v })} />
              <DateField label="até" value={b.until} onChange={(v) => setBarImage(i, { until: v })} />
              <Button variant="ghost" onClick={() => setConfig({ ...config, barImages: barImages.filter((_, j) => j !== i) })} aria-label="Remover imagem da faixa">
                <Trash2 className="h-4 w-4" />
              </Button>
            </div>
          </div>
        ))}
        {barImages.length < MAX_BAR_IMAGES && (
          <Button variant="ghost" onClick={() => setConfig({ ...config, barImages: [...barImages, emptyBarImage()] })}>
            <Plus className="h-4 w-4" />
            Imagem da faixa
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
          Imagem do banner (opcional): com as duas imagens, a home mostra a arte no lugar do título e do texto, e ela leva ao link do botão
          (obrigatório nesse caso). Os blocos de presente continuam embaixo.
        </p>
        <div className="grid grid-cols-1 gap-2 lg:grid-cols-2">
          <ImageUpload label="Computador" spec={ART.banner.desktop} value={bannerImage.desktop} onChange={(desktop) => setBannerImage({ desktop })} />
          <ImageUpload label="Celular" spec={ART.banner.mobile} value={bannerImage.mobile} onChange={(mobile) => setBannerImage({ mobile })} />
          <Field label="Descrição da imagem (para quem não vê)">
            <Input value={bannerImage.alt} maxLength={120} placeholder="ex.: Presentes de Natal com Pikachu e Luffy" onChange={(e) => setBannerImage({ alt: e.target.value })} />
          </Field>
          {hasImage(banner.image) && (
            <Button variant="ghost" onClick={() => setBanner({ image: undefined })}>
              <Trash2 className="h-4 w-4" />
              Tirar a imagem do banner
            </Button>
          )}
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
          title="Pop-ups de saída"
          hint="Só na página do produto e no carrinho, depois de 20 s, 1 vez por visita, quando a pessoa vai sair. Um produto, sem cupom. Vale o primeiro ligado, dentro das datas e com o produto à venda: agende vários e o site troca sozinho."
        />
        {popups.map((popup, i) => (
          <div key={i} className="space-y-2 rounded-md border border-slate-800 p-3">
            <div className="flex items-center justify-between">
              <label className="flex items-center gap-2 text-sm text-slate-300">
                <Checkbox checked={popup.enabled} onChange={(e) => setPopup(i, { enabled: e.target.checked })} />
                Ligado
              </label>
              <Button variant="ghost" onClick={() => setConfig({ ...config, popups: popups.filter((_, j) => j !== i) })} aria-label="Remover pop-up">
                <Trash2 className="h-4 w-4" />
              </Button>
            </div>
            <div className="grid grid-cols-1 gap-2 lg:grid-cols-2">
              <Field label="Produto">
                <ProductPicker products={products} value={popup.sku} onChange={(sku) => setPopup(i, { sku })} />
              </Field>
              <Field label="Título">
                <Input value={popup.title} maxLength={60} placeholder="ex.: Leve o kit 3 do booster" onChange={(e) => setPopup(i, { title: e.target.value })} />
              </Field>
              <Field label="Texto">
                <Input value={popup.text ?? ""} maxLength={160} placeholder="opcional" onChange={(e) => setPopup(i, { text: e.target.value })} />
              </Field>
              <div className="grid grid-cols-2 gap-2">
                <DateField label="de" value={popup.from} onChange={(v) => setPopup(i, { from: v })} />
                <DateField label="até" value={popup.until} onChange={(v) => setPopup(i, { until: v })} />
              </div>
            </div>
          </div>
        ))}
        {popups.length < MAX_POPUPS && (
          <Button variant="ghost" onClick={() => setConfig({ ...config, popups: [...popups, emptyPopup()] })}>
            <Plus className="h-4 w-4" />
            Pop-up
          </Button>
        )}
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

type ArtSpec = { w: number; h: number; kb: number };

// Uploads one art file and checks it against the size sent to the art team:
// a different proportion or a heavy file gets a warning, not a refusal.
function ImageUpload({ label, spec, value, onChange }: { label: string; spec: ArtSpec; value: string; onChange: (url: string) => void }) {
  const [busy, setBusy] = useState(false);
  const [note, setNote] = useState<string | null>(null);
  const [failure, setFailure] = useState<string | null>(null);

  const pick = async (file: File | undefined) => {
    if (!file) {
      return;
    }
    setBusy(true);
    setFailure(null);
    setNote(null);
    try {
      const warnings: string[] = [];
      const kb = Math.round(file.size / 1024);
      if (kb > spec.kb) {
        warnings.push(`${kb} KB, acima dos ${spec.kb} KB recomendados: a página fica mais lenta no celular`);
      }
      try {
        const bmp = await createImageBitmap(file);
        const want = spec.w / spec.h;
        const got = bmp.width / bmp.height;
        if (Math.abs(got - want) / want > 0.02) {
          warnings.push(`${bmp.width} × ${bmp.height} px, fora da proporção de ${spec.w} × ${spec.h}: o site vai cortar as bordas`);
        }
        bmp.close();
      } catch {
        // the browser could not read it; the server still checks the type
      }
      const url = await uploadVitrineImage(file);
      onChange(url);
      setNote(warnings.length > 0 ? `Enviada, mas: ${warnings.join("; ")}.` : `Enviada (${kb} KB).`);
    } catch (err: unknown) {
      setFailure(errorText(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="space-y-1">
      <span className="text-[10px] font-medium uppercase tracking-wide text-slate-500">
        {label}: {spec.w} × {spec.h} px, até {spec.kb} KB (WebP ou JPG)
      </span>
      {value && <img src={value} alt="" className="max-h-24 w-full rounded border border-slate-700 object-contain" />}
      <input
        type="file"
        accept="image/webp,image/jpeg,image/png"
        disabled={busy}
        onChange={(e) => {
          pick(e.target.files?.[0]);
          e.target.value = "";
        }}
        className="block w-full text-[12px] text-slate-300 file:mr-3 file:rounded file:border-0 file:bg-slate-700 file:px-3 file:py-1 file:text-slate-100"
      />
      {busy && <p className="text-[11px] text-slate-400">Enviando…</p>}
      {note && <p className="text-[11px] text-amber-300">{note}</p>}
      {failure && <p className="text-[11px] text-rose-300">{failure}</p>}
    </div>
  );
}
