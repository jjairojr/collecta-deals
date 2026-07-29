"use client";

import { useEffect, useState } from "react";
import Container from "@/components/ui/Container";
import Breadcrumb from "@/components/ui/Breadcrumb";
import ArtPlaceholder from "@/components/ui/ArtPlaceholder";
import Stepper from "@/components/ui/Stepper";
import { sealedItem, trackAddToCart, trackViewItem } from "@/lib/analytics";
import { useCart } from "@/lib/cart";
import { brl, pixelText } from "@/lib/format";
import { gamePixel } from "@/lib/games";
import type { AccessoryDetail, SealedDetail } from "@/lib/types";

export default function SealedDetailView({
  item,
}: {
  item: SealedDetail | AccessoryDetail;
}) {
  const { add } = useCart();
  const [qty, setQty] = useState(1);

  const sealed = item.kind === "sealed";
  const stockNoun = sealed ? "CAIXAS" : "UNIDADES";
  const preorder = item.badge === "PRE-VENDA";
  const soldPct = Math.round(
    ((item.stockTotal - item.stockLeft) / item.stockTotal) * 100,
  );

  const subtitle = [...new Set([item.set, item.meta].filter(Boolean))].join(
    " · ",
  );

  useEffect(() => {
    trackViewItem(sealedItem(item));
  }, [item]);

  const addToCart = () => {
    add(
      {
        slug: item.slug,
        kind: item.kind,
        name: item.name,
        meta: subtitle,
        seller: "Collecta Oficial",
        price: item.price,
        stock: item.stockLeft,
        imageURL: item.imageURL,
      },
      qty,
    );
    trackAddToCart(sealedItem(item, qty));
  };

  return (
    <Container className="py-9">
      <Breadcrumb
        items={
          sealed
            ? [
                {
                  label: gamePixel(item.game, item.gameLabel),
                  href: `/singles/${item.game}`,
                },
                { label: "SELADOS", href: `/selado/${item.game}` },
                { label: pixelText(item.name) },
              ]
            : [
                { label: "ACESSORIOS", href: "/acessorios" },
                { label: pixelText(item.name) },
              ]
        }
      />

      <div className="mt-6 grid gap-10 lg:grid-cols-2">
        {/* Left — gallery */}
        <div className="min-w-0">
          <div
            className="sticker sticker-5 rounded-[18px] bg-brand-soft p-6 sm:p-[26px]"
            style={{ ["--sh" as string]: "8px" }}
          >
            <div className="relative aspect-[5/7] overflow-hidden rounded-[8px] border-4 border-outline">
              <ArtPlaceholder
                hue="royal"
                angle="45"
                label={sealed ? "FOTO DA CAIXA\nLACRE VISIVEL" : "FOTO DO PRODUTO"}
                imageURL={item.imageURL}
                alt={item.name}
                priority
              />
            </div>
          </div>
        </div>

        {/* Right — buy */}
        <div className="min-w-0">
          <div className="flex flex-wrap gap-2">
            {preorder && (
              <span className="bg-brand px-2.5 py-1.5 font-pixel text-[8px] text-white">
                PRE-VENDA
              </span>
            )}
            <span className="bg-brand-soft px-2.5 py-1.5 font-pixel text-[8px] text-outline">
              {sealed ? "LACRADO" : "ACESSORIO"}
            </span>
          </div>

          <h1 className="font-display mt-4 text-4xl font-bold leading-none text-white sm:text-[50px]">
            {item.name}
          </h1>
          <p className="mt-3 text-base text-muted">{subtitle}</p>

          {/* Buy box */}
          <div
            className="sticker mt-6 rounded-[14px] bg-surface p-6"
            style={{ ["--sh" as string]: "6px" }}
          >
            <span className="font-pixel text-[22px] text-brand sm:text-[32px]">
              {brl(item.price)}
            </span>
            <div className="mt-4 font-pixel text-[9px] text-brand-soft">
              {soldPct > 0
                ? `RESTAM ${item.stockLeft} DE ${item.stockTotal} ${stockNoun}`
                : `RESTAM ${item.stockLeft} ${stockNoun}`}
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

          <p className="mt-6 text-[14px] leading-relaxed text-muted">
            {sealed ? (
              <>
                {item.name} é um produto selado de {item.gameLabel}
                {item.language ? ` em ${item.language.toLowerCase()}` : ""}, com
                lacre original de fábrica conferido antes do envio. Adicione ao
                carrinho, finalize o pedido pelo WhatsApp e receba em todo o
                Brasil.
              </>
            ) : (
              <>
                {item.name} é um acessório para TCG, novo e conferido antes do
                envio — serve para qualquer jogo da loja. Adicione ao carrinho,
                finalize o pedido pelo WhatsApp e receba em todo o Brasil.
              </>
            )}
          </p>
        </div>
      </div>
    </Container>
  );
}
