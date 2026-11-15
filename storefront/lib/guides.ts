import type { FaqItem } from "@/lib/seo";
import type { Accessory, Sealed } from "@/lib/types";

type Product = Sealed | Accessory;

export interface GuideShelf {
  title: string;
  kind: "sealed" | "accessory";
  game?: string;
  test: (p: Product) => boolean;
  href: string;
  hrefLabel: string;
}

export interface Guide {
  slug: string;
  title: string;
  metaTitle: string;
  description: string;
  eyebrow: string;
  games: string[];
  answer: string;
  sections: { title: string; body: string[] }[];
  faq: FaqItem[];
  shelf?: GuideShelf;
  links: { label: string; href: string }[];
  updated: string;
}

const named = (re: RegExp) => (p: Product) => re.test(p.name);

export const GUIDES: Guide[] = [
  {
    slug: "o-que-vem-na-etb-pokemon",
    title: "O que vem na ETB de Pokémon",
    metaTitle: "O que vem na ETB de Pokémon (Coleção Treinador Avançado)",
    description:
      "O que vem dentro da Elite Trainer Box de Pokémon, vendida no Brasil como Coleção Treinador Avançado: pacotes, carta promocional, sleeves, energias e dados.",
    eyebrow: "GUIA DE SELADOS",
    games: ["pokemon"],
    answer:
      "A ETB (Elite Trainer Box), vendida em português como Coleção Treinador Avançado, traz nas coleções principais 9 pacotes de booster, 1 carta promocional, sleeves, cartas de energia, dados e marcadores, tudo numa caixa rígida que serve de organizador.",
    sections: [
      {
        title: "Para quem a ETB vale a pena",
        body: [
          "Para quem está começando a coleção ou quer presentear: além dos pacotes, a caixa já entrega o que se usa para guardar e jogar, como sleeves, energias e dados.",
          "Quem só quer abrir o maior número de pacotes pelo menor preço costuma preferir a booster box ou o combo de pacotes, que não trazem acessórios.",
        ],
      },
      {
        title: "ETB e Coleção Treinador Avançado",
        body: [
          "São o mesmo produto. Elite Trainer Box é o nome em inglês; no Brasil a Copag vende como Coleção Treinador Avançado, com as cartas em português.",
          "Edições comemorativas e especiais podem ter conteúdo diferente do padrão. Nesses casos, vale o que está descrito na página do produto.",
        ],
      },
    ],
    faq: [
      {
        q: "Quantos pacotes vêm numa ETB de Pokémon?",
        a: "Nas coleções principais, 9 pacotes de booster. Edições especiais podem trazer outra quantidade.",
      },
      {
        q: "A carta promocional da ETB é exclusiva?",
        a: "Ela vem com a marca de promo e é a forma mais direta de conseguir aquela versão da carta, por isso costuma interessar a quem coleciona.",
      },
    ],
    shelf: {
      title: "ETBs em estoque",
      kind: "sealed",
      game: "pokemon",
      test: named(/\betb\b|elite trainer|treinador avan[cç]ado/i),
      href: "/selado/pokemon/elite-trainer-box",
      hrefLabel: "VER TODAS AS ETBS ›",
    },
    links: [
      { label: "Booster box: quantos pacotes vêm", href: "/guia/booster-box-pokemon-quantos-pacotes" },
      { label: "Selados Pokémon", href: "/selado/pokemon" },
    ],
    updated: "2026-11-14",
  },
  {
    slug: "booster-box-pokemon-quantos-pacotes",
    title: "Quantos pacotes vêm numa booster box de Pokémon",
    metaTitle: "Booster box Pokémon: quantos pacotes vêm na caixa",
    description:
      "Quantos pacotes de booster vêm numa booster box de Pokémon em português, inglês, japonês e chinês, e quando ela compensa mais que ETB e blister.",
    eyebrow: "GUIA DE SELADOS",
    games: ["pokemon"],
    answer:
      "Nas coleções principais de Pokémon em português e inglês, a booster box traz 36 pacotes de booster. Caixas japonesas costumam trazer 30 pacotes nas coleções regulares, e linhas chinesas como o GEM Pack têm contagem própria, informada em cada produto.",
    sections: [
      {
        title: "O que é a booster box",
        body: [
          "É a caixa display fechada que a loja abriria para vender os pacotes um a um. Lacrada, ela é o formato com menor preço por pacote de uma coleção.",
          "No Brasil ela aparece também com o nome Caixa de Booster.",
        ],
      },
      {
        title: "Quando compensa",
        body: [
          "Para abrir em quantidade, montar a coleção de um set ou guardar lacrado. Não vem carta promocional nem acessório: é só pacote.",
          "Se a ideia é abrir poucos pacotes, o blister sai mais barato no total. Se a ideia é ganhar sleeves, dados e uma caixa organizadora, a ETB entrega isso.",
        ],
      },
    ],
    faq: [
      {
        q: "Booster box garante carta rara?",
        a: "Não há garantia de uma carta específica. A caixa segue a distribuição de raridades da coleção, e o que sai em cada pacote é sorteio.",
      },
      {
        q: "Caixa de Booster e booster box são a mesma coisa?",
        a: "Sim. Caixa de Booster é o nome em português do mesmo produto.",
      },
    ],
    shelf: {
      title: "Booster boxes em estoque",
      kind: "sealed",
      game: "pokemon",
      test: named(/caixa de booster|booster box|box display/i),
      href: "/selado/pokemon/booster-box",
      hrefLabel: "VER TODAS AS BOOSTER BOXES ›",
    },
    links: [
      { label: "O que vem na ETB", href: "/guia/o-que-vem-na-etb-pokemon" },
      { label: "Combo de pacotes: o que é", href: "/guia/combo-de-pacotes-o-que-e" },
    ],
    updated: "2026-11-14",
  },
  {
    slug: "blister-triplo-ou-quadruplo",
    title: "Blister triplo ou quádruplo: qual a diferença",
    metaTitle: "Blister triplo ou quádruplo de Pokémon: qual a diferença",
    description:
      "A diferença entre blister triplo e blister quádruplo de Pokémon: quantos pacotes vêm, a carta promocional de cada um e como escolher.",
    eyebrow: "GUIA DE SELADOS",
    games: ["pokemon"],
    answer:
      "O blister triplo de Pokémon traz 3 pacotes de booster e 1 carta promocional. O blister quádruplo traz 4 pacotes e 1 carta promocional. A promo é diferente em cada um, e o nome do Pokémon dela aparece no nome do produto.",
    sections: [
      {
        title: "Como escolher",
        body: [
          "Pelo preço por pacote, compare os dois valores divididos por 3 e por 4: a diferença costuma ser pequena.",
          "Pela carta promocional: quem coleciona escolhe pelo Pokémon da promo, já que ela só vem naquele blister.",
        ],
      },
      {
        title: "Para quem é o blister",
        body: [
          "É a entrada mais barata de uma coleção: bom para presentear, para abrir alguns pacotes sem levar uma caixa inteira e para crianças começando.",
        ],
      },
    ],
    faq: [
      {
        q: "Blister é original?",
        a: "Na Collecta, sim: os blisters em português são os distribuídos pela Copag e vêm lacrados na cartela de fábrica.",
      },
    ],
    shelf: {
      title: "Blisters em estoque",
      kind: "sealed",
      game: "pokemon",
      test: named(/blister/i),
      href: "/selado/pokemon/blister",
      hrefLabel: "VER TODOS OS BLISTERS ›",
    },
    links: [
      { label: "Combo de pacotes: o que é", href: "/guia/combo-de-pacotes-o-que-e" },
      { label: "Cartinhas Pokémon: por onde começar", href: "/cartinhas-pokemon" },
    ],
    updated: "2026-11-14",
  },
  {
    slug: "combo-de-pacotes-o-que-e",
    title: "O que é o Combo de Pacotes de Booster",
    metaTitle: "Combo de Pacotes de Booster Pokémon: o que é e quantos pacotes vêm",
    description:
      "O que é o Combo de Pacotes de Booster de Pokémon, quantos pacotes vêm na embalagem e como ele se compara à booster box e ao blister.",
    eyebrow: "GUIA DE SELADOS",
    games: ["pokemon"],
    answer:
      "O Combo de Pacotes de Booster é uma embalagem com vários boosters da mesma coleção, sem acessórios. Nas coleções principais de Pokémon em português são 18 pacotes, metade de uma booster box.",
    sections: [
      {
        title: "Combo, booster box ou blister",
        body: [
          "A booster box tem 36 pacotes e o menor preço por pacote. O combo tem 18 e custa menos no total. O blister tem 3 ou 4 pacotes e uma carta promocional.",
          "O combo serve para quem quer abrir bastante de uma coleção sem pagar o valor de uma caixa inteira.",
        ],
      },
    ],
    faq: [
      {
        q: "O combo de pacotes vem com carta promocional?",
        a: "Não. São só os pacotes de booster, numa embalagem lacrada.",
      },
    ],
    shelf: {
      title: "Combos de pacotes em estoque",
      kind: "sealed",
      game: "pokemon",
      test: named(/combo de pacotes|booster bundle/i),
      href: "/selado/pokemon/combo-de-pacotes",
      hrefLabel: "VER TODOS OS COMBOS ›",
    },
    links: [
      { label: "Booster box: quantos pacotes vêm", href: "/guia/booster-box-pokemon-quantos-pacotes" },
      { label: "Blister triplo ou quádruplo", href: "/guia/blister-triplo-ou-quadruplo" },
    ],
    updated: "2026-11-14",
  },
  {
    slug: "gem-pack-pokemon-chines",
    title: "GEM Pack: o booster chinês de Pokémon",
    metaTitle: "GEM Pack Pokémon: o que é o booster chinês e como são os volumes",
    description:
      "O que é o GEM Pack de Pokémon, a linha de boosters em chinês simplificado com pacotes de 4 cartas holográficas, e como funcionam os volumes.",
    eyebrow: "GUIA DE SELADOS",
    games: ["pokemon"],
    answer:
      "O GEM Pack é uma linha de boosters de Pokémon feita para o mercado chinês, impressa em chinês simplificado. Os pacotes são pequenos, com 4 cartas, todas holográficas, e cada volume (Vol. 2, 3, 6, 7 e assim por diante) é uma caixa diferente, com sua própria seleção de artes.",
    sections: [
      {
        title: "Por que colecionadores procuram",
        body: [
          "A linha só existe em chinês, então as versões das cartas não saem em português, inglês ou japonês.",
          "Como todo pacote é holográfico, a abertura rende mais cartas de exibição por pacote do que um booster comum.",
        ],
      },
      {
        title: "Dá para jogar com elas?",
        body: [
          "Em torneio oficial no Brasil se joga com cartas em português e inglês. GEM Pack é produto de coleção.",
        ],
      },
    ],
    faq: [
      {
        q: "Quantas cartas vêm num pacote de GEM Pack?",
        a: "4 cartas por pacote, todas holográficas.",
      },
      {
        q: "Qual a diferença entre os volumes do GEM Pack?",
        a: "Cada volume é um lançamento separado, com outra seleção de cartas e de artes. Um volume não completa o outro.",
      },
    ],
    shelf: {
      title: "GEM Packs em estoque",
      kind: "sealed",
      game: "pokemon",
      test: named(/gem pack/i),
      href: "/selado/pokemon/gem-pack",
      hrefLabel: "VER A LINHA GEM PACK ›",
    },
    links: [
      { label: "Selados Pokémon em chinês", href: "/selado/pokemon/chines" },
      { label: "Selados Pokémon em japonês", href: "/selado/pokemon/japones" },
    ],
    updated: "2026-11-14",
  },
  {
    slug: "box-pokemon-center-japao",
    title: "Box do Pokémon Center do Japão",
    metaTitle: "Box Pokémon Center Japão: Fukuoka, Hiroshima e Tohoku",
    description:
      "O que são as boxes de coleção das lojas Pokémon Center do Japão, como as de Fukuoka, Hiroshima e Tohoku, e por que chegam em pouca quantidade ao Brasil.",
    eyebrow: "GUIA DE SELADOS",
    games: ["pokemon"],
    answer:
      "As boxes do Pokémon Center são caixas de coleção das lojas oficiais Pokémon Center do Japão, cada uma dedicada a uma unidade, como Fukuoka, Hiroshima e Tohoku. O produto é em japonês e sai pouco do país, por isso chega ao Brasil em quantidade pequena.",
    sections: [
      {
        title: "O que é o Pokémon Center",
        body: [
          "É a rede de lojas oficiais da Pokémon no Japão. Várias unidades têm produtos próprios, ligados à cidade ou à região da loja.",
        ],
      },
      {
        title: "O que vem em cada box",
        body: [
          "O conteúdo muda de uma unidade para outra. Cada box tem a sua página na Collecta, com foto e preço do produto lacrado.",
        ],
      },
    ],
    faq: [
      {
        q: "As boxes do Pokémon Center são em que idioma?",
        a: "Em japonês, importadas lacradas de origem.",
      },
    ],
    shelf: {
      title: "Boxes do Pokémon Center em estoque",
      kind: "sealed",
      game: "pokemon",
      test: named(/pok[eé]mon center/i),
      href: "/selado/pokemon/pokemon-center-japao",
      hrefLabel: "VER AS BOXES ›",
    },
    links: [
      { label: "Selados Pokémon em japonês", href: "/selado/pokemon/japones" },
      { label: "Boxes de coleção Pokémon", href: "/selado/pokemon/box-de-colecao" },
    ],
    updated: "2026-11-14",
  },
  {
    slug: "pokemon-30-anos-produtos",
    title: "Pokémon 30 anos: os produtos da celebração",
    metaTitle: "Pokémon 30 anos: produtos da Celebração de 30 Anos",
    description:
      "Os produtos selados da celebração de 30 anos de Pokémon disponíveis na Collecta, em português, inglês e chinês, com preço de cada um.",
    eyebrow: "GUIA DE SELADOS",
    games: ["pokemon"],
    answer:
      "Pokémon completou 30 anos em 27 de fevereiro de 2026, data da estreia da franquia no Japão em 1996. A celebração ganhou uma linha própria de produtos de cartas, vendida no Brasil em português pela Copag, além de itens em inglês e de kits em chinês.",
    sections: [
      {
        title: "O que existe na linha",
        body: [
          "Na Collecta a linha aparece em três frentes: os produtos em português da Celebração de 30 Anos, a Coleção Treinador Avançado em inglês e os kits Card Display 30th Celebration em chinês.",
          "A lista abaixo mostra o que está em estoque agora, com o preço de cada item.",
        ],
      },
    ],
    faq: [
      {
        q: "Quando Pokémon fez 30 anos?",
        a: "Em 27 de fevereiro de 2026. Os primeiros jogos saíram no Japão em 27 de fevereiro de 1996.",
      },
    ],
    shelf: {
      title: "Produtos de 30 anos em estoque",
      kind: "sealed",
      game: "pokemon",
      test: named(/30 anos|30th celebration/i),
      href: "/selado/pokemon/celebracao-30-anos",
      hrefLabel: "VER A COLECAO ›",
    },
    links: [
      { label: "Kits de 30 anos em chinês", href: "/selado/pokemon/30-anos-china" },
      { label: "Selados Pokémon", href: "/selado/pokemon" },
    ],
    updated: "2026-11-14",
  },
  {
    slug: "como-saber-se-carta-pokemon-e-original",
    title: "Como saber se uma carta Pokémon é original",
    metaTitle: "Como saber se carta Pokémon é original: 6 sinais para conferir",
    description:
      "Como identificar carta e produto selado de Pokémon falsificados: impressão, textura, verso, corte, lacre e preço. O que conferir antes de comprar.",
    eyebrow: "GUIA DO COLECIONADOR",
    games: ["pokemon"],
    answer:
      "Carta Pokémon original tem impressão nítida, texto sem erro, verso azul no tom certo e, nas raras de arte completa, textura em relevo que se sente no dedo. Falsificação costuma ser lisa, brilhar por inteiro, ter cores lavadas e vir por um preço muito abaixo do mercado.",
    sections: [
      {
        title: "Na carta",
        body: [
          "Texto e fonte: erro de ortografia, fonte diferente ou símbolo de energia torto indicam cópia.",
          "Textura: ultra raras modernas de arte completa têm relevo fino. Se a carta deveria ter e é totalmente lisa, desconfie.",
          "Brilho: falsas costumam ter um brilho uniforme na carta toda, inclusive onde a original é fosca.",
          "Verso: compare com uma carta que você sabe ser original. O azul da falsa costuma ser mais claro ou arroxeado.",
          "Corte e espessura: a original tem bordas regulares e uma rigidez que a cópia, mais fina, não tem.",
        ],
      },
      {
        title: "No produto selado",
        body: [
          "No Brasil, os produtos em português são distribuídos pela Copag. Desconfie de caixa sem lacre de fábrica, com plástico refeito ou com cola aparente.",
          "Compre de loja com CNPJ e endereço. Preço muito abaixo do mercado é o sinal mais comum de falsificação ou de produto relacrado.",
        ],
      },
    ],
    faq: [
      {
        q: "Cartinha de Pokémon de banca é original?",
        a: "Depende do produto. Bancas vendem tanto produto oficial quanto imitações. O oficial em português traz a marca da Copag na embalagem.",
      },
      {
        q: "Preciso rasgar a carta para saber se é falsa?",
        a: "Não. Comparar impressão, textura, verso e espessura com uma carta original resolve na grande maioria dos casos, sem destruir nada.",
      },
    ],
    links: [
      { label: "Cartas Pokémon avulsas conferidas", href: "/singles/pokemon" },
      { label: "Cartinhas Pokémon: onde comprar originais", href: "/cartinhas-pokemon" },
    ],
    updated: "2026-11-14",
  },
  {
    slug: "quanto-vale-minha-carta-pokemon",
    title: "Quanto vale minha carta Pokémon",
    metaTitle: "Quanto vale minha carta Pokémon: como descobrir o preço",
    description:
      "Como descobrir quanto vale uma carta Pokémon: achar o número e a coleção, identificar a raridade, avaliar a condição e comparar com ofertas reais.",
    eyebrow: "GUIA DO COLECIONADOR",
    games: ["pokemon"],
    answer:
      "O valor de uma carta Pokémon depende de quatro coisas: qual carta é exatamente (número e coleção), a raridade, o idioma e a condição. Com o número impresso no canto inferior, como 101/084, dá para achar a carta certa e comparar com o menor preço pedido por ela hoje.",
    sections: [
      {
        title: "1. Identifique a carta",
        body: [
          "O número fica no rodapé, no formato número/total, ao lado do código da coleção. A mesma arte pode existir em várias versões, e o número é o que separa uma da outra.",
          "Número maior que o total da coleção, como 101/084, indica carta secreta, em geral a mais valiosa daquele Pokémon no set.",
        ],
      },
      {
        title: "2. Avalie a condição",
        body: [
          "NM (Near Mint) é a carta sem marca de uso visível. LP, MP e HP indicam desgaste crescente, e cada degrau derruba o preço.",
          "Olhe cantos, bordas e a superfície contra a luz. Risco no holográfico e canto branco são os defeitos que mais pesam.",
        ],
      },
      {
        title: "3. Compare com ofertas reais",
        body: [
          "O que vale é o menor preço pedido por uma carta igual, no mesmo idioma e na mesma condição, e não o anúncio mais caro.",
          "Nas páginas de carta da Collecta, quando há dados recentes, aparece o preço de mercado em português e em inglês ao lado do nosso preço.",
        ],
      },
    ],
    faq: [
      {
        q: "Carta em português vale menos que em inglês?",
        a: "Depende da carta. Algumas valem mais em inglês, outras em português. Compare sempre no mesmo idioma.",
      },
      {
        q: "Carta antiga de Pokémon vale muito?",
        a: "Só a idade não define. Valem mais as raras das primeiras coleções em boa condição. Comum antiga e gasta vale pouco.",
      },
    ],
    links: [
      { label: "Cartas Pokémon avulsas com preço", href: "/singles/pokemon" },
      { label: "Como saber se a carta é original", href: "/guia/como-saber-se-carta-pokemon-e-original" },
    ],
    updated: "2026-11-14",
  },
  {
    slug: "como-comecar-one-piece-card-game",
    title: "Como começar no One Piece Card Game",
    metaTitle: "Como começar no One Piece Card Game: deck inicial e regras básicas",
    description:
      "Como começar a jogar One Piece Card Game: o que é um starter deck, como o deck é montado e qual dos decks ST-31 a ST-36 escolher.",
    eyebrow: "GUIA PARA COMECAR",
    games: ["onepiece"],
    answer:
      "O jeito mais simples de começar no One Piece Card Game é com um starter deck: ele já vem pronto para jogar, com 1 carta de Líder, o deck de 50 cartas e as 10 cartas DON!!. Basta um deck por jogador.",
    sections: [
      {
        title: "Como é um deck",
        body: [
          "Todo deck tem 1 Líder, 50 cartas e 10 cartas DON!!, que funcionam como o recurso para pagar custos. As cartas do deck precisam ter a cor do Líder, e cabem no máximo 4 cópias de cada carta.",
        ],
      },
      {
        title: "Qual starter deck escolher",
        body: [
          "Os decks ST-31 a ST-36 saíram juntos, um para cada Líder: ST-31 Monkey.D.Luffy (vermelho), ST-32 Roronoa Zoro (verde), ST-33 Kuzan (azul), ST-34 Charlotte Katakuri (roxo), ST-35 Sabo (vermelho e preto) e ST-36 Eustass \"Captain\" Kid (amarelo).",
          "Para a primeira compra, escolha pelo personagem. Todos são montados para funcionar direto da caixa.",
        ],
      },
      {
        title: "Depois do primeiro deck",
        body: [
          "O deck melhora trocando cartas por avulsas da mesma cor. Comprar a carta avulsa sai mais barato do que abrir pacotes atrás de uma carta específica.",
        ],
      },
    ],
    faq: [
      {
        q: "Existe ETB de One Piece?",
        a: "Não com esse nome. ETB é um formato de Pokémon. No One Piece Card Game os produtos são starter decks, booster boxes e coleções especiais.",
      },
      {
        q: "Starter deck de One Piece vem com DON!!?",
        a: "Sim. O starter deck traz o Líder, as 50 cartas do deck e as 10 cartas DON!!.",
      },
    ],
    shelf: {
      title: "Starter decks em estoque",
      kind: "sealed",
      game: "onepiece",
      test: named(/\bdeck\b/i),
      href: "/selado/onepiece/starter-decks-st31-st36",
      hrefLabel: "VER OS STARTER DECKS ›",
    },
    links: [
      { label: "Cartas One Piece avulsas", href: "/singles/onepiece" },
      { label: "Selados One Piece", href: "/selado/onepiece" },
    ],
    updated: "2026-11-14",
  },
  {
    slug: "como-comecar-riftbound",
    title: "Como começar no Riftbound",
    metaTitle: "Como começar no Riftbound, o TCG de League of Legends",
    description:
      "Como começar a jogar Riftbound, o card game de League of Legends: como o deck é montado, o que comprar primeiro e onde achar cartas avulsas.",
    eyebrow: "GUIA PARA COMECAR",
    games: ["riftbound"],
    answer:
      "Riftbound é o card game de League of Legends, da Riot Games. Um deck é montado em torno de uma Lenda e do campeão escolhido, com 40 cartas no deck principal, 12 runas e 3 campos de batalha. Para começar, o caminho mais curto é um deck pronto ou as cartas avulsas de uma lista.",
    sections: [
      {
        title: "O que comprar primeiro",
        body: [
          "Quem quer jogar logo monta um deck a partir de uma lista e compra as cartas avulsas que faltam: sai mais barato do que abrir pacotes atrás de cartas específicas.",
          "Quem gosta de abrir pacote e quer formar coleção vai de booster box da coleção atual.",
        ],
      },
      {
        title: "Idioma",
        body: ["As cartas vendidas na Collecta são em inglês, o idioma usado nas mesas e nos torneios no Brasil."],
      },
    ],
    faq: [
      {
        q: "Riftbound é da Riot?",
        a: "Sim. É o card game físico de League of Legends, publicado pela Riot Games.",
      },
    ],
    shelf: {
      title: "Selados de Riftbound em estoque",
      kind: "sealed",
      game: "riftbound",
      test: () => true,
      href: "/selado/riftbound",
      hrefLabel: "VER SELADOS DE RIFTBOUND ›",
    },
    links: [
      { label: "Cartas Riftbound avulsas", href: "/singles/riftbound" },
      { label: "Riftbound na Collecta", href: "/riftbound" },
    ],
    updated: "2026-11-14",
  },
  {
    slug: "pre-venda-como-funciona",
    title: "Como funciona a pré-venda na Collecta",
    metaTitle: "Pré-venda de TCG: como funciona na Collecta",
    description:
      "Como funciona a pré-venda de produtos selados na Collecta: pagamento, data de envio, retirada na loja e o que acontece depois da compra.",
    eyebrow: "COMPRAR NA COLLECTA",
    games: ["pokemon", "onepiece", "riftbound"],
    answer:
      "Na pré-venda você compra o produto antes de ele chegar ao estoque. O pagamento é feito no pedido, por Pix ou cartão, e garante a sua unidade. O envio e a retirada na loja começam na data indicada na página do produto.",
    sections: [
      {
        title: "Como identificar",
        body: [
          "Produto em pré-venda aparece com o selo PRÉ-VENDA e com a data de início do envio na página dele.",
        ],
      },
      {
        title: "Entrega e retirada",
        body: [
          "A partir da data indicada, o pedido segue pelos Correios com código de rastreio ou fica disponível para retirada na loja em Goiânia.",
        ],
      },
    ],
    faq: [
      {
        q: "Posso parcelar a pré-venda?",
        a: "Sim, no cartão pelo Mercado Pago, direto no checkout. O número de parcelas aparece antes de você pagar.",
      },
    ],
    shelf: {
      title: "Em pré-venda agora",
      kind: "sealed",
      test: (p) => Boolean(p.shipsFrom),
      href: "/selado",
      hrefLabel: "VER TODOS OS SELADOS ›",
    },
    links: [
      { label: "Frete e prazo de entrega", href: "/frete" },
      { label: "Política de devolução", href: "/politica-de-devolucao" },
    ],
    updated: "2026-11-14",
  },
  {
    slug: "sleeves-qual-tamanho",
    title: "Qual tamanho de sleeve usar",
    metaTitle: "Sleeves para Pokémon e One Piece: qual tamanho usar",
    description:
      "Qual tamanho de sleeve usar em cartas de Pokémon, One Piece, Riftbound, Lorcana e Gundam, e para que servem inner sleeve, outer sleeve e toploader.",
    eyebrow: "GUIA DE ACESSORIOS",
    games: ["acessorios"],
    answer:
      "Cartas de Pokémon, One Piece, Riftbound, Lorcana e Gundam têm o tamanho padrão, 63 × 88 mm, e usam sleeve tamanho standard, de cerca de 66 × 91 mm. O tamanho japonês, menor, é o de Yu-Gi-Oh! e não serve para esses jogos.",
    sections: [
      {
        title: "Uma camada ou duas",
        body: [
          "Para jogar, um sleeve standard já protege. Para cartas de valor, muita gente usa duas camadas: um inner sleeve justo por dentro e o sleeve standard por fora.",
          "O outer sleeve é uma capa transparente que vai por cima do sleeve, usada para proteger o próprio sleeve ou a arte dele.",
        ],
      },
      {
        title: "Toploader",
        body: [
          "É a capa rígida para guardar e enviar carta avulsa. A carta entra já dentro de um sleeve, para não riscar ao deslizar.",
        ],
      },
    ],
    faq: [
      {
        q: "Sleeve de Pokémon serve em One Piece?",
        a: "Sim. Os dois jogos usam carta no tamanho padrão, então o mesmo sleeve standard serve.",
      },
      {
        q: "Quantos sleeves preciso para um deck?",
        a: "Um por carta do deck: 60 no Pokémon e 51 no One Piece (50 cartas e o Líder), fora as cartas DON!!, que muita gente também protege.",
      },
    ],
    shelf: {
      title: "Sleeves em estoque",
      kind: "accessory",
      test: named(/sleeve|shield/i),
      href: "/acessorios/sleeves",
      hrefLabel: "VER TODOS OS SLEEVES ›",
    },
    links: [
      { label: "Deck boxes", href: "/acessorios/deck-box" },
      { label: "Todos os acessórios", href: "/acessorios" },
    ],
    updated: "2026-11-14",
  },
];

export function guideBySlug(slug: string): Guide | undefined {
  return GUIDES.find((g) => g.slug === slug);
}

export function guidePath(guide: Guide): string {
  return `/guia/${guide.slug}`;
}

export function guidesFor(game: string): Guide[] {
  return GUIDES.filter((g) => g.games.includes(game));
}

export function shelfProducts(shelf: GuideShelf, sealed: Sealed[], accessories: Accessory[]): Product[] {
  const pool: Product[] = shelf.kind === "sealed" ? sealed : accessories;
  return pool.filter((p) => (!shelf.game || p.game === shelf.game) && shelf.test(p)).sort((a, b) => a.price - b.price);
}
