import type { MetadataRoute } from "next";
import { siteURL } from "@/lib/site";

export default function robots(): MetadataRoute.Robots {
  return {
    rules: [{ userAgent: "*", allow: "/", disallow: ["/carrinho", "/img"] }],
    sitemap: `${siteURL}/sitemap.xml`,
  };
}
