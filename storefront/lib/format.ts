// Prices are integers in cents (centavos) throughout the marketplace, formatted
// pt-BR: brl(112320) → "R$ 1.123,20". Keeping money in cents avoids float drift
// when summing cart lines.
export function brl(cents: number): string {
  return (cents / 100)
    .toLocaleString("pt-BR", {
      style: "currency",
      currency: "BRL",
      minimumFractionDigits: 2,
      maximumFractionDigits: 2,
    })
    .replace(/\u00a0/g, " ");
}

// Press Start 2P has no accented glyphs — normalize any string rendered in it to
// uppercase, accent-free. pixelText("Obsidian Flames") → "OBSIDIAN FLAMES".
export function pixelText(s: string): string {
  return s
    .normalize("NFD")
    .replace(/\p{Diacritic}/gu, "")
    .toUpperCase();
}
