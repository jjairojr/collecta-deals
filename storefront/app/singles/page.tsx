import { Suspense } from "react";
import SinglesBrowse from "@/components/singles/SinglesBrowse";
import { loadCatalog } from "@/lib/catalog";

export const revalidate = 60;

export default async function SinglesPage() {
  const { singles, games, navGames } = await loadCatalog();
  const gameOpts = games.map((g) => ({ id: g.id, label: g.name }));
  const navOpts = navGames.map((g) => ({ id: g.id, label: g.name }));
  return (
    <Suspense fallback={null}>
      <SinglesBrowse catalog={singles} games={gameOpts} navGames={navOpts} />
    </Suspense>
  );
}
