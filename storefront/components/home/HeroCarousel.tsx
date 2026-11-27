"use client";

// Topo da home: carrossel de slides de venda que o dono monta na tela Vitrine.
// Product slides read photo, name and prices from the catalog (lib/hero.ts).
// Native scroll-snap does the swipe on phones; the timer only scrolls the track.
// Colours follow the store's own palette (pink header, royal blue, black
// outlines with hard shadows), in two variants: "rosa" and "azul".
import Link from "next/link";
import { useCallback, useEffect, useRef, useState } from "react";
import ArtPlaceholder from "@/components/ui/ArtPlaceholder";
import { cartLineItem } from "@/lib/analytics";
import { useCart } from "@/lib/cart";
import { brl, pixelText } from "@/lib/format";
import { gameArtHue } from "@/lib/games";
import type { HeroImageSlide, HeroProductSlide, HeroSlideView } from "@/lib/hero";
import { trackAddToCartFrom, trackSelectPromotion, trackViewPromotion, type Promotion } from "@/lib/promo-analytics";
import type { HeroStyle } from "@/lib/vitrine";

const LIST_NAME = "hero_carrossel";
const INTERVAL_MS = 5500;

// Two colour variants of one layout. Text stays white with the black hard shadow
// the site uses on its logo and titles, so it reads on both backgrounds.
const THEME = {
  rosa: {
    section: "bg-brand",
    glow: "radial-gradient(80% 110% at 78% 40%, rgba(255,255,255,.22) 0%, rgba(255,255,255,0) 60%)",
    card: "bg-royal",
    tray: "bg-[#fff2f8]",
    badge: "bg-royal text-white",
    accent: "text-outline",
    buy: "bg-royal text-white",
    dotOn: "bg-royal",
    // Like the logo: a step in the card colour, then black.
    shadow: "3px 3px 0 #1355b3, 5px 5px 0 #0b0b0c",
  },
  azul: {
    section: "bg-royal",
    glow: "radial-gradient(80% 110% at 78% 40%, rgba(246,85,155,.35) 0%, rgba(246,85,155,0) 60%)",
    card: "bg-brand",
    tray: "bg-[#eef4ff]",
    badge: "bg-brand text-white",
    accent: "text-brand-soft",
    buy: "bg-brand text-white",
    dotOn: "bg-brand",
    shadow: "3px 3px 0 #e23d85, 5px 5px 0 #0b0b0c",
  },
} as const;

type Theme = (typeof THEME)[HeroStyle];

function promoOf(slide: HeroSlideView, index: number): Promotion {
  const name = slide.kind === "produto" ? slide.name : slide.image.alt;
  return { id: LIST_NAME, name, slot: String(index + 1) };
}

function shipsLabel(date: string): string {
  const [, m, d] = date.split("-");
  return `${d}/${m}`;
}

export default function HeroCarousel({ slides, variant }: { slides: HeroSlideView[]; variant: HeroStyle }) {
  const trackRef = useRef<HTMLDivElement>(null);
  const [active, setActive] = useState(0);
  const [paused, setPaused] = useState(false);
  const [reduced, setReduced] = useState(false);
  const seen = useRef(new Set<number>());
  const { add, lines } = useCart();
  const many = slides.length > 1;
  const theme = THEME[variant] ?? THEME.rosa;

  useEffect(() => {
    const mq = window.matchMedia("(prefers-reduced-motion: reduce)");
    const update = () => setReduced(mq.matches);
    update();
    mq.addEventListener("change", update);
    return () => mq.removeEventListener("change", update);
  }, []);

  const goTo = useCallback(
    (index: number) => {
      const track = trackRef.current;
      if (!track) {
        return;
      }
      const last = slides.length - 1;
      const target = index > last ? 0 : index < 0 ? last : index;
      track.scrollTo({ left: target * track.clientWidth, behavior: reduced ? "auto" : "smooth" });
    },
    [reduced, slides.length],
  );

  // Which slide is in view: from the scroll position (works for swipe too).
  useEffect(() => {
    const track = trackRef.current;
    if (!track) {
      return;
    }
    const onScroll = () => setActive(Math.round(track.scrollLeft / Math.max(1, track.clientWidth)));
    track.addEventListener("scroll", onScroll, { passive: true });
    return () => track.removeEventListener("scroll", onScroll);
  }, []);

  // view_promotion once per slide shown.
  useEffect(() => {
    if (seen.current.has(active) || !slides[active]) {
      return;
    }
    seen.current.add(active);
    const s = slides[active];
    trackViewPromotion(promoOf(s, active), s.kind === "produto" ? [cartLineItem({ ...s.line, qty: 1 })] : []);
  }, [active, slides]);

  // Autoplay: paused on hover/focus, when the tab is hidden and for reduced motion.
  useEffect(() => {
    if (!many || paused || reduced) {
      return;
    }
    const id = window.setInterval(() => {
      if (document.visibilityState === "visible") {
        goTo(active + 1);
      }
    }, INTERVAL_MS);
    return () => window.clearInterval(id);
  }, [many, paused, reduced, active, goTo]);

  const onBuy = (slide: HeroProductSlide, index: number) => {
    const item = cartLineItem({ ...slide.line, qty: 1 });
    add(slide.line, 1);
    trackSelectPromotion(promoOf(slide, index), [item]);
    trackAddToCartFrom(item, LIST_NAME);
  };
  const onView = (slide: HeroSlideView, index: number) => {
    trackSelectPromotion(promoOf(slide, index), slide.kind === "produto" ? [cartLineItem({ ...slide.line, qty: 1 })] : []);
  };

  return (
    <section
      aria-roledescription="carousel"
      aria-label="Destaques da Collecta"
      className={`relative overflow-hidden border-b-[6px] border-outline ${theme.section}`}
      onMouseEnter={() => setPaused(true)}
      onMouseLeave={() => setPaused(false)}
      onFocusCapture={() => setPaused(true)}
      onBlurCapture={(e) => {
        if (!e.currentTarget.contains(e.relatedTarget as Node | null)) {
          setPaused(false);
        }
      }}
    >
      <h1 className="sr-only">Collecta TCG: loja de card games e cartinhas Pokémon em Goiânia</h1>
      <div className="pointer-events-none absolute inset-0" style={{ backgroundImage: theme.glow }} aria-hidden />
      <div
        ref={trackRef}
        className="relative flex snap-x snap-mandatory overflow-x-auto [scrollbar-width:none] [&::-webkit-scrollbar]:hidden"
        aria-live={paused || reduced ? "polite" : "off"}
      >
        {slides.map((slide, i) => (
          <div
            key={slide.key}
            role="group"
            aria-roledescription="slide"
            aria-label={`${i + 1} de ${slides.length}`}
            className="w-full shrink-0 snap-start"
          >
            {slide.kind === "produto" ? (
              <ProductSlide
                slide={slide}
                index={i}
                theme={theme}
                inCart={lines.some((l) => l.sku === slide.line.sku)}
                onBuy={onBuy}
                onView={onView}
              />
            ) : (
              <ImageSlide slide={slide} index={i} onView={onView} />
            )}
          </div>
        ))}
      </div>

      {many && (
        <div className="absolute inset-x-0 bottom-4 flex items-center justify-center gap-3">
          <button
            type="button"
            onClick={() => goTo(active - 1)}
            aria-label="Slide anterior"
            className="sticker grid h-11 w-11 place-items-center rounded-full bg-white font-pixel text-[12px] text-outline [--sh:3px]"
          >
            ‹
          </button>
          <div className="flex items-center gap-1">
            {slides.map((s, i) => (
              <button
                key={s.key}
                type="button"
                onClick={() => goTo(i)}
                aria-label={`Ir para o slide ${i + 1}`}
                aria-current={i === active ? "true" : undefined}
                className="grid h-11 w-7 place-items-center"
              >
                <span
                  className={`block h-3 rounded-full border-2 border-outline transition-all ${i === active ? `w-7 ${theme.dotOn}` : "w-3 bg-white"}`}
                />
              </button>
            ))}
          </div>
          <button
            type="button"
            onClick={() => goTo(active + 1)}
            aria-label="Próximo slide"
            className="sticker grid h-11 w-11 place-items-center rounded-full bg-white font-pixel text-[12px] text-outline [--sh:3px]"
          >
            ›
          </button>
        </div>
      )}
    </section>
  );
}

type ProductProps = {
  slide: HeroProductSlide;
  index: number;
  theme: Theme;
  inCart: boolean;
  onBuy: (slide: HeroProductSlide, index: number) => void;
  onView: (slide: HeroSlideView, index: number) => void;
};

function Badge({ text, className }: { text: string; className: string }) {
  return (
    <span className={`sticker inline-block rounded-[6px] px-2.5 py-1.5 font-pixel text-[9px] leading-none [--sh:3px] ${className}`}>
      {pixelText(text.toUpperCase())}
    </span>
  );
}

function ProductSlide({ slide, index, theme, inCart, onBuy, onView }: ProductProps) {
  return (
    <div className="mx-auto grid min-h-[540px] max-w-[1360px] items-center gap-5 px-5 pb-20 pt-6 sm:px-7 md:min-h-[470px] md:grid-cols-[1.05fr_.95fr] md:gap-10 md:pb-16">
      <div className="order-2 min-w-0 md:order-1">
        <div className="flex flex-wrap items-center gap-3">
          {slide.badge && <Badge text={slide.badge} className={theme.badge} />}
          {slide.preorderFrom && <Badge text={`Pré-venda · envio ${shipsLabel(slide.preorderFrom)}`} className="bg-white text-outline" />}
          <span className={`font-pixel text-[9px] ${theme.accent}`}>{pixelText(slide.gameLabel.toUpperCase())}</span>
        </div>
        <h2 style={{ textShadow: theme.shadow }} className="font-display mt-4 line-clamp-3 text-[28px] font-bold leading-[1.02] text-white sm:text-[42px] lg:text-[50px]">
          {slide.name}
        </h2>
        {slide.phrase && (
          <p className="mt-2 max-w-lg text-[15px] font-semibold leading-snug text-white sm:mt-3 sm:text-[18px]">{slide.phrase}</p>
        )}
        <div className="mt-3 sm:mt-5">
          <div className="flex flex-wrap items-baseline gap-x-3">
            <span style={{ textShadow: theme.shadow }} className="font-display text-[36px] font-bold leading-none text-white sm:text-[54px]">{brl(slide.price)}</span>
            <span className={`font-pixel text-[10px] ${theme.accent}`}>NO PIX</span>
          </div>
          <p className="mt-2 text-[14px] font-semibold text-white sm:text-[15px]">
            {slide.kitUnits ? `Kit com ${slide.kitUnits} · ${brl(slide.perUnit ?? 0)} cada no Pix · ` : ""}
            {slide.installment
              ? `ou ${slide.installment.count}x de ${brl(slide.installment.each)} sem juros`
              : slide.cardPrice !== slide.price
                ? `ou ${brl(slide.cardPrice)} no cartão em até ${slide.installments}x`
                : `ou em até ${slide.installments}x no cartão`}
          </p>
        </div>
        <div className="mt-5 flex flex-wrap gap-3">
          <button
            type="button"
            onClick={() => onBuy(slide, index)}
            className={`arcade-press sticker min-h-11 rounded-[10px] px-6 py-4 font-pixel text-[11px] ${theme.buy}`}
            style={{ ["--sh" as string]: "5px" }}
          >
            {inCart ? "+1 NO CARRINHO" : slide.preorderFrom ? "GARANTIR" : "COMPRAR"}
          </button>
          <Link
            href={slide.href}
            onClick={() => onView(slide, index)}
            className="arcade-press sticker min-h-11 rounded-[10px] bg-white px-6 py-4 font-pixel text-[11px] text-outline"
            style={{ ["--sh" as string]: "5px" }}
          >
            VER
          </Link>
        </div>
      </div>
      <Link
        href={slide.href}
        onClick={() => onView(slide, index)}
        aria-label={slide.name}
        tabIndex={-1}
        className={`sticker sticker-5 relative order-1 mx-auto block w-full max-w-[210px] rounded-[18px] p-3 md:order-2 md:max-w-[420px] md:p-4 ${theme.card}`}
        style={{ ["--sh" as string]: "8px" }}
      >
        <div className={`aspect-square overflow-hidden rounded-[12px] border-[3px] border-outline p-1 md:p-2 ${theme.tray}`}>
          <ArtPlaceholder
            hue={gameArtHue(slide.game)}
            label={slide.name}
            imageURL={slide.imageURL}
            alt={slide.name}
            priority={index === 0}
            fit="contain"
            blend
            sizes="(min-width: 768px) 420px, 210px"
          />
        </div>
      </Link>
    </div>
  );
}

function ImageSlide({ slide, index, onView }: { slide: HeroImageSlide; index: number; onView: (s: HeroSlideView, i: number) => void }) {
  return (
    <Link href={slide.href} onClick={() => onView(slide, index)} className="mx-auto block max-w-[1360px] px-5 pb-20 pt-6 sm:px-7">
      <picture>
        <source media="(min-width: 640px)" srcSet={slide.image.desktop} />
        {/* eslint-disable-next-line @next/next/no-img-element -- art already sized and compressed at upload */}
        <img
          src={slide.image.mobile}
          alt={slide.image.alt}
          width={1080}
          height={1080}
          loading={index === 0 ? "eager" : "lazy"}
          fetchPriority={index === 0 ? "high" : "auto"}
          className="sticker aspect-square w-full rounded-[16px] object-cover [--sh:7px] sm:aspect-[34/10]"
        />
      </picture>
    </Link>
  );
}
