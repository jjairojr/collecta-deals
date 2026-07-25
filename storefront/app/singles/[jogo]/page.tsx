import { notFound } from "next/navigation";
import type { Metadata } from "next";
import SinglesBrowse from "@/components/singles/SinglesBrowse";
import { loadCatalog } from "@/lib/catalog";
import { GAME_IDS, GAME_NAMES } from "@/lib/games";

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
    title: `Cartas ${name} avulsas (singles)`,
    description: `${count > 0 ? `${count} cartas` : "Cartas"} ${name} avulsas em estoque na Collecta — conferidas uma a uma, com envio para todo o Brasil e pedido pelo WhatsApp.`,
    alternates: { canonical: `/singles/${jogo}` },
    robots: live ? undefined : { index: false },
  };
}

export default async function SinglesGamePage({
  params,
}: {
  params: Promise<{ jogo: string }>;
}) {
  const { jogo } = await params;
  if (!GAME_NAMES[jogo]) {
    notFound();
  }
  const { singles, games, navGames } = await loadCatalog();
  const gameOpts = games.map((g) => ({ id: g.id, label: g.name }));
  const navOpts = navGames.map((g) => ({ id: g.id, label: g.name }));
  return (
    <SinglesBrowse
      catalog={singles}
      games={gameOpts}
      navGames={navOpts}
      initialGame={jogo}
    />
  );
}
