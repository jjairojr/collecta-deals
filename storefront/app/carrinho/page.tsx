import { Suspense } from "react";
import CartView from "@/components/cart/CartView";
import { loadCatalog } from "@/lib/catalog";

export const revalidate = 60;

export const metadata = {
  title: "Carrinho",
  robots: { index: false },
};

export default async function CarrinhoPage() {
  const { freeShippingMinBRL } = await loadCatalog();
  return (
    <Suspense>
      <CartView freeShippingMinBRL={freeShippingMinBRL} />
    </Suspense>
  );
}
