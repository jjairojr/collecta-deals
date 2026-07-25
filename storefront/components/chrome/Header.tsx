"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { useCart } from "@/lib/cart";
import Container from "@/components/ui/Container";

export default function Header() {
  const { count } = useCart();
  const router = useRouter();
  const [q, setQ] = useState("");

  const search = (e: React.FormEvent) => {
    e.preventDefault();
    const term = q.trim();
    router.push(term ? `/singles?q=${encodeURIComponent(term)}` : "/singles");
  };

  return (
    <header className="sticky top-0 z-40 border-b-[6px] border-outline bg-brand">
      <Container className="flex flex-wrap items-center gap-x-4 gap-y-3 py-3 sm:flex-nowrap sm:gap-6 sm:py-4">
        <Link
          href="/"
          className="flex shrink-0 items-center gap-3"
          aria-label="Collecta — início"
        >
          <span
            className="sticker grid place-items-center overflow-hidden rounded-[14px] bg-royal"
            style={{ ["--sh" as string]: "4px", height: 52, width: 52 }}
          >
            {/* eslint-disable-next-line @next/next/no-img-element */}
            <img
              src="/mascot.png"
              alt=""
              className="h-full w-full object-cover"
            />
          </span>
          <span className="font-display logo-shadow hidden text-[34px] leading-none text-white sm:block">
            COLLECTA
          </span>
        </Link>

        <form
          onSubmit={search}
          className="sticker order-last flex w-full min-w-0 items-stretch overflow-hidden rounded-[10px] bg-white sm:order-none sm:w-auto sm:max-w-[520px] sm:flex-1"
          style={{ ["--sh" as string]: "4px" }}
        >
          <input
            value={q}
            onChange={(e) => setQ(e.target.value)}
            placeholder="Buscar carta, set ou número…"
            aria-label="Buscar"
            className="min-w-0 flex-1 bg-transparent px-3.5 py-2.5 text-base text-outline outline-none placeholder:text-faint sm:text-sm"
          />
          <button
            type="submit"
            className="border-l-4 border-outline bg-brand-soft px-4 font-pixel text-[10px] text-outline"
          >
            BUSCAR
          </button>
        </form>

        <div className="ml-auto flex shrink-0 items-center gap-3 sm:gap-5">
          <div className="hidden text-right leading-tight sm:block">
            <div className="font-pixel text-[8px] text-brand-soft">SALDO</div>
            <div className="font-pixel text-[10px] text-white">R$ 0,00</div>
          </div>
          <Link
            href="/carrinho"
            className="arcade-press sticker relative flex items-center gap-2 rounded-[10px] bg-royal px-3.5 py-2.5 font-pixel text-[10px] text-white"
            style={{ ["--sh" as string]: "4px" }}
          >
            CARRINHO
            <span className="grid h-5 min-w-5 place-items-center rounded-full border-2 border-outline bg-brand-soft px-1 font-pixel text-[9px] text-outline">
              {count}
            </span>
          </Link>
        </div>
      </Container>
    </header>
  );
}
