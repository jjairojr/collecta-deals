import { errorResponse, NO_STORE } from "@/lib/account-api";
import { referralMe } from "@/lib/collecta";
import type { CheckoutCoupon } from "@/lib/coupons";
import { sessionUser } from "@/lib/session";

export async function GET(): Promise<Response> {
  const session = await sessionUser();
  if (!session) {
    return Response.json({ error: "login_required" }, { status: 401, headers: NO_STORE });
  }
  try {
    const overview = await referralMe(session.token);
    const coupons: CheckoutCoupon[] = overview.coupons
      .filter((c) => c.unlockedAt !== null && c.usedAt === null)
      .map((c) => ({ id: c.id, code: c.code, valueCents: c.valueCents, minOrderCents: c.minOrderCents }));
    return Response.json({ coupons, minOrderCents: overview.minOrderCents }, { headers: NO_STORE });
  } catch (err) {
    return errorResponse(err);
  }
}
