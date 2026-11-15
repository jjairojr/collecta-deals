import type { FaqItem } from "@/lib/seo";
import type { Accessory } from "@/lib/types";

export interface AccessoryCategory {
  slug: string;
  name: string;
  short: string;
  title: string;
  match: RegExp;
  summary: string;
  faq: FaqItem[];
  guide?: string;
}

export const ACCESSORY_CATEGORIES: AccessoryCategory[] = [
  {
    slug: "sleeves",
    name: "Sleeves para TCG",
    short: "Sleeves",
    title: "Sleeves para Pokémon, One Piece e outros TCGs",
    match: /sleeve|shield/i,
    summary:
      "Sleeves tamanho standard para cartas de 63 × 88 mm, o padrão de Pokémon, One Piece, Riftbound, Lorcana e Gundam. Tem opção fosca colorida para deck, transparente básica para guardar coleção e outer sleeve para a segunda camada.",
    faq: [
      {
        q: "Qual sleeve serve em carta de Pokémon?",
        a: "O tamanho standard, de cerca de 66 × 91 mm. Ele serve também em One Piece, Riftbound, Lorcana e Gundam, que usam carta do mesmo tamanho.",
      },
      {
        q: "Sleeve fosco ou transparente?",
        a: "O fosco com verso colorido é o usado em deck, porque embaralha melhor e esconde o verso das cartas. O transparente é para guardar a coleção e ver a carta dos dois lados.",
      },
    ],
    guide: "sleeves-qual-tamanho",
  },
  {
    slug: "deck-box",
    name: "Deck Box",
    short: "Deck Box",
    title: "Deck box para TCG: caixas para guardar o deck",
    match: /central box|deck ?box/i,
    summary:
      "Deck box é a caixa rígida para guardar e transportar o deck já com sleeves, sem amassar as cartas na mochila.",
    faq: [],
  },
  {
    slug: "slab-guard",
    name: "Slab Guard para cartas graduadas",
    short: "Slab Guard",
    title: "Slab guard: capa de silicone para carta graduada",
    match: /slab/i,
    summary:
      "O slab guard é a capa de silicone que envolve o estojo de uma carta graduada, protegendo o acrílico contra risco e queda.",
    faq: [
      {
        q: "O que é slab?",
        a: "É o estojo de acrílico lacrado em que a carta volta de uma empresa de graduação, com a nota impressa na etiqueta.",
      },
    ],
  },
  {
    slug: "album-e-toploader",
    name: "Álbuns e Toploaders",
    short: "Álbum e Toploader",
    title: "Álbum e toploader para cartas de TCG",
    match: /album|[aá]lbum|toploader/i,
    summary:
      "Álbum para organizar a coleção em páginas e toploader rígido para guardar ou enviar carta avulsa com segurança.",
    faq: [
      {
        q: "A carta vai direto no toploader?",
        a: "O ideal é colocar a carta num sleeve antes, para ela não riscar ao deslizar para dentro do toploader.",
      },
    ],
  },
];

export const MIN_CATEGORY_ITEMS = 2;

export function categoryBySlug(slug: string): AccessoryCategory | undefined {
  return ACCESSORY_CATEGORIES.find((c) => c.slug === slug);
}

export function categoryPath(category: AccessoryCategory): string {
  return `/acessorios/${category.slug}`;
}

export function liveCategories(accessories: Accessory[]): { category: AccessoryCategory; items: Accessory[] }[] {
  return ACCESSORY_CATEGORIES.map((category) => ({
    category,
    items: accessories.filter((a) => category.match.test(a.name)),
  })).filter((c) => c.items.length >= MIN_CATEGORY_ITEMS);
}
