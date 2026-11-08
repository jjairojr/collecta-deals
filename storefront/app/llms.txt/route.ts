import { loadCatalog } from "@/lib/catalog";
import { brl } from "@/lib/format";
import { GAME_NAMES, isGameId } from "@/lib/games";
import { shipsFromLabel } from "@/lib/payment";
import { collectionPath, liveCollections } from "@/lib/sealed-guide";
import { absoluteURL } from "@/lib/site";
import { SHIPPING, STORE, STORE_ADDRESS_LINE, STORE_PROFILES } from "@/lib/store";
import type { Accessory, Sealed } from "@/lib/types";

export const revalidate = 3600;

function link(label: string, path: string, note?: string): string {
  return `- [${label}](${absoluteURL(path)})${note ? `: ${note}` : ""}`;
}

function productLine(p: Sealed | Accessory, base: string): string {
  const notes = [
    `${brl(p.price)} no Pix`,
    p.cardPrice !== undefined ? `${brl(p.cardPrice)} no cartão` : "",
    p.language ?? "",
    p.shipsFrom ? `pré-venda, envio a partir de ${shipsFromLabel(p.shipsFrom)}` : "",
  ].filter(Boolean);
  return link(p.headline ?? p.name, `/${base}/${p.slug}`, notes.join(" · "));
}

export async function GET(): Promise<Response> {
  const { sealed, accessories, games } = await loadCatalog();
  const sealedGames = [...new Set(sealed.map((s) => s.game))].filter(isGameId);
  const lines = [
    "# Collecta TCG",
    "",
    `> Loja de card games (TCG) em ${STORE.city}-${STORE.state} com envio para todo o Brasil: produto selado (booster box, ETB, blister, boxes de coleção e decks), cartas avulsas e acessórios de Pokémon, One Piece Card Game, Riftbound, Disney Lorcana e Gundam Card Game. Só produto oficial, conferido antes do envio.`,
    "",
    "## A loja",
    `- Endereço: ${STORE_ADDRESS_LINE}`,
    `- Horário: todos os dias, das ${STORE.opens} às ${STORE.closes}`,
    `- WhatsApp: ${STORE.phone}`,
    `- Empresa: ${STORE.legalName}, CNPJ ${STORE.cnpj}`,
    `- Pagamento: Pix ou cartão pelo Mercado Pago, no checkout do site`,
    `- Entrega: Correios para todo o Brasil com rastreio, frete grátis acima de R$ ${SHIPPING.freeAboveBRL}; retirada grátis na loja`,
    `- Perfis: ${STORE_PROFILES.join(", ")}`,
    "",
    "## Páginas principais",
    link("Produtos selados", "/selado", "todos os selados em estoque, por jogo e coleção"),
    ...sealedGames.map((g) => link(`Selados ${GAME_NAMES[g]}`, `/selado/${g}`)),
    link("Cartinhas Pokémon", "/cartinhas-pokemon", "guia de onde comprar cartinhas Pokémon originais"),
    link("Cartas avulsas", "/singles"),
    ...games.filter((g) => isGameId(g.id)).map((g) => link(`Cartas avulsas de ${GAME_NAMES[g.id]}`, `/singles/${g.id}`)),
    link("Loja física", "/loja", "endereço, mapa, horário e fotos"),
    link("Frete e entrega", "/frete"),
    link("Política de devolução", "/politica-de-devolucao"),
    link("Sobre a Collecta", "/sobre"),
    "",
    "## Coleções seladas",
    ...liveCollections(sealed).map(({ collection, items }) =>
      link(collection.name, collectionPath(collection), `${items.length} produtos. ${collection.summary}`),
    ),
    "",
    "## Selados em estoque",
    ...sealed.map((s) => productLine(s, "selado")),
  ];
  if (accessories.length > 0) {
    lines.push("", "## Acessórios em estoque", ...accessories.map((a) => productLine(a, "acessorio")));
  }
  lines.push("", `Atualizado em ${new Date().toISOString().slice(0, 10)}.`, "");
  return new Response(lines.join("\n"), {
    headers: { "Content-Type": "text/markdown; charset=utf-8" },
  });
}
