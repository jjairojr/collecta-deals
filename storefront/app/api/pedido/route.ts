import { checkoutFetch } from "@/lib/backend";
import { sessionUser } from "@/lib/session";
import { siteURL } from "@/lib/site";

/** The Meta Pixel cookie (_fbp or _fbc) from the request, or "". */
function metaCookie(req: Request, name: string): string {
  const raw = req.headers.get("cookie") ?? "";
  for (const part of raw.split(";")) {
    const [k, ...v] = part.trim().split("=");
    if (k === name) {
      return decodeURIComponent(v.join("="));
    }
  }
  return "";
}

export async function POST(req: Request): Promise<Response> {
  let payload: Record<string, unknown>;
  try {
    payload = (await req.json()) as Record<string, unknown>;
  } catch {
    return Response.json({ error: "pedido inválido" }, { status: 400 });
  }
  const { couponIds: rawCoupons, promoCode: rawPromo, ...rest } = payload;
  delete rest.collectaUserId;
  const couponIds = Array.isArray(rawCoupons) ? rawCoupons.filter((c): c is string => typeof c === "string") : [];
  const promoCode = typeof rawPromo === "string" ? rawPromo.trim() : "";
  const session = await sessionUser();
  if ((couponIds.length > 0 || promoCode) && !session) {
    return Response.json({ error: "login_required" }, { status: 401 });
  }
  return checkoutFetch("/api/orders", {
    method: "POST",
    body: JSON.stringify({
      ...rest,
      backURLBase: siteURL,
      collectaUserId: session?.user.id ?? "",
      couponIds,
      promoCode,
      // For the purchase to reach Meta from the server (Conversions API) and match
      // the browser Pixel: the browser and the Pixel cookies, nothing else.
      ads: {
        userAgent: req.headers.get("user-agent") ?? "",
        fbp: metaCookie(req, "_fbp"),
        fbc: metaCookie(req, "_fbc"),
      },
    }),
  });
}
