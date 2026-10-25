import { checkoutFetch } from "@/lib/backend";
import { sessionUser } from "@/lib/session";
import { siteURL } from "@/lib/site";

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
    }),
  });
}
