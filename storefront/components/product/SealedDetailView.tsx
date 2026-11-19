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
import type { AccessoryDetail, KitOffer, SealedDetail } from "@/lib/types";

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
  const bestKit = kits.reduce<KitOffer | undefined>(
    (best, k) => (!best || k.price / k.units < best.price / best.units ? k : best),
    undefined,
  );
  const bestSaving = bestKit ? savingPct(item.price, bestKit.units, bestKit.price) : 0;
  const bestSKU = bestSaving > 0 ? bestKit?.sku : undefined;
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
                  <div className="flex flex-wrap items-center gap-2">
                    <span className="font-pixel text-[10px] text-white sm:text-[11px]">
                      LEVE MAIS, PAGUE MENOS
                    </span>
                    {bestSaving > 0 && (
                      <span className="bg-brand px-2 py-1 font-pixel text-[8px] text-white">
                        ATE -{bestSaving}%
                      </span>
                    )}
                  </div>
                  <div role="radiogroup" aria-label="Quantas unidades" className="mt-3 flex flex-col gap-2.5">
                    <KitOption
                      label="1 UNIDADE"
                      units={1}
                      price={item.price}
                      fullPrice={item.price}
                      active={!kit}
                      best={false}
                      onPick={() => pick(null)}
                    />
                    {kits.map((k) => (
                      <KitOption
                        key={k.sku}
                        label={`KIT ${k.units} UNIDADES`}
                        units={k.units}
                        price={k.price}
                        fullPrice={item.price * k.units}
                        active={kit?.sku === k.sku}
                        best={k.sku === bestSKU}
                        onPick={() => pick(k.sku)}
                      />
                    ))}
                  </div>
                </div>
              )}
              <div className="flex flex-wrap items-baseline gap-x-3 gap-y-1">
                <span className="font-pixel text-[22px] text-brand sm:text-[32px]">{brl(price)}</span>
                {cardPrice !== undefined && (
                  <span className="font-pixel text-[9px] text-brand-soft">NO PIX</span>
                )}
                {kit && kit.price < item.price * kit.units && (
                  <span className="text-base text-faint line-through">{brl(item.price * kit.units)}</span>
                )}
              </div>
              {cardPrice !== undefined && (
                <p className="mt-2 text-sm text-muted">
                  ou <strong className="text-white">{brl(cardPrice)}</strong> em até {installments}x no
                  cartão
                </p>
              )}
              {kit && (
                <p className="mt-2 text-sm text-muted">
                  Kit com {kit.units} unidades: sai a{" "}
                  <strong className="text-white">{brl(Math.round(kit.price / kit.units))}</strong> cada
                  {kit.price < item.price * kit.units && (
                    <>
                      {" "}
                      e você economiza{" "}
                      <strong className="text-brand-soft">{brl(item.price * kit.units - kit.price)}</strong>
                    </>
                  )}
                  .
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
  units,
  price,
  fullPrice,
  active,
  best,
  onPick,
}: {
  label: string;
  units: number;
  price: number;
  fullPrice: number;
  active: boolean;
  best: boolean;
  onPick: () => void;
}) {
  const saving = savingPct(fullPrice, 1, price);
  return (
    <button
      type="button"
      role="radio"
      aria-checked={active}
      onClick={onPick}
      className={`flex w-full items-center gap-3 rounded-[10px] border-[3px] px-3 py-3 text-left transition-colors sm:px-4 ${
        active
          ? "border-brand bg-brand/15"
          : "border-outline bg-outline hover:border-royal-light"
      }`}
    >
      <span
        aria-hidden
        className={`flex h-5 w-5 shrink-0 items-center justify-center border-[3px] ${
          active ? "border-brand" : "border-faint"
        }`}
      >
        {active && <span className="h-2 w-2 bg-brand" />}
      </span>
      <span className="min-w-0 flex-1">
        <span className="flex flex-wrap items-center gap-2">
          <span className="font-pixel text-[9px] text-white sm:text-[10px]">{label}</span>
          {saving > 0 && (
            <span className="bg-brand-soft px-1.5 py-0.5 font-pixel text-[8px] text-outline">
              -{saving}%
            </span>
          )}
          {best && (
            <span className="bg-brand px-1.5 py-0.5 font-pixel text-[8px] text-white">
              MELHOR PRECO
            </span>
          )}
        </span>
        <span className="mt-1.5 block text-sm text-muted">
          {units > 1 ? (
            <>
              <strong className="text-white">{brl(Math.round(price / units))}</strong> cada
            </>
          ) : (
            "preço avulso"
          )}
        </span>
      </span>
      <span className="shrink-0 text-right">
        <span className="block text-lg font-bold tabular-nums text-white">{brl(price)}</span>
        {saving > 0 && (
          <span className="block text-xs tabular-nums text-faint line-through">{brl(fullPrice)}</span>
        )}
      </span>
    </button>
  );
}
