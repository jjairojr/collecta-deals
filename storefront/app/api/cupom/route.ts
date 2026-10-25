import { checkoutFetch } from "@/lib/backend";
import { sessionUser } from "@/lib/session";

export async function POST(req: Request): Promise<Response> {
  let payload: { code?: unknown };
  try {
    payload = (await req.json()) as { code?: unknown };
  } catch {
    return Response.json({ error: "pedido inválido" }, { status: 400 });
  }
  const code = typeof payload.code === "string" ? payload.code.trim() : "";
  if (!code || code.length > 40) {
    return Response.json({ error: "coupon_invalid", message: "Cupom não encontrado" }, { status: 409 });
  }
  const session = await sessionUser();
  if (!session) {
    return Response.json({ error: "login_required" }, { status: 401 });
  }
  return checkoutFetch("/api/promos/check", {
    method: "POST",
    body: JSON.stringify({ code, collectaUserId: session.user.id }),
  });
}
