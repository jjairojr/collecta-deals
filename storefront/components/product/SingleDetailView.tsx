"use client";

import { useState } from "react";
import { Heart } from "lucide-react";
import Container from "@/components/ui/Container";
import Breadcrumb from "@/components/ui/Breadcrumb";
import ArtPlaceholder from "@/components/ui/ArtPlaceholder";
import Stepper from "@/components/ui/Stepper";
import { useCart } from "@/lib/cart";
import { brl, installments, pixelText } from "@/lib/format";
import { gameArtHue, gamePixel } from "@/lib/games";
import type { Offer, SingleDetail } from "@/lib/types";

const THUMBS = ["FRENTE", "VERSO", "CANTOS", "SELO PSA"];

export default function SingleDetailView({ item }: { item: SingleDetail }) {
  const { add } = useCart();
  const [qty, setQty] = useState(1);
  const [thumb, setThumb] = useState(0);
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

  const addOffer = (o: Offer) =>
    add({
      slug: item.slug,
      kind: "single",
      name: item.name,
      meta: `${item.set} · ${item.number} · ${o.condition}`,
      seller: o.seller.name,
      price: o.price,
      stock: item.qty,
      imageURL: item.imageURL,
    });

  return (
    <Container className="py-9">
      <Breadcrumb
        items={[
          {
            label: gamePixel(item.game, item.gameLabel),
            href: `/singles?jogo=${item.game}`,
          },
          { label: pixelText(item.set) },
          { label: pixelText(item.name) },
        ]}
      />

      <div className="mt-6 grid gap-10 lg:grid-cols-[.9fr_1.1fr]">
        {/* Left — gallery */}
        <div>
          <div
            className="sticker sticker-5 rounded-[18px] bg-brand p-6"
            style={{ ["--sh" as string]: "8px" }}
          >
            <div className="relative aspect-[5/7] overflow-hidden rounded-[8px] border-4 border-outline">
              <ArtPlaceholder
                hue={gameArtHue(item.game)}
                angle="90"
                label={`FOTO REAL DA CARTA\n${item.grade ? "SLAB PSA · " : ""}${THUMBS[thumb]}`}
                imageURL={item.imageURL}
                alt={item.name}
              />
            </div>
          </div>
          <div className="mt-3 grid grid-cols-4 gap-3">
            {THUMBS.map((label, i) => (
              <button
                key={label}
                type="button"
                onClick={() => setThumb(i)}
                aria-pressed={i === thumb}
                className={`grid aspect-square place-items-center rounded-[10px] border-[3px] bg-surface p-1 font-pixel text-[7px] leading-tight text-muted ${
                  i === thumb ? "border-brand text-white" : "border-outline"
                }`}
              >
                {label}
              </button>
            ))}
          </div>
        </div>

        {/* Right — buy */}
        <div>
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
              <span className="font-pixel text-[30px] text-brand sm:text-[34px]">
                {brl(item.price)}
              </span>
              {item.wasPrice && (
                <span className="text-sm text-faint line-through">
                  {brl(item.wasPrice)}
                </span>
              )}
            </div>
            <p className="mt-2 text-[13px] text-muted">
              ou {installments(item.price, 12)}
              {item.marketAvg
                ? ` · média de mercado ${brl(item.marketAvg)}`
                : ""}
            </p>

            <div className="mt-5 flex flex-wrap items-stretch gap-3">
              <Stepper value={qty} onChange={setQty} max={item.qty} />
              <button
                type="button"
                onClick={addMain}
                className="arcade-press sticker min-w-[220px] flex-1 rounded-[10px] bg-brand px-6 py-3.5 font-pixel text-[11px] text-white"
                style={{ ["--sh" as string]: "6px" }}
              >
                ADICIONAR AO CARRINHO
              </button>
              <button
                type="button"
                onClick={() => setWished((w) => !w)}
                aria-label="Adicionar à lista de desejos"
                aria-pressed={wished}
                className="grid w-14 place-items-center rounded-[10px] border-4 border-brand-soft text-brand-soft"
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
            {/* eslint-disable-next-line @next/next/no-img-element */}
            <img
              src="/mascot.png"
              alt=""
              className="h-12 w-12 shrink-0 rounded-full border-2 border-outline object-cover"
            />
            <div className="min-w-0">
              <div className="font-display text-xl font-bold leading-none text-white">
                {item.seller.name}
              </div>
              <div className="mt-1 font-pixel text-[9px] text-brand-soft">
                {item.seller.rating.toLocaleString("pt-BR")}% POSITIVO ·{" "}
                {item.seller.sales.toLocaleString("pt-BR")} VENDAS
              </div>
            </div>
            <span className="ml-auto shrink-0 rounded-[8px] border-2 border-outline bg-brand-soft px-2.5 py-1.5 font-pixel text-[8px] text-outline">
              {item.seller.ships}
            </span>
          </div>

          {/* Price history */}
          <div
            className="sticker mt-5 rounded-[14px] bg-surface p-5"
            style={{ ["--sh" as string]: "6px" }}
          >
            <div className="font-pixel text-[10px] text-brand-soft">
              HISTORICO DE PRECO - 6 MESES
            </div>
            <div className="mt-4 flex h-[130px] items-stretch gap-2">
              {item.priceHistory.map((p, i) => {
                const last = i === item.priceHistory.length - 1;
                return (
                  <div
                    key={p.label}
                    className="flex h-full flex-1 flex-col items-center justify-end gap-1.5"
                  >
                    <div className="flex w-full flex-1 items-end">
                      <div
                        className={`w-full rounded-t-[4px] border-[3px] border-outline ${
                          last ? "bg-brand" : "bg-brand-soft"
                        }`}
                        style={{ height: `${p.value}%` }}
                      />
                    </div>
                    <span className="font-pixel text-[7px] text-faint">
                      {p.label}
                    </span>
                  </div>
                );
              })}
            </div>
          </div>

          {/* Other sellers */}
          <div className="mt-6">
            <div className="mb-3 font-pixel text-[10px] text-brand">
              OUTROS VENDEDORES
            </div>
            <div className="space-y-2.5">
              {item.offers.map((o) => (
                <div
                  key={o.seller.id}
                  className="sticker grid grid-cols-[1fr_auto_auto] items-center gap-4 rounded-[10px] bg-surface px-4 py-3.5"
                  style={{ ["--sh" as string]: "0px", borderWidth: 3 }}
                >
                  <div className="min-w-0">
                    <div className="truncate text-[15px] font-bold text-white">
                      {o.seller.name}
                    </div>
                    <div className="truncate text-xs text-muted">
                      {o.condition} · {o.ships}
                    </div>
                  </div>
                  <span className="font-pixel text-[12px] text-brand-soft">
                    {brl(o.price)}
                  </span>
                  <button
                    type="button"
                    onClick={() => addOffer(o)}
                    aria-label={`Adicionar oferta de ${o.seller.name}`}
                    className="arcade-press sticker rounded-[8px] bg-royal px-2.5 py-1.5 font-pixel text-[9px] text-white"
                    style={{ ["--sh" as string]: "3px" }}
                  >
                    + ADD
                  </button>
                </div>
              ))}
            </div>
          </div>
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
