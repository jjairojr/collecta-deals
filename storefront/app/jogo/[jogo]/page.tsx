import Link from "next/link";
import { notFound } from "next/navigation";
import type { Metadata } from "next";
import Container from "@/components/ui/Container";
import SectionHead from "@/components/ui/SectionHead";
import SealedGrid from "@/components/SealedGrid";
import SingleCard from "@/components/SingleCard";
import TrackViewItemList from "@/components/analytics/TrackViewItemList";
import { loadCatalog } from "@/lib/catalog";
import { brl, pixelText } from "@/lib/format";
import { GAME_IDS, GAME_NAMES } from "@/lib/games";
import { setPath, setsOf } from "@/lib/sets";

export const revalidate = 60;

const GAME_BLURB: Record<string, string> = {
  pokemon:
    "Cartas Pokémon avulsas e produto selado lacrado: boosters, booster boxes, bundles e decks das coleções mais recentes, além de singles raras, full art e illustration rare conferidas uma a uma.",
  onepiece:
    "One Piece Card Game: leaders, alternate arts, mangas e secret rares avulsas, mais booster boxes e decks lacrados de cada OP, com condição descrita carta por carta.",
  riftbound:
    "Riftbound, o TCG de League of Legends: singles de Origins e das coleções seguintes, incluindo foils e alternate arts, além de boosters e boxes lacrados.",
  lorcana:
    "Disney Lorcana: cartas avulsas enchanted, legendary e promos, mais boosters e produtos selados das coleções lançadas.",
  gundam:
    "Gundam Card Game: singles e produto selado das primeiras coleções, conferidos uma a uma antes do envio.",
};

export function generateStaticParams() {
  return GAME_IDS.map((jogo) => ({ jogo }));
}

export async function generateMetadata({
  params,
}: {
  params: Promise<{ jogo: string }>;
}): Promise<Metadata> {
  const { jogo } = await params;
  const name = GAME_NAMES[jogo];
  if (!name) {
    return {};
  }
  const { singles, sealed, live } = await loadCatalog();
  const singlesCount = singles.filter((s) => s.game === jogo).length;
  const sealedCount = sealed.filter((s) => s.game === jogo).length;
  const parts = [
    singlesCount > 0 ? `${singlesCount} cartas avulsas` : "",
    sealedCount > 0 ? `${sealedCount} produtos selados` : "",
  ].filter(Boolean);
  return {
    title: `${name} — cartas avulsas, boosters e produto selado`,
    description: `${parts.length > 0 ? parts.join(" e ") : "Singles e selados"} de ${name} em estoque na Collecta, conferidos um a um. Pix ou cartão em até 12x, envio para todo o Brasil ou retirada na loja.`,
    alternates: { canonical: `/jogo/${jogo}` },
    robots: live && parts.length > 0 ? undefined : { index: false },
  };
}

export default async function JogoPage({
  params,
}: {
  params: Promise<{ jogo: string }>;
}) {
  const { jogo } = await params;
  const name = GAME_NAMES[jogo];
  if (!name) {
    notFound();
  }
  const { singles, sealed } = await loadCatalog();
  const sealedList = sealed.filter((s) => s.game === jogo);
  const gameSingles = singles.filter((s) => s.game === jogo);
  const top = [...gameSingles].sort((a, b) => b.price - a.price).slice(0, 8);
  const sets = setsOf(singles, jogo);
  return (
    <Container className="py-9">
      <SectionHead title={name} eyebrow="SINGLES E SELADOS" size="md" heading="h1" />
      <p className="-mt-2 mb-8 max-w-3xl text-[15px] leading-relaxed text-muted">
        {GAME_BLURB[jogo]} Pedido pelo site com Pix ou cartão, envio para todo o Brasil ou retirada na loja em Goiânia.
      </p>

      {sealedList.length > 0 && (
        <section className="mb-10">
          <SectionHead
            title={`${name} · Selado`}
            eyebrow="LACRADO E CONFERIDO"
            size="md"
            action={{ label: "VER TODOS ›", href: `/selado/${jogo}` }}
          />
          <SealedGrid list={sealedList.slice(0, 8)} listId={`jogo_${jogo}_selado`} listName={`${name} · Selado`} />
        </section>
      )}

      {top.length > 0 && (
        <section className="mb-10">
          <SectionHead
            title={`${name} · Singles em destaque`}
            eyebrow={pixelText(`${gameSingles.length} CARTAS EM ESTOQUE`)}
            size="md"
            action={{ label: "VER TODAS ›", href: `/singles/${jogo}` }}
          />
          <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4 lg:gap-5">
            <TrackViewItemList items={top} listId={`jogo_${jogo}_destaques`} listName={`${name} · Destaques`} />
            {top.map((s) => (
              <SingleCard key={s.slug} item={s} showLanguage />
            ))}
          </div>
        </section>
      )}

      {sets.length > 0 && (
        <section className="sticker rounded-[14px] bg-surface p-6 sm:p-8 [--sh:6px]">
          <h2 className="font-display text-3xl font-bold leading-none text-white">Coleções de {name}</h2>
          <p className="mt-3 text-sm text-muted">Cartas avulsas por coleção, com preço a partir de:</p>
          <div className="mt-5 flex flex-wrap gap-3">
            {sets.map((s) => (
              <Link
                key={s.code}
                href={setPath(jogo, s.code)}
                className="arcade-press sticker rounded-[10px] bg-royal px-4 py-2.5 font-pixel text-[9px] text-white [--sh:4px]"
              >
                {pixelText(s.label)} · {s.count} · {brl(s.minPrice)}
              </Link>
            ))}
          </div>
        </section>
      )}
    </Container>
  );
}
