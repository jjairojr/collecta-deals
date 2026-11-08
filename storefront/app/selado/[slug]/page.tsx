import { notFound, permanentRedirect } from "next/navigation";
import type { Metadata } from "next";
import Container from "@/components/ui/Container";
import SectionHead from "@/components/ui/SectionHead";
import SealedGrid from "@/components/SealedGrid";
import KindToggle from "@/components/ui/KindToggle";
import FaqSection from "@/components/ui/FaqSection";
import { CategoryIntro } from "@/components/singles/CategoryExtras";
import SealedDetailView from "@/components/product/SealedDetailView";
import { SealedCollectionAbout, SealedCollectionLinks, SealedFormatGuide } from "@/components/sealed/SealedExtras";
import JsonLd from "@/components/seo/JsonLd";
import { loadCatalog, loadSealedDetail } from "@/lib/catalog";
import { GAME_NAMES } from "@/lib/games";
import { breadcrumbJsonLd, itemListJsonLd, productOpenGraph, sealedProductJsonLd } from "@/lib/seo";
import { brl } from "@/lib/format";
import {
  collectionItems,
  collectionOf,
  collectionPath,
  liveCollections,
  MIN_COLLECTION_ITEMS,
} from "@/lib/sealed-guide";
import { productFaq, SEALED_GENERAL } from "@/lib/sealed-faq";

export const revalidate = 60;

// This segment serves both the per-game listings (/selado/pokemon) and the
// product pages (/selado/<slug>), so both go into the static set.
export async function generateStaticParams() {
  const { sealed } = await loadCatalog();
  const gameIDs = [...new Set(sealed.map((s) => s.game))];
  return [...gameIDs, ...sealed.map((s) => s.slug)].map((slug) => ({ slug }));
}

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
      title: `Selados ${gameName} — booster box, ETB e boosters lacrados`,
      description: `${count > 0 ? `${count} produtos selados` : "Produtos selados"} de ${gameName} em estoque na Collecta: booster box, ETB, blister e decks com lacre original. Pix ou cartão, envio para todo o Brasil e retirada na loja em Goiânia.`,
      alternates: { canonical: `/selado/${slug}` },
      robots: live ? undefined : { index: false },
    };
  }
  const res = await loadSealedDetail(slug);
  if (!res || "redirect" in res) {
    return { title: "Produto não encontrado" };
  }
  const { item, live } = res;
  return {
    title: `${item.headline ?? item.name} · ${item.gameLabel}`,
    description: item.soldOut
      ? `${item.name} está esgotado na Collecta. Último preço ${brl(item.price)}. Peça pelo WhatsApp e avisamos quando voltar ao estoque.`
      : `Comprar ${item.name} lacrado na Collecta: ${item.format ?? "produto selado"} de ${item.gameLabel}${item.language ? ` em ${item.language}` : ""}, com lacre original, por ${brl(item.price)}. Retire na loja em Goiânia ou receba em todo o Brasil.`,
    alternates: { canonical: `/selado/${slug}` },
    robots: live ? undefined : { index: false },
    openGraph: productOpenGraph(`/selado/${slug}`, item.imageURL),
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
      <>
        <JsonLd
          data={breadcrumbJsonLd([
            { name: "Collecta", path: "/" },
            { name: "Selados", path: "/selado" },
            { name: `Selados ${gameName}` },
          ])}
        />
        {list.length > 0 && (
          <JsonLd data={itemListJsonLd(`Selados ${gameName}`, list.map((s) => `/selado/${s.slug}`))} />
        )}
        <Container className="py-9">
          <SectionHead
            title={`Selados ${gameName}`}
            eyebrow="LACRADO E CONFERIDO"
            size="md"
            heading="h1"
          />
          <KindToggle singlesHref={`/singles/${slug}`} />
          <CategoryIntro>
            {list.length > 0 ? `${list.length} produtos selados de ${gameName} em estoque: ` : `Produtos selados de ${gameName}: `}
            booster boxes, ETBs, bundles, blisters e decks com lacre original de fábrica, conferidos antes do envio. Pix ou
            cartão em até 12x, envio para todo o Brasil ou retirada na loja em Goiânia.
          </CategoryIntro>
          <SealedGrid
            list={list}
            listId={`selado_${slug}`}
            listName={`${gameName} · Selado`}
          />
          <SealedCollectionLinks title={`Coleções de ${gameName}`} collections={liveCollections(sealed, slug)} />
          <SealedFormatGuide title={`Qual selado de ${gameName} comprar?`} list={list} />
          <FaqSection items={SEALED_GENERAL} />
        </Container>
      </>
    );
  }
  const res = await loadSealedDetail(slug);
  if (!res) {
    notFound();
  }
  if ("redirect" in res) {
    permanentRedirect(res.redirect);
  }
  const { item, live } = res;
  const { sealed } = await loadCatalog();
  const collection = collectionOf(item);
  const siblings = collection ? collectionItems(collection, sealed) : [];
  const others = siblings.filter((s) => s.slug !== item.slug);
  const hasCollectionPage = collection !== undefined && siblings.length >= MIN_COLLECTION_ITEMS;
  return (
    <>
      {live && <JsonLd data={sealedProductJsonLd(item)} />}
      {live && (
        <JsonLd
          data={breadcrumbJsonLd([
            { name: "Collecta", path: "/" },
            { name: `Selados ${item.gameLabel}`, path: `/selado/${item.game}` },
            ...(collection && hasCollectionPage ? [{ name: collection.short, path: collectionPath(collection) }] : []),
            { name: item.name },
          ])}
        />
      )}
      <SealedDetailView item={item} />
      <Container className="pb-6">
        {collection && <SealedCollectionAbout collection={collection} item={item} siblings={siblings.length} />}
        {others.length > 0 && collection && (
          <section className="mt-10">
            <SectionHead
              title={`Mais produtos de ${collection.short}`}
              eyebrow="MESMA COLECAO"
              size="md"
              action={hasCollectionPage ? { label: "VER A COLECAO ›", href: collectionPath(collection) } : undefined}
            />
            <SealedGrid list={others} listId="selado_mesma_colecao" listName={`Mais produtos de ${collection.short}`} />
          </section>
        )}
        <FaqSection items={productFaq(item, collection)} schema={live} />
      </Container>
    </>
  );
}
