"use client";

import Link from "next/link";
import { useEffect, useRef } from "react";
import Container from "@/components/ui/Container";
import { pixelText } from "@/lib/format";
import PromoImage from "@/components/promo/PromoImage";
import { trackSelectPromotion, trackViewPromotion, type Promotion } from "@/lib/promo-analytics";
import type { VitrineBanner } from "@/lib/vitrine";

const PROMO: Promotion = { id: "banner_home", name: "banner_home", slot: "home_topo" };

/**
 * The home banner from the dashboard's "Vitrine" screen (Natal: presentes por
 * faixa de preço). Measures view_promotion once it is on screen and
 * select_promotion on each link.
 */
export default function HolidayBanner({ banner }: { banner: VitrineBanner }) {
  const ref = useRef<HTMLElement>(null);
  const promo: Promotion = { ...PROMO, name: banner.title };

  useEffect(() => {
    const el = ref.current;
    if (!el || typeof IntersectionObserver === "undefined") {
      return;
    }
    const io = new IntersectionObserver(
      (entries) => {
        if (entries.some((e) => e.isIntersecting)) {
          trackViewPromotion({ ...PROMO, name: banner.title });
          io.disconnect();
        }
      },
      { threshold: 0.5 },
    );
    io.observe(el);
    return () => io.disconnect();
  }, [banner.title]);

  const tiers = banner.tiers ?? [];
  if (banner.image && banner.href) {
    return (
      <section aria-label={banner.title} className="border-b-[6px] border-outline bg-brand">
        <Container className="py-5">
          <PromoImage
            image={banner.image}
            href={banner.href}
            slot="banner_home"
            priority
            className="overflow-hidden rounded-[16px]"
          />
          {tiers.length > 0 && (
            <div className="mt-4 flex flex-wrap gap-3">
              {tiers.map((t) => (
                <Link
                  key={`${t.label}-${t.href}`}
                  href={t.href}
                  onClick={() => trackSelectPromotion({ ...promo, slot: `home_topo:${t.label}` })}
                  className="arcade-press sticker rounded-[10px] bg-surface px-4 py-3 font-pixel text-[10px] text-white [--sh:4px]"
                >
                  {pixelText(t.label)}
                </Link>
              ))}
            </div>
          )}
        </Container>
      </section>
    );
  }
  return (
    <section ref={ref} aria-label={banner.title} className="border-b-[6px] border-outline bg-brand">
      <Container className="flex flex-col gap-4 py-6 lg:flex-row lg:items-center lg:justify-between">
        <div className="min-w-0">
          <h2 className="font-display text-3xl font-bold leading-none text-white sm:text-4xl">{banner.title}</h2>
          {banner.text && <p className="mt-2 max-w-xl text-[15px] text-white/90">{banner.text}</p>}
        </div>
        {(tiers.length > 0 || (banner.cta && banner.href)) && (
          <div className="flex flex-wrap gap-3">
            {tiers.map((t) => (
              <Link
                key={`${t.label}-${t.href}`}
                href={t.href}
                onClick={() => trackSelectPromotion({ ...promo, slot: `home_topo:${t.label}` })}
                className="arcade-press sticker rounded-[10px] bg-surface px-4 py-3 font-pixel text-[10px] text-white [--sh:4px]"
              >
                {pixelText(t.label)}
              </Link>
            ))}
            {banner.cta && banner.href && (
              <Link
                href={banner.href}
                onClick={() => trackSelectPromotion({ ...promo, slot: "home_topo:cta" })}
                className="arcade-press sticker rounded-[10px] bg-outline px-4 py-3 font-pixel text-[10px] text-brand-soft [--sh:4px]"
              >
                {pixelText(banner.cta)}
              </Link>
            )}
          </div>
        )}
      </Container>
    </section>
  );
}
