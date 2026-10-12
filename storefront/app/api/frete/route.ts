import { checkoutFetch } from "@/lib/backend";

export async function POST(req: Request): Promise<Response> {
  const body = await req.text();
  return checkoutFetch("/api/shipping/quote", { method: "POST", body });
}
