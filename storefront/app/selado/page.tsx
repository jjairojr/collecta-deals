import Container from "@/components/ui/Container";
import SectionHead from "@/components/ui/SectionHead";
import SealedCard from "@/components/SealedCard";
import { loadCatalog } from "@/lib/catalog";

export const revalidate = 60;

export const metadata = {
  title: "Produto selado — Collecta",
  description:
    "Booster boxes, ETBs e mais — lacrados e conferidos. Pokémon, One Piece e Riftbound.",
};

export default async function SeladoPage({
  searchParams,
}: {
  searchParams: Promise<{ jogo?: string }>;
}) {
  const { jogo } = await searchParams;
  const { sealed, navGames } = await loadCatalog();
  const list = jogo ? sealed.filter((s) => s.game === jogo) : sealed;
  const gameName = jogo
    ? navGames.find((g) => g.id === jogo)?.name
    : undefined;
  return (
    <Container className="py-9">
      <SectionHead
        title={gameName ? `${gameName} · Selado` : "Produto selado"}
        eyebrow="LACRADO E CONFERIDO"
        size="md"
      />
      {list.length === 0 ? (
        <p className="font-pixel text-[10px] leading-relaxed text-faint">
          NENHUM SELADO DISPONIVEL NO MOMENTO
        </p>
      ) : (
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4 lg:gap-5">
          {list.map((s) => (
            <SealedCard key={s.slug} item={s} />
          ))}
        </div>
      )}
    </Container>
  );
}
