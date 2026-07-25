import { absoluteURL, siteURL } from "@/lib/site";
import type { Sealed, Single } from "@/lib/types";

export function organizationJsonLd() {
  return {
    "@context": "https://schema.org",
    "@type": "Organization",
    name: "Collecta",
    url: siteURL,
    logo: absoluteURL("/mascot.png"),
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

function offerJsonLd(path: string, priceCents: number, itemCondition: string) {
  return {
    "@type": "Offer",
    url: absoluteURL(path),
    priceCurrency: "BRL",
    price: (priceCents / 100).toFixed(2),
    availability: "https://schema.org/InStock",
    itemCondition,
    seller: { "@type": "Organization", name: "Collecta" },
  };
}

export function singleProductJsonLd(item: Single) {
  return {
    "@context": "https://schema.org",
    "@type": "Product",
    name: `${item.name} ${item.number}`,
    sku: item.number,
    description: `Carta ${item.name} (${item.number}) do set ${item.set}, condição ${item.condition}.`,
    image: item.imageURL ? absoluteURL(item.imageURL) : undefined,
    brand: { "@type": "Brand", name: item.gameLabel },
    category: "Trading Card Game",
    offers: offerJsonLd(
      `/carta/${item.slug}`,
      item.price,
      "https://schema.org/UsedCondition",
    ),
  };
}

export function sealedProductJsonLd(item: Sealed) {
  return {
    "@context": "https://schema.org",
    "@type": "Product",
    name: item.name,
    description: `${item.name} — produto selado de ${item.gameLabel}${item.language ? ` em ${item.language}` : ""}, lacre original.`,
    image: item.imageURL ? absoluteURL(item.imageURL) : undefined,
    brand: { "@type": "Brand", name: item.gameLabel },
    category: "Trading Card Game",
    offers: offerJsonLd(
      `/selado/${item.slug}`,
      item.price,
      "https://schema.org/NewCondition",
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
