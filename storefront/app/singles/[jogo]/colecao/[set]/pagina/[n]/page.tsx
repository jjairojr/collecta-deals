import { notFound } from "next/navigation";
import type { Metadata } from "next";
import SinglesBrowse from "@/components/singles/SinglesBrowse";
import { loadCatalog, scopeToGame } from "@/lib/catalog";
import { GAME_NAMES } from "@/lib/games";
import { pageCount } from "@/lib/paging";
import { findSet, setsOf } from "@/lib/sets";

export const revalidate = 60;

export async function generateStaticParams() {
  const { singles, games } = await loadCatalog();
  return games.flatMap((g) =>
    setsOf(singles, g.id).flatMap((s) =>
      Array.from({ length: pageCount(s.count) - 1 }, (_, i) => ({ jogo: g.id, set: s.slug, n: String(i + 2) })),
    ),
  );
}

function parsePage(n: string): number | null {
  const page = Number(n);
  return Number.isInteger(page) && page >= 2 ? page : null;
}

export async function generateMetadata({
  params,
}: {
  params: Promise<{ jogo: string; set: string; n: string }>;
}): Promise<Metadata> {
  const { jogo, set, n } = await params;
  const name = GAME_NAMES[jogo];
  const page = parsePage(n);
  if (!name || page === null) {
    return {};
  }
  const { singles, live } = await loadCatalog();
  const found = findSet(singles, jogo, set);
  if (!found) {
    return {};
  }
  return {
    title: `Cartas ${found.label} de ${name} (singles) — página ${page}`,
    description: `Página ${page} das ${found.count} cartas da coleção ${found.label} de ${name} em estoque na Collecta. Conferidas uma a uma, envio para todo o Brasil.`,
    alternates: { canonical: `/singles/${jogo}/colecao/${found.slug}/pagina/${page}` },
    robots: live ? undefined : { index: false },
  };
}

export default async function SetPagePage({
  params,
}: {
  params: Promise<{ jogo: string; set: string; n: string }>;
}) {
  const { jogo, set, n } = await params;
  const page = parsePage(n);
  if (!GAME_NAMES[jogo] || page === null) {
    notFound();
  }
  const scope = scopeToGame(await loadCatalog(), jogo);
  const found = findSet(scope.singles, jogo, set);
  if (!found || page > pageCount(found.count)) {
    notFound();
  }
  return (
    <SinglesBrowse
      catalog={scope.singles}
      games={scope.games}
      navGames={scope.navGames}
      setLabels={scope.setLabels}
      initialGame={jogo}
      initialSet={found.code}
      initialPage={page}
      basePath={`/singles/${jogo}/colecao/${found.slug}`}
    />
  );
}
