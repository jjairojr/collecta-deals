import Link from "next/link";
import { notFound } from "next/navigation";
import type { Metadata } from "next";
import SingleDetailView from "@/components/product/SingleDetailView";
import SingleCard from "@/components/SingleCard";
import TrackViewItemList from "@/components/analytics/TrackViewItemList";
import JsonLd from "@/components/seo/JsonLd";
import Container from "@/components/ui/Container";
import SectionHead from "@/components/ui/SectionHead";
import { loadCatalog, loadSingleDetail } from "@/lib/catalog";
import { breadcrumbJsonLd, productOpenGraph, singleProductJsonLd } from "@/lib/seo";
import { brl } from "@/lib/format";
import { baseCardName, setLabel, setPath } from "@/lib/sets";
import type { Single } from "@/lib/types";

export const revalidate = 60;

// Without this the route is server-rendered on demand and never lands in the
// CDN. Listing the slugs turns every card into an ISR page; cards added between
// builds still render on first request.
export async function generateStaticParams() {
  const { singles } = await loadCatalog();
  return singles.map((s) => ({ slug: s.slug }));
}

export async function generateMetadata({
  params,
}: {
  params: Promise<{ slug: string }>;
}): Promise<Metadata> {
  const { slug } = await params;
  const res = await loadSingleDetail(slug);
  if (!res) {
    return { title: "Carta não encontrada" };
  }
  const { item, live } = res;
  return {
    title: `${item.name} ${item.number}${item.variant ? ` ${item.variant}` : ""} · ${item.gameLabel} ${setLabel(item.game, item.set)}`,
    description: `Carta ${item.name} (${item.number}) de ${item.gameLabel}, coleção ${setLabel(item.game, item.set)}${item.variant ? `, versão ${item.variant}` : ""}, condição ${item.condition}, por ${brl(item.price)}. Original, conferida uma a uma na Collecta. Pix ou cartão em até 12x, envio para todo o Brasil.`,
    alternates: { canonical: `/carta/${slug}` },
    robots: live ? undefined : { index: false },
    openGraph: productOpenGraph(`/carta/${slug}`, item.imageURL),
  };
}

export default async function CartaPage({
  params,
}: {
  params: Promise<{ slug: string }>;
}) {
  const { slug } = await params;
  const res = await loadSingleDetail(slug);
  if (!res) {
    notFound();
  }
  const { item, live } = res;
  const { singles } = await loadCatalog();
  const others = singles.filter((s) => s.game === item.game && s.slug !== item.slug);
  const base = baseCardName(item.name);
  const versions = others.filter((s) => s.number === item.number);
  const sameName = others
    .filter((s) => s.number !== item.number && baseCardName(s.name) === base)
    .slice(0, 8);
  const sameSet = others
    .filter((s) => s.set === item.set && s.number !== item.number && baseCardName(s.name) !== base)
    .sort((a, b) => b.price - a.price)
    .slice(0, 8);
  const collection = setLabel(item.game, item.set);
  return (
    <>
      {live && <JsonLd data={singleProductJsonLd(item)} />}
      {live && (
        <JsonLd
          data={breadcrumbJsonLd([
            { name: "Collecta", path: "/" },
            { name: item.gameLabel, path: `/singles/${item.game}` },
            { name: collection, path: setPath(item.game, item.set) },
            { name: `${item.name} ${item.number}` },
          ])}
        />
      )}
      <SingleDetailView item={item} />
      <Container className="pb-6">
        <RelatedGrid
          title={`Outras versões de ${base} ${item.number}`}
          eyebrow="MESMA CARTA, OUTRA VARIANTE OU CONDICAO"
          items={versions}
          listId="carta_versoes"
        />
        <RelatedGrid
          title={`Mais cartas de ${base}`}
          eyebrow="OUTRAS CARTAS DO MESMO PERSONAGEM"
          items={sameName}
          listId="carta_mesmo_nome"
        />
        <RelatedGrid
          title={`Mais cartas de ${collection}`}
          eyebrow={`COLECAO ${item.set}`}
          items={sameSet}
          listId="carta_mesmo_set"
          action={{ label: "VER A COLECAO ›", href: setPath(item.game, item.set) }}
        />
        <section className="sticker rounded-[14px] bg-surface p-6 sm:p-8 [--sh:6px]">
          <h2 className="font-display text-3xl font-bold leading-none text-white">Perguntas frequentes</h2>
          <dl className="mt-5 grid gap-5 text-sm leading-relaxed text-[#c9c9d1] sm:grid-cols-2">
            <div>
              <dt className="font-bold text-white">A carta é original?</dt>
              <dd className="mt-1">
                Sim. Só vendemos cartas oficiais de {item.gameLabel}. Cada uma é conferida à mão antes de entrar no
                estoque e de novo antes do envio.
              </dd>
            </div>
            <div>
              <dt className="font-bold text-white">O que significa condição {item.condition}?</dt>
              <dd className="mt-1">
                {item.condition === "NM"
                  ? "Near Mint: carta sem marcas de uso visíveis, bordas e superfície íntegras, como saiu do booster."
                  : "A condição descreve o estado de conservação da carta (NM, LP, MP, HP). Qualquer marca relevante aparece na foto e na descrição."}
              </dd>
            </div>
            <div>
              <dt className="font-bold text-white">Como é o envio?</dt>
              <dd className="mt-1">
                Carta em sleeve e toploader, embalagem rígida e envio pelos Correios para todo o Brasil com código de
                rastreio. Em Goiânia você pode retirar na loja sem frete.
              </dd>
            </div>
            <div>
              <dt className="font-bold text-white">Como pagar?</dt>
              <dd className="mt-1">
                Pix ou cartão em até 12x pelo Mercado Pago, direto no checkout do site. Também dá para fechar pelo
                WhatsApp. Devolução em até 7 dias conforme a{" "}
                <Link href="/politica-de-devolucao" className="text-brand-soft underline-offset-4 hover:underline">
                  política de trocas
                </Link>
                .
              </dd>
            </div>
          </dl>
        </section>
      </Container>
    </>
  );
}

function RelatedGrid({
  title,
  eyebrow,
  items,
  listId,
  action,
}: {
  title: string;
  eyebrow: string;
  items: Single[];
  listId: string;
  action?: { label: string; href: string };
}) {
  if (items.length === 0) {
    return null;
  }
  return (
    <section className="mb-10">
      <SectionHead title={title} eyebrow={eyebrow} size="md" action={action} />
      <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4 lg:gap-5">
        <TrackViewItemList items={items} listId={listId} listName={title} />
        {items.map((s) => (
          <SingleCard key={`${listId}-${s.slug}`} item={s} showLanguage />
        ))}
      </div>
    </section>
  );
}
