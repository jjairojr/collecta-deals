import type { Metadata } from "next";
import Container from "@/components/ui/Container";
import SectionHead from "@/components/ui/SectionHead";
import SealedGrid from "@/components/SealedGrid";
import { loadCatalog } from "@/lib/catalog";

export const revalidate = 60;

export async function generateMetadata(): Promise<Metadata> {
  const { live } = await loadCatalog();
  return {
    title: "Acessórios de TCG",
    description:
      "Sleeves, deckboxes, playmats e mais acessórios para o seu TCG, com envio para todo o Brasil.",
    alternates: { canonical: "/acessorios" },
    robots: live ? undefined : { index: false },
  };
}

export default async function AcessoriosPage() {
  const { accessories } = await loadCatalog();
  return (
    <Container className="py-9">
      <SectionHead
        title="Acessórios"
        eyebrow="PROTEJA SUA COLECAO"
        size="md"
        heading="h1"
      />
      <SealedGrid
        list={accessories}
        listId="acessorios"
        listName="Acessórios"
        empty="NENHUM ACESSORIO DISPONIVEL NO MOMENTO"
      />
    </Container>
  );
}
