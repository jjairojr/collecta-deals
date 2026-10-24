import type { MetadataRoute } from "next";
import { loadCatalog } from "@/lib/catalog";
import { pageCount } from "@/lib/paging";
import { setsOf } from "@/lib/sets";
import { absoluteURL } from "@/lib/site";

export const revalidate = 3600;

export default async function sitemap(): Promise<MetadataRoute.Sitemap> {
  const { singles, sealed, accessories, soldOut, games, live } =
    await loadCatalog();
  const urls: MetadataRoute.Sitemap = [
    { url: absoluteURL("/"), changeFrequency: "daily", priority: 1 },
    { url: absoluteURL("/singles"), changeFrequency: "daily", priority: 0.9 },
    { url: absoluteURL("/selado"), changeFrequency: "daily", priority: 0.9 },
    { url: absoluteURL("/loja"), changeFrequency: "monthly", priority: 0.8 },
    {
      url: absoluteURL("/politica-de-devolucao"),
      changeFrequency: "yearly",
      priority: 0.3,
    },
    { url: absoluteURL("/frete"), changeFrequency: "yearly", priority: 0.3 },
    { url: absoluteURL("/termos"), changeFrequency: "yearly", priority: 0.3 },
    { url: absoluteURL("/privacidade"), changeFrequency: "yearly", priority: 0.3 },
    { url: absoluteURL("/sobre"), changeFrequency: "yearly", priority: 0.3 },
  ];
  if (live && accessories.length > 0) {
    urls.push({
      url: absoluteURL("/acessorios"),
      changeFrequency: "daily",
      priority: 0.9,
    });
  }
  if (!live) {
    return urls;
  }
  for (let n = 2; n <= pageCount(singles.length); n += 1) {
    urls.push({
      url: absoluteURL(`/singles/pagina/${n}`),
      changeFrequency: "daily",
      priority: 0.6,
    });
  }
  for (const g of games) {
    for (const s of setsOf(singles, g.id)) {
      urls.push({
        url: absoluteURL(`/singles/${g.id}/colecao/${s.slug}`),
        changeFrequency: "daily",
        priority: 0.7,
      });
      for (let n = 2; n <= pageCount(s.count); n += 1) {
        urls.push({
          url: absoluteURL(`/singles/${g.id}/colecao/${s.slug}/pagina/${n}`),
          changeFrequency: "daily",
          priority: 0.5,
        });
      }
    }
  }
  for (const g of games) {
    urls.push({
      url: absoluteURL(`/singles/${g.id}`),
      changeFrequency: "daily",
      priority: 0.8,
    });
    for (let n = 2; n <= pageCount(g.count); n += 1) {
      urls.push({
        url: absoluteURL(`/singles/${g.id}/pagina/${n}`),
        changeFrequency: "daily",
        priority: 0.6,
      });
    }
  }
  const sealedGames = [...new Set(sealed.map((s) => s.game))];
  for (const id of sealedGames) {
    urls.push({
      url: absoluteURL(`/selado/${id}`),
      changeFrequency: "daily",
      priority: 0.8,
    });
  }
  const product = (
    path: string,
    p: { imageURL?: string; addedAt?: string },
    priority: number,
  ) => {
    urls.push({
      url: absoluteURL(path),
      lastModified: p.addedAt,
      changeFrequency: "weekly",
      priority,
      images: p.imageURL ? [absoluteURL(p.imageURL)] : undefined,
    });
  };
  for (const s of singles) {
    product(`/carta/${s.slug}`, s, 0.7);
  }
  for (const s of sealed) {
    product(`/selado/${s.slug}`, s, 0.7);
  }
  for (const a of accessories) {
    product(`/acessorio/${a.slug}`, a, 0.7);
  }
  for (const s of soldOut.singles) {
    product(`/carta/${s.slug}`, s, 0.4);
  }
  for (const s of soldOut.sealed) {
    product(`/selado/${s.slug}`, s, 0.4);
  }
  for (const a of soldOut.accessories) {
    product(`/acessorio/${a.slug}`, a, 0.4);
  }
  return urls;
}
