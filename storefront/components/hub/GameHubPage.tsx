import Link from "next/link";
import type { Metadata } from "next";
import ShippingFacts from "@/components/ui/ShippingFacts";
import Container from "@/components/ui/Container";
import DocSection from "@/components/ui/DocSection";
import FaqSection from "@/components/ui/FaqSection";
import SectionHead from "@/components/ui/SectionHead";
import SealedGrid from "@/components/SealedGrid";
import GuideLinks from "@/components/guide/GuideLinks";
import { SealedCollectionLinks, SealedShelfLinks } from "@/components/sealed/SealedExtras";
import { CategoryIntro, SetLinks } from "@/components/singles/CategoryExtras";
import JsonLd from "@/components/seo/JsonLd";
import { loadCatalog } from "@/lib/catalog";
import { brl, pixelText } from "@/lib/format";
import type { GameHub } from "@/lib/game-hubs";
import { GAME_NAMES } from "@/lib/games";
import { guidesFor } from "@/lib/guides";
import { liveCollections, liveShelves } from "@/lib/sealed-guide";
import { SEALED_PAYMENT, SEALED_SHIPPING } from "@/lib/sealed-faq";
import { breadcrumbJsonLd, itemListJsonLd } from "@/lib/seo";
import { setsOf } from "@/lib/sets";

async function shelfOf(hub: GameHub) {
  const { sealed, singles, live } = await loadCatalog();
  const boxes = sealed.filter((s) => s.game === hub.game);
  const cards = singles.filter((s) => s.game === hub.game);
  return { sealed, boxes, cards, sets: setsOf(singles, hub.game), live };
}

function stockLine(boxes: { price: number }[], cards: { price: number }[]): string {
  const parts = [
    boxes.length > 0
      ? `${boxes.length} produtos selados a partir de ${brl(Math.min(...boxes.map((s) => s.price)))}`
      : "",
    cards.length > 0
      ? `${cards.length} cartas avulsas a partir de ${brl(Math.min(...cards.map((s) => s.price)))}`
      : "",
  ].filter(Boolean);
  return parts.length > 0 ? `Em estoque hoje: ${parts.join(" e ")}.` : "";
}

export async function hubMetadata(hub: GameHub): Promise<Metadata> {
  const { boxes, cards, live } = await shelfOf(hub);
  return {
    title: hub.title,
    description: `${hub.lead} ${stockLine(boxes, cards)} Pix ou cartão.`.replace(/\s+/g, " "),
    alternates: { canonical: hub.path },
    robots: live && boxes.length + cards.length > 0 ? undefined : { index: false },
  };
}

export default async function GameHubPage({ hub }: { hub: GameHub }) {
  const [{ sealed, boxes, cards, sets }, { freeShippingMinBRL }] = await Promise.all([shelfOf(hub), loadCatalog()]);
  const gameName = GAME_NAMES[hub.game];
  const shown = [...boxes].sort((a, b) => Number(Boolean(b.featured)) - Number(Boolean(a.featured))).slice(0, 8);
  const chip = "arcade-press sticker rounded-[10px] bg-royal px-4 py-2.5 font-pixel text-[10px] text-white [--sh:4px]";
  return (
    <>
      <JsonLd data={breadcrumbJsonLd([{ name: "Collecta", path: "/" }, { name: hub.name }])} />
      {shown.length > 0 && (
        <JsonLd data={itemListJsonLd(`Selados ${gameName}`, shown.map((s) => `/selado/${s.slug}`))} />
      )}
      <Container className="py-9">
        <SectionHead title={hub.name} eyebrow={hub.eyebrow} size="md" heading="h1" />
        <CategoryIntro>
          {hub.lead} {stockLine(boxes, cards)} Pague por Pix ou cartão e receba em casa ou retire na loja.
        </CategoryIntro>
        <ShippingFacts freeShippingMinBRL={freeShippingMinBRL} />
        <div className="mt-5 flex flex-wrap gap-3">
          {boxes.length > 0 && (
            <Link href={`/selado/${hub.game}`} className={chip}>
              SELADOS {pixelText(gameName)} · {boxes.length}
            </Link>
          )}
          {cards.length > 0 && (
            <Link href={`/singles/${hub.game}`} className={chip}>
              CARTAS AVULSAS · {cards.length}
            </Link>
          )}
        </div>

        {shown.length > 0 && (
          <section className="mt-10">
            <SectionHead
              title={`Selados ${gameName} em estoque`}
              eyebrow="LACRADO E CONFERIDO"
              size="md"
              action={{ label: "VER TODOS ›", href: `/selado/${hub.game}` }}
            />
            <SealedGrid list={shown} listId={`hub_${hub.game}`} listName={hub.name} />
          </section>
        )}

        <SealedShelfLinks title={`Selados de ${gameName} por formato e idioma`} shelves={liveShelves(sealed, hub.game)} />
        <SealedCollectionLinks title={`Coleções seladas de ${gameName}`} collections={liveCollections(sealed, hub.game)} />
        <SetLinks game={hub.game} name={gameName} sets={sets.slice(0, 18)} />

        <div className="mt-10 grid gap-4 sm:grid-cols-2">
          {hub.about.map((s) => (
            <DocSection key={s.title} title={s.title}>
              {s.body.map((p) => (
                <p key={p}>{p}</p>
              ))}
            </DocSection>
          ))}
        </div>

        <GuideLinks guides={guidesFor(hub.game)} title={`Guias de ${gameName}`} />
        <FaqSection items={[...hub.faq, SEALED_SHIPPING, SEALED_PAYMENT]} />
      </Container>
    </>
  );
}
