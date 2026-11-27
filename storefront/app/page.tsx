import Link from "next/link";
import type { Metadata } from "next";
import Container from "@/components/ui/Container";
import SectionHead from "@/components/ui/SectionHead";
import SingleCard from "@/components/SingleCard";
import SealedCard from "@/components/SealedCard";
import FaqSection from "@/components/ui/FaqSection";
import TrackViewItemList from "@/components/analytics/TrackViewItemList";
import HolidayBanner from "@/components/promo/HolidayBanner";
import HeroCarousel from "@/components/home/HeroCarousel";
import HomeHero from "@/components/home/HomeHero";
import { heroSlides } from "@/lib/hero";
import { loadCatalog } from "@/lib/catalog";
import { loadVitrine } from "@/lib/vitrine";
import { pixelText } from "@/lib/format";
import { SEALED_PAYMENT, SEALED_SHIPPING } from "@/lib/sealed-faq";
import type { FaqItem } from "@/lib/seo";
import { STORE, STORE_ADDRESS_LINE } from "@/lib/store";

const HOME_FAQ: FaqItem[] = [
  {
    q: "A Collecta tem loja física?",
    a: `Sim. Fica na ${STORE_ADDRESS_LINE}, aberta todos os dias das ${STORE.opens} às ${STORE.closes}. Dá para comprar na hora, retirar pedidos do site e jogar nas mesas da loja.`,
  },
  {
    q: "A Collecta tem torneios?",
    a: "Sim. Toda semana tem torneio de card games na loja em Goiânia: torneios com premiação, torneios gratuitos e encontros 4fun para jogar por diversão. A agenda sai no Instagram e pelo WhatsApp.",
  },
  {
    q: "Quais card games a Collecta vende?",
    a: "Pokémon, One Piece Card Game, Riftbound, Disney Lorcana e Gundam Card Game: produto selado, cartas avulsas e acessórios.",
  },
  {
    q: "A Collecta vende cartinhas Pokémon originais?",
    a: "Sim, só produto oficial: booster box, ETB e blister com lacre de fábrica, e cartas avulsas conferidas uma a uma.",
  },
  SEALED_SHIPPING,
  SEALED_PAYMENT,
];

export const revalidate = 60;

export async function generateMetadata(): Promise<Metadata> {
  const { live } = await loadCatalog();
  return {
    alternates: { canonical: "/" },
    robots: live ? undefined : { index: false },
  };
}

export default async function HomePage() {
  // Real catalog; high-scores stay mock until the backend serves them.
  const [catalog, vitrine] = await Promise.all([loadCatalog(), loadVitrine()]);
  const { singles, sealed, accessories, games } = catalog;
  const slides = heroSlides(catalog, vitrine.heroSlides);
  const featured = [...singles].sort((a, b) => b.price - a.price).slice(0, 8);
  const byFeatured = [...sealed].sort((a, b) => Number(Boolean(b.featured)) - Number(Boolean(a.featured)));
  const pokemonSealed = byFeatured.filter((s) => s.game === "pokemon").slice(0, 8);
  const otherSealed = byFeatured.filter((s) => s.game !== "pokemon").slice(0, 4);
  const pickedAccessories = accessories.filter((a) => a.featured);
  const featuredAccessories =
    pickedAccessories.length > 0 ? pickedAccessories : accessories.slice(0, 4);
  return (
    <>
      {slides.length > 0 ? (
        // With a live carousel the home shows only the top bar and the carousel;
        // the gift banner comes back only when there is no live slide.
        <HeroCarousel slides={slides} variant={vitrine.heroStyle ?? "rosa"} />
      ) : (
        <>
          {vitrine.banner && <HolidayBanner banner={vitrine.banner} />}
          <HomeHero />
        </>
      )}

      {/* Selados Pokémon */}
      {pokemonSealed.length > 0 && (
        <Container className="py-10">
          <SectionHead
            title="Selados Pokémon"
            eyebrow="BOOSTER BOX · BLISTER · BOX COLEÇÃO"
            action={{ label: "VER TODOS ›", href: "/selado/pokemon" }}
          />
          <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4 lg:gap-5">
            <TrackViewItemList
              items={pokemonSealed}
              listId="home_selados_pokemon"
              listName="Home · Selados Pokémon"
            />
            {pokemonSealed.map((s) => (
              <SealedCard key={s.slug} item={s} />
            ))}
          </div>
        </Container>
      )}

      {/* Outros selados */}
      {otherSealed.length > 0 && (
        <Container className="py-6">
          <SectionHead
            title={pokemonSealed.length > 0 ? "Outros selados" : "Produtos selados"}
            action={{ label: "VER TODOS ›", href: "/selado" }}
          />
          <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4 lg:gap-5">
            <TrackViewItemList
              items={otherSealed}
              listId="home_selados"
              listName="Home · Produtos selados"
            />
            {otherSealed.map((s) => (
              <SealedCard key={s.slug} item={s} />
            ))}
          </div>
        </Container>
      )}

      {/* Singles em destaque */}
      {featured.length > 0 && (
        <Container className="py-6">
          <SectionHead
            title="Singles em destaque"
            action={{ label: "VER TODAS ›", href: "/singles" }}
          />
          <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4 lg:gap-5">
            <TrackViewItemList
              items={featured}
              listId="home_destaques"
              listName="Home · Singles em destaque"
            />
            {featured.map((s) => (
              <SingleCard key={s.slug} item={s} showLanguage />
            ))}
          </div>
        </Container>
      )}

      {/* Acessórios */}
      {featuredAccessories.length > 0 && (
        <Container className="py-6">
          <SectionHead
            title="Acessórios"
            action={{ label: "VER TODOS ›", href: "/acessorios" }}
          />
          <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4 lg:gap-5">
            <TrackViewItemList
              items={featuredAccessories}
              listId="home_acessorios"
              listName="Home · Acessórios"
            />
            {featuredAccessories.map((a) => (
              <SealedCard key={a.slug} item={a} />
            ))}
          </div>
        </Container>
      )}

      <Container className="py-10">
        <section
          className="sticker rounded-[14px] bg-surface p-6 sm:p-8"
          style={{ ["--sh" as string]: "6px" }}
        >
          <h2 className="font-display text-3xl font-bold leading-none text-white">
            Loja de card games em Goiânia
          </h2>
          <p className="mt-4 max-w-3xl text-[15px] leading-relaxed text-muted">
            A Collecta é uma loja de TCG: vende{" "}
            <Link href="/selado" className="text-brand-soft underline-offset-4 hover:underline">
              produto selado
            </Link>{" "}
            (booster box, ETB, blister e decks), cartas avulsas e acessórios de
            Pokémon, One Piece, Riftbound e mais. Procurando{" "}
            <Link href="/cartinhas-pokemon" className="text-brand-soft underline-offset-4 hover:underline">
              cartinhas Pokémon
            </Link>
            ? Tem selado em português, inglês, japonês e chinês. Cada produto é
            conferido antes do envio; você paga por Pix ou cartão e retira na
            loja ou recebe em todo o Brasil.
          </p>
          <p className="mt-3 max-w-3xl text-[15px] leading-relaxed text-muted">
            Loja física na {STORE_ADDRESS_LINE}, aberta todos os dias das{" "}
            {STORE.opens} às {STORE.closes}.{" "}
            <Link href="/loja" className="text-brand-soft underline-offset-4 hover:underline">
              Veja como chegar
            </Link>
            .
          </p>
          <div className="mt-6 flex flex-wrap gap-3">
            {games.map((g) => (
              <Link
                key={g.id}
                href={`/singles/${g.id}`}
                className="arcade-press sticker rounded-[10px] bg-royal px-4 py-2.5 font-pixel text-[10px] text-white [--sh:4px]"
              >
                CARTAS {pixelText(g.name)}
              </Link>
            ))}
          </div>
        </section>
        <FaqSection title="Perguntas sobre a loja" items={HOME_FAQ} />
      </Container>
    </>
  );
}
