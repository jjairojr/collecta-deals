import { notFound } from "next/navigation";
import type { Metadata } from "next";
import SingleDetailView from "@/components/product/SingleDetailView";
import { loadSingleDetail } from "@/lib/catalog";

export const revalidate = 60;

export async function generateMetadata({
  params,
}: {
  params: Promise<{ slug: string }>;
}): Promise<Metadata> {
  const { slug } = await params;
  const item = await loadSingleDetail(slug);
  if (!item) {
    return { title: "Carta não encontrada — Collecta" };
  }
  return {
    title: `${item.name} · ${item.set} — Collecta`,
    description: `${item.name} (${item.number}) — ${item.condition}. Compre no Collecta e finalize no WhatsApp.`,
  };
}

export default async function CartaPage({
  params,
}: {
  params: Promise<{ slug: string }>;
}) {
  const { slug } = await params;
  const item = await loadSingleDetail(slug);
  if (!item) {
    notFound();
  }
  return <SingleDetailView item={item} />;
}
