"use client";

import Link from "next/link";
import { useEffect, useRef } from "react";
import { trackSelectPromotion, trackViewPromotion, type Promotion } from "@/lib/promo-analytics";
import type { VitrineImage } from "@/lib/vitrine";

/** Pixel sizes of the art files (what the art team delivers), per slot. */
const SIZES = {
  faixa_topo: { desktop: [2880, 120], mobile: [1170, 180] },
  banner_home: { desktop: [2720, 800], mobile: [1080, 1080] },
} as const;

/**
 * A campaign image from the Vitrine screen, the whole of it one link. Desktop and
 * phone art come as two files; width and height keep the space reserved while it
 * loads. view_promotion fires once it is half on screen, select_promotion on click.
 */
export default function PromoImage({
  image,
  href,
  slot,
  className = "",
  imgClassName = "",
  priority = false,
}: {
  image: VitrineImage;
  href: string;
  slot: keyof typeof SIZES;
  className?: string;
  imgClassName?: string;
  priority?: boolean;
}) {
  const ref = useRef<HTMLAnchorElement>(null);
  const promo: Promotion = { id: slot, name: image.alt, slot };
  const size = SIZES[slot];

  useEffect(() => {
    const el = ref.current;
    if (!el || typeof IntersectionObserver === "undefined") {
      return;
    }
    const io = new IntersectionObserver(
      (entries) => {
        if (entries.some((e) => e.isIntersecting)) {
          trackViewPromotion({ id: slot, name: image.alt, slot });
          io.disconnect();
        }
      },
      { threshold: 0.5 },
    );
    io.observe(el);
    return () => io.disconnect();
  }, [slot, image.alt, image.desktop]);

  return (
    <Link ref={ref} href={href} onClick={() => trackSelectPromotion(promo)} className={`block ${className}`}>
      <picture>
        <source media="(min-width: 640px)" srcSet={image.desktop} width={size.desktop[0]} height={size.desktop[1]} />
        {/* eslint-disable-next-line @next/next/no-img-element -- the art is already sized; next/image would re-encode it */}
        <img
          src={image.mobile}
          width={size.mobile[0]}
          height={size.mobile[1]}
          alt={image.alt}
          loading={priority ? "eager" : "lazy"}
          fetchPriority={priority ? "high" : "auto"}
          decoding="async"
          className={`block w-full ${imgClassName}`}
        />
      </picture>
    </Link>
  );
}
