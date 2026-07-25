import { notFound } from "next/navigation";
import type { Metadata } from "next";
import Container from "@/components/ui/Container";
import SectionHead from "@/components/ui/SectionHead";
import SealedGrid from "@/components/SealedGrid";
import SealedDetailView from "@/components/product/SealedDetailView";
import JsonLd from "@/components/seo/JsonLd";
import { loadCatalog, loadSealedDetail } from "@/lib/catalog";
import { GAME_NAMES } from "@/lib/games";
import { breadcrumbJsonLd, sealedProductJsonLd } from "@/lib/seo";
import { brl } from "@/lib/format";

export const revalidate = 60;

export async function generateMetadata({
  params,
}: {
  params: Promise<{ slug: string }>;
}): Promise<Metadata> {
  const { slug } = await params;
  const gameName = GAME_NAMES[slug];
  if (gameName) {
    const { sealed, live } = await loadCatalog();
    const count = sealed.filter((s) => s.game === slug).length;
    return {
      title: `Produto selado de ${gameName}`,
      description: `${count > 0 ? `${count} produtos selados` : "Produtos selados"} de ${gameName} na Collecta: booster boxes, ETBs e mais, com lacre original e envio para todo o Brasil.`,
      alternates: { canonical: `/selado/${slug}` },
      robots: live ? undefined : { index: false },
    };
  }
  const res = await loadSealedDetail(slug);
  if (!res) {
    return { title: "Produto não encontrado" };
  }
  const { item, live } = res;
  return {
    title: `${item.name} · ${item.gameLabel}`,
    description: `${item.name} — produto selado de ${item.gameLabel}${item.language ? ` em ${item.language}` : ""}, lacre original, por ${brl(item.price)}. Pedido pelo WhatsApp e envio para todo o Brasil na Collecta.`,
    alternates: { canonical: `/selado/${slug}` },
    robots: live ? undefined : { index: false },
    openGraph: item.imageURL ? { images: [item.imageURL] } : undefined,
  };
}

export default async function SeladoSlugPage({
  params,
}: {
  params: Promise<{ slug: string }>;
}) {
  const { slug } = await params;
  const gameName = GAME_NAMES[slug];
  if (gameName) {
    const { sealed } = await loadCatalog();
    const list = sealed.filter((s) => s.game === slug);
    return (
      <Container className="py-9">
        <SectionHead
          title={`${gameName} · Selado`}
          eyebrow="LACRADO E CONFERIDO"
          size="md"
          heading="h1"
        />
        <SealedGrid list={list} />
      </Container>
    );
  }
  const res = await loadSealedDetail(slug);
  if (!res) {
    notFound();
  }
  const { item, live } = res;
  return (
    <>
      {live && <JsonLd data={sealedProductJsonLd(item)} />}
      {live && (
        <JsonLd
          data={breadcrumbJsonLd([
            { name: "Collecta", path: "/" },
            { name: item.gameLabel, path: `/selado/${item.game}` },
            { name: item.name },
          ])}
        />
      )}
      <SealedDetailView item={item} />
    </>
  );
}
