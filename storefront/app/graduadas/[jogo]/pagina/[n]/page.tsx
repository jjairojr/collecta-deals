import { notFound } from "next/navigation";
import type { Metadata } from "next";
import SinglesBrowse from "@/components/singles/SinglesBrowse";
import { packSingles } from "@/lib/singles-pack";
import ListPageJsonLd from "@/components/seo/ListPageJsonLd";
import { loadCatalog, scopeToGraded } from "@/lib/catalog";
import { GAME_IDS, GAME_NAMES } from "@/lib/games";
import { pageCount } from "@/lib/paging";

export const revalidate = 60;

// Page 1 lives at /graduadas/[jogo]; these routes only cover 2..N.
export async function generateStaticParams() {
  const catalog = await loadCatalog();
  return GAME_IDS.flatMap((jogo) =>
    Array.from({ length: pageCount(scopeToGraded(catalog, jogo).singles.length) - 1 }, (_, i) => ({
      jogo,
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
  const { live } = await loadCatalog();
  return {
    title: `Cartas ${name} graduadas — página ${page}`,
    description: `Página ${page} das cartas ${name} graduadas (slabs) em estoque na Collecta. Pix ou cartão e envio para todo o Brasil.`,
    alternates: { canonical: `/graduadas/${jogo}/pagina/${page}` },
    robots: live ? undefined : { index: false },
  };
}

export default async function GradedGamePagePage({
  params,
}: {
  params: Promise<{ jogo: string; n: string }>;
}) {
  const { jogo, n } = await params;
  const page = parsePage(n);
  if (!GAME_NAMES[jogo] || page === null) {
    notFound();
  }
  const scope = scopeToGraded(await loadCatalog(), jogo);
  if (page > pageCount(scope.singles.length)) {
    notFound();
  }
  return (
    <>
      <ListPageJsonLd
        name={`Cartas ${GAME_NAMES[jogo]} graduadas`}
        crumbs={[{ name: "Collecta", path: "/" }, { name: "Graduadas", path: `/graduadas/${jogo}` }, { name: `Página ${page}` }]}
        slugs={scope.singles.map((s) => s.slug)}
        page={page}
      />
    <SinglesBrowse
      catalog={packSingles(scope.singles)}
      games={scope.games}
      navGames={scope.navGames}
      setLabels={scope.setLabels}
      initialGame={jogo}
      initialPage={page}
      basePath={`/graduadas/${jogo}`}
      kindLabel="Graduadas"
    />
    </>
  );
}
