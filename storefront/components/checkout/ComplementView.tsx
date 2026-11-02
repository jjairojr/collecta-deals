"use client";

import { useEffect, useRef, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import Container from "@/components/ui/Container";
import { trackWhatsAppContact } from "@/lib/analytics";
import { useCart } from "@/lib/cart";
import type { CartLine } from "@/lib/types";
import { whatsappURL } from "@/lib/whatsapp";

interface ComplementResponse {
  orderNumber?: number;
  lines?: CartLine[];
  error?: string;
}

type State = { kind: "loading" } | { kind: "expired"; orderNumber?: number } | { kind: "soldout"; orderNumber: number };

export default function ComplementView({ token }: { token: string }) {
  const { add, setQty } = useCart();
  const router = useRouter();
  const [state, setState] = useState<State>({ kind: "loading" });
  const started = useRef(false);

  useEffect(() => {
    if (started.current) {
      return;
    }
    started.current = true;
    fetch(`/api/complemento/${encodeURIComponent(token)}`, { cache: "no-store" })
      .then((res) => res.json())
      .then((data: ComplementResponse) => {
        const orderNumber = data.orderNumber;
        if (data.error || typeof orderNumber !== "number" || !data.lines) {
          setState({ kind: "expired", orderNumber });
          return;
        }
        if (data.lines.length === 0) {
          setState({ kind: "soldout", orderNumber });
          return;
        }
        for (const line of data.lines) {
          const { qty, ...rest } = line;
          add(rest, qty);
          setQty(line, qty);
        }
        router.replace(`/checkout?join=${orderNumber}`);
      })
      .catch(() => setState({ kind: "expired" }));
  }, [token, add, setQty, router]);

  if (state.kind === "loading") {
    return (
      <Container narrow className="py-16">
        <p className="text-center font-pixel text-[10px] text-brand-soft animate-blink">
          MONTANDO SEU CARRINHO...
        </p>
      </Container>
    );
  }

  const ref = state.orderNumber ? ` do pedido #${state.orderNumber}` : "";
  const title = state.kind === "soldout" ? "Os itens esgotaram" : "Esse link expirou";
  const message =
    state.kind === "soldout"
      ? "Os itens dessa sugestão já foram vendidos. Dá uma olhada no que chegou de novo na loja ou chama a gente no WhatsApp."
      : "O prazo para juntar itens a esse pedido já passou. Você ainda pode comprar normalmente na loja ou falar com a gente no WhatsApp.";
  const href = whatsappURL(`Olá! Abri o link de complemento${ref} e ${state.kind === "soldout" ? "os itens esgotaram" : "ele expirou"}. Pode me ajudar?`);

  return (
    <Container narrow className="py-16">
      <div
        className="sticker sticker-5 mx-auto flex max-w-md flex-col items-center gap-5 rounded-[18px] border-brand bg-outline px-8 py-14 text-center [--sh:8px]"
      >
        <h1 className="font-display text-4xl font-bold text-white">{title}</h1>
        <p className="text-sm text-[#c9c9d1]">{message}</p>
        <Link
          href="/"
          className="arcade-press sticker rounded-[10px] bg-brand px-5 py-3 font-pixel text-[10px] text-white [--sh:5px]"
        >
          IR PARA A LOJA
        </Link>
        <a
          href={href}
          target="_blank"
          rel="noopener noreferrer"
          onClick={() => trackWhatsAppContact("complemento")}
          className="text-xs text-muted underline-offset-4 hover:text-brand-soft hover:underline"
        >
          falar no WhatsApp
        </a>
      </div>
    </Container>
  );
}
