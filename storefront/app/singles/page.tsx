import { permanentRedirect } from "next/navigation";
import type { Metadata } from "next";
import SinglesBrowse from "@/components/singles/SinglesBrowse";
import { loadCatalog } from "@/lib/catalog";
import { isGameId } from "@/lib/games";

export const revalidate = 60;

export async function generateMetadata(): Promise<Metadata> {
  const { singles, live } = await loadCatalog();
  return {
    title: "Cartas avulsas (singles) de TCG",
    description: `${singles.length} cartas avulsas de Pokémon, One Piece e Riftbound em estoque — conferidas uma a uma, com envio para todo o Brasil e pedido pelo WhatsApp.`,
    alternates: { canonical: "/singles" },
    robots: live ? undefined : { index: false },
  };
}

export default async function SinglesPage({
  searchParams,
}: {
  searchParams: Promise<Record<string, string | string[] | undefined>>;
}) {
  const params = await searchParams;
  const jogo = typeof params.jogo === "string" ? params.jogo : undefined;
  const q = typeof params.q === "string" ? params.q : undefined;
  if (jogo && isGameId(jogo)) {
    // Forward shareable filter params, but not q: /singles/[jogo] is static
    // and does not apply a search query.
    const rest = new URLSearchParams();
    for (const [key, value] of Object.entries(params)) {
      if (key === "jogo" || key === "q" || value === undefined) {
        continue;
      }
      for (const v of Array.isArray(value) ? value : [value]) {
        rest.append(key, v);
      }
    }
    const qs = rest.toString();
    permanentRedirect(`/singles/${jogo}${qs ? `?${qs}` : ""}`);
  }
  const { singles, games, navGames } = await loadCatalog();
  const gameOpts = games.map((g) => ({ id: g.id, label: g.name }));
  const navOpts = navGames.map((g) => ({ id: g.id, label: g.name }));
  return (
    <SinglesBrowse
      catalog={singles}
      games={gameOpts}
      navGames={navOpts}
      query={q ?? ""}
    />
  );
}
