"use client";

import { useState } from "react";
import Link from "next/link";
import Container from "@/components/ui/Container";
import ArtPlaceholder from "@/components/ui/ArtPlaceholder";
import Stepper from "@/components/ui/Stepper";
import { useCart } from "@/lib/cart";
import { brl, installments } from "@/lib/format";
import { COUPON, FREE_SHIPPING, SHIPPING } from "@/lib/mock";
import type { CartLine } from "@/lib/types";

export default function CartView() {
  const { lines, total: subtotal, setQty, remove } = useCart();
  const [coupon, setCoupon] = useState("");
  const [applied, setApplied] = useState(false);

  if (lines.length === 0) {
    return (
      <Container narrow className="py-16">
        <div
          className="sticker sticker-5 mx-auto flex max-w-md flex-col items-center gap-5 rounded-[18px] border-brand bg-outline px-8 py-14 text-center"
          style={{ ["--sh" as string]: "8px" }}
        >
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img
            src="/mascot.png"
            alt=""
            className="h-24 w-24 rounded-full border-4 border-outline object-cover"
          />
          <h1 className="font-display text-4xl font-bold text-white">
            Carrinho vazio
          </h1>
          <p className="font-pixel text-[10px] leading-relaxed text-brand-soft">
            INSERT COIN - ESCOLHA SUAS CARTAS
          </p>
          <Link
            href="/singles"
            className="arcade-press sticker rounded-[10px] bg-brand px-5 py-3 font-pixel text-[10px] text-white"
            style={{ ["--sh" as string]: "5px" }}
          >
            EXPLORAR SINGLES
          </Link>
        </div>
      </Container>
    );
  }

  const freeShipping = subtotal >= FREE_SHIPPING;
  const shipping = freeShipping ? 0 : SHIPPING;
  const discount = applied ? Math.round((subtotal * COUPON.pct) / 100) : 0;
  const total = subtotal + shipping - discount;
  const missing = FREE_SHIPPING - subtotal;
  const count = lines.reduce((n, l) => n + l.qty, 0);

  const applyCoupon = () => {
    if (coupon.trim().toUpperCase() === COUPON.code) {
      setApplied(true);
    }
  };

  const checkout = () => {
    const number = process.env.NEXT_PUBLIC_WHATSAPP ?? "";
    const text = buildOrder(lines, total, applied ? COUPON.code : null);
    const url = number
      ? `https://wa.me/${number}?text=${encodeURIComponent(text)}`
      : `https://wa.me/?text=${encodeURIComponent(text)}`;
    window.open(url, "_blank", "noopener");
  };

  return (
    <Container narrow className="py-9">
      <h1 className="font-display text-4xl font-bold leading-none text-white sm:text-[52px]">
        Carrinho
      </h1>
      <div className="mt-3 font-pixel text-[9px] text-brand">
        {count} {count === 1 ? "ITEM" : "ITENS"} · 1 FICHA PARA CONTINUAR
      </div>

      <div className="mt-7 grid gap-7 lg:grid-cols-[1.4fr_.6fr] lg:items-start">
        {/* Line items */}
        <div className="space-y-4">
          {lines.map((l) => (
            <div
              key={`${l.slug}|${l.seller}|${l.meta}`}
              className="sticker grid grid-cols-[64px_1fr] items-center gap-4 rounded-[14px] bg-surface p-4 sm:grid-cols-[78px_1fr_auto_auto] sm:gap-[18px]"
              style={{ ["--sh" as string]: "5px" }}
            >
              <div className="relative aspect-[5/7] w-16 overflow-hidden rounded-[6px] border-[3px] border-outline sm:w-[78px]">
                <ArtPlaceholder
                  hue={l.kind === "sealed" ? "royal" : "brand"}
                  angle="90"
                  label=""
                  imageURL={l.imageURL}
                  alt={l.name}
                />
              </div>

              <div className="min-w-0">
                <div className="font-display truncate text-lg font-bold leading-tight text-white sm:text-[21px]">
                  {l.name}
                </div>
                <div className="truncate text-xs text-muted">{l.meta}</div>
                <div className="mt-1 truncate font-pixel text-[8px] text-brand-soft">
                  {l.seller.toUpperCase()}
                </div>
              </div>

              <div className="col-span-2 flex items-center justify-between gap-4 sm:col-span-1 sm:block">
                <Stepper
                  value={l.qty}
                  onChange={(n) => setQty(l, n)}
                  max={l.stock ?? 99}
                  size="sm"
                />
                <div className="text-right sm:mt-0">
                  <div className="font-pixel text-[13px] text-brand">
                    {brl(l.price * l.qty)}
                  </div>
                  <button
                    type="button"
                    onClick={() => remove(l)}
                    className="-mr-2 mt-0.5 rounded px-2 py-1.5 text-xs text-faint hover:text-brand"
                  >
                    remover
                  </button>
                </div>
              </div>
            </div>
          ))}

          {!freeShipping && (
            <div
              className="sticker flex items-center gap-4 rounded-[14px] bg-royal px-5 py-4"
              style={{ ["--sh" as string]: "5px" }}
            >
              {/* eslint-disable-next-line @next/next/no-img-element */}
              <img
                src="/mascot.png"
                alt=""
                className="h-11 w-11 shrink-0 rounded-full border-2 border-outline object-cover"
              />
              <p className="text-[15px] leading-snug text-white">
                Faltam{" "}
                <strong className="font-bold text-brand-soft">
                  {brl(missing)}
                </strong>{" "}
                para o frete grátis. Bora completar o set?
              </p>
            </div>
          )}
        </div>

        {/* Summary */}
        <div
          className="sticker sticker-5 rounded-[16px] border-brand bg-outline p-6 sm:p-[26px]"
          style={{ ["--sh" as string]: "8px" }}
        >
          <div className="font-pixel text-[11px] text-brand-soft">RESUMO</div>

          <dl className="mt-5 space-y-3 text-[15px]">
            <Row label="Subtotal" value={brl(subtotal)} />
            <Row
              label="Frete (Sedex)"
              value={freeShipping ? "Grátis" : brl(shipping)}
            />
            {applied && (
              <Row
                label={`Cupom ${COUPON.code}`}
                value={`−${brl(discount)}`}
                pink
              />
            )}
          </dl>

          {!applied && (
            <div
              className="sticker mt-4 flex items-stretch overflow-hidden rounded-[10px] bg-white"
              style={{ ["--sh" as string]: "0px", borderWidth: 3 }}
            >
              <input
                value={coupon}
                onChange={(e) => setCoupon(e.target.value)}
                placeholder="Cupom"
                aria-label="Cupom de desconto"
                className="min-w-0 flex-1 bg-transparent px-3 py-2 text-base text-outline outline-none placeholder:text-faint sm:text-sm"
              />
              <button
                type="button"
                onClick={applyCoupon}
                className="border-l-[3px] border-outline bg-brand-soft px-3 font-pixel text-[9px] text-outline"
              >
                APLICAR
              </button>
            </div>
          )}

          <div className="my-5 border-t-[3px] border-brand" />

          <div className="flex flex-wrap items-center justify-between gap-x-3 gap-y-1">
            <span className="font-pixel text-[10px] text-white">TOTAL</span>
            <span className="font-pixel text-[18px] text-brand sm:text-[22px]">
              {brl(total)}
            </span>
          </div>
          <p className="mt-2 text-xs text-muted">
            em até {installments(total, 12)}
          </p>

          <button
            type="button"
            onClick={checkout}
            className="mt-5 w-full rounded-[10px] border-4 border-white bg-brand px-6 py-3.5 font-pixel text-[11px] text-white transition-transform hover:-translate-y-0.5"
            style={{ boxShadow: "0 0 0 4px #0b0b0c" }}
          >
            INSERIR FICHA - PAGAR
          </button>
          <p className="mt-4 text-center font-pixel text-[8px] leading-relaxed text-faint">
            PIX - CARTAO - BOLETO
            <br />
            COMPRA PROTEGIDA COLLECTA
          </p>
        </div>
      </div>
    </Container>
  );
}

function Row({
  label,
  value,
  pink,
}: {
  label: string;
  value: string;
  pink?: boolean;
}) {
  return (
    <div className="flex items-center justify-between">
      <dt className="text-[#c9c9d1]">{label}</dt>
      <dd className={pink ? "font-semibold text-brand" : "text-white"}>
        {value}
      </dd>
    </div>
  );
}

// buildOrder writes the WhatsApp handoff message (checkout stays on WhatsApp per
// the product decision — coupon/shipping are shown but settled with us there).
function buildOrder(
  lines: CartLine[],
  total: number,
  coupon: string | null,
): string {
  const out: string[] = ["Olá! Quero fechar este pedido na Collecta:", ""];
  for (const l of lines) {
    out.push(`• ${l.name} (${l.meta}) — ${l.qty}x ${brl(l.price)}`);
  }
  out.push("");
  if (coupon) {
    out.push(`Cupom: ${coupon}`);
  }
  out.push(`Total: ${brl(total)}`);
  return out.join("\n");
}
