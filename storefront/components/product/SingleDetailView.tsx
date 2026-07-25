"use client";

import { useState } from "react";
import Image from "next/image";
import { Heart } from "lucide-react";
import Container from "@/components/ui/Container";
import Breadcrumb from "@/components/ui/Breadcrumb";
import ArtPlaceholder from "@/components/ui/ArtPlaceholder";
import Stepper from "@/components/ui/Stepper";
import { useCart } from "@/lib/cart";
import { brl, pixelText } from "@/lib/format";
import { gameArtHue, gamePixel } from "@/lib/games";
import type { SingleDetail } from "@/lib/types";

export default function SingleDetailView({ item }: { item: SingleDetail }) {
  const { add } = useCart();
  const [qty, setQty] = useState(1);
  const [wished, setWished] = useState(false);

  const meta = [item.set, item.number, item.rarity].filter(Boolean).join(" · ");

  const addMain = () =>
    add(
      {
        slug: item.slug,
        kind: "single",
        name: item.name,
        meta: `${item.set} · ${item.number} · ${item.condition}`,
        seller: item.seller.name,
        price: item.price,
        stock: item.qty,
        imageURL: item.imageURL,
      },
      qty,
    );

  return (
    <Container className="py-9">
      <Breadcrumb
        items={[
          {
            label: gamePixel(item.game, item.gameLabel),
            href: `/singles/${item.game}`,
          },
          { label: pixelText(item.set) },
          { label: pixelText(item.name) },
        ]}
      />

      <div className="mt-6 grid gap-10 lg:grid-cols-[.9fr_1.1fr]">
        {/* Left — gallery */}
        <div className="min-w-0">
          <div
            className="sticker sticker-5 rounded-[18px] bg-brand p-6"
            style={{ ["--sh" as string]: "8px" }}
          >
            <div className="relative aspect-[5/7] overflow-hidden rounded-[8px] border-4 border-outline">
              <ArtPlaceholder
                hue={gameArtHue(item.game)}
                angle="90"
                label={
                  item.grade
                    ? "FOTO REAL DA CARTA\nSLAB PSA"
                    : "FOTO REAL DA CARTA"
                }
                imageURL={item.imageURL}
                alt={item.name}
                sizes="(min-width: 1024px) 40vw, 90vw"
                priority
              />
            </div>
          </div>
        </div>

        {/* Right — buy */}
        <div className="min-w-0">
          <div className="flex flex-wrap gap-2">
            {item.grade && (
              <Badge className="bg-brand-soft text-outline">
                {item.grade} GEM MT
              </Badge>
            )}
            {item.language && (
              <Badge className="bg-royal text-white">{item.language}</Badge>
            )}
            {item.qty === 1 && (
              <Badge className="border-2 border-brand bg-outline text-brand">
                SO 1 UNIDADE
              </Badge>
            )}
          </div>

          <h1 className="font-display mt-4 text-4xl font-bold leading-none text-white sm:text-[54px]">
            {item.name}
          </h1>
          <p className="mt-3 text-base text-muted">{meta}</p>

          {/* Buy box */}
          <div
            className="sticker mt-6 rounded-[14px] bg-surface p-6"
            style={{ ["--sh" as string]: "6px" }}
          >
            <div className="flex flex-wrap items-baseline gap-3">
              <span className="font-pixel text-[22px] text-brand sm:text-[34px]">
                {brl(item.price)}
              </span>
              {item.wasPrice && (
                <span className="text-sm text-faint line-through">
                  {brl(item.wasPrice)}
                </span>
              )}
            </div>
            {item.marketAvg && (
              <p className="mt-2 text-[13px] text-muted">
                média de mercado {brl(item.marketAvg)}
              </p>
            )}

            <div className="mt-5 flex flex-wrap items-stretch gap-3">
              <Stepper value={qty} onChange={setQty} max={item.qty} />
              <button
                type="button"
                onClick={addMain}
                className="arcade-press sticker order-3 min-w-[220px] flex-1 rounded-[10px] bg-brand px-6 py-3.5 font-pixel text-[11px] text-white [--sh:6px] sm:order-2"
              >
                ADICIONAR AO CARRINHO
              </button>
              <button
                type="button"
                onClick={() => setWished((w) => !w)}
                aria-label="Adicionar à lista de desejos"
                aria-pressed={wished}
                className="order-2 grid w-14 place-items-center rounded-[10px] border-4 border-brand-soft py-2.5 text-brand-soft sm:order-3"
              >
                <Heart className={`h-5 w-5 ${wished ? "fill-brand-soft" : ""}`} />
              </button>
            </div>
          </div>

          {/* Seller bar */}
          <div
            className="sticker mt-5 flex items-center gap-4 rounded-[14px] bg-royal px-5 py-4"
            style={{ ["--sh" as string]: "6px" }}
          >
            <Image
              src="/mascot.png"
              alt=""
              width={48}
              height={50}
              className="h-12 w-12 shrink-0 rounded-full border-2 border-outline object-cover"
            />
            <div className="min-w-0">
              <div className="font-display text-xl font-bold leading-none text-white">
                {item.seller.name}
              </div>
              <div className="mt-1 font-pixel text-[9px] text-brand-soft">
                CARTAS CONFERIDAS UMA A UMA
              </div>
            </div>
          </div>

          <p className="mt-6 text-[14px] leading-relaxed text-muted">
            {item.name} ({item.number}) é uma carta de {item.gameLabel} do set{" "}
            {item.set}, em condição {item.condition}, conferida uma a uma antes
            do envio. Adicione ao carrinho, finalize o pedido pelo WhatsApp e
            receba em todo o Brasil.
          </p>
        </div>
      </div>
    </Container>
  );
}

function Badge({
  className,
  children,
}: {
  className?: string;
  children: React.ReactNode;
}) {
  return (
    <span className={`px-2.5 py-1.5 font-pixel text-[8px] ${className}`}>
      {children}
    </span>
  );
}
