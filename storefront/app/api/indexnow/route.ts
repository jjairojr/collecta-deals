import { gradedGames, loadCatalog } from "@/lib/catalog";
import { GAME_HUBS } from "@/lib/game-hubs";
import { GUIDES, guidePath } from "@/lib/guides";
import { collectionPath, liveCollections } from "@/lib/sealed-guide";
import { absoluteURL, siteURL } from "@/lib/site";

export const dynamic = "force-dynamic";

// Publicada em /public/<chave>.txt, que é como o IndexNow confere o dono do site.
const INDEXNOW_KEY = "8dcbe9217ff8190a0eeac3fa63ed1bac";
// O cron roda uma vez por dia (vercel.json); a margem cobre atraso do cron.
const RECENT_MS = 26 * 60 * 60 * 1000;
// Limite do IndexNow por pedido.
const BATCH = 10_000;

function recent(date: string | undefined, now: number): boolean {
  if (!date) {
    return false;
  }
  const t = Date.parse(date);
  return Number.isFinite(t) && now - t <= RECENT_MS;
}

// Avisa a api.indexnow.org (repassa a Bing, Yandex, Seznam, Naver e outros) das
// páginas que mudam todo dia: entradas, coleções, selados e acessórios (estoque e
// preço mudam), e das cartas e guias novos ou atualizados nas últimas 26 h. As
// cartas antigas ficam de fora para não mandar o catálogo inteiro todo dia.
export async function GET(req: Request): Promise<Response> {
  const secret = process.env.CRON_SECRET;
  if (!secret) {
    return new Response("CRON_SECRET not configured", { status: 503 });
  }
  if (req.headers.get("authorization") !== `Bearer ${secret}`) {
    return new Response("unauthorized", { status: 401 });
  }
  const catalog = await loadCatalog();
  const { singles, sealed, accessories, games, live } = catalog;
  if (!live) {
    return new Response("catalog unavailable", { status: 503 });
  }
  const now = Date.now();
  const paths = [
    ...new Set([
      "/",
      "/selado",
      "/singles",
      "/cartinhas-pokemon",
      ...GAME_HUBS.map((h) => h.path),
      ...games.map((g) => `/singles/${g.id}`),
      ...[...new Set(sealed.map((s) => s.game))].map((g) => `/selado/${g}`),
      ...[...gradedGames(catalog)].map((g) => `/graduadas/${g}`),
      ...liveCollections(sealed).map(({ collection }) => collectionPath(collection)),
      ...sealed.map((s) => `/selado/${s.slug}`),
      ...accessories.map((a) => `/acessorio/${a.slug}`),
      ...singles.filter((s) => recent(s.addedAt, now)).map((s) => `/carta/${s.slug}`),
      ...GUIDES.filter((g) => recent(g.updated, now)).map((g) => guidePath(g)),
    ]),
  ];
  const statuses: number[] = [];
  for (let i = 0; i < paths.length; i += BATCH) {
    const res = await fetch("https://api.indexnow.org/indexnow", {
      method: "POST",
      headers: { "Content-Type": "application/json; charset=utf-8" },
      body: JSON.stringify({
        host: new URL(siteURL).host,
        key: INDEXNOW_KEY,
        keyLocation: absoluteURL(`/${INDEXNOW_KEY}.txt`),
        urlList: paths.slice(i, i + BATCH).map((p) => absoluteURL(p)),
      }),
    });
    statuses.push(res.status);
  }
  return Response.json({ submitted: paths.length, status: statuses[0] ?? 0, statuses });
}
