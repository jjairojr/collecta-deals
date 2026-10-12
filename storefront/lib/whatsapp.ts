import { brl } from "@/lib/format";
import type { CartLine } from "@/lib/types";

export function buildOrder(lines: CartLine[], total: number): string {
  const out: string[] = ["Olá! Quero fechar este pedido na Collecta:", ""];
  for (const l of lines) {
    out.push(`• ${l.name} (${l.meta}) — ${l.qty}x ${brl(l.price)}`);
  }
  out.push("");
  out.push(`Total: ${brl(total)}`);
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
