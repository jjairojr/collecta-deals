import Link from "next/link";
import { notFound, redirect } from "next/navigation";
import type { Metadata } from "next";
import Breadcrumb from "@/components/ui/Breadcrumb";
import Container from "@/components/ui/Container";
import FaqSection from "@/components/ui/FaqSection";
import SectionHead from "@/components/ui/SectionHead";
import SealedGrid from "@/components/SealedGrid";
import { CategoryIntro } from "@/components/singles/CategoryExtras";
import GuideLinks from "@/components/guide/GuideLinks";
import JsonLd from "@/components/seo/JsonLd";
import {
  categoryBySlug,
  categoryPath,
  liveCategories,
  MIN_CATEGORY_ITEMS,
  type AccessoryCategory,
} from "@/lib/accessory-guide";
import { loadCatalog } from "@/lib/catalog";
import { brl, pixelText } from "@/lib/format";
import { GUIDES } from "@/lib/guides";
import { SEALED_SHIPPING } from "@/lib/sealed-faq";
import { breadcrumbJsonLd, itemListJsonLd } from "@/lib/seo";

export const revalidate = 60;

export async function generateStaticParams() {
  const { accessories } = await loadCatalog();
  return liveCategories(accessories).map(({ category }) => ({ categoria: category.slug }));
}

async function resolve(category: AccessoryCategory) {
  const { accessories, live } = await loadCatalog();
  const items = accessories.filter((a) => category.match.test(a.name)).sort((a, b) => a.price - b.price);
  return { accessories, items, live };
}

export async function generateMetadata({ params }: { params: Promise<{ categoria: string }> }): Promise<Metadata> {
  const { categoria } = await params;
  const category = categoryBySlug(categoria);
  if (!category) {
    return {};
  }
  const { items, live } = await resolve(category);
  if (items.length < MIN_CATEGORY_ITEMS) {
    return {};
  }
  return {
    title: category.title,
    description: `${items.length} opções de ${category.short.toLowerCase()} em estoque na Collecta, a partir de ${brl(items[0].price)}. Envio para todo o Brasil e retirada na loja em Goiânia.`,
    alternates: { canonical: categoryPath(category) },
    robots: live ? undefined : { index: false },
  };
}

export default async function AccessoryCategoryPage({ params }: { params: Promise<{ categoria: string }> }) {
  const { categoria } = await params;
  const category = categoryBySlug(categoria);
  if (!category) {
    notFound();
  }
  const { accessories, items } = await resolve(category);
  if (items.length < MIN_CATEGORY_ITEMS) {
    redirect("/acessorios");
  }
  const others = liveCategories(accessories).filter((c) => c.category.slug !== category.slug);
  return (
    <>
      <JsonLd
        data={breadcrumbJsonLd([
          { name: "Collecta", path: "/" },
          { name: "Acessórios", path: "/acessorios" },
          { name: category.name },
        ])}
      />
      <JsonLd data={itemListJsonLd(category.name, items.map((a) => `/acessorio/${a.slug}`))} />
      <Container className="py-9">
        <Breadcrumb items={[{ label: "ACESSORIOS", href: "/acessorios" }, { label: pixelText(category.short) }]} />
        <div className="mt-6">
          <SectionHead title={category.name} eyebrow="PROTEJA SUA COLECAO" size="md" heading="h1" />
        </div>
        <CategoryIntro>{category.summary}</CategoryIntro>
        <CategoryIntro>
          {items.length} opções em estoque, de {brl(items[0].price)} a {brl(items[items.length - 1].price)} no Pix. Envio
          para todo o Brasil ou retirada na loja em Goiânia.
        </CategoryIntro>
        <div className="mt-8">
          <SealedGrid
            list={items}
            listId={`acessorios_${category.slug}`}
            listName={category.name}
            empty="NENHUM ACESSORIO DISPONIVEL NO MOMENTO"
          />
        </div>
        <FaqSection
          items={[
            {
              q: `Quanto custa ${category.short.toLowerCase()} na Collecta?`,
              a: `${items.map((a) => `${a.name}: ${brl(a.price)}`).join("; ")}. Preços atualizados com o estoque do site.`,
            },
            ...category.faq,
            { ...SEALED_SHIPPING, q: "Como é o envio?" },
          ]}
        />
        <GuideLinks guides={GUIDES.filter((g) => g.slug === category.guide)} />
        <p className="mt-8 flex flex-wrap gap-x-4 gap-y-2 text-sm text-muted">
          <Link href="/acessorios" className="text-brand-soft underline-offset-4 hover:underline">
            Todos os acessórios
          </Link>
          {others.map(({ category: c }) => (
            <Link key={c.slug} href={categoryPath(c)} className="text-brand-soft underline-offset-4 hover:underline">
              {c.name}
            </Link>
          ))}
        </p>
      </Container>
    </>
  );
}
