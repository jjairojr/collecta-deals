import sharp from "sharp";
import { isProxiablePhoto, itemPhotoSource } from "@/lib/photo";

export const revalidate = 86400;

const MAX_WIDTH = 900;

async function toWebp(body: ArrayBuffer): Promise<Buffer | null> {
  try {
    return await sharp(Buffer.from(body))
      .resize({ width: MAX_WIDTH, withoutEnlargement: true })
      .webp({ quality: 82 })
      .toBuffer();
  } catch {
    return null;
  }
}

export async function GET(req: Request): Promise<Response> {
  const params = new URL(req.url).searchParams;
  const item = params.get("item");
  const external = params.get("u");
  const source = item ? itemPhotoSource(item) : external && isProxiablePhoto(external) ? external : undefined;
  if (!source) {
    return new Response("not found", { status: 404 });
  }
  try {
    const res = await fetch(source, { next: { revalidate: 86400 } });
    if (!res.ok) {
      return new Response("not found", { status: 404 });
    }
    const body = await res.arrayBuffer();
    const webp = await toWebp(body);
    return new Response(webp ? new Uint8Array(webp) : body, {
      headers: {
        "Content-Type": webp ? "image/webp" : (res.headers.get("Content-Type") ?? "image/jpeg"),
        "Cache-Control": "public, max-age=86400, s-maxage=604800, stale-while-revalidate=604800",
      },
    });
  } catch {
    return new Response("upstream error", { status: 502 });
  }
}
