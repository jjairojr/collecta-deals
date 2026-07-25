"use client";

import { useState } from "react";
import Container from "@/components/ui/Container";
import Breadcrumb from "@/components/ui/Breadcrumb";
import ArtPlaceholder from "@/components/ui/ArtPlaceholder";
import Stepper from "@/components/ui/Stepper";
import { useCart } from "@/lib/cart";
import { brl, pixelText } from "@/lib/format";
import { gamePixel } from "@/lib/games";
import type { SealedDetail } from "@/lib/types";

const THUMBS = ["LACRE", "LATERAL", "CODIGO"];

export default function SealedDetailView({ item }: { item: SealedDetail }) {
  const { add } = useCart();
  const [qty, setQty] = useState(1);

  const preorder = item.badge === "PRE-VENDA";
  const soldPct = Math.round(
    ((item.stockTotal - item.stockLeft) / item.stockTotal) * 100,
  );

  const addToCart = () =>
    add(
      {
        slug: item.slug,
        kind: "sealed",
        name: item.name,
        meta: `${item.set} · ${item.meta}`,
        seller: "Collecta Oficial",
        price: item.price,
        stock: item.stockLeft,
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
            href: `/singles?jogo=${item.game}`,
          },
          { label: "SELADOS", href: "/selado" },
          { label: pixelText(item.name) },
        ]}
      />

      <div className="mt-6 grid gap-10 lg:grid-cols-2">
        {/* Left — gallery */}
        <div>
          <div
            className="sticker sticker-5 rounded-[18px] bg-brand-soft p-6 sm:p-[26px]"
            style={{ ["--sh" as string]: "8px" }}
          >
            <div className="relative aspect-[4/3] overflow-hidden rounded-[8px] border-4 border-outline">
              <ArtPlaceholder
                hue="royal"
                angle="45"
                label={"FOTO DA CAIXA\nLACRE VISIVEL"}
                imageURL={item.imageURL}
                alt={item.name}
              />
            </div>
          </div>
          <div className="mt-3 grid grid-cols-3 gap-3">
            {THUMBS.map((label) => (
              <div
                key={label}
                className="grid aspect-square place-items-center rounded-[10px] border-[3px] border-outline bg-white font-pixel text-[7px] text-outline"
              >
                {label}
              </div>
            ))}
          </div>
        </div>

        {/* Right — buy */}
        <div>
          <div className="flex flex-wrap gap-2">
            {preorder && (
              <span className="bg-brand px-2.5 py-1.5 font-pixel text-[8px] text-white">
                PRE-VENDA
              </span>
            )}
            <span className="bg-brand-soft px-2.5 py-1.5 font-pixel text-[8px] text-outline">
              LACRADO
            </span>
          </div>

          <h1 className="font-display mt-4 text-4xl font-bold leading-none text-white sm:text-[50px]">
            {item.name}
          </h1>
          <p className="mt-3 text-base text-muted">
            {item.set} · {item.meta}
          </p>

          {/* Buy box */}
          <div
            className="sticker mt-6 rounded-[14px] bg-surface p-6"
            style={{ ["--sh" as string]: "6px" }}
          >
            <span className="font-pixel text-[28px] text-brand sm:text-[32px]">
              {brl(item.price)}
            </span>
            <p className="mt-2 text-[13px] text-muted">{item.installmentsNote}</p>

            <div className="mt-4 font-pixel text-[9px] text-brand-soft">
              {soldPct > 0
                ? `RESTAM ${item.stockLeft} DE ${item.stockTotal} CAIXAS`
                : `RESTAM ${item.stockLeft} CAIXAS`}
            </div>
            {soldPct > 0 && (
              <div className="mt-2 h-[18px] overflow-hidden rounded-[10px] border-[3px] border-brand bg-outline">
                <div
                  className="h-full"
                  style={{
                    width: `${soldPct}%`,
                    backgroundImage:
                      "repeating-linear-gradient(90deg,#F6559B 0 10px,#FDC4E5 10px 20px)",
                  }}
                />
              </div>
            )}

            <div className="mt-5 flex flex-wrap items-stretch gap-3">
              <Stepper value={qty} onChange={setQty} max={item.stockLeft} />
              <button
                type="button"
                onClick={addToCart}
                className="arcade-press sticker min-w-[220px] flex-1 rounded-[10px] bg-royal px-6 py-3.5 font-pixel text-[11px] text-white"
                style={{ ["--sh" as string]: "6px" }}
              >
                {preorder ? "GARANTIR PRE-VENDA" : "ADICIONAR AO CARRINHO"}
              </button>
            </div>
          </div>

          {/* Specs */}
          <div className="mt-6 grid grid-cols-1 gap-3.5 sm:grid-cols-2">
            {item.specs.map((s) => (
              <div
                key={s.key}
                className="rounded-[10px] border-[3px] border-royal bg-outline p-4"
              >
                <div className="font-pixel text-[8px] text-brand-soft">
                  {s.key}
                </div>
                <div className="mt-1.5 text-[15px] font-bold text-white">
                  {s.value}
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </Container>
  );
}
