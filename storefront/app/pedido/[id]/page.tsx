import OrderStatusView from "@/components/checkout/OrderStatusView";

export const metadata = {
  title: "Seu pedido",
  robots: { index: false },
};

export default async function PedidoPage({
  params,
  searchParams,
}: {
  params: Promise<{ id: string }>;
  searchParams: Promise<Record<string, string | string[] | undefined>>;
}) {
  const { id } = await params;
  const sp = await searchParams;
  const raw = sp.payment_id;
  const paymentID = typeof raw === "string" && /^\d+$/.test(raw) ? raw : undefined;
  return <OrderStatusView id={id} paymentID={paymentID} />;
}
