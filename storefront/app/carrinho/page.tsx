import { Suspense } from "react";
import CartView from "@/components/cart/CartView";

export const metadata = {
  title: "Carrinho",
  robots: { index: false },
};

export default function CarrinhoPage() {
  return (
    <Suspense>
      <CartView />
    </Suspense>
  );
}
