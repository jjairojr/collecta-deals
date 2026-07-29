import type { MetadataRoute } from "next";
import { siteURL } from "@/lib/site";

export default function robots(): MetadataRoute.Robots {
  return {
    // /img has to stay crawlable: it serves every card's art, and both the
    // Merchant Center feed (image_link) and the Product rich results point at
    // it. Blocking it disapproves the whole catalog for "image not accessible".
    //
    // Googlebot and Googlebot-Image are named explicitly because Merchant
    // Center refuses to run its policy/quality crawl otherwise — the wildcard
    // group alone fails its check. A named group fully replaces the wildcard
    // for that crawler, so /carrinho has to be repeated rather than inherited.
    rules: [
      { userAgent: "*", allow: "/", disallow: ["/carrinho"] },
      { userAgent: "Googlebot", allow: "/", disallow: ["/carrinho"] },
      { userAgent: "Googlebot-Image", allow: "/" },
    ],
    sitemap: `${siteURL}/sitemap.xml`,
  };
}
