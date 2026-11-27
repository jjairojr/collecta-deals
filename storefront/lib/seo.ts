import { absoluteURL, siteURL } from "@/lib/site";
import { feedID, maker } from "@/lib/product-id";
import { collectionOf, contentsOf } from "@/lib/sealed-guide";
import { SHIPPING, STORE, STORE_MAPS_URL, STORE_PHOTOS, STORE_PROFILES } from "@/lib/store";
import type { Accessory, KitOffer, Sealed, Single } from "@/lib/types";

const ORGANIZATION_ID = `${siteURL}/#organization`;
const STORE_ID = `${siteURL}/#loja`;

function postalAddress() {
  return {
    "@type": "PostalAddress",
    streetAddress: `${STORE.street}, ${STORE.building}`,
    addressLocality: STORE.city,
    addressRegion: STORE.state,
    postalCode: STORE.cep,
    addressCountry: STORE.country,
  };
}

export function organizationJsonLd() {
  return {
    "@context": "https://schema.org",
    "@type": "Organization",
    "@id": ORGANIZATION_ID,
    name: "Collecta",
    legalName: STORE.legalName,
    taxID: STORE.cnpj,
    url: siteURL,
    logo: absoluteURL("/mascot.png"),
    telephone: STORE.phone,
    address: postalAddress(),
    sameAs: STORE_PROFILES,
    hasMerchantReturnPolicy: returnPolicy(),
    contactPoint: {
      "@type": "ContactPoint",
      telephone: STORE.phone,
      contactType: "customer service",
      areaServed: "BR",
      availableLanguage: "Portuguese",
    },
  };
}

export function storeJsonLd() {
  return {
    "@context": "https://schema.org",
    "@type": ["Store", "HobbyShop"],
    "@id": STORE_ID,
    name: STORE.name,
    description:
      "Loja de card games em Goiânia com produto selado, cartas avulsas e acessórios, mesas para jogar e torneios semanais: com premiação, gratuitos e 4fun.",
    url: absoluteURL("/loja"),
    image: STORE_PHOTOS.map((p) => absoluteURL(p.src)),
    telephone: STORE.phone,
    address: postalAddress(),
    hasMap: STORE_MAPS_URL,
    priceRange: "$$",
    currenciesAccepted: "BRL",
    paymentAccepted: "Pix, cartão de crédito, cartão de débito",
    openingHoursSpecification: [
      {
        "@type": "OpeningHoursSpecification",
        dayOfWeek: ["Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"],
        opens: STORE.opens,
        closes: STORE.closes,
      },
    ],
    sameAs: STORE_PROFILES,
    parentOrganization: { "@id": ORGANIZATION_ID },
    areaServed: ["Goiânia", "Brasil"],
  };
}

export function webSiteJsonLd() {
  return {
    "@context": "https://schema.org",
    "@type": "WebSite",
    name: "Collecta",
    url: siteURL,
    inLanguage: "pt-BR",
  };
}

function returnPolicy() {
  return {
    "@type": "MerchantReturnPolicy",
    applicableCountry: "BR",
    returnPolicyCategory: "https://schema.org/MerchantReturnFiniteReturnWindow",
    merchantReturnDays: 7,
    returnMethod: "https://schema.org/ReturnByMail",
    returnFees: "https://schema.org/FreeReturn",
    merchantReturnLink: absoluteURL("/politica-de-devolucao"),
  };
}

function days(range: readonly [number, number]) {
  return { "@type": "QuantitativeValue", minValue: range[0], maxValue: range[1], unitCode: "DAY" };
}

// The checkout makes PAC free for the whole of Brazil once the products reach
// freeShippingMinBRL (internal/api/orders.go, applyFreeShipping), and the offer
// price here is the Pix one, the lowest: an offer at or above it ships free on
// its own. Below it the freight depends on the buyer's CEP (from about R$ 25 to
// R$ 40+), so no single rate is true: the offer leaves shippingDetails out and
// Google falls back to the Merchant Center shipping settings.
export function shippingDetails(priceCents: number, freeShippingMinBRL: number) {
  if (freeShippingMinBRL <= 0 || priceCents < Math.round(freeShippingMinBRL * 100)) {
    return undefined;
  }
  return {
    "@type": "OfferShippingDetails",
    shippingLabel: "PAC grátis",
    shippingRate: {
      "@type": "MonetaryAmount",
      value: 0,
      currency: "BRL",
    },
    shippingDestination: { "@type": "DefinedRegion", addressCountry: "BR" },
    deliveryTime: {
      "@type": "ShippingDeliveryTime",
      handlingTime: days(SHIPPING.handlingDays),
      transitTime: days(SHIPPING.transitDays),
    },
  };
}

function availability(soldOut?: boolean, shipsFrom?: string): string {
  if (soldOut) {
    return "https://schema.org/OutOfStock";
  }
  return shipsFrom ? "https://schema.org/PreOrder" : "https://schema.org/InStock";
}

function offerJsonLd(
  path: string,
  priceCents: number,
  itemCondition: string,
  freeShippingMinBRL: number,
  soldOut?: boolean,
  shipsFrom?: string,
) {
  return {
    "@type": "Offer",
    url: absoluteURL(path),
    priceCurrency: "BRL",
    price: (priceCents / 100).toFixed(2),
    availability: availability(soldOut, shipsFrom),
    availabilityStarts: shipsFrom,
    itemCondition,
    seller: { "@id": ORGANIZATION_ID },
    shippingDetails: shippingDetails(priceCents, freeShippingMinBRL),
    hasMerchantReturnPolicy: returnPolicy(),
  };
}

export function productOpenGraph(path: string, imageURL?: string) {
  return {
    type: "website" as const,
    url: absoluteURL(path),
    siteName: "Collecta",
    locale: "pt_BR",
    images: imageURL ? [imageURL] : undefined,
  };
}

export function singleProductJsonLd(item: Single, freeShippingMinBRL = 0) {
  return {
    "@context": "https://schema.org",
    "@type": "Product",
    name: `${item.name} ${item.number}${item.variant ? ` ${item.variant}` : ""}`,
    sku: feedID(item.slug),
    description: `Carta ${item.name} (${item.number}) de ${item.gameLabel}, set ${item.set}${item.variant ? `, versão ${item.variant}` : ""}, condição ${item.condition}.`,
    image: item.imageURL ? absoluteURL(item.imageURL) : undefined,
    brand: { "@type": "Brand", name: maker(item.game, item.gameLabel) },
    category: "Trading Card Game",
    // A slab carries its grade (grader + note) as a property; the condition stays
    // UsedCondition like every single, since the card itself is not new stock.
    additionalProperty: item.grade ? property("Graduação", item.grade) : undefined,
    offers: offerJsonLd(
      `/carta/${item.slug}`,
      item.price,
      "https://schema.org/UsedCondition",
      freeShippingMinBRL,
      item.soldOut,
    ),
  };
}

function property(name: string, value: string | undefined) {
  return value ? [{ "@type": "PropertyValue", name, value }] : [];
}

export function sealedProductJsonLd(item: Sealed, freeShippingMinBRL = 0) {
  const collection = collectionOf(item);
  const contents = contentsOf(item);
  return {
    "@context": "https://schema.org",
    "@type": "Product",
    name: item.name,
    sku: feedID(item.slug),
    gtin: item.gtin,
    description: [
      `${item.name} — produto selado de ${item.gameLabel}${item.language ? ` em ${item.language}` : ""}, lacre original.`,
      contents ? `O que vem: ${contents}` : "",
      collection?.summary ?? "",
    ]
      .filter(Boolean)
      .join(" "),
    releaseDate: collection?.released,
    additionalProperty: [
      ...property("Formato", item.format),
      ...property("Idioma", item.language),
      ...property("Coleção", collection?.name),
    ],
    image: item.imageURL ? absoluteURL(item.imageURL) : undefined,
    brand: { "@type": "Brand", name: maker(item.game, item.gameLabel) },
    category: "Trading Card Game",
    offers: withKitOffers(
      offerJsonLd(
        `/selado/${item.slug}`,
        item.price,
        "https://schema.org/NewCondition",
        freeShippingMinBRL,
        item.soldOut,
        item.shipsFrom,
      ),
      item,
      freeShippingMinBRL,
    ),
  };
}

// Kits (N units of the same product, with their own price) are extra offers on
// the same page. Google documents ProductGroup variants only for size, colour and
// the like, not quantity, so each kit is an Offer with eligibleQuantity and its
// total price; the single unit stays the first offer, the price the page leads with.
function withKitOffers(
  unit: ReturnType<typeof offerJsonLd>,
  item: { slug: string; qty?: number; kits?: KitOffer[]; soldOut?: boolean; shipsFrom?: string },
  freeShippingMinBRL: number,
) {
  const kits = item.kits ?? [];
  if (kits.length === 0) {
    return unit;
  }
  return [
    unit,
    ...kits.map((k) => ({
      ...offerJsonLd(
        `/selado/${item.slug}`,
        k.price,
        "https://schema.org/NewCondition",
        freeShippingMinBRL,
        // A kit needs `units` in stock: with fewer left, only the single unit sells.
        item.soldOut || (item.qty !== undefined && item.qty < k.units),
        item.shipsFrom,
      ),
      name: `Kit com ${k.units} unidades`,
      sku: feedID(`${item.slug}-kit${k.units}`),
      eligibleQuantity: { "@type": "QuantitativeValue", value: k.units, unitCode: "C62" },
    })),
  ];
}

// Accessory makers, matched at the start of the product name. Unknown maker: no
// brand at all, rather than naming the store as the brand of a third-party product.
const ACCESSORY_BRANDS = ["Central", "The One", "Ultra Pro", "Dragon Shield", "KMC", "Gamegenic", "Ultimate Guard", "BCW", "Katana"];

export function accessoryBrand(name: string): string | undefined {
  const n = name.trim().toLowerCase();
  return ACCESSORY_BRANDS.find((b) => n.startsWith(b.toLowerCase() + " ") || n === b.toLowerCase());
}

export function accessoryProductJsonLd(item: Accessory, freeShippingMinBRL = 0) {
  return {
    "@context": "https://schema.org",
    "@type": "Product",
    name: item.name,
    sku: feedID(item.slug),
    gtin: item.gtin,
    description: `${item.name} — acessório para TCG, produto novo, serve para qualquer jogo.`,
    image: item.imageURL ? absoluteURL(item.imageURL) : undefined,
    brand: accessoryBrand(item.name) ? { "@type": "Brand", name: accessoryBrand(item.name) } : undefined,
    category: "Trading Card Game",
    offers: offerJsonLd(
      `/acessorio/${item.slug}`,
      item.price,
      "https://schema.org/NewCondition",
      freeShippingMinBRL,
      item.soldOut,
      item.shipsFrom,
    ),
  };
}

export function breadcrumbJsonLd(items: { name: string; path?: string }[]) {
  return {
    "@context": "https://schema.org",
    "@type": "BreadcrumbList",
    itemListElement: items.map((it, i) => ({
      "@type": "ListItem",
      position: i + 1,
      name: it.name,
      item: it.path ? absoluteURL(it.path) : undefined,
    })),
  };
}

export function itemListJsonLd(name: string, paths: string[], start = 1) {
  return {
    "@context": "https://schema.org",
    "@type": "ItemList",
    name,
    itemListElement: paths.map((path, i) => ({
      "@type": "ListItem",
      position: start + i,
      url: absoluteURL(path),
    })),
  };
}

export function articleJsonLd(a: {
  title: string;
  description: string;
  path: string;
  updated: string;
  published?: string;
}) {
  const org = { "@type": "Organization", "@id": ORGANIZATION_ID, name: "Collecta", url: siteURL };
  return {
    "@context": "https://schema.org",
    "@type": "Article",
    headline: a.title,
    description: a.description,
    inLanguage: "pt-BR",
    // Guides record only their last update; with no separate publication date the
    // first known date is the update itself, never an invented earlier one.
    datePublished: a.published ?? a.updated,
    dateModified: a.updated,
    mainEntityOfPage: absoluteURL(a.path),
    image: absoluteURL("/opengraph-image"),
    author: org,
    publisher: { ...org, logo: { "@type": "ImageObject", url: absoluteURL("/mascot.png") } },
  };
}

export interface FaqItem {
  q: string;
  a: string;
}

export function faqJsonLd(items: FaqItem[]) {
  return {
    "@context": "https://schema.org",
    "@type": "FAQPage",
    mainEntity: items.map((it) => ({
      "@type": "Question",
      name: it.q,
      acceptedAnswer: { "@type": "Answer", text: it.a },
    })),
  };
}
