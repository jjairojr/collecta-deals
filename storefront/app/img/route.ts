// Proxies card-art requests to the Go backend so the backend host never appears
// in the browser. Only used for cards without a public TCGplayer product image
// (e.g. Pokémon or unmatched cards); those with a productID load straight from
// the TCGplayer CDN.
export const revalidate = 86400;

export async function GET(req: Request): Promise<Response> {
  const base = process.env.STOREFRONT_API;
  if (!base) {
    return new Response("storefront api not configured", { status: 500 });
  }
  const incoming = new URL(req.url);
  const target = `${base}/api/card-image?${incoming.searchParams.toString()}`;
  try {
    const res = await fetch(target, { next: { revalidate: 86400 } });
    if (!res.ok) {
      return new Response("not found", { status: 404 });
    }
    const body = await res.arrayBuffer();
    return new Response(body, {
      headers: {
        "Content-Type": res.headers.get("Content-Type") ?? "image/jpeg",
        "Cache-Control": "public, max-age=86400, stale-while-revalidate=604800",
      },
    });
  } catch {
    return new Response("upstream error", { status: 502 });
  }
}
