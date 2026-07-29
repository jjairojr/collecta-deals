import type { MetadataRoute } from "next";
import { siteURL } from "@/lib/site";

export default function robots(): MetadataRoute.Robots {
  return {
    // /img has to stay crawlable: it serves every card's art, and both the
    // Merchant Center feed (image_link) and the Product rich results point at
    // it. Blocking it disapproves the whole catalog for "image not accessible".
    rules: [{ userAgent: "*", allow: "/", disallow: ["/carrinho"] }],
    sitemap: `${siteURL}/sitemap.xml`,
  };
}
