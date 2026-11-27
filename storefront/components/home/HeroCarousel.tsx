"use client";

// Topo da home: carrossel de slides de venda que o dono monta na tela Vitrine.
// Product slides read photo, name and prices from the catalog (lib/hero.ts).
// Native scroll-snap does the swipe on phones; the timer only scrolls the track.
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
  // "destaque" shows two cards side by side on wide screens, so its last stop is
  // one card before the end.
  const perView = useRef(1);

  useEffect(() => {
    const mq = window.matchMedia("(prefers-reduced-motion: reduce)");
    const update = () => setReduced(mq.matches);
    update();
    mq.addEventListener("change", update);
    return () => mq.removeEventListener("change", update);
  }, []);

  const goTo = useCallback((index: number) => {
    const track = trackRef.current;
    const first = track?.firstElementChild as HTMLElement | null;
    if (!track || !first) {
      return;
    }
    const width = first.offsetWidth;
    perView.current = Math.max(1, Math.round(track.clientWidth / width));
    const last = Math.max(0, slides.length - perView.current);
    const target = index > last ? 0 : index < 0 ? last : index;
    track.scrollTo({ left: target * width, behavior: reduced ? "auto" : "smooth" });
  }, [reduced, slides.length]);

  // Which slide is in view: from the scroll position (works for swipe too).
  useEffect(() => {
    const track = trackRef.current;
    if (!track) {
      return;
    }
    const onScroll = () => {
      const first = track.firstElementChild as HTMLElement | null;
      if (first) {
        setActive(Math.round(track.scrollLeft / first.offsetWidth));
      }
    };
    track.addEventListener("scroll", onScroll, { passive: true });
    return () => track.removeEventListener("scroll", onScroll);
  }, []);

  // view_promotion once per slide shown.
  useEffect(() => {
    for (let i = active; i < Math.min(slides.length, active + perView.current); i++) {
      if (!seen.current.has(i)) {
        seen.current.add(i);
        const s = slides[i];
        trackViewPromotion(promoOf(s, i), s.kind === "produto" ? [cartLineItem({ ...s.line, qty: 1 })] : []);
      }
    }
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

  const spotlight = variant === "destaque";
  return (
    <section
      aria-roledescription="carousel"
      aria-label="Destaques da Collecta"
      className={`relative overflow-hidden border-b-[6px] border-outline ${spotlight ? "bg-ink py-6 sm:py-8" : "bg-ink"}`}
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
      {!spotlight && (
        <div
          className="pointer-events-none absolute inset-0"
          style={{ backgroundImage: "radial-gradient(90% 120% at 80% 40%, rgba(19,85,179,.35) 0%, rgba(19,85,179,0) 60%), radial-gradient(70% 90% at 10% 100%, rgba(246,85,155,.18) 0%, rgba(246,85,155,0) 60%)" }}
          aria-hidden
        />
      )}
      <div className={`relative mx-auto w-full ${spotlight ? "max-w-[1360px] px-5 sm:px-7" : ""}`}>
        <div
          ref={trackRef}
          className={`flex snap-x snap-mandatory overflow-x-auto [scrollbar-width:none] [&::-webkit-scrollbar]:hidden ${spotlight ? "gap-4" : ""}`}
          aria-live={paused || reduced ? "polite" : "off"}
        >
          {slides.map((slide, i) => (
            <div
              key={slide.key}
              role="group"
              aria-roledescription="slide"
              aria-label={`${i + 1} de ${slides.length}`}
              className={`shrink-0 snap-start ${spotlight ? "w-full lg:w-[calc(50%-8px)]" : "w-full"}`}
            >
              {slide.kind === "produto" ? (
                spotlight ? (
                  <SpotlightProduct slide={slide} index={i} inCart={lines.some((l) => l.sku === slide.line.sku)} onBuy={onBuy} onView={onView} />
                ) : (
                  <ShowcaseProduct slide={slide} index={i} inCart={lines.some((l) => l.sku === slide.line.sku)} onBuy={onBuy} onView={onView} />
                )
              ) : (
                <ImageSlide slide={slide} index={i} spotlight={spotlight} onView={onView} />
              )}
            </div>
          ))}
        </div>

        {many && (
          <div className={`flex items-center justify-center gap-3 ${spotlight ? "pt-5" : "absolute inset-x-0 bottom-4"}`}>
            <button
              type="button"
              onClick={() => goTo(active - 1)}
              aria-label="Slide anterior"
              className="grid h-11 w-11 place-items-center rounded-full border-2 border-outline bg-surface font-pixel text-[12px] text-white hover:bg-brand"
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
                  <span className={`block h-2.5 rounded-full transition-all ${i === active ? "w-6 bg-brand" : "w-2.5 bg-white/40"}`} />
                </button>
              ))}
            </div>
            <button
              type="button"
              onClick={() => goTo(active + 1)}
              aria-label="Próximo slide"
              className="grid h-11 w-11 place-items-center rounded-full border-2 border-outline bg-surface font-pixel text-[12px] text-white hover:bg-brand"
            >
              ›
            </button>
          </div>
        )}
      </div>
    </section>
  );
}

type ProductProps = {
  slide: HeroProductSlide;
  index: number;
  inCart: boolean;
  onBuy: (slide: HeroProductSlide, index: number) => void;
  onView: (slide: HeroSlideView, index: number) => void;
};

function Badge({ text, tone = "brand" }: { text: string; tone?: "brand" | "soft" }) {
  return (
    <span
      className={`inline-block rounded-[6px] border-2 border-outline px-2.5 py-1.5 font-pixel text-[9px] leading-none ${tone === "brand" ? "bg-brand text-white" : "bg-brand-soft text-outline"}`}
    >
      {pixelText(text.toUpperCase())}
    </span>
  );
}

function PriceBlock({ slide, big, onColor = false }: { slide: HeroProductSlide; big: boolean; onColor?: boolean }) {
  return (
    <div>
      <div className="flex flex-wrap items-baseline gap-x-2">
        <span className={`font-display font-bold leading-none text-white ${big ? "text-[34px] sm:text-[52px]" : "text-[30px] sm:text-[34px]"}`}>
          {brl(slide.price)}
        </span>
        <span className={`font-pixel text-[9px] ${onColor ? "text-white" : "text-brand-soft"}`}>NO PIX</span>
      </div>
      <p className={`mt-2 text-[14px] ${onColor ? "font-semibold text-white" : "text-muted"}`}>
        {slide.kitUnits ? `Kit com ${slide.kitUnits} · ${brl(slide.perUnit ?? 0)} cada no Pix · ` : ""}
        {slide.installment
          ? `ou ${slide.installment.count}x de ${brl(slide.installment.each)} sem juros`
          : slide.cardPrice !== slide.price
            ? `ou ${brl(slide.cardPrice)} no cartão em até ${slide.installments}x`
            : `ou em até ${slide.installments}x no cartão`}
      </p>
    </div>
  );
}

function Actions({ slide, index, inCart, onBuy, onView, compact }: ProductProps & { compact?: boolean }) {
  const pad = compact ? "px-4 py-3 text-[10px]" : "px-6 py-4 text-[11px]";
  return (
    <div className="mt-5 flex flex-wrap gap-3">
      <button
        type="button"
        onClick={() => onBuy(slide, index)}
        className={`arcade-press sticker min-h-11 rounded-[10px] bg-brand font-pixel text-white ${pad}`}
        style={{ ["--sh" as string]: "5px" }}
      >
        {inCart ? "+1 NO CARRINHO" : slide.preorderFrom ? "GARANTIR" : "COMPRAR"}
      </button>
      <Link
        href={slide.href}
        onClick={() => onView(slide, index)}
        className={`min-h-11 rounded-[10px] border-2 border-white/70 font-pixel text-white hover:bg-white hover:text-outline ${pad}`}
      >
        VER
      </Link>
    </div>
  );
}

// Estilo "vitrine": um produto por vez, grande e nítido, fundo limpo.
function ShowcaseProduct(props: ProductProps) {
  const { slide, index } = props;
  return (
    <div className="mx-auto grid min-h-[520px] max-w-[1360px] items-center gap-4 px-5 pb-20 pt-5 sm:px-7 md:min-h-[460px] md:grid-cols-[1fr_1fr] md:gap-10 md:pb-16 lg:min-h-[480px]">
      <div className="order-2 min-w-0 md:order-1">
        <div className="flex flex-wrap items-center gap-2">
          {slide.badge && <Badge text={slide.badge} />}
          {slide.preorderFrom && <Badge text={`Pré-venda · envio ${shipsLabel(slide.preorderFrom)}`} tone="soft" />}
          <span className="font-pixel text-[9px] text-muted">{pixelText(slide.gameLabel.toUpperCase())}</span>
        </div>
        <h2 className="font-display mt-3 line-clamp-3 text-[26px] font-bold leading-[1.02] text-white sm:text-[40px] lg:text-[48px]">
          {slide.name}
        </h2>
        {slide.phrase && <p className="mt-2 max-w-lg text-[15px] leading-snug text-brand-soft sm:mt-3 sm:text-[18px] sm:leading-relaxed">{slide.phrase}</p>}
        <div className="mt-3 sm:mt-5">
          <PriceBlock slide={slide} big />
        </div>
        <Actions {...props} />
      </div>
      <Link
        href={slide.href}
        onClick={() => props.onView(slide, index)}
        className="order-1 mx-auto block aspect-square w-full max-w-[180px] md:order-2 md:max-w-[400px]"
        aria-label={slide.name}
        tabIndex={-1}
      >
        <ArtPlaceholder
          hue={gameArtHue(slide.game)}
          label={slide.name}
          imageURL={slide.imageURL}
          alt={slide.name}
          priority={index === 0}
          fit="contain"
          sizes="(min-width: 768px) 400px, 180px"
        />
      </Link>
    </div>
  );
}

const SPOT_BG: Record<string, string> = {
  brand: "linear-gradient(135deg, #f6559b 0%, #e23d85 100%)",
  royal: "linear-gradient(135deg, #2f74d8 0%, #1355b3 100%)",
  soft: "linear-gradient(135deg, #fdc4e5 0%, #f6559b 100%)",
};

// Estilo "destaque": cards compactos com a cor do jogo, dois por vez no computador.
function SpotlightProduct(props: ProductProps) {
  const { slide, index } = props;
  return (
    <div
      className="sticker sticker-5 grid min-h-[300px] grid-cols-[42%_1fr] items-center gap-4 overflow-hidden rounded-[18px] p-4 sm:min-h-[320px] sm:gap-6 sm:p-6"
      style={{ ["--sh" as string]: "7px", backgroundImage: SPOT_BG[gameArtHue(slide.game)] ?? SPOT_BG.royal }}
    >
      <Link
        href={slide.href}
        onClick={() => props.onView(slide, index)}
        className="block aspect-[4/5] w-full rounded-[12px] bg-white/95 p-2"
        aria-label={slide.name}
        tabIndex={-1}
      >
        <ArtPlaceholder
          hue={gameArtHue(slide.game)}
          label={slide.name}
          imageURL={slide.imageURL}
          alt={slide.name}
          priority={index === 0}
          fit="contain"
          sizes="(min-width: 1024px) 260px, 40vw"
        />
      </Link>
      <div className="min-w-0">
        <div className="flex flex-wrap items-center gap-2">
          {slide.badge && <Badge text={slide.badge} tone="soft" />}
          {slide.preorderFrom && <Badge text={`Envio ${shipsLabel(slide.preorderFrom)}`} tone="soft" />}
        </div>
        <h2 className="font-display mt-3 line-clamp-3 text-[20px] font-bold leading-[1.05] text-white sm:text-[26px]">{slide.name}</h2>
        {slide.phrase && <p className="mt-2 line-clamp-2 text-[14px] leading-snug text-white/90">{slide.phrase}</p>}
        <div className="mt-3">
          <PriceBlock slide={slide} big={false} onColor />
        </div>
        <Actions {...props} compact />
      </div>
    </div>
  );
}

function ImageSlide({ slide, index, spotlight, onView }: { slide: HeroImageSlide; index: number; spotlight: boolean; onView: (s: HeroSlideView, i: number) => void }) {
  return (
    <Link
      href={slide.href}
      onClick={() => onView(slide, index)}
      className={`block ${spotlight ? "sticker sticker-5 overflow-hidden rounded-[18px]" : "mx-auto max-w-[1360px] px-5 pb-16 pt-6 sm:px-7"}`}
      style={spotlight ? { ["--sh" as string]: "7px" } : undefined}
    >
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
          className="aspect-square w-full rounded-[16px] object-cover sm:aspect-[34/10]"
        />
      </picture>
    </Link>
  );
}
