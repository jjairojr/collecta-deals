import { notFound } from "next/navigation";
import type { Metadata } from "next";
import SinglesBrowse from "@/components/singles/SinglesBrowse";
import { loadCatalog, scopeToGame } from "@/lib/catalog";
import { GAME_NAMES } from "@/lib/games";
import { pageCount } from "@/lib/paging";

export const revalidate = 60;

// Page 1 lives at /singles/[jogo] — these routes only ever cover 2..N, so the
// first page never has two URLs.
export async function generateStaticParams() {
  const { games } = await loadCatalog();
  return games.flatMap((g) =>
    Array.from({ length: pageCount(g.count) - 1 }, (_, i) => ({
      jogo: g.id,
      n: String(i + 2),
    })),
  );
}

function parsePage(n: string): number | null {
  const page = Number(n);
  return Number.isInteger(page) && page >= 2 ? page : null;
}

export async function generateMetadata({
  params,
}: {
  params: Promise<{ jogo: string; n: string }>;
}): Promise<Metadata> {
  const { jogo, n } = await params;
  const name = GAME_NAMES[jogo];
  const page = parsePage(n);
  if (!name || page === null) {
    return {};
  }
  const { games, live } = await loadCatalog();
  const count = games.find((g) => g.id === jogo)?.count ?? 0;
  return {
    title: `Cartas ${name} avulsas (singles) — página ${page}`,
    description: `Página ${page} do catálogo de ${count} cartas ${name} avulsas em estoque na Collecta, conferidas uma a uma. Pix ou cartão e envio para todo o Brasil.`,
    alternates: { canonical: `/singles/${jogo}/pagina/${page}` },
    robots: live ? undefined : { index: false },
  };
}

export default async function SinglesGamePagePage({
  params,
}: {
  params: Promise<{ jogo: string; n: string }>;
}) {
  const { jogo, n } = await params;
  const page = parsePage(n);
  if (!GAME_NAMES[jogo] || page === null) {
    notFound();
  }
  const scope = scopeToGame(await loadCatalog(), jogo);
  if (page > pageCount(scope.singles.length)) {
    notFound();
  }
  return (
    <SinglesBrowse
      catalog={scope.singles}
      games={scope.games}
      navGames={scope.navGames}
      setLabels={scope.setLabels}
      initialGame={jogo}
      initialPage={page}
      basePath={`/singles/${jogo}`}
    />
  );
}
