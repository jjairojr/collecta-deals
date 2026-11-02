import ComplementView from "@/components/checkout/ComplementView";

export const metadata = {
  title: "Complemento do pedido",
  robots: { index: false },
};

export default async function ComplementoPage({
  params,
}: {
  params: Promise<{ token: string }>;
}) {
  const { token } = await params;
  return <ComplementView token={token} />;
}
