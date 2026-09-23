"use client";

export const META_PIXEL_ID = process.env.NEXT_PUBLIC_META_PIXEL_ID;

type PixelValue = string | number | string[] | { id: string; quantity: number }[];
export type PixelParams = Record<string, PixelValue | undefined>;

type Fbq = (action: "track" | "init", name: string, params?: PixelParams) => void;

declare global {
  interface Window {
    fbq?: Fbq;
  }
}

export function pixelTrack(event: string, params?: PixelParams): void {
  if (!META_PIXEL_ID || typeof window === "undefined" || !window.fbq) {
    return;
  }
  window.fbq("track", event, params);
}
