import { notFound } from "next/navigation";
import type { Metadata } from "next";
import JsonLd from "@/components/seo/JsonLd";
import SinglesBrowse from "@/components/singles/SinglesBrowse";
import { loadCatalog } from "@/lib/catalog";
import { brl } from "@/lib/format";
import { GAME_NAMES } from "@/lib/games";
import { breadcrumbJsonLd } from "@/lib/seo";
import { findSet, setsOf } from "@/lib/sets";

export const revalidate = 60;

export async function generateStaticParams() {
  const { singles, games } = await loadCatalog();
  return games.flatMap((g) => setsOf(singles, g.id).map((s) => ({ jogo: g.id, set: s.slug })));
}

export async function generateMetadata({
  params,
}: {
  params: Promise<{ jogo: string; set: string }>;
}): Promise<Metadata> {
  const { jogo, set } = await params;
  const name = GAME_NAMES[jogo];
  if (!name) {
    return {};
  }
  const { singles, live } = await loadCatalog();
  const found = findSet(singles, jogo, set);
  if (!found) {
    return {};
  }
  return {
    title: `Cartas ${found.label} de ${name} (singles)`,
    description: `${found.count} cartas da coleção ${found.label} de ${name} em estoque na Collecta, a partir de ${brl(found.minPrice)}. Conferidas uma a uma, Pix ou cartão em até 12x e envio para todo o Brasil.`,
    alternates: { canonical: `/singles/${jogo}/colecao/${found.slug}` },
    robots: live ? undefined : { index: false },
  };
}

export default async function SetPage({ params }: { params: Promise<{ jogo: string; set: string }> }) {
  const { jogo, set } = await params;
  const name = GAME_NAMES[jogo];
  if (!name) {
    notFound();
  }
  const { singles, games, navGames } = await loadCatalog();
  const found = findSet(singles, jogo, set);
  if (!found) {
    notFound();
  }
  const gameOpts = games.map((g) => ({ id: g.id, label: g.name }));
  const navOpts = navGames.map((g) => ({ id: g.id, label: g.name }));
  return (
    <>
      <JsonLd
        data={breadcrumbJsonLd([
          { name: "Collecta", path: "/" },
          { name, path: `/singles/${jogo}` },
          { name: found.label },
        ])}
      />
      <SinglesBrowse
        catalog={singles}
        games={gameOpts}
        navGames={navOpts}
        initialGame={jogo}
        initialSet={found.code}
        basePath={`/singles/${jogo}/colecao/${found.slug}`}
      />
    </>
  );
}
