import { brl } from "@/lib/format";
import type { CartLine } from "@/lib/types";

export interface OrderCoupon {
  code: string;
  valueCents: number;
}

export function buildOrder(lines: CartLine[], total: number, coupons: OrderCoupon[] = []): string {
  const out: string[] = ["Olá! Quero fechar este pedido na Collecta:", ""];
  for (const l of lines) {
    out.push(`• ${l.name} (${l.meta}) — ${l.qty}x ${brl(l.price)}`);
  }
  out.push("");
  const discount = coupons.reduce((sum, c) => sum + c.valueCents, 0);
  if (discount > 0) {
    out.push(`Subtotal: ${brl(total)}`);
    out.push(`Cupons: ${coupons.map((c) => `${c.code} (${brl(c.valueCents)})`).join(", ")}`);
    out.push(`Total: ${brl(Math.max(total - discount, 0))}`);
  } else {
    out.push(`Total: ${brl(total)}`);
  }
  out.push("Frete: a combinar");
  return out.join("\n");
}

export function whatsappURL(text: string): string {
  const number = process.env.NEXT_PUBLIC_WHATSAPP ?? "";
  const encoded = encodeURIComponent(text);
  return number
    ? `https://wa.me/${number}?text=${encoded}`
    : `https://wa.me/?text=${encoded}`;
}
