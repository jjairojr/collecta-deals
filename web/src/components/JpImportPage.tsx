import { useEffect, useMemo, useState, type ReactNode } from "react";
import { ArrowDown, Banknote, CalendarClock, ExternalLink, Layers, Store } from "lucide-react";
import { cardImageURL, getJpImport, type JpImportResponse, type JpImportRow, type JpLang } from "../api";
import { brl0, stampLabel } from "../format";
import CardArt from "./CardArt";
import EmptyState from "./EmptyState";
import { Kpi } from "./PortfolioPage";
import { Badge } from "./ui/badge";
import { Card } from "./ui/card";
import { Checkbox } from "./ui/checkbox";
import { Input } from "./ui/input";
import { Select } from "./ui/select";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "./ui/table";
import { ToggleGroup } from "./ui/toggle-group";

type Era = "all" | "mega" | "sv";
type Rarity = "all" | "AR" | "SAR";
type PtFilter = "all" | "with" | "without";
type SortKey = "realX" | "realBRL" | "costBRL" | "jpFloor" | "ptFloor" | "ptSold" | "number";

interface Filters {
  era: Era;
  set: string;
  rarity: Rarity;
  pt: PtFilter;
  minX: number;
  inStock: boolean;
  query: string;
}

const defaultFilters: Filters = { era: "all", set: "", rarity: "all", pt: "all", minX: 0, inStock: true, query: "" };

const pageSize = 200;

const sortValue: Record<SortKey, (r: JpImportRow) => number> = {
  realX: (r) => r.realX,
  realBRL: (r) => r.realBRL,
  costBRL: (r) => r.costBRL,
  jpFloor: (r) => r.jp.floorBRL,
  ptFloor: (r) => r.pt.floorBRL,
  ptSold: (r) => r.pt.sold,
  number: (r) => -r.number,
};

function isEra(v: string): v is Era {
  return v === "all" || v === "mega" || v === "sv";
}

function isRarity(v: string): v is Rarity {
  return v === "all" || v === "AR" || v === "SAR";
}

function isPtFilter(v: string): v is PtFilter {
  return v === "all" || v === "with" || v === "without";
}

export default function JpImportPage() {
  const [resp, setResp] = useState<JpImportResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [filters, setFilters] = useState<Filters>(defaultFilters);
  const [sort, setSort] = useState<SortKey>("realX");
  const [shown, setShown] = useState(pageSize);

  useEffect(() => {
    let current = true;
    getJpImport()
      .then((r) => current && setResp(r))
      .catch((err: unknown) => current && setError(err instanceof Error ? err.message : "falha ao carregar"));
    return () => {
      current = false;
    };
  }, []);

  useEffect(() => setShown(pageSize), [filters, sort]);

  const rows = useMemo(() => {
    if (!resp) {
      return [];
    }
    const q = filters.query.trim().toLowerCase();
    const out = resp.rows.filter((r) => {
      if (filters.era !== "all" && r.era !== filters.era) return false;
      if (filters.set && r.set !== filters.set) return false;
      if (filters.rarity !== "all" && r.rarity !== filters.rarity) return false;
      if (filters.pt === "with" && r.westStatus !== "ok") return false;
      if (filters.pt === "without" && r.westStatus === "ok") return false;
      if (filters.minX > 0 && r.realX < filters.minX) return false;
      if (filters.inStock && r.stock <= 0) return false;
      if (q && !`${r.name} ${r.nameJP} ${r.set} ${r.setName}`.toLowerCase().includes(q)) return false;
      return true;
    });
    const value = sortValue[sort];
    return [...out].sort((a, b) => value(b) - value(a));
  }, [resp, filters, sort]);

  if (error) {
    return <EmptyState>Não deu para carregar: {error}</EmptyState>;
  }
  if (!resp) {
    return <EmptyState>Carregando…</EmptyState>;
  }
  if (resp.rows.length === 0) {
    return <EmptyState>Sem captura da Card Rush ainda. Peça ao Claude para capturar os preços.</EmptyState>;
  }

  const sets = resp.sets.filter((s) => filters.era === "all" || s.era === filters.era);
  const capturedDay = resp.capturedAt.slice(0, 10);

  return (
    <div className="space-y-5">
      <p className="max-w-3xl text-sm text-slate-400">
        AR e SAR japonesas da Card Rush contra a Liga. Como o cliente prefere a versão em português quando ela existe, o
        preço que dá para cobrar é o menor entre o piso japonês e o português (NM, sem a Collecta). "×" é esse preço sobre
        o custo da Card Rush convertido, sem frete nem imposto.
      </p>

      <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
        <Kpi icon={<Layers className="h-5 w-5" />} label="Cartas" value={String(rows.length)} sub={`de ${resp.count} AR/SAR`} />
        <Kpi icon={<Store className="h-5 w-5" />} label="Card Rush" value={stampLabel(capturedDay)} sub="captura de preços" />
        <Kpi icon={<CalendarClock className="h-5 w-5" />} label="Liga" value={resp.trackingDate ? stampLabel(resp.trackingDate) : "—"} sub="snapshot mais novo" />
        <Kpi icon={<Banknote className="h-5 w-5" />} label="Câmbio" value={`¥1 = R$ ${resp.fxJPYBRL.toFixed(4).replace(".", ",")}`} sub="frankfurter" />
      </div>

      <FiltersCard filters={filters} sets={sets} onChange={setFilters} />

      {rows.length === 0 ? (
        <EmptyState>Nada passa nesses filtros.</EmptyState>
      ) : (
        <>
          <JpTable rows={rows.slice(0, shown)} sort={sort} onSort={setSort} />
          {shown < rows.length && (
            <button
              type="button"
              onClick={() => setShown((n) => n + pageSize)}
              className="sticker sticker-sm w-full rounded-[12px] bg-panel py-2 text-sm font-medium text-slate-300"
            >
              Mostrar mais ({rows.length - shown} restantes)
            </button>
          )}
        </>
      )}

      <p className="text-xs text-slate-500">
        A versão em português é casada pelo nome no bloco IR (AR) ou SIR (SAR) da coleção ocidental. Confira o número
        antes de comprar quando houver duas artes com o mesmo nome. "Vend." é o que sumiu dos anúncios nos últimos 14 dias.
      </p>
    </div>
  );
}

function FilterField({ label, children }: { label: string; children: ReactNode }) {
  return (
    <label className="flex flex-col gap-1.5 text-[10px] font-medium uppercase tracking-wide text-slate-500">
      {label}
      {children}
    </label>
  );
}

function FiltersCard({
  filters,
  sets,
  onChange,
}: {
  filters: Filters;
  sets: JpImportResponse["sets"];
  onChange: (f: Filters) => void;
}) {
  return (
    <Card className="flex flex-wrap items-end gap-5 p-4">
      <FilterField label="Era">
        <ToggleGroup
          value={filters.era}
          onChange={(v) => isEra(v) && onChange({ ...filters, era: v, set: "" })}
          options={[
            { value: "all", label: "Todas" },
            { value: "mega", label: "Mega" },
            { value: "sv", label: "Escarlate/Violeta" },
          ]}
        />
      </FilterField>
      <FilterField label="Coleção">
        <Select value={filters.set} onChange={(e) => onChange({ ...filters, set: e.target.value })} className="w-56">
          <option value="">Todas</option>
          {sets.map((s) => (
            <option key={s.jp} value={s.jp}>
              {s.jp} — {s.name}
            </option>
          ))}
        </Select>
      </FilterField>
      <FilterField label="Raridade">
        <ToggleGroup
          value={filters.rarity}
          onChange={(v) => isRarity(v) && onChange({ ...filters, rarity: v })}
          options={[
            { value: "all", label: "Todas" },
            { value: "AR", label: "AR" },
            { value: "SAR", label: "SAR" },
          ]}
        />
      </FilterField>
      <FilterField label="Versão PT">
        <ToggleGroup
          value={filters.pt}
          onChange={(v) => isPtFilter(v) && onChange({ ...filters, pt: v })}
          options={[
            { value: "all", label: "Todas" },
            { value: "with", label: "Com PT" },
            { value: "without", label: "Sem PT" },
          ]}
        />
      </FilterField>
      <FilterField label="× mínimo">
        <Input
          type="number"
          value={filters.minX}
          min={0}
          step={0.5}
          onChange={(e) => onChange({ ...filters, minX: Math.max(0, Number(e.target.value)) })}
          className="w-24"
        />
      </FilterField>
      <FilterField label="Buscar">
        <Input
          value={filters.query}
          placeholder="nome, coleção…"
          onChange={(e) => onChange({ ...filters, query: e.target.value })}
          className="w-48"
        />
      </FilterField>
      <label className="flex cursor-pointer items-center gap-2 pb-2 text-xs text-slate-300">
        <Checkbox
          accent="emerald"
          checked={filters.inStock}
          onChange={(e) => onChange({ ...filters, inStock: e.target.checked })}
        />
        Só com estoque na Card Rush
      </label>
    </Card>
  );
}

function SortHead({
  label,
  k,
  sort,
  onSort,
}: {
  label: string;
  k: SortKey;
  sort: SortKey;
  onSort: (k: SortKey) => void;
}) {
  return (
    <TableHead className="text-right">
      <button
        type="button"
        onClick={() => onSort(k)}
        className={`inline-flex items-center gap-1 ${sort === k ? "text-fg" : "text-slate-500 hover:text-slate-300"}`}
      >
        {label}
        {sort === k && <ArrowDown className="h-3 w-3" />}
      </button>
    </TableHead>
  );
}

function xTone(x: number): string {
  if (x >= 2.5) return "text-emerald-300";
  if (x >= 1.5) return "text-amber-300";
  if (x > 0) return "text-rose-400";
  return "text-slate-500";
}

function LangCell({ lang }: { lang: JpLang }) {
  if (lang.floorBRL <= 0) {
    return <span className="text-slate-600">—</span>;
  }
  return (
    <>
      <div className="tabular-nums text-slate-100">{brl0(lang.floorBRL)}</div>
      <div className="text-[11px] text-slate-500">
        {lang.copies} un · vend. {lang.sold}
      </div>
    </>
  );
}

function WestCell({ row }: { row: JpImportRow }) {
  if (row.westStatus === "unreleased") {
    return <Badge variant="amber">PT não lançado</Badge>;
  }
  if (row.westStatus === "untracked") {
    return <Badge>sem dados</Badge>;
  }
  if (row.westStatus === "missing") {
    return <Badge variant="rose">sem PT</Badge>;
  }
  return (
    <>
      <LangCell lang={row.pt} />
      <div className="font-mono text-[10px] text-slate-500">
        {row.west} {String(row.westNumber).padStart(3, "0")}
      </div>
    </>
  );
}

function pad(n: number): string {
  return String(n).padStart(3, "0");
}

function JpTable({ rows, sort, onSort }: { rows: JpImportRow[]; sort: SortKey; onSort: (k: SortKey) => void }) {
  return (
    <div className="sticker sticker-sm overflow-x-auto rounded-[12px] bg-panel">
      <Table className="min-w-[1180px]">
        <TableHeader>
          <TableRow className="hover:bg-transparent">
            <TableHead className="px-3">Carta</TableHead>
            <TableHead>Nome</TableHead>
            <SortHead label="Card Rush" k="costBRL" sort={sort} onSort={onSort} />
            <SortHead label="Liga JP" k="jpFloor" sort={sort} onSort={onSort} />
            <SortHead label="Liga PT" k="ptFloor" sort={sort} onSort={onSort} />
            <TableHead className="text-right">Liga EN</TableHead>
            <SortHead label="Preço real" k="realBRL" sort={sort} onSort={onSort} />
            <SortHead label="×" k="realX" sort={sort} onSort={onSort} />
            <SortHead label="Giro PT" k="ptSold" sort={sort} onSort={onSort} />
            <TableHead className="text-right">Links</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((r) => (
            <TableRow key={`${r.set}-${r.number}-${r.rarity}`}>
              <TableCell className="whitespace-nowrap px-3 py-2.5 font-mono text-xs text-slate-300">
                <span className="flex items-center gap-2">
                  <CardArt
                    set={r.set}
                    number={pad(r.number)}
                    name={r.name}
                    game="pokemon"
                    imageURL={r.ligaURL ? `${cardImageURL(r.set, pad(r.number), undefined, "pokemon")}&url=${encodeURIComponent(r.ligaURL)}` : undefined}
                    className="h-12 w-9 rounded-[4px] object-cover"
                  />
                  <span>
                    <span className="text-slate-500">{r.set} ·</span> {pad(r.number)}
                    <div>
                      <Badge variant={r.rarity === "SAR" ? "violet" : "sky"}>{r.rarity}</Badge>
                    </div>
                  </span>
                </span>
              </TableCell>
              <TableCell className="text-slate-100">
                <div>{r.name || r.nameJP}</div>
                <div className="text-xs text-slate-500">
                  {r.nameJP ? `${r.nameJP} · ${r.setName}` : r.setName}
                </div>
              </TableCell>
              <TableCell className="whitespace-nowrap text-right">
                <div className="tabular-nums text-slate-100">{brl0(r.costBRL)}</div>
                <div className="text-[11px] text-slate-500">
                  ¥{r.yen.toLocaleString("pt-BR")} · {r.stock} un
                </div>
              </TableCell>
              <TableCell className="whitespace-nowrap text-right">
                <LangCell lang={r.jp} />
              </TableCell>
              <TableCell className="whitespace-nowrap text-right">
                <WestCell row={r} />
              </TableCell>
              <TableCell className="whitespace-nowrap text-right">
                {r.en.floorBRL > 0 ? <span className="tabular-nums text-slate-400">{brl0(r.en.floorBRL)}</span> : <span className="text-slate-600">—</span>}
              </TableCell>
              <TableCell className="whitespace-nowrap text-right tabular-nums text-slate-100">
                {r.realBRL > 0 ? brl0(r.realBRL) : "—"}
              </TableCell>
              <TableCell className={`whitespace-nowrap text-right font-semibold tabular-nums ${xTone(r.realX)}`}>
                {r.realX > 0 ? `${r.realX.toFixed(1).replace(".", ",")}×` : "—"}
              </TableCell>
              <TableCell className="whitespace-nowrap text-right tabular-nums text-slate-300">
                {r.westStatus === "ok" ? r.pt.sold : "—"}
              </TableCell>
              <TableCell className="whitespace-nowrap text-right text-xs">
                <span className="inline-flex gap-2">
                  <ExtLink href={r.cardRushURL} label="CR" />
                  {r.ligaURL && <ExtLink href={r.ligaURL} label="JP" />}
                  {r.westURL && <ExtLink href={r.westURL} label="PT" />}
                </span>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

function ExtLink({ href, label }: { href: string; label: string }) {
  return (
    <a href={href} target="_blank" rel="noreferrer" className="inline-flex items-center gap-0.5 text-slate-400 hover:text-fg">
      {label}
      <ExternalLink className="h-3 w-3" />
    </a>
  );
}
