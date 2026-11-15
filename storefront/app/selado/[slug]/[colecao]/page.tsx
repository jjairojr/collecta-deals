import Link from "next/link";
import { notFound, redirect } from "next/navigation";
import type { Metadata } from "next";
import Breadcrumb from "@/components/ui/Breadcrumb";
import Container from "@/components/ui/Container";
import FaqSection from "@/components/ui/FaqSection";
import SectionHead from "@/components/ui/SectionHead";
import SealedGrid from "@/components/SealedGrid";
import { CategoryIntro } from "@/components/singles/CategoryExtras";
import { SealedCollectionLinks, SealedShelfLinks } from "@/components/sealed/SealedExtras";
import GuideLinks from "@/components/guide/GuideLinks";
import JsonLd from "@/components/seo/JsonLd";
import { loadCatalog } from "@/lib/catalog";
import { brl, pixelText } from "@/lib/format";
import { GAME_NAMES, gamePixel } from "@/lib/games";
import { guidesFor } from "@/lib/guides";
import {
  collectionBySlug,
  collectionItems,
  collectionPath,
  contentsOf,
  formatsIn,
  liveCollections,
  liveShelves,
  MIN_COLLECTION_ITEMS,
  releaseLabel,
  shelfBySlug,
  shelfPath,
  type SealedCollection,
  type SealedShelf,
} from "@/lib/sealed-guide";
import { collectionFaq, SEALED_ORIGINAL, SEALED_SHIPPING, shelfPriceFaq } from "@/lib/sealed-faq";
import { breadcrumbJsonLd, itemListJsonLd } from "@/lib/seo";
import type { Sealed } from "@/lib/types";

export const revalidate = 60;

export async function generateStaticParams() {
  const { sealed } = await loadCatalog();
  return [
    ...liveCollections(sealed).map(({ collection }) => ({ slug: collection.game, colecao: collection.slug })),
    ...liveShelves(sealed).map(({ shelf }) => ({ slug: shelf.game, colecao: shelf.slug })),
  ];
}

type Found =
  | { kind: "collection"; collection: SealedCollection; items: Sealed[]; sealed: Sealed[]; live: boolean }
  | { kind: "shelf"; shelf: SealedShelf; items: Sealed[]; sealed: Sealed[]; live: boolean };

async function resolve(game: string, slug: string): Promise<Found | null> {
  if (!GAME_NAMES[game]) {
    return null;
  }
  const collection = collectionBySlug(game, slug);
  const shelf = shelfBySlug(game, slug);
  if (!collection && !shelf) {
    return null;
  }
  const { sealed, live } = await loadCatalog();
  if (collection) {
    return { kind: "collection", collection, items: collectionItems(collection, sealed), sealed, live };
  }
  if (shelf) {
    return { kind: "shelf", shelf, items: sealed.filter((s) => s.game === game && shelf.test(s)), sealed, live };
  }
  return null;
}

function lowest(items: Sealed[]): number {
  return Math.min(...items.map((s) => s.price));
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
  const { items, live } = found;
  if (found.kind === "shelf") {
    return {
      title: found.shelf.title,
      description: `${items.length} produtos de ${found.shelf.name} em estoque na Collecta, a partir de ${brl(lowest(items))}. Lacre original, Pix ou cartão, envio para todo o Brasil e retirada na loja em Goiânia.`,
      alternates: { canonical: shelfPath(found.shelf) },
      robots: live ? undefined : { index: false },
    };
  }
  const { collection } = found;
  return {
    title: `${collection.name}: selados, preços e o que vem`,
    description: `${items.length} produtos selados de ${collection.short} em estoque na Collecta, a partir de ${brl(lowest(items))}: ${formatsIn(items)
      .map((f) => f.label.replace(/\s*\(.*\)$/, ""))
      .join(", ")}. Lacre original, Pix ou cartão e envio para todo o Brasil.`,
    alternates: { canonical: collectionPath(collection) },
    robots: live ? undefined : { index: false },
  };
}

function PriceTable({ title, items }: { title: string; items: Sealed[] }) {
  return (
    <section className="sticker mt-10 overflow-x-auto rounded-[14px] bg-surface p-6 sm:p-8 [--sh:6px]">
      <h2 className="font-display text-3xl font-bold leading-none text-white">{title}</h2>
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
  );
}

function BackLinks({ game, gameName }: { game: string; gameName: string }) {
  return (
    <p className="mt-8 text-sm text-muted">
      <Link href={`/selado/${game}`} className="text-brand-soft underline-offset-4 hover:underline">
        Ver todos os selados de {gameName}
      </Link>{" "}
      ·{" "}
      <Link href={`/singles/${game}`} className="text-brand-soft underline-offset-4 hover:underline">
        Cartas avulsas de {gameName}
      </Link>
    </p>
  );
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
  const { items, sealed } = found;
  if (items.length < MIN_COLLECTION_ITEMS) {
    redirect(`/selado/${slug}`);
  }
  const gameName = GAME_NAMES[slug];
  const name = found.kind === "shelf" ? found.shelf.name : found.collection.name;
  const short = found.kind === "shelf" ? found.shelf.short : found.collection.short;
  const sorted = [...items].sort((a, b) => a.price - b.price);
  return (
    <>
      <JsonLd
        data={breadcrumbJsonLd([
          { name: "Collecta", path: "/" },
          { name: `Selados ${gameName}`, path: `/selado/${slug}` },
          { name },
        ])}
      />
      <JsonLd data={itemListJsonLd(`Selados ${name}`, items.map((s) => `/selado/${s.slug}`))} />
      <Container className="py-9">
        <Breadcrumb
          items={[
            { label: gamePixel(slug, gameName), href: `/selado/${slug}` },
            { label: "SELADOS", href: `/selado/${slug}` },
            { label: pixelText(short) },
          ]}
        />
        {found.kind === "collection" ? (
          <>
            <div className="mt-6">
              <SectionHead
                title={found.collection.name}
                eyebrow={
                  found.collection.released
                    ? `LANCAMENTO ${releaseLabel(found.collection.released)}`
                    : "COLECAO SELADA"
                }
                size="md"
                heading="h1"
              />
            </div>
            <CategoryIntro>{found.collection.summary}</CategoryIntro>
            <CategoryIntro>
              {items.length} produtos selados de {short} em estoque na Collecta, a partir de {brl(lowest(items))}, com
              lacre original e conferidos antes do envio. Pix ou cartão, envio para todo o Brasil ou retirada na loja em
              Goiânia.
              {found.collection.nameEN && found.collection.nameEN !== short
                ? ` Em inglês, a coleção se chama ${found.collection.nameEN}.`
                : ""}
            </CategoryIntro>
            <div className="mt-8">
              <SealedGrid list={items} listId={`colecao_${colecao}`} listName={name} />
            </div>
            <PriceTable title={`O que vem em cada produto de ${short}`} items={items} />
            <FaqSection items={collectionFaq(found.collection, items)} />
          </>
        ) : (
          <>
            <div className="mt-6">
              <SectionHead title={name} eyebrow={`SELADOS ${gamePixel(slug, gameName)}`} size="md" heading="h1" />
            </div>
            <CategoryIntro>{found.shelf.summary}</CategoryIntro>
            <CategoryIntro>
              Hoje são {items.length} produtos em estoque na Collecta, de {brl(lowest(items))} a{" "}
              {brl(Math.max(...items.map((s) => s.price)))} no Pix, com lacre original e conferidos antes do envio. Envio
              para todo o Brasil ou retirada na loja em Goiânia.
            </CategoryIntro>
            <div className="mt-8">
              <SealedGrid list={sorted} listId={`prateleira_${colecao}`} listName={name} />
            </div>
            <PriceTable title={`Preços de ${short} hoje`} items={sorted} />
            <SealedCollectionLinks title={`Coleções de ${gameName}`} collections={liveCollections(sealed, slug)} />
            <FaqSection items={[shelfPriceFaq(found.shelf, sorted), ...found.shelf.faq, SEALED_ORIGINAL, SEALED_SHIPPING]} />
          </>
        )}
        <SealedShelfLinks
          title={`Selados de ${gameName} por formato e idioma`}
          shelves={liveShelves(sealed, slug).filter(({ shelf }) => shelf.slug !== colecao)}
        />
        <GuideLinks guides={guidesFor(slug)} />
        <BackLinks game={slug} gameName={gameName} />
      </Container>
    </>
  );
}
