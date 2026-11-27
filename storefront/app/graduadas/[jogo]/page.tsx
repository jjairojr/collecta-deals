import { notFound } from "next/navigation";
import type { Metadata } from "next";
import JsonLd from "@/components/seo/JsonLd";
import { CategoryIntro } from "@/components/singles/CategoryExtras";
import SinglesBrowse from "@/components/singles/SinglesBrowse";
import { packSingles } from "@/lib/singles-pack";
import { loadCatalog, scopeToGraded } from "@/lib/catalog";
import { GAME_IDS, GAME_NAMES } from "@/lib/games";
import { SINGLES_PAGE_SIZE } from "@/lib/paging";
import { breadcrumbJsonLd, itemListJsonLd } from "@/lib/seo";
import { SEM_JUROS } from "@/lib/payment";

export const revalidate = 60;

export function generateStaticParams() {
  return GAME_IDS.map((jogo) => ({ jogo }));
}

export async function generateMetadata({
  params,
}: {
  params: Promise<{ jogo: string }>;
}): Promise<Metadata> {
  const { jogo } = await params;
  const name = GAME_NAMES[jogo];
  if (!name) {
    return {};
  }
  const catalog = await loadCatalog();
  const count = scopeToGraded(catalog, jogo).singles.length;
  return {
    title: `Cartas ${name} graduadas — slabs PSA, BGS e CGC`,
    description: `${count > 0 ? `${count} cartas` : "Cartas"} ${name} graduadas em estoque na Collecta: slabs PSA, BGS, CGC e outras, com foto real de cada uma. Pix ou cartão em até 6x${SEM_JUROS}, envio para todo o Brasil ou retirada em Goiânia.`,
    alternates: { canonical: `/graduadas/${jogo}` },
    robots: catalog.live && count > 0 ? undefined : { index: false },
  };
}

export default async function GradedGamePage({
  params,
}: {
  params: Promise<{ jogo: string }>;
}) {
  const { jogo } = await params;
  const name = GAME_NAMES[jogo];
  if (!name) {
    notFound();
  }
  const scope = scopeToGraded(await loadCatalog(), jogo);
  if (scope.singles.length === 0) {
    notFound();
  }
  return (
    <>
      <JsonLd
        data={breadcrumbJsonLd([
          { name: "Collecta", path: "/" },
          { name: `Cartas ${name}`, path: `/singles/${jogo}` },
          { name: "Graduadas" },
        ])}
      />
      <JsonLd
        data={itemListJsonLd(
          `Cartas ${name} graduadas`,
          scope.singles.slice(0, SINGLES_PAGE_SIZE).map((s) => `/carta/${s.slug}`),
        )}
      />
      <SinglesBrowse
        catalog={packSingles(scope.singles)}
        games={scope.games}
        navGames={scope.navGames}
        setLabels={scope.setLabels}
        initialGame={jogo}
        basePath={`/graduadas/${jogo}`}
        kindLabel="Graduadas"
        intro={
          <CategoryIntro>
            {scope.singles.length} {scope.singles.length === 1 ? "carta graduada" : "cartas graduadas"} em estoque
            agora, cada uma com a foto real da slab e a nota da etiqueta. Pague por Pix ou cartão em até 6x{SEM_JUROS}, receba em
            todo o Brasil ou retire na loja em Goiânia.
          </CategoryIntro>
        }
      />
    </>
  );
}
