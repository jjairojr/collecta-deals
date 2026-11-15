import type { FaqItem } from "@/lib/seo";

export interface GameHub {
  game: string;
  path: string;
  name: string;
  title: string;
  eyebrow: string;
  lead: string;
  about: { title: string; body: string[] }[];
  faq: FaqItem[];
}

export const GAME_HUBS: GameHub[] = [
  {
    game: "pokemon",
    path: "/pokemon-tcg",
    name: "Pokémon TCG",
    title: "Pokémon TCG: loja de cartas Pokémon, selados e avulsas",
    eyebrow: "LOJA DE POKEMON",
    lead: "A Collecta é uma loja de Pokémon TCG com loja física em Goiânia e envio para todo o Brasil.",
    about: [
      {
        title: "O que tem de Pokémon na Collecta",
        body: [
          "Produto selado das coleções atuais em português, distribuídas no Brasil pela Copag: booster box, Coleção Treinador Avançado (ETB), combo de pacotes, blister e boxes de coleção.",
          "Importados para colecionador: caixas e boxes em japonês e linhas em chinês que não saem em português, como os GEM Pack.",
          "Cartas avulsas originais, conferidas uma a uma, com a condição descrita em cada anúncio.",
        ],
      },
      {
        title: "Por onde começar",
        body: [
          "Para presentear ou abrir os primeiros pacotes, um blister ou uma ETB. Para abrir em quantidade, a booster box. Para completar o deck ou a coleção, a carta avulsa.",
        ],
      },
    ],
    faq: [
      {
        q: "Onde comprar cartas Pokémon originais?",
        a: "Em loja especializada que vende só produto oficial. A Collecta vende selados com lacre de fábrica e cartas avulsas conferidas uma a uma, com loja física em Goiânia e envio para todo o Brasil.",
      },
      {
        q: "A Collecta vende Pokémon em português?",
        a: "Sim. As coleções atuais em português, distribuídas pela Copag, além de produtos em inglês, japonês e chinês.",
      },
    ],
  },
  {
    game: "onepiece",
    path: "/one-piece-card-game",
    name: "One Piece Card Game",
    title: "One Piece Card Game: decks, promos e cartas avulsas",
    eyebrow: "LOJA DE ONE PIECE",
    lead: "A Collecta vende One Piece Card Game com loja física em Goiânia e envio para todo o Brasil.",
    about: [
      {
        title: "O que tem de One Piece na Collecta",
        body: [
          "Starter decks prontos para jogar, promos japonesas e itens de edição limitada, além de cartas avulsas: líderes, alternate arts e raras das coleções OP, EB e ST.",
          "Cada carta avulsa é conferida uma a uma e tem a condição descrita no anúncio.",
        ],
      },
      {
        title: "Para quem está começando",
        body: [
          "Um starter deck já traz o Líder, as 50 cartas e as 10 cartas DON!!. Depois, o deck melhora com cartas avulsas da mesma cor.",
        ],
      },
    ],
    faq: [
      {
        q: "Onde comprar One Piece Card Game no Brasil?",
        a: "Na Collecta, pelo site com envio para todo o Brasil ou na loja física em Goiânia. Vendemos produto selado oficial e cartas avulsas conferidas.",
      },
      {
        q: "As cartas de One Piece são em que idioma?",
        a: "As avulsas e os starter decks são em inglês, o idioma usado nas mesas no Brasil. Promos e itens Base Shop são japoneses, e o idioma aparece em cada produto.",
      },
    ],
  },
  {
    game: "riftbound",
    path: "/riftbound",
    name: "Riftbound",
    title: "Riftbound: cartas avulsas e selados do TCG de League of Legends",
    eyebrow: "LOJA DE RIFTBOUND",
    lead: "A Collecta vende Riftbound, o card game de League of Legends, com loja física em Goiânia e envio para todo o Brasil.",
    about: [
      {
        title: "O que tem de Riftbound na Collecta",
        body: [
          "Cartas avulsas em inglês das coleções lançadas, incluindo foils e alternate arts, conferidas uma a uma, e produto selado quando há estoque.",
          "Comprar a carta avulsa é o caminho mais barato para fechar um deck a partir de uma lista.",
        ],
      },
    ],
    faq: [
      {
        q: "Onde comprar cartas de Riftbound no Brasil?",
        a: "Na Collecta, pelo site com envio para todo o Brasil ou na loja física em Goiânia. As cartas avulsas ficam separadas por coleção, com o preço de cada uma.",
      },
      {
        q: "O que é Riftbound?",
        a: "É o card game físico de League of Legends, publicado pela Riot Games.",
      },
    ],
  },
];

export function hubFor(game: string): GameHub | undefined {
  return GAME_HUBS.find((h) => h.game === game);
}
