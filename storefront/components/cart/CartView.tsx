"use client";

import { useEffect, useRef, useState } from "react";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import Container from "@/components/ui/Container";
import ArtPlaceholder from "@/components/ui/ArtPlaceholder";
import Stepper from "@/components/ui/Stepper";
import {
  trackBeginCheckout,
  trackGenerateLead,
  trackRemoveFromCart,
  trackViewCart,
} from "@/lib/analytics";
import { useAccount } from "@/lib/account";
import { useCart } from "@/lib/cart";
import { brl } from "@/lib/format";
import type { CartLine } from "@/lib/types";
import { buildOrder, whatsappURL } from "@/lib/whatsapp";

const HANDOFF_EXPIRED = "O link do app expirou. Volte ao app e toque em Pagar no site de novo.";

function handoffNotice(count: number, missing: number): string {
  const loaded = count === 0 ? "Nenhum item do app está disponível." : `Carrinho do app carregado: ${count} ${count === 1 ? "item" : "itens"}.`;
  if (missing === 0) {
    return loaded;
  }
  return `${loaded} ${missing} ${missing === 1 ? "item esgotou" : "itens esgotaram"} e ficou de fora.`;
}

export default function CartView() {
  const { lines, total, setQty, remove, clear, add } = useCart();
  const { refresh } = useAccount();
  const router = useRouter();
  const params = useSearchParams();
  const fromApp = params.get("origem") === "app";
  const handoffFailed = params.get("erro") === "1";
  const [notice, setNotice] = useState<string | null>(null);
  const viewed = useRef(false);
  const imported = useRef(false);

  useEffect(() => {
    if (!fromApp || imported.current) {
      return;
    }
    imported.current = true;
    if (handoffFailed) {
      setNotice(HANDOFF_EXPIRED);
      router.replace("/carrinho");
      return;
    }
    fetch("/api/carrinho/app/linhas", { cache: "no-store" })
      .then((res) => res.json())
      .then((data: { lines: CartLine[]; missing: number }) => {
        clear();
        for (const line of data.lines) {
          const { qty, ...rest } = line;
          add(rest, qty);
        }
        setNotice(handoffNotice(data.lines.length, data.missing));
      })
      .catch(() => setNotice(HANDOFF_EXPIRED))
      .finally(() => {
        void refresh();
        router.replace("/carrinho");
      });
  }, [fromApp, handoffFailed, clear, add, refresh, router]);

  useEffect(() => {
    if (!viewed.current && lines.length > 0) {
      viewed.current = true;
      trackViewCart(lines);
    }
  }, [lines]);

  const banner = notice ? (
    <div
      className="sticker mb-6 rounded-[12px] bg-royal px-5 py-4 text-[15px] leading-snug text-white"
      style={{ ["--sh" as string]: "4px" }}
    >
      {notice}
    </div>
  ) : null;

  if (lines.length === 0) {
    return (
      <Container narrow className="py-16">
        {banner}
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

  const count = lines.reduce((n, l) => n + l.qty, 0);

  const whatsapp = () => {
    trackBeginCheckout(lines, total);
    trackGenerateLead(total);
    window.open(whatsappURL(buildOrder(lines, total)), "_blank", "noopener");
  };

  return (
    <Container narrow className="py-9">
      <h1 className="font-display text-4xl font-bold leading-none text-white sm:text-[52px]">
        Carrinho
      </h1>
      <div className="mt-3 font-pixel text-[9px] text-brand">
        {count} {count === 1 ? "ITEM" : "ITENS"} · 1 FICHA PARA CONTINUAR
      </div>
      {banner ? <div className="mt-6">{banner}</div> : null}

      <div className="mt-7 grid gap-7 lg:grid-cols-[1.4fr_.6fr] lg:items-start">
        {/* Line items */}
        <div className="min-w-0 space-y-4">
          {lines.map((l) => (
            <div
              key={`${l.slug}|${l.seller}|${l.meta}`}
              className="sticker grid grid-cols-[64px_1fr] items-center gap-4 rounded-[14px] bg-surface p-4 sm:grid-cols-[78px_1fr_auto_auto] sm:gap-[18px]"
              style={{ ["--sh" as string]: "5px" }}
            >
              <div className="relative aspect-[5/7] w-16 overflow-hidden rounded-[6px] border-[3px] border-outline sm:w-[78px]">
                <ArtPlaceholder
                  hue={l.kind === "single" ? "brand" : "royal"}
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

              <div className="col-span-2 flex flex-wrap items-center justify-between gap-x-4 gap-y-2 sm:col-span-1 sm:block">
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
                    onClick={() => {
                      trackRemoveFromCart(l);
                      remove(l);
                    }}
                    className="-mr-2 mt-0.5 rounded px-2 py-1.5 text-xs text-faint hover:text-brand"
                  >
                    remover
                  </button>
                </div>
              </div>
            </div>
          ))}

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
              Pague com{" "}
              <strong className="font-bold text-brand-soft">
                Pix ou cartão
              </strong>{" "}
              pelo Mercado Pago. O frete é calculado pelo seu CEP na próxima
              etapa, e enviamos pelos Correios com código de rastreio.
            </p>
          </div>
        </div>

        {/* Summary */}
        <div
          className="sticker sticker-5 min-w-0 rounded-[16px] border-brand bg-outline p-5 sm:p-[26px]"
          style={{ ["--sh" as string]: "8px" }}
        >
          <div className="font-pixel text-[11px] text-brand-soft">RESUMO</div>

          <dl className="mt-5 space-y-3 text-[15px]">
            <Row label="Subtotal" value={brl(total)} />
            <Row label="Frete" value="Calculado no checkout" />
          </dl>

          <div className="my-5 border-t-[3px] border-brand" />

          <div className="flex flex-wrap items-center justify-between gap-x-3 gap-y-1">
            <span className="font-pixel text-[10px] text-white">TOTAL</span>
            <span className="font-pixel text-[18px] text-brand sm:text-[22px]">
              {brl(total)}
            </span>
          </div>
          <p className="mt-2 text-xs text-muted">
            sem o frete · Pix ou cartão em até 12x
          </p>

          <Link
            href="/checkout"
            onClick={() => trackBeginCheckout(lines, total)}
            className="mt-5 block w-full rounded-[10px] border-4 border-white bg-brand px-6 py-3.5 text-center font-pixel text-[11px] text-white transition-transform hover:-translate-y-0.5"
            style={{ boxShadow: "0 0 0 4px #0b0b0c" }}
          >
            FINALIZAR COMPRA
          </Link>
          <button
            type="button"
            onClick={whatsapp}
            className="mt-4 block w-full rounded px-2 py-2 text-center text-xs text-muted underline-offset-4 hover:text-brand-soft hover:underline"
          >
            ou feche o pedido pelo WhatsApp
          </button>
          <p className="mt-3 text-center font-pixel text-[8px] leading-relaxed text-faint">
            PAGAMENTO SEGURO VIA MERCADO PAGO
            <br />
            FRETE CALCULADO PELO SEU CEP
          </p>
        </div>
      </div>
    </Container>
  );
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between">
      <dt className="text-[#c9c9d1]">{label}</dt>
      <dd className="text-white">{value}</dd>
    </div>
  );
}
