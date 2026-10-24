import { absoluteURL, siteURL } from "@/lib/site";
import { feedID, maker } from "@/lib/product-id";
import { SHIPPING, STORE, STORE_MAPS_URL, STORE_PHOTOS } from "@/lib/store";
import type { Accessory, Sealed, Single } from "@/lib/types";

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
    sameAs: [STORE.instagram],
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
    sameAs: [STORE.instagram],
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

function shippingDetails(priceCents: number) {
  return {
    "@type": "OfferShippingDetails",
    shippingRate: {
      "@type": "MonetaryAmount",
      value: priceCents >= SHIPPING.freeAboveBRL * 100 ? 0 : SHIPPING.minBRL,
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

function offerJsonLd(path: string, priceCents: number, itemCondition: string, soldOut?: boolean) {
  return {
    "@type": "Offer",
    url: absoluteURL(path),
    priceCurrency: "BRL",
    price: (priceCents / 100).toFixed(2),
    availability: soldOut ? "https://schema.org/OutOfStock" : "https://schema.org/InStock",
    itemCondition,
    seller: { "@id": ORGANIZATION_ID },
    shippingDetails: shippingDetails(priceCents),
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

export function singleProductJsonLd(item: Single) {
  return {
    "@context": "https://schema.org",
    "@type": "Product",
    name: `${item.name} ${item.number}${item.variant ? ` ${item.variant}` : ""}`,
    sku: feedID(item.slug),
    description: `Carta ${item.name} (${item.number}) de ${item.gameLabel}, set ${item.set}${item.variant ? `, versão ${item.variant}` : ""}, condição ${item.condition}.`,
    image: item.imageURL ? absoluteURL(item.imageURL) : undefined,
    brand: { "@type": "Brand", name: maker(item.game, item.gameLabel) },
    category: "Trading Card Game",
    offers: offerJsonLd(
      `/carta/${item.slug}`,
      item.price,
      "https://schema.org/UsedCondition",
      item.soldOut,
    ),
  };
}

export function sealedProductJsonLd(item: Sealed) {
  return {
    "@context": "https://schema.org",
    "@type": "Product",
    name: item.name,
    sku: feedID(item.slug),
    gtin: item.gtin,
    description: `${item.name} — produto selado de ${item.gameLabel}${item.language ? ` em ${item.language}` : ""}, lacre original.`,
    image: item.imageURL ? absoluteURL(item.imageURL) : undefined,
    brand: { "@type": "Brand", name: maker(item.game, item.gameLabel) },
    category: "Trading Card Game",
    offers: offerJsonLd(
      `/selado/${item.slug}`,
      item.price,
      "https://schema.org/NewCondition",
      item.soldOut,
    ),
  };
}

export function accessoryProductJsonLd(item: Accessory) {
  return {
    "@context": "https://schema.org",
    "@type": "Product",
    name: item.name,
    sku: feedID(item.slug),
    gtin: item.gtin,
    description: `${item.name} — acessório para TCG, produto novo, serve para qualquer jogo.`,
    image: item.imageURL ? absoluteURL(item.imageURL) : undefined,
    brand: { "@type": "Brand", name: item.set || "Collecta" },
    category: "Trading Card Game",
    offers: offerJsonLd(
      `/acessorio/${item.slug}`,
      item.price,
      "https://schema.org/NewCondition",
      item.soldOut,
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

export function itemListJsonLd(name: string, paths: string[]) {
  return {
    "@context": "https://schema.org",
    "@type": "ItemList",
    name,
    itemListElement: paths.map((path, i) => ({
      "@type": "ListItem",
      position: i + 1,
      url: absoluteURL(path),
    })),
  };
}
