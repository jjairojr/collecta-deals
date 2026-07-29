import { buildFeed, type FeedItem } from "@/lib/feed";
import { siteURL } from "@/lib/site";

export const revalidate = 3600;

function esc(s: string): string {
  return s
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}

function tag(name: string, value: string): string {
  return `    <${name}>${esc(value)}</${name}>`;
}

function itemXML(it: FeedItem): string {
  const lines = [
    "  <item>",
    tag("title", it.title),
    tag("link", it.link),
    tag("description", it.description),
    tag("g:id", it.id),
    tag("g:image_link", it.imageLink),
    tag("g:availability", it.inStock ? "in_stock" : "out_of_stock"),
    tag("g:price", it.price),
    tag("g:condition", it.condition),
    tag("g:brand", it.brand),
    // Trading cards carry no GTIN or MPN. Without this Merchant Center rejects
    // every item for a missing unique identifier.
    tag("g:identifier_exists", "no"),
    tag("g:google_product_category", it.googleCategory),
    tag("g:product_type", it.productType),
    ...it.labels.map((l, i) => tag(`g:custom_label_${i}`, l)),
    "  </item>",
  ];
  return lines.join("\n");
}

export async function GET() {
  const items = await buildFeed();
  if (!items) {
    return new Response("catalog unavailable", {
      status: 503,
      headers: { "cache-control": "no-store" },
    });
  }
  const xml = [
    '<?xml version="1.0" encoding="UTF-8"?>',
    '<rss version="2.0" xmlns:g="http://base.google.com/ns/1.0">',
    "<channel>",
    tag("title", "Collecta — cartas e selados de TCG"),
    tag("link", siteURL),
    tag(
      "description",
      "Catálogo de cartas avulsas, produto selado e acessórios de TCG da Collecta.",
    ),
    ...items.map(itemXML),
    "</channel>",
    "</rss>",
    "",
  ].join("\n");
  return new Response(xml, {
    headers: {
      "content-type": "application/xml; charset=utf-8",
      "cache-control": "public, s-maxage=3600, stale-while-revalidate=86400",
      "x-robots-tag": "noindex",
    },
  });
}
