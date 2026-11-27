import { freeShippingSentence, pageShowsFreeShipping } from "@/lib/free-shipping";

// The free PAC rule next to the price, from the server's limit (freeShippingMinBRL),
// so the page confirms what the ads and the checkout say. Hidden when there is no
// free shipping or the product alone is above the PAC value limit.
export default function FreeShippingNote({ minBRL, priceCents }: { minBRL: number; priceCents: number }) {
  if (!pageShowsFreeShipping(minBRL, priceCents)) {
    return null;
  }
  return (
    <p className="mt-3 rounded-[8px] border-[3px] border-royal bg-outline px-3 py-2 text-sm text-white">
      <strong>Entrega:</strong> {freeShippingSentence(minBRL)}. As outras opções de envio continuam pagas.
    </p>
  );
}
