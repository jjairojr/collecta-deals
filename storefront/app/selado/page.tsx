import Link from "next/link";
import { permanentRedirect } from "next/navigation";
import type { Metadata } from "next";
import Container from "@/components/ui/Container";
import FaqSection from "@/components/ui/FaqSection";
import SectionHead from "@/components/ui/SectionHead";
import SealedGrid from "@/components/SealedGrid";
import KindToggle from "@/components/ui/KindToggle";
import { CategoryIntro } from "@/components/singles/CategoryExtras";
import { SealedCollectionLinks, SealedFormatGuide } from "@/components/sealed/SealedExtras";
import JsonLd from "@/components/seo/JsonLd";
import { loadCatalog } from "@/lib/catalog";
import { pixelText } from "@/lib/format";
import { GAME_NAMES, isGameId } from "@/lib/games";
import { liveCollections } from "@/lib/sealed-guide";
import { SEALED_GENERAL } from "@/lib/sealed-faq";
import { breadcrumbJsonLd, itemListJsonLd } from "@/lib/seo";

export const revalidate = 60;

export async function generateMetadata(): Promise<Metadata> {
  const { live } = await loadCatalog();
  return {
    title: "Selados de TCG: booster box, ETB e blister de Pokémon e One Piece",
    description:
      "Selados de card games com lacre original: booster box, ETB, blister, boxes de coleção e decks de Pokémon, One Piece e Riftbound. Pix ou cartão, envio para todo o Brasil e loja física em Goiânia.",
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
  const games = [...new Set(sealed.map((s) => s.game))].filter(isGameId);
  return (
    <>
      <JsonLd
        data={breadcrumbJsonLd([
          { name: "Collecta", path: "/" },
          { name: "Selados" },
        ])}
      />
      {sealed.length > 0 && (
        <JsonLd data={itemListJsonLd("Produtos selados de TCG", sealed.map((s) => `/selado/${s.slug}`))} />
      )}
      <Container className="py-9">
        <SectionHead
          title="Produtos selados"
          eyebrow="LACRADO E CONFERIDO"
          size="md"
          heading="h1"
        />
        <KindToggle singlesHref="/singles" />
        <CategoryIntro>
          {sealed.length > 0 ? `${sealed.length} produtos selados em estoque: ` : "Produtos selados: "}
          booster box, ETB, blister, boxes de coleção e decks de card games com lacre original de fábrica, conferidos
          antes do envio. Pix ou cartão, envio para todo o Brasil ou retirada na loja em Goiânia.
        </CategoryIntro>
        {games.length > 0 && (
          <div className="mb-8 mt-5 flex flex-wrap gap-3">
            {games.map((g) => (
              <Link
                key={g}
                href={`/selado/${g}`}
                className="arcade-press sticker rounded-[10px] bg-royal px-4 py-2.5 font-pixel text-[10px] text-white [--sh:4px]"
              >
                SELADOS {pixelText(GAME_NAMES[g])}
              </Link>
            ))}
          </div>
        )}
        <SealedGrid list={sealed} />
        <SealedCollectionLinks title="Coleções seladas" collections={liveCollections(sealed)} />
        <SealedFormatGuide title="Qual selado comprar?" list={sealed} />
        <FaqSection items={SEALED_GENERAL} />
      </Container>
    </>
  );
}
