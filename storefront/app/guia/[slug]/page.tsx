import Link from "next/link";
import { notFound } from "next/navigation";
import type { Metadata } from "next";
import Breadcrumb from "@/components/ui/Breadcrumb";
import Container from "@/components/ui/Container";
import DocSection from "@/components/ui/DocSection";
import FaqSection from "@/components/ui/FaqSection";
import SectionHead from "@/components/ui/SectionHead";
import SealedGrid from "@/components/SealedGrid";
import GuideLinks from "@/components/guide/GuideLinks";
import JsonLd from "@/components/seo/JsonLd";
import { loadCatalog } from "@/lib/catalog";
import { brl, pixelText } from "@/lib/format";
import { GUIDES, guideBySlug, guidePath, shelfProducts } from "@/lib/guides";
import { articleJsonLd, breadcrumbJsonLd, type FaqItem } from "@/lib/seo";

export const revalidate = 300;
export const dynamicParams = false;

export function generateStaticParams() {
  return GUIDES.map((g) => ({ slug: g.slug }));
}

export async function generateMetadata({ params }: { params: Promise<{ slug: string }> }): Promise<Metadata> {
  const { slug } = await params;
  const guide = guideBySlug(slug);
  if (!guide) {
    return {};
  }
  return {
    title: guide.metaTitle,
    description: guide.description,
    alternates: { canonical: guidePath(guide) },
  };
}

export default async function GuidePage({ params }: { params: Promise<{ slug: string }> }) {
  const { slug } = await params;
  const guide = guideBySlug(slug);
  if (!guide) {
    notFound();
  }
  const { sealed, accessories } = await loadCatalog();
  const products = guide.shelf ? shelfProducts(guide.shelf, sealed, accessories) : [];
  const base = guide.shelf?.kind === "accessory" ? "acessorio" : "selado";
  const priceFaq: FaqItem[] =
    guide.shelf && products.length > 0
      ? [
          {
            q: `Quanto custa na Collecta hoje?`,
            a: `${products
              .slice(0, 8)
              .map((p) => `${p.headline ?? p.name}: ${brl(p.price)} no Pix`)
              .join("; ")}. Preços atualizados com o estoque do site.`,
          },
        ]
      : [];
  const others = GUIDES.filter((g) => g.slug !== guide.slug && g.games.some((id) => guide.games.includes(id)));
  return (
    <>
      <JsonLd
        data={breadcrumbJsonLd([
          { name: "Collecta", path: "/" },
          { name: "Guias", path: "/guia" },
          { name: guide.title },
        ])}
      />
      <JsonLd
        data={articleJsonLd({
          title: guide.title,
          description: guide.description,
          path: guidePath(guide),
          updated: guide.updated,
          published: guide.published,
        })}
      />
      <Container narrow className="py-9">
        <Breadcrumb items={[{ label: "GUIAS", href: "/guia" }, { label: pixelText(guide.title) }]} />
        <div className="mt-6">
          <SectionHead title={guide.title} eyebrow={guide.eyebrow} size="md" heading="h1" />
        </div>
        <p className="max-w-3xl text-[17px] leading-relaxed text-white">{guide.answer}</p>
        <div className="mt-8 grid gap-4">
          {guide.sections.map((s) => (
            <DocSection key={s.title} title={s.title}>
              {s.body.map((p) => (
                <p key={p}>{p}</p>
              ))}
            </DocSection>
          ))}
        </div>
        {guide.shelf && products.length > 0 && (
          <section className="mt-10">
            <SectionHead
              title={guide.shelf.title}
              eyebrow="NA COLLECTA"
              size="md"
              action={{ label: guide.shelf.hrefLabel, href: guide.shelf.href }}
            />
            <SealedGrid list={products.slice(0, 8)} listId={`guia_${guide.slug}`} listName={guide.title} />
            <ul className="mt-6 space-y-1 text-sm text-muted">
              {products.slice(0, 8).map((p) => (
                <li key={p.slug}>
                  <Link href={`/${base}/${p.slug}`} className="text-white underline-offset-4 hover:underline">
                    {p.headline ?? p.name}
                  </Link>
                  : {brl(p.price)} no Pix
                </li>
              ))}
            </ul>
          </section>
        )}
        <FaqSection items={[...priceFaq, ...guide.faq]} />
        <section className="sticker mt-10 rounded-[14px] bg-surface p-6 sm:p-8 [--sh:6px]">
          <h2 className="font-display text-3xl font-bold leading-none text-white">Continue por aqui</h2>
          <ul className="mt-5 grid gap-x-8 gap-y-3 text-[15px] sm:grid-cols-2">
            {guide.links.map((l) => (
              <li key={l.href}>
                <Link href={l.href} className="text-brand-soft underline-offset-4 hover:underline">
                  {l.label}
                </Link>
              </li>
            ))}
          </ul>
        </section>
        <GuideLinks guides={others.slice(0, 6)} title="Outros guias" />
        <p className="mt-8 text-xs text-faint">
          Atualizado em {guide.updated.split("-").reverse().join("/")}. Preços e estoque desta página vêm do site em
          tempo real.
        </p>
      </Container>
    </>
  );
}
