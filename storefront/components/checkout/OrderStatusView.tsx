"use client";

import { useEffect, useRef, useState } from "react";
import Link from "next/link";
import Container from "@/components/ui/Container";
import { trackPurchase } from "@/lib/analytics";
import { useCart } from "@/lib/cart";
import { ORDER_STORAGE_PREFIX, cents, isJoin, isPickup, type PublicOrder } from "@/lib/checkout";
import { brl, pixelText } from "@/lib/format";

const POLL_MS = 3000;
const POLL_LIMIT = 60;

type Phase = "loading" | "missing" | "ready";

export default function OrderStatusView({
  id,
  paymentID,
}: {
  id: string;
  paymentID?: string;
}) {
  const { clear } = useCart();
  const [order, setOrder] = useState<PublicOrder | null>(null);
  const [phase, setPhase] = useState<Phase>("loading");
  const [polls, setPolls] = useState(0);
  const settled = useRef(false);

  useEffect(() => {
    let cancelled = false;
    const query = paymentID && polls === 0 ? `?payment_id=${paymentID}` : "";
    fetch(`/api/pedido/${id}${query}`, { cache: "no-store" })
      .then(async (res) => {
        if (res.status === 404) {
          throw new Error("missing");
        }
        if (!res.ok) {
          throw new Error("retry");
        }
        return (await res.json()) as PublicOrder;
      })
      .then((o) => {
        if (cancelled) {
          return;
        }
        setOrder(o);
        setPhase("ready");
      })
      .catch((err: unknown) => {
        if (cancelled) {
          return;
        }
        if (err instanceof Error && err.message === "missing") {
          setPhase("missing");
        }
      });
    return () => {
      cancelled = true;
    };
  }, [id, paymentID, polls]);

  useEffect(() => {
    if (!order || order.status !== "pending" || polls >= POLL_LIMIT) {
      return;
    }
    const t = setTimeout(() => setPolls((n) => n + 1), POLL_MS);
    return () => clearTimeout(t);
  }, [order, polls]);

  useEffect(() => {
    if (!order || settled.current) {
      return;
    }
    if (order.status !== "paid" && order.status !== "shipped") {
      return;
    }
    settled.current = true;
    clear();
    const key = ORDER_STORAGE_PREFIX + order.id;
    try {
      if (localStorage.getItem(key)) {
        return;
      }
      localStorage.setItem(key, "1");
    } catch {
      return;
    }
    trackPurchase(
      order.id,
      order.lines.map((l) => ({
        name: l.name,
        kind: l.kind,
        meta: l.meta,
        qty: l.qty,
        unitBRL: l.unitBRL,
      })),
      order.totals.totalBRL,
      order.totals.shippingBRL,
    );
  }, [order, clear]);

  if (phase === "missing") {
    return (
      <Shell title="Pedido não encontrado" eyebrow="GAME OVER">
        <p className="text-sm text-muted">
          Esse link não corresponde a nenhum pedido. Se você acabou de pagar, fale
          com a gente no WhatsApp com o comprovante.
        </p>
        <Actions primary={{ href: "/", label: "VOLTAR PARA A LOJA" }} />
      </Shell>
    );
  }

  if (phase === "loading" || !order) {
    return (
      <Shell title="Conferindo seu pedido" eyebrow="LOADING">
        <p className="font-pixel text-[9px] text-brand-soft animate-blink">
          AGUARDE...
        </p>
      </Shell>
    );
  }

  const number = `#${order.number}`;

  switch (order.status) {
    case "paid":
    case "shipped":
      return (
        <Shell
          title={`Pagamento confirmado, ${order.firstName}!`}
          eyebrow={pixelText(`PEDIDO ${number} · ${order.status === "shipped" ? "ENVIADO" : "PAGO"}`)}
          tone="win"
        >
          {isPickup(order.shipping.serviceID) ? (
            <p className="text-sm text-[#c9c9d1]">
              Recebemos seu pagamento. Vamos combinar a{" "}
              <strong className="text-white">retirada em {order.shipping.company}</strong>{" "}
              pelo WhatsApp.
            </p>
          ) : isJoin(order.shipping.serviceID) ? (
            <p className="text-sm text-[#c9c9d1]">
              Recebemos seu pagamento. Estes itens vão na mesma encomenda do{" "}
              <strong className="text-white">{order.shipping.company}</strong>.
            </p>
          ) : (
            <p className="text-sm text-[#c9c9d1]">
              Recebemos seu pagamento. Vamos preparar o envio por{" "}
              <strong className="text-white">
                {order.shipping.company} {order.shipping.serviceName}
              </strong>{" "}
              e te avisamos pelo WhatsApp com o código de rastreio.
            </p>
          )}
          {order.trackingCode && (
            <p className="sticker mt-4 rounded-[10px] bg-brand-soft px-4 py-3 font-pixel text-[9px] text-outline [--sh:3px]">
              RASTREIO: {order.trackingCode}
            </p>
          )}
          <OrderLines order={order} />
          <Actions primary={{ href: "/singles", label: "CONTINUAR COMPRANDO" }} />
        </Shell>
      );
    case "pending":
      return (
        <Shell title="Aguardando o pagamento" eyebrow={pixelText(`PEDIDO ${number} · PENDENTE`)}>
          <p className="text-sm text-[#c9c9d1]">
            Ainda não recebemos a confirmação do Mercado Pago. Se você pagou com
            Pix, isso costuma levar alguns segundos. Esta página atualiza sozinha.
          </p>
          {polls >= POLL_LIMIT && (
            <p className="mt-3 text-sm text-brand">
              Está demorando mais que o normal. Se o pagamento foi feito, ele vai
              confirmar e avisamos por e-mail e WhatsApp.
            </p>
          )}
          <OrderLines order={order} />
          <Actions
            primary={order.initPoint ? { href: order.initPoint, label: "VOLTAR AO PAGAMENTO", external: true } : undefined}
            secondary={{ href: "/carrinho", label: "ver carrinho" }}
          />
        </Shell>
      );
    case "failed":
      return (
        <Shell title="O pagamento não foi aprovado" eyebrow={pixelText(`PEDIDO ${number} · TRY AGAIN`)}>
          <p className="text-sm text-[#c9c9d1]">
            O Mercado Pago recusou a tentativa. Seus itens continuam reservados
            por alguns minutos: tente outro cartão ou pague com Pix.
          </p>
          <OrderLines order={order} />
          <Actions
            primary={order.initPoint ? { href: order.initPoint, label: "TENTAR DE NOVO", external: true } : { href: "/carrinho", label: "VOLTAR AO CARRINHO" }}
          />
        </Shell>
      );
    case "expired":
    case "cancelled":
      return (
        <Shell title="Pedido expirado" eyebrow={pixelText(`PEDIDO ${number} · TIME OUT`)}>
          <p className="text-sm text-[#c9c9d1]">
            A reserva de 30 minutos acabou e os itens voltaram para a loja. Monte
            o carrinho de novo para fazer um novo pedido.
          </p>
          <Actions primary={{ href: "/carrinho", label: "VOLTAR AO CARRINHO" }} />
        </Shell>
      );
    case "oversold":
    case "refunded":
      return (
        <Shell title="Estamos conferindo seu pedido" eyebrow={pixelText(`PEDIDO ${number}`)}>
          <p className="text-sm text-[#c9c9d1]">
            Recebemos seu pagamento, mas precisamos confirmar o estoque à mão.
            Entramos em contato pelo WhatsApp em breve; se faltar algum item, o
            valor é estornado pelo mesmo meio de pagamento.
          </p>
          <OrderLines order={order} />
          <Actions primary={{ href: "/", label: "VOLTAR PARA A LOJA" }} />
        </Shell>
      );
  }
}

function Shell({
  title,
  eyebrow,
  tone = "neutral",
  children,
}: {
  title: string;
  eyebrow: string;
  tone?: "neutral" | "win";
  children: React.ReactNode;
}) {
  return (
    <Container narrow className="py-12">
      <div
        className={`sticker sticker-5 mx-auto max-w-2xl rounded-[18px] p-6 sm:p-9 ${
          tone === "win" ? "border-brand bg-outline" : "bg-surface"
        }`}
        style={{ ["--sh" as string]: "8px" }}
      >
        <div className="font-pixel text-[9px] text-brand">{eyebrow}</div>
        <h1 className="mt-3 font-display text-3xl font-bold leading-tight text-white sm:text-[40px]">
          {title}
        </h1>
        <div className="mt-4">{children}</div>
      </div>
    </Container>
  );
}

function OrderLines({ order }: { order: PublicOrder }) {
  return (
    <div className="mt-5 border-t-[3px] border-brand pt-4">
      <ul className="space-y-2 text-sm">
        {order.lines.map((l, i) => (
          <li key={`${l.name}-${i}`} className="flex items-center justify-between gap-3">
            <span className="flex min-w-0 items-center gap-3">
              <LineArt src={l.imageURL} alt={l.name} />
              <span className="min-w-0">
                <span className="block truncate text-white">
                  {l.qty}× {l.name}
                </span>
                {l.meta && <span className="block truncate text-xs text-muted">{l.meta}</span>}
              </span>
            </span>
            <span className="shrink-0 font-pixel text-[10px] text-white">
              {brl(cents(l.unitBRL * l.qty))}
            </span>
          </li>
        ))}
      </ul>
      <dl className="mt-4 space-y-2 text-sm">
        <div className="flex justify-between text-[#c9c9d1]">
          <dt>
            {isPickup(order.shipping.serviceID)
              ? "Retirada em mãos"
              : isJoin(order.shipping.serviceID)
                ? `Junto com o ${order.shipping.company}`
                : `Frete · ${order.shipping.serviceName}`}
          </dt>
          <dd>{order.totals.shippingBRL === 0 ? "Grátis" : brl(cents(order.totals.shippingBRL))}</dd>
        </div>
        {order.totals.discountBRL > 0 && (
          <div className="flex justify-between text-[#c9c9d1]">
            <dt className="truncate">
              Cupom{order.coupons && order.coupons.length > 1 ? "s" : ""}
              {order.coupons && order.coupons.length > 0 ? ` · ${order.coupons.map((c) => c.code).join(", ")}` : ""}
            </dt>
            <dd className="shrink-0 text-brand-soft">−{brl(cents(order.totals.discountBRL))}</dd>
          </div>
        )}
        <div className="flex justify-between">
          <dt className="font-pixel text-[10px] text-white">TOTAL</dt>
          <dd className="font-pixel text-[14px] text-brand">{brl(cents(order.totals.totalBRL))}</dd>
        </div>
      </dl>
    </div>
  );
}

function LineArt({ src, alt }: { src?: string; alt: string }) {
  const [failed, setFailed] = useState(false);
  if (!src || failed) {
    return <span className="block h-14 w-10 shrink-0 rounded-[6px] border-[3px] border-outline bg-royal" aria-hidden />;
  }
  return (
    <img
      src={src}
      alt={alt}
      width={40}
      height={56}
      loading="lazy"
      onError={() => setFailed(true)}
      className="h-14 w-10 shrink-0 rounded-[6px] border-[3px] border-outline object-cover"
    />
  );
}

function Actions({
  primary,
  secondary,
}: {
  primary?: { href: string; label: string; external?: boolean };
  secondary?: { href: string; label: string };
}) {
  const cls =
    "arcade-press sticker inline-block rounded-[10px] bg-brand px-5 py-3 font-pixel text-[10px] text-white [--sh:5px]";
  return (
    <div className="mt-6 flex flex-wrap items-center gap-4">
      {primary &&
        (primary.external ? (
          <a href={primary.href} className={cls}>
            {primary.label}
          </a>
        ) : (
          <Link href={primary.href} className={cls}>
            {primary.label}
          </Link>
        ))}
      {secondary && (
        <Link
          href={secondary.href}
          className="text-xs text-muted underline-offset-4 hover:text-brand-soft hover:underline"
        >
          {secondary.label}
        </Link>
      )}
    </div>
  );
}
