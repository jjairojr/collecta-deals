import { useEffect, useState, type ReactNode } from "react";
import { AlertTriangle, Banknote, Copy, Layers, PiggyBank, ShoppingBag } from "lucide-react";
import EmptyState from "./EmptyState";
import {
  getExports,
  getImports,
  type ExportDeal,
  type ExportFilters,
  type ExportSort,
  type ExportsResponse,
  type ImportDeal,
  type ImportFilters,
  type ImportSort,
  type ImportsGame,
  type ImportsResponse,
} from "../api";
import { brl, pct, stampLabel, usd } from "../format";
import { brandFor } from "../brand";
import CardArt from "./CardArt";
import { BrSupply, UsTrust } from "./DealDepth";
import { Kpi } from "./PortfolioPage";
import { Badge } from "./ui/badge";
import { Card } from "./ui/card";
import { Checkbox } from "./ui/checkbox";
import { Input } from "./ui/input";
import { Select } from "./ui/select";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "./ui/table";
import { ToggleGroup } from "./ui/toggle-group";

const importGames = [
  { id: "onepiece", label: "One Piece" },
  { id: "riftbound", label: "Riftbound" },
];

const massEntryProductLine: Record<string, string> = {
  onepiece: "One Piece Card Game",
  riftbound: "Riftbound: League of Legends Trading Card Game",
};

const defaultFilters: ImportFilters = {
  games: importGames.map((g) => g.id),
  minSellers: 3,
  minUSD: 3,
  minProfit: 10,
  sort: "profit",
  limit: 100,
};

const defaultExportFilters: ExportFilters = {
  games: importGames.map((g) => g.id),
  minSellers: 1,
  minUSListings: 2,
  minUSD: 10,
  minProfit: 10,
  sort: "score",
  limit: 100,
};

type Direction = "import" | "export";

function isImportSort(v: string): v is ImportSort {
  return v === "profit" || v === "margin";
}

function isExportSort(v: string): v is ExportSort {
  return v === "score" || v === "profit" || v === "margin" || v === "copies";
}

function readDirection(): Direction {
  return new URLSearchParams(window.location.search).get("dir") === "export" ? "export" : "import";
}

export default function ImportsPage() {
  const [direction, setDirectionState] = useState<Direction>(readDirection);
  const setDirection = (next: Direction) => {
    setDirectionState(next);
    const params = new URLSearchParams(window.location.search);
    if (next === "import") {
      params.delete("dir");
    } else {
      params.set("dir", next);
    }
    const q = params.toString();
    window.history.replaceState(null, "", q ? `?${q}` : window.location.pathname);
  };
  return (
    <div className="space-y-5">
      <ToggleGroup
        value={direction}
        onChange={(v) => setDirection(v === "export" ? "export" : "import")}
        options={[
          { value: "import", label: "EUA → BR" },
          { value: "export", label: "BR → EUA" },
        ]}
      />
      {direction === "export" ? <ExportView /> : <ImportView />}
    </div>
  );
}

function GameStamps({ games }: { games: ImportsGame[] }) {
  if (games.length === 0) {
    return null;
  }
  return (
    <div className="flex flex-wrap gap-2 text-xs text-slate-500">
      {games.map((g) => (
        <span key={g.game.id} className="pill pill-sm bg-panel px-2.5 py-1">
          <span className="font-bold text-slate-300">{brandFor(g.game.id).short}</span> · Liga{" "}
          {g.trackingDate ? stampLabel(g.trackingDate) : "—"} · {g.sets} coleções · TCG {stampLabel(g.updatedAt.slice(0, 13))}
        </span>
      ))}
    </div>
  );
}

function ImportView() {
  const [filters, setFilters] = useState<ImportFilters>(defaultFilters);
  const [resp, setResp] = useState<ImportsResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const id = window.setTimeout(() => {
      let current = true;
      setLoading(true);
      setError(null);
      getImports(filters)
        .then((r) => {
          if (current) {
            setResp(r);
          }
        })
        .catch((err: unknown) => {
          if (current) {
            setError(err instanceof Error ? err.message : "falha ao carregar");
          }
        })
        .finally(() => current && setLoading(false));
      return () => {
        current = false;
      };
    }, 250);
    return () => window.clearTimeout(id);
  }, [filters]);

  const deals = resp?.deals ?? [];
  const totalProfit = deals.reduce((acc, d) => acc + d.profitBRL, 0);
  const capitalUSD = deals.reduce((acc, d) => acc + d.costUSD, 0);

  return (
    <div className="space-y-5">
      <p className="max-w-3xl text-sm text-slate-400">
        Custo é o anúncio mais barato da TCGplayer convertido pelo câmbio; venda é o piso em inglês NM da Liga menos
        R$ 0,01, sem contar a Collecta. Sem alfândega nem frete.
      </p>

      <Filters filters={filters} onChange={setFilters} />

      <GameStamps games={resp?.games ?? []} />

      {error ? (
        <Panel>Não deu para carregar: {error}</Panel>
      ) : !resp && loading ? (
        <Panel>Carregando…</Panel>
      ) : !resp ? (
        <Panel>Sem dados ainda.</Panel>
      ) : deals.length === 0 ? (
        <Panel>Nada passa nesses filtros. Tente baixar o mínimo de lojas ou de lucro.</Panel>
      ) : (
        <>
          <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
            <Kpi icon={<Layers className="h-5 w-5" />} label="Cartas" value={String(deals.length)} sub="uma cópia de cada" />
            <Kpi icon={<PiggyBank className="h-5 w-5" />} label="Lucro" value={brl(totalProfit)} sub="Σ lucro/un" />
            <Kpi icon={<Banknote className="h-5 w-5" />} label="Capital" value={usd(capitalUSD)} sub="Σ custo, 1 cópia cada" />
            <Kpi
              icon={<ShoppingBag className="h-5 w-5" />}
              label="Retorno"
              value={capitalUSD > 0 && resp.games[0] ? pct((totalProfit / (capitalUSD / resp.games[0].fxRate)) * 100) : "—"}
              sub="sobre o capital em R$"
            />
          </div>

          <div className="flex flex-wrap items-center justify-between gap-3 text-sm text-slate-400">
            <span>
              <span className="font-medium text-slate-200">{deals.length}</span> carta{deals.length === 1 ? "" : "s"}
              {loading && <span className="ml-2 text-xs text-slate-500">atualizando…</span>}
            </span>
            <div className="flex flex-wrap gap-2">
              {importGames
                .filter((g) => deals.some((d) => d.game === g.id))
                .map((g) => (
                  <MassEntryButton key={g.id} game={g.id} label={g.label} deals={deals.filter((d) => d.game === g.id)} />
                ))}
            </div>
          </div>

          <ImportsTable deals={deals} />
        </>
      )}

      <p className="text-xs text-slate-500">
        Custo "tabela" é o preço do TCGCSV, que atrasa; o ao vivo é o anúncio atual. Número ambíguo (mais de um produto
        na TCGplayer para a mesma carta) usa a variante mais cara como custo. O Mass Entry leva só carta e quantidade —
        a escolha de vendedor se perde.
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

function NumberField({
  label,
  value,
  step,
  prefix,
  onChange,
}: {
  label: string;
  value: number;
  step: number;
  prefix?: string;
  onChange: (v: number) => void;
}) {
  return (
    <FilterField label={label}>
      <div className="relative">
        {prefix && (
          <span className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-xs text-slate-500">{prefix}</span>
        )}
        <Input
          type="number"
          value={value}
          min={0}
          step={step}
          onChange={(e) => onChange(Math.max(0, Number(e.target.value)))}
          className={`w-28 ${prefix ? "pl-10" : ""}`}
        />
      </div>
    </FilterField>
  );
}

function GamesField({ games, onChange }: { games: string[]; onChange: (games: string[]) => void }) {
  return (
    <FilterField label="Jogos">
      <div className="flex items-center gap-4 pb-1.5 normal-case tracking-normal">
        {importGames.map((g) => (
          <label key={g.id} className="flex cursor-pointer items-center gap-2 text-xs text-slate-300">
            <Checkbox
              accent="emerald"
              checked={games.includes(g.id)}
              onChange={(e) => onChange(e.target.checked ? [...games, g.id] : games.filter((id) => id !== g.id))}
            />
            {g.label}
          </label>
        ))}
      </div>
    </FilterField>
  );
}


function LimitField({ limit, onChange }: { limit: number; onChange: (limit: number) => void }) {
  return (
    <FilterField label="Mostrar">
      <ToggleGroup
        value={String(limit)}
        onChange={(v) => onChange(Number(v))}
        options={[
          { value: "50", label: "50" },
          { value: "100", label: "100" },
          { value: "250", label: "250" },
        ]}
      />
    </FilterField>
  );
}

function Filters({ filters, onChange }: { filters: ImportFilters; onChange: (f: ImportFilters) => void }) {
  return (
    <Card className="flex flex-wrap items-end gap-5 p-4">
      <GamesField games={filters.games} onChange={(games) => onChange({ ...filters, games })} />
      <NumberField label="Min lojas EN" value={filters.minSellers} step={1} onChange={(v) => onChange({ ...filters, minSellers: v })} />
      <NumberField label="Min custo" value={filters.minUSD} step={1} prefix="US$" onChange={(v) => onChange({ ...filters, minUSD: v })} />
      <NumberField label="Min lucro/un" value={filters.minProfit} step={5} prefix="R$" onChange={(v) => onChange({ ...filters, minProfit: v })} />
      <FilterField label="Ordenar por">
        <Select
          value={filters.sort}
          onChange={(e) => {
            if (isImportSort(e.target.value)) {
              onChange({ ...filters, sort: e.target.value });
            }
          }}
          className="w-52"
        >
          <option value="profit">Lucro por unidade</option>
          <option value="margin">Margem %</option>
        </Select>
      </FilterField>
      <LimitField limit={filters.limit} onChange={(limit) => onChange({ ...filters, limit })} />
    </Card>
  );
}

function ExportFiltersCard({ filters, onChange }: { filters: ExportFilters; onChange: (f: ExportFilters) => void }) {
  return (
    <Card className="flex flex-wrap items-end gap-5 p-4">
      <GamesField games={filters.games} onChange={(games) => onChange({ ...filters, games })} />
      <NumberField label="Min lojas BR" value={filters.minSellers} step={1} onChange={(v) => onChange({ ...filters, minSellers: v })} />
      <NumberField label="Min anúncios US" value={filters.minUSListings} step={1} onChange={(v) => onChange({ ...filters, minUSListings: v })} />
      <NumberField label="Min venda" value={filters.minUSD} step={5} prefix="US$" onChange={(v) => onChange({ ...filters, minUSD: v })} />
      <NumberField label="Min lucro/un" value={filters.minProfit} step={5} prefix="R$" onChange={(v) => onChange({ ...filters, minProfit: v })} />
      <FilterField label="Ordenar por">
        <Select
          value={filters.sort}
          onChange={(e) => {
            if (isExportSort(e.target.value)) {
              onChange({ ...filters, sort: e.target.value });
            }
          }}
          className="w-52"
        >
          <option value="score">Lucro × cópias (melhores)</option>
          <option value="profit">Lucro por unidade</option>
          <option value="margin">Margem %</option>
          <option value="copies">Cópias no piso</option>
        </Select>
      </FilterField>
      <LimitField limit={filters.limit} onChange={(limit) => onChange({ ...filters, limit })} />
    </Card>
  );
}

function ExportView() {
  const [filters, setFilters] = useState<ExportFilters>(defaultExportFilters);
  const [resp, setResp] = useState<ExportsResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const id = window.setTimeout(() => {
      let current = true;
      setLoading(true);
      setError(null);
      getExports(filters)
        .then((r) => {
          if (current) {
            setResp(r);
          }
        })
        .catch((err: unknown) => {
          if (current) {
            setError(err instanceof Error ? err.message : "falha ao carregar");
          }
        })
        .finally(() => current && setLoading(false));
      return () => {
        current = false;
      };
    }, 250);
    return () => window.clearTimeout(id);
  }, [filters]);

  const deals = resp?.exports ?? [];
  const totalProfit = deals.reduce((acc, d) => acc + d.profitBRL * d.suggestedQty, 0);
  const capitalBRL = deals.reduce((acc, d) => acc + d.costBRL * d.suggestedQty, 0);
  const units = deals.reduce((acc, d) => acc + d.suggestedQty, 0);

  return (
    <>
      <p className="max-w-3xl text-sm text-slate-400">
        Custo é o piso em inglês NM da Liga, loja a loja e sem a Collecta; venda é o anúncio ao vivo mais barato da
        TCGplayer menos US$ 0,01, bruto, sem taxa nem frete. "Sugerido" é quantas cópias existem naquele piso — o que dá
        pra comprar por esse preço.
      </p>

      <ExportFiltersCard filters={filters} onChange={setFilters} />

      <GameStamps games={resp?.games ?? []} />

      {error ? (
        <Panel>Não deu para carregar: {error}</Panel>
      ) : !resp && loading ? (
        <Panel>Carregando…</Panel>
      ) : !resp ? (
        <Panel>Sem dados ainda.</Panel>
      ) : deals.length === 0 ? (
        <Panel>Nada passa nesses filtros. Tente baixar o mínimo de anúncios US ou de lucro.</Panel>
      ) : (
        <>
          <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
            <Kpi icon={<Layers className="h-5 w-5" />} label="Cartas" value={String(deals.length)} sub={`${units} cópias no piso`} />
            <Kpi icon={<PiggyBank className="h-5 w-5" />} label="Lucro projetado" value={brl(totalProfit)} sub="lucro/un × sugerido" />
            <Kpi icon={<Banknote className="h-5 w-5" />} label="Capital" value={brl(capitalBRL)} sub="piso × sugerido" />
            <Kpi
              icon={<ShoppingBag className="h-5 w-5" />}
              label="Retorno"
              value={capitalBRL > 0 ? pct((totalProfit / capitalBRL) * 100) : "—"}
              sub="bruto, antes da taxa da TCGplayer"
            />
          </div>

          <div className="flex flex-wrap items-center justify-between gap-3 text-sm text-slate-400">
            <span>
              <span className="font-medium text-slate-200">{deals.length}</span> carta{deals.length === 1 ? "" : "s"}
              {loading && <span className="ml-2 text-xs text-slate-500">atualizando…</span>}
            </span>
            <div className="flex flex-wrap gap-2">
              {importGames
                .filter((g) => deals.some((d) => d.game === g.id))
                .map((g) => (
                  <BuyListButton key={g.id} label={g.label} deals={deals.filter((d) => d.game === g.id)} />
                ))}
            </div>
          </div>

          <ExportsTable deals={deals} />
        </>
      )}

      <p className="text-xs text-slate-500">
        O anúncio ao vivo só existe pra cartas que o pipeline já precificou na TCGplayer; as outras mostram o preço de
        tabela do TCGCSV e ficam de fora até o filtro de anúncios ser zerado. Número ambíguo (mais de um produto na
        TCGplayer) vende pela variante mais barata.
      </p>
    </>
  );
}

function buyListLines(deals: ExportDeal[]): string[] {
  return deals.map((d) => {
    const stores = d.floorStores.map((s) => `${s.storeName} ×${s.qty}`).join(", ");
    return `${d.suggestedQty}× ${d.name} (${d.set} ${d.number}) — ${brl(d.costBRL)}${stores ? ` — ${stores}` : ""} — ${d.ligaUrl}`;
  });
}

function BuyListButton({ label, deals }: { label: string; deals: ExportDeal[] }) {
  const [copied, setCopied] = useState(false);
  const copy = async () => {
    try {
      await navigator.clipboard.writeText(buyListLines(deals).join("\n"));
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1500);
    } catch {
      setCopied(false);
    }
  };
  return (
    <button
      type="button"
      onClick={copy}
      className="inline-flex items-center gap-1.5 rounded-[8px] border-2 border-outline bg-panel px-2.5 py-1.5 text-xs font-bold text-slate-200 transition-colors hover:bg-raised"
    >
      <Copy className="h-3.5 w-3.5" />
      {copied ? "Copiado" : `Lista de compra ${label}`}
    </button>
  );
}

function ExportsTable({ deals }: { deals: ExportDeal[] }) {
  return (
    <div className="sticker sticker-sm overflow-x-auto rounded-[12px] bg-panel">
      <Table className="min-w-[1240px]">
        <TableHeader>
          <TableRow className="hover:bg-transparent">
            <TableHead className="px-3">Jogo</TableHead>
            <TableHead>Carta</TableHead>
            <TableHead>Nome</TableHead>
            <TableHead className="text-right">Custo Liga</TableHead>
            <TableHead>Onde comprar</TableHead>
            <TableHead className="text-right">Custo US$</TableHead>
            <TableHead className="text-right">Vende a US$</TableHead>
            <TableHead>Mercado US</TableHead>
            <TableHead className="text-right">Lucro/un</TableHead>
            <TableHead className="text-right">Margem</TableHead>
            <TableHead className="text-right">Sugerido</TableHead>
            <TableHead className="text-right">Links</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {deals.map((d) => (
            <TableRow key={`${d.game}-${d.set}-${d.number}-${d.productId}`}>
              <TableCell className="px-3 py-2.5">
                <Badge variant="sky">{brandFor(d.game).short}</Badge>
              </TableCell>
              <TableCell className="whitespace-nowrap font-mono text-xs text-slate-300">
                <span className="flex items-center gap-2">
                  <CardArt
                    set={d.set}
                    number={d.number}
                    name={d.name}
                    productID={d.productId}
                    game={d.game}
                    className="h-12 w-9 rounded-[4px] object-cover"
                  />
                  <span>
                    <span className="text-slate-500">{d.set} ·</span> {d.number}
                  </span>
                </span>
              </TableCell>
              <TableCell className="text-slate-100">
                <div>{d.name}</div>
                <div className="flex flex-wrap items-center gap-1 text-xs text-slate-500">
                  {d.rarity && <span>{d.rarity}</span>}
                  {d.variant && d.variant !== "Normal" && <span>· {d.variant}</span>}
                  {d.usVariants > 1 && (
                    <span
                      title={`${d.usVariants} produtos na TCGplayer para esta carta — venda pela variante mais barata`}
                      className="inline-flex items-center gap-1 rounded-[8px] border-2 border-outline bg-amber-500/15 px-1.5 py-0.5 text-[10px] font-bold text-amber-300"
                    >
                      <AlertTriangle className="h-3 w-3" /> ambíguo ×{d.usVariants}
                    </span>
                  )}
                </div>
              </TableCell>
              <TableCell className="whitespace-nowrap text-right tabular-nums text-slate-100">
                {brl(d.costBRL)}
                {d.ownQty > 0 && (
                  <div className="text-[11px] text-brand-label">
                    já temos {d.ownQty} a {brl(d.ownPriceBRL)}
                  </div>
                )}
              </TableCell>
              <TableCell className="whitespace-nowrap">
                <BrSupply copies={d.floorCopies} sellers={d.sellers} />
                {d.floorStores.length > 0 && (
                  <div className="max-w-[220px] truncate text-[11px] text-slate-500" title={d.floorStores.map((s) => `${s.storeName} ×${s.qty}`).join(" · ")}>
                    {d.floorStores.map((s) => `${s.storeName} ×${s.qty}`).join(" · ")}
                  </div>
                )}
              </TableCell>
              <TableCell className="whitespace-nowrap text-right tabular-nums text-slate-400">{usd(d.costUSD)}</TableCell>
              <TableCell className="whitespace-nowrap text-right tabular-nums text-slate-100">
                {usd(d.sellUSD)}
                {!d.usLive && (
                  <span title="preço de tabela do TCGCSV, não do anúncio ao vivo" className="ml-1 text-[10px] text-amber-300">
                    tabela
                  </span>
                )}
              </TableCell>
              <TableCell className="whitespace-nowrap">
                <UsTrust listings={d.usListings} qty={d.usQty} />
              </TableCell>
              <TableCell className={`whitespace-nowrap text-right font-semibold tabular-nums ${profitTone(d.profitBRL)}`}>
                {usd(d.profitUSD)}
                <div className="text-[11px] font-normal text-slate-500">{brl(d.profitBRL)}</div>
              </TableCell>
              <TableCell className={`whitespace-nowrap text-right tabular-nums ${profitTone(d.marginPct)}`}>
                {pct(d.marginPct)}
              </TableCell>
              <TableCell className="whitespace-nowrap text-right tabular-nums text-slate-300">{d.suggestedQty}</TableCell>
              <TableCell className="whitespace-nowrap text-right">
                <div className="flex justify-end gap-1.5">
                  <a
                    href={d.tcgUrl}
                    target="_blank"
                    rel="noreferrer"
                    className="rounded-[8px] border-2 border-outline bg-sky-500/15 px-2 py-1 text-xs font-bold text-sky-300 hover:bg-sky-500/25"
                  >
                    TCG
                  </a>
                  <a
                    href={d.ligaUrl}
                    target="_blank"
                    rel="noreferrer"
                    className="rounded-[8px] border-2 border-outline bg-emerald-500/15 px-2 py-1 text-xs font-bold text-emerald-300 hover:bg-emerald-500/25"
                  >
                    Liga
                  </a>
                </div>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

function massEntryLines(deals: ImportDeal[]): string[] {
  return deals.map((d) => `1 ${d.tcgName}`);
}

function massEntryURL(game: string, lines: string[]): string {
  const productline = encodeURIComponent(massEntryProductLine[game] ?? "");
  return `https://www.tcgplayer.com/massentry?productline=${productline}&c=${lines.map(encodeURIComponent).join("||")}`;
}

function MassEntryButton({ game, label, deals }: { game: string; label: string; deals: ImportDeal[] }) {
  const [copied, setCopied] = useState(false);
  const lines = massEntryLines(deals);
  const copy = async () => {
    try {
      await navigator.clipboard.writeText(lines.join("\n"));
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1500);
    } catch {
      setCopied(false);
    }
  };
  return (
    <div className="flex items-center gap-1.5">
      <button
        type="button"
        onClick={copy}
        className="inline-flex items-center gap-1.5 rounded-[8px] border-2 border-outline bg-panel px-2.5 py-1.5 text-xs font-bold text-slate-200 transition-colors hover:bg-raised"
      >
        <Copy className="h-3.5 w-3.5" />
        {copied ? "Copiado" : `Mass Entry ${label}`}
      </button>
      <a
        href={massEntryURL(game, lines)}
        target="_blank"
        rel="noreferrer"
        className="rounded-[8px] border-2 border-outline bg-sky-500/15 px-2 py-1.5 text-xs font-bold text-sky-300 hover:bg-sky-500/25"
      >
        abrir
      </a>
    </div>
  );
}

function profitTone(v: number): string {
  return v >= 0 ? "text-emerald-300" : "text-rose-400";
}

function ImportsTable({ deals }: { deals: ImportDeal[] }) {
  return (
    <div className="sticker sticker-sm overflow-x-auto rounded-[12px] bg-panel">
      <Table className="min-w-[1240px]">
        <TableHeader>
          <TableRow className="hover:bg-transparent">
            <TableHead className="px-3">Jogo</TableHead>
            <TableHead>Carta</TableHead>
            <TableHead>Nome</TableHead>
            <TableHead className="text-right">Custo US$</TableHead>
            <TableHead>Mercado US</TableHead>
            <TableHead className="text-right">Custo R$</TableHead>
            <TableHead className="text-right">Piso Liga</TableHead>
            <TableHead>Oferta BR</TableHead>
            <TableHead className="text-right">Vende a</TableHead>
            <TableHead className="text-right">Lucro/un</TableHead>
            <TableHead className="text-right">Margem</TableHead>
            <TableHead className="text-right">Links</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {deals.map((d) => (
            <TableRow key={`${d.game}-${d.set}-${d.number}-${d.productId}`}>
              <TableCell className="px-3 py-2.5">
                <Badge variant="sky">{brandFor(d.game).short}</Badge>
              </TableCell>
              <TableCell className="whitespace-nowrap font-mono text-xs text-slate-300">
                <span className="flex items-center gap-2">
                  <CardArt
                    set={d.set}
                    number={d.number}
                    name={d.name}
                    productID={d.productId}
                    game={d.game}
                    className="h-12 w-9 rounded-[4px] object-cover"
                  />
                  <span>
                    <span className="text-slate-500">{d.set} ·</span> {d.number}
                  </span>
                </span>
              </TableCell>
              <TableCell className="text-slate-100">
                <div>{d.name}</div>
                <div className="flex flex-wrap items-center gap-1 text-xs text-slate-500">
                  {d.rarity && <span>{d.rarity}</span>}
                  {d.variant && d.variant !== "Normal" && <span>· {d.variant}</span>}
                  {d.usVariants > 1 && (
                    <span
                      title={`${d.usVariants} produtos na TCGplayer para esta carta — custo pela variante mais cara`}
                      className="inline-flex items-center gap-1 rounded-[8px] border-2 border-outline bg-amber-500/15 px-1.5 py-0.5 text-[10px] font-bold text-amber-300"
                    >
                      <AlertTriangle className="h-3 w-3" /> ambíguo ×{d.usVariants}
                    </span>
                  )}
                </div>
              </TableCell>
              <TableCell className="whitespace-nowrap text-right tabular-nums text-slate-100">
                {usd(d.costUSD)}
                {!d.usLive && (
                  <span title="preço de tabela do TCGCSV, não do anúncio ao vivo" className="ml-1 text-[10px] text-amber-300">
                    tabela
                  </span>
                )}
              </TableCell>
              <TableCell className="whitespace-nowrap">
                <UsTrust listings={d.usListings} qty={d.usQty} />
              </TableCell>
              <TableCell className="whitespace-nowrap text-right tabular-nums text-slate-400">{brl(d.costBRL)}</TableCell>
              <TableCell className="whitespace-nowrap text-right tabular-nums text-slate-300">{brl(d.floorBRL)}</TableCell>
              <TableCell className="whitespace-nowrap">
                <BrSupply copies={d.floorCopies} sellers={d.sellers} />
                {d.ownQty > 0 && (
                  <div className="text-[11px] text-brand-label">
                    já temos {d.ownQty} a {brl(d.ownPriceBRL)}
                  </div>
                )}
              </TableCell>
              <TableCell className="whitespace-nowrap text-right tabular-nums text-slate-100">{brl(d.sellBRL)}</TableCell>
              <TableCell className={`whitespace-nowrap text-right font-semibold tabular-nums ${profitTone(d.profitBRL)}`}>
                {brl(d.profitBRL)}
              </TableCell>
              <TableCell className={`whitespace-nowrap text-right tabular-nums ${profitTone(d.marginPct)}`}>
                {pct(d.marginPct)}
              </TableCell>
              <TableCell className="whitespace-nowrap text-right">
                <div className="flex justify-end gap-1.5">
                  <a
                    href={d.tcgUrl}
                    target="_blank"
                    rel="noreferrer"
                    className="rounded-[8px] border-2 border-outline bg-sky-500/15 px-2 py-1 text-xs font-bold text-sky-300 hover:bg-sky-500/25"
                  >
                    TCG
                  </a>
                  <a
                    href={d.ligaUrl}
                    target="_blank"
                    rel="noreferrer"
                    className="rounded-[8px] border-2 border-outline bg-emerald-500/15 px-2 py-1 text-xs font-bold text-emerald-300 hover:bg-emerald-500/25"
                  >
                    Liga
                  </a>
                </div>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

function Panel({ children }: { children: ReactNode }) {
  return <EmptyState>{children}</EmptyState>;
}
