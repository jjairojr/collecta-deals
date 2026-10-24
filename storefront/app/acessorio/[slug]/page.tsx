import { notFound, permanentRedirect } from "next/navigation";
import type { Metadata } from "next";
import SealedDetailView from "@/components/product/SealedDetailView";
import JsonLd from "@/components/seo/JsonLd";
import { loadAccessoryDetail, loadCatalog } from "@/lib/catalog";
import { accessoryProductJsonLd, breadcrumbJsonLd, productOpenGraph } from "@/lib/seo";
import { brl } from "@/lib/format";

export const revalidate = 60;

export async function generateStaticParams() {
  const { accessories } = await loadCatalog();
  return accessories.map((a) => ({ slug: a.slug }));
}

export async function generateMetadata({
  params,
}: {
  params: Promise<{ slug: string }>;
}): Promise<Metadata> {
  const { slug } = await params;
  const res = await loadAccessoryDetail(slug);
  if (!res || "redirect" in res) {
    return { title: "Produto não encontrado" };
  }
  const { item, live } = res;
  return {
    title: `${item.name} · Acessórios`,
    description: item.soldOut
      ? `${item.name} está esgotado na Collecta. Último preço ${brl(item.price)}. Peça pelo WhatsApp e avisamos quando voltar ao estoque.`
      : `${item.name}, acessório para TCG, produto novo, por ${brl(item.price)}. Serve para qualquer jogo. Retire na loja da Collecta em Goiânia ou receba em todo o Brasil.`,
    alternates: { canonical: `/acessorio/${slug}` },
    robots: live ? undefined : { index: false },
    openGraph: productOpenGraph(`/acessorio/${slug}`, item.imageURL),
  };
}

export default async function AcessorioSlugPage({
  params,
}: {
  params: Promise<{ slug: string }>;
}) {
  const { slug } = await params;
  const res = await loadAccessoryDetail(slug);
  if (!res) {
    notFound();
  }
  if ("redirect" in res) {
    permanentRedirect(res.redirect);
  }
  const { item, live } = res;
  return (
    <>
      {live && <JsonLd data={accessoryProductJsonLd(item)} />}
      {live && (
        <JsonLd
          data={breadcrumbJsonLd([
            { name: "Collecta", path: "/" },
            { name: "Acessórios", path: "/acessorios" },
            { name: item.name },
          ])}
        />
      )}
      <SealedDetailView item={item} />
    </>
  );
}
