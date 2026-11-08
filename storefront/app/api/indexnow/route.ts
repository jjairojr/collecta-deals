import { loadCatalog } from "@/lib/catalog";
import { collectionPath, liveCollections } from "@/lib/sealed-guide";
import { absoluteURL, siteURL } from "@/lib/site";

export const dynamic = "force-dynamic";

const INDEXNOW_KEY = "8dcbe9217ff8190a0eeac3fa63ed1bac";

export async function GET(req: Request): Promise<Response> {
  const secret = process.env.CRON_SECRET;
  if (!secret) {
    return new Response("CRON_SECRET not configured", { status: 503 });
  }
  if (req.headers.get("authorization") !== `Bearer ${secret}`) {
    return new Response("unauthorized", { status: 401 });
  }
  const { sealed, accessories, live } = await loadCatalog();
  if (!live) {
    return new Response("catalog unavailable", { status: 503 });
  }
  const paths = [
    "/",
    "/selado",
    "/cartinhas-pokemon",
    ...[...new Set(sealed.map((s) => s.game))].map((g) => `/selado/${g}`),
    ...liveCollections(sealed).map(({ collection }) => collectionPath(collection)),
    ...sealed.map((s) => `/selado/${s.slug}`),
    ...accessories.map((a) => `/acessorio/${a.slug}`),
  ];
  const res = await fetch("https://api.indexnow.org/indexnow", {
    method: "POST",
    headers: { "Content-Type": "application/json; charset=utf-8" },
    body: JSON.stringify({
      host: new URL(siteURL).host,
      key: INDEXNOW_KEY,
      keyLocation: absoluteURL(`/${INDEXNOW_KEY}.txt`),
      urlList: paths.map((p) => absoluteURL(p)),
    }),
  });
  return Response.json({ submitted: paths.length, status: res.status });
}
