import { notFound } from "next/navigation";
import type { Metadata } from "next";
import SingleDetailView from "@/components/product/SingleDetailView";
import JsonLd from "@/components/seo/JsonLd";
import { loadSingleDetail } from "@/lib/catalog";
import { breadcrumbJsonLd, singleProductJsonLd } from "@/lib/seo";
import { brl } from "@/lib/format";

export const revalidate = 60;

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
    title: `${item.name} ${item.number} · ${item.set}`,
    description: `Carta ${item.name} (${item.number}) do set ${item.set}, ${item.condition}, por ${brl(item.price)}. Conferida uma a uma na Collecta — pedido pelo WhatsApp, envio para todo o Brasil.`,
    alternates: { canonical: `/carta/${slug}` },
    robots: live ? undefined : { index: false },
    openGraph: item.imageURL ? { images: [item.imageURL] } : undefined,
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
  return (
    <>
      {live && <JsonLd data={singleProductJsonLd(item)} />}
      {live && (
        <JsonLd
          data={breadcrumbJsonLd([
            { name: "Collecta", path: "/" },
            { name: item.gameLabel, path: `/singles/${item.game}` },
            { name: `${item.name} ${item.number}` },
          ])}
        />
      )}
      <SingleDetailView item={item} />
    </>
  );
}
