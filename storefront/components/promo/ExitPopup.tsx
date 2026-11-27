"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useCallback, useEffect, useRef, useState } from "react";
import { cartLineItem } from "@/lib/analytics";
import { useCart } from "@/lib/cart";
import { brl, pixelText } from "@/lib/format";
import { GAME_IDS } from "@/lib/games";
import { trackAddToCartFrom, trackSelectPromotion, trackViewPromotion, type Promotion } from "@/lib/promo-analytics";
import type { PopupProduct } from "@/lib/vitrine";

// Rules from the owner (27/11/2026): only on a product page and in the cart,
// never on entry, only after 20 s on the page, once per browser session, a clear
// way to close, a product and no coupon.
const DELAY_MS = 20_000;
const SESSION_KEY = "collecta:popup-saida";
const LIST_NAME = "popup_saida";

/** Product pages (/carta/x, /selado/x, /acessorio/x) and the cart; not the /selado/<jogo> listings. */
export function popupAllowedOn(pathname: string): boolean {
  if (pathname === "/carrinho") {
    return true;
  }
  if (/^\/(carta|acessorio)\/[^/]+$/.test(pathname)) {
    return true;
  }
  const sealed = pathname.match(/^\/selado\/([^/]+)$/);
  return sealed !== null && !GAME_IDS.includes(sealed[1]);
}

function alreadyShown(): boolean {
  try {
    return window.sessionStorage.getItem(SESSION_KEY) !== null;
  } catch {
    return true; // no storage (private mode): never risk showing it on every page
  }
}

function markShown(): void {
  try {
    window.sessionStorage.setItem(SESSION_KEY, "1");
  } catch {
    // ignore
  }
}

/**
 * Exit-intent pop-up with the product set on the dashboard's "Vitrine" screen.
 * Desktop: the mouse leaves through the top of the window (towards the tabs or
 * the address bar). Phone: there is no mouse, so it fires on a fast scroll back
 * up after the buyer has read down the page — the gesture of someone heading
 * back to the menu to leave. It never hijacks the back button or fires when the
 * tab regains focus: by then the buyer has already gone.
 */
export default function ExitPopup({ product }: { product: PopupProduct | null }) {
  const pathname = usePathname();
  const { lines, add } = useCart();
  const [open, setOpen] = useState(false);
  const dialogRef = useRef<HTMLDivElement>(null);
  const returnFocus = useRef<HTMLElement | null>(null);
  const inCartRef = useRef(false);
  inCartRef.current = product !== null && lines.some((l) => l.sku === product.line.sku);

  const promo: Promotion | null = product ? { id: LIST_NAME, name: product.title, slot: "popup" } : null;

  useEffect(() => {
    if (!product || !popupAllowedOn(pathname) || pathname === product.href || alreadyShown()) {
      return;
    }
    let armed = false;
    const timer = window.setTimeout(() => {
      armed = true;
    }, DELAY_MS);

    const show = () => {
      if (!armed || inCartRef.current || alreadyShown()) {
        return;
      }
      markShown();
      returnFocus.current = document.activeElement as HTMLElement | null;
      setOpen(true);
    };

    const onMouseOut = (e: MouseEvent) => {
      if (e.relatedTarget === null && e.clientY <= 0) {
        show();
      }
    };

    let lastY = window.scrollY;
    let lastT = performance.now();
    let maxY = lastY;
    const onScroll = () => {
      const y = window.scrollY;
      const t = performance.now();
      maxY = Math.max(maxY, y);
      const speed = (lastY - y) / Math.max(t - lastT, 1); // px/ms, positive going up
      if (maxY > 600 && maxY - y > 300 && speed > 1.2) {
        show();
      }
      lastY = y;
      lastT = t;
    };

    const coarse = window.matchMedia?.("(pointer: coarse)").matches ?? false;
    document.addEventListener("mouseout", onMouseOut);
    if (coarse) {
      window.addEventListener("scroll", onScroll, { passive: true });
    }
    return () => {
      window.clearTimeout(timer);
      document.removeEventListener("mouseout", onMouseOut);
      window.removeEventListener("scroll", onScroll);
    };
  }, [pathname, product]);

  const close = useCallback(() => {
    setOpen(false);
    returnFocus.current?.focus?.();
  }, []);

  useEffect(() => {
    if (!open || !product || !promo) {
      return;
    }
    trackViewPromotion(promo, [cartLineItem({ ...product.line, qty: 1 })]);
    dialogRef.current?.querySelector<HTMLElement>("[data-autofocus]")?.focus();
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        close();
      }
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
    // promo is derived from product; tracking once per opening is the point.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, product, close]);

  if (!open || !product || !promo) {
    return null;
  }

  const item = cartLineItem({ ...product.line, qty: 1 });
  const addToCart = () => {
    add(product.line, 1);
    trackSelectPromotion(promo, [item]);
    trackAddToCartFrom(item, LIST_NAME);
    close();
  };

  return (
    <div
      className="fixed inset-0 z-[60] flex items-end justify-center bg-black/60 p-4 sm:items-center"
      onClick={(e) => {
        if (e.target === e.currentTarget) {
          close();
        }
      }}
    >
      <div
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby="popup-saida-titulo"
        className="sticker animate-rise relative w-full max-w-[380px] rounded-[14px] bg-surface p-5 [--sh:6px]"
      >
        <button
          type="button"
          onClick={close}
          aria-label="Fechar"
          className="absolute right-3 top-3 grid h-8 w-8 place-items-center rounded-[8px] border-2 border-outline bg-outline font-pixel text-[10px] text-white"
        >
          X
        </button>
        <h2 id="popup-saida-titulo" className="pr-10 font-display text-2xl font-bold leading-tight text-white">
          {product.title}
        </h2>
        {product.text && <p className="mt-2 text-[14px] text-muted">{product.text}</p>}
        <div className="mt-4 flex items-center gap-3">
          {product.line.imageURL && (
            // eslint-disable-next-line @next/next/no-img-element
            <img
              src={product.line.imageURL}
              alt={product.line.name}
              className="h-20 w-16 shrink-0 rounded-[6px] border-2 border-outline object-cover"
            />
          )}
          <div className="min-w-0">
            <Link
              href={product.href}
              onClick={() => {
                trackSelectPromotion(promo, [item]);
                close();
              }}
              className="line-clamp-2 text-sm font-bold text-white underline-offset-2 hover:underline"
            >
              {product.line.name}
            </Link>
            <div className="mt-1 font-pixel text-[13px] text-brand">{brl(product.line.price)}</div>
            {product.line.cardPrice && product.line.cardPrice !== product.line.price && (
              <div className="text-[12px] text-muted">{brl(product.line.cardPrice)} no cartão</div>
            )}
          </div>
        </div>
        <div className="mt-5 flex flex-col gap-2 sm:flex-row-reverse">
          <button
            type="button"
            data-autofocus
            onClick={addToCart}
            className="arcade-press sticker flex-1 rounded-[10px] bg-brand px-4 py-3 font-pixel text-[10px] text-white [--sh:4px]"
          >
            {pixelText("Adicionar ao carrinho")}
          </button>
          <button
            type="button"
            onClick={close}
            className="flex-1 rounded-[10px] border-2 border-outline px-4 py-3 font-pixel text-[10px] text-muted"
          >
            {pixelText("Agora não")}
          </button>
        </div>
      </div>
    </div>
  );
}
