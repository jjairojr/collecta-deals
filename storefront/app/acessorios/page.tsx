import Link from "next/link";
import type { Metadata } from "next";
import Container from "@/components/ui/Container";
import SectionHead from "@/components/ui/SectionHead";
import SealedGrid from "@/components/SealedGrid";
import { CategoryIntro } from "@/components/singles/CategoryExtras";
import GuideLinks from "@/components/guide/GuideLinks";
import { categoryPath, liveCategories } from "@/lib/accessory-guide";
import { loadCatalog } from "@/lib/catalog";
import { pixelText } from "@/lib/format";
import { guidesFor } from "@/lib/guides";

export const revalidate = 60;

export async function generateMetadata(): Promise<Metadata> {
  const { live } = await loadCatalog();
  return {
    title: "Acessórios de TCG: sleeves, deck box, álbum e toploader",
    description:
      "Sleeves, deck boxes, álbuns, toploaders e slab guards para o seu TCG, com envio para todo o Brasil e retirada na loja em Goiânia.",
    alternates: { canonical: "/acessorios" },
    robots: live ? undefined : { index: false },
  };
}

export default async function AcessoriosPage() {
  const { accessories } = await loadCatalog();
  const categories = liveCategories(accessories);
  return (
    <Container className="py-9">
      <SectionHead
        title="Acessórios"
        eyebrow="PROTEJA SUA COLECAO"
        size="md"
        heading="h1"
      />
      <CategoryIntro>
        Sleeves, deck boxes, álbuns, toploaders e slab guards para guardar e jogar com cartas de Pokémon, One Piece,
        Riftbound e outros card games.
      </CategoryIntro>
      {categories.length > 0 && (
        <div className="mb-8 mt-5 flex flex-wrap gap-3">
          {categories.map(({ category, items }) => (
            <Link
              key={category.slug}
              href={categoryPath(category)}
              className="arcade-press sticker rounded-[10px] bg-royal px-4 py-2.5 font-pixel text-[10px] text-white [--sh:4px]"
            >
              {pixelText(category.short)} · {items.length}
            </Link>
          ))}
        </div>
      )}
      <SealedGrid
        list={accessories}
        listId="acessorios"
        listName="Acessórios"
        empty="NENHUM ACESSORIO DISPONIVEL NO MOMENTO"
      />
      <GuideLinks guides={guidesFor("acessorios")} />
    </Container>
  );
}
