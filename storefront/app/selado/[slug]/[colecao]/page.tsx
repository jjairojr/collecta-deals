import Link from "next/link";
import { notFound, redirect } from "next/navigation";
import type { Metadata } from "next";
import Breadcrumb from "@/components/ui/Breadcrumb";
import Container from "@/components/ui/Container";
import FaqSection from "@/components/ui/FaqSection";
import SectionHead from "@/components/ui/SectionHead";
import SealedGrid from "@/components/SealedGrid";
import { CategoryIntro } from "@/components/singles/CategoryExtras";
import JsonLd from "@/components/seo/JsonLd";
import { loadCatalog } from "@/lib/catalog";
import { brl, pixelText } from "@/lib/format";
import { GAME_NAMES, gamePixel } from "@/lib/games";
import {
  collectionBySlug,
  collectionItems,
  collectionPath,
  contentsOf,
  formatsIn,
  liveCollections,
  MIN_COLLECTION_ITEMS,
  releaseLabel,
} from "@/lib/sealed-guide";
import { collectionFaq } from "@/lib/sealed-faq";
import { breadcrumbJsonLd, itemListJsonLd } from "@/lib/seo";

export const revalidate = 60;

export async function generateStaticParams() {
  const { sealed } = await loadCatalog();
  return liveCollections(sealed).map(({ collection }) => ({ slug: collection.game, colecao: collection.slug }));
}

async function resolve(game: string, slug: string) {
  const collection = collectionBySlug(game, slug);
  if (!collection || !GAME_NAMES[game]) {
    return null;
  }
  const { sealed, live } = await loadCatalog();
  return { collection, items: collectionItems(collection, sealed), live };
}

export async function generateMetadata({
  params,
}: {
  params: Promise<{ slug: string; colecao: string }>;
}): Promise<Metadata> {
  const { slug, colecao } = await params;
  const found = await resolve(slug, colecao);
  if (!found || found.items.length < MIN_COLLECTION_ITEMS) {
    return {};
  }
  const { collection, items, live } = found;
  const from = Math.min(...items.map((s) => s.price));
  return {
    title: `${collection.name}: selados, preços e o que vem`,
    description: `${items.length} produtos selados de ${collection.short} em estoque na Collecta, a partir de ${brl(from)}: ${formatsIn(items)
      .map((f) => f.label.replace(/\s*\(.*\)$/, ""))
      .join(", ")}. Lacre original, Pix ou cartão e envio para todo o Brasil.`,
    alternates: { canonical: collectionPath(collection) },
    robots: live ? undefined : { index: false },
  };
}

export default async function SealedCollectionPage({
  params,
}: {
  params: Promise<{ slug: string; colecao: string }>;
}) {
  const { slug, colecao } = await params;
  const found = await resolve(slug, colecao);
  if (!found) {
    notFound();
  }
  const { collection, items } = found;
  if (items.length < MIN_COLLECTION_ITEMS) {
    redirect(`/selado/${slug}`);
  }
  const gameName = GAME_NAMES[slug];
  return (
    <>
      <JsonLd
        data={breadcrumbJsonLd([
          { name: "Collecta", path: "/" },
          { name: `Selados ${gameName}`, path: `/selado/${slug}` },
          { name: collection.name },
        ])}
      />
      <JsonLd data={itemListJsonLd(`Selados ${collection.name}`, items.map((s) => `/selado/${s.slug}`))} />
      <Container className="py-9">
        <Breadcrumb
          items={[
            { label: gamePixel(slug, gameName), href: `/selado/${slug}` },
            { label: "SELADOS", href: `/selado/${slug}` },
            { label: pixelText(collection.short) },
          ]}
        />
        <div className="mt-6">
          <SectionHead
            title={collection.name}
            eyebrow={collection.released ? `LANCAMENTO ${releaseLabel(collection.released)}` : "COLECAO SELADA"}
            size="md"
            heading="h1"
          />
        </div>
        <CategoryIntro>{collection.summary}</CategoryIntro>
        <CategoryIntro>
          {items.length} produtos selados de {collection.short} em estoque na Collecta, a partir de{" "}
          {brl(Math.min(...items.map((s) => s.price)))}, com lacre original e conferidos antes do envio. Pix ou cartão,
          envio para todo o Brasil ou retirada na loja em Goiânia.
          {collection.nameEN && collection.nameEN !== collection.short ? ` Em inglês, a coleção se chama ${collection.nameEN}.` : ""}
        </CategoryIntro>
        <div className="mt-8">
          <SealedGrid list={items} listId={`colecao_${collection.slug}`} listName={collection.name} />
        </div>
        <section className="sticker mt-10 overflow-x-auto rounded-[14px] bg-surface p-6 sm:p-8 [--sh:6px]">
          <h2 className="font-display text-3xl font-bold leading-none text-white">
            O que vem em cada produto de {collection.short}
          </h2>
          <table className="mt-5 w-full min-w-[520px] text-left text-[14px]">
            <thead>
              <tr className="font-pixel text-[8px] text-faint">
                <th className="pb-2 pr-4 font-normal">PRODUTO</th>
                <th className="pb-2 pr-4 font-normal">O QUE VEM</th>
                <th className="pb-2 font-normal">PRECO NO PIX</th>
              </tr>
            </thead>
            <tbody>
              {items.map((s) => (
                <tr key={s.slug} className="border-t-2 border-outline align-top text-white">
                  <td className="py-3 pr-4">
                    <Link href={`/selado/${s.slug}`} className="font-bold underline-offset-4 hover:underline">
                      {s.headline ?? s.name}
                    </Link>
                    {s.language ? <span className="block text-[12px] text-muted">{s.language}</span> : null}
                  </td>
                  <td className="py-3 pr-4 text-muted">{contentsOf(s) ?? "Produto lacrado de fábrica."}</td>
                  <td className="py-3 font-bold">{brl(s.price)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </section>
        <FaqSection items={collectionFaq(collection, items)} />
        <p className="mt-8 text-sm text-muted">
          <Link href={`/selado/${slug}`} className="text-brand-soft underline-offset-4 hover:underline">
            Ver todos os selados de {gameName}
          </Link>{" "}
          ·{" "}
          <Link href={`/singles/${slug}`} className="text-brand-soft underline-offset-4 hover:underline">
            Cartas avulsas de {gameName}
          </Link>
        </p>
      </Container>
    </>
  );
}
