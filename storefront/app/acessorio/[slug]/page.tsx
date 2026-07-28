import { notFound } from "next/navigation";
import type { Metadata } from "next";
import SealedDetailView from "@/components/product/SealedDetailView";
import JsonLd from "@/components/seo/JsonLd";
import { loadAccessoryDetail } from "@/lib/catalog";
import { accessoryProductJsonLd, breadcrumbJsonLd } from "@/lib/seo";
import { brl } from "@/lib/format";

export const revalidate = 60;

export async function generateMetadata({
  params,
}: {
  params: Promise<{ slug: string }>;
}): Promise<Metadata> {
  const { slug } = await params;
  const res = await loadAccessoryDetail(slug);
  if (!res) {
    return { title: "Produto não encontrado" };
  }
  const { item, live } = res;
  return {
    title: `${item.name} · Acessórios`,
    description: `${item.name} — acessório para TCG, produto novo, por ${brl(item.price)}. Serve para qualquer jogo. Pedido pelo WhatsApp e envio para todo o Brasil na Collecta.`,
    alternates: { canonical: `/acessorio/${slug}` },
    robots: live ? undefined : { index: false },
    openGraph: item.imageURL ? { images: [item.imageURL] } : undefined,
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
