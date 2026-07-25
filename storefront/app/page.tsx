import Link from "next/link";
import Container from "@/components/ui/Container";
import SectionHead from "@/components/ui/SectionHead";
import HighScores from "@/components/home/HighScores";
import SingleCard from "@/components/SingleCard";
import SealedCard from "@/components/SealedCard";
import { loadCatalog } from "@/lib/catalog";

export const revalidate = 60;

const heroLayers =
  "radial-gradient(120% 150% at 78% 45%, rgba(246,85,155,.55) 0%, rgba(246,85,155,0) 55%)," +
  "linear-gradient(to right, rgba(255,255,255,.06) 1px, transparent 1px)";

export default async function HomePage() {
  // Real catalog; high-scores stay mock until the backend serves them.
  const { singles, sealed, games } = await loadCatalog();
  const featured = singles.slice(0, 8);
  const featuredSealed = sealed.slice(0, 4);
  return (
    <>
      {/* Hero */}
      <section className="relative overflow-hidden border-b-[6px] border-outline bg-royal">
        <div
          className="pointer-events-none absolute inset-0"
          style={{
            backgroundImage: heroLayers,
            backgroundSize: "auto, 96px 100%",
          }}
          aria-hidden
        />
        <div
          className="pointer-events-none absolute inset-x-0 bottom-0 h-[120px]"
          style={{
            backgroundImage:
              "linear-gradient(to top, rgba(11,11,12,.35), transparent)",
          }}
          aria-hidden
        />
        <Container className="relative grid items-center gap-10 py-16 lg:grid-cols-[1.15fr_.85fr] lg:py-[72px]">
          <div className="min-w-0">
            <span
              className="sticker inline-block rounded-[8px] bg-brand-soft px-3 py-2 font-pixel text-[9px] text-outline"
              style={{ ["--sh" as string]: "4px" }}
            >
              PLAYER 1 · READY
            </span>
            <h1 className="font-display hero-shadow mt-6 text-5xl font-bold leading-[.92] text-white sm:text-6xl lg:text-[76px]">
              SUA COLEÇÃO
              <br />
              COMEÇA AQUI
            </h1>
            <p className="mt-6 max-w-md text-[17px] leading-relaxed text-brand-soft sm:text-[19px]">
              Singles avulsas e produto selado de{" "}
              <strong className="font-bold text-white">Pokémon</strong>,{" "}
              <strong className="font-bold text-white">One Piece</strong> e{" "}
              <strong className="font-bold text-white">Riftbound</strong>.
              Conferido carta por carta, enviado do Brasil.
            </p>
            <div className="mt-8 flex flex-wrap gap-4">
              <Link
                href="/singles"
                className="arcade-press sticker rounded-[10px] bg-brand px-6 py-3.5 font-pixel text-[11px] text-white"
                style={{ ["--sh" as string]: "6px" }}
              >
                EXPLORAR SINGLES
              </Link>
              <Link
                href="/selado"
                className="arcade-press sticker rounded-[10px] bg-brand-soft px-6 py-3.5 font-pixel text-[11px] text-outline"
                style={{ ["--sh" as string]: "6px" }}
              >
                VER SELADOS
              </Link>
            </div>
          </div>

          <div className="relative mx-auto hidden h-[300px] w-[300px] md:block">
            <div
              className="absolute left-6 top-6 h-[300px] w-[300px] rounded-full bg-brand"
              aria-hidden
            />
            <div className="animate-bob relative h-[300px] w-[300px] overflow-hidden rounded-full border-[6px] border-outline bg-royal-light">
              {/* eslint-disable-next-line @next/next/no-img-element */}
              <img
                src="/mascot.png"
                alt="Mascote da Collecta segurando cartas"
                className="h-full w-full object-cover"
              />
            </div>
            <span
              className="sticker absolute -bottom-2 right-2 rotate-[-4deg] rounded-[8px] bg-white px-3 py-2 font-pixel text-[9px] text-outline"
              style={{ ["--sh" as string]: "4px" }}
            >
              PEGA ESSA!
            </span>
          </div>
        </Container>
      </section>

      {/* Produto selado */}
      {featuredSealed.length > 0 && (
        <Container className="py-10">
          <SectionHead
            title="Produtos selados"
            action={{ label: "VER TODOS ›", href: "/selado" }}
          />
          <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4 lg:gap-5">
            {featuredSealed.map((s) => (
              <SealedCard key={s.slug} item={s} />
            ))}
          </div>
        </Container>
      )}

      {/* Singles em destaque */}
      <Container className="py-6">
        <SectionHead
          title="Singles em destaque"
          action={{ label: "VER TODAS ›", href: "/singles" }}
        />
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4 lg:gap-5">
          {featured.map((s) => (
            <SingleCard key={s.slug} item={s} showLanguage />
          ))}
        </div>
      </Container>

      {/* High scores */}
      <Container className="py-6">
        <HighScores />
      </Container>
    </>
  );
}
