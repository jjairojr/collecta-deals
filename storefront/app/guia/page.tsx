import Link from "next/link";
import type { Metadata } from "next";
import Container from "@/components/ui/Container";
import SectionHead from "@/components/ui/SectionHead";
import { CategoryIntro } from "@/components/singles/CategoryExtras";
import JsonLd from "@/components/seo/JsonLd";
import { GUIDES, guidePath } from "@/lib/guides";
import { breadcrumbJsonLd, itemListJsonLd } from "@/lib/seo";

export const metadata: Metadata = {
  title: "Guias de TCG: selados, cartas e acessórios",
  description:
    "Guias da Collecta sobre card games: o que vem em cada produto selado de Pokémon, como saber se a carta é original, quanto vale uma carta e como começar em One Piece e Riftbound.",
  alternates: { canonical: "/guia" },
};

export default function GuiaIndexPage() {
  return (
    <>
      <JsonLd data={breadcrumbJsonLd([{ name: "Collecta", path: "/" }, { name: "Guias" }])} />
      <JsonLd data={itemListJsonLd("Guias de TCG", GUIDES.map(guidePath))} />
      <Container narrow className="py-9">
        <SectionHead title="Guias de TCG" eyebrow="COLLECTA" size="md" heading="h1" />
        <CategoryIntro>
          Respostas diretas para as dúvidas mais comuns de quem compra card games: o que vem em cada produto selado,
          como conferir se uma carta é original, como descobrir o valor de uma carta e por onde começar em cada jogo.
        </CategoryIntro>
        <div className="mt-8 grid gap-4 sm:grid-cols-2">
          {GUIDES.map((g) => (
            <Link
              key={g.slug}
              href={guidePath(g)}
              className="arcade-press sticker block rounded-[14px] bg-surface p-6 [--sh:6px]"
            >
              <div className="font-pixel text-[8px] text-brand">{g.eyebrow}</div>
              <h2 className="mt-2 font-display text-2xl font-bold leading-tight text-white">{g.title}</h2>
              <p className="mt-2 text-sm leading-relaxed text-muted">{g.description}</p>
            </Link>
          ))}
        </div>
      </Container>
    </>
  );
}
