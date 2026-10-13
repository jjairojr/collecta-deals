"use client";

import Link from "next/link";
import Image from "next/image";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { User } from "lucide-react";
import { useAccount } from "@/lib/account";
import { trackSearch } from "@/lib/analytics";
import { useCart } from "@/lib/cart";
import { pixelText } from "@/lib/format";
import Container from "@/components/ui/Container";

export default function Header() {
  const { count } = useCart();
  const { status, user } = useAccount();
  const router = useRouter();
  const [q, setQ] = useState("");

  const search = (e: React.FormEvent) => {
    e.preventDefault();
    const term = q.trim();
    if (term) {
      trackSearch(term);
    }
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
            <Image
              src="/mascot.png"
              alt=""
              width={52}
              height={54}
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
          <Link
            href={status === "ready" ? "/conta" : "/conta/entrar"}
            aria-label={status === "ready" ? "Minha conta" : "Entrar"}
            className="flex items-center gap-2 text-right leading-tight"
          >
            <span className="sticker grid h-10 w-10 place-items-center rounded-[10px] bg-white text-outline [--sh:3px]">
              <User size={18} strokeWidth={2.5} aria-hidden />
            </span>
            <span className="hidden sm:block">
              <span className="block font-pixel text-[8px] text-brand-soft">
                {status === "ready" ? "PLAYER 1" : "CONTA"}
              </span>
              <span className="block max-w-[120px] truncate font-pixel text-[10px] text-white">
                {status === "ready" && user ? pixelText((user.displayName || user.username).split(" ")[0]) : "ENTRAR"}
              </span>
            </span>
          </Link>
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
