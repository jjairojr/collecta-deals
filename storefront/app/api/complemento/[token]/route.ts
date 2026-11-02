import { NO_STORE } from "@/lib/account-api";
import { checkoutFetch } from "@/lib/backend";
import { resolveCartLines } from "@/lib/catalog";

interface PostsaleLink {
  orderNumber?: number;
  lines?: { sku?: unknown; qty?: unknown }[];
}

export async function GET(
  _req: Request,
  { params }: { params: Promise<{ token: string }> },
): Promise<Response> {
  const { token } = await params;
  if (!/^[A-Za-z0-9_-]{1,200}$/.test(token)) {
    return Response.json({ error: "not_found" }, { status: 404, headers: NO_STORE });
  }
  const res = await checkoutFetch(`/api/postsale/link/${token}`);
  if (res.status === 404) {
    return Response.json({ error: "not_found" }, { status: 404, headers: NO_STORE });
  }
  if (res.status === 410) {
    const gone = (await res.json().catch(() => ({}))) as PostsaleLink;
    const orderNumber = typeof gone.orderNumber === "number" ? gone.orderNumber : undefined;
    return Response.json({ error: "expired", orderNumber }, { status: 410, headers: NO_STORE });
  }
  if (!res.ok) {
    return Response.json({ error: "unavailable" }, { status: 502, headers: NO_STORE });
  }
  const data = (await res.json()) as PostsaleLink;
  if (typeof data.orderNumber !== "number" || !Array.isArray(data.lines)) {
    return Response.json({ error: "unavailable" }, { status: 502, headers: NO_STORE });
  }
  const wanted = data.lines.flatMap((l) =>
    typeof l.sku === "string" && typeof l.qty === "number" && l.qty > 0 ? [{ sku: l.sku, qty: l.qty }] : [],
  );
  const { lines, missing } = await resolveCartLines(wanted);
  return Response.json({ orderNumber: data.orderNumber, lines, missing }, { headers: NO_STORE });
}
