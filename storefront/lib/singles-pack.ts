import { GAME_NAMES } from "@/lib/games";
import type { Single } from "@/lib/types";

// The card lists ship every card of the scope to the browser so the filters work
// there. Three fields repeat on every row and can be rebuilt: `kind` is always
// "single", `gameLabel` is the game's known name, `language` is usually empty.
// Leaving them out trims the page payload without changing what the list shows.
export type PackedSingle = Omit<Single, "kind" | "gameLabel" | "language"> & {
  gameLabel?: string;
  language?: string;
};

export function packSingles(list: Single[]): PackedSingle[] {
  return list.map(({ kind: _kind, gameLabel, language, ...rest }) => ({
    ...rest,
    ...(gameLabel !== GAME_NAMES[rest.game] ? { gameLabel } : {}),
    ...(language ? { language } : {}),
  }));
}

export function unpackSingles(list: PackedSingle[]): Single[] {
  return list.map((p) => ({
    ...p,
    kind: "single",
    gameLabel: p.gameLabel ?? GAME_NAMES[p.game] ?? p.game,
    language: p.language ?? "",
  }));
}
