import { checkoutFetch } from "@/lib/backend";
import { siteURL } from "@/lib/site";

export async function POST(req: Request): Promise<Response> {
  let payload: Record<string, unknown>;
  try {
    payload = (await req.json()) as Record<string, unknown>;
  } catch {
    return Response.json({ error: "pedido inválido" }, { status: 400 });
  }
  return checkoutFetch("/api/orders", {
    method: "POST",
    body: JSON.stringify({ ...payload, backURLBase: siteURL }),
  });
}
