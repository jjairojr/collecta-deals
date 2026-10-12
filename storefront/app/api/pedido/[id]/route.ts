import { checkoutFetch } from "@/lib/backend";

export async function GET(
  req: Request,
  { params }: { params: Promise<{ id: string }> },
): Promise<Response> {
  const { id } = await params;
  if (!/^[a-f0-9]{16}$/.test(id)) {
    return Response.json({ error: "not found" }, { status: 404 });
  }
  const paymentID = new URL(req.url).searchParams.get("payment_id");
  const query = paymentID && /^\d+$/.test(paymentID) ? `?payment_id=${paymentID}` : "";
  return checkoutFetch(`/api/orders/${id}${query}`);
}
