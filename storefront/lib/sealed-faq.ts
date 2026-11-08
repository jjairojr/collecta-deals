import { brl } from "@/lib/format";
import { CARD_PRICED_INSTALLMENTS, MAX_INSTALLMENTS, shipsFromLabel } from "@/lib/payment";
import { contentsOf, releaseLabel, type SealedCollection } from "@/lib/sealed-guide";
import type { FaqItem } from "@/lib/seo";
import { SHIPPING } from "@/lib/store";
import type { Sealed } from "@/lib/types";

function priceLine(s: Sealed): string {
  const card = s.cardPrice !== undefined ? ` (${brl(s.cardPrice)} no cartão)` : "";
  return `${s.headline ?? s.name}: ${brl(s.price)} no Pix${card}`;
}

export const SEALED_ORIGINAL: FaqItem = {
  q: "Os produtos selados são originais?",
  a: "Sim. Só vendemos produto oficial com lacre de fábrica. Os selados de Pokémon em português são os distribuídos pela Copag no Brasil, e os importados (inglês, japonês e chinês) chegam lacrados de origem. Cada caixa é conferida antes do envio.",
};

export const SEALED_LANGUAGES: FaqItem = {
  q: "Qual a diferença entre selado em português, inglês, japonês e chinês?",
  a: "O jogo é o mesmo, muda o idioma impresso nas cartas e, às vezes, a linha de produtos: os GEM Pack, por exemplo, só existem em chinês. Português e inglês são os mais usados para jogar no Brasil; japonês e chinês são procurados por colecionadores.",
};

export const SEALED_SHIPPING: FaqItem = {
  q: "Como é o envio dos selados?",
  a: `Embalagem reforçada e envio pelos Correios para todo o Brasil com código de rastreio, com frete grátis acima de R$ ${SHIPPING.freeAboveBRL}. Em Goiânia você retira na loja física sem pagar frete.`,
};

export const SEALED_PAYMENT: FaqItem = {
  q: "Como pagar?",
  a: `Pix ou cartão pelo Mercado Pago, direto no checkout do site. Produtos com preço próprio no cartão parcelam em até ${CARD_PRICED_INSTALLMENTS}x; os demais, em até ${MAX_INSTALLMENTS}x.`,
};

export const SEALED_GENERAL: FaqItem[] = [SEALED_ORIGINAL, SEALED_LANGUAGES, SEALED_SHIPPING, SEALED_PAYMENT];

function contentsFaq(items: Sealed[]): FaqItem[] {
  return items.flatMap((s) => {
    const contents = contentsOf(s);
    return contents ? [{ q: `O que vem no ${s.headline ?? s.name}?`, a: contents }] : [];
  });
}

export function collectionFaq(collection: SealedCollection, items: Sealed[]): FaqItem[] {
  const out: FaqItem[] = [
    {
      q: `Quanto custam os produtos de ${collection.short} na Collecta?`,
      a: `${items.map(priceLine).join("; ")}. Preços atualizados com o estoque do site.`,
    },
  ];
  if (collection.released) {
    out.push({
      q: `Quando foi lançada ${collection.short}?`,
      a: `${collection.name} foi lançada em ${releaseLabel(collection.released)}${collection.nameEN && collection.nameEN !== collection.short ? `; em inglês, a coleção se chama ${collection.nameEN}` : ""}.`,
    });
  }
  return [...out, ...contentsFaq(items), SEALED_ORIGINAL, SEALED_SHIPPING];
}

export function productFaq(item: Sealed, collection: SealedCollection | undefined): FaqItem[] {
  const out: FaqItem[] = [];
  if (!item.soldOut) {
    out.push({ q: `Quanto custa o ${item.headline ?? item.name}?`, a: `${priceLine(item)}, na Collecta.` });
  }
  out.push(...contentsFaq([item]));
  if (item.shipsFrom) {
    out.push({
      q: "Quando a pré-venda é enviada?",
      a: `O envio e a retirada na loja começam em ${shipsFromLabel(item.shipsFrom)}.`,
    });
  }
  if (collection?.released) {
    out.push({
      q: `Quando foi lançada ${collection.short}?`,
      a: `${collection.name} foi lançada em ${releaseLabel(collection.released)}.`,
    });
  }
  return [...out, ...SEALED_GENERAL];
}
