import { permanentRedirect } from "next/navigation";
import type { Metadata } from "next";
import Container from "@/components/ui/Container";
import SectionHead from "@/components/ui/SectionHead";
import SealedGrid from "@/components/SealedGrid";
import { loadCatalog } from "@/lib/catalog";
import { isGameId } from "@/lib/games";

export const revalidate = 60;

export async function generateMetadata(): Promise<Metadata> {
  const { live } = await loadCatalog();
  return {
    title: "Produto selado de TCG",
    description:
      "Booster boxes, ETBs e mais — lacrados e conferidos. Pokémon, One Piece e Riftbound com envio para todo o Brasil.",
    alternates: { canonical: "/selado" },
    robots: live ? undefined : { index: false },
  };
}

export default async function SeladoPage({
  searchParams,
}: {
  searchParams: Promise<{ jogo?: string }>;
}) {
  const { jogo } = await searchParams;
  if (jogo && isGameId(jogo)) {
    permanentRedirect(`/selado/${jogo}`);
  }
  const { sealed } = await loadCatalog();
  return (
    <Container className="py-9">
      <SectionHead
        title="Produto selado"
        eyebrow="LACRADO E CONFERIDO"
        size="md"
        heading="h1"
      />
      <SealedGrid list={sealed} />
    </Container>
  );
}
