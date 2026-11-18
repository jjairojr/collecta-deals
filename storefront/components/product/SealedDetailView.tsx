"use client";

import { useEffect, useState } from "react";
import Container from "@/components/ui/Container";
import Breadcrumb from "@/components/ui/Breadcrumb";
import ArtPlaceholder from "@/components/ui/ArtPlaceholder";
import Stepper from "@/components/ui/Stepper";
import SoldOutBox from "@/components/product/SoldOutBox";
import { sealedItem, trackAddToCart, trackViewItem } from "@/lib/analytics";
import { useCart } from "@/lib/cart";
import { kitLabel, kitStock, kitToCartLine, sealedToCartLine } from "@/lib/cart-lines";
import { brl, pixelText } from "@/lib/format";
import { gamePixel } from "@/lib/games";
import { CARD_PRICED_INSTALLMENTS, MAX_INSTALLMENTS, shipsFromLabel } from "@/lib/payment";
import type { AccessoryDetail, SealedDetail } from "@/lib/types";

export default function SealedDetailView({
  item,
}: {
  item: SealedDetail | AccessoryDetail;
}) {
  const { add } = useCart();
  const [qty, setQty] = useState(1);
  const [kitSKU, setKitSKU] = useState<string | null>(null);

  const sealed = item.kind === "sealed";
  const stockNoun = sealed ? "CAIXAS" : "UNIDADES";
  const preorder = item.badge === "PRE-VENDA";
  const kits = item.soldOut ? [] : (item.kits ?? []);
  const kit = kits.find((k) => k.sku === kitSKU);
  const price = kit ? kit.price : item.price;
  const cardPrice = kit ? kit.cardPrice : item.cardPrice;
  const maxQty = kit ? kitStock(item.stockLeft, kit.units) : item.stockLeft;
  const installments = cardPrice !== undefined ? CARD_PRICED_INSTALLMENTS : MAX_INSTALLMENTS;
  const soldPct =
    item.stockTotal > 0
      ? Math.round(((item.stockTotal - item.stockLeft) / item.stockTotal) * 100)
      : 0;

  const subtitle = [...new Set([item.set, item.meta].filter(Boolean))].join(
    " · ",
  );

  useEffect(() => {
    trackViewItem(sealedItem(item));
  }, [item]);

  const pick = (sku: string | null) => {
    setKitSKU(sku);
    setQty(1);
  };

  const addToCart = () => {
    if (kit) {
      add(kitToCartLine(item, kit, item.stockLeft), qty);
      trackAddToCart({
        ...sealedItem(item, qty),
        item_name: `${item.name} · ${kitLabel(kit.units)}`,
        price: kit.price / 100,
      });
      return;
    }
    add(sealedToCartLine(item, item.stockLeft), qty);
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

          {item.soldOut ? (
            <SoldOutBox
              name={item.name}
              price={item.price}
              path={`/${sealed ? "selado" : "acessorio"}/${item.slug}`}
            />
          ) : (
            <div
              className="sticker mt-6 rounded-[14px] bg-surface p-6"
              style={{ ["--sh" as string]: "6px" }}
            >
              {kits.length > 0 && (
                <div className="mb-5">
                  <div className="font-pixel text-[9px] text-brand-soft">QUANTAS UNIDADES?</div>
                  <div className="mt-2 flex flex-wrap gap-2.5">
                    <KitOption
                      label="1 UN"
                      price={item.price}
                      active={!kit}
                      onPick={() => pick(null)}
                    />
                    {kits.map((k) => (
                      <KitOption
                        key={k.sku}
                        label={`KIT ${k.units} UN`}
                        price={k.price}
                        saving={savingPct(item.price, k.units, k.price)}
                        active={kit?.sku === k.sku}
                        onPick={() => pick(k.sku)}
                      />
                    ))}
                  </div>
                </div>
              )}
              <span className="font-pixel text-[22px] text-brand sm:text-[32px]">
                {brl(price)}
              </span>
              {cardPrice !== undefined && (
                <>
                  <span className="ml-2 font-pixel text-[9px] text-brand-soft">NO PIX</span>
                  <p className="mt-2 text-sm text-muted">
                    ou <strong className="text-white">{brl(cardPrice)}</strong> em até {installments}x no
                    cartão
                  </p>
                </>
              )}
              {kit && (
                <p className="mt-2 text-sm text-muted">
                  Kit com {kit.units} unidades: sai a{" "}
                  <strong className="text-white">{brl(Math.round(kit.price / kit.units))}</strong> cada.
                </p>
              )}
              {item.shipsFrom && (
                <p className="mt-4 rounded-[8px] border-[3px] border-brand bg-outline px-3 py-2 text-sm text-white">
                  <strong>Pré-venda:</strong> envio e retirada a partir de {shipsFromLabel(item.shipsFrom)}.
                </p>
              )}
              {!preorder && (
                <div className="mt-4 font-pixel text-[9px] text-brand-soft">
                  {soldPct > 0
                    ? `RESTAM ${item.stockLeft} DE ${item.stockTotal} ${stockNoun}`
                    : `RESTAM ${item.stockLeft} ${stockNoun}`}
                </div>
              )}
              {!preorder && soldPct > 0 && (
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
                <Stepper value={qty} onChange={setQty} max={maxQty} />
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
          )}

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
                lacre original de fábrica conferido antes do envio. Pague por
                Pix ou cartão em até {installments}x no checkout do site e receba em todo o
                Brasil, ou retire na loja em Goiânia.
              </>
            ) : (
              <>
                {item.name} é um acessório para TCG, novo e conferido antes do
                envio — serve para qualquer jogo da loja. Pague por Pix ou cartão
                em até {installments}x no checkout do site e receba em todo o Brasil, ou
                retire na loja em Goiânia.
              </>
            )}
          </p>
        </div>
      </div>
    </Container>
  );
}

function savingPct(unitPrice: number, units: number, kitPrice: number): number {
  const full = unitPrice * units;
  if (full <= 0) {
    return 0;
  }
  return Math.round(((full - kitPrice) / full) * 100);
}

function KitOption({
  label,
  price,
  saving = 0,
  active,
  onPick,
}: {
  label: string;
  price: number;
  saving?: number;
  active: boolean;
  onPick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onPick}
      aria-pressed={active}
      className={`relative min-w-[104px] rounded-[10px] border-[3px] border-outline px-3 py-2.5 text-left transition-colors ${
        active ? "bg-brand text-white" : "bg-outline text-white hover:bg-royal"
      }`}
    >
      <span className="block font-pixel text-[9px]">{label}</span>
      <span className="mt-1.5 block text-sm font-bold tabular-nums">{brl(price)}</span>
      {saving > 0 && (
        <span className="absolute -right-2 -top-2.5 rounded-[6px] border-2 border-outline bg-brand-soft px-1.5 py-0.5 font-pixel text-[7px] text-outline">
          -{saving}%
        </span>
      )}
    </button>
  );
}
