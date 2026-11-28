"use client";

export const META_PIXEL_ID = process.env.NEXT_PUBLIC_META_PIXEL_ID;

type PixelValue = string | number | string[] | { id: string; quantity: number }[];
export type PixelParams = Record<string, PixelValue | undefined>;

type Fbq = (action: "track" | "init", name: string, params?: PixelParams, options?: { eventID: string }) => void;

declare global {
  interface Window {
    fbq?: Fbq;
  }
}

/**
 * eventID is the same event_id the server sends through the Conversions API
 * (internal/api/metaevents.go), so Meta counts the purchase once.
 */
export function pixelTrack(event: string, params?: PixelParams, eventID?: string): void {
  if (!META_PIXEL_ID || typeof window === "undefined" || !window.fbq) {
    return;
  }
  if (eventID) {
    window.fbq("track", event, params, { eventID });
    return;
  }
  window.fbq("track", event, params);
}
