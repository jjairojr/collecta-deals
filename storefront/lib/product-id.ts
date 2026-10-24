import "server-only";
import { createHash } from "node:crypto";

// Feed `brand` is the manufacturer, not the store. Falls back to the game label
// for any game the backend starts carrying before this map catches up.
const MAKERS: Record<string, string> = {
  pokemon: "The Pokémon Company",
  onepiece: "Bandai",
  riftbound: "Riot Games",
  lorcana: "Ravensburger",
  gundam: "Bandai",
};

export function maker(game: string, fallback: string): string {
  return MAKERS[game] ?? fallback;
}

// Merchant Center caps `id` at 50 characters and rejects the whole row past it.
// Slugs that already fit are used verbatim so ids stay readable and, more
// importantly, stable — the id is the key Merchant Center tracks a product by,
// and rewriting it orphans the product and resets its review. Longer slugs get
// trimmed with a digest of the full slug appended, which keeps them unique and
// deterministic across builds.
const MAX_ID = 50;
const DIGEST = 8;

export function feedID(slug: string): string {
  if (slug.length <= MAX_ID) {
    return slug;
  }
  const hash = createHash("sha1").update(slug).digest("hex").slice(0, DIGEST);
  return `${slug.slice(0, MAX_ID - DIGEST - 1)}-${hash}`;
}
