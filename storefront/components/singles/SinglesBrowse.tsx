"use client";

import { useMemo, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { SlidersHorizontal } from "lucide-react";
import Container from "@/components/ui/Container";
import FilterChip from "@/components/ui/FilterChip";
import SingleCard from "@/components/SingleCard";
import { brl } from "@/lib/format";
import type { Single } from "@/lib/types";

const IDIOMA_OPTS = ["EN", "JP", "PT"];
const SORTS = [
  { id: "relevancia", label: "RELEVANCIA" },
  { id: "menor-preco", label: "MENOR PRECO" },
  { id: "novidades", label: "NOVIDADES" },
];
const PAGE_SIZE = 8;

function toggle<T>(list: T[], value: T): T[] {
  return list.includes(value)
    ? list.filter((v) => v !== value)
    : [...list, value];
}

export default function SinglesBrowse({
  catalog,
  games,
  navGames,
}: {
  catalog: Single[];
  games: { id: string; label: string }[];
  navGames: { id: string; label: string }[];
}) {
  const router = useRouter();
  const params = useSearchParams();
  const initialGame = params.get("jogo");
  const q = (params.get("q") ?? "").trim().toLowerCase();

  // Price bounds come from the real catalog (cents). The range filter only shows
  // when there is an actual spread to slide across.
  const prices = catalog.map((c) => c.price);
  const priceFloor = prices.length ? Math.min(...prices) : 0;
  const priceCeil = prices.length ? Math.max(...prices) : 0;
  const hasPriceRange = priceCeil > priceFloor;

  const [gameSel, setGameSel] = useState<string[]>(
    initialGame && navGames.some((g) => g.id === initialGame)
      ? [initialGame]
      : [],
  );
  const [estado, setEstado] = useState<string[]>([]);
  const [idioma, setIdioma] = useState<string[]>([]);
  const [priceMin, setPriceMin] = useState(priceFloor);
  const [priceMax, setPriceMax] = useState(priceCeil);
  const [sort, setSort] = useState("relevancia");
  const [page, setPage] = useState(1);
  const [sheetOpen, setSheetOpen] = useState(false);

  // Estado options come from the real conditions in stock.
  const estadoOpts = useMemo(
    () => [...new Set(catalog.map((c) => c.condition).filter(Boolean))].sort(),
    [catalog],
  );
  const hasLanguages = useMemo(
    () => catalog.some((c) => c.language),
    [catalog],
  );

  const filtered = useMemo(() => {
    let list = catalog.filter((c) => {
      if (gameSel.length && !gameSel.includes(c.game)) return false;
      if (estado.length && !estado.includes(c.condition)) return false;
      // Language filter only bites cards that carry a language.
      if (idioma.length && c.language && !idioma.includes(c.language))
        return false;
      if (c.price < priceMin || c.price > priceMax) return false;
      if (q && !`${c.name} ${c.number} ${c.set}`.toLowerCase().includes(q))
        return false;
      return true;
    });
    if (sort === "menor-preco") {
      list = [...list].sort((a, b) => a.price - b.price);
    } else if (sort === "novidades") {
      list = [...list].reverse();
    }
    return list;
  }, [catalog, gameSel, estado, idioma, priceMin, priceMax, q, sort]);

  const pages = Math.max(1, Math.ceil(filtered.length / PAGE_SIZE));
  const current = Math.min(page, pages);
  const shown = filtered.slice((current - 1) * PAGE_SIZE, current * PAGE_SIZE);

  const selectedLabel =
    gameSel.length === 1
      ? navGames.find((g) => g.id === gameSel[0])?.label
      : undefined;
  const title = selectedLabel ? `${selectedLabel} · Singles` : "Todos · Singles";

  const reset = () => setPage(1);

  const sidebar = (
    <aside
      className="sticker h-max rounded-[14px] bg-surface p-5"
      style={{ ["--sh" as string]: "6px" }}
    >
      <div className="mb-5 font-pixel text-[12px] text-brand">FILTROS</div>

      {games.length > 1 && (
        <FilterGroup heading="JOGO">
          {games.map((o) => (
            <FilterChip
              key={o.id}
              active={gameSel.includes(o.id)}
              onClick={() => {
                setGameSel((g) => toggle(g, o.id));
                reset();
              }}
            >
              {o.label}
            </FilterChip>
          ))}
        </FilterGroup>
      )}

      <FilterGroup heading="TIPO">
        <FilterChip active onClick={() => {}}>
          Singles
        </FilterChip>
        <FilterChip
          active={false}
          onClick={() =>
            router.push(
              gameSel.length === 1
                ? `/selado?jogo=${gameSel[0]}`
                : "/selado",
            )
          }
        >
          Selados
        </FilterChip>
      </FilterGroup>

      {estadoOpts.length > 0 && (
        <FilterGroup heading="ESTADO">
          {estadoOpts.map((o) => (
            <FilterChip
              key={o}
              active={estado.includes(o)}
              onClick={() => {
                setEstado((e) => toggle(e, o));
                reset();
              }}
            >
              {o}
            </FilterChip>
          ))}
        </FilterGroup>
      )}

      {hasLanguages && (
        <FilterGroup heading="IDIOMA">
          {IDIOMA_OPTS.map((o) => (
            <FilterChip
              key={o}
              active={idioma.includes(o)}
              onClick={() => {
                setIdioma((l) => toggle(l, o));
                reset();
              }}
            >
              {o}
            </FilterChip>
          ))}
        </FilterGroup>
      )}

      {hasPriceRange && (
        <div className="mt-6">
          <div className="mb-3 font-pixel text-[9px] text-brand-soft">PRECO</div>
          <div className="relative h-3.5">
            <div className="absolute inset-x-0 top-1/2 h-3.5 -translate-y-1/2 rounded-[10px] border-[3px] border-brand bg-outline" />
            <div
              className="absolute top-1/2 h-3.5 -translate-y-1/2 rounded-[6px] bg-brand"
              style={{
                left: `${((priceMin - priceFloor) / (priceCeil - priceFloor)) * 100}%`,
                right: `${(1 - (priceMax - priceFloor) / (priceCeil - priceFloor)) * 100}%`,
              }}
            />
            <input
              type="range"
              aria-label="Preço mínimo"
              min={priceFloor}
              max={priceCeil}
              step={100}
              value={priceMin}
              onChange={(e) => {
                setPriceMin(Math.min(Number(e.target.value), priceMax));
                reset();
              }}
              className="range-thumb absolute inset-x-0 top-1/2 w-full -translate-y-1/2 appearance-none bg-transparent"
            />
            <input
              type="range"
              aria-label="Preço máximo"
              min={priceFloor}
              max={priceCeil}
              step={100}
              value={priceMax}
              onChange={(e) => {
                setPriceMax(Math.max(Number(e.target.value), priceMin));
                reset();
              }}
              className="range-thumb absolute inset-x-0 top-1/2 w-full -translate-y-1/2 appearance-none bg-transparent"
            />
          </div>
          <div className="mt-2 flex justify-between font-pixel text-[8px] text-faint">
            <span>{brl(priceMin)}</span>
            <span>{brl(priceMax)}</span>
          </div>
        </div>
      )}
    </aside>
  );

  return (
    <Container className="grid gap-7 py-9 md:grid-cols-[270px_1fr] md:items-start">
      <div className="hidden md:block">{sidebar}</div>

      <div className="min-w-0">
        {/* Toolbar */}
        <div className="mb-6 flex flex-wrap items-end justify-between gap-4">
          <div>
            <h1 className="font-display text-3xl font-bold leading-none text-white sm:text-[38px]">
              {title}
            </h1>
            <div className="mt-2 font-pixel text-[9px] text-faint">
              {filtered.length.toLocaleString("pt-BR")} RESULTADOS
            </div>
          </div>
          <div className="flex flex-wrap gap-2">
            {SORTS.map((s) => (
              <FilterChip
                key={s.id}
                active={sort === s.id}
                onClick={() => {
                  setSort(s.id);
                  reset();
                }}
              >
                {s.label}
              </FilterChip>
            ))}
          </div>
        </div>

        {/* Mobile filters toggle */}
        <button
          type="button"
          onClick={() => setSheetOpen((v) => !v)}
          className="arcade-press sticker mb-5 flex items-center gap-2 rounded-[10px] bg-surface px-4 py-2.5 font-pixel text-[10px] text-white md:hidden"
          style={{ ["--sh" as string]: "4px" }}
        >
          <SlidersHorizontal className="h-3.5 w-3.5" /> FILTROS
        </button>
        {sheetOpen && <div className="mb-6 md:hidden">{sidebar}</div>}

        {shown.length === 0 ? (
          <div className="sticker rounded-[14px] bg-surface p-10 text-center">
            <div className="font-display text-2xl font-bold text-white">
              Nenhuma carta encontrada
            </div>
            <p className="mt-2 font-pixel text-[9px] text-brand-soft">
              TENTE OUTRO FILTRO
            </p>
          </div>
        ) : (
          <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4 lg:gap-5">
            {shown.map((s) => (
              <SingleCard key={s.slug} item={s} showLanguage />
            ))}
          </div>
        )}

        {pages > 1 && (
          <div className="mt-9 flex items-center justify-center gap-2.5">
            <PageChip
              label="‹"
              disabled={current === 1}
              onClick={() => setPage(current - 1)}
            />
            {Array.from({ length: pages }, (_, i) => i + 1).map((n) => (
              <PageChip
                key={n}
                label={String(n)}
                active={n === current}
                onClick={() => setPage(n)}
              />
            ))}
            <PageChip
              label="›"
              disabled={current === pages}
              onClick={() => setPage(current + 1)}
            />
          </div>
        )}
      </div>
    </Container>
  );
}

function FilterGroup({
  heading,
  children,
}: {
  heading: string;
  children: React.ReactNode;
}) {
  return (
    <div className="mb-6">
      <div className="mb-3 font-pixel text-[9px] text-brand-soft">{heading}</div>
      <div className="flex flex-wrap gap-2">{children}</div>
    </div>
  );
}

function PageChip({
  label,
  active,
  disabled,
  onClick,
}: {
  label: string;
  active?: boolean;
  disabled?: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      disabled={disabled}
      onClick={onClick}
      aria-current={active ? "page" : undefined}
      className={`grid h-9 min-w-9 place-items-center rounded-[8px] border-[3px] border-outline px-2 font-pixel text-[10px] disabled:opacity-30 ${
        active ? "bg-brand text-white" : "bg-outline text-[#c9c9d1] hover:text-white"
      }`}
    >
      {label}
    </button>
  );
}
