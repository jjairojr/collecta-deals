import { pixelText } from "@/lib/format";

// Game → visual mappings, generalized to every game the real catalog may carry
// (the arcade design ships 3, but stock can include Lorcana/Gundam too).

type ArtHue = "brand" | "royal" | "soft";
type Surface = "pink" | "royal" | "soft";

const ART_HUE: Record<string, ArtHue> = {
  pokemon: "brand",
  onepiece: "royal",
  riftbound: "soft",
  lorcana: "brand",
  gundam: "royal",
};

const SURFACE: Record<string, Surface> = {
  pokemon: "pink",
  onepiece: "royal",
  riftbound: "soft",
  lorcana: "pink",
  gundam: "royal",
};

const PIXEL: Record<string, string> = {
  pokemon: "POKEMON",
  onepiece: "ONE PIECE",
  riftbound: "RIFTBOUND",
  lorcana: "LORCANA",
  gundam: "GUNDAM",
};

export const GAME_NAMES: Record<string, string> = {
  pokemon: "Pokémon",
  onepiece: "One Piece",
  riftbound: "Riftbound",
  lorcana: "Disney Lorcana",
  gundam: "Gundam Card Game",
};

export const GAME_IDS = Object.keys(GAME_NAMES);

export function isGameId(id: string): boolean {
  return id in GAME_NAMES;
}

// Card/box art fill color.
export function gameArtHue(game: string): ArtHue {
  return ART_HUE[game] ?? "royal";
}

// Cabinet / surface color for the Home game grid.
export function gameSurface(game: string): Surface {
  return SURFACE[game] ?? "royal";
}

// Accent-free pixel label for Press Start 2P (breadcrumbs, cabinets). Falls back
// to the accent-stripped display label for unknown games.
export function gamePixel(game: string, label: string): string {
  return PIXEL[game] ?? pixelText(label);
}
