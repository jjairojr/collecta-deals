import type { MetadataRoute } from "next";
import { loadCatalog } from "@/lib/catalog";
import { pageCount } from "@/lib/paging";
import { absoluteURL } from "@/lib/site";

export const revalidate = 3600;

export default async function sitemap(): Promise<MetadataRoute.Sitemap> {
  const { singles, sealed, accessories, games, navGames, live } =
    await loadCatalog();
  const urls: MetadataRoute.Sitemap = [
    { url: absoluteURL("/"), changeFrequency: "daily", priority: 1 },
    { url: absoluteURL("/singles"), changeFrequency: "daily", priority: 0.9 },
    { url: absoluteURL("/selado"), changeFrequency: "daily", priority: 0.9 },
    {
      url: absoluteURL("/politica-de-devolucao"),
      changeFrequency: "yearly",
      priority: 0.3,
    },
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
  for (const g of navGames) {
    urls.push({
      url: absoluteURL(`/jogo/${g.id}`),
      changeFrequency: "daily",
      priority: 0.8,
    });
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
  for (const s of singles) {
    urls.push({
      url: absoluteURL(`/carta/${s.slug}`),
      changeFrequency: "weekly",
      priority: 0.7,
    });
  }
  for (const s of sealed) {
    urls.push({
      url: absoluteURL(`/selado/${s.slug}`),
      changeFrequency: "weekly",
      priority: 0.7,
    });
  }
  for (const a of accessories) {
    urls.push({
      url: absoluteURL(`/acessorio/${a.slug}`),
      changeFrequency: "weekly",
      priority: 0.7,
    });
  }
  return urls;
}
