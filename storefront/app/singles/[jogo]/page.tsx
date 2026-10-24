import { notFound } from "next/navigation";
import type { Metadata } from "next";
import JsonLd from "@/components/seo/JsonLd";
import { GameFaq, GameIntro, SetLinks } from "@/components/singles/CategoryExtras";
import SinglesBrowse from "@/components/singles/SinglesBrowse";
import { loadCatalog, scopeToGame } from "@/lib/catalog";
import { GAME_IDS, GAME_NAMES } from "@/lib/games";
import { SINGLES_PAGE_SIZE } from "@/lib/paging";
import { breadcrumbJsonLd, itemListJsonLd } from "@/lib/seo";

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
  const { singles, live } = await loadCatalog();
  const count = singles.filter((s) => s.game === jogo).length;
  return {
    title: `Cartas ${name} avulsas — singles, preços e coleções`,
    description: `${count > 0 ? `${count} cartas` : "Cartas"} ${name} avulsas em estoque na Collecta, por coleção e com preço de cada carta. Originais, conferidas uma a uma. Pix ou cartão em até 12x, envio para todo o Brasil ou retirada em Goiânia.`,
    alternates: { canonical: `/singles/${jogo}` },
    robots: live && count > 0 ? undefined : { index: false },
  };
}

export default async function SinglesGamePage({
  params,
}: {
  params: Promise<{ jogo: string }>;
}) {
  const { jogo } = await params;
  const name = GAME_NAMES[jogo];
  if (!name) {
    notFound();
  }
  const scope = scopeToGame(await loadCatalog(), jogo);
  return (
    <>
      <JsonLd
        data={breadcrumbJsonLd([
          { name: "Collecta", path: "/" },
          { name: `Cartas ${name}` },
        ])}
      />
      <JsonLd
        data={itemListJsonLd(
          `Cartas ${name} avulsas`,
          scope.singles.slice(0, SINGLES_PAGE_SIZE).map((s) => `/carta/${s.slug}`),
        )}
      />
      <SinglesBrowse
        catalog={scope.singles}
        games={scope.games}
        navGames={scope.navGames}
        setLabels={scope.setLabels}
        initialGame={jogo}
        basePath={`/singles/${jogo}`}
        intro={<GameIntro game={jogo} name={name} count={scope.singles.length} />}
        footer={
          <>
            <SetLinks game={jogo} name={name} sets={scope.sets} />
            <GameFaq name={name} />
          </>
        }
      />
    </>
  );
}
