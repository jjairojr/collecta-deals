import { notFound } from "next/navigation";
import type { Metadata } from "next";
import Container from "@/components/ui/Container";
import SectionHead from "@/components/ui/SectionHead";
import SealedGrid from "@/components/SealedGrid";
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
  const { singles, sealed, live } = await loadCatalog();
  const singlesCount = singles.filter((s) => s.game === jogo).length;
  const sealedCount = sealed.filter((s) => s.game === jogo).length;
  const parts = [
    singlesCount > 0 ? `${singlesCount} cartas avulsas` : "",
    sealedCount > 0 ? `${sealedCount} produtos selados` : "",
  ].filter(Boolean);
  return {
    title: `${name} — singles e selados`,
    description: `${parts.length > 0 ? parts.join(" e ") : "Singles e selados"} de ${name} em estoque na Collecta — conferidos um a um, com envio para todo o Brasil e pedido pelo WhatsApp.`,
    alternates: { canonical: `/jogo/${jogo}` },
    robots: live ? undefined : { index: false },
  };
}

export default async function JogoPage({
  params,
}: {
  params: Promise<{ jogo: string }>;
}) {
  const { jogo } = await params;
  const name = GAME_NAMES[jogo];
  if (!name) {
    notFound();
  }
  const { singles, sealed, games, navGames } = await loadCatalog();
  const sealedList = sealed.filter((s) => s.game === jogo);
  const hasSingles = singles.some((s) => s.game === jogo);
  const gameOpts = games.map((g) => ({ id: g.id, label: g.name }));
  const navOpts = navGames.map((g) => ({ id: g.id, label: g.name }));
  return (
    <>
      {sealedList.length > 0 && (
        <Container className="pt-9">
          <SectionHead
            title={`${name} · Selado`}
            eyebrow="LACRADO E CONFERIDO"
            size="md"
            heading={hasSingles ? "h2" : "h1"}
            action={{ label: "VER TODOS ›", href: `/selado/${jogo}` }}
          />
          <SealedGrid list={sealedList} />
        </Container>
      )}
      {(hasSingles || sealedList.length === 0) && (
        <SinglesBrowse
          catalog={singles}
          games={gameOpts}
          navGames={navOpts}
          initialGame={jogo}
          filtersOnTop
        />
      )}
    </>
  );
}
