import { gradedGames, scopeToGraded, type Catalog } from "@/lib/catalog";
import { brl } from "@/lib/format";
import { GAME_NAMES, isGameId } from "@/lib/games";
import { MAX_INSTALLMENTS, shipsFromLabel, SEM_JUROS } from "@/lib/payment";
import { categoryPath, liveCategories } from "@/lib/accessory-guide";
import { GAME_HUBS } from "@/lib/game-hubs";
import { GUIDES, guidePath } from "@/lib/guides";
import { collectionPath, liveCollections, liveShelves, shelfPath } from "@/lib/sealed-guide";
import { absoluteURL } from "@/lib/site";
import { freeShippingMinLabel } from "@/lib/free-shipping";
import { SHIPPING, STORE, STORE_ADDRESS_LINE, STORE_PROFILES } from "@/lib/store";
import type { Accessory, Sealed } from "@/lib/types";

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

/** Lines of /llms.txt: the store, its main pages and what is in stock. */
export function llmsLines(catalog: Catalog): string[] {
  const { sealed, accessories, games, freeShippingMinBRL } = catalog;
  const graded = [...gradedGames(catalog)].filter(isGameId);
  const handling = `${SHIPPING.handlingDays[1]} ${SHIPPING.handlingDays[1] === 1 ? "dia útil" : "dias úteis"}`;
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
    `- Pagamento: Pix ou cartão pelo Mercado Pago, no checkout do site; no cartão, em até ${MAX_INSTALLMENTS}x${SEM_JUROS}, sobre o preço no cartão`,
    `- Entrega: Correios para todo o Brasil com rastreio, frete calculado pelo CEP no checkout (a partir de uns ${brl(SHIPPING.minBRL * 100)})${freeShippingMinBRL > 0 ? `; PAC grátis a partir de ${freeShippingMinLabel(freeShippingMinBRL)} em produtos` : ""}; postagem em até ${handling} e transporte de ${SHIPPING.transitDays[0]} a ${SHIPPING.transitDays[1]} dias úteis; retirada grátis na loja em Goiânia`,
    "- Torneios: toda semana na loja, com torneios com premiação, torneios gratuitos e encontros 4fun; mesas para jogar todos os dias",
    `- Perfis: ${STORE_PROFILES.join(", ")}`,
    "",
    "## Páginas principais",
    ...GAME_HUBS.map((h) => link(h.name, h.path, h.lead)),
    link("Produtos selados", "/selado", "todos os selados em estoque, por jogo e coleção"),
    ...sealedGames.map((g) => link(`Selados ${GAME_NAMES[g]}`, `/selado/${g}`)),
    link("Cartinhas Pokémon", "/cartinhas-pokemon", "guia de onde comprar cartinhas Pokémon originais"),
    link("Cartas avulsas", "/singles"),
    ...graded.map((g) =>
      link(
        `Cartas ${GAME_NAMES[g]} graduadas`,
        `/graduadas/${g}`,
        `${scopeToGraded(catalog, g).singles.length} slabs PSA, BGS, CGC, SGC ou TAG com a nota da etiqueta e foto real`,
      ),
    ),
    ...games.filter((g) => isGameId(g.id)).map((g) => link(`Cartas avulsas de ${GAME_NAMES[g.id]}`, `/singles/${g.id}`)),
    link("Loja física", "/loja", "endereço, mapa, horário e fotos"),
    link("Torneios", "/torneios", "torneios semanais de card games em Goiânia: com premiação, gratuitos e 4fun"),
    link("Frete e entrega", "/frete"),
    link("Política de devolução", "/politica-de-devolucao"),
    link("Sobre a Collecta", "/sobre"),
    link("Versão completa deste arquivo", "/llms-full.txt", "políticas, perguntas frequentes e coleções de cartas avulsas em texto"),
    "",
    "## Coleções seladas",
    ...liveCollections(sealed).map(({ collection, items }) =>
      link(collection.name, collectionPath(collection), `${items.length} produtos. ${collection.summary}`),
    ),
    "",
    "## Selados por formato e idioma",
    ...liveShelves(sealed).map(({ shelf, items }) =>
      link(shelf.name, shelfPath(shelf), `${items.length} produtos. ${shelf.summary}`),
    ),
    "",
    "## Guias",
    ...GUIDES.map((g) => link(g.title, guidePath(g), g.answer)),
    "",
    "## Selados em estoque",
    ...sealed.map((s) => productLine(s, "selado")),
  ];
  if (accessories.length > 0) {
    lines.push(
      "",
      "## Acessórios por categoria",
      ...liveCategories(accessories).map(({ category, items }) =>
        link(category.name, categoryPath(category), `${items.length} produtos. ${category.summary}`),
      ),
      "",
      "## Acessórios em estoque",
      ...accessories.map((a) => productLine(a, "acessorio")),
    );
  }
  return lines;
}
