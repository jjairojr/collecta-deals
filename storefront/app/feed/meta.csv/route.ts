import { buildFeed, type FeedItem } from "@/lib/feed";

export const revalidate = 3600;

// Meta's column names overlap Google's but not its vocabulary: availability is
// spelled with a space, and stock goes in quantity_to_sell_on_facebook.
const COLUMNS = [
  "id",
  "title",
  "description",
  "availability",
  "condition",
  "price",
  "link",
  "image_link",
  "brand",
  "quantity_to_sell_on_facebook",
  "google_product_category",
  "product_type",
  "custom_label_0",
  "custom_label_1",
  "custom_label_2",
  "custom_label_3",
];

function cell(value: string): string {
  return `"${value.replace(/"/g, '""')}"`;
}

function row(it: FeedItem): string {
  return [
    it.id,
    it.title,
    it.description,
    it.inStock ? "in stock" : "out of stock",
    it.condition,
    it.price,
    it.link,
    it.imageLink,
    it.brand,
    String(it.quantity),
    it.googleCategory,
    it.productType,
    it.labels[0] ?? "",
    it.labels[1] ?? "",
    it.labels[2] ?? "",
    it.labels[3] ?? "",
  ]
    .map(cell)
    .join(",");
}

export async function GET() {
  const items = await buildFeed();
  if (!items) {
    return new Response("catalog unavailable", {
      status: 503,
      headers: { "cache-control": "no-store" },
    });
  }
  const csv = [COLUMNS.map(cell).join(","), ...items.map(row), ""].join("\n");
  return new Response(csv, {
    headers: {
      "content-type": "text/csv; charset=utf-8",
      "cache-control": "public, s-maxage=3600, stale-while-revalidate=86400",
      "x-robots-tag": "noindex",
    },
  });
}
