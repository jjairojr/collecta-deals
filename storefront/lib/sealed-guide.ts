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
  { match: /combo de pacotes/i, text: "18 pacotes de booster da coleção numa só embalagem, sem acessórios." },
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
    slug: "celebracao-30-anos",
    name: "Celebração de 30 Anos",
    short: "Celebração de 30 Anos",
    match: /celebra[cç][aã]o de 30 anos/i,
    summary:
      "Linha comemorativa dos 30 anos de Pokémon, franquia que estreou no Japão em 27 de fevereiro de 1996. No Brasil os produtos saem em português pela Copag, e a loja também traz itens da linha em inglês.",
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

export interface SealedShelf {
  game: string;
  slug: string;
  name: string;
  short: string;
  title: string;
  test: (s: Sealed) => boolean;
  summary: string;
  faq: { q: string; a: string }[];
}

const named = (re: RegExp) => (s: Sealed) => re.test(s.name);

export const SEALED_SHELVES: SealedShelf[] = [
  {
    game: "pokemon",
    slug: "booster-box",
    name: "Booster Box Pokémon",
    short: "Booster Box",
    title: "Booster Box Pokémon: caixas de booster lacradas e preços",
    test: named(/caixa de booster|booster box|box display/i),
    summary:
      "A booster box é a caixa display fechada com os pacotes de booster de uma coleção. Nas coleções principais em português e inglês são 36 pacotes; caixas japonesas e chinesas têm contagens próprias, informadas em cada produto. É o menor preço por pacote e o formato preferido de quem abre em quantidade ou guarda lacrado.",
    faq: [
      {
        q: "Quantos pacotes vêm numa booster box de Pokémon?",
        a: "Nas coleções principais em português e inglês, 36 pacotes de booster. Linhas especiais e caixas japonesas ou chinesas trazem outra quantidade, indicada na página de cada produto.",
      },
      {
        q: "Booster box ou ETB: qual comprar?",
        a: "A booster box entrega mais pacotes por real e nada além de pacotes. A ETB (Coleção Treinador Avançado) tem menos pacotes, mas vem com carta promocional, sleeves, energias, dados e uma caixa organizadora.",
      },
    ],
  },
  {
    game: "pokemon",
    slug: "elite-trainer-box",
    name: "Elite Trainer Box (ETB) Pokémon",
    short: "Elite Trainer Box",
    title: "ETB Pokémon: Elite Trainer Box e Coleção Treinador Avançado",
    test: named(/\betb\b|elite trainer|treinador avan[cç]ado/i),
    summary:
      "A Elite Trainer Box, vendida em português como Coleção Treinador Avançado, é a caixa de colecionador de cada coleção: pacotes de booster, carta promocional, sleeves, cartas de energia, dados e marcadores numa caixa que serve de organizador.",
    faq: [
      {
        q: "ETB e Coleção Treinador Avançado são a mesma coisa?",
        a: "Sim. Coleção Treinador Avançado é o nome em português do produto que em inglês se chama Elite Trainer Box, ou ETB.",
      },
      {
        q: "O que vem numa ETB de Pokémon?",
        a: "Nas coleções principais: 9 pacotes de booster, 1 carta promocional, sleeves, cartas de energia, dados e marcadores, numa caixa que serve de organizador. Edições comemorativas podem variar, e o conteúdo está na página de cada produto.",
      },
    ],
  },
  {
    game: "pokemon",
    slug: "blister",
    name: "Blister Pokémon",
    short: "Blister",
    title: "Blister Pokémon: triplo, quádruplo e duplo com carta promocional",
    test: named(/blister/i),
    summary:
      "O blister é a cartela lacrada com alguns pacotes de booster e, em geral, uma carta promocional. É a entrada mais barata para abrir boosters de uma coleção ou dar de presente.",
    faq: [
      {
        q: "Qual a diferença entre blister triplo e quádruplo?",
        a: "O triplo traz 3 pacotes de booster e 1 carta promocional; o quádruplo, 4 pacotes e 1 carta promocional. A carta promocional muda de um para o outro, e ela costuma decidir a escolha de quem coleciona.",
      },
    ],
  },
  {
    game: "pokemon",
    slug: "combo-de-pacotes",
    name: "Combo de Pacotes de Booster Pokémon",
    short: "Combo de Pacotes",
    title: "Combo de Pacotes Pokémon: vários boosters numa só embalagem",
    test: named(/combo de pacotes|booster bundle/i),
    summary:
      "O combo de pacotes junta vários boosters da mesma coleção numa só embalagem, sem acessórios. Fica entre o blister e a booster box em quantidade e em preço por pacote.",
    faq: [
      {
        q: "Quantos pacotes vêm no Combo de Pacotes de Booster?",
        a: "Nas coleções principais em português, 18 pacotes de booster, metade de uma booster box. Edições comemorativas podem trazer outra quantidade, indicada na página do produto.",
      },
    ],
  },
  {
    game: "pokemon",
    slug: "box-de-colecao",
    name: "Box de Coleção e Kits Pokémon",
    short: "Box de Coleção",
    title: "Box de Coleção Pokémon: boxes temáticos e kits colecionáveis",
    test: named(/box cole[cç][aã]o|kit colecion|cole[cç][aã]o com p[oô]ster/i),
    summary:
      "Boxes de coleção e kits são produtos especiais dedicados a um Pokémon ou a um tema, com pacotes de booster e cartas promocionais ou de exibição. São procurados por colecionadores e para presente.",
    faq: [
      {
        q: "O que vem num box de coleção de Pokémon?",
        a: "Varia de um box para outro: em geral, pacotes de booster e cartas promocionais do Pokémon de destaque. O conteúdo de cada um está na página do produto.",
      },
    ],
  },
  {
    game: "pokemon",
    slug: "japones",
    name: "Selados Pokémon em japonês",
    short: "Japonês",
    title: "Pokémon japonês selado: booster box e boxes do Pokémon Center",
    test: (s) => s.language === "Japonês",
    summary:
      "Produtos selados de Pokémon impressos em japonês, importados lacrados de origem. O Japão recebe coleções e boxes que não saem em português, como as boxes das lojas Pokémon Center.",
    faq: [
      {
        q: "Carta de Pokémon em japonês vale para jogar no Brasil?",
        a: "Não em torneio oficial: fora da Ásia, cartas em japonês não são aceitas, e no Brasil se joga com português e inglês. Produto japonês é comprado para coleção.",
      },
    ],
  },
  {
    game: "pokemon",
    slug: "chines",
    name: "Selados Pokémon em chinês",
    short: "Chinês",
    title: "Pokémon chinês selado: GEM Pack e kits em chinês simplificado",
    test: named(/\(CHN\)|gem pack|\bCSV\d/i),
    summary:
      "Produtos selados de Pokémon impressos em chinês simplificado, feitos para o mercado chinês e importados lacrados. Inclui linhas que só existem nesse idioma, como os GEM Pack.",
    faq: [
      {
        q: "O que é o GEM Pack de Pokémon?",
        a: "É uma linha de boosters do mercado chinês, em chinês simplificado, com pacotes pequenos de 4 cartas, todas holográficas. Cada volume é uma caixa diferente, com sua própria seleção de artes.",
      },
      {
        q: "Carta de Pokémon em chinês vale para jogar no Brasil?",
        a: "Não em torneio oficial: no Brasil se joga com cartas em português e inglês. Produto chinês é comprado para coleção.",
      },
    ],
  },
  {
    game: "onepiece",
    slug: "promos-japonesas",
    name: "Promos e itens japoneses de One Piece",
    short: "Promos japonesas",
    title: "One Piece Card Game: promos japonesas e itens Base Shop",
    test: named(/promo|base shop/i),
    summary:
      "Cartas promocionais e itens de edição limitada do One Piece Card Game vindos do Japão, como promos de revista e produtos da linha Base Shop. Saem em tiragem pequena e chegam ao Brasil em pouca quantidade.",
    faq: [],
  },
];

export function shelfBySlug(game: string, slug: string): SealedShelf | undefined {
  return SEALED_SHELVES.find((s) => s.game === game && s.slug === slug);
}

export function shelfPath(shelf: SealedShelf): string {
  return `/selado/${shelf.game}/${shelf.slug}`;
}

export function liveShelves(sealed: Sealed[], game?: string): { shelf: SealedShelf; items: Sealed[] }[] {
  return SEALED_SHELVES.filter((s) => !game || s.game === game)
    .map((shelf) => ({ shelf, items: sealed.filter((s) => s.game === shelf.game && shelf.test(s)) }))
    .filter((s) => s.items.length >= MIN_COLLECTION_ITEMS);
}

export function formatsIn(list: { name: string }[]): FormatInfo[] {
  return FORMAT_GUIDE.filter((f) => list.some((s) => f.match.test(s.name)));
}

export function releaseLabel(date: string): string {
  const [y, m, d] = date.split("-");
  return `${d}/${m}/${y}`;
}
