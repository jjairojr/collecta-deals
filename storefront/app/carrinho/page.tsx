import { Suspense } from "react";
import CartView from "@/components/cart/CartView";
import { loadCatalog } from "@/lib/catalog";
import { pickBoosterKit } from "@/lib/offers";

export const metadata = {
  title: "Carrinho",
  robots: { index: false },
};

export const revalidate = 60;

export default async function CarrinhoPage() {
  const { sealed, freeShippingMinBRL } = await loadCatalog();
  return (
    <Suspense>
      <CartView kitOffer={pickBoosterKit(sealed)} freeShippingMinBRL={freeShippingMinBRL} />
    </Suspense>
  );
}
