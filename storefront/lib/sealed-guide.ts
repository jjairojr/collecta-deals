import type { Sealed } from "@/lib/types";

export interface BoxContents {
  match: RegExp;
  text: string;
}

export interface SealedCollection {
  game: string;
  slug: string;
  name: string;
  short: string;
  nameEN?: string;
  released?: string;
  match: RegExp;
  summary: string;
  contents: BoxContents[];
}

const POKEMON_MAIN_SET: BoxContents[] = [
  { match: /caixa de booster|booster box/i, text: "36 pacotes de booster da coleção na caixa display lacrada." },
  {
    match: /\betb\b|treinador avan[cç]ado|elite trainer/i,
    text: "9 pacotes de booster, 1 carta promocional, sleeves, cartas de energia, dados e marcadores, numa caixa que serve de organizador.",
  },
  { match: /blister triplo/i, text: "3 pacotes de booster e 1 carta promocional." },
  { match: /blister qu[aá]druplo/i, text: "4 pacotes de booster e 1 carta promocional." },
];

export const SEALED_COLLECTIONS: SealedCollection[] = [
  {
    game: "pokemon",
    slug: "reinado-delta",
    name: "Megaevolução — Reinado Delta",
    short: "Reinado Delta",
    nameEN: "Mega Evolution—Delta Reign",
    released: "2026-11-06",
    match: /reinado delta/i,
    summary:
      "Sexta expansão da série Megaevolução de Pokémon Estampas Ilustradas, com lançamento mundial em 6 de novembro de 2026. A carta de destaque é Mega Rayquaza ex, de volta ao jogo pela primeira vez desde a era XY, e a coleção estreia os Estádios Lendários, formados pela combinação de duas cartas de Estádio.",
    contents: POKEMON_MAIN_SET,
  },
  {
    game: "pokemon",
    slug: "escuridao-absoluta",
    name: "Megaevolução — Escuridão Absoluta",
    short: "Escuridão Absoluta",
    nameEN: "Mega Evolution—Pitch Black",
    released: "2026-07-17",
    match: /escurid[aã]o absoluta/i,
    summary:
      "Quinta expansão da série Megaevolução de Pokémon Estampas Ilustradas, lançada mundialmente em 17 de julho de 2026, com Darkrai como Pokémon de destaque.",
    contents: POKEMON_MAIN_SET,
  },
  {
    game: "pokemon",
    slug: "caos-ascendente",
    name: "Megaevolução — Caos Ascendente",
    short: "Caos Ascendente",
    nameEN: "Mega Evolution—Chaos Rising",
    released: "2026-05-22",
    match: /caos ascendente/i,
    summary:
      "Quarta expansão da série Megaevolução de Pokémon Estampas Ilustradas, lançada em 22 de maio de 2026, com Mega Greninja ex na capa e Mega Floette ex entre os destaques.",
    contents: POKEMON_MAIN_SET,
  },
  {
    game: "pokemon",
    slug: "gem-pack",
    name: "GEM Pack (chinês simplificado)",
    short: "GEM Pack",
    match: /gem pack/i,
    summary:
      "Linha de boosters do mercado chinês, impressa em chinês simplificado, com pacotes pequenos de 4 cartas, todas holográficas. Cada volume (Vol. 2, 3, 6, 7…) é uma caixa diferente, com sua própria seleção de artes.",
    contents: [{ match: /gem pack/i, text: "Caixa lacrada com pacotes de 4 cartas, todas holográficas, em chinês simplificado." }],
  },
  {
    game: "pokemon",
    slug: "pokemon-center-japao",
    name: "Box Coleção Pokémon Center (Japão)",
    short: "Pokémon Center Japão",
    match: /pok[eé]mon center/i,
    summary:
      "Boxes de coleção das lojas Pokémon Center do Japão, cada uma dedicada a uma unidade (Fukuoka, Hiroshima, Tohoku), com produto em japonês. Saem pouco do Japão, por isso chegam ao Brasil em quantidade pequena.",
    contents: [],
  },
  {
    game: "pokemon",
    slug: "30-anos-china",
    name: "Card Display 30th Celebration (chinês)",
    short: "30 anos (China)",
    match: /30th celebration/i,
    summary:
      "Kits colecionáveis em chinês da celebração de 30 anos de Pokémon, cada um com um Pokémon de destaque, como Sylveon e Greninja.",
    contents: [],
  },
  {
    game: "pokemon",
    slug: "mega-forcas",
    name: "Box Coleção Mega Forças",
    short: "Mega Forças",
    match: /mega for[cç]as/i,
    summary:
      "Boxes de coleção em português da linha Mega Forças, uma para cada Pokémon de destaque, como Mega Darkrai e Mega Zeraora.",
    contents: [],
  },
  {
    game: "onepiece",
    slug: "starter-decks-st31-st36",
    name: "Starter Decks ST-31 a ST-36",
    short: "Starter Decks ST-31 a ST-36",
    match: /\bST-?3[1-6]\b/i,
    summary:
      "Seis decks iniciais do One Piece Card Game lançados juntos em inglês, um para cada líder: ST-31 Monkey.D.Luffy (vermelho), ST-32 Roronoa Zoro (verde), ST-33 Kuzan (azul), ST-34 Charlotte Katakuri (roxo), ST-35 Sabo (vermelho/preto) e ST-36 Eustass \"Captain\" Kid (amarelo). É o jeito mais barato de começar a jogar com um deck pronto.",
    contents: [{ match: /deck/i, text: "Deck pronto para jogar, montado em torno do líder da cor indicada no nome." }],
  },
  {
    game: "riftbound",
    slug: "vendetta",
    name: "Vendetta",
    short: "Vendetta",
    nameEN: "Vendetta",
    released: "2026-07-31",
    match: /vendetta/i,
    summary:
      "Quarta coleção do Riftbound, o TCG de League of Legends, lançada em inglês em 31 de julho de 2026.",
    contents: [{ match: /caixa de booster|booster box/i, text: "24 pacotes de booster de 14 cartas cada." }],
  },
];

export interface FormatInfo {
  label: string;
  match: RegExp;
  text: string;
}

export const FORMAT_GUIDE: FormatInfo[] = [
  {
    label: "Booster Box",
    match: /caixa de booster|booster box|box display/i,
    text: "A caixa display fechada com os pacotes de booster da coleção. É o menor preço por pacote e a escolha de quem quer abrir em quantidade ou guardar lacrado.",
  },
  {
    label: "Elite Trainer Box (Coleção Treinador Avançado)",
    match: /\betb\b|elite trainer|treinador avan[cç]ado/i,
    text: "Caixa de colecionador com pacotes de booster, carta promocional, sleeves, cartas de energia, dados e marcadores. Boa para presentear e para quem está montando a primeira coleção.",
  },
  {
    label: "Blister",
    match: /blister/i,
    text: "Pacotes de booster lacrados numa cartela, em geral com uma carta promocional. A entrada mais barata para abrir boosters ou dar de presente.",
  },
  {
    label: "Box de coleção e kits",
    match: /box cole[cç][aã]o|kit colecion/i,
    text: "Produtos especiais com boosters e cartas promocionais ou de exibição de um Pokémon ou tema específico. Procurados por colecionadores.",
  },
  {
    label: "Combo de pacotes",
    match: /combo de pacotes|booster bundle/i,
    text: "Vários pacotes de booster da mesma coleção vendidos juntos numa só embalagem, sem acessórios.",
  },
  {
    label: "Deck",
    match: /\bdeck\b/i,
    text: "Baralho pronto para jogar, montado em torno de um líder ou tema. O jeito mais simples de aprender o jogo e já sentar na mesa.",
  },
];

export function collectionOf(item: { game: string; name: string }): SealedCollection | undefined {
  return SEALED_COLLECTIONS.find((c) => c.game === item.game && c.match.test(item.name));
}

export function collectionBySlug(game: string, slug: string): SealedCollection | undefined {
  return SEALED_COLLECTIONS.find((c) => c.game === game && c.slug === slug);
}

export function contentsOf(item: { game: string; name: string }): string | undefined {
  return collectionOf(item)?.contents.find((c) => c.match.test(item.name))?.text;
}

export function collectionItems(collection: SealedCollection, sealed: Sealed[]): Sealed[] {
  return sealed.filter((s) => s.game === collection.game && collection.match.test(s.name));
}

export const MIN_COLLECTION_ITEMS = 2;

export function collectionPath(collection: SealedCollection): string {
  return `/selado/${collection.game}/${collection.slug}`;
}

export function liveCollections(sealed: Sealed[], game?: string): { collection: SealedCollection; items: Sealed[] }[] {
  return SEALED_COLLECTIONS.filter((c) => !game || c.game === game)
    .map((collection) => ({ collection, items: collectionItems(collection, sealed) }))
    .filter((c) => c.items.length >= MIN_COLLECTION_ITEMS);
}

export function formatsIn(list: { name: string }[]): FormatInfo[] {
  return FORMAT_GUIDE.filter((f) => list.some((s) => f.match.test(s.name)));
}

export function releaseLabel(date: string): string {
  const [y, m, d] = date.split("-");
  return `${d}/${m}/${y}`;
}
