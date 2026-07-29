"use client";

import { useCallback, useEffect, useMemo, useState, useSyncExternalStore } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { ChevronDown, SlidersHorizontal } from "lucide-react";
import Container from "@/components/ui/Container";
import FilterChip from "@/components/ui/FilterChip";
import SingleCard from "@/components/SingleCard";
import {
  singleItem,
  trackFilterChange,
  trackSortChange,
  trackViewItemList,
} from "@/lib/analytics";
import { brl } from "@/lib/format";
import { SINGLES_PAGE_SIZE } from "@/lib/paging";
import type { Single } from "@/lib/types";

const IDIOMA_OPTS = ["EN", "JP", "PT"];
const SORTS = [
  { id: "relevancia", label: "RELEVANCIA" },
  { id: "menor-preco", label: "MENOR PRECO" },
  { id: "maior-preco", label: "MAIOR PRECO" },
  { id: "novidades", label: "NOVIDADES" },
];

function toggle<T>(list: T[], value: T): T[] {
  return list.includes(value)
    ? list.filter((v) => v !== value)
    : [...list, value];
}

// Filter/sort/page state lives in the URL so any view is shareable. Reads go
// through useSyncExternalStore on location.search (instead of useSearchParams,
// which would force a Suspense/CSR bailout on the statically generated pages
// and drop the grid from their HTML); writes are shallow history updates
// announced via a custom event.
const QUERY_EVENT = "collecta:query";

function subscribeQuery(onChange: () => void) {
  window.addEventListener("popstate", onChange);
  window.addEventListener(QUERY_EVENT, onChange);
  return () => {
    window.removeEventListener("popstate", onChange);
    window.removeEventListener(QUERY_EVENT, onChange);
  };
}

function useUrlQuery() {
  return useSyncExternalStore(
    subscribeQuery,
    () => window.location.search,
    () => "",
  );
}

export default function SinglesBrowse({
  catalog,
  games,
  navGames,
  initialGame = null,
  initialPage = 1,
  basePath = null,
  query = "",
  filtersOnTop = false,
}: {
  catalog: Single[];
  games: { id: string; label: string }[];
  navGames: { id: string; label: string }[];
  initialGame?: string | null;
  initialPage?: number;
  basePath?: string | null;
  query?: string;
  filtersOnTop?: boolean;
}) {
  const router = useRouter();
  const q = query.trim().toLowerCase();

  // Price bounds come from the real catalog (cents), snapped outward to whole
  // reais so both slider ends land exactly on the R$1 step grid. The range
  // filter only shows when there is an actual spread to slide across.
  const prices = catalog.map((c) => c.price);
  const priceFloor = prices.length
    ? Math.floor(Math.min(...prices) / 100) * 100
    : 0;
  const priceCeil = prices.length
    ? Math.ceil(Math.max(...prices) / 100) * 100
    : 0;
  const hasPriceRange = priceCeil > priceFloor;

  const search = useUrlQuery();
  const urlParams = useMemo(() => new URLSearchParams(search), [search]);

  // Query-param patcher. On a /pagina/N route a filter change must reset the
  // page, and the page lives in the path — so that one case is a real
  // navigation back to basePath; everything else is a shallow history write.
  const patch = useCallback(
    (updates: Record<string, string | null>, push = false) => {
      const params = new URLSearchParams(window.location.search);
      for (const [key, value] of Object.entries(updates)) {
        if (value === null) {
          params.delete(key);
        } else {
          params.set(key, value);
        }
      }
      const qs = params.toString();
      if (initialPage > 1 && basePath) {
        router.replace(`${basePath}${qs ? `?${qs}` : ""}`, { scroll: false });
        return;
      }
      const url = `${window.location.pathname}${qs ? `?${qs}` : ""}`;
      if (push) {
        window.history.pushState(null, "", url);
      } else {
        window.history.replaceState(null, "", url);
      }
      window.dispatchEvent(new Event(QUERY_EVENT));
    },
    [initialPage, basePath, router],
  );

  // The route game is the default selection; the `jogo` param overrides it
  // ("todos" encodes an explicit empty selection on game-scoped routes).
  const defaultGames = useMemo(
    () =>
      initialGame && navGames.some((g) => g.id === initialGame)
        ? [initialGame]
        : [],
    [initialGame, navGames],
  );
  const gameSel = useMemo(() => {
    const raw = urlParams.get("jogo");
    if (raw === null) {
      return defaultGames;
    }
    if (raw === "todos") {
      return [];
    }
    return raw.split(",").filter((id) => navGames.some((g) => g.id === id));
  }, [urlParams, defaultGames, navGames]);

  // Estado options come from the real conditions in stock.
  const estadoOpts = useMemo(
    () => [...new Set(catalog.map((c) => c.condition).filter(Boolean))].sort(),
    [catalog],
  );
  const hasLanguages = useMemo(
    () => catalog.some((c) => c.language),
    [catalog],
  );

  const estado = useMemo(
    () =>
      (urlParams.get("estado") ?? "")
        .split(",")
        .filter((v) => estadoOpts.includes(v)),
    [urlParams, estadoOpts],
  );
  const idioma = useMemo(
    () =>
      (urlParams.get("idioma") ?? "")
        .split(",")
        .filter((v) => IDIOMA_OPTS.includes(v)),
    [urlParams],
  );
  const colecao = useMemo(() => {
    const v = urlParams.get("colecao");
    return v ? [v] : [];
  }, [urlParams]);

  const urlSort = urlParams.get("ordem");
  const sort =
    urlSort !== null && SORTS.some((s) => s.id === urlSort)
      ? urlSort
      : "relevancia";

  const urlPagina = Number(urlParams.get("pagina"));
  const page =
    Number.isInteger(urlPagina) && urlPagina >= 1 ? urlPagina : initialPage;

  // The price slider keeps local state (a drag fires far too many events for
  // history.replaceState, which Safari rate-limits) and syncs with the `preco`
  // param — reais in the URL, cents in state — through a debounced write.
  const [priceMin, setPriceMin] = useState(priceFloor);
  const [priceMax, setPriceMax] = useState(priceCeil);
  const [sheetOpen, setSheetOpen] = useState(false);

  const urlPreco = urlParams.get("preco");
  useEffect(() => {
    const m = urlPreco?.match(/^(\d+)-(\d+)$/);
    const min = m
      ? Math.min(Math.max(Number(m[1]) * 100, priceFloor), priceCeil)
      : priceFloor;
    const max = m
      ? Math.min(Math.max(Number(m[2]) * 100, min), priceCeil)
      : priceCeil;
    setPriceMin(min);
    setPriceMax(max);
  }, [urlPreco, priceFloor, priceCeil]);

  useEffect(() => {
    const t = setTimeout(() => {
      const narrowed =
        hasPriceRange && (priceMin > priceFloor || priceMax < priceCeil);
      const value = narrowed ? `${priceMin / 100}-${priceMax / 100}` : null;
      if (new URLSearchParams(window.location.search).get("preco") !== value) {
        patch({ preco: value, pagina: null });
      }
    }, 250);
    return () => clearTimeout(t);
  }, [priceMin, priceMax, priceFloor, priceCeil, hasPriceRange, patch]);

  useEffect(() => {
    document.body.style.overflow = sheetOpen ? "hidden" : "";
    return () => {
      document.body.style.overflow = "";
    };
  }, [sheetOpen]);

  const applyGames = (next: string[]) => {
    const sameAsDefault =
      next.length === defaultGames.length &&
      next.every((id) => defaultGames.includes(id));
    patch({
      jogo: sameAsDefault ? null : next.length ? next.join(",") : "todos",
      pagina: null,
    });
  };

  const goPage = (n: number) =>
    patch({ pagina: n > 1 ? String(n) : null }, true);

  // Collection options scoped to the selected games. Selections that fall out
  // of scope stay stored but stop applying (and reappear if the game returns).
  const colecaoOpts = useMemo(() => {
    const scope = gameSel.length
      ? catalog.filter((c) => gameSel.includes(c.game))
      : catalog;
    return [...new Set(scope.map((c) => c.set).filter(Boolean))].sort((a, b) =>
      a.localeCompare(b, "pt-BR"),
    );
  }, [catalog, gameSel]);
  const colecaoSel = useMemo(
    () => colecao.filter((s) => colecaoOpts.includes(s)),
    [colecao, colecaoOpts],
  );

  const filtered = useMemo(() => {
    let list = catalog.filter((c) => {
      if (gameSel.length && !gameSel.includes(c.game)) return false;
      if (estado.length && !estado.includes(c.condition)) return false;
      // Language filter only bites cards that carry a language.
      if (idioma.length && c.language && !idioma.includes(c.language))
        return false;
      if (colecaoSel.length && !colecaoSel.includes(c.set)) return false;
      if (c.price < priceMin || c.price > priceMax) return false;
      if (q && !`${c.name} ${c.number} ${c.set}`.toLowerCase().includes(q))
        return false;
      return true;
    });
    if (sort === "menor-preco") {
      list = [...list].sort((a, b) => a.price - b.price);
    } else if (sort === "maior-preco") {
      list = [...list].sort((a, b) => b.price - a.price);
    } else if (sort === "novidades") {
      list = [...list].reverse();
    }
    return list;
  }, [catalog, gameSel, estado, idioma, colecaoSel, priceMin, priceMax, q, sort]);

  const pages = Math.max(1, Math.ceil(filtered.length / SINGLES_PAGE_SIZE));
  const current = Math.min(page, pages);
  const shown = useMemo(
    () =>
      filtered.slice(
        (current - 1) * SINGLES_PAGE_SIZE,
        current * SINGLES_PAGE_SIZE,
      ),
    [filtered, current],
  );

  const selectedLabel =
    gameSel.length === 1
      ? navGames.find((g) => g.id === gameSel[0])?.label
      : undefined;
  const title = selectedLabel ? `${selectedLabel} · Singles` : "Todos · Singles";

  useEffect(() => {
    if (shown.length > 0) {
      trackViewItemList(shown.map((s) => singleItem(s)), "singles_browse", title);
    }
  }, [shown, title]);

  const activeFilters =
    gameSel.length +
    estado.length +
    idioma.length +
    colecaoSel.length +
    (hasPriceRange && (priceMin > priceFloor || priceMax < priceCeil) ? 1 : 0);

  // Pagination stays real <a> links to /pagina/N while the view still matches
  // what the path encodes — that is what gives crawlers a path to every card.
  // Once a filter or sort is applied the page moves into the `pagina` query
  // param instead, so the chips fall back to shallow history writes.
  const gameMatchesURL =
    initialGame === null
      ? gameSel.length === 0
      : gameSel.length === 1 && gameSel[0] === initialGame;
  const linkedBase =
    basePath &&
    gameMatchesURL &&
    estado.length === 0 &&
    idioma.length === 0 &&
    colecaoSel.length === 0 &&
    sort === "relevancia" &&
    !q &&
    (!hasPriceRange || (priceMin === priceFloor && priceMax === priceCeil))
      ? basePath
      : null;
  const pageHref = linkedBase
    ? (n: number) => (n === 1 ? linkedBase : `${linkedBase}/pagina/${n}`)
    : null;

  const clearFilters = () => {
    setPriceMin(priceFloor);
    setPriceMax(priceCeil);
    setSheetOpen(false);
    if (q || initialGame) {
      router.push("/singles");
    } else {
      patch({
        jogo: null,
        colecao: null,
        estado: null,
        idioma: null,
        preco: null,
        ordem: null,
        pagina: null,
      });
    }
  };

  const filters = (
    <>
      <div className="mb-5 font-pixel text-[12px] text-brand">FILTROS</div>

      {games.length > 1 && (
        <FilterGroup heading="JOGO">
          {games.map((o) => (
            <FilterChip
              key={o.id}
              active={gameSel.includes(o.id)}
              onClick={() => {
                trackFilterChange("jogo", o.id, !gameSel.includes(o.id));
                applyGames(toggle(gameSel, o.id));
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
                ? `/selado/${gameSel[0]}`
                : "/selado",
            )
          }
        >
          Selados
        </FilterChip>
      </FilterGroup>

      {colecaoOpts.length > 1 && (
        <div className="mb-6">
          <div className="mb-3 font-pixel text-[9px] text-brand-soft">
            COLECAO
          </div>
          <FilterSelect
            value={colecaoSel[0] ?? ""}
            options={colecaoOpts}
            onChange={(v) => {
              trackFilterChange("colecao", v || "todas", Boolean(v));
              patch({ colecao: v || null, pagina: null });
            }}
          />
        </div>
      )}

      {estadoOpts.length > 0 && (
        <FilterGroup heading="ESTADO">
          {estadoOpts.map((o) => (
            <FilterChip
              key={o}
              active={estado.includes(o)}
              onClick={() => {
                trackFilterChange("estado", o, !estado.includes(o));
                const next = toggle(estado, o);
                patch({
                  estado: next.length ? next.join(",") : null,
                  pagina: null,
                });
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
                trackFilterChange("idioma", o, !idioma.includes(o));
                const next = toggle(idioma, o);
                patch({
                  idioma: next.length ? next.join(",") : null,
                  pagina: null,
                });
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
    </>
  );

  return (
    <Container
      className={
        filtersOnTop
          ? "py-9"
          : "grid gap-7 py-9 md:grid-cols-[270px_1fr] md:items-start"
      }
    >
      {!filtersOnTop && (
        <div className="hidden md:block">
          <aside className="sticker h-max rounded-[14px] bg-surface p-5 [--sh:6px]">
            {filters}
          </aside>
        </div>
      )}

      <div className="min-w-0">
        {/* Toolbar */}
        <div className="mb-6 flex flex-wrap items-end justify-between gap-4">
          <div>
            <h1 className="font-display text-3xl font-bold leading-none text-white sm:text-[38px]">
              {title}
            </h1>
            <div className="mt-2 font-pixel text-[9px] text-faint">
              {filtered.length.toLocaleString("pt-BR")}{" "}
              {filtered.length === 1 ? "RESULTADO" : "RESULTADOS"}
            </div>
          </div>
          <div className="flex flex-wrap gap-2">
            {SORTS.map((s) => (
              <FilterChip
                key={s.id}
                active={sort === s.id}
                onClick={() => {
                  if (sort !== s.id) {
                    trackSortChange(s.id);
                  }
                  patch({
                    ordem: s.id === "relevancia" ? null : s.id,
                    pagina: null,
                  });
                }}
              >
                {s.label}
              </FilterChip>
            ))}
          </div>
        </div>

        {/* Filters toggle: always visible in top mode, mobile-only otherwise */}
        <button
          type="button"
          onClick={() => setSheetOpen(true)}
          className={`arcade-press sticker mb-5 flex items-center gap-2 rounded-[10px] bg-surface px-4 py-2.5 font-pixel text-[10px] text-white [--sh:4px] ${filtersOnTop ? "" : "md:hidden"}`}
        >
          <SlidersHorizontal className="h-3.5 w-3.5" /> FILTROS
          {activeFilters > 0 && (
            <span className="grid h-5 min-w-5 place-items-center rounded-full border-2 border-outline bg-brand-soft px-1 font-pixel text-[9px] text-outline">
              {activeFilters}
            </span>
          )}
        </button>
        {sheetOpen && (
          <div
            className={`fixed inset-0 z-50 flex flex-col justify-end ${filtersOnTop ? "" : "md:hidden"}`}
          >
            <button
              type="button"
              aria-label="Fechar filtros"
              onClick={() => setSheetOpen(false)}
              className="absolute inset-0 bg-outline/70"
            />
            <div className="relative max-h-[80dvh] w-full overflow-y-auto rounded-t-[20px] border-t-4 border-brand bg-surface p-5 md:mx-auto md:max-w-xl md:border-x-4">
              {filters}
              <button
                type="button"
                onClick={() => setSheetOpen(false)}
                className="arcade-press sticker mt-2 w-full rounded-[10px] bg-brand px-6 py-3.5 font-pixel text-[11px] text-white [--sh:4px]"
              >
                VER {filtered.length.toLocaleString("pt-BR")}{" "}
                {filtered.length === 1 ? "RESULTADO" : "RESULTADOS"}
              </button>
            </div>
          </div>
        )}

        {shown.length === 0 ? (
          <div className="sticker rounded-[14px] bg-surface p-10 text-center">
            <div className="font-display text-2xl font-bold text-white">
              Nenhuma carta encontrada
            </div>
            <p className="mt-2 font-pixel text-[9px] text-brand-soft">
              TENTE OUTRO FILTRO
            </p>
            <div className="mt-6 flex flex-wrap justify-center gap-3">
              <button
                type="button"
                onClick={clearFilters}
                className="arcade-press sticker rounded-[10px] bg-brand px-5 py-3 font-pixel text-[10px] text-white [--sh:4px]"
              >
                LIMPAR FILTROS
              </button>
              <Link
                href={
                  gameSel.length === 1
                    ? `/selado/${gameSel[0]}`
                    : "/selado"
                }
                className="arcade-press sticker rounded-[10px] bg-brand-soft px-5 py-3 font-pixel text-[10px] text-outline [--sh:4px]"
              >
                VER SELADOS
              </Link>
            </div>
          </div>
        ) : (
          <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4 lg:gap-5">
            {shown.map((s) => (
              <SingleCard key={s.slug} item={s} showLanguage />
            ))}
          </div>
        )}

        {pages > 1 && (
          <div className="mt-9 flex flex-wrap items-center justify-center gap-2.5">
            <PageChip
              label="‹"
              ariaLabel="Página anterior"
              disabled={current === 1}
              href={pageHref && current > 1 ? pageHref(current - 1) : undefined}
              onClick={() => goPage(current - 1)}
            />
            {pageItems(pages, current).map((it, i) =>
              it === "gap" ? (
                <span
                  key={`gap-${i}`}
                  className="px-1 font-pixel text-[10px] text-faint"
                >
                  …
                </span>
              ) : (
                <PageChip
                  key={it}
                  label={String(it)}
                  ariaLabel={`Página ${it}`}
                  active={it === current}
                  href={pageHref ? pageHref(it) : undefined}
                  onClick={() => goPage(it)}
                />
              ),
            )}
            <PageChip
              label="›"
              ariaLabel="Próxima página"
              disabled={current === pages}
              href={
                pageHref && current < pages ? pageHref(current + 1) : undefined
              }
              onClick={() => goPage(current + 1)}
            />
          </div>
        )}
      </div>
    </Container>
  );
}

function pageItems(pages: number, current: number): (number | "gap")[] {
  if (pages <= 7) {
    return Array.from({ length: pages }, (_, i) => i + 1);
  }
  const wanted = [
    ...new Set(
      [1, current - 1, current, current + 1, pages].filter(
        (n) => n >= 1 && n <= pages,
      ),
    ),
  ].sort((a, b) => a - b);
  const out: (number | "gap")[] = [];
  wanted.forEach((n, i) => {
    if (i > 0 && n - wanted[i - 1] > 1) {
      out.push("gap");
    }
    out.push(n);
  });
  return out;
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

// Single-choice dropdown styled like the filter chips: chip trigger (pink when
// a collection is picked) and a sticker panel with the options.
function FilterSelect({
  value,
  options,
  onChange,
}: {
  value: string;
  options: string[];
  onChange: (value: string) => void;
}) {
  const [open, setOpen] = useState(false);
  const pick = (v: string) => {
    onChange(v);
    setOpen(false);
  };
  return (
    <div className="relative">
      <button
        type="button"
        aria-expanded={open}
        onClick={() => setOpen((o) => !o)}
        className={`flex w-full items-center justify-between gap-2 rounded-[8px] border-[3px] border-outline px-3 py-2 text-xs font-bold transition-colors ${
          value
            ? "bg-brand text-white"
            : "bg-outline text-[#c9c9d1] hover:text-white"
        }`}
      >
        <span className="truncate">{value || "Todas"}</span>
        <ChevronDown
          className={`h-3.5 w-3.5 shrink-0 transition-transform ${open ? "rotate-180" : ""}`}
        />
      </button>
      {open && (
        <>
          <button
            type="button"
            aria-label="Fechar coleções"
            onClick={() => setOpen(false)}
            className="fixed inset-0 z-10 cursor-default"
          />
          <div className="sticker absolute inset-x-0 top-full z-20 mt-2 max-h-60 overflow-y-auto rounded-[10px] bg-surface p-1.5 [--sh:4px]">
            {["", ...options].map((o) => (
              <button
                key={o || "todas"}
                type="button"
                onClick={() => pick(o)}
                className={`block w-full rounded-[6px] px-3 py-2 text-left text-xs font-bold transition-colors ${
                  o === (value || "")
                    ? "bg-brand text-white"
                    : "text-[#c9c9d1] hover:bg-outline hover:text-white"
                }`}
              >
                {o || "Todas"}
              </button>
            ))}
          </div>
        </>
      )}
    </div>
  );
}

function PageChip({
  label,
  ariaLabel,
  active,
  disabled,
  href,
  onClick,
}: {
  label: string;
  ariaLabel: string;
  active?: boolean;
  disabled?: boolean;
  href?: string;
  onClick: () => void;
}) {
  const className = `grid h-9 min-w-9 place-items-center rounded-[8px] border-[3px] border-outline px-2 font-pixel text-[10px] disabled:opacity-30 ${
    active ? "bg-brand text-white" : "bg-outline text-[#c9c9d1] hover:text-white"
  }`;
  if (href && !disabled) {
    return (
      <Link
        href={href}
        aria-label={ariaLabel}
        aria-current={active ? "page" : undefined}
        className={className}
      >
        {label}
      </Link>
    );
  }
  return (
    <button
      type="button"
      disabled={disabled}
      onClick={onClick}
      aria-label={ariaLabel}
      aria-current={active ? "page" : undefined}
      className={className}
    >
      {label}
    </button>
  );
}
