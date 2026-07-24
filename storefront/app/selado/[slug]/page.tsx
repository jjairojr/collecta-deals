import { notFound } from "next/navigation";
import type { Metadata } from "next";
import SealedDetailView from "@/components/product/SealedDetailView";
import { loadSealedDetail } from "@/lib/catalog";

export const revalidate = 60;

export async function generateMetadata({
  params,
}: {
  params: Promise<{ slug: string }>;
}): Promise<Metadata> {
  const { slug } = await params;
  const item = await loadSealedDetail(slug);
  if (!item) {
    return { title: "Produto não encontrado — Collecta" };
  }
  return {
    title: `${item.name} · ${item.set} — Collecta`,
    description: `${item.name} — ${item.meta}. Produto selado no Collecta.`,
  };
}

export default async function SeladoDetailPage({
  params,
}: {
  params: Promise<{ slug: string }>;
}) {
  const { slug } = await params;
  const item = await loadSealedDetail(slug);
  if (!item) {
    notFound();
  }
  return <SealedDetailView item={item} />;
}
