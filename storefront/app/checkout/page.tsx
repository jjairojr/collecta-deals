import CheckoutView from "@/components/checkout/CheckoutView";

export const metadata = {
  title: "Checkout",
  robots: { index: false },
};

export default async function CheckoutPage({
  searchParams,
}: {
  searchParams: Promise<Record<string, string | string[] | undefined>>;
}) {
  const sp = await searchParams;
  const raw = sp.join;
  const joinOrder = typeof raw === "string" && /^\d+$/.test(raw) ? raw : undefined;
  return <CheckoutView joinOrder={joinOrder} />;
}
