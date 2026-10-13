import { notFound } from "next/navigation";
import type { Metadata } from "next";
import SinglesBrowse from "@/components/singles/SinglesBrowse";
import { loadCatalog } from "@/lib/catalog";
import { pageCount } from "@/lib/paging";

export const revalidate = 60;

export async function generateStaticParams() {
  const { singles } = await loadCatalog();
  return Array.from({ length: pageCount(singles.length) - 1 }, (_, i) => ({ n: String(i + 2) }));
}

function parsePage(n: string): number | null {
  const page = Number(n);
  return Number.isInteger(page) && page >= 2 ? page : null;
}

export async function generateMetadata({ params }: { params: Promise<{ n: string }> }): Promise<Metadata> {
  const { n } = await params;
  const page = parsePage(n);
  if (page === null) {
    return {};
  }
  const { singles, live } = await loadCatalog();
  return {
    title: `Cartas avulsas (singles) de TCG — página ${page}`,
    description: `Página ${page} do catálogo de ${singles.length} cartas avulsas de Pokémon, One Piece, Riftbound, Lorcana e Gundam na Collecta. Pix ou cartão e envio para todo o Brasil.`,
    alternates: { canonical: `/singles/pagina/${page}` },
    robots: live ? undefined : { index: false },
  };
}

export default async function SinglesPagePage({ params }: { params: Promise<{ n: string }> }) {
  const { n } = await params;
  const page = parsePage(n);
  if (page === null) {
    notFound();
  }
  const { singles, games, navGames } = await loadCatalog();
  if (page > pageCount(singles.length)) {
    notFound();
  }
  const gameOpts = games.map((g) => ({ id: g.id, label: g.name }));
  const navOpts = navGames.map((g) => ({ id: g.id, label: g.name }));
  return (
    <SinglesBrowse catalog={singles} games={gameOpts} navGames={navOpts} initialPage={page} basePath="/singles" />
  );
}
