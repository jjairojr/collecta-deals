import Link from "next/link";
import ArtPlaceholder from "@/components/ui/ArtPlaceholder";
import { gamePixel } from "@/lib/games";
import type { Game } from "@/lib/types";

const surface: Record<Game["hue"], string> = {
  pink: "bg-brand",
  royal: "bg-royal",
  soft: "bg-brand-soft",
};

const art: Record<Game["hue"], "brand" | "royal" | "soft"> = {
  pink: "royal",
  royal: "soft",
  soft: "brand",
};

const nameColor: Record<Game["hue"], string> = {
  pink: "text-white",
  royal: "text-white",
  soft: "text-outline",
};

const countColor: Record<Game["hue"], string> = {
  pink: "text-brand-soft",
  royal: "text-brand-soft",
  soft: "text-on-soft",
};

// Home "arcade cabinet" for one game. Links to the browse screen pre-filtered.
export default function GameCabinet({ game }: { game: Game }) {
  return (
    <Link
      href={`/singles?jogo=${game.id}`}
      className={`arcade-press sticker sticker-5 flex flex-col overflow-hidden rounded-[18px] ${surface[game.hue]}`}
      style={{ ["--sh" as string]: "8px" }}
    >
      <div className="relative h-[186px] border-b-[5px] border-outline">
        <ArtPlaceholder
          hue={art[game.hue]}
          angle="135"
          label={gamePixel(game.id, game.name)}
        />
      </div>
      <div className="flex items-end justify-between gap-3 p-5">
        <span className={`font-display min-w-0 truncate text-[30px] font-bold leading-none ${nameColor[game.hue]}`}>
          {game.name}
        </span>
        <span className={`shrink-0 whitespace-nowrap font-pixel text-[9px] ${countColor[game.hue]}`}>
          {game.count.toLocaleString("pt-BR")} {game.count === 1 ? "CARTA" : "CARTAS"}
        </span>
      </div>
    </Link>
  );
}
